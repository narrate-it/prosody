# Deletion cleanup preparation

Source assessment: `b629600f47e48943055ee5772ea6582d3459f9b5`.
This document identifies the missing domain slice before any cleanup-authorizing
API or SQL is added. It does not admit physical deletion or a new scheduler.

## Existing behavior

RFC 0001 requires tombstones to deny new item/asset access, cancel publication,
and schedule removal of original and derived assets while retaining retry
identity. `store.Tombstone` changes the source status and deleted timestamp,
returning the item ID to a mandatory same-transaction cleanup callback. It
retains source, key and digest; the caller owns authority, rollback and commit.
Production composition must stage cleanup through the qualified native durable
job path; no supported cleanup registration or consumer is composed here.

`store.Publish` locks the processing source for update and referenced live assets
for share, then inserts a rendition and changes source status to ready. A source
tombstone conflicts with publication on that source row. Asset identity is the
workspace/asset pair; a SHA-256 value does not establish ownership. Published
segments contain a playback `assetId` plus `imageIds`. Assets may be referenced
by multiple items in the same workspace.

The ready reader validates exact persisted text source and published manifest
with `scanReadyBriefing`, including source workspace/feed, segment UUIDs, kinds, image
and segment bounds and narration duration. Cleanup must reuse that parsing
contract rather than inventing a second permissive manifest decoder. The
current `store.Accept` input/persisted source shape is `TextSubmission`; the separate mixed-content
`content.Manifest` type is not persisted through `store.Accept`.

## Missing proof and race boundary

The current asset table has metadata and a deleted timestamp, but no original
item/work/upload owner. Renditions store manifest bytes with no reverse asset
reference relation. The schema permits more than one rendition per source;
ordinary ready reads reject that ambiguity. An inventory of one rendition,
a bounded feed page or a caller-provided list of peer manifests cannot prove
that no other live source references an asset. Unpublished worker/upload
artifacts cannot be traced completely from a published manifest either.

A safe cleanup transaction needs all of the following domain inputs and writer
rules, reviewed together before a migration or callable cleanup API:

1. A retained exact workspace/item tombstone and original work identity, with
   the host holding current cleanup authority and the native job fence.
2. Immutable upload/work provenance for originals and derived artifacts,
   including unpublished/failed artifacts; shared assets retain separate
   references. Caller IDs, digests or an `exclusive` boolean cannot supply this
   evidence.
3. A complete, indexed reference relation populated by every admitted source,
   publication and relevant artifact writer. Each entry binds exact workspace,
   asset and source/rendition identity. Existing data needs a bounded verified
   backfill or an explicit unavailable profile; missing entries cannot mean
   unreferenced.
4. One reviewed lock order and writer protocol that prevents references from
   appearing between the last-reference check and asset tombstoning. Publication
   already uses asset share locks, but source admission and upload/finalization
   must participate too. Merely checking a count does not establish this fence.
5. A finite candidate-page and retained cursor contract over the complete
   relation. A per-page absence is never a global no-reference result. Any
   malformed, truncated or ambiguous source/manifest fails closed.
6. Durable cleanup intent and unknown-outcome reconciliation through the existing
   scheduler. Physical removal follows authorized asset tombstoning and exact
   object identity; private inventory output is not an effect capability.

Only after these slots and migration/backfill behavior are accepted can domain
SQL test current tombstones, cross-workspace isolation, shared references,
concurrent reference creation, rollback and lost completion meaningfully.
No native tables or replacement scheduler are needed or proposed here.

## Bounded safe preparation

A pure reference inventory may report a deduplicated, deterministic, bounded set
of workspace/asset IDs from exact validated source and published manifest bytes.
Such output would describe observed references only. It must not label assets
exclusive, deleteable or unreferenced, subtract a purported complete peer set,
construct native receipts, or call `media.Files.Delete`.

No standalone inventory API is added: the existing ready parser and presentation
resource closure already cover observation. Duplicating them would not supply
missing ownership or reference-completeness proof. No cleanup implementation,
migration or physical deletion is added. No deletion-specific Go tests, SQL
fixture or native acceptance ran; the one-time database approval remains consumed.
PR #18 lint, hook and 12-package Go results cover its preference work. Separately,
PR #19's required repository hook passed at `2fe874d2`: zero lint issues and all
12 Go packages passed with the database URL absent. That documentation commit
followed fresh capacity admission and its lease was released; the earlier hold
is historical. Its hook results do not validate cleanup ownership or deletion
behavior.


## Source anchors and required negative tests

These line numbers refer to the assessed revision above:

- `internal/store/publication.go:145`: source tombstone and cleanup callback.
- `internal/store/publication.go:35`: source update lock, shared asset locks and
  rendition publication.
- `internal/store/store.go:89`: text-only admission and immutable source storage.
- `internal/store/reads.go:314`: `scanReadyBriefing` validates persisted source,
  manifest and the exactly-one-rendition rule.
- `internal/presentation/presentation.go:175`: existing briefing resource closure
  and bounded projection; it supplies no cleanup eligibility.
- `internal/store/migrations/001_domain.sql:39`: rendition composite primary key
  admits multiple renditions per item; manifest bytes have no reference index.
- `internal/store/migrations/001_domain.sql:68`: asset workspace/ID key and metadata
  without item/work/upload provenance.
- `internal/media/files.go:178`: physical deletion exists but has no domain proof;
  no new caller is admitted by this document.

Before enabling cleanup, real domain transaction tests must reject a missing or
live source, wrong workspace, mismatched work identity, duplicate/ambiguous
renditions, malformed/truncated manifests and incomplete reference backfill.
An asset shared by another live item, including another feed in the same
workspace, must remain available after one source is tombstoned. Exact concurrent
publication/source-reference creation must either retain the asset or reject the
new reference under the same reviewed locking protocol; a stale zero-reference
read must never authorize deletion. Tests must cover unpublished original and
derived artifacts, multiple candidates/pages, callback failure and rollback,
cancellation, duplicate cleanup delivery and unknown commit reconciliation.
Native fence/current-authority revocation and physical-object outcomes require
separate native/provider acceptance; ordinary transaction tests cannot fabricate
those receipts. Restore and backup retention need their own accepted lifecycle.
