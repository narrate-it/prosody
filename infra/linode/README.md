# Linode Nanos staging — Go/Pulumi

This is a **staging graph with an explicit, reviewed boot gate**, not a
qualified Prosody deployment. It declares a protected Fremont (`us-west`)
2 GB instance, a 1,300 MB root disk from an explicitly supplied private app
image, a bounded custody data disk from a separate private custody image, and
a direct-disk configuration attaching both (`sda` root, `sdb` custody). Boot
is the `booted` stack configuration (default `false`); a powered instance
without the SES verified-event gate, TLS and custody would be non-functional,
so boot is only selected after those gates close.

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
- An operator-reviewed, accessible `existingCustodyImageId` (`private/<number>`),
  at most 512 MB. The custody image is the Nanos TFS volume built by
  `scripts/assemble-custody.py` + `ops volume create prosody-custody -d …`;
  Nanos probes it as a data volume and mounts it at `/custody` through the app
  image manifest (`backend/host/ops-build.json` `Mounts`). The app image carries
  no secrets; custody travels only in this separate private image. Treat the
  custody image as secret material: private to the account, replaced (new image
  ID) on rotation, and deleted when superseded.
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

Image upload uses the real Linode `POST /v4/images/upload` API
(`scripts/linode-image-upload.py`); the upload region must have Object Storage
capability (us-east is used for upload only — the instance remains us-west).
No object-storage subscription is created. Do not use `ops instance create` or
another CLI to introduce a competing owner for this graph. An eventual image publication path, firewall/TLS/data services,
workload boot, monitoring and backup/restore must be added through reviewed IaC
and actual qualification before exposing `api.prosody.chat`.

## Verification scope

Run unit/mock tests and vet/lint within this nested module. The repository root
hook does not traverse it. Mock registration tests inspect protection, secret
flags, the price/image gates, exact numeric child references and offline state;
they do not contact Linode. Real provider preview, image/boot acceptance and the
full budget remain separate gates. No infrastructure is created by tests.


## 2026-11-13 staging evidence (this session)

- App image `prosody-api-20261113b` uploaded and available (200 MB; ops
  0.1.47 build of `backend/host` linux/amd64 at the reviewed PR head — the
  binary embeds `vcs.revision` attesting the commit; manifest carries the
  `/custody` mountpoint and non-secret env paths only). A pre-commit build
  was superseded and deleted. The exact private image ID is operator custody
  recorded in the private sitrep and the local stack configuration, not here.
- Custody image `prosody-custody-20261113` uploaded and available (64 MB TFS
  volume: db custody with in-volume storeDir, native key store, Let's Encrypt
  TLS custody, cursor key, prosody-ses AWS profile). SES custody (`ses.json`)
  is intentionally absent until the verified-event gate closes; the host
  refuses startup without it. The custody image ID names secret-bearing
  material and stays in the private sitrep / local stack configuration.
- `pulumi preview` (stack `prod`, real Linode API): 5 resources to create,
  compute quote 1200 cents/month (g6-standard-1, us-west), status
  `powered-off-staging-not-qualified`. No instance has been created; apply
  waits for the SES verified-event gate so the instance is functional.
- Before boot: add the instance IPv4 (and IPv6 if used) to the managed
  database allow-list, and confirm the SES gate evidence in
  `docs/sitrep/2026-11-13.md`.
