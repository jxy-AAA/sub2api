package service

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	defaultTraceExportTaskPollInterval = 15 * time.Second
	defaultTraceExportTaskTimeout      = 0
	defaultTraceExportTaskExportDir    = "./data/trace-exports"
	defaultTraceExportTaskBatchSize    = 200
	defaultTraceExportTaskCleanupBatch = 50
	defaultTraceExportTaskMaxRecords   = int64(100000)
)

var errTraceExportTaskCanceled = errors.New("trace export task canceled")

type TraceExportTaskExecutorOptions struct {
	Enabled          bool
	ExportDir        string
	PollInterval     time.Duration
	BatchSize        int
	TaskTimeout      time.Duration
	CleanupBatchSize int
	MaxRecords       int64
}

func DefaultTraceExportTaskExecutorOptions() TraceExportTaskExecutorOptions {
	return TraceExportTaskExecutorOptions{
		Enabled:          true,
		ExportDir:        defaultTraceExportTaskExportDir,
		PollInterval:     defaultTraceExportTaskPollInterval,
		BatchSize:        defaultTraceExportTaskBatchSize,
		TaskTimeout:      defaultTraceExportTaskTimeout,
		CleanupBatchSize: defaultTraceExportTaskCleanupBatch,
		MaxRecords:       defaultTraceExportTaskMaxRecords,
	}
}

func (o *TraceExportTaskExecutorOptions) normalize() {
	defaults := DefaultTraceExportTaskExecutorOptions()
	if strings.TrimSpace(o.ExportDir) == "" {
		o.ExportDir = defaults.ExportDir
	}
	o.ExportDir = filepath.Clean(strings.TrimSpace(o.ExportDir))
	if o.PollInterval <= 0 {
		o.PollInterval = defaults.PollInterval
	}
	if o.BatchSize <= 0 {
		o.BatchSize = defaults.BatchSize
	}
	if o.BatchSize > 1000 {
		o.BatchSize = 1000
	}
	if o.TaskTimeout < 0 {
		o.TaskTimeout = defaults.TaskTimeout
	}
	if o.CleanupBatchSize <= 0 {
		o.CleanupBatchSize = defaults.CleanupBatchSize
	}
	if o.MaxRecords <= 0 {
		o.MaxRecords = defaults.MaxRecords
	}
}

type traceExportTaskCaptureReader interface {
	List(ctx context.Context, filter ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error)
}

type TraceExportTaskExecutor struct {
	repo         TraceExportTaskRepository
	traceService traceExportTaskCaptureReader
	opts         TraceExportTaskExecutorOptions

	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
	stopCh    chan struct{}
	running   atomic.Int32

	lastCleanupWeekStart time.Time
}

func NewTraceExportTaskExecutor(repo TraceExportTaskRepository, traceService traceExportTaskCaptureReader, opts TraceExportTaskExecutorOptions) *TraceExportTaskExecutor {
	opts.normalize()
	return &TraceExportTaskExecutor{
		repo:         repo,
		traceService: traceService,
		opts:         opts,
		stopCh:       make(chan struct{}),
	}
}

func (s *TraceExportTaskExecutor) Start() {
	if s == nil || s.repo == nil || s.traceService == nil || !s.opts.Enabled {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.loop()
		logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] started (poll=%s dir=%s)", s.opts.PollInterval, s.opts.ExportDir)
	})
}

func (s *TraceExportTaskExecutor) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
	})
}

func (s *TraceExportTaskExecutor) loop() {
	defer s.wg.Done()

	s.runOnce(context.Background())

	ticker := time.NewTicker(s.opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runOnce(context.Background())
		}
	}
}

