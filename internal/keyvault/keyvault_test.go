package keyvault

import (
	"bytes"
	"testing"
)

func TestCredentialScopeAndTamperProtection(t *testing.T) {
	v, err := New(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	s := Scope{"installation", "workspace", "credential", "1"}
	a, err := v.Seal(s, []byte("synthetic-test-key"))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := v.Seal(s, []byte("synthetic-test-key"))
	if bytes.Equal(a, b) {
		t.Fatal("nonce reused")
	}
	plain, err := v.Open(s, a)
	if err != nil || string(plain) != "synthetic-test-key" {
		t.Fatal("round trip failed")
	}
	for _, other := range []Scope{{"other", "workspace", "credential", "1"}, {"installation", "other", "credential", "1"}, {"installation", "workspace", "other", "1"}, {"installation", "workspace", "credential", "2"}} {
		if _, err = v.Open(other, a); err == nil {
			t.Fatal("cross-scope decryption accepted")
		}
	}
	a[len(a)-1] ^= 1
	if _, err = v.Open(s, a); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err = v.Open(s, []byte{1}); err == nil {
		t.Fatal("short envelope accepted")
	}
}
