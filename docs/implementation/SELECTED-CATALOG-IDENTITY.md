# Selected provider catalog identity

Source preparation from main `dae37693b5ee11cd9a70cd353ef70bed0f6a3bef`.
`configuration.Selection.Validate` previously accepted ambiguous selected IDs:
text validity was overwritten by the last matching row, while speech validity
retained any matching row with accepted capabilities. Identical duplicates also
passed. Row order could therefore change a text selection's validity or combine
contradictory descriptions into an accepted speech choice.

Each selected text/speech ID must now resolve to exactly one catalog row. The
existing text-token, dedicated speech modality, listed voice and reviewed MP3
requirements are unchanged. One row can satisfy both roles if it passes both
capability checks. Unselected duplicate IDs retain their existing behavior;
there is no new global catalog uniqueness policy. Failure remains the same
fixed `configuration.ErrInvalid`, with unchanged signatures and wire shapes.

## Caller and composition assessment

`SaveSettings` calls Selection.Validate before its first SQL callback and before
credential mutation. Its existing Go caller found during inspection is the
settings database fixture; no production catalog merger/duplicate resolution
contract was found. The two OpenRouter catalog methods return raw Data without
implementing a merge/duplicate-resolution policy. Flutter's
`providerSelectionError` requires exactly one selected row per role, and its
catalog parsing already rejects duplicate IDs. This Go validation change keeps
selected identity unambiguous; it does not invent a combined-catalog merge.

Synthetic tests cover identical and conflicting selected text/speech rows in
both orders, a duplicated single ID used for both roles, unique separate roles,
one capable row used for both roles, and unselected duplicates. The same-row
fixture supplies both capabilities for a reviewed MP3 ID; it is not evidence
that the real provider currently offers that combined profile.

A separate no-DB SaveSettings port test uses valid IDs, a real nonnil test vault
and complete callback surfaces. Its unique-catalog control must reach the SQL
callback and return controlled ErrUnavailable. Duplicate selected IDs must
return configuration.ErrInvalid with zero SQL callbacks and no snapshot.
This proves pre-SQL rejection only, not an instrumented vault-bypass, real SQL,
transaction, provider or current-authority result. No native capability is added.

## Executed scoped verification and present limits

The coordinator tested exact source tree
`7b22444e4cb63b3ac91e602abc275f629d622ea9` on main
`dae37693b5ee11cd9a70cd353ef70bed0f6a3bef`. A compiled overlay replaced ONLY
production `configuration/settings.go` with that base source, retaining the new
tests. The old-source run was RED: 19 failed test/subtest records including
parents, five passed, no skips, exit 1. Ten negative leaf regressions failed.
Four required positive controls passed: unique distinct roles, one row serving
both roles, unselected duplicate IDs and the SaveSettings unique-catalog SQL
control. The fifth pass was the text/conflicting/forward negative case, which
the old source already rejected; it is not another positive control.

Restored-source scoped configuration/store checks passed all 39 named records
across six top-level tests: exit 0, no failures or skips. The exact owned build
lease was released. These are synthetic catalog and transaction-port results,
not database locking, real SQL/provider behavior, vault-bypass instrumentation,
mobile or native execution. Lint, the full Go unit gate and the required commit
hook are separate from this scoped receipt; their actual results are recorded
in the PR verification record.

No database fixture/run, provider HTTP, real credential, environment discovery,
mobile/device execution, native import/authority, infrastructure or deployment is
part of this slice. Schema, AMOS, pipeline/provider request code and shared
STATUS/WORK records are unchanged. No actual model quality, usage settlement,
mobile or native integration acceptance follows from catalog validation.
