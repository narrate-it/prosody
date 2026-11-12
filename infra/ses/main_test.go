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

type emailMonitor struct {
	mu       sync.Mutex
	account  string
	arn      string
	sesCount int
	calls    int
}

// Call supplies synthetic caller metadata and checks explicit provider routing.
func (m *emailMonitor) Call(a pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if a.Token != "aws:index/getCallerIdentity:getCallerIdentity" || a.Provider == "" {
		return nil, fmt.Errorf("unexpected or unbound account lookup")
	}
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return resource.NewPropertyMapFromMap(map[string]any{"accountId": m.account, "arn": m.arn}), nil
}

// NewResource rejects unexpected side effects and checks the protected scoped SES graph.
func (m *emailMonitor) NewResource(a pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if a.TypeToken == "pulumi:pulumi:Stack" {
		return "stack", a.Inputs, nil
	}
	if a.RegisterRPC == nil || a.RegisterRPC.GetVersion() != providerVersion {
		return "", nil, fmt.Errorf("provider version not pinned")
	}
	if a.TypeToken == "pulumi:providers:aws" {
		if a.Inputs["region"].StringValue() != "us-west-2" || len(a.Inputs["allowedAccountIds"].ArrayValue()) != 1 || a.Inputs["allowedAccountIds"].ArrayValue()[0].StringValue() != "123456789012" {
			return "", nil, fmt.Errorf("wrong provider account or region")
		}
		return "provider", a.Inputs, nil
	}
	if !a.RegisterRPC.GetProtect() || !a.RegisterRPC.GetRetainOnDelete() || a.Provider == "" {
		return "", nil, fmt.Errorf("missing protection, retention or explicit provider")
	}
	out := a.Inputs.Copy()
	switch a.TypeToken {
	case "aws:ses/domainIdentity:DomainIdentity":
		if !slices.Contains(a.RegisterRPC.GetAdditionalSecretOutputs(), "verificationToken") {
			return "", nil, fmt.Errorf("verification token secrecy missing")
		}
		if a.Inputs["domain"].StringValue() != domain {
			return "", nil, fmt.Errorf("wrong domain")
		}
		out["verificationToken"] = resource.NewStringProperty("synthetic-verification-token")
	case "aws:ses/domainDkim:DomainDkim":
		if !slices.Contains(a.RegisterRPC.GetAdditionalSecretOutputs(), "dkimTokens") {
			return "", nil, fmt.Errorf("DKIM token secrecy missing")
		}
		if a.Inputs["domain"].StringValue() != domain || len(a.RegisterRPC.GetDependencies()) != 1 {
			return "", nil, fmt.Errorf("DKIM domain or identity ordering missing")
		}
		if !strings.Contains(a.RegisterRPC.GetDependencies()[0], "domainIdentity") {
			return "", nil, fmt.Errorf("DKIM dependency is not the domain identity")
		}
		out["dkimTokens"] = resource.NewArrayProperty([]resource.PropertyValue{resource.NewStringProperty("first"), resource.NewStringProperty("second"), resource.NewStringProperty("third")})
	case "aws:ses/configurationSet:ConfigurationSet":
		if a.Inputs["name"].StringValue() != configurationSet {
			return "", nil, fmt.Errorf("wrong configuration set")
		}
	default:
		return "", nil, fmt.Errorf("unexpected non-SES resource")
	}
	m.mu.Lock()
	m.sesCount++
	m.mu.Unlock()
	return a.Name, out, nil
}

