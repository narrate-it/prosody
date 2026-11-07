# RFC 0001 implementation record

Status: started, incomplete. Branch: `feat/prosody-rfc-0001`.

## Scope and acceptance

Implement the full RFC: AMOS-hosted Go backend, durable authorized submissions, narrated and supplied media, BYOK models/voices, Flutter iOS client, personal/team progress, notifications, content lifecycle, and Nanos VM packaging. Completion requires actual integrated checks, independent review, and separate evidence for provider/device/Nanos acceptance. Deployment and spending are not authorized by source implementation alone.

## Current evidence

- AMOS integration-gap reconciliation (owner, exact `809e060e`): `EffectDefinitions` only binds codecs/transactional enqueue. `jobs.Worker` provides consumer Execute/Reconcile and lease/fence-protected queue resolution; ProductionHost only wires identity-email execution. No supported public application-consumer seam provides current person/membership/financial reauthorization or fenced external publication. The durableworker interop fixture is not that boundary. AMOS must design/review/qualify it; no competing scheduler or private-authority workaround is being added.
- Human mobile authentication is separately blocked: native issuer public code+S256 PKCE, opaque <=15-minute access token, POST current/revoke form token+client_id, no refresh/OIDC ID token, and registered numeric-loopback redirects only. Removed the speculative AppAuth/OIDC/refresh implementation and custom-scheme registrations/dependencies. Production Flutter composition now has no InteractiveSession provider and visibly disables sign-in. This is explicit missing capability, not a stub identity or a working app. Existing browser/PG evidence is not mobile qualification; bearer consumption APIs also remain unqualified.

- Latest exact-head test update (owner-reported): on MacBook, selected Go tests in `app/operation`, `internal/mcpgen`, `examples/reference/mcp` and `tests/interop/mcp` passed at `809e060e64c00160d8751ad8a35b3d76e2436b45`; JS tool-inventory unit test also passed. First combined run timed out after the first three packages; remaining package passed separately. Both owned leases released. This supersedes the earlier unrun-head statement only for these selected unit/fixture tests. No real MCP client, database replay, CIMD, lost-response, provider or release acceptance is established. Full service fixture and OAuth expiry/revocation fixes remain open, as do existing Prosody integration gates. Receipts have not been independently inspected here.

- Earlier source checkpoint (owner-reported; selected test status superseded above): isolated head `809e060e` aligns descriptor-bound JSON keys with the accepted nonblank valid UTF-8 / 1–256 decoded-byte contract, retains exact values and matches any parsed header; legacy header behavior is unchanged. Independent source review found no blocking issue in that delta; code/tests have not run on this head. Earlier source exclusions below are superseded. Domain persistence remains unqualified: accepted NUL-containing keys may fail if full inputs are stored as PostgreSQL JSONB/text. Prosody must qualify lossless source/digest/receipt/job/audit storage and real transaction/restart replay, not infer complete compatibility from descriptor validation. The local escaped-lone-surrogate validation gap remains open.

- Earlier key-validation mismatch (owner-reported; source exclusions superseded by `809e060e`): AMOS additionally rejects leading/trailing whitespace and NUL/CR/LF, while Prosody's stated contract is nonblank valid UTF-8 of 1–256 decoded bytes. No narrower restriction has been accepted. Preserve exact decoded keys; alignment must not depend on copying JSON keys into HTTP headers. AMOS invalid-UTF-8/lone-surrogate validation source is under independent review; no tests or qualification claimed. Local Prosody raw UTF-8 checks and byte-boundary tests do not establish strict rejection of escaped lone surrogates by its JSON decoder; that remains a local validation gap to address and test.

- Earlier AMOS candidate (owner-reported): `4eaea3a37c29978c5445b29f849d22d55ef85424` includes ordinary SDK body-key, replay/conflict and controlled lost-response retry fixture source. Independent source review closed unsafe uncertain-retry and malformed-result findings; this does not establish execution success. Final head has not been compiled or run; client/database/CIMD/lost-response qualification remains pending. Consent-selected workspace and `/feedId` descriptor binding are confirmed source semantics according to the owner, not independently inspected here. Existing authority, transaction, team-workspace and operating gates remain open.

