package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	TraceExportTaskStatusPending   = "pending"
	TraceExportTaskStatusRunning   = "running"
	TraceExportTaskStatusSucceeded = "succeeded"
	TraceExportTaskStatusFailed    = "failed"
	TraceExportTaskStatusCanceled  = "canceled"

	TraceExportTaskFormatJSONArray = "json_array"

	TraceExportTaskDefaultTargetRecords = int64(500)
)

type TraceExportTaskFilters struct {
	Model           string     `json:"model,omitempty"`
	UserID          *int64     `json:"user_id,omitempty"`
	APIKeyID        *int64     `json:"api_key_id,omitempty"`
	CaptureRuleID   *int64     `json:"capture_rule_id,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Keyword         string     `json:"keyword,omitempty"`
	MinInputTokens  *int64     `json:"min_input_tokens,omitempty"`
	MaxInputTokens  *int64     `json:"max_input_tokens,omitempty"`
	MinOutputTokens *int64     `json:"min_output_tokens,omitempty"`
	MaxOutputTokens *int64     `json:"max_output_tokens,omitempty"`
	MinTotalTokens  *int64     `json:"min_total_tokens,omitempty"`
	MaxTotalTokens  *int64     `json:"max_total_tokens,omitempty"`
}

func (f *TraceExportTaskFilters) Normalize() {
	if f == nil {
		return
	}

	f.Model = strings.TrimSpace(f.Model)
	f.Keyword = strings.TrimSpace(f.Keyword)
}

func (f TraceExportTaskFilters) Validate() error {
	if f.StartTime != nil && f.EndTime != nil && f.StartTime.After(*f.EndTime) {
		return fmt.Errorf("start_time must be before end_time")
	}
	if err := validateTraceExportTaskOptionalPositiveInt64("user_id", f.UserID); err != nil {
		return err
	}
	if err := validateTraceExportTaskOptionalPositiveInt64("api_key_id", f.APIKeyID); err != nil {
		return err
	}
	if err := validateTraceExportTaskOptionalPositiveInt64("capture_rule_id", f.CaptureRuleID); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("min_input_tokens", f.MinInputTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("max_input_tokens", f.MaxInputTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("min_output_tokens", f.MinOutputTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("max_output_tokens", f.MaxOutputTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("min_total_tokens", f.MinTotalTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("max_total_tokens", f.MaxTotalTokens); err != nil {
		return err
	}
	if err := validateTraceExportTaskInt64Range("input_tokens", f.MinInputTokens, f.MaxInputTokens); err != nil {
		return err
	}
	if err := validateTraceExportTaskInt64Range("output_tokens", f.MinOutputTokens, f.MaxOutputTokens); err != nil {
		return err
	}
	if err := validateTraceExportTaskInt64Range("total_tokens", f.MinTotalTokens, f.MaxTotalTokens); err != nil {
		return err
	}
	return nil
}

func (f TraceExportTaskFilters) ToModelTraceCaptureListFilter() ModelTraceCaptureListFilter {
	return ModelTraceCaptureListFilter{
		Model:           strings.TrimSpace(f.Model),
		UserID:          cloneTraceExportTaskInt64Ptr(f.UserID),
		APIKeyID:        cloneTraceExportTaskInt64Ptr(f.APIKeyID),
		CaptureRuleID:   cloneTraceExportTaskInt64Ptr(f.CaptureRuleID),
		StartTime:       cloneTraceExportTaskTimePtr(f.StartTime),
		EndTime:         cloneTraceExportTaskTimePtr(f.EndTime),
		Keyword:         strings.TrimSpace(f.Keyword),
		MinInputTokens:  cloneTraceExportTaskInt64Ptr(f.MinInputTokens),
		MaxInputTokens:  cloneTraceExportTaskInt64Ptr(f.MaxInputTokens),
		MinOutputTokens: cloneTraceExportTaskInt64Ptr(f.MinOutputTokens),
		MaxOutputTokens: cloneTraceExportTaskInt64Ptr(f.MaxOutputTokens),
		MinTotalTokens:  cloneTraceExportTaskInt64Ptr(f.MinTotalTokens),
		MaxTotalTokens:  cloneTraceExportTaskInt64Ptr(f.MaxTotalTokens),
	}
}

type TraceExportTask struct {
	ID int64 `json:"id"`

	Status string `json:"status"`
	Format string `json:"format"`

	Filters    TraceExportTaskFilters `json:"filters"`
	IncludeRaw bool                   `json:"include_raw"`

	RequestedBy      int64  `json:"requested_by"`
	DownloadFilename string `json:"download_filename,omitempty"`
	FileSizeBytes    int64  `json:"file_size_bytes"`
	TargetRecords    int64  `json:"target_records"`
	TotalRecords     int64  `json:"total_records"`
	ProcessedRecords int64  `json:"processed_records"`

	ErrorMsg *string `json:"error_message,omitempty"`

	CanceledBy   *int64     `json:"canceled_by,omitempty"`
	CanceledAt   *time.Time `json:"canceled_at,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	DownloadedAt *time.Time `json:"downloaded_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	FilePath string `json:"-"`
}

func (t TraceExportTask) IsTerminal() bool {
	switch t.Status {
	case TraceExportTaskStatusSucceeded, TraceExportTaskStatusFailed, TraceExportTaskStatusCanceled:
		return true
	default:
		return false
	}
}

type TraceExportTaskRepository interface {
	Create(ctx context.Context, task *TraceExportTask) error
	List(ctx context.Context, params pagination.PaginationParams) ([]TraceExportTask, *pagination.PaginationResult, error)
	GetByID(ctx context.Context, id int64) (*TraceExportTask, error)
	Cancel(ctx context.Context, id int64, canceledBy int64) (bool, error)
	ClaimNextPending(ctx context.Context, startedAt time.Time) (*TraceExportTask, error)
	UpdateProgress(ctx context.Context, id int64, totalRecords, processedRecords int64, updatedAt time.Time) (bool, error)
	MarkSucceeded(ctx context.Context, id int64, filePath string, fileSizeBytes, totalRecords, processedRecords int64, finishedAt time.Time) (bool, error)
	MarkFailed(ctx context.Context, id int64, totalRecords, processedRecords int64, errorMessage string, finishedAt time.Time) (bool, error)
	MarkDownloaded(ctx context.Context, id int64, downloadedAt time.Time) (bool, error)
	FailStaleRunning(ctx context.Context, staleBefore time.Time, errorMessage string, failedAt time.Time) (int64, error)
	ListReadyForFileCleanup(ctx context.Context, finishedBefore time.Time, limit int) ([]TraceExportTask, error)
	ClearFileForTask(ctx context.Context, id int64, updatedAt time.Time) (bool, error)
}

func validateTraceExportTaskOptionalPositiveInt64(name string, value *int64) error {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return fmt.Errorf("%s must be > 0", name)
	}
	return nil
}

func validateTraceExportTaskInt64Range(name string, min, max *int64) error {
	if min == nil || max == nil {
		return nil
	}
	if *min > *max {
		return fmt.Errorf("invalid %s range", name)
	}
	return nil
}

func cloneTraceExportTaskInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTraceExportTaskTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
