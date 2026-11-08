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

## Sign-in recovery discovery failure

Initial hosted review completed without inline actionable findings but identified
an additional pre-activation weakness: sign-in could keep an old in-memory grant
if pending-record discovery failed before active cleanup. The new regression
fails on that candidate with a normal sign-out control passing. Sign-in now
invalidates its in-memory token and local expiry before any recovery I/O, so
read errors or malformed recovery records cannot reopen that old local access
when the busy flag clears. Active storage is not deleted before pending discovery
and safe persistence; storage retention after a failed operation does not assert
successful logout or remote revocation. This follow-up remains in the same PR;
the earlier hosted SUCCESS is not approval of the new head.

Older binaries that do not recognize the pending slot are not qualified for
recovery or downgrade with simultaneous active and pending records. Before
activation, qualify the supported rollout/downgrade set, physical interruption
boundaries and actual issuer handling of repeated revocation. This source does
not impose a server version policy or prove those operating properties.

## Stored-target recovery after a failed replacement read

The next hosted review found a still-valid case: after failed sign-in discovery,
an explicit sign-out could delete the unchanged active Keychain record without
revoking its grant, because memory no longer held it. That candidate was not
merged. The new original-candidate regression fails with a normal sign-out
control passing.

Explicit sign-out alone may now read the exact active slot as a recovery-only
target when pending storage is absent. It never installs that value as an active
session, never calls current to establish access, and still persists pending
before active deletion. Pending records take precedence; an unreadable or
malformed pending record cannot fall through. Failed active reads preserve
storage and fail sign-out. Generation is checked after each asynchronous read
and before shared pending assignment, preventing stale results from resurrecting
completed recovery. Fresh-client logout needs no successful restore first.

This protects the retained original record; it does not expand the contract to
preserve every prior grant after an unrelated login replacement has already
completed discovery and deleted old storage. Broader replacement/restart and
rollout semantics remain separately qualified, not inferred from this fix.
