# Mobile activity slice

## Scope and behavior

Added an activity screen reachable through the feed toolbar’s “Briefing activity” action. It displays processing, failed, ready and unavailable states from the selected space’s existing feed-page API. Processing and failed briefings can expose their exact original text for reading when supplied by the service. Empty, loading and request-failure states are separate. Refresh and pagination are explicit user actions; the view neither polls nor retries paid generation.

The feed’s ready-only playback queue is unchanged. Opening activity suspends foreground exposure tracking through the feed’s existing route-visibility mechanism; returning restores it. Existing playback is not replaced by activity navigation.

Owned files:

- `mobile/lib/ui/feed.dart`: navigation and empty-feed activity hint.
- `mobile/lib/ui/activity.dart`: status cards, original reader, refresh and bounded pagination.
- `mobile/test/activity_widget_test.dart`: five mocked widget tests for status/original visibility, request failure versus empty, explicit pagination with duplicate status updates, cursor-cycle rejection and disposal during a request.
- `docs/implementation/worker-mobile.md`: this record.

## Contract limits

Only `GET /prosody/v1/feeds/{feed}/items` is used. This does not establish availability of a separate activity endpoint or complete retained history. Pagination is capped at 20 pages or 500 distinct briefings per refresh and discloses that limit if reached. Loaded statuses may be stale until refreshed. The current DTO has no detailed failure reason, so failure cards state that the reason is unavailable and offer reading/refresh rather than inventing a diagnosis or generation retry. Ready items discovered here are not inserted into an already-loaded playable queue.

## Evidence and remaining checks

Source implementation and mocked widget test source are authored; `dart format` completed on the three assigned Dart files. Analyzer, tests, builds, device acceptance and real backend behavior have not been run for this slice; the parent must grant the serialized verification slot first. No provider calls, credentials, signing, deployment or Git commits are part of this work. The production sign-in gate remains unchanged.
