package repository

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
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

func TestAffiliateDistributionSettlementAmountsUsesActualCostAsPaidUsage(t *testing.T) {
	rawUsageUSD, paidUsageUSD, consumerRate, hasConsumerRate := affiliateDistributionSettlementAmounts(service.AffiliateDistributionUsageSettlementCommand{
		TotalCost:         10,
		ActualCost:        3,
		RateMultiplier:    0.3,
		HasRateMultiplier: true,
	})

	require.InDelta(t, 10.0, rawUsageUSD, 1e-9)
	require.InDelta(t, 3.0, paidUsageUSD, 1e-9)
	require.True(t, hasConsumerRate)
	require.InDelta(t, 0.3, consumerRate, 1e-9)

	eligibleRawUsage := affiliateDistributionProratedRawUsage(rawUsageUSD, paidUsageUSD, paidUsageUSD)
	require.InDelta(t, 10.0, eligibleRawUsage, 1e-9)
	require.Zero(t, affiliateDistributionRebateAmountRMB(eligibleRawUsage, 0.3, consumerRate))
	require.InDelta(t, 10.0, affiliateDistributionBusinessAmountUSD(eligibleRawUsage, consumerRate), 1e-9)
}

func TestAffiliateDistributionProratedRawUsageScalesPartialPaidCredit(t *testing.T) {
	require.InDelta(t, 5.0, affiliateDistributionProratedRawUsage(10, 3, 1.5), 1e-9)
}

func TestAffiliateDistributionRecordUsageSettlementUsesConfiguredRateForDirectConsumerAndBusinessAmount(t *testing.T) {
	client, mock := newAffiliateDistributionEntMockClient(t)
	repo := NewAffiliateDistributionRepository(client)
	ctx := context.Background()

	parentUserID := int64(11)
	consumerUserID := int64(22)
	groupID := int64(33)
	usageLogID := int64(44)
	settlementAt := time.Date(2026, 5, 27, 8, 30, 0, 0, time.UTC)

	expectEnsureUserAffiliateWithNoRootAdmin(mock, consumerUserID, &parentUserID)
	mock.ExpectQuery(queryPattern(affiliateDistributionLoadAffiliateChainSQL)).
		WithArgs(consumerUserID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "inviter_id", "depth"}).
			AddRow(consumerUserID, parentUserID, 0).
			AddRow(parentUserID, nil, 1))

	expectEnsureUserAffiliateWithNoRootAdmin(mock, parentUserID, nil)
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupExplicitUserGroupRateSQL)).
		WithArgs(parentUserID, groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier", "source_type"}).
			AddRow(0.4, affiliateDistributionSourceAdminOverride))

	expectEnsureUserAffiliateWithNoRootAdmin(mock, consumerUserID, &parentUserID)
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupExplicitUserGroupRateSQL)).
		WithArgs(consumerUserID, groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier", "source_type"}).
			AddRow(2.5, affiliateDistributionSourceAdminOverride))
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupExplicitUserGroupRateSQL)).
		WithArgs(consumerUserID, groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier", "source_type"}).
			AddRow(2.5, affiliateDistributionSourceAdminOverride))

	mock.ExpectExec("INSERT INTO affiliate_distribution_usage_settlements").
		WithArgs(
			usageLogID,
			"44:11:gpt-5.5",
			parentUserID,
			consumerUserID,
			consumerUserID,
			"gpt-5.5",
			10.0,
			10.0,
			0.4,
			2.5,
			2.1,
			truncateToUTCDate(settlementAt),
			int64(1),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO affiliate_distribution_daily_metrics").
		WithArgs(parentUserID, truncateToUTCDate(settlementAt), 10.0, 2.1, settlementAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO affiliate_distribution_rebate_balances").
		WithArgs(parentUserID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE affiliate_distribution_rebate_balances").
		WithArgs(2.1, parentUserID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := repo.recordUsageSettlementWithTx(ctx, client, service.RecordAffiliateUsageSettlementInput{
		UsageLogID:                usageLogID,
		ConsumerUserID:            consumerUserID,
		GroupID:                   groupID,
		ModelKey:                  "gpt-5.5",
		UsageAmountUSD:            10,
		SettlementAt:              settlementAt,
		RootRate:                  9.9,
		ConsumerRateMultiplier:    0.3,
		HasConsumerRateMultiplier: true,
	})
	require.NoError(t, err)
	require.Len(t, result.AppliedEntries, 1)
	require.InDelta(t, 0.4, result.AppliedEntries[0].ParentRateMultiplier, 1e-9)
	require.InDelta(t, 2.5, result.AppliedEntries[0].ChildRateMultiplier, 1e-9)
	require.InDelta(t, 10.0, result.AppliedEntries[0].RevenueAmountUSD, 1e-9)
	require.InDelta(t, 2.1, result.AppliedEntries[0].RebateAmountRMB, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAffiliateDistributionResolveGroupRateRootUsesDefaultUserGroupRateBeforeFallbacks(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewAffiliateDistributionRepository(nil)
	ctx := context.Background()

	const (
		rootUserID int64 = 55
		groupID    int64 = 66
	)

	expectEnsureUserAffiliateWithNoRootAdmin(mock, rootUserID, nil)
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupExplicitUserGroupRateSQL)).
		WithArgs(rootUserID, groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier", "source_type"}))
	mock.ExpectQuery(queryPattern(affiliateDistributionQueryAffiliateByUserIDSQL)).
		WithArgs(rootUserID).
		WillReturnRows(affiliateSummaryRows(rootUserID, nil))
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupDefaultUserGroupRateSQL)).
		WithArgs(groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier"}).AddRow(2.4))

	rate, err := repo.resolveGroupRateWithMemo(ctx, db, rootUserID, groupID, 9.9, make(map[[2]int64]float64))
	require.NoError(t, err)
	require.InDelta(t, 2.4, rate, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAffiliateDistributionResolveGroupRateRootUsesGroupRateBeforeRootGuard(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewAffiliateDistributionRepository(nil)
	ctx := context.Background()

	const (
		rootUserID int64 = 77
		groupID    int64 = 88
	)

	expectEnsureUserAffiliateWithNoRootAdmin(mock, rootUserID, nil)
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupExplicitUserGroupRateSQL)).
		WithArgs(rootUserID, groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier", "source_type"}))
	mock.ExpectQuery(queryPattern(affiliateDistributionQueryAffiliateByUserIDSQL)).
		WithArgs(rootUserID).
		WillReturnRows(affiliateSummaryRows(rootUserID, nil))
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupDefaultUserGroupRateSQL)).
		WithArgs(groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier"}))
	mock.ExpectQuery(queryPattern(affiliateDistributionLookupGroupDefaultRateSQL)).
		WithArgs(groupID).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier"}).AddRow(1.7))

	rate, err := repo.resolveGroupRateWithMemo(ctx, db, rootUserID, groupID, 9.9, make(map[[2]int64]float64))
	require.NoError(t, err)
	require.InDelta(t, 1.7, rate, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}

const affiliateDistributionQueryAffiliateByUserIDSQL = `
SELECT user_id,
       aff_code,
       aff_code_custom,
       aff_rebate_rate_percent,
       inviter_id,
       inviter_source,
       aff_count,
       aff_quota::double precision,
       aff_frozen_quota::double precision,
       aff_history_quota::double precision,
       created_at,
       updated_at
FROM user_affiliates
WHERE user_id = $1`

const affiliateDistributionQueryFirstActiveAdminIDSQL = `
SELECT id
FROM users
WHERE role = $1
  AND status = $2
ORDER BY id ASC
LIMIT 1`

const affiliateDistributionLoadAffiliateChainSQL = `
WITH RECURSIVE chain AS (
    SELECT user_id, inviter_id, 0 AS depth
    FROM user_affiliates
    WHERE user_id = $1
    UNION ALL
    SELECT parent.user_id, parent.inviter_id, chain.depth + 1
    FROM user_affiliates parent
    JOIN chain ON chain.inviter_id = parent.user_id
)
SELECT user_id, inviter_id, depth
FROM chain
ORDER BY depth ASC`

const affiliateDistributionLookupExplicitUserGroupRateSQL = `
SELECT rate_multiplier::double precision,
       source_type
FROM affiliate_distribution_user_group_rates
WHERE user_id = $1
  AND group_id = $2
LIMIT 1`

const affiliateDistributionLookupDefaultUserGroupRateSQL = `
SELECT rate_multiplier::double precision
FROM affiliate_distribution_default_user_group_rates
WHERE group_id = $1
LIMIT 1`

const affiliateDistributionLookupGroupDefaultRateSQL = `
SELECT rate_multiplier::double precision
FROM groups
WHERE id = $1
  AND deleted_at IS NULL
  AND status = 'active'
LIMIT 1`

func newAffiliateDistributionEntMockClient(t *testing.T) (*dbent.Client, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))

	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})

	return client, mock
}

