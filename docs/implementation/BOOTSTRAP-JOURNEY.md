# Configured Dart bootstrap journey

`main` delegates the existing release-input parsing and native-session creation
to `ProsodyApp.fromReleaseInputs`. It passes the same three compile-time strings
and the already configured playback handler. The factory shares the production
composition with widget tests; it adds no configuration default or activation
flag. Existing explicit `ProsodyApp` construction remains available.

The tests execute this factory, the real Dart `NativeSession`, `ProsodyApp`
startup and its sign-in button. Native platform calls and playback are mocked.
They run at a 390x844 test viewport with test fonts, not on a physical phone.
They do not execute `main`'s OS AudioService initialization, Swift PKCE/Keychain/
ASWebAuthenticationSession, a successful token exchange, real backend requests,
AOT release build, signing or installation.

Nine widget cases cover:

- Completely absent inputs keep sign-in disabled without platform calls.
- Each missing origin/issuer/audience input refuses partial activation.
- A malformed value remains disabled and is not rendered in diagnostics.
- A complete explicit binding composes the native client, restores signed-out
  state and enables the button without opening a browser automatically.
- Pending revocation storage refuses authenticated feed display and active-token
  use, preserving its separate recovery record.
- Keychain read failures show fixed generic diagnostics and do not open a browser.
- Cancelled browser login stays signed out; only a second explicit button press
  starts another attempt. The declared issuer, client, private callback, personal
  scope and S256 request mode are checked. Mocked proof values establish neither
  native randomness nor cryptographic correctness.

The pending-record fixture respects exact active/pending Keychain bindings;
there is no broad return-any-grant adapter. No sample feed or fake grant supplies
access after a production configuration failure. Invalid release values never
fall back to preview data.

## Evidence and remaining acceptance

Analyzer zero issues, focused56/fullFlutter450 named PASS0fail0skip with successful
completion (overlap, not additive). These include the new nine widget cases and
existing configuration/session controls. No earlier production bug or old-source
RED result is claimed for this behavior-preserving composition refactor.
Required hook, independent review and the isolated omission control are separate
stages recorded in WORK.

The shipping workflow supplies no native-session configuration, so production
sign-in remains inactive. Latest uploaded1.0.2/build42 remains accepted and valid;
this source is newer and selects no duplicate or new upload. Backend origin/
audience/client registration, transactional email, installed native storage/key
profiles, real browser return and current authorized application access still
need qualification. Basic native sign-in host composition can be prepared using
AMOS public host/issuer APIs independently of application generation or media;
this does not qualify its deployment or those domain endpoints.

In an isolated compiled copy, removing only session wiring makes the configured
browser journey fail its exact expected-first-attempt assertion; the disabled
control passes. The first audit incorrectly looked for assertion text only in
error events and failed. A retained separate event/line audit finds the actual
Flutter TestFailure in a print event, with one intended failure and one control;
no rerun or new source result is claimed. Production source hash stayed unchanged.
