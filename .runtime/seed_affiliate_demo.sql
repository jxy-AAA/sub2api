DO $$
DECLARE
    admin_id BIGINT;
    first_id BIGINT;
    third_id BIGINT;
    customer_id BIGINT;
    api_key_id BIGINT;
    account_id BIGINT;
    usage_id BIGINT;
    first_idx INT;
    third_idx INT;
    customer_idx INT;
    usage_day INT;
    thirds_count INT;
    usage_amount NUMERIC(20,8);
    first_rate NUMERIC(10,4);
    third_rate NUMERIC(10,4);
    customer_rate NUMERIC(10,4);
    now_ts TIMESTAMPTZ := NOW();
BEGIN
    SELECT id INTO admin_id FROM users WHERE email = 'admin@sub2api.local' AND deleted_at IS NULL LIMIT 1;
    IF admin_id IS NULL THEN
        RAISE EXCEPTION 'admin user not found';
    END IF;

    INSERT INTO user_affiliates (user_id, aff_code, inviter_id, aff_count, aff_quota, aff_history_quota, created_at, updated_at, aff_code_custom, aff_frozen_quota)
    VALUES (admin_id, 'ADMINROOT', NULL, 0, 0, 0, now_ts, now_ts, true, 0)
    ON CONFLICT (user_id) DO UPDATE SET aff_code='ADMINROOT', inviter_id=NULL, updated_at=now_ts, aff_code_custom=true;

    INSERT INTO affiliate_distribution_user_model_rates (user_id, upstream_user_id, model_key, rate_multiplier, source_aff_code, source_type, created_at, updated_at)
    VALUES (admin_id, NULL, 'gpt-5.4', 1.0000, '', 'root_default', now_ts, now_ts)
    ON CONFLICT (user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, upstream_user_id=NULL, updated_at=now_ts;

    INSERT INTO affiliate_distribution_invite_model_rates (inviter_user_id, model_key, rate_multiplier, created_at, updated_at)
    VALUES (admin_id, 'gpt-5.4', 1.4000, now_ts, now_ts)
    ON CONFLICT (inviter_user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, updated_at=now_ts;

    INSERT INTO accounts (name, platform, type, credentials, extra, concurrency, priority, status, created_at, updated_at, schedulable, rate_multiplier)
    VALUES ('demo-affiliate-seed-account', 'openai', 'api_key', '{}'::jsonb, '{}'::jsonb, 100, 50, 'active', now_ts, now_ts, true, 1.0)
    ON CONFLICT DO NOTHING;
    SELECT id INTO account_id FROM accounts WHERE name='demo-affiliate-seed-account' AND deleted_at IS NULL LIMIT 1;

    FOR first_idx IN 1..5 LOOP
        first_rate := 1.40 + (first_idx - 1) * 0.03;
        INSERT INTO users (email, password_hash, role, balance, concurrency, status, username, notes, created_at, updated_at, signup_source)
        VALUES (format('demo.agent%1$s@sub2api.local', first_idx), '$2a$10$demo.seed.hash.not.for.login', 'user', 10000, 20, 'active', format('一级下级代理 %1$s', first_idx), 'demo affiliate seed: first-level agent', now_ts, now_ts, 'email')
        ON CONFLICT DO NOTHING;
        SELECT id INTO first_id FROM users WHERE email = format('demo.agent%1$s@sub2api.local', first_idx) AND deleted_at IS NULL LIMIT 1;

        INSERT INTO user_affiliates (user_id, aff_code, inviter_id, aff_count, aff_quota, aff_history_quota, created_at, updated_at, aff_code_custom, aff_frozen_quota)
        VALUES (first_id, format('A%1$s-CODE', first_idx), admin_id, 0, 0, 0, now_ts, now_ts, true, 0)
        ON CONFLICT (user_id) DO UPDATE SET aff_code=EXCLUDED.aff_code, inviter_id=admin_id, updated_at=now_ts, aff_code_custom=true;

        INSERT INTO affiliate_distribution_user_model_rates (user_id, upstream_user_id, model_key, rate_multiplier, source_aff_code, source_type, created_at, updated_at)
        VALUES (first_id, admin_id, 'gpt-5.4', first_rate, 'ADMINROOT', 'invite_code', now_ts, now_ts)
        ON CONFLICT (user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, upstream_user_id=admin_id, source_aff_code='ADMINROOT', source_type='invite_code', updated_at=now_ts;

        INSERT INTO affiliate_distribution_invite_model_rates (inviter_user_id, model_key, rate_multiplier, created_at, updated_at)
        VALUES (first_id, 'gpt-5.4', first_rate + 0.20, now_ts, now_ts)
        ON CONFLICT (inviter_user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, updated_at=now_ts;

        thirds_count := 10 + first_idx; -- 11..15, satisfying 10-15 per first-level agent.
        FOR third_idx IN 1..thirds_count LOOP
            third_rate := first_rate + 0.20 + ((third_idx % 3) * 0.02);
            INSERT INTO users (email, password_hash, role, balance, concurrency, status, username, notes, created_at, updated_at, signup_source)
            VALUES (format('demo.agent%1$s.%2$s@sub2api.local', first_idx, third_idx), '$2a$10$demo.seed.hash.not.for.login', 'user', 5000, 10, 'active', format('三级代理 %1$s-%2$s', first_idx, third_idx), 'demo affiliate seed: third-level agent', now_ts, now_ts, 'email')
            ON CONFLICT DO NOTHING;
            SELECT id INTO third_id FROM users WHERE email = format('demo.agent%1$s.%2$s@sub2api.local', first_idx, third_idx) AND deleted_at IS NULL LIMIT 1;

            INSERT INTO user_affiliates (user_id, aff_code, inviter_id, aff_count, aff_quota, aff_history_quota, created_at, updated_at, aff_code_custom, aff_frozen_quota)
            VALUES (third_id, format('A%1$s-T%2$s-CODE', first_idx, third_idx), first_id, 0, 0, 0, now_ts, now_ts, true, 0)
            ON CONFLICT (user_id) DO UPDATE SET aff_code=EXCLUDED.aff_code, inviter_id=first_id, updated_at=now_ts, aff_code_custom=true;

            INSERT INTO affiliate_distribution_user_model_rates (user_id, upstream_user_id, model_key, rate_multiplier, source_aff_code, source_type, created_at, updated_at)
            VALUES (third_id, first_id, 'gpt-5.4', third_rate, format('A%1$s-CODE', first_idx), 'invite_code', now_ts, now_ts)
            ON CONFLICT (user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, upstream_user_id=first_id, source_aff_code=format('A%1$s-CODE', first_idx), source_type='invite_code', updated_at=now_ts;

            INSERT INTO affiliate_distribution_invite_model_rates (inviter_user_id, model_key, rate_multiplier, created_at, updated_at)
            VALUES (third_id, 'gpt-5.4', third_rate + 0.15, now_ts, now_ts)
            ON CONFLICT (inviter_user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, updated_at=now_ts;

            FOR customer_idx IN 1..5 LOOP
                customer_rate := third_rate + 0.15 + ((customer_idx % 2) * 0.01);
                INSERT INTO users (email, password_hash, role, balance, concurrency, status, username, notes, created_at, updated_at, signup_source)
                VALUES (format('demo.customer%1$s.%2$s.%3$s@sub2api.local', first_idx, third_idx, customer_idx), '$2a$10$demo.seed.hash.not.for.login', 'user', 1000, 5, 'active', format('客户 %1$s-%2$s-%3$s', first_idx, third_idx, customer_idx), 'demo affiliate seed: customer', now_ts, now_ts, 'email')
                ON CONFLICT DO NOTHING;
                SELECT id INTO customer_id FROM users WHERE email = format('demo.customer%1$s.%2$s.%3$s@sub2api.local', first_idx, third_idx, customer_idx) AND deleted_at IS NULL LIMIT 1;

                INSERT INTO user_affiliates (user_id, aff_code, inviter_id, aff_count, aff_quota, aff_history_quota, created_at, updated_at, aff_code_custom, aff_frozen_quota)
                VALUES (customer_id, format('C%1$s%2$s%3$s-CODE', first_idx, third_idx, customer_idx), third_id, 0, 0, 0, now_ts, now_ts, true, 0)
                ON CONFLICT (user_id) DO UPDATE SET aff_code=EXCLUDED.aff_code, inviter_id=third_id, updated_at=now_ts, aff_code_custom=true;

                INSERT INTO affiliate_distribution_user_model_rates (user_id, upstream_user_id, model_key, rate_multiplier, source_aff_code, source_type, created_at, updated_at)
                VALUES (customer_id, third_id, 'gpt-5.4', customer_rate, format('A%1$s-T%2$s-CODE', first_idx, third_idx), 'invite_code', now_ts, now_ts)
                ON CONFLICT (user_id, model_key) DO UPDATE SET rate_multiplier=EXCLUDED.rate_multiplier, upstream_user_id=third_id, source_aff_code=format('A%1$s-T%2$s-CODE', first_idx, third_idx), source_type='invite_code', updated_at=now_ts;

                INSERT INTO api_keys (user_id, key, name, status, created_at, updated_at, quota, quota_used)
                VALUES (customer_id, format('sk-demo-%1$s-%2$s-%3$s', first_idx, third_idx, customer_idx), 'demo usage key', 'active', now_ts, now_ts, 0, 0)
                ON CONFLICT DO NOTHING;
                SELECT id INTO api_key_id FROM api_keys WHERE key = format('sk-demo-%1$s-%2$s-%3$s', first_idx, third_idx, customer_idx) LIMIT 1;

                FOR usage_day IN 0..6 LOOP
                    usage_amount := round((20 + random() * 280)::numeric, 8);
                    INSERT INTO usage_logs (
                        user_id, api_key_id, account_id, request_id, model,
                        input_tokens, output_tokens, input_cost, output_cost, total_cost, actual_cost,
                        stream, duration_ms, created_at, rate_multiplier, billing_type, requested_model, upstream_model, inbound_endpoint
                    ) VALUES (
                        customer_id, api_key_id, account_id, format('demo-aff-%1$s-%2$s-%3$s-%4$s', first_idx, third_idx, customer_idx, usage_day), 'gpt-5.4',
                        (1000 + floor(random()*9000))::int, (500 + floor(random()*5000))::int,
                        usage_amount * 0.45, usage_amount * 0.55, usage_amount, usage_amount,
                        false, (300 + floor(random()*5000))::int, (CURRENT_DATE - usage_day + make_interval(hours => (8 + floor(random()*10))::int))::timestamptz,
                        customer_rate, 0, 'gpt-5.4', 'gpt-5.4', '/v1/chat/completions'
                    )
                    ON CONFLICT DO NOTHING
                    RETURNING id INTO usage_id;

                    IF usage_id IS NOT NULL THEN
                        INSERT INTO billing_usage_entries (usage_log_id, user_id, api_key_id, billing_type, applied, delta_usd, created_at)
                        VALUES (usage_id, customer_id, api_key_id, 0, true, -usage_amount, now_ts)
                        ON CONFLICT DO NOTHING;
                    END IF;
                    usage_id := NULL;
                END LOOP;
            END LOOP;
        END LOOP;
    END LOOP;

    UPDATE user_affiliates ua
    SET aff_count = child_counts.cnt,
        updated_at = now_ts
    FROM (
        SELECT inviter_id, COUNT(*)::int AS cnt
        FROM user_affiliates
        WHERE inviter_id IS NOT NULL
        GROUP BY inviter_id
    ) child_counts
    WHERE ua.user_id = child_counts.inviter_id;
END $$;
