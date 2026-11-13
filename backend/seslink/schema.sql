-- Consumer-owned protected SNS-envelope retention (Prosody composition).
-- AMOS supplies the verified-source contract (ses.RawSNSEventSource); this
-- schema is the operator-owned durable journal it deliberately does not supply.
-- Insert-only for the runtime role: conflicting evidence is retained, never
-- overwritten; no UPDATE/DELETE is granted to the runtime.
BEGIN;
CREATE SCHEMA IF NOT EXISTS prosody_events AUTHORIZATION prosody_owner;
CREATE TABLE IF NOT EXISTS prosody_events.sns_envelopes (
    job_id          uuid        NOT NULL,
    envelope_sha256 char(64)    NOT NULL,
    received_at     timestamptz NOT NULL DEFAULT now(),
    envelope        bytea       NOT NULL CHECK (octet_length(envelope) <= 262144),
    PRIMARY KEY (job_id, envelope_sha256)
);
CREATE INDEX IF NOT EXISTS sns_envelopes_received_at
    ON prosody_events.sns_envelopes (received_at);
GRANT USAGE ON SCHEMA prosody_events TO prosody_runtime;
GRANT SELECT, INSERT ON prosody_events.sns_envelopes TO prosody_runtime;
COMMIT;
