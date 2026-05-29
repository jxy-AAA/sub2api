ALTER TABLE model_trace_captures
    ADD COLUMN IF NOT EXISTS main_session_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS main_session_key CHAR(64) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS model_trace_captures_main_session_key_unique
    ON model_trace_captures (main_session_key)
    WHERE main_session_key <> '';

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_main_session_id
    ON model_trace_captures (main_session_id)
    WHERE main_session_id <> '';

ALTER TABLE trace_export_tasks
    ADD COLUMN IF NOT EXISTS downloaded_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_trace_export_tasks_downloaded_at_cleanup
    ON trace_export_tasks (downloaded_at, id)
    WHERE file_path <> '' AND downloaded_at IS NOT NULL;
