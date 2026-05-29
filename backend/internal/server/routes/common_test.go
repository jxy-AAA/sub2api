package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubRouteHealthChecker struct {
	err error
}

func (s stubRouteHealthChecker) Check(ctx context.Context) error {
	return s.err
}

func TestRegisterCommonRoutesHealthReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("healthy", func(t *testing.T) {
		router := gin.New()
		RegisterCommonRoutes(router, HealthDependencies{
			Postgres: stubRouteHealthChecker{},
			Redis:    stubRouteHealthChecker{},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, "ok", body["status"])
	})

	t.Run("degraded", func(t *testing.T) {
		router := gin.New()
		RegisterCommonRoutes(router, HealthDependencies{
			Postgres: stubRouteHealthChecker{err: context.DeadlineExceeded},
			Redis:    stubRouteHealthChecker{},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Contains(t, w.Body.String(), "degraded")
		require.Contains(t, w.Body.String(), "context deadline exceeded")
	})

	t.Run("livez", func(t *testing.T) {
		router := gin.New()
		RegisterCommonRoutes(router, HealthDependencies{
			Postgres: stubRouteHealthChecker{},
			Redis:    stubRouteHealthChecker{},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/livez", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"status":"ok"`)
	})
}
