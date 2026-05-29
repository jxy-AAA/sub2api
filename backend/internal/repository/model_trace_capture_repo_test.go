package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelTraceCaptureRepositoryCreateGetListAndDelete(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelTraceCaptureRepository{sql: db}

	capture := newModelTraceCaptureFixture()
	require.NoError(t, capture.Validate())

	createdAt := time.Date(2026, 5, 27, 12, 34, 56, 0, time.UTC)
	start := createdAt.Add(-time.Hour)
	end := createdAt.Add(time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO model_trace_captures")).
		WithArgs(modelTraceCaptureInsertArgs(capture)...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(99), createdAt))

	inserted, err := repo.Create(context.Background(), capture)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, int64(99), capture.ID)
	require.Equal(t, createdAt, capture.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE task_id = $1")).
		WithArgs(capture.TaskID).
		WillReturnRows(modelTraceCaptureRows(capture, createdAt))

	gotByTaskID, err := repo.GetByTaskID(context.Background(), capture.TaskID)
	require.NoError(t, err)
	assertModelTraceCaptureMatches(t, capture, gotByTaskID)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1")).
		WithArgs(capture.ID).
		WillReturnRows(modelTraceCaptureRows(capture, createdAt))

	gotByID, err := repo.GetByID(context.Background(), capture.ID)
	require.NoError(t, err)
	assertModelTraceCaptureMatches(t, capture, gotByID)

	minInput := int64(100)
	maxInput := int64(200)
	minOutput := int64(50)
	maxOutput := int64(100)
	minTotal := int64(150)
	maxTotal := int64(300)
	filter := service.ModelTraceCaptureListFilter{
		Model:           capture.Model,
		UserID:          capture.UserID,
		APIKeyID:        capture.APIKeyID,
		CaptureRuleID:   capture.CaptureRuleID,
		StartTime:       &start,
		EndTime:         &end,
		Keyword:         "hello 100%",
		MinInputTokens:  &minInput,
		MaxInputTokens:  &maxInput,
		MinOutputTokens: &minOutput,
		MaxOutputTokens: &maxOutput,
		MinTotalTokens:  &minTotal,
		MaxTotalTokens:  &maxTotal,
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM model_trace_captures")).
		WithArgs(
			capture.Model,
			*capture.UserID,
			*capture.APIKeyID,
			*capture.CaptureRuleID,
			start,
			end,
			traceCaptureLikePattern(filter.Keyword),
			minInput,
			maxInput,
			minOutput,
			maxOutput,
			minTotal,
			maxTotal,
		).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta("FROM model_trace_captures")).
		WithArgs(
			capture.Model,
			*capture.UserID,
			*capture.APIKeyID,
			*capture.CaptureRuleID,
			start,
			end,
			traceCaptureLikePattern(filter.Keyword),
			minInput,
			maxInput,
			minOutput,
			maxOutput,
			minTotal,
			maxTotal,
			20,
			0,
		).
		WillReturnRows(modelTraceCaptureRows(capture, createdAt))

	items, result, err := repo.List(context.Background(), filter, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	assertModelTraceCaptureMatches(t, capture, items[0])

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM model_trace_captures WHERE id = $1")).
		WithArgs(capture.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	deleted, err := repo.DeleteByID(context.Background(), capture.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM model_trace_captures WHERE id IN ($1, $2)")).
		WithArgs(capture.ID, int64(100)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	deletedCount, err := repo.DeleteByIDs(context.Background(), []int64{capture.ID, 100, capture.ID, 0, -1})
	require.NoError(t, err)
	require.Equal(t, int64(2), deletedCount)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceCaptureRepositoryCreateDuplicateSkipsInsert(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelTraceCaptureRepository{sql: db}

	capture := newModelTraceCaptureFixture()
	require.NoError(t, capture.Validate())

	createdAt := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO model_trace_captures")).
		WithArgs(modelTraceCaptureInsertArgs(capture)...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}))

	existing := newModelTraceCaptureFixture()
	existing.ID = 321
	require.NoError(t, existing.Validate())
	mock.ExpectQuery(regexp.QuoteMeta("WHERE task_id = $1")).
		WithArgs(capture.TaskID).
		WillReturnRows(modelTraceCaptureRows(existing, createdAt))

	inserted, err := repo.Create(context.Background(), capture)
	require.NoError(t, err)
	require.False(t, inserted)
	require.Equal(t, existing.ID, capture.ID)
	require.Equal(t, createdAt, capture.CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceCaptureServiceRoundTripUsesCanonicalTrimmedTaskIDAndModel(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &modelTraceCaptureRepository{sql: db}
	traceService := service.NewModelTraceCaptureService(repo)

	capture := newModelTraceCaptureFixture()
	capture.TaskID = "  task-canonical  "
	capture.Protocol = "  openai.responses  "
	capture.Model = "  gpt-4.1  "
	capture.Scaffold = "  claude-code  "
	capture.ScaffoldVersion = "  2.1.140  "

	expected := newModelTraceCaptureFixture()
	expected.ID = 77
	expected.TaskID = "task-canonical"
	expected.Protocol = "openai.responses"
	expected.Model = "gpt-4.1"
	expected.Scaffold = "claude-code"
	expected.ScaffoldVersion = "2.1.140"
	require.NoError(t, expected.Validate())

	createdAt := time.Date(2026, 5, 27, 15, 0, 0, 0, time.UTC)
	expected.CreatedAt = createdAt

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO model_trace_captures")).
		WithArgs(modelTraceCaptureInsertArgs(expected)...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(expected.ID, createdAt))

	inserted, err := traceService.Create(context.Background(), capture)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, expected.TaskID, capture.TaskID)
	require.Equal(t, expected.Protocol, capture.Protocol)
	require.Equal(t, expected.Model, capture.Model)
	require.Equal(t, expected.Scaffold, capture.Scaffold)
	require.Equal(t, expected.ScaffoldVersion, capture.ScaffoldVersion)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE task_id = $1")).
		WithArgs(expected.TaskID).
		WillReturnRows(modelTraceCaptureRows(expected, createdAt))

	gotByTaskID, err := traceService.GetByTaskID(context.Background(), "  task-canonical  ")
	require.NoError(t, err)
	assertModelTraceCaptureMatches(t, expected, gotByTaskID)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM model_trace_captures")).
		WithArgs(expected.Model).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta("FROM model_trace_captures")).
		WithArgs(expected.Model, 20, 0).
		WillReturnRows(modelTraceCaptureRows(expected, createdAt))

	items, page, err := traceService.List(
		context.Background(),
		service.ModelTraceCaptureListFilter{Model: "  gpt-4.1  "},
		pagination.PaginationParams{Page: 1, PageSize: 20},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, items, 1)
	require.Equal(t, expected.Model, items[0].Model)
	require.Equal(t, expected.TaskID, items[0].TaskID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func newModelTraceCaptureFixture() *service.ModelTraceCapture {
	requestID := "req-001"
	responseID := "resp-001"
	userID := int64(42)
	apiKeyID := int64(7)
	groupID := int64(9)
	accountID := int64(11)
	captureRuleID := int64(12)
	requestedModel := "gpt-4.1"
	upstreamModel := "gpt-4.1-mini"
	inputTokens := int64(120)
	outputTokens := int64(80)
	totalTokens := int64(200)
	upstreamStatusCode := 200

	return &service.ModelTraceCapture{
		ID:                  99,
		TaskID:              "task-001",
		RequestID:           &requestID,
		ResponseID:          &responseID,
		UserID:              &userID,
		APIKeyID:            &apiKeyID,
		GroupID:             &groupID,
		AccountID:           &accountID,
		CaptureRuleID:       &captureRuleID,
		Protocol:            "openai.responses",
		Model:               "gpt-4.1",
		RequestedModel:      &requestedModel,
		UpstreamModel:       &upstreamModel,
		RequestContentType:  "application/json",
		ResponseContentType: "text/event-stream",
		InputTokens:         &inputTokens,
		OutputTokens:        &outputTokens,
		TotalTokens:         &totalTokens,
		UpstreamStatusCode:  &upstreamStatusCode,
		Scaffold:            "claude-code",
		ScaffoldVersion:     "2.1.140",
		Prompt:              json.RawMessage(`[{"role":"system","content":"system"},{"role":"user","content":"hello 100%"}]`),
		Candidates:          json.RawMessage(`[{"index":0,"message":{"role":"assistant","content":"done"}}]`),
		Tools:               json.RawMessage(`[{"type":"function","function":{"name":"search"}}]`),
		Signature:           json.RawMessage(`[{"turn":1,"signature":"sig-1"}]`),
		Meta:                json.RawMessage(`{"time":"2026-05-27T00:00:00Z","model":"gpt-4.1"}`),
		RawRequest:          json.RawMessage(`{"input":"hello 100%"}`),
		RawResponse:         json.RawMessage(`{"output":"done"}`),
		RawRequestText:      `{"input":"hello 100%"}`,
		RawResponseText:     "data: {\"output\":\"done\"}",
	}
}

func modelTraceCaptureInsertArgs(capture *service.ModelTraceCapture) []driver.Value {
	return []driver.Value{
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
		*capture.RequestedModel,
		*capture.UpstreamModel,
		capture.RequestContentType,
		capture.ResponseContentType,
		nullInt64(capture.InputTokens),
		nullInt64(capture.OutputTokens),
		nullInt64(capture.TotalTokens),
		intPtrParam(capture.UpstreamStatusCode),
		capture.Scaffold,
		capture.ScaffoldVersion,
		string(capture.Prompt),
		string(capture.Candidates),
		string(capture.Tools),
		string(capture.Signature),
		string(capture.Meta),
		string(capture.RawRequest),
		string(capture.RawResponse),
		capture.RawRequestText,
		capture.RawResponseText,
		capture.DedupeHash,
		capture.PromptHash,
	}
}

func modelTraceCaptureRows(capture *service.ModelTraceCapture, createdAt time.Time) *sqlmock.Rows {
	var (
		requestID          any
		responseID         any
		userID             any
		apiKeyID           any
		groupID            any
		accountID          any
		captureRuleID      any
		requestedModel     any
		upstreamModel      any
		inputTokens        any
		outputTokens       any
		totalTokens        any
		upstreamStatusCode any
	)
	if capture.RequestID != nil {
		requestID = *capture.RequestID
	}
	if capture.ResponseID != nil {
		responseID = *capture.ResponseID
	}
	if capture.UserID != nil {
		userID = *capture.UserID
	}
	if capture.APIKeyID != nil {
		apiKeyID = *capture.APIKeyID
	}
	if capture.GroupID != nil {
		groupID = *capture.GroupID
	}
	if capture.AccountID != nil {
		accountID = *capture.AccountID
	}
	if capture.CaptureRuleID != nil {
		captureRuleID = *capture.CaptureRuleID
	}
	if capture.RequestedModel != nil {
		requestedModel = *capture.RequestedModel
	}
	if capture.UpstreamModel != nil {
		upstreamModel = *capture.UpstreamModel
	}
	if capture.InputTokens != nil {
		inputTokens = *capture.InputTokens
	}
	if capture.OutputTokens != nil {
		outputTokens = *capture.OutputTokens
	}
	if capture.TotalTokens != nil {
		totalTokens = *capture.TotalTokens
	}
	if capture.UpstreamStatusCode != nil {
		upstreamStatusCode = *capture.UpstreamStatusCode
	}

	return sqlmock.NewRows([]string{
		"id",
		"task_id",
		"request_id",
		"response_id",
		"user_id",
		"api_key_id",
		"group_id",
		"account_id",
		"capture_rule_id",
		"protocol",
		"model",
		"requested_model",
		"upstream_model",
		"request_content_type",
		"response_content_type",
		"input_tokens",
		"output_tokens",
		"total_tokens",
		"upstream_status_code",
		"scaffold",
		"scaffold_version",
		"prompt_json",
		"candidates_json",
		"tools_json",
		"signature_json",
		"meta_json",
		"raw_request_json",
		"raw_response_json",
		"raw_request_text",
		"raw_response_text",
		"dedupe_hash",
		"prompt_hash",
		"created_at",
	}).AddRow(
		capture.ID,
		capture.TaskID,
		requestID,
		responseID,
		userID,
		apiKeyID,
		groupID,
		accountID,
		captureRuleID,
		capture.Protocol,
		capture.Model,
		requestedModel,
		upstreamModel,
		capture.RequestContentType,
		capture.ResponseContentType,
		inputTokens,
		outputTokens,
		totalTokens,
		upstreamStatusCode,
		capture.Scaffold,
		capture.ScaffoldVersion,
		[]byte(capture.Prompt),
		[]byte(capture.Candidates),
		[]byte(capture.Tools),
		[]byte(capture.Signature),
		[]byte(capture.Meta),
		[]byte(capture.RawRequest),
		[]byte(capture.RawResponse),
		capture.RawRequestText,
		capture.RawResponseText,
		capture.DedupeHash,
		capture.PromptHash,
		createdAt,
	)
}

func assertModelTraceCaptureMatches(t *testing.T, expected, actual *service.ModelTraceCapture) {
	t.Helper()

	require.NotNil(t, actual)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.TaskID, actual.TaskID)
	require.Equal(t, expected.RequestID, actual.RequestID)
	require.Equal(t, expected.ResponseID, actual.ResponseID)
	require.Equal(t, expected.UserID, actual.UserID)
	require.Equal(t, expected.APIKeyID, actual.APIKeyID)
	require.Equal(t, expected.GroupID, actual.GroupID)
	require.Equal(t, expected.AccountID, actual.AccountID)
	require.Equal(t, expected.CaptureRuleID, actual.CaptureRuleID)
	require.Equal(t, expected.Protocol, actual.Protocol)
	require.Equal(t, expected.Model, actual.Model)
	require.Equal(t, expected.RequestedModel, actual.RequestedModel)
	require.Equal(t, expected.UpstreamModel, actual.UpstreamModel)
	require.Equal(t, expected.RequestContentType, actual.RequestContentType)
	require.Equal(t, expected.ResponseContentType, actual.ResponseContentType)
	require.Equal(t, expected.InputTokens, actual.InputTokens)
	require.Equal(t, expected.OutputTokens, actual.OutputTokens)
	require.Equal(t, expected.TotalTokens, actual.TotalTokens)
	require.Equal(t, expected.UpstreamStatusCode, actual.UpstreamStatusCode)
	require.Equal(t, expected.Scaffold, actual.Scaffold)
	require.Equal(t, expected.ScaffoldVersion, actual.ScaffoldVersion)
	require.Equal(t, expected.Prompt, actual.Prompt)
	require.Equal(t, expected.Candidates, actual.Candidates)
	require.Equal(t, expected.Tools, actual.Tools)
	require.Equal(t, expected.Signature, actual.Signature)
	require.Equal(t, expected.Meta, actual.Meta)
	require.Equal(t, expected.RawRequest, actual.RawRequest)
	require.Equal(t, expected.RawResponse, actual.RawResponse)
	require.Equal(t, expected.RawRequestText, actual.RawRequestText)
	require.Equal(t, expected.RawResponseText, actual.RawResponseText)
	require.Equal(t, expected.DedupeHash, actual.DedupeHash)
	require.Equal(t, expected.PromptHash, actual.PromptHash)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
}
