# Ready-notification payload preparation

RFC 0001 §4.4 requires per-feed opt-in, OS permission, playable-only notification
and at most one notification per person/item publication. Default lock-screen
content must omit private briefing content. This codec implements only the
payload part; no push registration, delivery or notification-opening route is
activated.

`internal/notifications.ReadyPayload(ReadyTarget)` returns a fresh byte slice or
an error with no payload. The local version-one profile contains only:

- `aps.alert`: the fixed generic text `A new briefing is ready.`
- `prosody`: `version:1`, `kind:"ready"`, and feed/item/rendition IDs.

There are no caller-supplied titles, source/script text, agent labels, person or
workspace IDs, delivery tokens, URLs, images, sound/badge changes or background
execution flags. Routing IDs are plaintext metadata visible to the push provider;
they are not anonymous, secret capabilities or proof of access. Feed IDs follow
the existing canonical UUIDv7 contract; item and rendition IDs are canonical
nonzero UUID shapes, without imposing a new UUIDv7 requirement on existing data.
No relationship between those IDs is established by this syntax check.

Apple defines the `aps` alert form and 4096-byte regular-push ceiling in
[Generating a remote notification](https://developer.apple.com/documentation/usernotifications/generating-a-remote-notification).
This codec stays within that ceiling; local JSON/schema checks are not Apple or
device acceptance. The local profile is not an AMOS registration or native codec
receipt and does not freeze a new HTTP notification-opening API.

## Required composition

Only a qualified dispatcher consuming the committed ready-publication intent
may select the target. It must bind current person/workspace/feed/item/rendition,
current server-side opt-in, a currently enrolled recipient/device, tombstones,
and the native job fence/current authority. iOS independently enforces OS
permission at presentation; a stored client permission flag cannot prove current
OS state or delivery. Staging is atomic with publication through Prosody's
mandatory StageReady policy. StageReady is application policy, not a universal
framework effect. Existing native scheduling/journaling must own per-person/item
publication deduplication and uncertain-send reconciliation; identical payload
bytes are not deduplication or delivery proof.

A notification tap must treat the IDs as untrusted hints, discard/reject invalid
or unsupported profiles, and reauthenticate/recheck current item and asset access.
It must not trust an old notification as current authority or activate playback
of stale cached private content. Device enrollment, APNs credentials/transport,
native registration and permission/tap behavior remain unimplemented/unqualified.
No new scheduler, private authority lookup or permissive callback is introduced.

## Local verification

Ordinary and race checks each execute 39 named records across four roots with
zero failures/skips (overlap, not additive). Golden/closed-shape checks cover the
generic alert, exact routing fields, size, deterministic encoding and independent
buffers; negative cases include private text/injection/URLs, malformed IDs and
wrong feed version/variant. Existing v4 item/rendition controls remain valid.

A compiled validation-guard omission fails the private-text leaf, two additional
feed version/variant leaves and their parent; its valid golden control passes.
The initial supervisor expected only two failed records and FAILED because its
selector also matched the two extra feed cases. An exact event audit verified all
four intended negative failures and the one control without rerunning or erasing
that harness failure. No live Apple request, provider credentials, device dialog,
DB/native receipt or notification delivery ran. Required repository hook,
independent/hosted review and integrated acceptance remain separate gates.
