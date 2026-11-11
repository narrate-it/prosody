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

Initial nested mock tests passed 19 named records across four roots with zero
failures/skips; vet passed and lint reported zero issues. Independent review then
identified the regional DKIM-suffix issue. Corrected source restricts the profile
to an explicitly selected us-west-2 and adds refusal/secret-output assertions.
Those corrected checks remain unexecuted because final capacity admission held.
The initial result is not a verdict on this corrected revision. No live preview,
apply, sender verification or email delivery has run. The root hook is also
unexecuted for this lane. Retain the checkpoint; do not bypass the hook to commit.

## Scoped runtime profile (bootstrap)

SES work uses the named AWS profile `prosody-ses` (region `us-west-2`), not root.
The IAM user `prosody-ses` carries the customer-managed policy
`prosody-ses-boundary` (ARN `arn:aws:iam::394824061124:policy/prosody-ses-boundary`),
whose exact document is checked in as [`iam-policy.json`](iam-policy.json): SES
identity/DKIM/configuration-set management and sending scoped to `prosody.chat` and
`prosody-identity` in us-west-2, plus `sts:GetCallerIdentity` and SES list reads.
IAM/EC2 and all other services are denied.

The user and an access key were created as a CLI bootstrap (the stack runs as this
user and cannot create itself); the policy is reconciled into code here. Rotate or
adopt the user into a dedicated admin-managed IAM stack before production. The
secret access key lives only in `~/.aws/credentials` (mode 600), never in source.
