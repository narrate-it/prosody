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
)

var ErrInvalid = errors.New("invalid provider credential envelope")

type Scope struct{ Installation, Workspace, Credential, Revision string }
type Vault struct{ aead cipher.AEAD }

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
func binding(s Scope) ([]byte, error) {
	if s.Installation == "" || s.Workspace == "" || s.Credential == "" || s.Revision == "" {
		return nil, ErrInvalid
	}
	return json.Marshal(s)
}
func (v *Vault) Seal(s Scope, secret []byte) ([]byte, error) {
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
func (v *Vault) Open(s Scope, envelope []byte) ([]byte, error) {
	aad, err := binding(s)
	if err != nil {
		return nil, err
	}
	n := v.aead.NonceSize()
	if len(envelope) < n+v.aead.Overhead() || len(envelope) > n+v.aead.Overhead()+8192 {
		return nil, ErrInvalid
	}
	plain, err := v.aead.Open(nil, envelope[:n], envelope[n:], aad)
	if err != nil {
		return nil, ErrInvalid
	}
	return plain, nil
}
