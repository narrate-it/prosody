# Native sign-out recovery

The mobile client separates local access termination from issuer revocation
acknowledgement. Sign-out clears its in-memory active grant synchronously.
It retains one original token and its captured expiry in a configuration-bound
pending record, using the existing Keychain bridge and protection policy. The
pending binding is JSON `[activeBinding,"pending-revocation.v1"]`; it cannot be
an active four-field configuration binding. No token or response body enters
logs, task records or a filesystem journal. No native bridge API or Keychain
ACL, default, search list or lock policy changes.

Every recovery path persists pending before deleting active storage, including
restore and sign-in after a prior persistence failure. An interruption leaving
both records therefore cannot restore active access: restore observes pending,
reestablishes its storage and deletes active data, then returns signed out.
Restore performs no remote revocation retry. A user-initiated sign-out retry,
or explicit sign-in, can perform one revoke of the same original target. A
failed recovery prevents a new browser flow. Concurrent sign-outs share one
attempt; overlapping recovery requests share the same pending revoke future.
Generation checks continue to reject late session operations.

Pending is deleted only after active deletion and an observed revoke200/204
acknowledgement. Neither conservative local expiry nor a changed device clock
proves issuer expiry; pending is retained even after nominal expiry until that
acknowledgement. Its original expiry is informational recovery data, not new
authority. Storage is bounded to one pending slot per exact configuration and
a bounded token, with no growing queue or background retry. A corrupt pending
record refuses restoration with a fixed safe diagnostic; it is not discarded
in order to recover access.

Storage failures remain failures. If the first pending write fails, durable
logout intent has not been established: memory access stays denied and explicit
recovery can retry the write, but a successful logout or interruption-safe
persistence cannot be claimed. Subsequent restore/sign-in recovery must persist
that memory-only target before active deletion. A lost/failed revoke response
retains pending; this does not assert remote cancellation, rollback, no-effect,
refund or server-wide revocation. Existing bounded response/header handling is
unchanged.

## Verification and remaining gates

Tests use a synthetic Keychain map and mocked issuer responses. Recreated client
objects and injected storage failures model ordering and interruption boundaries;
they do not establish real process termination, physical Keychain durability,
server revocation or browser/device behavior. Existing native-session fixture
reads now respect the exact binding so the active and pending slots remain
distinct. No real signing key or device credential is used.

Original source reproduced the lost-target retry assertion while a normal
successful sign-out control passed. The initial eight-case run retained five
failures and three passes; one browser failure was a malformed mock and is
excluded from substantive bug evidence. That fixture was corrected. The first
corrected analysis run stopped on two test brace-style findings before tests;
that failure is retained. Independent review then identified an additional
memory-only-pending persistence ordering gap, corrected with recreated-client
and restore-path cases. Local-expiry cleanup was removed before qualification.
Final corrected test counts and hook evidence are recorded in WORK.

Production sign-in remains inactive. Build42 delivery is unchanged; this source
is newer and selects no upload, deployment, AMOS pin change or native activation.
