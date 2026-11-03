package presentation_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/narrate-it/prosody/internal/presentation"
	"github.com/narrate-it/prosody/internal/store"
)

const workspace = "00000000-0000-7000-8000-000000000001"
const feedID = "00000000-0000-7000-8000-000000000002"
const origin = "https://media.example.invalid"

func uuid(n int) string { return fmt.Sprintf("00000000-0000-7000-8000-%012d", n) }
func fixture() (presentation.Scope, store.BriefingReadPage, presentation.BriefingInputs) {
	scope := presentation.Scope{WorkspaceID: workspace, FeedID: feedID}
	id := presentation.ItemIdentity{WorkspaceID: workspace, FeedID: feedID, ItemID: uuid(3), RenditionID: uuid(4)}
	page := store.BriefingReadPage{Items: []store.ReadyBriefing{{ID: id.ItemID, RenditionID: id.RenditionID, Title: " Original title ", Status: "ready", Source: "Original\nsource", Script: "Exact script.", Segments: []store.PublishedSegment{{ID: uuid(5), Kind: "narration", AssetID: uuid(6), DurationMS: 1000}, {ID: uuid(7), Kind: "image", ImageIDs: []string{uuid(8)}}}}}, NextCursor: "opaque-cursor"}
	inputs := presentation.BriefingInputs{Attributions: []presentation.Attribution{{ItemIdentity: id, Label: "Release worker"}}}
	for i, asset := range []string{uuid(6), uuid(8)} {
		mime := "audio/mpeg"
		if i == 1 {
			mime = "image/png"
		}
		m := presentation.AssetMetadata{AssetIdentity: presentation.AssetIdentity{ItemIdentity: id, AssetID: asset}, SHA256: strings.Repeat("a", 64), MIME: mime, ByteSize: 100}
		inputs.Published = append(inputs.Published, m)
		inputs.Links = append(inputs.Links, presentation.MediaLink{AssetMetadata: m, URL: origin + "/already-bound/" + asset})
	}
	return scope, page, inputs
}
func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestBriefingWirePreservesContentAndResourceIDs(t *testing.T) {
	scope, page, inputs := fixture()
	got, err := presentation.MarshalBriefings(scope, page, inputs, origin)
	if err != nil {
		t.Fatal(err)
	}
	value := decode(t, got)
	item := value["items"].([]any)[0].(map[string]any)
	for key, want := range map[string]string{"id": page.Items[0].ID, "renditionId": page.Items[0].RenditionID, "title": page.Items[0].Title, "source": page.Items[0].Source, "script": page.Items[0].Script, "agent": "Release worker", "status": "ready"} {
		if item[key] != want {
			t.Fatalf("field %s changed", key)
		}
	}
	segments := item["segments"].([]any)
	audio := segments[0].(map[string]any)
	image := segments[1].(map[string]any)
	if value["nextCursor"] != page.NextCursor || audio["assetId"] != uuid(6) || audio["url"] != inputs.Links[0].URL || audio["durationMs"] != float64(1000) || image["imageIds"].([]any)[0] != uuid(8) || image["images"].([]any)[0] != inputs.Links[1].URL || image["durationMs"] != float64(0) {
		t.Fatal("Flutter fields or resource selectors lost")
	}
	before := append([]byte(nil), got...)
	page.Items[0].Segments[1].ImageIDs[0] = "mutated"
	inputs.Attributions[0].Label = "mutated"
	inputs.Links[0].URL = "mutated"
	if !bytes.Equal(got, before) {
		t.Fatal("output aliases caller-owned inputs")
	}
	got[0] = '!'
	if page.Items[0].Title != " Original title " {
		t.Fatal("output mutation changed input")
	}
}

