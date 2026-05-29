package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type modelInteractionTraceRepository struct {
	sql sqlExecutor
}

func NewModelInteractionTraceRepository(sqlDB *sql.DB) service.ModelInteractionTraceRepository {
	return &modelInteractionTraceRepository{sql: sqlDB}
}

func (r *modelInteractionTraceRepository) Create(ctx context.Context, trace *service.ModelInteractionTrace) (bool, error) {
	if trace == nil {
		return false, nil
	}
	if trace.TaskID == "" {
		return false, fmt.Errorf("model interaction trace task_id is required")
	}
	if trace.DedupeHash == "" {
		return false, fmt.Errorf("model interaction trace dedupe_hash is required")
	}

	query := `
		INSERT INTO model_interaction_traces (
			task_id,
			prompt,
			candidates,
			tools,
			signature,
			meta,
			scaffold,
			scaffold_version,
			model,
			user_id,
			api_key_id,
			request_id,
			dedupe_hash
		) VALUES (
			$1,
			$2::json,
			$3::json,
			$4::json,
			$5::json,
			$6::json,
			$7::json,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13
		)
		ON CONFLICT (dedupe_hash) DO UPDATE SET
			task_id = EXCLUDED.task_id,
			signature = EXCLUDED.signature,
			meta = EXCLUDED.meta,
			scaffold = EXCLUDED.scaffold,
			scaffold_version = EXCLUDED.scaffold_version,
			model = COALESCE(EXCLUDED.model, model_interaction_traces.model),
			user_id = COALESCE(EXCLUDED.user_id, model_interaction_traces.user_id),
			api_key_id = COALESCE(EXCLUDED.api_key_id, model_interaction_traces.api_key_id)
			request_id = COALESCE(EXCLUDED.request_id, model_interaction_traces.request_id)
		WHERE (
			COALESCE(length(EXCLUDED.signature::text), 0) +
			COALESCE(length(EXCLUDED.meta::text), 0) +
			COALESCE(length(EXCLUDED.scaffold::text), 0)
		) > (
			COALESCE(length(model_interaction_traces.signature::text), 0) +
			COALESCE(length(model_interaction_traces.meta::text), 0) +
			COALESCE(length(model_interaction_traces.scaffold::text), 0)
		)
		RETURNING id, created_at
	`

	var createdAt time.Time
	err := scanSingleRow(ctx, r.sql, query, []any{
		trace.TaskID,
		jsonParam(trace.Prompt),
		jsonParam(trace.Candidates),
		jsonParam(trace.Tools),
		jsonParam(trace.Signature),
		jsonParam(trace.Meta),
		jsonParam(trace.Scaffold),
		trace.ScaffoldVersion,
		stringPtrParam(trace.Model),
		int64PtrParam(trace.UserID),
		int64PtrParam(trace.APIKeyID),
		stringPtrParam(trace.RequestID),
		trace.DedupeHash,
	}, &trace.ID, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	trace.CreatedAt = createdAt
	return true, nil
}

func (r *modelInteractionTraceRepository) ListAll(ctx context.Context) ([]*service.ModelInteractionTrace, error) {
	query := `
		SELECT
			id,
			task_id,
			prompt,
			candidates,
			tools,
			signature,
			meta,
			scaffold,
			scaffold_version,
			model,
			user_id,
			api_key_id,
			request_id,
			dedupe_hash,
			created_at
		FROM model_interaction_traces
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.ModelInteractionTrace, 0)
	for rows.Next() {
		item, scanErr := scanModelInteractionTrace(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *modelInteractionTraceRepository) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*service.ModelInteractionTrace, *pagination.PaginationResult, error) {
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		return nil, nil, fmt.Errorf("invalid model interaction trace time range")
	}

	conditions := make([]string, 0, 4)
	args := make([]any, 0, 4)

	if !startTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, startTime)
	}
	if !endTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, endTime)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM model_interaction_traces " + whereClause
	if err := scanSingleRow(ctx, r.sql, countQuery, args, &total); err != nil {
		return nil, nil, err
	}
	if total == 0 {
		return []*service.ModelInteractionTrace{}, paginationResultFromTotal(0, params), nil
	}

	listQuery := `
		SELECT
			id,
			task_id,
			prompt,
			candidates,
			tools,
			signature,
			meta,
			scaffold,
			scaffold_version,
			model,
			user_id,
			api_key_id,
			request_id,
			dedupe_hash,
			created_at
		FROM model_interaction_traces
	` + whereClause + `
		ORDER BY created_at DESC, id DESC
		LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)

	rows, err := r.sql.QueryContext(ctx, listQuery, append(args, params.Limit(), params.Offset())...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.ModelInteractionTrace, 0, params.Limit())
	for rows.Next() {
		item, scanErr := scanModelInteractionTrace(rows)
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

func scanModelInteractionTrace(rows *sql.Rows) (*service.ModelInteractionTrace, error) {
	item := &service.ModelInteractionTrace{}
	var (
		promptRaw     []byte
		candidatesRaw []byte
		toolsRaw      []byte
		metaRaw       []byte
		scaffoldRaw   []byte
		signatureRaw  []byte
		scaffoldVer   sql.NullString
		model         sql.NullString
		userID        sql.NullInt64
		apiKeyID      sql.NullInt64
		requestID     sql.NullString
	)
	if err := rows.Scan(
		&item.ID,
		&item.TaskID,
		&promptRaw,
		&candidatesRaw,
		&toolsRaw,
		&signatureRaw,
		&metaRaw,
		&scaffoldRaw,
		&scaffoldVer,
		&model,
		&userID,
		&apiKeyID,
		&requestID,
		&item.DedupeHash,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}

	item.Prompt = append(json.RawMessage(nil), promptRaw...)
	item.Candidates = append(json.RawMessage(nil), candidatesRaw...)
	item.Tools = append(json.RawMessage(nil), toolsRaw...)
	item.Signature = append(json.RawMessage(nil), signatureRaw...)
	item.Meta = append(json.RawMessage(nil), metaRaw...)
	item.Scaffold = append(json.RawMessage(nil), scaffoldRaw...)
	if scaffoldVer.Valid {
		item.ScaffoldVersion = scaffoldVer.String
	}
	if model.Valid {
		v := model.String
		item.Model = &v
	}
	if userID.Valid {
		v := userID.Int64
		item.UserID = &v
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		item.APIKeyID = &v
	}
	if requestID.Valid {
		v := requestID.String
		item.RequestID = &v
	}

	return item, nil
}

func jsonParam(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func stringPtrParam(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func int64PtrParam(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func joinConditions(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	out := conditions[0]
	for i := 1; i < len(conditions); i++ {
		out += " AND " + conditions[i]
	}
	return out
}
