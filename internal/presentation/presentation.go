// Package presentation shapes local store projections for the Flutter client.
// Its inputs are data, never authorization or evidence of a live media grant.
package presentation

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/narrate-it/prosody/internal/store"
)

const (
	MaxItems     = 10
	MaxSegments  = 100
	MaxImages    = 20
	MaxResources = 256
	MaxJSONBytes = 1 << 20
	MaxVisits    = 32768
)

// ErrInvalid is deliberately fixed: caller content, URLs and diagnostics are private.
var ErrInvalid = errors.New("invalid consumption presentation")

// Scope must come from the caller's current native human-read operation.
type Scope struct{ WorkspaceID, FeedID string }

// ItemIdentity binds attribution to an exact published rendition.
type ItemIdentity struct{ WorkspaceID, FeedID, ItemID, RenditionID string }

// Attribution is a caller-resolved public display label, never a private authority ref.
type Attribution struct {
	ItemIdentity
	Label string
}

// AssetIdentity retains the complete closure identity; it is not a capability.
type AssetIdentity struct {
	ItemIdentity
	AssetID string
}

// AssetMetadata is caller-held published metadata from the same read snapshot.
// The host must separately verify current grants and recompute delivered bytes.
type AssetMetadata struct {
	AssetIdentity
	SHA256, MIME string
	ByteSize     int64
}

// MediaLink is an already-bound absolute link and the metadata it was bound to.
// Matching metadata here establishes data consistency only, never grant freshness.
type MediaLink struct {
	AssetMetadata
	URL string
}

// BriefingInputs must contain exactly the attributions and assets used by a page.
type BriefingInputs struct {
	Attributions []Attribution
	Published    []AssetMetadata
	Links        []MediaLink
}

// FeedPermission is an explicit native permission projection, including false.
type FeedPermission struct {
	WorkspaceID, FeedID string
	CanManage           bool
}

