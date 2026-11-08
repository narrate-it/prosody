# Read-only App Store build status

The manual `asc-build-status.yml` workflow inspects one existing iOS build.
It runs only from trusted `main` on the approved self-hosted macOS runner,
using the protected `prosody-asc` environment. It does not sign, build, upload,
modify Keychain settings or change Apple records.

Supply the exact marketing version and build number (initially `1.0.1` and
`41`). Marketing versions contain three canonical numeric components, each
0–999; build numbers are canonical positive decimal integers of at most
18 digits. Invalid selectors fail before credentials are consumed.

The reader first verifies the existing bundle ID against the app query,
then queries builds using that verified app ID, iOS platform, marketing
version and build number. Each request receives its own five-minute JWT with
one exact GET scope. No pagination, redirects, retry, alternate origin or
broader query is permitted. Responses use verified TLS and a 64 KiB body
limit. The response must contain exactly one matching build and included
prerelease version, matching app and relationship IDs, a known processing
state and a boolean expiry value. In the build response, duplicate decoded
keys, invalid UTF-8, non-JSON constants and overflowing numbers fail closed
with safe diagnostics. The legacy app-response decoder is unchanged.

Only the requested version/build, processing state and expiry flag are
reported. `VALID` is not proof of export compliance, tester membership,
TestFlight visibility or installation. Team association is also unverified.
An absent or ambiguous result is a refusal, never permission to upload again.
Build41 already has accepted upload receipts; do not duplicate that upload.

The workflow has a three-minute deadline and places only the four ASC API
secrets on its read step. It never receives the Keychain password. Live
execution requires the reviewed workflow on main, the approved available
runner and the existing execution gates. Authentication failures stop rather
than authorize automatic retry.

Apple documents the [build-list filters](https://developer.apple.com/documentation/appstoreconnectapi/get-v1-builds)
and [processing/expiry attributes](https://developer.apple.com/documentation/appstoreconnectapi/build/attributes-data.dictionary).

## Source verification

Sixteen new hermetic roots pass, along with seventeen existing ASC preflight
roots and twenty-nine release roots. These are separate suites, not live
Apple evidence. The initial new-suite run failed on an HTTPError test fixture;
that failure is retained, and the fixture was corrected before the green run.
Review identified numeric overflow; explicit positive/negative overflow and
finite-number controls are included in the corrected sixteen-root run.
No pre-fix overflow execution is claimed. The exact owned verification lease
was released. Required repository hook and independent review remain separate.
No live build status has been read by this source preparation.

## Live read and fixed refusal stages

PR70 landed at213e205. The one read-only workflow run42513773221 reached the
build-response validator but refused its metadata; it did not verify processing,
expiry or tester visibility. No authentication failure was reported. Its exact
owned lease was released; build41's accepted upload remains unchanged.

The follow-up reader reports only a fixed local validation stage: `json`,
`envelope`, `build-count`, `train-count`, `build-resource`, `app-binding`,
`train-binding`, `build-version` or `status`. These identify where validation
stopped, not the remote cause. No provider field, identifier, value, response
body or raw exception is printed. The same exact-match predicates still apply;
a refusal cannot authorize upload. Another read waits for reviewed landed
source and current execution admission; authentication failure stops retries.
