package main

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type stagingMonitor struct {
	mu           sync.Mutex
	count        int
	calls        int
	image        map[string]any
	custodyImage map[string]any
	monthly      float64
	booted       bool
}

// Call supplies explicit synthetic image and price results, never cloud evidence.
func (m *stagingMonitor) Call(a pulumi.MockCallArgs) (resource.PropertyMap, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	switch a.Token {
	case "linode:index/getImage:getImage":
		if a.Args["id"].StringValue() == m.custodyImage["id"] {
			return resource.NewPropertyMapFromMap(m.custodyImage), nil
		}
		return resource.NewPropertyMapFromMap(m.image), nil
	case "linode:index/getInstanceType:getInstanceType":
		return resource.NewPropertyMapFromMap(map[string]any{"id": instanceType, "disk": 51200, "prices": []any{map[string]any{"monthly": m.monthly}}}), nil
	default:
		return nil, fmt.Errorf("unexpected invoke")
	}
}

// NewResource checks the actual registration graph, including child references,
// explicit offline state, secret password handling and the direct-disk profile.
func (m *stagingMonitor) NewResource(a pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if a.TypeToken == "pulumi:pulumi:Stack" {
		return "stack", a.Inputs, nil
	}
	if a.RegisterRPC == nil || !a.RegisterRPC.GetProtect() || a.RegisterRPC.GetVersion() != providerVersion {
		return "", nil, fmt.Errorf("resource lacks protection or pinned provider")
	}
	id := ""
	switch a.TypeToken {
	case "linode:index/instance:Instance":
		if a.Inputs["booted"].BoolValue() != m.booted || a.Inputs["backupsEnabled"].BoolValue() || !a.Inputs["privateIp"].BoolValue() || a.Inputs["region"].StringValue() != region || a.Inputs["type"].StringValue() != instanceType {
			return "", nil, fmt.Errorf("unexpected instance plan or boot state")
		}
		id = "100"
	case "linode:index/instanceDisk:InstanceDisk":
		if a.Inputs["linodeId"].NumberValue() != 100 || !a.Inputs["rootPass"].IsSecret() || !slices.Contains(a.RegisterRPC.GetAdditionalSecretOutputs(), "rootPass") {
			return "", nil, fmt.Errorf("disk binding or secret handling changed")
		}
		switch a.Inputs["image"].StringValue() {
		case "private/123":
			if a.Inputs["size"].NumberValue() != diskSizeMB {
				return "", nil, fmt.Errorf("root disk size changed")
			}
			id = "200"
		case "private/456":
			if a.Inputs["size"].NumberValue() != 64 || a.Inputs["label"].StringValue() != "prosody-custody" {
				return "", nil, fmt.Errorf("custody disk must match the bounded custody image")
			}
			id = "201"
		default:
			return "", nil, fmt.Errorf("unexpected disk image")
		}
	case "linode:index/instanceConfig:InstanceConfig":
		if a.Inputs["linodeId"].NumberValue() != 100 || a.Inputs["booted"].BoolValue() != m.booted || a.Inputs["kernel"].StringValue() != "linode/direct-disk" || a.Inputs["virtMode"].StringValue() != "paravirt" || a.Inputs["rootDevice"].StringValue() != "/dev/sda" {
			return "", nil, fmt.Errorf("boot configuration is not the reviewed direct-disk profile")
		}
		devices := a.Inputs["device"].ArrayValue()
		if len(devices) != 2 || devices[0].ObjectValue()["deviceName"].StringValue() != "sda" || devices[0].ObjectValue()["diskId"].NumberValue() != 200 ||
			devices[1].ObjectValue()["deviceName"].StringValue() != "sdb" || devices[1].ObjectValue()["diskId"].NumberValue() != 201 {
			return "", nil, fmt.Errorf("root/custody device mapping changed")
		}
		helpers := a.Inputs["helpers"].ArrayValue()
		if len(helpers) != 1 || len(helpers[0].ObjectValue()) != 5 {
			return "", nil, fmt.Errorf("boot helper profile changed")
		}
		for key, expected := range map[string]bool{"distro": false, "modulesDep": false, "network": true, "devtmpfsAutomount": false, "updatedbDisabled": false} {
			value := helpers[0].ObjectValue()[resource.PropertyKey(key)]
			if !value.IsBool() || value.BoolValue() != expected {
				return "", nil, fmt.Errorf("boot helper mapping changed")
			}
		}
		id = "300"
	default:
		return "", nil, fmt.Errorf("unexpected resource")
	}
	m.mu.Lock()
	m.count++
	m.mu.Unlock()
	return id, a.Inputs, nil
}

// sampleMonitor describes an available synthetic image and the observed list price.
func sampleMonitor() *stagingMonitor {
	return &stagingMonitor{
		image:        map[string]any{"id": "private/123", "status": "available", "isPublic": false, "size": 100},
		custodyImage: map[string]any{"id": "private/456", "status": "available", "isPublic": false, "size": 64},
		monthly:      12,
	}
}

