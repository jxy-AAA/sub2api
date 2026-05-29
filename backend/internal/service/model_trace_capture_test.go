package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelTraceCaptureExportDerivesPromptRolesAndExcludesRawByDefault(t *testing.T) {
	t.Parallel()

	requestID := "req-001"
	captureRuleID := int64(9)
	inputTokens := int64(123)
	outputTokens := int64(45)
	totalTokens := int64(168)
	upstreamStatusCode := 200
	capture := &ModelTraceCapture{
		TaskID:              "task_001",
		RequestID:           &requestID,
		MainSessionID:       "session-001",
		CaptureRuleID:       &captureRuleID,
		Protocol:            "openai.responses",
		Model:               "gpt-4.1",
		RequestContentType:  "application/json",
		ResponseContentType: "text/event-stream",
		InputTokens:         &inputTokens,
		OutputTokens:        &outputTokens,
		TotalTokens:         &totalTokens,
		UpstreamStatusCode:  &upstreamStatusCode,
		Scaffold:            "claude-code",
		ScaffoldVersion:     "2.1.140",
		Prompt: json.RawMessage(`[
  {"role":"system","content":"system text"},
  {"role":"user","content":"first user"},
  {"role":"tool","content":{"tool_call_id":"call_1","output":"ok"}},
  {"role":"assistant","content":"first assistant"},
  {"role":"user","content":"second user"}
]`),
		Candidates:      json.RawMessage(`[{"index":0,"message":{"role":"assistant","content":"final"}}]`),
		Tools:           json.RawMessage(`[{"type":"function","function":{"name":"search"}}]`),
		Signature:       json.RawMessage(`[{"turn":1,"signature":"sig-1"}]`),
		Meta:            json.RawMessage(`{"time":"2026-05-27T00:00:00Z","model":"gpt-4.1"}`),
		RawRequest:      json.RawMessage(`{"authorization":"Bearer secret"}`),
		RawResponse:     json.RawMessage(`{"body":"secret"}`),
		RawRequestText:  "raw request body",
		RawResponseText: "raw response body",
	}

	require.NoError(t, capture.Validate())

	export := capture.Export(ModelTraceCaptureExportOptions{})
	require.NotNil(t, export)
	require.Equal(t, "task_001", export.TaskID)
	require.NotNil(t, export.RequestID)
	require.Equal(t, requestID, *export.RequestID)
	require.Equal(t, "session-001", export.MainSessionID)
	require.NotNil(t, export.CaptureRuleID)
	require.Equal(t, captureRuleID, *export.CaptureRuleID)
	require.Equal(t, "openai.responses", export.Protocol)
	require.Equal(t, "gpt-4.1", export.Model)
	require.Equal(t, "application/json", export.RequestContentType)
	require.Equal(t, "text/event-stream", export.ResponseContentType)
	require.Equal(t, inputTokens, *export.InputTokens)
	require.Equal(t, outputTokens, *export.OutputTokens)
	require.Equal(t, totalTokens, *export.TotalTokens)
	require.Equal(t, upstreamStatusCode, *export.UpstreamStatusCode)
	require.Equal(t, "claude-code", export.Scaffold)
	require.Equal(t, "2.1.140", export.ScaffoldVersion)
	require.Equal(t, `{"role":"system","content":"system text"}`, string(export.System))
	require.Equal(t, `[{"role":"user","content":"first user"},{"role":"user","content":"second user"}]`, string(export.User))
	require.Equal(t, `[{"role":"tool","content":{"tool_call_id":"call_1","output":"ok"}}]`, string(export.Tool))
	require.Equal(t, `[{"role":"assistant","content":"first assistant"}]`, string(export.Assistant))
	require.JSONEq(t, `[{"role":"system","content":"system text"},{"role":"user","content":"first user"},{"role":"tool","content":{"tool_call_id":"call_1","output":"ok"}},{"role":"assistant","content":"first assistant"},{"role":"user","content":"second user"}]`, string(export.Prompt))
	require.Equal(t, `[{"index":0,"message":{"role":"assistant","content":"final"}}]`, string(export.Candidates))
	require.Equal(t, `[{"type":"function","function":{"name":"search"}}]`, string(export.Tools))
	require.Equal(t, `[{"turn":1,"signature":"sig-1"}]`, string(export.Signature))
	require.JSONEq(t, `{"time":"2026-05-27T00:00:00Z","model":"gpt-4.1"}`, string(export.Meta))
	require.Empty(t, export.RawRequest)
	require.Empty(t, export.RawResponse)
	require.Empty(t, export.RawRequestText)
	require.Empty(t, export.RawResponseText)

	exportWithRaw := capture.Export(ModelTraceCaptureExportOptions{IncludeRaw: true})
	require.Equal(t, `{"authorization":"Bearer secret"}`, string(exportWithRaw.RawRequest))
	require.Equal(t, `{"body":"secret"}`, string(exportWithRaw.RawResponse))
	require.Equal(t, "raw request body", exportWithRaw.RawRequestText)
	require.Equal(t, "raw response body", exportWithRaw.RawResponseText)
}

