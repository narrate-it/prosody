package seslink

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ajent-social/amos/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// dbCustody loads the operator database custody for the real-DB store test.
// The test is skipped visibly when custody is absent; it never uses mocks.
func dbCustody(t *testing.T) (storage.RuntimeConfig, map[string]string, bool) {
	t.Helper()
	root := os.Getenv("PROSODY_PRIVATE")
	if root == "" {
		return storage.RuntimeConfig{}, nil, false
	}
	read := func(name string, v any) error {
		data, err := os.ReadFile(filepath.Join(root, "db", name))
		if err != nil {
			return err
		}
		return json.Unmarshal(data, v)
	}
	var roles struct {
		Roles map[string]string `json:"roles"`
		Host  string            `json:"host"`
		Port  uint16            `json:"port"`
	}
	var ssl struct {
		CA string `json:"ca_certificate"`
	}
	var inst struct {
		Database    string `json:"database"`
		RuntimeRole string `json:"runtimeRole"`
	}
	if err := read("roles.json", &roles); err != nil {
		t.Fatalf("roles custody: %v", err)
	}
	if err := read("ssl.json", &ssl); err != nil {
		t.Fatalf("ssl custody: %v", err)
	}
	if err := read("installation.json", &inst); err != nil {
		t.Fatalf("installation custody: %v", err)
	}
	ca := []byte(ssl.CA)
	if decoded, err := base64.StdEncoding.DecodeString(ssl.CA); err == nil {
		ca = decoded
	}
	return storage.RuntimeConfig{
		Host: roles.Host, Port: roles.Port, Database: inst.Database,
		User: inst.RuntimeRole, Password: roles.Roles[inst.RuntimeRole], RootCAPEM: ca,
	}, roles.Roles, true
}

func TestEnvelopeStoreRealDB(t *testing.T) {
	db, rolePasswords, ok := dbCustody(t)
	if !ok {
		t.Skip("PROSODY_PRIVATE custody absent; real-DB envelope store test not run")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store, err := openEnvelopeStore(ctx, db)
	if err != nil {
		t.Fatalf("open envelope store: %v", err)
	}
	defer store.Close()

	jobA := uuid.Must(uuid.NewV7())
	jobB := uuid.Must(uuid.NewV7())
	first := []byte(`{"Type":"Notification","Message":"first-evidence"}`)
	conflict := []byte(`{"Type":"Notification","Message":"conflicting-evidence"}`)

	if raw, found, err := store.LookupSNSEnvelope(ctx, jobA); err != nil || found || raw != nil {
		t.Fatalf("absent job must return absent, got found=%v err=%v", found, err)
	}
	if err := store.Retain(ctx, jobA, first); err != nil {
		t.Fatalf("retain first: %v", err)
	}
	if err := store.Retain(ctx, jobA, first); err != nil {
		t.Fatalf("idempotent redelivery: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := store.Retain(ctx, jobA, conflict); err != nil {
		t.Fatalf("retain conflict: %v", err)
	}
	if err := store.Retain(ctx, jobB, first); err != nil {
		t.Fatalf("retain second job: %v", err)
	}
	raw, found, err := store.LookupSNSEnvelope(ctx, jobA)
	if err != nil || !found || !bytes.Equal(raw, conflict) {
		t.Fatalf("newest evidence must win, found=%v err=%v equal=%v", found, err, bytes.Equal(raw, conflict))
	}
	raw, found, err = store.LookupSNSEnvelope(ctx, jobB)
	if err != nil || !found || !bytes.Equal(raw, first) {
		t.Fatalf("second job evidence lost, found=%v err=%v", found, err)
	}

	// The runtime role must not be able to overwrite or delete evidence.
	if err := store.pool.QueryRow(ctx, `SELECT has_table_privilege($1, $2, 'UPDATE')`, db.User, envelopeTable).Scan(&ok); err != nil || ok {
		t.Fatalf("runtime must not hold UPDATE (ok=%v err=%v)", ok, err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT has_table_privilege($1, $2, 'DELETE')`, db.User, envelopeTable).Scan(&ok); err != nil || ok {
		t.Fatalf("runtime must not hold DELETE (ok=%v err=%v)", ok, err)
	}

	// Owner custody cleans the test rows; the runtime role never deletes.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	ownerConn(t, cleanupCtx, db, rolePasswords["prosody_owner"],
		`DELETE FROM `+envelopeTable+` WHERE job_id = ANY($1)`, []uuid.UUID{jobA, jobB})
}

func ownerConn(t *testing.T, ctx context.Context, db storage.RuntimeConfig, password, sql string, args ...any) {
	t.Helper()
	u := &url.URL{
		Scheme: "postgres", User: url.UserPassword("prosody_owner", password),
		Host: fmt.Sprintf("%s:%d", db.Host, db.Port), Path: db.Database, RawQuery: "sslmode=verify-full",
	}
	cfg, err := pgx.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(db.RootCAPEM) {
		t.Fatal("bad CA")
	}
	cfg.TLSConfig = &tls.Config{RootCAs: roots, ServerName: db.Host, MinVersion: tls.VersionTLS12}
	cfg.DialFunc = ipv4DialFunc()
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("owner cleanup connect: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("owner cleanup: %v", err)
	}
}
