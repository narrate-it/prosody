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
or mint tokens. Paths are already-bound opaque host input, not discovered or
proved safe by this helper. The host must ensure they contain no credentials or
bearer capabilities and must enforce a no-redirect delivery policy.

Output retains every assetId/imageIds selector and adds url/images; it adds the
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
delta re-review before execution. Required hook gates are pending.

Capacity admitted: load 6.06/7.18/7.00, interval idle 19.11/45.14%
(average 32.125%), normal pressure, zero swap churn and no thermal/performance
warnings. The owned build lease was released after the bounded package tests. No provider calls,
DB provisioning, native changes, signing, installation or deployment.
