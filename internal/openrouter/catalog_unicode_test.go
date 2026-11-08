package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// catalogUnicodeClient serves exact catalog bytes through an owned TLS fixture.
func catalogUnicodeClient(t *testing.T, raw []byte, speech bool) (*Client, *atomic.Int32) {
	t.Helper()
	calls := new(atomic.Int32)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		query := ""
		if speech {
			query = "output_modalities=speech"
		}
		if r.Method != http.MethodGet || r.URL.Path != "/models" || r.URL.RawQuery != query || r.Header.Get("Authorization") != "Bearer synthetic" {
			t.Error("catalog request binding changed")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(raw); err != nil {
			t.Errorf("write catalog fixture: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	client := New()
	client.http = server.Client()
	client.http.Timeout = 5 * time.Second
	client.base = server.URL
	t.Cleanup(client.http.CloseIdleConnections)
	return client, calls
}

// TestCatalogUnicodeAdmission refuses a whole catalog instead of adopting repaired IDs.
func TestCatalogUnicodeAdmission(t *testing.T) {
	fields := []struct{ name, prefix, suffix string }{
		{"id", `{"data":[{"id":"good","name":"Good"},{"id":"private-`, `","name":"Bad"}]}`},
		{"name", `{"data":[{"id":"good","name":"Good"},{"id":"selected","name":"private-`, `"}]}`},
		{"voice", `{"data":[{"id":"good","name":"Good"},{"id":"selected","name":"Selected","supported_voices":["private-`, `"]}]}`},
		{"ignored-value", `{"data":[{"id":"selected","name":"Selected"}],"ignored":"private-`, `"}`},
		{"ignored-key", `{"data":[{"id":"selected","name":"Selected"}],"private-`, `":true}`},
	}
	malformed := []struct{ name, encoded string }{
		{"invalid-utf8", string([]byte{0xff})},
		{"high-surrogate", `\ud800`},
		{"low-surrogate", `\udc00`},
		{"wrong-pair", `\ud800\u0061`},
	}
	for _, speech := range []bool{false, true} {
		route := "models"
		if speech {
			route = "speech-models"
		}
		for _, field := range fields {
			for _, bad := range malformed {
				t.Run(route+"/"+field.name+"/"+bad.name, func(t *testing.T) {
					client, calls := catalogUnicodeClient(t, []byte(field.prefix+bad.encoded+field.suffix), speech)
					var models []Model
					var err error
					if speech {
						models, err = client.SpeechModels(context.Background(), "synthetic")
					} else {
						models, err = client.Models(context.Background(), "synthetic")
					}
					if err == nil || err.Error() != "invalid model catalog" || models != nil {
						t.Errorf("lossy catalog admitted: rows=%d error=%v", len(models), err)
					}
					if got := calls.Load(); got != 1 {
						t.Errorf("catalog made %d requests", got)
					}
				})
			}
		}
	}
}

// TestCatalogValidUnicodePreserved retains legitimate metadata and empty catalogs.
func TestCatalogValidUnicodePreserved(t *testing.T) {
	controls := []struct{ name, encoded, value string }{
		{"literal-replacement", "\ufffd", "\ufffd"},
		{"escaped-replacement", `\ufffd`, "\ufffd"},
		{"paired-escape", `\ud83d\ude80`, "\U0001f680"},
		{"literal-non-bmp", "\U0001f680", "\U0001f680"},
		{"backslash-lookalike", `\\ud800`, `\ud800`},
		{"empty-catalog", "", ""},
	}
	for _, speech := range []bool{false, true} {
		route := "models"
		if speech {
			route = "speech-models"
		}
		for _, tc := range controls {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				raw := []byte(`{"data":[{"id":"id-` + tc.encoded + `","name":"name-` + tc.encoded + `","supported_voices":["voice-` + tc.encoded + `"]}]}`)
				if tc.name == "empty-catalog" {
					raw = []byte(`{"data":[]}`)
				}
				client, calls := catalogUnicodeClient(t, raw, speech)
				var models []Model
				var err error
				if speech {
					models, err = client.SpeechModels(context.Background(), "synthetic")
				} else {
					models, err = client.Models(context.Background(), "synthetic")
				}
				if err != nil {
					t.Fatalf("valid catalog rejected: %v", err)
				}
				if tc.name == "empty-catalog" {
					if len(models) != 0 {
						t.Fatal("empty catalog changed")
					}
				} else if len(models) != 1 || models[0].ID != "id-"+tc.value || models[0].Name != "name-"+tc.value || len(models[0].Voices) != 1 || models[0].Voices[0] != "voice-"+tc.value {
					t.Fatal("valid catalog metadata changed")
				}
				if got := calls.Load(); got != 1 {
					t.Errorf("catalog made %d requests", got)
				}
			})
		}
	}
}
