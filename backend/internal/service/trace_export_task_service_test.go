package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type traceExportTaskRepoStub struct {
	createdTask     *TraceExportTask
	listTasks       []TraceExportTask
	listResult      *pagination.PaginationResult
	listErr         error
	taskByID        map[int64]*TraceExportTask
	getErr          error
	cancelOK        bool
	cancelErr       error
	downloadMarkID  int64
	downloadedAt    *time.Time
	markDownloadErr error
}

func (s *traceExportTaskRepoStub) Create(ctx context.Context, task *TraceExportTask) error {
	if s.getErr != nil {
		return s.getErr
	}
	cloned := *task
	cloned.ID = 99
	if cloned.CreatedAt.IsZero() {
		cloned.CreatedAt = time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	}
	cloned.UpdatedAt = cloned.CreatedAt
	s.createdTask = &cloned
	*task = cloned
	return nil
}

func (s *traceExportTaskRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]TraceExportTask, *pagination.PaginationResult, error) {
	return s.listTasks, s.listResult, s.listErr
}

func (s *traceExportTaskRepoStub) GetByID(ctx context.Context, id int64) (*TraceExportTask, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.taskByID == nil {
		return nil, sql.ErrNoRows
	}
	task, ok := s.taskByID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cloned := *task
	return &cloned, nil
}

func (s *traceExportTaskRepoStub) Cancel(ctx context.Context, id int64, canceledBy int64) (bool, error) {
	return s.cancelOK, s.cancelErr
}

func (s *traceExportTaskRepoStub) ClaimNextPending(ctx context.Context, startedAt time.Time) (*TraceExportTask, error) {
	return nil, sql.ErrNoRows
}

func (s *traceExportTaskRepoStub) UpdateProgress(ctx context.Context, id int64, totalRecords, processedRecords int64, updatedAt time.Time) (bool, error) {
	return true, nil
}

func (s *traceExportTaskRepoStub) MarkSucceeded(ctx context.Context, id int64, filePath string, fileSizeBytes, totalRecords, processedRecords int64, finishedAt time.Time) (bool, error) {
	return true, nil
}

func (s *traceExportTaskRepoStub) MarkFailed(ctx context.Context, id int64, totalRecords, processedRecords int64, errorMessage string, finishedAt time.Time) (bool, error) {
	return true, nil
}

func (s *traceExportTaskRepoStub) MarkDownloaded(ctx context.Context, id int64, downloadedAt time.Time) (bool, error) {
	if s.markDownloadErr != nil {
		return false, s.markDownloadErr
	}
	s.downloadMarkID = id
	at := downloadedAt.UTC()
	s.downloadedAt = &at
	if task := s.taskByID[id]; task != nil && task.DownloadedAt == nil {
		task.DownloadedAt = &at
	}
	return true, nil
}

func (s *traceExportTaskRepoStub) FailStaleRunning(ctx context.Context, staleBefore time.Time, errorMessage string, failedAt time.Time) (int64, error) {
	return 0, nil
}

func (s *traceExportTaskRepoStub) ListReadyForFileCleanup(ctx context.Context, finishedBefore time.Time, limit int) ([]TraceExportTask, error) {
	return nil, nil
}

func (s *traceExportTaskRepoStub) ClearFileForTask(ctx context.Context, id int64, updatedAt time.Time) (bool, error) {
	return false, nil
}

func TestTraceExportTaskServiceCreateTaskNormalizesFilters(t *testing.T) {
	repo := &traceExportTaskRepoStub{}
	svc := NewTraceExportTaskService(repo)

	userID := int64(42)
	minTotal := int64(10)
	maxTotal := int64(50)
	start := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	task, err := svc.CreateTask(context.Background(), TraceExportTaskFilters{
		Model:          "  gpt-4.1  ",
		UserID:         &userID,
		StartTime:      &start,
		EndTime:        &end,
		Keyword:        "  incident  ",
		MinTotalTokens: &minTotal,
		MaxTotalTokens: &maxTotal,
	}, true, 250, 7)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, TraceExportTaskStatusPending, task.Status)
	require.Equal(t, TraceExportTaskFormatJSONArray, task.Format)
	require.Equal(t, "gpt-4.1", task.Filters.Model)
	require.Equal(t, "incident", task.Filters.Keyword)
	require.True(t, task.IncludeRaw)
	require.Equal(t, int64(250), task.TargetRecords)
	require.Equal(t, int64(7), task.RequestedBy)
	require.NotNil(t, repo.createdTask)
	require.Equal(t, task.ID, repo.createdTask.ID)
}

