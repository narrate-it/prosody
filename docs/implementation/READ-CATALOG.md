# LOCAL consumption read SQL catalog

`ConsumptionReadCatalog` inventories six unique SQL statements used by seven
read callsites at source main `9b5a9224d87289e9fa074cb077271a9a97c29987`.
Only existing literals in `reads.go` and `activity.go` moved to constants in
`read_catalog.go`; query bytes, parameter expressions/order, scans, locking,
pagination, cursor validation and public read behavior are unchanged.

This is descriptive LOCAL preparation for future human-read registration, not
an AMOS API, admitted SQL, wrapper, route mount or grant. No execution callback,
SQL pool, authority constructor, native table query or deletion/publication/
checkpoint statement is supplied. Both functions return fresh metadata slices;
mutating returned arguments, columns, route statement IDs or hashes cannot
change subsequent catalog calls. SQL constants themselves are immutable.

## Exact UTF-8 SQL SHA256

The full SQL is the named literal in `internal/store/read_catalog.go`, including
all bytea bounds, count subqueries, share locks, ordering and whitespace.

| ID | Constant | SHA256 |
| --- | --- | --- |
| feeds | readFeedsSQL | b7c2e5f1b89accc26294ee5810415f967d9ed23804f0206c6a30cf7a516fce00 |
| clock | readClockSQL | 92da62dc3b3f7b341c4a327909755f3e26e99a747f830cef79e3a582a80b7556 |
| readable-feed | readableFeedSQL | 7c5b6875f75832209bd0ae01c804cb14f7b884152bbb842f397e8b01f92d4108 |
| activity | readActivitySQL | 6abe76f4608c74eef17898aaef4c08439dd65e3728aeb1c2e56127c2e9ebe548 |
| ready-list | readReadyBriefingsSQL | 90fa33c94380af96600f8451ce3b2b0fbe80a265a10030ef699ea58aa2932b9d |
| ready-detail | readReadyBriefingSQL | 11ced9a211a89c477221fc895871bebdb9754fee9bde95bc4bddc0150b13b36e |

Changing even whitespace invalidates the catalog review. Hash equality is
inventory consistency, not semantic SQL safety or current authority proof.
No regex SQL admission or automatic statement discovery is implemented.

## Parameters, results and call budgets

All statements have local phase `human-read`. `FOR SHARE` statements need the
caller's suitable transaction; that phase does not mean PostgreSQL READ ONLY.
The zero-argument exact clock statement alone has workspace parameter 0.
Every data statement binds workspace at $1. Native person/session/grant/fence
identity slots are empty; native control queries and current authority remain
AMOS-owned. These IDs/hashes cannot authorize a read.

| Statement | Arguments in exact ordinal order | Source rows | Proposed initial rows |
| --- | --- | --- | --- |
| feeds | workspace UUID; after-feed UUID cursor; lookahead integer | 51 | 11 |
| clock | none | 1 | 1 |
| readable-feed | workspace UUID; resource feed UUID | 1 | 1 |
| activity | workspace UUID; resource feed UUID; cutoff timestamptz; cursor-present boolean; accepted-at timestamptz; after-item UUID cursor; lookahead integer | 51 | 11 |
| ready-list | workspace UUID; resource feed UUID; cutoff timestamptz; cursor-present boolean; published-at timestamptz; after-item UUID cursor; after-rendition UUID cursor; lookahead integer | 51 | 11 |
| ready-detail | workspace UUID; resource feed UUID; resource item UUID | 1 | 1 |

UUID strings have a 36-byte representation. Timestamp/boolean/integer arguments
are typed Go/SQL values, with no invented string codec. Cursor identities are
locations, never resource authority selectors. Clock/cursor cutoffs are not
cross-request snapshot or gap-free arrival guarantees. Native registration
must still review its exact typed codecs and parameter binding semantics.

The existing source accepts page limit 1..50 and sends limit+1 (maximum51) to
SQL. Future initial human routes must enforce 1..10 and SQL limit<=11 with at
most11 returned application rows. This catalog changes neither source limit nor
runtime behavior and does not silently truncate an oversized initial response.
One lookahead row remains legitimate pagination, not an overflow workaround.

`ConsumptionReadBudgets` records the proposed application-query call ceilings:
feeds1 (`feeds`), activity3 (`readable-feed`, `clock`, `activity`), ready-list3
(`readable-feed`, `clock`, `ready-list`), detail2 (`readable-feed`, `ready-detail`).
Each listed statement executes at most once per route call. Native authority/
resource control queries require separate finite, reviewed budgets.

Column metadata names exact scan order and SQL types. SQL CASE bounds source
bytes to65536, script bytes to256000 and manifest bytes to262144; oversized
values become NULL and existing codecs reject them. Ready rows therefore
carry up to583680 bounded payload bytes per row, excluding scalar/transport
encodings: 51 source rows can exceed29MB, while 11 initial rows carry up to
6420480 bytes. Activity source bytes cap65536; feed title bytes cap4096.
Column metadata distinguishes SQL bounds, typed scalars and source validation
(e.g. personal/team, processing/failed/ready, exactly one rendition).
The proposed initial cumulative raw application payload budget is8MiB and
encoded JSON budget1MiB; neither is enforced by these descriptive functions.

Future native result/route manifests must separately cover every returned
workspace/feed/item/rendition/asset resource and required parent edge, exact
published manifest membership, bounded array selectors, 256 unique native
resources and32768 occurrence visits. Manifest-local segment IDs are not native
resources. SQL matching alone cannot discharge those closure/locking checks.
No audience, human bearer wire, media URL template or player compatibility is
selected here. No native currentness, least-privilege or semantic SQL admission
claim follows from this catalog.

## Verification scope

Hermetic tests freeze all six full SQL hashes, parse both source files to check
seven callsites and exact argument expressions/order, and parse constant literal
bindings to reject drift. Additional cases check IDs, workspace-zero exception,
slot classes, source51/initial11 and page50/10 separation, exact route call
budgets, payload bounds and returned metadata isolation. They do not execute
SQL or qualify provider/native/service behavior. Existing database tests require
an explicitly owned disposable fixture (`PROSODY_TEST_DATABASE_URL`); without
one they skip and do not establish database behavior. No fixture was provisioned
and no native route or service was enabled for this task.

Executed Go 1.27.0 / GOMAXPROCS=2, `go test -json -p 1 -count=1
-timeout 60s ./internal/store`: exit0, 49 passed and20 skipped test/subtest
records. All four catalog tests plus six hash subtests passed (10 pass records).
The database-dependent cases skipped without an owned fixture. A source-only
expansion check also reproduced both original source files byte-for-byte.
Preserved full commit gates are run before the initial source commit.

Capacity admission: one-minute load3.71; interval CPU idle81.68/83.61%
(average82.645%), normal pressure, zero swap churn, no thermal/performance
warnings. The exact owned build lease covers only these bounded checks/gates
and is released afterward. No native/provider/database qualification is inferred.
