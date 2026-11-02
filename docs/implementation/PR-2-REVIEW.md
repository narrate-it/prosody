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

Current capacity admission: interval idle50.31/53.22%, normal memory pressure,
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
