// Package keyvault encrypts provider credentials with scope-bound authenticated
// encryption. Master-key custody and authorization belong to the host.
package keyvault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

var ErrInvalid = errors.New("invalid provider credential envelope")

type Scope struct{ Installation, Workspace, Credential, Revision string }
type Vault struct{ aead cipher.AEAD }

// New constructs AES-256-GCM from a caller-owned 32-byte master key.
func New(key []byte) (*Vault, error) {
	if len(key) != 32 {
		return nil, ErrInvalid
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	a, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	return &Vault{a}, nil
}

// binding preserves the exact v1 JSON representation of nonempty valid UTF-8
// fields; malformed bytes must not be replaced before authenticated binding.
func binding(s Scope) ([]byte, error) {
	for _, field := range []string{s.Installation, s.Workspace, s.Credential, s.Revision} {
		if field == "" || !utf8.ValidString(field) {
			return nil, ErrInvalid
		}
	}
	return json.Marshal(s)
}

// Seal encrypts a 1..8192-byte credential without normalizing scope fields.
func (v *Vault) Seal(s Scope, secret []byte) ([]byte, error) {
	if v == nil || v.aead == nil {
		return nil, ErrInvalid
	}
	if len(secret) == 0 || len(secret) > 8192 {
		return nil, ErrInvalid
	}
	aad, err := binding(s)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return v.aead.Seal(nonce, nonce, secret, aad), nil
}

// Open authenticates the unchanged envelope format and refuses empty credentials.
func (v *Vault) Open(s Scope, envelope []byte) ([]byte, error) {
	if v == nil || v.aead == nil {
		return nil, ErrInvalid
	}
	aad, err := binding(s)
	if err != nil {
		return nil, err
	}
	n := v.aead.NonceSize()
	if len(envelope) < n+v.aead.Overhead()+1 || len(envelope) > n+v.aead.Overhead()+8192 {
		return nil, ErrInvalid
	}
	plain, err := v.aead.Open(nil, envelope[:n], envelope[n:], aad)
	if err != nil {
		return nil, ErrInvalid
	}
	return plain, nil
}
