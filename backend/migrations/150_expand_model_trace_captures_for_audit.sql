ALTER TABLE IF EXISTS model_trace_captures
    ADD COLUMN IF NOT EXISTS capture_rule_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS request_content_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS response_content_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS input_tokens BIGINT NULL,
    ADD COLUMN IF NOT EXISTS output_tokens BIGINT NULL,
    ADD COLUMN IF NOT EXISTS total_tokens BIGINT NULL,
    ADD COLUMN IF NOT EXISTS upstream_status_code INTEGER NULL,
    ADD COLUMN IF NOT EXISTS raw_request_text TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS raw_response_text TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_captures_capture_rule_fk'
    ) THEN
        ALTER TABLE model_trace_captures
            ADD CONSTRAINT model_trace_captures_capture_rule_fk
            FOREIGN KEY (capture_rule_id)
            REFERENCES model_trace_capture_rules (id)
            ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_captures_input_tokens_chk'
    ) THEN
        ALTER TABLE model_trace_captures
            ADD CONSTRAINT model_trace_captures_input_tokens_chk
            CHECK (input_tokens IS NULL OR input_tokens >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_captures_output_tokens_chk'
    ) THEN
        ALTER TABLE model_trace_captures
            ADD CONSTRAINT model_trace_captures_output_tokens_chk
            CHECK (output_tokens IS NULL OR output_tokens >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_captures_total_tokens_chk'
    ) THEN
        ALTER TABLE model_trace_captures
            ADD CONSTRAINT model_trace_captures_total_tokens_chk
            CHECK (total_tokens IS NULL OR total_tokens >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_captures_upstream_status_code_chk'
    ) THEN
        ALTER TABLE model_trace_captures
            ADD CONSTRAINT model_trace_captures_upstream_status_code_chk
            CHECK (upstream_status_code IS NULL OR (upstream_status_code >= 100 AND upstream_status_code <= 999));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_capture_rule_id
    ON model_trace_captures (capture_rule_id);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_model
    ON model_trace_captures (model);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_total_tokens
    ON model_trace_captures (total_tokens);
