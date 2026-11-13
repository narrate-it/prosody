package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const providerVersion = "6.23.2"

type existingZone struct {
	Name    string `pulumi:"name"`
	Status  string `pulumi:"status"`
	Type    string `pulumi:"type"`
	Account struct {
		ID string `pulumi:"id"`
	} `pulumi:"account"`
}

// main uses Pulumi's public Go registration API with the pinned provider schema.
// Only a separately reviewed update may adopt state or change Cloudflare.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		path := config.New(ctx, "").Require("inventoryFile")
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open private inventory: %w", err)
		}
		v, err := loadInventory(f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return fmt.Errorf("close private inventory: %w", closeErr)
		}
		return registerDNS(ctx, v)
	})
}

// registerDNS admits the exact existing zone before registering protected imports.
// Resource tokens and property names follow Cloudflare provider v6.23.2's schema.
func registerDNS(ctx *pulumi.Context, v inventory) error {
	if err := validateStateCustody(v); err != nil {
		return err
	}
	var zone existingZone
	if err := ctx.Invoke("cloudflare:index/getZone:getZone", map[string]any{"zoneId": v.ZoneID}, &zone, pulumi.Version(providerVersion)); err != nil {
		return fmt.Errorf("read existing zone: %w", err)
	}
	if zone.Name != domain || zone.Account.ID != v.AccountID || zone.Status != "active" || zone.Type != "full" {
		return fmt.Errorf("existing zone does not match the admitted active account/domain")
	}
	for _, r := range v.Records {
		args := pulumi.Map{"zoneId": pulumi.String(v.ZoneID), "name": pulumi.String(r.Name), "type": pulumi.String(r.Type), "content": pulumi.ToSecret(r.Content), "ttl": pulumi.Float64(r.TTL), "proxied": pulumi.Bool(r.Proxied), "tags": pulumi.ToSecret(pulumi.ToStringArray(r.Tags))}
		if r.Priority != nil {
			args["priority"] = pulumi.Int(*r.Priority)
		}
		if r.Comment != nil {
			args["comment"] = pulumi.ToSecret(*r.Comment)
		}
		s := pulumi.Map{}
		if r.Settings.FlattenCname != nil {
			s["flattenCname"] = pulumi.Bool(*r.Settings.FlattenCname)
		}
		if r.Settings.IPv4Only != nil {
			s["ipv4Only"] = pulumi.Bool(*r.Settings.IPv4Only)
		}
		if r.Settings.IPv6Only != nil {
			s["ipv6Only"] = pulumi.Bool(*r.Settings.IPv6Only)
		}
		if len(s) > 0 {
			args["settings"] = s
		}
		resName := "dns-" + r.ID
		opts := []pulumi.ResourceOption{pulumi.Protect(true), pulumi.RetainOnDelete(true), pulumi.AdditionalSecretOutputs([]string{"content", "comment", "tags"}), pulumi.Version(providerVersion)}
		if r.ID == "" {
			// Reviewed creation of a new record (e.g. the api A record once the
			// host IP exists); existing records remain protected imports.
			resName = "dns-create-" + strings.ToLower(r.Type) + "-" + strings.ReplaceAll(r.Name, ".", "-")
		} else {
			opts = append(opts, pulumi.Import(pulumi.ID(v.ZoneID+"/"+r.ID)))
		}
		var state pulumi.CustomResourceState
		if err := ctx.RegisterResource("cloudflare:index/dnsRecord:DnsRecord", resName, args, &state, opts...); err != nil {
			return fmt.Errorf("register protected DNS record: %w", err)
		}
	}
	ctx.Export("domain", pulumi.String(domain))
	ctx.Export("managedRecordCount", pulumi.Int(len(v.Records)))
	return nil
}