func TestTraceExportTaskServiceCancelTask(t *testing.T) {
	repo := &traceExportTaskRepoStub{
		taskByID: map[int64]*TraceExportTask{
			1: {ID: 1, Status: TraceExportTaskStatusPending},
			2: {ID: 2, Status: TraceExportTaskStatusSucceeded},
		},
		cancelOK: true,
	}
	svc := NewTraceExportTaskService(repo)

	require.NoError(t, svc.CancelTask(context.Background(), 1, 9))

	err := svc.CancelTask(context.Background(), 2, 9)
	require.Error(t, err)
}

func TestTraceExportTaskServiceOpenDownload(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "trace-export.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"task_id":"task-1"}]`), 0o600))

	repo := &traceExportTaskRepoStub{
		taskByID: map[int64]*TraceExportTask{
			8: {
				ID:               8,
				Status:           TraceExportTaskStatusSucceeded,
				Format:           TraceExportTaskFormatJSONArray,
				FilePath:         path,
				DownloadFilename: "trace-export.json",
			},
		},
	}
	svc := NewTraceExportTaskService(repo)

	download, err := svc.OpenDownload(context.Background(), 8)
	require.NoError(t, err)
	require.NotNil(t, download)
	require.Equal(t, "trace-export.json", download.Filename)
	require.Equal(t, "application/json; charset=utf-8", download.ContentType)
	require.Equal(t, int64(8), repo.downloadMarkID)
	require.NotNil(t, repo.downloadedAt)
	body, err := io.ReadAll(download.Reader)
	require.NoError(t, err)
	require.Equal(t, `[{"task_id":"task-1"}]`, string(body))
	require.NoError(t, download.Reader.Close())
}

func TestTraceExportTaskServiceOpenDownloadMapsMissingFile(t *testing.T) {
	repo := &traceExportTaskRepoStub{
		taskByID: map[int64]*TraceExportTask{
			9: {
				ID:       9,
				Status:   TraceExportTaskStatusSucceeded,
				Format:   TraceExportTaskFormatJSONArray,
				FilePath: filepath.Join(t.TempDir(), "missing.json"),
			},
		},
	}
	svc := NewTraceExportTaskService(repo)

	_, err := svc.OpenDownload(context.Background(), 9)
	require.Error(t, err)
}

func TestTraceExportTaskServiceGetTaskNotFound(t *testing.T) {
	repo := &traceExportTaskRepoStub{getErr: sql.ErrNoRows}
	svc := NewTraceExportTaskService(repo)

	_, err := svc.GetTask(context.Background(), 1)
	require.Error(t, err)
}

func TestTraceExportTaskServiceRejectsInvalidFilters(t *testing.T) {
	repo := &traceExportTaskRepoStub{}
	svc := NewTraceExportTaskService(repo)

	start := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	end := start.Add(-time.Hour)
	_, err := svc.CreateTask(context.Background(), TraceExportTaskFilters{StartTime: &start, EndTime: &end}, false, 500, 1)
	require.Error(t, err)
}

func TestTraceExportTaskServiceCreateTaskDefaultsTargetRecords(t *testing.T) {
	repo := &traceExportTaskRepoStub{}
	svc := NewTraceExportTaskService(repo)

	task, err := svc.CreateTask(context.Background(), TraceExportTaskFilters{}, false, 0, 7)
	require.NoError(t, err)
	require.Equal(t, TraceExportTaskDefaultTargetRecords, task.TargetRecords)
	require.Equal(t, TraceExportTaskDefaultTargetRecords, repo.createdTask.TargetRecords)
}

func TestTraceExportTaskServicePropagatesRepositoryErrors(t *testing.T) {
	repo := &traceExportTaskRepoStub{getErr: errors.New("boom")}
	svc := NewTraceExportTaskService(repo)

	_, err := svc.GetTask(context.Background(), 3)
	require.Error(t, err)
}
