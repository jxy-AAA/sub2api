-- Scheduler outbox dedup hot path:
-- WHERE event_type = $1
--   AND account_id IS NOT DISTINCT FROM $2
--   AND group_id IS NOT DISTINCT FROM $3
--   AND created_at >= NOW() - interval ...
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheduler_outbox_dedup_scope_created_at
    ON scheduler_outbox (event_type, account_id, group_id, created_at DESC);

-- Scheduler outbox polling hot path:
-- WHERE id > $1 ORDER BY id ASC LIMIT $2
-- plus lag checks that read created_at of the oldest/newest rows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheduler_outbox_poll_id_created_at
    ON scheduler_outbox (id, created_at);
