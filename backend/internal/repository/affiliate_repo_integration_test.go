//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func querySingleFloat(t *testing.T, ctx context.Context, client *dbent.Client, query string, args ...any) float64 {
	t.Helper()
	rows, err := client.QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	require.True(t, rows.Next(), "expected one row")
	var value float64
	require.NoError(t, rows.Scan(&value))
	require.NoError(t, rows.Err())
	return value
}

func querySingleInt(t *testing.T, ctx context.Context, client *dbent.Client, query string, args ...any) int {
	t.Helper()
	rows, err := client.QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	require.True(t, rows.Next(), "expected one row")
	var value int
	require.NoError(t, rows.Scan(&value))
	require.NoError(t, rows.Err())
	return value
}

func TestAffiliateRepository_TransferQuotaToBalance_UsesClaimedQuotaBeforeClear(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-transfer-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      5.5,
		Concurrency:  5,
	})

	affCode := fmt.Sprintf("AFF%09d", time.Now().UnixNano()%1_000_000_000)
	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, $3, $3, NOW(), NOW())`, u.ID, affCode, 12.34)
	require.NoError(t, err)

	transferred, balance, err := repo.TransferQuotaToBalance(txCtx, u.ID)
	require.NoError(t, err)
	require.InDelta(t, 12.34, transferred, 1e-9)
	require.InDelta(t, 17.84, balance, 1e-9)

	affQuota := querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID)
	require.InDelta(t, 0.0, affQuota, 1e-9)

	persistedBalance := querySingleFloat(t, txCtx, client,
		"SELECT balance::double precision FROM users WHERE id = $1", u.ID)
	require.InDelta(t, 17.84, persistedBalance, 1e-9)

	ledgerCount := querySingleInt(t, txCtx, client,
		"SELECT COUNT(*) FROM user_affiliate_ledger WHERE user_id = $1 AND action = 'transfer'", u.ID)
	require.Equal(t, 1, ledgerCount)

	rows, err := client.QueryContext(txCtx, `
SELECT amount::double precision,
       balance_after::double precision,
       aff_quota_after::double precision,
       aff_frozen_quota_after::double precision,
       aff_history_quota_after::double precision