func TestModelTraceCaptureValidateComputesScopedMainSessionKey(t *testing.T) {
	t.Parallel()

	userID := int64(1)
	apiKeyID := int64(2)
	groupID := int64(3)
	first := &ModelTraceCapture{
		TaskID:          "task_001",
		MainSessionID:   "session-001",
		UserID:          &userID,
		APIKeyID:        &apiKeyID,
		GroupID:         &groupID,
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt:          json.RawMessage(`[{"role":"user","content":"hello"}]`),
	}
	secondUserID := int64(99)
	second := &ModelTraceCapture{
		TaskID:          "task_002",
		MainSessionID:   "session-001",
		UserID:          &secondUserID,
		APIKeyID:        &apiKeyID,
		GroupID:         &groupID,
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt:          json.RawMessage(`[{"role":"user","content":"hello"}]`),
	}

	require.NoError(t, first.Validate())
	require.NoError(t, second.Validate())
	require.Len(t, first.MainSessionKey, 64)
	require.Len(t, second.MainSessionKey, 64)
	require.NotEqual(t, first.MainSessionKey, second.MainSessionKey)
}

func TestModelTraceCaptureValidateDefaultsOptionalJSONFields(t *testing.T) {
	t.Parallel()

	capture := &ModelTraceCapture{
		TaskID:          "task_001",
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt:          json.RawMessage(`[{"role":"system","content":"system text"}]`),
	}

	require.NoError(t, capture.Validate())
	require.Equal(t, `[]`, string(capture.Candidates))
	require.Equal(t, `[]`, string(capture.Tools))
	require.Equal(t, `[]`, string(capture.Signature))
	require.Equal(t, `{}`, string(capture.Meta))
	require.Len(t, capture.PromptHash, 64)
	require.Len(t, capture.DedupeHash, 64)
}

func TestModelTraceCaptureValidateCanonicalizesCoreFields(t *testing.T) {
	t.Parallel()

	capture := &ModelTraceCapture{
		TaskID:              "  task_001  ",
		Protocol:            "  openai.responses  ",
		Model:               "  gpt-4.1  ",
		RequestContentType:  "  application/json  ",
		ResponseContentType: "  text/event-stream  ",
		Scaffold:            "  claude-code  ",
		ScaffoldVersion:     "  2.1.140  ",
		Prompt:              json.RawMessage(`[{"role":"system","content":"system text"}]`),
	}

	require.NoError(t, capture.Validate())
	require.Equal(t, "task_001", capture.TaskID)
	require.Equal(t, "openai.responses", capture.Protocol)
	require.Equal(t, "gpt-4.1", capture.Model)
	require.Equal(t, "application/json", capture.RequestContentType)
	require.Equal(t, "text/event-stream", capture.ResponseContentType)
	require.Equal(t, "claude-code", capture.Scaffold)
	require.Equal(t, "2.1.140", capture.ScaffoldVersion)
}

