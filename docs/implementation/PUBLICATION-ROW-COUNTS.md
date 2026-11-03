# Publication affected-row guard extraction

This child change extracts the existing guard from checkpoint PR #7 at
`f596a1ba068f52fa364c498b0729563966441b2a`. Both `publication.go` and
`publication_rows_test.go` were copied verbatim into the isolated
`fix/publication-row-counts` branch from main `359c7504`. PR #7 depends on this
child; its existing worktree, claims and source remain untouched. After this child
lands, PR #7 can rebase and inherit/drop the identical guard patch. There is one
implementation, not a competing publication path.

`store.Publish` requires exactly one affected row for the final rendition insert
and the source-ready update before calling mandatory `StageReady`. A nil result,
zero/multiple rows or unavailable row count rejects without staging readiness.
The SQL literals and their exact call argument expressions are unchanged. Current
source authority, native job fences and same-transaction notification admission
remain caller-owned requirements. These functions never commit or prove commit;
on failure the host must abort/reconcile its transaction as appropriate, including
unknown commit outcomes. A matching count is not a native receipt or authority.

Source comparison verified identical SQL literals/call arguments against the base
and identical extracted files against PR #7. Under an owned lease, the named test
`TestPublicationRequiresExactlyOneAffectedRow` ran against original main source:
all eight negative cases failed and the one/one positive control passed. Restoring
the extracted guard made all nine cases pass. The test uses transaction doubles;
it proves guard sequencing and refusal to stage, not real database rollback,
row locks, affected-row behavior, native atomicity or notification delivery.

No checkpoint code, checkpoint migration or native interface is part of this
child. The checkpoint database/migration/native receipt gates stay closed. No DB
fixture was created/adopted and the test environment explicitly removed
`PROSODY_TEST_DATABASE_URL`. Required lint/unit commit hooks remain enabled; final
independent review and refreshed hosted review are required before merge.
