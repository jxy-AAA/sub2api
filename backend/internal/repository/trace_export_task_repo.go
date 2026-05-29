package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const traceExportTaskSelectColumns = `
	id,
	status,
	format,
	filters,
	include_raw,
	target_records,
	requested_by,
	download_filename,
	file_path,
	file_size_bytes,
	total_records,
	processed_records,
	error_message,
	canceled_by,
	canceled_at,
	started_at,
	finished_at,
	downloaded_at,
	created_at,
	updated_at
`

const traceExportTaskReturningColumns = `
	t.id,
	t.status,
	t.format,
	t.filters,
	t.include_raw,
	t.target_records,
	t.requested_by,
	t.download_filename,
	t.file_path,
	t.file_size_bytes,
	t.total_records,
	t.processed_records,
	t.error_message,
	t.canceled_by,
	t.canceled_at,
	t.started_at,
	t.finished_at,
	t.downloaded_at,
	t.created_at,
	t.updated_at
`

type traceExportTaskRepository struct {
	sql sqlExecutor
}

func NewTraceExportTaskRepository(sqlDB *sql.DB) service.TraceExportTaskRepository {
	return &traceExportTaskRepository{sql: sqlDB}
}

func (r *traceExportTaskRepository) Create(ctx context.Context, task *service.TraceExportTask) error {
	if task == nil {
		return nil
	}

	filtersJSON, err := json.Marshal(task.Filters)
	if err != nil {
		return fmt.Errorf("marshal trace export task filters: %w", err)
	}

	query := `
		INSERT INTO trace_export_tasks (
			status,
			format,
			filters,
			include_raw,
			target_records,
			requested_by,
			download_filename,
			file_path,
			file_size_bytes,
			total_records,
			processed_records
		) VALUES (
			$1,
			$2,
			$3::jsonb,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11
		)
		RETURNING id, created_at, updated_at
	`

	return scanSingleRow(
		ctx,
		r.sql,
		query,
		[]any{
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
		},
		&task.ID,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
}

func (r *traceExportTaskRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.TraceExportTask, *pagination.PaginationResult, error) {
	var total int64
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM trace_export_tasks", nil, &total); err != nil {
		return nil, nil, err
	}
	if total == 0 {
		return []service.TraceExportTask{}, paginationResultFromTotal(0, params), nil
	}

	query := `
		SELECT
	` + traceExportTaskSelectColumns + `
		FROM trace_export_tasks
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.sql.QueryContext(ctx, query, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.TraceExportTask, 0, params.Limit())
	for rows.Next() {
		item, scanErr := scanTraceExportTask(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *traceExportTaskRepository) GetByID(ctx context.Context, id int64) (*service.TraceExportTask, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id is required")
	}

	query := `
		SELECT
	` + traceExportTaskSelectColumns + `
		FROM trace_export_tasks
		WHERE id = $1
	`
	rows, err := r.sql.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	item, err := scanTraceExportTask(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *traceExportTaskRepository) Cancel(ctx context.Context, id int64, canceledBy int64) (bool, error) {
	query := `
		UPDATE trace_export_tasks
		SET status = $1,
			canceled_by = $3,
			canceled_at = $4,
			finished_at = $4,
			error_message = NULL,
			updated_at = $4
		WHERE id = $2
			AND status IN ($5, $6)
		RETURNING id
	`

	var returnedID int64
	now := time.Now().UTC()
	err := scanSingleRow(
		ctx,
		r.sql,
		query,
		[]any{
			service.TraceExportTaskStatusCanceled,
			id,
			canceledBy,
			now,
			service.TraceExportTaskStatusPending,
			service.TraceExportTaskStatusRunning,
		},
		&returnedID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *traceExportTaskRepository) ClaimNextPending(ctx context.Context, startedAt time.Time) (*service.TraceExportTask, error) {
	query := `
		WITH next_task AS (
			SELECT id
			FROM trace_export_tasks
			WHERE status = $1
			ORDER BY created_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE trace_export_tasks AS t
		SET status = $2,
			started_at = $3,
			finished_at = NULL,
			file_path = '',
			file_size_bytes = 0,
			total_records = 0,
			processed_records = 0,
			error_message = NULL,
			updated_at = $3
		FROM next_task
		WHERE t.id = next_task.id
		RETURNING
	` + traceExportTaskReturningColumns
	return r.scanOneTask(ctx, query,
		service.TraceExportTaskStatusPending,
		service.TraceExportTaskStatusRunning,
		startedAt.UTC(),
	)
}

func (r *traceExportTaskRepository) UpdateProgress(ctx context.Context, id int64, totalRecords, processedRecords int64, updatedAt time.Time) (bool, error) {
	query := `
		UPDATE trace_export_tasks
		SET total_records = $2,
			processed_records = $3,
			updated_at = $4
		WHERE id = $1
			AND status = $5
		RETURNING id
	`
	var returnedID int64
	err := scanSingleRow(ctx, r.sql, query, []any{
		id,
		totalRecords,
		processedRecords,
		updatedAt.UTC(),
		service.TraceExportTaskStatusRunning,
	}, &returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *traceExportTaskRepository) MarkSucceeded(ctx context.Context, id int64, filePath string, fileSizeBytes, totalRecords, processedRecords int64, finishedAt time.Time) (bool, error) {
	query := `
		UPDATE trace_export_tasks
		SET status = $2,
			file_path = $3,
			file_size_bytes = $4,
			total_records = $5,
			processed_records = $6,
			error_message = NULL,
			finished_at = $7,
			updated_at = $7
		WHERE id = $1
			AND status = $8
		RETURNING id
	`
	var returnedID int64
	err := scanSingleRow(ctx, r.sql, query, []any{
		id,
		service.TraceExportTaskStatusSucceeded,
		strings.TrimSpace(filePath),
		fileSizeBytes,
		totalRecords,
		processedRecords,
		finishedAt.UTC(),
		service.TraceExportTaskStatusRunning,
	}, &returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *traceExportTaskRepository) MarkFailed(ctx context.Context, id int64, totalRecords, processedRecords int64, message string, finishedAt time.Time) (bool, error) {
	query := `
		UPDATE trace_export_tasks
		SET status = $2,
			file_path = '',
			file_size_bytes = 0,
			total_records = $3,
			processed_records = $4,
			error_message = $5,
			finished_at = $6,
			updated_at = $6
		WHERE id = $1
			AND status = $7
		RETURNING id
	`
	var returnedID int64
	err := scanSingleRow(ctx, r.sql, query, []any{
		id,
		service.TraceExportTaskStatusFailed,
		totalRecords,
		processedRecords,
		strings.TrimSpace(message),
		finishedAt.UTC(),
		service.TraceExportTaskStatusRunning,
	}, &returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *traceExportTaskRepository) MarkDownloaded(ctx context.Context, id int64, downloadedAt time.Time) (bool, error) {
	query := `
		UPDATE trace_export_tasks
		SET downloaded_at = $2,
			updated_at = $2
		WHERE id = $1
			AND status = $3
			AND file_path <> ''
			AND downloaded_at IS NULL
		RETURNING id
	`
	var returnedID int64
	err := scanSingleRow(ctx, r.sql, query, []any{
		id,
		downloadedAt.UTC(),
		service.TraceExportTaskStatusSucceeded,
	}, &returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *traceExportTaskRepository) FailStaleRunning(ctx context.Context, staleBefore time.Time, message string, failedAt time.Time) (int64, error) {
	res, err := r.sql.ExecContext(ctx, `
		UPDATE trace_export_tasks
		SET status = $2,
			file_path = '',
			file_size_bytes = 0,
			error_message = $3,
			finished_at = $4,
			updated_at = $4
		WHERE status = $1
			AND updated_at < $5
	`, service.TraceExportTaskStatusRunning, service.TraceExportTaskStatusFailed, strings.TrimSpace(message), failedAt.UTC(), staleBefore.UTC())
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *traceExportTaskRepository) ListReadyForFileCleanup(ctx context.Context, downloadedBefore time.Time, limit int) ([]service.TraceExportTask, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT
	` + traceExportTaskSelectColumns + `
		FROM trace_export_tasks
		WHERE file_path <> ''
			AND downloaded_at IS NOT NULL
			AND downloaded_at < $1
			AND status = $2
		ORDER BY downloaded_at ASC, id ASC
		LIMIT $3
	`
	rows, err := r.sql.QueryContext(ctx, query,
		downloadedBefore.UTC(),
		service.TraceExportTaskStatusSucceeded,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.TraceExportTask, 0, limit)
	for rows.Next() {
		item, scanErr := scanTraceExportTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *traceExportTaskRepository) ClearFileForTask(ctx context.Context, id int64, updatedAt time.Time) (bool, error) {
	query := `
		UPDATE trace_export_tasks
		SET file_path = '',
			file_size_bytes = 0,
			updated_at = $2
		WHERE id = $1
			AND file_path <> ''
		RETURNING id
	`
	var returnedID int64
	err := scanSingleRow(ctx, r.sql, query, []any{id, updatedAt.UTC()}, &returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *traceExportTaskRepository) scanOneTask(ctx context.Context, query string, args ...any) (*service.TraceExportTask, error) {
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	item, err := scanTraceExportTask(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return item, nil
}

func scanTraceExportTask(scanner interface{ Scan(...any) error }) (*service.TraceExportTask, error) {
	task := &service.TraceExportTask{}
	var (
		filtersJSON      []byte
		errorMessage     sql.NullString
		canceledBy       sql.NullInt64
		canceledAt       sql.NullTime
		startedAt        sql.NullTime
		finishedAt       sql.NullTime
		downloadedAt     sql.NullTime
		downloadFilename sql.NullString
		filePath         sql.NullString
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Status,
		&task.Format,
		&filtersJSON,
		&task.IncludeRaw,
		&task.TargetRecords,
		&task.RequestedBy,
		&downloadFilename,
		&filePath,
		&task.FileSizeBytes,
		&task.TotalRecords,
		&task.ProcessedRecords,
		&errorMessage,
		&canceledBy,
		&canceledAt,
		&startedAt,
		&finishedAt,
		&downloadedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(filtersJSON) > 0 {
		if err := json.Unmarshal(filtersJSON, &task.Filters); err != nil {
			return nil, fmt.Errorf("parse trace export task filters: %w", err)
		}
	}
	if downloadFilename.Valid {
		task.DownloadFilename = downloadFilename.String
	}
	if filePath.Valid {
		task.FilePath = filePath.String
	}
	if errorMessage.Valid {
		task.ErrorMsg = &errorMessage.String
	}
	if canceledBy.Valid {
		value := canceledBy.Int64
		task.CanceledBy = &value
	}
	if canceledAt.Valid {
		value := canceledAt.Time
		task.CanceledAt = &value
	}
	if startedAt.Valid {
		value := startedAt.Time
		task.StartedAt = &value
	}
	if finishedAt.Valid {
		value := finishedAt.Time
		task.FinishedAt = &value
	}
	if downloadedAt.Valid {
		value := downloadedAt.Time
		task.DownloadedAt = &value
	}
	return task, nil
}