func (s *TraceExportTaskExecutor) runOnce(ctx context.Context) {
	if s == nil || s.repo == nil || s.traceService == nil || !s.opts.Enabled {
		return
	}
	if !s.running.CompareAndSwap(0, 1) {
		return
	}
	defer s.running.Store(0)

	now := time.Now().UTC()
	if staleBefore := now.Add(-s.opts.TaskTimeout); s.opts.TaskTimeout > 0 {
		failed, err := s.repo.FailStaleRunning(ctx, staleBefore, "trace export task timed out before completion", now)
		if err != nil {
			logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] fail stale running tasks error: %v", err)
		} else if failed > 0 {
			logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] marked %d stale running task(s) as failed", failed)
		}
	}
	if err := s.cleanupExpiredFiles(ctx, now); err != nil {
		logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] cleanup expired files error: %v", err)
	}

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		task, err := s.repo.ClaimNextPending(ctx, time.Now().UTC())
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] claim pending task error: %v", err)
			return
		}
		if task == nil {
			return
		}
		if execErr := s.executeTask(task); execErr != nil && !errors.Is(execErr, errTraceExportTaskCanceled) {
			logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] task=%d execution error: %v", task.ID, execErr)
		}
	}
}

func (s *TraceExportTaskExecutor) executeTask(task *TraceExportTask) error {
	if task == nil {
		return nil
	}

	targetRecords := task.TargetRecords
	if targetRecords <= 0 {
		targetRecords = TraceExportTaskDefaultTargetRecords
	}
	if s.opts.MaxRecords > 0 && targetRecords > s.opts.MaxRecords {
		return s.failTask(task.ID, targetRecords, 0, fmt.Errorf("target_records %d exceeds max_records_per_task=%d", targetRecords, s.opts.MaxRecords))
	}

	taskCtx, cancel := s.newTaskContext()
	defer cancel()

	exportDir, err := s.ensureExportDir()
	if err != nil {
		return s.failTask(task.ID, 0, 0, err)
	}

	finalPath := filepath.Join(exportDir, s.storageFilename(task))
	tempPath := finalPath + ".tmp"
	_ = os.Remove(tempPath)

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return s.failTask(task.ID, 0, 0, fmt.Errorf("create export file: %w", err))
	}

	var (
		total     = targetRecords
		processed int64
		renamed   bool
	)

	defer func() {
		_ = file.Close()
		if !renamed {
			_ = os.Remove(tempPath)
		}
	}()

	buffered := bufio.NewWriterSize(file, 64*1024)
	counting := &traceExportTaskCountingWriter{writer: buffered}
	if err := s.writeString(counting, "["); err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("write export header: %w", err))
	}

	filter := task.Filters.ToModelTraceCaptureListFilter()
	if filter.StartTime == nil {
		start := task.CreatedAt
		if start.IsZero() && task.StartedAt != nil {
			start = *task.StartedAt
		}
		if !start.IsZero() {
			start = start.UTC()
			filter.StartTime = &start
		}
	}

	if err := s.updateProgress(task.ID, total, processed); err != nil {
		return err
	}

	seen := make(map[int64]struct{}, int(targetRecords))
	wroteRecord := false
	for processed < targetRecords {
		newRecords, err := s.collectTraceExportBatch(taskCtx, task, filter, targetRecords, seen, counting, buffered, &wroteRecord, &processed)
		if err != nil {
			return err
		}
		if processed >= targetRecords {
			break
		}
		if newRecords == 0 {
			if err := s.waitForMoreCaptures(taskCtx, task.ID); err != nil {
				return s.failTask(task.ID, total, processed, err)
			}
		}
	}

	if err := s.writeString(counting, "]"); err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("write export footer: %w", err))
	}
	if err := buffered.Flush(); err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("flush export footer: %w", err))
	}
	if err := file.Sync(); err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("sync export file: %w", err))
	}
	if err := file.Close(); err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("close export file: %w", err))
	}

	info, err := os.Stat(tempPath)
	if err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("stat export file: %w", err))
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return s.failTask(task.ID, total, processed, fmt.Errorf("finalize export file: %w", err))
	}
	renamed = true

	stateCtx, stateCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stateCancel()

	ok, err := s.repo.MarkSucceeded(stateCtx, task.ID, finalPath, info.Size(), total, processed, time.Now().UTC())
	if err != nil {
		_ = os.Remove(finalPath)
		return fmt.Errorf("mark trace export task %d succeeded: %w", task.ID, err)
	}
	if !ok {
		_ = os.Remove(finalPath)
		if canceled, cancelErr := s.isTaskCanceled(stateCtx, task.ID); cancelErr == nil && canceled {
			return errTraceExportTaskCanceled
		}
		return fmt.Errorf("trace export task %d left running state before success commit", task.ID)
	}
	return nil
}

