// Command host runs the Prosody native account host (company18 issuer) through
// the public AMOS composition. It loads operator custody, never installs or
// repairs schema/keys, and fails visibly when a runtime prerequisite is absent.
// The SES verified-event subsystem (SNS topic, envelope retention, signing-cert
// pins) and a public TLS certificate are operator prerequisites; this executable
// does not fabricate them.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ajent-social/amos/recovery/filestore"
	"github.com/ajent-social/amos/storage"
	"github.com/google/uuid"
	"github.com/narrate-it/prosody/backend/authhost"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "host:", err)
		os.Exit(1)
	}
}

type installation struct {
	Scope struct {
		InstallationID string `json:"installationId"`
		ApplicationID  string `json:"applicationId"`
		EnvironmentID  string `json:"environmentId"`
	} `json:"scope"`
	StoreID             string                       `json:"storeID"`
	StoreDir            string                       `json:"storeDir"`
	ActiveMaterialKeyID string                       `json:"activeMaterialKeyID"`
	MaterialKeys        map[string]map[string]string `json:"materialKeys"`
	RateKey             map[string]string            `json:"rateKey"`
	Database            string                       `json:"database"`
	RuntimeRole         string                       `json:"runtimeRole"`
}

func run() error {
	root := os.Getenv("PROSODY_PRIVATE")
	if root == "" {
		return errors.New("PROSODY_PRIVATE custody directory required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var inst installation
	if err := readJSON(filepath.Join(root, "db", "installation.json"), &inst); err != nil {
		return fmt.Errorf("installation metadata: %w", err)
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
	ca, err := decodeCA(ssl.CA)
	if err != nil {
		return err
	}

	set, err := openKeys(inst)
	if err != nil {
		return err
	}
	cert, key, err := loadTLS(filepath.Join(root, "tls"))
	if err != nil {
		return err
	}
	cursor, err := loadOrCreateCursor(filepath.Join(root, "cursor.key"))
	if err != nil {
		return err
	}
	db := storage.RuntimeConfig{
		Host: roles.Host, Port: roles.Port, Database: inst.Database,
		User: inst.RuntimeRole, Password: roles.Roles[inst.RuntimeRole],
		RootCAPEM: ca, StartupTimeout: 30 * time.Second,
		MaxOpenConns: 8, MaxIdleConns: 4, ConnMaxLifetime: 5 * time.Minute, ConnMaxIdleTime: time.Minute,
	}
	sub, err := openSES(ctx, root, inst, db)
	if err != nil {
		return err
	}
	defer sub.Close()
	// Ingestion authenticates and retains SES Send evidence before the issuer
	// reconciles any send; it runs for the lifetime of the host.
	go sub.RunIngest(ctx)

	cfg := authhost.Config{
		Database: db,
		Realm: filestore.Scope{
			InstallationID: uuid.MustParse(inst.Scope.InstallationID),
			ApplicationID:  uuid.MustParse(inst.Scope.ApplicationID),
			EnvironmentID:  uuid.MustParse(inst.Scope.EnvironmentID),
		},
		RecoveryKeys: set, CertificatePEM: cert, PrivateKeyPEM: key,
		DirectoryCursorKey: cursor, EmailSender: sub.Sender,
	}
	addr := os.Getenv("PROSODY_LISTEN")
	if addr == "" {
		addr = ":443"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() { _ = listener.Close() }()
	fmt.Println("host: serving native account host on", addr)
	return authhost.Run(ctx, cfg, listener)
}

func openKeys(inst installation) (*filestore.NativeSet, error) {
	storeID := uuid.MustParse(inst.StoreID)
	ref := func(m map[string]string) (filestore.Reference, error) {
		var hash [32]byte
		b, err := hex.DecodeString(m["versionSHA256"])
		if err != nil || len(b) != 32 {
			return filestore.Reference{}, errors.New("invalid key reference hash")
		}
		copy(hash[:], b)
		return filestore.Reference{StoreID: uuid.MustParse(m["storeId"]), EntryID: uuid.MustParse(m["entryId"]), VersionSHA256: hash}, nil
	}
	cfg := filestore.NativeConfig{
		Scope: filestore.Scope{
			InstallationID: uuid.MustParse(inst.Scope.InstallationID),
			ApplicationID:  uuid.MustParse(inst.Scope.ApplicationID),
			EnvironmentID:  uuid.MustParse(inst.Scope.EnvironmentID),
		},
		ActiveMaterialKeyID: inst.ActiveMaterialKeyID,
		MaterialKeys:        make(map[string]filestore.Reference),
	}
	for id, m := range inst.MaterialKeys {
		r, err := ref(m)
		if err != nil {
			return nil, err
		}
		cfg.MaterialKeys[id] = r
	}
	rate, err := ref(inst.RateKey)
	if err != nil {
		return nil, err
	}
	cfg.RateKey = rate
	store, err := filestore.Open(filestore.StoreConfig{Directory: inst.StoreDir, StoreID: storeID})
	if err != nil {
		return nil, fmt.Errorf("open key store: %w", err)
	}
	set, err := store.Native(context.Background(), cfg)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("admit native set: %w", err)
	}
	// The store must outlive the host; closure is owned by process exit.
	return set, nil
}

func loadTLS(dir string) ([]byte, []byte, error) {
	cert, err := os.ReadFile(filepath.Join(dir, "cert.pem"))
	if err != nil {
		return nil, nil, fmt.Errorf("public TLS certificate required at tls/cert.pem: %w", err)
	}
	key, err := os.ReadFile(filepath.Join(dir, "key.pem"))
	if err != nil {
		return nil, nil, fmt.Errorf("TLS private key required at tls/key.pem: %w", err)
	}
	return cert, key, nil
}

func loadOrCreateCursor(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		if len(data) != 32 {
			return nil, errors.New("directory cursor key must be 32 bytes")
		}
		return data, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, err
	}
	return key, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func decodeCA(value string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return []byte(value), nil
}
