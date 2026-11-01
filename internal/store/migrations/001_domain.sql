-- Prosody owns only this schema. AMOS owns principals, authority and job execution.
CREATE TABLE IF NOT EXISTS public.app_prosody_schema_version(version integer PRIMARY KEY);
CREATE TABLE IF NOT EXISTS public.app_prosody_feeds (
 workspace_id uuid NOT NULL,
 id uuid NOT NULL,
 kind text NOT NULL CHECK (kind IN ('personal','team')),
 title bytea NOT NULL,
 settings_revision bigint NOT NULL DEFAULT 1 CHECK (settings_revision > 0),
 deleted_at timestamptz,
 PRIMARY KEY(workspace_id,id)
);
-- Bytea preserves accepted NUL/Unicode in keys and every user-controlled payload.
-- This is a domain receipt, not an independent authority or retry scheduler.
CREATE TABLE IF NOT EXISTS public.app_prosody_submissions (
 installation_id uuid NOT NULL,
 application_id text NOT NULL,
 environment_id text NOT NULL,
 workspace_id uuid NOT NULL,
 person_id uuid NOT NULL,
 operation_id text NOT NULL,
 key_bytes bytea NOT NULL CHECK(octet_length(key_bytes) BETWEEN 1 AND 256),
 digest bytea NOT NULL CHECK(octet_length(digest)=32),
 id uuid NOT NULL UNIQUE,
 feed_id uuid NOT NULL,
 source_bytes bytea NOT NULL,
 settings_bytes bytea NOT NULL,
 authority_ref bytea NOT NULL,
 status text NOT NULL DEFAULT 'processing' CHECK(status IN ('processing','ready','failed','deleted')),
 accepted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 deleted_at timestamptz,
 PRIMARY KEY(installation_id,application_id,environment_id,workspace_id,person_id,operation_id,key_bytes),
 FOREIGN KEY(workspace_id,feed_id) REFERENCES public.app_prosody_feeds(workspace_id,id),
 UNIQUE(workspace_id,id)
);
CREATE TABLE IF NOT EXISTS public.app_prosody_work (
 item_id uuid PRIMARY KEY REFERENCES public.app_prosody_submissions(id),
 work_id uuid NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS public.app_prosody_renditions (
 workspace_id uuid NOT NULL,
 item_id uuid NOT NULL,
 id uuid NOT NULL,
 settings_revision bigint NOT NULL CHECK(settings_revision>0),
 manifest_bytes bytea NOT NULL,
 script_bytes bytea NOT NULL,
 provenance_bytes bytea NOT NULL,
 narration_ms bigint NOT NULL CHECK(narration_ms BETWEEN 0 AND 600000),
 published_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(workspace_id,item_id,id),
 FOREIGN KEY(workspace_id,item_id) REFERENCES public.app_prosody_submissions(workspace_id,id)
);
CREATE TABLE IF NOT EXISTS public.app_prosody_progress (
 workspace_id uuid NOT NULL,
 person_id uuid NOT NULL,
 item_id uuid NOT NULL,
 rendition_id uuid NOT NULL,
 revision bigint NOT NULL CHECK(revision>=0),
 value_bytes bytea NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(workspace_id,person_id,item_id,rendition_id),
 FOREIGN KEY(workspace_id,item_id,rendition_id) REFERENCES public.app_prosody_renditions(workspace_id,item_id,id)
);
CREATE INDEX IF NOT EXISTS submissions_feed_order ON public.app_prosody_submissions(workspace_id,feed_id,accepted_at,id);
INSERT INTO public.app_prosody_schema_version VALUES(1) ON CONFLICT DO NOTHING;

REVOKE ALL ON public.app_prosody_schema_version, public.app_prosody_feeds, public.app_prosody_submissions, public.app_prosody_renditions, public.app_prosody_progress, public.app_prosody_work FROM PUBLIC;

CREATE TABLE IF NOT EXISTS public.app_prosody_assets (
 workspace_id uuid NOT NULL,
 id uuid NOT NULL,
 mime text NOT NULL CHECK(mime IN ('image/jpeg','image/png','audio/mpeg','video/mp4')),
 byte_size bigint NOT NULL CHECK(byte_size>0),
 sha256 bytea NOT NULL CHECK(octet_length(sha256)=32),
 duration_ms bigint NOT NULL CHECK(duration_ms>=0),
 generated boolean NOT NULL,
 deleted_at timestamptz,
 PRIMARY KEY(workspace_id,id)
);
CREATE TABLE IF NOT EXISTS public.app_prosody_settings (
 workspace_id uuid NOT NULL,
 feed_id uuid NOT NULL,
 revision bigint NOT NULL CHECK(revision>0),
 value_bytes bytea NOT NULL,
 key_envelope bytea,
 credential_id uuid,
 credential_revision bigint,
 PRIMARY KEY(workspace_id,feed_id),
 FOREIGN KEY(workspace_id,feed_id) REFERENCES public.app_prosody_feeds(workspace_id,id)
);
CREATE TABLE IF NOT EXISTS public.app_prosody_preferences (
 workspace_id uuid NOT NULL,
 feed_id uuid NOT NULL,
 person_id uuid NOT NULL,
 notifications boolean NOT NULL DEFAULT false,
 PRIMARY KEY(workspace_id,feed_id,person_id),
 FOREIGN KEY(workspace_id,feed_id) REFERENCES public.app_prosody_feeds(workspace_id,id)
);
REVOKE ALL ON public.app_prosody_assets, public.app_prosody_settings, public.app_prosody_preferences FROM PUBLIC;
