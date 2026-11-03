# Paid provider attempt observations

PR5 adds `openrouter.AttemptOutcome` to rewrite/speech results and carries it
through briefing ports. `AttemptError` supports errors.As while preserving
errors.Is for existing local sentinels. Its rendered message contains only a
known state label; raw transport errors/provider bodies are never included in that rendered message.
Wrapped transport/read and arbitrary port causes remain diagnostic values and
must not be logged. Their identity is retained for errors.Is/As. Flat compatibility
fields are synchronized from the finalized outcome after error-only metadata
recovery.

| State | Observation | Host obligation |
| --- | --- | --- |
| not_dispatched | Local validation/cancellation stopped the call before HTTP/port handoff. | Re-admit any later request; this is not authority to dispatch. |
| response_received | Complete successful HTTP response read, or provider port returned without error. | Validate output and reconcile accounting; not proof of generation success or billing finality. |
| may_have_acted | Transport handoff followed by transport/read/limit/HTTP failure, untyped port error, or invalid state. | Reconcile before another paid attempt; unsupported lookup/dedup stays unknown/manual review. |

Request/generation IDs and known token counts survive output rejection and typed
error-only receipts. Empty references/zero counts are not evidence of no effect.
A rejected rewrite never yields usable script/text; rejected speech/media yields
no usable audio. This layer supplies neither a retry scheduler nor a journal,
provider lookup/dedup proof, native authorization or financial permission. AMOS
must apply its registered provider contract and durable stage/publication rules.

Hermetic checks cover both paid methods: zero HTTP requests on credential
rejection; one request after server drop/truncation, arbitrary HTTP failure or
cancellation triggered by handler entry; successful/rejected complete response
metadata. A separate injected deadline transport checks timeout classification
without claiming server acceptance. Briefing cases cover error-only accounting,
invalid-state sanitization, policy/media rejection and sentinel preservation.

Independent review found missing error-only token transfer and unsafe state
rendering in the first checkpoint; new regressions fail on that checkpoint and
pass with both fixes. Initial timed server fixture could hang in teardown; it
was replaced by handler-triggered cancellation and an explicit release channel.
No live provider calls, billing acceptance, native integration or deployment.