func queryPattern(query string) string {
	return regexp.QuoteMeta(query)
}

func affiliateSummaryRows(userID int64, inviterID *int64) *sqlmock.Rows {
	now := time.Now().UTC()
	var inviter any
	if inviterID != nil {
		inviter = *inviterID
	}
	return sqlmock.NewRows([]string{
		"user_id",
		"aff_code",
		"aff_code_custom",
		"aff_rebate_rate_percent",
		"inviter_id",
		"inviter_source",
		"aff_count",
		"aff_quota",
		"aff_frozen_quota",
		"aff_history_quota",
		"created_at",
		"updated_at",
	}).AddRow(
		userID,
		"AFFCODE",
		false,
		nil,
		inviter,
		"",
		0,
		0.0,
		0.0,
		0.0,
		now,
		now,
	)
}

func expectEnsureUserAffiliateWithNoRootAdmin(mock sqlmock.Sqlmock, userID int64, inviterID *int64) {
	mock.ExpectQuery(queryPattern(affiliateDistributionQueryAffiliateByUserIDSQL)).
		WithArgs(userID).
		WillReturnRows(affiliateSummaryRows(userID, inviterID))
	mock.ExpectQuery(queryPattern(affiliateDistributionQueryFirstActiveAdminIDSQL)).
		WithArgs(service.RoleAdmin, service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(queryPattern(affiliateDistributionQueryAffiliateByUserIDSQL)).
		WithArgs(userID).
		WillReturnRows(affiliateSummaryRows(userID, inviterID))
}
