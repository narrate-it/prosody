# iOS preview and TestFlight preparation

Prosody is **not ready for a useful TestFlight beta**. A native simulator build
and hot-reload connection have been demonstrated, but production sign-in and the
complete authorized submit → generate → listen journey remain unavailable.
This checklist does not authorize signing changes, paid provider calls, deployment
or upload. The owning Prosody session keeps implementation/release decisions;
AMOS keeps its generic authority, worker and human-session interfaces.

## Read-only preflight

Run from the repository root:

```sh
scripts/ios-preflight.sh
scripts/ios-preflight.sh --preview
```

The default reports source configuration and open release gates. Exit `1` means
blocked or unqualified; it never declares a release approved from static checks.
Preview mode exits `0` when local source/tool prerequisites and an existing
simulator app are present. That does not establish artifact freshness, a booted
simulator or a live debug connection. Exit `2` denotes invocation/tool setup error.

The script reads only known project/configuration files and artifact existence.
It locates tool executables and inspects the selected Xcode directory without
running Flutter, CocoaPods, a build or an installer. It does not read `.env`, query
Keychain, print team/certificate details, connect to Apple, create profiles or
change any settings. Signing properties supplied by external overrides are not
resolved. Never use a passing preview preflight as a release gate.

## Current source and evidence

| Area | Observed state | What remains |
| --- | --- | --- |
| App identity | Runner Release uses `run.prosody.prosody`. | Confirm intended registered identifier and matching App Store Connect record before first upload. Their existence was not inspected. |
| Version | `mobile/pubspec.yaml` contains `0.1.0+1`; Info.plist uses Flutter build-name/build-number variables. | Pick the beta version and an unused build number after checking App Store Connect history. Local source cannot establish uniqueness. |
| Signing | No `DEVELOPMENT_TEAM` in the checked-in Runner Release settings. | Configure the approved existing team/signing route. This does not mean David lacks an Apple account or certificate; neither was inspected. |
| Native project | Xcode, CocoaPods and Flutter executables found; background audio is declared. | Qualify release builds, plugin privacy requirements and actual physical-device behavior. |
| Preview | Implementation record reports successful debug simulator build and `r` hot reload in the dedicated Prosody Preview simulator. | Live session state and binary freshness must be checked at use time. No current worker attachment is inferred from an old process handle. |
| Human login | Production `main` supplies no human `InteractiveSession`; sign-in is visibly disabled. | AMOS must provide a supported mobile return/session and current-authority feed boundary, then Prosody must integrate and test it on iOS. No OIDC/refresh/custom-scheme assumption. |
| Agent-to-listener path | Domain/database and adapter tests exist; synthesis uses local test fixtures. | Qualified agent submission, admitted provider work, durable results, private media delivery and successful playback in the actual app. |
| Distribution | No signed archive, upload receipt or installed TestFlight build is claimed. | Separate signing, upload authorization, processing and tester installation evidence. |

## Repeatable existing hot-reload preview

The owning session successfully used this command from `mobile`:

```sh
flutter run --debug --no-pub \
  -d 31D5A590-40F4-4086-B3DE-A8EEAF34C081 \
  --use-application-binary build/ios/iphonesimulator/Runner.app
```

That device is the dedicated local **Prosody Preview** iPhone 17 simulator. The
identifier is machine-local: confirm it still exists using `xcrun simctl list
devices available`. Do not reset or reuse another session's simulator. If the
owner's Flutter session is already attached, use that session rather than starting
a competing runner. In its terminal, `r` requests hot reload, `R` hot restart,
`d` detaches and `q` quits. The current preview shows real onboarding and disabled
sign-in; it does not contain a fake signed-in feed.

The command reuses an existing debug binary. Native/plugin changes require a new
build; source changes still need appropriate analysis/tests. When a new binary is
required, first pass the applicable capacity/storage admission and acquire the
owned build lease. The previous successful build command was:

```sh
cd mobile
flutter build ios --simulator --debug --no-pub
```

Neither preflight nor the hourly reminder retries a refused admission, launches
a build or migrates execution to another machine. Preserve the already authorized
internal-disk exception and current host-specific limits. A simulator artifact
cannot be uploaded to TestFlight.

## Latest AMOS dependency checkpoint

