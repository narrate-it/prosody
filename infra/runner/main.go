package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// main binds the reviewed host-local reconciler into a protected IaC resource.
// GitHub registration credentials are obtained inside the reconciler, never in state.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		c := config.New(ctx, "")
		root, err := filepath.Abs("../..")
		if err != nil {
			return fmt.Errorf("resolve runner source: %w", err)
		}
		registrationRoot := root
		if configured := c.Get("registrationSourceDirectory"); configured != "" {
			registrationRoot, err = filepath.Abs(configured)
			if err != nil {
				return fmt.Errorf("resolve registration source: %w", err)
			}
		}
		code, err := os.ReadFile(filepath.Join(registrationRoot, "scripts/reassign-prosody-runner.py"))
		if err != nil {
			return fmt.Errorf("read runner reconciler: %w", err)
		}
		currentCode, err := os.ReadFile(filepath.Join(root, "scripts/reassign-prosody-runner.py"))
		if err != nil || !bytes.Equal(code, currentCode) {
			return fmt.Errorf("registration source must match the reviewed reconciler")
		}
		digest := sha256.Sum256(code)
		var state pulumi.CustomResourceState
		err = ctx.RegisterResource("command:local:Command", "prosody-macbook-runner", pulumi.Map{
			"dir":    pulumi.String(registrationRoot),
			"create": pulumi.String("python3 scripts/reassign-prosody-runner.py --apply"),
			"update": pulumi.String("python3 scripts/reassign-prosody-runner.py --apply"),
			"delete": pulumi.String("python3 scripts/reassign-prosody-runner.py --refuse-destroy"),
			"environment": pulumi.StringMap{
				"PROSODY_RUNNER_DIR":           pulumi.String(c.Require("runnerDirectory")),
				"PROSODY_RUNNER_CUSTODY":       pulumi.String(c.Require("custodyDirectory")),
				"PROSODY_RUNNER_EXPECTED_HOST": pulumi.String(c.Require("expectedHost")),
			},
			"triggers": pulumi.StringArray{pulumi.String(hex.EncodeToString(digest[:]))},
		}, &state, pulumi.Version("1.3.0"), pulumi.Protect(true), pulumi.AdditionalSecretOutputs([]string{"stdout", "stderr"}))
		if err != nil || !c.GetBool("useGuiSecuritySession") {
			return err
		}
		sessionCode, err := os.ReadFile(filepath.Join(root, "scripts/runner-gui-session.py"))
		if err != nil {
			return fmt.Errorf("read session reconciler: %w", err)
		}
		sessionDigest := sha256.Sum256(append(code, sessionCode...))
		sessionCommand := "/usr/bin/python3 -I scripts/runner-gui-session.py --apply"
		if c.GetBool("resumeStoppedGuiSession") {
			sessionCommand = "/usr/bin/python3 -I scripts/runner-gui-session.py --resume-stopped"
		}
		var session pulumi.CustomResourceState
		return ctx.RegisterResource("command:local:Command", "prosody-runner-gui-session", pulumi.Map{
			"dir":    pulumi.String(root),
			"create": pulumi.String(sessionCommand),
			"update": pulumi.String(sessionCommand),
			"delete": pulumi.String("/usr/bin/python3 -I scripts/runner-gui-session.py --refuse-destroy"),
			"environment": pulumi.StringMap{
				"PROSODY_RUNNER_DIR":           pulumi.String(c.Require("runnerDirectory")),
				"PROSODY_RUNNER_GUI_CUSTODY":   pulumi.String(c.Require("guiSessionCustodyDirectory")),
				"PROSODY_RUNNER_EXPECTED_HOST": pulumi.String(c.Require("expectedHost")),
			},
			"triggers": pulumi.StringArray{pulumi.String(hex.EncodeToString(sessionDigest[:]))},
		}, &session, pulumi.Version("1.3.0"), pulumi.Protect(true),
			pulumi.DependsOn([]pulumi.Resource{&state}),
			pulumi.AdditionalSecretOutputs([]string{"stdout", "stderr"}))
	})
}
