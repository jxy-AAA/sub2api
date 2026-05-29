CREATE TABLE IF NOT EXISTS trace_export_tasks (
    id BIGSERIAL PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    format VARCHAR(32) NOT NULL DEFAULT 'json_array',
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    include_raw BOOLEAN NOT NULL DEFAULT FALSE,
    target_records BIGINT NOT NULL DEFAULT 500,
    requested_by BIGINT NOT NULL,
    download_filename TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL DEFAULT '',
    file_size_bytes BIGINT NOT NULL DEFAULT 0,
    total_records BIGINT NOT NULL DEFAULT 0,
    processed_records BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NULL,
    canceled_by BIGINT NULL,
    canceled_at TIMESTAMPTZ NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    ALTER TABLE trace_export_tasks
        ADD COLUMN IF NOT EXISTS target_records BIGINT NOT NULL DEFAULT 500;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_status_chk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_status_chk
            CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled'));
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_format_chk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_format_chk
            CHECK (format IN ('json_array'));
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_filters_object_chk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_filters_object_chk
            CHECK (jsonb_typeof(filters) = 'object');
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_target_records_chk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_target_records_chk
            CHECK (target_records > 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_progress_chk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_progress_chk
            CHECK (
                file_size_bytes >= 0 AND
                total_records >= 0 AND
                processed_records >= 0 AND
                (total_records = 0 OR processed_records <= total_records)
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_requested_by_fk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_requested_by_fk
            FOREIGN KEY (requested_by)
            REFERENCES users (id)
            ON DELETE RESTRICT;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trace_export_tasks_canceled_by_fk'
    ) THEN
        ALTER TABLE trace_export_tasks
            ADD CONSTRAINT trace_export_tasks_canceled_by_fk
            FOREIGN KEY (canceled_by)
            REFERENCES users (id)
            ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_trace_export_tasks_status_created_at
    ON trace_export_tasks (status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_trace_export_tasks_requested_by_created_at
    ON trace_export_tasks (requested_by, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_trace_export_tasks_created_at
    ON trace_export_tasks (created_at DESC, id DESC);
