package pipeline_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/narrate-it/prosody/internal/pipeline"
)

// Cancellation controls deliberately return completed synthetic port results.
// They prove local call ordering, not native authority, durable receipts or
// transport cancellation. The embedded fixture retains original attempts.
type cancelPort struct {
	*fixture
	at     string
	cancel context.CancelFunc
}

// after cancels the test context only after the selected port has completed.
func (p *cancelPort) after(event string) {
	if event == p.at {
		p.cancel()
	}
}
func (p *cancelPort) LookupPublication(ctx context.Context) (pipeline.PublicationRecord, error) {
	r, e := p.fixture.LookupPublication(ctx)
	p.after("publication.lookup")
	return r, e
}
func (p *cancelPort) ValidatePublication(ctx context.Context, r pipeline.PublicationReceipt) error {
	e := p.fixture.ValidatePublication(ctx, r)
	p.after("publication.validate")
	return e
}
func (p *cancelPort) LookupStage(ctx context.Context, s pipeline.Stage) (pipeline.StageRecord, error) {
	r, e := p.fixture.LookupStage(ctx, s)
	p.after(string(s) + ".lookup")
	return r, e
}
func (p *cancelPort) Dispatch(ctx context.Context, s pipeline.Stage, r pipeline.CheckpointReceipt) (pipeline.Candidate, error) {
	c, e := p.fixture.Dispatch(ctx, s, r)
	p.after(string(s) + ".dispatch")
	return c, e
}
func (p *cancelPort) Checkpoint(ctx context.Context, c pipeline.Candidate) (pipeline.CheckpointReceipt, error) {
	r, e := p.fixture.Checkpoint(ctx, c)
	p.after(string(c.Projection().Stage) + ".checkpoint")
	return r, e
}
func (p *cancelPort) ValidateCheckpoint(ctx context.Context, r pipeline.CheckpointReceipt) error {
	e := p.fixture.ValidateCheckpoint(ctx, r)
	p.after(string(r.Projection().Stage) + ".validate")
	return e
}
func (p *cancelPort) Publish(ctx context.Context, a, b pipeline.CheckpointReceipt) (pipeline.PublicationReceipt, error) {
	r, e := p.fixture.Publish(ctx, a, b)
	p.after("publish")
	return r, e
}
func (p *cancelPort) Resume(ctx context.Context) (pipeline.Execution, error) {
	_, e := p.fixture.Resume(ctx)
	p.after("resume")
	return p, e
}

// TestPipelineCancellationBoundaries checks call prefixes and original-attempt
// synthetic recovery after cancellation between ordinary processing phases.
func TestPipelineCancellationBoundaries(t *testing.T) {
	t.Parallel()
	order := []string{"publication.lookup", "rewrite.lookup", "rewrite.dispatch", "rewrite.checkpoint", "rewrite.validate", "speech.lookup", "speech.dispatch", "speech.checkpoint", "speech.validate", "publish", "publication.validate"}
	for _, at := range order[:len(order)-1] {
		t.Run(at, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			p := &cancelPort{fixture: f, at: at, cancel: cancel}
			r, err := pipeline.Run(ctx, p)
			if r != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("receipt=%v err=%v", r, err)
			}
			cut := 0
			for i, event := range order {
				if event == at {
					cut = i + 1
					break
				}
			}
			if !reflect.DeepEqual(f.events, order[:cut]) {
				t.Fatalf("continued after %s: events=%v", at, f.events)
			}
			before := map[pipeline.Stage]int{pipeline.Rewrite: f.calls[pipeline.Rewrite], pipeline.Speech: f.calls[pipeline.Speech]}
			f.events = nil
			if r, err := pipeline.Reenter(context.Background(), f); r == nil || err != nil {
				t.Fatalf("original-state recovery receipt=%v err=%v", r, err)
			}
			for _, s := range []pipeline.Stage{pipeline.Rewrite, pipeline.Speech} {
				if f.calls[s] > 1 || (before[s] == 1 && f.calls[s] != 1) {
					t.Fatalf("replayed accepted attempt: %s calls=%d", s, f.calls[s])
				}
			}
		})
	}
}

// TestPipelineCancelledEntryAndReconciliation checks pre-cancelled entry and
// cancellation before completion validation or recovery admission.
func TestPipelineCancelledEntryAndReconciliation(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"run", "reenter", "committed lookup", "before resume", "after resume"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			p := &cancelPort{fixture: f, cancel: cancel}
			var want []string
			switch mode {
			case "run", "reenter":
				cancel()
			case "committed lookup":
				f.publication = pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: &published{work: f.work}}
				p.at = "publication.lookup"
				want = []string{"publication.lookup"}
			case "before resume":
				p.at = "publication.lookup"
				want = []string{"publication.lookup"}
			case "after resume":
				p.at = "resume"
				want = []string{"publication.lookup", "resume"}
			}
			var r pipeline.PublicationReceipt
			var err error
			if mode == "run" || mode == "committed lookup" {
				r, err = pipeline.Run(ctx, p)
			} else {
				r, err = pipeline.Reenter(ctx, p)
			}
			if r != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("receipt=%v err=%v events=%v", r, err, f.events)
			}
			if !reflect.DeepEqual(f.events, want) {
				t.Fatalf("events=%v want=%v", f.events, want)
			}
		})
	}
}

// TestPipelineCancellationDoesNotEraseUncertainty preserves unresolved original
// attempts instead of using cancellation as permission for another dispatch.
func TestPipelineCancellationDoesNotEraseUncertainty(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.dispatchErr = pipeline.ErrUnresolved
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	p := &cancelPort{fixture: f, at: "rewrite.dispatch", cancel: cancel}
	if r, err := pipeline.Run(ctx, p); r != nil || !errors.Is(err, pipeline.ErrUnresolved) {
		t.Fatalf("receipt=%v err=%v", r, err)
	}
	if f.records[pipeline.Rewrite].Status != pipeline.Unresolved {
		t.Fatal("cancellation erased uncertain attempt")
	}
	if _, err := pipeline.Reenter(context.Background(), f); !errors.Is(err, pipeline.ErrUnresolved) {
		t.Fatal(err)
	}
	if f.calls[pipeline.Rewrite] != 1 || f.calls[pipeline.Speech] != 0 {
		t.Fatal(f.calls)
	}
}

// TestPipelineFinalValidationRetainsCommittedReceipt preserves a completed final
// validation even when its callback cancels the context before returning.
func TestPipelineFinalValidationRetainsCommittedReceipt(t *testing.T) {
	t.Parallel()
	f := newFixture()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	p := &cancelPort{fixture: f, at: "publication.validate", cancel: cancel}
	r, err := pipeline.Run(ctx, p)
	if r == nil || err != nil || ctx.Err() != context.Canceled || f.publication.Status != pipeline.PublicationCommitted {
		t.Fatalf("receipt=%v err=%v", r, err)
	}
	// Cancellation after the final completed port has no next phase to prevent.
	// The synthetic committed receipt is not native commit qualification.
}
