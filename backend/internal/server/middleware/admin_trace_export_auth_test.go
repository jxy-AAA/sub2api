//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminTraceExportRouteRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil)

	adminUser := &service.User{
		ID:           1,
		Email:        "admin@example.com",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		TokenVersion: 1,
		Concurrency:  1,
	}
	regularUser := &service.User{
		ID:           2,
		Email:        "user@example.com",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		TokenVersion: 1,
		Concurrency:  1,
	}

	userRepo := &stubUserRepo{
		getByID: func(ctx context.Context, id int64) (*service.User, error) {
			switch id {
			case adminUser.ID:
				clone := *adminUser
				return &clone, nil
			case regularUser.ID:
				clone := *regularUser
				return &clone, nil
			default:
				return nil, service.ErrUserNotFound
			}
		},
	}
	userService := service.NewUserService(userRepo, nil, nil, nil)

	t.Run("regular user jwt is forbidden", func(t *testing.T) {
		router := newAdminTraceExportTestRouter(NewAdminAuthMiddleware(authService, userService, nil))

		token, err := authService.GenerateToken(&service.User{
			ID:           regularUser.ID,
			Email:        regularUser.Email,
			Role:         regularUser.Role,
			TokenVersion: regularUser.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
		require.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("admin jwt is allowed", func(t *testing.T) {
		router := newAdminTraceExportTestRouter(NewAdminAuthMiddleware(authService, userService, nil))

		token, err := authService.GenerateToken(&service.User{
			ID:           adminUser.ID,
			Email:        adminUser.Email,
			Role:         adminUser.Role,
			TokenVersion: adminUser.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"auth_method":"jwt"`)
	})

	t.Run("admin api key is allowed", func(t *testing.T) {
		settingSvc := service.NewSettingService(&adminAuthSettingRepoStub{
			values: map[string]string{
				service.SettingKeyAdminAPIKey: "admin-export-key",
			},
		}, &config.Config{})
		router := newAdminTraceExportTestRouter(NewAdminAuthMiddleware(nil, userService, settingSvc))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export", nil)
		req.Header.Set("x-api-key", "admin-export-key")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"auth_method":"admin_api_key"`)
	})
}

func newAdminTraceExportTestRouter(mw AdminAuthMiddleware) *gin.Engine {
	router := gin.New()
	router.Use(gin.HandlerFunc(mw))
	router.GET("/api/v1/admin/traces/export", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":          true,
			"auth_method": c.GetString("auth_method"),
		})
	})
	return router
}
