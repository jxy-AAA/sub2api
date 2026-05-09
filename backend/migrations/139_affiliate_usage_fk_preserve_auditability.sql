ALTER TABLE affiliate_distribution_usage_settlements
    DROP CONSTRAINT IF EXISTS affiliate_distribution_usage_settlements_usage_log_id_fkey;

ALTER TABLE affiliate_distribution_usage_settlements
    ADD CONSTRAINT affiliate_distribution_usage_settlements_usage_log_id_fkey
    FOREIGN KEY (usage_log_id)
    REFERENCES usage_logs(id)
    ON DELETE RESTRICT;

ALTER TABLE affiliate_distribution_usage_jobs
    DROP CONSTRAINT IF EXISTS affiliate_distribution_usage_jobs_usage_log_id_fkey;

ALTER TABLE affiliate_distribution_usage_jobs
    ADD CONSTRAINT affiliate_distribution_usage_jobs_usage_log_id_fkey
    FOREIGN KEY (usage_log_id)
    REFERENCES usage_logs(id)
    ON DELETE RESTRICT;