func TestModelTraceCaptureValidateRejectsInvalidTokenMetadata(t *testing.T) {
	t.Parallel()

	inputTokens := int64(-1)
	upstreamStatusCode := 99
	capture := &ModelTraceCapture{
		TaskID:              "task_002",
		Protocol:            "openai.responses",
		Model:               "gpt-4.1",
		RequestContentType:  " application/json ",
		ResponseContentType: " text/event-stream ",
		InputTokens:         &inputTokens,
		UpstreamStatusCode:  &upstreamStatusCode,
		Scaffold:            "claude-code",
		ScaffoldVersion:     "2.1.140",
		Prompt:              json.RawMessage(`[{"role":"system","content":"system text"}]`),
	}

	err := capture.Validate()
	require.EqualError(t, err, "input_tokens must be >= 0")

	inputTokens = 1
	err = capture.Validate()
	require.EqualError(t, err, "upstream_status_code must be between 100 and 999")
}

func TestModelTraceCaptureExportSystemUsesRoleBasedSelection(t *testing.T) {
	t.Parallel()

	capture := &ModelTraceCapture{
		TaskID:          "task_004",
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt: json.RawMessage(`[
  {"role":"user","content":"first user"},
  {"role":"assistant","content":"first assistant"},
  {"role":"system","content":"actual system"}
]`),
	}

	require.NoError(t, capture.Validate())
	export := capture.Export(ModelTraceCaptureExportOptions{})
	require.Equal(t, `{"role":"system","content":"actual system"}`, string(export.System))
	require.Equal(t, `[{"role":"user","content":"first user"}]`, string(export.User))
	require.Equal(t, `[{"role":"assistant","content":"first assistant"}]`, string(export.Assistant))
}

func TestModelTraceCaptureExportSystemAbsentWhenNoSystemRoleExists(t *testing.T) {
	t.Parallel()

	capture := &ModelTraceCapture{
		TaskID:          "task_005",
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt: json.RawMessage(`[
  {"role":"user","content":"first user"},
  {"role":"tool","content":{"tool_call_id":"call_1","output":"ok"}}
]`),
	}

	require.NoError(t, capture.Validate())
	export := capture.Export(ModelTraceCaptureExportOptions{})
	require.Empty(t, export.System)
}

func TestModelTraceCaptureHashesIgnoreJSONFormatting(t *testing.T) {
	t.Parallel()

	first := &ModelTraceCapture{
		TaskID:          "task_002",
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt:          json.RawMessage(`[{"role":"system","content":{"b":2,"a":1}},{"role":"user","content":"x"}]`),
		Candidates:      json.RawMessage(`[{"message":{"content":"done","role":"assistant"},"index":0}]`),
		Tools:           json.RawMessage(`[{"function":{"parameters":{"type":"object","properties":{"q":{"type":"string"}}},"name":"search"},"type":"function"}]`),
		Meta:            json.RawMessage(`{"x":1}`),
	}

	second := &ModelTraceCapture{
		TaskID:          "task_003",
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "claude-code",
		ScaffoldVersion: "2.1.140",
		Prompt:          json.RawMessage("[\n  {\"content\":{\"a\":1,\"b\":2},\"role\":\"system\"},\n  {\"content\":\"x\",\"role\":\"user\"}\n]"),
		Candidates:      json.RawMessage("[\n  {\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"done\"}}\n]"),
		Tools:           json.RawMessage("[\n  {\"type\":\"function\",\"function\":{\"name\":\"search\",\"parameters\":{\"properties\":{\"q\":{\"type\":\"string\"}},\"type\":\"object\"}}}\n]"),
		Meta:            json.RawMessage(`{"x":1}`),
	}

	require.NoError(t, first.Validate())
	require.NoError(t, second.Validate())
	require.Equal(t, first.PromptHash, second.PromptHash)
	require.Equal(t, first.DedupeHash, second.DedupeHash)
}
