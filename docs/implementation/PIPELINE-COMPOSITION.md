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


## Rebased verification, 2026-11-03

Rebased the existing PR13 branch onto landed main
`1cd648b1f510ba14bcf2ada1ce7a8ec2f5222b8f`, which includes briefing policy
revision 2 and the provider-catalog changes. Test source is byte-identical to
the initial independently reviewed candidate. The same focused command above
passed freshly: three top-level tests plus seven subtests, 10 pass records and
no skips. The preserved full Go hook gates the rebased checkpoint; hosted
review/merge remain pending. The earlier capacity receipt is historical.

Current admission: load 6.06, interval idle 50.0/44.35% (average 47.175%),
normal pressure, zero swap churn and no thermal/performance warning. The exact
owned lease covers these bounded checks and is released after the gate.
Synthetic receipts remain data-flow fixtures, not native/provider/database or
mobile qualification. No live call, fixture provisioning or device change.
