# Prosody mobile

Flutter iOS/Android client. No bundled sample feed or implicit development identity.
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
