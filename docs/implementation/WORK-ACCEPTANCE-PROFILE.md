# Ordinary text-work acceptance profile

Status: Prosody integration proposal for AMOS review, not an enabled registration.
This specifies the first rewrite → speech path. It does not narrow the full RFC,
authorize a provider call, or supply a native authority or receipt constructor.

Source baseline: Prosody `c04aed92d7d1578855ae51257eb17b7f44d5449e`.
Private checkpoint preparation is separately pinned to draft PR #7,
`1b55d2baffebf764a240d07775cf09348f644044`; its migration is unenabled.
The AMOS counterpart inspected for this proposal is
`15d4e14ad60e536298f67c8e822a05ca9a539a0a`, specifically
`docs/contracts/application-work-v1.md`. Its insertion-backed delegated capture
is a primitive, not a durable binder or worker adapter.

## Acceptance transaction

The existing domain entry point is `internal/store.Accept`. A future native
adapter must hold original delegation, current destination authority and the
accepted settings read through this entire transaction:

1. Resolve the consent-selected workspace and canonical UUIDv7 feed. Lock the
   current, nondeleted destination and read settings using `CurrentSettings`.
2. Validate the submitted text and selection using the registered input/profile
   codecs. Insert the domain submission, preserving exact persisted source and
   settings bytes. Domain insertion alone is not native acceptance.
3. For a fresh submission only, `StageWork` enqueues through the native effect
   boundary in the SAME transaction. The native owner joins genuine insertion
   custody and original delegated authority; neither a job payload nor
   `InvocationContext` descriptive fields can reconstruct this authority.
4. Bind the actual native job and original invocation to the exact inserted
   submission, settings and admitted resource closure. Complete the domain work
   receipt and native binding atomically. Any binding or enqueue error aborts
   the whole acceptance; no externally accepted receipt precedes commit.

`StageWork(context.Context, itemID)` currently returns only a job ID. The native
adapter still needs an owning composition that validates actual inserted rows,
settings and native insertion proof before accepting a durable binding. This
proposal does not claim that the existing callback signature provides it.

## Byte and identity bindings

| Value | Existing domain source | Required native comparison |
| --- | --- | --- |
| Original scope | `store.Scope` and submission row | Native installation/application/environment, original person/workspace/operation; no caller-selected authority |
| Item/feed | Inserted submission `id`, `workspace_id`, `feed_id` | Actual inserted item and currently admitted feed closure, not matching-looking payload IDs |
| Source | `source_bytes`, produced by `json.Marshal(content.TextSubmission)` | SHA-256 of the actual stored bytes and exact codec revision; original text/title preserved |
| Settings | `settings_bytes`, copied from `CurrentSettings` | SHA-256 of actual copied bytes, validated selection and exact settings/credential revisions |
| Logical replay | `key_bytes`, `digest` | Existing actor/workspace/operation replay scope; same key with different feed/body conflicts |
| Job | `app_prosody_work.work_id` and native insertion receipt | Same original job through restart, lease changes and reconciliation |
| Invocation/attempt | Native accepted invocation and per-stage attempt journal | Original immutable identities, never regenerated from the current worker fence |

Do not decode/re-encode stored bytes before hashing them. `Submission.Digest`
is the logical submission digest; it must not silently stand in for an
independently checked persisted-source hash, even where the current encoder
produces the same bytes. `authority_ref` is bounded opaque domain data, not an
application-readable grant or a worker authorization capability.

The current input decoder limits HTTP JSON to 65,536 bytes. Text is nonblank
valid UTF-8 with at most 4,096 runes; an optional title has at most 160 runes.
The exact decoded idempotency key is nonblank valid UTF-8, 1–256 bytes, including
otherwise valid NUL/whitespace characters. Its storage is `bytea`. The native
profile must validate the actual persisted representation, not use a JSONB/text
conversion that loses this contract. The HTTP body limit is not a bound proven
by `store.Accept` on all direct typed callers.

Settings snapshots contain `textModel`, `speechModel`, `voice`,
`settingsRevision`, `credentialId` and `credentialRevision`. They contain
credential references, never plaintext keys. `SaveSettings` creates these
bytes; `CurrentSettings` locks the live settings/feed and returns those bytes.
The acceptance adapter must validate and capture the exact selected revision;
it must not read later mutable settings when resuming old work. Current
credential permission/revocation is still checked before dispatch; a captured
credential reference is not permission to use it forever.

## Native resource and lifetime requirements

The native owner must freeze a typed resource profile for the admitted feed,
original submission and required parent closure, including workspace
incarnation, ownership, version and tombstone predicates. The existing domain
tables have no complete resource-incarnation/version history. Do not invent a
version from `settings_revision`, `accepted_at`, a UUID or a content digest.
That missing native profile/schema is an explicit integration gate.

The initial qualification path is bound to the ORIGINAL submitting session,
consent/source grant and explicit job deadline/maximum age. It rechecks current
person/workspace/resource/financial authority before dispatch and publication.
The numeric maximum age is not selected here. Original source expiry/revocation
stops dispatch/publication while preserving receipt, source and logical replay
binding. Unrelated listener logout is not source revocation. Durable grants
surviving source signout remain an unresolved production policy.

The current submission status enum is only `processing`, `ready`, `failed`,
`deleted`; it does not yet distinguish cancelled/expired source authority.
Do not misrepresent that schema as the required state model. Retention periods,
receipt tombstones and safe cleanup/backfill are also not selected by this
profile; no deletion or retention policy is activated.

## Stage, checkpoint and publication chain

The first registered order is `rewrite`, then `speech`. Each stage freezes its
profile, request/result codec and maximum output bytes under one stable parent
job. Remote/accounting settlement and usable output are separate facts. A
response or a rejected result is not native settlement/no-effect proof.
Missing or ambiguous remote results require original-attempt reconciliation;
there is no inline paid retry or second scheduler.

PR #7's private binding envelope version 1 binds source/settings digests,
settings revision, original job/invocation/attempt, item/feed, stage and
registration/profile/codec/request/result/manifest/artifact digests. Its local
rewrite byte ceiling is 256,000; speech is 67,108,864. These are private artifact
codec bounds, not accepted HTTP media limits or native authority. Rewriting
requires valid UTF-8. Speech bytes still require actual MP3/duration admission.

Native Checkpoint must compare accepted candidate and actual bytes, then commit
the private rows AND opaque native receipt atomically. First insertion does not
require its own already-committed receipt. Replay compares the original receipt
and exact rows; unknown commit reconciles that SAME identity before advancing.
Generic journals contain metadata, not source/script/audio or raw errors.

Publication requires current native authority, financial admission and job
fence; matching committed checkpoint receipts; exact domain affected counts;
and real `StageReady` plus native publication receipt in the SAME transaction.
Look up an uncertain publication first: a ready item correctly fails the
processing-only checkpoint read and must not thereby become lost output.

## Review and qualification exit

This document freezes no supported Go signature, SQL privilege catalog, native
schema or runtime grant. AMOS must review the missing typed acceptance/closure
and durable binding interfaces; Prosody must adapt only to the resulting
supported revision. Raw SQL callback lifetime guards are not a SQL sandbox.

Required negatives include altered persisted bytes with unchanged claimed hash,
different settings revision, wrong original job/invocation, stale resource
version or tombstone, expired/revoked source authority, cross-workspace input,
replay with changed feed/body, ignored bind failure, and unknown commit followed
by restart. Require one original paid attempt, rollback of mismatched acceptance,
same-identity checkpoint recovery and atomic publication/StageReady rollback.
No new database, native, provider, device or deployment evidence is claimed here.
