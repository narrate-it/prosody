# Consumption read seam — bounded worker record

Status: source implementation complete; tests **not run**. Parent owns integration,
serialized build admission and independent final review. No commits, pushes,
external calls, AMOS modifications or infrastructure effects were performed.

## Owned files

- `internal/store/reads.go`
- `internal/store/reads_test.go`
- This record.

## Contract

`ReadTransaction` exposes only `QueryContext` and `QueryRowContext`. `FromSQLRead`
adapts an already-open SQL transaction; an AMOS host can instead adapt its
restricted callback DB. None of these functions authenticates a caller or proves
current authority. Human consumption admission, transaction lifetime and authority
finality remain host obligations. Do not expose this seam as an agent receipt API.

- `ReadFeeds` pages nondeleted feeds from exactly one authorized workspace in ID
  order. It never enumerates arbitrary authorized-workspace membership itself.
- `ReadReadyBriefings` pages ready, nondeleted items in one selected feed, ordered
  by publication time, item ID and rendition ID.
- `ReadReadyBriefing` returns a currently ready item only within the supplied
  workspace and feed; wrong-scope, nonready or deleted records return the same
  unavailable error.
- V1 expects one published rendition per item. Multiple renditions fail closed
  with `ErrAmbiguousRendition`; there is no invented latest/current selection.
- Cursors are bounded base64url closed JSON, with exact field names, duplicate and
  malformed Unicode rejection, version and scope binding. A cursor is a location,
  not a signed access capability. Every request still needs fresh authorization.
- A publication cutoff is retained between page requests. It is a timestamp bound,
  **not a cross-transaction database snapshot**. Later commits/backdated timestamps
  can require refresh. No gap-free arrival-delivery claim is made.
- Page sizes are bounded to 1–50. Stored title/source/script/manifest bytes have
  read bounds before allocation. The HTTP adapter must additionally enforce its
  response-byte limit and may use a smaller page size.
- Consumption DTOs expose source text, title, published script and asset IDs only.
  They exclude keys, provider/credential settings, work IDs, authority references
  and provenance payloads. Asset delivery must separately check current access and
  deletion before minting URLs; this read seam does not create asset capabilities.

## Tests written, pending execution

Database tests use the existing explicit `PROSODY_TEST_DATABASE_URL` disposable
fixture helper. Every database case has a bounded context; committed fixtures are
scoped to newly allocated workspaces and cleaned by existing owned-row cleanup.
Cases cover stable tie ordering, pagination through a new transaction, publication
cutoff and refresh, cross-workspace item denial, scoped cursors, source/script
preservation including embedded NUL, absence of private fields, tombstoned items
and feeds, and ambiguous multiple-rendition rejection. Pure tests reject malformed,
oversized, duplicate-field, null, unknown-field and cross-scope cursors before SQL.

The fixtures seed ready metadata directly, deliberately testing read behavior,
not media validation, publication authority, native AMOS sessions or effect
finality. Heavy checks were not run because no serialized slot was granted.

## Remaining integration decisions

Current schema does not carry an agent display name or current `canManage`
authorization. Neither value is fabricated in these projections; the host must
obtain them through supported current-authority/attribution seams. Mixed-manifest
source ingestion is not yet represented by the existing inline-text source shape;
the projection decodes that existing immutable text record and fails closed on
incompatible stored source. Future regeneration requires an explicit current-
rendition decision/schema, not a silent change in pagination behavior.

## 2026-11-01 review remediation

Independent review found that omitted cursor fields could inherit the request's
prepopulated version/workspace/feed. The decoder now requires the complete exact
field inventory for each cursor type, and provided briefing cursors decode into
a zero value before scope checks. New cases omit each of the three feed fields
and seven briefing fields, checking both prepopulated decoder rejection and public
read rejection before any SQL.

Ready-list rows now use `FOR SHARE OF s`, matching detail reads' source lifecycle
fence. A bounded two-connection PostgreSQL test observes the writer's actual lock
wait, releases the read transaction, waits for tombstone commit and verifies a
fresh page excludes the item. It cancels/drains the worker before fixture cleanup.
This protects returned rows only while the transaction remains held; it makes no
claim about response delivery after transaction or authority release.

Parent reports the preceding PostgreSQL tests passed before this remediation.
These new changes are formatted but not tested by this worker: the parent holds
the serialized Flutter slot and will schedule database re-verification afterward.

## 2026-11-01 Activity source read seam

Added `internal/store/activity.go` and `activity_test.go` without modifying the
existing playback reads. `ReadActivity` accepts the existing restricted
`ReadTransaction`, current workspace/feed, cursor and page size (1–50). It returns
only item ID, original title/text, preparation status and acceptance time for
processing/failed/ready submissions. Tombstoned submissions and feeds are denied;
workspace and feed constrain every query. Ready playback/rendition details remain
in the ready-briefing seam. The schema has no failure reason, so this projection
does not fabricate one; the RFC's actionable failure reason remains a domain gap.

Ordering is oldest-first `(accepted_at, item ID)`. Each cursor requires exactly
version/workspace/feed/cutoff/acceptance-time/item, rejects duplicates, unknown or
missing fields and nulls, and binds to the requested workspace/feed. It decodes
into a fresh value. The acceptance cutoff is a pagination bound, not a database
snapshot across requests; late commits, backdated arrivals and changed preparation
status require refresh. Returned source rows and the feed are share-locked until
the caller transaction closes. Current authority remains a caller obligation;
this is not an HTTP route, bearer API or native-session qualification.

Authored tests exercise all three statuses, exact source/title retention including
NUL, tied timestamps across pages, same-workspace cross-feed isolation, other
workspace isolation, deleted feed/source exclusion, secret-free DTOs, each missing
cursor field (including a prepopulated decoder destination), malformed cursors,
and cross-scope cursors. PostgreSQL fixtures deliberately seed lifecycle status;
they do not qualify processing or publication. Only gofmt ran for this slice.
Builds/tests were not run: the most recent heavy-capacity admission was refused,
and the parent explicitly prohibited another capacity sample or execution. The
prior ready-read remediation also remains pending re-verification after that
refusal; no build lease or PostgreSQL fixture was started by that attempt.
