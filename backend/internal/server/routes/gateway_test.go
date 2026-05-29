package routes

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func handlerName(fn gin.HandlerFunc) string {
	if fn == nil {
		return ""
	}
	pc := reflect.ValueOf(fn).Pointer()
	if pc == 0 {
		return ""
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return ""
	}
	return f.Name()
}

func chainContainsHandler(handlers []gin.HandlerFunc, want string) bool {
	for _, handler := range handlers {
		if strings.Contains(handlerName(handler), want) {
			return true
		}
	}
	return false
}

func newGatewayRoutesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group:   &service.Group{Platform: service.PlatformOpenAI},
			})
			c.Next()
		}),
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
	)

	return router
}

func TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/responses/compact",
		"/responses/compact",
		"/backend-api/codex/responses",
		"/backend-api/codex/responses/compact",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI responses handler", path)
	}
}

func TestGatewayRoutesOpenAIImagesPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/images/generations",
		"/v1/images/edits",
		"/images/generations",
		"/images/edits",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-image-2","prompt":"draw a cat"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI images handler", path)
	}
}

func TestAppendHandlersPreservesOrder(t *testing.T) {
	seen := make([]string, 0, 3)
	marker := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			seen = append(seen, name)
			c.Next()
		}
	}

	router := gin.New()
	router.POST("/test", appendHandlers(
		[]gin.HandlerFunc{marker("base-1"), marker("base-2")},
		marker("final"),
	)...)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []string{"base-1", "base-2", "final"}, seen)
}

func TestMatchGroupPlatformHandlerTreatsOpenAICompatibleAsOpenAI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	groupID := int64(1)
	c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
		GroupID: &groupID,
		Group:   &service.Group{Platform: service.PlatformOpenAICompatible},
	})

	matched := false
	fallback := false
	handler := matchGroupPlatformHandler(
		service.PlatformOpenAI,
		func(c *gin.Context) { matched = true },
		func(c *gin.Context) { fallback = true },
	)
	handler(c)

	require.True(t, matched)
	require.False(t, fallback)
}

func TestBuildGatewayRouteChainsIncludeGatewayTraceRecorder(t *testing.T) {
	chains := buildGatewayRouteChains(
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Next() },
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) { c.Next() }),
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
	)

	for name, handlers := range map[string][]gin.HandlerFunc{
		"anthropic":            chains.anthropic,
		"google":               chains.google,
		"antigravityAnthropic": chains.antigravityAnthropic,
		"antigravityGoogle":    chains.antigravityGoogle,
	} {
		require.Truef(t, chainContainsHandler(handlers, "GatewayTraceRecorder"), "%s chain should include GatewayTraceRecorder", name)
	}
	require.True(t, chainContainsHandler(chains.antigravityAnthropic, "ForcePlatform"))
	require.True(t, chainContainsHandler(chains.antigravityGoogle, "ForcePlatform"))
}

func TestGatewayRoutesExplicitModelsResponsesAndCountTokenPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()
	routeSet := collectRouteSet(router.Routes())

	for _, route := range []string{
		"GET /v1/models",
		"GET /v1/responses",
		"GET /responses",
		"GET /backend-api/codex/responses",
		"GET /antigravity/models",
		"GET /antigravity/v1/models",
		"GET /antigravity/v1beta/models",
		"GET /antigravity/v1beta/models/:model",
		"POST /v1/messages/count_tokens",
		"POST /antigravity/v1/messages/count_tokens",
		"POST /v1beta/models/*modelAction",
		"POST /antigravity/v1beta/models/*modelAction",
	} {
		_, ok := routeSet[route]
		require.Truef(t, ok, "expected route to remain registered: %s", route)
	}
}
