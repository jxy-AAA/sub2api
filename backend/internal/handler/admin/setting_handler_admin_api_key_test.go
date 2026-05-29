//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingHandlerFirstAdminStub struct {
	admin *service.User
	err   error
}

func (s *settingHandlerFirstAdminStub) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.admin == nil {
		return nil, nil
	}
	clone := *s.admin
	return &clone, nil
}

func TestSettingHandler_AdminAPIKeyEndpointsRequireRootJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingHandlerRepoStub{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{})
	rootAdmin := &service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive}
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, &settingHandlerFirstAdminStub{admin: rootAdmin})

	tests := []struct {
		name       string
		method     string
		handlerFn  func(*gin.Context)
		authMethod string
		userID     int64
		wantStatus int
	}{
		{name: "non_root_jwt_get_rejected", method: http.MethodGet, handlerFn: handler.GetAdminAPIKey, authMethod: "jwt", userID: 2, wantStatus: http.StatusForbidden},
		{name: "api_key_auth_get_rejected", method: http.MethodGet, handlerFn: handler.GetAdminAPIKey, authMethod: "admin_api_key", userID: 0, wantStatus: http.StatusForbidden},
		{name: "non_root_jwt_regenerate_rejected", method: http.MethodPost, handlerFn: handler.RegenerateAdminAPIKey, authMethod: "jwt", userID: 2, wantStatus: http.StatusForbidden},
		{name: "non_root_jwt_delete_rejected", method: http.MethodDelete, handlerFn: handler.DeleteAdminAPIKey, authMethod: "jwt", userID: 2, wantStatus: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(tc.method, "/api/v1/admin/settings/admin-api-key", nil)
			c.Set("auth_method", tc.authMethod)
			c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
				UserID:        tc.userID,
				PrincipalID:   "user:test",
				PrincipalType: "user",
				IsSystem:      tc.authMethod == "admin_api_key",
			})

			tc.handlerFn(c)

			require.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestSettingHandler_GetAdminAPIKey_RootJWTGetsMetadataWithoutRawKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingHandlerRepoStub{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{})
	rawKey, err := svc.GenerateAdminAPIKey(context.Background())
	require.NoError(t, err)

	rootAdmin := &service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive}
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, &settingHandlerFirstAdminStub{admin: rootAdmin})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/admin-api-key", nil)
	c.Set("auth_method", "jwt")
	c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
		UserID:        rootAdmin.ID,
		PrincipalID:   "user:1",
		PrincipalType: "user",
	})

	handler.GetAdminAPIKey(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), rawKey)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, data["exists"])
	require.Equal(t, "configured", data["masked_key"])
	_, hasRawKey := data["key"]
	require.False(t, hasRawKey)
}
