package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const gatewayTraceRecordTimeout = 5 * time.Second

func GatewayTraceRecorder(traceService *service.ModelInteractionTraceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if traceService == nil || c == nil || c.Request == nil {
			return
		}
		entries := service.GetGatewayTraceCaptures(c)
		if len(entries) == 0 {
			return
		}

		input := service.GatewayTraceRecordInput{
			Method: strings.TrimSpace(c.Request.Method),
		}
		if c.Request.URL != nil {
			input.Path = strings.TrimSpace(c.Request.URL.Path)
		}
		if requestID, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
			input.RequestID = strings.TrimSpace(requestID)
		}
		if subject, ok := GetAuthSubjectFromContext(c); ok {
			if userID, human := subject.HumanUserID(); human {
				input.UserID = &userID
			}
		}
		if apiKey, ok := GetAPIKeyFromContext(c); ok && apiKey != nil {
			apiKeyID := apiKey.ID
			input.APIKeyID = &apiKeyID
			if input.UserID == nil && apiKey.UserID > 0 {
				userID := apiKey.UserID
				input.UserID = &userID
			}
			if apiKey.GroupID != nil {
				groupID := *apiKey.GroupID
				input.GroupID = &groupID
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), gatewayTraceRecordTimeout)
		defer cancel()
		if _, err := traceService.RecordGatewayTraceCaptures(ctx, entries, input); err != nil {
			logger.FromContext(c.Request.Context()).Warn("gateway trace capture was not persisted",
				zap.String("request_id", input.RequestID),
				zap.Error(err),
			)
		}
	}
}
