# Keyvault input validation and legacy compatibility

Source preparation from `75dbc4de1201a80b3db2ce0a6e54d053f14c0f4c` hardens
the public Seal/Open library boundary. Current SaveSettings callers already
validate UUID scope IDs and format a positive credential revision; this is not
a claim of a known reachable service exploit or new current-authority admission.
Master-key custody, current identity and transaction authority remain host-owned.

## Input contract

Each Installation, Workspace, Credential and Revision string must remain
nonempty and must now be valid UTF-8 before encoding/json Marshal. All nonempty
valid UTF-8 is preserved exactly: no trimming, case folding, Unicode NFC, ID
syntax or new scope-length bound. Whitespace, NUL and literal U+FFFD remain
valid library input. The existing v1 JSON field names/order/escaping, AES-GCM
algorithm, nonce/envelope layout and production random-nonce source are unchanged.
There is no json/v2 migration or global random-source override.

The [encoding/json Marshal contract](https://pkg.go.dev/encoding/json#Marshal)
replaces malformed string bytes, losing their original distinction in the
authenticated bytes. On the executed Go 1.27.0 toolchain, the old-source overlay
accepts malformed FF scope bytes against an envelope sealed for literal U+FFFD.
The explicitly escaped replacement-token vectors are separate denial controls;
they passed even on the old source. Encoder representation details must not be
inferred from a different installed toolchain's source. Valid U+FFFD itself must
remain compatible, while malformed scope inputs must fail before marshaling.

Nil *Vault and zero Vault receivers return nil ErrInvalid before touching AEAD
state. No production panic recovery is added. Both directions require credential
plaintext length 1..8192 bytes; Open rejects even an independently authenticated
legacy envelope holding empty plaintext. Arbitrary binary credential bytes are
still allowed. New validation failures use the existing fixed ErrInvalid and
expose no scope or credential content. Algorithms, keys and nonce policy are not changed.

## Authored public-API checks and independent oracles

Tests reject malformed bytes in every scope field for Seal. Independent
explicit-AAD vectors use JSON field names/order and the lower-case
`\ufffd` escape; Open with either malformed FF or FE must refuse those vectors.
The oracle never calls the private binding helper or marshals Scope. A separate
literal-U+FFFD vector must open under the legitimate scope; malformed-scope Open
of public Seal's legitimate replacement-rune envelope must remain denied.

Additional explicit-AAD vectors retain valid Unicode, whitespace-only fields,
NUL, exact case distinction and precomposed/decomposed Unicode distinction.
New Seal output is opened independently with explicit legacy AAD; independent
legacy vectors are opened through public Open. A fixed TEST-ONLY nonce is used
only to build synthetic vectors. Existing random fresh-nonce, roundtrip,
cross-scope and tamper controls remain, with the second Seal error now checked.

Nil/zero public-method tests use valid scope/plaintext/envelopes so unrelated
preflight rejection cannot hide an old nil dereference. Recovery exists ONLY in
those tests to report old panics while the matrix continues. Length vectors
cover authenticated empty plaintext, one byte, 8192 bytes and 8193 bytes; empty
scope fields retain their old rejection. All keys and credentials are synthetic.

## Verification scope

Fresh capacity admission and an owned shared build lease allowed the first
execution. Fixed source: 38 named checks PASS, zero failures/skips. A compiled
Go overlay replacing only keyvault.go with the base implementation produced
19 failed named records (including parents) and 19 passing controls, zero skips.
Failures cover malformed Seal input, nil/zero receivers, authenticated empty
plaintext, and malformed Open against literal replacement-rune envelopes.
Explicitly escaped-envelope denial controls pass on both implementations.
The overlay leaves working source unchanged. Required repository lint/full-Go
checks and subsequent review must be recorded separately; these package results
do not establish them.

No .env/keychain/provider/master-key discovery, live credentials, HTTP, database,
AMOS/native API, retry/scheduler, infrastructure, signing, deployment or spending
is part of this library slice. Synthetic vector compatibility does not qualify
host custody, integration, provider effects or production cryptographic operation.
