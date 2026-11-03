# Ordinary two-stage pipeline: LOCAL preparation only

`internal/pipeline` implements one reusable rewrite → speech → publication
consumer and reconciliation re-entry. It has no production caller, AMOS import,
worker, journal, retry loop, SQL pool, authority constructor or provider call.
The interfaces are local consumer contracts, **not supported native APIs**.
The only scheduler remains native `jobs.Worker`.

## Advancement and custody

`Run` first looks up the original publication identity. A matching committed
publication is validated and returned without reading processing-only private
checkpoints. An unresolved publication stops processing. This ordering also
applies before `Reenter` asks the native adapter to reconcile/resume work.

Only the fixed `rewrite` and `speech` stage IDs are dispatched. Each can be
called at most once per invocation of the function. Repeated entry relies on
native lookup proving the original stage state, never a process-local map or
new attempt. Only native-proven `Undispatched` permits a call; zero/missing,
unresolved and output-lost states stop. The adapter must not translate lookup
failure, transport drop/truncation or unknown commit into `Undispatched`.

Accepted candidates expose bounded sensitive artifact bytes in memory. The
initial local guard requires exactly one nonempty script artifact <=64 KiB
or one audio artifact <=32 MiB, its role/ID/type and full-byte SHA256. These
are consumer ceilings, not admitted native registration limits. Registered
codecs must additionally validate canonical request/result envelopes, the
ordered manifest, actual UTF-8/script/audio format, immutable artifact identity
and declared MIME. No byte payload is put into this package's errors or
metadata. Go memory lifetime is not secure zeroization.

Checkpoint must validate the actual accepted candidate and atomically commit
private bytes plus its opaque native committed receipt. Callback success,
accepted result metadata, or a private row alone cannot advance. Returned
receipt bindings must equal the candidate bindings exactly; receipt validation
under current native admission is required before moving to the next stage.
Previously committed receipts are also revalidated. This package does not
implement that atomic transaction or native validation.

Publication must validate both committed receipts and current source/financial/
resource/fence admission, invoke `store.Publish` and its **mandatory StageReady
callback in the same transaction**, require exact affected count 1, and return
a committed native publication receipt. Staged notification identity must
preserve native person/item/publication deduplication. A nil callback error
is not a commit receipt. Notification delivery's current recipient admission
is a separate native responsibility. The package deliberately does not expose
an application callback that could claim those guarantees.

## Reconciliation and immutable identity

`Work` is descriptive and binds original realm/workspace/job/invocation,
item/feed, settings revision and exact persisted source/settings byte SHA256,
plus the frozen work payload digest. Logical `Submission.Digest`, normalized
text and current settings cannot substitute for those hashes. Binding also
includes stage, original accepted attempt, profile/registration/codec revisions,
and request/result/manifest digests. No current authority is derived from IDs
or matching hashes. Source grant/session/person/workspace, financial and fence
authority stay native. Listener logout does not replace source-grant checks.

`Reenter` has no dispatch method on its reconciliation port. Native `Resume`
must settle every ambiguous original attempt before providing an execution
capability, preserving original invocation/accepted attempt identity under a
newly admitted fence. Prior receipt commit fences never grant current authority.
UNKNOWN means reconcile the same identity before another effect. Unsupported
lookup remains UNKNOWN/manual. Settled lost output requires qualified retrieval
of that same accepted result or manual output-lost handling, never another paid
call. Native Resume/Continue semantics and any safe retry authorization are
not implemented here. The local `Run` loop iterates two fixed stages once; it
does not retry a stage.

Raw `openrouter.AttemptOutcome` observations (`NotDispatched`,
`ResponseReceived`, `MayHaveActed`) are not native settlement, no-effect or
refund proof. The local interfaces do not convert them into accepted states.
An accepted/rejected result or matching-looking fixture receipt grants no
power: native adapters must validate their hidden capability when every port
is invoked. The interfaces intentionally have no production implementations
or constructors. Local projection guards are defense in depth only.

## Blocking native seams

The AMOS proposal v2 and the 2026-11-02 B checkpoint/catalog dispositions remain
proposals. Native registration must supply sealed accepted candidate custody,
committed checkpoint receipt creation/lookup/consumption, typed immutable
binding slots and byte/manifest validation, same-attempt reconciliation,
qualified Resume/Continue, and fenced publication with native StageReady
staging. Publication lookup must reconcile original commit identity before
processing-only checkpoint reads. An adapter cannot be wired until those exact
supported source interfaces and native qualification receipts exist.

The separate private checkpoint implementation/PR is neither imported nor
modified. Its database receipt is not treated as native authority. No database
or external-object durability is claimed by this package.

## Local evidence

Tests use visibly test-only candidates, checkpoint/publication receipts and
native-port fixtures. They check fixed order, one call per stage, no automatic
retry on drop/truncation/checkpoint uncertainty, missing/unsettled receipt and
lost-output barriers, stale job/workspace/invocation/source/settings/request/
result/attempt/manifest/artifact rejection, committed checkpoint replay, and
publication-first recovery even when private checkpoints are unavailable.

A passing fixture test does not qualify AMOS current authority, provider
settlement, transactional rollback, native notification staging, database
atomicity/durability, installation or the visible submit → listen journey.
