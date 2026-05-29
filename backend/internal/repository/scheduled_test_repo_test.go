package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

const scheduledTestClaimDueQueryPattern = `WITH due AS \([\s\S]*FROM scheduled_test_plans[\s\S]*WHERE enabled = true AND next_run_at IS NOT NULL AND next_run_at <= \$1[\s\S]*LIMIT \$2[\s\S]*FOR UPDATE SKIP LOCKED[\s\S]*UPDATE scheduled_test_plans AS plans[\s\S]*SET next_run_at = \$3, updated_at = NOW\(\)[\s\S]*RETURNING plans.id`

func TestScheduledTestPlanRepositoryClaimDue(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewScheduledTestPlanRepository(db)

	now := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	leaseUntil := now.Add(6 * time.Minute)
	createdAt := now.Add(-time.Hour)
	updatedAt := now.Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(scheduledTestClaimDueQueryPattern).
		WithArgs(now, 2, leaseUntil).
		WillReturnRows(scheduledTestPlanRows().
			AddRow(
				int64(7),
				int64(11),
				"gpt-4.1",
				"* * * * *",
				true,
				25,
				true,
				nil,
				leaseUntil,
				createdAt,
				updatedAt,
			))

	mock.ExpectCommit()
	plans, err := repo.ClaimDue(context.Background(), now, leaseUntil, 2)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Equal(t, int64(7), plans[0].ID)
	require.Equal(t, int64(11), plans[0].AccountID)
	require.NotNil(t, plans[0].NextRunAt)
	require.Equal(t, leaseUntil, *plans[0].NextRunAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScheduledTestPlanRepositoryClaimDueReturnsNoneWhenNoPlanMatches(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewScheduledTestPlanRepository(db)

	now := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	leaseUntil := now.Add(6 * time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(scheduledTestClaimDueQueryPattern).
		WithArgs(now, 5, leaseUntil).
		WillReturnRows(scheduledTestPlanRows())

	mock.ExpectCommit()
	plans, err := repo.ClaimDue(context.Background(), now, leaseUntil, 5)
	require.NoError(t, err)
	require.Empty(t, plans)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScheduledTestPlanRepositoryClaimDueConcurrentCallsDoNotDuplicatePlan(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.MatchExpectationsInOrder(false)
	repo := NewScheduledTestPlanRepository(db)

	now := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	leaseUntil := now.Add(6 * time.Minute)
	createdAt := now.Add(-time.Hour)
	updatedAt := now.Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(scheduledTestClaimDueQueryPattern).
		WithArgs(now, 1, leaseUntil).
		WillReturnRows(scheduledTestPlanRows().
			AddRow(
				int64(9),
				int64(22),
				"gpt-4.1-mini",
				"* * * * *",
				true,
				10,
				false,
				nil,
				leaseUntil,
				createdAt,
				updatedAt,
			))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectQuery(scheduledTestClaimDueQueryPattern).
		WithArgs(now, 1, leaseUntil).
		WillReturnRows(scheduledTestPlanRows())
	mock.ExpectCommit()

	results := make(chan []int64, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plans, err := repo.ClaimDue(context.Background(), now, leaseUntil, 1)
			if err != nil {
				errs <- err
				return
			}

			ids := make([]int64, 0, len(plans))
			for _, plan := range plans {
				ids = append(ids, plan.ID)
			}
			results <- ids
		}()
	}

	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	claimedIDs := make(map[int64]int)
	totalClaims := 0
	for ids := range results {
		totalClaims += len(ids)
		for _, id := range ids {
			claimedIDs[id]++
		}
	}

	require.Equal(t, 1, totalClaims)
	require.Equal(t, map[int64]int{9: 1}, claimedIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func scheduledTestPlanRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"account_id",
		"model_id",
		"cron_expression",
		"enabled",
		"max_results",
		"auto_recover",
		"last_run_at",
		"next_run_at",
		"created_at",
		"updated_at",
	})
}

func TestScheduledTestPlanRepositoryUpdateAfterRunRequiresCurrentLease(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewScheduledTestPlanRepository(db)

	claimedUntil := time.Date(2026, 5, 28, 10, 6, 0, 0, time.UTC)
	lastRunAt := time.Date(2026, 5, 28, 10, 7, 0, 0, time.UTC)
	nextRunAt := time.Date(2026, 5, 28, 11, 0, 0, 0, time.UTC)

	mock.ExpectExec(`UPDATE scheduled_test_plans[\s\S]*WHERE id = \$1 AND next_run_at = \$2`).
		WithArgs(int64(8), claimedUntil, lastRunAt, nextRunAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := repo.UpdateAfterRun(context.Background(), 8, claimedUntil, lastRunAt, nextRunAt)
	require.NoError(t, err)
	require.True(t, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScheduledTestPlanRepositoryUpdateAfterRunReturnsFalseWhenLeaseOwnershipLost(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewScheduledTestPlanRepository(db)

	claimedUntil := time.Date(2026, 5, 28, 10, 6, 0, 0, time.UTC)
	lastRunAt := time.Date(2026, 5, 28, 10, 7, 0, 0, time.UTC)
	nextRunAt := time.Date(2026, 5, 28, 11, 0, 0, 0, time.UTC)

	mock.ExpectExec(`UPDATE scheduled_test_plans[\s\S]*WHERE id = \$1 AND next_run_at = \$2`).
		WithArgs(int64(8), claimedUntil, lastRunAt, nextRunAt).
		WillReturnResult(sqlmock.NewResult(0, 0))

	updated, err := repo.UpdateAfterRun(context.Background(), 8, claimedUntil, lastRunAt, nextRunAt)
	require.NoError(t, err)
	require.False(t, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}
