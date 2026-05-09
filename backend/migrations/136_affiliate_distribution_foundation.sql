CREATE TABLE IF NOT EXISTS affiliate_distribution_default_model_rates (
    model_key VARCHAR(255) PRIMARY KEY,
    default_markup DECIMAL(10,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO affiliate_distribution_default_model_rates (model_key, default_markup, created_at, updated_at)
VALUES ('*', 0, NOW(), NOW())
ON CONFLICT (model_key) DO NOTHING;

CREATE TABLE IF NOT EXISTS affiliate_distribution_invite_model_rates (
    inviter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_key VARCHAR(255) NOT NULL,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (inviter_user_id, model_key)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_invite_model_rates_model
    ON affiliate_distribution_invite_model_rates(model_key);

CREATE TABLE IF NOT EXISTS affiliate_distribution_user_model_rates (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    upstream_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    model_key VARCHAR(255) NOT NULL,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    source_aff_code VARCHAR(32) NULL,
    source_type VARCHAR(32) NOT NULL DEFAULT 'invite_code',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, model_key)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_user_model_rates_upstream
    ON affiliate_distribution_user_model_rates(upstream_user_id, model_key);

CREATE TABLE IF NOT EXISTS affiliate_distribution_usage_settlements (
    id BIGSERIAL PRIMARY KEY,
    usage_log_id BIGINT NOT NULL REFERENCES usage_logs(id) ON DELETE CASCADE,
    settlement_key VARCHAR(255) NOT NULL UNIQUE,
    beneficiary_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    direct_child_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    consumer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_key VARCHAR(255) NOT NULL,
    usage_amount_usd DECIMAL(20,8) NOT NULL,
    revenue_amount_usd DECIMAL(20,8) NOT NULL,
    parent_rate_multiplier DECIMAL(10,4) NOT NULL,
    child_rate_multiplier DECIMAL(10,4) NOT NULL,
    rebate_amount DECIMAL(20,8) NOT NULL,
    settlement_day DATE NOT NULL,
    depth INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_aff_dist_usage_settlement_usage_agent UNIQUE (usage_log_id, beneficiary_user_id)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_usage_settlements_beneficiary_day
    ON affiliate_distribution_usage_settlements(beneficiary_user_id, settlement_day DESC);

CREATE INDEX IF NOT EXISTS idx_aff_dist_usage_settlements_consumer_day
    ON affiliate_distribution_usage_settlements(consumer_user_id, settlement_day DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_usage_jobs (
    usage_log_id BIGINT PRIMARY KEY REFERENCES usage_logs(id) ON DELETE CASCADE,
    status VARCHAR(16) NOT NULL,
    last_error TEXT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS affiliate_distribution_daily_metrics (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_date DATE NOT NULL,
    revenue_amount_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    rebate_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    usage_count INTEGER NOT NULL DEFAULT 0,
    last_usage_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, metric_date)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_daily_metrics_rank
    ON affiliate_distribution_daily_metrics(metric_date DESC, revenue_amount_usd DESC, rebate_amount DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_rebate_balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    lifetime_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    last_reset_month DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_rebate_balances_rank
    ON affiliate_distribution_rebate_balances(current_amount DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_rebate_adjustments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operator_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    adjustment_type VARCHAR(32) NOT NULL,
    previous_amount DECIMAL(20,8) NOT NULL,
    new_amount DECIMAL(20,8) NOT NULL,
    delta_amount DECIMAL(20,8) NOT NULL,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_rebate_adjustments_user_created
    ON affiliate_distribution_rebate_adjustments(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_monthly_archives (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    archive_month DATE NOT NULL,
    archived_amount DECIMAL(20,8) NOT NULL,
    operator_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    operator_name VARCHAR(128) NULL,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_aff_dist_monthly_archive UNIQUE (user_id, archive_month)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_monthly_archives_month_rank
    ON affiliate_distribution_monthly_archives(archive_month DESC, archived_amount DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_monthly_reset_runs (
    archive_month DATE PRIMARY KEY,
    operator_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    operator_name VARCHAR(128) NULL,
    archived_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
