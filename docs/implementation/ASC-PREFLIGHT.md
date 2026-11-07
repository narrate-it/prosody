# Read-only App Store Connect preflight

The manual `ASC read-only preflight` workflow runs only from this repository's
main branch on the approved `self-hosted`, `macOS`, `prosody` runner. It reads the
existing `run.sire.prosody` app using four environment-only `ASC_*` secrets
in the main-restricted `prosody-asc` GitHub environment. The workflow uses JSON-form YAML so its job/step security structure is parsed
with the standard JSON decoder in hermetic checks. Checkout is pinned; the fixed system Python runs in isolated mode to refuse retained module/PYTHONPATH shadowing. Git credentials are not persisted. No PR/push event executes this job.

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


## Protected environment activation gate

The job-level main condition is defense in depth, not an immutable authorization
boundary: a workflow writer can edit it. `infra/github-asc` prepares the separately
admin-controlled environment policy through Go/Pulumi. Preview/review/apply and
actual policy verification must complete before secret activation or dispatch.
The four `ASC_*` names must exist ONLY in the protected environment before
activation, never at repository or organization scope for this repository.
GitHub can resolve a repository copy when an environment value is missing; the
workflow mapping cannot prevent that fallback. Retire broader copies through
reviewed IaC after checking authorized consumers, then verify absence.

The owner reports copying the original values and metadata confirms all four
names in `prosody-asc`: `ASC_API_KEY`, `ASC_ISSUER_ID`, `ASC_KEY_ID`, and
`ASC_TEAM_ID`. Metadata does not prove value equality or Apple validity. The
main-only environment policy is already applied. The owner subsequently deleted the repository copies. Read-only verification
confirms zero repository and accessible organization secrets, all four protected
environment names, and one exact main branch policy. This owner action clears
the placement prerequisite; it was not a retirement performed by our IaC helper.
PR38 subsequently passed independent/hosted review and merged at494f76a. The
first trusted-main live account check (run42340941986) failed at HTTP access;
the original diagnostic exposed no status. Account access is not accepted. No
secret value was read by the agent and no distribution signing or upload occurred.

## Safe HTTP diagnostics

A failed HTTP response reports only its bounded integer status. Connection
failures and invalid responses use separate fixed messages; no remote reason,
URL, headers or response body is disclosed. A status is diagnostic evidence, not
a credential-validity verdict or permission to retry. No automatic retry occurs.
