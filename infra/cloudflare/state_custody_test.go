package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// custodyCases exercises documentary input classification and prohibited metadata.
// Synthetic marker values must never appear in an error or provider invocation.
func custodyCases() []struct {
	name   string
	change func(*inventory)
} {
	return []struct {
		name   string
		change func(*inventory)
	}{
		{"missing review", func(v *inventory) { v.Records[2].StateInputClass = "" }},
		{"confidential class", func(v *inventory) { v.Records[2].StateInputClass = "private" }},
		{"unknown class", func(v *inventory) { v.Records[2].StateInputClass = "public" }},
		{"nonempty comment", func(v *inventory) {
			note := "SENSITIVE_COMMENT_MARKER"
			v.Records[2].Comment = &note
		}},
		{"nonempty tags", func(v *inventory) { v.Records[2].Tags = []string{"SENSITIVE_TAG_MARKER"} }},
	}
}

// TestStateCustodyInventory refuses unsupported confidentiality before admission.
func TestStateCustodyInventory(t *testing.T) {
	for _, tc := range custodyCases() {
		t.Run(tc.name, func(t *testing.T) {
			v := sampleInventory()
			v.Records[2].Content = "SENSITIVE_CONTENT_MARKER"
			tc.change(&v)
			input, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			_, err = loadInventory(strings.NewReader(string(input)))
			if !errors.Is(err, errStateCustody) {
				t.Fatal("state-custody violation was not refused")
			}
			if strings.Contains(err.Error(), "SENSITIVE_") {
				t.Fatal("rejection exposed record material")
			}
		})
	}
}

// TestStateCustodyStopsBeforeProvider protects callers that bypass JSON loading.
func TestStateCustodyStopsBeforeProvider(t *testing.T) {
	for _, tc := range custodyCases() {
		t.Run(tc.name, func(t *testing.T) {
			v := sampleInventory()
			tc.change(&v)
			m := &dnsMonitor{account: v.AccountID, name: domain}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerDNS(ctx, v) }, pulumi.WithMocks("prosody-dns", "unit", m))
			if !errors.Is(err, errStateCustody) {
				t.Fatal("direct registration bypassed state-custody refusal")
			}
			m.mu.Lock()
			calls, records := m.calls, m.count
			m.mu.Unlock()
			if calls != 0 || records != 0 {
				t.Fatal("provider was called before state-custody admission")
			}
		})
	}
}

// TestReviewedPublicInputsPreserveExactBytes covers this limited admission profile.
// A marker is not proof of actual publicity or persisted-input encryption.
func TestReviewedPublicInputsPreserveExactBytes(t *testing.T) {
	v := sampleInventory()
	v.Records[2].Content = `"public TXT value with exact spaces"`
	input, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadInventory(strings.NewReader(string(input)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Records[2].Content != v.Records[2].Content {
		t.Fatal("review marker rewrote DNS bytes")
	}
	m := &dnsMonitor{account: v.AccountID, name: domain}
	if err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerDNS(ctx, got) }, pulumi.WithMocks("prosody-dns", "unit", m)); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	calls, records := m.calls, m.count
	m.mu.Unlock()
	if calls != 1 || records != len(v.Records) {
		t.Fatal("reviewed public-input control did not reach expected graph")
	}
}