- Confirmed MCP integration contract: consent-selected `active_principal` only, no `explicit_member` expansion; canonical UUIDv7 feed via `/feedId` with current destination authorization. Tool JSON key is authoritative; mismatched HTTP header rejects. RFC's stale header-only description is superseded. AMOS worker is adding ordinary real Go SDK lost-response retry qualification source; service execution awaits an owned fixture and capacity admission. No new revision or executed qualification receipt is claimed. Team/account authority and other existing gates remain open.

- Earlier AMOS checkpoint (owner-reported): isolated unmerged candidate was `6ee04091e96be4bf710d7139dcc9609a9b668bfd`. Per-call body key and optional `202` binding are implemented with focused source reviews, but exact-head service/CIMD tests are unrun. Expiry-after-wait and native grant revocation transactional gaps require held-authority finality and dedicated scoped journal/audit design; no unsafe workaround applied. Local heavy-capacity admission refused on CPU idle, and the owned verified-TLS test cluster is not yet provisioned. No deployment/spend. Details/next action reportedly reside in the private AMOS salvage ledger, not inspected here. Execution constraints do not close qualification gates or authorize alternate infrastructure.

- Additional native-grant findings (owner-reported): current grant management always selects personal workspace, and retained history over its bound can refuse all revoke operations. AMOS owns fixes in progress; account authorization and revocation remain unqualified on the current candidate. Team-workspace support is explicitly gated on qualified consent/grant selection, current authority, cross-workspace isolation and revocation under over-bound retained history. Accepted RFC contract alignment does not establish team runtime support or personal-account readiness.

- Earlier AMOS candidate (owner-reported): `6ee04091e96be4bf710d7139dcc9609a9b668bfd` supersedes `4ab3c94a4e81de03868f664dcb9699630a7cc4b6`. Optional descriptor-bound `SuccessKind` permits binding Prosody HTTP `202` with `ResultAccepted`. Focused independent source review found no blocking issue in that delta; tests and real transaction behavior remain unqualified. OAuth expiry across waits remains open: the owner reports that a safe fix requires held same-transaction identity/workspace/financial receipts. No release/deployment; source and receipts have not been inspected here. Prior-head evidence does not qualify this head.

- Additional AMOS gates (owner-reported independent-review findings): issuance/refresh can wait across native/grant expiry without a final time check after all blocking writes; OpenAPI `202` is not bound/enforced by the operation descriptor. Fixes are being inspected, not qualified. Expired OAuth basis/grant/token capacity reclamation is a separate operating-acceptance gap. RFC records all three with required evidence. Candidate `4ab3c94a4e81de03868f664dcb9699630a7cc4b6` remains unready; prior contract alignment and unit-test passes do not close these gates.

- Earlier AMOS candidate (owner-reported): `4ab3c94a4e81de03868f664dcb9699630a7cc4b6`, superseding `5a1a1e8f`, adds opt-in real Go SDK body-key/replay fixture source and independently reviewed fixture controls. Focused unit tests reportedly pass; real database/client test is unrun. No merge or deployment. Native consent/current authority and same-transaction durable-effect admission remain qualification gates. Source/receipts have not been inspected here; fixture-control review is not full integration review or successful execution evidence.

- Key contract correction: decoded idempotency keys are nonblank valid UTF-8, 1–256 bytes, not 128 characters. Clients generate globally unique keys per logical submission across feeds; retries retain the key. Changed feed/body under the same key conflicts within AMOS replay scope. Latest AMOS evidence reaches executor and HTTP denial paths; successful MCP/database acceptance is still absent.
- Byte-limit regression tests failed on the old validator (256-byte ASCII rejection and multibyte overflow acceptance), then all content-package tests passed after the correction. Tests cover ASCII/multibyte boundaries through validation and JSON decoding, empty keys and invalid UTF-8. Bounded local run used `GOMAXPROCS=2`, `-p 1`, and the authorized internal cache; sampled CPU idle was 47.27%/40.58%, swap deltas zero, pressure level 1, no thermal/performance warning. Build lease `8b25ed22b3887ff86780cd802c033bc923670c22` released after tests. These are isolated domain checks, not AMOS acceptance.

- Earlier AMOS candidate (owner-reported): isolated commit `5a1a1e8f` combines main `5fc6ec27`, MCP candidate `3377b54a`, and an optional descriptor-bound body idempotency field. Focused operation tests pass according to the owner; independent review is underway. `ProductionMCPConfig` remains the entry point. Not merged or client/database qualified; source and receipts have not yet been inspected here. This supersedes the staged-index/worker-in-progress update below. RFC compatibility is confirmed for actor/workspace/operation replay scope with installation/application/environment isolation: feed stays in the canonical payload, and distinct cross-feed submissions use distinct keys. Cross-transport qualification remains open.

