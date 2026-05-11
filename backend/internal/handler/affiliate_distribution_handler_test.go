//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type affiliateDistributionRepoStub struct {
	overview                    *service.AgentDistributionOverview
	overviewErr                 error
	affiliateByCode             *service.AffiliateSummary
	getAffiliateByCodeErr       error
	inviteRates                 []service.AgentGroupRate
	permissions                 *service.AgentDistributionPermission
	savedInviteRates            []service.AgentGroupRateInput
	directMembers               []service.AgentDirectMember
	directMembersErr            error
	history                     []service.AgentHistoryItem
	historyTotal                int64
	scopedDailyRanking          []service.AgentDailyBusinessRankingItem
	scopedDailyTotal            int64
	scopedRebateRanking         []service.AgentRebateBalanceRankingItem
	scopedRebateTotal           int64
	scopedTree                  []service.AgentTreeNode
	scopedPricing               []service.AgentGroupRate
	lastUpdatedSubordinateID    int64
	lastUpdatedSubordinateRates []service.AgentGroupRateInput
	lastScopedPricingUserID     int64
	lastScopedPricingRates      []service.AgentGroupRateInput
	updateSubordinateErr        error
	updatePermissionsErr        error
	scopedDailyErr              error
	scopedRebateErr             error
	scopedTreeErr               error
	scopedPricingErr            error
	updateScopedPricingErr      error
	lastParentUserID            int64
}