func TestBriefingRejectsInvalidBindingsWithoutOutput(t *testing.T) {
	cases := []struct {
		name   string
		change func(*presentation.Scope, *store.BriefingReadPage, *presentation.BriefingInputs, *string)
	}{
		{"invalid utf8 URL", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = origin + "/" + string([]byte{255})
		}},
		{"missing attribution", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Attributions = nil
		}},
		{"blank attribution", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Attributions[0].Label = " "
		}},
		{"private control label", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Attributions[0].Label = "name\nsecret"
		}},
		{"invalid utf8 label", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Attributions[0].Label = string([]byte{255})
		}},
		{"long label", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Attributions[0].Label = strings.Repeat("a", 257)
		}},
		{"stale attribution", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Attributions[0].RenditionID = uuid(99)
		}},
		{"missing asset", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Published = i.Published[:1]
			i.Links = i.Links[:1]
		}},
		{"foreign scope", func(s *presentation.Scope, _ *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			s.WorkspaceID = uuid(99)
		}},
		{"foreign asset", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].FeedID = uuid(99)
		}},
		{"duplicate metadata", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Published[1] = i.Published[0]
		}},
		{"duplicate mapping", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[1] = i.Links[0]
		}},
		{"digest mismatch", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].SHA256 = strings.Repeat("b", 64)
		}},
		{"mime mismatch", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].MIME = "video/mp4"
		}},
		{"size mismatch", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].ByteSize++
		}},
		{"wrong kind mime", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Published[0].MIME = "video/mp4"
			i.Links[0].MIME = "video/mp4"
		}},
		{"bad digest", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Published[0].SHA256 = "not-a-digest"
			i.Links[0].SHA256 = "not-a-digest"
		}},
		{"foreign origin", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = "https://foreign.invalid/media"
		}},
		{"userinfo token", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = "https://secret@media.example.invalid/media"
		}},
		{"query token", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = origin + "/media?token=private"
		}},
		{"empty query", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = origin + "/media?"
		}},
		{"fragment", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = origin + "/media#private"
		}},
		{"http", func(_ *presentation.Scope, _ *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			i.Links[0].URL = "http://media.example.invalid/media"
		}},
		{"origin path", func(_ *presentation.Scope, _ *store.BriefingReadPage, _ *presentation.BriefingInputs, o *string) {
			*o = origin + "/invented"
		}},
		{"duplicate item", func(_ *presentation.Scope, p *store.BriefingReadPage, i *presentation.BriefingInputs, _ *string) {
			p.Items = append(p.Items, p.Items[0])
			i.Attributions = append(i.Attributions, i.Attributions[0])
		}},
		{"invalid content utf8", func(_ *presentation.Scope, p *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			p.Items[0].Source = string([]byte{255})
		}},
		{"no script", func(_ *presentation.Scope, p *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			p.Items[0].Script = ""
		}},
		{"too long narration", func(_ *presentation.Scope, p *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			p.Items[0].Segments[0].DurationMS = 600001
		}},
		{"too many items", func(_ *presentation.Scope, p *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			p.Items = make([]store.ReadyBriefing, 11)
		}},
		{"too many segments", func(_ *presentation.Scope, p *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			p.Items[0].Segments = make([]store.PublishedSegment, 101)
		}},
		{"too many images", func(_ *presentation.Scope, p *store.BriefingReadPage, _ *presentation.BriefingInputs, _ *string) {
			p.Items[0].Segments[1].ImageIDs = make([]string, 21)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scope, page, inputs := fixture()
			o := origin
			tc.change(&scope, &page, &inputs, &o)
			b, err := presentation.MarshalBriefings(scope, page, inputs, o)
			if !errors.Is(err, presentation.ErrInvalid) || b != nil || err.Error() != "invalid consumption presentation" {
				t.Fatalf("unsafe result: %q %v", b, err)
			}
		})
	}
}

func TestUnexpectedAssetIsRejected(t *testing.T) {
	scope, page, inputs := fixture()
	m := inputs.Published[1]
	m.AssetID = uuid(99)
	inputs.Published = append(inputs.Published, m)
	inputs.Links = append(inputs.Links, presentation.MediaLink{AssetMetadata: m, URL: origin + "/unused"})
	b, err := presentation.MarshalBriefings(scope, page, inputs, origin)
	if !errors.Is(err, presentation.ErrInvalid) || b != nil {
		t.Fatal("unexpected mapping accepted")
	}
}

func TestCompleteNativeGraphIncludesParentsButNotSegmentIDs(t *testing.T) {
	for _, assets := range []int{252, 253} {
		t.Run(fmt.Sprint(assets), func(t *testing.T) {
			scope, page, inputs := fixture()
			page.Items[0].Segments = nil
			inputs.Published = nil
			inputs.Links = nil
			for n := 0; n < assets; n++ {
				m := presentation.AssetMetadata{AssetIdentity: presentation.AssetIdentity{ItemIdentity: inputs.Attributions[0].ItemIdentity, AssetID: uuid(100 + n)}, SHA256: strings.Repeat("a", 64), MIME: "image/png", ByteSize: 1}
				inputs.Published = append(inputs.Published, m)
				inputs.Links = append(inputs.Links, presentation.MediaLink{AssetMetadata: m, URL: origin + "/x"})
				if n%20 == 0 {
					page.Items[0].Segments = append(page.Items[0].Segments, store.PublishedSegment{ID: uuid(1000 + n), Kind: "image"})
				}
				last := len(page.Items[0].Segments) - 1
				page.Items[0].Segments[last].ImageIDs = append(page.Items[0].Segments[last].ImageIDs, m.AssetID)
			}
			b, err := presentation.MarshalBriefings(scope, page, inputs, origin)
			if assets == 252 {
				if err != nil || len(b) == 0 {
					t.Fatalf("exact 256-node graph rejected: %v", err)
				}
			} else if !errors.Is(err, presentation.ErrInvalid) || b != nil {
				t.Fatal("257-node graph accepted")
			}
		})
	}
}

