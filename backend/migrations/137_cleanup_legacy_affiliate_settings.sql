DELETE FROM settings
WHERE key IN (
    'affiliate_rebate_rate',
    'affiliate_rebate_freeze_hours',
    'affiliate_rebate_duration_days',
    'affiliate_rebate_per_invitee_cap'
);

UPDATE user_affiliates
SET aff_rebate_rate_percent = NULL
WHERE aff_rebate_rate_percent IS NOT NULL;

DROP INDEX IF EXISTS idx_user_affiliates_admin_settings;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_admin_settings
    ON user_affiliates (updated_at)
    WHERE aff_code_custom = true;

COMMENT ON COLUMN user_affiliates.aff_rebate_rate_percent IS 'Deprecated legacy affiliate rebate rate column kept nullable for backward compatibility';
