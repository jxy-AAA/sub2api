//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayService_RoutingAccountIDsForRequestSupportsAnthropicCompatibleGroup(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	svc := &GatewayService{
		groupRepo: &mockGroupRepoForGateway{
			groups: map[int64]*Group{
				groupID: {
					ID:                  groupID,
					Platform:            PlatformAnthropicCompatible,
					ModelRoutingEnabled: true,
					ModelRouting: map[string][]int64{
						"claude-*": {11, 12},
					},
				},
			},
		},
	}

	ids := svc.routingAccountIDsForRequest(context.Background(), &groupID, "claude-sonnet-4-5", PlatformAnthropic)
	require.Equal(t, []int64{11, 12}, ids)
}

func TestGatewayService_IsModelSupportedByAccountAllowsOpenAICompatiblePassthrough(t *testing.T) {
	t.Parallel()

	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformOpenAICompatible,
		Type:     AccountTypeUpstream,
	}

	require.True(t, svc.isModelSupportedByAccount(account, "deepseek-chat"))
}

func TestGatewayService_IsModelSupportedByAccountHonorsOpenAICompatibleMapping(t *testing.T) {
	t.Parallel()

	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformOpenAICompatible,
		Type:     AccountTypeUpstream,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"deepseek-chat": "deepseek-chat",
			},
		},
	}

	require.True(t, svc.isModelSupportedByAccount(account, "deepseek-chat"))
	require.False(t, svc.isModelSupportedByAccount(account, "gpt-4o"))
}

type recordingDoHTTPUpstream struct {
	response *http.Response
	request  *http.Request
	body     []byte
}

func (u *recordingDoHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.request = req
	if req != nil && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		u.body = body
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if u.response == nil {
		return nil, fmt.Errorf("missing mocked response")
	}
	return u.response, nil
}

func (u *recordingDoHTTPUpstream) DoWithTLS(_ *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected DoWithTLS call")
}

func TestOpenAICompatiblePassthroughAppliesAccountModelMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := newJSONResponse(http.StatusOK, `{"id":"resp_1","model":"provider-model","usage":{"input_tokens":1,"output_tokens":2}}`)
	upstream := &recordingDoHTTPUpstream{response: resp}
	svc := &OpenAIGatewayService{
		cfg:          testConfig(),
		httpUpstream: upstream,
	}
	account := &Account{
		ID:       7,
		Name:     "compat",
		Platform: PlatformOpenAICompatible,
		Type:     AccountTypeUpstream,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://compat.example",
			"model_mapping": map[string]any{
				"gpt-alias": "provider-model",
			},
		},
	}

	result, err := svc.forwardOpenAIPassthrough(
		context.Background(),
		c,
		account,
		[]byte(`{"model":"gpt-alias","input":"hi","stream":false}`),
		"gpt-alias",
		nil,
		false,
		time.Now(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "gpt-alias", result.Model)
	require.Equal(t, "provider-model", result.UpstreamModel)
	require.NotNil(t, upstream.request)
	require.Equal(t, "https://compat.example/v1/responses", upstream.request.URL.String())

	var upstreamBody map[string]any
	require.NoError(t, json.Unmarshal(upstream.body, &upstreamBody))
	require.Equal(t, "provider-model", upstreamBody["model"])
	require.Contains(t, rec.Body.String(), `"model":"gpt-alias"`)
}

func TestOpenAIGatewayService_ListSchedulableAccountsIncludesCompatibleProviders(t *testing.T) {
	t.Parallel()

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
			{ID: 2, Platform: PlatformOpenAICompatible, Status: StatusActive, Schedulable: true},
			{ID: 3, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true},
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo: repo,
		cfg:         testConfig(),
	}

	accounts, err := svc.listSchedulableAccounts(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.ElementsMatch(t, []string{PlatformOpenAI, PlatformOpenAICompatible}, []string{accounts[0].Platform, accounts[1].Platform})
}
