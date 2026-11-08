// Package pipeline prepares the ordinary rewrite/speech consumer against LOCAL
// ports. These interfaces are not supported AMOS APIs. There is deliberately no
// production adapter, scheduler, retry policy, authority constructor or storage.
package pipeline

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
)

var (
	ErrInvalid    = errors.New("invalid pipeline binding")
	ErrUnresolved = errors.New("pipeline requires original-identity reconciliation")
	ErrOutputLost = errors.New("settled pipeline output unavailable; manual recovery required")
)

type Stage string

const (
	Rewrite Stage = "rewrite"
	Speech  Stage = "speech"
)

// Work is a descriptive immutable projection, never authority. SourceDigest and
// SettingsDigest bind exact persisted bytes, not logical Submission.Digest.
// Invocation and Job remain the original identities across execution fences.
type Work struct {
	Realm, Workspace, Job, Invocation, Item, Feed, SettingsRevision string
	PayloadDigest, SourceDigest, SettingsDigest                     [32]byte
}

// Binding projects the native accepted-result chain. Fences are deliberately
// absent: an adapter must recheck CURRENT native admission when consuming a
// prior committed receipt, rather than treating its old fence as authority.
type Binding struct {
	Work                                                      Work
	Stage                                                     Stage
	Attempt, Profile, Registration, RequestCodec, ResultCodec string
	RequestDigest, ResultDigest, ManifestDigest               [32]byte
}

type Artifact struct {
	ID, Role, MediaType string
	Digest              [32]byte
	Bytes               []byte
}

// Candidate and receipt interfaces carry opaque adapter-owned native state.
// Their projections are untrusted descriptive data. Matching projections do
// not confer authority; every effect/receipt consumption MUST validate the
// underlying native capability in the adapter. There are no local constructors.
// Artifact bytes are bounded process memory, never generic journal evidence.
type Candidate interface {
	Projection() Binding
	Artifacts() []Artifact
}
type CheckpointReceipt interface{ Projection() Binding }
type PublicationReceipt interface{ Work() Work }

type StageStatus uint8

const (
	// Undispatched is native-proven absence of a prior attempt, not zero/default.
	Undispatched StageStatus = iota + 1
	Unresolved
	Accepted
	Checkpointed
	OutputLost
)

type StageRecord struct {
	Status    StageStatus
	Candidate Candidate
	Receipt   CheckpointReceipt
}

type PublicationStatus uint8

const (
	PublicationAbsent PublicationStatus = iota + 1
	PublicationUnresolved
	PublicationCommitted
)

type PublicationRecord struct {
	Status  PublicationStatus
	Receipt PublicationReceipt
}

// Publications looks up the ORIGINAL publication identity, including unknown
// commit completion. Publish must validate both receipts, current native source,
// financial/resource/fence authority, then run store.Publish AND mandatory
// StageReady in the SAME transaction, checking exact affected count 1. Neither
// callback success nor a private row is a committed publication receipt.
type Publications interface {
	LookupPublication(context.Context) (PublicationRecord, error)
	ValidatePublication(context.Context, PublicationReceipt) error
}

// Stages must load the original accepted attempt/result identity, never invent
// a fresh attempt because lookup/output is missing. Dispatch selects only a
// frozen registered stage and constructs its registered canonical request from
// accepted work/upstream receipt. It commits native intent before transport.
// A raw provider ResponseReceived is not Accepted/native settlement proof.
type Stages interface {
	LookupStage(context.Context, Stage) (StageRecord, error)
	Dispatch(context.Context, Stage, CheckpointReceipt) (Candidate, error)
	// Checkpoint atomically commits bounded private bytes and the opaque native
	// receipt, validating actual artifact bytes against the accepted manifest.
	// An error may mean UNKNOWN; a nil error without a receipt is NOT success.
	Checkpoint(context.Context, Candidate) (CheckpointReceipt, error)
}

type Execution interface {
	Publications
	Stages
	Work() Work
	// ValidateCheckpoint checks native receipt identity, settled accounting,
	// immutable digest/artifact bindings and current admission. Projection checks
	// below are only additional consumer guards, not native validation.
	ValidateCheckpoint(context.Context, CheckpointReceipt) error
	Publish(context.Context, CheckpointReceipt, CheckpointReceipt) (PublicationReceipt, error)
}

// Reconciliation has no dispatch method. Resume reconciles EVERY ambiguous
// attempt under its SAME original identity before returning execution; absent
// lookup proof stays UNKNOWN/manual. Settled missing output is OutputLost or
// qualified same-result retrieval, never a new paid call. Native jobs.Worker
// remains the only scheduler and owner of any Continue transition.
type Reconciliation interface {
	Publications
	Work() Work
	Resume(context.Context) (Execution, error)
}

