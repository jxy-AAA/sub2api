-- 144c_ent_data_model_guards.sql
-- Harden schema-level intent that Ent alone did not enforce strongly enough:
--   1. explicit enum-like check constraints for high-value fields
--   2. TLS fingerprint profile soft-reference consistency guards
--   3. append-only protection for usage_logs with a maintenance escape hatch

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_platform_allowed'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_platform_allowed
            CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity'))
            NOT VALID;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_type_allowed'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_type_allowed
            CHECK (type IN ('oauth', 'setup-token', 'apikey', 'upstream', 'bedrock', 'service_account'))
            NOT VALID;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_status_allowed'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_status_allowed
            CHECK (status IN ('active', 'disabled', 'error'))
            NOT VALID;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'payment_orders_order_type_allowed'
    ) THEN
        ALTER TABLE payment_orders
            ADD CONSTRAINT payment_orders_order_type_allowed
            CHECK (order_type IN ('balance', 'subscription'))
            NOT VALID;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'payment_orders_status_allowed'
    ) THEN
        ALTER TABLE payment_orders
            ADD CONSTRAINT payment_orders_status_allowed
            CHECK (
                status IN (
                    'PENDING',
                    'PAID',
                    'RECHARGING',
                    'COMPLETED',
                    'EXPIRED',
                    'CANCELLED',
                    'FAILED',
                    'REFUND_REQUESTED',
                    'REFUNDING',
                    'PARTIALLY_REFUNDED',
                    'REFUNDED',
                    'REFUND_FAILED'
                )
            )
            NOT VALID;
    END IF;
END $$;

CREATE OR REPLACE FUNCTION validate_account_tls_fingerprint_profile_ref()
RETURNS trigger AS $$
DECLARE
    profile_id_text text;
    profile_id bigint;
BEGIN
    IF NEW.extra IS NULL THEN
        RETURN NEW;
    END IF;

    profile_id_text := btrim(COALESCE(NEW.extra->>'tls_fingerprint_profile_id', ''));
    IF profile_id_text = '' OR profile_id_text = '0' OR profile_id_text = '-1' THEN
        RETURN NEW;
    END IF;

    IF profile_id_text !~ '^[0-9]+$' THEN
        RAISE EXCEPTION 'accounts.extra.tls_fingerprint_profile_id must be a positive integer, 0, or -1'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'accounts_tls_fingerprint_profile_ref';
    END IF;

    profile_id := profile_id_text::bigint;
    IF NOT EXISTS (
        SELECT 1
        FROM tls_fingerprint_profiles
        WHERE id = profile_id
    ) THEN
        RAISE EXCEPTION 'tls fingerprint profile % does not exist', profile_id
            USING ERRCODE = '23503',
                  CONSTRAINT = 'accounts_tls_fingerprint_profile_ref';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS accounts_tls_fingerprint_profile_ref_guard ON accounts;
CREATE TRIGGER accounts_tls_fingerprint_profile_ref_guard
    BEFORE INSERT OR UPDATE OF extra ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION validate_account_tls_fingerprint_profile_ref();

CREATE OR REPLACE FUNCTION prevent_tls_fingerprint_profile_delete_in_use()
RETURNS trigger AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM accounts
        WHERE deleted_at IS NULL
          AND extra IS NOT NULL
          AND btrim(COALESCE(extra->>'tls_fingerprint_profile_id', '')) ~ '^[0-9]+$'
          AND (extra->>'tls_fingerprint_profile_id')::bigint = OLD.id
    ) THEN
        RAISE EXCEPTION 'tls fingerprint profile % is still referenced by active accounts', OLD.id
            USING ERRCODE = '23503',
                  CONSTRAINT = 'tls_fingerprint_profiles_in_use';
    END IF;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tls_fingerprint_profiles_delete_in_use_guard ON tls_fingerprint_profiles;
CREATE TRIGGER tls_fingerprint_profiles_delete_in_use_guard
    BEFORE DELETE ON tls_fingerprint_profiles
    FOR EACH ROW
    EXECUTE FUNCTION prevent_tls_fingerprint_profile_delete_in_use();

CREATE OR REPLACE FUNCTION protect_usage_logs_immutability()
RETURNS trigger AS $$
BEGIN
    IF current_setting('sub2api.usage_log_maintenance', true) = 'on' THEN
        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NEW;
    END IF;

    RAISE EXCEPTION 'usage_logs is append-only; % is not allowed', lower(TG_OP)
        USING ERRCODE = '55000',
              CONSTRAINT = 'usage_logs_append_only_guard';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS usage_logs_append_only_guard ON usage_logs;
CREATE TRIGGER usage_logs_append_only_guard
    BEFORE UPDATE OR DELETE ON usage_logs
    FOR EACH ROW
    EXECUTE FUNCTION protect_usage_logs_immutability();
