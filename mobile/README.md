# Prosody mobile

Requires Flutter 3.47.9 or newer (Dart 3.13.7 or newer).

From the repository root, run `make preview` to launch Flutter in debug mode.
Choose a connected device when prompted, or use `make preview DEVICE=<device-id>`.
If Flutter is not on PATH, set `FLUTTER=/path/to/flutter`. In the attached terminal,
press `r` for hot reload, `R` for hot restart, and `q` to quit. This command may
resolve dependencies and build the app; Flutter and the target platform toolchain
must be installed. This command selects the debug-only `lib/preview_main.dart`
entry point: bundled spoken briefings, personal/team sample feeds, and
session-local progress. Without explicit provisioning it stays an OFFLINE DEMO.
The feed dropdown also opens Create live briefing. The bounded live preview uses
GPT-4.1 mini to condense entered text, then elevenlabs/eleven-v4-turbo with Laura.
It does not test AMOS sign-in, agent submissions, notifications or cross-device
persistence. Production authentication remains separate.

For the owner-authorized $1 test session, provision the running debug app through
its authenticated localhost VM service tunnel (the URL printed by Flutter):

```sh
dart scripts/configure-live-preview.dart <localhost-vm-service-url> /path/to/.env
```

Run this from the repository root. The key is read locally and sent through the
debug tunnel into app memory; it is never a build flag, asset, source file or log.
Restart/sign-out clears the key. A fixed session ledger in app support storage
reserves $0.25 before each paid attempt, at most four attempts; failures and
uncertain outcomes are not refunded or automatically retried. Reservations
survive restarts, but are local-install scoped: do not delete app data/reinstall
to reset the allowance. Provider catalog prices are checked conservatively; this
is not an OpenRouter-enforced account limit. Input/output bounds limit request
cost and actual decoded audio must be at most ten minutes. Generated audio stays
in private app files; feed metadata/progress reset with the preview instance.
Use `flutter run -t lib/main.dart` for the production composition described below.

Flutter iOS/Android client. Production `main.dart` has no sample-feed fallback or implicit development identity.
The configured service must provide qualified Prosody consumption APIs and an
AMOS human mobile session boundary. Production `main` currently supplies no
`InteractiveSession`: sign-in is visibly unavailable and no credential is minted,
restored, refreshed or substituted. The interface is for future qualified
composition and isolated tests, not evidence of an available native adapter.

AMOS exact `809e060e` source (owner-reconciled) exposes optional native public
code+S256 PKCE with opaque access tokens of at most 15 minutes. It has no refresh
token or OIDC ID token. `/oauth/current` and `/oauth/revoke` are POST endpoints
using form `token` and `client_id`. Only registered-path numeric `127.0.0.1`
loopback redirects are accepted. This does not establish an iOS Flutter browser
return or general bearer consumption API. MCP OAuth remains a separate protocol.
The earlier AppAuth/OIDC adapter, refresh assumptions and custom-scheme platform
registrations have been removed. No token-paste or alternate identity fallback
is provided. Native browser/PG evidence is not mobile acceptance.

`PROSODY_ORIGIN` is the HTTPS service origin for future qualified composition.
Release signing is not configured with a debug identity.

Playback uses one AudioService handler, just_audio for audio, and video_player
for supplied video. iOS audio background mode and Android media playback service
are declared. Background transitions across audio/video, interruption recovery,
progress persistence, and physical-device acceptance remain required. The waveform
is decorative playback activity and respects system reduced motion.

Package sources consulted: https://pub.dev/packages/audio_service,
https://pub.dev/packages/just_audio, https://pub.dev/packages/flutter_secure_storage.
Source/dependency installation is not a successful analyzer, build or device run.

In the personal preview inbox, open **Read → Delete briefing** to remove an item
with confirmation. Generated local audio is removed before metadata; cleanup
failure keeps the item for an explicit retry. Bundled demo assets remain part of
the app. Team deletion and production deletion controls are not enabled. This
local action does not erase provider-held data or reset the spending allowance.
