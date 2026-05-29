package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewTraceExportTaskRepository(t *testing.T) {
	db, _ := newSQLMock(t)
	repo := NewTraceExportTaskRepository(db)
	require.NotNil(t, repo)
}

func TestTraceExportTaskRepositoryCreateListGetAndCancel(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &traceExportTaskRepository{sql: db}

	task := newTraceExportTaskFixture()
	createdAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt

	mock.ExpectQuery("INSERT INTO trace_export_tasks").
		WithArgs(
			task.Status,
			task.Format,
			sqlmock.AnyArg(),
			task.IncludeRaw,
			task.TargetRecords,
			task.RequestedBy,
			task.DownloadFilename,
			task.FilePath,
			task.FileSizeBytes,
			task.TotalRecords,
			task.ProcessedRecords,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(11), createdAt, updatedAt))

	err := repo.Create(context.Background(), task)
	require.NoError(t, err)
	require.Equal(t, int64(11), task.ID)
	require.Equal(t, createdAt, task.CreatedAt)
	require.Equal(t, updatedAt, task.UpdatedAt)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM trace_export_tasks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery("FROM trace_export_tasks").
		WithArgs(20, 0).
		WillReturnRows(traceExportTaskRows(task, createdAt, updatedAt))

	items, result, err := repo.List(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assertTraceExportTaskMatches(t, task, &items[0], createdAt, updatedAt)
	require.Equal(t, int64(1), result.Total)

	mock.ExpectQuery("FROM trace_export_tasks").
		WithArgs(task.ID).
		WillReturnRows(traceExportTaskRows(task, createdAt, updatedAt))

	got, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	assertTraceExportTaskMatches(t, task, got, createdAt, updatedAt)

	mock.ExpectQuery("UPDATE trace_export_tasks").
		WithArgs(service.TraceExportTaskStatusCanceled, task.ID, int64(9), sqlmock.AnyArg(), service.TraceExportTaskStatusPending, service.TraceExportTaskStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(task.ID))

	ok, err := repo.Cancel(context.Background(), task.ID, 9)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTraceExportTaskRepositoryGetByIDNotFound(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &traceExportTaskRepository{sql: db}

	mock.ExpectQuery("FROM trace_export_tasks").
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "format", "filters", "include_raw", "target_records", "requested_by",
			"download_filename", "file_path", "file_size_bytes", "total_records", "processed_records",
			"error_message", "canceled_by", "canceled_at", "started_at", "finished_at", "created_at", "updated_at",
		}))

	_, err := repo.GetByID(context.Background(), 99)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTraceExportTaskRepositoryCancelNoRows(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &traceExportTaskRepository{sql: db}

	mock.ExpectQuery("UPDATE trace_export_tasks").
		WithArgs(service.TraceExportTaskStatusCanceled, int64(7), int64(8), sqlmock.AnyArg(), service.TraceExportTaskStatusPending, service.TraceExportTaskStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	ok, err := repo.Cancel(context.Background(), 7, 8)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTraceExportTaskRepositoryExecutionLifecycle(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &traceExportTaskRepository{sql: db}

	task := newTraceExportTaskFixture()
	task.Status = service.TraceExportTaskStatusRunning
	task.ErrorMsg = nil
	createdAt := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	startedAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	updatedAt := startedAt
	task.StartedAt = &startedAt

	mock.ExpectQuery("WITH next_task AS").
		WithArgs(service.TraceExportTaskStatusPending, service.TraceExportTaskStatusRunning, startedAt).
		WillReturnRows(traceExportTaskRows(task, createdAt, updatedAt))

	claimed, err := repo.ClaimNextPending(context.Background(), startedAt)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, service.TraceExportTaskStatusRunning, claimed.Status)

	progressAt := startedAt.Add(2 * time.Minute)
	mock.ExpectQuery("UPDATE trace_export_tasks").
		WithArgs(task.ID, int64(8), int64(3), progressAt, service.TraceExportTaskStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(task.ID))

	ok, err := repo.UpdateProgress(context.Background(), task.ID, 8, 3, progressAt)
	require.NoError(t, err)
	require.True(t, ok)

	finishedAt := startedAt.Add(5 * time.Minute)
	mock.ExpectQuery("UPDATE trace_export_tasks").
		WithArgs(task.ID, service.TraceExportTaskStatusSucceeded, task.FilePath, task.FileSizeBytes, int64(8), int64(8), finishedAt, service.TraceExportTaskStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(task.ID))

	ok, err = repo.MarkSucceeded(context.Background(), task.ID, task.FilePath, task.FileSizeBytes, 8, 8, finishedAt)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTraceExportTaskRepositoryFailureAndCleanupHelpers(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &traceExportTaskRepository{sql: db}

	failedAt := time.Date(2026, 5, 27, 11, 0, 0, 0, time.UTC)
	mock.ExpectExec("UPDATE trace_export_tasks").
		WithArgs(service.TraceExportTaskStatusRunning, service.TraceExportTaskStatusFailed, "timeout", failedAt, failedAt.Add(-time.Hour)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	affected, err := repo.FailStaleRunning(context.Background(), failedAt.Add(-time.Hour), "timeout", failedAt)
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)

	task := newTraceExportTaskFixture()
	mock.ExpectQuery("FROM trace_export_tasks").
		WithArgs(failedAt, service.TraceExportTaskStatusSucceeded, service.TraceExportTaskStatusFailed, service.TraceExportTaskStatusCanceled, 10).
		WillReturnRows(traceExportTaskRows(task, failedAt.Add(-time.Hour), failedAt.Add(-time.Hour)))

	items, err := repo.ListReadyForFileCleanup(context.Background(), failedAt, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, task.FilePath, items[0].FilePath)

	mock.ExpectQuery("UPDATE trace_export_tasks").
		WithArgs(task.ID, failedAt).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(task.ID))

	ok, err := repo.ClearFileForTask(context.Background(), task.ID, failedAt)
	require.NoError(t, err)
	require.True(t, ok)

	mock.ExpectQuery("UPDATE trace_export_tasks").
		WithArgs(task.ID, service.TraceExportTaskStatusFailed, int64(8), int64(3), "boom", failedAt, service.TraceExportTaskStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(task.ID))

	ok, err = repo.MarkFailed(context.Background(), task.ID, 8, 3, "boom", failedAt)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, mock.ExpectationsWereMet())
}

