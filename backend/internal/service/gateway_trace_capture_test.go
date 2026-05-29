//go:build unit

package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCaptureGatewayUpstreamResponseBuffer_PreservesOpenAISSEToolCalls(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	traceBuf := newGatewayTraceBodyBuffer()
	traceBuf.WriteLine(`data: {"type":"response.output_item.added","item":{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{\"q\":\"weather\"}"}}`)
	traceBuf.WriteLine("")
	traceBuf.WriteLine(`data: {"type":"response.completed","response":{"id":"resp_1","output":[{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{\"q\":\"weather\"}"}]}}`)
	traceBuf.WriteLine("")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"rid_openai_trace"},
		},
	}

	captureGatewayUpstreamResponseBuffer(c, inferGatewayTraceProtocol(c, "openai"), resp, traceBuf, nil)

	captures := GetGatewayTraceCaptures(c)
	require.Len(t, captures, 1)
	require.Equal(t, GatewayTraceStageUpstreamResponse, captures[0].Stage)
	require.Equal(t, "openai.responses", captures[0].Protocol)
	require.Equal(t, strings.Join([]string{
		`data: {"type":"response.output_item.added","item":{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{\"q\":\"weather\"}"}}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_1","output":[{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{\"q\":\"weather\"}"}]}}`,
		``,
	}, "\n")+"\n", captures[0].Body)
	require.Equal(t, "sse", captures[0].Meta["raw_transport"])
	require.Equal(t, 2, captures[0].Meta["raw_event_count"])
	require.Equal(t, true, captures[0].Meta["raw_stream_terminated"])
	require.Equal(t, "rid_openai_trace", captures[0].Meta["upstream_request_id"])
}

func TestInferGatewayTraceProtocolFromPath_GeminiNative(t *testing.T) {
	t.Parallel()

	require.Equal(t, "gemini.stream_generate_content", inferGatewayTraceProtocolFromPath("/v1beta/models/gemini-2.5-pro:streamGenerateContent", "gemini"))
	require.Equal(t, "gemini.generate_content", inferGatewayTraceProtocolFromPath("/v1beta/models/gemini-2.5-pro:generateContent", "gemini"))
	require.Equal(t, "gemini.count_tokens", inferGatewayTraceProtocolFromPath("/v1beta/models/gemini-2.5-pro:countTokens", "gemini"))
}

func TestCaptureGatewayUpstreamResponseBuffer_CountsLongSSEDataLines(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	longData := "data: " + strings.Repeat("x", 70*1024)
	traceBuf := newGatewayTraceBodyBuffer()
	traceBuf.WriteLine(`event: giant`)
	traceBuf.WriteLine(longData)
	traceBuf.WriteLine("")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
	}

	captureGatewayUpstreamResponseBuffer(c, inferGatewayTraceProtocol(c, "openai"), resp, traceBuf, nil)

	captures := GetGatewayTraceCaptures(c)
	require.Len(t, captures, 1)
	require.Contains(t, captures[0].Body, longData)
	require.Equal(t, "sse", captures[0].Meta["raw_transport"])
	require.Equal(t, 1, captures[0].Meta["raw_event_count"])
	require.Equal(t, true, captures[0].Meta["raw_stream_terminated"])
}
