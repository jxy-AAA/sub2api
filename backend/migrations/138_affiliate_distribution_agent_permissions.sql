CREATE TABLE IF NOT EXISTS affiliate_distribution_agent_permissions (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    can_view_downline_daily_revenue BOOLEAN NOT NULL DEFAULT FALSE,
    can_view_downline_rebate_balances BOOLEAN NOT NULL DEFAULT FALSE,
    can_manage_downline_pricing BOOLEAN NOT NULL DEFAULT FALSE,
    granted_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_agent_permissions_granted_by
    ON affiliate_distribution_agent_permissions(granted_by_user_id, updated_at DESC);
