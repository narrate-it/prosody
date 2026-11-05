package keyvault_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"testing"

	"github.com/narrate-it/prosody/internal/keyvault"
)

const legacyASCIIAAD = `{"Installation":"installation","Workspace":"workspace","Credential":"credential","Revision":"1"}`

// validationKey is synthetic test material, never a discovered or real master key.
func validationKey() []byte { return bytes.Repeat([]byte{0x42}, 32) }

// validationVault exercises the public constructor with a valid synthetic key.
func validationVault(t *testing.T) *keyvault.Vault {
	t.Helper()
	v, err := keyvault.New(validationKey())
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// validationScope supplies valid fields and changes only the indexed field.
func validationScope(field int, value string) keyvault.Scope {
	s := keyvault.Scope{Installation: "installation", Workspace: "workspace", Credential: "credential", Revision: "1"}
	fields := []*string{&s.Installation, &s.Workspace, &s.Credential, &s.Revision}
	*fields[field] = value
	return s
}

// legacyFieldAAD spells pre-change field names/order explicitly. The caller
// supplies a complete JSON string token, including literal or escaped U+FFFD;
// no private binding helper or Scope marshaling derives this oracle.
func legacyFieldAAD(field int, encodedValue string) []byte {
	values := []string{`"installation"`, `"workspace"`, `"credential"`, `"1"`}
	values[field] = encodedValue
	return []byte(`{"Installation":` + values[0] + `,"Workspace":` + values[1] + `,"Credential":` + values[2] + `,"Revision":` + values[3] + `}`)
}

// legacyAEAD independently constructs the existing AES-GCM algorithm for vectors.
func legacyAEAD(t *testing.T) cipher.AEAD {
	t.Helper()
	block, err := aes.NewCipher(validationKey())
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return aead
}

// legacyEnvelope uses a fixed TEST-ONLY nonce to construct an independent
// legacy-format vector. Production nonce randomness/global rand.Reader is untouched.
func legacyEnvelope(t *testing.T, aad, plain []byte) []byte {
	t.Helper()
	aead := legacyAEAD(t)
	nonce := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	if aead.NonceSize() != len(nonce) {
		t.Fatal("legacy nonce format changed")
	}
	return aead.Seal(bytes.Clone(nonce), nonce, plain, aad)
}

// assertLegacyOpen checks new Seal output against explicit legacy AAD bytes.
func assertLegacyOpen(t *testing.T, aad, envelope, want []byte) {
	t.Helper()
	aead := legacyAEAD(t)
	n := aead.NonceSize()
	if len(envelope) < n+aead.Overhead() {
		t.Fatal("short sealed vector")
	}
	plain, err := aead.Open(nil, envelope[:n], envelope[n:], aad)
	if err != nil || !bytes.Equal(plain, want) {
		t.Fatal("Seal changed legacy AAD/envelope encoding")
	}
}

// TestVaultMalformedScopeSealDenied rejects invalid UTF-8 in every scope field
// before JSON replacement can collapse different malformed input byte strings.
func TestVaultMalformedScopeSealDenied(t *testing.T) {
	t.Parallel()
	for field, name := range []string{"installation", "workspace", "credential", "revision"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := validationVault(t)
			for _, bad := range []string{string([]byte{0xff}), string([]byte{0xfe})} {
				out, err := v.Seal(validationScope(field, bad), []byte("synthetic"))
				if !errors.Is(err, keyvault.ErrInvalid) || out != nil {
					t.Error("malformed scope was encrypted")
				}
			}
		})
	}
}

// TestVaultMalformedScopeLegacyOpenDenied refuses explicitly escaped replacement
// AAD vectors independently of the public/private binding encoder.
func TestVaultMalformedScopeLegacyOpenDenied(t *testing.T) {
	t.Parallel()
	for field, name := range []string{"installation", "workspace", "credential", "revision"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := validationVault(t)
			envelope := legacyEnvelope(t, legacyFieldAAD(field, `"\ufffd"`), []byte("synthetic"))
			for _, bad := range []string{string([]byte{0xfe}), string([]byte{0xff})} {
				plain, err := v.Open(validationScope(field, bad), envelope)
				if !errors.Is(err, keyvault.ErrInvalid) || plain != nil {
					t.Error("legacy malformed-byte alias authenticated")
				}
			}
		})
	}
}