- Replay-scope decision: Prosody accepts AMOS's owner-reported installation/application/environment/workspace/person/operation scope, excluding feed. A new logical submission needs a distinct key across feeds within that scope; retries retain the original key. `feedId` remains in the canonical payload, so reusing a key for another feed conflicts. This supersedes the RFC's earlier feed-scoped identity. Reviewed per-tool JSON binding and cross-transport replay/conflict qualification remain outstanding; no runtime support is inferred from this decision.

- AMOS successor update (owner-reported): isolated worktree at verified main `5fc6ec27`, with candidate `3377b54a` integrated into the index for review; original branch and claims preserved. Candidate host composition is `ProductionMCPConfig` with registry, OpenAPI/mapping, permissions, resource/effect definitions, migrations and explicit `EnableCIMD`. A bounded AMOS worker owns generic per-tool input idempotency investigation/implementation. No supported API, final-head receipt, landed merge or deployed readiness is established. Earlier `7ed0c2a3` / `service-a25` 21-check CIMD evidence does not qualify this successor or Prosody. Host wiring awaits exact successor source and the supported idempotency contract; other Prosody work remains independent.

- Candidate seam update: AMOS reports `active_principal` workspace selection from consent, canonical UUIDv7 resource IDs, pointer-shaped `/feedId`, and header-based MCP idempotency. Prosody accepts consent-selected workspace and UUIDv7 feeds. Tool-input idempotency mapping, exact descriptor schema and final-head compatibility remain unqualified. Existing `content.TextSubmission` is not an MCP wire contract; future transport adapters must resolve trusted workspace context and enforce canonical feed IDs without trusting a body selector.

- AMOS owner acknowledged responsibility for reconciliation and implementation, including the integrated operation seam, candidate continuation amendment, final-head qualification and retained claims. Prosody remains owned here; no overlapping AMOS edits started.
- Playback domain and tests now pass: coverage merging, seek-versus-listen semantics, revision conflicts, foreground exposure for visual-only segments, and independent value state. Persistence/current-authority binding remains outstanding.
- Narrate pinned to `v1.0.1-0.20260926182619-c43bf2e0c0b3`. Briefing rewrite composes canonical craft with a condensation policy and records the instruction digest. Tests pass for source isolation and no unadmitted repair. Actual synthesis/duration verification and human listening acceptance remain outstanding.

- Existing RFC and approved design artifacts preserved unchanged except prior confirmed amendments.
- Initial content domain source enforces strict inline-text decoding, canonical payload digests, bounded mixed manifests, and the aggregate generated-narration ceiling.
- Initial key-vault source uses AES-256-GCM with installation/workspace/credential/revision binding. Host key custody remains to be composed.
- Focused Go tests pass for content, key-vault and OpenRouter packages. The provider tests use an HTTP test server; no live provider acceptance is claimed. Content checks include strict field casing, duplicate fields, invalid UTF-8, payload bounds, canonical digests and aggregate duration. Provider checks cover source/instruction separation, truncation rejection, no redirects/retries and redacted errors. No working service or mobile implementation is claimed.
- Local AMOS is `5fc6ec274d97d978967afa08a8a225992622cfe0`. GitHub lookup of `docs/contracts/mcp-endpoint-source.md` returned 404 on the default branch. This does not disprove the reported continuation amendment on the unmerged candidate. Exact candidate source/receipts still need reconciliation.
- Owner explicitly approved internal disk for this task. Tests use the ignored `.local/gocache`, `GOMAXPROCS=2`, and package parallelism 1. On this four-core MBP, sampled interval CPU idle was 37.27% and 64.50%, pressure level 1, zero sampled swap churn, and no thermal/performance warning. The build lease was acquired for the tests. Neither `wazi` nor `wazi-contract` is on PATH. No formal-plan conformance claimed.
- Owner authorized AMOS implementation coordination. Queued message `01a190e1-6f6d-7c52-8b39-7b992a6b503d` to active AMOS thread `01a19071-59c2-7c51-91c4-75368427f977` using SSH and `codex queue --resume`. Requested exact candidate contracts/receipts and implementation of missing support, preserving the continuation-amendment correction. Offered this MBP for a bounded headless task while AMOS retains integration ownership. Owner responses are recorded above; queue acceptance is not task completion.

