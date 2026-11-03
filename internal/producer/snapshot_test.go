package producer

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"
)

func fixture() Snapshot {
	return Snapshot{Version: Version, Profile: Profile, ProfileRevision: ProfileRevision,
		LabelSource: OwnerAssignedConnectionLabel, OriginalClientID: "https://agent.example/client-metadata.json", Label: "Research assistant",
		Scope: Scope{WorkspaceID: "01900000-0000-7000-8000-000000000001", SourceItemID: "01900000-0000-7000-8000-000000000002", OriginalInvocationID: "01900000-0000-7000-8000-000000000003", WorkID: "01900000-0000-7000-8000-000000000004"}, CapturedAt: time.Date(2026, 11, 3, 1, 2, 3, 400, time.UTC)}
}

func TestCanonicalSnapshotRoundTrip(t *testing.T) {
	t.Parallel()
	for _, class := range []LabelSource{OwnerAssignedConnectionLabel, ClientAdvertisedApplicationName} {
		t.Run(string(class), func(t *testing.T) {
			value := fixture()
			value.LabelSource = class
			raw, digest, err := Encode(value)
			if err != nil || sha256.Sum256(raw) != digest {
				t.Fatalf("encode: %v", err)
			}
			got, err := Decode(raw, digest)
			if err != nil || got != value {
				t.Fatalf("round trip: %v", err)
			}
			again, other, err := Encode(value)
			if err != nil || !bytes.Equal(raw, again) || digest != other {
				t.Fatal("nondeterministic snapshot")
			}
		})
	}
}

func TestSnapshotRejectsMissingOrInvalidCaptureData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{"zero version", func(s *Snapshot) { s.Version = 0 }},
		{"unknown profile", func(s *Snapshot) { s.Profile = "other" }},
		{"unknown revision", func(s *Snapshot) { s.ProfileRevision = 2 }},
		{"no source class", func(s *Snapshot) { s.LabelSource = "" }},
		{"unknown source class", func(s *Snapshot) { s.LabelSource = "agent-verified" }},
		{"no label despite client", func(s *Snapshot) { s.Label = "" }},
		{"blank label", func(s *Snapshot) { s.Label = "  " }},
		{"byte oversized label", func(s *Snapshot) { s.Label = strings.Repeat("界", 85) + "ab" }},
		{"bad UTF8", func(s *Snapshot) { s.Label = string([]byte{0xff}) }},
		{"NUL label", func(s *Snapshot) { s.Label = "private\x00label" }},
		{"line feed", func(s *Snapshot) { s.Label = "first\nsecond" }},
		{"C1 control", func(s *Snapshot) { s.Label = "name\u0085" }},
		{"line separator", func(s *Snapshot) { s.Label = "name\u2028" }},
		{"no client", func(s *Snapshot) { s.OriginalClientID = "" }},
		{"oversized client", func(s *Snapshot) { s.OriginalClientID = strings.Repeat("c", MaxClientIDBytes+1) }},
		{"client control", func(s *Snapshot) { s.OriginalClientID = "client\r" }},
		{"missing workspace", func(s *Snapshot) { s.Scope.WorkspaceID = "" }},
		{"invalid item", func(s *Snapshot) { s.Scope.SourceItemID = "item" }},
		{"invalid invocation", func(s *Snapshot) { s.Scope.OriginalInvocationID = "invocation" }},
		{"invalid optional work", func(s *Snapshot) { s.Scope.WorkID = "work" }},
		{"no timestamp", func(s *Snapshot) { s.CapturedAt = time.Time{} }},
		{"timestamp before profile range", func(s *Snapshot) { s.CapturedAt = time.Date(1969, 1, 1, 0, 0, 0, 0, time.UTC) }},
		{"timestamp overflow", func(s *Snapshot) { s.CapturedAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := fixture()
			tc.mutate(&value)
			raw, digest, err := Encode(value)
			if !errors.Is(err, ErrInvalid) || len(raw) != 0 || digest != [32]byte{} {
				t.Fatal("invalid data encoded or fallback supplied")
			}
		})
	}
}

