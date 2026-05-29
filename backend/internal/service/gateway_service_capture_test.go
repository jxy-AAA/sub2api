package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayService_ForwardCapturesUpstreamRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"claude-3-haiku-20240307","max_tokens":32,"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid_gateway_capture"},
			},
			Body: io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-3-haiku-20240307","stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":2}}`)),
		},
	}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
		deferredService:      &DeferredService{},
	}
	account := &Account{
		ID:          301,
		Name:        "anthropic-key",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-anthropic-key"},
		Status:      StatusActive,
		Schedulable: true,
	}
	parsed := &ParsedRequest{
		Body:   body,
		Model:  "claude-3-haiku-20240307",
		Stream: false,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)

	capture := findGatewayTraceCapture(GetGatewayTraceCaptures(c), GatewayTraceStageUpstreamRequest)
	require.NotNil(t, capture)
	require.Equal(t, "anthropic.messages", capture.Protocol)
	require.Equal(t, string(upstream.lastBody), capture.Body)
	require.Equal(t, "claude-3-haiku-20240307", capture.Meta["original_model"])
	require.Equal(t, "claude-3-haiku-20240307", capture.Meta["upstream_model"])
	require.Equal(t, 1, capture.Meta["attempt"])
}

func TestGatewayService_ForwardCountTokensCapturesDedicatedUpstreamRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"claude-3-haiku-20240307","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid_gateway_count_tokens"},
			},
			Body: io.NopCloser(strings.NewReader(`{"input_tokens":12}`)),
		},
	}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
	}
	account := &Account{
		ID:          302,
		Name:        "anthropic-key",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-anthropic-key"},
		Status:      StatusActive,
		Schedulable: true,
	}
	parsed := &ParsedRequest{
		Body:  body,
		Model: "claude-3-haiku-20240307",
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)

	capture := findGatewayTraceCapture(GetGatewayTraceCaptures(c), GatewayTraceStageUpstreamRequest)
	require.NotNil(t, capture)
	require.Equal(t, "anthropic.count_tokens", capture.Protocol)
	require.Equal(t, string(upstream.lastBody), capture.Body)
	require.Equal(t, true, capture.Meta["count_tokens"])
	require.Equal(t, "claude-3-haiku-20240307", capture.Meta["original_model"])
	require.Equal(t, "claude-3-haiku-20240307", capture.Meta["upstream_model"])
}

func TestGatewayService_ForwardCountTokensAntigravityLocal404DoesNotCaptureUpstreamRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1/messages/count_tokens", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	svc := &GatewayService{}
	account := &Account{
		ID:       303,
		Name:     "antigravity-oauth",
		Platform: PlatformAntigravity,
		Type:     AccountTypeOAuth,
	}
	parsed := &ParsedRequest{
		Body:  body,
		Model: "claude-sonnet-4-5",
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, GetGatewayTraceCaptures(c))
}

func TestGatewayService_ForwardCountTokensBedrockLocal404DoesNotCaptureUpstreamRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"anthropic.claude-3-5-sonnet-20241022-v2:0","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	svc := &GatewayService{}
	account := &Account{
		ID:       304,
		Name:     "bedrock-key",
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
	}
	parsed := &ParsedRequest{
		Body:  body,
		Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, GetGatewayTraceCaptures(c))
}
