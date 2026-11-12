package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	providerVersion  = "7.54.0"
	supportedRegion  = "us-west-2"
	domain           = "prosody.chat"
	sender           = "noreply@prosody.chat"
	configurationSet = "prosody-identity"
)

var (
	accountPattern   = regexp.MustCompile(`^[0-9]{12}$`)
	dkimTokenPattern = regexp.MustCompile(`^[a-z0-9]{1,63}$`)
)

type awsProvider struct{ pulumi.ProviderResourceState }
type identityState struct {
	pulumi.CustomResourceState
	VerificationToken pulumi.StringOutput `pulumi:"verificationToken"`
}
type dkimState struct {
	pulumi.CustomResourceState
	DkimTokens pulumi.StringArrayOutput `pulumi:"dkimTokens"`
}
type dnsRecord struct{ Name, Type, Content string }

// main requires an explicit account and region; preview and apply are separate effects.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		c := config.New(ctx, "")
		return registerEmail(ctx, c.Require("accountId"), c.Require("region"))
	})
}

// validateTarget rejects absent or noncanonical account/region selections before registration.
func validateTarget(account, region string) error {
	if !accountPattern.MatchString(account) || region != supportedRegion {
		return fmt.Errorf("an explicit AWS account and supported us-west-2 region are required")
	}
	return nil
}

// dkimRecords projects exactly three distinct provider tokens into fixed-domain DNS records.
func dkimRecords(tokens []string) ([]dnsRecord, error) {
	if len(tokens) != 3 {
		return nil, fmt.Errorf("expected three DKIM tokens")
	}
	seen := map[string]bool{}
	out := make([]dnsRecord, 0, 3)
	for _, token := range tokens {
		if !dkimTokenPattern.MatchString(token) || seen[token] {
			return nil, fmt.Errorf("invalid or duplicate DKIM token")
		}
		seen[token] = true
		out = append(out, dnsRecord{Name: token + "._domainkey." + domain, Type: "CNAME", Content: token + ".dkim.amazonses.com"})
	}
	return out, nil
}

// registerEmail declares only SES prerequisites, never sends mail or changes DNS.
// Account admission precedes all SES resources; DNS outputs remain encrypted in state.
func registerEmail(ctx *pulumi.Context, account, region string) error {
	if err := validateTarget(account, region); err != nil {
		return err
	}
	var provider awsProvider
	if err := ctx.RegisterResource("pulumi:providers:aws", "ses-account", pulumi.Map{
		"region": pulumi.String(region), "allowedAccountIds": pulumi.StringArray{pulumi.String(account)},
	}, &provider, pulumi.Version(providerVersion)); err != nil {
		return err
	}
	var caller struct {
		AccountID string `pulumi:"accountId"`
		Arn       string `pulumi:"arn"`
	}
	if err := ctx.Invoke("aws:index/getCallerIdentity:getCallerIdentity", map[string]any{}, &caller, pulumi.Provider(&provider), pulumi.Version(providerVersion)); err != nil {
		return fmt.Errorf("read SES account: %w", err)
	}
	if caller.AccountID != account {
		return fmt.Errorf("SES account does not match selected account")
	}
	if strings.HasSuffix(caller.Arn, ":root") {
		return fmt.Errorf("SES apply requires the scoped prosody-ses principal, not root")
	}
	opts := []pulumi.ResourceOption{pulumi.Provider(&provider), pulumi.Version(providerVersion), pulumi.Protect(true), pulumi.RetainOnDelete(true)}
	var identity identityState
	if err := ctx.RegisterResource("aws:ses/domainIdentity:DomainIdentity", "prosody-email-domain", pulumi.Map{"domain": pulumi.String(domain)}, &identity, append(opts, pulumi.AdditionalSecretOutputs([]string{"verificationToken"}))...); err != nil {
		return err
	}
	var dkim dkimState
	if err := ctx.RegisterResource("aws:ses/domainDkim:DomainDkim", "prosody-email-dkim", pulumi.Map{"domain": pulumi.String(domain)}, &dkim, append(opts, pulumi.DependsOn([]pulumi.Resource{&identity}), pulumi.AdditionalSecretOutputs([]string{"dkimTokens"}))...); err != nil {
		return err
	}
	var set pulumi.CustomResourceState
	if err := ctx.RegisterResource("aws:ses/configurationSet:ConfigurationSet", "prosody-email-events", pulumi.Map{"name": pulumi.String(configurationSet)}, &set, opts...); err != nil {
		return err
	}
	ctx.Export("sender", pulumi.String(sender))
	ctx.Export("configurationSet", pulumi.String(configurationSet))
	ctx.Export("region", pulumi.String(region))
	ctx.Export("verificationTXT", pulumi.ToSecret(identity.VerificationToken.ApplyT(func(token string) (map[string]string, error) {
		if token == "" || len(token) > 512 {
			return nil, fmt.Errorf("invalid verification token")
		}
		return map[string]string{"name": "_amazonses." + domain, "type": "TXT", "content": token}, nil
	})))
	ctx.Export("dkimCNAMEs", pulumi.ToSecret(dkim.DkimTokens.ApplyT(func(tokens []string) ([]map[string]string, error) {
		records, err := dkimRecords(tokens)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]string, 0, len(records))
		for _, r := range records {
			out = append(out, map[string]string{"name": r.Name, "type": r.Type, "content": r.Content})
		}
		return out, nil
	})))
	ctx.Export("status", pulumi.String("ses-prerequisites-only-not-verified-or-sending"))
	return nil
}
