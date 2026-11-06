# Read-only App Store Connect preflight

The manual `ASC read-only preflight` workflow runs only from this repository's
main branch on the approved `self-hosted`, `macOS`, `prosody` runner. It reads the
existing `run.sire.prosody` app using the four configured ASC secrets. Checkout is
pinned; the fixed system Python runs in isolated mode to refuse retained module/PYTHONPATH shadowing. Git credentials are not persisted. No PR/push event executes this job.

`ASC_API_KEY` accepts the original PKCS#8 PEM or strict single-line base64 of that
PEM. The key must be P-256. Its temporary file is mode 0600 in a private temporary
directory, removed on normal/handled exits, including controlled SIGTERM unwinding. A process/host hard kill can leave
private files: inspect owned custody before cleanup; no secure erasure is claimed.
The script consumes secret environment entries before starting the signer; values
remain sensitive process memory and are not promised zeroized. The short-lived
JWT is scoped to app-list GET. No secret, token, response body, app ID or raw error
is printed or uploaded as an artifact. Redirects and implicit proxies are refused;
system TLS verification stays enabled. The job is capped at three minutes, signer
commands at ten seconds and network I/O at fifteen-second socket timeouts. Socket
timeouts do not promise a total response deadline; the runner job is the outer bound.

A PASS means exactly one matching app is readable by that key at observation time.
Team ID syntax is checked, but the app-list response does not prove that configured
team's association. Build-number history, distribution signing identity/profile,
Apple roles for upload, export/privacy declarations, backend/native/device behavior
and TestFlight delivery remain separate gates. No app, certificate, profile, signing
keychain or tester is created/changed. This is not the upload workflow.

Source references: [Apple JWT contract](https://developer.apple.com/documentation/AppStoreConnectAPI/generating-tokens-for-api-requests)
and [apps API](https://developer.apple.com/documentation/appstoreconnectapi/apps).
