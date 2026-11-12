# SES identity prerequisites

David selected AWS SES, `noreply@prosody.chat`, and resource audience
`https://api.prosody.chat/`. The audience is an issuer registration input, not
an SES field or proof of registration. Backend hosting remains Linode/Fremont
with the existing total $50/month ceiling.

This Go/Pulumi program pins AWS provider v7.54.0 and requires an explicit AWS
account and the supported `us-west-2` region (Oregon); there is no default.
Other regions are rejected until their exact DKIM domain is reviewed.
[AWS documents the regional suffix differences](https://docs.aws.amazon.com/general/latest/gr/ses.html).
The provider allowlist and caller-identity check must match
before any SES resource is declared. It declares a protected domain identity,
DKIM tokens and the `prosody-identity` configuration set. No mail, IAM credential,
event destination, DNS record, server, database or image is created by this code.

The TXT and three DKIM CNAME projections are marked secret for encrypted Pulumi
state; actual state-backend/secrets-provider custody must be qualified before apply.
Reconcile
existing SES identities and Cloudflare DNS ownership before a reviewed preview
or apply. The owning Cloudflare IaC must adopt/create the exact reviewed DNS
records; do not make hand changes or let two stacks own one record. Domain
verification, DKIM verification, regional sandbox/production sending access,
scoped runtime credentials, verified event custody/reconciliation, bounces and
suppression remain operating gates. Creating a configuration set does not
configure or qualify its event path or establish inbox delivery.

There is no stack or live account approval embedded here. Use existing approved
secret custody for provider authentication; do not put credentials in checked-in
configuration. Review the actual preview and complete the total budget before
apply. Protected retained resources require deliberate custody and cost
reconciliation if the stack is removed. No SES sending is enabled by a source
test or this prerequisite graph.

Resource inputs follow the official Pulumi schemas:
[DomainIdentity](https://www.pulumi.com/registry/packages/aws/api-docs/ses/domainidentity/),
[DomainDkim](https://www.pulumi.com/registry/packages/aws/api-docs/ses/domaindkim/),
and [ConfigurationSet](https://www.pulumi.com/registry/packages/aws/api-docs/ses/configurationset/).
The root Go hook does not traverse this nested module; run its own tests, vet
and lint. Mock registration checks do not contact AWS or verify credentials,
DNS propagation, mail delivery, backend readiness or Nanos compatibility.

## Current source checkpoint

Executed: the corrected hermetic mock suite passes (now 23 named records across
six roots, including bad-target zero-registration and root-principal refusal),
`go vet` clean and `golangci-lint` 0 issues (Go 1.27.0 toolchain). Committed through
a package-aware hook that runs this nested module's lint+test (the root hook skips
nested modules); PR #77. No live preview, apply, sender verification or email
delivery has run.

Independent headless review (PR #77) returned CLEAR with hardening findings, since
addressed: a `.gitignore` matching sibling modules; hoisted regexps; a non-root
principal refusal (the account check alone admitted root in the same account);
bad-target and DKIM-dependency-type assertions; and the IAM policy revision below.
Not implementable with the current Pulumi mock: asserting the `getCallerIdentity`
Invoke version (MockCallArgs carries no version) and capturing export secrecy
directly (resource-side AdditionalSecretOutputs and pulumi.ToSecret are checked).

## Scoped runtime profile (bootstrap)

SES work uses the named AWS profile `prosody-ses` (region `us-west-2`), not root.
The IAM user `prosody-ses` carries the customer-managed policy
`prosody-ses-boundary` (ARN `arn:aws:iam::394824061124:policy/prosody-ses-boundary`),
whose exact document is checked in as [`iam-policy.json`](iam-policy.json). It is
SES-only: create/verify/read management and list actions are action-scoped with an
`aws:RequestedRegion = us-west-2` condition (verified to deny out-of-region calls);
sending is scoped to the `prosody.chat` identity AND `prosody-identity` configuration-set
ARNs (SES requires both when sending with a configuration set); plus
`sts:GetCallerIdentity`. IAM/EC2 and all other services are denied. Management actions
use `Resource:"*"` because SES classic create/verify actions may not support
resource-level permissions; confirm exact support at the gated apply and tighten if
possible. Delete actions (`ses:DeleteIdentity`/`ses:DeleteConfigurationSet`) are
deliberately absent: they have no resource type and would be account-wide, so deletion
is reserved for a separately controlled principal (verified `implicitDeny` via policy
simulation). The stack's Protect+RetainOnDelete does not need them.

Note: `prosody-ses-boundary` is an identity policy (it grants), not an IAM permissions
boundary; the denials hold because no other policy is attached. Before production,
split the IaC-apply principal from the runtime-send principal (drop notification/list
actions from apply; scope send to identity + configuration set in a distinct runtime
role) and adopt the bootstrap user into an admin-managed IAM stack.

The user and an access key were created as a CLI bootstrap (the stack runs as this
user and cannot create itself); the policy is reconciled into code here. The secret
access key lives only in `~/.aws/credentials` (mode 600), never in source.
