package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type gatewayRouteChains struct {
	anthropic            []gin.HandlerFunc
	google               []gin.HandlerFunc
	antigravityAnthropic []gin.HandlerFunc
	antigravityGoogle    []gin.HandlerFunc
}

// RegisterGatewayRoutes 娉ㄥ唽 API 缃戝叧璺敱锛圕laude/OpenAI/Gemini 鍏煎锛?
func RegisterGatewayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	traceService *service.ModelInteractionTraceService,
	cfg *config.Config,
) {
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()
	chains := buildGatewayRouteChains(
		bodyLimit,
		clientRequestID,
		opsErrorLogger,
		endpointNorm,
		apiKeyAuth,
		apiKeyService,
		subscriptionService,
		settingService,
		traceService,
		cfg,
	)

	messagesHandler := matchGroupPlatformHandler(service.PlatformOpenAI, h.OpenAIGateway.Messages, h.Gateway.Messages)
	countTokensHandler := matchGroupPlatformHandler(
		service.PlatformOpenAI,
		anthropicNotFoundHandler("Token counting is not supported for this platform"),
		h.Gateway.CountTokens,
	)
	responsesHandler := matchGroupPlatformHandler(service.PlatformOpenAI, h.OpenAIGateway.Responses, h.Gateway.Responses)
	chatCompletionsHandler := matchGroupPlatformHandler(service.PlatformOpenAI, h.OpenAIGateway.ChatCompletions, h.Gateway.ChatCompletions)
	imagesHandler := matchGroupPlatformHandler(
		service.PlatformOpenAI,
		h.OpenAIGateway.Images,
		openAINotFoundHandler("Images API is not supported for this platform"),
	)

	gateway := r.Group("/v1", chains.anthropic...)
	{
		gateway.POST("/messages", messagesHandler)
		gateway.POST("/messages/count_tokens", countTokensHandler)
		// Models are served locally today; keep the recorder on the route chain,
		// but avoid synthesizing empty upstream_request captures in the handler.
		gateway.GET("/models", h.Gateway.Models)
		gateway.GET("/usage", h.Gateway.Usage)
		gateway.POST("/responses", responsesHandler)
		gateway.POST("/responses/*subpath", responsesHandler)
		// GET /responses upgrades to websocket. HTTP trace persistence only
		// happens once the websocket forwarder emits capture entries itself.
		gateway.GET("/responses", h.OpenAIGateway.ResponsesWebSocket)
		gateway.POST("/chat/completions", chatCompletionsHandler)
		// Images use dedicated handlers with multipart/raw request shapes; route
		// registration stays here, while capture is owned by the image service.
		gateway.POST("/images/generations", imagesHandler)
		gateway.POST("/images/edits", imagesHandler)
	}

	gemini := r.Group("/v1beta", chains.google...)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	r.POST("/responses", appendHandlers(chains.anthropic, responsesHandler)...)
	r.POST("/responses/*subpath", appendHandlers(chains.anthropic, responsesHandler)...)
	r.GET("/responses", appendHandlers(chains.anthropic, h.OpenAIGateway.ResponsesWebSocket)...)

	codexDirect := r.Group("/backend-api/codex", chains.anthropic...)
	{
		codexDirect.POST("/responses", responsesHandler)
		codexDirect.POST("/responses/*subpath", responsesHandler)
		codexDirect.GET("/responses", h.OpenAIGateway.ResponsesWebSocket)
	}

	r.POST("/chat/completions", appendHandlers(chains.anthropic, chatCompletionsHandler)...)
	r.POST("/images/generations", appendHandlers(chains.anthropic, imagesHandler)...)
	r.POST("/images/edits", appendHandlers(chains.anthropic, imagesHandler)...)

	r.GET("/antigravity/models", appendHandlers(chains.antigravityAnthropic, h.Gateway.AntigravityModels)...)

	antigravityV1 := r.Group("/antigravity/v1", chains.antigravityAnthropic...)
	{
		antigravityV1.POST("/messages", h.Gateway.Messages)
		antigravityV1.POST("/messages/count_tokens", h.Gateway.CountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta", chains.antigravityGoogle...)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}
}

func buildGatewayRouteChains(
	bodyLimit gin.HandlerFunc,
	clientRequestID gin.HandlerFunc,
	opsErrorLogger gin.HandlerFunc,
	endpointNorm gin.HandlerFunc,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	settingService *service.SettingService,
	traceService *service.ModelInteractionTraceService,
	cfg *config.Config,
) gatewayRouteChains {
	base := []gin.HandlerFunc{
		bodyLimit,
		clientRequestID,
		opsErrorLogger,
		endpointNorm,
		middleware.GatewayTraceRecorder(traceService),
	}
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)

	return gatewayRouteChains{
		anthropic: appendHandlers(base, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic),
		google: appendHandlers(
			base,
			middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg),
			requireGroupGoogle,
		),
		antigravityAnthropic: appendHandlers(
			base,
			middleware.ForcePlatform(service.PlatformAntigravity),
			gin.HandlerFunc(apiKeyAuth),
			requireGroupAnthropic,
		),
		antigravityGoogle: appendHandlers(
			base,
			middleware.ForcePlatform(service.PlatformAntigravity),
			middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg),
			requireGroupGoogle,
		),
	}
}

func appendHandlers(handlers []gin.HandlerFunc, extra ...gin.HandlerFunc) []gin.HandlerFunc {
	combined := append([]gin.HandlerFunc{}, handlers...)
	return append(combined, extra...)
}

func matchGroupPlatformHandler(platform string, matched gin.HandlerFunc, fallback gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupPlatform := getGroupPlatform(c)
		if (platform == service.PlatformOpenAI && service.IsOpenAIProtocolPlatform(groupPlatform)) ||
			(platform == service.PlatformAnthropic && service.IsAnthropicProtocolPlatform(groupPlatform)) ||
			groupPlatform == platform {
			matched(c)
			return
		}
		fallback(c)
	}
}

func anthropicNotFoundHandler(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "not_found_error",
				"message": message,
			},
		})
	}
}

func openAINotFoundHandler(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": message,
			},
		})
	}
}

// getGroupPlatform extracts the group platform from the API Key stored in context.
func getGroupPlatform(c *gin.Context) string {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}