// TestStagingGraph keeps boot disabled while checking exact child dependencies.
func TestStagingGraph(t *testing.T) {
	m := sampleMonitor()
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return registerStaging(ctx, "private/123", "private/456", pulumi.ToSecret("synthetic-not-a-real-password").(pulumi.StringOutput), false)
	}, pulumi.WithMocks("prosody-linode", "unit", m))
	if err != nil || m.count != 4 {
		t.Fatalf("staging graph failed: %v; resources %d", err, m.count)
	}
}

// TestBootedGraph admits the explicit reviewed boot selection on both the
// instance and the configuration profile, with the custody device attached.
func TestBootedGraph(t *testing.T) {
	m := sampleMonitor()
	m.booted = true
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return registerStaging(ctx, "private/123", "private/456", pulumi.ToSecret("synthetic").(pulumi.StringOutput), true)
	}, pulumi.WithMocks("prosody-linode", "unit", m))
	if err != nil || m.count != 4 {
		t.Fatalf("booted graph failed: %v; resources %d", err, m.count)
	}
}

// TestStagingRefusals prevents incomplete image/price observations adding resources.
func TestStagingRefusals(t *testing.T) {
	for _, name := range []string{"public", "missing-private-flag", "pending", "oversized", "missing-size", "foreign-id", "price-increase", "zero-price", "invalid-id", "custody-public", "custody-oversized", "custody-invalid-id"} {
		t.Run(name, func(t *testing.T) {
			m := sampleMonitor()
			id, custodyID := "private/123", "private/456"
			switch name {
			case "public":
				m.image["isPublic"] = true
			case "missing-private-flag":
				delete(m.image, "isPublic")
			case "pending":
				m.image["status"] = "pending"
			case "oversized":
				m.image["size"] = diskSizeMB + 1
			case "missing-size":
				delete(m.image, "size")
			case "foreign-id":
				m.image["id"] = "private/456"
			case "price-increase":
				m.monthly = 12.01
			case "zero-price":
				m.monthly = 0
			case "invalid-id":
				id = "linode/debian12"
			case "custody-public":
				m.custodyImage["isPublic"] = true
			case "custody-oversized":
				m.custodyImage["size"] = custodyDiskMaxMB + 1
			case "custody-invalid-id":
				custodyID = "linode/debian12"
			}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				return registerStaging(ctx, id, custodyID, pulumi.ToSecret("synthetic").(pulumi.StringOutput), false)
			}, pulumi.WithMocks("prosody-linode", "unit", m))
			want, calls := "image is not an available bounded private image", 1
			switch name {
			case "invalid-id":
				want, calls = "an explicit existing private image ID is required", 0
			case "price-increase", "zero-price":
				want, calls = "compute price exceeds reviewed ceiling or is invalid", 3
			case "custody-public", "custody-oversized":
				want, calls = "custody image is not an available bounded private image", 2
			case "custody-invalid-id":
				want, calls = "an explicit existing private custody image ID is required", 1
			}
			if err == nil || !strings.Contains(err.Error(), want) || m.count != 0 || m.calls != calls {
				t.Fatalf("expected refusal %q, zero resources and %d lookups; got %v, %d resources and %d lookups", want, calls, err, m.count, m.calls)
			}
		})
	}
}

// TestComputePriceBounds checks finite exact-cent quotes and the fixed compute cap.
func TestComputePriceBounds(t *testing.T) {
	for _, value := range []float64{0, -1, 12.01, 1.001, math.NaN(), math.Inf(1)} {
		if _, err := computePrice(typeInfo{ID: instanceType, Disk: diskSizeMB, Prices: []price{{Monthly: value}}}); err == nil {
			t.Fatal("invalid or increased quote accepted")
		}
	}
	got, err := computePrice(typeInfo{ID: instanceType, Disk: diskSizeMB, Prices: []price{{Monthly: 12}}})
	if err != nil || got != 1200 {
		t.Fatal("exact reviewed quote rejected")
	}
}

// TestRegionalComputePrice checks override precedence and incomplete quote refusal.
func TestRegionalComputePrice(t *testing.T) {
	for _, name := range []string{"increase", "discount", "foreign-region", "duplicate-region", "missing-base", "multiple-base"} {
		t.Run(name, func(t *testing.T) {
			v := typeInfo{ID: instanceType, Disk: diskSizeMB, Prices: []price{{Monthly: 12}}}
			regional := struct {
				ID      string  `pulumi:"id"`
				Monthly float64 `pulumi:"monthly"`
			}{ID: region, Monthly: 11.5}
			want, denied := 1150, false
			switch name {
			case "increase":
				regional.Monthly, denied = 12.01, true
			case "foreign-region":
				regional.ID, regional.Monthly, want = "other-region", 99, 1200
			case "duplicate-region":
				v.RegionPrices = append(v.RegionPrices, regional)
				denied = true
			case "missing-base":
				v.Prices, denied = nil, true
			case "multiple-base":
				v.Prices, denied = append(v.Prices, price{Monthly: 1}), true
			}
			v.RegionPrices = append(v.RegionPrices, regional)
			got, err := computePrice(v)
			if denied {
				if err == nil {
					t.Fatal("increased or ambiguous quote accepted")
				}
			} else if err != nil || got != want {
				t.Fatal("regional quote precedence changed")
			}
		})
	}
}
