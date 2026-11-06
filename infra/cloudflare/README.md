# prosody.chat DNS — Go/Pulumi

This project adopts the existing zone's reviewed DNS records. It **does not**
create/delete a zone, change nameservers, select a paid plan, provision Linode,
activate a backend, or invent an `api.prosody.chat` address. The zone is looked up
and checked against the exact account and active `prosody.chat` domain before
registering any resources. Each record requires its existing ID, uses a stable
logical name based on that ID, and has both `protect` and `retainOnDelete`.

Cloudflare provider v6.23.2 is pinned in main.go; the Pulumi Go SDK is pinned
in go.mod/go.sum. The program uses Pulumi’s public registration/invoke API with
the pinned provider’s resource tokens and property schema. Existing
A/AAAA/CNAME/MX/TXT records, TTL, proxy state, priority, comments, tags and the
supported settings are preserved. Other types/settings fail closed until their
schemas are implemented. TXT content is not normalized or stripped of quotes.
Resource content, private comments and tags are marked as Pulumi secrets, so state uses the stack's secret
provider. The same three fields are explicitly listed as additional secret
outputs, without relying on provider-side inference. Secret values must never be committed or displayed in preview logs.

## Inventory and custody

The initial read-only Cloudflare MCP discovery found one active full zone and
14 records, including existing website and Google mail records. The exact
snapshot is in ignored `.private/inventory.json`, mode 0600; it is deliberately
not committed to the public repository. It must be re-read and compared before
any import/apply. Its shape is:

```json
{
  "zoneId": "EXISTING_ZONE_ID",
  "accountId": "EXISTING_ACCOUNT_ID",
  "records": [{
    "id": "EXISTING_RECORD_ID", "name": "prosody.chat", "type": "A",
    "content": "EXISTING_ADDRESS", "ttl": 1, "proxied": true,
    "comment": null, "tags": [], "settings": {}
  }]
}
```

This is an illustrative schema, not executable desired state. Use the actual
reviewed snapshot. IDs must be canonical 32-character hex. All records must
belong to this domain. Omitting import IDs is rejected locally; invalid or foreign IDs must fail the
provider import/preview. Their existence is not proven by local format checks.

## Preview before apply

Cloudflare MCP access is not automatically a credential for the Pulumi provider.
Provide a dedicated zone-scoped `CLOUDFLARE_API_TOKEN` through approved secret
custody (Zone Read and DNS Edit for this zone), or encrypted Pulumi provider
configuration. Never extract the MCP OAuth token or put any token in source,
this inventory, CLI arguments, output or unencrypted state.

Before the first import, reconcile any existing DNS IaC owner/state; do not
register a competing stack for records already managed elsewhere. The legacy
app repository has no matching IaC paths in the inspected source tree, which
does not prove that no other infrastructure owner exists.

Use an approved Pulumi state backend and secrets provider, then from this folder:

```sh
pulumi stack init prod
pulumi config set inventoryFile .private/inventory.json
pulumi preview --diff
```

Review the complete preview and compare current DNS before an authorized
`pulumi up`. The first adoption must contain imports only; unexpected updates,
replacements or deletes are a stop gate. A preview does not apply changes.
Protect and retain flags are safeguards, not authorization to remove records.
Removing a managed entry must not be used to silently drop custody; explicitly
reconcile state and retained live records. Import IDs remain stable on reruns.

The program does not manage zone settings, DNSSEC, mail credentials or unrelated
records by omission. An eventual API address must come from the reviewed backend
IaC outputs in a subsequent explicit change, after image/boot qualification.

## Checks

Run `go test ./...` and `go vet ./...` within this nested module under the shared
capacity/build-lease rules. Pure validation tests are not Cloudflare preview,
import or live DNS evidence. The repository's root Go hook does not traverse
this nested module; qualify it separately. No live update is implicit in tests.

## Current verification boundary

The nested module's 18 named unit/mock records passed with no skips, and vet
passed. A separate private-snapshot probe passed 17 records (16 repeated parser
records plus the snapshot probe) and verified the captured inventory has all 14
DNS records. Those totals overlap; do not add them. Compiled omission of the
Unicode/canonical-key guards failed both intended roots while 12 controls
passed. Initial dependency compilation attempts timed out before any tests;
the completed later run supersedes only that local module verification gate.

Cloudflare MCP zone/DNS reads succeeded. Its OAuth connection returned 403 on
`GET /user/tokens/permission_groups` with no supported OAuth scope. Token creation
was not attempted; no token was created or alternate authentication route used. Pulumi's provider still needs its own approved credential and a
reviewed state backend before live preview. No live preview, import or DNS
update is claimed. Unit mocks do not establish a no-change Cloudflare plan.

Build this module from the full repository checkout: it reuses the repository
Unicode validator through the local parent-module replacement in go.mod.
