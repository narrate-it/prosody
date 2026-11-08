# Active implementation work

Owner: current Prosody session. Full RFC implementation remains authorized and incomplete.
The hourly cron queues this session; it is a reminder, not a separate worker or effect scheduler.

## Delivery sequence and acceptance

1. **In progress — durable domain boundary:** strict Unicode decoding; byte-preserving PostgreSQL schema; caller-transaction submission, rendition and progress persistence. Verify real local database rollback/replay/conflict and NUL key round trips. AMOS owns authority and durable effect scheduling; no competing job scheduler.
2. **Source in progress; verification held — mobile:** Flutter application with approved theme, authenticated transport, distinct feed switching, vertical playback, animated audio surface, background queue, original/script views, models/voices and team settings. Verify analyzer, domain/widget tests and simulator where capacity permits.
3. **Source in progress; verification held — processing/media:** private asset storage and bounded upload/finalization, measured audio duration, narration pipeline and fenced publication, key/model catalogs and previews. Verify failure/uncertain-provider outcomes and supplied-media preservation.
4. **First adapter source added; composition gates open — integrated host:** inspect exact AMOS candidate; register typed operations, permissions/resources/effects and schema migrations. Verify real current-authority admission, atomic work intent, lost response and cancellation. Unqualified upstream capabilities stay gated.
5. **Pending — lifecycle/operations:** notifications, deletion/retention, configuration, Nanos image definition, IaC and recovery/runbooks. Build and test artifacts without deployment or spend.
6. **Pending — acceptance/review:** full affected checks, independent review, remediation and exact-revision verification. Record provider/device/Nanos gaps honestly; do not mark complete on unit tests alone.

## Reminder lifecycle

Installed user crontab entry tagged `prosody-rfc-hourly-refocus`, at minute 0 each hour.
`scripts/refocus.py` queues the existing thread with `--resume`, a 45-second subprocess timeout and a process lock; it preserves the session model. Latest queue output is bounded to `.local/refocus-last.log`.
On completion remove only this tagged crontab entry and write `docs/implementation/COMPLETE` with evidence links. This file must not exist until completion is established.

## Current verification gate

The 2026-10-31 MBP capacity sample refused heavy work (4.20%/3.93% idle).
The owned lease was released. Do not automatically retry or switch hosts based
on the reminder alone. A user update or other authorized change is needed to
resolve admission. Superseded by David’s update that only one other Codex session is running: a fresh check passed (46.17%/47.31% idle, normal pressure, no swap churn or thermal warning). Bounded verification resumed with owned leases; recheck at phase boundaries.

The first four source/domain verification steps below passed after the fresh admission (see latest evidence). Native builds, device acceptance and actual AMOS host qualification remain open:

1. `GOMAXPROCS=2 GOCACHE="$PWD/.local/gocache" go test -p 1 ./internal/...` (and
   race checks for the added competing-transaction test once the owned DB runs).
2. `PROSODY_BUILD_ADMITTED=1 scripts/test-postgres.sh` — creates/stops only its own
   local Unix-socket fixture, retains diagnostic data, does not qualify AMOS TLS.
3. In `integration/amos`, run the constructor/codec tests against the exact pinned
   candidate through `.local/amos.work`; qualify actual host separately.
4. In `mobile`, `flutter analyze` and `flutter test`, then unsigned native builds
   and simulator/physical-device acceptance under the native platform gates.
5. Reconcile new findings before broadening implementation or claiming readiness.

Independent source review caught and prompted fixes; it is not executed testing.
Video metadata probing cannot be used as complete media admission. A real
sample-map/codec/timeline validator and adversarial video fixtures remain needed.
AMOS owner reconciliation confirms missing public application-job/current-authority/publication
and human mobile return/consumption-API boundaries on exact `809e060e`. Those
require AMOS design, review and qualification. No worker is inferred from enqueue;
production Flutter sign-in is disabled and speculative OIDC/refresh/custom-scheme
code has been removed. Preserve independent Prosody ownership without private-root
access, fixture promotion or another scheduler.

## Bounded coverage verification — hourly continuation

Ran `dart mobile/tool/check_coverage.dart` using the direct installed Dart SDK;
exit 0 in approximately one second. This dependency-free check exercises sampling
arithmetic only: continuous interval merging/draining, seek discontinuity, long
scheduling gaps, pause/buffering resets, invalid speeds and positions, double-speed
playback, backward jumps and an 80 ms clip. Invalid negative positions/timestamps
and non-finite speeds now reset the baseline instead of producing played coverage.
The Flutter test wrapper invokes the same checks, but was not run. No Flutter
engine, analyzer, native build, database fixture or heavy capacity retry was used.

Next verification remains the admitted database/adapter/mobile sequence above.
Real decoder event ordering and background playback still require device acceptance.
Full RFC implementation is incomplete; the owned reminder remains installed.

## AMOS owner boundary acknowledgment

The owner recorded worker and mobile acceptance as distinct generic AMOS slices.
Retain `jobs.Worker` as sole queue scheduler. Current authority/financial admission
before provider dispatch and fenced authorized local publication each require new
owning interfaces and qualification. Ambiguous provider outcomes must reconcile
before retry. No new implemented API or qualification receipt was reported.

Native redirects remain numeric loopback only. Registered HTTPS app-links are an
option, not an accepted architecture; a supported current-authority feed boundary
and actual iOS acceptance remain required. No OIDC, refresh or MCP-as-human-auth
assumption is introduced. Updated operation/tool names are recorded, not qualified.
The RFC now states these distinctions explicitly. Prosody's independent implementation
work and existing capacity hold remain unchanged.

## Playback response validation — hourly continuation

Implemented fail-closed decoding in `mobile/lib/data/models.dart`: known segment
kinds, nonempty segment identities, nonnegative representable durations (positive
for timed media), required HTTPS media, image-only presentation consistency,
unique segment IDs, and a rendition plus segments for ready items. Aggregate
generated narration is capped at 600000 ms; supplied audio/video is exempt.
Processing items may omit rendition/segments. Decoded lists are immutable.

Executed dependency-free `dart mobile/tool/check_models.dart`: exit 0, under one
second. Checks include the exact narration limit, aggregate overflow, two-hour
supplied clips, pending items, malformed/duplicate segments and unsafe URL forms.
Dart formatting also completed. The Flutter test wrapper shares these checks but
has not run; no Flutter engine, heavy admission retry, provider request, deployment
or spend occurred. This validates response decoding, not real server DTOs, media
content, redirect behavior, device playback or AMOS authority. Heavy verification
and upstream gates remain open; full implementation is incomplete and cron stays.

## Saved-progress validation — hourly continuation

Added a pure `SavedProgress` decoder shared by initial feed restoration and the
progress recorder. Rejects invalid/non-exact revisions, out-of-range positions,
segment-count/duration mismatches against the requested immutable rendition, and
mutation responses that do not advance exactly one revision. Initial restoration
no longer silently clamps invalid responses; existing UI/sync error paths handle
rejection. Service authority and rendition identity binding remain server duties;
matching durations alone cannot prove identity. Responses currently require the
full progress shape used by the Go domain, pending real consumption-API acceptance.

Executed `dart mobile/tool/check_progress.dart`: exit 0 in under one second.
Verified initial revision zero, saved positions, visual-only position zero,
malformed values, duration mismatch, and stale/skipped revision rejection. Shared
Flutter wrapper added but not run. This was bounded pure-Dart validation with no
Flutter engine, heavy admission retry, database, provider or deployment activity.
Full service/device verification and implementation remain incomplete. Keep the
owned hourly reminder; the next admitted checks remain listed above.

## Queue pagination correctness — hourly continuation

Fixed duplicate items within a single incoming page. Feed presentation and the
background playback handler now share `newPlayableItems`, including initial
loading. It preserves server order and already queued renditions, admits each new
ready item once, and allows a previously processing item when it later becomes
ready. Also fenced pagination error reporting by feed-load revision so an old
request cannot show an error in a newly selected space (source change only).

Executed `dart mobile/tool/check_queue.dart`: exit 0 in under one second. Checks
cover page overlap/replay, within-page duplicates, stable rendition selection,
processing-to-ready transition, rejected failed/empty items and unchanged inputs.
Shared Flutter test wrapper added, not executed. Handler/UI integration remains
unqualified pending admitted Flutter/device checks; pure queue behavior alone
does not establish background playback or feed-switch race acceptance. No heavy
admission retry, deployment, spend or AMOS interface assumption. Full authorized
implementation remains incomplete; retain the owned hourly cron.

## Pagination cycle prevention — hourly continuation

Added per-feed-load cursor tracking. Rejects immediate repeats, multi-page cycles,
empty cursors, out-of-sequence responses and continuation after a terminal page.
Rejected pages leave the accepted cursor unchanged; transport failures do not
advance it, so retries remain possible. Feed switching creates fresh tracking,
and stale responses are checked before acceptance. Also added the missing load
revision check after awaiting playback stop during feed selection (source only).

Executed `dart mobile/tool/check_pagination.dart`: exit 0 in under one second.
Checks cover A→B→A and longer cycles, rejection state preservation, valid retry,
terminal/empty feeds and independent feed cursor reuse. Flutter wrapper added but
not run. Pure pagination checks do not qualify HTTP, UI race handling or background
playback; heavy verification remains capacity-held and AMOS gates remain open.
No deployment/spend or admission retry. Full implementation remains incomplete;
the owned hourly reminder remains installed.

## Provider settings validation — hourly continuation

Settings Save now checks selected model/voice against the loaded catalog, rejects
ambiguous selections and requires a nonblank replacement key for first setup.
An existing key may be retained by leaving the replacement empty. Retired choices
that render as empty dropdowns can no longer be submitted as hidden old values.
Model/voice/key edits are disabled during save; the handler also guards duplicate
saves and read-only feeds. Key-controller clearing now occurs only while mounted,
avoiding access after screen disposal. Server authority and current provider
capability checks remain required; this client helper grants no authority.

Executed `dart mobile/tool/check_provider_selection.dart`: exit 0 in under one
second. Checks cover valid/replacement/retained keys, blank keys, missing/retired
models and voices, empty catalogs and duplicate IDs. Added a shared Flutter test
wrapper, not executed. UI lifecycle and actual provider behavior remain unqualified.
No heavy capacity retry, provider call, deployment or spend. Full implementation
and acceptance remain incomplete; retain the owned hourly reminder.

## Flutter verification after fresh admission

Phase sample passed at 23.38%/39.37% idle, normal pressure, zero sampled swap churn,
no thermal/performance warning. `flutter analyze --no-pub` initially found 44 lint
issues and no type errors. Applied braces/import fixes, matched the overridden
seek parameter name, and used stdout for standalone tool output. Re-ran analysis:
exit 0, no issues. `flutter test --no-pub --concurrency=1 --reporter=expanded`
passed all nine existing tests. These cover transport and pure domain behavior;
there are no device/widget playback acceptance claims. Owned build lease released.

The previous capacity refusal is superseded for these executed phases. All Go
packages, the disposable PostgreSQL storage suite, exact-candidate AMOS adapter
constructor/codec tests, Flutter analysis and existing Flutter tests have now
passed. AMOS runtime authority/worker/mobile gates, native builds/device behavior,
provider acceptance and complete product implementation remain outstanding.

## Settings screen acceptance in Flutter tests

Fresh admission passed (87.81%/84.00% interval idle, normal memory pressure, zero
swap churn, no thermal/performance warning). Added real widget tests using only
an in-process mock HTTP transport. They verify team members cannot edit shared
settings, retired selections issue no write, saves carry the expected revision
and submitted choices/key, controls lock while saving, and leaving the screen
before the response does not access disposed UI or submit again.

Initial widget execution timed out in fake-clock response handling. Fixed the
harness to start asynchronous mock-stream work via tester.runAsync; no production
UI change was needed. The targeted three-widget-test run passed, and Flutter
analysis remains clean. One command was accidentally launched from the repository
root and rejected for missing pubspec; rerun used the mobile project directory.
No provider/network effects were made by these tests. This verifies settings
widgets only, not native auth, OS playback, provider admission or the full app.
Owned lease released. Full implementation remains incomplete; retain cron.

## Real database publication verification

Admission passed at 23.57%/53.37% interval idle, normal memory pressure, zero swap
churn, no thermal/performance warning. Added PostgreSQL publication tests covering
exact ten-minute narration, single and aggregate over-limit rejection, supplied
two-hour audio exemption, foreign-workspace asset denial, mismatched duration,
missing script, and ready-notification staging failure. Successful publication
persists one rendition and ready status; a failing callback requires caller
rollback and leaves the committed source processing with no rendition. Invalid
media never reaches the notification callback.

Both executions of the disposable PostgreSQL storage suite passed, the second
after adding the aggregate-limit case. Both owned fixture servers stopped; the
build lease was released. Asset metadata is a test fixture, not actual media
decoding evidence. These tests verify domain transaction behavior only, not AMOS
held authority, job fences, provider output, or notification dispatch. No source
fix was needed. Full implementation remains incomplete; keep the hourly reminder.

