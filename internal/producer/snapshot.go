// Package producer defines inactive descriptive producer snapshots. No encoder,
// digest or source-class value establishes native identity or capture authority.
package producer

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/narrate-it/prosody/internal/strictjson"
)

const (
	Version          = 1
	Profile          = "prosody.producer-snapshot"
	ProfileRevision  = 1
	MaxLabelBytes    = 256
	MaxClientIDBytes = 2048
	MaxSnapshotBytes = 8192
)

type LabelSource string

const (
	OwnerAssignedConnectionLabel    LabelSource = "owner-assigned-connection-label"
	ClientAdvertisedApplicationName LabelSource = "client-advertised-application-name"
)

var (
	ErrInvalid       = errors.New("invalid producer snapshot")
	ErrScopeMismatch = errors.New("producer snapshot scope mismatch")
)

// Scope contains descriptive original-capture bindings, not native selectors or
// authority. WorkID may be absent only when the future admitted capture profile
// permits it; omission is immutable and cannot be filled by a later retry.
type Scope struct {
	WorkspaceID          string `json:"workspaceId"`
	SourceItemID         string `json:"sourceItemId"`
	OriginalInvocationID string `json:"originalInvocationId"`
	WorkID               string `json:"workId,omitempty"`
}

// Snapshot captures label provenance through the explicit source class, original
// public client reference, scope and time. Neither source class is a default or
// enabled product policy. An application name is not a verified individual agent.
// The future host must supply a nonsecret original client reference and prove the
// capture relationship; this package cannot establish either from strings.
type Snapshot struct {
	Version          int         `json:"version"`
	Profile          string      `json:"profile"`
	ProfileRevision  int         `json:"profileRevision"`
	LabelSource      LabelSource `json:"labelSource"`
	OriginalClientID string      `json:"originalClientId"`
	Label            string      `json:"label"`
	Scope            Scope       `json:"scope"`
	CapturedAt       time.Time   `json:"capturedAt"`
}

// Encode validates and deterministically encodes local profile revision 1. It
// never supplies missing fields, chooses a label source or derives a label from
// an ID. Only the timestamp offset/monotonic component is canonicalized to UTC.
// Source/settings/tool-input encodings and idempotency digests are untouched.
func Encode(snapshot Snapshot) ([]byte, [sha256.Size]byte, error) {
	if snapshot.Version != Version || snapshot.Profile != Profile || snapshot.ProfileRevision != ProfileRevision ||
		(snapshot.LabelSource != OwnerAssignedConnectionLabel && snapshot.LabelSource != ClientAdvertisedApplicationName) ||
		!publicText(snapshot.Label, MaxLabelBytes) || !publicText(snapshot.OriginalClientID, MaxClientIDBytes) || !validScope(snapshot.Scope) || snapshot.CapturedAt.IsZero() {
		return nil, [sha256.Size]byte{}, ErrInvalid
	}
	snapshot.CapturedAt = snapshot.CapturedAt.UTC()
	if snapshot.CapturedAt.Year() < 1970 || snapshot.CapturedAt.Year() > 9999 {
		return nil, [sha256.Size]byte{}, ErrInvalid
	}
	raw, err := json.Marshal(snapshot)
	if err != nil || len(raw) > MaxSnapshotBytes {
		return nil, [sha256.Size]byte{}, ErrInvalid
	}
	return raw, sha256.Sum256(raw), nil
}

// Decode requires the exact canonical bytes and their expected digest. Canonical
// equality rejects duplicate fields (including escaped aliases), unknown fields,
// case-folded names, alternative timestamp forms and noncanonical JSON. Digest
// integrity is not proof that a client supplied truthful metadata or owns a name.
func Decode(raw []byte, expected [sha256.Size]byte) (Snapshot, error) {
	if len(raw) == 0 || len(raw) > MaxSnapshotBytes || sha256.Sum256(raw) != expected || strictjson.Validate(raw) != nil {
		return Snapshot{}, ErrInvalid
	}
	var snapshot Snapshot
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&snapshot) != nil {
		return Snapshot{}, ErrInvalid
	}
	canonical, _, err := Encode(snapshot)
	if err != nil || !bytes.Equal(raw, canonical) {
		return Snapshot{}, ErrInvalid
	}
	return snapshot, nil
}

// RetainOriginal checks a persisted snapshot against the original captured scope.
// It accepts no replacement client/label metadata: a later name or valid client
// cannot overwrite an existing submission's producer. This is a pure replay model,
// not a database receipt, commit, authorization or idempotency lookup implementation.
func RetainOriginal(raw []byte, digest [sha256.Size]byte, original Scope) (Snapshot, error) {
	if !validScope(original) {
		return Snapshot{}, ErrInvalid
	}
	snapshot, err := Decode(raw, digest)
	if err != nil {
		return Snapshot{}, err
	}
	if snapshot.Scope != original {
		return Snapshot{}, ErrScopeMismatch
	}
	return snapshot, nil
}

func publicText(value string, maxBytes int) bool {
	if len(value) == 0 || len(value) > maxBytes || !utf8.ValidString(value) || strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return false
		}
	}
	return true
}

func validScope(s Scope) bool {
	return canonicalUUID(s.WorkspaceID) && canonicalUUID(s.SourceItemID) && canonicalUUID(s.OriginalInvocationID) && (s.WorkID == "" || canonicalUUID(s.WorkID))
}

func canonicalUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, c := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return value[19] == '8' || value[19] == '9' || value[19] == 'a' || value[19] == 'b'
}
