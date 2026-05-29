package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubTraceExportService struct {
	bundle *service.TaodingTraceExportBundle
	err    error
}

func (s *stubTraceExportService) Export(ctx context.Context, now time.Time) (*service.TaodingTraceExportBundle, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.bundle != nil {
		return s.bundle, nil
	}
	return &service.TaodingTraceExportBundle{
		Type:          service.TaodingTraceExportType,
		Version:       service.TaodingTraceExportVersion,
		SchemaVersion: service.TaodingTraceSchemaVersion,
		ExportedAt:    now.UTC().Format(time.RFC3339Nano),
		Records:       []service.CodexTraceExport{},
	}, nil
}

type stubTraceRootAdminReader struct {
	user *service.User
	err  error
}

func (s *stubTraceRootAdminReader) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func TestTraceExportHandlerRootAdminJWTDownloadsJSONArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TraceExportHandler{
		traceService: &stubTraceExportService{
			bundle: &service.TaodingTraceExportBundle{
				Type:          service.TaodingTraceExportType,
				Version:       service.TaodingTraceExportVersion,
				SchemaVersion: service.TaodingTraceSchemaVersion,
				ExportedAt:    "2026-05-27T00:00:00Z",
				Count:         1,
				Records: []service.CodexTraceExport{{
					TaskID:          "task-1",
					Prompt:          json.RawMessage(`[{"role":"user","content":"hi"}]`),
					Candidates:      json.RawMessage(`[{"message":{"role":"assistant","content":"ok"}}]`),
					Tools:           json.RawMessage(`[]`),
					Signature:       json.RawMessage(`{"available":false}`),
					Meta:            json.RawMessage(`{"source":"unit"}`),
					Scaffold:        json.RawMessage(`{"name":"sub2api"}`),
					ScaffoldVersion: service.TaodingTraceScaffoldVersion,
				}},
			},
		},
		userService: &stubTraceRootAdminReader{user: &service.User{ID: 1, Role: service.RoleAdmin}},
	}

	router := gin.New()
	router.GET("/api/v1/admin/traces/export", func(c *gin.Context) {
		c.Set("auth_method", "jwt")
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, PrincipalType: "user"})
		handler.Export(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Disposition"), "sub2api-taoding-data-")
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var payload []service.CodexTraceExport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload, 1)
	require.Equal(t, "task-1", payload[0].TaskID)
	require.Equal(t, service.TaodingTraceExportType, rec.Header().Get("X-Taoding-Trace-Export-Type"))
}

func TestTraceExportHandlerRequiresRootAdminJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		authMethod string
		subject    middleware.AuthSubject
		firstAdmin *service.User
	}{
		{
			name:       "admin api key denied",
			authMethod: "admin_api_key",
			subject:    middleware.AuthSubject{IsSystem: true, PrincipalType: "admin_api_key"},
			firstAdmin: &service.User{ID: 1, Role: service.RoleAdmin},
		},
		{
			name:       "non root admin denied",
			authMethod: "jwt",
			subject:    middleware.AuthSubject{UserID: 2, PrincipalType: "user"},
			firstAdmin: &service.User{ID: 1, Role: service.RoleAdmin},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			handler := &TraceExportHandler{
				traceService: &stubTraceExportService{},
				userService:  &stubTraceRootAdminReader{user: tc.firstAdmin},
			}
			router := gin.New()
			router.GET("/api/v1/admin/traces/export", func(c *gin.Context) {
				c.Set("auth_method", tc.authMethod)
				c.Set(string(middleware.ContextKeyUser), tc.subject)
				handler.Export(c)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export", nil)
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}

func TestTraceExportHandlerRootResolverError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TraceExportHandler{
		traceService: &stubTraceExportService{},
		userService:  &stubTraceRootAdminReader{err: errors.New("boom")},
	}
	router := gin.New()
	router.GET("/api/v1/admin/traces/export", func(c *gin.Context) {
		c.Set("auth_method", "jwt")
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		handler.Export(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
