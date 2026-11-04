package openrouter

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// testOriginTransport refuses non-owned origins before any transport dispatch.
// Paths at the owned origin remain reachable to avoid masking redirect policy.
type testOriginTransport struct {
	scheme, host string
	underlying   http.RoundTripper
}

// RoundTrip passes only the exact owned TLS origin to the underlying transport.
func (t testOriginTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != t.scheme || r.URL.Host != t.host {
		return nil, errors.New("test provider origin refused")
	}
	return t.underlying.RoundTrip(r)
}

// NewClientForTLSServer is compiled only for tests. It retains New's timeout and
// redirect refusal; its origin guard prevents a changed base from dialing out.
func NewClientForTLSServer(t testing.TB, server *httptest.Server) *Client {
	t.Helper()
	origin, err := url.Parse(server.URL)
	if err != nil || origin.Scheme != "https" || origin.Host != server.Listener.Addr().String() {
		t.Fatal("owned TLS test origin required")
	}
	owned, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("owned TLS transport required")
	}
	transport := owned.Clone()
	transport.Proxy = nil
	// Cleanup targets the actual cloned transport, even though the wrapper does
	// not implement CloseIdleConnections. No global/default transport is changed.
	t.Cleanup(transport.CloseIdleConnections)
	client := New()
	client.http.Transport = testOriginTransport{scheme: origin.Scheme, host: origin.Host, underlying: transport}
	client.base = server.URL + "/api/v1"
	return client
}
