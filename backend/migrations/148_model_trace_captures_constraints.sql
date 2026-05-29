CREATE UNIQUE INDEX IF NOT EXISTS model_trace_captures_task_id_unique
    ON model_trace_captures (task_id);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_prompt_hash
    ON model_trace_captures (prompt_hash);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_request_id
    ON model_trace_captures (request_id);

CREATE INDEX IF NOT EXISTS idx_model_trace_captures_response_id
    ON model_trace_captures (response_id);
