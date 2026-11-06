package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleInventory() inventory {
	p := 1
	flatten := false
	note := "existing record note"
	return inventory{ZoneID: strings.Repeat("a", 32), AccountID: strings.Repeat("b", 32), Records: []record{{ID: strings.Repeat("c", 32), Name: domain, Type: "A", Content: "192.0.2.1", TTL: 1, Proxied: true}, {ID: strings.Repeat("d", 32), Name: domain, Type: "MX", Content: "smtp.example.com", TTL: 1, Priority: &p}, {ID: strings.Repeat("e", 32), Name: "_dmarc." + domain, Type: "TXT", Content: `"v=DMARC1; p=none"`, TTL: 1}, {ID: strings.Repeat("f", 32), Name: "www." + domain, Type: "CNAME", Content: domain, TTL: 1, Proxied: true, Comment: &note, Tags: []string{"service:dns"}, Settings: settings{FlattenCname: &flatten}}, {ID: strings.Repeat("0", 32), Name: domain, Type: "AAAA", Content: "2001:db8::1", TTL: 1, Proxied: true}}}
}

// TestInventory preserves public mail/text data exactly while admitting only
// existing records within the selected domain.
func TestInventory(t *testing.T) {
	v := sampleInventory()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadInventory(strings.NewReader(string(b)))
	if err != nil || len(got.Records) != 5 || got.Records[2].Content != v.Records[2].Content {
		t.Fatal("valid inventory rejected or TXT changed")
	}
	for _, tc := range []struct {
		name   string
		change func(*inventory)
	}{
		{"foreign zone name", func(v *inventory) { v.Records[0].Name = "prosody.chat.attacker.example" }},
		{"duplicate id", func(v *inventory) { v.Records[1].ID = v.Records[0].ID }},
		{"missing import identity", func(v *inventory) { v.Records[0].ID = "" }},
		{"wrong address family", func(v *inventory) { v.Records[0].Type = "AAAA" }},
		{"proxied mail", func(v *inventory) { v.Records[1].Proxied = true }},
		{"mail without priority", func(v *inventory) { v.Records[1].Priority = nil }},
		{"priority on address", func(v *inventory) { v.Records[0].Priority = v.Records[1].Priority }},
		{"unsupported schema", func(v *inventory) { v.Records[0].Type = "SRV" }},
		{"unsupported ttl", func(v *inventory) { v.Records[0].TTL = 2 }},
		{"noncanonical name", func(v *inventory) { v.Records[0].Name = "bad.." + domain }},
		{"no records", func(v *inventory) { v.Records = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := sampleInventory()
			tc.change(&v)
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = loadInventory(strings.NewReader(string(b))); err == nil {
				t.Fatal("invalid inventory accepted")
			}
		})
	}
}

// TestInventoryEnvelope refuses malformed, unknown and excessive input.
func TestInventoryEnvelope(t *testing.T) {
	for _, s := range []string{"{} {}", `{"unknown":true}`, "null", strings.Repeat(" ", 1<<20+1)} {
		if _, err := loadInventory(strings.NewReader(s)); err == nil {
			t.Fatal("invalid envelope accepted")
		}
	}
}

// TestInventoryDuplicateFields prevents ambiguous imported desired state.
func TestInventoryDuplicateFields(t *testing.T) {
	b, err := json.Marshal(sampleInventory())
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"zoneId":"`, `"content":"`} {
		at := strings.Index(string(b), needle)
		if at < 0 {
			t.Fatal("fixture field missing")
		}
		raw := string(b[:at]) + strings.Split(needle, ":")[0] + `:"overwritten",` + string(b[at:])
		if _, err = loadInventory(strings.NewReader(raw)); err == nil {
			t.Fatal("duplicate field accepted")
		}
	}
}

// TestInventoryCaseAliases rejects encoding/json's case-insensitive aliases.
func TestInventoryCaseAliases(t *testing.T) {
	b, err := json.Marshal(sampleInventory())
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		strings.Replace(string(b), `"zoneId":`, `"ZoneId":"`+strings.Repeat("f", 32)+`","zoneId":`, 1),
		strings.Replace(string(b), `"content":`, `"Content":"overwritten","content":`, 1),
		strings.Replace(string(b), `"settings":{}`, `"settings":{"flatten_cname":false,"Flatten_cname":true}`, 1),
	} {
		if _, err = loadInventory(strings.NewReader(raw)); err == nil {
			t.Fatal("case alias accepted")
		}
	}
}

// TestInventoryUnicode requires lossless strings while accepting literal U+FFFD.
func TestInventoryUnicode(t *testing.T) {
	v := sampleInventory()
	v.Records[2].Content = "sentinel"
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range []string{`"\ud800"`, `"\udc00"`, `"` + string([]byte{0xff}) + `"`} {
		raw := strings.Replace(string(b), `"sentinel"`, replacement, 1)
		if _, err = loadInventory(strings.NewReader(raw)); err == nil {
			t.Fatal("lossy Unicode accepted")
		}
	}
	for _, replacement := range []string{`"�"`, `"\ufffd"`, `"\ud83d\ude00"`} {
		raw := strings.Replace(string(b), `"sentinel"`, replacement, 1)
		if _, err = loadInventory(strings.NewReader(raw)); err != nil {
			t.Fatal("valid Unicode rejected")
		}
	}
}
