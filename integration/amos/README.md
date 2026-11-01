# AMOS first-operation adapter (unqualified candidate)

Pinned source: `809e060e64c00160d8751ad8a35b3d76e2436b45`.
This isolated Go module targets the candidate public `ProductionMCPConfig` APIs.
It is separate from the pure domain module so an unmerged AMOS dependency does
not silently become the domain build's supported foundation.

Implemented source:

- Closed text input with strict Unicode, canonical UUIDv7 `/feedId`, authoritative
  JSON key and current consent-selected workspace.
- `ResultAccepted`/202 descriptor, registry-derived mapping digest, explicit CIMD.
- Application-only SQL grants and migration fragment; operator assigns native
  migration boundary and applies it separately.
- Byte-preserving source/receipt persistence and same-transaction AMOS effect
  enqueue. Job payload holds only workspace/item IDs, not content, keys or credentials.
- Required snapshot/admission callback. `CurrentProviderSnapshot` supplies current
  provider configuration references; the host must additionally admit finite
  budgets. A nil callback fails configuration rather than fabricating admission.

AMOS owner reconciliation confirms that the native host lacks a supported public
application-job consumer/current-authority/financial/publication boundary. Its
identity-email worker and durableworker test fixture do not supply one. The native
human issuer also lacks a qualified iOS browser-return and bearer consumption
path; it does not issue refresh or OIDC ID tokens. These are explicit integration
gaps requiring AMOS design, review and qualification. This
module neither constructs a private authority root nor starts a second scheduler.
No complete host, database/client/CIMD acceptance or provider readiness is claimed.

For source qualification, use a verified checkout of the pinned AMOS revision
in a local Go workspace with this module and the repository root. Do not rewrite
AMOS owner worktrees or treat a different main branch as equivalent. The selected
unit tests are source checks; real transactional admission and current authority
need the owned service fixture and independent acceptance.

For this unpublished candidate, the local workspace must also bind the exact
required version to that verified checkout, so dependency graph loading does not
try to fetch the unpublished revision:

```go
replace github.com/ajent-social/amos v0.0.0-20261031083639-809e060e64c0 => ./amos-candidate
```

The relative path above is from the ignored `.local/amos.work` used here. Verify
checkout HEAD equals the complete pinned revision before testing; this replacement
does not permit substitution of another branch or a modified owner worktree.
