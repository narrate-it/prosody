package seslink

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/ajent-social/amos/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// envelopeTable is the consumer-owned protected retention journal. The runtime
// role holds SELECT and INSERT only: evidence is insert-only, conflicting
// evidence is retained for review, and nothing is ever overwritten.
const envelopeTable = "prosody_events.sns_envelopes"

// EnvelopeStore implements ses.RawSNSEventSource over protected PostgreSQL
// retention. It returns retained raw bytes only; authentication is exclusively
// the AMOS verified source's authority (LookupVerified reauthenticates).
type EnvelopeStore struct {
	pool *pgxpool.Pool
}

func openEnvelopeStore(ctx context.Context, db storage.RuntimeConfig) (*EnvelopeStore, error) {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(db.User, db.Password),
		Host:     fmt.Sprintf("%s:%d", db.Host, db.Port),
		Path:     db.Database,
		RawQuery: "sslmode=verify-full",
	}
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, fmt.Errorf("envelope store configuration: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(db.RootCAPEM) {
		return nil, errors.New("envelope store: database root CA PEM is not valid")
	}
	cfg.ConnConfig.TLSConfig = &tls.Config{RootCAs: roots, ServerName: db.Host, MinVersion: tls.VersionTLS12}
	cfg.ConnConfig.DialFunc = ipv4DialFunc()
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("envelope store: %w", err)
	}
	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := pool.Ping(readyCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("envelope store unreachable: %w", err)
	}
	var present *string
	if err := pool.QueryRow(readyCtx, `SELECT to_regclass($1)::text`, envelopeTable).Scan(&present); err != nil || present == nil {
		pool.Close()
		return nil, errors.New("SNS envelope retention store not provisioned; apply backend/seslink/schema.sql with owner custody")
	}
	return &EnvelopeStore{pool: pool}, nil
}

// ipv4DialFunc prefers IPv4: the managed database's IPv6 path can blackhole on
// operator networks (verified on the deployment host); a stalled IPv6 attempt
// must not hold startup hostage. It falls back to the default dial when IPv4
// is unavailable. TLS verification still uses the canonical hostname.
func ipv4DialFunc() func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if ips, lookupErr := net.DefaultResolver.LookupNetIP(ctx, "ip4", host); lookupErr == nil {
			for _, ip := range ips {
				if conn, dialErr := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(ip.String(), port)); dialErr == nil {
					return conn, nil
				}
			}
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

// LookupSNSEnvelope returns the most recently retained raw envelope for the
// job, or absent. With conflicting evidence the newest bytes are returned and
// the verified source reauthenticates them; conflicts degrade to unknown,
// never to a fabricated outcome. The passed deadline is honored by pgx.
func (s *EnvelopeStore) LookupSNSEnvelope(ctx context.Context, jobID uuid.UUID) ([]byte, bool, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx,
		`SELECT envelope FROM `+envelopeTable+` WHERE job_id = $1 ORDER BY received_at DESC, envelope_sha256 DESC LIMIT 1`,
		jobID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

// Retain stores authenticated raw evidence insert-only. Identical redelivery
// is idempotent; a different envelope for the same job is retained alongside
// the existing evidence (primary key job_id + envelope_sha256).
func (s *EnvelopeStore) Retain(ctx context.Context, jobID uuid.UUID, raw []byte) error {
	sum := sha256.Sum256(raw)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+envelopeTable+` (job_id, envelope_sha256, envelope) VALUES ($1, $2, $3) ON CONFLICT (job_id, envelope_sha256) DO NOTHING`,
		jobID, hex.EncodeToString(sum[:]), raw)
	return err
}

// Close releases the pool.
func (s *EnvelopeStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}
