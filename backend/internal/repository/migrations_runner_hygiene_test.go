package repository

import (
	"io/fs"
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigrationsHygiene_NoUnexpectedDuplicateNumericPrefixes(t *testing.T) {
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)

	actual := duplicateMigrationNumericPrefixes(files)
	expected := map[string][]string{
		"006": {"006_add_users_allowed_groups_compat.sql", "006_fix_invalid_subscription_expires_at.sql", "006b_guard_users_allowed_groups.sql"},
		"028": {"028_add_account_notes.sql", "028_add_usage_logs_user_agent.sql", "028_group_image_pricing.sql"},
		"029": {"029_add_group_claude_code_restriction.sql", "029_usage_log_image_fields.sql"},
		"033": {"033_add_promo_codes.sql", "033_ops_monitoring_vnext.sql"},
		"034": {"034_ops_upstream_error_events.sql", "034_usage_dashboard_aggregation_tables.sql"},
		"036": {"036_ops_error_logs_add_is_count_tokens.sql", "036_scheduler_outbox.sql"},
		"037": {"037_add_account_rate_multiplier.sql", "037_ops_alert_silences.sql"},
		"042": {"042_add_usage_cleanup_tasks.sql", "042b_add_ops_system_metrics_switch_count.sql"},
		"043": {"043_add_usage_cleanup_cancel_audit.sql", "043b_add_group_invalid_request_fallback.sql"},
		"044": {"044_add_user_totp.sql", "044b_add_group_mcp_xml_inject.sql"},
		"045": {"045_add_accounts_extra_index.sql", "045_add_announcements.sql", "045_add_api_key_quota.sql"},
		"046": {"046_add_sora_accounts.sql", "046_add_usage_log_reasoning_effort.sql", "046b_add_group_supported_model_scopes.sql"},
		"047": {"047_add_sora_pricing_and_media_type.sql", "047_add_user_group_rate_multipliers.sql"},
		"052": {"052_add_group_sort_order.sql", "052_migrate_upstream_to_apikey.sql"},
		"053": {"053_add_security_secrets.sql", "053_add_skip_monitoring_to_error_passthrough.sql"},
		"054": {"054_drop_legacy_cache_columns.sql", "054_ops_system_logs.sql"},
		"060": {"060_add_gemini31_flash_image_to_model_mapping.sql", "060_add_usage_log_openai_ws_mode.sql"},
		"070": {"070_add_scheduled_test_auto_recover.sql", "070_add_usage_log_service_tier.sql"},
		"071": {"071_add_gemini25_flash_image_to_model_mapping.sql", "071_add_usage_billing_dedup.sql"},
		"075": {"075_add_usage_log_upstream_model.sql", "075_map_haiku45_to_sonnet46.sql"},
		"081": {"081_add_group_account_filter.sql", "081_create_channels.sql"},
		"095": {"095_channel_features.sql", "095_subscription_plans.sql"},
		"101": {"101_add_account_stats_pricing.sql", "101_add_balance_notify_fields.sql", "101_add_channel_features_config.sql", "101_add_payment_mode.sql"},
		"102": {"102_add_balance_notify_threshold_type.sql", "102_add_out_trade_no_to_payment_orders.sql"},
		"108": {"108_auth_identity_foundation_core.sql", "108a_widen_auth_identity_migration_report_type.sql"},
		"120": {"120_enforce_payment_orders_out_trade_no_unique_notx.sql", "120a_align_payment_orders_out_trade_no_index_name.sql"},
		"125": {"125_add_channel_monitors.sql", "125_add_group_rpm_limit.sql"},
		"126": {"126_add_channel_monitor_aggregation.sql", "126_add_user_rpm_limit.sql"},
		"127": {"127_add_user_group_rpm_override.sql", "127_drop_channel_monitor_deleted_at.sql"},
		"134": {"134_affiliate_ledger_audit_snapshots.sql", "134_image_generation_group_controls.sql"},
		"135": {"135_allow_email_oauth_provider_types.sql", "135_content_moderation.sql"},
		"147": {"147_add_model_interaction_traces.sql", "147_add_model_trace_captures.sql"},
		"148": {"148_harden_model_interaction_traces.sql", "148_model_trace_captures_constraints.sql"},
	}
	require.Equal(t, expected, actual)
}

func TestMigrationsHygiene_ExecutionModeRules(t *testing.T) {
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	sort.Strings(files)

	for _, name := range files {
		contentBytes, readErr := fs.ReadFile(migrations.FS, name)
		require.NoErrorf(t, readErr, "read migration %s", name)

		_, validateErr := validateMigrationExecutionMode(name, string(contentBytes))
		require.NoErrorf(t, validateErr, "migration %s violates execution mode rule", name)
	}
}

func duplicateMigrationNumericPrefixes(files []string) map[string][]string {
	seen := make(map[string][]string, len(files))
	for _, name := range files {
		prefix := leadingMigrationNumericPrefix(name)
		if prefix == "" {
			continue
		}
		seen[prefix] = append(seen[prefix], name)
	}

	dups := make(map[string][]string)
	for prefix, group := range seen {
		if len(group) <= 1 {
			continue
		}
		sort.Strings(group)
		dups[prefix] = group
	}
	return dups
}

func leadingMigrationNumericPrefix(name string) string {
	idx := 0
	for idx < len(name) && name[idx] >= '0' && name[idx] <= '9' {
		idx++
	}
	if idx == 0 {
		return ""
	}
	return name[:idx]
}