FROM user_affiliate_ledger
WHERE user_id = $1 AND action = 'transfer'
LIMIT 1`, u.ID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	require.True(t, rows.Next(), "expected transfer ledger")
	var amount, balanceAfter, quotaAfter, frozenAfter, historyAfter float64
	require.NoError(t, rows.Scan(&amount, &balanceAfter, &quotaAfter, &frozenAfter, &historyAfter))
	require.InDelta(t, 12.34, amount, 1e-9)
	require.InDelta(t, 17.84, balanceAfter, 1e-9)
	require.InDelta(t, 0.0, quotaAfter, 1e-9)
	require.InDelta(t, 0.0, frozenAfter, 1e-9)
	require.InDelta(t, 12.34, historyAfter, 1e-9)
}

// TestAffiliateRepository_AccrueQuota_ReusesOuterTransaction guards the
// cross-layer tx propagation invariant: when AccrueQuota is called with a ctx
// that already carries a transaction (via dbent.NewTxContext), repo.withTx
// must reuse that tx rather than opening a nested one. If this invariant
// breaks, AccrueQuota would commit independently and survive a rollback of
// the outer tx, which would violate payment_fulfillment's all-or-nothing
// semantics.
func TestAffiliateRepository_AccrueQuota_ReusesOuterTransaction(t *testing.T) {
	ctx := context.Background()

	outerTx, err := integrationEntClient.Tx(ctx)
	require.NoError(t, err, "begin outer tx")
	// Defensive cleanup: if any require.* below fires before the explicit
	// Rollback, this prevents the tx from leaking until container teardown.
	// Rollback is idempotent at the driver level (extra rollback returns an
	// error we ignore).
	t.Cleanup(func() { _ = outerTx.Rollback() })
	client := outerTx.Client()
	txCtx := dbent.NewTxContext(ctx, outerTx)

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-invitee-%d@example.com", time.Now().UnixNano()+1),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})

	repo := NewAffiliateRepository(client, integrationDB)
	_, err = repo.EnsureUserAffiliate(txCtx, inviter.ID)
	require.NoError(t, err)
	_, err = repo.EnsureUserAffiliate(txCtx, invitee.ID)
	require.NoError(t, err)

	bound, err := repo.BindInviter(txCtx, invitee.ID, inviter.ID)
	require.NoError(t, err)
	require.True(t, bound, "invitee must bind to inviter")

	applied, err := repo.AccrueQuota(txCtx, inviter.ID, invitee.ID, 3.5, 0, nil)
	require.NoError(t, err)
	require.True(t, applied, "AccrueQuota must report applied=true")

	// Visible inside the outer tx.
	innerQuota := querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", inviter.ID)
	require.InDelta(t, 3.5, innerQuota, 1e-9)

	// Roll back the outer tx; if AccrueQuota had opened its own inner tx and
	// committed it, the rows would still be visible to the global client.
	require.NoError(t, outerTx.Rollback())

	rows, err := integrationEntClient.QueryContext(ctx,
		"SELECT COUNT(*) FROM user_affiliates WHERE user_id IN ($1, $2)",
		inviter.ID, invitee.ID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	require.True(t, rows.Next())
	var postRollbackCount int
	require.NoError(t, rows.Scan(&postRollbackCount))
	require.Equal(t, 0, postRollbackCount,
		"AccrueQuota must propagate the outer tx — found persisted rows after rollback")
}

func TestAffiliateRepository_TransferQuotaToBalance_EmptyQuota(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-empty-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      3.21,
		Concurrency:  5,
	})

	affCode := fmt.Sprintf("AFF%09d", time.Now().UnixNano()%1_000_000_000)
	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, 0, 0, NOW(), NOW())`, u.ID, affCode)
	require.NoError(t, err)

	transferred, balance, err := repo.TransferQuotaToBalance(txCtx, u.ID)
	require.ErrorIs(t, err, service.ErrAffiliateQuotaEmpty)
	require.InDelta(t, 0.0, transferred, 1e-9)
	require.InDelta(t, 0.0, balance, 1e-9)

	persistedBalance := querySingleFloat(t, txCtx, client,
		"SELECT balance::double precision FROM users WHERE id = $1", u.ID)
	require.InDelta(t, 3.21, persistedBalance, 1e-9)
}

// TestAffiliateRepository_AdminCustomCode covers the success path of admin
// invite-code rewrite + reset within a shared test transaction:
// - UpdateUserAffCode replaces aff_code, sets aff_code_custom=true, lookup works
// - the old code can no longer be found
// - ResetUserAffCode reverts aff_code_custom and assigns a new system-format code
//
// The conflict path (duplicate code → ErrAffiliateCodeTaken) lives in its own
// test because a unique-violation aborts the surrounding Postgres tx, which
// would poison subsequent assertions in the same transaction.
func TestAffiliateRepository_AdminCustomCode(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-custom-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	original, err := repo.EnsureUserAffiliate(txCtx, u.ID)
	require.NoError(t, err)
	require.False(t, original.AffCodeCustom, "system-generated codes start as non-custom")
	originalCode := original.AffCode

	// Rewrite to a custom code
	customCode := fmt.Sprintf("VIP%09d", time.Now().UnixNano()%1_000_000_000)
	require.NoError(t, repo.UpdateUserAffCode(txCtx, u.ID, customCode))

	updated, err := repo.EnsureUserAffiliate(txCtx, u.ID)
	require.NoError(t, err)
	require.Equal(t, customCode, updated.AffCode)
	require.True(t, updated.AffCodeCustom)

	// Lookup by new custom code finds the user
	byCode, err := repo.GetAffiliateByCode(txCtx, customCode)
	require.NoError(t, err)
	require.Equal(t, u.ID, byCode.UserID)

	// Old system code should no longer match
	_, err = repo.GetAffiliateByCode(txCtx, originalCode)
	require.ErrorIs(t, err, service.ErrAffiliateProfileNotFound)

	// Reset back to a fresh system code, clears custom flag
	newSysCode, err := repo.ResetUserAffCode(txCtx, u.ID)
	require.NoError(t, err)
	require.NotEqual(t, customCode, newSysCode)

	reset, err := repo.EnsureUserAffiliate(txCtx, u.ID)
	require.NoError(t, err)
	require.Equal(t, newSysCode, reset.AffCode)
	require.False(t, reset.AffCodeCustom)

	// The old custom code is now free again
	_, err = repo.GetAffiliateByCode(txCtx, customCode)
	require.ErrorIs(t, err, service.ErrAffiliateProfileNotFound)
}

