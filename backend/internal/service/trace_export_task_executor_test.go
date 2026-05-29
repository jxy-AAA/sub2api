package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type traceExportTaskExecutorRepoStub struct {
	tasks             map[int64]*TraceExportTask
	pending           []int64
	cancelAtProcessed int64
	progressUpdates   []traceExportTaskProgressUpdate
	cleanupCalls      int
	cleanupBefore     time.Time
}

type traceExportTaskProgressUpdate struct {
	TotalRecords     int64
	ProcessedRecords int64
}

func (s *traceExportTaskExecutorRepoStub) Create(ctx context.Context, task *TraceExportTask) error {
	if s.tasks == nil {
		s.tasks = map[int64]*TraceExportTask{}
	}
	cloned := cloneTraceExportTask(task)
	s.tasks[task.ID] = cloned
	return nil
}

func (s *traceExportTaskExecutorRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]TraceExportTask, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *traceExportTaskExecutorRepoStub) GetByID(ctx context.Context, id int64) (*TraceExportTask, error) {
	if task, ok := s.tasks[id]; ok {
		return cloneTraceExportTask(task), nil
	}
	return nil, sql.ErrNoRows
}

func (s *traceExportTaskExecutorRepoStub) Cancel(ctx context.Context, id int64, canceledBy int64) (bool, error) {
	task, ok := s.tasks[id]
	if !ok {
		return false, nil
	}
	now := time.Now().UTC()
	task.Status = TraceExportTaskStatusCanceled
	task.CanceledBy = &canceledBy
	task.CanceledAt = &now
	task.FinishedAt = &now
	task.UpdatedAt = now
	task.ErrorMsg = nil
	return true, nil
}

func (s *traceExportTaskExecutorRepoStub) ClaimNextPending(ctx context.Context, startedAt time.Time) (*TraceExportTask, error) {
	for len(s.pending) > 0 {
		id := s.pending[0]
		s.pending = s.pending[1:]
		task, ok := s.tasks[id]
		if !ok || task.Status != TraceExportTaskStatusPending {
			continue
		}
		task.Status = TraceExportTaskStatusRunning
		ts := startedAt.UTC()
		task.StartedAt = &ts
		task.UpdatedAt = ts
		task.TotalRecords = 0
		task.ProcessedRecords = 0
		task.FilePath = ""
		task.FileSizeBytes = 0
		task.ErrorMsg = nil
		return cloneTraceExportTask(task), nil
	}
	return nil, sql.ErrNoRows
}

func (s *traceExportTaskExecutorRepoStub) UpdateProgress(ctx context.Context, id int64, totalRecords, processedRecords int64, updatedAt time.Time) (bool, error) {
	task, ok := s.tasks[id]
	if !ok || task.Status != TraceExportTaskStatusRunning {
		return false, nil
	}
	if s.cancelAtProcessed > 0 && processedRecords >= s.cancelAtProcessed {
		task.Status = TraceExportTaskStatusCanceled
		task.UpdatedAt = updatedAt.UTC()
		return false, nil
	}
	s.progressUpdates = append(s.progressUpdates, traceExportTaskProgressUpdate{
		TotalRecords:     totalRecords,
		ProcessedRecords: processedRecords,
	})
	task.TotalRecords = totalRecords
	task.ProcessedRecords = processedRecords
	task.UpdatedAt = updatedAt.UTC()
	return true, nil
}

func (s *traceExportTaskExecutorRepoStub) MarkSucceeded(ctx context.Context, id int64, filePath string, fileSizeBytes, totalRecords, processedRecords int64, finishedAt time.Time) (bool, error) {
	task, ok := s.tasks[id]
	if !ok || task.Status != TraceExportTaskStatusRunning {
		return false, nil
	}
	finished := finishedAt.UTC()
	task.Status = TraceExportTaskStatusSucceeded
	task.FilePath = filePath
	task.FileSizeBytes = fileSizeBytes
	task.TotalRecords = totalRecords
	task.ProcessedRecords = processedRecords
	task.FinishedAt = &finished
	task.UpdatedAt = finished
	task.ErrorMsg = nil
	return true, nil
}

