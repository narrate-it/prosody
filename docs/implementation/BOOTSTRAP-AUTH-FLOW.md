# Configured Dart authentication flow

The release-input factory is now exercised through a successful synthetic browser
return, token exchange, current-session validation, exact-binding storage write,
and authenticated empty-directory request. The real Dart NativeSession and
ProsodyApi clients are composed by the existing factory; http.runWithClient
supplies test-only transport at their ordinary client construction sites.
Production source, release configuration and shipping workflow are unchanged.

Three additional widget cases cover acceptance, token403 denial and a current
response carrying another audience. They check one browser/token attempt, exact
client/code/verifier/redirect form fields, current token and binding, no redirects,
Bearer delivery only to the configured API, ordered requests and safe diagnostics.
Denied cases neither write an active record nor render the feed or expose a token.
The accepted case must render the successful empty inbox, not merely FeedScreen.

All responses, cryptographic proof values and platform storage are synthetic.
These tests establish Dart composition only. They do not qualify actual PKCE
cryptography, ASWebAuthenticationSession, Keychain durability, real issuer/grant,
backend authorization, media access, physical device or release behavior. No
old-production RED bug is claimed for this coverage addition. Production remains
inactive and accepted TestFlight1.0.2/build42 is unchanged.

## Verification

Verification receipts and independent review are recorded in WORK.md. The initial
source review requested an explicit empty-inbox assertion because FeedScreen can
also render a directory error; that test-fidelity correction is part of this lane.

Final strengthened-assertion revision: analyzer zero issues and focused59 named
PASS0fail0skip/done-success-true; exactdf76 owned lease RELEASED. Prior full453
passed before this test-only assertion delta, not a fresh full-suite verdict for
the final revision. Production bytes remain unchanged; focused rerun covers all
changed widget cases. Independent affected source review CLEAR. Hook and hosted
review remain separate.
