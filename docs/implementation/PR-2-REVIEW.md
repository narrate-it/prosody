# PR 2 review readiness — 2026-11-02

Code reviewed: `83fcd53a76dfc89529937f0f0928521423d5799a` against base
`6c7e12f4a5cfdc799c5e1402b86731878ccd28d9`. This receipt is documentation-only;
it does not change the code that was tested or authorize merge/deployment.

## Current verification

- All 110 Flutter tests passed with one test worker; Flutter analysis clean.
- Four hermetic iOS preflight CLI tests passed. Artifact checks never qualify
  runtime behavior or TestFlight.
- Required root Go lint and full unit tests passed at the reviewed code commit;
  cached test results reflect unchanged inputs. The documentation commit retains
  the same required hook checks.
- Prior real PostgreSQL settings fixtures passed after the initial cold-cache
  compile timeout and explicit cache warm. That receipt is historical, not a new
  database acceptance run on this wake.
- Incremental independent reviews and final aggregate source review reported no
  remaining concrete blocker. The final review checked native migration, unchanged
  signing identity/bundle, preview/production separation, RAM-only key provisioning,
  local budget/configuration fences, file cleanup and playback-control composition.

Current capacity admission: interval idle 50.31/53.22%, normal memory pressure,
zero swap churn, no thermal/performance warning. Checks used the owned build lease.
No provider calls, signing/install, device update or deployment on this wake.

## Evidence limits

Earlier signed physical-iPhone debug launches, hot reload and key provisioning
were observed on preceding revisions. They do not qualify the accumulated final
head's native decoder, gestures, interruption/background behavior or Home Screen
relaunch. Current widget tests use fake players where stated.

The four-attempt preview allowance is local-install scoped, with conservative
catalog price checks; it is not an OpenRouter account billing cap. Existing
recordings remain unchanged by script-prompt updates. Actual Laura output and
narration quality require listening acceptance.

Production mobile login, AMOS current authority/worker/publication, agent-to-listen
service integration, production retention, Nanos deployment and TestFlight
acceptance remain gated. No production fallback identity or competing scheduler
was introduced. CodeRabbit had skipped the draft PR; readiness requests an actual
review, not approval or merge. Full RFC work remains incomplete.

## CodeRabbit follow-up — 2026-11-02

CodeRabbit completed at `4c41f87` with a successful check and three concrete
comments. A successful check did not mean there were no findings.

- **4443418255:** corrected missing spaces in the evidence notes without changing
  recorded facts.
