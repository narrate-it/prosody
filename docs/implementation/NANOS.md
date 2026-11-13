# Nanos packaging and recovery contract

`backend/host` is the composed Prosody native account host (AMOS company18
issuer) with `backend/seslink` composing the SES verified-event subsystem;
`backend/installer` is operator installation tooling. Neither is operating-
qualified by source alone: the SES verified-event AWS path, TLS custody,
volume boot and provider journey remain explicit gates recorded in the
operator sitrep. The isolated `integration/amos` adapter remains unqualified
for hosting. Go `package main` entry points under `infra/` are Pulumi
infrastructure programs, not the application host. Repository Go/Pulumi modules now cover Linode staging, Cloudflare DNS,
protected ASC configuration and the self-hosted runner; the earlier no-IaC
snapshot is superseded. Their existence does not establish a running backend,
image provenance, durable storage or recovery acceptance.

[Linode staging](../../infra/linode/README.md) declares an explicitly powered-off
graph from an admitted existing private image. A complete total-cost plan, state
and secret custody, backend configuration and actual image/boot/readiness remain
separate gates. [READINESS.md](READINESS.md) distinguishes accepted preview delivery
from usable-service and full RFC acceptance. No image build, VM boot or deployment
is selected by this documentation reconciliation.

## Offline package inputs

`backend/host/ops-build.json` is the reviewed OPS configuration for the host
image. It carries no embedded files, credentials or secret interpolation: only
the `/custody` volume mount (`prosody-custody` label), non-secret custody env
paths and the listen address. Operator custody travels exclusively in the
separate private TFS volume image assembled by `scripts/assemble-custody.py`;
`packaging/nanos/contract.json` records that delivery and the remaining
qualification gates, and `scripts/nanos-package.py` enforces the exact shape
offline. It is not a complete production configuration. OPS documents JSON Args/Env/Files and its explicit
`ops build <app>` image-build command. See [OPS configuration](https://docs.ops.city/ops/configuration)
and [official OPS README](https://github.com/nanovms/ops) (consulted 2026-11-04).
[Nanos](https://github.com/nanovms/nanos) runs one application. Runtime workers
must use qualified in-process libraries or separately authorized adapters.

Run the bounded offline consistency check:

```sh
python3 scripts/nanos-package.py --self-check
python3 scripts/nanos-package-test.py
```

When a real composed Linux amd64 host artifact exists, inspect inputs with:

```sh
python3 scripts/nanos-package.py --binary /path/to/host \
  --source-revision FULL_PROSODY_COMMIT --amos-revision FULL_AMOS_COMMIT \
  --ops-version EXACT_OPS_VERSION --nanos-version EXACT_NANOS_VERSION
```

Artifact inspection accepts regular files up to 256 MiB, reads in bounded chunks
through one nonblocking-opened descriptor, and rejects observed size/metadata
changes. Concurrent writers must still be excluded by the artifact owner; this
is not an immutable-file or provenance guarantee.

The command emits deterministic sorted JSON with binary size/SHA-256, exact
caller-supplied source/tool pins, and contract/config checksums. It reads the
artifact and prints a manifest; it does not execute the artifact or OPS. Pins
are declarations requiring independent provenance verification. OPS and Nanos
pins must be a three-part numeric release identifier (optional lowercase `v`
prefix and prerelease suffix) or a full 40-character lowercase Git commit ID.
Bare moving aliases such as `main` and `latest`, abbreviated commits, surrounding
whitespace and other strings outside this grammar fail closed. Syntax cannot
distinguish a release-shaped branch such as `1.2.3-rc.1` from a release tag.
Tags can also move upstream; verify immutable source and artifact digests
independently for every accepted pin. ELF inspection
checks class, byte order, machine and executable type only; it cannot establish
Linux ABI, static linkage, Go runtime compatibility or Nanos acceptance. The
manifest intentionally remains unqualified. No dummy binary is supplied.

Before image build, verify source and dependency receipts, exact OPS/Nanos
versions and kernel digest, executable linkage, all trust roots and packaged
assets, and a complete host configuration loader. Build commands must be
reviewed against the pinned installed OPS help. No OPS parser validation, image
build, VM boot, tool installation, provider call or infrastructure apply has
been performed by this lane. The static checker does not replace those checks.

## Configuration boundary

`contract.json` lists semantic runtime inputs, not supported environment-variable
names. The eventual host must fail closed before serving on missing identity,
authority, database/object storage, encryption, TLS or finite worker/budget
configuration. Map those inputs to supported host APIs only after composition
exists. Never infer a new AMOS API or start a second job scheduler.

Secret references belong in reviewed configuration/IaC; secret values must stay
out of source, manifests, images, logs, instance metadata and plaintext IaC state.
A reference alone does not deliver a secret. Choose and qualify scoped runtime
resolution, rotation, expiration and boot failure behavior before deployment.
Customer OpenRouter credentials remain encrypted under the owned credential
boundary. Retain older wrapping keys needed to restore encrypted records.
Local media-file adapters are not qualified durable production storage.

## Recovery acceptance hooks (operator contract; not implemented commands)

There is no generic safe backup shell hook inside this single-process image.
The database/object-store/key custodians must supply reviewed external backup
and restore procedures through their own authority. No backup or restore is
claimed here. Before any production rollout record:

1. Database consistent restore point and schema migration revision; private
   object versions/checksums referenced by that database; retained decryption
   key versions; measured recovery point and recovery time objectives.
2. Previous immutable image and binary/kernel/config checksums, exact source
   and dependency pins, and schema compatibility for rollback. Reverting an
   image does not reverse a migration; destructive migrations need a separately
   reviewed restore or forward-fix procedure.
3. A restore drill in an isolated destination with outbound generation disabled.
   Restore records, objects and keys, verify decryptability and private access,
   then revalidate current authority through AMOS before resuming work.
4. Crash-after-admission, crash-during-provider-call and crash-before-publication
   drills using the authoritative job journal. Reconcile uncertain effects
   before retry; verify no duplicate charge, publication or resurrected deletion.
5. Restart/upgrade evidence for TLS, DNS, external database reconnect, bounded
   workers, authorization, private ranged delivery and readiness. Drain through
   supported host lifecycle controls; do not invent a health route or signal
   handler. Enable traffic only after the qualified readiness check succeeds.

Collect exact revisions, image IDs, receipts and observed outcomes without
credentials or private content. Keep restore drills, provider behavior, image
build, VM boot and visible product acceptance as separate evidence. Deployment,
spending, IaC apply and AMOS owner changes require their existing authority.