## Encrypted provider settings database verification

Admission passed at 65.93%/88.12% interval idle, normal pressure, zero swap churn,
no thermal/performance warning. Added a real PostgreSQL test for encrypted key
storage and scope-bound decryption, secret-free admission snapshots, stale-update
rejection, key preservation on model-only edits, rotation to a fresh credential
identity and explicit revocation. Revocation clears envelope/references and makes
CurrentSettings unavailable. Old credential identity cannot decrypt the rotated
envelope; wrong-workspace decryption also fails.

The first run rejected the synthetic catalog because the fixture used the wrong
voice-field name. Corrected it to the existing adapter's supported_voices shape;
this is not a live-provider compatibility claim. The next run passed; after
adding rotation, the final full PostgreSQL storage suite passed again. All three
owned fixture servers stopped and the exact build lease was released. No real
credentials/provider requests, deployment or spend. These checks do not establish
AMOS admin authority or revocation of in-flight worker effects. Full implementation
and acceptance remain incomplete; retain the hourly reminder.

## Requested native hot-reload preview

Built the actual iOS simulator app with `flutter build ios --simulator --debug
--no-pub`; Xcode build succeeded in 138.4 seconds. Initial admission passed at
50.26%/84.20% idle with normal pressure, no swap churn or thermal warning. Exact
build lease released afterward. Created the dedicated Prosody Preview iPhone 17
simulator. First boot required approximately five minutes of iOS migration.

Started `flutter run --debug --no-pub -d 31D5A590-40F4-4086-B3DE-A8EEAF34C081
--use-application-binary build/ios/iphonesimulator/Runner.app` from mobile, reusing
the built binary rather than another native build. Runner remains attached in
PTY session 26517 for requested hot reload. Sent `r`: completed in 639 ms, zero
changed libraries. Screenshot visually confirms actual onboarding and the
explicit disabled sign-in state. Screenshot: ignored `.local/prosody-running.png`.
No working authentication/feed journey or device media playback is established.
Simulator is intentionally left open for David. No release signing/deployment.

## Bounded narration synthesis stage

Implemented briefing.Synthesize: one explicitly caller-admitted speech request,
no retry/repair, input and byte bounds, actual full MP3 decode against the remaining
aggregate narration budget, preserved original accepted bytes and SHA-256, and a
provider generation reference retained even when audio validation fails. Returned
audio owns a copy rather than aliasing the provider buffer. This is a processing
function, not an AMOS worker, authority grant, scheduler or publication adapter.
The caller must durably reconcile uncertain outcomes before retrying.

Added a small locally generated silent MP3 fixture with its reproduction command;
ffmpeg was used only to generate test data, not as a production dependency. Focused
Go tests passed for accepted decoded audio, truncated/malformed/oversized output,
remaining-duration rejection, buffer isolation, uncertain-result non-retry and
pre-dispatch cancellation/budget validation. Admission passed at 88.68%/79.35%
interval idle, normal pressure, no sampled swap churn or thermal warning. Exact
build lease released. No .env key read by tests, provider calls or charges.
AMOS dispatch/current-authority/journal/publication qualification remains open.
Full authorized implementation remains incomplete; retain cron and the requested
preview session.

## Immutable object write reconciliation

Added Files.Reconcile for the local development object adapter. It compares a
previously recorded expected size and SHA-256 to the existing immutable object
using a bounded, cancelable read. Returns distinct missing/matching/conflicting
outcomes without writing or overwriting bytes. Errors do not authorize retry;
the owning AMOS lifecycle must still decide recovery under current authority.
This adds no scheduler, journal, credential or external storage assumption.

Focused media and briefing Go tests passed after admission (35.40%/50.51% interval
idle, normal pressure, zero swap churn, no thermal/performance warning). Tests
exercise lost-receipt reconciliation, digest/size conflicts, invalid metadata,
cancellation and preservation of original bytes. Exact build lease released.
Nanos external object storage and fenced publication remain open. No provider
calls, deployment or spend. Full implementation remains incomplete; retain cron.

## Authorized parallel headless implementation

David explicitly requested parallel headless sessions. Three inherited-model
workers launched/reused with bounded briefs and disjoint ownership:

- Backend worker `/root/review_storage`: new store reads.go/reads_test.go and
  worker-storage.md; scoped feed/ready-item reads and cursor tests. No AMOS
  authentication claim; ambiguous multi-rendition items fail closed in v1.
- Mobile worker `/root/review_mobile`: feed.dart plus new activity.dart,
  activity_widget_test.dart and worker-mobile.md; processing/failure visibility
  without paid generation retries, new auth or fabricated backend routes.
- Release worker `/root/beta_readiness`: scripts/ios-preflight.sh and
  TESTFLIGHT.md; read-only configuration checks, no signing/upload/account edits.

Root retains integration, review, STATUS/WORK, AMOS coordination and serialized
heavy test admission. Workers initially perform source/lightweight checks only;
heavy commands need an explicitly assigned slot and owned lease. No cloud jobs,
provider calls, .env transfers, deployments, commits or overlapping AMOS edits.
Queued AMOS owner checkpoint request (01a195bd-8aeb-7673-a306-1a47101a4154) for
worker-authority/publication and human-mobile/feed boundary delivery sequence.
This is active parallel work, not completed or qualified delivery.

## AMOS owner checkpoint: no bounded integration available yet

Owner reports candidate `809e060e64c00160d8751ad8a35b3d76e2436b45` remains clean,
local and unmerged, based on main `5fc6ec27`. Remote main is now `e2546aed`;
a dry merge-tree reports no conflicts. This is not a rebase or qualification.
Selected Go/JS tests passed at the candidate; real MCP/CIMD/PostgreSQL lost-response
acceptance remains open. Neither generic worker authority/financial/fenced
publication nor human iOS return/current-authority feed admission has a reviewed,
tested bounded interface yet. Both remain proposals.

Owner's dependency sequence:
1. Reconcile candidate with current main and required service tests/review.
2. Admit/implement single-scheduler worker dispatch, UNKNOWN reconciliation and
   fenced local publication; test revocation and stale-fence denial.
3. Admit iOS return plus same-current-authority feed admission; test actual iOS,
   expiry and revocation.
4. Jointly verify one submit-to-listen journey before separately authorized
   TestFlight distribution.

OAuth expiry across waits, scoped grant revocation and an owned TLS fixture remain
dependencies. No deployment/spend/readiness claim. Prosody's independent workers
continue without assuming these interfaces or editing AMOS-owned source.

## Parallel integration checkpoint

Activity screen: five widget tests passed; Flutter analysis clean after one test
lint correction. iOS preflight independently checked: preview prerequisites pass,
release remains explicitly blocked/unqualified. Both root-owned verification
leases released. Backend cursor omission and list-tombstone fencing review fixes
are authored; dedicated database worker granted the next exclusive test slot
with fresh capacity admission, exact lease and required release. Final corrected
read-seam acceptance is pending that result. No merge/deploy/provider calls.
Full RFC and TestFlight acceptance remain incomplete; keep the hourly reminder.

## Capacity hold and bounded catalog validation

Backend worker's one permitted admission sample refused: CPU idle 0.92%/0.14%,
normal pressure, no swap churn or thermal warning. No lease acquired, database
fixture started, or test executed. Corrected cursor/deletion-fence tests remain
pending; prior pass does not qualify them. No automatic retry or fallback is
permitted from an hourly reminder alone; a changed admission basis is needed.
Independent source follow-up requested for the corrections without builds.

Continued bounded pure-Dart work: provider model decoding now rejects blank model
identities/names, malformed or duplicate voice IDs, duplicate model IDs and invalid
catalog shapes. Returned catalogs/voices are immutable copies. API model loading
uses the decoder, so ambiguous dropdown data fails through existing settings-error
handling rather than reaching Flutter dropdown assertions. Standalone
`dart mobile/tool/check_catalog.dart` passed; formatting completed. Checks include
valid/empty/text-only catalogs, duplicate and malformed entries and alias isolation.
Flutter wrapper added but not executed; API/widget integration remains unverified
for this change under the new hold. No provider calls/spend/deployment. Full RFC
implementation remains incomplete and the reminder stays installed.

## Mixed-media timeline seek correction

Independent source follow-up confirmed the backend cursor inventory and list
row-lock corrections with no new blocking source issue; database execution is
still held after the refusal. No automatic capacity retry was attempted.

Added pure timeline seek resolution and wired it into the playback handler.
OS seeking resolves timed segments while skipping zero-duration images; seeking
at/beyond the total duration lands at the end of the last timed segment instead
of a trailing image. All-image timelines stay at the first image. Image/no-item
positions and image buffering no longer inherit stale audio-player positions;
segment seeks on images do not touch the audio player. Normal visual progression
and actual-view tracking remain separate from seek selection.

Standalone Dart checks passed for leading/interstitial/trailing images, exact
boundaries, negative and beyond-end seeks, all-image items and invalid timelines.
Formatting parsed changed source. Flutter wrapper added but not run; handler/OS
integration and actual playback remain unverified under the capacity hold. No
provider calls, deployment or spend. Full implementation is incomplete; keep cron.

## Activity query implementation assigned; settings revision validation

Resumed backend worker with disjoint new activity.go/activity_test.go ownership
for all-status, scoped submission summaries with strict pagination and tombstone
exclusion. This closes an implementation gap between Activity UI and ready-only
read functions; no HTTP/current-authority capability is implied. Source work only:
existing capacity refusal still prohibits automatic heavy verification retries.

Settings response decoding now requires an exact nonnegative revision within
cross-platform integer precision, explicit key/notification booleans and valid
nonblank optional model/voice strings. Revision zero and null selections remain
valid for initial setup. Standalone Dart checks passed for initial/configured
settings and malformed revisions, booleans and selections. Formatting completed;
Flutter/API integration tests for this change remain unrun under the hold.
No provider calls, spending, deployment or admission retry. Activity worker source
is in progress, not accepted. Full implementation incomplete; retain hourly cron.

## Activity backend/mobile contract alignment

Collected source-only Activity read queries and tests. Independent source review
found no blocking issue in strict cursor validation, scope, tombstone exclusion,
source projection or ordering; execution and concurrency/cutoff behavior remain
unverified. Found and fixed integration mismatch: Activity was decoding playable
Briefing objects, which reject ready summaries without a rendition. Added separate
ActivitySummary/ActivityPage decoding and aligned screen/API/test source to the
proposed `/prosody/v1/feeds/{feedId}/activity` route. Removed fabricated dependency
on an agent display label absent from the domain projection. No HTTP endpoint or
AMOS human authority support is claimed. Contract recorded in ACTIVITY-CONTRACT.md.

Standalone Dart summary checks passed for all supported statuses, source NUL
preservation, missing manifests, malformed fields/cursors and empty results.
Formatting parsed changed source. Changed widget/API integration and PostgreSQL
tests remain unexecuted under the existing admission refusal; no retry occurred.
No provider calls/spend/deployment. Full implementation remains incomplete; keep
owned hourly reminder.

## Absolute response-body deadline

Replaced per-chunk-only HTTP body timeout with a bounded collector using one
absolute deadline. A server trickling chunks cannot keep the body stage alive
indefinitely. Size overflow/deadline/error ends collection and cancels its stream
subscription; transport socket-abort behavior remains client-specific. Bytes do
not appear in limit errors, and the API preserves the existing safe oversized
response message. No automatic retries added.

Standalone Dart stream checks passed for exact-limit bytes, oversized cancellation,
continuous trickle timeout, empty responses and transport-error propagation;
rerun passed after adding synchronous-listen exception cleanup. Formatting parsed
the source. Flutter wrapper added but not run; real HTTP/socket and widget
integration remain unverified. Existing heavy admission refusal remains in force;
no retry, provider calls, deployment or spend. Full implementation incomplete;
keep owned hourly reminder.

## Resume position independent of listening coverage

Added a pure ResumePosition tracker and wired progress recording to enqueue
position-only changes. Paused seeks can now be captured by the existing sampling/
flush cycle without manufacturing played intervals or viewed/read status. Position
is bounded to segment duration; unchanged positions do not continually dirty the
queue, and segment changes reset state. Existing conflict/uncertain-write handling
is preserved. This does not guarantee final position persistence on force-kill.

Standalone Dart checks passed for paused/backward seeks with empty coverage,
unchanged positions, negative/overshoot bounds and zero-duration/new segments.
Formatting parsed source. Recorder/HTTP/device integration tests remain unrun
under the existing heavy-admission refusal, and no retry was attempted. Full
implementation remains incomplete; no provider calls/spend/deploy; retain cron.

## Complete bounded space-directory loading

Mobile feeds() now follows feed-directory pagination rather than discarding
nextCursor after the first response. Collects at most 20 pages/500 feeds, rejects
cycles/duplicate IDs and malformed capability fields, preserves service order,
and fails the refresh instead of silently presenting a partial directory after
page failure or limit exhaustion. Current cross-workspace authority enumeration
and canManage projection remain host duties, not implemented by this collector.

