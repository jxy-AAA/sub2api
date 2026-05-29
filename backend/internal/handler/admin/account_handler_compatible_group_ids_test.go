package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type compatibleGroupCaptureAdminService struct {
	*stubAdminService
	lastUpdatedAccountID int64
	lastUpdatedGroupIDs  []int64
	lastBulkGroupIDs     []int64
	lastBulkFilter       *service.BulkUpdateAccountFilters
}

func (s *compatibleGroupCaptureAdminService) UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.lastUpdatedAccountID = id
	s.lastUpdatedGroupIDs = nil
	if input != nil && input.GroupIDs != nil {
		s.lastUpdatedGroupIDs = append([]int64(nil), (*input.GroupIDs)...)
	}
	return s.stubAdminService.UpdateAccount(ctx, id, input)
}

func (s *compatibleGroupCaptureAdminService) BulkUpdateAccounts(ctx context.Context, input *service.BulkUpdateAccountsInput) (*service.BulkUpdateAccountsResult, error) {
	s.lastBulkGroupIDs = nil
	s.lastBulkFilter = nil
	if input != nil {
		if input.GroupIDs != nil {
			s.lastBulkGroupIDs = append([]int64(nil), (*input.GroupIDs)...)
		}
		if input.Filters != nil {
			filterCopy := *input.Filters
			s.lastBulkFilter = &filterCopy
		}
	}
	return s.stubAdminService.BulkUpdateAccounts(ctx, input)
}

func setupCompatibleGroupAccountRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/check-mixed-channel", accountHandler.CheckMixedChannel)
	router.POST("/api/v1/admin/accounts", accountHandler.Create)
	router.PUT("/api/v1/admin/accounts/:id", accountHandler.Update)
	router.POST("/api/v1/admin/accounts/bulk-update", accountHandler.BulkUpdate)
	return router
}

func TestAccountHandlerCreateCompatiblePlatformPassesGroupIDs(t *testing.T) {
	adminSvc := &compatibleGroupCaptureAdminService{stubAdminService: newStubAdminService()}
	router := setupCompatibleGroupAccountRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"name":        "compat-openai",
		"platform":    service.PlatformOpenAICompatible,
		"type":        service.AccountTypeAPIKey,
		"credentials": map[string]any{"api_key": "sk-test", "base_url": "https://example.com/v1"},
		"group_ids":   []int64{101, 102},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.Equal(t, service.PlatformOpenAICompatible, adminSvc.createdAccounts[0].Platform)
	require.Equal(t, []int64{101, 102}, adminSvc.createdAccounts[0].GroupIDs)
}

func TestAccountHandlerUpdateCompatiblePlatformPassesGroupIDs(t *testing.T) {
	adminSvc := &compatibleGroupCaptureAdminService{stubAdminService: newStubAdminService()}
	router := setupCompatibleGroupAccountRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"group_ids": []int64{201, 202},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(3), adminSvc.lastUpdatedAccountID)
	require.Equal(t, []int64{201, 202}, adminSvc.lastUpdatedGroupIDs)
}

func TestAccountHandlerBulkUpdateCompatiblePlatformPassesGroupIDs(t *testing.T) {
	adminSvc := &compatibleGroupCaptureAdminService{stubAdminService: newStubAdminService()}
	router := setupCompatibleGroupAccountRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"filters": map[string]any{
			"platform": service.PlatformAnthropicCompatible,
		},
		"group_ids": []int64{301, 302},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{301, 302}, adminSvc.lastBulkGroupIDs)
	require.NotNil(t, adminSvc.lastBulkFilter)
	require.Equal(t, service.PlatformAnthropicCompatible, adminSvc.lastBulkFilter.Platform)
}

func TestAccountHandlerCheckMixedChannelCompatiblePlatformForwardsGroupIDs(t *testing.T) {
	adminSvc := &compatibleGroupCaptureAdminService{stubAdminService: newStubAdminService()}
	router := setupCompatibleGroupAccountRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"platform":  service.PlatformAnthropicCompatible,
		"group_ids": []int64{401, 402},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/check-mixed-channel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.PlatformAnthropicCompatible, adminSvc.lastMixedCheck.platform)
	require.Equal(t, []int64{401, 402}, adminSvc.lastMixedCheck.groupIDs)
}
