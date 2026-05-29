package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type interactionTraceRepoStub struct {
	created   []*ModelInteractionTrace
	createErr error
}

func (s *interactionTraceRepoStub) Create(ctx context.Context, trace *ModelInteractionTrace) (bool, error) {
	if s.createErr != nil {
		return false, s.createErr
	}
	if trace == nil {
		return false, nil
	}
	cloned := *trace
	cloned.Prompt = cloneRawJSON(trace.Prompt)
	cloned.Candidates = cloneRawJSON(trace.Candidates)
	cloned.Tools = cloneRawJSON(trace.Tools)
	cloned.Signature = cloneRawJSON(trace.Signature)
	cloned.Meta = cloneRawJSON(trace.Meta)
	cloned.Scaffold = cloneRawJSON(trace.Scaffold)
	s.created = append(s.created, &cloned)
	return true, nil
}

func (s *interactionTraceRepoStub) ListAll(ctx context.Context) ([]*ModelInteractionTrace, error) {
	out := make([]*ModelInteractionTrace, 0, len(s.created))
	for _, item := range s.created {
		if item == nil {
			continue
		}
		cloned := *item
		cloned.Prompt = cloneRawJSON(item.Prompt)
		cloned.Candidates = cloneRawJSON(item.Candidates)
		cloned.Tools = cloneRawJSON(item.Tools)
		cloned.Signature = cloneRawJSON(item.Signature)
		cloned.Meta = cloneRawJSON(item.Meta)
		cloned.Scaffold = cloneRawJSON(item.Scaffold)
		out = append(out, &cloned)
	}
	return out, nil
}

func (s *interactionTraceRepoStub) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*ModelInteractionTrace, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

type traceCaptureRepoStub struct {
	created   []*ModelTraceCapture
	createErr error
}

func (s *traceCaptureRepoStub) Create(ctx context.Context, capture *ModelTraceCapture) (bool, error) {
	if s.createErr != nil {
		return false, s.createErr
	}
	if capture == nil {
		return false, nil
	}
	cloned := *capture
	cloned.RequestID = traceCaptureCloneStringPtr(capture.RequestID)
	cloned.ResponseID = traceCaptureCloneStringPtr(capture.ResponseID)
	cloned.UserID = traceCaptureCloneInt64Ptr(capture.UserID)
	cloned.APIKeyID = traceCaptureCloneInt64Ptr(capture.APIKeyID)
	cloned.GroupID = traceCaptureCloneInt64Ptr(capture.GroupID)
	cloned.AccountID = traceCaptureCloneInt64Ptr(capture.AccountID)
	cloned.CaptureRuleID = traceCaptureCloneInt64Ptr(capture.CaptureRuleID)
	cloned.RequestedModel = traceCaptureCloneStringPtr(capture.RequestedModel)
	cloned.UpstreamModel = traceCaptureCloneStringPtr(capture.UpstreamModel)
	cloned.InputTokens = traceCaptureCloneInt64Ptr(capture.InputTokens)
	cloned.OutputTokens = traceCaptureCloneInt64Ptr(capture.OutputTokens)
	cloned.TotalTokens = traceCaptureCloneInt64Ptr(capture.TotalTokens)
	cloned.UpstreamStatusCode = traceCaptureCloneIntPtr(capture.UpstreamStatusCode)
	cloned.Prompt = cloneRawJSON(capture.Prompt)
	cloned.Candidates = cloneRawJSON(capture.Candidates)
	cloned.Tools = cloneRawJSON(capture.Tools)
	cloned.Signature = cloneRawJSON(capture.Signature)
	cloned.Meta = cloneRawJSON(capture.Meta)
	cloned.RawRequest = cloneRawJSON(capture.RawRequest)
	cloned.RawResponse = cloneRawJSON(capture.RawResponse)
	s.created = append(s.created, &cloned)
	return true, nil
}

func (s *traceCaptureRepoStub) GetByID(ctx context.Context, id int64) (*ModelTraceCapture, error) {
	return nil, nil
}

func (s *traceCaptureRepoStub) GetByTaskID(ctx context.Context, taskID string) (*ModelTraceCapture, error) {
	return nil, nil
}

