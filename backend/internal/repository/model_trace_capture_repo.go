package repository

import (
	"bytes"
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

const modelTraceCaptureSelectColumns = `
	id,
	task_id,
	request_id,
	response_id,
	user_id,
	api_key_id,
	group_id,
	account_id,
	capture_rule_id,
	protocol,
	model,
	requested_model,
	upstream_model,
	request_content_type,
	response_content_type,
	input_tokens,
	output_tokens,
	total_tokens,
	upstream_status_code,
	scaffold,
	scaffold_version,
	prompt_json,
	candidates_json,
	tools_json,
	signature_json,
	meta_json,
	raw_request_json,
	raw_response_json,
	raw_request_text,
	raw_response_text,
	dedupe_hash,
	prompt_hash,
	created_at
`

type modelTraceCaptureRepository struct {
	sql sqlExecutor
}

func NewModelTraceCaptureRepository(sqlDB *sql.DB) service.ModelTraceCaptureRepository {
	return &modelTraceCaptureRepository{sql: sqlDB}
}

func (r *modelTraceCaptureRepository) Create(ctx context.Context, capture *service.ModelTraceCapture) (bool, error) {
	if capture == nil {
		return false, nil
	}
	if err := capture.Validate(); err != nil {
		return false, err
	}

	query := `
		INSERT INTO model_trace_captures (
			task_id,
			request_id,
			response_id,
			user_id,
			api_key_id,
			group_id,
			account_id,
			capture_rule_id,
			protocol,
			model,
			requested_model,
			upstream_model,
			request_content_type,
			response_content_type,
			input_tokens,
			output_tokens,
			total_tokens,
			upstream_status_code,
			scaffold,
			scaffold_version,
			prompt_json,
			candidates_json,
			tools_json,
			signature_json,
			meta_json,
			raw_request_json,
			raw_response_json,
			raw_request_text,
			raw_response_text,
			dedupe_hash,
			prompt_hash
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			$18,
			$19,
			$20,
			$21::jsonb,
			$22::jsonb,
			$23::jsonb,
			$24::jsonb,
			$25::jsonb,
			$26::jsonb,
			$27::jsonb,
			$28,
			$29,
			$30,
			$31
		)
		ON CONFLICT DO NOTHING
		RETURNING id, created_at
	`

	args := []any{
		capture.TaskID,
		stringPtrParam(capture.RequestID),
		stringPtrParam(capture.ResponseID),
		nullInt64(capture.UserID),
		nullInt64(capture.APIKeyID),
		nullInt64(capture.GroupID),
		nullInt64(capture.AccountID),
		nullInt64(capture.CaptureRuleID),
		capture.Protocol,
		capture.Model,
		traceCaptureOptionalString(capture.RequestedModel),
		traceCaptureOptionalString(capture.UpstreamModel),
		capture.RequestContentType,
		capture.ResponseContentType,
		nullInt64(capture.InputTokens),
		nullInt64(capture.OutputTokens),
		nullInt64(capture.TotalTokens),
		intPtrParam(capture.UpstreamStatusCode),
		capture.Scaffold,
		capture.ScaffoldVersion,
		jsonValueParam(capture.Prompt, "[]"),
		jsonValueParam(capture.Candidates, "[]"),
		jsonValueParam(capture.Tools, "[]"),
		jsonValueParam(capture.Signature, "[]"),
		jsonValueParam(capture.Meta, "{}"),
		jsonValueParam(capture.RawRequest, "{}"),
		jsonValueParam(capture.RawResponse, "{}"),
		capture.RawRequestText,
		capture.RawResponseText,
		capture.DedupeHash,
		capture.PromptHash,
	}

	var createdAt time.Time
	err := scanSingleRow(ctx, r.sql, query, args, &capture.ID, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		if existing, lookupErr := r.GetByTaskID(ctx, capture.TaskID); lookupErr == nil && existing != nil {
			capture.ID = existing.ID
			capture.CreatedAt = existing.CreatedAt
			return false, nil
		}
		if capture.DedupeHash != "" {
			if existing, lookupErr := r.GetByDedupeHash(ctx, capture.DedupeHash); lookupErr == nil && existing != nil {
				capture.ID = existing.ID
				capture.CreatedAt = existing.CreatedAt
				return false, nil
			}
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}

	capture.CreatedAt = createdAt
	return true, nil
}

func (r *modelTraceCaptureRepository) GetByID(ctx context.Context, id int64) (*service.ModelTraceCapture, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id is required")
	}
	return r.getOne(ctx, "id = $1", id)
}

func (r *modelTraceCaptureRepository) GetByTaskID(ctx context.Context, taskID string) (*service.ModelTraceCapture, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	return r.getOne(ctx, "task_id = $1", taskID)
}

func (r *modelTraceCaptureRepository) GetByDedupeHash(ctx context.Context, dedupeHash string) (*service.ModelTraceCapture, error) {
	dedupeHash = strings.TrimSpace(dedupeHash)
	if dedupeHash == "" {
		return nil, fmt.Errorf("dedupe_hash is required")
	}
	return r.getOne(ctx, "dedupe_hash = $1", dedupeHash)
}

func (r *modelTraceCaptureRepository) List(ctx context.Context, filter service.ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*service.ModelTraceCapture, *pagination.PaginationResult, error) {
	if filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime) {
		return nil, nil, fmt.Errorf("invalid model trace capture time range")
	}
	if filter.MinInputTokens != nil && *filter.MinInputTokens < 0 {
		return nil, nil, fmt.Errorf("min_input_tokens must be >= 0")
	}
	if filter.MaxInputTokens != nil && *filter.MaxInputTokens < 0 {
		return nil, nil, fmt.Errorf("max_input_tokens must be >= 0")
	}
	if filter.MinInputTokens != nil && filter.MaxInputTokens != nil && *filter.MinInputTokens > *filter.MaxInputTokens {
		return nil, nil, fmt.Errorf("invalid input token range")
	}
	if filter.MinOutputTokens != nil && *filter.MinOutputTokens < 0 {
		return nil, nil, fmt.Errorf("min_output_tokens must be >= 0")
	}
	if filter.MaxOutputTokens != nil && *filter.MaxOutputTokens < 0 {
		return nil, nil, fmt.Errorf("max_output_tokens must be >= 0")
	}
	if filter.MinOutputTokens != nil && filter.MaxOutputTokens != nil && *filter.MinOutputTokens > *filter.MaxOutputTokens {
		return nil, nil, fmt.Errorf("invalid output token range")
	}
	if filter.MinTotalTokens != nil && *filter.MinTotalTokens < 0 {
		return nil, nil, fmt.Errorf("min_total_tokens must be >= 0")
	}
	if filter.MaxTotalTokens != nil && *filter.MaxTotalTokens < 0 {
		return nil, nil, fmt.Errorf("max_total_tokens must be >= 0")
	}
	if filter.MinTotalTokens != nil && filter.MaxTotalTokens != nil && *filter.MinTotalTokens > *filter.MaxTotalTokens {
		return nil, nil, fmt.Errorf("invalid total token range")
	}

	conditions := make([]string, 0, 12)
	args := make([]any, 0, 12)

	model := strings.TrimSpace(filter.Model)
	if model != "" {
		conditions = append(conditions, fmt.Sprintf("model = $%d", len(args)+1))
		args = append(args, model)
	}
	if filter.UserID != nil {
		if *filter.UserID <= 0 {
			return nil, nil, fmt.Errorf("user_id must be > 0")
		}
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)+1))
		args = append(args, *filter.UserID)
	}
	if filter.APIKeyID != nil {
		if *filter.APIKeyID <= 0 {
			return nil, nil, fmt.Errorf("api_key_id must be > 0")
		}
		conditions = append(conditions, fmt.Sprintf("api_key_id = $%d", len(args)+1))
		args = append(args, *filter.APIKeyID)
	}
	if filter.CaptureRuleID != nil {
		if *filter.CaptureRuleID <= 0 {
			return nil, nil, fmt.Errorf("capture_rule_id must be > 0")
		}
		conditions = append(conditions, fmt.Sprintf("capture_rule_id = $%d", len(args)+1))
		args = append(args, *filter.CaptureRuleID)
	}
	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, *filter.EndTime)
	}
	keyword := strings.TrimSpace(filter.Keyword)
	if keyword != "" {
		placeholder := len(args) + 1
		conditions = append(conditions, fmt.Sprintf(`(
			prompt_json::text ILIKE $%[1]d ESCAPE '\'
			OR candidates_json::text ILIKE $%[1]d ESCAPE '\'
			OR tools_json::text ILIKE $%[1]d ESCAPE '\'
			OR signature_json::text ILIKE $%[1]d ESCAPE '\'
			OR meta_json::text ILIKE $%[1]d ESCAPE '\'
			OR raw_request_json::text ILIKE $%[1]d ESCAPE '\'
			OR raw_response_json::text ILIKE $%[1]d ESCAPE '\'
			OR raw_request_text ILIKE $%[1]d ESCAPE '\'
			OR raw_response_text ILIKE $%[1]d ESCAPE '\'
		)`, placeholder))
		args = append(args, traceCaptureLikePattern(keyword))
	}
	if filter.MinInputTokens != nil {
		conditions = append(conditions, fmt.Sprintf("input_tokens >= $%d", len(args)+1))
		args = append(args, *filter.MinInputTokens)
	}
	if filter.MaxInputTokens != nil {
		conditions = append(conditions, fmt.Sprintf("input_tokens <= $%d", len(args)+1))
		args = append(args, *filter.MaxInputTokens)
	}
	if filter.MinOutputTokens != nil {
		conditions = append(conditions, fmt.Sprintf("output_tokens >= $%d", len(args)+1))
		args = append(args, *filter.MinOutputTokens)
	}
	if filter.MaxOutputTokens != nil {
		conditions = append(conditions, fmt.Sprintf("output_tokens <= $%d", len(args)+1))
		args = append(args, *filter.MaxOutputTokens)
	}
	if filter.MinTotalTokens != nil {
		conditions = append(conditions, fmt.Sprintf("total_tokens >= $%d", len(args)+1))
		args = append(args, *filter.MinTotalTokens)
	}
	if filter.MaxTotalTokens != nil {
		conditions = append(conditions, fmt.Sprintf("total_tokens <= $%d", len(args)+1))
		args = append(args, *filter.MaxTotalTokens)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM model_trace_captures " + whereClause
	if err := scanSingleRow(ctx, r.sql, countQuery, args, &total); err != nil {
		return nil, nil, err
	}
	if total == 0 {
		return []*service.ModelTraceCapture{}, paginationResultFromTotal(0, params), nil
	}

	listQuery := `
		SELECT
	` + modelTraceCaptureSelectColumns + `
		FROM model_trace_captures
	` + whereClause + `
		ORDER BY created_at DESC, id DESC
		LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)

	rows, err := r.sql.QueryContext(ctx, listQuery, append(args, params.Limit(), params.Offset())...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.ModelTraceCapture, 0, params.Limit())
	for rows.Next() {
		item, scanErr := scanModelTraceCapture(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *modelTraceCaptureRepository) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*service.ModelTraceCapture, *pagination.PaginationResult, error) {
	filter := service.ModelTraceCaptureListFilter{}
	if !startTime.IsZero() {
		filter.StartTime = &startTime
	}
	if !endTime.IsZero() {
		filter.EndTime = &endTime
	}
	return r.List(ctx, filter, params)
}

func (r *modelTraceCaptureRepository) DeleteByID(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("id is required")
	}
	res, err := r.sql.ExecContext(ctx, "DELETE FROM model_trace_captures WHERE id = $1", id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *modelTraceCaptureRepository) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	ids = normalizePositiveIDs(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	for i, id := range ids {
		args = append(args, id)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}
	query := "DELETE FROM model_trace_captures WHERE id IN (" + strings.Join(placeholders, ", ") + ")"
	res, err := r.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *modelTraceCaptureRepository) getOne(ctx context.Context, predicate string, args ...any) (*service.ModelTraceCapture, error) {
	query := `
		SELECT
	` + modelTraceCaptureSelectColumns + `
		FROM model_trace_captures
		WHERE ` + predicate + `
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`

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

	item, err := scanModelTraceCapture(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return item, nil
}

func scanModelTraceCapture(scanner interface{ Scan(...any) error }) (*service.ModelTraceCapture, error) {
	item := &service.ModelTraceCapture{}
	var (
		requestID           sql.NullString
		responseID          sql.NullString
		userID              sql.NullInt64
		apiKeyID            sql.NullInt64
		groupID             sql.NullInt64
		accountID           sql.NullInt64
		captureRuleID       sql.NullInt64
		requested           sql.NullString
		upstream            sql.NullString
		requestContentType  sql.NullString
		responseContentType sql.NullString
		inputTokens         sql.NullInt64
		outputTokens        sql.NullInt64
		totalTokens         sql.NullInt64
		upstreamStatusCode  sql.NullInt64
		rawPrompt           []byte
		rawCandidates       []byte
		rawTools            []byte
		rawSignature        []byte
		rawMeta             []byte
		rawRequest          []byte
		rawResponse         []byte
		rawRequestText      sql.NullString
		rawResponseText     sql.NullString
	)
	if err := scanner.Scan(
		&item.ID,
		&item.TaskID,
		&requestID,
		&responseID,
		&userID,
		&apiKeyID,
		&groupID,
		&accountID,
		&captureRuleID,
		&item.Protocol,
		&item.Model,
		&requested,
		&upstream,
		&requestContentType,
		&responseContentType,
		&inputTokens,
		&outputTokens,
		&totalTokens,
		&upstreamStatusCode,
		&item.Scaffold,
		&item.ScaffoldVersion,
		&rawPrompt,
		&rawCandidates,
		&rawTools,
		&rawSignature,
		&rawMeta,
		&rawRequest,
		&rawResponse,
		&rawRequestText,
		&rawResponseText,
		&item.DedupeHash,
		&item.PromptHash,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}

	item.Prompt = append(json.RawMessage(nil), rawPrompt...)
	item.Candidates = append(json.RawMessage(nil), rawCandidates...)
	item.Tools = append(json.RawMessage(nil), rawTools...)
	item.Signature = append(json.RawMessage(nil), rawSignature...)
	item.Meta = append(json.RawMessage(nil), rawMeta...)
	item.RawRequest = append(json.RawMessage(nil), rawRequest...)
	item.RawResponse = append(json.RawMessage(nil), rawResponse...)

	if requestID.Valid && strings.TrimSpace(requestID.String) != "" {
		value := requestID.String
		item.RequestID = &value
	}
	if responseID.Valid && strings.TrimSpace(responseID.String) != "" {
		value := responseID.String
		item.ResponseID = &value
	}
	if userID.Valid {
		value := userID.Int64
		item.UserID = &value
	}
	if apiKeyID.Valid {
		value := apiKeyID.Int64
		item.APIKeyID = &value
	}
	if groupID.Valid {
		value := groupID.Int64
		item.GroupID = &value
	}
	if accountID.Valid {
		value := accountID.Int64
		item.AccountID = &value
	}
	if captureRuleID.Valid {
		value := captureRuleID.Int64
		item.CaptureRuleID = &value
	}
	if requested.Valid && strings.TrimSpace(requested.String) != "" {
		value := requested.String
		item.RequestedModel = &value
	}
	if upstream.Valid && strings.TrimSpace(upstream.String) != "" {
		value := upstream.String
		item.UpstreamModel = &value
	}
	if requestContentType.Valid {
		item.RequestContentType = requestContentType.String
	}
	if responseContentType.Valid {
		item.ResponseContentType = responseContentType.String
	}
	if inputTokens.Valid {
		value := inputTokens.Int64
		item.InputTokens = &value
	}
	if outputTokens.Valid {
		value := outputTokens.Int64
		item.OutputTokens = &value
	}
	if totalTokens.Valid {
		value := totalTokens.Int64
		item.TotalTokens = &value
	}
	if upstreamStatusCode.Valid {
		value := int(upstreamStatusCode.Int64)
		item.UpstreamStatusCode = &value
	}
	if rawRequestText.Valid {
		item.RawRequestText = rawRequestText.String
	}
	if rawResponseText.Valid {
		item.RawResponseText = rawResponseText.String
	}

	return item, nil
}

func traceCaptureOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func jsonValueParam(value json.RawMessage, defaultJSON string) any {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return defaultJSON
	}
	return string(trimmed)
}

func intPtrParam(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func traceCaptureLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(value) + "%"
}

func normalizePositiveIDs(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
