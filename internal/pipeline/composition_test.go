package pipeline_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/briefing"
	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/openrouter"
	"github.com/narrate-it/prosody/internal/pipeline"
)

// All ports/candidates/receipts below are SYNTHETIC TEST-ONLY capabilities. They
// connect real briefing/media code to pipeline.Run for data-flow assertions,
// not native admission, settlement, canonical codecs, atomicity or durability.
// No transport, provider account, scheduler, native journal or lookup proof is
// implemented. Raw ResponseReceived/MayHaveActed remain observations only.
type compositionProvider struct {
	script                    string
	audio                     []byte
	rewriteErr, speechErr     error
	rewriteCalls, speechCalls int
	sourceSeen, scriptSeen    string
}

func (p *compositionProvider) Rewrite(_ context.Context, _, _, _, source string, _ int) (openrouter.RewriteResult, error) {
	p.rewriteCalls++
	p.sourceSeen = source
	observation, err := openrouter.FinishAttempt(openrouter.AttemptOutcome{State: openrouter.ResponseReceived, RequestID: "synthetic-rewrite-request", GenerationID: "synthetic-rewrite-generation"}, p.rewriteErr)
	return openrouter.RewriteResult{Text: p.script, Outcome: observation, RequestID: observation.RequestID, GenerationID: observation.GenerationID}, err
}
func (p *compositionProvider) Speech(_ context.Context, _, _, _, script string) (openrouter.SpeechResult, error) {
	p.speechCalls++
	p.scriptSeen = script
	observation, err := openrouter.FinishAttempt(openrouter.AttemptOutcome{State: openrouter.ResponseReceived, RequestID: "synthetic-speech-request", GenerationID: "synthetic-speech-generation"}, p.speechErr)
	return openrouter.SpeechResult{Audio: p.audio, Outcome: observation, RequestID: observation.RequestID, GenerationID: observation.GenerationID, ContentType: "audio/mpeg"}, err
}

type compositionCandidate struct {
	binding  pipeline.Binding
	artifact pipeline.Artifact
	duration time.Duration
}

func (c *compositionCandidate) Projection() pipeline.Binding { return c.binding }
func (c *compositionCandidate) Artifacts() []pipeline.Artifact {
	return []pipeline.Artifact{c.artifact}
}

type compositionReceipt struct{ candidate compositionCandidate }

func (r *compositionReceipt) Projection() pipeline.Binding { return r.candidate.binding }

type compositionPublication struct {
	work          pipeline.Work
	script, audio []byte
	duration      time.Duration
}

func (p *compositionPublication) Work() pipeline.Work { return p.work }

type compositionExecution struct {
	work            pipeline.Work
	source          content.TextSubmission
	provider        *compositionProvider
	records         map[pipeline.Stage]pipeline.StageRecord
	committed       map[pipeline.Stage]*compositionReceipt
	checkpointCalls map[pipeline.Stage]int
	checkpointLoss  pipeline.Stage
	publication     *compositionPublication
	publishCalls    int
	rewriteResult   briefing.Result
	speechResult    briefing.Synthesized
	originalError   error
}

