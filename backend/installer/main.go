// Command installer provisions the Prosody company18 installation through the
// public AMOS operator path: it generates an owner-custodied native key set,
// then runs identitybootstrap.InstallFresh against the prepared empty database.
// It prints only non-secret receipts; credentials and key material stay in the
// private custody directory. This is operator tooling, not a runtime host.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ajent-social/amos/operations/identitybootstrap"
	"github.com/ajent-social/amos/recovery/filestore"
	"github.com/ajent-social/amos/storage"
	"github.com/google/uuid"
)

type dbCustody struct {
	Database string            `json:"database"`
	Roles    map[string]string `json:"roles"`
	Host     string            `json:"host"`
	Port     uint16            `json:"port"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "installer:", err)
		os.Exit(1)
	}
}

func run() error {
	root := os.Getenv("PROSODY_PRIVATE")
	if root == "" {
		return fmt.Errorf("PROSODY_PRIVATE custody directory required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var cust dbCustody
	if err := readJSON(filepath.Join(root, "db", "roles.json"), &cust); err != nil {
		return err
	}
	var ssl struct {
		CA string `json:"ca_certificate"`
	}
	if err := readJSON(filepath.Join(root, "db", "ssl.json"), &ssl); err != nil {
		return err
	}
	ca, err := decodeCA(ssl.CA)
	if err != nil {
		return err
	}

	// Fresh owner-custodied native key set.
	keysDir := filepath.Join(root, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return err
	}
	storeDir := filepath.Join(keysDir, "store")
	scope := filestore.Scope{InstallationID: mustID(), ApplicationID: mustID(), EnvironmentID: mustID()}
	storeID := mustID()
	genCfg := filestore.GenerateConfig{
		Directory: storeDir, StoreID: storeID, Scope: scope,
		ActiveMaterialKeyID: "material-v1", MaterialKeyIDs: []string{"material-v1"},
	}
	nativeCfg, err := filestore.GenerateNative(ctx, genCfg)
	if err != nil {
		return fmt.Errorf("generate native keys: %w", err)
	}
	store, err := filestore.Open(filestore.StoreConfig{Directory: storeDir, StoreID: storeID})
	if err != nil {
		return fmt.Errorf("open key store: %w", err)
	}
	defer func() { _ = store.Close() }()
	set, err := store.Native(ctx, nativeCfg)
	if err != nil {
		return fmt.Errorf("admit native set: %w", err)
	}

	owner := storage.RuntimeConfig{
		Host: cust.Host, Port: cust.Port, Database: cust.Database,
		User: "prosody_owner", Password: cust.Roles["prosody_owner"],
		RootCAPEM: ca, StartupTimeout: 30 * time.Second,
		MaxOpenConns: 4, MaxIdleConns: 2, ConnMaxLifetime: 5 * time.Minute, ConnMaxIdleTime: time.Minute,
	}
	cfg := identitybootstrap.Config{
		Database: owner, RuntimeRole: "prosody_runtime", FounderRole: "prosody_founder", LabelsRole: "prosody_labels",
		Scope: scope, OrganizationCapacityID: mustID(), RecoveryKeys: set,
	}
	receipt, err := identitybootstrap.InstallFresh(ctx, cfg)
	if err != nil {
		return fmt.Errorf("install fresh: %w", err)
	}

	// Persist non-secret installation metadata for the runtime host.
	meta := map[string]any{
		"version": receipt.Version, "operation": receipt.Operation, "state": receipt.State,
		"created": receipt.Created, "requestSHA256": receipt.RequestSHA256, "profileSHA256": receipt.ProfileSHA256,
		"scope": map[string]string{
			"installationId": scope.InstallationID.String(), "applicationId": scope.ApplicationID.String(), "environmentId": scope.EnvironmentID.String(),
		},
		"storeID": storeID.String(), "storeDir": storeDir,
		"activeMaterialKeyID": nativeCfg.ActiveMaterialKeyID,
		"materialKeys":        refsJSON(nativeCfg.MaterialKeys),
		"rateKey":             refJSON(nativeCfg.RateKey),
		"database":            cust.Database, "runtimeRole": "prosody_runtime",
	}
	out := filepath.Join(root, "db", "installation.json")
	data, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(out, data, 0600); err != nil {
		return err
	}
	fmt.Printf("installation %s state=%s created=%v\n", receipt.Version, receipt.State, receipt.Created)
	fmt.Printf("installationId=%s\n", scope.InstallationID)
	fmt.Printf("metadata written to %s\n", out)
	return nil
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func refJSON(r filestore.Reference) map[string]string {
	return map[string]string{"storeId": r.StoreID.String(), "entryId": r.EntryID.String(), "versionSHA256": fmt.Sprintf("%x", r.VersionSHA256)}
}

func refsJSON(m map[string]filestore.Reference) map[string]map[string]string {
	out := make(map[string]map[string]string, len(m))
	for k, r := range m {
		out[k] = refJSON(r)
	}
	return out
}
