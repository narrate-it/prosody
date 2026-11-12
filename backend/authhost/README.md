# Native account-host composition candidate

This isolated Go module selects the public AMOS company18 recipe supplied in the
[owner handoff](https://github.com/ajent-social/AMOS/pull/514#issuecomment-6745223644)
at exact source `845a8042ee6c1ba0b06731811303b7337fba9c85`. It does not change the
older `integration/amos` module's pin or activate that adapter.

`Run(ctx, Config, net.Listener)` calls the actual public `apphost.NewProduction`
and `ProductionHost.Serve`, then closes the owned native host. No alternate HTTP
server, authority callback, scheduler, database handle or private native SQL is
introduced. This is a composition library, not a complete executable, installer,
secret loader or deployable Nanos image. Executed local verification is recorded
under "Qualification state" below; it is not native runtime acceptance.

The fixed registration uses the selected origin `https://api.prosody.chat`,
audience `https://api.prosody.chat/`, native client `prosody-ios` and exact callback
`run.prosody.prosody:/oauth/callback`. These are the intended/selected values; the
audience approval receipt is not yet recorded, and READINESS still treats the
issuer/client registration as unqualified. Display name is presentation only, not
the separately undecided original-producer attribution snapshot. The host selects
issuer + directory + organization; NativeMethods, OrganizationCreation, worker,
MCP, paid operations, document profiles and all application routes remain unset.
The account homepage cannot serve successful fake feed/media responses.

Before use, the operator must retain the matching actual company18 installation
from public `identitybootstrap.InstallFresh`, its exact realm/role/ledger and
recovery binding, sealed `filestore.NativeSet`, separate runtime DB credentials,
trusted DB CA, native HTTPS certificate/key and distinct 32-byte directory cursor
key. Runtime readiness remains AMOS-owned and must actually pass. The bootstrap
requires independent owner/runtime/founder/label principals; never mount owner or
founder/label credentials into this runtime. Enrollment and email verification,
founder provisioning and presentation label approval remain separate operations.
Selecting the organization profile grants no person a membership or company scope.

`EmailSender` has concrete type `*ses.Sender` and must be the real SES adapter for the selected sender
`noreply@prosody.chat`, with explicitly selected account/region, scoped credentials
and qualified verified-event reconciliation/custody. No capture/no-op sender,
fabricated acknowledgment or inline paid retry is supplied. The identity-email
worker stays inside the native host. Generating briefings is a separate Prosody
integration; these host fields do not activate an application worker.

Config byte buffers for TLS, DB CA and directory cursor projection are copied. This is
not an immutable secret lifetime, zeroization or authority proof. The runtime
caller must retain native key-provider and sender lifetimes until Run returns,
own listener cleanup on startup refusal, and prevent concurrent configuration
mutation. The native TLS server controls its accepted listener; failed startup
does not authorize a listener, credential or infrastructure retry.

## Listener contract

`Serve` requires a plain `*net.TCPListener` bound to the origin port (`:443` for
this origin). `Run` accepts any `net.Listener`, but a wrapped/TLS listener or wrong
port is refused only after `NewProduction` has spent AMOS's process-global,
never-reset native authority activation. A post-admission `Serve` refusal therefore
consumes the process's single activation and requires a process restart, not an
in-process retry with a corrected listener. Supply a plain TCP listener on the
origin port before calling `Run`.

## Reproducing the verification

The module pins `github.com/ajent-social/amos v0.0.0-20261109161105-845a8042ee6c`,
which resolves only through a local workspace replace (the candidate is unpublished).
To reproduce from a fresh clone:

1. Fetch the exact source into `.local/amos-845a8042` (e.g.
   `gh api repos/ajent-social/AMOS/tarball/845a8042ee6c1ba0b06731811303b7337fba9c85`
   then extract with `--strip-components=1`).
2. Create `.local/amos.work` with `use ( ../backend/authhost ./amos-845a8042 )` and
   `replace github.com/ajent-social/amos v0.0.0-20261109161105-845a8042ee6c => ./amos-845a8042`.
3. In `backend/authhost`, run with
   `GOWORK=$PWD/../../.local/amos.work GOPRIVATE='github.com/ajent-social/*' GOTOOLCHAIN=go1.27.0`:
   `go test ./...` and `golangci-lint run ./...` (the linter needs the go1.27.0
   export-data format). `go list -m github.com/ajent-social/amos` must show the
   local `.local/amos-845a8042` dir.

No CI workflow or root hook covers this nested module yet; a scoped verify entry
is a follow-up. The root pre-commit `go test ./...` skips nested modules.

## Qualification state

Compiled and tested against the exact local AMOS `845a8042` checkout: projection,
copied buffers, public issuer validation, account-only GET/HEAD/method/path,
missing-listener/context refusal, and isolated directory-cursor-key guards
(zero/nil/short/long key, cancelled context) all PASS; `go vet` and `golangci-lint`
clean. The projection test deliberately does not construct a fake NativeSet or
claim that incomplete config passed admission.

This is executed local verification, not native runtime acceptance. No installer,
DB fixture, key-store creation, SES send, DNS/VM mutation, AMOS activation or upload
is selected. Real installation, external-module native startup, HTTPS
enrollment/email/PKCE, current/revoke, iPhone/Keychain and deployment acceptance
remain required.
