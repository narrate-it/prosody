package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	providerVersion = "6.9.0"
	region          = "us-west"
	instanceType    = "g6-standard-1"
	maxComputeCents = 1200
	diskSizeMB      = 1300
)

type imageInfo struct {
	ID       string `pulumi:"id"`
	Status   string `pulumi:"status"`
	IsPublic *bool  `pulumi:"isPublic"`
	Size     int    `pulumi:"size"`
}

type price struct {
	Monthly float64 `pulumi:"monthly"`
}

type typeInfo struct {
	ID           string  `pulumi:"id"`
	Disk         int     `pulumi:"disk"`
	Prices       []price `pulumi:"prices"`
	RegionPrices []struct {
		ID      string  `pulumi:"id"`
		Monthly float64 `pulumi:"monthly"`
	} `pulumi:"regionPrices"`
}

// main declares staging resources only; a reviewed preview is required before apply.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		c := config.New(ctx, "")
		return registerStaging(ctx, c.Require("existingPrivateImageId"), c.RequireSecret("diskRootPassword"))
	})
}

// computePrice rejects incomplete, ambiguous or increased compute quotes.
// It does not quote image storage, data services, transfer, tax or the whole bill.
func computePrice(v typeInfo) (int, error) {
	if v.ID != instanceType || v.Disk < diskSizeMB || len(v.Prices) != 1 {
		return 0, fmt.Errorf("unexpected compute type or incomplete price")
	}
	monthly := v.Prices[0].Monthly
	found := false
	for _, p := range v.RegionPrices {
		if p.ID == region {
			if found {
				return 0, fmt.Errorf("ambiguous regional compute price")
			}
			monthly, found = p.Monthly, true
		}
	}
	cents := monthly * 100
	if math.IsNaN(cents) || math.IsInf(cents, 0) || cents <= 0 || cents > maxComputeCents || math.Abs(cents-math.Round(cents)) > 0.000001 {
		return 0, fmt.Errorf("compute price exceeds reviewed ceiling or is invalid")
	}
	return int(math.Round(cents)), nil
}

// numericID converts provider IDs into numeric child-resource references.
func numericID(v pulumi.IDOutput) pulumi.IntOutput {
	return v.ApplyT(func(id pulumi.ID) (int, error) {
		n, err := strconv.Atoi(string(id))
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid provider resource identity")
		}
		return n, nil
	}).(pulumi.IntOutput)
}

// registerStaging checks the accessible image and price before adding a protected,
// powered-off instance, image disk and direct-disk configuration to the graph.
// A readable image ID is not evidence of image provenance or boot compatibility.
func registerStaging(ctx *pulumi.Context, imageID string, password pulumi.StringOutput) error {
	if !regexp.MustCompile(`^private/[1-9][0-9]{0,15}$`).MatchString(imageID) {
		return fmt.Errorf("an explicit existing private image ID is required")
	}
	var image imageInfo
	if err := ctx.Invoke("linode:index/getImage:getImage", map[string]any{"id": imageID}, &image, pulumi.Version(providerVersion)); err != nil {
		return fmt.Errorf("read existing private image: %w", err)
	}
	if image.ID != imageID || image.Status != "available" || image.IsPublic == nil || *image.IsPublic || image.Size <= 0 || image.Size > diskSizeMB {
		return fmt.Errorf("image is not an available bounded private image")
	}
	var typ typeInfo
	if err := ctx.Invoke("linode:index/getInstanceType:getInstanceType", map[string]any{"id": instanceType}, &typ, pulumi.Version(providerVersion)); err != nil {
		return fmt.Errorf("read compute price: %w", err)
	}
	cents, err := computePrice(typ)
	if err != nil {
		return err
	}
	options := []pulumi.ResourceOption{pulumi.Protect(true), pulumi.Version(providerVersion)}
	var vm pulumi.CustomResourceState
	if err := ctx.RegisterResource("linode:index/instance:Instance", "prosody-api-staging", pulumi.Map{
		"label": pulumi.String("prosody-api-staging"), "region": pulumi.String(region),
		"type": pulumi.String(instanceType), "booted": pulumi.Bool(false),
		"backupsEnabled": pulumi.Bool(false), "privateIp": pulumi.Bool(true),
	}, &vm, options...); err != nil {
		return err
	}
	var disk pulumi.CustomResourceState
	if err := ctx.RegisterResource("linode:index/instanceDisk:InstanceDisk", "prosody-nanos-root", pulumi.Map{
		"label": pulumi.String("prosody-nanos-root"), "linodeId": numericID(vm.ID()),
		"image": pulumi.String(imageID), "size": pulumi.Int(diskSizeMB), "rootPass": pulumi.ToSecret(password),
	}, &disk, append(options, pulumi.AdditionalSecretOutputs([]string{"rootPass"}))...); err != nil {
		return err
	}
	var boot pulumi.CustomResourceState
	if err := ctx.RegisterResource("linode:index/instanceConfig:InstanceConfig", "prosody-nanos-direct-disk", pulumi.Map{
		"label": pulumi.String("prosody-nanos-direct-disk"), "linodeId": numericID(vm.ID()),
		"booted": pulumi.Bool(false), "kernel": pulumi.String("linode/direct-disk"),
		"virtMode": pulumi.String("paravirt"), "rootDevice": pulumi.String("/dev/sda"),
		"device":  pulumi.Array{pulumi.Map{"deviceName": pulumi.String("sda"), "diskId": numericID(disk.ID())}},
		"helpers": pulumi.Array{pulumi.Map{"distro": pulumi.Bool(false), "modulesDep": pulumi.Bool(false), "network": pulumi.Bool(true), "devtmpfsAutomount": pulumi.Bool(false), "updatedbDisabled": pulumi.Bool(false)}},
	}, &boot, options...); err != nil {
		return err
	}
	ctx.Export("computeMonthlyCents", pulumi.Int(cents))
	ctx.Export("instanceId", vm.ID())
	ctx.Export("bootProfileId", boot.ID())
	ctx.Export("status", pulumi.String("powered-off-staging-not-qualified"))
	return nil
}