// TestAffiliateRepository_AdminCustomCode_Conflict isolates the unique-violation
// path. PostgreSQL aborts the enclosing tx when a unique constraint fires, so
// this test must be the only assertion and run in its own tx — production
// callers each have their own outer tx, so this matches real behavior.
func TestAffiliateRepository_AdminCustomCode_Conflict(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	taker := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-conflict-taker-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser, Status: service.StatusActive,
	})
	requester := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-conflict-req-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser, Status: service.StatusActive,
	})

	takenCode := fmt.Sprintf("HOT%09d", time.Now().UnixNano()%1_000_000_000)
	require.NoError(t, repo.UpdateUserAffCode(txCtx, taker.ID, takenCode))

	// Now requester tries to grab the same code → conflict.
	err := repo.UpdateUserAffCode(txCtx, requester.ID, takenCode)
	require.ErrorIs(t, err, service.ErrAffiliateCodeTaken)
}

// TestAffiliateRepository_AdminRebateRate covers per-user exclusive rate
// set/clear and the Batch variant including NULL semantics.
func TestAffiliateRepository_AdminRebateRate(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	u1 := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-rate-%d-a@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	u2 := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-rate-%d-b@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	// Set exclusive rate for u1
	rate := 42.5
	require.NoError(t, repo.SetUserRebateRate(txCtx, u1.ID, &rate))

	got, err := repo.EnsureUserAffiliate(txCtx, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, got.AffRebateRatePercent)
	require.InDelta(t, 42.5, *got.AffRebateRatePercent, 1e-9)

	// Clear exclusive rate
	require.NoError(t, repo.SetUserRebateRate(txCtx, u1.ID, nil))
	cleared, err := repo.EnsureUserAffiliate(txCtx, u1.ID)
	require.NoError(t, err)
	require.Nil(t, cleared.AffRebateRatePercent)

	// Batch set both users
	batchRate := 15.0
	require.NoError(t, repo.BatchSetUserRebateRate(txCtx, []int64{u1.ID, u2.ID}, &batchRate))

	for _, uid := range []int64{u1.ID, u2.ID} {
		v, err := repo.EnsureUserAffiliate(txCtx, uid)
		require.NoError(t, err)
		require.NotNil(t, v.AffRebateRatePercent)
		require.InDelta(t, 15.0, *v.AffRebateRatePercent, 1e-9)
	}

	// Batch clear
	require.NoError(t, repo.BatchSetUserRebateRate(txCtx, []int64{u1.ID, u2.ID}, nil))
	for _, uid := range []int64{u1.ID, u2.ID} {
		v, err := repo.EnsureUserAffiliate(txCtx, uid)
		require.NoError(t, err)
		require.Nil(t, v.AffRebateRatePercent)
	}
}

