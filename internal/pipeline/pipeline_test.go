package pipeline_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"

	"github.com/narrate-it/prosody/internal/pipeline"
)

// Fixture capabilities exist only in tests. They provide NO native authority,
// provider reconciliation, database atomicity or durability qualification.
type candidate struct {
	binding   pipeline.Binding
	artifacts []pipeline.Artifact
}

func (c *candidate) Projection() pipeline.Binding   { return c.binding }
func (c *candidate) Artifacts() []pipeline.Artifact { return c.artifacts }

type checkpoint struct{ binding pipeline.Binding }

func (c *checkpoint) Projection() pipeline.Binding { return c.binding }

type published struct{ work pipeline.Work }

func (p *published) Work() pipeline.Work { return p.work }

type fixture struct {
	work                                                           pipeline.Work
	records                                                        map[pipeline.Stage]pipeline.StageRecord
	publication                                                    pipeline.PublicationRecord
	events                                                         []string
	calls                                                          map[pipeline.Stage]int
	dispatchErr, checkpointErr, validateErr, publishErr, resumeErr error
	mutateCandidate                                                func(*candidate)
	mutateReceipt                                                  func(*checkpoint)
	missingReceipt                                                 bool
	denyPublication                                                bool
	resumed                                                        pipeline.Execution
}

func newFixture() *fixture {
	digest := sha256.Sum256([]byte("exact persisted bytes\n"))
	return &fixture{
		work:        pipeline.Work{Realm: "realm", Workspace: "workspace", Job: "original-job", Invocation: "original-invocation", Item: "item", Feed: "feed", SettingsRevision: "revision", PayloadDigest: digest, SourceDigest: digest, SettingsDigest: digest},
		records:     map[pipeline.Stage]pipeline.StageRecord{pipeline.Rewrite: {Status: pipeline.Undispatched}, pipeline.Speech: {Status: pipeline.Undispatched}},
		publication: pipeline.PublicationRecord{Status: pipeline.PublicationAbsent}, calls: map[pipeline.Stage]int{},
	}
}
func (f *fixture) Work() pipeline.Work { return f.work }
func (f *fixture) LookupPublication(context.Context) (pipeline.PublicationRecord, error) {
	f.events = append(f.events, "publication.lookup")
	return f.publication, nil
}
func (f *fixture) ValidatePublication(context.Context, pipeline.PublicationReceipt) error {
	f.events = append(f.events, "publication.validate")
	if f.denyPublication {
		return pipeline.ErrInvalid
	}
	return nil
}
func (f *fixture) LookupStage(_ context.Context, s pipeline.Stage) (pipeline.StageRecord, error) {
	f.events = append(f.events, string(s)+".lookup")
	return f.records[s], nil
}
func (f *fixture) makeCandidate(s pipeline.Stage) *candidate {
	digest := sha256.Sum256([]byte("codec-envelope"))
	role := "script"
	if s == pipeline.Speech {
		role = "audio"
	}
	data := []byte("private bounded output")
	return &candidate{binding: pipeline.Binding{Work: f.work, Stage: s, Attempt: "original-" + string(s), Profile: "ordinary-v1", Registration: "registration-v1", RequestCodec: "request-v1", ResultCodec: "result-v1", RequestDigest: digest, ResultDigest: digest, ManifestDigest: digest}, artifacts: []pipeline.Artifact{{ID: string(s) + "-artifact", Role: role, MediaType: "registered-type", Bytes: data, Digest: sha256.Sum256(data)}}}
}
func (f *fixture) Dispatch(_ context.Context, s pipeline.Stage, upstream pipeline.CheckpointReceipt) (pipeline.Candidate, error) {
	f.events = append(f.events, string(s)+".dispatch")
	f.calls[s]++
	if s == pipeline.Speech && upstream == nil {
		return nil, pipeline.ErrInvalid
	}
	if f.dispatchErr != nil {
		f.records[s] = pipeline.StageRecord{Status: pipeline.Unresolved}
		return nil, f.dispatchErr
	}
	c := f.makeCandidate(s)
	if f.mutateCandidate != nil {
		f.mutateCandidate(c)
	}
	f.records[s] = pipeline.StageRecord{Status: pipeline.Accepted, Candidate: c}
	return c, nil
}
func (f *fixture) Checkpoint(_ context.Context, c pipeline.Candidate) (pipeline.CheckpointReceipt, error) {
	f.events = append(f.events, string(c.Projection().Stage)+".checkpoint")
	if f.checkpointErr != nil {
		f.records[c.Projection().Stage] = pipeline.StageRecord{Status: pipeline.Unresolved}
		return nil, f.checkpointErr
	}
	if f.missingReceipt {
		return nil, nil
	}
	r := &checkpoint{binding: c.Projection()}
	if f.mutateReceipt != nil {
		f.mutateReceipt(r)
	}
	f.records[c.Projection().Stage] = pipeline.StageRecord{Status: pipeline.Checkpointed, Receipt: r}
	return r, nil
}
func (f *fixture) ValidateCheckpoint(_ context.Context, r pipeline.CheckpointReceipt) error {
	f.events = append(f.events, string(r.Projection().Stage)+".validate")
	return f.validateErr
}
func (f *fixture) Publish(_ context.Context, rewrite, speech pipeline.CheckpointReceipt) (pipeline.PublicationReceipt, error) {
	f.events = append(f.events, "publish")
	if f.publishErr != nil {
		f.publication.Status = pipeline.PublicationUnresolved
		return nil, f.publishErr
	}
	if rewrite == nil || speech == nil {
		return nil, pipeline.ErrInvalid
	}
	p := &published{work: f.work}
	f.publication = pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: p}
	return p, nil
}
func (f *fixture) Resume(context.Context) (pipeline.Execution, error) {
	f.events = append(f.events, "resume")
	if f.resumeErr != nil {
		return nil, f.resumeErr
	}
	if f.resumed != nil {
		return f.resumed, nil
	}
	return f, nil
}