func (s *traceExportTaskExecutorRepoStub) MarkFailed(ctx context.Context, id int64, totalRecords, processedRecords int64, errorMessage string, finishedAt time.Time) (bool, error) {
	task, ok := s.tasks[id]
	if !ok || task.Status != TraceExportTaskStatusRunning {
		return false, nil
	}
	finished := finishedAt.UTC()
	task.Status = TraceExportTaskStatusFailed
	task.TotalRecords = totalRecords
	task.ProcessedRecords = processedRecords
	task.FilePath = ""
	task.FileSizeBytes = 0
	task.FinishedAt = &finished
	task.UpdatedAt = finished
	task.ErrorMsg = &errorMessage
	return true, nil
}

func (s *traceExportTaskExecutorRepoStub) MarkDownloaded(ctx context.Context, id int64, downloadedAt time.Time) (bool, error) {
	task, ok := s.tasks[id]
	if !ok || task.Status != TraceExportTaskStatusSucceeded || task.FilePath == "" || task.DownloadedAt != nil {
		return false, nil
	}
	at := downloadedAt.UTC()
	task.DownloadedAt = &at
	task.UpdatedAt = at
	return true, nil
}

func (s *traceExportTaskExecutorRepoStub) FailStaleRunning(ctx context.Context, staleBefore time.Time, errorMessage string, failedAt time.Time) (int64, error) {
	return 0, nil
}

