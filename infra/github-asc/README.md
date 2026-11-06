# Main-only ASC environment

Go/Pulumi owns a protected host-local command (provider v1.3.0) for the exact
`narrate-it/prosody` repository ID and `prosody-asc` environment. Use the existing
private backend/secrets provider; review `pulumi preview` before apply. The helper
uses the existing authorized `gh` login without placing its token in IaC inputs.
`--plan` is read-only. Never invoke `--apply` outside the reviewed IaC operation.

Creation enables custom deployment policies before adding one exact `main` branch
rule. A failure between those writes leaves the environment without an admitted
branch. Reconcile actual state after ambiguous failure; do not delete/recreate it.
A pre-existing broader or foreign policy is refused, never silently overwritten.
The helper writes no secret, changes no repository secret, and refuses destroy.
A matching existing profile may be verified without changing it. GitHub repository
admins remain able to change policy; this does not defend against an administrator.

The workflow maps four environment-only names to the existing process interface:
`PROSODY_ASC_API_KEY`, `PROSODY_ASC_ISSUER_ID`, `PROSODY_ASC_KEY_ID`,
`PROSODY_ASC_TEAM_ID`. Never put these new names in repository-level secrets.
Missing environment values fail preflight rather than falling back to the old
`ASC_*` repository copies. Environment creation alone is not secret migration.

GitHub does not expose stored secret values for copying. An authorized owner must
enter the original values into this environment, then remove the old repository
copies only after checking all authorized consumers. Until that separate migration
is verified, the old copies remain broadly available to repository workflow writers
and the security gate/operational account preflight remain OPEN. No live preflight
or signing/upload is authorized by a successful environment-only apply.
