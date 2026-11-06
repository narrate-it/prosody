package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// main owns only a main-branch GitHub environment, never secret values or signing.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		root, err := filepath.Abs("../..")
		if err != nil {
			return fmt.Errorf("resolve environment source: %w", err)
		}
		code, err := os.ReadFile(filepath.Join(root, "scripts/configure-asc-environment.py"))
		if err != nil {
			return fmt.Errorf("read environment reconciler: %w", err)
		}
		digest := sha256.Sum256(code)
		var state pulumi.CustomResourceState
		return ctx.RegisterResource("command:local:Command", "prosody-asc-main-only", pulumi.Map{
			"dir":      pulumi.String(root),
			"create":   pulumi.String("/usr/bin/python3 -I scripts/configure-asc-environment.py --apply"),
			"update":   pulumi.String("/usr/bin/python3 -I scripts/configure-asc-environment.py --apply"),
			"delete":   pulumi.String("/usr/bin/python3 -I scripts/configure-asc-environment.py --refuse-destroy"),
			"triggers": pulumi.StringArray{pulumi.String(hex.EncodeToString(digest[:]))},
		}, &state, pulumi.Version("1.3.0"), pulumi.Protect(true), pulumi.AdditionalSecretOutputs([]string{"stdout", "stderr"}))
	})
}