// Run performs at most one dispatch per each of the two fixed stages. Any
// uncertainty stops immediately. Calling Run again is safe only through native
// lookup/admission: a failed dispatch/checkpoint must NOT appear Undispatched.
// Cancellation stops subsequent port calls, not work already handed off. Prior
// port errors keep their classification; cancellation is never no-effect proof.
func Run(ctx context.Context, execution Execution) (PublicationReceipt, error) {
	if nilValue(execution) || !validWork(execution.Work()) {
		return nil, ErrInvalid
	}
	work := execution.Work()
	receipt, done, err := publication(ctx, execution, work)
	if err != nil || done {
		return receipt, safeError(err)
	}
	var prior CheckpointReceipt
	var rewrite CheckpointReceipt
	for _, stage := range [...]Stage{Rewrite, Speech} {
		if err := ctx.Err(); err != nil {
			return nil, safeError(err)
		}
		record, err := execution.LookupStage(ctx, stage)
		if err != nil {
			return nil, safeError(err)
		}
		switch record.Status {
		case Unresolved:
			return nil, ErrUnresolved
		case OutputLost:
			return nil, ErrOutputLost
		case Undispatched:
			if !nilValue(record.Candidate) || !nilValue(record.Receipt) {
				return nil, ErrInvalid
			}
			if err := ctx.Err(); err != nil {
				return nil, safeError(err)
			}
			record.Candidate, err = execution.Dispatch(ctx, stage, prior)
			if err != nil {
				return nil, safeError(err)
			}
			fallthrough
		case Accepted:
			if !nilValue(record.Receipt) || !validCandidate(record.Candidate, work, stage) {
				return nil, ErrInvalid
			}
			binding := record.Candidate.Projection()
			if err := ctx.Err(); err != nil {
				return nil, safeError(err)
			}
			record.Receipt, err = execution.Checkpoint(ctx, record.Candidate)
			if err != nil {
				return nil, safeError(err)
			}
			if nilValue(record.Receipt) || record.Receipt.Projection() != binding {
				return nil, ErrInvalid
			}
		case Checkpointed:
			if !nilValue(record.Candidate) {
				return nil, ErrInvalid
			}
		default:
			return nil, ErrInvalid
		}
		if nilValue(record.Receipt) || !validBinding(record.Receipt.Projection(), work, stage) {
			return nil, ErrInvalid
		}
		if err := ctx.Err(); err != nil {
			return nil, safeError(err)
		}
		if err := execution.ValidateCheckpoint(ctx, record.Receipt); err != nil {
			return nil, safeError(err)
		}
		prior = record.Receipt
		if stage == Rewrite {
			rewrite = prior
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, safeError(err)
	}
	receipt, err = execution.Publish(ctx, rewrite, prior)
	if err != nil {
		return nil, safeError(err)
	}
	if nilValue(receipt) || receipt.Work() != work {
		return nil, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, safeError(err)
	}
	if err := execution.ValidatePublication(ctx, receipt); err != nil {
		return nil, safeError(err)
	}
	return receipt, nil
}

// Reenter always looks up committed publication before processing-only private
// checkpoints. An uncertain publication can already have made those private
// checkpoints unavailable. Resume must reconcile original native attempt IDs.
func Reenter(ctx context.Context, reconciliation Reconciliation) (PublicationReceipt, error) {
	if nilValue(reconciliation) || !validWork(reconciliation.Work()) {
		return nil, ErrInvalid
	}
	work := reconciliation.Work()
	receipt, done, err := publication(ctx, reconciliation, work)
	if err != nil || done {
		return receipt, safeError(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, safeError(err)
	}
	execution, err := reconciliation.Resume(ctx)
	if err != nil {
		return nil, safeError(err)
	}
	if nilValue(execution) || execution.Work() != work {
		return nil, ErrInvalid
	}
	return Run(ctx, execution)
}

// publication checks the original completion before any stage work, preserving
// unresolved outcomes and validating known receipts through the admitted port.
func publication(ctx context.Context, port Publications, work Work) (PublicationReceipt, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, safeError(err)
	}
	record, err := port.LookupPublication(ctx)
	if err != nil {
		return nil, false, safeError(err)
	}
	switch record.Status {
	case PublicationAbsent:
		if !nilValue(record.Receipt) {
			return nil, false, ErrInvalid
		}
		return nil, false, nil
	case PublicationUnresolved:
		return nil, false, ErrUnresolved
	case PublicationCommitted:
		if nilValue(record.Receipt) || record.Receipt.Work() != work {
			return nil, false, ErrInvalid
		}
		if err := ctx.Err(); err != nil {
			return nil, false, safeError(err)
		}
		if err := port.ValidatePublication(ctx, record.Receipt); err != nil {
			return nil, false, safeError(err)
		}
		return record.Receipt, true, nil
	default:
		return nil, false, ErrInvalid
	}
}

func validWork(w Work) bool {
	return w.Realm != "" && w.Workspace != "" && w.Job != "" && w.Invocation != "" && w.Item != "" && w.Feed != "" && w.SettingsRevision != "" && w.PayloadDigest != [32]byte{} && w.SourceDigest != [32]byte{} && w.SettingsDigest != [32]byte{}
}
func validBinding(b Binding, w Work, stage Stage) bool {
	return b.Work == w && b.Stage == stage && b.Attempt != "" && b.Profile != "" && b.Registration != "" && b.RequestCodec != "" && b.ResultCodec != "" && b.RequestDigest != [32]byte{} && b.ResultDigest != [32]byte{} && b.ManifestDigest != [32]byte{}
}

// First profile: exactly one script or audio artifact, held only in memory
// until the native atomic checkpoint. Registered codecs remain responsible for
// UTF-8/script/audio validity, canonical envelope/manifest and MIME validation.
func validCandidate(c Candidate, w Work, stage Stage) bool {
	if nilValue(c) || !validBinding(c.Projection(), w, stage) {
		return false
	}
	artifacts := c.Artifacts()
	if len(artifacts) != 1 {
		return false
	}
	a := artifacts[0]
	maxBytes := 65536
	role := "script"
	if stage == Speech {
		maxBytes, role = 32<<20, "audio"
	}
	return a.ID != "" && a.Role == role && a.MediaType != "" && len(a.Bytes) > 0 && len(a.Bytes) <= maxBytes && sha256.Sum256(a.Bytes) == a.Digest
}

func nilValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// Port failures can originate near sensitive candidate/provider bytes. Return
// only fixed classifications; never expose arbitrary adapter error strings.
func safeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ErrInvalid):
		return ErrInvalid
	case errors.Is(err, ErrOutputLost):
		return ErrOutputLost
	default:
		return ErrUnresolved
	}
}
