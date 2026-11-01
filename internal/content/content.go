// Package content defines transport-independent submission invariants.
// Selectors in these structures are not evidence of authorization.
package content

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/narrate-it/prosody/internal/strictjson"
)

const MaxNarrationDuration = 10 * time.Minute

var ErrInvalid = errors.New("invalid content")

type TextSubmission struct {
	WorkspaceID    string `json:"workspaceId"`
	FeedID         string `json:"feedId"`
	IdempotencyKey string `json:"idempotencyKey"`
	Text           string `json:"text"`
	Title          string `json:"title,omitempty"`
}

func bounded(s string, max int) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= max
}

func (s TextSubmission) Validate() error {
	for name, value := range map[string]string{"workspaceId": s.WorkspaceID, "feedId": s.FeedID} {
		if !bounded(value, 128) {
			return fmt.Errorf("%w: %s", ErrInvalid, name)
		}
	}
	return s.ValidatePayload()
}

// ValidatePayload checks content before a transport resolves trusted workspace context.
func (s TextSubmission) ValidatePayload() error {
	// AMOS bounds decoded UTF-8 bytes, not JSON Schema character counts.
	if !utf8.ValidString(s.IdempotencyKey) || strings.TrimSpace(s.IdempotencyKey) == "" || len(s.IdempotencyKey) > 256 {
		return fmt.Errorf("%w: idempotencyKey", ErrInvalid)
	}
	if !bounded(s.Text, 4096) {
		return fmt.Errorf("%w: text", ErrInvalid)
	}
	if s.Title != "" && !bounded(s.Title, 160) {
		return fmt.Errorf("%w: title", ErrInvalid)
	}
	return nil
}

// DecodeText rejects unknown fields, multiple JSON values and duplicate keys.
// The transport must additionally enforce a finite HTTP body limit.
func DecodeText(r io.Reader) (TextSubmission, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 65537))
	if err != nil || len(raw) > 65536 || strictjson.Validate(raw) != nil {
		return TextSubmission{}, ErrInvalid
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return TextSubmission{}, ErrInvalid
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		t, err = d.Token()
		if err != nil {
			return TextSubmission{}, ErrInvalid
		}
		key, ok := t.(string)
		if !ok {
			return TextSubmission{}, ErrInvalid
		}
		switch key {
		case "workspaceId", "feedId", "idempotencyKey", "text", "title":
		default:
			return TextSubmission{}, ErrInvalid
		}
		if _, exists := fields[key]; exists {
			return TextSubmission{}, fmt.Errorf("%w: duplicate field", ErrInvalid)
		}
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return TextSubmission{}, ErrInvalid
		}
		if string(value) == "null" || key == "title" && string(value) == `""` {
			return TextSubmission{}, ErrInvalid
		}
		fields[key] = value
	}
	if _, err = d.Token(); err != nil {
		return TextSubmission{}, ErrInvalid
	}
	if _, err = d.Token(); err != io.EOF {
		return TextSubmission{}, ErrInvalid
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return TextSubmission{}, ErrInvalid
	}
	d = json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	var s TextSubmission
	if err = d.Decode(&s); err != nil {
		return s, ErrInvalid
	}
	return s, s.Validate()
}

// Digest hashes a canonical typed payload; original text is never trimmed or rewritten.
func (s TextSubmission) Digest() ([32]byte, error) {
	if err := s.Validate(); err != nil {
		return [32]byte{}, err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(b), nil
}

type SegmentKind string

const (
	Narration SegmentKind = "narration"
	Audio     SegmentKind = "audio"
	Video     SegmentKind = "video"
	Image     SegmentKind = "image"
)

type Segment struct {
	Kind     SegmentKind `json:"kind"`
	Text     string      `json:"text,omitempty"`
	AssetIDs []string    `json:"assetIds,omitempty"`
}
type Manifest struct {
	Version  int       `json:"version"`
	Segments []Segment `json:"segments"`
}
type Limits struct{ MaxSegments, MaxTextRunes, MaxAssets int }

// Validate checks structure only. Asset ownership, detected media type and current
// destination authority must be checked transactionally by the application.
func (m Manifest) Validate(l Limits) error {
	if l.MaxSegments < 1 || l.MaxTextRunes < 1 || l.MaxAssets < 1 {
		return fmt.Errorf("%w: missing limits", ErrInvalid)
	}
	if m.Version != 1 || len(m.Segments) == 0 || len(m.Segments) > l.MaxSegments {
		return ErrInvalid
	}
	textRunes, assets := 0, 0
	for _, s := range m.Segments {
		if !utf8.ValidString(s.Text) {
			return ErrInvalid
		}
		textRunes += utf8.RuneCountInString(s.Text)
		assets += len(s.AssetIDs)
		if textRunes > l.MaxTextRunes || assets > l.MaxAssets {
			return ErrInvalid
		}
		for _, id := range s.AssetIDs {
			if !bounded(id, 128) {
				return ErrInvalid
			}
		}
		switch s.Kind {
		case Narration:
			if strings.TrimSpace(s.Text) == "" {
				return ErrInvalid
			}
		case Audio, Video, Image:
			if s.Text != "" || len(s.AssetIDs) != 1 {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	return nil
}

// ValidateNarrationDuration applies only to generated narration, summed across
// the entire item. Supplied media never enters this budget.
func ValidateNarrationDuration(parts []time.Duration) error {
	if len(parts) == 0 {
		return ErrInvalid
	}
	var total time.Duration
	for _, part := range parts {
		if part <= 0 || part > MaxNarrationDuration-total {
			return fmt.Errorf("%w: narration duration", ErrInvalid)
		}
		total += part
	}
	return nil
}
