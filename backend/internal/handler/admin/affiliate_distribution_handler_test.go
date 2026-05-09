//go:build unit

package admin

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

type affiliateDistributionAdminRepoStub struct {
	lastBalanceOperator    int64
	lastBalanceUserID      int64
	lastBalanceAmount      float64
	lastBalanceNote        string
	lastPermissionOperator int64
	lastPermissionUserID   int64
	lastPermissionInput    service.UpdateAgentDistributionPermissionInput
	lastPricingUserID      int64
	lastPricingRates       []service.AgentModelRateInput
	permissions            *service.AgentDistributionPermission
	rebateRanking          []service.AgentRebateBalanceRankingItem
	rebateRankingTotal     int64
	monthlyArchives        []service.AgentMonthlyArchiveItem
	monthlyTotal           int64
	updatePricingErr       error
	setBalanceErr          error
	updatePermissionsErr   error
	getPermissionsErr      error
}

func (s *affiliateDistributionAdminRepoStub) EnsureUserAffiliate(context.Context, int64) (*service.AffiliateSummary, error) {
	return &service.AffiliateSummary{}, nil
}
func (s *affiliateDistributionAdminRepoStub) GetAffiliateByCode(context.Context, string) (*service.AffiliateSummary, error) {
	return nil, service.ErrAffiliateProfileNotFound
}
func (s *affiliateDistributionAdminRepoStub) BindInviter(context.Context, int64, int64) (bool, error) {
	return true, nil
}
func (s *affiliateDistributionAdminRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	return false, nil
}
func (s *affiliateDistributionAdminRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}
func (s *affiliateDistributionAdminRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}
func (s *affiliateDistributionAdminRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	return 0, 0, nil
}
func (s *affiliateDistributionAdminRepoStub) ListInvitees(context.Context, int64, int) ([]service.AffiliateInvitee, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	return nil
}
func (s *affiliateDistributionAdminRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	return "", nil
}
func (s *affiliateDistributionAdminRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	return nil
}
func (s *affiliateDistributionAdminRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	return nil
}
func (s *affiliateDistributionAdminRepoStub) ListUsersWithCustomSettings(context.Context, service.AffiliateAdminFilter) ([]service.AffiliateAdminEntry, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionAdminRepoStub) ListAffiliateInviteRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateInviteRecord, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionAdminRepoStub) ListAffiliateRebateRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateRebateRecord, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionAdminRepoStub) ListAffiliateTransferRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateTransferRecord, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionAdminRepoStub) GetAffiliateUserOverview(context.Context, int64) (*service.AffiliateUserOverview, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) GetDistributionOverview(context.Context, int64) (*service.AgentDistributionOverview, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) ListInviteModelRates(context.Context, int64) ([]service.AgentModelRate, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) SaveInviteModelRates(context.Context, int64, []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) ListDirectSubordinates(context.Context, int64) ([]service.AgentDirectMember, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) UpdateDirectSubordinateModelRates(context.Context, int64, int64, []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) ListUserDistributionHistory(context.Context, int64, service.AgentHistoryFilter) ([]service.AgentHistoryItem, int64, error) {
	return nil, 0, nil
}
func (s *affiliateDistributionAdminRepoStub) ListDailyBusinessRanking(context.Context, service.AgentRankingFilter) ([]service.AgentDailyBusinessRankingItem, int64, error) {
	return []service.AgentDailyBusinessRankingItem{{Rank: 1, UserID: 8, BusinessUSD: 300}}, 1, nil
}
func (s *affiliateDistributionAdminRepoStub) ListRebateBalanceRanking(context.Context, service.AgentRankingFilter) ([]service.AgentRebateBalanceRankingItem, int64, error) {
	return append([]service.AgentRebateBalanceRankingItem(nil), s.rebateRanking...), s.rebateRankingTotal, nil
}
func (s *affiliateDistributionAdminRepoStub) GetAgentDistributionPermissions(context.Context, int64) (*service.AgentDistributionPermission, error) {
	if s.getPermissionsErr != nil {
		return nil, s.getPermissionsErr
	}
	if s.permissions == nil {
		return &service.AgentDistributionPermission{UserID: 0}, nil
	}
	copyValue := *s.permissions
	return &copyValue, nil
}
func (s *affiliateDistributionAdminRepoStub) UpdateAgentDistributionPermissions(_ context.Context, operatorUserID, userID int64, input service.UpdateAgentDistributionPermissionInput) (*service.AgentDistributionPermission, error) {
	if s.updatePermissionsErr != nil {
		return nil, s.updatePermissionsErr
	}
	s.lastPermissionOperator = operatorUserID
	s.lastPermissionUserID = userID
	s.lastPermissionInput = input
	result := &service.AgentDistributionPermission{
		UserID:                        userID,
		CanViewDownlineDailyRevenue:   input.CanViewDownlineDailyRevenue,
		CanViewDownlineRebateBalances: input.CanViewDownlineRebateBalances,
		CanManageDownlinePricing:      input.CanManageDownlinePricing,
	}
	s.permissions = result
	return result, nil
}
func (s *affiliateDistributionAdminRepoStub) AdminSetRebateBalance(_ context.Context, operatorUserID, userID int64, amount float64, note string) (*service.AgentRebateBalanceAdjustment, error) {
	if s.setBalanceErr != nil {
		return nil, s.setBalanceErr
	}
	s.lastBalanceOperator = operatorUserID
	s.lastBalanceUserID = userID
	s.lastBalanceAmount = amount
	s.lastBalanceNote = note
	return &service.AgentRebateBalanceAdjustment{
		UserID:             userID,
		OperatorUserID:     operatorUserID,
		PreviousBalanceRMB: 40,
		NewBalanceRMB:      amount,
		Note:               note,
		AdjustedAt:         time.Now().UTC(),
	}, nil
}
func (s *affiliateDistributionAdminRepoStub) GetDistributionTree(context.Context, service.AgentTreeFilter) ([]service.AgentTreeNode, error) {
	return nil, nil
}
func (s *affiliateDistributionAdminRepoStub) GetUserDistributionPricing(context.Context, int64) ([]service.AgentModelRate, error) {
	return []service.AgentModelRate{{Model: "gpt-4.1", Multiplier: 1.6}}, nil
}
func (s *affiliateDistributionAdminRepoStub) AdminUpdateUserDistributionPricing(_ context.Context, _ int64, userID int64, rates []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	if s.updatePricingErr != nil {
		return nil, s.updatePricingErr
	}
	s.lastPricingUserID = userID
	s.lastPricingRates = append([]service.AgentModelRateInput(nil), rates...)
	out := make([]service.AgentModelRate, 0, len(rates))
	for _, rate := range rates {
		out = append(out, service.AgentModelRate{Model: rate.Model, Multiplier: rate.Multiplier})
	}
	return out, nil
}
func (s *affiliateDistributionAdminRepoStub) ListMonthlyRebateArchives(context.Context, service.AgentMonthlyArchiveFilter) ([]service.AgentMonthlyArchiveItem, int64, error) {
	return append([]service.AgentMonthlyArchiveItem(nil), s.monthlyArchives...), s.monthlyTotal, nil
}
func (s *affiliateDistributionAdminRepoStub) ArchiveMonthlyRebateBalances(context.Context, time.Time, *int64, string) (int64, error) {
	return 0, nil
}

