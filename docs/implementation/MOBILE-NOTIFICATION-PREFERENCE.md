# Mobile notification preference

The settings model and API already exposed an explicit per-person, per-feed
notification preference, but SettingsScreen offered no control and waited for
provider catalogs before displaying settings. The screen now exposes an arrival
notification switch independently of provider-management rights and catalogs.
The initial setting comes from the explicit settings boolean; no local true
fallback or OS permission is inferred. A missing/unconfirmed value is disabled
and labeled loading or failed, distinct from a confirmed false preference.

A change sends one PUT using the captured API/feed. The prior confirmed value
stays visible while saving and all mutation controls are disabled. One settings
readback confirms the current preference after success or an uncertain write
failure; the write is never retried automatically. A 401/403 stops readback,
clears the preference and key editor, and requires an explicit refresh. Failed
readback leaves the preference unknown until a manual read; raw transport and
readback diagnostics are not shown. Reconciliation can confirm either true or
false without claiming that a failed write succeeded.

Feed, API, permission and feed-kind changes invalidate prior operations. Old
rendered callbacks, pending loads, writes and readbacks cannot populate or dispatch
into a new target. Keys and model selections reset with the target; callbacks do
not touch disposed UI. The existing AccessSession interface has no principal or
session-revision event. The host must unmount settings or supply a fresh API on
logout/principal changes; hidden principal mutation within the same API instance
cannot be inferred by this widget. No authentication implementation is added.

The preview adapter stores this preference in memory separately for each feed
and instance, defaults off, resets it at sign-out and clears it at close. The UI
explicitly labels it local-only and says device notifications are not sent.
Provider settings remain read-only in preview; unsupported/malformed preference
writes are rejected without changing state.

The production copy states that delivery also requires device notification
permission. This change does not request OS permission, register APNs, deliver a
notification or qualify native authority/routing. The existing settings and
notifications routes remain proposed pending the native service composition.

Source was prepared from main `b629600f47e48943055ee5772ea6582d3459f9b5` in the
isolated notification-preference lifecycle. Hermetic source covers non-admin
access, catalog failure independence, single-flight/confirmed state, lost-write
response reconciliation, unknown readback/manual refresh, authority denial,
feed/API switching, stale callbacks/loads, unmount, and preview isolation/reset.

## Executed mobile verification — 2026-11-04 UTC

The coordinator verified the exact product/test bytes in staged tree
`7bfb9d98cb2ccb411de8cd75399694642755499e` using the installed current Flutter
SDK. This receipt update changes documentation only. Offline dependency resolution
passed and left the tracked dependency manifest and lockfile unchanged.

Earlier checks exposed two analyzer infos: an unbraced multiline save guard and
a custom mounted/staleness helper that the context analyzer could not infer.
Braces and an explicit mounted/revision guard fixed those without changing the
intended behavior. The first focused run then passed 22 cases and failed two
401/403 cases because the harness selected both the outer list and a nested text
field scrollable. Explicit outer-list scrolling fixed the harness. The next full
suite passed 127 cases and failed two existing settings cases whose controls were
not mounted in the taller lazy list. Those tests now scroll to each control/error
and require all three disabled dropdowns before checking their handlers, keeping
the revision, no-write, disabled and single-flight assertions meaningful.

Final results on the exact tree above: Flutter analysis passed with no issues;
27 named focused tests passed, zero failures/skips; the complete Flutter suite
passed 129 named tests, zero failures/skips. Dart formatting and whitespace checks
also passed. Required Go lint/unit commit gates are separate from this Flutter
receipt; their actual results belong in the PR verification record.

Fresh capacity admission passed: load 6.09, interval CPU idle 47.69/34.17%
(average 40.93%), normal memory pressure, zero swap churn and no thermal warning.
Checks held one owned build lease, which was confirmed released afterward.
These are hermetic mobile results; OS permission, APNs registration/delivery,
native service authority and physical-device behavior remain unqualified. No
provider calls, fixture creation, signing, device build, installation, deployment
or spend occurred.