## Outstanding delivery coverage

1. Pin and qualify AMOS extension, current-caller authority, CIMD descriptors and durable effects against actual source.
2. Persist feeds, immutable sources, deduplication receipts, jobs, renditions and per-person progress with tenant isolation and revocation fences.
3. Integrate Narrate craft and OpenRouter catalog, BYOK requests, voice previews, bounded synthesis, measured duration and cost reconciliation.
4. Implement private uploads, detected media validation, ordered playback manifests, signed delivery, retention and deletion.
5. Implement Flutter authentication, feed switching, vertical playback, waveform/reduced motion, background audio, source details, setup/settings and recovery states.
6. Implement team administration through AMOS and per-person notification preferences/durable delivery.
7. Package the Go application as a Nanos image, repository-owned deployment IaC, recovery and operational checks.
8. Run cross-component acceptance, independent code review, fixes, and qualified delivery. Never substitute generated mockups or isolated tests for live integration evidence.

This record is a progress inventory, not a validated execution graph or an external lifecycle enrollment.

## Active implementation continuation

- Hourly reminder command smoke-check succeeded: queued message `01a193b9-1a2b-7f11-b0de-a5cc572e82fb` to the existing Prosody thread with `--resume`. The tagged minute-zero crontab entry is installed. This verifies queue command acceptance, not an observed future cron trigger or machine wake from sleep.
- Installed the user-requested hourly crontab reminder, tagged `prosody-rfc-hourly-refocus`, queuing the existing session with `--resume`; no competing worker. Completion sentinel prevents further queues, and the owned entry must be removed when done.
- Added strict Unicode JSON validation, byte-preserving PostgreSQL schema and caller-owned transaction functions for submission receipts and per-person progress. Added an explicitly configured disposable-PostgreSQL test for NUL/Unicode round trips, same-key replay/conflict and rollback. These new checks have not run yet.
- Local capacity admission refused heavy work: sampled idle 4.20%/3.93%, pressure 1, zero swap deltas, no thermal warning. Lease `f1608db0c79d37a4526c90b68678065de7c4edb7` released; no automatic capacity retry or fallback host. Source/dependency work continues.
- Flutter source implementation started: authenticated bounded HTTP transport, native session adapter, shared playback handler, audio/video presentation, feed switching, original/script view and settings. Platform/provider behavior remains unqualified; no fake feed is substituted for unavailable service data.

- Exact AMOS source inspection now confirms `ProductionMCPConfig`, `app_` table allowlists, UUIDv7 resource selection and restricted callback `DBTX`. Adapted the draft SQL names to `public.app_prosody_*` and a transaction-only callback adapter; no AMOS owner paths changed.
- Independent bounded source review found explicit-empty-title acceptance, unconstrained reused work IDs, incomplete durability/concurrency tests and an ineffective UUID case test. Corrected validation, bound each domain receipt to a unique AMOS-returned work ID via a same-transaction `StageWork` callback, and added committed/reconnected replay, competing-key and rollback tests. These fixes/tests have not executed under the current capacity hold.
- Added private original-byte file adapter (development/qualified-volume only), image bounds/full-decode validation, MP3 frame completeness/sample-count duration checks, asset/rendition/settings/preference schema, publication checks and transactional deletion/notification hooks. Nanos external object storage and AMOS authority/job fences remain separate required composition.
- Flutter dependencies resolved and Dart sources formatted after correcting syntax findings. Generated iOS/Android project files; declared background playback and redirect configuration. Release signing remains unconfigured. No analyzer/build/simulator acceptance yet.

- Independent adapter source review found AMOS rejects `/api` as a reserved operation root. Corrected the proposed service/mobile route prefix to `/prosody/v1`; no reserved-root bypass. Also forwarded the concrete input codec schema document required by descriptor-bound body-key validation. These are source fixes, not executed contract acceptance.
- The adapter's operation ID is now `prosody.messages.submit-text` (hyphen), because exact AMOS source rejects underscores in operation IDs. MCP tool name remains `prosody_submit_text`. RFC and source use the corrected proposed identity; no deployed identity or existing replay records were migrated.
- Mobile source review found and prompted fixes for coverage-baseline initialization, final decoder coverage, cumulative lock-screen timeline, bounded/surfaced progress-sync failure, generation fencing around session activation, image gallery/exposure and queue prefetch. These remediations require analyzer/tests and device checks; no passing claim.

