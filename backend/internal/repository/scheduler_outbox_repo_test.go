package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type schedulerOutboxExecStub struct {
	query string
	args  []any
	err   error
}

func (s *schedulerOutboxExecStub) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	s.query = query
	s.args = append([]any(nil), args...)
	return nil, s.err
}

func (s *schedulerOutboxExecStub) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, nil
}

func TestEnqueueSchedulerOutbox_DedupEventsUseCompositePredicate(t *testing.T) {
	accountID := int64(101)
	groupID := int64(202)
	exec := &schedulerOutboxExecStub{}

	err := enqueueSchedulerOutbox(
		context.Background(),
		exec,
		service.SchedulerOutboxEventAccountChanged,
		&accountID,
		&groupID,
		map[string]any{"group_ids": []int64{groupID}},
	)
	require.NoError(t, err)
	require.Contains(t, exec.query, "WHERE NOT EXISTS")
	require.Contains(t, exec.query, "event_type = $1")
	require.Contains(t, exec.query, "account_id IS NOT DISTINCT FROM $2")
	require.Contains(t, exec.query, "group_id IS NOT DISTINCT FROM $3")
	require.Contains(t, exec.query, "created_at >= NOW() - make_interval")
	require.Len(t, exec.args, 5)
	require.Equal(t, schedulerOutboxDedupWindow.Seconds(), exec.args[4])
}

func TestEnqueueSchedulerOutbox_NonDedupEventsUsePlainInsert(t *testing.T) {
	exec := &schedulerOutboxExecStub{}

	err := enqueueSchedulerOutbox(
		context.Background(),
		exec,
		service.SchedulerOutboxEventAccountLastUsed,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(exec.query), "where not exists")
	require.Len(t, exec.args, 4)
}

func TestSchedulerOutboxEventSupportsDedup(t *testing.T) {
	require.True(t, schedulerOutboxEventSupportsDedup(service.SchedulerOutboxEventAccountChanged))
	require.True(t, schedulerOutboxEventSupportsDedup(service.SchedulerOutboxEventGroupChanged))
	require.True(t, schedulerOutboxEventSupportsDedup(service.SchedulerOutboxEventFullRebuild))
	require.False(t, schedulerOutboxEventSupportsDedup(service.SchedulerOutboxEventAccountLastUsed))
}