Standalone Dart checks passed for a second-page team space, cursor cycles,
duplicate identities, entry/page caps, malformed entries, empty results and a
later-page failure without retry or partial success. Formatting parsed changed
source. Flutter/HTTP/UI integration tests remain pending under the existing
capacity refusal; no fresh admission attempted. No provider calls/spend/deploy.
Full implementation remains incomplete; keep owned hourly reminder.

## Explicit per-person mark-as-read action

Added Mark as read to the source/briefing details sheet. The API reads and validates
saved progress, submits the current rendition/revision and saved position with
markedRead=true, no played intervals and viewed=false, then requires the next
revision and confirmed mark. Action is single-flight and makes no automatic
uncertain-write retry. Concurrent recorder/device updates can produce the existing
visible conflict response; no revision is manufactured. It does not send agent
communication or declare playback coverage. Host must still bind current person
and authority in the proposed progress endpoint.

Standalone Dart payload checks passed: rendition/revision/position retained,
no invented listening/image consumption and no client authority fields. Formatting
parsed changed source. Widget/HTTP/recorder-concurrency acceptance is unrun under
the existing heavy-admission refusal. No provider calls/spend/deployment or
capacity retry. Full implementation remains incomplete; keep hourly reminder.

## Mark-as-read API client verification without a service

Executed standalone Dart tests through the actual ProsodyApi and an in-process
MockClient (no sockets/provider/credentials). Verified one revision-bound write
on success; no write after denied access or malformed saved progress; no automatic
retry on conflict or a simulated response lost after commit; stale/unconfirmed
mutation results fail; saved position and empty listening coverage remain intact;
private server error bodies are not exposed. All seven scenarios passed.

Added the equivalent Flutter test wrapper but did not run Flutter under the heavy
admission hold. No production change was needed. This establishes client behavior
with a controlled transport, not actual server commit/replay, UI, AMOS authority
or device acceptance. Formatting completed. No admission retry, spend or deploy.
Full implementation remains incomplete; retain owned hourly reminder.


### 2026-11-01 — Directory and Activity API transport checks

Added and passed seven standalone Dart scenarios through the actual ProsodyApi
with an in-process MockClient. Directory pagination preserves opaque cursor
characters and team management flags; a revoked second page, duplicate identity
or cursor cycle fails the refresh without returning partial results or retrying.
Activity accepts a ready summary without playback assets, preserves source NUL,
and rejects unknown statuses and revoked access. Both routes use authenticated
GET with redirects disabled and hide private server denial details.

Added a Flutter test wrapper; it remains unrun under the existing heavy-build
admission hold. No production change was required. This is controlled client
transport evidence, not server authority, database, Flutter UI or iOS acceptance.
No capacity retry, provider call, spend or deployment. Full implementation and
acceptance remain incomplete; retain the owned hourly reminder.


### 2026-11-01 — Reject malformed playback identities before queueing

Briefing decoding now rejects empty/missing item IDs, unknown status, incorrectly
typed metadata and malformed segment containers with FormatException. Feed-page
decoding validates entries/cursors and returns an immutable item list. Existing
cross-page queue deduplication remains unchanged.

Expanded standalone model checks passed, including eleven invalid briefing
shapes, four malformed page shapes and immutable valid-page behavior. Existing
standalone queue checks passed against the changed decoder. Formatting completed.
Flutter analysis/widget/device checks remain pending under the existing capacity
hold; no heavy admission retry or provider call was made. These are local client
checks, not AMOS/service acceptance. Full implementation remains incomplete;
retain the owned hourly reminder.


### 2026-11-01 — Bound aggregate native playback duration

Fixed a playback-decoding gap: individually representable supplied clips could
sum to an overflowing native Duration, corrupting the item timeline. The decoder
now checks each duration addition before accepting a rendition. Generated
narration's existing ten-minute aggregate limit is checked during accumulation;
supplied audio/video remain exempt from that product limit.

Standalone Dart model checks passed, including a representable long supplied
clip and rejection of two such clips whose combined native timeline overflows.
Existing mixed-media seek checks passed. This verifies native Dart arithmetic,
not browser arithmetic or actual media playback. Formatting completed; Flutter,
service and device acceptance remain pending. No heavy admission retry, provider
call, spend or deployment. Full work remains incomplete; retain hourly reminder.


### 2026-11-01 — PR #1 independent headless review and admitted validation

David replaced the unavailable CodeRabbit gate with independent headless review.
PR #1 contains initial source commit 0a8a3ba; local follow-up 3ca4b11 fixes known
provider receipt loss and playback transition/dwell findings. Two reviewers
inspected backend and mobile separately, then cross-reviewed each other's fixes.
Cross-review accepted the Go fixes but found an additional stale advancement
ownership race in playback; that finding remains open while its fix is prepared.
No merge is claimed by this entry.

Fresh admission for this explicit review/merge task passed: interval CPU idle
81.0/62.5%, normal pressure, zero swap churn, no thermal/performance warning.
A phase-boundary sample also passed at 30.73/39.0% idle. The numeric load ceiling
in the shared policy applies to the named ten-core host, not this four-core Mac.
The owned shared build lease was acquired for bounded checks.

Executed: full root Go tests; disposable PostgreSQL storage tests including
later reads/Activity cases; pinned AMOS 809e060e adapter tests; Flutter analysis
(clean after ten test-tool style fixes); all 29 Flutter tests including two new
playback regressions. The new cross-review race still needs its own regression.
Go lint initially hit toolchain export incompatibility; the already installed
Go 1.27.0 matched the linter. After checking cleanup errors, lint passed with
zero issues. Commit 3ca4b11 ran full root lint and tests through a temporary
package-aware hook. The original hook remains enabled and unchanged; its broken
multi-directory invocation was replaced only for that commit, with no check
skipped. Initial PR commit's explicitly authorized hook bypass is historical.

These are local source/fixture checks. Real provider, AMOS authority, mobile
authentication, physical background playback, Nanos and TestFlight acceptance
remain unfinished. No deployment/spend/signing. Hourly reminder remains owned
and active because the full RFC implementation is incomplete.


### 2026-11-01 — Review findings closed; final playback regression passes

The independent cross-review accepted the generation-scoped advancement fix.
All three targeted handler tests now pass: visual dwell/cancellation without view
credit, stale replacement after stop, and a pending obsolete decoder load that
must not block a newer visual queue. Flutter analysis is clean on the final
source. The earlier full 29-test run passed; only handler code/tests changed
subsequently and the three affected tests passed. Initial fake-clock harness
hang and a fake-player name collision were corrected, not counted as passes.
Root independently inspected the reviewers' fixes; each reviewer cross-reviewed
the other domain's delta. No remaining concrete source-review blockers reported.

A final phase admission passed at 39.21/58.1% idle, normal pressure, no swap
churn or thermal/performance warning. Root Go lint/full tests, real disposable
PostgreSQL tests and pinned AMOS adapter tests passed as recorded above. Merge
is authorized as an initial partial implementation, not deployment or full RFC
acceptance. The exact remote merge receipt will be recorded in PR #1. Existing
runtime/mobile/provider acceptance gaps and the hourly reminder remain open.


### 2026-11-01 — CodeRabbit follow-up source and validation

CodeRabbit completed review of 6673de7 with eleven inline findings. Valid findings
were fixed and cross-reviewed; detailed dispositions are in PR-1-REVIEW.md.
The additional receipt-contract mismatch is aligned to persisted acceptance
identity/time. The owner-specific refocus-helper configuration nitpick is deferred
without changing or removing the owned cron. Full RFC completion is still false.

Fresh admission passed at 56.62/65.32% idle, normal pressure, no swap churn or
thermal/performance warning; a later phase passed at 69.34/41.50%. Shared lease
acquired for bounded verification. Four fixture control-flow tests passed. Real
PostgreSQL tests passed despite injected PGHOSTADDR/PGUSER/PGSERVICE, and only
the successful run's data was removed while logs were retained. Updated AMOS
receipt tests and rewrite-budget tests passed. All 48 Flutter tests passed on
final API denial precedence; analysis is clean. Source hashes verify copied
Dart/Darwin dependency files unchanged; only three documented Android baseline
files differ plus new guard/instrumentation source. Android compilation and
device instrumentation are pending because the SDK is unavailable here.

An independent cross-review found and closed a new optional-resume denial-body
edge case. Known HTTP denial now survives oversized, failed or nonterminating
bodies and cleanup errors. No provider calls, signing, deployment or runtime
acceptance inferred. Review fixes are ready for final lint/commit/remote checks;
exact merge outcome belongs to the PR receipt, not this pre-merge checkpoint.


Final follow-up checks: root and pinned nested AMOS golangci-lint both report
zero issues on Go 1.27.0. Nested lint initially exceeded a two-minute dependency
load deadline; the bounded, serialized five-minute run passed. Final Flutter
suite passed all 48 tests and analysis is clean. Independent cross-review closed
all accepted source findings. Vendored upstream whitespace is preserved via a
vendor-only .gitattributes rule so baseline hashes remain meaningful; authored
files continue to pass whitespace checks. No Android compilation/device evidence
is claimed. Original hook remains enabled; final code commit runs the corrected
package-aware lint/test hook, with no gate skipped.


### 2026-11-01 — Second CodeRabbit follow-up: bounded models and unknown resume

Accepted findings 4438319330 and 4438319339 against 7e4f511. Rewrite now takes
current selected model metadata plus an explicitly admitted completion budget and
trusted full-prompt token counter. The cap is the minimum of admitted budget,
provider completion limit, and remaining context. Missing/null limits, unsupported
max_tokens, invalid counts or counter failure prevent provider dispatch. There is
no generic tokenizer estimate, model substitution or paid counting request.
The host must qualify current provider-route metadata, complete message/framing
accounting and financial admission; catalog metadata alone does not qualify all
routes or guarantee reasoning/multilingual completion.

Resume lookup now distinguishes saved, genuinely absent (404), and unknown state.
Unknown state permits playback but blocks recorder capture and mutations until a
known reload or explicit Use this device's position action. The latter clears
unrecorded fallback observations and requires a current revision read before a
future write. Independent cross-review closed both source findings. Go targeted
budget/catalog/configuration tests passed; 20 targeted Flutter tests passed,
including recorder-level preservation and explicit-override regressions.

An existing integration gap remains explicit: recorder revision lookup treats
404 as a sync failure rather than constructing revision zero. The composed human
progress API must define initial progress semantics; this does not bypass the
unknown-resume or authorization protections. No real provider or service/device
acceptance claimed. Further final checks and exact PR receipt precede merge.
Fresh admission passed at 39.81/51.41% interval idle, normal pressure, zero swap
churn and no thermal/performance warning; bounded checks hold an owned lease.


Final verification for the second follow-up: all 50 Flutter tests passed and
Flutter analysis is clean. Targeted Go budget/catalog/configuration tests, root
Go lint and the real PostgreSQL suite passed. The final commit additionally runs
root lint/full Go tests through the package-aware hook. Independent reviewers
cross-checked the other domain's changes and reported no remaining concrete
blocker for these findings. Platform/provider/AMOS runtime gates remain open;
this is source validation for the initial implementation PR, not full acceptance.


## PR #1 later-page resume follow-up (2026-11-01)

CodeRabbit finding 4438401301 exposed a missing saved-progress lookup when the
first playable item arrives on a later page. Both initial selection paths now
share resume lookup, current-load checks and the unknown-progress write gate
before restoring the saved segment/position. Existing active queues append
without another resume lookup. Feed replacement suppresses old-page callbacks
until installation completes; a new page identity prevents a previous space's
visible page from overriding the restored first item. Independent review caught
that retained-page edge case during this follow-up and it was fixed.

New widget regressions cover saved/unknown/401/403 later-page initialization,
stale workspace responses, stop-callback prefetch, and switching from page three.
Verification and final independent review are recorded in the PR receipt.
No provider/device/deployment acceptance is implied. Original hook remains enabled.

Final local checks: all 57 Flutter tests passed; Flutter analysis has no issues.
The seven new widget cases use a recording player and HTTP fixtures, not native
decoders or a deployed service. Go/adapter sources are unchanged from the prior
verified checkpoint. Owned build lease released after verification.
Independent final source re-review found no remaining blocker in this delta;
the retained-page finding is closed by the new page identity and regression.


### 2026-11-01 — Preview command

Added root `make preview` for interactive Flutter debug/hot reload, with optional
DEVICE and FLUTTER overrides. Checked recipe argument forwarding using a local
stub and dry runs; no native build, provider call or device acceptance claimed.
Preview currently reaches onboarding with disabled sign-in, not a fixture feed.
Work uses an isolated internal-disk worktree authorized by David; he clarified
that BuildOffload is a Mac Mini requirement, not a MacBook Pro requirement.


### 2026-11-01 — Flutter stable migration and iPhone build