- Added provider configuration validation and encrypted settings persistence with optimistic revisions and secret-free snapshots. Added actual stored-rendition duration lookup for progress, plus database test source for person isolation, stale revisions, deletion and distinct-submission work-ID reuse. Still unexecuted.
- Lightweight source checks completed: Go formatting clean, Dart formatting parsed all current library source, iOS plist lint passed, Android XML and descriptor JSON parsed, reminder Python syntax parsed. These do not establish compilation or runtime behavior.
- MP4 probe was explicitly renamed to return unverified metadata after source review identified missing sample-map/codec/timeline qualification. It must not be used as publication admission or authoritative measured duration.
- The first AMOS adapter source is isolated under `integration/amos`, pinned to exact `809e060e64c00160d8751ad8a35b3d76e2436b45`. A local ignored workspace resolves the read-only candidate copy. No AMOS owner branch or claim changed. Queued concrete questions about application-job execution and native mobile session seams; no alternate authority system implemented.

- Queued integration-gap acknowledgment and minimum worker/mobile acceptance paths to the AMOS owner (`01a193bc-ddcc-7343-ad0e-2f7c1b2caff2`). Source-only corrections passed Dart formatting, dependency resolution and iOS plist lint; no build/service/mobile acceptance was run.

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

- AMOS owner inspected identifier syntax on `809e060e`: operation `prosody.messages.submit-text`, tool `prosody_submit_text`, and path `/prosody/v1/messages/submit-text` match their respective rules; the path is outside reserved roots. This is source inspection only, not complete descriptor registration or runtime acceptance. Separate private AMOS worker/mobile dependency briefs are recorded; proposed designs remain unaccepted. RFC updated accordingly.

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

## Capacity reopened and executed Go/database checks

David reported only one other Codex session running. Fresh admission passed:
interval CPU idle 46.17%/47.31%, pressure normal, swap churn zero, no thermal or
performance warning. `GOMAXPROCS=2 go test -p 1 -timeout 90s ./internal/...`
passed all packages (database-specific tests skipped without their fixture).
Then phase admission passed at 75.79%/84.76% idle with the same healthy pressure/
swap/thermal measurements. `PROSODY_BUILD_ADMITTED=1 scripts/test-postgres.sh`
passed the real PostgreSQL storage tests, including committed replay across a
new connection, concurrent-key admission, rollback, unique work intent binding,
NUL-key persistence, progress isolation/revisions and deletion. Its owned Unix
socket fixture stopped afterward. Both phase leases were released exactly.
These are local domain/database checks, not AMOS transaction or service acceptance.

AMOS adapter phase admission passed at 73.48%/76.37% idle. Initial attempt failed
resolving the unpublished candidate version; added an exact-version replacement
in the ignored local workspace to the independently checked-out pinned source.
Candidate source remains unchanged. Adapter compilation and all constructor/codec tests passed against exact `809e060e64c00160d8751ad8a35b3d76e2436b45` after dependency compilation (about four minutes). This is not real MCP client, AMOS database, CIMD or worker/mobile acceptance. The adapter build lease was released.

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

## Parallel worker integration checks — in progress

Collected all three bounded worker deliveries. Initial new consumption-read
PostgreSQL suite passed after admission (62.93%/64.42% idle, normal pressure,
zero swap churn, no thermal warning). Independent source review found omitted
cursor fields could inherit request values; backend worker is fixing exact field
requirements and adding returned-row deletion fencing/concurrency evidence.
That initial pass does not qualify the pending fix.

Activity screen's five widget tests passed after correcting one missing-braces
lint in the test. Cases cover processing/failure/original text, empty/error,
manual pagination/status updates, cursor cycles and disposal. Mobile phase
admission passed at 77.16%/75.57% idle with healthy pressure/swap/thermal readings.
Activity uses the existing proposed feed-items route; real all-status server
composition is still pending and newly ready items do not refresh the playback
queue automatically. No end-to-end capability claimed.

Parent ran iOS preflight syntax and both modes: preview prerequisites pass;
release correctly exits blocked for sign-in/signing configuration and unqualified
product/account acceptance. No signing, Keychain, account inspection or upload.

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

