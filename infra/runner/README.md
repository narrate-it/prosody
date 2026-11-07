# Existing MacBook runner reassignment

This reviewed IaC reassigns the **existing** `prosody-laptop-runner` from
`sirerun/prosody-legacy` to `narrate-it/prosody`. It does not install a runner,
download a replacement package, build/sign an app or change Keychain access.
The owner explicitly approved this reassignment. New-repository jobs still need
their own source/review/capacity/signing/upload acceptance.

Go/Pulumi uses command provider v1.3.0 to own one protected host-local operation.
Preview declares the operation; `--plan` on the reconciler separately inspects
the real host, local metadata, GitHub identity and stopped service without effects.
Only a reviewed `pulumi up` executes `--apply`. The host, existing directory and
private custody directory are explicit stack config; never commit machine paths.
Use the already approved private state backend/secrets provider.

The reconciler requires the exact local/server ID and name, an idle/offline old
runner, no local listener/worker process and no colliding target name.
A successful complete source inventory may instead prove the old server record
is already absent. Only that explicit case uses the installed runner's supported
local-config removal; authorization/read failures never select that branch. It refuses a loaded old LaunchAgent or a
service pointing elsewhere. For an existing server registration, it obtains both one-hour token permissions before
changing the old local setup. Official config commands receive tokens only via
the runner-supported `ACTIONS_RUNNER_INPUT_TOKEN` environment, never arguments,
stdout or Pulumi inputs. No credential files are read or copied by this helper.

Only the exact stopped runner-owned `.service` and LaunchAgent are removed;
the public service definition is retained in private custody. Official runner
configuration removes the old registration and writes the new one. The helper
then installs/starts the new LaunchAgent and checks GitHub online status under a
bounded deadline. It preserves the existing package and legacy work directory;
new jobs use the separate `_work-narrate-prosody` directory. Target platform/project
labels are verified.

A nonsecret checkpoint and retained service backup are file/directory-fsynced
before destructive work, including newly created custody parent entries. Owned
subprocess groups are finitely reaped on timeout/interruption, including when the
leader exits before its pipe-holding descendants. Untrappable hard termination
remains uncertain and requires inspection. A checkpoint records partial phases. Never blindly repeat
after failure: inspect local registration, both repository inventories and service
state. An already confirmed target registration is verified without rotating
credentials again. Missing local metadata after a partial transition requires an
explicit recovery decision; this helper will not invent ownership or roll back
another registration. Protected destroy is intentionally refused and needs a
separate reviewed custody action.

The Mac must remain awake and connected. Runner availability does not repair
Apple signing or complete Prosody's backend. No workflows are automatically
dispatched by this resource. Future iOS jobs must retain self-hosted macOS labels,
shared capacity/lease gates, immutable source and existing review/signing rules.

Hermetic tests exercise mocked API/config/service transitions, denied permissions,
idempotent verification, host/identity boundaries and token transport. These are
not real runner, Keychain, Apple or TestFlight acceptance. Actual preview/apply and
GitHub online evidence are recorded separately with private recovery custody.

## Opt-in GUI security session

`useGuiSecuritySession: true` adds a second protected local resource for the
already registered runner. Set `guiSessionCustodyDirectory` to the dedicated
host-local custody directory required by `scripts/runner-gui-session.py`.
The default remains unchanged. Changing this setting is a reviewed runner
configuration effect, not signing authorization or a Keychain permission repair.

The session helper changes only the Boolean `SessionCreate` from true to false
in the existing owned LaunchAgent, so the service uses the logged-in GUI session
instead of requesting a new security session. It preserves all other plist
values, registration and platform labels. It never changes the installed runner
template, credentials, Keychain ACLs/default/search list/lock policy, or Apple
assets. The source of this candidate remedy is the matching service-versus-
Terminal failure reported in actions/runner issue 3407; it is not a guaranteed
fix. Our Terminal test signed with the expected certificate, while the service
archive still failed; that distinction alone does not prove causality.

Run `--plan` with the explicit host/directory environment before Pulumi preview.
Apply requires an online idle exact runner, the existing GUI domain, no observed
job worker, and a second idle/source check immediately before stop. Admission
observations cannot eliminate an external job-dispatch race; coordinate an empty
queue and no new dispatch during this short operation. The source retains the
original plist and fsynced phase receipt before stop, uses bounded launchctl
calls, then checks restart/online status. No automatic rollback or blind retry
is performed after an ambiguous failure. Keep the logged-in Mac awake. A stopped
or partial service requires inspection of the exact retained phase and plist,
not re-registration or deletion. The protected destroy intentionally refuses.

Review the real Pulumi preview before any apply. After successful application,
qualify one tiny signing operation in the actual runner context before another
full archive. No session apply, native signing check, archive or upload is
established by the helper's hermetic tests.

Review corrections serialize applies using a retained, nonblocking host-local
lock inode held before service/receipt observations through terminal verification.
The lock file is never unlinked on release. This prevents overlapping reconcilers,
not external job dispatch. Pre-stop busy/worker/drift refusals leave no transition
receipt; `stop-started` is the first recorded phase after final admission, before
any stop call. An actual ambiguous stop still requires explicit reconciliation.