func (s *affiliateDistributionRepoStub) EnsureUserAffiliate(context.Context, int64) (*service.AffiliateSummary, error) {
	return &service.AffiliateSummary{}, nil
}
func (s *affiliateDistributionRepoStub) GetAffiliateByCode(context.Context, string) (*service.AffiliateSummary, error) {
	if s.getAffiliateByCodeErr != nil {
		return nil, s.getAffiliateByCodeErr
	}
	if s.affiliateByCode != nil {
		copyValue := *s.affiliateByCode
		return &copyValue, nil
	}
	return nil, service.ErrAffiliateProfileNotFound
}
func (s *affiliateDistributionRepoStub) BindInviter(context.Context, int64, int64) (bool, error) {
	return true, nil
}
func (s *affiliateDistributionRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	return false, nil
}
func (s *affiliateDistributionRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}
func (s *affiliateDistributionRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}
func (s *affiliateDistributionRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	return 0, 0, nil
}
func (s *affiliateDistributionRepoStub) ListInvitees(context.Context, int64, int) ([]service.AffiliateInvitee, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	return nil
}
func (s *affiliateDistributionRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	return "", nil
}
func (s *affiliateDistributionRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	return nil
}
func (s *affiliateDistributionRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	return nil
}
func (s *affiliateDistributionRepoStub) ListUsersWithCustomSettings(context.Context, service.AffiliateAdminFilter) ([]service.AffiliateAdminEntry, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) ListAffiliateInviteRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateInviteRecord, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) ListAffiliateRebateRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateRebateRecord, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) ListAffiliateTransferRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateTransferRecord, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) GetAffiliateUserOverview(context.Context, int64) (*service.AffiliateUserOverview, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) GetDistributionOverview(context.Context, int64) (*service.AgentDistributionOverview, error) {
	if s.overviewErr != nil {
		return nil, s.overviewErr
	}
	return s.overview, nil
}
func (s *affiliateDistributionRepoStub) ListInviteGroupRates(context.Context, int64) ([]service.AgentGroupRate, error) {
	return append([]service.AgentGroupRate(nil), s.inviteRates...), nil
}
func (s *affiliateDistributionRepoStub) SaveInviteGroupRates(_ context.Context, _ int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	s.savedInviteRates = append([]service.AgentGroupRateInput(nil), rates...)
	out := make([]service.AgentGroupRate, 0, len(rates))
	for _, rate := range rates {
		out = append(out, service.AgentGroupRate{GroupID: rate.GroupID, RateMultiplier: rate.RateMultiplier})
	}
	return out, nil
}
func (s *affiliateDistributionRepoStub) ListDirectSubordinates(context.Context, int64) ([]service.AgentDirectMember, error) {
	if s.directMembersErr != nil {
		return nil, s.directMembersErr
	}
	return append([]service.AgentDirectMember(nil), s.directMembers...), nil
}
func (s *affiliateDistributionRepoStub) UpdateDirectSubordinateGroupRates(_ context.Context, _ int64, subordinateUserID int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	if s.updateSubordinateErr != nil {
		return nil, s.updateSubordinateErr
	}
	s.lastUpdatedSubordinateID = subordinateUserID
	s.lastUpdatedSubordinateRates = append([]service.AgentGroupRateInput(nil), rates...)
	out := make([]service.AgentGroupRate, 0, len(rates))
	for _, rate := range rates {
		out = append(out, service.AgentGroupRate{GroupID: rate.GroupID, RateMultiplier: rate.RateMultiplier})
	}
	return out, nil
}
func (s *affiliateDistributionRepoStub) ListUserDistributionHistory(context.Context, int64, service.AgentHistoryFilter) ([]service.AgentHistoryItem, int64, error) {
	return append([]service.AgentHistoryItem(nil), s.history...), s.historyTotal, nil
}
func (s *affiliateDistributionRepoStub) ListDailyBusinessRanking(context.Context, service.AgentRankingFilter) ([]service.AgentDailyBusinessRankingItem, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) ListRebateBalanceRanking(context.Context, service.AgentRankingFilter) ([]service.AgentRebateBalanceRankingItem, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) GetAgentDistributionPermissions(context.Context, int64) (*service.AgentDistributionPermission, error) {
	if s.updatePermissionsErr != nil {
		return nil, s.updatePermissionsErr
	}
	if s.permissions == nil {
		return &service.AgentDistributionPermission{}, nil
	}
	copyValue := *s.permissions
	return &copyValue, nil
}
func (s *affiliateDistributionRepoStub) UpdateAgentDistributionPermissions(context.Context, int64, int64, service.UpdateAgentDistributionPermissionInput) (*service.AgentDistributionPermission, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) AdminSetRebateBalance(context.Context, int64, int64, float64, string) (*service.AgentRebateBalanceAdjustment, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) GetDistributionTree(context.Context, service.AgentTreeFilter) ([]service.AgentTreeNode, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) GetUserDistributionGroupRates(context.Context, int64) ([]service.AgentGroupRate, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) AdminUpdateUserDistributionGroupRates(context.Context, int64, int64, []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	return nil, nil
}
func (s *affiliateDistributionRepoStub) ListMonthlyRebateArchives(context.Context, service.AgentMonthlyArchiveFilter) ([]service.AgentMonthlyArchiveItem, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionRepoStub) ArchiveMonthlyRebateBalances(context.Context, time.Time, *int64, string) (int64, error) {
	return 0, nil
}
func (s *affiliateDistributionRepoStub) ListDailyBusinessRankingScoped(context.Context, int64, service.AgentRankingFilter) ([]service.AgentDailyBusinessRankingItem, int64, error) {
	if s.scopedDailyErr != nil {
		return nil, 0, s.scopedDailyErr
	}
	return append([]service.AgentDailyBusinessRankingItem(nil), s.scopedDailyRanking...), s.scopedDailyTotal, nil
}
func (s *affiliateDistributionRepoStub) ListRebateBalanceRankingScoped(context.Context, int64, service.AgentRankingFilter) ([]service.AgentRebateBalanceRankingItem, int64, error) {
	if s.scopedRebateErr != nil {
		return nil, 0, s.scopedRebateErr
	}
	return append([]service.AgentRebateBalanceRankingItem(nil), s.scopedRebateRanking...), s.scopedRebateTotal, nil
}
func (s *affiliateDistributionRepoStub) GetDistributionTreeScoped(context.Context, int64, service.AgentTreeFilter) ([]service.AgentTreeNode, error) {
	if s.scopedTreeErr != nil {
		return nil, s.scopedTreeErr
	}
	return append([]service.AgentTreeNode(nil), s.scopedTree...), nil
}
func (s *affiliateDistributionRepoStub) GetUserDistributionGroupRatesScoped(context.Context, int64, int64) ([]service.AgentGroupRate, error) {
	if s.scopedPricingErr != nil {
		return nil, s.scopedPricingErr
	}
	return append([]service.AgentGroupRate(nil), s.scopedPricing...), nil
}
func (s *affiliateDistributionRepoStub) UpdateUserDistributionGroupRatesScoped(_ context.Context, _ int64, userID int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	if s.updateScopedPricingErr != nil {
		return nil, s.updateScopedPricingErr
	}
	s.lastScopedPricingUserID = userID
	s.lastScopedPricingRates = append([]service.AgentGroupRateInput(nil), rates...)
	out := make([]service.AgentGroupRate, 0, len(rates))
	for _, rate := range rates {
		out = append(out, service.AgentGroupRate{GroupID: rate.GroupID, RateMultiplier: rate.RateMultiplier})
	}
	return out, nil
}