func (s *traceExportTaskExecutorRepoStub) ListReadyForFileCleanup(ctx context.Context, downloadedBefore time.Time, limit int) ([]TraceExportTask, error) {
	s.cleanupCalls++
	s.cleanupBefore = downloadedBefore.UTC()
	if limit <= 0 {
		limit = 50
	}
	items := make([]TraceExportTask, 0, limit)
	for _, task := range s.tasks {
		if task == nil || strings.TrimSpace(task.FilePath) == "" || task.DownloadedAt == nil {
			continue
		}
		if !task.DownloadedAt.UTC().Before(downloadedBefore.UTC()) {
			continue
		}
		if task.Status == TraceExportTaskStatusSucceeded {
			items = append(items, *cloneTraceExportTask(task))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		leftDownloaded := time.Time{}
		rightDownloaded := time.Time{}
		if left.DownloadedAt != nil {
			leftDownloaded = left.DownloadedAt.UTC()
		}
		if right.DownloadedAt != nil {
			rightDownloaded = right.DownloadedAt.UTC()
		}
		if !leftDownloaded.Equal(rightDownloaded) {
			return leftDownloaded.Before(rightDownloaded)
		}
		return left.ID < right.ID
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *traceExportTaskExecutorRepoStub) ClearFileForTask(ctx context.Context, id int64, updatedAt time.Time) (bool, error) {
	task, ok := s.tasks[id]
	if !ok || task.FilePath == "" {
		return false, nil
	}
	task.FilePath = ""
	task.FileSizeBytes = 0
	task.UpdatedAt = updatedAt.UTC()
	return true, nil
}

type traceExportTaskCaptureReaderStub struct {
	pages     map[int][]*ModelTraceCapture
	total     int64
	snapshots []traceExportTaskCaptureReaderSnapshot

	calls      int
	lastFilter ModelTraceCaptureListFilter
}

type traceExportTaskCaptureReaderSnapshot struct {
	pages map[int][]*ModelTraceCapture
	total int64
}

func (s *traceExportTaskCaptureReaderStub) List(ctx context.Context, filter ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error) {
	s.calls++
	s.lastFilter = filter
	pages := s.pages
	total := s.total
	if len(s.snapshots) > 0 {
		idx := s.calls - 1
		if idx >= len(s.snapshots) {
			idx = len(s.snapshots) - 1
		}
		pages = s.snapshots[idx].pages
		total = s.snapshots[idx].total
	}
	items := pages[params.Page]
	if items == nil {
		items = []*ModelTraceCapture{}
	}
	return items, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

func TestTraceExportTaskExecutorSchemaContractWithoutRaw(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			1: {
				ID:            1,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				Filters:       TraceExportTaskFilters{Model: "gpt-4.1"},
				TargetRecords: 2,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending: []int64{1},
	}
	reader := &traceExportTaskCaptureReaderStub{
		total: 2,
		pages: map[int][]*ModelTraceCapture{
			1: {newTraceExportTaskCaptureFixture("task-1", false)},
			2: {newTraceExportTaskCaptureFixture("task-2", false)},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		BatchSize:        1,
		TaskTimeout:      time.Minute,
		CleanupBatchSize: 10,
		MaxRecords:       10,
	})

	executor.runOnce(context.Background())

	task := repo.tasks[1]
	require.Equal(t, TraceExportTaskStatusSucceeded, task.Status)
	require.Equal(t, int64(2), task.TotalRecords)
	require.Equal(t, int64(2), task.ProcessedRecords)
	require.NotEmpty(t, task.FilePath)
	require.NotNil(t, task.StartedAt)
	require.Nil(t, reader.lastFilter.EndTime)
	require.NotNil(t, reader.lastFilter.StartTime)
	require.True(t, reader.lastFilter.StartTime.Equal(time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC)))

	body, err := os.ReadFile(task.FilePath)
	require.NoError(t, err)

	var exported []map[string]any
	require.NoError(t, json.Unmarshal(body, &exported))
	require.Len(t, exported, 2)
	require.Contains(t, exported[0], "task_id")
	require.Contains(t, exported[0], "prompt")
	require.Contains(t, exported[0], "candidates")
	require.Contains(t, exported[0], "tools")
	require.Contains(t, exported[0], "signature")
	require.Contains(t, exported[0], "meta")
	require.Contains(t, exported[0], "system")
	require.Contains(t, exported[0], "user")
	require.NotContains(t, exported[0], "raw_request")
	require.NotContains(t, exported[0], "raw_response")
	require.NotContains(t, exported[0], "raw_request_text")
	require.NotContains(t, exported[0], "raw_response_text")
}

func TestTraceExportTaskExecutorSchemaContractWithRaw(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			2: {
				ID:            2,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				IncludeRaw:    true,
				TargetRecords: 1,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending: []int64{2},
	}
	reader := &traceExportTaskCaptureReaderStub{
		total: 1,
		pages: map[int][]*ModelTraceCapture{
			1: {newTraceExportTaskCaptureFixture("task-raw", true)},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		BatchSize:        10,
		TaskTimeout:      time.Minute,
		CleanupBatchSize: 10,
		MaxRecords:       10,
	})

	executor.runOnce(context.Background())

	body, err := os.ReadFile(repo.tasks[2].FilePath)
	require.NoError(t, err)

	var exported []map[string]any
	require.NoError(t, json.Unmarshal(body, &exported))
	require.Len(t, exported, 1)
	require.Contains(t, exported[0], "raw_request")
	require.Contains(t, exported[0], "raw_upstream_request")
	require.Contains(t, exported[0], "raw_upstream_request_text")
	require.Contains(t, exported[0], "raw_response")
	require.Contains(t, exported[0], "raw_request_text")
	require.Contains(t, exported[0], "raw_response_text")
	require.Equal(t, map[string]any{"upstream": "request"}, exported[0]["raw_upstream_request"])
	require.Equal(t, `{"upstream":"request"}`, exported[0]["raw_upstream_request_text"])
}

func TestTraceExportTaskExecutorCancelsRunningTask(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			3: {
				ID:            3,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				TargetRecords: 1,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending:           []int64{3},
		cancelAtProcessed: 1,
	}
	reader := &traceExportTaskCaptureReaderStub{
		total: 1,
		pages: map[int][]*ModelTraceCapture{
			1: {newTraceExportTaskCaptureFixture("task-cancel", false)},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		BatchSize:        1,
		TaskTimeout:      time.Minute,
		CleanupBatchSize: 10,
		MaxRecords:       10,
	})

	executor.runOnce(context.Background())

	task := repo.tasks[3]
	require.Equal(t, TraceExportTaskStatusCanceled, task.Status)
	require.Empty(t, task.FilePath)
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 0)
}

func TestTraceExportTaskExecutorMarksFailedWhenRecordLimitExceeded(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			4: {
				ID:            4,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				TargetRecords: 2,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending: []int64{4},
	}
	reader := &traceExportTaskCaptureReaderStub{
		total: 2,
		pages: map[int][]*ModelTraceCapture{
			1: {newTraceExportTaskCaptureFixture("task-limit-1", false)},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		BatchSize:        10,
		TaskTimeout:      time.Minute,
		CleanupBatchSize: 10,
		MaxRecords:       1,
	})

	executor.runOnce(context.Background())

	task := repo.tasks[4]
	require.Equal(t, TraceExportTaskStatusFailed, task.Status)
	require.NotNil(t, task.ErrorMsg)
	require.Contains(t, *task.ErrorMsg, "max_records_per_task")
	require.Empty(t, task.FilePath)
}

func TestTraceExportTaskExecutorWaitsUntilTargetRecordsAreCaptured(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			5: {
				ID:            5,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				TargetRecords: 1,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending: []int64{5},
	}
	reader := &traceExportTaskCaptureReaderStub{
		snapshots: []traceExportTaskCaptureReaderSnapshot{
			{total: 0, pages: map[int][]*ModelTraceCapture{1: {}}},
			{total: 1, pages: map[int][]*ModelTraceCapture{1: {newTraceExportTaskCaptureFixture("task-wait", false)}}},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		PollInterval:     time.Millisecond,
		BatchSize:        10,
		TaskTimeout:      0,
		CleanupBatchSize: 10,
		MaxRecords:       10,
	})

	executor.runOnce(context.Background())

	task := repo.tasks[5]
	require.GreaterOrEqual(t, reader.calls, 2)
	require.Equal(t, TraceExportTaskStatusSucceeded, task.Status)
	require.Equal(t, int64(1), task.TotalRecords)
	require.Equal(t, int64(1), task.ProcessedRecords)
	body, err := os.ReadFile(task.FilePath)
	require.NoError(t, err)
	var exported []map[string]any
	require.NoError(t, json.Unmarshal(body, &exported))
	require.Len(t, exported, 1)
}

func TestTraceExportTaskExecutorWaitsForFullTargetAfterPartialCapture(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			6: {
				ID:            6,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				TargetRecords: 500,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending: []int64{6},
	}
	reader := &traceExportTaskCaptureReaderStub{
		snapshots: []traceExportTaskCaptureReaderSnapshot{
			{total: 58, pages: map[int][]*ModelTraceCapture{1: newTraceExportTaskCaptureFixtures("task-partial", 58, false)}},
			{total: 500, pages: map[int][]*ModelTraceCapture{1: newTraceExportTaskCaptureFixtures("task-partial", 500, false)}},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		PollInterval:     time.Millisecond,
		BatchSize:        1000,
		TaskTimeout:      0,
		CleanupBatchSize: 10,
		MaxRecords:       1000,
	})

	executor.runOnce(context.Background())

	task := repo.tasks[6]
	require.GreaterOrEqual(t, reader.calls, 2)
	require.Equal(t, TraceExportTaskStatusSucceeded, task.Status)
	require.Equal(t, int64(500), task.TotalRecords)
	require.Equal(t, int64(500), task.ProcessedRecords)
	require.Contains(t, repo.progressUpdates, traceExportTaskProgressUpdate{
		TotalRecords:     500,
		ProcessedRecords: 58,
	})
	body, err := os.ReadFile(task.FilePath)
	require.NoError(t, err)
	var exported []map[string]any
	require.NoError(t, json.Unmarshal(body, &exported))
	require.Len(t, exported, 500)
}

func TestTraceExportTaskExecutorAllowsLargeExportFiles(t *testing.T) {
	tempDir := t.TempDir()
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			7: {
				ID:            7,
				Status:        TraceExportTaskStatusPending,
				Format:        TraceExportTaskFormatJSONArray,
				IncludeRaw:    true,
				TargetRecords: 1,
				RequestedBy:   9,
				CreatedAt:     time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
			},
		},
		pending: []int64{7},
	}
	capture := newTraceExportTaskCaptureFixture("task-large-raw", true)
	capture.RawRequestText = strings.Repeat("x", 2*1024*1024)
	reader := &traceExportTaskCaptureReaderStub{
		total: 1,
		pages: map[int][]*ModelTraceCapture{
			1: {capture},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, reader, TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        tempDir,
		BatchSize:        10,
		TaskTimeout:      time.Minute,
		CleanupBatchSize: 10,
		MaxRecords:       10,
	})

	executor.runOnce(context.Background())

	task := repo.tasks[7]
	require.Equal(t, TraceExportTaskStatusSucceeded, task.Status)
	require.Nil(t, task.ErrorMsg)
	require.NotEmpty(t, task.FilePath)
	require.Greater(t, task.FileSizeBytes, int64(2*1024*1024))
	require.FileExists(t, task.FilePath)
}

