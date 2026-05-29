package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelTraceCaptureRuleRepositoryCRUD(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelTraceCaptureRuleRepository{sql: db}

	rule := newModelTraceCaptureRuleFixture()
	require.NoError(t, rule.Validate())

	createdAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO model_trace_capture_rules")).
		WithArgs(
			rule.Name,
			rule.Enabled,
			rule.Priority,
			`["gpt-4.1","claude-*"]`,
			`[42,84]`,
			`[7,8]`,
			`["incident","vip"]`,
			nullInt64(rule.MinTokens),
			nullInt64(rule.MaxTokens),
			rule.SamplingRatio,
			traceCaptureRuleTimeParam(rule.ActiveFrom),
			traceCaptureRuleTimeParam(rule.ActiveTo),
		).
		WillReturnRows(modelTraceCaptureRuleRows(1, rule, createdAt, updatedAt))

	created, err := repo.Create(context.Background(), rule)
	require.NoError(t, err)
	assertModelTraceCaptureRuleMatches(t, int64(1), rule, created, createdAt, updatedAt)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1")).
		WithArgs(int64(1)).
		WillReturnRows(modelTraceCaptureRuleRows(1, rule, createdAt, updatedAt))

	got, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assertModelTraceCaptureRuleMatches(t, int64(1), rule, got, createdAt, updatedAt)

	updated := newModelTraceCaptureRuleFixture()
	updated.ID = 1
	updated.Name = "Priority rule"
	updated.Enabled = false
	updated.Priority = 20
	updated.ModelPatterns = []string{"gpt-4.1-mini"}
	updated.UserIDs = []int64{99}
	updated.APIKeyIDs = []int64{9}
	updated.Keywords = []string{"audit"}
	minTokens := int64(1)
	maxTokens := int64(10)
	updated.MinTokens = &minTokens
	updated.MaxTokens = &maxTokens
	updated.SamplingRatio = 0.25
	activeFrom := createdAt.Add(time.Hour)
	activeTo := activeFrom.Add(2 * time.Hour)
	updated.ActiveFrom = &activeFrom
	updated.ActiveTo = &activeTo
	require.NoError(t, updated.Validate())

	updatedAt = createdAt.Add(30 * time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE model_trace_capture_rules")).
		WithArgs(
			updated.ID,
			updated.Name,
			updated.Enabled,
			updated.Priority,
			`["gpt-4.1-mini"]`,
			`[99]`,
			`[9]`,
			`["audit"]`,
			nullInt64(updated.MinTokens),
			nullInt64(updated.MaxTokens),
			updated.SamplingRatio,
			traceCaptureRuleTimeParam(updated.ActiveFrom),
			traceCaptureRuleTimeParam(updated.ActiveTo),
		).
		WillReturnRows(modelTraceCaptureRuleRows(updated.ID, updated, createdAt, updatedAt))

	storedUpdated, err := repo.Update(context.Background(), updated)
	require.NoError(t, err)
	assertModelTraceCaptureRuleMatches(t, updated.ID, updated, storedUpdated, createdAt, updatedAt)

	mock.ExpectQuery(regexp.QuoteMeta("FROM model_trace_capture_rules")).
		WillReturnRows(modelTraceCaptureRuleRows(updated.ID, updated, createdAt, updatedAt))

	items, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	assertModelTraceCaptureRuleMatches(t, updated.ID, updated, items[0], createdAt, updatedAt)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM model_trace_capture_rules WHERE id = $1")).
		WithArgs(updated.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	deleted, err := repo.DeleteByID(context.Background(), updated.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	require.NoError(t, mock.ExpectationsWereMet())
}

func newModelTraceCaptureRuleFixture() *service.ModelTraceCaptureRule {
	minTokens := int64(100)
	maxTokens := int64(500)
	activeFrom := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	activeTo := activeFrom.Add(24 * time.Hour)

	return &service.ModelTraceCaptureRule{
		Name:          "Incident rule",
		Enabled:       true,
		Priority:      10,
		ModelPatterns: []string{"gpt-4.1", "claude-*"},
		UserIDs:       []int64{42, 84},
		APIKeyIDs:     []int64{7, 8},
		Keywords:      []string{"incident", "vip"},
		MinTokens:     &minTokens,
		MaxTokens:     &maxTokens,
		SamplingRatio: 0.5,
		ActiveFrom:    &activeFrom,
		ActiveTo:      &activeTo,
	}
}

func modelTraceCaptureRuleRows(id int64, rule *service.ModelTraceCaptureRule, createdAt, updatedAt time.Time) *sqlmock.Rows {
	var minTokens any
	var maxTokens any
	var activeFrom any
	var activeTo any
	if rule.MinTokens != nil {
		minTokens = *rule.MinTokens
	}
	if rule.MaxTokens != nil {
		maxTokens = *rule.MaxTokens
	}
	if rule.ActiveFrom != nil {
		activeFrom = *rule.ActiveFrom
	}
	if rule.ActiveTo != nil {
		activeTo = *rule.ActiveTo
	}

	return sqlmock.NewRows([]string{
		"id",
		"name",
		"enabled",
		"priority",
		"model_patterns",
		"user_ids",
		"api_key_ids",
		"keywords",
		"min_tokens",
		"max_tokens",
		"sampling_ratio",
		"active_from",
		"active_to",
		"created_at",
		"updated_at",
	}).AddRow(
		id,
		rule.Name,
		rule.Enabled,
		rule.Priority,
		[]byte(mustJSONString(rule.ModelPatterns)),
		[]byte(mustJSONString(rule.UserIDs)),
		[]byte(mustJSONString(rule.APIKeyIDs)),
		[]byte(mustJSONString(rule.Keywords)),
		minTokens,
		maxTokens,
		rule.SamplingRatio,
		activeFrom,
		activeTo,
		createdAt,
		updatedAt,
	)
}

func mustJSONString(value any) string {
	raw, err := traceCaptureRuleJSONParam(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func assertModelTraceCaptureRuleMatches(t *testing.T, id int64, expected, actual *service.ModelTraceCaptureRule, createdAt, updatedAt time.Time) {
	t.Helper()

	require.NotNil(t, actual)
	require.Equal(t, id, actual.ID)
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.Enabled, actual.Enabled)
	require.Equal(t, expected.Priority, actual.Priority)
	require.Equal(t, expected.ModelPatterns, actual.ModelPatterns)
	require.Equal(t, expected.UserIDs, actual.UserIDs)
	require.Equal(t, expected.APIKeyIDs, actual.APIKeyIDs)
	require.Equal(t, expected.Keywords, actual.Keywords)
	require.Equal(t, expected.MinTokens, actual.MinTokens)
	require.Equal(t, expected.MaxTokens, actual.MaxTokens)
	require.Equal(t, expected.SamplingRatio, actual.SamplingRatio)
	require.Equal(t, expected.ActiveFrom, actual.ActiveFrom)
	require.Equal(t, expected.ActiveTo, actual.ActiveTo)
	require.Equal(t, createdAt, actual.CreatedAt)
	require.Equal(t, updatedAt, actual.UpdatedAt)
}
