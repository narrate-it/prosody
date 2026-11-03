# Verified media-range preparation

`media.ReadVerifiedRange` is a local building block for the proposed bounded
human media callback, not a registered AMOS interface or HTTP endpoint. The
caller supplies the already-open object, expected full SHA-256/size and closed
inclusive offsets. The caller owns reader custody, deadline and cleanup.

The function buffers at most 1 MiB of selected bytes plus a 32 KiB read buffer.
It hashes the full object while capturing the selected interval in the same
pass; it does not verify a hash and then reopen/re-read potentially changed
content. It returns nothing usable before full digest and exact-size agreement.
Overlong/truncated/mismatched data, invalid bounds, cancellation, no-progress or
reader errors return an empty result. Digest length is checked before decoding.
Reader/cancellation failures render a fixed safe message while preserving causes
for errors.Is/As; unwrapped causes are internal diagnostics and must not be logged. Object I/O is limited to declared size +1,
with a hard 1 GiB object bound. Context checks cannot interrupt an arbitrary
blocking Reader; a qualified object adapter must enforce its I/O deadline.

Native AMOS code must still admit/hold the current person/session/grant and
workspace/feed/item/rendition/published-asset closure, bind expected MIME/digest/
length, validate a single bounded closed range, buffer/final-check/commit and
encode the actual response. This helper proves none of those permissions. It
accepts no URL/path, performs no transport/retry, publishes no metadata, and has
no production caller. Nanos durable storage and actual iOS closed-range player
compatibility are separate gates. No bearer fallback or replacement route exists.

Source/tests are prepared in the isolated `feat/verified-media-ranges` worktree.
Initial verification was held by failed capacity admission (interval idle4.82/34.98%,
average19.90% below20%, normal pressure, zero swap churn, no warnings). At that checkpoint only gofmt and
diff checks ran; no tests, hook, commit, server, lease or provider call ran.


## First executed verification

After fresh admission at interval idle46.20/53.74%, normal pressure, zero swap
churn and no warnings, five named range tests passed. Rebased the staged work
onto main a5de1fa while preserving range source/test bytes exactly and both
sets of status records; retained the original recovery stash. These tests use
in-memory/fake readers. They establish local arithmetic/buffering/error behavior,
not real object adapter latency/deadlines, native authorization, iOS requests or
compatible media wire. The up-to1GiB full read per range requires a separate
held-resource/I/O budget and actual lifetime measurement before activation.
