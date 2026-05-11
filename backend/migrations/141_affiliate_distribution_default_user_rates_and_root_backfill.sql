ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS inviter_source VARCHAR(32) NOT NULL DEFAULT 'unknown';

CREATE TABLE IF NOT EXISTS affiliate_distribution_default_user_group_rates (
    group_id BIGINT PRIMARY KEY REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS affiliate_distribution_invite_group_rates (
    inviter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (inviter_user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_invite_group_rates_group
    ON affiliate_distribution_invite_group_rates(group_id);

CREATE TABLE IF NOT EXISTS affiliate_distribution_user_group_rates (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    upstream_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    source_aff_code VARCHAR(32) NULL,
    source_type VARCHAR(32) NOT NULL DEFAULT 'invite_code',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_aff_dist_user_group_rates_upstream
    ON affiliate_distribution_user_group_rates(upstream_user_id, group_id);

CREATE INDEX IF NOT EXISTS idx_aff_dist_user_group_rates_group
    ON affiliate_distribution_user_group_rates(group_id);

ALTER TABLE user_affiliates
    DROP CONSTRAINT IF EXISTS user_affiliates_inviter_source_check;

ALTER TABLE user_affiliates
    ADD CONSTRAINT user_affiliates_inviter_source_check
    CHECK (inviter_source IN ('unknown', 'affiliate_code', 'default_root', 'admin_override', 'none'));

ALTER TABLE affiliate_distribution_default_user_group_rates
    DROP CONSTRAINT IF EXISTS aff_dist_default_user_group_rates_rate_check;

ALTER TABLE affiliate_distribution_default_user_group_rates
    ADD CONSTRAINT aff_dist_default_user_group_rates_rate_check
    CHECK (rate_multiplier > 0 AND rate_multiplier <= 100);

ALTER TABLE affiliate_distribution_invite_group_rates
    DROP CONSTRAINT IF EXISTS aff_dist_invite_group_rates_rate_check;

ALTER TABLE affiliate_distribution_invite_group_rates
    ADD CONSTRAINT aff_dist_invite_group_rates_rate_check
    CHECK (rate_multiplier > 0 AND rate_multiplier <= 100);

ALTER TABLE affiliate_distribution_user_group_rates
    DROP CONSTRAINT IF EXISTS aff_dist_user_group_rates_rate_check;

ALTER TABLE affiliate_distribution_user_group_rates
    ADD CONSTRAINT aff_dist_user_group_rates_rate_check
    CHECK (rate_multiplier > 0 AND rate_multiplier <= 100);

ALTER TABLE affiliate_distribution_user_group_rates
    DROP CONSTRAINT IF EXISTS aff_dist_user_group_rates_source_type_check;

ALTER TABLE affiliate_distribution_user_group_rates
    ADD CONSTRAINT aff_dist_user_group_rates_source_type_check
    CHECK (
        source_type IN (
            'admin_override',
            'upstream_override',
            'invite_code',
            'default_explicit',
            'group_inherited',
            'group_default',
            'root_default'
        )
    );

ALTER TABLE affiliate_distribution_user_group_rates
    DROP CONSTRAINT IF EXISTS aff_dist_user_group_rates_no_self_upstream_check;

ALTER TABLE affiliate_distribution_user_group_rates
    ADD CONSTRAINT aff_dist_user_group_rates_no_self_upstream_check
    CHECK (upstream_user_id IS NULL OR upstream_user_id <> user_id);

WITH first_admin AS (
    SELECT id
    FROM users
    WHERE role = 'admin'
      AND status = 'active'
    ORDER BY id ASC
    LIMIT 1
)
UPDATE user_affiliates ua
SET inviter_id = fa.id,
    inviter_source = 'default_root',
    updated_at = NOW()
FROM first_admin fa
WHERE ua.inviter_id IS NULL
  AND ua.user_id <> fa.id;

UPDATE user_affiliates ua
SET aff_count = COALESCE(children.child_count, 0),
    updated_at = NOW()
FROM (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS child_count
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
) children
WHERE ua.user_id = children.user_id;

UPDATE user_affiliates ua
SET aff_count = 0,
    updated_at = NOW()
WHERE NOT EXISTS (
    SELECT 1
    FROM user_affiliates child
    WHERE child.inviter_id = ua.user_id
)
  AND ua.aff_count <> 0;

DO $$
BEGIN
    IF to_regclass('affiliate_distribution_default_user_model_rates') IS NOT NULL THEN
        IF to_regclass('affiliate_distribution_default_user_model_rates_legacy_141') IS NOT NULL THEN
            RAISE EXCEPTION 'legacy archive already exists for affiliate_distribution_default_user_model_rates';
        END IF;
        EXECUTE 'ALTER TABLE affiliate_distribution_default_user_model_rates RENAME TO affiliate_distribution_default_user_model_rates_legacy_141';
    END IF;

    IF to_regclass('affiliate_distribution_user_model_rates') IS NOT NULL THEN
        IF to_regclass('affiliate_distribution_user_model_rates_legacy_141') IS NOT NULL THEN
            RAISE EXCEPTION 'legacy archive already exists for affiliate_distribution_user_model_rates';
        END IF;
        EXECUTE 'ALTER TABLE affiliate_distribution_user_model_rates RENAME TO affiliate_distribution_user_model_rates_legacy_141';
    END IF;

    IF to_regclass('affiliate_distribution_invite_model_rates') IS NOT NULL THEN
        IF to_regclass('affiliate_distribution_invite_model_rates_legacy_141') IS NOT NULL THEN
            RAISE EXCEPTION 'legacy archive already exists for affiliate_distribution_invite_model_rates';
        END IF;
        EXECUTE 'ALTER TABLE affiliate_distribution_invite_model_rates RENAME TO affiliate_distribution_invite_model_rates_legacy_141';
    END IF;

    IF to_regclass('affiliate_distribution_default_model_rates') IS NOT NULL THEN
        IF to_regclass('affiliate_distribution_default_model_rates_legacy_141') IS NOT NULL THEN
            RAISE EXCEPTION 'legacy archive already exists for affiliate_distribution_default_model_rates';
        END IF;
        EXECUTE 'ALTER TABLE affiliate_distribution_default_model_rates RENAME TO affiliate_distribution_default_model_rates_legacy_141';
    END IF;
END
$$;