func (s *TraceExportTaskExecutor) collectTraceExportBatch(
	ctx context.Context,
	task *TraceExportTask,
	filter ModelTraceCaptureListFilter,
	targetRecords int64,
	seen map[int64]struct{},
	counting *traceExportTaskCountingWriter,
	buffered *bufio.Writer,
	wroteRecord *bool,
	processed *int64,
) (int64, error) {
	page := 1
	var newRecords int64
	for *processed < targetRecords {
		if err := s.ensureTaskRunnable(task.ID); err != nil {
			return 0, err
		}
		if err := ctx.Err(); err != nil {
			return 0, s.failTask(task.ID, targetRecords, *processed, err)
		}

		items, result, err := s.traceService.List(ctx, filter, pagination.PaginationParams{
			Page:     page,
			PageSize: s.opts.BatchSize,
		})
		if err != nil {
			return 0, s.failTask(task.ID, targetRecords, *processed, fmt.Errorf("list trace captures: %w", err))
		}
		if len(items) == 0 {
			break
		}

		for _, capture := range items {
			if capture == nil {
				continue
			}
			if _, ok := seen[capture.ID]; ok {
				continue
			}
			seen[capture.ID] = struct{}{}

			raw, marshalErr := json.Marshal(capture.Export(ModelTraceCaptureExportOptions{IncludeRaw: task.IncludeRaw}))
			if marshalErr != nil {
				return 0, s.failTask(task.ID, targetRecords, *processed, fmt.Errorf("marshal trace capture %d: %w", capture.ID, marshalErr))
			}
			if *wroteRecord {
				if err := s.writeString(counting, ","); err != nil {
					return 0, s.failTask(task.ID, targetRecords, *processed, fmt.Errorf("write export separator: %w", err))
				}
			}
			if _, err := counting.Write(raw); err != nil {
				return 0, s.failTask(task.ID, targetRecords, *processed, fmt.Errorf("write trace capture %d: %w", capture.ID, err))
			}
			*wroteRecord = true
			(*processed)++
			newRecords++
			if *processed >= targetRecords {
				break
			}
		}

		if err := buffered.Flush(); err != nil {
			return 0, s.failTask(task.ID, targetRecords, *processed, fmt.Errorf("flush export batch: %w", err))
		}
		if err := s.updateProgress(task.ID, targetRecords, *processed); err != nil {
			return 0, err
		}
		if *processed >= targetRecords || len(items) < s.opts.BatchSize {
			break
		}
		if result != nil && result.Pages > 0 && page >= result.Pages {
			break
		}
		page++
	}
	return newRecords, nil
}

func (s *TraceExportTaskExecutor) waitForMoreCaptures(ctx context.Context, taskID int64) error {
	if err := s.ensureTaskRunnable(taskID); err != nil {
		return err
	}
	timer := time.NewTimer(s.opts.PollInterval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.stopCh:
		return context.Canceled
	case <-timer.C:
		return s.ensureTaskRunnable(taskID)
	}
}

func (s *TraceExportTaskExecutor) updateProgress(taskID, total, processed int64) error {
	stateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ok, err := s.repo.UpdateProgress(stateCtx, taskID, total, processed, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update trace export task %d progress: %w", taskID, err)
	}
	if ok {
		return nil
	}
	if canceled, cancelErr := s.isTaskCanceled(stateCtx, taskID); cancelErr == nil && canceled {
		return errTraceExportTaskCanceled
	}
	return fmt.Errorf("trace export task %d is no longer running", taskID)
}

func (s *TraceExportTaskExecutor) ensureTaskRunnable(taskID int64) error {
	stateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task, err := s.repo.GetByID(stateCtx, taskID)
	if err != nil {
		return fmt.Errorf("load trace export task %d: %w", taskID, err)
	}
	switch task.Status {
	case TraceExportTaskStatusRunning:
		return nil
	case TraceExportTaskStatusCanceled:
		return errTraceExportTaskCanceled
	default:
		return fmt.Errorf("trace export task %d entered unexpected status %q", taskID, task.Status)
	}
}