func (s *traceCaptureRepoStub) GetByMainSessionKey(ctx context.Context, mainSessionKey string) (*ModelTraceCapture, error) {
	return nil, nil
}

func (s *traceCaptureRepoStub) GetByDedupeHash(ctx context.Context, dedupeHash string) (*ModelTraceCapture, error) {
	return nil, nil
}

func (s *traceCaptureRepoStub) List(ctx context.Context, filter ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

func (s *traceCaptureRepoStub) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

func (s *traceCaptureRepoStub) DeleteByID(ctx context.Context, id int64) (bool, error) {
	return false, nil
}

func (s *traceCaptureRepoStub) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	return 0, nil
}

type traceRuleRepoStub struct {
	rules []*ModelTraceCaptureRule
	err   error
}

func (s *traceRuleRepoStub) Create(ctx context.Context, rule *ModelTraceCaptureRule) (*ModelTraceCaptureRule, error) {
	return nil, nil
}

func (s *traceRuleRepoStub) Update(ctx context.Context, rule *ModelTraceCaptureRule) (*ModelTraceCaptureRule, error) {
	return nil, nil
}

func (s *traceRuleRepoStub) GetByID(ctx context.Context, id int64) (*ModelTraceCaptureRule, error) {
	return nil, nil
}

func (s *traceRuleRepoStub) List(ctx context.Context) ([]*ModelTraceCaptureRule, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rules, nil
}

func (s *traceRuleRepoStub) DeleteByID(ctx context.Context, id int64) (bool, error) {
	return false, nil
}

func TestModelInteractionTraceServiceRecordGatewayTraceCapturesPersistsRawAndMatchedAdminCapture(t *testing.T) {
	t.Parallel()

	rawRepo := &interactionTraceRepoStub{}
	captureRepo := &traceCaptureRepoStub{}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	activeFrom := now.Add(-time.Hour)
	activeTo := now.Add(time.Hour)
	minTokens := int64(100)
	maxTokens := int64(300)
	ruleRepo := &traceRuleRepoStub{rules: []*ModelTraceCaptureRule{{
		ID:            44,
		Name:          "incident rule",
		Enabled:       true,
		Priority:      10,
		ModelPatterns: []string{"gpt-*"},
		UserIDs:       []int64{42},
		APIKeyIDs:     []int64{7},
		Keywords:      []string{"incident"},
		MinTokens:     &minTokens,
		MaxTokens:     &maxTokens,
		SamplingRatio: 1,
		ActiveFrom:    &activeFrom,
		ActiveTo:      &activeTo,
	}}}

	svc := NewModelInteractionTraceService(rawRepo, captureRepo, ruleRepo)
	svc.now = func() time.Time { return now }

	stored, err := svc.RecordGatewayTraceCaptures(context.Background(), newGatewayTraceEntries(), newGatewayTraceInput())
	require.NoError(t, err)
	require.True(t, stored)
	require.Empty(t, rawRepo.created)
	require.Len(t, captureRepo.created, 1)

	capture := captureRepo.created[0]
	require.NotNil(t, capture.CaptureRuleID)
	require.Equal(t, int64(44), *capture.CaptureRuleID)
	require.NotNil(t, capture.RequestID)
	require.Equal(t, "req_123", *capture.RequestID)
	require.NotNil(t, capture.GroupID)
	require.Equal(t, int64(5), *capture.GroupID)
	require.NotNil(t, capture.AccountID)
	require.Equal(t, int64(99), *capture.AccountID)
	require.Equal(t, "gpt-4.1-mini", capture.Model)
	require.NotNil(t, capture.TotalTokens)
	require.Equal(t, int64(200), *capture.TotalTokens)
	require.Equal(t, newGatewayTraceEntries()[0].Body, capture.RawRequestText)
	require.Equal(t, newGatewayTraceEntries()[1].Body, capture.RawResponseText)
}