func TestTraceExportTaskExecutorCleansDownloadsOlderThanRetention(t *testing.T) {
	tempDir := t.TempDir()
	oldPath := filepath.Join(tempDir, "old-download.json")
	currentPath := filepath.Join(tempDir, "recent-download.json")
	require.NoError(t, os.WriteFile(oldPath, []byte(`[{"old":true}]`), 0o600))
	require.NoError(t, os.WriteFile(currentPath, []byte(`[{"current":true}]`), 0o600))

	finished := time.Date(2026, 5, 26, 9, 0, 0, 0, time.UTC)
	oldDownloaded := time.Date(2026, 5, 26, 11, 59, 0, 0, time.UTC)
	currentDownloaded := time.Date(2026, 5, 26, 12, 30, 0, 0, time.UTC)
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			8: {
				ID:           8,
				Status:       TraceExportTaskStatusSucceeded,
				FilePath:     oldPath,
				FinishedAt:   &finished,
				DownloadedAt: &oldDownloaded,
			},
			9: {
				ID:           9,
				Status:       TraceExportTaskStatusSucceeded,
				FilePath:     currentPath,
				FinishedAt:   &finished,
				DownloadedAt: &currentDownloaded,
			},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, nil, TraceExportTaskExecutorOptions{
		Enabled:           true,
		ExportDir:         tempDir,
		CleanupBatchSize:  10,
		MaxRecords:        10,
		DownloadRetention: 24 * time.Hour,
	})

	err := executor.cleanupExpiredFiles(context.Background(), time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC), repo.cleanupBefore)
	require.Empty(t, repo.tasks[8].FilePath)
	_, statErr := os.Stat(oldPath)
	require.True(t, os.IsNotExist(statErr))
	require.Equal(t, currentPath, repo.tasks[9].FilePath)
	require.FileExists(t, currentPath)
}

