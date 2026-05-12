//go:build integration || contract

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageCleanupRepositoryIntegration_DeleteUsageLogsBatchProtectsAffiliatePaidUsageSettlements(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newUsageCleanupRepositoryWithSQL(nil, tx)

	userID := insertUsageCleanupContractUser(t, tx, "cleanup-protected")
	accountID := insertUsageCleanupContractAccount(t, tx, "cleanup-protected")
	apiKeyID := insertUsageCleanupContractAPIKey(t, tx, userID, "cleanup-protected")

	baseTime := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Second)
	protectedUsageID := insertUsageCleanupContractUsageLog(t, tx, userID, apiKeyID, accountID, "protected", baseTime)
	deletableUsageID := insertUsageCleanupContractUsageLog(t, tx, userID, apiKeyID, accountID, "deletable", baseTime.Add(5*time.Minute))

	_, err := tx.ExecContext(ctx, `
INSERT INTO affiliate_distribution_paid_usage_settlements (
	usage_log_id,
	user_id,
	usage_amount_usd,
	settlement_day,
	settled_at
) VALUES ($1, $2, $3, $4, $5)
`,
		protectedUsageID,
		userID,
		"1.25000000",
		baseTime,
		baseTime.Add(10*time.Minute),
	)
	require.NoError(t, err)

	deleted, err := repo.DeleteUsageLogsBatch(ctx, service.UsageCleanupFilters{
		StartTime: baseTime.Add(-time.Minute),
		EndTime:   baseTime.Add(10 * time.Minute),
	}, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted, "only the unreferenced usage log should be deleted")

	requireUsageCleanupContractRowExists(t, tx, "usage_logs", protectedUsageID, true)
	requireUsageCleanupContractRowExists(t, tx, "usage_logs", deletableUsageID, false)
	requireUsageCleanupContractRowExists(t, tx, "affiliate_distribution_paid_usage_settlements", protectedUsageID, true)
}

func insertUsageCleanupContractUser(t *testing.T, tx *sql.Tx, prefix string) int64 {
	t.Helper()

	email := fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
	var userID int64
	err := tx.QueryRowContext(context.Background(), `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ($1, 'hash', 'user', 'active', 0, 1)
RETURNING id
`, email).Scan(&userID)
	require.NoError(t, err)
	return userID
}

func insertUsageCleanupContractAccount(t *testing.T, tx *sql.Tx, prefix string) int64 {
	t.Helper()

	var accountID int64
	err := tx.QueryRowContext(context.Background(), `
INSERT INTO accounts (name, platform, type)
VALUES ($1, 'openai', 'apikey')
RETURNING id
`, fmt.Sprintf("%s-account-%d", prefix, time.Now().UnixNano())).Scan(&accountID)
	require.NoError(t, err)
	return accountID
}

func insertUsageCleanupContractAPIKey(t *testing.T, tx *sql.Tx, userID int64, prefix string) int64 {
	t.Helper()

	var apiKeyID int64
	err := tx.QueryRowContext(context.Background(), `
INSERT INTO api_keys (user_id, key, name, status)
VALUES ($1, $2, $3, 'active')
RETURNING id
`,
		userID,
		fmt.Sprintf("sk-%s-%d", prefix, time.Now().UnixNano()),
		fmt.Sprintf("%s-key", prefix),
	).Scan(&apiKeyID)
	require.NoError(t, err)
	return apiKeyID
}

func insertUsageCleanupContractUsageLog(t *testing.T, tx *sql.Tx, userID, apiKeyID, accountID int64, requestLabel string, createdAt time.Time) int64 {
	t.Helper()

	var usageLogID int64
	err := tx.QueryRowContext(context.Background(), `
INSERT INTO usage_logs (
	user_id,
	api_key_id,
	account_id,
	request_id,
	model,
	input_tokens,
	output_tokens,
	total_cost,
	actual_cost,
	created_at
)
VALUES ($1, $2, $3, $4, 'gpt-cleanup-test', 1, 1, 0.01, 0.01, $5)
RETURNING id
`,
		userID,
		apiKeyID,
		accountID,
		fmt.Sprintf("req-%s-%d", requestLabel, time.Now().UnixNano()),
		createdAt,
	).Scan(&usageLogID)
	require.NoError(t, err)
	return usageLogID
}

func requireUsageCleanupContractRowExists(t *testing.T, tx *sql.Tx, table string, id int64, expected bool) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), fmt.Sprintf(`
SELECT EXISTS (
	SELECT 1
	FROM %s
	WHERE id = $1
)
`, table), id).Scan(&exists)
	require.NoError(t, err)
	require.Equal(t, expected, exists, "unexpected row existence in %s for id=%d", table, id)
}
