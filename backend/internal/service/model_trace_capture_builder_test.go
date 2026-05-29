package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelTraceCaptureFromGatewayTraceDerivesSSEMetadataAndPreservesRawBodies(t *testing.T) {
	t.Parallel()

	userID := int64(42)
	apiKeyID := int64(7)
	groupID := int64(5)
	entries := []GatewayTraceCaptureEntry{
		{
			Stage:       GatewayTraceStageClientRequest,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			Body:        `{"model":"gpt-4.1","metadata":{"user_id":"user_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_account__session_123e4567-e89b-12d3-a456-426614174000"},"messages":[{"role":"user","content":"incident please inspect"}]}`,
			Meta: map[string]any{
				"account_id": int64(99),
			},
		},
		{
			Stage:       GatewayTraceStageUpstreamRequest,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			Body:        `{"model":"gpt-4.1-mini","input":[{"role":"user","content":"incident please inspect"}]}`,
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
	input := GatewayTraceRecordInput{
		UserID:    &userID,
		APIKeyID:  &apiKeyID,
		GroupID:   &groupID,
		RequestID: "req_123",
		Method:    "POST",
		Path:      "/v1/responses",
	}

	trace, err := BuildCodexTraceExportFromGatewayCaptures(entries, input)
	require.NoError(t, err)

	capture, err := modelTraceCaptureFromGatewayTrace(trace, entries, input)
	require.NoError(t, err)
	require.NotNil(t, capture)
	require.NotNil(t, capture.RequestID)
	require.Equal(t, "req_123", *capture.RequestID)
	require.NotNil(t, capture.ResponseID)
	require.Equal(t, "resp_123", *capture.ResponseID)
	require.Equal(t, "123e4567-e89b-12d3-a456-426614174000", capture.MainSessionID)
	require.Len(t, capture.MainSessionKey, 64)
	require.NotNil(t, capture.AccountID)
	require.Equal(t, int64(99), *capture.AccountID)
	require.Equal(t, "gpt-4.1-mini", capture.Model)
	require.NotNil(t, capture.RequestedModel)
	require.Equal(t, "gpt-4.1", *capture.RequestedModel)
	require.NotNil(t, capture.UpstreamModel)
	require.Equal(t, "gpt-4.1-mini", *capture.UpstreamModel)
	require.NotNil(t, capture.InputTokens)
	require.Equal(t, int64(120), *capture.InputTokens)
	require.NotNil(t, capture.OutputTokens)
	require.Equal(t, int64(80), *capture.OutputTokens)
	require.NotNil(t, capture.TotalTokens)
	require.Equal(t, int64(200), *capture.TotalTokens)
	require.JSONEq(t, entries[0].Body, string(capture.RawRequest))
	require.Equal(t, entries[0].Body, capture.RawRequestText)
	require.Nil(t, capture.RawResponse)
	require.Equal(t, entries[2].Body, capture.RawResponseText)
	exportWithRaw := capture.Export(ModelTraceCaptureExportOptions{IncludeRaw: true})
	require.JSONEq(t, entries[1].Body, string(exportWithRaw.RawUpstreamRequest))
	require.Equal(t, entries[1].Body, exportWithRaw.RawUpstreamRequestText)

	export := codexTraceExportFromModelTraceCapture(capture)
	require.Equal(t, capture.TaskID, export.TaskID)
	require.JSONEq(t, string(capture.Prompt), string(export.Prompt))
	require.JSONEq(t, string(capture.Candidates), string(export.Candidates))
	var scaffold map[string]any
	require.NoError(t, json.Unmarshal(export.Scaffold, &scaffold))
	require.Equal(t, "sub2api_gateway_capture", scaffold["source"])
}

func TestModelTraceCaptureFromGatewayTraceUsesHeaderMainSessionFallback(t *testing.T) {
	t.Parallel()

	userID := int64(42)
	entries := []GatewayTraceCaptureEntry{
		{
			Stage:       GatewayTraceStageClientRequest,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			Body:        `{"model":"gpt-4.1","input":"hello"}`,
		},
		{
			Stage:       GatewayTraceStageUpstreamResponse,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			StatusCode:  200,
			Body:        `{"id":"resp_456","model":"gpt-4.1","output":"ok"}`,
		},
	}
	input := GatewayTraceRecordInput{
		UserID:        &userID,
		MainSessionID: "header-session-001",
		RequestID:     "req_456",
	}

	trace, err := BuildCodexTraceExportFromGatewayCaptures(entries, input)
	require.NoError(t, err)
	capture, err := modelTraceCaptureFromGatewayTrace(trace, entries, input)
	require.NoError(t, err)
	require.Equal(t, "header-session-001", capture.MainSessionID)
	require.Len(t, capture.MainSessionKey, 64)
}