### 2026-11-01 — Rewind, feed header and bounded live preview

PR2 source fixes rewind/forward to ten seconds using current player position and
segment duration. Header now has the feed dropdown and direct logout; activity,
settings and live creation are in the dropdown. Regression covers track transition
and pending-directory sign-out races.

Debug preview adds user-entered text through GPT-4.1 mini then
`elevenlabs/eleven-v4-turbo`, Laura. Owner authorized $1 total for this session.
Four $0.25 reservations are flushed to a local append-only ledger before paid
calls; no automatic retries or refunds. Prices/voices/modalities are checked
before reservation. Key provisioning uses the authenticated localhost VM tunnel
and RAM only. This is not production AMOS auth or a provider-enforced account cap.
Go catalog support now requests output_modalities=speech and validates speech
modality/voice. Generated media must decode locally and be <=10 minutes.

Verification: 91 Flutter tests passed, then 30 targeted provider/session/preview
checks passed after cleanup. PostgreSQL settings fixtures passed after the first
cold-cache compile timed out (no test result); explicit cache warm then bounded
retry succeeded. Independent peer reviews found no blocking source issue.
Provider inference and current device installation are not yet accepted here.

Device update: signed build passed (51.0s); installation/launch and debug attach
passed (34.0s install). Authenticated debug provisioning confirmed four remaining
attempts; on-device widget tree shows LIVE PREVIEW and FeedScreen. No paid request
was made by this session. Owner identified canned macOS voice and nonfunctional
slider drag: canned clips are now explicitly labelled macOS demo recordings.
Scrubber now tracks local thumb/time, seeks once on release, and rejects stale
track releases. Independent review clear; 12 controls/preview tests passed and
full Flutter analysis clean. Actual physical scrubbing and Laura generation still
need user acceptance; these are not implied by widget tests or installation.
Go lint/full unit tests also passed in the preserved commit-hook check.

### 2026-11-01 — Preserve reserved generation across editor navigation

Continued PR2's unblocked preview lifecycle work. A completed provider result was
previously discarded if the editor had been disposed while generation ran. The
screen now delivers the result to its existing destination callback regardless
of editor lifetime. It only closes itself if still mounted and the current route,
so completion cannot pop another screen. No request retry, budget refund, new
scheduler or production authority was introduced.

Two fake-provider widget tests passed for removed and covered editors, proving
single delivery, retained reservation and unchanged unrelated navigation. Seven
session/ledger tests also passed; Flutter analysis clean. The first widget harness
used completers from Flutter's fake clock across runAsync and timed out; corrected
the test to create real-async completers, then both cases passed. Independent
source review found no blocking issue. No provider calls, deployment, rebuild or
phone hot reload performed for this hourly task. Physical slider/Laura acceptance
and the existing AMOS integration gates remain open; full RFC is not complete.
Capacity admission: idle 49.9/58.70%, normal pressure, swap 0/0.096 MiB/s, no thermal
or performance warning. Scoped build lease held for checks and commit gates.

### 2026-11-01 — Human spoken briefing craft

Owner requested voice-conversation and podcast craft. Audit: Go already composed
Narrate conversational craft; the live Flutter preview only used a basic
condensation prompt. Both now share adapted single-listener guidance from voice
conversation, emotional pacing, concreteness, narrative spine and rapport skills.
Use natural contractions/thought groups, varied rhythm, grounded significance,
and source-appropriate warmth; prohibit fabricated emotion, examples, intimacy,
interview dialogue and unsupported speech tags. Policy revision 2 changes the Go
instruction digest. Supplied audio/video and bundled recordings stay unchanged.
See docs/design/narration-craft.md for rationale and an authored example.

Verification: 17 mocked live-provider tests and Go briefing tests passed;
independent source review found no blocker. Existing accounting counts the full
Go prompt, preview limits/reservations unchanged. No paid generation or expressive
voice acceptance is claimed; Laura output requires listening comparison.
Capacity: idle 79.83/50.78%, normal pressure, swap 0/0.048 MiB/s, no thermal warning.

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

### 2026-11-02 — Invalidate preview results after sign-out

