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

func querySingleString(t *testing.T, ctx context.Context, client *dbent.Client, query string, args ...any) string {
	t.Helper()
	rows, err := client.QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	require.True(t, rows.Next(), "expected one row")
	var value string
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
	groupPositive := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-group-positive-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   1.05,
	})
	groupNegative := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-group-negative-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   1.25,
	})

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

	_, err = distributionRepo.SaveInviteGroupRates(txCtx, root.ID, []service.AgentGroupRateInput{
		{GroupID: groupPositive.ID, RateMultiplier: 1.2},
		{GroupID: groupNegative.ID, RateMultiplier: 1.4},
	})
	require.NoError(t, err)
	_, err = distributionRepo.SaveInviteGroupRates(txCtx, child.ID, []service.AgentGroupRateInput{
		{GroupID: groupPositive.ID, RateMultiplier: 1.5},
		{GroupID: groupNegative.ID, RateMultiplier: 1.3},
	})
	require.NoError(t, err)
	_, err = distributionRepo.SaveInviteGroupRates(txCtx, otherRoot.ID, []service.AgentGroupRateInput{
		{GroupID: groupPositive.ID, RateMultiplier: 1.3},
	})
	require.NoError(t, err)

	now := truncateToUTCDate(time.Now()).Add(2 * time.Hour)
	createAffiliateDistributionUsageLog(t, txCtx, client, leaf, now, "gpt-4o", 100)
	require.NoError(t, distributionRepo.SettleUsageDistribution(txCtx, service.AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     queryLatestUsageLogIDForUser(t, txCtx, client, leaf.ID),
		UserID:         leaf.ID,
		GroupID:        groupPositive.ID,
		Model:          "gpt-4o",
		RequestedModel: "gpt-4o",
		TotalCost:      100,
		ActualCost:     100,
		UsageCreatedAt: now,
	}))
	createAffiliateDistributionUsageLog(t, txCtx, client, leaf, now.Add(5*time.Minute), "claude-3-5-sonnet", 50)
	require.NoError(t, distributionRepo.SettleUsageDistribution(txCtx, service.AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     queryLatestUsageLogIDForUser(t, txCtx, client, leaf.ID),
		UserID:         leaf.ID,
		GroupID:        groupNegative.ID,
		Model:          "claude-3-5-sonnet",
		RequestedModel: "claude-3-5-sonnet",
		TotalCost:      50,
		ActualCost:     50,
		UsageCreatedAt: now.Add(5 * time.Minute),
	}))

	createAffiliateDistributionUsageLog(t, txCtx, client, otherLeaf, now, "gpt-4o", 50)
	require.NoError(t, distributionRepo.SettleUsageDistribution(txCtx, service.AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     queryLatestUsageLogIDForUser(t, txCtx, client, otherLeaf.ID),
		UserID:         otherLeaf.ID,
		GroupID:        groupPositive.ID,
		Model:          "gpt-4o",
		RequestedModel: "gpt-4o",
		TotalCost:      50,
		ActualCost:     50,
		UsageCreatedAt: now,
	}))

	childBusinessUSD, childRebateRMB, err := distributionRepo.getDailyTotals(txCtx, child.ID, truncateToUTCDate(now))
	require.NoError(t, err)
	require.InDelta(t, 150.0, childBusinessUSD, 1e-9)
	require.InDelta(t, 3.0, childRebateRMB, 1e-9)

	rootBusinessUSD, rootRebateRMB, err := distributionRepo.getDailyTotals(txCtx, root.ID, truncateToUTCDate(now))
	require.NoError(t, err)
	require.InDelta(t, 150.0, rootBusinessUSD, 1e-9)
	require.InDelta(t, 4.0, rootRebateRMB, 1e-9)

	childBalance, err := distributionRepo.getCurrentRebateBalance(txCtx, child.ID)
	require.NoError(t, err)
	require.InDelta(t, 3.0, childBalance, 1e-9)

	directSubordinates, err := distributionRepo.ListDirectSubordinates(txCtx, root.ID)
	require.NoError(t, err)
	require.Len(t, directSubordinates, 1)
	require.Equal(t, child.ID, directSubordinates[0].UserID)
	require.InDelta(t, 150.0, directSubordinates[0].TodayBusinessUSD, 1e-9)
	require.InDelta(t, 3.0, directSubordinates[0].TodayRebateRMB, 1e-9)

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
	require.InDelta(t, 150.0, dailyItems[0].BusinessUSD, 1e-9)
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

