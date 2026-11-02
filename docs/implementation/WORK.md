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
review clear at83fcd53/base6c7e12f. Recorded exact evidence/limits in PR-2-REVIEW.md.
Next: ready PR for actual CodeRabbit review (draft had been skipped), address concrete
findings and preserve separate native/provider/AMOS/full RFC acceptance gates.

2026-11-02 hourly: CodeRabbit completed with three findings, all verified/fixed.
MP3 selection uses reviewed exact IDs plus current catalog/voice; HTTP timeout
aborts transport and cleans late responses; evidence spacing corrected.
21 provider tests, analysis, Go checks and real PostgreSQL fixtures passed;
independent cross-review clear. No paid calls/device changes/deployment. Push
for review of the new head; do not infer approval or full acceptance from prior
successful CodeRabbit check.
