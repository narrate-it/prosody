# Inactive descriptive producer snapshot codec

The product decision selecting the label source is pending David. This package
is local preparation only: neither supported source class is a default or an
enabled product policy. There is no native projection, registration, migration,
route, tool argument, `Accept` change or capture callback. Existing source/settings
encodings and logical idempotency digests remain unchanged.

The codec represents two explicit alternatives:

- `OwnerAssignedConnectionLabel`: an owner-assigned connection label.
- `ClientAdvertisedApplicationName`: a name advertised by the original client.

An advertised application name is not a verified individual agent identity.
Choosing either enum does not prove metadata ownership, verified original-client
binding or who assigned a label. No label is synthesized from client ID, person
ID, credentials, authority references or session/grant/basis tables. A missing or
blank label/source class is an error, never a fallback.

## Local encoding profile

`Snapshot` explicitly requires version 1, profile `prosody.producer-snapshot`,
profile revision 1, source class, original public client reference, public label,
original capture scope and timestamp. Source class plus the client reference,
scope and time describe capture provenance; they are not native proof fields or
constructors for authority. The original client reference is opaque, nonblank,
valid UTF-8 and at most 2048 bytes. The host must supply a nonsecret public value;
a string validator cannot detect every credential or prove its origin.

The label is nonblank UTF-8, at most 256 decoded bytes, with no control characters
or Unicode line/paragraph separators. Byte limits apply before JSON escaping.
Labels/client references are preserved exactly, without trimming or normalization.
The original workspace, source item and invocation are canonical lowercase UUIDs.
An optional work UUID can only be supplied if actually captured during the same
admitted enqueue transaction. No code defaults or invents it. A future accepted
profile may require it; absence cannot be filled in later by mutable caller data.

Capture timestamps are nonzero instants from UTC year 1970 through 9999. Encoding
canonicalizes offsets and monotonic components to UTC; this proves no wall-clock
freshness or actual capture time. Native capture validation remains required.

The canonical bytes are compact Go `encoding/json` for the declared struct field
order/tags, UTC RFC3339Nano time and standard JSON string escaping. The final
encoded document is bounded to 8192 bytes. SHA-256 covers that complete byte
sequence, including source class, label, original client, scope and capture time.
`Decode` requires the expected digest and exact canonical bytes. It rejects
unknown or duplicate fields (including escaped/case-folded aliases), invalid
UTF-8/lone surrogates, alternate encodings and invalid data. Digest integrity does
not establish truthful metadata, authorization or the original native identity.

`RetainOriginal` is a pure replay model: it accepts stored canonical bytes, their
expected digest and the original captured scope. It deliberately accepts no new
client/name proposal. Later renames or a new valid requesting client cannot alter
an existing item's producer. Scope substitutions fail without returning data;
missing legacy snapshots remain unavailable rather than receiving backfilled
producer labels. This models no database commit, idempotency lookup or rollback.

## Required capture and persistence boundary

Before enabling capture, the accepted product/native contract must specify:

1. The selected source class, who controls its label, and the native original
   client projection available during the accepted submission invocation.
2. Exact nonsecret original client reference, public source label and capture time
   from that projection; the app cannot derive them from the later listener.
3. Native-captured workspace/source-item/original-invocation identities and whether
   the same enqueue operation supplies a work binding in this initial profile.
4. An exact immutable snapshot blob/digest write in the SAME source, idempotency
   receipt and work-intent transaction, with native-held authority and rollback/
   uncertain-commit reconciliation. A matching digest parameter is not authority.
5. Replay loading the original receipt/snapshot before considering current caller
   metadata, and authorized read-projection/size tests that expose no secrets.

No table or SQL proposal is frozen here because the selected source class and
native producer capture slots remain unaccepted. The current codec is reusable
local data preparation, not a supported native interface or proof shape.

## Evidence

Unit source covers deterministic round trips for both explicit source classes,
byte limits, UTF-8/control rejection, timestamp normalization/bounds, digest
mismatch, duplicate/nested/escaped fields, original-scope checks and immutable
replay. Runtime results and independent review are recorded with the eventual
source checkpoint; no database, native identity, provider, mobile or deployment
acceptance follows from these pure checks.

Local verification on 2026-11-03 UTC: `go test -p 1 -timeout 30s
./internal/producer` passed. The source received independent read-only review;
the final documentation/test follow-up is pending. Capacity admission passed at
52.98/44.48% interval idle, normal pressure, zero swap churn and no thermal warning;
the required commit gate runs under an owned shared build lease. No DB fixture,
provider call, native binding, device update or deployment was performed.