// TestTarget rejects incomplete or malformed selections before any cloud registration.
func TestTarget(t *testing.T) {
	for _, c := range []struct{ name, account, region string }{
		{"empty account", "", "us-west-2"}, {"short account", "123", "us-west-2"},
		{"account whitespace", " 123456789012", "us-west-2"}, {"account letters", "12345678901x", "us-west-2"},
		{"empty region", "123456789012", ""},
		{"regional DKIM suffix", "123456789012", "af-south-1"},
		{"unreviewed region", "123456789012", "us-east-1"}, {"uppercase region", "123456789012", "US-WEST-2"},
		{"region whitespace", "123456789012", "us-west-2\n"}, {"region overflow", "123456789012", strings.Repeat("a", 40)},
	} {
		t.Run(c.name, func(t *testing.T) {
			if validateTarget(c.account, c.region) == nil {
				t.Fatal("invalid target accepted")
			}
		})
	}
	if err := validateTarget("123456789012", "us-west-2"); err != nil {
		t.Fatal(err)
	}
}

// TestDKIM bounds and fixes names/targets without accepting provider-shaped arbitrary DNS.
func TestDKIM(t *testing.T) {
	for _, c := range []struct {
		name   string
		tokens []string
	}{
		{"absent", nil}, {"short", []string{"a", "b"}}, {"long", []string{"a", "b", "c", "d"}},
		{"duplicate", []string{"a", "a", "c"}}, {"empty", []string{"a", "", "c"}},
		{"foreign suffix", []string{"a.evil.invalid", "b", "c"}}, {"overlong label", []string{strings.Repeat("a", 64), "b", "c"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, e := dkimRecords(c.tokens)
			if e == nil || r != nil {
				t.Fatal("invalid tokens produced DNS")
			}
		})
	}
	r, err := dkimRecords([]string{"first", "second", "third"})
	if err != nil {
		t.Fatal(err)
	}
	for i, token := range []string{"first", "second", "third"} {
		if r[i] != (dnsRecord{Name: token + "._domainkey.prosody.chat", Type: "CNAME", Content: token + ".dkim.amazonses.com"}) {
			t.Fatal("DNS projection changed")
		}
	}
}

// TestEmailGraph verifies exactly three scoped SES resources and the single account read.
func TestEmailGraph(t *testing.T) {
	m := &emailMonitor{account: "123456789012"}
	if err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerEmail(ctx, "123456789012", "us-west-2") }, pulumi.WithMocks("email", "test", m)); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sesCount != 3 || m.calls != 1 {
		t.Fatalf("SES resources=%d account calls=%d", m.sesCount, m.calls)
	}
}

// TestForeignAccount proves that credential/account mismatch cannot create SES resources.
func TestForeignAccount(t *testing.T) {
	m := &emailMonitor{account: "999999999999"}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerEmail(ctx, "123456789012", "us-west-2") }, pulumi.WithMocks("email", "test", m))
	if err == nil || !strings.Contains(err.Error(), "SES account does not match selected account") {
		t.Fatalf("wrong account refusal: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sesCount != 0 || m.calls != 1 {
		t.Fatal("mismatch reached SES resources or skipped account read")
	}
}

// TestBadTargetRegistersNothing proves an invalid target stops before any provider
// registration or account read, not merely inside validateTarget.
func TestBadTargetRegistersNothing(t *testing.T) {
	m := &emailMonitor{account: "123456789012"}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerEmail(ctx, "123456789012", "us-east-1") }, pulumi.WithMocks("email", "test", m))
	if err == nil {
		t.Fatal("invalid region accepted")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sesCount != 0 || m.calls != 0 {
		t.Fatalf("bad target registered resources=%d calls=%d", m.sesCount, m.calls)
	}
}

// TestRootPrincipal proves the account check alone is insufficient: a root caller
// in the same account is refused before any SES resource.
func TestRootPrincipal(t *testing.T) {
	m := &emailMonitor{account: "123456789012", arn: "arn:aws:iam::123456789012:root"}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error { return registerEmail(ctx, "123456789012", "us-west-2") }, pulumi.WithMocks("email", "test", m))
	if err == nil || !strings.Contains(err.Error(), "not root") {
		t.Fatalf("root principal refusal: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sesCount != 0 || m.calls != 1 {
		t.Fatal("root reached SES resources or skipped account read")
	}
}