// TestVaultLiteralReplacementRunePreserved checks legitimate U+FFFD remains
// usable while malformed bytes are refused. The executed old-source overlay
// authenticates malformed bytes against the literal replacement-rune envelope.
func TestVaultLiteralReplacementRunePreserved(t *testing.T) {
	t.Parallel()
	for field, name := range []string{"installation", "workspace", "credential", "revision"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := validationVault(t)
			scope := validationScope(field, "\ufffd")
			secret := []byte("synthetic")
			legacy := legacyEnvelope(t, legacyFieldAAD(field, `"�"`), secret)
			plain, err := v.Open(scope, legacy)
			if err != nil || !bytes.Equal(plain, secret) {
				t.Fatal("valid replacement rune lost legacy compatibility")
			}
			sealed, err := v.Seal(scope, secret)
			if err != nil {
				t.Fatal(err)
			}
			assertLegacyOpen(t, legacyFieldAAD(field, `"�"`), sealed, secret)
			if plain, err := v.Open(validationScope(field, string([]byte{0xff})), sealed); !errors.Is(err, keyvault.ErrInvalid) || plain != nil {
				t.Fatal("malformed scope opened legitimate replacement-rune envelope")
			}
		})
	}
}

// TestVaultValidUnicodeBindingPreservesExactBytes prevents trimming, case/NFC
// normalization, NUL removal or a newly imposed identifier grammar.
func TestVaultValidUnicodeBindingPreservesExactBytes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		scope keyvault.Scope
		aad   string
		other keyvault.Scope
	}{
		{"unicode whitespace NUL", keyvault.Scope{Installation: "部署", Workspace: " é ", Credential: "credential\x00", Revision: "\t1\n"}, `{"Installation":"部署","Workspace":" é ","Credential":"credential\u0000","Revision":"\t1\n"}`, keyvault.Scope{Installation: "部署", Workspace: "é", Credential: "credential\x00", Revision: "\t1\n"}},
		{"nonempty whitespace fields", keyvault.Scope{Installation: " ", Workspace: "\t", Credential: "\x00", Revision: "\n"}, `{"Installation":" ","Workspace":"\t","Credential":"\u0000","Revision":"\n"}`, keyvault.Scope{Installation: "  ", Workspace: "\t", Credential: "\x00", Revision: "\n"}},
		{"case stays distinct", keyvault.Scope{Installation: "INSTALLATION", Workspace: "workspace", Credential: "credential", Revision: "1"}, `{"Installation":"INSTALLATION","Workspace":"workspace","Credential":"credential","Revision":"1"}`, keyvault.Scope{Installation: "installation", Workspace: "workspace", Credential: "credential", Revision: "1"}},
		{"NFC stays distinct", keyvault.Scope{Installation: "é", Workspace: "workspace", Credential: "credential", Revision: "1"}, `{"Installation":"é","Workspace":"workspace","Credential":"credential","Revision":"1"}`, keyvault.Scope{Installation: "e\u0301", Workspace: "workspace", Credential: "credential", Revision: "1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := validationVault(t)
			secret := []byte{0, 1, 255}
			legacy := legacyEnvelope(t, []byte(tc.aad), secret)
			plain, err := v.Open(tc.scope, legacy)
			if err != nil || !bytes.Equal(plain, secret) {
				t.Fatal("valid exact scope no longer accepts legacy AAD")
			}
			envelope, err := v.Seal(tc.scope, secret)
			if err != nil {
				t.Fatal(err)
			}
			assertLegacyOpen(t, []byte(tc.aad), envelope, secret)
			if plain, err := v.Open(tc.other, envelope); !errors.Is(err, keyvault.ErrInvalid) || plain != nil {
				t.Fatal("distinct valid UTF-8 scope was normalized/aliased")
			}
		})
	}
}

