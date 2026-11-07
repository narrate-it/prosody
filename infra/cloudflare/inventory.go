package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/narrate-it/prosody/internal/strictjson"
	"io"
	"net/netip"
	"regexp"
	"strings"
)

const domain = "prosody.chat"

// publicStateInputClass records explicit operator review, not inferred publicity
// or permission to apply. Canonical imported inputs may remain plaintext.
const publicStateInputClass = "public-reviewed-v1"

var errStateCustody = errors.New("canonical import inputs may be plaintext: explicit public input review and no annotations required")

var cloudID = regexp.MustCompile(`^[a-f0-9]{32}$`)
var label = regexp.MustCompile(`^[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?$`)

type inventory struct {
	ZoneID    string   `json:"zoneId"`
	AccountID string   `json:"accountId"`
	Records   []record `json:"records"`
}

type record struct {
	StateInputClass string   `json:"stateInputClass"`
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Content         string   `json:"content"`
	TTL             int      `json:"ttl"`
	Proxied         bool     `json:"proxied"`
	Priority        *int     `json:"priority,omitempty"`
	Comment         *string  `json:"comment,omitempty"`
	Tags            []string `json:"tags"`
	Settings        settings `json:"settings"`
}

type settings struct {
	FlattenCname *bool `json:"flatten_cname,omitempty"`
	IPv4Only     *bool `json:"ipv4_only,omitempty"`
	IPv6Only     *bool `json:"ipv6_only,omitempty"`
}

// loadInventory accepts a bounded private desired-state document. It never logs
// DNS contents or silently drops unsupported resource fields/types.
func loadInventory(r io.Reader) (inventory, error) {
	var v inventory
	b, err := io.ReadAll(io.LimitReader(r, 1<<20+1))
	if err != nil || len(b) > 1<<20 {
		return v, errors.New("inventory exceeds byte limit or cannot be read")
	}
	if strictjson.Validate(b) != nil {
		return inventory{}, errors.New("inventory contains invalid JSON Unicode")
	}
	if err := uniqueJSON(json.NewDecoder(strings.NewReader(string(b))), 0); err != nil {
		return inventory{}, errors.New("duplicate or invalid JSON structure")
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err = d.Decode(&v); err != nil {
		return inventory{}, errors.New("invalid inventory JSON")
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return inventory{}, errors.New("inventory must contain exactly one JSON value")
	}
	if !cloudID.MatchString(v.ZoneID) || !cloudID.MatchString(v.AccountID) || len(v.Records) == 0 || len(v.Records) > 100 {
		return inventory{}, errors.New("inventory needs exact account/zone IDs and 1–100 records")
	}
	seen := map[string]bool{}
	for i, r := range v.Records {
		if err = r.validate(); err != nil {
			return inventory{}, fmt.Errorf("invalid inventory record %d: %w", i, err)
		}
		if seen[r.ID] {
			return inventory{}, errors.New("duplicate record ID")
		}
		seen[r.ID] = true
	}
	if err := validateStateCustody(v); err != nil {
		return inventory{}, err
	}
	return v, nil
}

// validateStateCustody rejects potentially confidential canonical import inputs.
// A config label does not prove encryption, publicity or permission to apply.
func validateStateCustody(v inventory) error {
	for _, r := range v.Records {
		if r.StateInputClass != publicStateInputClass || (r.Comment != nil && *r.Comment != "") || len(r.Tags) != 0 {
			return errStateCustody
		}
	}
	return nil
}

// validate scopes record ownership to this domain and preserves exact contents.
func (r record) validate() error {
	if !cloudID.MatchString(r.ID) || (r.Name != domain && !strings.HasSuffix(r.Name, "."+domain)) || len(r.Name) > 253 {
		return errors.New("record identity outside the admitted zone")
	}
	for i, l := range strings.Split(r.Name, ".") {
		if (i != 0 || l != "*") && !label.MatchString(l) {
			return errors.New("noncanonical DNS name")
		}
	}
	if r.Content == "" || len(r.Content) > 16384 || (r.TTL != 1 && (r.TTL < 60 || r.TTL > 86400)) {
		return errors.New("invalid content/TTL bounds")
	}
	switch r.Type {
	case "A", "AAAA":
		ip, err := netip.ParseAddr(r.Content)
		if err != nil || ip.Zone() != "" || (r.Type == "A" && !ip.Is4()) || (r.Type == "AAAA" && (!ip.Is6() || ip.Is4In6())) {
			return errors.New("address type mismatch")
		}
	case "CNAME", "MX", "TXT":
	default:
		return errors.New("unsupported record type requires explicit implementation")
	}
	if r.Proxied && r.Type != "A" && r.Type != "AAAA" && r.Type != "CNAME" {
		return errors.New("record type cannot be proxied")
	}
	if r.Proxied && r.TTL != 1 {
		return errors.New("proxied records require automatic TTL")
	}
	if r.Type == "MX" {
		if r.Priority == nil || *r.Priority < 0 || *r.Priority > 65535 {
			return errors.New("MX needs valid priority")
		}
	} else if r.Priority != nil {
		return errors.New("priority only supported on MX")
	}
	if len(r.Tags) > 20 {
		return errors.New("too many record tags")
	}
	return nil
}

// uniqueJSON rejects overwritten duplicate keys and excessive nesting before
// typed decoding, including duplicates inside record settings.
func uniqueJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("JSON nesting exceeds bound")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := k.(string)
			if !ok || !canonicalField(name) || seen[name] {
				return errors.New("duplicate object field")
			}
			seen[name] = true
			if err = uniqueJSON(d, depth+1); err != nil {
				return err
			}
		}
		t, err = d.Token()
		if err != nil || t != json.Delim('}') {
			return errors.New("invalid object")
		}
	case '[':
		for d.More() {
			if err = uniqueJSON(d, depth+1); err != nil {
				return err
			}
		}
		t, err = d.Token()
		if err != nil || t != json.Delim(']') {
			return errors.New("invalid array")
		}
	default:
		return errors.New("invalid delimiter")
	}
	return nil
}

// canonicalField rejects encoding/json's case-insensitive struct-field aliases.
// Typed decoding additionally enforces the fields allowed in each object.
func canonicalField(s string) bool {
	switch s {
	case "zoneId", "accountId", "records", "stateInputClass", "id", "name", "type", "content", "ttl", "proxied", "priority", "comment", "tags", "settings", "flatten_cname", "ipv4_only", "ipv6_only":
		return true
	default:
		return false
	}
}
