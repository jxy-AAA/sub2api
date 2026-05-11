package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardStatsContractRepo struct {
	service.UsageLogRepository
	stats *usagestats.DashboardStats
}

func (r *dashboardStatsContractRepo) GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	return r.stats, nil
}

func TestDashboardHandler_GetStatsIncludesAccountCostFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &dashboardStatsContractRepo{stats: &usagestats.DashboardStats{
		TotalCost:        10.25,
		TotalActualCost:  8.75,
		TotalAccountCost: 4.50,
		TodayCost:        2.25,
		TodayActualCost:  1.75,
		TodayAccountCost: 0.50,
	}}
	dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
	handler := NewDashboardHandler(dashboardSvc, nil)
	router := gin.New()
	router.GET("/admin/dashboard/stats", handler.GetStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.InDelta(t, 4.50, body.Data["total_account_cost"], 0.0001)
	require.InDelta(t, 0.50, body.Data["today_account_cost"], 0.0001)
}
