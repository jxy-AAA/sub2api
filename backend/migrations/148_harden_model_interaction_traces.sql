-- Align trace storage with the Taoding export contract after early development drafts.
DROP INDEX IF EXISTS model_interaction_traces_task_id_unique;

ALTER TABLE IF EXISTS model_interaction_traces
    ALTER COLUMN prompt SET DEFAULT '[]'::json,
    ALTER COLUMN candidates SET DEFAULT '[]'::json,
    ALTER COLUMN tools SET DEFAULT '[]'::json,
    ALTER COLUMN meta SET DEFAULT '{}'::json,
    ALTER COLUMN scaffold SET DEFAULT '{}'::json,
    ALTER COLUMN scaffold_version SET DEFAULT 'sub2api-taoding-trace-v1';

ALTER TABLE IF EXISTS model_interaction_traces
    ALTER COLUMN signature TYPE json USING (
        CASE
            WHEN signature IS NULL OR btrim(signature::text) = '' THEN '{"available":false}'::json
            WHEN left(btrim(signature::text), 1) IN ('{', '[', '"') THEN signature::json
            ELSE to_json(signature::text)::json
        END
    );

UPDATE model_interaction_traces
SET
    prompt = COALESCE(prompt, '[]'::json),
    candidates = COALESCE(candidates, '[]'::json),
    tools = COALESCE(tools, '[]'::json),
    signature = COALESCE(signature, '{"available":false}'::json),
    meta = COALESCE(meta, '{}'::json),
    scaffold = COALESCE(scaffold, '{}'::json),
    scaffold_version = COALESCE(NULLIF(scaffold_version, ''), 'sub2api-taoding-trace-v1');

ALTER TABLE IF EXISTS model_interaction_traces
    ALTER COLUMN prompt SET NOT NULL,
    ALTER COLUMN candidates SET NOT NULL,
    ALTER COLUMN tools SET NOT NULL,
    ALTER COLUMN signature SET NOT NULL,
    ALTER COLUMN meta SET NOT NULL,
    ALTER COLUMN scaffold SET NOT NULL,
    ALTER COLUMN scaffold_version SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_task_id
    ON model_interaction_traces (task_id);
