# Activity consumption contract (proposed HTTP composition)

`GET /prosody/v1/feeds/{feedId}/activity?cursor=...` projects the caller-authorized
`store.ReadActivity` result. This route is referenced by Flutter but has no
qualified HTTP/current-authority host binding yet. Feed/workspace parameters are
not credentials. No private root, fabricated identity or MCP-as-human adapter.

Response: `items` contains `id`, `title`, `source`, `status` and `acceptedAt`.
Status is processing, failed or ready; tombstones are excluded. acceptedAt is an
RFC3339 timestamp with an explicit timezone. nextCursor is omitted/null at end,
otherwise an opaque nonempty cursor. Source may contain valid Unicode including
NUL; JSON escaping must preserve it. Title may be empty when not supplied.

This is a submission summary, not a playback manifest. Ready summaries need no
rendition, segments, audio URLs or invented agent label. The listening-feed read
path separately resolves the published rendition and authorized media delivery.
Activity performs no progress writes or generation retries. There is no detailed
failure-reason field in the current schema.

Pagination is acceptance-time/item-ID keyset order with a cutoff bound, not a
cross-transaction snapshot. Refresh is required for status changes, late commits
or backdated arrivals. The UI caps manual pagination at 20 pages/500 summaries.
Production host must enforce current person/workspace/feed authority and ensure
source-content access is authorized before serializing the domain result.