func newTraceExportTaskFixture() *service.TraceExportTask {
	start := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	userID := int64(42)
	apiKeyID := int64(7)
	captureRuleID := int64(3)
	minTotalTokens := int64(100)
	maxTotalTokens := int64(200)
	errorMsg := "boom"
	startedAt := start.Add(2 * time.Hour)
	finishedAt := startedAt.Add(time.Hour)

	return &service.TraceExportTask{
		ID:               11,
		Status:           service.TraceExportTaskStatusSucceeded,
		Format:           service.TraceExportTaskFormatJSONArray,
		Filters:          service.TraceExportTaskFilters{Model: "gpt-4.1", UserID: &userID, APIKeyID: &apiKeyID, CaptureRuleID: &captureRuleID, StartTime: &start, EndTime: &end, Keyword: "incident", MinTotalTokens: &minTotalTokens, MaxTotalTokens: &maxTotalTokens},
		IncludeRaw:       true,
		TargetRecords:    500,
		RequestedBy:      1,
		DownloadFilename: "sub2api-trace-export-20260527100000.json",
		FilePath:         `D:\exports\trace-task-11.json`,
		FileSizeBytes:    256,
		TotalRecords:     8,
		ProcessedRecords: 8,
		ErrorMsg:         &errorMsg,
		StartedAt:        &startedAt,
		FinishedAt:       &finishedAt,
	}
}

func traceExportTaskRows(task *service.TraceExportTask, createdAt, updatedAt time.Time) *sqlmock.Rows {
	filtersJSON, err := json.Marshal(task.Filters)
	if err != nil {
		panic(err)
	}

	var (
		errorMsg   any
		canceledBy any
		canceledAt any
		startedAt  any
		finishedAt any
	)
	if task.ErrorMsg != nil {
		errorMsg = *task.ErrorMsg
	}
	if task.CanceledBy != nil {
		canceledBy = *task.CanceledBy
	}
	if task.CanceledAt != nil {
		canceledAt = *task.CanceledAt
	}
	if task.StartedAt != nil {
		startedAt = *task.StartedAt
	}
	if task.FinishedAt != nil {
		finishedAt = *task.FinishedAt
	}

	return sqlmock.NewRows([]string{
		"id", "status", "format", "filters", "include_raw", "target_records", "requested_by",
		"download_filename", "file_path", "file_size_bytes", "total_records", "processed_records",
		"error_message", "canceled_by", "canceled_at", "started_at", "finished_at", "created_at", "updated_at",
	}).AddRow(
		task.ID,
		task.Status,
		task.Format,
		filtersJSON,
		task.IncludeRaw,
		task.TargetRecords,
		task.RequestedBy,
		task.DownloadFilename,
		task.FilePath,
		task.FileSizeBytes,
		task.TotalRecords,
		task.ProcessedRecords,
		errorMsg,
		canceledBy,
		canceledAt,
		startedAt,
		finishedAt,
		createdAt,
		updatedAt,
	)
}

func assertTraceExportTaskMatches(t *testing.T, expected, actual *service.TraceExportTask, createdAt, updatedAt time.Time) {
	t.Helper()

	require.NotNil(t, actual)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Format, actual.Format)
	require.Equal(t, expected.Filters, actual.Filters)
	require.Equal(t, expected.IncludeRaw, actual.IncludeRaw)
	require.Equal(t, expected.TargetRecords, actual.TargetRecords)
	require.Equal(t, expected.RequestedBy, actual.RequestedBy)
	require.Equal(t, expected.DownloadFilename, actual.DownloadFilename)
	require.Equal(t, expected.FilePath, actual.FilePath)
	require.Equal(t, expected.FileSizeBytes, actual.FileSizeBytes)
	require.Equal(t, expected.TotalRecords, actual.TotalRecords)
	require.Equal(t, expected.ProcessedRecords, actual.ProcessedRecords)
	require.Equal(t, expected.ErrorMsg, actual.ErrorMsg)
	require.Equal(t, expected.CanceledBy, actual.CanceledBy)
	require.Equal(t, expected.CanceledAt, actual.CanceledAt)
	require.Equal(t, expected.StartedAt, actual.StartedAt)
	require.Equal(t, expected.FinishedAt, actual.FinishedAt)
	require.Equal(t, createdAt, actual.CreatedAt)
	require.Equal(t, updatedAt, actual.UpdatedAt)
}