Owner requested latest stable Flutter/Dart: installed Flutter 3.47.9 with Dart
3.13.7. Its first run migrated this preview worktree to iOS 15 minimum, UIScene
lifecycle and Swift Package Manager plugin integration, retaining the bundle ID
and background audio mode. SDK-pinned dependency changes are preserved in lockfiles.
The signed physical-iPhone Xcode build passed in 103.8s after local signing access
was recovered; no certificate/keychain replacement or access-control change was
needed. Plist and whitespace checks passed. Installation/debug attachment remains
separate evidence. Full Flutter tests/analysis on this new SDK and independent
migration review remain pending; prior SDK test receipts do not qualify it.
No TestFlight distribution, production deployment or provider calls occurred.


### 2026-11-01 — Usable offline preview

Owner found onboarding-only preview unusable. `make preview` now selects an
explicit debug-only entrypoint opening sample feeds directly, with a visible
OFFLINE DEMO banner, three bundled locally generated spoken recordings, and
in-memory progress/mark-read/deletion. Personal and team fixtures are separate;
settings are read-only and credentials are rejected. A deny-network API and
allowlisted asset player keep this preview offline. Production main is unchanged.
No AMOS sign-in, real submissions, provider synthesis, notification delivery or
cross-device persistence is claimed. Preview state resets with its instance.
Independent source review found no blockers. All 62 Flutter tests passed on
Flutter 3.47.9/Dart 3.13.7, and analysis is clean. Native preview installation
and visible/audio behavior are recorded separately. Minimum SDK constraints now
match the migrated project; settings controls use the supported initialValue API.

Offline preview device receipt: signed Xcode build passed in 56.3s; Flutter
reported installation/launch and debugger attachment on David's physical iPhone.
Hot reload passed in 570ms. On-device VM widget-tree inspection confirmed the
OFFLINE DEMO banner, FeedScreen and a sample title. Actual audible playback,
user gestures and background/lock-screen behavior still need human/device
acceptance; debugger attachment alone does not establish them. Build lease
released; preview remains attached for hot reload.

2026-11-01: PR2 live preview and user feedback fixes implemented; see STATUS.
User selected live text, elevenlabs/eleven-v4-turbo/Laura and $1 session allowance.
91 Flutter tests +30 targeted follow-up passed; real PG fixture passed after
cold-cache timeout and warm. Provider requests remain unexecuted. Physical iPhone
update/key provisioning is next; preserve four attempts for owner testing.

Follow-up: iPhone live preview installed and key provisioned (four attempts remain;
no agent-paid generation). User reported slider and Mac sample voice; fixed drag
state/single-release seek with two regressions and independent review; 12 targeted
tests +full analysis passed. Explicit sample label prevents voice confusion.

2026-11-01 hourly continuation: fixed PR2 live-editor navigation dropping an
already reserved generation result. Both removed/covered-route regressions pass;
seven budget tests pass; analysis clean; independent review clear. No provider
calls or phone changes. AMOS and actual device/provider acceptance remain open;
keep the owned reminder until full authorized implementation/acceptance completes.

2026-11-01: adapted David's voice and podcast craft into both rewrite prompts;
Go previously used Narrate conversational, preview had minimal instructions.
17 mocked provider tests +Go briefing tests passed; independent review clear.
Factual/source/budget limits retained. Audio warmth still needs listening; no paid
request was made. See narration-craft.md for editorial guidance and limits.

### 2026-11-01 — Resume recorder stimulus follow-up

PR #1 merged at 6c7e12f with exact-tree verification; its final non-blocking
review note requested stronger later-page unknown-resume capture coverage.
This follow-up simulates active playback at 400 ms before completion and checks
read/write counts. The saved-resume case is a positive control proving that the
same stimulus reaches dispatch; unknown resume must retain one read/no writes.
Source only: tests and analysis have not run. Capacity measurement timed out,
so no build lease, heavy test or automatic admission retry was attempted.
The user subsequently connected an iPhone; the owned Prosody simulator was shut
down to reduce load. Physical-device signing/playback acceptance remains open.


### 2026-11-01 — PR #3 targeted verification

Fresh admission passed: interval CPU idle 59.89/60.4%, pressure normal, swap
rates below 0.15 MiB/s and no thermal/performance warning. Under an owned lease,
all seven feed-resume widget cases passed on the PR1 toolchain (Flutter 3.32.1,
Dart 3.8.1), preserving this test-only PR's dependency lock. Initial run exposed
a harness timeout: pumpAndSettle cannot finish while the waveform animates.
Replaced only the new stimulus waits with bounded asynchronous delivery/pump.
Mutation check temporarily removed the caller write gate: the unknown case failed
with Expected 1 / Actual 2 progress reads, demonstrating the new assertion catches
the missing gate. Restored production source byte-for-byte; restored unknown case
passed, and analysis is clean. The fake's zero completion duration means this
checks dispatch suppression, not realistic end-of-segment timing. No native
build or new provider/device acceptance claimed. Lease released; iPhone preview
from the separate PR2 worktree was left running.
Independent final source review found no remaining blocker in this test delta.

2026-11-02 hourly: prevented preview generation/result from surviving sign-out or
same-key reconfiguration. Result-bound configuration checks also cover decoder
and playback-stop awaits. Review exposed two handoff races, fixed with tests.
13 targeted tests +analysis passed. No paid calls or device changes; provider may
finish an already reserved attempt, no retry/refund. Keep reminder: full RFC and
AMOS/device/provider acceptance remain open.

2026-11-02 hourly: corrected live-preview failure text to distinguish no reserved
attempt from retained reservation and uncertain budget state. 15 targeted tests,
analysis and independent source review passed. No paid calls or phone changes.
Remaining work includes actual iPhone/provider acceptance and AMOS integration;
source tests do not qualify those paths or complete the RFC.

2026-11-02 hourly: added physical-iPhone artifact preflight and reconciled stale
simulator-only TestFlight/preview instructions. Four hermetic CLI tests and actual
read-only --preview-device pass; no runtime/release acceptance inferred. No spend
or device changes. Keep owned reminder: full RFC acceptance remains open.

2026-11-02 hourly: preview DELETE API now removes registered generated MP3s before
metadata, coalesces concurrent deletion and preserves retryability on cleanup
failure. 15 tests pass, including actual temporary-file cleanup/budget preservation;
targeted analysis clean after style fixes. UI delete action and restart orphan
retention remain separate unblocked follow-ups; production retention unqualified.
No spend, phone updates or deployment.

2026-11-02 hourly: closed prior deletion review and added confirmed personal-only
preview delete UI. Explicit callback required; production/team action hidden.
19 focused tests passed; analysis style fix verified; independent review clear.
Physical iPhone delete/queue acceptance and restart orphan cleanup remain open.
No spend, deployment or phone changes; full RFC not complete.

2026-11-02 hourly: verified composed preview deletion through real root/widget/API
and temporary filesystem; 2 tests pass for success/failure+retry, queue restoration,
stop-before-delete and unchanged budget. The native player is fake, so iPhone acceptance
is still open. No paid calls/device updates or automatic purge of published recordings.

2026-11-02 hourly: completed aggregate PR2 validation rather than adding more
preview scope. 110 Flutter tests+analysis and 4 preflight tests pass; final independent
review clear at 83fcd53 / base 6c7e12f. Recorded exact evidence/limits in PR-2-REVIEW.md.
Next: ready PR for actual CodeRabbit review (draft had been skipped), address concrete
findings and preserve separate native/provider/AMOS/full RFC acceptance gates.

2026-11-02 hourly: CodeRabbit completed with three findings, all verified/fixed.
MP3 selection uses reviewed exact IDs plus current catalog/voice; HTTP timeout
aborts transport and cleans late responses; evidence spacing corrected.
21 provider tests, analysis, Go checks and real PostgreSQL fixtures passed;
independent cross-review clear. No paid calls/device changes/deployment. Push
for review of the new head; do not infer approval or full acceptance from prior
successful CodeRabbit check.

2026-11-02 hourly: reconciled CodeRabbit success/no actionable comments at dd7df26
and passed unsigned physical-iPhone preview compilation (arm64/iphoneos). Exact
receipt in PR-2-REVIEW.md; no product source changed, phone updates or paid calls.
Signing/install and actual runtime acceptance remain separate. Full RFC incomplete.

2026-11-02 hourly: full corrected-head Flutter regression suite passed, 114 tests
at 254983b (product code dd7df26). CodeRabbit check succeeded on that head;
summary has no actionable comments, with docstring-coverage warning retained.
Receipt in PR-2-REVIEW.md. No phone update, paid calls or deployment. PR2 remains
open; actual device/provider/AMOS/Nanos acceptance and full RFC remain incomplete.
Keep the owned reminder.

### 2026-11-02 — MP4 metadata protection regression

Isolated continuation branch `fix/mp4-probe-protection`, code `9fa78fe`, from
main `6c7e12f`; the preview PR2 and resume PR3 worktrees remain untouched.
The old probe accepted protection containers in video/audio sample entries,
duplicate AVC configuration and truncated audio children. All four regressions
failed on the original code and passed after rejection and bounded audio-child
inspection were added. Clear synthetic metadata and every truncated prefix are
also checked. Media tests and required Go lint/full unit tests passed.

This is metadata-only hardening, not decodable-video, sample-map/timeline,
publication or runtime acceptance. Independent review remains pending. No paid
calls, phone changes or deployment. Capacity: load 5.76/6.10/5.85; interval idle
26.66/25.76%, normal pressure, no swap churn or thermal/performance warning.
Full RFC acceptance remains incomplete; retain owned hourly reminder.

### 2026-11-02 — MP4 independent review and positive child coverage

Independent read-only review of PR4 at cb00966675d26d9d5d64d358f935136639fee90a
against 6c7e12f found no blocking finding. Reviewer checked surrounding parser,
shared atom budget and callers (no production callers use this probe). Added
its suggested positive case: a clear audio entry with a structurally well-formed
child box, alongside the rejection cases. All focused ProbeMP4 tests passed.
The synthetic esds payload is not an AAC descriptor/decode validation fixture.

Preparing PR4 for hosted review; the earlier CodeRabbit check was skipped because
it was draft. No merge, provider calls, native changes or deployment. Capacity:
load 4.38/4.11/4.02; interval idle 34.55/23.74%, normal pressure, zero swap churn,
no thermal/performance warning. Full RFC remains incomplete; retain reminder.

### 2026-11-02 — CodeRabbit audio-version correction

CodeRabbit check on 23b2b03 succeeded but comment 4446593715 found a valid defect:
the fixed 28-byte audio header offset rejected clear version-1 entries and could
consume a following protection box as packet data. Reproduced five failing cases
before fixing version dispatch: v0 uses 28 bytes, v1 uses 44, unsupported versions
and truncated extensions reject. Six added cases cover clear v1, protected v1
with zero/nonzero samples, short/truncated v1 and unknown versions. All media
tests pass after the fix. This supersedes earlier review's no-blocker conclusion
for this issue; a passing hosted check alone was not sufficient.

Apple's sound sample description version 1 reference confirms the four extra
32-bit fields: https://developer.apple.com/documentation/quicktime-file-format/sound_sample_description_version_1
No codec/timeline/publication qualification is inferred from these structural
fixtures. No provider calls, deployment or phone changes. Capacity load
3.39/3.36/3.49; interval idle 36.19/28.52%, normal pressure, zero swap churn,
no thermal/performance warning. Full RFC incomplete; retain reminder.

### 2026-11-02 — AVC configuration ordering coverage

Hosted check on e82a642 passed and CodeRabbit marked the audio-offset thread
resolved. The final review body also suggested undersized-then-valid AVC coverage.
Added that ordered case and a sole undersized configuration, with fixture override
that replaces rather than appends to the default valid configuration. Confirmed
the ordered case fails against original main 6c7e12f (incorrectly accepted), then
restored the corrected source and passed the full media package. No production
code changed. Merge held for review of the added coverage; no bypass or deployment.

Capacity load 4.29/4.51/4.47; interval idle 46.26/36.31%, normal pressure,
zero swap churn and no thermal/performance warning. No paid calls or device
changes. Full RFC remains incomplete; keep the owned reminder.

### 2026-11-02 — Typed paid-provider outcomes (PR5)

Isolated worktree prosody-provider-outcomes / feat/provider-attempt-outcomes.
Added typed pre-dispatch/response-received/may-have-acted observations and safe
errors through OpenRouter and briefing ports; IDs/known usage survive rejection,
with unusable output cleared. Hermetic dropped/truncated/canceled/HTTP-failure
checks exercise one call and no retry; preflight rejects without HTTP. Unknown
states normalize conservatively. No lookup/dedup or native authority introduced.
See PROVIDER-OUTCOMES.md for semantics, review corrections and fixture limits.
Initial focused checks and full Go lint/unit hook passed; final affected checks
and independent review follow the corrections. Capacity admitted at load
8.60/8.07/7.75, interval idle23.88/40.35%, normal pressure, no swap churn or
thermal warning. No paid calls/device changes/deployment; full RFC incomplete.
AMOS operational ownership is now reconciled in its retained ledger; its staged
held-issuance source is not a qualified worker/mobile interface. Keep adapter
pinned and retain the hourly reminder.

### 2026-11-02 — PR5 review corrections, verification held

