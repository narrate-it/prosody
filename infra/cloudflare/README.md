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
A/AAAA/CNAME/MX/TXT records, TTL, proxy state, priority and supported settings
are preserved. The limited custody profile rejects nonempty comments/tags and
requires explicit per-record public-input review. Other types/settings fail closed until their
schemas are implemented. TXT content is not normalized or stripped of quotes.
Desired content/comment/tag properties retain Pulumi secret flags and additional
secret-output flags. These flags do NOT prove confidentiality of canonical inputs
returned during import. Actual adoption found plaintext canonical content inputs;
only qualifying output fields were encrypted. Do not import confidential values
through this profile, commit inventory/state or expose previews containing them.

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
    "stateInputClass": "public-reviewed-v1",
    "content": "EXISTING_ADDRESS", "ttl": 1, "proxied": true,
    "comment": null, "tags": [], "settings": {}
  }]
}
```

`stateInputClass` is documentary operator classification, not a grant to apply.
An absent/unknown/private classification rejects the whole inventory, including
before provider invocation when direct registration is used. Nonempty comments
or tags are unsupported even when the content was classified public. Do not strip
existing annotations or omit managed entries just to satisfy the guard. A public
DNS name does not make every API value public: for example, a proxied origin may
be intentionally hidden. Review each exact value before marking it public; this
program cannot infer that decision or prove confidentiality from a string label.
The guard checks only the supplied inventory. Remote comments, tags or other
canonical values can change between review and provider import; this code cannot
intercept or prove confidentiality of everything the provider returns. Fresh
inventory comparison and private state custody remain necessary. The guard is not
a qualified remote-state confidentiality boundary.

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

Before this custody guard, the nested module's 18 named unit/mock records
passed with no skips, and vet passed. A separate private-snapshot probe passed 17 records (16 repeated parser
records plus the snapshot probe) and verified the captured inventory has all 14
DNS records. Those totals overlap; do not add them. Compiled omission of the
Unicode/canonical-key guards failed both intended roots while 12 controls
passed. Initial dependency compilation attempts timed out before any tests;
the completed later run supersedes only that local module verification gate.

The original authentication/source-only note is historical. Using the separately
owner-provided provider token, the reviewed adoption-only saved plan later applied
14 imports. The post-adoption preview had no changes and independent Cloudflare
MCP inventories matched before/after; no DNS contents changed. That evidence does
not qualify this new guard against the provider or authorize another apply.

Persisted-state inspection at the pinned provider found all 14 canonical
`inputs.content` values plaintext. The present content/tag output fields were
encrypted. Input comment/tag fields and output comments were absent, which is NOT
proof of encryption. The credential was not found in the inspected artifacts;
private state/export custody remains required. No public state was published.

This source-only correction does not rewrite that state, erase historical copies,
update the live inventory or classify any real record. Existing inventories without
the review marker now fail closed. Confidential input support remains blocked until
exact persisted-input protection and custody are independently demonstrated. Do not
assume a no-op update or mock secret flag repairs an existing imported snapshot.
Historical state and backups need separate reviewed custody/remediation; this guard
supplies no erasure or zero-residual-exposure claim.

This guard's ordinary suite passes 31 named records across 10 roots, with zero
failures/skips. A compiled omission of both custody admission calls fails 12
records (including parents) while the valid public-input control passes. The first
race build hit its 300-second watchdog before any test event; it is NOT a race
pass. Fresh race, vet/lint, root-hook and independent-review evidence remain
separately recorded. No live provider or persisted-state check ran for this guard.

The one bounded race follow-up completed: 31 named records across 10 roots PASS,
zero failures/skips, with no cached test verdict. This overlaps the ordinary suite;
do not add the totals. Vet/lint pass with zero issues, and the required root hook
passes all 12 Go packages without a DB fixture. The initial compile timeout remains
historical failed evidence, not a race pass or a reason to relax admission.

Build this module from the full repository checkout: it reuses the repository
Unicode validator through the local parent-module replacement in go.mod.
