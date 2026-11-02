# PR #1 review disposition

This records review of the initial implementation, not release acceptance.
David authorized independent headless review instead of unavailable CodeRabbit.
Two headless reviewers inspected backend/mobile, fixed findings and cross-reviewed
the changes. CodeRabbit subsequently became available and reviewed head
`6673de74e43b7b317170a25d82972ff28585d8a5` against base
`7acaabd7f4e42a4d3c95933840ca163d45581b99`.

## CodeRabbit findings

| Comment | Disposition |
| --- | --- |
| 4438225820 | Added visible per-screen descriptions for all ten design boards and associated them with each image. |
| 4438225821 | Enabled vertical touch scrolling in the motion study. |
| 4438225825 | Corrected Activity documentation/comments to the dedicated `/activity` projection. |
| 4438225835 | Raised bounded rewrite completion allowance to 8192 tokens; 1100-word rejection and no automatic retry remain. This is not a guarantee that every reasoning model finishes. Host model/budget admission remains required. |
| 4438225862 | Enabled `discarded_futures` and made intentional asynchronous calls explicit. |
| 4438225865 | Maintained exact audio_service 0.18.19 source with guards on both exported Android entry points. Reject missing/null/wrong-type/malformed key events. Kept licensing and source hashes. Android compilation/instrumentation remains pending; see vendored patch record. |
| 4438225869 | Resume failure can fall back to zero for missing/transient/malformed progress; known authorization failures remain fail-closed. Cross-review additionally required preserving denial status before any body processing. |
| 4438225877 | Successful fixture runs delete only their own data after confirmed shutdown, retaining logs. Failed runs retain diagnostics. |
| 4438225881 | Direct process argument arrays avoid shell path parsing; PostgreSQL connection values are escaped separately. |
| 4438225883 | Own the foreground PostgreSQL child before waiting for readiness; stop that exact child after startup failure. |
| 4438225884 | Strip ambient PG settings and explicitly bind the fixture socket/user. |
| Summary: receipt mismatch | Aligned schema/handler/RFC to `itemId`, `workId`, `status: accepted`, and persisted `acceptedAt`, retaining the same receipt on replay. |
| Nitpick: signing comment | Corrected comment to say release signing is not configured. No signing change. |
| Nitpick: refocus configuration | Deferred: this is the owner's explicitly configured per-session cron helper, not an application API. The identifier is routing metadata, not a credential. Preserve the existing reminder until full implementation/acceptance; portable configuration can be a separate change. |

## Verification scope

- Earlier independent review fixes passed root Go lint/tests, disposable PostgreSQL
  tests, pinned AMOS adapter tests, Flutter analysis and 29 Flutter tests followed
  by three targeted handler regressions.
- Review follow-up fixture mock tests pass (four scenarios), and the real fixture
  passes with deliberately conflicting `PGHOSTADDR`, `PGUSER` and `PGSERVICE`.
  The successful fixture's data is absent and its logs remain.
- Updated acceptance receipt/schema and rewrite budget tests pass.
- All 48 Flutter tests pass on the final denial-preserving code, including eight
  streamed 401/403 body/cancellation failures and ten optional-resume scenarios.
  Flutter analysis reports no issues. Independent cross-review closed the
  denial-precedence finding after checking the fix.
- Vendor hashes verify no copied Dart/Darwin source changes. The only copied
  baseline changes are two Android entry-point guards and instrumentation setup.
  New guard/test source is documented separately. No Android SDK compilation or
  device execution is claimed.
- HTML structure verifies ten described boards/forty screen descriptions and
  vertical-scroll CSS. No new browser/device visual acceptance is claimed.

The original local pre-commit hook remains enabled. A temporary package-aware
hook enforces lint and tests when the original multi-directory invocation cannot
handle the staged Go changes. No required check or branch protection is bypassed.
No provider call, signing, deployment, TestFlight or full RFC completion is claimed.


Final follow-up checks: root and pinned nested AMOS golangci-lint both report
zero issues on Go 1.27.0. Nested lint initially exceeded a two-minute dependency
load deadline; the bounded, serialized five-minute run passed. Final Flutter
suite passed all 48 tests and analysis is clean. Independent cross-review closed
all accepted source findings. Vendored upstream whitespace is preserved via a
vendor-only .gitattributes rule so baseline hashes remain meaningful; authored
files continue to pass whitespace checks. No Android compilation/device evidence
is claimed. Original hook remains enabled; final code commit runs the corrected
package-aware lint/test hook, with no gate skipped.