// TestAffiliateRepository_ListUsersWithCustomSettings verifies the admin list
// only includes users with at least one override applied.
func TestAffiliateRepository_ListUsersWithCustomSettings(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	// User without any custom config — should NOT appear in the list.
	plainEmail := fmt.Sprintf("affiliate-plain-%d@example.com", time.Now().UnixNano())
	uPlain := mustCreateUser(t, client, &service.User{
		Email: plainEmail, PasswordHash: "hash",
		Role: service.RoleUser, Status: service.StatusActive,
	})
	_, err := repo.EnsureUserAffiliate(txCtx, uPlain.ID)
	require.NoError(t, err)

	// User with a custom code — should appear.
	uCode := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-codeonly-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser, Status: service.StatusActive,
	})
	require.NoError(t, repo.UpdateUserAffCode(txCtx, uCode.ID, fmt.Sprintf("VIP%09d", time.Now().UnixNano()%1_000_000_000)))

	// User with only an exclusive rate — should appear.
	uRate := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-rateonly-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser, Status: service.StatusActive,
	})
	r := 33.3
	require.NoError(t, repo.SetUserRebateRate(txCtx, uRate.ID, &r))

	entries, total, err := repo.ListUsersWithCustomSettings(txCtx, service.AffiliateAdminFilter{
		Page: 1, PageSize: 100,
	})
	require.NoError(t, err)

	// Build a quick lookup to assert per-user attributes (other tests may have
	// inserted custom rows in the same DB; we only care about our 3).
	byUserID := make(map[int64]service.AffiliateAdminEntry, len(entries))
	for _, e := range entries {
		byUserID[e.UserID] = e
	}

	require.NotContains(t, byUserID, uPlain.ID, "users without overrides must not appear")

	codeEntry, ok := byUserID[uCode.ID]
	require.True(t, ok, "custom-code user missing from list")
	require.True(t, codeEntry.AffCodeCustom)
	require.Nil(t, codeEntry.AffRebateRatePercent)

	rateEntry, ok := byUserID[uRate.ID]
	require.True(t, ok, "custom-rate user missing from list")
	require.False(t, rateEntry.AffCodeCustom)
	require.NotNil(t, rateEntry.AffRebateRatePercent)
	require.InDelta(t, 33.3, *rateEntry.AffRebateRatePercent, 1e-9)

	require.GreaterOrEqual(t, total, int64(2), "total must include at least our 2 custom rows")
}

func TestAffiliateDistributionRepository_AgentPermissions_DefaultAndUpdate(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateDistributionRepository(client)

	operator := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-perm-operator-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
	})
	target := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-perm-target-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	permission, err := repo.GetAgentDistributionPermissions(txCtx, target.ID)
	require.NoError(t, err)
	require.Equal(t, target.ID, permission.UserID)
	require.False(t, permission.CanViewDownlineDailyRevenue)
	require.False(t, permission.CanViewDownlineRebateBalances)
	require.False(t, permission.CanManageDownlinePricing)
	require.Nil(t, permission.GrantedByUserID)

	updated, err := repo.UpdateAgentDistributionPermissions(txCtx, operator.ID, target.ID, service.UpdateAgentDistributionPermissionInput{
		CanViewDownlineDailyRevenue:   true,
		CanViewDownlineRebateBalances: true,
		CanManageDownlinePricing:      false,
	})
	require.NoError(t, err)
	require.Equal(t, target.ID, updated.UserID)
	require.True(t, updated.CanViewDownlineDailyRevenue)
	require.True(t, updated.CanViewDownlineRebateBalances)
	require.False(t, updated.CanManageDownlinePricing)
	require.NotNil(t, updated.GrantedByUserID)
	require.Equal(t, operator.ID, *updated.GrantedByUserID)
	require.NotNil(t, updated.CreatedAt)
	require.NotNil(t, updated.UpdatedAt)
}

