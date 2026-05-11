package handler

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type groupRatePayload struct {
	GroupID        int64   `json:"group_id"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type updateGroupRatesRequest struct {
	GroupRates []groupRatePayload `json:"group_rates" binding:"required"`
}

// GetAffiliate returns the current user's distribution overview.
// GET /api/v1/user/aff
func (h *UserHandler) GetAffiliate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	detail, err := h.affiliateService.GetDistributionOverview(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

// GetMyInviteGroupRates returns the current user's invite-code group pricing.
// GET /api/v1/user/aff/invite-pricing
func (h *UserHandler) GetMyInviteGroupRates(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	rates, err := h.affiliateService.ListInviteGroupRates(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_rates": rates, "invite_group_rates": rates})
}

// SaveMyInviteGroupRates updates the current user's invite-code group pricing.
// PUT /api/v1/user/aff/invite-pricing
func (h *UserHandler) SaveMyInviteGroupRates(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	rates, err := bindGroupRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.SaveInviteGroupRates(c.Request.Context(), subject.UserID, rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_rates": updated, "invite_group_rates": updated})
}

// ListDirectAffiliateMembers returns the current user's direct subordinates.
// GET /api/v1/user/aff/direct-members
func (h *UserHandler) ListDirectAffiliateMembers(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	items, err := h.affiliateService.ListDirectSubordinates(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

// UpdateDirectAffiliateMemberRates updates one direct subordinate's group pricing.
// PUT /api/v1/user/aff/direct-members/:user_id/pricing
func (h *UserHandler) UpdateDirectAffiliateMemberRates(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	targetUserID, err := parsePositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rates, err := bindGroupRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.UpdateDirectSubordinateGroupRates(c.Request.Context(), subject.UserID, targetUserID, rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"user_id":             targetUserID,
		"group_rates":         updated,
		"current_group_rates": updated,
	})
}

// ListMyAffiliateBusinessHistory returns the current user's daily business/rebate history.
// GET /api/v1/user/aff/history
func (h *UserHandler) ListMyAffiliateBusinessHistory(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	filter, err := parseAgentHistoryFilter(c, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items, total, err := h.affiliateService.ListUserDistributionHistory(c.Request.Context(), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// GetManagedDistributionPermissions returns the current user's downline-management permissions.
// GET /api/v1/user/aff/managed/permissions
func (h *UserHandler) GetManagedDistributionPermissions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	permissions, err := h.affiliateService.GetAgentDistributionPermissions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, permissions)
}

// ListManagedDailyBusinessRanking returns the current agent's scoped daily business list.
// GET /api/v1/user/aff/managed/daily-revenue
func (h *UserHandler) ListManagedDailyBusinessRanking(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	permissions, err := h.requireManagedPermission(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !permissions.CanViewDownlineDailyRevenue {
		response.ErrorFrom(c, infraerrors.Forbidden("DOWNLINE_DAILY_REVENUE_FORBIDDEN", "downline daily revenue access required"))
		return
	}

	page, pageSize := response.ParsePagination(c)
	filter, err := parseManagedAgentRankingFilter(c, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items, total, err := h.affiliateService.ListDailyBusinessRankingScoped(c.Request.Context(), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// ListManagedRebateBalanceRanking returns the current agent's scoped rebate balance list.
// GET /api/v1/user/aff/managed/rebate-balances
func (h *UserHandler) ListManagedRebateBalanceRanking(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	permissions, err := h.requireManagedPermission(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !permissions.CanViewDownlineRebateBalances {
		response.ErrorFrom(c, infraerrors.Forbidden("DOWNLINE_REBATE_BALANCES_FORBIDDEN", "downline rebate balance access required"))
		return
	}

	page, pageSize := response.ParsePagination(c)
	filter, err := parseManagedAgentRankingFilter(c, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items, total, err := h.affiliateService.ListRebateBalanceRankingScoped(c.Request.Context(), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// GetManagedDistributionTree returns the current agent's scoped downline tree.
// GET /api/v1/user/aff/managed/tree
func (h *UserHandler) GetManagedDistributionTree(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	permissions, err := h.requireManagedPermission(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !permissions.CanViewDownlineDailyRevenue && !permissions.CanViewDownlineRebateBalances && !permissions.CanManageDownlinePricing {
		response.ErrorFrom(c, infraerrors.Forbidden("DOWNLINE_TREE_FORBIDDEN", "downline management access required"))
		return
	}

	items, err := h.affiliateService.GetDistributionTreeScoped(c.Request.Context(), subject.UserID, service.AgentTreeFilter{
		Search:          strings.TrimSpace(c.Query("search")),
		OnlyDescendants: true,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

// GetManagedUserDistributionPricing returns one scoped downline user's pricing.
// GET /api/v1/user/aff/managed/users/:user_id/pricing
func (h *UserHandler) GetManagedUserDistributionPricing(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	permissions, err := h.requireManagedPermission(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !permissions.CanManageDownlinePricing {
		response.ErrorFrom(c, infraerrors.Forbidden("DOWNLINE_PRICING_FORBIDDEN", "downline pricing access required"))
		return
	}

	targetUserID, err := parsePositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	rates, err := h.affiliateService.GetUserDistributionGroupRatesScoped(c.Request.Context(), subject.UserID, targetUserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": targetUserID, "group_rates": rates, "current_group_rates": rates})
}

// UpdateManagedUserDistributionPricing updates one scoped downline user's pricing.
// PUT /api/v1/user/aff/managed/users/:user_id/pricing
func (h *UserHandler) UpdateManagedUserDistributionPricing(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	permissions, err := h.requireManagedPermission(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !permissions.CanManageDownlinePricing {
		response.ErrorFrom(c, infraerrors.Forbidden("DOWNLINE_PRICING_FORBIDDEN", "downline pricing access required"))
		return
	}

	targetUserID, err := parsePositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rates, err := bindGroupRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.UpdateUserDistributionGroupRatesScoped(c.Request.Context(), subject.UserID, targetUserID, rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": targetUserID, "group_rates": updated, "current_group_rates": updated})
}

// TransferAffiliateQuota is retained for compatibility but disabled.
// POST /api/v1/user/aff/transfer
func (h *UserHandler) TransferAffiliateQuota(c *gin.Context) {
	response.ErrorFrom(c, infraerrors.New(http.StatusGone, "AFFILIATE_TRANSFER_REMOVED", "affiliate transfer is no longer available in distribution mode"))
}

func bindGroupRatesRequest(c *gin.Context) ([]service.AgentGroupRateInput, error) {
	var req updateGroupRatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, infraerrors.BadRequest("INVALID_REQUEST", "invalid request: "+err.Error())
	}
	if len(req.GroupRates) == 0 {
		return nil, infraerrors.BadRequest("INVALID_GROUP_RATES", "group_rates cannot be empty")
	}

	rates := make([]service.AgentGroupRateInput, 0, len(req.GroupRates))
	seen := make(map[int64]struct{}, len(req.GroupRates))
	for _, item := range req.GroupRates {
		if item.GroupID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_GROUP_RATES", "group_id is required")
		}
		if _, exists := seen[item.GroupID]; exists {
			return nil, infraerrors.BadRequest("INVALID_GROUP_RATES", "duplicate group_id")
		}
		if item.RateMultiplier <= 0 || math.IsNaN(item.RateMultiplier) || math.IsInf(item.RateMultiplier, 0) {
			return nil, infraerrors.BadRequest("INVALID_GROUP_RATES", "rate_multiplier must be greater than 0")
		}
		seen[item.GroupID] = struct{}{}
		rates = append(rates, service.AgentGroupRateInput{
			GroupID:        item.GroupID,
			RateMultiplier: item.RateMultiplier,
		})
	}
	return rates, nil
}

func parsePositiveInt64(raw, field string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, infraerrors.BadRequest("INVALID_"+strings.ToUpper(field), "invalid "+field)
	}
	return value, nil
}

func parseAgentHistoryFilter(c *gin.Context, page, pageSize int) (service.AgentHistoryFilter, error) {
	filter := service.AgentHistoryFilter{
		Page:     page,
		PageSize: pageSize,
	}
	if start := strings.TrimSpace(c.Query("start_date")); start != "" {
		parsed, err := time.Parse("2006-01-02", start)
		if err != nil {
			return filter, infraerrors.BadRequest("INVALID_START_DATE", "invalid start_date")
		}
		filter.StartAt = &parsed
	}
	if end := strings.TrimSpace(c.Query("end_date")); end != "" {
		parsed, err := time.Parse("2006-01-02", end)
		if err != nil {
			return filter, infraerrors.BadRequest("INVALID_END_DATE", "invalid end_date")
		}
		endOfDay := parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		filter.EndAt = &endOfDay
	}
	return filter, nil
}

func parseManagedAgentRankingFilter(c *gin.Context, page, pageSize int) (service.AgentRankingFilter, error) {
	filter := service.AgentRankingFilter{
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
	}
	if raw := strings.TrimSpace(c.Query("date")); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return filter, infraerrors.BadRequest("INVALID_DATE", "invalid date")
		}
		filter.StatDate = &parsed
	}
	return filter, nil
}

func (h *UserHandler) requireManagedPermission(c *gin.Context, userID int64) (*service.AgentDistributionPermission, error) {
	permissions, err := h.affiliateService.GetAgentDistributionPermissions(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	if permissions == nil {
		return nil, infraerrors.Forbidden("DOWNLINE_MANAGEMENT_FORBIDDEN", "downline management access required")
	}
	return permissions, nil
}
