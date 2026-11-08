# Native session bootstrap

The production entry point can compose the existing native session client from
three explicit release inputs: `PROSODY_ORIGIN`, `PROSODY_ISSUER_ORIGIN` and
`PROSODY_RESOURCE_AUDIENCE`. No value is inferred from another, and no production
value is selected by this source change. The reviewed release workflow supplies
none of these inputs, so released previews remain sign-in disabled.

All three absent preserves that disabled state. Partial or malformed configuration
refuses the entire binding; main renders the unavailable sign-in state with no
preview-data fallback. Complete configuration supplies both the real native
session and API origin to the existing app. Constructing the client performs no
network or platform call; existing restore, browser/S256 exchange, current-grant
checks and sign-out remain responsible for their actual operations.

This conservative release-input profile accepts only ASCII canonical HTTPS
origins without paths, userinfo, query or fragment. The audience is an explicit
absolute HTTPS resource with a nonempty clean path (including `/`), no query,
fragment, encoded path or repeated slash. This is intentionally narrower than a
general URL parser and is not a replacement for AMOS registration validation.
Invalid values yield one fixed diagnostic without rendering their contents.
The issuer and API may have different origins; a string relationship is not
assumed to confer permission. The existing client ID and private callback are
unchanged. No additional token type, refresh, secret, OIDC or cookie fallback is
introduced.

## Activation gate

Build configuration is a declaration, not proof that a server exists, a client
or audience is registered, or a person has access. Before a production release
supplies these inputs, qualify the exact deployed issuer/client/return/audience,
real human enrollment/browser flow, Keychain continuity and current-grant
consumption against the application-owned API. The server must independently
authorize every feed/media request. Do not relabel AMOS's bounded document profile
as published-feed/media authority. No live endpoint or supported adapter is
created by this bootstrap.

## Delivered preview and remaining service work

Latest source through127bf69 shipped as1.0.2/build42. Resume upload run42526001942
succeeded, both original and child journals record ACCEPTED, and the exact-read
run42526428033 reports VALID/not expired. Tester assignment/compliance/visibility
and device installation remain unverified. No duplicate upload is permitted.
This bootstrap is newer source and is not in build42.

At the last read, the selected backend name had no DNS answer (NXDOMAIN), and
there was no composed/deployed Prosody application host. The AMOS integration
request on PR514 remains unanswered; its qualified finite-document V2 public
host is bounded nonpublication evidence, not published feed/media or complete
worker/enrollment/deployment acceptance. Domain integration remains Prosody's
responsibility. Completing the service requires those real interfaces and
installation/profile prerequisites, reviewed deployment IaC and the actual
end-to-end sign-in/read/player journey. No DNS change, infrastructure apply,
AMOS pin change, role/schema adoption, signing or upload is selected here.
