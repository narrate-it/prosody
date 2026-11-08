// Package notifications defines Prosody notification data, not delivery or
// authority. Encoding a payload cannot authorize an effect or access an item.
package notifications

import (
	"encoding/json"
	"errors"
)

// ErrInvalidTarget rejects malformed routing hints without returning a payload.
var ErrInvalidTarget = errors.New("invalid ready notification target")

// ReadyTarget identifies the immutable publication to resolve after a tap.
// These opaque IDs are routing hints, never permissions, delivery credentials or
// proof that a publication committed. They are visible to the push provider.
type ReadyTarget struct {
	FeedID, ItemID, RenditionID string
}

// ReadyPayload encodes the default privacy-preserving APNs alert profile.
// It accepts no private title, source, script, agent label, URL or bearer token.
// A dispatcher must separately establish committed publication, current opt-in,
// recipient/device authority and per-person/publication delivery deduplication.
func ReadyPayload(target ReadyTarget) ([]byte, error) {
	if !validReadyTarget(target) {
		return nil, ErrInvalidTarget
	}
	payload := struct {
		APS struct {
			Alert string `json:"alert"`
		} `json:"aps"`
		Prosody struct {
			Version     int    `json:"version"`
			Kind        string `json:"kind"`
			FeedID      string `json:"feedId"`
			ItemID      string `json:"itemId"`
			RenditionID string `json:"renditionId"`
		} `json:"prosody"`
	}{}
	payload.APS.Alert = "A new briefing is ready."
	payload.Prosody.Version = 1
	payload.Prosody.Kind = "ready"
	payload.Prosody.FeedID = target.FeedID
	payload.Prosody.ItemID = target.ItemID
	payload.Prosody.RenditionID = target.RenditionID
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if len(b) > 4096 {
		return nil, ErrInvalidTarget
	}
	return b, nil
}

// validReadyTarget enforces the existing UUIDv7 feed contract and canonical
// nonzero item/rendition IDs. Validation does not establish their relationship.
func validReadyTarget(t ReadyTarget) bool {
	return canonicalUUID(t.FeedID) && t.FeedID[14] == '7' &&
		(t.FeedID[19] == '8' || t.FeedID[19] == '9' || t.FeedID[19] == 'a' || t.FeedID[19] == 'b') &&
		canonicalUUID(t.ItemID) && canonicalUUID(t.RenditionID)
}

// canonicalUUID is a wire-boundary check; it performs no store or authority read.
func canonicalUUID(value string) bool {
	if len(value) != 36 || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