type segment struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	AssetID    string   `json:"assetId,omitempty"`
	ImageIDs   []string `json:"imageIds,omitempty"`
	DurationMS int64    `json:"durationMs"`
	URL        string   `json:"url,omitempty"`
	Images     []string `json:"images,omitempty"`
}
type briefing struct {
	ID          string    `json:"id"`
	RenditionID string    `json:"renditionId"`
	Title       string    `json:"title"`
	Agent       string    `json:"agent"`
	Status      string    `json:"status"`
	Source      string    `json:"source"`
	Script      string    `json:"script"`
	Segments    []segment `json:"segments"`
}
type briefingPage struct {
	Items      []briefing `json:"items"`
	NextCursor string     `json:"nextCursor,omitempty"`
}
type feed struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	CanManage bool   `json:"canManage"`
}
type feedPage struct {
	Feeds      []feed `json:"feeds"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type visits int

func (v *visits) add(n int) bool { *v += visits(n); return *v <= MaxVisits }

type graph map[string]bool

func (g graph) add(kind, id string) bool {
	g[kind+":"+id] = true
	return len(g) <= MaxResources
}

func text(s string, max int) bool { return len(s) <= max && utf8.ValidString(s) }
func label(s string) bool {
	if !text(s, 256) || strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func validScope(s Scope) bool { return store.UUID(s.WorkspaceID) && store.FeedID(s.FeedID) }
func identity(s Scope, item store.ReadyBriefing) ItemIdentity {
	return ItemIdentity{s.WorkspaceID, s.FeedID, item.ID, item.RenditionID}
}
func metadata(m AssetMetadata) bool {
	if !validScope(Scope{m.WorkspaceID, m.FeedID}) || !store.UUID(m.ItemID) || !store.UUID(m.RenditionID) || !store.UUID(m.AssetID) || len(m.SHA256) != 64 || m.ByteSize < 1 || m.ByteSize > 1<<30 {
		return false
	}
	if _, err := hex.DecodeString(m.SHA256); err != nil || strings.ToLower(m.SHA256) != m.SHA256 {
		return false
	}
	return m.MIME == "audio/mpeg" || m.MIME == "video/mp4" || m.MIME == "image/jpeg" || m.MIME == "image/png"
}
func originURL(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || !utf8.ValidString(raw) || len(raw) > 4096 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#") || u.Opaque != "" {
		return nil, false
	}
	return u, true
}
func linkURL(raw string, origin *url.URL) bool {
	u, ok := originURL(raw)
	return ok && u.Host == origin.Host && u.Path != "" && u.Path[0] == '/'
}
func encode(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil || len(b) > MaxJSONBytes {
		return nil, ErrInvalid
	}
	return b, nil
}

// MarshalBriefings produces fresh Flutter-compatible JSON without granting access.
// Every mapping must be exact; malformed or oversized pages produce no bytes.
// Origin is configuration (HTTPS scheme and host only), not a URL allowlist grant.
func MarshalBriefings(scope Scope, page store.BriefingReadPage, inputs BriefingInputs, origin string) ([]byte, error) {
	base, ok := originURL(origin)
	if !ok || (base.Path != "" && base.Path != "/") || !validScope(scope) || len(page.Items) > MaxItems || !text(page.NextCursor, 4096) || len(inputs.Attributions) != len(page.Items) || len(inputs.Published) > MaxResources || len(inputs.Links) != len(inputs.Published) {
		return nil, ErrInvalid
	}
	budget := visits(2)
	attrs := map[ItemIdentity]string{}
	for _, a := range inputs.Attributions {
		if !budget.add(4) || !label(a.Label) {
			return nil, ErrInvalid
		}
		if _, exists := attrs[a.ItemIdentity]; exists {
			return nil, ErrInvalid
		}
		attrs[a.ItemIdentity] = a.Label
	}
	published := map[AssetIdentity]AssetMetadata{}
	assetFacts := map[string]AssetMetadata{}
	for _, m := range inputs.Published {
		if !budget.add(5) || !metadata(m) {
			return nil, ErrInvalid
		}
		if _, exists := published[m.AssetIdentity]; exists {
			return nil, ErrInvalid
		}
		if prior, exists := assetFacts[m.AssetID]; exists && (prior.SHA256 != m.SHA256 || prior.MIME != m.MIME || prior.ByteSize != m.ByteSize) {
			return nil, ErrInvalid
		}
		assetFacts[m.AssetID] = m
		published[m.AssetIdentity] = m
	}
	links := map[AssetIdentity]string{}
	for _, m := range inputs.Links {
		expected, exists := published[m.AssetIdentity]
		if !budget.add(5) || !exists || expected != m.AssetMetadata || !linkURL(m.URL, base) {
			return nil, ErrInvalid
		}
		if _, exists = links[m.AssetIdentity]; exists {
			return nil, ErrInvalid
		}
		links[m.AssetIdentity] = m.URL
	}
	g := graph{}
	g.add("workspace", scope.WorkspaceID)
	g.add("feed", scope.FeedID)
	out := briefingPage{Items: make([]briefing, 0, len(page.Items)), NextCursor: page.NextCursor}
	seenItems := map[string]bool{}
	used := map[AssetIdentity]bool{}
	for _, item := range page.Items {
		id := identity(scope, item)
		agent, exists := attrs[id]
		if !budget.add(4) || !exists || !store.UUID(item.ID) || !store.UUID(item.RenditionID) || seenItems[item.ID] || item.Status != "ready" || !text(item.Title, 4096) || !text(item.Source, 65536) || !text(item.Script, 256000) || len(item.Segments) == 0 || len(item.Segments) > MaxSegments || !g.add("item", item.ID) || !g.add("rendition", item.RenditionID) {
			return nil, ErrInvalid
		}
		seenItems[item.ID] = true
		b := briefing{ID: item.ID, RenditionID: item.RenditionID, Title: item.Title, Agent: agent, Status: item.Status, Source: item.Source, Script: item.Script, Segments: make([]segment, 0, len(item.Segments))}
		seenSegments := map[string]bool{}
		var narrationMS, totalMS int64
		for _, s := range item.Segments {
			if !budget.add(1) || !store.UUID(s.ID) || seenSegments[s.ID] || len(s.ImageIDs) > MaxImages || s.DurationMS < 0 {
				return nil, ErrInvalid
			}
			seenSegments[s.ID] = true
			if s.DurationMS > 9007199254740991-totalMS {
				return nil, ErrInvalid
			}
			totalMS += s.DurationMS
			target := segment{ID: s.ID, Kind: s.Kind, AssetID: s.AssetID, ImageIDs: append([]string(nil), s.ImageIDs...), DurationMS: s.DurationMS}
			if s.Kind == "image" {
				if s.AssetID != "" || s.DurationMS != 0 || len(s.ImageIDs) == 0 {
					return nil, ErrInvalid
				}
			} else {
				if (s.Kind != "narration" && s.Kind != "audio" && s.Kind != "video") || s.DurationMS == 0 {
					return nil, ErrInvalid
				}
				expectedMIME := "audio/mpeg"
				if s.Kind == "video" {
					expectedMIME = "video/mp4"
				}
				asset := AssetIdentity{id, s.AssetID}
				m, found := published[asset]
				if !budget.add(5) || !found || m.MIME != expectedMIME || !g.add("asset", s.AssetID) {
					return nil, ErrInvalid
				}
				target.URL = links[asset]
				used[asset] = true
				if s.Kind == "narration" {
					if s.DurationMS > 600000-narrationMS {
						return nil, ErrInvalid
					}
					narrationMS += s.DurationMS
				}
			}
			seenImages := map[string]bool{}
			for _, image := range s.ImageIDs {
				asset := AssetIdentity{id, image}
				m, found := published[asset]
				if !budget.add(5) || seenImages[image] || !found || (m.MIME != "image/jpeg" && m.MIME != "image/png") || !g.add("asset", image) {
					return nil, ErrInvalid
				}
				seenImages[image] = true
				target.Images = append(target.Images, links[asset])
				used[asset] = true
			}
			b.Segments = append(b.Segments, target)
		}
		if narrationMS > 0 && item.Script == "" {
			return nil, ErrInvalid
		}
		out.Items = append(out.Items, b)
	}
	if len(used) != len(published) {
		return nil, ErrInvalid
	}
	return encode(out)
}

// MarshalFeeds requires one explicit, scope-bound permission per feed. It never
// infers management rights from a role, feed kind or absent permission record.
func MarshalFeeds(workspace string, page store.FeedReadPage, permissions []FeedPermission) ([]byte, error) {
	if !store.UUID(workspace) || len(page.Feeds) > MaxItems || len(permissions) != len(page.Feeds) || !text(page.NextCursor, 4096) {
		return nil, ErrInvalid
	}
	rights := map[string]bool{}
	for _, p := range permissions {
		if p.WorkspaceID != workspace || !store.FeedID(p.FeedID) {
			return nil, ErrInvalid
		}
		if _, exists := rights[p.FeedID]; exists {
			return nil, ErrInvalid
		}
		rights[p.FeedID] = p.CanManage
	}
	out := feedPage{Feeds: make([]feed, 0, len(page.Feeds)), NextCursor: page.NextCursor}
	seen := map[string]bool{}
	for _, f := range page.Feeds {
		manage, exists := rights[f.ID]
		if !exists || seen[f.ID] || !store.FeedID(f.ID) || !text(f.Name, 4096) || strings.TrimSpace(f.Name) == "" || (f.Kind != "personal" && f.Kind != "team") {
			return nil, ErrInvalid
		}
		seen[f.ID] = true
		out.Feeds = append(out.Feeds, feed{f.ID, f.Name, f.Kind, manage})
	}
	return encode(out)
}