func TestOrdinaryFixedOrderAndCommittedReplay(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if _, err := pipeline.Run(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	want := []string{"publication.lookup", "rewrite.lookup", "rewrite.dispatch", "rewrite.checkpoint", "rewrite.validate", "speech.lookup", "speech.dispatch", "speech.checkpoint", "speech.validate", "publish", "publication.validate"}
	if !reflect.DeepEqual(f.events, want) {
		t.Fatalf("events=%v", f.events)
	}
	f.events = nil
	if _, err := pipeline.Run(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.events, []string{"publication.lookup", "publication.validate"}) {
		t.Fatal(f.events)
	}
	if f.calls[pipeline.Rewrite] != 1 || f.calls[pipeline.Speech] != 1 {
		t.Fatal(f.calls)
	}
}

func TestUncertaintyNeverAutomaticallyRetries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setup func(*fixture)
	}{
		{"transport drop", func(f *fixture) { f.dispatchErr = pipeline.ErrUnresolved }},
		{"truncated result", func(f *fixture) { f.dispatchErr = pipeline.ErrUnresolved }},
		{"checkpoint commit response lost", func(f *fixture) { f.checkpointErr = pipeline.ErrUnresolved }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			tc.setup(f)
			for range 2 {
				if _, err := pipeline.Run(context.Background(), f); !errors.Is(err, pipeline.ErrUnresolved) {
					t.Fatalf("err=%v", err)
				}
			}
			if f.calls[pipeline.Rewrite] != 1 || f.calls[pipeline.Speech] != 0 {
				t.Fatal(f.calls)
			}
		})
	}
}

func TestCheckpointRequiredBeforeNextStage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setup func(*fixture)
		want  error
	}{
		{"callback success without receipt", func(f *fixture) { f.missingReceipt = true }, pipeline.ErrInvalid},
		{"unsettled receipt", func(f *fixture) { f.validateErr = pipeline.ErrUnresolved }, pipeline.ErrUnresolved},
		{"output lost", func(f *fixture) { f.records[pipeline.Rewrite] = pipeline.StageRecord{Status: pipeline.OutputLost} }, pipeline.ErrOutputLost},
		{"unrecognized absence", func(f *fixture) { delete(f.records, pipeline.Rewrite) }, pipeline.ErrInvalid},
		{"typed nil candidate", func(f *fixture) {
			var c *candidate
			f.records[pipeline.Rewrite] = pipeline.StageRecord{Status: pipeline.Accepted, Candidate: c}
		}, pipeline.ErrInvalid},
		{"typed nil receipt", func(f *fixture) {
			var r *checkpoint
			f.records[pipeline.Rewrite] = pipeline.StageRecord{Status: pipeline.Checkpointed, Receipt: r}
		}, pipeline.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			tc.setup(f)
			if _, err := pipeline.Run(context.Background(), f); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v", err)
			}
			if f.calls[pipeline.Speech] != 0 {
				t.Fatal(f.calls)
			}
			for _, event := range f.events {
				if event == "publish" {
					t.Fatal(f.events)
				}
			}
		})
	}
}

