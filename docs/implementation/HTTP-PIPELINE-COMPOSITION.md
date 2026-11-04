# Hermetic TLS provider-to-pipeline composition

Source preparation from main `f5ace049e3d00f736fbac9c3c3d724591ee29589`.
The existing pipeline composition uses bounded provider doubles, while provider
HTTP outcome tests exercise the client separately. These new tests connect the
actual OpenRouter client to real briefing Rewrite/Synthesize and the LOCAL
pipeline using owned TLS unit servers. No production source/API changes or
existing test relocation is involved.

## Test-only client and network boundary

`internal/openrouter/export_test.go` exposes a factory only in the Go test build.
External-package `openrouter_test` imports that factory plus briefing/pipeline;
there is no production-importable fake helper package or native constructor.
The factory starts with New, preserving its timeout and CheckRedirect refusal.
It clones only the caller-owned TLS server transport, disables proxying and
changes the base to that server's `/api/v1` path. Parsed URL scheme/host must
match the actual TLS listener. A test-only RoundTripper refuses any differing
scheme or origin before underlying dispatch, including accidental base changes.

The origin guard leaves all paths at the owned origin reachable; it does not
mask redirect policy by blocking an unapproved same-origin path. Redirect refusal
is retained from New rather than replaced. These cases do not independently
exercise redirects. The actual cloned transport gets explicit idle-connection
cleanup even though the guard wrapper has no CloseIdleConnections method.
There is no global/default transport mutation or external-origin request.
Ordinary owned httptest TLS servers are hermetic unit fixtures, not service or
native profile qualification.

## Cases and assertions

One named composition test has six modes:

- A valid rewrite response feeds exact checkpointed script bytes to speech;
  real bundled MP3 bytes, SHA256 and decoder-measured duration survive the
  synthetic checkpoint/publication flow. HTTP request headers/body, model token
  cap, voice and MP3 request format are checked by the server.
- Rewrite request accepted/read by the server, then the connection closes before
  headers. There is one actual rewrite HTTP call, no speech or publication.
- A valid rewrite followed by speech HTTP truncation declares Content-Length
  greater than the complete MP3 body. The client must retain ErrUnexpectedEOF,
  available request/generation headers, and raw MayHaveActed. No speech artifact
  can checkpoint or publish. Using an otherwise complete valid MP3 ensures a
  missing HTTP read-error guard cannot be hidden by codec rejection.
- Complete HTTP with invalid MP3, and complete HTTP with invalid MIME, fail
  separately while retaining raw ResponseReceived. Neither is native Confirmed
  settlement, usable audio or publication proof.
- Empty-key client preflight sends zero HTTP requests and retains raw
  NotDispatched. No native no-effect/refund/retry authority is inferred.

Actual server counters assert one call per dispatched stage, with no automatic
transport retry. Same-process second Run observes synthetic unresolved or
published state without another HTTP call. That is not restart durability or
native reconciliation. The fixture captures the real typed briefing error before
pipeline redaction; errors.As checks its raw state and available request/generation
references against the returned observation and flat result fields. The truncated
case also preserves errors.Is(ErrUnexpectedEOF) at that briefing boundary. Pipeline
returns only its fixed ErrUnresolved classification and safe rendering; it does
not retain typed provider causes.

A future native adapter must capture bounded typed observations at the briefing
boundary BEFORE pipeline redaction. Raw errors and sensitive response bodies must
never become generic journal payloads. The fixture's process-memory error field
is test-only inspection, not a native adapter or durable journal implementation.
No error wrapping or native capability is invented by these assertions; rendered
errors at both boundaries must omit key/source/body markers.

## Synthetic boundaries and executed verification

All candidate/receipt/publication tokens and maps live only in `_test.go`.
Model/token metadata and request/result/manifest projection digests are synthetic
shape stand-ins. No accepted native result, current authority, durable journal,
financial admission/settlement, atomic private checkpoint, store.Publish or
StageReady transaction is implemented or qualified. The scoped unit run below
exercised the real client and MP3 inspection data flow. No native/provider/mobile
acceptance follows from ordinary TLS unit fixtures.

The coordinator executed exact source tree
`fe7119036df76aec64193b8f9d094e499357dbbd` on main
`f5ace049e3d00f736fbac9c3c3d724591ee29589`. The standard export_test bridge,
external-package imports and origin-bound TLS client factory compiled and ran.
All six modes plus their parent passed: seven named records, exit 0, no failures
or skips. The valid control exercised the permitted owned loopback origin.

A compiled overlay removed ONLY client.request's io.ReadAll error guard while
retaining all test scaffolding and leaving the working production file untouched.
The speech-truncation case and its parent failed (two failure records), while
the valid-flow positive control passed (one record): exit 1 as expected, no skips.
The otherwise valid MP3 makes this a response-framing/read-error regression,
not a failure hidden by invalid codec bytes. Owned test-server/transport cleanups
completed and the exact owned build lease was released.

These scoped observations do not qualify redirects, race freedom, native or
billing behavior, public-origin/provider interoperability or database effects.
Lint, the full Go unit gate and the required commit hook are separate from this
scoped receipt; their actual results are recorded in the PR verification record.

No live provider, real key, environment/credential lookup, database fixture/run,
cloud provisioning, spend, device, signing, deployment or AMOS/C1 source change
is part of this lane. Existing native scheduling and all delivery/durability
qualification gates remain separate.
