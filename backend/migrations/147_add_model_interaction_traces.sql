-- Raw model interaction trace storage for admin export / PDF generation.
-- JSON columns use `json` instead of `jsonb` so the original structure is preserved as closely as possible.
CREATE TABLE IF NOT EXISTS model_interaction_traces (
    id BIGSERIAL PRIMARY KEY,
    task_id TEXT NOT NULL,
    prompt JSON NOT NULL DEFAULT '[]'::json,
    candidates JSON NOT NULL DEFAULT '[]'::json,
    tools JSON NOT NULL DEFAULT '[]'::json,
    signature JSON NOT NULL DEFAULT '{"available":false}'::json,
    meta JSON NOT NULL DEFAULT '{}'::json,
    scaffold JSON NOT NULL DEFAULT '{}'::json,
    scaffold_version TEXT NOT NULL DEFAULT 'sub2api-taoding-trace-v1',
    model TEXT NULL,
    user_id BIGINT NULL,
    api_key_id BIGINT NULL,
    request_id TEXT NULL,
    dedupe_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_task_id
    ON model_interaction_traces (task_id);

CREATE UNIQUE INDEX IF NOT EXISTS model_interaction_traces_dedupe_hash_unique
    ON model_interaction_traces (dedupe_hash);

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_created_at_id
    ON model_interaction_traces (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_request_id
    ON model_interaction_traces (request_id);

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_model
    ON model_interaction_traces (model);

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_user_id
    ON model_interaction_traces (user_id);

CREATE INDEX IF NOT EXISTS idx_model_interaction_traces_api_key_id
    ON model_interaction_traces (api_key_id);