func TestAffiliateDistributionRepository_ListDefaultUserGroupRates_FallsBackToGroupMultiplier(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	distributionRepo := NewAffiliateDistributionRepository(client)
	explicitGroup := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-default-explicit-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   1.6,
	})
	inheritedGroup := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-default-inherited-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   2.2,
	})
	inactiveGroup := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-default-inactive-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusDisabled,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   9.9,
	})

	rates, err := distributionRepo.SaveDefaultUserGroupRates(txCtx, []service.AgentGroupRateInput{{
		GroupID:        explicitGroup.ID,
		RateMultiplier: 1.9,
	}})
	require.NoError(t, err)

	byGroupID := make(map[int64]service.AgentGroupRate, len(rates))
	for _, rate := range rates {
		byGroupID[rate.GroupID] = rate
	}

	explicitRate, ok := byGroupID[explicitGroup.ID]
	require.True(t, ok)
	require.InDelta(t, 1.6, explicitRate.GroupDefaultRateMultiplier, 1e-9)
	require.InDelta(t, 1.9, explicitRate.RateMultiplier, 1e-9)
	require.Equal(t, "default_explicit", explicitRate.SourceType)

	inheritedRate, ok := byGroupID[inheritedGroup.ID]
	require.True(t, ok)
	require.InDelta(t, 2.2, inheritedRate.GroupDefaultRateMultiplier, 1e-9)
	require.InDelta(t, 2.2, inheritedRate.RateMultiplier, 1e-9)
	require.Equal(t, "group_default", inheritedRate.SourceType)

	_, hasInactive := byGroupID[inactiveGroup.ID]
	require.False(t, hasInactive)
}

