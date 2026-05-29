CREATE TABLE IF NOT EXISTS model_trace_captures (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(128) NOT NULL,
    request_id VARCHAR(128) NULL,
    response_id VARCHAR(128) NULL,
    user_id BIGINT NULL,
    api_key_id BIGINT NULL,
    group_id BIGINT NULL,
    account_id BIGINT NULL,
    protocol VARCHAR(80) NOT NULL DEFAULT '',
    model VARCHAR(200) NOT NULL DEFAULT '',
    requested_model VARCHAR(200) NOT NULL DEFAULT '',
    upstream_model VARCHAR(200) NOT NULL DEFAULT '',
    scaffold VARCHAR(100) NOT NULL DEFAULT '',
    scaffold_version VARCHAR(120) NOT NULL DEFAULT '',
    prompt_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    candidates_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    tools_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    signature_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    meta_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_request_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_response_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    dedupe_hash CHAR(64) NOT NULL,
    prompt_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_model_trace_captures_dedupe_hash
    ON model_trace_captures (dedupe_hash);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_created_at
    ON model_trace_captures (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_user_id
    ON model_trace_captures (user_id);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_api_key_id
    ON model_trace_captures (api_key_id);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_account_id
    ON model_trace_captures (account_id);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_protocol
    ON model_trace_captures (protocol);