Continued unblocked PR2 lifecycle work. LiveSession captures its key/configuration
revision before the first await and binds each completed result to that revision.
Sign-out/reconfiguration invalidates old requests and results even if the same key
is provisioned again. Local publication checks the result's issuing configuration
before file work, after decoding and after stopping playback; rejected files are
removed. Existing provider work may finish, but its stale result cannot enter the
feed. The reservation is retained, with no refund or automatic retry. This local
preview lifecycle guard is not AMOS authority or provider cancellation.

Independent review found two handoff gaps in the first version (capture after
initialize and re-capture at publication); both were corrected with regressions.
Final 13 session/navigation tests passed and Flutter analysis is clean. No paid
calls, phone installation/hot reload, deployment or full RFC acceptance claimed.
Capacity admitted at 54.88/59.90% idle, normal pressure, zero swap churn and no
thermal/performance warning. Build lease used for checks and commit gates.

### 2026-11-02 — Accurate preview failure/budget messaging

Continued PR2 live-preview acceptance work. Pre-reservation failures no longer
claim an attempt was spent. The session records successful ledger completion and
uses fixed safe messages for unreserved failure, retained reservation, and blocked
uncertain budget state. Provider text and credentials never enter these messages;
no automatic retry/refund or change to provider dispatch was added.

15 session/navigation tests passed, including two new pre/post-reservation failure
cases; Flutter analysis clean; independent source review clear. Commit hook runs
required Go lint/tests unchanged. No paid calls, device update or deployment.
Capacity admitted: 47.14/39.94% idle, normal pressure, zero swap churn, no thermal
warning. Full RFC acceptance remains open; retain owned hourly reminder.

### 2026-11-02 — Physical-iPhone preview preflight

The preview preflight and launch runbook still assumed the historical simulator
workflow. Added read-only --preview-device to check the physical iPhone artifact;
--preview retains its simulator behavior. Neither option builds or launches an app, verifies binary freshness or qualifies
a release. Updated TestFlight guidance with
physical debug receipts, current explicit preview composition and debugger-assisted
relaunch recovery. Keep reservations/signing credentials intact; no automatic
uninstall, budget reset or release-mode fallback.

Four hermetic Python CLI tests passed: device/simulator isolation, partial artifact
rejection, invalid option and release remaining blocked with both artifacts. Local
--preview-device passed source/tool/artifact checks; runtime remains unverified.
Shell syntax and whitespace checks passed. No provider call, device connection,
signing, install or deployment. Commit-gate admission: 28.37/38.99% interval idle,
normal pressure, zero swap churn and no thermal/performance warning.

### 2026-11-02 — Preview deletion removes owned generated media

Connected the existing preview DELETE API to private generated-file cleanup.
Only registered owned file paths can be removed; bundled/unknown IDs never become
filesystem paths. Already missing files are idempotent. Concurrent deletes share
cleanup; metadata/progress disappear only after cleanup succeeds. Failure returns
a fixed safe error and retains item/progress for explicit retry. The budget ledger
is never a cleanup target. This covers the local API/composition, not production
retention or a newly implemented human deletion control (UI wiring remains open).

15 preview/control tests passed, including real temporary-file deletion preserving
the budget file, duplicate pending deletes, and failure/retry. Analysis initially
flagged two style issues; corrected and targeted analysis clean. No paid calls,
phone update, production deletion or deployment. Orphan cleanup across app restarts
is not added: in-memory preview metadata still resets, as documented.
Capacity admitted 25.21/38.31% interval idle, normal pressure, zero swap churn and
no thermal warning. Full RFC acceptance remains open; retain hourly reminder.

### 2026-11-02 — Personal preview deletion control

Prior media-cleanup independent review is now clear. Added a confirmed Delete
briefing action to the Read sheet only when explicitly composed and in a personal
feed. The preview root rechecks personal membership, stops the active clip before
cleanup and rebuilds its queue after success or failure. Pending actions disable
the button; cancel has no effect, safe failure permits explicit retry. Production
main does not supply the capability and team feeds do not expose it. This is not
production owner/admin authorization acceptance or provider-side erasure.

19 focused UI/API tests passed (confirmation/cancel, single flight, failure/retry,
personal/production/team visibility and existing file cleanup/control regressions).
One analyzer style finding was corrected; targeted recheck clean. Independent
source review found no blocker. Actual iPhone deletion/decoder/queue restoration
remains an acceptance check; no phone update, paid calls or deployment performed.
Capacity 48.37/49.10% interval idle, normal pressure, zero swap churn, no thermal
warning. Full RFC remains incomplete; retain owned reminder.

