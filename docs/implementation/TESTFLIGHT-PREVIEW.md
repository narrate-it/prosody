# Current-preview TestFlight delivery

The owner authorizes a TestFlight preview of current functionality without waiting
for full RFC/backend completion. This does not enable unfinished production auth,
backend endpoints or paid generation. The stable app identity remains
`run.sire.prosody`. Existing distribution setup is owner-managed.

## Invocation and scope

After exact-source review and guarded main merge, dispatch the manual
`TestFlight preview upload` workflow on trusted main. It uses only the existing
approved self-hosted macOS runner and `prosody-asc` environment. Upload defaults
to false; set `upload=true` only for an owner-authorized delivery. Never dispatch
an active-review branch or bypass an environment, capacity or signing refusal.

The job reads the existing app, complete bounded build inventory and iOS store
versions with exact GET-scoped tokens. It chooses a higher numeric build and, if
needed, the next patch marketing version. Unsupported history, pagination or
rollover refuses rather than guessing. This is observation, not a server-side
build-number reservation; a concurrent release can still invalidate selection.

The runner must already contain the owner's `prosody-ci` identity and one current
matching App Store profile. The script never unlocks keychains, requests passwords,
changes the default/search list, broadens key access or retries authentication.
An owner may need to answer a new-identity signing prompt locally; a failure stops
that phase. Preserve existing identities and private recovery material.

Fresh sampled capacity and an exact canonical build lease precede the heavy
phase. Installed Flutter and iOS26 SDK are used; no hosted Mac fallback or tool
installation occurs. The script builds unsigned Flutter output, creates a manually
signed archive, checks metadata and signature, exports an IPA and optionally
validates/uploads once through the installed Apple upload tool. Tracked source
mutation stops upload. No permanent source version rewrite is required.

## Evidence and recovery

Private logs, archive/export and receipt stay in a unique permission-restricted
`prosody-testflight-*` runner cache directory. They are not uploaded as public
workflow artifacts. Individual phase logs are capped at16MiB; IPA hashing is
streamed through a bounded non-symlink regular descriptor. Child phases have finite
watchdogs. Their owned process groups are stopped even after leader failure;
unconfirmed cleanup records the group and retains its lease for reconciliation.
Receipt-write failures cannot silently skip exact-token release after cleanup.

The upload state is atomically written/fsynced as UNKNOWN before entering the
external call. Failed final updates do not truncate the prior receipt. Timeout,
cancellation, response loss or ambiguous outcome requires checking the SAME
version/build and Apple processing history before any new upload. No automatic
retry or alternate build number is authorized by an uncertain response.

A successful upload-tool response means accepted transport, not completed Apple
processing or availability in the TestFlight app. Verify processing, export
compliance and existing tester/group access separately. Do not invite testers or
send notifications without the applicable authority. Full backend/provider/device
qualification remains independent of this preview delivery.

## Source verification

Twenty named hermetic roots pass with zero failures/skips. They cover version
selection, exact JWT/transport, bounded inventory, workflow mutation controls,
profile/identity negatives, streamed artifact/FIFO/symlink refusal, actual bounded
synthetic child cleanup/log caps and atomic UNKNOWN ordering/failure controls.
The first14-root run had two macOS process-observation errors; those remain failed
historical evidence. Corrected17/18/19-root results are superseded by the final20.
Synthetic process and mocked Apple/Keychain controls do not establish native
archive, signing, upload, Apple processing or TestFlight acceptance. Independent
source review is clear; required hook and hosted review evidence are separate.

## Locked keychain refusal

After actual CodeSign failure with prosody-ci confirmed locked, release admission
checks native keychain lock metadata before credentials, heavy work and lease
acquisition, and again before archive/export signing. It never calls unlock or
authentication APIs. Owner unlock/remediation is required before a new phase; no
automatic retry occurs. An unlocked observation can become stale and does not
prove private-key access permissions or actual signing acceptance.
