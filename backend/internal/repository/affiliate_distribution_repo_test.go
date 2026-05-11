package repository

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateDistributionRepositorySourceDropsLegacyModelRateTables(t *testing.T) {
	source, err := os.ReadFile("affiliate_distribution_repo.go")
	require.NoError(t, err)

	content := string(source)
	require.NotContains(t, content, "affiliate_distribution_default_model_rates")
	require.NotContains(t, content, "affiliate_distribution_invite_model_rates")
	require.NotContains(t, content, "affiliate_distribution_user_model_rates")
	require.NotContains(t, content, "affiliate_distribution_default_user_model_rates")
}