### 2026-11-02 — Composed preview deletion verification

Closed the earlier root-composition test gap with two widget journeys through
ProsodyPreview, its personal Read/delete confirmation, real PreviewApi and actual
temporary MP3/ledger files. A recording player substitutes only the native decoder.
Success stops the selected clip before deletion, removes file/map/item and loads
the remaining queue. Injected cleanup failure preserves media, restores the old
queue, keeps the sheet with a safe error and permits one explicit successful retry.
Both paths preserve the budget ledger byte-for-byte.

Both new tests passed. Full analysis identified one test style issue, corrected;
targeted analysis is clean. No product behavior change, provider call, phone update,
deployment or native-device acceptance claimed. Existing generated recordings are
not automatically purged after restart: ephemeral metadata alone is not evidence
that a published clip is disposable. A recovery/retention design remains open.
Capacity 40.45/48.64% interval idle, normal pressure, zero swap churn and no thermal
warning. Full RFC/AMOS/device/provider acceptance remains incomplete; keep reminder.

### 2026-11-02 — Aggregate PR2 review readiness

At source 83fcd53 against base 6c7e12f, all 110 Flutter tests passed, analysis clean,
and 4 hermetic preflight tests passed. Final aggregate independent source review
found no blocker; required Go lint/full tests already passed at that code commit.
See PR-2-REVIEW.md for exact scope and historical/device/DB evidence limits.
PR2 was still draft with CodeRabbit skipped. Proceeding to review readiness after
publishing this documentation receipt; no merge, deployment or acceptance implied.
No paid calls or phone changes. Full RFC incomplete; retain owned hourly reminder.

### 2026-11-02 — CodeRabbit PR2 findings addressed

Verified and fixed all three findings on 4c41f87: evidence-note spacing, unsupported
MP3 speech selection, and HTTP timeout without transport abort. The MP3 policy is
an explicitly documented exact-ID compatibility list because current catalog data
has no format field; unknown/PCM-only models remain blocked. Timeout now aborts
transport, cleans late responses and cancels timers while retaining unknown billing
and no-retry semantics. Dependency minimum is 1.5.0; locked 1.6.0 unchanged.

21 Flutter provider tests and analysis passed. Go openrouter/configuration/briefing
checks and real PostgreSQL fixtures passed; independent cross-review clear.
See PR-2-REVIEW.md for comment IDs, sources and evidence limits. No paid calls,
phone changes or deployment. Capacity recheck 49.61/50.83% interval idle, normal
pressure, zero swap churn and no thermal warning. Full RFC remains incomplete.

### 2026-11-02 — Corrected-head native compilation

CodeRabbit's completed check on dd7df26 succeeded; its latest review summary says
no actionable comments. Verified an unsigned physical-iPhone debug preview build
at that exact code revision: Flutter exit 0, Xcode 69.8s, total 98.7s. Artifact is
arm64/iphoneos with expected bundle ID; working tree stayed clean. Receipt is in
PR-2-REVIEW.md. No signing/install, phone update, provider call or deployment.
Local build output is unsigned and must not be treated as install-ready.
Actual device/provider/AMOS/Nanos/full RFC acceptance remains open; keep reminder.

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

Earlier capacity holds and unexecuted-test notes above are historical. Subsequent
source/helper work landed with recorded local verification receipts: PR3
`b93bf7f` (resume regression), PR8 `9b5a922` (verified ranges), PR9 `63173b7`
(local pipeline), PR10 `c4d7762` (presentation), PR11 `1a9c716` (inactive producer
codec), and PR12 `cb092b4` (consumption read catalog). These are source/helper and
check results, not native integration or delivery acceptance. PR7 `f596a1b`
remains draft; its database tests were SKIPPED without the required fixture.

PR2 `fb2f52d` passed seven merged resume Flutter tests, lint and the full ten-package
Go unit gate. Its hosted check succeeded, but the concrete receipt-spacing finding
still required this documentation-only correction and refreshed hosted review.
The producer label-source choice remains pending; neither alternative is activated.
Native B/C integration, source bridge, provider/device, Nanos and TestFlight
acceptance remain open. No deployment, spend or signing in this continuation.
The full RFC remains incomplete; retain the owned reminder.


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
