# Linode Nanos staging — Go/Pulumi

This is a **powered-off staging graph**, not a working Prosody deployment.
It declares a protected Fremont (`us-west`) 2 GB instance, a 1,300 MB disk
from an explicitly supplied existing private image, and an offline direct-disk
configuration. Both instance and configuration explicitly set `booted: false`.
There is no boot switch, backend mount, DNS change, database or image upload.

The public registration API uses Linode provider **v6.9.0** and the pinned Pulumi
Go SDK. Inputs follow its instance, instanceDisk, instanceConfig and lookup schemas.
The direct-disk/paravirt/helper profile follows the inspected
[OPS Linode source](https://github.com/nanovms/ops/blob/a52795e38ba33a006f0ba740fee0f18fec58fbbb/provider/linode/linode_instance.go),
without copying its example password, polling loops or direct API effects.
This source comparison does not establish current Nanos boot compatibility.

## Inputs and gates

- An operator-reviewed, accessible `existingPrivateImageId` (`private/<number>`).
  The lookup must return that exact available private image with a positive size
  fitting the disk. This verifies provider metadata, not image bytes, custody,
  provenance, workload identity or native/runtime qualification.
- `diskRootPassword` supplied as encrypted Pulumi secret configuration. It is
  forwarded as a secret input/output solely for the provider image-disk API;
  it does not create a Nanos login service. Never embed secrets in an image,
  checked-in config, shell arguments or logs. No example credential is provided.
- Linode provider authentication through approved secret custody; the existing
  `.env` credential is not read by this program or committed. Reconcile the exact
  target account, existing infrastructure owner and state before preview.
- An approved state backend/secrets provider and a complete reviewed cost plan.

The compute-type lookup refuses missing/ambiguous prices, nonfinite values,
sub-cent quotes and any quote above **$12/month**. A regional quote overrides the
base quote. This is only a compute guard, **not enforcement of the owner's total
$50/month cap**. Image storage, database, media storage, transfer, tax and backup
costs still need explicit accounting and operational limits before apply.
Powered-off instances can still incur charges. Built-in backups are disabled;
this is not approval to operate without qualified backup/restore.

Only after inputs, custody and total cost are admitted, use an approved Pulumi
stack and review `pulumi preview --diff`. No preview or apply is implied by this
source. No missing image should be replaced with a Linux/demo image. Initial
apply may partially create resources before an error; reconcile actual state
before retry or cleanup. Protection must not be silently removed, and resources
left offline continue to require billing/custody reconciliation.

Do not use `ops instance create` or another CLI to introduce a competing owner
for this graph. An eventual image publication path, firewall/TLS/data services,
workload boot, monitoring and backup/restore must be added through reviewed IaC
and actual qualification before exposing `api.prosody.chat`.

## Verification scope

Run unit/mock tests and vet/lint within this nested module. The repository root
hook does not traverse it. Mock registration tests inspect protection, secret
flags, the price/image gates, exact numeric child references and offline state;
they do not contact Linode. Real provider preview, image/boot acceptance and the
full budget remain separate gates. No infrastructure is created by tests.