func compositionFixture(t *testing.T) *compositionExecution {
	t.Helper()
	audio, err := os.ReadFile("../briefing/testdata/silence.mp3")
	if err != nil {
		t.Fatal(err)
	}
	source := content.TextSubmission{WorkspaceID: "synthetic-workspace", FeedID: "synthetic-feed", IdempotencyKey: "synthetic-only", Text: "The first run succeeded. A second run is still required."}
	sourceBytes, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	settingsBytes := []byte(`{"revision":"synthetic-settings-revision","model":"synthetic-model","voice":"synthetic-voice"}`)
	return &compositionExecution{
		work:      pipeline.Work{Realm: "synthetic-realm", Workspace: "synthetic-workspace", Job: "synthetic-original-job", Invocation: "synthetic-original-invocation", Item: "synthetic-item", Feed: "synthetic-feed", SettingsRevision: "synthetic-settings-revision", PayloadDigest: sha256.Sum256([]byte("synthetic-frozen-work")), SourceDigest: sha256.Sum256(sourceBytes), SettingsDigest: sha256.Sum256(settingsBytes)},
		source:    source,
		provider:  &compositionProvider{script: "The first run succeeded; a second run is still required.", audio: audio},
		records:   map[pipeline.Stage]pipeline.StageRecord{pipeline.Rewrite: {Status: pipeline.Undispatched}, pipeline.Speech: {Status: pipeline.Undispatched}},
		committed: map[pipeline.Stage]*compositionReceipt{}, checkpointCalls: map[pipeline.Stage]int{},
	}
}
func (e *compositionExecution) Work() pipeline.Work { return e.work }
func (e *compositionExecution) LookupPublication(context.Context) (pipeline.PublicationRecord, error) {
	if e.publication != nil {
		return pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: e.publication}, nil
	}
	return pipeline.PublicationRecord{Status: pipeline.PublicationAbsent}, nil
}
func (e *compositionExecution) ValidatePublication(_ context.Context, r pipeline.PublicationReceipt) error {
	if r != e.publication || e.publication == nil {
		return pipeline.ErrInvalid
	}
	return nil
}
func (e *compositionExecution) LookupStage(_ context.Context, s pipeline.Stage) (pipeline.StageRecord, error) {
	return e.records[s], nil
}
func compositionModel() openrouter.Model {
	m := openrouter.Model{ID: "synthetic-model", ContextLength: 32768, SupportedParameters: []string{"max_tokens"}}
	contextLimit, completionLimit := 32768, 2048
	m.TopProvider.ContextLength = &contextLimit
	m.TopProvider.MaxCompletionTokens = &completionLimit
	m.Architecture.Input = []string{"text"}
	m.Architecture.Output = []string{"text"}
	return m
}
func (e *compositionExecution) Dispatch(ctx context.Context, s pipeline.Stage, upstream pipeline.CheckpointReceipt) (pipeline.Candidate, error) {
	var data []byte
	var duration time.Duration
	var err error
	role, mime := "script", "text/plain"
	switch s {
	case pipeline.Rewrite:
		e.rewriteResult, err = briefing.Rewrite(ctx, e.provider, "synthetic-key", compositionModel(), e.source, briefing.TokenBudget{MaxCompletionTokens: 1024, CountPromptTokens: func(_, _, _ string) (int, error) { return 100, nil }})
		data = []byte(e.rewriteResult.Script)
	case pipeline.Speech:
		r, ok := upstream.(*compositionReceipt)
		if !ok || r != e.committed[pipeline.Rewrite] {
			return nil, pipeline.ErrInvalid
		}
		e.speechResult, err = briefing.Synthesize(ctx, e.provider, "synthetic-key", "synthetic-model", "synthetic-voice", string(r.candidate.artifact.Bytes), time.Second, 1<<20)
		data, duration = e.speechResult.Audio, e.speechResult.Duration
		role, mime = "audio", "audio/mpeg"
	default:
		return nil, pipeline.ErrInvalid
	}
	if err != nil {
		e.originalError = err
		// Synthetic UNKNOWN is never reset to Undispatched. This is not durable
		// native reconciliation or a financial settlement decision.
		e.records[s] = pipeline.StageRecord{Status: pipeline.Unresolved}
		return nil, err
	}
	digest := sha256.Sum256(data)
	// These stand-in projection digests merely satisfy LOCAL shape guards. They
	// are NOT registered canonical request/result/manifest digests or proof.
	c := &compositionCandidate{binding: pipeline.Binding{Work: e.work, Stage: s, Attempt: "synthetic-original-" + string(s), Profile: "synthetic", Registration: "synthetic", RequestCodec: "synthetic", ResultCodec: "synthetic", RequestDigest: sha256.Sum256([]byte("synthetic-request-" + string(s))), ResultDigest: digest, ManifestDigest: digest}, artifact: pipeline.Artifact{ID: "synthetic-" + string(s), Role: role, MediaType: mime, Digest: digest, Bytes: data}, duration: duration}
	e.records[s] = pipeline.StageRecord{Status: pipeline.Accepted, Candidate: c}
	return c, nil
}
func (e *compositionExecution) Checkpoint(_ context.Context, candidate pipeline.Candidate) (pipeline.CheckpointReceipt, error) {
	c, ok := candidate.(*compositionCandidate)
	if !ok {
		return nil, pipeline.ErrInvalid
	}
	stage := c.binding.Stage
	e.checkpointCalls[stage]++
	if e.checkpointLoss == stage {
		e.records[stage] = pipeline.StageRecord{Status: pipeline.Unresolved}
		return nil, pipeline.ErrUnresolved
	}
	r := &compositionReceipt{candidate: *c}
	r.candidate.artifact.Bytes = bytes.Clone(c.artifact.Bytes)
	e.committed[stage] = r
	e.records[stage] = pipeline.StageRecord{Status: pipeline.Checkpointed, Receipt: r}
	return r, nil
}
func (e *compositionExecution) ValidateCheckpoint(_ context.Context, receipt pipeline.CheckpointReceipt) error {
	r, ok := receipt.(*compositionReceipt)
	if !ok || e.committed[r.Projection().Stage] != r {
		return pipeline.ErrInvalid
	}
	return nil
}
func (e *compositionExecution) Publish(ctx context.Context, rewrite, speech pipeline.CheckpointReceipt) (pipeline.PublicationReceipt, error) {
	if err := e.ValidateCheckpoint(ctx, rewrite); err != nil {
		return nil, err
	}
	if err := e.ValidateCheckpoint(ctx, speech); err != nil {
		return nil, err
	}
	e.publishCalls++
	a, b := e.committed[pipeline.Rewrite], e.committed[pipeline.Speech]
	e.publication = &compositionPublication{work: e.work, script: bytes.Clone(a.candidate.artifact.Bytes), audio: bytes.Clone(b.candidate.artifact.Bytes), duration: b.candidate.duration}
	return e.publication, nil
}

