# Native account-host composition candidate

This isolated Go module selects the public AMOS company18 recipe supplied in the
[owner handoff](https://github.com/ajent-social/AMOS/pull/514#issuecomment-6745223644)
at exact source `845a8042ee6c1ba0b06731811303b7337fba9c85`. It does not change the
older `integration/amos` module's pin or activate that adapter.

`Run(ctx, Config, net.Listener)` calls the actual public `apphost.NewProduction`
and `ProductionHost.Serve`, then closes the owned native host. No alternate HTTP
server, authority callback, scheduler, database handle or private native SQL is
introduced. This is a composition library, not a complete executable, installer,
secret loader or deployable Nanos image. There is no claimed build/run result.

The fixed registration uses the owner-approved origin `https://api.prosody.chat`,
audience `https://api.prosody.chat/`, native client `prosody-ios` and exact callback
`run.prosody.prosody:/oauth/callback`. Display name is presentation only, not the
separately undecided original-producer attribution snapshot. The host selects
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

## Qualification state

Source-only, uncompiled and unexecuted under the existing capacity hold. Tests
are authored for exact profile/client projection, copied buffers, public issuer
configuration validation, account-only GET/HEAD/method/path responses and local
missing-listener/context refusal. The projection test deliberately does not
construct a fake NativeSet or claim that incomplete config passed admission.

No go.sum/dependency-resolution, build, test, lint or required hook result is
claimed for this module. No commit/PR bypass, installer, DB fixture, key-store
creation, SES send, DNS/VM mutation, AMOS activation or upload is selected.
Real installation, external-module native startup, HTTPS enrollment/email/PKCE,
current/revoke, iPhone/Keychain and deployment acceptance remain required.
