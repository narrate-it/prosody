// Package store persists Prosody records inside caller-owned SQL transactions.
// It does not authenticate callers, authorize effects, schedule jobs, or commit.
// The AMOS host must hold current authority through the same transaction.
package store

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/narrate-it/prosody/internal/content"
)

//go:embed migrations/001_domain.sql
var Migration string

var (
	ErrConflict    = errors.New("submission key conflicts with existing payload")
	ErrUnavailable = errors.New("destination unavailable")
	ErrInvalid     = errors.New("invalid persistence input")
)

type Scope struct {
	InstallationID, ApplicationID, EnvironmentID, WorkspaceID, PersonID, OperationID string
}

func (s Scope) valid() bool {
	return UUID(s.InstallationID) && UUID(s.WorkspaceID) && UUID(s.PersonID) && safeLabel(s.ApplicationID) && safeLabel(s.EnvironmentID) && safeLabel(s.OperationID)
}
func safeLabel(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// UUID accepts canonical lowercase UUIDs. Only feed IDs require version 7.
func UUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return s[19] == '8' || s[19] == '9' || s[19] == 'a' || s[19] == 'b'
}
func FeedID(s string) bool { return UUID(s) && s[14] == '7' }

type Receipt struct {
	ItemID     string    `json:"itemId"`
	WorkID     string    `json:"workId"`
	AcceptedAt time.Time `json:"acceptedAt"`
}

type Acceptance struct {
	Scope      Scope
	Submission content.TextSubmission
	ItemID     string
	// StageWork must call the AMOS transaction-bound effect enqueuer and return its job ID.
	StageWork func(context.Context, string) (string, error)
	// Snapshot contains configuration references, never a plaintext provider key.
	SettingsSnapshot   []byte
	AuthorityReference []byte
}

// Accept stores a receipt and immutable source. New=false means a same-payload
// replay. StageWork runs only for a new source, using the SAME transaction.
// The host must roll back the transaction on any returned error, including
// enqueue/admission failure; this function never commits or retries a job.
// A returned receipt is not accepted externally until the host commits.
func Accept(ctx context.Context, tx Transaction, a Acceptance) (receipt Receipt, fresh bool, err error) {
	if !tx.valid() || !a.Scope.valid() || !UUID(a.ItemID) || a.StageWork == nil || !FeedID(a.Submission.FeedID) || a.Submission.WorkspaceID != a.Scope.WorkspaceID || len(a.SettingsSnapshot) == 0 || len(a.SettingsSnapshot) > 65536 || len(a.AuthorityReference) == 0 || len(a.AuthorityReference) > 4096 {
		return receipt, false, ErrInvalid
	}
	digest, err := a.Submission.Digest()
	if err != nil {
		return receipt, false, err
	}
	source, err := json.Marshal(a.Submission)
	if err != nil {
		return receipt, false, fmt.Errorf("encode source: %w", err)
	}
	// Destination lock fences deletion; current AMOS authority must also be held.
	var feed string
	err = tx.QueryRowContext(ctx, `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR SHARE`, a.Scope.WorkspaceID, a.Submission.FeedID).Scan(&feed)
	if errors.Is(err, sql.ErrNoRows) {
		return receipt, false, ErrUnavailable
	}
	if err != nil {
		return receipt, false, fmt.Errorf("lock destination: %w", err)
	}
	s := a.Scope
	err = tx.QueryRowContext(ctx, `INSERT INTO public.app_prosody_submissions
 (installation_id,application_id,environment_id,workspace_id,person_id,operation_id,key_bytes,digest,id,feed_id,source_bytes,settings_bytes,authority_ref)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
 ON CONFLICT(installation_id,application_id,environment_id,workspace_id,person_id,operation_id,key_bytes) DO NOTHING
 RETURNING id,accepted_at`, s.InstallationID, s.ApplicationID, s.EnvironmentID, s.WorkspaceID, s.PersonID, s.OperationID, []byte(a.Submission.IdempotencyKey), digest[:], a.ItemID, a.Submission.FeedID, source, a.SettingsSnapshot, a.AuthorityReference).Scan(&receipt.ItemID, &receipt.AcceptedAt)
	if err == nil {
		receipt.WorkID, err = a.StageWork(ctx, receipt.ItemID)
		if err != nil {
			return Receipt{}, false, fmt.Errorf("stage narration work: %w", err)
		}
		if !UUID(receipt.WorkID) {
			return Receipt{}, false, ErrInvalid
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_work(item_id,work_id) VALUES($1,$2)`, receipt.ItemID, receipt.WorkID); err != nil {
			return Receipt{}, false, fmt.Errorf("bind work receipt: %w", err)
		}
		return receipt, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return receipt, false, fmt.Errorf("persist submission: %w", err)
	}
	var stored []byte
	err = tx.QueryRowContext(ctx, `SELECT s.id,w.work_id,s.accepted_at,s.digest FROM public.app_prosody_submissions s JOIN public.app_prosody_work w ON w.item_id=s.id WHERE installation_id=$1 AND application_id=$2 AND environment_id=$3 AND workspace_id=$4 AND person_id=$5 AND operation_id=$6 AND key_bytes=$7`, s.InstallationID, s.ApplicationID, s.EnvironmentID, s.WorkspaceID, s.PersonID, s.OperationID, []byte(a.Submission.IdempotencyKey)).Scan(&receipt.ItemID, &receipt.WorkID, &receipt.AcceptedAt, &stored)
	if err != nil {
		return Receipt{}, false, fmt.Errorf("read submission receipt: %w", err)
	}
	if !bytes.Equal(digest[:], stored) {
		return Receipt{}, false, ErrConflict
	}
	return receipt, false, nil
}
