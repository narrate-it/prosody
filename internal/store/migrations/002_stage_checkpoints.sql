-- Private provisional bytes only. Native checkpoint receipts and current
-- authority/fence finality MUST enclose these writes in the caller transaction.
CREATE TABLE IF NOT EXISTS public.app_prosody_stage_checkpoints (
 workspace_id uuid NOT NULL,
 job_id uuid NOT NULL,
 invocation_id uuid NOT NULL,
 stage_id text NOT NULL CHECK(stage_id IN ('rewrite','speech')),
 item_id uuid NOT NULL,
 feed_id uuid NOT NULL,
 binding_bytes bytea NOT NULL CHECK(octet_length(binding_bytes) BETWEEN 1 AND 8192),
 artifact_bytes bytea NOT NULL CHECK(octet_length(artifact_bytes) BETWEEN 1 AND CASE WHEN stage_id='rewrite' THEN 256000 ELSE 67108864 END),
 artifact_digest bytea NOT NULL CHECK(octet_length(artifact_digest)=32),
 PRIMARY KEY(workspace_id,job_id,invocation_id,stage_id),
 FOREIGN KEY(workspace_id,item_id) REFERENCES public.app_prosody_submissions(workspace_id,id),
 FOREIGN KEY(workspace_id,feed_id) REFERENCES public.app_prosody_feeds(workspace_id,id)
);
REVOKE ALL ON public.app_prosody_stage_checkpoints FROM PUBLIC;
