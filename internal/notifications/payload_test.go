package notifications_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/narrate-it/prosody/internal/notifications"
)

// target supplies nonsecret fixture identities, not an admitted publication.
func target() notifications.ReadyTarget {
	return notifications.ReadyTarget{
		FeedID:      "019a3f00-1234-7000-8000-000000000001",
		ItemID:      "019a3f00-1234-7000-8000-000000000002",
		RenditionID: "019a3f00-1234-7000-8000-000000000003",
	}
}

// TestReadyPayloadGolden freezes the local profile independently of a transport.
func TestReadyPayloadGolden(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/notifications/ready-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.TrimSpace(fixture)
	b, err := notifications.ReadyPayload(target())
	if err != nil || !bytes.Equal(b, want) {
		t.Fatalf("payload=%s error=%v", b, err)
	}
	if len(b) > 4096 {
		t.Fatal("APNs payload exceeds regular notification limit")
	}
	again, err := notifications.ReadyPayload(target())
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("nondeterministic payload: %v", err)
	}
	b[0] = 'x'
	if again[0] != '{' {
		t.Fatal("separate payload results alias mutable storage")
	}
}

// TestReadyPayloadPrivateSurface refuses accidental private-content expansion.
func TestReadyPayloadPrivateSurface(t *testing.T) {
	b, err := notifications.ReadyPayload(target())
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 {
		t.Fatal("unexpected top-level payload fields")
	}
	var aps map[string]any
	if err := json.Unmarshal(decoded["aps"], &aps); err != nil {
		t.Fatal(err)
	}
	if len(aps) != 1 || aps["alert"] != "A new briefing is ready." {
		t.Fatalf("private/extra APS fields: %v", aps)
	}
	var route map[string]any
	if err := json.Unmarshal(decoded["prosody"], &route); err != nil {
		t.Fatal(err)
	}
	if len(route) != 5 || route["version"] != float64(1) || route["kind"] != "ready" || route["feedId"] != target().FeedID || route["itemId"] != target().ItemID || route["renditionId"] != target().RenditionID {
		t.Fatalf("unexpected routing shape: %v", route)
	}
	for _, key := range []string{"title", "source", "script", "agent", "personId", "workspaceId", "deviceToken", "url", "sound", "badge", "mutable-content", "content-available"} {
		if bytes.Contains(b, []byte(`"`+key+`"`)) {
			t.Fatalf("unexpected private/behavior field %s", key)
		}
	}
}

// TestReadyPayloadInvalidTargets rejects hostile text and malformed identities
// instead of serializing them into provider-visible routing metadata.
func TestReadyPayloadInvalidTargets(t *testing.T) {
	for _, field := range []string{"feed", "item", "rendition"} {
		for _, bad := range []struct{ name, value string }{
			{"empty", ""}, {"zero", "00000000-0000-0000-0000-000000000000"},
			{"uppercase", strings.ToUpper(target().ItemID)},
			{"short", "019a3f00"}, {"separator", "019a3f00_1234-7000-8000-000000000002"},
			{"private_text", "Private source/script/agent text"},
			{"json_injection", `"},"source":"private"`},
			{"url", "https://example.invalid/private?token=secret"},
			{"nul", target().ItemID + "\x00"}, {"newline", target().ItemID + "\n"},
			{"oversize", strings.Repeat("private", 1000)},
		} {
			t.Run(field+"/"+bad.name, func(t *testing.T) {
				input := target()
				switch field {
				case "feed":
					input.FeedID = bad.value
				case "item":
					input.ItemID = bad.value
				case "rendition":
					input.RenditionID = bad.value
				}
				b, err := notifications.ReadyPayload(input)
				if b != nil || !errors.Is(err, notifications.ErrInvalidTarget) {
					t.Fatalf("accepted invalid target: bytes=%d error=%v", len(b), err)
				}
				if err.Error() != "invalid ready notification target" {
					t.Fatal("diagnostic exposes input")
				}
			})
		}
	}
	for _, bad := range []struct{ name, value string }{
		{"feed_v4", "019a3f00-1234-4000-8000-000000000001"},
		{"feed_variant", "019a3f00-1234-7000-0000-000000000001"},
	} {
		t.Run(bad.name, func(t *testing.T) {
			input := target()
			input.FeedID = bad.value
			b, err := notifications.ReadyPayload(input)
			if b != nil || !errors.Is(err, notifications.ErrInvalidTarget) {
				t.Fatalf("bytes=%d error=%v", len(b), err)
			}
		})
	}
}

// TestReadyPayloadExistingItemUUIDs preserves non-v7 item/rendition identities;
// the UUIDv7 restriction belongs only to the feed contract.
func TestReadyPayloadExistingItemUUIDs(t *testing.T) {
	input := target()
	input.ItemID = "550e8400-e29b-41d4-a716-446655440001"
	input.RenditionID = "550e8400-e29b-41d4-a716-446655440002"
	b, err := notifications.ReadyPayload(input)
	if err != nil || !bytes.Contains(b, []byte(input.ItemID)) || !bytes.Contains(b, []byte(input.RenditionID)) {
		t.Fatalf("existing item identities changed: %v", err)
	}
}
