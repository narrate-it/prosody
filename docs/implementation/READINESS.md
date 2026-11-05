# Prosody remaining delivery and acceptance

Snapshot: main `b6076182d1a487fa02e01585f4c1d0295f455390`, after PR #30 and
the earlier PR #31. This is a source/evidence inventory, not release approval.
The RFC remains product scope. Earlier chronological STATUS/WORK entries retain
their original verification scope; package checks do not close an integrated
journey. This revision supersedes this document's earlier draft-lane snapshot.

## Landed source and observed verification

| Area | Landed source and evidence | Acceptance still required |
| --- | --- | --- |
| iOS session client | PR #26 at `c04aed92d7d1578855ae51257eb17b7f44d5449e`; analyzer zero issues, 18 focused and 207 full Flutter named passes, zero errors/skips and done-success-true; final unsigned physical-device iOS debug compile passed with exact callback scheme | Qualified AMOS issuer/audience/client registration and feed/media interfaces; actual iPhone browser return, expiry, revoke and playback |
| Authorized media transport | PR #27 at `036a85fa93aa33fdb0c35c6bbaeb9bda9fc6bd10`; analyzer zero issues, 41 focused and 189 full Flutter named passes, zero errors/skips and done-success-true | Native every-request feed/item/asset admission; actual TLS/player/lock-screen behavior and measured limits/cancellation |
| Nanos packaging | PR #28 at `e7c6a9745b1f56390e50d216b2c5a103e65e7775`; six Python checks and offline self-check passed; final docstring-only revision preserved executable logic | Actual executable host composition, image build/boot, provenance, storage/backup/recovery acceptance |
| Rewrite envelope | PR #31 at `9ea0bda4a69cf308de6ccfd9f8aadd2d1dcc138e`; compiled old source failed three intended negative leaves plus parent while four controls passed; fixed eight envelope and 62 briefing records passed, zero skips | Integrated narration budget, provider settlement and usable output remain separate |
| Worker acceptance profile | PR #30 at the snapshot above; independent review and required Go hook passed | AMOS review/implementation of typed accepted-source/settings/resource binding and native worker boundaries; the profile is a proposal, not an enabled registration |

Each landed PR above had independent source review, hosted review, the preserved
repository Go hook and landed-tree verification. These receipts have different
scopes: Flutter checks used hermetic doubles, Nanos checks did not build an image,
and Go hook passes without a database do not establish SQL/native acceptance.
The final PR #26 docstring-only edits preserved executable lines; final unsigned
compilation was also performed. The two Flutter totals describe their respective
verified revisions and must not be added together.

Production sign-in and media composition remain disabled/unmounted. The media
client's candidate 64 MiB buffered profile is not an accepted reduction of RFC
supplied-media limits. Earlier failed fixtures, interrupted harness runs and
historical capacity holds remain failures/history; only corrected runs passed.

## Device and configuration gates

David's connected iPhone was identified and final unsigned compilation passed.
Signing with the existing development identity failed with
`errSecInternalComponent` involving `sire-ci`. Signing/authentication retries
are stopped. No install, app launch, real native sign-in or TestFlight upload
succeeded in that attempt. Repair access to the approved identity before a
new signing phase; do not guess passwords or alter Keychain defaults/access.

The selected backend origin is `https://api.prosody.chat`. This is not DNS/TLS
installation, a deployment or a registered native resource audience. The exact
issuer/audience and public client registration still need supported composition.
Intended client/return are `prosody-ios` and
`run.prosody.prosody:/oauth/callback`; source support does not register them.
App Store Connect app/team/access, release signing and unused build number remain
unverified. Producer-label source class remains unselected and its codec inactive.

## AMOS integration boundary

AMOS implementation is cloud-owned; previous local native writers remain
stopped. At inspected source `15d4e14ad60e536298f67c8e822a05ca9a539a0a`,
the human-consumption contract records tested personal grant/session/workspace
permission composition but explicitly excludes the typed resource closure,
catalog execution, media/routes, company policy and device qualification.
The application-work contract records tested insertion-backed delegated capture
but explicitly excludes durable binding/schema, worker admission/dispatch and
settlement/publication composition. These are source/owner-record inspections,
not Prosody reruns or a supported complete integration handoff.

Private-use native return support landed in AMOS PR #288 with recorded native
PG16 TLS/HTTPS checks. Actual Prosody registration and iPhone acceptance remain
open. No adapter pin or production route is activated by these source advances.
See [WORK-ACCEPTANCE-PROFILE.md](WORK-ACCEPTANCE-PROFILE.md) for the exact
Prosody-side byte, replay and checkpoint proposal and remaining native bridges.

## Shortest TestFlight path

1. Obtain the reviewed AMOS revision and supported interfaces/receipts for human
   sign-in, current-authority feed/media consumption and application work.
2. Compose the Prosody host, accepted submission and durable work binding through
   those interfaces. Use the sole AMOS worker, original source authority,
   per-stage provider accounting and atomic checkpoint/publication/StageReady.
3. Demonstrate one real agent text submission through rewrite/speech to a ready
   briefing for the intended human. Exercise expiry/revoke, restart, replay and
   uncertain provider/commit outcomes without duplicate paid work.
4. Repair approved signing access and run the authenticated iPhone journey:
   navigation, original/script, seek/rewind, audio/video transitions,
   interruptions and lock-screen continuation. Record revision/device/OS.
5. Verify ASC/team/release signing, registered origin/audience and unused build
   number. Prepare the reviewed signed artifact, obtain upload authorization,
   then verify installation through TestFlight.

Team spaces remain accepted scope. A personal-only beta requires an explicit
product decision; it is not substituted for team acceptance here.

## Broader RFC completion

| Area | Existing evidence | Remaining predicate |
| --- | --- | --- |
| Private stage checkpoints | Draft PR #7; one approved ordinary SQL invocation completed: 132 mixed named records/32 roots, including `TestPrivateCheckpointPostgres` | Native atomic receipt, runtime grants/migration registration, current fence/authority and unknown-commit reconciliation; review/merge |
| Narration/provider processing | Domain pipeline, typed observations, measured MP3 duration and real owned TLS fault fixtures | Explicitly budgeted real provider acceptance; aggregate narration-budget and bounded shortening/re-synthesis; output durability/accounting separate from response receipt |
| Supplied media | Local storage, verified range helper and MP4 structural checks | Actual codec/timeline/physical-object admission; MP4 probe remains unverified metadata |
| Attribution/settings | Presentation, inactive provenance codec and settings checks | Selected legitimate producer snapshot; real BYOK/model/voice/team-policy journey |
| Progress/notifications | Local mobile/domain preferences and state coverage | Current-authority per-person progress persistence, actual delivery/device permission and deduplication |
| Retention/deletion | Domain lifecycle and cleanup-gap inventory | Ownership/reference completeness, shared locking/backfill, reviewed eligibility and authorized cleanup/recovery |
| Deployment/operations | Nanos tooling | Executable host, secret custody, owning IaC preview, image boot, monitoring, backup/restore and separately authorized deployment |

PR #7's one-run SQL approval is completed/consumed, not permission for a new
fixture. Its checkpoint migration remains unregistered/unenabled. See
[TESTFLIGHT.md](TESTFLIGHT.md), [DELETION-CLEANUP.md](DELETION-CLEANUP.md),
[VAULT-VALIDATION.md](VAULT-VALIDATION.md) and RFC 0001 for scope.

No full RFC or TestFlight acceptance is claimed. The owned hourly reminder stays
enabled until full authorized implementation and acceptance are complete.