func TestTraceExportTaskExecutorCleanupContinuesAfterUnmanagedPath(t *testing.T) {
	exportDir := filepath.Join(t.TempDir(), "exports")
	require.NoError(t, os.MkdirAll(exportDir, 0o750))
	outsideDir := t.TempDir()
	unmanagedPath := filepath.Join(outsideDir, "external.json")
	managedPath := filepath.Join(exportDir, "previous-week.json")
	require.NoError(t, os.WriteFile(unmanagedPath, []byte(`[{"external":true}]`), 0o600))
	require.NoError(t, os.WriteFile(managedPath, []byte(`[{"managed":true}]`), 0o600))

	finished := time.Date(2026, 5, 26, 9, 0, 0, 0, time.UTC)
	unmanagedDownloaded := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	managedDownloaded := time.Date(2026, 5, 26, 10, 30, 0, 0, time.UTC)
	repo := &traceExportTaskExecutorRepoStub{
		tasks: map[int64]*TraceExportTask{
			10: {
				ID:           10,
				Status:       TraceExportTaskStatusSucceeded,
				FilePath:     unmanagedPath,
				FinishedAt:   &finished,
				DownloadedAt: &unmanagedDownloaded,
			},
			11: {
				ID:           11,
				Status:       TraceExportTaskStatusSucceeded,
				FilePath:     managedPath,
				FinishedAt:   &finished,
				DownloadedAt: &managedDownloaded,
			},
		},
	}
	executor := NewTraceExportTaskExecutor(repo, nil, TraceExportTaskExecutorOptions{
		Enabled:           true,
		ExportDir:         exportDir,
		CleanupBatchSize:  1,
		MaxRecords:        10,
		DownloadRetention: 24 * time.Hour,
	})

	err := executor.cleanupExpiredFiles(context.Background(), time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Empty(t, repo.tasks[10].FilePath)
	require.Empty(t, repo.tasks[11].FilePath)
	require.FileExists(t, unmanagedPath)
	_, statErr := os.Stat(managedPath)
	require.True(t, os.IsNotExist(statErr))
	require.GreaterOrEqual(t, repo.cleanupCalls, 3)
}

func newTraceExportTaskCaptureFixtures(prefix string, count int, includeRaw bool) []*ModelTraceCapture {
	captures := make([]*ModelTraceCapture, 0, count)
	for i := 1; i <= count; i++ {
		captures = append(captures, newTraceExportTaskCaptureFixture(fmt.Sprintf("%s-%d", prefix, i), includeRaw))
	}
	return captures
}

func newTraceExportTaskCaptureFixture(taskID string, includeRaw bool) *ModelTraceCapture {
	userID := int64(7)
	apiKeyID := int64(8)
	createdAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	capture := &ModelTraceCapture{
		ID:                  traceExportTaskFixtureID(taskID),
		TaskID:              taskID,
		UserID:              &userID,
		APIKeyID:            &apiKeyID,
		Protocol:            "responses",
		Model:               "gpt-4.1",
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		Scaffold:            "sub2api",
		ScaffoldVersion:     "trace-v1",
		Prompt:              json.RawMessage(`[{"role":"system","content":"policy"},{"role":"user","content":"hello"}]`),
		Candidates:          json.RawMessage(`[{"content":"world"}]`),
		Tools:               json.RawMessage(`[{"type":"function","function":{"name":"lookup"}}]`),
		Signature:           json.RawMessage(`{"alg":"sha256","value":"sig"}`),
		Meta:                json.RawMessage(`{"source":"trace-export-task"}`),
		RawRequest:          json.RawMessage(`{"prompt":"hello"}`),
		RawResponse:         json.RawMessage(`{"answer":"world"}`),
		RawRequestText:      "raw-request-text",
		RawResponseText:     "raw-response-text",
		DedupeHash:          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PromptHash:          "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CreatedAt:           createdAt,
	}
	if !includeRaw {
		capture.RawRequest = nil
		capture.RawResponse = nil
		capture.RawRequestText = ""
		capture.RawResponseText = ""
	} else {
		capture.Meta = mustTraceRawJSON(map[string]any{
			"source": "trace-export-task",
			traceCaptureScaffoldJSONMetaKey: map[string]any{
				"captures": []GatewayTraceCaptureEntry{
					{
						Stage: GatewayTraceStageUpstreamRequest,
						Body:  `{"upstream":"request"}`,
					},
				},
			},
		})
	}
	return capture
}

func traceExportTaskFixtureID(taskID string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(taskID))
	return int64(hasher.Sum64() & 0x7fffffffffffffff)
}