The AMOS owner reports no reviewed/tested bounded worker or mobile interface yet;
both remain proposals. The local candidate beginning `809e060e` is unmerged. A
conflict-free dry merge onto newer main is source reconciliation information,
not service qualification. OAuth expiry across blocking writes, scoped grant
revocation and an owned verified-TLS service fixture remain dependencies.

The required order is: reconcile main and service evidence with independent review;
qualify worker dispatch, unknown-outcome reconciliation and fenced publication
with revocation/stale-fence tests; qualify the iOS return and current-authority
feed boundary with actual iOS expiry/revocation tests; run the joint authorized
submit-to-listen journey; only then consider separately authorized TestFlight
distribution. No current source adapter or local simulator receipt skips a step.

## Release checklist, in execution order

1. **Close the product gates — Prosody and AMOS owners.** Qualify human login,
   current feed access, agent consent/submission and the worker's current-authority,
   spending-admission and fenced-publication boundaries. Demonstrate the real
   text-to-listening path. Keep team support gated until its own qualification;
   a smaller personal-only beta is a proposed scope change requiring David's
   decision, not an already accepted release plan.
2. **Verify native behavior — Prosody owner.** Test a release configuration on a
   real iPhone: source/script views, swipe navigation, audio/video transitions,
   lock-screen queue continuation, interruption recovery, expiring/revoked
   authority, stale progress and private asset access. Record revision/device/OS
   and evidence. Simulator onboarding alone is insufficient.
3. **Resolve Apple custody — account owner or authorized release operator.**
   Confirm Developer Program access, authorized App Store Connect role, intended
   bundle registration/app record, and stable approved signing identity/profile.
   Inspect account state through the approved account workflow when authorized;
   do not infer absence from repository configuration or create/replace identities
   as a convenience. Do not alter Keychain defaults or key access.
4. **Prepare the beta configuration — Prosody owner.** Set the approved service
   origin and qualified session composition, final app icons, version/build number,
   release signing settings and any required capabilities. Verify privacy/export
   declarations against actual data use and SDKs; do not copy template answers.
   Prepare beta description, testing instructions and access instructions based
   on the working build. Confirm current Apple SDK/upload requirements at release
   time rather than hardcoding a historical minimum here.
5. **Create and verify the signed archive — authorized release operator.** After
   capacity admission and signing authority are established, Flutter's documented
   `flutter build ipa` workflow creates an archive and IPA. Use the approved
   existing signing route; inspect bundle/version/configuration and validate the
   archive before requesting upload approval. A successful build is not evidence
   of a successful install or backend behavior. [Flutter iOS release guide](https://docs.flutter.dev/deployment/ios).
6. **Upload only when authorized — authorized release operator.** Obtain approval
   for the concrete reviewed artifact, then use the approved Apple upload route.
   Record the revision, archive identity, version/build and processing outcome.
   Apple processing must finish before the build is available in App Store
   Connect. [Apple upload guidance](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds/).
7. **Admit testers and verify installation — release operator and David.** Choose
   internal/external testing deliberately, complete the applicable TestFlight
   information/review steps, and verify that David can install and perform the
   documented journey. Do not claim availability from an upload receipt alone.
   [Apple TestFlight overview](https://developer.apple.com/help/app-store-connect/test-a-beta-version/testflight-overview/).

## What specifically needs David

No Apple account action is assumed necessary today: account state has not been
inspected, and engineering gates remain open. Near signing/upload, David or an
already authorized account operator must confirm the intended Apple team and
existing access route. Only the account holder should handle any founder-only
membership, agreement or access decision discovered then. Do not ask him to share
passwords/private keys, or to recreate working signing material. Upload, deployment
and paid provider acceptance remain separate authorizations.

## Verification of this preparation

This worker ran `sh -n scripts/ios-preflight.sh`, the default read-only preflight,
and preview mode. Isolated temporary-fixture checks also verified rejection of an
unknown option, missing simulator binary and malformed version. The default
reported blocked/unqualified as expected; it did
not mistake local tools/source for TestFlight readiness. Preview mode checked the
existing binary only. No build, account inspection, signing or upload was performed.
Current Apple/Flutter links above were checked while preparing this record; they
are guidance, not evidence that Prosody satisfies the requirements.