func TestRepeatedReferenceVisitsHaveIndependentBudget(t *testing.T) {
	for _, items := range []int{3, 4} {
		t.Run(fmt.Sprint(items), func(t *testing.T) {
			scope, page, inputs := fixture()
			template := page.Items[0]
			page.Items = nil
			inputs.Attributions = nil
			inputs.Published = nil
			inputs.Links = nil
			for n := 0; n < items; n++ {
				item := template
				item.ID = uuid(100 + n)
				item.RenditionID = uuid(200 + n)
				item.Segments = nil
				id := presentation.ItemIdentity{WorkspaceID: workspace, FeedID: feedID, ItemID: item.ID, RenditionID: item.RenditionID}
				inputs.Attributions = append(inputs.Attributions, presentation.Attribution{ItemIdentity: id, Label: "Explicit worker"})
				var images []string
				for a := 0; a < 20; a++ {
					m := presentation.AssetMetadata{AssetIdentity: presentation.AssetIdentity{ItemIdentity: id, AssetID: uuid(300 + a)}, SHA256: strings.Repeat("a", 64), MIME: "image/png", ByteSize: 1}
					images = append(images, m.AssetID)
					inputs.Published = append(inputs.Published, m)
					inputs.Links = append(inputs.Links, presentation.MediaLink{AssetMetadata: m, URL: origin + "/x"})
				}
				for s := 0; s < 100; s++ {
					item.Segments = append(item.Segments, store.PublishedSegment{ID: uuid(1000 + s), Kind: "image", ImageIDs: images})
				}
				page.Items = append(page.Items, item)
			}
			b, err := presentation.MarshalBriefings(scope, page, inputs, origin)
			if items == 3 {
				if err != nil || len(b) == 0 {
					t.Fatalf("bounded repeated references rejected: %v", err)
				}
			} else if !errors.Is(err, presentation.ErrInvalid) || b != nil {
				t.Fatal("repeated reference visit budget bypassed")
			}
		})
	}
}

func TestJSONSizeLimitAfterEscaping(t *testing.T) {
	scope, page, inputs := fixture()
	page.Items[0].Script = strings.Repeat("<", 200000)
	b, err := presentation.MarshalBriefings(scope, page, inputs, origin)
	if !errors.Is(err, presentation.ErrInvalid) || b != nil {
		t.Fatal("escaped JSON size cap bypassed")
	}
}

func TestFeedPermissionsAreExplicitAndScopeBound(t *testing.T) {
	page := store.FeedReadPage{Feeds: []store.FeedSummary{{ID: feedID, Name: "Exact feed", Kind: "personal"}}}
	for _, canManage := range []bool{false, true} {
		b, err := presentation.MarshalFeeds(workspace, page, []presentation.FeedPermission{{WorkspaceID: workspace, FeedID: feedID, CanManage: canManage}})
		if err != nil {
			t.Fatal(err)
		}
		f := decode(t, b)["feeds"].([]any)[0].(map[string]any)
		if f["canManage"] != canManage || f["name"] != "Exact feed" || f["kind"] != "personal" || f["id"] != feedID {
			t.Fatal("permission or feed wire changed")
		}
	}
	for _, permissions := range [][]presentation.FeedPermission{nil, {{WorkspaceID: uuid(99), FeedID: feedID}}, {{WorkspaceID: workspace, FeedID: uuid(99)}}, {{WorkspaceID: workspace, FeedID: feedID}, {WorkspaceID: workspace, FeedID: feedID}}} {
		b, err := presentation.MarshalFeeds(workspace, page, permissions)
		if !errors.Is(err, presentation.ErrInvalid) || b != nil {
			t.Fatal("missing, duplicate or foreign permission accepted")
		}
	}
}