func cloneTraceExportTask(task *TraceExportTask) *TraceExportTask {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.Filters = task.Filters
	cloned.Filters.UserID = cloneTraceExportTaskInt64Ptr(task.Filters.UserID)
	cloned.Filters.APIKeyID = cloneTraceExportTaskInt64Ptr(task.Filters.APIKeyID)
	cloned.Filters.CaptureRuleID = cloneTraceExportTaskInt64Ptr(task.Filters.CaptureRuleID)
	cloned.Filters.StartTime = cloneTraceExportTaskTimePtr(task.Filters.StartTime)
	cloned.Filters.EndTime = cloneTraceExportTaskTimePtr(task.Filters.EndTime)
	cloned.Filters.MinInputTokens = cloneTraceExportTaskInt64Ptr(task.Filters.MinInputTokens)
	cloned.Filters.MaxInputTokens = cloneTraceExportTaskInt64Ptr(task.Filters.MaxInputTokens)
	cloned.Filters.MinOutputTokens = cloneTraceExportTaskInt64Ptr(task.Filters.MinOutputTokens)
	cloned.Filters.MaxOutputTokens = cloneTraceExportTaskInt64Ptr(task.Filters.MaxOutputTokens)
	cloned.Filters.MinTotalTokens = cloneTraceExportTaskInt64Ptr(task.Filters.MinTotalTokens)
	cloned.Filters.MaxTotalTokens = cloneTraceExportTaskInt64Ptr(task.Filters.MaxTotalTokens)
	cloned.CanceledBy = cloneTraceExportTaskInt64Ptr(task.CanceledBy)
	cloned.CanceledAt = cloneTraceExportTaskTimePtr(task.CanceledAt)
	cloned.StartedAt = cloneTraceExportTaskTimePtr(task.StartedAt)
	cloned.FinishedAt = cloneTraceExportTaskTimePtr(task.FinishedAt)
	cloned.DownloadedAt = cloneTraceExportTaskTimePtr(task.DownloadedAt)
	if task.ErrorMsg != nil {
		msg := *task.ErrorMsg
		cloned.ErrorMsg = &msg
	}
	return &cloned
}
