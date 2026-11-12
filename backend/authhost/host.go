// Package authhost composes Prosody's native account host through AMOS public APIs.
// It is not an installer, credential loader, application worker or feed/media server.
package authhost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/ajent-social/amos/apphost"
	"github.com/ajent-social/amos/delivery/email/ses"
	"github.com/ajent-social/amos/identity/issuer"
	"github.com/ajent-social/amos/recovery/filestore"
	"github.com/ajent-social/amos/storage"
)

const (
	Origin   = "https://api.prosody.chat"
	Audience = "https://api.prosody.chat/"
	Sender   = "noreply@prosody.chat"
	ClientID = "prosody-ios"
	Callback = "run.prosody.prosody:/oauth/callback"
)

// Config contains operator-supplied runtime dependencies, never installation authority.
// Keys and credentials must come from separately admitted backend custody, not an image.
type Config struct {
	Database                      storage.RuntimeConfig
	Realm                         filestore.Scope
	RecoveryKeys                  *filestore.NativeSet
	CertificatePEM, PrivateKeyPEM []byte
	DirectoryCursorKey            []byte
	EmailSender                   *ses.Sender
}

var ErrConfiguration = errors.New("prosody account host configuration unavailable")

// productionConfig projects the explicit company18 recipe; it grants no authority.
// AMOS still validates the installation, native provider, roles, origin and TLS.
func productionConfig(c Config) apphost.ProductionConfig {
	db := c.Database
	db.RootCAPEM = append([]byte(nil), c.Database.RootCAPEM...)
	return apphost.ProductionConfig{
		Database:       db,
		InstallationID: c.Realm.InstallationID,
		ApplicationID:  c.Realm.ApplicationID,
		EnvironmentID:  c.Realm.EnvironmentID,
		Origin:         Origin,
		CertificatePEM: append([]byte(nil), c.CertificatePEM...),
		PrivateKeyPEM:  append([]byte(nil), c.PrivateKeyPEM...),
		RecoveryKeys:   c.RecoveryKeys,
		Directory:      &apphost.DirectoryConfig{CursorKey: append([]byte(nil), c.DirectoryCursorKey...)},
		Organization:   &apphost.OrganizationConfig{},
		Issuer: &apphost.IssuerConfig{Clients: []apphost.IssuerClient{{
			ID: ClientID, Name: "Prosody", Audience: Audience, Native: true,
			NativeReturnProfile: issuer.NativeReturnPrivateUseV1,
			RedirectURIs:        []string{Callback},
		}}},
		EmailFrom:   Sender,
		EmailSender: c.EmailSender,
		Home:        http.HandlerFunc(accountHome),
	}
}

// Run admits the native host and serves its account/issuer routes until it stops.
// The caller supplies an owned listener and retains cleanup responsibility on
// startup refusal; AMOS controls TLS serving, current authority and its sole worker.
// This function neither installs nor repairs schema, roles, keys or registrations.
func Run(ctx context.Context, c Config, listener net.Listener) (result error) {
	if ctx == nil || listener == nil || len(c.DirectoryCursorKey) != 32 || bytes.Equal(c.DirectoryCursorKey, make([]byte, 32)) {
		return ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	host, err := apphost.NewProduction(ctx, productionConfig(c))
	if err != nil {
		// Preserve AMOS's fixed configuration/unavailable sentinels for diagnosis.
		return errors.Join(ErrConfiguration, err)
	}
	defer func() { result = errors.Join(result, host.Close()) }()
	return host.Serve(ctx, listener)
}

const home = "<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><title>Prosody accounts</title><h1>Prosody account services</h1></html>"

// accountHome presents only the account service; it cannot impersonate a working feed.
func accountHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(home)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = io.WriteString(w, home)
	}
}
