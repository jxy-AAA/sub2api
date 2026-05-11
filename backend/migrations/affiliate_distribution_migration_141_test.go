package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration141ArchivesLegacyModelRateTablesAndAddsChecks(t *testing.T) {
	content, err := FS.ReadFile("141_affiliate_distribution_default_user_rates_and_root_backfill.sql")
	require.NoError(t, err)

	sql := string(content)
	require.NotContains(t, sql, "DROP TABLE IF EXISTS affiliate_distribution_default_user_model_rates")
	require.NotContains(t, sql, "DROP TABLE IF EXISTS affiliate_distribution_user_model_rates")
	require.NotContains(t, sql, "DROP TABLE IF EXISTS affiliate_distribution_invite_model_rates")
	require.NotContains(t, sql, "DROP TABLE IF EXISTS affiliate_distribution_default_model_rates")

	require.Contains(t, sql, "RENAME TO affiliate_distribution_default_user_model_rates_legacy_141")
	require.Contains(t, sql, "RENAME TO affiliate_distribution_user_model_rates_legacy_141")
	require.Contains(t, sql, "RENAME TO affiliate_distribution_invite_model_rates_legacy_141")
	require.Contains(t, sql, "RENAME TO affiliate_distribution_default_model_rates_legacy_141")
	require.Contains(t, sql, "user_affiliates_inviter_source_check")
	require.Contains(t, sql, "aff_dist_user_group_rates_source_type_check")
	require.Contains(t, sql, "aff_dist_user_group_rates_no_self_upstream_check")
	require.Contains(t, sql, "rate_multiplier > 0 AND rate_multiplier <= 100")
	require.True(t, strings.Contains(sql, "to_regclass('affiliate_distribution_user_model_rates')"))
}
