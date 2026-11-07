# Main-only ASC environment

Go/Pulumi owns a protected host-local command (provider v1.3.0) for the exact
`narrate-it/prosody` repository ID and `prosody-asc` environment. Use the existing
private backend/secrets provider; review `pulumi preview` and the read-only helper
plan (`/usr/bin/python3 -I scripts/configure-asc-environment.py --plan`, run from
the repository root) before apply.
Pulumi preview does not execute the helper or enumerate its GitHub-side changes.
The helper
uses the existing authorized `gh` login without placing its token in IaC inputs.
`--plan` is read-only. Never invoke `--apply` outside the reviewed IaC operation.

Creation enables custom deployment policies before adding one exact `main` branch
rule. A failure between those writes leaves the environment without an admitted
branch. Reconcile actual state after ambiguous failure; do not delete/recreate it.
A pre-existing broader or foreign policy is refused, never silently overwritten.
The helper writes no secret, changes no repository secret, and refuses destroy.
A matching existing profile may be verified without changing it. GitHub repository
admins remain able to change policy; this does not defend against an administrator.

The workflow uses the four owner-supplied environment names: `ASC_API_KEY`,
`ASC_ISSUER_ID`, `ASC_KEY_ID`, and `ASC_TEAM_ID`. These names must exist ONLY in
`prosody-asc` before activation. GitHub resolves repository secrets when an
identically named environment secret is absent, so the workflow mapping alone
cannot enforce this boundary. Verify protected placement and retire all broader
copies after checking authorized consumers. Missing protected values must not be
allowed to fall back to repository or organization copies.

The owner reports entering the original values; metadata confirms all four names
are present in the environment. This does not establish equality or Apple validity:
GitHub cannot reveal stored secret values. The owner subsequently deleted the repository copies. Fresh GitHub metadata
confirms zero repository and accessible organization secrets, all four protected
environment names, and the exact main-only policy. This was an owner action, not
an IaC retirement performed by this helper. The environment-only reconciler does
not retire secrets. Source review/merge and the live Apple check remain separate.