CodeRabbit check on0de0cc8 succeeded but comments4449100666/4449100704 found
valid gaps: finalized error-only metadata was not backfilled into legacy flat
fields, and fresh transport/read errors discarded cancellation/deadline identity.
Working source now synchronizes flat refs/accounting and wraps original causes
inside the fixed safe AttemptError. Regression assertions added for rewrite and
speech error-only fields and errors.Is cancellation/deadline/unexpected EOF.

Formatting and diff checks pass. Tests, lint and commit hook NOT RUN on this delta:
capacity refused at interval idle1.12/2.99% (load19.30/17.16/16.51), normal pressure,
zero swap churn and no thermal warning. No lease claimed or automatic admission
retry. PR5 remains at published0de0cc8; working changes/private patch preserve
this unverified correction until an authorized capacity change permits checks.
No paid calls, native changes, deployment or full acceptance; retain reminder.

### 2026-11-02 — PR5 corrections verified after fresh admission

Fresh capacity audit after the new availability event passed: interval idle
25.34/36.91%, load4.53/4.64/4.83, normal pressure, zero swap churn, no thermal
warning. Owned lease acquired for bounded verification. Both CodeRabbit fixes
were exercised against published0de0cc8: cancellation/read/deadline identity and
flat error-only metadata regressions fail on old source. Restored corrected
source passes complete openrouter/briefing packages. Independent read-only
review of this correction found no blocker. Required hook remains enforced at
commit; no provider calls, deployment, native API qualification or sign-in change.

Read AMOS public-interface-proposal-v2 as LOCAL preparation only. Its worker/
human route names remain unimplemented, unpublished interfaces. Session-bound
first qualification profile and intended private iOS callback/client accepted;
canonical audience and actual closed-range player compatibility remain open.
Production consumption and adapter migration remain disabled. Full RFC incomplete;
keep the owned reminder.

### 2026-11-02 — Typed-nil provider diagnostic regression

AMOS semantic review identified a typed-nil AttemptError panic in landed PR5.
Isolated worktree prosody-provider-nil / fix/typed-nil-provider-errors now handles
that malformed diagnostic as MayHaveActed, retaining available result refs/usage.
Error/Unwrap are nil-receiver safe, including wrapped/joined chain traversal;
legitimate sentinels remain discoverable. Rejected script/audio stays unusable,
with exactly one provider-port call and no retry. No authority/journal expansion.

New tests reproduce the panic on original main2d93bf4 in direct and briefing
paths. Corrected source63fa4b2 passes full openrouter/briefing packages and required
Go lint/full unit gate (zero lint issues). Independent source review clear.
Capacity admitted: load6.89/7.78/7.47, interval idle14.82/59.34%, normal pressure,
zero swap churn, no thermal/performance warning. No live calls/native changes or
deployment. Hosted review/merge remain separate; full RFC incomplete, keep reminder.

### 2026-11-02 — Local verified media-range preparation

New isolated worktree prosody-media-ranges, branch feat/verified-media-ranges,
base main2d93bf4. Prepared ReadVerifiedRange and hermetic tests for <=1MiB
closed ranges, full-object size/digest binding, same-pass capture, corruption
outside the requested interval, overflow/bounds and no-output failure handling.
This is local port preparation only; no AMOS/HTTP registration or native grant,
MIME/resource closure, publication or actual player compatibility is asserted.
See MEDIA-RANGES.md. Capacity refused: load7.26/6.84/6.61, interval idle
4.82/34.98% (average19.90%), normal pressure, zero swap churn, no warnings.
No tests/hook/commit/lease or automatic admission retry. Source and recovery
patch are retained privately until verification can run; no draft PR can be
created before the required commit gate. Full RFC incomplete; retain reminder.

Independent source review found two issues in initial range preparation: digest
length must be checked before hex decoding to preserve the allocation bound,
and arbitrary reader errors require safe rendering. Both corrected in staged
source; tests added for oversized digest/no reader access, private read-error
sanitization with errors.Is, mid-read cancellation and data returned with EOF.
These regressions remain UNEXECUTED under capacity hold; source review is not
runtime evidence. Original two findings superseded only at source level.

### 2026-11-02 — Verified media-range first executed checks

Staged source reconciled onto maina5de1fa without changing range source/tests;
original recovery stash retained, both histories preserved. Fresh admission
passed: load3.13/3.04/2.87, idle46.20/53.74%, normal pressure, zero swap churn,
no thermal/performance warning. Five named range tests passed under owned lease.
Independent source review already cleared exact helper/test bytes after digest
allocation bound and safe error corrections. No DB fixture, native route, service,
provider calls or iPhone changes. Actual I/O/deadline/cleanup and iOS range profile
remain acceptance gates. Required hook stays enabled; full RFC incomplete.


### 2026-11-03 UTC — Current continuation checkpoint

Landed source/helper checkpoints and their local-check receipts: PR3 `b93bf7f`,
PR8 `9b5a922`, PR9 `63173b7`, PR10 `c4d7762`, PR11 `1a9c716`, PR12 `cb092b4`.
Earlier holds above remain historical. PR7 `f596a1b` is still draft, with database
tests SKIPPED. PR2 `fb2f52d` passed seven merged resume Flutter tests plus lint/full
ten-package Go units; hosted success did not close its concrete spacing finding.
This documentation-only checkpoint corrects that finding pending refreshed review.
Producer source choice is pending, neither alternative activated; native B/C,
source bridge, provider/device, Nanos and TestFlight acceptance remain open.
No deployment, spend or signing; full RFC incomplete, retain the owned reminder.


### 2026-11-03 — Landed preview and rebased composition checkpoint

PR2 is LANDED at `1cd648b1f510ba14bcf2ada1ce7a8ec2f5222b8f`; earlier
pending-review entries are superseded. The final CodeRabbit check was SKIPPED
because no new files required review after the source-equivalent rebase;
this is not pending or fresh code-review evidence.

PR13's test-only composition branch rebased onto that landed main, retaining
its test bytes and synthetic authority/receipt boundary. A fresh Go 1.27.0 /
GOMAXPROCS=2 focused run passed all three top-level composition tests and seven
subtests (10 pass records, no skips), exercising current briefing policy revision
2, real synthesis/MP3 decoding and exact artifact bytes/hash/duration. The
preserved Go commit hook gates this checkpoint. Hosted review and merge of PR13
remain pending. These local fixtures do not qualify native settlement, database
atomicity/durability, provider transport or device acceptance. Full RFC, native
B/C/source bridge, approved database fixture and device/TestFlight gates remain
OPEN; retain the owned incomplete-RFC reminder. No paid calls, provisioning,
deployment or signing occurred.

### 2026-11-03 — Publication row-count guard child

Extracted the existing PR #7 publication guard/test verbatim from f596a1b into
`fix/publication-row-counts`, based on main `359c7504`. PR #7 depends on this child;
original checkpoint worktree/claims remain preserved. SQL literals/argument
expressions are unchanged. Eight negative cases fail on original main; all nine
pass with the guard. These are transaction-double tests, not DB/native atomicity.
See PUBLICATION-ROW-COUNTS.md for traceability and acceptance limits. No checkpoint
migration, service, provider call, DB fixture, signing or deployment in this child.
Fresh capacity interval idle 16.83/26.53% (average 21.68%), normal pressure, zero swap
churn, no thermal warning; verification uses the owned shared build lease. Full
RFC remains incomplete; keep the owned reminder.


### 2026-11-03 — Typed-nil briefing provider guard

Reproduced the missing-port bug on main `dc95bca3`: actual nil
`*openrouter.Client` values panic through both public briefing APIs, while
callable nil receivers can reach dispatch. The unchanged-source regression was
RED (5 passed / 8 failed test/subtest records). A shared unexported nil-provider
guard now rejects missing ports before token accounting or invocation, without
recovering arbitrary provider panics or changing post-handoff handling.

Fresh Go 1.27.0 / GOMAXPROCS=2 full briefing suite passed all 54 test/subtest
records, including all 13 new missing-port/context records; no skips. Existing
valid-provider, original-cause and post-handoff uncertainty controls passed.
Missing ports return typed raw NotDispatched with no usable output, identifiers
or retry; that observation grants no native no-effect/refund authority. The
preserved lint/full-Go hook gates the checkpoint. Independent review and hosted
review/merge remain pending. No provider HTTP, real key, database fixture,
provisioning or device changes. Native/C1, database/device and full-RFC gates
remain OPEN; retain the incomplete-RFC reminder.


### 2026-11-03 — Local Files.Put pre-publication cancellation

On main `faf5adb3`, a tiny owned-file regression reproduced a final reader
returning bytes plus EOF while canceling context: Put published the 14-byte
target and returned success. Already-canceled/expired calls created workspace
directories. All four new test/subtest records were RED. Put now checks context
before filesystem effects, after copying and immediately before the atomic link.
Fresh Go 1.27.0 / GOMAXPROCS=2 full media suite passed all 31 named records,
including the four regressions, with no skips. Existing byte-identity,
immutability and Reconcile controls passed; the unchanged lint/full-Go hook
gates this checkpoint. Independent review and hosted review/merge are pending.

This is owned local-development/qualified-volume behavior, not Nanos production
storage qualification. The fresh-ID regressions observed the original context
cause, zero Object, no target and no upload files; temporary cleanup is best
effort. Cancellation does not prove absence of a prior immutable object.
Cancellation can race publication; post-Link errors remain uncertain and require reconciliation, not
target deletion or rollback/no-effect/refund proof. No general reader-interrupt
guarantee is added. No native/C1/database/provider/device/provisioning changes;
full RFC and native/DB/device gates remain OPEN, retain the owned reminder.


Cancellation replacement follow-up: the tiny existing-object regression against
an old-source overlay preserved the original bytes/hash but returned Link's
file-exists error after final-read cancellation (1 passed / 2 failed named
records, including the parent test). With unchanged fixed production source,
the fresh full media suite passed all 34 records, no skips. Both already-canceled
and late-canceled replacements return the context cause/zero Object and retain
the original bytes, size and hash; the owned fixtures leave no upload temp.
These observations are not rollback/no-effect proof or global cleanup promises.


Post-close boundary follow-up: public Put now delegates to its private
implementation core with fixed real `(*os.File).Close`; Files/NewFiles/public
API and deferred actual-close cleanup are unchanged. A per-call test closer
performs real Close, verifies the descriptor is closed, then cancels context.
The internal test exercises that same core, not the public wrapper directly;
existing public Put controls retain the fixed real-close path. A compiled
mutation removing only the final pre-Link context check published the 14-byte
target and failed the new test. Restored source passed all 35 named records in the media suite, with no
skips. The required lint/full-Go hook also passed for this follow-up.
No mutable global/instance hook, timing race or filesystem framework was added.
Post-Link uncertainty, best-effort cleanup, prior-object reconciliation and all
native/I/O/Nanos qualification limits remain unchanged; no new authority proof.


## Current checkpoint — 2026-11-04

This checkpoint supersedes older READY, local-staged and CPU-hold labels for
the work listed below. Earlier entries remain historical evidence, not current
pending-source status.

- PR #17 landed at `6ec335655cf3676fec04b2f2de3a9401ed420566`. Flutter analysis
  reported zero issues; 27 focused and 129 full-suite named tests passed with
  zero failures or skips. These are hermetic results, not device/APNs acceptance.
- PR #18 landed at `ccdd549f8f9d44e20429fe4c1a4ac8d58b8ca99e`, matching reviewed
  tree `5217b17a2c7d8c54e260c983470d16dfe819485b`. The preference suite passed
  33 named records across seven top-level tests, zero failures/skips. Compiled
  old-source cancellation regressions produced seven failure records while the
  positive control passed. Latest lint reported zero issues and all 12 Go
  packages passed the required unit gate; this does not qualify real preference
  SQL locking or native authority.
- PR #7 remains DRAFT at `1b55d2baffebf764a240d07775cf09348f644044`; its domain
  PostgreSQL evidence is published. Checkpoint source inputs remain identical
  to tested `28ce03d2`: 132 named store test/subtest records with the database
  fixture available, across 32 top-level tests, passed with zero failures/skips.
  `TestPrivateCheckpointPostgres` executed and passed. The one fixture approval is COMPLETED and
  CONSUMED, not renewed. Migration 002 remains unregistered/disabled in the
  application; native atomic receipts and current authority remain open.

Deletion preparation is documented in DELETION-CLEANUP.md. The missing asset
ownership/provenance, complete reference index, backfill and shared-reference
locking protocol still prevent safe cleanup eligibility; no supported cleanup
consumer is composed. The deletion source anchors assessed at `b629600f` remain
byte-identical on the current `ccdd549f` base. No deletion API or physical cleanup
has been added.

No owned heavy job or build lease is active for this checkpoint; that observation
does not admit future work without a fresh capacity check. C1 remains parked and
the AMOS pin is unchanged. No qualified AMOS application-worker/publication or
human feed/media interface handoff, production InteractiveSession, APNs delivery,
physical-device or TestFlight acceptance is claimed. No new fixture, provider call, spend, deployment or signing is authorized
by this record. Producer-label policy, public origin and App Store Connect setup
remain unselected or unverified; no defaults have been activated. The full RFC
remains incomplete, so the owned implementation reminder stays in place.


