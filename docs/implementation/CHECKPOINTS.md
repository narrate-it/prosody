# Bytes-only provisional checkpoints

This is local preparation for future native callbacks. The separate
`CheckpointMigration` is not added to `store.Migration` or the AMOS adapter.
No production worker, native receipt, current-authority wrapper or human route
is registered. `CHECKPOINT-SQL.json` contains the three verbatim query constants,
UTF-8 digests and ordinal inventories; inventory is not SQL safety admission.

`WriteCheckpoint` validates actual bounded private bytes against ArtifactDigest,
locks the processing submission/work/feed, checks source realm/person/feed and
SHA-256 of exact persisted source/settings bytes plus settings revision, then
inserts immutable private bytes. Existing application/environment labels remain
labels; only fields with existing UUID contracts require UUIDs. Identical replay
matches serialized bindings and actual artifact bytes; changed bindings conflict.
`ReadCheckpoint` repeats source/tombstone checks and returns only exact bindings.

Rewrite bytes are bounded UTF-8 <=256000; speech bytes <=64MiB. There are no
external-object fields, ready flags, final rendition/manifest writes, published
asset links or notification intents. This is a byte codec, not MP3/duration or
remote settlement qualification. Native accepted-result/profile/codec validation
must recompute actual result/ordered artifact manifest bindings, not trust supplied
hash parameters. The local function checks actual artifact bytes, not that native
proof. Frozen source/settings bindings must originate in actual enqueue, not be
minted later from caller JSON. No authority is conferred by the binding struct.

The host must stage its opaque checkpoint receipt atomically in the SAME native
transaction. The fresh boolean is not a receipt or commit proof. First write uses
accepted candidate bindings; committed receipts are required on replay/advance,
not before their own insertion. Error/cancel/lost COMMIT may be UNKNOWN: reconcile
the SAME identity/bytes/receipt before advance or any retry. Lost remotely settled
output without qualified original-result lookup remains manual, never a new paid
call. A new admitted fence may validate an earlier immutable receipt; its prior
fence is not current authority. Current native APIs do not implement this seam.

Publication additionally checks each final write affected exactly one row before
calling mandatory StageReady. Any error requires caller rollback; readiness and
native notification staging must commit together. These callbacks never commit.

Historical source-checkpoint evidence (superseded for domain PostgreSQL only by
the 2026-11-03 receipt below): pure codec test passes. Publication transaction-double regressions
reject zero/multiple affected rows before notification; they fail against old
publication source and pass after the guards. Independent source review caught
and closed the initial UUID-only application/environment mismatch (RED then GREEN
with actual labels). Required Go lint/unit hook passes at initial checkpoint.
Real PostgreSQL regression SOURCE covers rollback, new-transaction replay,
changed result/attempt/artifact, stale source/settings, foreign workspace, source
processing/no final rendition and feed tombstones. It has NOT run: no approved DB
fixture is configured and DB-backed tests skip. No SQL migration execution, native
receipt/fence/commit qualification, provider calls, service or deployment occurred.

## Local codec and callback revision — 2026-11-03

The private binding encoding is version 1: compact Go `encoding/json` of an
ordered `version`, `binding` struct envelope, exposed through the pure
`EncodeCheckpointBinding` codec so future registration need not duplicate it.
This encoder creates no native capability. It preserves the declared binding struct
field names/order. The explicit local version is separate from `CodecDigest`,
which identifies the native-accepted stage result codec. Changing this byte
encoding requires a new encoding version and native registration; old unversioned
local fixture rows are not silently reinterpreted. No deployed checkpoint rows or
supported native registration exist to migrate in this preparation branch.

The current fence is deliberately absent from the immutable envelope. Original
invocation and accepted attempt identities remain fixed when a new worker fence
is admitted. Before processing-only checkpoint recovery, the future host must
first perform native final-publication reconciliation. An already-published job
must not be changed back to processing to retrieve provisional bytes.

Snapshot validation now requires one exactly spelled `settingsRevision` positive
integer, valid lossless JSON and no duplicate top-level fields; field-name casing
cannot change which revision is observed. Other settings fields still belong to
the frozen configuration codec. Source/settings SHA-256 remains over the exact
persisted bytes. The read SQL additionally bounds binding bytes and uses the
stage-specific artifact byte ceiling. Catalog argument/column types, row counts,
callback call-count ceilings and schema/SQL digests are explicit review targets.

