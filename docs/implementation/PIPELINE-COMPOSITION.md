# Hermetic ordinary pipeline composition

`internal/pipeline/composition_test.go` connects `pipeline.Run` to the real
`briefing.Rewrite` and `briefing.Synthesize` functions with a bounded provider
double. Synthesis uses the existing checked-in `briefing/testdata/silence.mp3`
fixture and the real MP3 frame/decoder duration and full-byte SHA256 validation.
No provider HTTP call is made. Typed `MayHaveActed` failures represent synthetic
post-handoff drop/truncation observations; they are not actual HTTP fault tests.

The positive path carries the validated script through a test-only checkpoint
to speech input, then carries exact audio bytes, full-byte hashes and measured
duration through another test-only checkpoint to synthetic publication. It
checks one call per stage and that provider-buffer mutation cannot change the
accepted/checkpointed/published audio. Committed synthetic replay performs no
additional provider call or publication.

Negative cases reject a fenced-code script, invalid MP3 and truncated MP3;
rewrite/speech post-handoff uncertainty stops advancement. Separate rewrite
and speech checkpoint-loss cases stop on UNKNOWN and do not dispatch again on
re-entry. Original provider observations/references stay at the fixture's
briefing boundary; the pipeline returns its fixed uncertainty classification.
No retry or lookup proof is invented.

Every provider port, candidate, committed receipt, publication, model/accounting
selection and authority decision in the fixture is visibly synthetic and lives
only in `_test.go`. In-memory maps are ephemeral test state, not a new journal
or scheduler. Shape-only request/result/manifest digest stand-ins do not claim
native canonical-codec or accepted-result binding. Source/settings work hashes
are of the exact fixture byte representations, not logical Submission.Digest.
No production adapter or native receipt constructor is added.

The tests do not run `store.Publish` or StageReady, a database transaction or
native current authority, financial admission, settlement, reconciliation,
atomicity or durability. They do not qualify actual OpenRouter transport,
AMOS integration, installation or the visible mobile submit-to-listen journey.
Those remain separate required acceptance gates. The existing native
`jobs.Worker` remains the sole scheduler.

Verification: Go 1.27.0 / GOMAXPROCS=2, `go test -json -p 1 -count=1
-timeout 60s ./internal/pipeline -run '^TestOrdinaryComposition'` exited 0.
All three top-level tests and seven named subtests passed (10 pass records),
with no skips. The preserved full commit hook runs before the source checkpoint.

Capacity admission: one-minute load 5.06, interval CPU idle 44.2/49.1%
(average 46.65%), normal pressure, zero swap churn and no thermal/performance
warning. Checks use one exact owned build lease, released after the gate.
No live provider call, credential lookup, database fixture, provisioning,
device, signing or deployment action is part of this test-only task.
