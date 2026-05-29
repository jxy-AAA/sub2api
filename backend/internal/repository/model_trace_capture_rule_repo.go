package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const modelTraceCaptureRuleSelectColumns = `
	id,
	name,
	enabled,
	priority,
	model_patterns,
	user_ids,
	api_key_ids,
	keywords,
	min_tokens,
	max_tokens,
	sampling_ratio,
	active_from,
	active_to,
	created_at,
	updated_at
`

type modelTraceCaptureRuleRepository struct {
	sql sqlExecutor
}

func NewModelTraceCaptureRuleRepository(sqlDB *sql.DB) service.ModelTraceCaptureRuleRepository {
	return &modelTraceCaptureRuleRepository{sql: sqlDB}
}

func (r *modelTraceCaptureRuleRepository) Create(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error) {
	if rule == nil {
		return nil, nil
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	modelPatterns, err := traceCaptureRuleJSONParam(rule.ModelPatterns)
	if err != nil {
		return nil, err
	}
	userIDs, err := traceCaptureRuleJSONParam(rule.UserIDs)
	if err != nil {
		return nil, err
	}
	apiKeyIDs, err := traceCaptureRuleJSONParam(rule.APIKeyIDs)
	if err != nil {
		return nil, err
	}
	keywords, err := traceCaptureRuleJSONParam(rule.Keywords)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO model_trace_capture_rules (
			name,
			enabled,
			priority,
			model_patterns,
			user_ids,
			api_key_ids,
			keywords,
			min_tokens,
			max_tokens,
			sampling_ratio,
			active_from,
			active_to
		) VALUES (
			$1,
			$2,
			$3,
			$4::jsonb,
			$5::jsonb,
			$6::jsonb,
			$7::jsonb,
			$8,
			$9,
			$10,
			$11,
			$12
		)
		RETURNING
	` + modelTraceCaptureRuleSelectColumns

	return r.getOne(ctx, query, []any{
		rule.Name,
		rule.Enabled,
		rule.Priority,
		modelPatterns,
		userIDs,
		apiKeyIDs,
		keywords,
		nullInt64(rule.MinTokens),
		nullInt64(rule.MaxTokens),
		rule.SamplingRatio,
		traceCaptureRuleTimeParam(rule.ActiveFrom),
		traceCaptureRuleTimeParam(rule.ActiveTo),
	})
}

func (r *modelTraceCaptureRuleRepository) Update(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error) {
	if rule == nil {
		return nil, nil
	}
	if rule.ID <= 0 {
		return nil, fmt.Errorf("id is required")
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	modelPatterns, err := traceCaptureRuleJSONParam(rule.ModelPatterns)
	if err != nil {
		return nil, err
	}
	userIDs, err := traceCaptureRuleJSONParam(rule.UserIDs)
	if err != nil {
		return nil, err
	}
	apiKeyIDs, err := traceCaptureRuleJSONParam(rule.APIKeyIDs)
	if err != nil {
		return nil, err
	}
	keywords, err := traceCaptureRuleJSONParam(rule.Keywords)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE model_trace_capture_rules
		SET
			name = $2,
			enabled = $3,
			priority = $4,
			model_patterns = $5::jsonb,
			user_ids = $6::jsonb,
			api_key_ids = $7::jsonb,
			keywords = $8::jsonb,
			min_tokens = $9,
			max_tokens = $10,
			sampling_ratio = $11,
			active_from = $12,
			active_to = $13,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
	` + modelTraceCaptureRuleSelectColumns

	return r.getOne(ctx, query, []any{
		rule.ID,
		rule.Name,
		rule.Enabled,
		rule.Priority,
		modelPatterns,
		userIDs,
		apiKeyIDs,
		keywords,
		nullInt64(rule.MinTokens),
		nullInt64(rule.MaxTokens),
		rule.SamplingRatio,
		traceCaptureRuleTimeParam(rule.ActiveFrom),
		traceCaptureRuleTimeParam(rule.ActiveTo),
	})
}

func (r *modelTraceCaptureRuleRepository) GetByID(ctx context.Context, id int64) (*service.ModelTraceCaptureRule, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id is required")
	}

	query := `
		SELECT
	` + modelTraceCaptureRuleSelectColumns + `
		FROM model_trace_capture_rules
		WHERE id = $1
	`
	return r.getOne(ctx, query, []any{id})
}

func (r *modelTraceCaptureRuleRepository) List(ctx context.Context) ([]*service.ModelTraceCaptureRule, error) {
	query := `
		SELECT
	` + modelTraceCaptureRuleSelectColumns + `
		FROM model_trace_capture_rules
		ORDER BY priority DESC, id ASC
	`

	rows, err := r.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.ModelTraceCaptureRule, 0)
	for rows.Next() {
		item, scanErr := scanModelTraceCaptureRule(rows)
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

func (r *modelTraceCaptureRuleRepository) DeleteByID(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("id is required")
	}

	res, err := r.sql.ExecContext(ctx, "DELETE FROM model_trace_capture_rules WHERE id = $1", id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *modelTraceCaptureRuleRepository) getOne(ctx context.Context, query string, args []any) (*service.ModelTraceCaptureRule, error) {
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

	item, err := scanModelTraceCaptureRule(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return item, nil
}

func scanModelTraceCaptureRule(scanner interface{ Scan(...any) error }) (*service.ModelTraceCaptureRule, error) {
	item := &service.ModelTraceCaptureRule{}
	var (
		modelPatterns []byte
		userIDs       []byte
		apiKeyIDs     []byte
		keywords      []byte
		minTokens     sql.NullInt64
		maxTokens     sql.NullInt64
		activeFrom    sql.NullTime
		activeTo      sql.NullTime
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.Enabled,
		&item.Priority,
		&modelPatterns,
		&userIDs,
		&apiKeyIDs,
		&keywords,
		&minTokens,
		&maxTokens,
		&item.SamplingRatio,
		&activeFrom,
		&activeTo,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	var err error
	if item.ModelPatterns, err = decodeTraceCaptureRuleStrings(modelPatterns); err != nil {
		return nil, err
	}
	if item.UserIDs, err = decodeTraceCaptureRuleInt64s(userIDs); err != nil {
		return nil, err
	}
	if item.APIKeyIDs, err = decodeTraceCaptureRuleInt64s(apiKeyIDs); err != nil {
		return nil, err
	}
	if item.Keywords, err = decodeTraceCaptureRuleStrings(keywords); err != nil {
		return nil, err
	}
	if minTokens.Valid {
		value := minTokens.Int64
		item.MinTokens = &value
	}
	if maxTokens.Valid {
		value := maxTokens.Int64
		item.MaxTokens = &value
	}
	if activeFrom.Valid {
		value := activeFrom.Time
		item.ActiveFrom = &value
	}
	if activeTo.Valid {
		value := activeTo.Time
		item.ActiveTo = &value
	}

	return item, nil
}

func traceCaptureRuleJSONParam(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func traceCaptureRuleTimeParam(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func decodeTraceCaptureRuleStrings(raw []byte) ([]string, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}, nil
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return []string{}, nil
	}
	return values, nil
}

func decodeTraceCaptureRuleInt64s(raw []byte) ([]int64, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return []int64{}, nil
	}

	var values []int64
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return []int64{}, nil
	}
	return values, nil
}
