# Caller-transaction notification preferences

Source preparation from main `b629600f47e48943055ee5772ea6582d3459f9b5`.
Accepted RFC0001 §§2/4.4 requires arrival notifications off by default and a
separate choice per person/feed. Existing `app_prosody_preferences` already has
workspace/feed/person primary key and a non-null boolean default false; no
migration is needed.

`ReadNotificationPreference(ctx, ReadTransaction, NotificationPreferenceKey)`
returns false only when a preference row is absent after a scoped live feed
has been locked. Missing, foreign-workspace/feed and tombstoned feeds return
ErrUnavailable; SQL failures never become default-false success. Reads do not
insert. An explicit saved false remains a distinct row, preserving the user's
choice.

`SetNotificationPreference(ctx, Transaction, key, enabled)` locks the live feed
FOR UPDATE, then parameterizes one UPSERT for the exact workspace/feed/person
key. Exactly one affected row is required; zero/multiple is ErrConflict, missing
result ErrInvalid, SQL/count failures preserve errors.Is causes. The caller
must roll back on error and owns commit/rollback. Success is not a committed
receipt. Neither function opens a connection or transaction, schedules work or
stages delivery.

Context is checked before SQL, after the successful feed-lock callback, after
preference scans (including missing/default false), after successful Exec and
before a successful affected-count return. A callback's SQL/count error keeps
its original errors.Is cause. Cancellation observed after an executed write
is not rollback, no-effect or commit proof: the caller must roll back and
reconcile any unknown transaction completion before retrying.

UUID syntax bounds identity strings to 36 bytes; feed IDs require version 7.
The person must come from the caller's current authenticated human operation.
A syntactically valid foreign-person ID is NOT authorization: these helpers
cannot infer current person, membership, session or grant authority from IDs.
The future native host must bind/recheck current person/workspace/feed admission
and hold its authority until finalization. An ID copy or callback grants nothing.
No native principal tables, public route or auth constructor is introduced.

The read-side feed FOR SHARE lock remains held through the caller transaction.
The write-side FOR UPDATE lock both excludes tombstoning and serializes all
preference writes for that feed, including different people. It conflicts with
read locks even when a preference is absent, so the default-false read is stable
against this writer protocol. Existing preference rows are also read FOR SHARE.
This deliberate coarse locking is bounded to one feed and requires a suitable
caller transaction; it is not PostgreSQL READ ONLY. Actual lock/rollback
behavior remains unexecuted database qualification, not fixture proof.

## Exact local SQL inventory

These literals/hashes are descriptive review targets for eventual native
catalog work. They are NOT supported AMOS registration or SQL safety admission.
The existing frozen consumption-read catalog is unchanged. Native current
authority/control queries need separate reviewed budgets. Every statement here
binds workspace at $1; no workspace-zero clock exception or native authority
identity slot is added. Row bounds rely on existing scoped primary keys.

### preferenceReadFeedSQL

read phase; arguments $1 workspace UUID, $2 feed UUID; one returned feed UUID (36 bytes); FOR SHARE; no native-person slot.

SHA256: `7c5b6875f75832209bd0ae01c804cb14f7b884152bbb842f397e8b01f92d4108`

```sql
SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR SHARE
```

### preferenceWriteFeedSQL

write phase; arguments $1 workspace UUID, $2 feed UUID; one returned feed UUID (36 bytes); FOR UPDATE; no native-person slot.

SHA256: `b446e6b1cd323653d6e9bd58615ea373fcf2276ed2cb4457bed4fe10a148ab50`

```sql
SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE
```

### preferenceReadSQL

read phase; arguments $1 workspace UUID, $2 feed UUID, $3 authenticated-person UUID binding; zero/one non-null boolean row; missing row defaults false only after live feed lock.

SHA256: `648410231aaaeb2c73ba31886feac636d14188c9a9d9d0d9cccdd8f1a31f02c9`

```sql
SELECT notifications FROM public.app_prosody_preferences WHERE workspace_id=$1 AND feed_id=$2 AND person_id=$3 FOR SHARE
```

### preferenceWriteSQL

write phase; arguments $1 workspace UUID, $2 feed UUID, $3 authenticated-person UUID binding, $4 boolean; exactly one affected row required. All UUID representations <=36 bytes; the bool has no caller-selected text codec.

SHA256: `087b6bb1cb6542cf65f3ad06cb1a105ab9bbc8a5efd148e45b7293a09450daf6`

```sql
INSERT INTO public.app_prosody_preferences(workspace_id,feed_id,person_id,notifications) VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,feed_id,person_id) DO UPDATE SET notifications=EXCLUDED.notifications
```

Read budget: two single-row application statements, no writes. Write budget:
one single-row feed lock plus one UPSERT, each executed once; no retry loop.
No cursor, arbitrary SQL, caller-selected identifier or additional payload is
accepted. Notification delivery still requires OS permission, current opt-in,
playable publication, per-person/item-publication deduplication, private default
payload and authorized item opening. Storing true alone provides none of those
proofs and emits no notification.

## Source verification and outstanding evidence

Hermetic public-store-API port fixtures cover exact literal/parameter order and
lock mode, default/no insertion, explicit on/off, person isolation, malformed
inputs, unavailable/tombstoned/foreign feeds, mismatched lock projections,
read/write/count failures, exact affected 1, and cancellation before SQL and
during feed/preference Scan, Exec and RowsAffected callbacks. Concurrent
cancellation does not mask a callback's SQL/count sentinel cause.
The fixtures are ephemeral maps and synthetic query results: they do not prove
native admission, real SQL semantics, database isolation/rollback or delivery.

The coordinator's scoped Go run (`^TestNotificationPreference`) on composed
source tree `b2c7930d0d20ef93ed99ba02accddfcd94c7bae5`, based on landed main
`6ec335655cf3676fec04b2f2de3a9401ed420566`, passed all 33 named test/subtest
records across seven top-level tests: exit 0, no failures or skips.

A compiled overlay replacing ONLY `preferences.go` with the earlier
`57c9b3a5385a4931343f0044e6a180590d68674d` source retained the current tests.
The default/on/off/person-isolation positive control passed (one record), while
all six cancellation-boundary cases plus their parent failed (seven failure
records), exit 1 as expected. This proves the cancellation regressions against
the old source, not real transaction effects. The exact owned build lease was
released; no DB/native/HTTP or notification-delivery execution occurred.

Lint, the full Go unit gate and the required commit hook are separate from
this scoped receipt; their actual results are recorded in the PR verification
record. No new database fixture or database run is
authorized. No HTTP/native route, APNs/scheduler, provider call, credential
lookup, infrastructure, device, deployment or signing action is part of this
slice. Native and delivery/mobile acceptance remain separate open gates.