type firstAdminLookupStub struct {
	user *service.User
	err  error
}

func (s *firstAdminLookupStub) GetFirstAdmin(context.Context) (*service.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func TestAffiliateHandlerSetRebateBalanceRequiresNote(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAffiliateHandler(service.NewAffiliateService(&affiliateDistributionAdminRepoStub{}, nil, nil, nil), newStubAdminService(), &firstAdminLookupStub{user: &service.User{ID: 1}})
	body := []byte(`{"amount":88.8,"note":""}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/rebate-balances/8", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "8"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.SetRebateBalance(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAffiliateHandlerUpdateUserDistributionPricing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService(), &firstAdminLookupStub{user: &service.User{ID: 1}})
	body := []byte(`{"model_rates":[{"model":"gpt-4.1","multiplier":1.6}]}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/users/8/pricing", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "8"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.UpdateUserDistributionPricing(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(8), repo.lastPricingUserID)
	require.Len(t, repo.lastPricingRates, 1)

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
}

func TestAffiliateHandlerUpdateUserDistributionPricing_AdminCanEditAnyUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService())
	body := []byte(`{"model_rates":[{"model":"claude-3-7-sonnet","multiplier":1.9}]}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/users/99/pricing", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "99"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.UpdateUserDistributionPricing(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(99), repo.lastPricingUserID)
	require.Len(t, repo.lastPricingRates, 1)
	require.Equal(t, "claude-3-7-sonnet", repo.lastPricingRates[0].Model)
	require.Equal(t, 1.9, repo.lastPricingRates[0].Multiplier)
}

func TestAffiliateHandlerSetRebateBalanceByBody_OrdinaryAgentCannotAdjustOwnBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{
		setBalanceErr: infraerrors.Forbidden("INSUFFICIENT_PERMISSIONS", "insufficient permissions"),
	}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService())
	body := []byte(`{"user_id":21,"amount":88.8,"note":"self adjust"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/rebate-balances", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 21})

	handler.SetRebateBalanceByBody(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	var resp struct {
		Code   int    `json:"code"`
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, http.StatusForbidden, resp.Code)
	require.Equal(t, "ROOT_ADMIN_REQUIRED", resp.Reason)
	require.Zero(t, repo.lastBalanceUserID)
	require.Zero(t, repo.lastBalanceOperator)
}

func TestAffiliateHandlerSetRebateBalanceUsesRMBFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService(), &firstAdminLookupStub{user: &service.User{ID: 1}})
	body := []byte(`{"amount":88.8,"note":"manual settle"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/rebate-balances/8", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "8"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.SetRebateBalance(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, float64(40), resp.Data["previous_balance_rmb"])
	require.Equal(t, float64(88.8), resp.Data["new_balance_rmb"])
	_, hasOldPrev := resp.Data["previous_balance_usd"]
	_, hasOldNew := resp.Data["new_balance_usd"]
	require.False(t, hasOldPrev)
	require.False(t, hasOldNew)
}

func TestAffiliateHandlerListRebateBalanceRankingUsesRMBFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{
		rebateRanking: []service.AgentRebateBalanceRankingItem{{
			Rank:                    1,
			UserID:                  8,
			Email:                   "agent@example.com",
			Username:                "agent",
			CurrentRebateBalanceRMB: 88.8,
			TodayRebateRMB:          12.5,
			MonthlyRebateRMB:        66.6,
			DirectUsers:             2,
			DirectAgents:            1,
			LastAdjustedAt:          time.Now().UTC(),
			LastAdjustmentNote:      "manual settle",
		}},
		rebateRankingTotal: 1,
	}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/affiliates/rebate-balances?page=1&page_size=20", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.ListRebateBalanceRanking(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, float64(88.8), resp.Data.Items[0]["current_rebate_balance_rmb"])
	require.Equal(t, float64(12.5), resp.Data.Items[0]["today_rebate_rmb"])
	require.Equal(t, float64(66.6), resp.Data.Items[0]["monthly_rebate_rmb"])
	require.Equal(t, float64(2), resp.Data.Items[0]["direct_users"])
	require.Equal(t, float64(1), resp.Data.Items[0]["direct_agents"])
	_, hasCurrentUSD := resp.Data.Items[0]["current_amount_usd"]
	_, hasLifetimeUSD := resp.Data.Items[0]["lifetime_amount_usd"]
	_, hasOldCurrentRebateUSD := resp.Data.Items[0]["current_rebate_balance_usd"]
	require.False(t, hasCurrentUSD)
	require.False(t, hasLifetimeUSD)
	require.False(t, hasOldCurrentRebateUSD)
}

func TestAffiliateHandlerGetAgentDistributionPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{
		permissions: &service.AgentDistributionPermission{
			UserID:                        8,
			CanViewDownlineDailyRevenue:   true,
			CanViewDownlineRebateBalances: false,
			CanManageDownlinePricing:      true,
		},
	}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/affiliates/users/8/permissions", nil)
	c.Params = gin.Params{{Key: "user_id", Value: "8"}}

	handler.GetAgentDistributionPermissions(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, float64(8), resp.Data["user_id"])
	require.Equal(t, true, resp.Data["can_view_downline_daily_revenue"])
	require.Equal(t, false, resp.Data["can_view_downline_rebate_balances"])
	require.Equal(t, true, resp.Data["can_manage_downline_pricing"])
}

func TestAffiliateHandlerUpdateAgentDistributionPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService())
	body := []byte(`{"can_view_downline_daily_revenue":true,"can_view_downline_rebate_balances":true,"can_manage_downline_pricing":false}`)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/users/8/permissions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "8"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.UpdateAgentDistributionPermissions(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(1), repo.lastPermissionOperator)
	require.Equal(t, int64(8), repo.lastPermissionUserID)
	require.True(t, repo.lastPermissionInput.CanViewDownlineDailyRevenue)
	require.True(t, repo.lastPermissionInput.CanViewDownlineRebateBalances)
	require.False(t, repo.lastPermissionInput.CanManageDownlinePricing)
}

func TestAffiliateHandlerSetRebateBalanceRequiresRootAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService(), &firstAdminLookupStub{user: &service.User{ID: 1}})
	body := []byte(`{"amount":88.8,"note":"manual settle"}`)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/affiliates/rebate-balances/8", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "user_id", Value: "8"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 2})

	handler.SetRebateBalance(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	var resp struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "ROOT_ADMIN_REQUIRED", resp.Reason)
	require.Zero(t, repo.lastBalanceOperator)
}

func TestAffiliateHandlerListMonthlyRebateArchivesUsesRMBFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateDistributionAdminRepoStub{
		monthlyArchives: []service.AgentMonthlyArchiveItem{{
			UserID:            8,
			Email:             "agent@example.com",
			Username:          "agent",
			Month:             "2026-05",
			ArchivedRebateRMB: 120.5,
			ArchivedAt:        time.Now().UTC(),
		}},
		monthlyTotal: 1,
	}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), newStubAdminService())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/affiliates/monthly-archives?page=1&page_size=20", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})

	handler.ListMonthlyRebateArchives(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, float64(120.5), resp.Data.Items[0]["archived_rebate_rmb"])
	_, hasArchivedUSD := resp.Data.Items[0]["archived_rebate_usd"]
	require.False(t, hasArchivedUSD)
}
