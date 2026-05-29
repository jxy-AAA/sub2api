package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type TraceExportTaskDownload struct {
	Task        *TraceExportTask
	Reader      io.ReadCloser
	ContentType string
	Filename    string
	Size        int64
}

type TraceExportTaskService struct {
	repo TraceExportTaskRepository
}

func NewTraceExportTaskService(repo TraceExportTaskRepository) *TraceExportTaskService {
	return &TraceExportTaskService{repo: repo}
}

func (s *TraceExportTaskService) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]TraceExportTask, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("trace export task repository is not configured")
	}
	return s.repo.List(ctx, params)
}

func (s *TraceExportTaskService) CreateTask(ctx context.Context, filters TraceExportTaskFilters, includeRaw bool, targetRecords int64, requestedBy int64) (*TraceExportTask, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("trace export task repository is not configured")
	}
	if requestedBy <= 0 {
		return nil, infraerrors.BadRequest("TRACE_EXPORT_TASK_INVALID_REQUESTOR", "invalid trace export task requestor")
	}
	if targetRecords <= 0 {
		targetRecords = TraceExportTaskDefaultTargetRecords
	}

	filters.Normalize()
	if err := filters.Validate(); err != nil {
		return nil, infraerrors.BadRequest("TRACE_EXPORT_TASK_INVALID_FILTERS", err.Error())
	}

	task := &TraceExportTask{
		Status:           TraceExportTaskStatusPending,
		Format:           TraceExportTaskFormatJSONArray,
		Filters:          filters,
		IncludeRaw:       includeRaw,
		RequestedBy:      requestedBy,
		DownloadFilename: defaultTraceExportTaskFilename(time.Now().UTC()),
		TargetRecords:    targetRecords,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TraceExportTaskService) GetTask(ctx context.Context, id int64) (*TraceExportTask, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("trace export task repository is not configured")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("TRACE_EXPORT_TASK_INVALID_ID", "invalid trace export task id")
	}

	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("TRACE_EXPORT_TASK_NOT_FOUND", "trace export task not found")
		}
		return nil, err
	}
	return task, nil
}

func (s *TraceExportTaskService) CancelTask(ctx context.Context, id int64, canceledBy int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("trace export task repository is not configured")
	}
	if canceledBy <= 0 {
		return infraerrors.BadRequest("TRACE_EXPORT_TASK_INVALID_CANCELLER", "invalid trace export task canceller")
	}

	task, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.Status == TraceExportTaskStatusCanceled {
		return nil
	}
	if task.Status != TraceExportTaskStatusPending && task.Status != TraceExportTaskStatusRunning {
		return infraerrors.Conflict("TRACE_EXPORT_TASK_CANCEL_CONFLICT", "trace export task cannot be canceled in current status")
	}

	ok, err := s.repo.Cancel(ctx, id, canceledBy)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	current, getErr := s.GetTask(ctx, id)
	if getErr == nil && current.Status == TraceExportTaskStatusCanceled {
		return nil
	}
	return infraerrors.Conflict("TRACE_EXPORT_TASK_CANCEL_CONFLICT", "trace export task cannot be canceled in current status")
}

func (s *TraceExportTaskService) OpenDownload(ctx context.Context, id int64) (*TraceExportTaskDownload, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("trace export task repository is not configured")
	}

	task, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.Status != TraceExportTaskStatusSucceeded {
		return nil, infraerrors.Conflict("TRACE_EXPORT_TASK_NOT_READY", "trace export task is not ready for download")
	}

	path := strings.TrimSpace(task.FilePath)
	if path == "" {
		return nil, infraerrors.NotFound("TRACE_EXPORT_TASK_FILE_NOT_FOUND", "trace export file not found")
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, infraerrors.NotFound("TRACE_EXPORT_TASK_FILE_NOT_FOUND", "trace export file not found")
		}
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	size := task.FileSizeBytes
	if size <= 0 {
		size = info.Size()
	}

	filename := strings.TrimSpace(task.DownloadFilename)
	if filename == "" {
		filename = defaultTraceExportTaskFilename(task.CreatedAt)
	}

	return &TraceExportTaskDownload{
		Task:        task,
		Reader:      file,
		ContentType: "application/json; charset=utf-8",
		Filename:    filename,
		Size:        size,
	}, nil
}

func defaultTraceExportTaskFilename(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return fmt.Sprintf("sub2api-trace-export-%s.json", now.UTC().Format("20060102150405"))
}
