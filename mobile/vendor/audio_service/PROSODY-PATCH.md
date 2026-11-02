# Maintained Android intent-validation patch

Baseline: installed `audio_service` 0.18.19 package, copied without modifying
the shared pub cache. The prior hosted lock entry records archive SHA-256
`95f3267f3449eb5cf71c8fcf1d556f57af1e898e2dc5815fb168d1843653edb7`.
`UPSTREAM-SHA256.json` records each copied baseline file before patching; this
is a local source-integrity manifest, not a new archive verification receipt.
Original MIT license, notices, README, changelog, package metadata, Dart library
and all Android/Darwin platform files are retained. Example and upstream test
directories are omitted. No dependency versions have been upgraded.

Original repository: https://github.com/ryanheise/audio_service

The installed receiver delegates directly to AndroidX. Its service separately
calls `MediaButtonReceiver.handleIntent`. Both public entry points therefore
need validation. Replacing only the receiver would leave service dispatch
exposed; changing component names would also break upstream's explicit service
bindings and notification PendingIntents.

Production patch:

- New package-private `MediaButtonIntentGuard` rejects missing, null, wrong-type
  and malformed parcelable key events for `ACTION_MEDIA_BUTTON`.
- `MediaButtonReceiver.onReceive` returns before delegation for rejected events.
- `AudioService.onStartCommand` returns `START_NOT_STICKY` before delegation for
  rejected events.
- Valid key events, null service restart intents and non-media/custom actions
  retain their existing paths. Intent contents are not logged by the guard.

The AndroidX dispatch behavior was checked against its published source:
https://android.googlesource.com/platform/frameworks/support/+/b827e60cec1f802a9fa2dbfbe6a0ddd7576bafca/media/media/src/main/java/androidx/media/session/MediaButtonReceiver.java

Device test source in `android/src/androidTest` covers invalid payloads through
both entry points and verifies valid/custom classification. The custom framework
Instrumentation runner is configured in `android/build.gradle.kts`; it adds no
test dependency or deprecated `android.test` library requirement. It does not prove
valid headset/notification dispatch or platform playback. No Android SDK was
found in the conventional local platforms directory during this change;
Android compilation and instrumentation execution are pending. Flutter tests
exercise Dart code and cannot qualify this Java patch. Before Android acceptance,
run these tests and actual media-button/notification controls on a device.

When updating this dependency, compare these two entry-point guards against the
new upstream source and retain them until equivalent upstream handling is
verified. Do not silently revert to the vulnerable hosted package.