func TestOrdinaryCompositionPreservesValidatedArtifacts(t *testing.T) {
	t.Parallel()
	e := compositionFixture(t)
	wantAudio := bytes.Clone(e.provider.audio)
	wantScript := e.provider.script
	receipt, err := pipeline.Run(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	if receipt != e.publication || e.provider.rewriteCalls != 1 || e.provider.speechCalls != 1 || e.publishCalls != 1 {
		t.Fatal("ordinary composition did not run once per stage")
	}
	if e.provider.sourceSeen != e.source.Text || e.provider.scriptSeen != wantScript || !bytes.Equal(e.publication.script, []byte(wantScript)) || !bytes.Equal(e.publication.audio, wantAudio) {
		t.Fatal("source/script/audio changed across component boundary")
	}
	scriptHash, audioHash := sha256.Sum256([]byte(wantScript)), sha256.Sum256(wantAudio)
	if e.committed[pipeline.Rewrite].candidate.artifact.Digest != scriptHash || e.committed[pipeline.Speech].candidate.artifact.Digest != audioHash || e.speechResult.SHA256 != audioHash {
		t.Fatal("accepted full-byte hashes changed across checkpoints")
	}
	if e.publication.duration != e.speechResult.Duration || e.publication.duration < 100*time.Millisecond || e.publication.duration > 200*time.Millisecond {
		t.Fatal("decoded MP3 duration lost or not measured")
	}
	// Provider-owned buffers cannot mutate accepted/checkpointed/published audio.
	e.provider.audio[0] ^= 1
	if !bytes.Equal(e.speechResult.Audio, wantAudio) || !bytes.Equal(e.committed[pipeline.Speech].candidate.artifact.Bytes, wantAudio) || !bytes.Equal(e.publication.audio, wantAudio) {
		t.Fatal("artifact custody aliases provider buffer")
	}
	if _, err = pipeline.Run(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if e.provider.rewriteCalls != 1 || e.provider.speechCalls != 1 || e.publishCalls != 1 {
		t.Fatal("committed replay called provider or published again")
	}
}

func TestOrdinaryCompositionRejectsProviderOutputWithoutRetry(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		change      func(*compositionExecution)
		speechCalls int
	}{
		{"rewrite post-handoff drop", func(e *compositionExecution) {
			_, e.provider.rewriteErr = openrouter.FinishAttempt(openrouter.AttemptOutcome{State: openrouter.MayHaveActed}, errors.New("synthetic connection drop"))
		}, 0},
		{"rejected script", func(e *compositionExecution) { e.provider.script = "```unusable script```" }, 0},
		{"speech post-handoff truncation", func(e *compositionExecution) {
			_, e.provider.speechErr = openrouter.FinishAttempt(openrouter.AttemptOutcome{State: openrouter.MayHaveActed}, errors.New("synthetic response truncation"))
		}, 1},
		{"invalid MP3", func(e *compositionExecution) { e.provider.audio = []byte("not MP3") }, 1},
		{"truncated MP3", func(e *compositionExecution) { e.provider.audio = e.provider.audio[:len(e.provider.audio)-1] }, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := compositionFixture(t)
			tc.change(e)
			for range 2 {
				receipt, err := pipeline.Run(context.Background(), e)
				if receipt != nil || !errors.Is(err, pipeline.ErrUnresolved) {
					t.Fatal("rejected or ambiguous output advanced")
				}
			}
			if e.provider.rewriteCalls != 1 || e.provider.speechCalls != tc.speechCalls || e.publishCalls != 0 || e.checkpointCalls[pipeline.Speech] != 0 {
				t.Fatal("failure retried or published")
			}
			if e.rewriteResult.Script == "" && e.checkpointCalls[pipeline.Rewrite] != 0 {
				t.Fatal("rejected script checkpointed")
			}
			if e.speechResult.Audio != nil {
				t.Fatal("rejected audio exposed as usable")
			}
			if e.originalError == nil {
				t.Fatal("synthetic adapter lost original failure")
			}
			switch tc.name {
			case "rewrite post-handoff drop":
				var attempt *openrouter.AttemptError
				if !errors.As(e.originalError, &attempt) || attempt.Outcome.State != openrouter.MayHaveActed || e.rewriteResult.Provider.Outcome.State != openrouter.MayHaveActed || e.rewriteResult.Provider.RequestID == "" {
					t.Fatal("rewrite uncertainty/available refs lost")
				}
			case "speech post-handoff truncation":
				var attempt *openrouter.AttemptError
				if !errors.As(e.originalError, &attempt) || attempt.Outcome.State != openrouter.MayHaveActed || e.speechResult.RequestID == "" || e.speechResult.GenerationID == "" {
					t.Fatal("speech uncertainty/available refs lost")
				}
			}
		})
	}
}

func TestOrdinaryCompositionCheckpointUnknownStopsPaidReplay(t *testing.T) {
	t.Parallel()
	for _, stage := range []pipeline.Stage{pipeline.Rewrite, pipeline.Speech} {
		t.Run(string(stage), func(t *testing.T) {
			t.Parallel()
			e := compositionFixture(t)
			e.checkpointLoss = stage
			for range 2 {
				if _, err := pipeline.Run(context.Background(), e); !errors.Is(err, pipeline.ErrUnresolved) {
					t.Fatal("unknown checkpoint advanced")
				}
			}
			wantSpeech := 0
			if stage == pipeline.Speech {
				wantSpeech = 1
			}
			if e.provider.rewriteCalls != 1 || e.provider.speechCalls != wantSpeech || e.checkpointCalls[stage] != 1 || e.publishCalls != 0 {
				t.Fatal("unknown checkpoint caused another paid call or publication")
			}
			if _, exists := e.committed[stage]; exists {
				t.Fatal("synthetic unknown fabricated a committed receipt")
			}
		})
	}
}