## 2026-11-04 — Activity access-denial cache invalidation

Activity refresh and pagination previously kept loaded cards and readable
originals after an explicit HTTP 401 or 403. The view now discards those cards
and its pagination state on either denial. It retains loaded activity after
ordinary transport, malformed-response or server failures. Reading originals
is disabled during a pending refresh/page request; captured callbacks also
refuse access after denial or replacement. Recovery requires an explicit new
first-page request, not an automatic retry of a stale cursor.

At source tree `e2c76dadad481378ee22ec225ed016e958a63d93`, Flutter analysis
reported no issues. All 17 focused Activity tests and all 141 full-suite tests
passed, zero failures/skips. Compiling the same tests against the original
Activity source produced six expected failure records (denial/pending cases)
while 11 controls passed. The fixed source was restored exactly afterward.
Independent final source review cleared this bounded change. These Flutter results do not establish the
required repository commit gate; consult the PR verification record for that
separate outcome.

These are hermetic widget observations and local view invalidation, not recall
of already delivered bytes, native session/revocation or physical-device proof.
No production sign-in, AMOS adapter, credential, provider, fixture, deployment or
signing behavior changed. PR #21's separate test-only HTTP composition landed at
`d93eecd58a3e532acd0a24c04d80bcd002823a0f`, with its reviewed tree verified on
main; it also supplies no native/provider/mobile acceptance. PR #7 remains draft,
its one SQL-fixture approval completed/consumed, and C1 remains parked. The full
RFC and TestFlight acceptance remain incomplete; the hourly reminder stays.


## 2026-11-04 — Bounded MP4 sample-map consistency

The existing `ProbeMP4` now checks required size/chunk/time tables, raw track
tick totals, self-contained data references and declared mdat containment, with
shared sample/chunk/metadata-read bounds. It still returns unverified movie
header duration and dimensions. No sample bodies are read or rewritten, and no
production caller, media/publication admission or native route was added.

At tree `fbb983833fe270557afb6c74a35b1c46cec2763c`, 129 named focused MP4
records across 15 top-level tests passed with zero failures/skips. Compiled old
source accepted the missing-size-table and wrong-time-total negatives; the
containment-only omission accepted both header-range negatives. Each overlay
produced three failure records (two cases plus parent), while the valid child
and parent passed. The scoped supervisor stopped on an incorrect expected-count
assertion after the containment command selected an additional suffix-matching
case. Independent log audit confirmed the actual executed cases; its planned
full-media step was not reached and is not claimed as passed. Consult the PR
verification record for the separate required repository hook outcome.

See MP4-SAMPLE-MAPS.md for profile limits and verification scope. Synthetic
containers are not decodable-media evidence. Complete codec/presentation timing,
physical/immutable object identity, actual player/Nanos behavior and publication
remain unqualified. The full RFC/TestFlight gates and owned hourly reminder stay
open; no provider, database fixture, signing, deployment or AMOS pin change occurred.


## 2026-11-04 — Replacement-key editor cleanup after save denial

A current settings save returning HTTP 401 or 403 now clears the typed
replacement OpenRouter key from its editor. Other failures retain the unsaved
draft. The existing request-generation guard keeps a late denial from an old
feed from clearing a new feed's draft; error messages, busy state, preferences,
model catalogs and authentication interfaces are unchanged.

At source tree `5779200e67e8a07aac2e82b581a22819f1110477`, compiling the new
widget tests with the original Settings source produced the two intended denial
failures while eight controls passed. With the fix, analysis found no issues;
all 10 focused settings tests and all 148 full Flutter tests passed, with zero
failures/skips and successful final runner completion. The fixed source was
restored exactly after the old-source check. Independent source review cleared
the change. Consult PR verification for the separate required repository gate.

This is local editor cleanup, not memory zeroization, removal of request copies,
stored-key deletion, credential revocation or proof of native session authority.
Only synthetic keys and hermetic HTTP doubles were used; no live provider,
database fixture, device, signing, deployment or AMOS pin change is included.
MP4 structural checks separately landed in PR #23 at
`92b22827dec7fd24f4d6d0450db8f1ddb9b997f1`; their decoding/publication limits
remain open. Full RFC/TestFlight acceptance is incomplete, so the owned hourly
reminder remains enabled.

## Rewrite output envelope

Rewrite now rejects invalid UTF-8 and scripts above 64,000 bytes after its
existing whitespace normalization, matching speech preflight. Rejection keeps
provider request/generation references and token observations while withholding
both usable script and provider text; it performs no repair or paid retry.

The compiled original-source overlay failed the three intended negative leaves
(ASCII overflow, multibyte overflow and invalid UTF-8) plus their parent, while
four valid controls passed. Corrected source passed all eight named envelope
records and all 62 briefing records, with zero failures/skips. These are hermetic
provider-double checks, not live provider, native settlement, database or device
acceptance. Full RFC/TestFlight acceptance and the owned hourly reminder remain.

## Worker acceptance-profile preparation

[WORK-ACCEPTANCE-PROFILE.md](WORK-ACCEPTANCE-PROFILE.md) maps the actual
submission/settings bytes, replay identity and draft checkpoint chain to the
remaining AMOS durable-binding requirements. It is a review proposal, not a
native schema, constructor, grant or enabled worker. Resource incarnation/version
history, explicit cancelled/expired-authority state and retention decisions remain
open. No source/backend behavior changes or new database/provider/device tests
are claimed by this documentation. AMOS implementation remains cloud-owned;
production sign-in, full RFC and TestFlight acceptance remain incomplete.

## Current readiness reconciliation after PR #30

PRs #26, #27, #28, #30 and #31 are landed. The current source and remaining
acceptance predicates are listed in [READINESS.md](READINESS.md). Native mobile
authentication and application-work integration remain AMOS cloud-owned; local
AMOS writers remain stopped. The client source and unsigned iOS compilation
do not qualify real authenticated playback. Signing attempts stopped after
`errSecInternalComponent`; no install or TestFlight upload succeeded.
PR #7 remains draft with its one ordinary SQL approval completed/consumed;
native receipt/grant/fence gates remain open. Full RFC and TestFlight acceptance
are incomplete, so the owned hourly reminder remains enabled.

## Existing Agent Prosody identity

The owner chose reuse of the existing Agent Prosody app/account. Runner and
RunnerTests identifiers now use `run.sire.prosody` and its test suffix; display
name is Agent Prosody. The intended OAuth callback is unchanged and remains
separately gated. Six project fields and only the plist display-name value
changed; four hermetic preflight checks pass. No native build/signing/upload
was performed. AGENTS.md requires approved self-hosted macOS runners; the new
repository currently lists zero available repository runners. Four ASC secret
names exist, but Apple API validity and actual build history remain unverified.

## Reused-identity paid-preview guard

PR #33 review identified that a new app sandbox could reset the original $1
preview allowance. Paid preview now defaults disabled and is explicitly disabled
at startup. Initialization does not create/read a ledger; configuration and
generation refuse before key retention, reservation or provider dispatch. Sample
briefings remain available. No ledger migration, allowance renewal or access to
another app sandbox is performed. Synthetic tests explicitly opt in where needed.

The first analyzer run failed because this isolated worktree lacked its generated
package configuration; tests were not reached. Offline dependency resolution
restored that prerequisite. Corrected analysis passed and all 19 focused tests
passed with zero errors/skips and successful runner completion. Independent source
review cleared the six-file runtime/test/documentation followup. The complete
Flutter suite then passed all 209 named tests, with zero errors/skips and successful
runner completion. Required repository-hook evidence is recorded separately.
No provider, native build, signing, device, upload or deployment acceptance is
claimed. Full RFC acceptance remains incomplete and the hourly reminder remains.

## Decoded aggregate narration measurement

[NARRATION-BUDGET.md](NARRATION-BUDGET.md) records the new local validator:
actual generated MP3 parts share one duration/byte budget, with exact sample-time
comparison before rounding. All 17 focused records passed with zero skips;
compiled per-part-budget and per-part-rounding omissions failed their intended
assertions while positive controls passed. One-part synthesis uses the helper,
but complete native publication must still validate every generated part from
its admitted manifest. No paid retry, shortening loop, native receipt or
production activation was added. Full RFC/TestFlight acceptance remains open.

## Cloudflare Go/Pulumi DNS preparation

`infra/cloudflare` prepares protected adoption of the existing prosody.chat DNS
records. The zone/account are verified before existing-ID resource registration;
zone lifecycle, billing and backend activation remain outside this slice. The
private read-only snapshot contains 14 records and is ignored, mode 0600.

The nested module passed 18 named unit/mock records, vet and lint with zero issues. Separate parser/
private-snapshot evidence is scoped in its README; initial compilation timeouts
executed zero tests and are retained as historical failures. Live preview/import/
apply have not run. Cloudflare MCP OAuth token-permission discovery returned
403; creation was not attempted and no token was created. Pulumi provider
authentication remains a gate. No DNS or Linode
resources changed, and no TestFlight upload/full RFC acceptance is claimed.

## Strict Nanos template JSON inputs

The offline Nanos checker rejects duplicate decoded JSON keys instead of silently
keeping the last value. It requires UTF-8 objects, finite values, valid Unicode
strings and bounded decoder input/nesting, plus exact integer schema-version and
boolean reboot types. Valid escaped keys remain accepted. No executable host,
image build, boot, runtime provenance or deployment is supplied by this guard.

The nine Python test roots passed with zero failures/errors/skips, including
66 successful subcases (overlapping the root suite, not an additive total). The
new regression table executed against original source produced seven assertion
failures and one wrong-exception error; six controls passed, comprising two valid
inputs and four invalid inputs already rejected. Independent review identified an initially unbounded file read and exponent
overflow in the decoder; both were corrected with direct guard tests. Corrected
self-check passed while explicitly retaining runtime/image qualification BLOCKED. Repository-hook
and independent-review receipts remain separate. No fixture/provider/signing or
infrastructure effects occurred; full RFC acceptance and the reminder remain open.

## Linode powered-off Nanos staging source

`infra/linode` declares an offline protected Fremont VM, private-image disk and
direct-disk boot profile through Go/Pulumi, with Linode provider v6.9.0 pinned.
Both boot flags remain false. Existing image metadata and a compute-only quote
ceiling of $12/month are checked before resource registration. The complete
$50/month plan, image provenance, data services, backup/restore, firewall/TLS,
actual boot and backend acceptance remain open. Powered-off instances can still
incur charges; no live preview/apply or resource creation occurred.

The nested module passed 19 named unit/mock records with zero failures/skips,
plus vet/lint. Compiled boot-flag mutation failed its graph check while the price
control passed. Compiled compute-cap omission failed the intended rejection and
its parent while the valid graph control passed. These are synthetic registration
checks, not provider, runtime or billing acceptance. Root-hook and final source
review evidence remain separate. No DNS, signing, provider work or AMOS pin change
is included; full RFC/TestFlight acceptance and the reminder remain incomplete.

## Approved MacBook runner reassignment source

The owner approved reassignment of the existing Prosody MacBook runner to this
repository. `infra/runner` owns the operation through Go/Pulumi; the host-local
helper preserves the installed package and legacy workspace, requires an exact
host/local/server identity profile, and never puts one-hour tokens in argv or
IaC inputs. Local processes, busy registrations and foreign services are refused.
Checkpoints/backups and custody parent entries are synced before configuration
changes; interrupted owned subprocess groups have bounded cleanup. Hard termination
still requires actual-state reconciliation, never blind re-registration.

Thirteen hermetic Python roots pass. A real read-only plan passes with no effects:
the old server registration is now absent, while local metadata remains. Only a
successful complete source inventory permits supported local-only cleanup; denied
reads never authorize that branch. The nested Go glue compiles and passes vet/lint; it has no Go test cases.
Independent source review is clear. Actual IaC preview/apply/online evidence
remains separate. No signing or TestFlight build is
claimed by this source checkpoint. Full RFC acceptance and the reminder remain open.

## MacBook runner operational verification

The owner-approved saved IaC plan applied successfully. GitHub now reports one
online `prosody-laptop-runner` in `narrate-it/prosody`, with `self-hosted`, `macOS`,
`X64` and `prosody` labels. The local repository/ID/work-folder binding matches
the server, the validated target LaunchAgent is loaded, and new jobs use the
isolated workspace. A follow-up Pulumi preview reports two same resources and
zero changes. This closes runner registration/availability at observation time,
not permanent uptime; the Mac must remain awake and connected.

The actual transition used the verified stale-server branch; it did not delete
another server registration. An initial operator verification read failed on the
official metadata's UTF-8 BOM; correcting that read-only parser confirmed the
same state without any reconfiguration. No iOS job, signing retry, Apple API call
or TestFlight upload occurred. Signing/backend/native acceptance remain separate
gates. Full RFC acceptance and the hourly reminder remain open.

## Runner hosted-review corrections

