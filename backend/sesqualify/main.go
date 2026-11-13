// Command sesqualify qualifies the real SES verified-event journey end to end:
// it composes the same consumer subsystem the host uses (seslink), sends one
// real sign-in email through Amazon SES to an operator-supplied verified
// recipient, ingests the signed SNS Send evidence, and reconciles the job to an
// accepted receipt. It never fabricates events; failure is reported visibly.
// This is a required-service qualification tool, not a mock harness.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ajent-social/amos/delivery/email"
	"github.com/ajent-social/amos/delivery/email/ses"
	"github.com/ajent-social/amos/storage"
	"github.com/google/uuid"
	"github.com/narrate-it/prosody/backend/seslink"
)

func main() {
	recipient := flag.String("recipient", "noreply@prosody.chat", "verified recipient for the qualification send")
	wait := flag.Duration("wait", 3*time.Minute, "maximum time to wait for verified reconciliation")
	flag.Parse()
	if err := run(*recipient, *wait); err != nil {
		fmt.Fprintln(os.Stderr, "sesqualify:", err)
		os.Exit(1)
	}
}

func run(recipient string, wait time.Duration) error {
	root := os.Getenv("PROSODY_PRIVATE")
	if root == "" {
		return errors.New("PROSODY_PRIVATE custody directory required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var inst struct {
		Scope struct {
			InstallationID string `json:"installationId"`
			ApplicationID  string `json:"applicationId"`
			EnvironmentID  string `json:"environmentId"`
		} `json:"scope"`
		Database    string `json:"database"`
		RuntimeRole string `json:"runtimeRole"`
	}
	if err := readJSON(filepath.Join(root, "db", "installation.json"), &inst); err != nil {
		return err
	}
	var roles struct {
		Roles map[string]string `json:"roles"`
		Host  string            `json:"host"`
		Port  uint16            `json:"port"`
	}
	if err := readJSON(filepath.Join(root, "db", "roles.json"), &roles); err != nil {
		return err
	}
	var ssl struct {
		CA string `json:"ca_certificate"`
	}
	if err := readJSON(filepath.Join(root, "db", "ssl.json"), &ssl); err != nil {
		return err
	}
	ca := []byte(ssl.CA)
	if decoded, err := base64.StdEncoding.DecodeString(ssl.CA); err == nil {
		ca = decoded
	}
	db := storage.RuntimeConfig{
		Host: roles.Host, Port: roles.Port, Database: inst.Database,
		User: inst.RuntimeRole, Password: roles.Roles[inst.RuntimeRole],
		RootCAPEM: ca, StartupTimeout: 30 * time.Second,
		MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: 5 * time.Minute, ConnMaxIdleTime: time.Minute,
	}
	c, err := seslink.LoadCustody(root)
	if err != nil {
		return err
	}
	realm := ses.Realm{
		InstallationID: uuid.MustParse(inst.Scope.InstallationID),
		ApplicationID:  uuid.MustParse(inst.Scope.ApplicationID),
		EnvironmentID:  uuid.MustParse(inst.Scope.EnvironmentID),
	}
	sub, err := seslink.Open(ctx, c, realm, db)
	if err != nil {
		return err
	}
	defer sub.Close()
	go sub.RunIngest(ctx)

	renderer, err := email.NewRenderer(email.RenderConfig{
		FromAddress: seslink.From, ApplicationOrigin: seslink.Origin, MaxBodyBytes: email.MaxBodyBytes,
	})
	if err != nil {
		return err
	}
	jobID := uuid.Must(uuid.NewV7())
	message, err := renderer.Render(jobID, email.Request{
		Template: email.TemplateSignIn, MaterialRef: email.SecretReference("material:" + uuid.Must(uuid.NewV7()).String()),
		ExpiresInSeconds: 600,
	}, email.PrivateMaterial{
		Recipient: recipient,
		ActionURL: seslink.Origin + "/magic-link?token=qualification-" + jobID.String(),
	})
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	fmt.Println("sesqualify: sending qualification job", jobID)
	receipt, err := sub.Sender.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("send: %w (state=%s)", err, stateName(receipt.State()))
	}
	fmt.Printf("sesqualify: send accepted by provider messageId=%s state=%s\n", receipt.MessageID(), stateName(receipt.State()))

	deadline := time.Now().Add(wait)
	for {
		reconciled, err := sub.Sender.Reconcile(ctx, jobID)
		if err == nil && reconciled.State() == email.ReceiptAccepted {
			fmt.Printf("sesqualify: VERIFIED ACCEPTED job=%s messageId=%s\n", jobID, reconciled.MessageID())
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("reconciliation did not reach accepted within %s (last state=%s err=%v)", wait, stateName(reconciled.State()), err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func stateName(s email.ReceiptState) string {
	switch s {
	case email.ReceiptAccepted:
		return "accepted"
	case email.ReceiptRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
