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
Call. A new admitted fence may validate an earlier immutable receipt; its prior
fence is not current authority. Current native APIs do not implement this seam.

Publication additionally checks each final write affected exactly one row before
calling mandatory StageReady. Any error requires caller rollback; readiness and
native notification staging must commit together. These callbacks never commit.

Evidence: pure codec test passes. Publication transaction-double regressions
reject zero/multiple affected rows before notification; they fail against old
publication source and pass after the guards. Independent source review caught
and closed the initial UUID-only application/environment mismatch (RED then GREEN
with actual labels). Required Go lint/unit hook passes at initial checkpoint.
Real PostgreSQL regression SOURCE covers rollback, new-transaction replay,
changed result/attempt/artifact, stale source/settings, foreign workspace, source
processing/no final rendition and feed tombstones. It has NOT run: no approved DB
fixture is configured and DB-backed tests skip. No SQL migration execution, native
receipt/fence/commit qualification, provider calls, service or deployment occurred.