func TestAffiliateDistributionRepository_AdminUpdateUserDistributionGroupRates_AllowsRecursiveDescendants(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	affiliateRepo := NewAffiliateRepository(client, integrationDB)
	distributionRepo := NewAffiliateDistributionRepository(client)
	targetGroup := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-manage-group-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   1.1,
	})

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

	updatedRates, err := distributionRepo.AdminUpdateUserDistributionGroupRates(txCtx, root.ID, grandchild.ID, []service.AgentGroupRateInput{{
		GroupID:        targetGroup.ID,
		RateMultiplier: 1.88,
	}})
	require.NoError(t, err)

	byGroupID := make(map[int64]service.AgentGroupRate, len(updatedRates))
	for _, rate := range updatedRates {
		byGroupID[rate.GroupID] = rate
	}
	require.Contains(t, byGroupID, targetGroup.ID)
	require.InDelta(t, 1.88, byGroupID[targetGroup.ID].RateMultiplier, 1e-9)

	currentRates, err := distributionRepo.GetUserDistributionGroupRates(txCtx, grandchild.ID)
	require.NoError(t, err)
	byGroupID = make(map[int64]service.AgentGroupRate, len(currentRates))
	for _, rate := range currentRates {
		byGroupID[rate.GroupID] = rate
	}
	require.Contains(t, byGroupID, targetGroup.ID)
	require.InDelta(t, 1.88, byGroupID[targetGroup.ID].RateMultiplier, 1e-9)

	_, err = distributionRepo.AdminUpdateUserDistributionGroupRates(txCtx, root.ID, outsider.ID, []service.AgentGroupRateInput{{
		GroupID:        targetGroup.ID,
		RateMultiplier: 1.99,
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

func TestAffiliateRepository_BindInviter_DefaultRootCanRebindButManualBindingsCannot(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	affiliateRepo := NewAffiliateRepository(client, integrationDB)

	rootAdmin := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-root-admin-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
	})
	realInviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-real-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	firstInviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-first-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	defaultRootUser := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-default-root-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	manualRootUser := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-manual-root-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	directBoundUser := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-direct-bound-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	for _, user := range []*service.User{rootAdmin, realInviter, firstInviter, defaultRootUser, manualRootUser, directBoundUser} {
		_, err := affiliateRepo.EnsureUserAffiliate(txCtx, user.ID)
		require.NoError(t, err)
	}

	defaultSummary, err := affiliateRepo.EnsureUserAffiliate(txCtx, defaultRootUser.ID)
	require.NoError(t, err)
	require.NotNil(t, defaultSummary.InviterID)
	require.Equal(t, rootAdmin.ID, *defaultSummary.InviterID)
	require.Equal(t, affiliateInviterSourceDefaultRoot, defaultSummary.InviterSource)

	bound, err := affiliateRepo.BindInviter(txCtx, defaultRootUser.ID, realInviter.ID)
	require.NoError(t, err)
	require.True(t, bound)

	updatedDefaultSummary, err := affiliateRepo.EnsureUserAffiliate(txCtx, defaultRootUser.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedDefaultSummary.InviterID)
	require.Equal(t, realInviter.ID, *updatedDefaultSummary.InviterID)
	require.Equal(t, affiliateInviterSourceAffiliateCode, updatedDefaultSummary.InviterSource)

	manualSummary, err := affiliateRepo.EnsureUserAffiliate(txCtx, manualRootUser.ID)
	require.NoError(t, err)
	require.NotNil(t, manualSummary.InviterID)
	require.Equal(t, rootAdmin.ID, *manualSummary.InviterID)
	require.Equal(t, affiliateInviterSourceDefaultRoot, manualSummary.InviterSource)

	_, err = client.ExecContext(txCtx, `
UPDATE user_affiliates
SET inviter_source = $2,
    updated_at = NOW()
WHERE user_id = $1`, manualRootUser.ID, affiliateDistributionSourceAdminOverride)
	require.NoError(t, err)

	bound, err = affiliateRepo.BindInviter(txCtx, manualRootUser.ID, realInviter.ID)
	require.NoError(t, err)
	require.False(t, bound)

	updatedManualSummary, err := affiliateRepo.EnsureUserAffiliate(txCtx, manualRootUser.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedManualSummary.InviterID)
	require.Equal(t, rootAdmin.ID, *updatedManualSummary.InviterID)
	require.Equal(t, affiliateDistributionSourceAdminOverride, updatedManualSummary.InviterSource)

	bound, err = affiliateRepo.BindInviter(txCtx, directBoundUser.ID, firstInviter.ID)
	require.NoError(t, err)
	require.True(t, bound)

	bound, err = affiliateRepo.BindInviter(txCtx, directBoundUser.ID, realInviter.ID)
	require.NoError(t, err)
	require.False(t, bound)

	updatedDirectSummary, err := affiliateRepo.EnsureUserAffiliate(txCtx, directBoundUser.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedDirectSummary.InviterID)
	require.Equal(t, firstInviter.ID, *updatedDirectSummary.InviterID)
	require.Equal(t, affiliateInviterSourceAffiliateCode, updatedDirectSummary.InviterSource)
}

func TestAffiliateDistributionRepository_GetUserDistributionGroupRates_UsesDynamicInheritanceAndIgnoresCachedRows(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	affiliateRepo := NewAffiliateRepository(client, integrationDB)
	distributionRepo := NewAffiliateDistributionRepository(client)
	group := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("aff-dist-inherit-%d", time.Now().UnixNano()),
		Platform:         service.PlatformAnthropic,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		RateMultiplier:   1.05,
	})

	root := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-inherit-root-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	child := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-inherit-child-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	leaf := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-inherit-leaf-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	for _, user := range []*service.User{root, child, leaf} {
		_, err := affiliateRepo.EnsureUserAffiliate(txCtx, user.ID)
		require.NoError(t, err)
	}
	bound, err := affiliateRepo.BindInviter(txCtx, child.ID, root.ID)
	require.NoError(t, err)
	require.True(t, bound)
	bound, err = affiliateRepo.BindInviter(txCtx, leaf.ID, child.ID)
	require.NoError(t, err)
	require.True(t, bound)

	_, err = distributionRepo.SaveInviteGroupRates(txCtx, root.ID, []service.AgentGroupRateInput{{
		GroupID:        group.ID,
		RateMultiplier: 1.2,
	}})
	require.NoError(t, err)

	childRates, err := distributionRepo.GetUserDistributionGroupRates(txCtx, child.ID)
	require.NoError(t, err)
	childRateByGroup := make(map[int64]service.AgentGroupRate, len(childRates))
	for _, rate := range childRates {
		childRateByGroup[rate.GroupID] = rate
	}
	require.Contains(t, childRateByGroup, group.ID)
	require.InDelta(t, 1.2, childRateByGroup[group.ID].RateMultiplier, 1e-9)
	require.Equal(t, affiliateDistributionSourceInviteCode, childRateByGroup[group.ID].SourceType)

	leafRates, err := distributionRepo.GetUserDistributionGroupRates(txCtx, leaf.ID)
	require.NoError(t, err)
	leafRateByGroup := make(map[int64]service.AgentGroupRate, len(leafRates))
	for _, rate := range leafRates {
		leafRateByGroup[rate.GroupID] = rate
	}
	require.Contains(t, leafRateByGroup, group.ID)
	require.InDelta(t, 1.2, leafRateByGroup[group.ID].RateMultiplier, 1e-9)
	require.Equal(t, affiliateDistributionSourceGroupInherited, leafRateByGroup[group.ID].SourceType)

	cachedRows := querySingleInt(t, txCtx, client,
		`SELECT COUNT(*) FROM affiliate_distribution_user_group_rates WHERE user_id IN ($1, $2)`,
		child.ID, leaf.ID,
	)
	require.Equal(t, 0, cachedRows)

	childSummary, err := affiliateRepo.EnsureUserAffiliate(txCtx, child.ID)
	require.NoError(t, err)
	_, err = client.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_user_group_rates (
    user_id, upstream_user_id, group_id, rate_multiplier, source_aff_code, source_type, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (user_id, group_id)
DO UPDATE SET
    upstream_user_id = EXCLUDED.upstream_user_id,
    rate_multiplier = EXCLUDED.rate_multiplier,
    source_aff_code = EXCLUDED.source_aff_code,
    source_type = EXCLUDED.source_type,
    updated_at = NOW()`,
		leaf.ID, child.ID, group.ID, 9.9, childSummary.AffCode, affiliateDistributionSourceGroupInherited,
	)
	require.NoError(t, err)

	_, err = distributionRepo.SaveInviteGroupRates(txCtx, root.ID, []service.AgentGroupRateInput{{
		GroupID:        group.ID,
		RateMultiplier: 1.4,
	}})
	require.NoError(t, err)

	updatedLeafRates, err := distributionRepo.GetUserDistributionGroupRates(txCtx, leaf.ID)
	require.NoError(t, err)
	updatedLeafRateByGroup := make(map[int64]service.AgentGroupRate, len(updatedLeafRates))
	for _, rate := range updatedLeafRates {
		updatedLeafRateByGroup[rate.GroupID] = rate
	}
	require.Contains(t, updatedLeafRateByGroup, group.ID)
	require.InDelta(t, 1.4, updatedLeafRateByGroup[group.ID].RateMultiplier, 1e-9)
	require.Equal(t, affiliateDistributionSourceGroupInherited, updatedLeafRateByGroup[group.ID].SourceType)
}

func TestAffiliateDistributionSettlementService_MissingGroupIDMarksUsageAsFailed(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	distributionRepo := NewAffiliateDistributionRepository(client)
	settlementSvc := service.NewAffiliateDistributionSettlementService(distributionRepo, distributionRepo)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-missing-group-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})

	createAffiliateDistributionUsageLog(t, txCtx, client, user, time.Now().UTC(), "gpt-4o", 12.5)
	usageLogID := queryLatestUsageLogIDForUser(t, txCtx, client, user.ID)

	applied, err := settlementSvc.SettleUsage(txCtx, service.AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     usageLogID,
		UserID:         user.ID,
		GroupID:        0,
		Model:          "gpt-4o",
		RequestedModel: "gpt-4o",
		TotalCost:      12.5,
		ActualCost:     12.5,
		UsageCreatedAt: time.Now().UTC(),
	})
	require.ErrorIs(t, err, service.ErrAffiliateDistributionGroupIDRequired)
	require.False(t, applied)

	status := querySingleString(t, txCtx, client,
		`SELECT status FROM affiliate_distribution_usage_jobs WHERE usage_log_id = $1`,
		usageLogID,
	)
	require.Equal(t, "failed", status)
}

func TestAffiliateDistributionRepository_TryBeginUsageSettlement_RetriesFailedAndExpiredProcessingAndKeepsDoneIdempotent(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	distributionRepo := NewAffiliateDistributionRepository(client)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("aff-dist-job-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	now := time.Now().UTC()

	createAffiliateDistributionUsageLog(t, txCtx, client, user, now, "gpt-4o", 1)
	failedUsageLogID := queryLatestUsageLogIDForUser(t, txCtx, client, user.ID)
	claimed, err := distributionRepo.TryBeginUsageSettlement(txCtx, failedUsageLogID)
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, distributionRepo.MarkUsageSettlementFailed(txCtx, failedUsageLogID))
	claimed, err = distributionRepo.TryBeginUsageSettlement(txCtx, failedUsageLogID)
	require.NoError(t, err)
	require.True(t, claimed)

	createAffiliateDistributionUsageLog(t, txCtx, client, user, now.Add(time.Minute), "gpt-4o", 1)
	staleUsageLogID := queryLatestUsageLogIDForUser(t, txCtx, client, user.ID)
	staleAt := now.Add(-2 * affiliateDistributionUsageJobClaimTTL)
	_, err = client.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_usage_jobs (usage_log_id, status, claimed_at, updated_at)
VALUES ($1, 'processing', $2, $2)`, staleUsageLogID, staleAt)
	require.NoError(t, err)
	claimed, err = distributionRepo.TryBeginUsageSettlement(txCtx, staleUsageLogID)
	require.NoError(t, err)
	require.True(t, claimed)

	createAffiliateDistributionUsageLog(t, txCtx, client, user, now.Add(2*time.Minute), "gpt-4o", 1)
	freshUsageLogID := queryLatestUsageLogIDForUser(t, txCtx, client, user.ID)
	_, err = client.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_usage_jobs (usage_log_id, status, claimed_at, updated_at)
VALUES ($1, 'processing', $2, $2)`, freshUsageLogID, now)
	require.NoError(t, err)
	claimed, err = distributionRepo.TryBeginUsageSettlement(txCtx, freshUsageLogID)
	require.NoError(t, err)
	require.False(t, claimed)

	createAffiliateDistributionUsageLog(t, txCtx, client, user, now.Add(3*time.Minute), "gpt-4o", 1)
	doneUsageLogID := queryLatestUsageLogIDForUser(t, txCtx, client, user.ID)
	claimed, err = distributionRepo.TryBeginUsageSettlement(txCtx, doneUsageLogID)
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, distributionRepo.MarkUsageSettlementDone(txCtx, doneUsageLogID))
	claimed, err = distributionRepo.TryBeginUsageSettlement(txCtx, doneUsageLogID)
	require.NoError(t, err)
	require.False(t, claimed)
}