func TestUserHandlerTransferAffiliateQuotaDeprecated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(&affiliateDistributionRepoStub{}, nil, nil, nil))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/aff/transfer", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 10})

	handler.TransferAffiliateQuota(c)

	require.Equal(t, http.StatusGone, recorder.Code)
	var resp struct {
		Code    int    `json:"code"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, http.StatusGone, resp.Code)
	require.Equal(t, "AFFILIATE_TRANSFER_REMOVED", resp.Reason)
}

func TestUserHandlerSaveMyInviteGroupRates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))
	body := []byte(`{"group_rates":[{"group_id":1,"rate_multiplier":1.6},{"group_id":2,"rate_multiplier":1.8}]}`)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/user/aff/invite-pricing", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.SaveMyInviteGroupRates(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, repo.savedInviteRates, 2)
	require.Equal(t, int64(1), repo.savedInviteRates[0].GroupID)
	require.Equal(t, 1.6, repo.savedInviteRates[0].RateMultiplier)
}

func TestUserHandlerGetAffiliateUsesRMBFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		overview: &service.AgentDistributionOverview{
			UserID:                  21,
			InviteCode:              "AGENT21",
			TodayBusinessUSD:        2000,
			TodayRebateRMB:          40,
			CurrentRebateBalanceRMB: 88.8,
		},
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.GetAffiliate(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, float64(40), resp.Data["today_rebate_rmb"])
	require.Equal(t, float64(88.8), resp.Data["current_rebate_balance_rmb"])
	_, hasTodayUSD := resp.Data["today_rebate_usd"]
	_, hasBalanceUSD := resp.Data["current_rebate_balance_usd"]
	require.False(t, hasTodayUSD)
	require.False(t, hasBalanceUSD)
}

func TestUserHandlerGetAffiliateIncludesDirectChildrenFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		overview: &service.AgentDistributionOverview{
			UserID:                  21,
			InviteCode:              "AGENT21",
			TodayBusinessUSD:        2000,
			TodayRebateRMB:          40,
			CurrentRebateBalanceRMB: 88.8,
		},
		directMembers: []service.AgentDirectMember{{
			UserID:             33,
			Email:              "child@example.com",
			Username:           "child",
			CurrentGroupRates:  []service.AgentGroupRate{{GroupID: 1, GroupName: "Default", RateMultiplier: 1.6}},
			ParentCanEditRates: true,
		}},
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/distribution", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.GetAffiliate(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.JSONEq(t, `[{"user_id":33,"email":"child@example.com","username":"child","is_agent":false,"today_business_usd":0,"today_rebate_rmb":0,"current_rebate_balance_rmb":0,"current_group_rates":[{"group_id":1,"group_name":"Default","group_rate_multiplier":0,"rate_multiplier":1.6}],"parent_can_edit_rates":true}]`, string(resp.Data["direct_children"]))
	require.JSONEq(t, `1`, string(resp.Data["direct_children_count"]))
	require.JSONEq(t, `1`, string(resp.Data["direct_member_count"]))
}

func TestUserHandlerListDirectAffiliateMembersUsesRMBFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		directMembers: []service.AgentDirectMember{{
			UserID:                  33,
			Email:                   "child@example.com",
			Username:                "child",
			IsAgent:                 true,
			TodayBusinessUSD:        1200,
			TodayRebateRMB:          24,
			CurrentRebateBalanceRMB: 66.6,
			CurrentGroupRates:       []service.AgentGroupRate{{GroupID: 1, GroupName: "Default", RateMultiplier: 1.6}},
			ParentCanEditRates:      true,
		}},
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/direct-members", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.ListDirectAffiliateMembers(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, float64(24), resp.Data[0]["today_rebate_rmb"])
	require.Equal(t, float64(66.6), resp.Data[0]["current_rebate_balance_rmb"])
	_, hasTodayUSD := resp.Data[0]["today_rebate_usd"]
	_, hasBalanceUSD := resp.Data[0]["current_rebate_balance_usd"]
	require.False(t, hasTodayUSD)
	require.False(t, hasBalanceUSD)
}

func TestUserHandlerListDirectAffiliateMembersReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(&affiliateDistributionRepoStub{}, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/direct-members", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.ListDirectAffiliateMembers(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.JSONEq(t, `[]`, string(resp.Data))
}

func TestUserHandlerListDirectAffiliateMembersReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		directMembersErr: infraerrors.ServiceUnavailable("DIRECT_MEMBERS_FAILED", "failed to load direct members"),
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/direct-members", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.ListDirectAffiliateMembers(c)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	var resp struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "DIRECT_MEMBERS_FAILED", resp.Reason)
}

func TestUserHandlerListMyAffiliateBusinessHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().UTC()
	repo := &affiliateDistributionRepoStub{
		history: []service.AgentHistoryItem{{
			StatDate:       "2026-05-09",
			BusinessUSD:    200,
			RebateRMB:      40,
			LastCalculated: now,
		}},
		historyTotal: 1,
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/history?page=1&page_size=20", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.ListMyAffiliateBusinessHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, float64(40), resp.Data.Items[0]["rebate_rmb"])
	_, hasOldUSD := resp.Data.Items[0]["rebate_usd"]
	require.False(t, hasOldUSD)
}

func TestUserHandlerGetManagedDistributionPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		permissions: &service.AgentDistributionPermission{
			UserID:                        21,
			CanViewDownlineDailyRevenue:   true,
			CanViewDownlineRebateBalances: false,
			CanManageDownlinePricing:      true,
		},
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/managed/permissions", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.GetManagedDistributionPermissions(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, true, resp.Data["can_view_downline_daily_revenue"])
	require.Equal(t, false, resp.Data["can_view_downline_rebate_balances"])
	require.Equal(t, true, resp.Data["can_manage_downline_pricing"])
}

func TestUserHandlerListManagedDailyBusinessRankingRequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		permissions: &service.AgentDistributionPermission{UserID: 21},
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/managed/daily-revenue?page=1&page_size=20", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.ListManagedDailyBusinessRanking(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	var resp struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "DOWNLINE_DAILY_REVENUE_FORBIDDEN", resp.Reason)
}

func TestUserHandlerListManagedRebateBalanceRanking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		permissions: &service.AgentDistributionPermission{
			UserID:                        21,
			CanViewDownlineRebateBalances: true,
		},
		scopedRebateRanking: []service.AgentRebateBalanceRankingItem{{
			UserID:                  33,
			CurrentRebateBalanceRMB: 88.8,
			TodayRebateRMB:          12.3,
		}},
		scopedRebateTotal: 1,
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/aff/managed/rebate-balances?page=1&page_size=20", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.ListManagedRebateBalanceRanking(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, float64(88.8), resp.Data.Items[0]["current_rebate_balance_rmb"])
}

func TestUserHandlerUpdateManagedUserDistributionPricingRejectsBranchUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionRepoStub{
		permissions: &service.AgentDistributionPermission{
			UserID:                   21,
			CanManageDownlinePricing: true,
		},
		updateScopedPricingErr: infraerrors.Forbidden("INSUFFICIENT_PERMISSIONS", "insufficient permissions"),
	}
	handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil))
	body := []byte(`{"group_rates":[{"group_id":1,"rate_multiplier":1.7}]}`)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/user/aff/managed/users/55/pricing", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "55"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.UpdateManagedUserDistributionPricing(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	var resp struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "INSUFFICIENT_PERMISSIONS", resp.Reason)
	require.Zero(t, repo.lastScopedPricingUserID)
}

func TestUserHandlerUpdateDirectAffiliateMemberRates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		actorUserID        int64
		targetUserID       string
		repo               *affiliateDistributionRepoStub
		wantStatus         int
		wantReason         string
		wantUpdatedUserID  int64
		wantUpdatedRateLen int
	}{
		{
			name:               "direct parent updates subordinate successfully",
			actorUserID:        21,
			targetUserID:       "33",
			repo:               &affiliateDistributionRepoStub{},
			wantStatus:         http.StatusOK,
			wantUpdatedUserID:  33,
			wantUpdatedRateLen: 1,
		},
		{
			name:         "parent updating non direct user is forbidden",
			actorUserID:  21,
			targetUserID: "34",
			repo: &affiliateDistributionRepoStub{
				updateSubordinateErr: infraerrors.Forbidden("INSUFFICIENT_PERMISSIONS", "insufficient permissions"),
			},
			wantStatus: http.StatusForbidden,
			wantReason: "INSUFFICIENT_PERMISSIONS",
		},
		{
			name:         "ordinary agent updating self is forbidden",
			actorUserID:  21,
			targetUserID: "21",
			repo: &affiliateDistributionRepoStub{
				updateSubordinateErr: infraerrors.Forbidden("INSUFFICIENT_PERMISSIONS", "insufficient permissions"),
			},
			wantStatus: http.StatusForbidden,
			wantReason: "INSUFFICIENT_PERMISSIONS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(tt.repo, nil, nil, nil))
			body := []byte(`{"group_rates":[{"group_id":1,"rate_multiplier":1.6}]}`)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/user/aff/direct-members/"+tt.targetUserID+"/pricing", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "user_id", Value: tt.targetUserID}}
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: tt.actorUserID})

			handler.UpdateDirectAffiliateMemberRates(c)

			require.Equal(t, tt.wantStatus, recorder.Code)
			if tt.wantStatus == http.StatusOK {
				require.Equal(t, tt.wantUpdatedUserID, tt.repo.lastUpdatedSubordinateID)
				require.Len(t, tt.repo.lastUpdatedSubordinateRates, tt.wantUpdatedRateLen)
				return
			}

			var resp struct {
				Code   int    `json:"code"`
				Reason string `json:"reason"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
			require.Equal(t, tt.wantStatus, resp.Code)
			require.Equal(t, tt.wantReason, resp.Reason)
			require.Zero(t, tt.repo.lastUpdatedSubordinateID)
			require.Empty(t, tt.repo.lastUpdatedSubordinateRates)
		})
	}
}
