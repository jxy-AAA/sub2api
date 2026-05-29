package repository

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelInteractionTraceRepositoryCreateAndListByTimeRange(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelInteractionTraceRepository{sql: db}

	ctx := context.Background()
	userID := int64(42)
	apiKeyID := int64(7)
	requestID := "req-001"
	scaffoldVersion := "v1"
	model := "gpt-4.1"
	trace := &service.ModelInteractionTrace{
		TaskID:          "task-001",
		Prompt:          json.RawMessage(`[{"role":"system","content":"raw system"},{"role":"user","content":[{"type":"text","text":"hi"}]}]`),
		Candidates:      json.RawMessage(`[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"search","arguments":"{\"q\":\"x\"}"}}],"reasoning":"keep-this"}}]`),
		Tools:           json.RawMessage(`[{"type":"function","function":{"name":"search","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}}]`),
		Signature:       json.RawMessage(`{"value":"sig-123"}`),
		Meta:            json.RawMessage(`{"time":"2026-05-27T00:00:00Z","model":"gpt-4.1","endpoint":"/v1/responses"}`),
		Scaffold:        json.RawMessage(`{"blocks":[{"kind":"title","value":"Trace"}]}`),
		ScaffoldVersion: scaffoldVersion,
		Model:           &model,
		UserID:          &userID,
		APIKeyID:        &apiKeyID,
		RequestID:       &requestID,
		DedupeHash:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	createdAt := time.Date(2026, 5, 27, 12, 34, 56, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO model_interaction_traces")).
		WithArgs(
			trace.TaskID,
			string(trace.Prompt),
			string(trace.Candidates),
			string(trace.Tools),
			string(trace.Signature),
			string(trace.Meta),
			string(trace.Scaffold),
			scaffoldVersion,
			model,
			userID,
			apiKeyID,
			requestID,
			trace.DedupeHash,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(99), createdAt))

	inserted, err := repo.Create(ctx, trace)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, int64(99), trace.ID)
	require.Equal(t, createdAt, trace.CreatedAt)
	require.Equal(t, json.RawMessage(`[{"role":"system","content":"raw system"},{"role":"user","content":[{"type":"text","text":"hi"}]}]`), trace.Prompt)
	require.Equal(t, json.RawMessage(`[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"search","arguments":"{\"q\":\"x\"}"}}],"reasoning":"keep-this"}}]`), trace.Candidates)
	require.Equal(t, json.RawMessage(`[{"type":"function","function":{"name":"search","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}}]`), trace.Tools)
	require.Equal(t, json.RawMessage(`{"time":"2026-05-27T00:00:00Z","model":"gpt-4.1","endpoint":"/v1/responses"}`), trace.Meta)
	require.Equal(t, json.RawMessage(`{"blocks":[{"kind":"title","value":"Trace"}]}`), trace.Scaffold)

	start := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM model_interaction_traces")).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("FROM model_interaction_traces")).
		WithArgs(start, end, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_id", "prompt", "candidates", "tools", "signature", "meta", "scaffold",
			"scaffold_version", "model", "user_id", "api_key_id", "request_id", "dedupe_hash", "created_at",
		}).AddRow(
			int64(99),
			trace.TaskID,
			[]byte(trace.Prompt),
			[]byte(trace.Candidates),
			[]byte(trace.Tools),
			[]byte(trace.Signature),
			[]byte(trace.Meta),
			[]byte(trace.Scaffold),
			scaffoldVersion,
			model,
			userID,
			apiKeyID,
			requestID,
			trace.DedupeHash,
			createdAt,
		))

	items, result, err := repo.ListByTimeRange(ctx, start, end, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	got := items[0]
	require.Equal(t, trace.TaskID, got.TaskID)
	require.Equal(t, trace.Prompt, got.Prompt)
	require.Equal(t, trace.Candidates, got.Candidates)
	require.Equal(t, trace.Tools, got.Tools)
	require.Equal(t, trace.Signature, got.Signature)
	require.Equal(t, trace.Meta, got.Meta)
	require.Equal(t, trace.Scaffold, got.Scaffold)
	require.Equal(t, scaffoldVersion, got.ScaffoldVersion)
	require.NotNil(t, got.Model)
	require.Equal(t, model, *got.Model)
	require.NotNil(t, got.UserID)
	require.Equal(t, userID, *got.UserID)
	require.NotNil(t, got.APIKeyID)
	require.Equal(t, apiKeyID, *got.APIKeyID)
	require.NotNil(t, got.RequestID)
	require.Equal(t, requestID, *got.RequestID)
	require.Equal(t, trace.DedupeHash, got.DedupeHash)
	require.Equal(t, createdAt, got.CreatedAt)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelInteractionTraceRepositoryCreateDuplicateSkipsInsert(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelInteractionTraceRepository{sql: db}

	trace := &service.ModelInteractionTrace{
		TaskID:     "task-dup",
		DedupeHash: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO model_interaction_traces")).
		WithArgs(trace.TaskID, nil, nil, nil, nil, nil, nil, "", nil, nil, nil, nil, trace.DedupeHash).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}))

	inserted, err := repo.Create(context.Background(), trace)
	require.NoError(t, err)
	require.False(t, inserted)
	require.Zero(t, trace.ID)
	require.Zero(t, trace.CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelInteractionTraceRepositoryListByTimeRangeEmpty(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelInteractionTraceRepository{sql: db}

	start := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM model_interaction_traces")).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	items, result, err := repo.ListByTimeRange(context.Background(), start, end, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, items)
	require.Equal(t, int64(0), result.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}