func TestModelInteractionTraceServiceRecordGatewayTraceCapturesDefaultsToCaptureWhenNoRulesConfigured(t *testing.T) {
	t.Parallel()

	rawRepo := &interactionTraceRepoStub{}
	captureRepo := &traceCaptureRepoStub{}
	ruleRepo := &traceRuleRepoStub{rules: []*ModelTraceCaptureRule{}}

	svc := NewModelInteractionTraceService(rawRepo, captureRepo, ruleRepo)

	stored, err := svc.RecordGatewayTraceCaptures(context.Background(), newGatewayTraceEntries(), newGatewayTraceInput())
	require.NoError(t, err)
	require.True(t, stored)
	require.Empty(t, rawRepo.created)
	require.Len(t, captureRepo.created, 1)
}

func TestModelInteractionTraceServiceRecordGatewayTraceCapturesSkipsWhenEnabledRulesDoNotMatch(t *testing.T) {
	t.Parallel()

	rawRepo := &interactionTraceRepoStub{}
	captureRepo := &traceCaptureRepoStub{}
	ruleRepo := &traceRuleRepoStub{rules: []*ModelTraceCaptureRule{{
		ID:            2,
		Name:          "different model",
		Enabled:       true,
		ModelPatterns: []string{"claude-*"},
		SamplingRatio: 1,
	}}}

	svc := NewModelInteractionTraceService(rawRepo, captureRepo, ruleRepo)

	stored, err := svc.RecordGatewayTraceCaptures(context.Background(), newGatewayTraceEntries(), newGatewayTraceInput())
	require.NoError(t, err)
	require.False(t, stored)
	require.Empty(t, rawRepo.created)
	require.Empty(t, captureRepo.created)
}

func newGatewayTraceEntries() []GatewayTraceCaptureEntry {
	return []GatewayTraceCaptureEntry{
		{
			Stage:       GatewayTraceStageClientRequest,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			Body:        `{"model":"gpt-4.1","messages":[{"role":"user","content":"incident please inspect"}]}`,
			Meta: map[string]any{
				"account_id": int64(99),
			},
		},
		{
			Stage:       GatewayTraceStageUpstreamResponse,
			Protocol:    "openai.responses",
			ContentType: "text/event-stream",
			StatusCode:  200,
			Body: strings.Join([]string{
				`data: {"type":"response.output_text.delta","delta":"partial"}`,
				``,
				`data: {"type":"response.completed","response":{"id":"resp_123","model":"gpt-4.1-mini","usage":{"input_tokens":120,"output_tokens":80,"total_tokens":200}}}`,
				``,
			}, "\n"),
		},
	}
}

func newGatewayTraceInput() GatewayTraceRecordInput {
	userID := int64(42)
	apiKeyID := int64(7)
	groupID := int64(5)
	return GatewayTraceRecordInput{
		UserID:    &userID,
		APIKeyID:  &apiKeyID,
		GroupID:   &groupID,
		RequestID: "req_123",
		Method:    "POST",
		Path:      "/v1/responses",
	}
}

func TestModelInteractionTraceServiceExportStillUsesRawTraceRepository(t *testing.T) {
	t.Parallel()

	requestID := "req_123"
	model := "gpt-4.1"
	rawRepo := &interactionTraceRepoStub{
		created: []*ModelInteractionTrace{{
			TaskID:          "task-export-001",
			RequestID:       &requestID,
			Model:           &model,
			Prompt:          json.RawMessage(`[{"role":"user","content":"incident"}]`),
			Candidates:      json.RawMessage(`[{"message":{"role":"assistant","content":"done"}}]`),
			Tools:           json.RawMessage(`[]`),
			Signature:       json.RawMessage(`{"available":false}`),
			Meta:            json.RawMessage(`{"model":"gpt-4.1"}`),
			Scaffold:        json.RawMessage(`{"source":"sub2api_gateway_capture"}`),
			ScaffoldVersion: TaodingTraceScaffoldVersion,
		}},
	}
	svc := NewModelInteractionTraceService(rawRepo, nil, nil)

	bundle, err := svc.Export(context.Background(), time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Len(t, bundle.Records, 1)
	require.Equal(t, "task-export-001", bundle.Records[0].TaskID)
}