// assertUninitializedVaultRefused catches panics ONLY in tests, allowing an
// old-source RED matrix to continue. No cryptographic panic recovery is added.
func assertUninitializedVaultRefused(t *testing.T, call func() ([]byte, error)) {
	t.Helper()
	defer func() {
		if recover() != nil {
			t.Error("uninitialized vault panicked")
		}
	}()
	out, err := call()
	if !errors.Is(err, keyvault.ErrInvalid) || out != nil {
		t.Error("uninitialized vault did not return nil ErrInvalid")
	}
}

// TestVaultUninitializedReceiversFailClosed exercises public methods with valid
// scope/plaintext/ciphertext so unrelated preflight failures cannot mask a panic.
func TestVaultUninitializedReceiversFailClosed(t *testing.T) {
	t.Parallel()
	scope := validationScope(0, "installation")
	envelope := legacyEnvelope(t, []byte(legacyASCIIAAD), []byte("synthetic"))
	for _, tc := range []struct {
		name  string
		vault *keyvault.Vault
	}{{"nil", nil}, {"zero", &keyvault.Vault{}}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Run("Seal", func(t *testing.T) {
				assertUninitializedVaultRefused(t, func() ([]byte, error) { return tc.vault.Seal(scope, []byte("synthetic")) })
			})
			t.Run("Open", func(t *testing.T) {
				assertUninitializedVaultRefused(t, func() ([]byte, error) { return tc.vault.Open(scope, envelope) })
			})
		})
	}
}

// TestVaultLegacyCredentialLengths checks an explicit ASCII AAD/fixed-nonce
// vector, authenticated empty plaintext refusal and unchanged binary size bounds.
func TestVaultLegacyCredentialLengths(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 1, 8192, 8193} {
		name := map[int]string{0: "empty", 1: "one byte", 8192: "maximum", 8193: "over maximum"}[size]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := validationVault(t)
			scope := validationScope(0, "installation")
			secret := bytes.Repeat([]byte{255}, size)
			legacy := legacyEnvelope(t, []byte(legacyASCIIAAD), secret)
			plain, openErr := v.Open(scope, legacy)
			envelope, sealErr := v.Seal(scope, secret)
			if size == 0 || size == 8193 {
				if !errors.Is(openErr, keyvault.ErrInvalid) || plain != nil || !errors.Is(sealErr, keyvault.ErrInvalid) || envelope != nil {
					t.Fatal("credential bounds accepted empty/oversized data")
				}
				return
			}
			if openErr != nil || sealErr != nil || !bytes.Equal(plain, secret) {
				t.Fatal("valid legacy credential byte length rejected")
			}
			assertLegacyOpen(t, []byte(legacyASCIIAAD), envelope, secret)
			roundtrip, err := v.Open(scope, envelope)
			if err != nil || !bytes.Equal(roundtrip, secret) {
				t.Fatal("valid credential roundtrip changed")
			}
		})
	}
}

// TestVaultEmptyScopeFieldsRemainInvalid preserves the existing nonempty-field
// requirement even for an independently authenticated legacy empty-field vector.
func TestVaultEmptyScopeFieldsRemainInvalid(t *testing.T) {
	t.Parallel()
	for field, name := range []string{"installation", "workspace", "credential", "revision"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := validationVault(t)
			scope := validationScope(field, "")
			if envelope, err := v.Seal(scope, []byte("synthetic")); !errors.Is(err, keyvault.ErrInvalid) || envelope != nil {
				t.Fatal("empty scope field accepted for Seal")
			}
			legacy := legacyEnvelope(t, legacyFieldAAD(field, `""`), []byte("synthetic"))
			if plain, err := v.Open(scope, legacy); !errors.Is(err, keyvault.ErrInvalid) || plain != nil {
				t.Fatal("empty scope field accepted for Open")
			}
		})
	}
}
