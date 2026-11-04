package openrouter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/briefing"
	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/openrouter"
	"github.com/narrate-it/prosody/internal/pipeline"
)

const httpPrivateKey = "PRIVATE-KEY-MARKER"
const httpSourceMarker = "PRIVATE-SOURCE-MARKER"
const httpPrivateSource = httpSourceMarker + ": The first run succeeded; another is required."
const httpPrivateBody = "PRIVATE-BODY-MARKER"
const httpScript = "The first run succeeded; another run is required."

// httpCalls counts actual owned-server requests, not synthetic provider calls.
type httpCalls struct{ total, rewrite, speech atomic.Int32 }

// pipelineTLSServer serves real bounded provider HTTP exchanges. Each mode has
// its own server; no global client, transport or shared failure hook is changed.
func pipelineTLSServer(t *testing.T, mode string, audio []byte) (*httptest.Server, *httpCalls) {
	t.Helper()
	calls := &httpCalls{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.total.Add(1)
		raw, err := io.ReadAll(io.LimitReader(r.Body, 65537))
		if err != nil || len(raw) > 65536 {
			t.Error("fixture request body unreadable/oversized")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+httpPrivateKey {
			t.Error("provider request method/credential binding changed")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/api/v1/chat/completions":
			calls.rewrite.Add(1)
			var request struct {
				Model     string
				Messages  []struct{ Role, Content string }
				Stream    bool
				MaxTokens int `json:"max_tokens"`
			}
			if json.Unmarshal(raw, &request) != nil || request.Model != "synthetic-text" || len(request.Messages) != 2 || request.Messages[0].Role != "system" || !strings.Contains(request.Messages[0].Content, "Prosody briefing policy") || request.Messages[1].Role != "user" || request.Messages[1].Content != httpPrivateSource || request.Stream || request.MaxTokens != 1024 {
				t.Error("rewrite wire lost bounded source/craft shape")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if mode == "rewrite drop" {
				connection, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				if err := connection.Close(); err != nil {
					t.Error(err)
				}
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-Id", "rewrite-request")
			w.Header().Set("X-Generation-Id", "rewrite-header-generation")
			response := `{"id":"rewrite-generation","choices":[{"message":{"content":"` + httpScript + `"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10}}`
			if _, err := io.WriteString(w, response); err != nil {
				t.Error(err)
			}
		case "/api/v1/audio/speech":
			calls.speech.Add(1)
			var request struct {
				Input, Voice, Model string
				Format              string `json:"response_format"`
			}
			if json.Unmarshal(raw, &request) != nil || request.Input != httpScript || request.Model != "synthetic-speech" || request.Voice != "synthetic-voice" || request.Format != "mp3" {
				t.Error("speech did not receive exact checkpointed script/voice/format")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("X-Request-Id", "speech-request")
			w.Header().Set("X-Generation-Id", "speech-generation")
			w.Header().Set("Content-Type", "audio/mpeg")
			body := audio
			switch mode {
			case "speech truncated":
				w.Header().Set("Content-Length", strconv.Itoa(len(audio)+17))
			case "invalid MP3":
				body = []byte(httpPrivateBody)
			case "invalid MIME":
				w.Header().Set("Content-Type", "text/plain")
			}
			if _, err := w.Write(body); err != nil {
				t.Error(err)
			}
		default:
			t.Error("unexpected provider path")
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, calls
}

// httpArtifact is a SYNTHETIC test-only candidate/receipt/publication token.
// Its projection hashes are shape stand-ins, not native codecs or authority.
type httpArtifact struct {
	binding  pipeline.Binding
	artifact pipeline.Artifact
	duration time.Duration
}

// Projection exposes only the fixture's descriptive stage bindings.
func (a *httpArtifact) Projection() pipeline.Binding { return a.binding }

// Artifacts supplies actual validated script/audio bytes to local shape guards.
func (a *httpArtifact) Artifacts() []pipeline.Artifact { return []pipeline.Artifact{a.artifact} }

// Work exposes the synthetic immutable work projection for publication replay.
func (a *httpArtifact) Work() pipeline.Work { return a.binding.Work }

// httpExecution keeps ephemeral TEST-ONLY state around real briefing calls. It
// is not native acceptance, a durable journal, a scheduler or settlement proof.
type httpExecution struct {
	client      *openrouter.Client
	work        pipeline.Work
	key         string
	records     map[pipeline.Stage]pipeline.StageRecord
	receipts    map[pipeline.Stage]*httpArtifact
	publication *httpArtifact
	checkpoints map[pipeline.Stage]int
	publishes   int
	rewrite     briefing.Result
	speech      briefing.Synthesized
	stageErr    error
}

// newHTTPExecution builds local port fixtures without creating native receipts.
func newHTTPExecution(t *testing.T, client *openrouter.Client) *httpExecution {
	t.Helper()
	source, err := json.Marshal(httpSource())
	if err != nil {
		t.Fatal(err)
	}
	return &httpExecution{client: client, key: httpPrivateKey,
		work:    pipeline.Work{Realm: "synthetic", Workspace: "synthetic-workspace", Job: "synthetic-original-job", Invocation: "synthetic-original-invocation", Feed: "synthetic-feed", Item: "synthetic-item", SettingsRevision: "synthetic-revision", SourceDigest: sha256.Sum256(source), SettingsDigest: sha256.Sum256([]byte("synthetic-settings")), PayloadDigest: sha256.Sum256([]byte("synthetic-payload"))},
		records: map[pipeline.Stage]pipeline.StageRecord{pipeline.Rewrite: {Status: pipeline.Undispatched}, pipeline.Speech: {Status: pipeline.Undispatched}}, receipts: map[pipeline.Stage]*httpArtifact{}, checkpoints: map[pipeline.Stage]int{}}
}

// httpSource keeps private input markers out of fixture error messages.
func httpSource() content.TextSubmission {
	return content.TextSubmission{WorkspaceID: "synthetic-workspace", FeedID: "synthetic-feed", IdempotencyKey: "synthetic-only", Text: httpPrivateSource}
}

// httpTextModel supplies synthetic current token limits; it is not a model claim.
func httpTextModel() openrouter.Model {
	m := openrouter.Model{ID: "synthetic-text", ContextLength: 8192, SupportedParameters: []string{"max_tokens"}}
	contextLimit, completionLimit := 8192, 2048
	m.TopProvider.ContextLength = &contextLimit
	m.TopProvider.MaxCompletionTokens = &completionLimit
	m.Architecture.Input = []string{"text"}
	m.Architecture.Output = []string{"text"}
	return m
}

// Work returns the fixture's fixed descriptive identity, never authority.
func (e *httpExecution) Work() pipeline.Work { return e.work }

// LookupPublication returns only same-process synthetic publication state.
func (e *httpExecution) LookupPublication(context.Context) (pipeline.PublicationRecord, error) {
	if e.publication != nil {
		return pipeline.PublicationRecord{Status: pipeline.PublicationCommitted, Receipt: e.publication}, nil
	}
	return pipeline.PublicationRecord{Status: pipeline.PublicationAbsent}, nil
}

// ValidatePublication checks fixture token identity without native validation.
func (e *httpExecution) ValidatePublication(_ context.Context, r pipeline.PublicationReceipt) error {
	if r != e.publication || e.publication == nil {
		return pipeline.ErrInvalid
	}
	return nil
}

// LookupStage preserves synthetic UNKNOWN after a failed HTTP/briefing call.
func (e *httpExecution) LookupStage(_ context.Context, s pipeline.Stage) (pipeline.StageRecord, error) {
	return e.records[s], nil
}

// Dispatch uses the real HTTP client, briefing craft and MP3 validation. Fixture
// candidate creation on success does NOT turn raw observations into settlement.
func (e *httpExecution) Dispatch(ctx context.Context, s pipeline.Stage, upstream pipeline.CheckpointReceipt) (pipeline.Candidate, error) {
	var data []byte
	var duration time.Duration
	var err error
	role, mime := "script", "text/plain"
	switch s {
	case pipeline.Rewrite:
		e.rewrite, err = briefing.Rewrite(ctx, e.client, e.key, httpTextModel(), httpSource(), briefing.TokenBudget{MaxCompletionTokens: 1024, CountPromptTokens: func(_, _, _ string) (int, error) { return 100, nil }})
		data = []byte(e.rewrite.Script)
	case pipeline.Speech:
		if upstream != e.receipts[pipeline.Rewrite] || upstream == nil {
			return nil, pipeline.ErrInvalid
		}
		e.speech, err = briefing.Synthesize(ctx, e.client, e.key, "synthetic-speech", "synthetic-voice", string(e.receipts[pipeline.Rewrite].artifact.Bytes), time.Second, 1<<20)
		data, duration = e.speech.Audio, e.speech.Duration
		role, mime = "audio", "audio/mpeg"
	default:
		return nil, pipeline.ErrInvalid
	}
	if err != nil {
		e.stageErr = err
		e.records[s] = pipeline.StageRecord{Status: pipeline.Unresolved}
		return nil, err
	}
	digest := sha256.Sum256(data)
	a := &httpArtifact{binding: pipeline.Binding{Work: e.work, Stage: s, Attempt: "synthetic-original-" + string(s), Profile: "synthetic", Registration: "synthetic", RequestCodec: "synthetic", ResultCodec: "synthetic", RequestDigest: sha256.Sum256([]byte("synthetic-request-" + string(s))), ResultDigest: digest, ManifestDigest: digest}, artifact: pipeline.Artifact{ID: "synthetic-" + string(s), Role: role, MediaType: mime, Bytes: data, Digest: digest}, duration: duration}
	e.records[s] = pipeline.StageRecord{Status: pipeline.Accepted, Candidate: a}
	return a, nil
}

// Checkpoint clones bytes into an ephemeral synthetic receipt, not durable SQL.
func (e *httpExecution) Checkpoint(_ context.Context, c pipeline.Candidate) (pipeline.CheckpointReceipt, error) {
	a, ok := c.(*httpArtifact)
	if !ok {
		return nil, pipeline.ErrInvalid
	}
	r := *a
	r.artifact.Bytes = bytes.Clone(a.artifact.Bytes)
	stage := a.binding.Stage
	e.checkpoints[stage]++
	e.receipts[stage] = &r
	e.records[stage] = pipeline.StageRecord{Status: pipeline.Checkpointed, Receipt: &r}
	return &r, nil
}

// ValidateCheckpoint checks same-process fixture identity, not held admission.
func (e *httpExecution) ValidateCheckpoint(_ context.Context, r pipeline.CheckpointReceipt) error {
	a, ok := r.(*httpArtifact)
	if !ok || e.receipts[a.binding.Stage] != a {
		return pipeline.ErrInvalid
	}
	return nil
}

// Publish records fixture output only after both synthetic receipts are present.
func (e *httpExecution) Publish(ctx context.Context, rewrite, speech pipeline.CheckpointReceipt) (pipeline.PublicationReceipt, error) {
	if err := e.ValidateCheckpoint(ctx, rewrite); err != nil {
		return nil, err
	}
	if err := e.ValidateCheckpoint(ctx, speech); err != nil {
		return nil, err
	}
	e.publishes++
	p := *e.receipts[pipeline.Speech]
	p.artifact.Bytes = bytes.Clone(p.artifact.Bytes)
	e.publication = &p
	return &p, nil
}

// TestProviderHTTPBriefingPipelineComposition crosses actual TLS/HTTP, provider
// parsing, briefing validation and local pipeline barriers without live calls.
func TestProviderHTTPBriefingPipelineComposition(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		rewrites, speeches int
		state              openrouter.AttemptState
	}{
		{"valid", 1, 1, openrouter.ResponseReceived}, {"rewrite drop", 1, 0, openrouter.MayHaveActed}, {"speech truncated", 1, 1, openrouter.MayHaveActed}, {"invalid MP3", 1, 1, openrouter.ResponseReceived}, {"invalid MIME", 1, 1, openrouter.ResponseReceived}, {"preflight", 0, 0, openrouter.NotDispatched},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			audio, err := os.ReadFile("../briefing/testdata/silence.mp3")
			if err != nil {
				t.Fatal(err)
			}
			server, calls := pipelineTLSServer(t, tc.name, audio)
			execution := newHTTPExecution(t, openrouter.NewClientForTLSServer(t, server))
			if tc.name == "preflight" {
				execution.key = ""
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			receipt, err := pipeline.Run(ctx, execution)
			if tc.speeches == 1 {
				if execution.rewrite.Provider.RequestID != "rewrite-request" || execution.rewrite.Provider.GenerationID != "rewrite-generation" || execution.speech.RequestID != "speech-request" || execution.speech.GenerationID != "speech-generation" {
					t.Fatal("available provider references lost across briefing")
				}
			}
			if tc.name == "valid" {
				if err != nil || receipt != execution.publication {
					t.Fatalf("valid TLS pipeline failed: %v", err)
				}
				if execution.rewrite.Provider.Outcome.State != openrouter.ResponseReceived || execution.speech.Outcome.State != openrouter.ResponseReceived {
					t.Fatal("complete HTTP observation changed")
				}
				if !bytes.Equal(execution.receipts[pipeline.Rewrite].artifact.Bytes, []byte(httpScript)) || execution.receipts[pipeline.Rewrite].artifact.Digest != sha256.Sum256([]byte(httpScript)) {
					t.Fatal("checkpointed script bytes/hash changed")
				}
				if execution.rewrite.Script != httpScript || execution.checkpoints[pipeline.Rewrite] != 1 || execution.checkpoints[pipeline.Speech] != 1 || execution.publishes != 1 {
					t.Fatal("valid stages did not checkpoint/publish once")
				}
				if !bytes.Equal(execution.publication.artifact.Bytes, audio) || execution.publication.artifact.Digest != sha256.Sum256(audio) || execution.speech.SHA256 != sha256.Sum256(audio) || execution.publication.duration != execution.speech.Duration || execution.speech.Duration < 100*time.Millisecond || execution.speech.Duration > 200*time.Millisecond {
					t.Fatal("actual MP3 bytes/hash/decoded duration changed")
				}
			} else {
				if !errors.Is(err, pipeline.ErrUnresolved) || receipt != nil || execution.publishes != 0 || execution.checkpoints[pipeline.Speech] != 0 || len(execution.speech.Audio) != 0 {
					t.Fatal("HTTP/format failure advanced or exposed usable audio")
				}
				observation := execution.speech.Outcome
				requestID, generationID := execution.speech.RequestID, execution.speech.GenerationID
				if tc.speeches == 0 {
					observation = execution.rewrite.Provider.Outcome
					requestID, generationID = execution.rewrite.Provider.RequestID, execution.rewrite.Provider.GenerationID
					if execution.rewrite.Script != "" || execution.rewrite.Provider.Text != "" || execution.checkpoints[pipeline.Rewrite] != 0 {
						t.Fatal("failed rewrite exposed/checkpointed output")
					}
				}
				if observation.State != tc.state {
					t.Fatalf("raw observation=%s want=%s", observation.State, tc.state)
				}
				var attempt *openrouter.AttemptError
				if !errors.As(execution.stageErr, &attempt) || attempt == nil || attempt.Outcome.State != observation.State || attempt.Outcome.RequestID != requestID || attempt.Outcome.GenerationID != generationID || observation.RequestID != requestID || observation.GenerationID != generationID {
					t.Fatal("briefing error lost typed raw observation or available references")
				}
				assertSafeHTTPFailure(t, err)
				assertSafeHTTPFailure(t, execution.stageErr)
				if tc.name == "speech truncated" {
					if !errors.Is(execution.stageErr, io.ErrUnexpectedEOF) || execution.speech.RequestID != "speech-request" || execution.speech.GenerationID != "speech-generation" {
						t.Fatal("truncation cause/header references lost")
					}
				}
			}
			// Same-process state prevents replay; it is not durable/native reconciliation.
			_, secondErr := pipeline.Run(ctx, execution)
			if tc.name == "valid" {
				if secondErr != nil || execution.publishes != 1 {
					t.Fatal("publication replay changed")
				}
			} else if !errors.Is(secondErr, pipeline.ErrUnresolved) {
				t.Fatal("UNKNOWN replay did not stop")
			}
			if int(calls.total.Load()) != tc.rewrites+tc.speeches || int(calls.rewrite.Load()) != tc.rewrites || int(calls.speech.Load()) != tc.speeches {
				t.Fatal("HTTP request retried or stage call count changed")
			}
		})
	}
}

// assertSafeHTTPFailure checks renderings only; callers must not log raw causes.
func assertSafeHTTPFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected diagnostic")
	}
	for _, marker := range []string{httpPrivateKey, httpSourceMarker, httpPrivateBody} {
		if strings.Contains(err.Error(), marker) {
			t.Error("private key/source/body marker exposed in error")
		}
	}
}