Cancellation is checked before callbacks and before returning fresh/private-byte
results. Cancellation after an insert does not prove rollback: the native owner
must reconcile the same transaction identity and receipt before advancement.
No private row, fresh flag or successful callback constructs a native receipt.

New hermetic transaction-port cases exercise exact argument slots and read/write
sequencing, malformed settings with matching hashes, realm/person/feed mismatches,
missing/tombstoned source results, stale actual bytes and binding/result/attempt
changes, insert zero/one/multiple/nil/error results and cancellation before/after
callbacks. Publication cases now include nil and erroring row counts on both
writes. These doubles do not establish SQL isolation, rollback, row locks,
concurrency, atomic native receipt insertion or real source tombstone filtering.

Verification: the checkpoint/catalog/port and publication row-count tests pass
with `PROSODY_TEST_DATABASE_URL` explicitly removed. Regression probes against
the previous callback source fail for duplicate/case-folded/lossy-Unicode settings
and cancellation before/after callbacks; restored source passes. The prior-source
probe adds only a version constant so the new test sources can compile; it does
not change the tested old callback behavior. An initial RED attempt reached the
port's exact-argument guard because the test fixture had not refreshed its binding
bytes after changing a digest; fixing that fixture yielded the intended semantic
failure assertions. This evidence is not real PostgreSQL or native acceptance.

Resource coordination: first fresh admission passed but the lease was held by
another worker; no tests ran until the parent explicitly reported its release.
The authorized next phase passed (34.50/48.73% interval idle, normal pressure,
zero swap churn, no thermal/performance warning) and acquired an owned lease.
Real DB acceptance still requires an explicitly owned admitted fixture; none has
been adopted or provisioned. Migration remains disabled. Independent review of
this implementation delta and native registration/receipt qualification remain
pending; the implementer is not the independent reviewer of this delta.


## Publication guard dependency landed — 2026-11-03

PR #14 landed at `dc95bca3`; PR #7 is rebased onto that main revision and now
inherits the identical publication guard/test instead of carrying its own delta.
The saved original head is `f596a1b` at backup ref
`backup/checkpoints-before-publication-dc95`. Checkpoint code/tests, migration002
and SQL catalog are byte-identical to that original head. Rebase record conflicts
preserve complete entries from both parents, ordered by date.

Fresh scoped checkpoint/catalog/port tests passed. The real PostgreSQL checkpoint
test explicitly skipped with the DB environment removed. Migration remains
UNENABLED, this PR stays DRAFT, and native atomic checkpoint receipts are still
unsupported. Neither a rebase nor unit evidence opens DB/native acceptance gates.


### 2026-11-03 — Checkpoint domain PostgreSQL acceptance

David approved one disposable domain fixture invocation, now completed and
consumed. At exact PR #7 head `28ce03d2`, installed PostgreSQL 14.21 over a private
Unix socket (no TCP) passed all 132 named store checks (32 top-level), with zero
failures or skips, including `TestPrivateCheckpointPostgres`. This supplies real
domain SQL/replay/rollback evidence; earlier unconfigured skips remain historical.
Migration 002 was applied only in the disposable test database. The application
migration registry and adapter remain UNREGISTERED/UNENABLED for checkpoints.

The exact owned server stopped successfully, its data was removed, socket files
were absent, logs retained, and the build lease released. The one-time approval
does not authorize another fixture invocation. All six checkpoint source/test,
migration and SQL catalog blobs remain byte-identical to `28ce03d2` and `f596a1b`;
backup `backup/checkpoints-before-pg14-record` preserves the tested head. This
record-only continuation keeps the tested base; no rebase or source change.

PR #7 remains DRAFT. Native PostgreSQL 16/TLS/roles, same-transaction opaque
checkpoint receipts, current authority/grants and runtime-profile acceptance
remain open; ordinary PostgreSQL 14 tests do not qualify them. No provider,
HTTP, iOS or deployment acceptance follows. The full RFC remains incomplete.

The retained receipt records runner SHA-256
`6d4dcab579a5fa2ae2e582568a829eb1aa38cf04341164efb19c2195fa200960`.
Its source head is `28ce03d2eb8bc566b625ab5ce58a5433188d3eb3`; proof is scoped
to those exact unchanged domain inputs, not a newly qualified native integration.
