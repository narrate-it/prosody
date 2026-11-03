# Local consumption presentation

`internal/presentation` bridges existing store DTOs to the current Flutter JSON
shape. This is a pure local shaping library, not an HTTP handler, native AMOS
integration, authority check, media grant or acceptance receipt. Store DTOs and
Flutter parsing are unchanged. No media path contract is invented.

`MarshalBriefings` requires one workspace/feed scope, a ready briefing page,
explicit attribution for every exact item/rendition, separately caller-held
published asset metadata and already-bound media links. Attribution must be a
nonblank, valid UTF-8 display label of at most 256 bytes without controls. The
host supplies this label from legitimate public attribution; private authority
references and raw person IDs are not a label source or a fallback.

Each asset mapping binds workspace, feed, item, rendition and asset ID, plus
SHA-256, MIME and byte size. Duplicate, missing, foreign, stale-rendition,
unexpected and metadata-mismatched inputs deny the whole response. Metadata must
come from the caller's published snapshot, not be copied from an untrusted link
claim. Digest comparison here checks the two data inputs only: it does not read
or verify an object. A stale snapshot supplied consistently cannot be detected
here. Narration/audio use audio/mpeg, video uses video/mp4, images use image/png
or image/jpeg, matching current publication validation.

Links must be absolute HTTPS at exactly the configured origin (including port),
without user info, query (even empty), or fragment. The origin has no path beyond
an optional slash. The helper does not create paths, fetch URLs, follow redirects
or mint tokens. A required pure `LinkRenderer` port (`BriefingInputs.ExpectedURL`) receives each
exact workspace/feed/item/rendition/asset identity. Every supplied link must equal
the renderer's expected raw URL exactly, and that expected URL must pass the same
UTF-8/origin/no-credentials/query/fragment guards. A different endpoint at the same
origin is denied. There is no default renderer or native route constructor; no
approved renderer/template currently exists in this package. Renderer errors deny
the entire response with the fixed safe error and no raw diagnostic.

The future host must bind the renderer to its exact admitted template and current
metadata. Returning a matching string is a data consistency check, never approval
or proof of authority. The helper does not discover or prove a route safe. The host must ensure they contain no credentials or
bearer capabilities and must enforce a no-redirect delivery policy.

Output retains every assetId/imageIds selector and adds url/images in the same
order and cardinality regardless of link-map order. It preserves the nonzero store
publishedAt timestamp (including its JSON-representable offset/precision) as an
additive field without freezing a new native DTO contract. It adds the
explicit agent label and preserves title, source and script byte-for-byte. Native
parent IDs remain in the input closure binding; the output preserves Flutter's
existing wire shape. `MarshalFeeds` requires exactly one explicit workspace/feed
permission record per feed, including false; kind and role never infer canManage.
Neither the flag nor a private asset selector authorizes any effect.

Local whole-response targets are ten items/feeds, 100 segments per item,
20 images per segment, 256 unique native resources including workspace/feed/
item/rendition/assets, and 1 MiB after JSON escaping. Manifest-local segment IDs
are not native resources. A separate 32,768-visit limit counts each supplied
attribution's four identity references, each metadata/link's five references,
page scope and item parent references, each segment identifier, and each asset
reference's five identity/parent visits, including repetitions. Input bounds and
duplicate checks reject excess/lookahead records without truncation. Native
AMOS control-query budgets remain the host's responsibility.

The API returns fresh JSON bytes and a fixed error with no private input details;
no partially shaped response is returned on failure. Inputs are not mutated or
retained. The host must bound decoding/allocation before calling this package;
this library receives already-materialized Go values.

Before real service composition, the future native HumanRead host must hold
current authority and closure (including independently declared native parents),
resolve legitimate attribution and native canManage permission, bind all media
metadata/links in the same current read scope, recheck revocation/deletion and
rendition freshness, enforce media resource grants and MIME, recompute delivered
body length/digest and retain exact delivery identity. It must separately qualify
serialization size, native routing, iOS media behavior and currentness through
response delivery. These pure helpers do not discharge those obligations.

## Candidate verification

Source prepared from main `9b5a9224d87289e9fa074cb077271a9a97c29987` in the
separate consumption-presentation lifecycle. Hermetic tests cover Flutter fields,
exact content/resource preservation, independent output bytes, explicit false
permissions, attribution failures, missing/foreign/stale/duplicate/unexpected
bindings, digest/MIME/size mismatch, origin and token-bearing URL rejection,
JSON escaping size, complete native graph boundary and repeated-reference budget.
Executed `go test -json -p 1 -timeout 90s ./internal/presentation`: all seven
top-level tests passed (42 named test/subtest pass records), exit 0. Independent
source review found one invalid-UTF-8 URL fidelity issue; the shared origin/link
guard and regression fixed it, and the exact corrected source passed independent
delta re-review before execution. That receipt applies to the initial candidate; the route-renderer/timestamp
follow-up below requires fresh tests and preserved commit gates.

Capacity admitted: load 6.06/7.18/7.00, interval idle 19.11/45.14%
(average 32.125%), normal pressure, zero swap churn and no thermal/performance
warnings. The owned build lease was released after the bounded package tests. No provider calls,
DB provisioning, native changes, signing, installation or deployment.


## Route-renderer and publication-time follow-up

The local helper now requires the future host's exact route renderer rather than
accepting any endpoint at the configured origin. No route template or authority
API is supplied here. It also preserves PublishedAt from the existing store DTO
and rejects a zero timestamp, consistent with the store read boundary; timestamps
that JSON cannot represent also fail with the fixed error. Added regression cases
for a wrong endpoint at the correct origin, renderer failure without private error
text/body, malformed expected links, missing renderer, timestamp preservation, and
image/link reordering without identity, order or cardinality loss. Executed follow-up: all eight top-level tests pass (50 named test/subtest pass
records). An isolated Go overlay control removing timestamp emission and exact
URL equality caused both corresponding regressions to fail; the actual reviewed
source was never mutated. The full corrected package then passed. The preserved
lint/full-unit hook runs for this checkpoint. Capacity admitted at load
3.79/3.72/3.74, interval idle 47.42/47.63%, normal pressure, zero swap churn
and no thermal/performance warnings. Agent attribution still requires an explicit
legitimate label; the source-of-attribution choice remains a host decision.


## Hosted-review duplicate-item coverage follow-up

The old duplicate-item fixture duplicated attribution too, so it failed before
the item-loop guard. Renamed that case to duplicate attribution and added a
same-item/distinct-rendition fixture with separate correctly bound attribution,
metadata and links. It denies the whole response at the duplicate-item guard.
Independent test-delta source review is clear. An isolated Go overlay removing
only that guard made the new test fail; the production source remained unchanged.
All nine top-level presentation tests passed (51 named test/subtest pass records).
The preserved lint/full-unit hook runs for this test-only checkpoint.

Capacity: load 13.87/13.16/12.34 (warning), interval idle 32.18/41.26%, normal
pressure, zero swap churn and no thermal/performance warning. One owned lease
bounds verification; no provider/native/DB/deployment acceptance was added.