func TestStaleCandidateAndReceiptDenials(t *testing.T) {
	t.Parallel()
	mutations := []struct {
		name   string
		mutate func(*candidate)
	}{
		{"job", func(c *candidate) { c.binding.Work.Job = "foreign" }},
		{"workspace", func(c *candidate) { c.binding.Work.Workspace = "foreign" }},
		{"invocation", func(c *candidate) { c.binding.Work.Invocation = "new-fence-is-not-invocation" }},
		{"source digest", func(c *candidate) { c.binding.Work.SourceDigest = sha256.Sum256([]byte("normalized source")) }},
		{"settings digest", func(c *candidate) { c.binding.Work.SettingsDigest = sha256.Sum256([]byte("live settings")) }},
		{"duplicate stage", func(c *candidate) { c.binding.Stage = pipeline.Speech }},
		{"artifact digest", func(c *candidate) { c.artifacts[0].Bytes = []byte("substituted") }},
		{"artifact ID", func(c *candidate) { c.artifacts[0].ID = "" }},
		{"oversized artifact", func(c *candidate) {
			c.artifacts[0].Bytes = make([]byte, 65537)
			c.artifacts[0].Digest = sha256.Sum256(c.artifacts[0].Bytes)
		}},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			f.mutateCandidate = tc.mutate
			if _, err := pipeline.Run(context.Background(), f); !errors.Is(err, pipeline.ErrInvalid) {
				t.Fatal(err)
			}
			if f.calls[pipeline.Speech] != 0 {
				t.Fatal(f.calls)
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func(*checkpoint)
	}{
		{"request", func(r *checkpoint) { r.binding.RequestDigest = sha256.Sum256([]byte("different request")) }},
		{"result", func(r *checkpoint) { r.binding.ResultDigest = sha256.Sum256([]byte("different result")) }},
		{"attempt", func(r *checkpoint) { r.binding.Attempt = "different attempt" }},
		{"manifest", func(r *checkpoint) { r.binding.ManifestDigest = sha256.Sum256([]byte("different artifact")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			f.mutateReceipt = tc.mutate
			if _, err := pipeline.Run(context.Background(), f); !errors.Is(err, pipeline.ErrInvalid) {
				t.Fatal(err)
			}
			if f.calls[pipeline.Speech] != 0 {
				t.Fatal(f.calls)
			}
		})
	}
}

func TestReentryUsesCommittedCheckpointWithoutPaidReplay(t *testing.T) {
	t.Parallel()
	f := newFixture()
	c := f.makeCandidate(pipeline.Rewrite)
	f.records[pipeline.Rewrite] = pipeline.StageRecord{Status: pipeline.Checkpointed, Receipt: &checkpoint{binding: c.binding}}
	if _, err := pipeline.Reenter(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if f.calls[pipeline.Rewrite] != 0 || f.calls[pipeline.Speech] != 1 {
		t.Fatal(f.calls)
	}
	if f.events[0] != "publication.lookup" || f.events[1] != "resume" {
		t.Fatal(f.events)
	}
}

func TestPublicationLookupBeforeProcessingCheckpointReconcile(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.publishErr = pipeline.ErrUnresolved
	if _, err := pipeline.Run(context.Background(), f); !errors.Is(err, pipeline.ErrUnresolved) {
		t.Fatal(err)
	}
	f.events = nil
	if _, err := pipeline.Reenter(context.Background(), f); !errors.Is(err, pipeline.ErrUnresolved) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.events, []string{"publication.lookup"}) {
		t.Fatal(f.events)
	}
	// Native same-publication lookup recovered the committed publication. Private
	// checkpoints can now be unavailable; they must never be consulted first.
	f.publication = pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: &published{work: f.work}}
	f.records = nil
	f.events = nil
	if _, err := pipeline.Reenter(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.events, []string{"publication.lookup", "publication.validate"}) {
		t.Fatal(f.events)
	}
	if f.calls[pipeline.Rewrite] != 1 || f.calls[pipeline.Speech] != 1 {
		t.Fatal(f.calls)
	}
}

func TestReentryRequiresNativeProofAndOriginalWork(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setup func(*fixture)
		want  error
	}{
		{"unsupported lookup", func(f *fixture) { f.resumeErr = pipeline.ErrUnresolved }, pipeline.ErrUnresolved},
		{"lost settled output", func(f *fixture) { f.resumeErr = pipeline.ErrOutputLost }, pipeline.ErrOutputLost},
		{"new invocation", func(f *fixture) { other := newFixture(); other.work.Invocation = "replacement"; f.resumed = other }, pipeline.ErrInvalid},
		{"native admission denied", func(f *fixture) { f.validateErr = pipeline.ErrInvalid }, pipeline.ErrInvalid},
		{"foreign publication", func(f *fixture) {
			w := f.work
			w.Workspace = "foreign"
			f.publication = pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: &published{work: w}}
		}, pipeline.ErrInvalid},
		{"current publication admission denied", func(f *fixture) {
			f.denyPublication = true
			f.publication = pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: &published{work: f.work}}
		}, pipeline.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			tc.setup(f)
			if _, err := pipeline.Reenter(context.Background(), f); !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			if f.calls[pipeline.Speech] != 0 {
				t.Fatal(f.calls)
			}
		})
	}
}
