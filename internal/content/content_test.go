package content

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIdempotencyKeyByteLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		valid bool
	}{
		{"empty", "", false},
		{"one byte", "k", true},
		{"ascii boundary", strings.Repeat("a", 256), true},
		{"ascii overflow", strings.Repeat("a", 257), false},
		{"unicode boundary", strings.Repeat("界", 85) + "a", true},
		{"unicode overflow", strings.Repeat("界", 86), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := TextSubmission{WorkspaceID: "w", FeedID: "f", IdempotencyKey: tc.key, Text: "source"}
			if err := s.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate: valid=%t, error=%v", tc.valid, err)
			}
			body, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeText(strings.NewReader(string(body))); (err == nil) != tc.valid {
				t.Fatalf("DecodeText: valid=%t, error=%v", tc.valid, err)
			}
		})
	}
	s := TextSubmission{WorkspaceID: "w", FeedID: "f", IdempotencyKey: string([]byte{255}), Text: "source"}
	if s.Validate() == nil {
		t.Fatal("accepted invalid UTF-8 key")
	}
}

func TestDecodeStrictSubmission(t *testing.T) {
	valid := `{"workspaceId":"w","feedId":"f","idempotencyKey":"k","text":" Original text. "}`
	s, err := DecodeText(strings.NewReader(valid))
	if err != nil || s.Text != " Original text. " {
		t.Fatalf("source not preserved: %v", err)
	}
	for _, input := range []string{
		strings.Replace(valid, `"text":`, `"Text":`, 1),
		strings.Replace(valid, `"text":`, `"title":"","text":`, 1),
		strings.Replace(valid, `"k"`, `"\ud800"`, 1),
		strings.Replace(valid, `"k"`, `"\udc00"`, 1),
		strings.Replace(valid, "Original", string([]byte{255}), 1),
		valid + ` {}`,
		strings.Replace(valid, `"text":`, `"role":"owner","text":`, 1),
		strings.Replace(valid, `"text":`, `"text":"other","text":`, 1),
		strings.Replace(valid, `" Original text. "`, `null`, 1),
		strings.Replace(valid, `" Original text. "`, `" "`, 1),
	} {
		if _, err := DecodeText(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid input: %s", input)
		}
	}
}
func TestDigestPreservesSourceAndIgnoresFieldOrder(t *testing.T) {
	a, _ := DecodeText(strings.NewReader(`{"workspaceId":"w","feedId":"f","idempotencyKey":"k","text":"a"}`))
	b, _ := DecodeText(strings.NewReader(`{"text":"a","feedId":"f","workspaceId":"w","idempotencyKey":"k"}`))
	da, _ := a.Digest()
	db, _ := b.Digest()
	if da != db {
		t.Fatal("field order changed identity")
	}
	b.Text = "a "
	db, _ = b.Digest()
	if da == db {
		t.Fatal("source change lost")
	}
}
func TestNarrationLimitIsAggregate(t *testing.T) {
	for _, parts := range [][]time.Duration{nil, {0}, {-1}, {6 * time.Minute, 5 * time.Minute}, {MaxNarrationDuration + 1}} {
		if ValidateNarrationDuration(parts) == nil {
			t.Fatalf("accepted %v", parts)
		}
	}
	if err := ValidateNarrationDuration([]time.Duration{4 * time.Minute, 6 * time.Minute}); err != nil {
		t.Fatal(err)
	}
}
func TestMixedManifest(t *testing.T) {
	m := Manifest{Version: 1, Segments: []Segment{{Kind: Narration, Text: "An explanation.", AssetIDs: []string{"image"}}, {Kind: Video, AssetIDs: []string{"clip"}}}}
	l := Limits{MaxSegments: 3, MaxTextRunes: 100, MaxAssets: 3}
	if err := m.Validate(l); err != nil {
		t.Fatal(err)
	}
	m.Segments[1].Text = "Rewrite this video"
	if m.Validate(l) == nil {
		t.Fatal("accepted text on supplied video segment")
	}
	if m.Validate(Limits{}) == nil {
		t.Fatal("accepted missing admission limits")
	}
}
