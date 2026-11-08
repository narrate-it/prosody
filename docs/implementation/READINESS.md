# Prosody delivery and remaining acceptance

Source snapshot: main `8c0633c303b465f902837369bb95735ed9e49f59`, after PR #68.
This inventory separates delivered preview, later source work and usable-service
acceptance. Earlier snapshots remain in Git history; chronological STATUS/WORK
entries retain their original failures and verification scopes.

## Delivered preview

Agent Prosody **1.0.1 (41)**, bundle `run.sire.prosody`, was accepted by the upload
transport. [Protected self-hosted run 42426530316](https://github.com/narrate-it/prosody/actions/runs/42426530316)
completed successfully on 2026-11-07 at 23:45:56 UTC. Native export, Apple
validation and upload succeeded. Both private upload journals record ACCEPTED;
the continuation remains linked to the original attempt. Do not upload this
version/build again.

The app was built from `5bf817366bb791652468d1e9a89d52f3c6bf7b88`; the resumed
upload used controller `4a5cd77c42b075b951e1ee32de67f007ded95a1d` with identical
mobile inputs. PR #51 records delivery evidence. This supersedes the old signing
and upload refusals: signing access worked for this delivery. Any future release
still needs fresh runner, capacity, identity/profile and upload admission through
[the protected workflow](TESTFLIGHT-PREVIEW.md).

**Apple processing, export-compliance disposition and tester availability have
not been verified by this session.** If build41 is not visible, read its exact
App Store Connect status before taking another action. Do not infer a remote
compliance finding from a local plist, make an unapproved legal declaration or
retry the accepted upload. The existing ASC preflight reads app identity only;
its PASS does not inspect this build's processing or tester access.

Build41 is a preview, not the authenticated submit → generate → listen service.
Production sign-in remains disabled. A TestFlight install or successful upload
cannot qualify the unfinished backend, provider or device journey.

## Source newer than build41

The current mobile tree differs from the built input. The black-and-white brand
and logo from PR #52 and subsequent mobile fixes are **not in build41**. A new
reviewed release is required to deliver them; this record selects no new upload.

| Area | Latest relevant evidence | Remaining acceptance |
| --- | --- | --- |
| Mobile source | PR #66: analyzer zero issues, 31 focused and 395 full Flutter named records PASS, zero failures/skips and done success=true; the mobile tree is unchanged through this snapshot | New artifact/install for later source; real browser return, expiry/revoke, interruption, lock-screen and player checks |
| Provider and briefing | PR #68: ordinary/race each 223 named records across 30 roots PASS, zero failures/skips; scoped lint zero issues; owned TLS fixtures, not a live provider | Budgeted real provider behavior, usable output, artifact durability and native accounting/reconciliation |
| Local pipeline | PR #9/#13/#21/#60: ordinary orchestration, real owned HTTP fault paths and phase-cancellation checks; native capabilities/receipts in composition tests remain synthetic | Supported generic worker composition and real accepted-work/checkpoint/completion recovery |
| Media | Same-pass verified range helper, MP4 structural metadata checks and MP3 cancellation/measurement source | Current published-asset admission, full codec/timeline/object proof and actual device transport/playback; MP4 metadata remains unverified |
| Notifications | Generic ready-payload encoder, mapped hint and bounded raw-JSON validators; local preferences | Native hookup, OS permission, current recipient/device/access checks, committed-publication dedup and actual delivery/tap journey |

These evidence scopes overlap and must not be added together. Root Go hook passes
without a database do not qualify SQL or native services. The candidate buffered
media profile does not reduce RFC supplied-media limits.

## Ownership and service composition

Prosody owns feeds/items, rewrite and speech orchestration, private artifacts,
checkpoints, rendition/publication rules, selected ready effects, queries/media
handlers, notifications and deletion. AMOS owns reusable scoped authority,
workspace/tenant enforcement, held finality, fences, generic jobs/recovery and
transactional effect capabilities. Prosody's StageReady requirement is its own
selected publication policy, not a mandatory domain concept for every AMOS app.

At inspected AMOS main `5f01ec409208b6bb5db01317326d27d5c7989fdd`, its contracts
report a bounded document-host profile and bounded accepted-record claim work.
Those are real scoped advances, not a complete Prosody host: the record claim
adapter still delegates unsupported Resolve; published feed/media, full worker
completion and operating/device qualification remain excluded. This is source
and owner-receipt inspection, not a Prosody rerun. Local AMOS writers remain
stopped; the cloud owner retains implementation.

The [generic-interface request on PR #514](https://github.com/ajent-social/AMOS/pull/514#issuecomment-6718543866)
asks for exact supported revision/signatures and qualification. It has no peer
reply at this snapshot. Do not replace the missing interface with private native
SQL, a permissive authority callback, copied domain logic or a second scheduler.
The Prosody adapter remains on its existing pin; no activation follows here.

The selected origin is `https://api.prosody.chat`; the exact resource audience and
actual issuer/client registration remain unqualified. Intended client/return are
`prosody-ios` and `run.prosody.prosody:/oauth/callback`. The App Store bundle is a
separate identity. Producer-label source class remains unselected and its codec
inactive. No listener identity, mutable client metadata or existing-row fallback
supplies original producer provenance.

## Next usable-service acceptance

1. Obtain the supported generic worker and held scoped-read/finality handoff,
   then compose Prosody-owned operations and endpoints against its exact revision.
2. Bind original accepted source/settings and authority atomically with native
   work intent. Use the sole worker, current authority/fence checks, app-declared
   checkpoint/completion writes and effects, committed receipts and same-identity
   UNKNOWN reconciliation. Do not infer settlement from provider response receipt.
3. Qualify one real submission through rewrite/speech and publication to the
   intended listener, including replay, restart, lost responses and revocation
   without duplicate paid work.
4. Verify the configured iPhone sign-in, feed/media access, player, original/script,
   progress and team settings on a reviewed build. Complete the separately scoped
   Apple processing/tester checks for each delivery.

Team spaces remain RFC scope. A personal-only release is not substituted for team
acceptance without an explicit product decision.

## Other RFC gates

| Area | Recorded source/evidence | Remaining predicate |
| --- | --- | --- |
| Private checkpoints | PR #7 remains draft; historical one-run ordinary SQL evidence at `28ce03d2`: 132 mixed named records/32 roots, including `TestPrivateCheckpointPostgres` | Current-revision review, native atomic receipts, runtime grants/migration registration, fences and UNKNOWN commit recovery; migration remains unenabled |
| Supplied media and storage | Local private-file and range adapters; structural probes | Qualified durable production storage, complete media admission and actual player/range budgets |
| Attribution/BYOK | Inactive snapshot codec, presentation and settings/catalog checks | Legitimate original producer projection and selected source class; real model/voice/team-policy journey |
| Progress/notifications | Local domain/mobile source and tests | Current-authority persistence and real device/delivery acceptance |
| Retention/deletion | Tombstones and [cleanup-gap inventory](DELETION-CLEANUP.md) | Complete provenance/reference index, backfill, shared lock protocol, reviewed eligibility, durable cleanup and recovery |
| Packaging/operations | Nanos templates/checker and Go/Pulumi modules under `infra/` | Composed backend executable/image, secret/state custody, total cost plan, image boot, readiness, monitoring and backup/restore |

PR #7's ordinary SQL approval is **completed/consumed**, not permission for another
fixture. Linode source is a powered-off staging graph, not a working backend;
its compute guard does not enforce the owner's total $50/month budget. Existing
infrastructure requires its own custody/receipt reconciliation before any apply.
See [NANOS.md](NANOS.md), [infra/linode](../../infra/linode/README.md) and
[TESTFLIGHT-PREVIEW.md](TESTFLIGHT-PREVIEW.md).

Full authorized implementation and acceptance remain incomplete. Keep the owned
hourly reminder until those predicates are actually satisfied.