- **4443418304:** selection now requires an explicit MP3 compatibility policy in
  `internal/openrouter/speech_formats.go`, plus current catalog modalities and
  listed voice. The public catalog has no output-format field. Exact reviewed
  IDs are accepted; PCM-only and unknown/future models are rejected until qualified.
  This is a local request-format policy, not invented provider metadata or proof
  of synthesis availability. See the [official TTS format documentation](https://openrouter.ai/docs/guides/overview/multimodal/tts).
- **4443418321:** preview requests now use `http.AbortableRequest` with a deadline
  covering headers and body, explicit abort on failure, late-response cleanup and
  timer cleanup on success. Raised the dependency floor to 1.5.0; the lock already
  uses 1.6.0. See [AbortableRequest](https://pub.dev/documentation/http/latest/http/AbortableRequest-class.html).
  Cancellation is not proof that remote work stopped or was not billed. Reserved
  attempts remain reserved and no automatic retry is added.

Verification: 21 provider tests passed, including four new transport probes; Flutter
analysis clean; Go openrouter/configuration/briefing tests and real PostgreSQL
fixtures passed. Independent cross-review found no blocking issue in either code
delta. One initial probe asserted before asynchronous abort notification; flushing
that notification made the intended signal assertion deterministic. No live
provider request, device update, deployment or merge was performed.

## Native compilation receipt — 2026-11-02

At code revision `dd7df26d56f6e9e95a47486bb52bd635b2ceda40`, CodeRabbit's check
completed successfully and its latest summary reported no actionable comments.
The prior three findings were corrected; this is not a merge or release approval.

Ran `flutter build ios --debug --no-pub --no-codesign -t lib/preview_main.dart`
under an owned build lease and a five-minute process deadline. Exit 0, total 98.7s,
Xcode 69.8s. The resulting Runner is Mach-O arm64 with DTPlatformName `iphoneos`
and bundle `run.prosody.prosody`. Git status remained clean after the build.
This receipt commit changes documentation only; product code remains that revision.

Compilation is now verified for the corrected preview code. The artifact was not
signed or installed; it does not establish iPhone gestures, native playback,
background behavior, production login, provider output, Nanos or TestFlight
acceptance. No Keychain changes, provider calls or deployment occurred. The local
build output is now unsigned; do not reuse it as an install-ready signed artifact.

Capacity admission: interval idle 47.22/47.90%, normal pressure, zero swap churn,
no thermal/performance warning. Original hooks and shared lease protocol retained.

## Full corrected-head Flutter regression receipt — 2026-11-02

At `254983b099adcb6379dc46a4b63af985bed48f87` (product code unchanged from
`dd7df26`), the complete Flutter suite passed: 114 tests, exit 0, 55 seconds
reported test time. Command: `flutter test --no-pub --concurrency=2
--reporter=expanded`, with a five-minute process deadline and owned build lease.
This closes the aggregate-suite gap after the four new transport-abort tests;
prior full-suite evidence covered 110 tests before those changes.

CodeRabbit check on that exact head completed successfully at 16:13:13Z; its
updated summary reports no actionable comments. Its docstring-coverage warning
is still present; a successful check does not constitute release acceptance.
Capacity: load 3.43/3.31/3.53, interval idle 35.31/41.60%, normal memory pressure,
zero swap churn, no thermal/performance warning. No product changes, native
installation, live provider calls, deployment or merge in this verification.


## Current-main integration candidate — 2026-11-02

Reconciled the existing PR2 branch at `2f66a3867021dcd494d9d3d618e33959c22c8780`
with main `9b5a9224d87289e9fa074cb077271a9a97c29987`. Provider source merged
without conflicts: the speech catalog query and exact MP3 compatibility policy
remain alongside main's typed attempt observations, receipt preservation and
nil-safe diagnostic handling. The narration policy revision remains 2. Both
branches' append-only STATUS/WORK evidence was preserved when resolving conflicts.
No mobile source changed during reconciliation; debug-only preview composition,
feed controls, local deletion and HTTP abort behavior retain their prior bytes.

GitHub's three concrete review comments are marked addressed in `dd7df26`; the
latest review summary reports no actionable comments. Added a hermetic regression
for the combined speech-catalog/typed-outcome path: one forbidden response must
retain its status and receipt identifiers, expose no private body and never retry.
The new integration regression subsequently passed; see the executed receipt below.
The initial capacity refusal below is historical, not the final verification state.

Heavy verification was refused: load 9.30/10.40/11.56 during the sample, interval
CPU idle 0.48/0.14% (average 0.31%), normal memory pressure, zero swap churn and
no thermal/performance warning. No lease was claimed and no automatic admission
retry, hook bypass, commit, provider call, signing, installation or deployment was
performed. Prior 114-test Flutter/analysis/unsigned-iPhone receipts above remain
historical evidence for unchanged mobile bytes, not a new integrated-head result.


Lightweight checks during the hold: four hermetic iOS preflight CLI tests passed
(`python3 -m unittest scripts/test_ios_preflight.py`); staged whitespace clean.

Source inspection also confirmed a separate production wire blocker:
`store.ReadyBriefing` omits the `agent` string required by Flutter's
`Briefing.fromJson`, and `store.PublishedSegment` carries private `assetId` /
`imageIds` while Flutter requires `url` / `images` values that use HTTPS; each media request still requires current user authorization.
A raw JSON serialization of `BriefingReadPage` therefore cannot serve the native
items API. An explicit host presentation/delivery adapter must resolve legitimate
attribution and current authorized media URLs; no fake attribution, guessed media
endpoint, relaxed parser or production auth fallback was added here.


## Executed current-main provider integration — 2026-11-02

After an actual shared-workload release event, fresh admission passed: load
5.13/4.88/4.49, interval idle 84.97/84.41%, normal pressure, zero swap churn,
no thermal/performance warnings. The single hermetic
`TestSpeechCatalogFailureRetainsAttemptOutcome` passed at the resolved merge
candidate. It checks the speech catalog query, typed uncertain outcome, preserved
status/request/generation IDs, fixed safe diagnostic and one dispatch only.

The merge retains main's provider outcome and typed-nil regression bytes exactly.
Mobile source is unchanged from the previously verified PR2 head; no new Flutter
inputs require repetition of its 114-test/analysis/unsigned-iPhone checks. The
required lint/full-unit hook runs on the merge commit under the owned lease.
Four preflight CLI tests also passed on this integrated source. PostgreSQL fixture,
provider output, native playback/gestures, production wire delivery/auth and
TestFlight acceptance are not asserted by this receipt.
