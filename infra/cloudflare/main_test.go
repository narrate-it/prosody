package main

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type dnsMonitor struct {
	account string
	name    string
	mu      sync.Mutex
	count   int
	calls   int
}

// Call simulates only the exact zone lookup; it supplies no real Cloudflare proof.
func (m *dnsMonitor) Call(a pulumi.MockCallArgs) (resource.PropertyMap, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if a.Token != "cloudflare:index/getZone:getZone" {
		return nil, fmt.Errorf("unexpected invoke")
	}
	return resource.NewPropertyMapFromMap(map[string]any{"name": m.name, "status": "active", "type": "full", "account": map[string]any{"id": m.account}}), nil
}

// NewResource checks the actual Pulumi registration RPC rather than calling a
// cloud provider. It asserts import identity, protection and desired secret flags,
// not persisted canonical-input encryption.
func (m *dnsMonitor) NewResource(a pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if a.TypeToken != "cloudflare:index/dnsRecord:DnsRecord" {
		return a.Name, a.Inputs, nil
	}
	if a.RegisterRPC == nil || !a.RegisterRPC.GetProtect() || !a.RegisterRPC.GetRetainOnDelete() || a.RegisterRPC.GetVersion() != providerVersion {
		return "", nil, fmt.Errorf("missing protected pinned import options")
	}
	for _, field := range []string{"content", "comment", "tags"} {
		if !slices.Contains(a.RegisterRPC.GetAdditionalSecretOutputs(), field) {
			return "", nil, fmt.Errorf("missing secret output flag")
		}
	}
	if a.ID == "" {
		if !strings.HasPrefix(a.Name, "dns-create-") {
			return "", nil, fmt.Errorf("created record lacks stable create identity")
		}
	} else if !strings.HasPrefix(a.ID, strings.Repeat("a", 32)+"/") || a.Name != "dns-"+strings.TrimPrefix(a.ID, strings.Repeat("a", 32)+"/") {
		return "", nil, fmt.Errorf("wrong stable import identity")
	}
	if !a.Inputs["content"].IsSecret() || !a.Inputs["tags"].IsSecret() {
		return "", nil, fmt.Errorf("desired DNS inputs not marked secret")
	}

	if a.Inputs["type"].StringValue() == "MX" {
		p := a.Inputs["priority"]
		if !p.IsNumber() || p.NumberValue() != 1 {
			return "", nil, fmt.Errorf("mail priority changed")
		}
	}
	if a.Inputs["type"].StringValue() == "CNAME" {
		s := a.Inputs["settings"]
		if !s.IsObject() || !s.ObjectValue()["flattenCname"].IsBool() || s.ObjectValue()["flattenCname"].BoolValue() {
			return "", nil, fmt.Errorf("CNAME settings changed")
		}
		if !a.Inputs["comment"].IsSecret() || a.Inputs["comment"].SecretValue().Element.StringValue() != "" {
			return "", nil, fmt.Errorf("empty comment lost or desired secret flag missing")
		}
	}
	m.mu.Lock()
	m.count++
	m.mu.Unlock()
	return a.ID, a.Inputs, nil
}

// TestProtectedRegistration checks Pulumi resource options without cloud effects.
func TestProtectedRegistration(t *testing.T) {
	v := sampleInventory()
	m := &dnsMonitor{account: v.AccountID, name: domain}
	if err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerDNS(ctx, v) }, pulumi.WithMocks("prosody-dns", "unit", m)); err != nil {
		t.Fatal(err)
	}
	if m.count != len(v.Records) {
		t.Fatal("records missing from resource graph")
	}
}

// TestCreateRegistration admits a reviewed new record without an import ID.
func TestCreateRegistration(t *testing.T) {
	v := sampleInventory()
	v.Records = append(v.Records, record{StateInputClass: publicStateInputClass, Name: "api." + domain, Type: "A", Content: "192.0.2.9", TTL: 300, Tags: []string{}})
	m := &dnsMonitor{account: v.AccountID, name: domain}
	if err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerDNS(ctx, v) }, pulumi.WithMocks("prosody-dns", "unit", m)); err != nil {
		t.Fatal(err)
	}
	if m.count != len(v.Records) {
		t.Fatal("created record missing from resource graph")
	}
}

// TestForeignZoneStopsRegistration keeps a wrong account/domain out of the graph.
func TestForeignZoneStopsRegistration(t *testing.T) {
	v := sampleInventory()
	for _, m := range []*dnsMonitor{{account: v.AccountID, name: "other.example"}, {account: strings.Repeat("f", 32), name: domain}} {
		if err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerDNS(ctx, v) }, pulumi.WithMocks("prosody-dns", "unit", m)); err == nil || m.count != 0 {
			t.Fatal("foreign zone registered")
		}
	}
}
