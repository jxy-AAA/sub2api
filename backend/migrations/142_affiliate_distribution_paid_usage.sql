CREATE TABLE IF NOT EXISTS affiliate_distribution_paid_credit_balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    paid_credit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    settled_paid_usage_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    last_credit_at TIMESTAMPTZ NULL,
    last_settlement_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT aff_dist_paid_credit_non_negative_check CHECK (paid_credit_usd >= 0),
    CONSTRAINT aff_dist_paid_usage_non_negative_check CHECK (settled_paid_usage_usd >= 0)
);

CREATE TABLE IF NOT EXISTS affiliate_distribution_paid_credit_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_order_id BIGINT NOT NULL UNIQUE,
    amount_usd DECIMAL(20,8) NOT NULL,
    credited_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT aff_dist_paid_credit_events_amount_check CHECK (amount_usd > 0)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_paid_credit_events_user_time
    ON affiliate_distribution_paid_credit_events(user_id, credited_at DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_paid_credit_reversals (
    id BIGSERIAL PRIMARY KEY,
    credit_event_id BIGINT NOT NULL REFERENCES affiliate_distribution_paid_credit_events(id) ON DELETE RESTRICT,
    source_order_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_usd DECIMAL(20,8) NOT NULL,
    reason TEXT NOT NULL DEFAULT 'refund',
    reversed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT aff_dist_paid_credit_reversal_amount_check CHECK (amount_usd > 0)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_paid_credit_reversals_event
    ON affiliate_distribution_paid_credit_reversals(credit_event_id);

CREATE INDEX IF NOT EXISTS idx_aff_dist_paid_credit_reversals_order
    ON affiliate_distribution_paid_credit_reversals(source_order_id);

CREATE TABLE IF NOT EXISTS affiliate_distribution_paid_usage_settlements (
    usage_log_id BIGINT PRIMARY KEY REFERENCES usage_logs(id) ON DELETE RESTRICT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    usage_amount_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    settlement_day DATE NOT NULL,
    settled_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT aff_dist_paid_usage_amount_non_negative_check CHECK (usage_amount_usd >= 0)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_paid_usage_user_day
    ON affiliate_distribution_paid_usage_settlements(user_id, settlement_day DESC);

CREATE TABLE IF NOT EXISTS affiliate_distribution_paid_credit_allocations (
    credit_event_id BIGINT NOT NULL REFERENCES affiliate_distribution_paid_credit_events(id) ON DELETE RESTRICT,
    usage_log_id BIGINT NOT NULL REFERENCES affiliate_distribution_paid_usage_settlements(usage_log_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_usd DECIMAL(20,8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (credit_event_id, usage_log_id),
    CONSTRAINT aff_dist_paid_credit_alloc_amount_check CHECK (amount_usd > 0)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_paid_credit_allocations_usage
    ON affiliate_distribution_paid_credit_allocations(usage_log_id);

CREATE INDEX IF NOT EXISTS idx_aff_dist_paid_credit_allocations_user
    ON affiliate_distribution_paid_credit_allocations(user_id, credit_event_id);