func TestAffiliateDistributionRepository_ScopedRankingsAndTree(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	affiliateRepo := NewAffiliateRepository(client, integrationDB)
	distributionRepo := NewAffiliateDistributionRepository(client)

	root := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-root-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	child := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-child-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	leaf := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-leaf-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	otherRoot := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-other-root-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	otherLeaf := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-other-leaf-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	for _, user := range []*service.User{root, child, leaf, otherRoot, otherLeaf} {
		_, err := affiliateRepo.EnsureUserAffiliate(txCtx, user.ID)
		require.NoError(t, err)
	}
	bound, err := affiliateRepo.BindInviter(txCtx, child.ID, root.ID)
	require.NoError(t, err)
	require.True(t, bound)
	bound, err = affiliateRepo.BindInviter(txCtx, leaf.ID, child.ID)
	require.NoError(t, err)
	require.True(t, bound)
	bound, err = affiliateRepo.BindInviter(txCtx, otherLeaf.ID, otherRoot.ID)
	require.NoError(t, err)
	require.True(t, bound)

	_, err = distributionRepo.SaveInviteModelRates(txCtx, root.ID, []service.AgentModelRateInput{{Model: "gpt-4o", Multiplier: 1.2}})
	require.NoError(t, err)
	_, err = distributionRepo.SaveInviteModelRates(txCtx, child.ID, []service.AgentModelRateInput{{Model: "gpt-4o", Multiplier: 1.5}})
	require.NoError(t, err)
	_, err = distributionRepo.SaveInviteModelRates(txCtx, otherRoot.ID, []service.AgentModelRateInput{{Model: "gpt-4o", Multiplier: 1.3}})
	require.NoError(t, err)

	now := truncateToUTCDate(time.Now()).Add(2 * time.Hour)
	createAffiliateDistributionUsageLog(t, txCtx, client, leaf, now, "gpt-4o", 100)
	require.NoError(t, distributionRepo.SettleUsageDistribution(txCtx, service.AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     queryLatestUsageLogIDForUser(t, txCtx, client, leaf.ID),
		UserID:         leaf.ID,
		Model:          "gpt-4o",
		RequestedModel: "gpt-4o",
		TotalCost:      100,
		ActualCost:     100,
		UsageCreatedAt: now,
	}))

	createAffiliateDistributionUsageLog(t, txCtx, client, otherLeaf, now, "gpt-4o", 50)
	require.NoError(t, distributionRepo.SettleUsageDistribution(txCtx, service.AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     queryLatestUsageLogIDForUser(t, txCtx, client, otherLeaf.ID),
		UserID:         otherLeaf.ID,
		Model:          "gpt-4o",
		RequestedModel: "gpt-4o",
		TotalCost:      50,
		ActualCost:     50,
		UsageCreatedAt: now,
	}))

	rootScope := root.ID
	statDate := truncateToUTCDate(now)
	dailyItems, total, err := distributionRepo.ListDailyBusinessRanking(txCtx, service.AgentRankingFilter{
		Page:            1,
		PageSize:        20,
		StatDate:        &statDate,
		RootUserID:      &rootScope,
		OnlyDescendants: true,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, dailyItems, 1)
	require.Equal(t, child.ID, dailyItems[0].UserID)
	require.InDelta(t, 100.0, dailyItems[0].BusinessUSD, 1e-9)
	require.InDelta(t, 15.0, dailyItems[0].BusinessRMB, 1e-9)
	require.Equal(t, 1, dailyItems[0].DirectUsers)
	require.Equal(t, 0, dailyItems[0].DirectAgents)

	rebateItems, total, err := distributionRepo.ListRebateBalanceRanking(txCtx, service.AgentRankingFilter{
		Page:            1,
		PageSize:        20,
		RootUserID:      &rootScope,
		OnlyDescendants: true,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rebateItems, 1)
	require.Equal(t, child.ID, rebateItems[0].UserID)
	require.InDelta(t, 3.0, rebateItems[0].CurrentRebateBalanceRMB, 1e-9)
	require.InDelta(t, 3.0, rebateItems[0].TodayRebateRMB, 1e-9)
	require.InDelta(t, 3.0, rebateItems[0].MonthlyRebateRMB, 1e-9)
	require.Equal(t, 1, rebateItems[0].DirectUsers)
	require.Equal(t, 0, rebateItems[0].DirectAgents)

	tree, err := distributionRepo.GetDistributionTree(txCtx, service.AgentTreeFilter{
		RootUserID:      &rootScope,
		OnlyDescendants: true,
	})
	require.NoError(t, err)
	treeUserIDs := make(map[int64]struct{}, len(tree))
	for _, node := range tree {
		treeUserIDs[node.UserID] = struct{}{}
	}
	_, hasRoot := treeUserIDs[root.ID]
	_, hasChild := treeUserIDs[child.ID]
	_, hasLeaf := treeUserIDs[leaf.ID]
	_, hasOtherRoot := treeUserIDs[otherRoot.ID]
	require.False(t, hasRoot)
	require.True(t, hasChild)
	require.True(t, hasLeaf)
	require.False(t, hasOtherRoot)
}

func TestAffiliateDistributionRepository_AdminUpdateUserDistributionPricing_AllowsRecursiveDescendants(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	affiliateRepo := NewAffiliateRepository(client, integrationDB)
	distributionRepo := NewAffiliateDistributionRepository(client)

	root := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-manage-root-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	child := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-manage-child-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	grandchild := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-manage-grandchild-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	outsider := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-manage-outsider-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	for _, user := range []*service.User{root, child, grandchild, outsider} {
		_, err := affiliateRepo.EnsureUserAffiliate(txCtx, user.ID)
		require.NoError(t, err)
	}
	bound, err := affiliateRepo.BindInviter(txCtx, child.ID, root.ID)
	require.NoError(t, err)
	require.True(t, bound)
	bound, err = affiliateRepo.BindInviter(txCtx, grandchild.ID, child.ID)
	require.NoError(t, err)
	require.True(t, bound)

	_, err = distributionRepo.AdminUpdateUserDistributionPricing(txCtx, root.ID, grandchild.ID, []service.AgentModelRateInput{{
		Model:      "gpt-4o",
		Multiplier: 1.88,
	}})
	require.NoError(t, err)

	inviteRates, err := distributionRepo.ListInviteModelRates(txCtx, grandchild.ID)
	require.NoError(t, err)
	require.Len(t, inviteRates, 1)
	require.Equal(t, "gpt-4o", inviteRates[0].Model)
	require.InDelta(t, 1.88, inviteRates[0].Multiplier, 1e-9)

	_, err = distributionRepo.AdminUpdateUserDistributionPricing(txCtx, root.ID, outsider.ID, []service.AgentModelRateInput{{
		Model:      "gpt-4o",
		Multiplier: 1.99,
	}})
	require.Error(t, err)
}

func createAffiliateDistributionUsageLog(t *testing.T, ctx context.Context, client *dbent.Client, user *service.User, createdAt time.Time, model string, totalCost float64) {
	t.Helper()
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    fmt.Sprintf("sk-aff-dist-%d", time.Now().UnixNano()),
		Name:   "aff-dist",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: fmt.Sprintf("aff-dist-account-%d", time.Now().UnixNano()),
	})
	usageRepo := newUsageLogRepositoryWithSQL(client, dbent.TxFromContext(ctx))
	inserted, err := usageRepo.Create(ctx, &service.UsageLog{
		UserID:         user.ID,
		APIKeyID:       apiKey.ID,
		AccountID:      account.ID,
		RequestID:      fmt.Sprintf("aff-dist-usage-%d", time.Now().UnixNano()),
		Model:          model,
		RequestedModel: model,
		InputTokens:    10,
		OutputTokens:   20,
		TotalCost:      totalCost,
		ActualCost:     totalCost,
		CreatedAt:      createdAt,
	})
	require.NoError(t, err)
	require.True(t, inserted)
}

func queryLatestUsageLogIDForUser(t *testing.T, ctx context.Context, client *dbent.Client, userID int64) int64 {
	t.Helper()
	var usageLogID int64
	err := scanSingleRow(ctx, dbent.TxFromContext(ctx), `
SELECT id
FROM usage_logs
WHERE user_id = $1
ORDER BY id DESC
LIMIT 1`, []any{userID}, &usageLogID)
	require.NoError(t, err)
	return usageLogID
}