func TestSnapshotLimitsAndUTCCanonicalization(t *testing.T) {
	value := fixture()
	value.Label = strings.Repeat("界", 85) + "a"
	value.OriginalClientID = strings.Repeat("c", MaxClientIDBytes)
	value.Scope.WorkID = ""
	value.CapturedAt = value.CapturedAt.In(time.FixedZone("offset", 3600))
	raw, digest, err := Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(raw, digest)
	if err != nil || len(got.Label) != MaxLabelBytes || len(got.OriginalClientID) != MaxClientIDBytes || got.Scope.WorkID != "" || got.CapturedAt.Location() != time.UTC || !got.CapturedAt.Equal(value.CapturedAt) {
		t.Fatal("boundary bytes or capture instant changed")
	}
}

func TestSnapshotRejectsTamperingAndNoncanonicalJSON(t *testing.T) {
	raw, digest, err := Encode(fixture())
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte("Research assistant"), []byte("Different producer"), 1)
	if got, err := Decode(changed, digest); !errors.Is(err, ErrInvalid) || got != (Snapshot{}) {
		t.Fatal("tampered snapshot returned")
	}
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"duplicate root", bytes.Replace(raw, []byte(`"label":`), []byte(`"label":"first","label":`), 1)},
		{"escaped duplicate", bytes.Replace(raw, []byte(`"label":`), []byte(`"la\u0062el":"first","label":`), 1)},
		{"duplicate nested", bytes.Replace(raw, []byte(`"workspaceId":`), []byte(`"workspaceId":"other","workspaceId":`), 1)},
		{"case folded", bytes.Replace(raw, []byte(`"label":`), []byte(`"Label":`), 1)},
		{"invalid timestamp", bytes.Replace(raw, []byte(`"capturedAt":"`), []byte(`"capturedAt":"invalid-`), 1)},
		{"alternate timestamp offset", bytes.Replace(raw, []byte(`Z"`), []byte(`+00:00"`), 1)},
		{"unknown field", append([]byte(`{"authority":"pretend",`), raw[1:]...)},
		{"whitespace", append([]byte(" "), raw...)},
		{"lone surrogate", bytes.Replace(raw, []byte(`"Research assistant"`), []byte(`"\ud800"`), 1)},
		{"invalid UTF8", bytes.Replace(raw, []byte("Research assistant"), []byte{0xff}, 1)},
		{"oversized document", bytes.Repeat([]byte{' '}, MaxSnapshotBytes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode(tc.raw, sha256.Sum256(tc.raw))
			if !errors.Is(err, ErrInvalid) || got != (Snapshot{}) {
				t.Fatal("ambiguous/malformed JSON returned")
			}
		})
	}
}

func TestReplayRetainsOriginalProducerAndScope(t *testing.T) {
	original := fixture()
	raw, digest, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), raw...)
	later := original
	later.Label = "Renamed client"
	later.OriginalClientID = "different-valid-client"
	later.LabelSource = ClientAdvertisedApplicationName
	later.CapturedAt = later.CapturedAt.Add(time.Hour)
	replacement, _, err := Encode(later)
	if err != nil || bytes.Equal(raw, replacement) {
		t.Fatal("invalid changed proposal fixture")
	}
	// The API intentionally takes no replacement producer metadata.
	got, err := RetainOriginal(raw, digest, original.Scope)
	if err != nil || got != original || !bytes.Equal(raw, before) {
		t.Fatal("replay altered original capture")
	}
	for _, mutate := range []func(*Scope){func(s *Scope) { s.WorkspaceID = "01900000-0000-7000-8000-000000000009" }, func(s *Scope) { s.SourceItemID = "01900000-0000-7000-8000-000000000009" }, func(s *Scope) { s.OriginalInvocationID = "01900000-0000-7000-8000-000000000009" }, func(s *Scope) { s.WorkID = "" }} {
		scope := original.Scope
		mutate(&scope)
		if got, err := RetainOriginal(raw, digest, scope); !errors.Is(err, ErrScopeMismatch) || got != (Snapshot{}) {
			t.Fatal("foreign/changed capture scope exposed snapshot")
		}
	}
	if got, err := RetainOriginal(nil, [32]byte{}, original.Scope); !errors.Is(err, ErrInvalid) || got != (Snapshot{}) {
		t.Fatal("missing legacy snapshot fabricated a producer")
	}
}
