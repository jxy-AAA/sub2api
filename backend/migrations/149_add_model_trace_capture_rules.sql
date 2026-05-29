CREATE TABLE IF NOT EXISTS model_trace_capture_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,
    model_patterns JSONB NOT NULL DEFAULT '[]'::jsonb,
    user_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    api_key_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    min_tokens BIGINT NULL,
    max_tokens BIGINT NULL,
    sampling_ratio DOUBLE PRECISION NOT NULL DEFAULT 1,
    active_from TIMESTAMPTZ NULL,
    active_to TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_model_patterns_array_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_model_patterns_array_chk
            CHECK (jsonb_typeof(model_patterns) = 'array');
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_user_ids_array_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_user_ids_array_chk
            CHECK (jsonb_typeof(user_ids) = 'array');
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_api_key_ids_array_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_api_key_ids_array_chk
            CHECK (jsonb_typeof(api_key_ids) = 'array');
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_keywords_array_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_keywords_array_chk
            CHECK (jsonb_typeof(keywords) = 'array');
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_token_range_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_token_range_chk
            CHECK (
                (min_tokens IS NULL OR min_tokens >= 0) AND
                (max_tokens IS NULL OR max_tokens >= 0) AND
                (min_tokens IS NULL OR max_tokens IS NULL OR min_tokens <= max_tokens)
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_sampling_ratio_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_sampling_ratio_chk
            CHECK (sampling_ratio > 0 AND sampling_ratio <= 1);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'model_trace_capture_rules_active_window_chk'
    ) THEN
        ALTER TABLE model_trace_capture_rules
            ADD CONSTRAINT model_trace_capture_rules_active_window_chk
            CHECK (active_from IS NULL OR active_to IS NULL OR active_from <= active_to);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_model_trace_capture_rules_enabled_priority
    ON model_trace_capture_rules (enabled, priority DESC, id ASC);

CREATE INDEX IF NOT EXISTS idx_model_trace_capture_rules_active_window
    ON model_trace_capture_rules (active_from, active_to);

CREATE INDEX IF NOT EXISTS idx_model_trace_capture_rules_updated_at
    ON model_trace_capture_rules (updated_at DESC, id DESC);
