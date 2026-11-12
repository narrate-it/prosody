# Path to the usable Prosody service

Status at 2026-11-11 (Claude Code takeover). This separates the delivered preview
from the authenticated submit -> generate -> listen service. It selects no spend,
deployment, signing or upload by itself; each effect keeps its own gate.

## Verified now (Track B)

- `infra/ses` corrected IaC tests: 21 named PASS / 4 roots, vet 0 (hermetic Pulumi
  mocks; no AWS effect). Explicit us-west-2 + account allowlist revision executed.
- `backend/authhost` company18 composition: first compile + tests PASS, 7 named /
  3 roots, vet 0, against exact local AMOS `845a8042` (go list -m dir confirmed).
  Composition library only; not installed or deployed.
- AWS login present (account 394824061124) but ROOT credentials; scoped IAM needed.

## Critical external blocker (cannot be bypassed)

AMOS is landing primitives incrementally (main `df9a9877`, #555/#557/#558-560):
queue-write witness `ContinueHeldWithWitnessTxV1`/`ConsumeTxV1`, optional seven-table
queue-COMMIT migration `ApplicationWorkQueueCommitsV1`, internal checkpoint owner
`NewRecordCheckpointQueueCommitOwnerV1` (internal, not importable). These are
provisional storage facts, NOT native source/financial authority or F/COMMIT proof.

Still MISSING (per owner handoff comments 6745223644 / 6747883387): the complete
public B/C handoff — native queue/witness consumption, next-stage/Worker Resolve,
financial/provider work, publication, and current published human feed/media
composition. Native application-work Resolve remains unsupported. Do NOT migrate the
pinned adapter, use private AMOS authority tables, or add a second scheduler /
application-domain feed/provider logic to fill the bridge. Continue coordination on
PR #514; the working briefing pipeline waits on this interface.

## Remaining steps, in dependency order

1. AMOS complete public worker + published-feed/media handoff (external; above).
2. Compose Prosody application operations/endpoints against that exact revision:
   sole-worker dispatch, current authority/fence checks, app-declared checkpoint/
   completion writes, effects, committed receipts, same-identity UNKNOWN reconcile.
3. Backend deployment (partly parallel; needs custody + spend approval):
   - Actual company18 `InstallFresh` with distinct owner/founder/label principals;
     runtime DB/roles/keys/TLS custody; 32-byte directory cursor key.
   - SES: scoped IAM (replace root), domain identity + DKIM via owning Cloudflare
     IaC, sandbox -> production sending access, verified-event custody.
   - DNS `api.prosody.chat` (currently NXDOMAIN) via Cloudflare IaC.
   - Linode Fremont hosting within the approved total $50/month; reviewed private
     Nanos image, boot/readiness/backup qualification.
4. Native sign-in activation: release configuration bound to the deployed issuer
   origin/audience; real iPhone browser-return, expiry and revocation acceptance.
5. Real BYOK OpenRouter behavior: budgeted rewrite/synthesis, usable output,
   artifact durability, native accounting/reconciliation.
6. Joint authorized submit-to-listen journey + on-device acceptance (player,
   lock-screen queue, interruption, progress, private media, team settings).
7. Full-service beta release: separate product/signing/account/upload gates from
   the preview path; Apple processing + tester installation verified.

## Preview path (Track A, independent)

The approved `TestFlight preview upload` workflow delivers current functionality
with sign-in DISABLED and sample feeds. It does not activate the backend. Build 43
dispatched from main `d270866` (run 43102956670). A preview install never qualifies
the unfinished backend/provider/device journey above.
