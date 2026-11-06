# Decoded aggregate narration budget

`media.MeasureNarration` validates generated MP3 parts against one shared
duration and encoded-byte budget. It decodes actual PCM samples and compares
their exact rational durations before rounding the total down once to
nanoseconds. It does not trust reported duration, MIME or a provider receipt.
Parts may use different supported sample rates; one recording still cannot
change its rate internally. Any failed part, cancellation, byte overflow or
duration overflow returns an empty measurement, never a partial success.

The caller supplies a positive duration ceiling no greater than 600 seconds and
a combined positive byte budget no greater than the existing 64 MiB generated
speech codec bound. At most 100 parts are accepted, matching the current domain
segment bound. This local codec profile does not select new general RFC upload,
supplied-media or production buffering limits. Byte lengths are checked together
before decoding. Before a duration overflow, earlier accepted parts have passed
that ceiling and at most one further duration-limited decode runs, with decoder
frame/look-ahead overhead. Encoded scanning is separately byte-bounded. This
does not establish a precise CPU-time bound, service latency or a launch SLO.

Inputs are caller-owned in-memory recordings and must not be mutated during
validation. The returned timing is descriptive data, not immutable artifact
custody, authorization, a native checkpoint receipt or publication proof. The
native adapter must bind the exact bytes to its accepted artifacts and pass ALL
generated parts from the complete admitted manifest. Supplied audio/video must
not be classified as generated narration or counted against this ceiling.

`briefing.Synthesize` uses this validator for its one completed speech response.
Its existing authority/financial admission and typed provider observations are
unchanged. A failed measurement retains request/generation references and yields
no usable audio. One-part synthesis does not establish aggregate completion of a
multi-part rendition. Publication still needs the native current-authority,
fence, complete-artifact and same-transaction receipt/StageReady boundary.

No paid retry or shortening loop was added. The RFC's at-most-one additional
admitted shortening/synthesis cycle still requires separately registered stages,
native accounting and original-attempt reconciliation before activation.

## Verification

All 17 named focused records passed with zero failures/skips. They exercise
real Go MP3 decoding of synthetic zero-content recordings, including exact
600-second acceptance and an additional-frame refusal, combined byte limits,
late malformed/truncated parts, finite part count, cancellation, mixed rates and
sub-nanosecond excess that per-part rounding would hide.

A compiled aggregate-guard omission replacing total comparison with per-part
comparison failed the intended aggregate case and parent; its exact-budget
control passed. A separate compiled per-part-rounding omission failed both
rational and mixed-rate cases while the exact-budget child and parent passed.
Source files were not replaced by these compiler overlays. These checks do not
qualify external provider encodings, native services, durable storage, actual
mobile playback or a whole production submission-to-listen journey.
