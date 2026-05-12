package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunContentModeration_ReturnsBlockingDecisionOnFailClosedError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	moderationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream unavailable"}}`))
	}))
	defer moderationServer.Close()

	cfg := &service.ContentModerationConfig{
		Enabled:      true,
		Mode:         service.ContentModerationModePreBlock,
		BaseURL:      moderationServer.URL,
		Model:        "omni-moderation-latest",
		APIKeys:      []string{"sk-test"},
		RetryCount:   0,
		SampleRate:   100,
		AllGroups:    true,
		BlockStatus:  http.StatusServiceUnavailable,
		BlockMessage: "内容审核服务暂时不可用，请稍后重试",
	}
	rawCfg, err := json.Marshal(cfg)
	require.NoError(t, err)

	repo := &contentModerationHandlerTestRepo{}
	settingRepo := &contentModerationHandlerSettingRepo{values: map[string]string{
		service.SettingKeyRiskControlEnabled:      "true",
		service.SettingKeyContentModerationConfig: string(rawCfg),
	}}
	moderationSvc := service.NewContentModerationService(
		settingRepo,
		repo,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	body := []byte(`{"messages":[{"role":"user","content":"please check this"}]}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	decision := runContentModeration(
		c,
		zap.NewNop(),
		moderationSvc,
		nil,
		middleware.AuthSubject{UserID: 1001},
		service.ContentModerationProtocolOpenAIChat,
		"gpt-5.5",
		body,
	)

	require.NotNil(t, decision)
	require.False(t, decision.Allowed)
	require.True(t, decision.Blocked)
	require.Equal(t, service.ContentModerationActionError, decision.Action)
	require.Equal(t, http.StatusServiceUnavailable, decision.StatusCode)
	require.Equal(t, "内容审核服务暂时不可用，请稍后重试", decision.Message)
	require.Len(t, repo.logs, 1)
	require.Equal(t, service.ContentModerationActionError, repo.logs[0].Action)
}