Hosted review found malformed inventory/metadata/token shapes could escape the
fixed refusal path, and the synthetic ownership fixture depended on the invoking
UID. The helper now validates JSON/plist objects, runner rows and IDs, label/token
strings and checkpoint fields before use. Linked or malformed checkpoints refuse
recovery. The test fixture models only its own tree as user-owned, preserving an
explicit root-refusal control.

Twenty-one hermetic test roots PASS with zero failures/skips. Added controls cover
partial removal/registration failure, registered-but-offline reconciliation,
missing/invalid/linked custody, and process/service changes at the second guard.
New rejection tests over the previous production source fail with five assertion
failures and eighteen errors across five selected roots; the read-only plan
control passes. These are source regressions, not live recovery exercises. No
reassignment/apply, runner restart, credential rotation or signing occurred. The
source digest changes a future IaC preview; the already-online installation was
not reconfigured. Full RFC and TestFlight acceptance remain open.

## Runner token preflight finality correction

Final independent review identified that a NUL-bearing target token could pass
string/whitespace validation and fail later during environment construction,
after old registration cleanup. Both token entry points now require bounded
printable ASCII before any removal. The target-only malformed-token control
keeps the removal token valid and proves zero local/configuration effects with
the old service and registration retained. NUL, invalid Unicode, non-ASCII and
excessive-length variants are covered. Twenty-two hermetic roots PASS with zero
failures/skips; independent final source review clears this delta.

The prior-validator overlay produced two assertion-failure records (the NUL
subtest and parent), with the read-only plan control passing. Later malformed
variants were not reached in that old-source run. Its supervisor incorrectly
expected four failures and also failed; a read-only audit retains that failure
and the actual named results without rerunning or promoting unexecuted controls.
No live tokens, runner reconfiguration, API mutation, signing or upload occurred.
Prior 21-root evidence remains scoped to its earlier checkpoint. Required hooks
passed lint zero/all twelve cached Go packages; no database fixture was configured.
Hosted review of the new revision and full RFC acceptance remain open.

## Cloudflare canonical import-input custody guard

Historical adoption of fourteen records succeeded without DNS drift, but persisted
canonical content inputs were plaintext despite desired Pulumi secret flags.
The source documentation now distinguishes desired flags, encrypted output fields,
absent fields and actual canonical input custody. This does not rewrite or erase
that private state, backups or prior snapshots.

The new source requires per-record `public-reviewed-v1` classification and refuses
nonempty comments/tags. Both JSON loading and direct registration check every
record before provider invocation. A marker is operator classification, not proof
of public data, encryption or authority to apply; proxied origins can be private.
The guard checks only supplied inventory. Remote values may drift before import;
private state custody and fresh comparison remain necessary, and confidential
canonical-input handling is still unqualified.

The ordinary nested-module suite passed 31 named records across ten roots with
zero failures/skips. Compiled omission of both admission checks produced twelve
expected failure records, including parents, with the valid public-input control
passing. Initial race compilation hit a 300-second watchdog before any test event;
no race pass is claimed. Independent source review is clear with the documented
remote-drift limitation. Race, vet/lint and required-hook results stay separately
scoped. No real inventory was classified or edited; no DNS/provider/state update,
secret read, signing, upload or deployment occurred. Full RFC/reminder remain open.

## Cloudflare custody guard race follow-up

After fresh capacity admission and a new exact owned lease, one bounded race
follow-up completed on unchanged source: 31 named records across ten roots PASS,
zero failures/skips, with no cached test verdict. These overlap the ordinary
31-record suite and are not additive. The first five-minute compilation watchdog
still executed zero cases and remains a retained failure. Its owned process group
was confirmed empty before the final follow-up receipt.

Nested-module vet/lint pass with zero issues. The preserved root hook passed lint
zero/all twelve Go packages without a DB fixture. Independent source review clears
the supplied-inventory guard with its explicit remote-drift limitation. Draft PR39
holds the source; hosted review of the final revision remains separate. No real
inventory, provider state or DNS was read/changed in qualification; no credential,
signing, upload, infrastructure apply or deployment occurred. Sensitive canonical
input support, historical custody/remediation and full RFC acceptance remain open.

## Read-only archive identity source

Added a standalone checker for a future owner-controlled xcarchive's selected
bundle/version/build and modern thin ARM64/iOS stamps. It rejects simulator or
ambiguous app metadata, traversal/symlinked descendants, identity disagreement,
invalid load-command bounds and observed mutation during the read. It does not
build, sign, install, inspect Keychain state or contact Apple. Success means only
metadata identity; signatures, profiles, source provenance, SDK/upload rules,
unused build numbers, executable validity and release acceptance stay unverified.

Twenty-two hermetic roots PASS, zero failures/skips, using synthetic archives and
non-runnable header fixtures only. Independent source review found binary-plist
allocation amplification despite the input byte bound. A pre-decode object/extent,
reference-budget and acyclic-depth guard now rejects compact alias/overlap/cycle,
deep and oversized-reference fixtures before plistlib allocation while allowing
legitimate shared object-index references. The earlier twenty-root smoke result
predated that correction and is not allocation qualification.

In-memory omission of the platform gate produces two expected failures with a
passing identity control. Omission of binary-graph admission produces five
expected adversarial failure records with a passing shared-reference control.
These are hermetic boundary checks, not native signed-archive or Apple evidence.
No actual artifact was created/inspected, no credentials were read, and no signing,
upload, provider, state or deployment effect occurred. Required-hook and final
independent-review evidence stay separate; full RFC and the reminder remain open.

## Read-only ASC preflight source

A new manual-only workflow is restricted to trusted main and the approved
self-hosted macOS Prosody runner. It uses the existing four repository secrets
to read exactly one matching `run.sire.prosody` app through a fixed-origin TLS GET.
The short-lived JWT has read-only app-list scope. It does not configure signing,
change Keychain state, reserve a build number, upload or enable production auth.
Team association, build history and actual Apple API access remain unverified.

Eleven hermetic Python roots PASS with no failures/skips, including real system
signing/verification using an ephemeral synthetic key, malformed/ambiguous app
refusal, redirect/error redaction, isolated-Python shadow refusal and SIGTERM
private-key cleanup. Initial tests exposed a platform-specific curve-text parser
and a fixture newline expectation; both were corrected before this result. No
Apple credential or live request was used. Independent review found module-shadow
and termination-cleanup gaps; source now uses fixed system Python isolated mode
and controlled SIGTERM unwinding. Required-hook/final review results are recorded
separately; source tests do not establish account/signing/TestFlight acceptance.
Full RFC acceptance remains incomplete and the hourly reminder remains enabled.

## ASC hosted-review corrections

The manual workflow now uses JSON-form YAML so hermetic tests parse and inspect
the actual job/step structure rather than matching text. Tests require trusted
main, exact self-hosted labels, read-only permissions, immutable checkout without
persisted credentials, and secrets only on the read step. Five intentionally
misplaced guard/credential/permission/secret variants are rejected. All twelve
hermetic roots PASS with zero failures/skips. Every named Python function now has
a docstring; this addresses the hosted 50-percent coverage warning without
changing production preflight code. Independent review clears the final delta.
No Apple request, signing, upload or account change occurred. Hosted review of the
new revision remains a separate gate; full RFC acceptance remains incomplete.

## ASC environment protection prerequisite

The next hosted review completed with docstring coverage 100 percent, but raised
a valid secret-boundary finding: repository-wide ASC credentials are not protected
by a mutable workflow branch condition. The job now selects `prosody-asc` and
uses only new `PROSODY_ASC_*` names. Missing environment values cannot fall back
to old repository `ASC_*` copies. Go/Pulumi prepares a protected, exact-repository
main-branch-only environment through a bounded reconciler; it changes no secret.

Eight hermetic environment-policy roots and thirteen ASC preflight roots PASS,
zero failures/skips. These cover creation ordering, foreign/broader policy refusal,
partial-write reconciliation, explicit-status absence, opener/header/query binding,
malformed-links safe errors, and structural environment/secret-fallback denials.
Independent source review is clear. Source tests do not prove live policy, secret
migration, Apple access or signing. Preview/apply evidence remains separate.

Read-only GitHub inspection found no environment yet, four legacy repository
ASC names, no namespaced replacements, and zero accessible organization secrets.
GitHub cannot return stored secret values for copying. Owner entry into the
protected environment and verified removal of old copies after consumer checks
remain required. No secret was read, copied or deleted; no Apple call, signing or
upload occurred. The security thread/merge and operational activation remain OPEN
until required configuration/migration evidence exists; full RFC remains incomplete.

## Main-only ASC environment operational checkpoint

The independently reviewed saved Go/Pulumi plan applied successfully. GitHub
confirms `prosody-asc` has custom deployment policies and exactly one branch-only
`main` rule. The protected command is retained in private IaC state; the subsequent
preview reports two same resources and zero changes. This is an observed policy
checkpoint, not permanent protection against a repository administrator.

Secret metadata confirms the environment is empty, the four old `ASC_*`
repository copies remain, and no new `PROSODY_ASC_*` names exist at repository
or accessible organization scope. No secret value was read, copied or deleted.
The owner must supply the original values in the new environment; broader copies
must be retired after consumer verification. Until then, the security finding,
source merge and live preflight remain gated. No Apple request, signing, upload,
backend deployment or native qualification occurred; full RFC/reminder stay open.

## Owner-confirmed protected ASC copies

The owner reports entering the original four values into `prosody-asc`. Fresh
GitHub metadata confirms the standard `ASC_*` names there and the same four names
still at repository scope. Workflow/test expectations now match the names supplied;
no re-entry is requested. Metadata proves placement, not value equality or Apple
validity. The source explicitly retains the security/activation gate until reviewed
IaC retirement of broader copies and consumer/absence verification. No secret
value was read or changed, and no Apple/signing/upload operation occurred. The
updated preflight suite executed thirteen named hermetic roots PASS, zero failures
or skips; this includes structural mutation denials and synthetic OpenSSL signing,
not Apple account access. Fresh admission averaged 70.965 percent idle with normal
pressure, zero swap churn and no thermal warning. The exact owned build lease was
released. The required repository hook and new independent/hosted review remain
open for this uncommitted delta.

## Protected ASC placement verified after owner deletion

The owner reports deleting the repository secrets. Fresh GitHub metadata confirms
zero repository secrets and zero accessible organization secrets, with all four
standard ASC names retained in `prosody-asc`. The environment still has custom
branch policies with exactly one branch-only `main` rule. This is the owner's
manual action, not an IaC retirement performed by this session. No secret value
was read or changed by the agent. The placement prerequisite is now satisfied;
independent/hosted review, guarded merge and actual Apple account access remain
separate gates. No signing, upload or full RFC acceptance is claimed.

## Live ASC preflight and safe diagnostic follow-up

PR38 landed at 494f76a with reviewed full-tree equality and main reachability.
The first trusted-main self-hosted run42340941986 completed with failure at HTTP
access, after setup, checkout and local credential/JWT processing succeeded.
The fixed diagnostic did not expose the HTTP status; no account-validity, signing
or upload success was established and no blind retry occurred.

This follow-up exposes only an integer HTTP status in100..599, separately from
fixed connection/invalid-response diagnostics. It never renders remote reason,
URL, headers or body and adds no retry, endpoint or credential scope. HTTP status
alone does not establish the root cause or which credential/account setting is
wrong. Source checks/review and any subsequent admitted live run remain separate.

Validation of the diagnostic delta: sixteen named hermetic roots PASS, zero
failures/skips. The original-source overlay runs the three new roots and fails
thirteen subtest assertions. This proves the new distinctions, not Apple access.
Independent source review clears code/tests; its stale-guide finding was fixed.
No live request was repeated. Required hook and hosted review remain separate.

## Exact ASC read-request scope correction

The owner confirms the API key has Admin role. The latest actual read-only run
42348943865 returned HTTP403. Apple documents matching scoped JWTs against the
request path and query (ignoring only limit/cursor/sort). The prior token scoped
only GET/v1/apps while the request also supplied a bundle filter and fields.
This correction binds both token and request to the same frozen filtered path,
retaining the GET-only scope and all credential/transport protections. It does
not remove scope, retry automatically, change credentials or prove the cause of
the observed403 until a separately admitted live check succeeds.

Seventeen named hermetic roots PASS, zero failures/skips. The two scope-specific
roots fail against the original source (two failures), including the actual
synthetic OpenSSL JWT payload and captured request comparison. This is local
request-construction evidence; no new Apple request has occurred.

## Current-preview TestFlight upload lane

The owner explicitly authorizes uploading current built functionality without
waiting for full RFC/backend completion. PR42 landed901aeb1 and actual trusted-main
run42350764030 PASS: the configured key can read the existing app with exact scope.
New distribution identity and matching profile were supplied by the owner; actual
archive signing remains untested. The new manual main-only self-hosted workflow
prepares bounded private archive/export evidence and optional explicit upload.
No hosted Mac fallback, keychain defaults/search-list/ACL change or automatic
retry is introduced. Full backend/auth/device acceptance remains separate.