func (s *TraceExportTaskExecutor) failTask(taskID, total, processed int64, cause error) error {
	if errors.Is(cause, errTraceExportTaskCanceled) {
		return cause
	}

	stateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ok, err := s.repo.MarkFailed(stateCtx, taskID, total, processed, trimTraceExportTaskError(cause), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("mark trace export task %d failed: %w (root cause: %v)", taskID, err, cause)
	}
	if !ok {
		if canceled, cancelErr := s.isTaskCanceled(stateCtx, taskID); cancelErr == nil && canceled {
			return errTraceExportTaskCanceled
		}
		return cause
	}
	return cause
}

func (s *TraceExportTaskExecutor) cleanupExpiredFiles(ctx context.Context, now time.Time) error {
	weekStart := traceExportTaskWeekStart(now)
	if weekStart.IsZero() || (!s.lastCleanupWeekStart.IsZero() && !weekStart.After(s.lastCleanupWeekStart)) {
		return nil
	}

	hadError := false
	for {
		tasks, err := s.repo.ListReadyForFileCleanup(ctx, weekStart, s.opts.CleanupBatchSize)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			break
		}

		cleared := 0
		for _, task := range tasks {
			path := strings.TrimSpace(task.FilePath)
			if path == "" {
				continue
			}
			if s.isManagedExportPath(path) {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					hadError = true
					logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] remove expired export file task=%d path=%s error=%v", task.ID, path, err)
					continue
				}
			} else {
				logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] skip deleting unmanaged expired export path task=%d path=%s", task.ID, path)
			}
			stateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, clearErr := s.repo.ClearFileForTask(stateCtx, task.ID, time.Now().UTC())
			cancel()
			if clearErr != nil {
				hadError = true
				logger.LegacyPrintf("service.trace_export_task", "[TraceExportTaskExecutor] clear expired export metadata task=%d error=%v", task.ID, clearErr)
				continue
			}
			cleared++
		}
		if len(tasks) < s.opts.CleanupBatchSize || cleared == 0 {
			break
		}
	}
	if !hadError {
		s.lastCleanupWeekStart = weekStart
	}
	return nil
}

func traceExportTaskWeekStart(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	utc := now.UTC()
	year, month, day := utc.Date()
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	daysSinceMonday := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -daysSinceMonday)
}

func (s *TraceExportTaskExecutor) ensureExportDir() (string, error) {
	dir := filepath.Clean(strings.TrimSpace(s.opts.ExportDir))
	if dir == "" || dir == "." {
		return "", fmt.Errorf("trace export dir is empty")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("mkdir trace export dir %q: %w", dir, err)
	}
	return dir, nil
}

func (s *TraceExportTaskExecutor) storageFilename(task *TraceExportTask) string {
	base := sanitizeTraceExportTaskFilename(task.DownloadFilename)
	if base == "" {
		when := time.Now().UTC()
		if task.StartedAt != nil && !task.StartedAt.IsZero() {
			when = task.StartedAt.UTC()
		}
		base = defaultTraceExportTaskFilename(when)
	}
	return fmt.Sprintf("trace-export-task-%d-%s", task.ID, base)
}

func (s *TraceExportTaskExecutor) isManagedExportPath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	base, err := filepath.Abs(s.opts.ExportDir)
	if err != nil {
		return false
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
}

func (s *TraceExportTaskExecutor) isTaskCanceled(ctx context.Context, taskID int64) (bool, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return false, err
	}
	return task.Status == TraceExportTaskStatusCanceled, nil
}

func (s *TraceExportTaskExecutor) newTaskContext() (context.Context, context.CancelFunc) {
	parent := context.Background()
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if s.opts.TaskTimeout > 0 {
		ctx, cancel = context.WithTimeout(parent, s.opts.TaskTimeout)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}
	go func() {
		select {
		case <-s.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

func (s *TraceExportTaskExecutor) writeString(writer io.Writer, value string) error {
	_, err := io.WriteString(writer, value)
	return err
}

func sanitizeTraceExportTaskFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "-", " ", "_")
	name = replacer.Replace(name)
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}
	return name
}

func trimTraceExportTaskError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) <= 1024 {
		return msg
	}
	return msg[:1024]
}

type traceExportTaskCountingWriter struct {
	writer io.Writer
	count  int64
}

func (w *traceExportTaskCountingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.count += int64(n)
	return n, err
}

func (w *traceExportTaskCountingWriter) size() int64 {
	if w == nil {
		return 0
	}
	return w.count
}
