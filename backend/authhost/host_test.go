package authhost

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajent-social/amos/identity/issuer"
	"github.com/google/uuid"
)

// TestCompanyRecipe checks configuration projection only, never native admission.
func TestCompanyRecipe(t *testing.T) {
	c := Config{DirectoryCursorKey: make([]byte, 32), CertificatePEM: []byte("synthetic certificate"), PrivateKeyPEM: []byte("synthetic key")}
	c.Database.RootCAPEM = []byte("synthetic CA")
	c.Realm.InstallationID = uuid.MustParse("019a1a00-0000-7000-8000-000000000001")
	c.Realm.ApplicationID = uuid.MustParse("019a1a00-0000-7000-8000-000000000002")
	c.Realm.EnvironmentID = uuid.MustParse("019a1a00-0000-7000-8000-000000000003")
	p := productionConfig(c)
	if p.InstallationID != c.Realm.InstallationID || p.ApplicationID != c.Realm.ApplicationID || p.EnvironmentID != c.Realm.EnvironmentID {
		t.Fatal("realm projection changed")
	}
	if p.Directory == nil || p.Organization == nil || p.Issuer == nil || p.Home == nil {
		t.Fatal("company18 profile incomplete")
	}
	if p.NativeMethods != nil || p.OrganizationCreation != nil || p.MCP != nil || p.NativeAcceptedWorkRuntime != nil || len(p.NativeExtensions) != 0 || len(p.NativeFiniteReads) != 0 || p.NativeApplication != nil || p.Application != nil || p.Protocol != nil || p.NativePersonalOperations != nil || p.NativePersonalTOTP != nil || p.NativePersonalCheckout != nil || len(p.Routes) != 0 || len(p.AuthenticatedConsumerCallbacks) != 0 {
		t.Fatal("unselected host capabilities enabled")
	}
	if p.MaterialActiveKeyID != "" || len(p.MaterialKeys) != 0 || len(p.RateKey) != 0 || p.BusinessNavigation != nil || p.NativeDocumentRoles != nil || p.NativeExtensionsProfile != 0 {
		t.Fatal("alternate key-material or document/navigation profile enabled")
	}
	if p.Origin != Origin || p.EmailFrom != Sender || len(p.Issuer.Clients) != 1 {
		t.Fatal("application binding changed")
	}
	client := p.Issuer.Clients[0]
	if client.ID != ClientID || client.Name != "Prosody" || client.Audience != Audience || !client.Native || client.NativeReturnProfile != issuer.NativeReturnPrivateUseV1 || len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != Callback || client.SecretSHA256 != ([32]byte{}) {
		t.Fatal("native client binding changed")
	}
	if _, err := issuer.CopyConfig(issuer.Config{Clients: p.Issuer.Clients}); err != nil {
		t.Fatal(err)
	}
	c.Database.RootCAPEM[0] = 'X'
	c.DirectoryCursorKey[0] = 1
	c.CertificatePEM[0] = 'X'
	c.PrivateKeyPEM[0] = 'X'
	if p.Database.RootCAPEM[0] == 'X' || p.Directory.CursorKey[0] != 0 || p.CertificatePEM[0] == 'X' || p.PrivateKeyPEM[0] == 'X' {
		t.Fatal("configuration aliases caller byte buffers")
	}
}

// TestHome distinguishes account presentation from undeployed application routes.
func TestHome(t *testing.T) {
	for _, c := range []struct {
		name, method, path string
		status             int
		body               bool
	}{
		{"get", http.MethodGet, "/", http.StatusOK, true},
		{"head", http.MethodHead, "/", http.StatusOK, false},
		{"method", http.MethodPost, "/", http.StatusMethodNotAllowed, true},
		{"missing feed", http.MethodGet, "/prosody/v1/feeds", http.StatusNotFound, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			accountHome(w, httptest.NewRequest(c.method, c.path, nil))
			if w.Code != c.status || (w.Body.Len() > 0) != c.body {
				t.Fatal("unexpected account presentation response")
			}
			if c.status == http.StatusOK && (w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Security-Policy") != "default-src 'none'; frame-ancestors 'none'") {
				t.Fatal("presentation security headers changed")
			}
		})
	}
}

// TestRunRejectsMissingListener verifies local refusal before any native dependency.
func TestRunRejectsMissingListener(t *testing.T) {
	if err := Run(context.Background(), Config{}, nil); !errors.Is(err, ErrConfiguration) {
		t.Fatal("missing listener did not refuse")
	}
	//nolint:staticcheck // deliberately nil to verify Run rejects a nil context
	if err := Run(nil, Config{}, nil); !errors.Is(err, ErrConfiguration) {
		t.Fatal("nil context did not refuse")
	}
}

// TestRunConfigurationGuards proves the directory-cursor-key guard in isolation.
// AMOS itself only checks len(key)==32, so Run's non-zero check is the sole defense
// against a degenerate key. A real listener is present so the key predicate (not the
// missing-listener predicate) is what refuses; all cases reject before NewProduction,
// fabricating no native admission.
func TestRunConfigurationGuards(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	validKey := bytes.Repeat([]byte{1}, 32)
	for _, c := range []struct {
		name string
		key  []byte
		lis  net.Listener
	}{
		{"zero key", make([]byte, 32), listener},
		{"nil key", nil, listener},
		{"short key", make([]byte, 31), listener},
		{"long key", make([]byte, 33), listener},
		{"valid key nil listener", validKey, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			// Assert sentinel identity, not errors.Is: the guard returns the bare
			// ErrConfiguration, while a fall-through to NewProduction returns
			// errors.Join(ErrConfiguration, ...). errors.Is matches both and would
			// survive deletion of the guard; identity proves the guard itself fires.
			if err := Run(context.Background(), Config{DirectoryCursorKey: c.key}, c.lis); err != ErrConfiguration {
				t.Fatalf("expected bare ErrConfiguration guard refusal, got %v", err)
			}
		})
	}
	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := Run(ctx, Config{DirectoryCursorKey: validKey}, listener); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