Current-preview release controls: twenty named hermetic roots PASS0fail0skip;
initial14-root run had two macOS observation errors, corrected before final20.
Independent final source review is clear. The workflow retains crash-safe UNKNOWN
receipts before upload, bounded private logs/child cleanup and exact lease
release (or explicit retention if cleanup is uncertain). No real archive, signing
or upload has executed in this lane yet; hosted/native gates remain separate.
See [preview delivery](TESTFLIGHT-PREVIEW.md) for operator/recovery scope.

## Preview archive provisioning correction

Actual run42352763746 compiled Flutter release successfully, then failed native
archive validation because the command-line app provisioning profile also
applied to dependency/resource-bundle targets. Receipt records version1.0.1,
build41, upload NOT ATTEMPTED. The owned build lease is absent after cleanup;
no authentication failure or Apple upload occurred.

The archive now passes a custom PROSODY_APP_PROFILE setting used only by the
Runner Release target; it no longer globally sets PROVISIONING_PROFILE_SPECIFIER.
Export retains explicit manual app/profile mapping. Empty custom setting preserves
the ordinary unselected profile behavior. Runtime Dart code and signing
credentials/defaults/search lists/ACLs are unchanged. Source controls and actual
next native archive remain separate evidence.

Corrected source:22 named release-control roots PASS0fail0skip. The parsed
project guard fails against the original project (one assertion failure); the
prior real native archive failure is separately retained. No new native build or
upload occurred during these tests. Independent source review is clear; required
hook and new-head hosted review remain separate.

## Locked signing keychain admission guard

Actual corrected run42353827442 compiled Flutter release and reached CodeSign,
then failed errSecInternalComponent against the new prosody-ci keychain. The
read-only native status query confirmed it was locked. Receipt1.0.1/build41 says
upload NOT ATTEMPTED; owned job lease was released. Authentication retries remain
STOPPED pending owner remediation; no certificate/profile replacement is implied.

The release script now reads lock metadata before ASC credentials/capacity/lease
and immediately before archive/export signing. Locked or unknown status refuses
without unlock, password entry, ACL/default/search-list change or retry. Unlocked
is only a point-in-time prerequisite, not key-use authorization or proof that
signing will work; later lock/access failures still stop. No live signing retry
is authorized by this source correction.

Validation:25 named hermetic release-control roots PASS0fail0skip. Omitting only
the early lock guard yields one intended assertion failure before credentials;
all keychain FFI is mocked in these tests. Independent source review is clear;
its CFRelease void-signature cleanup is incorporated. No native keychain query,
unlock, archive, signing or upload was repeated during this correction.

## Owner-configured explicit CI keychain unlock

The owner reports the GUI menu says Lock Keychain, while read-only native status
still reports locked; the exact cause is unproven. The owner supplied a new
PROSODY_KEYCHAIN_PASSWORD secret in the protected prosody-asc environment.
Metadata confirms its presence, not its value or correctness. This is an explicit
credential/remediation event, not permission to guess passwords or change ACLs.

The runner removes all five secrets from its environment before subprocesses.
If metadata reports locked, its initial admission may make ONE native unlock
call with the supplied password in process memory, never argv/logs. Failure or
unconfirmed status stops authentication attempts. Already-unlocked status does
not test the password; subsequent archive/export rechecks remain read-only and
never repeat unlock. Defaults, search list, private-key access and lock settings
remain unchanged. No live unlock/signing/upload has run from this source yet.

Validation:29 named hermetic release-control roots PASS0fail0skip, including
UTF-8 byte length, one unlock call, failed/unconfirmed stop, no password test when
already unlocked and removal of all five secrets before child processes. Native
FFI is mocked; no real credential value was read or used by these tests.
Independent source review is clear, with credential-validation wording corrected.
Required hook, hosted review and actual runner unlock/signing remain separate.

## Runner GUI-session source preparation

Established isolated lane fix/runner-gui-security-session at main78ffae1. Prepare
an opt-in protected Pulumi resource and exact-owner session-only reconciler,
retaining original plist and partial-phase custody, idle/drift checks, bounded
restart and no automatic rollback. Fifteen hermetic roots pass; no launchd,
credential, keychain, signing or upload effect performed. Independent source
review is clear; required repository/nested-module checks pending. SessionCreate is a
candidate explanation for Terminal-success/service-failure, not proven cause.
Actual TestFlight and full RFC acceptance remain open; preserve hourly reminder.

Review corrections: controlled termination unwinds child cleanup; unresolved
prior phases refuse overwrite/retry; new resource runs /usr/bin/python3 -I.
15 new roots and 22 legacy roots PASS0skip; no effect execution. The interruption
regression uses a mocked child, not actual OS-signal execution. Actual read-only
host plan succeeded. Required hook/nested module remain pending.

Bounded nested Go compile TIMEOUT at240s before package/test results; subsequent
nested vet/lint and root hook/commit NOT REACHED. Own build lease0d91c796... was
released after process cleanup. Preserve staged source/private patch; no safe
commit/draft PR or live effect yet, no automatic compile/signing retry.

PR47 first hosted review completed with coverage100 but concrete retry-state and
concurrent-apply findings, not merge clearance. Fixed by removing pre-admission
receipt and adding a nonblocking exclusive control lock held through completion.
18 current session roots PASS0skip; old-source busy-then-retry control RED1 failed assertion.
Earlier nested compile/vet/lint0 and root hook/all12Go passed the first committed
head. New delta final review/hook/hosted review remain pending, no live effects.

## Runner stopped-phase recovery

PR47 merged/landed verified. Actual saved-plan apply failed at retained stopped
phase before changing the original plist; runner offline, no keychain/signing
change, exact owned lease released. Preserve that receipt and backup, no blind
apply/signing retry. New isolated source adds finite shutdown waiting and explicit
resume of only this inspected phase with exact source/registration/backup checks.
29 hermetic roots PASS0skip; native recovery and independent/required gates remain
separate pending acceptance. Preserve full-RFC hourly reminder.

## Actual runner-job signing probe

PR48 merged/landed verified; reviewed recovery apply succeeded with old controller
SAME, runner online/idle, SessionCreate false, terminal phase signing-unqualified.
No signing/keychain-policy mutation and owned lease released. Prepare one actual
protected manual-main Actions probe before a full archive: bounded disposable
binary, exact selected identity/keychain, existing native admission/lease and
one initial configured unlock rule, no signing retry. Probe-only explicitly
refuses upload and returns before SDK/Flutter/archive/upload. Eight hermetic
probe roots plus29existing release roots PASS0skip; actual runner probe not run,
source review clear, required/hosted gates pending, full RFC reminder remains.

## Resume existing verified archive

Actual runner probe passed; full42423277966 produced signed/verified archive but
stopped on read-only locked status before export, uploadNOTATTEMPTED/lease released.
Preserve archive/receipt; no full rebuild or blind upload retry. Prepare explicit
owned-cache/frozen-CDHash resume with trusted-main ancestor/unchanged mobile input,
current Apple selection and fresh signature/metadata checks. NEW job retains one
initial native unlock, no additional in-job unlock or keychain-policy change.
11resume+9probe rootsPASS0skip;29release rootsPASS after reviewed-shape expectation
update; initial1shape failure retained. Source review/required/hosted/native gates
separate; full RFC/reminder unchanged.

Review correction binds every continuation upload durably to the original archive
under a retained exclusive lock. UNKNOWN+stable attempt pointer precede transport;
ACCEPTED/UNKNOWN origins never replay on stale Apple inventory. 15resume rootsPASS;
journal-persistence omissionRED1assertion/controlPASS1. Initial acceptance display
expectation failure retained/corrected, no remote acceptance inferred. Final
source review/required/hosted/native gates remain separate.

## Apple upload accepted; tester availability pending

Actualrun42426530316 SUCCESS23:45:56Z, version1.0.1/build41. Verified signed-archive
resume exported successfully; Apple validation and upload both succeeded. Original
and continuation journals both ACCEPTED; owned lease absent after release. No
rebuild or duplicate upload selected. Processing/tester visibility remains pending
human confirmation because browser access is unavailable. Full RFC/backend/native
functional acceptance remain open; preserve reminder and all upload evidence.

### 2026-11-07 — Owner-selected monochrome branding

- Replaced blue-accent theme with the supplied Ink/Paper/Fog/Graphite palette,
  explicit neutral light/dark surfaces and accessible secondary text. Reserved
  orange Signal for unread indicators; errors retain semantic red.
- Replaced in-app mark with the exact supplied SVG path and generated native
  launcher artwork, including iOS dark/tinted appearances and Android densities.
  Retained vector source and reproducible generator; typography is unchanged.
- Flutter analysis: no issues. Two light/dark rendered contrast checks PASS;
  full Flutter suite: 211 named PASS, zero failure/skip, done success=true.
  Static icon audit: 21 iOS catalog entries match declared square dimensions,
  RGB/no alpha. Earlier initial analysis failure (preview banner's removed blue
  constant) and short-canvas render failures were corrected before these runs.
- Required hook/independent/hosted review are separate delivery gates. This is
  source verification, not a new signed build or device/TestFlight acceptance.
  Already accepted build 41 is unchanged. Full RFC/reminder remain incomplete.

### 2026-11-07 — Activity timestamp validation

The next independent client task rejects invalid `acceptedAt` calendar, clock and
UTC-offset fields instead of letting Dart normalize them. The source response
field is Go `time.Time` JSON (`internal/store/activity.go`); valid RFC 3339 forms
with explicit timezone remain supported. Leap days, numeric offsets and
nanosecond-form fractions are accepted with Dart's existing microsecond precision.
No authority, cursor or service-currentness meaning is inferred from the date.

Original source: 16 malformed-case failures, six valid positive controls PASS,
and two already-rejected malformed controls PASS (24 named total, zero skips).
Corrected analysis: no issues. Focused activity model/widget checks: 42 named
PASS; full Flutter: 235 named PASS, zero failures/skips, done success=true.
Independent two-file source/test review CLEAR; final docs/hook/hosted review are
separate gates. No DB/native/device/provider or new upload acceptance is claimed.
Accepted build 41 is unchanged; full RFC and hourly reminder remain incomplete.

### 2026-11-07 — Cancellation-aware MP3 preflight

MP3 validation now checks cancellation before validation, during each frame-header
iteration, after scanning and before returning decoded timing. Decoder setup uses
context-aware non-seeking compressed input, avoiding its unused eager seek index;
PCM reads remain context-aware. This does not preempt computation inside a decoder
read or establish a CPU-time/service-latency bound. Cancellation returns no partial
measurement, not remote cancellation, absence, settlement or retry permission.
Nil contexts remain unsupported; caller-owned bytes must remain unchanged.

Original public regression: three malformed/deadline leaves plus their parent
FAIL, while three controls PASS (including already-cancelled valid audio).
Initial corrected focused and scoped-race checks: nine records/three roots PASS;
ordinary media package: 172 records/36 roots PASS, zero failures/skips. Added a
scan-only cancellation regression: final focused ten records/four roots PASS;
compiled omission of only the per-frame check fails exactly that regression while
its structural control passes. The direct compressed-reader test is an adapter
control, not a public-path decoder-setup acceptance test.

A broader media race invocation timed out at its 90-second package deadline in
the existing ten-minute narration fixture, after 45 passing records; it is NOT
package PASS. Its owned lease was released. A fresh admitted bounded followup
uses the unchanged assertions and a 300-second package/360-second supervisor
window; its final result is recorded separately. No fixture, provider, signing,
upload, deployment, AMOS pin or authority change is selected. Full RFC remains
incomplete and the hourly reminder remains.

The explicitly re-admitted longer race run completed PASS in231.247s: all173
named records across37roots passed, zero failures/skips, package exit0. It used
unchanged assertions under the300s package/360s supervisor budget. The prior90s
timeout remains historical failure, not a passing run. Owned Go process group
absence and exact lease release were confirmed. No broader provider/native or
wall-clock latency qualification follows from this local race result.

### 2026-11-07 — Private ready-notification payload codec

Added a pure notification payload encoder for RFC §4.4: fixed generic alert plus
versioned feed/item/rendition routing hints, never title/source/script/agent text,
URLs or delivery tokens. IDs are provider-visible metadata and confer no current
access. Feed UUIDv7 and canonical nonzero item/rendition shapes are checked; no
new v7 requirement on stored item/rendition IDs. No notification is sent or
scheduled, and OS permission/device enrollment/native receipt/dedup/tap admission
remain separate missing composition. See [NOTIFICATIONS.md](NOTIFICATIONS.md).

Ordinary and race: each39 named records/four roots PASS0fail0skip (overlap).
Compiled validation omission fails private-text and two feed negatives plus parent,
with one golden control PASS. Initial supervisor count assertion FAILED expecting
only two records; an exact-event audit confirms all four intended failures and
one control, no rerun or erasure of the failed harness. Required hook/review and
actual APNs/native/device acceptance remain separate. No keys, provider requests,
new signing/upload, deployment, spend or AMOS pin changes. Full RFC/reminder stay.
