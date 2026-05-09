package admin

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type adminModelRatePayload struct {
	Model      string  `json:"model"`
	Multiplier float64 `json:"multiplier"`
}

type adminUpdateModelRatesRequest struct {
	ModelRates []adminModelRatePayload `json:"model_rates" binding:"required"`
}

type updateAgentDistributionPermissionsRequest struct {
	CanViewDownlineDailyRevenue   bool `json:"can_view_downline_daily_revenue"`
	CanViewDownlineRebateBalances bool `json:"can_view_downline_rebate_balances"`
	CanManageDownlinePricing      bool `json:"can_manage_downline_pricing"`
}

type setRebateBalanceRequest struct {
	UserID int64   `json:"user_id"`
	Amount float64 `json:"amount" binding:"required"`
	Note   string  `json:"note" binding:"required"`
}

// ListDailyBusinessRanking returns ranked daily business totals for agents.
// GET /api/v1/admin/affiliates/daily-business
func (h *AffiliateHandler) ListDailyBusinessRanking(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter, err := parseAgentRankingFilter(c, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items, total, err := h.affiliateService.ListDailyBusinessRanking(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// ListRebateBalanceRanking returns ranked current rebate balances.
// GET /api/v1/admin/affiliates/rebate-balances
func (h *AffiliateHandler) ListRebateBalanceRanking(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter, err := parseAgentRankingFilter(c, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items, total, err := h.affiliateService.ListRebateBalanceRanking(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// GetAgentDistributionPermissions returns one agent's downline-management permissions.
// GET /api/v1/admin/affiliates/users/:user_id/permissions
func (h *AffiliateHandler) GetAgentDistributionPermissions(c *gin.Context) {
	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	permissions, err := h.affiliateService.GetAgentDistributionPermissions(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, permissions)
}

// UpdateAgentDistributionPermissions updates one agent's downline-management permissions.
// PUT /api/v1/admin/affiliates/users/:user_id/permissions
func (h *AffiliateHandler) UpdateAgentDistributionPermissions(c *gin.Context) {
	operator, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req updateAgentDistributionPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REQUEST", "invalid request: "+err.Error()))
		return
	}

	permissions, err := h.affiliateService.UpdateAgentDistributionPermissions(c.Request.Context(), operator.UserID, userID, service.UpdateAgentDistributionPermissionInput{
		CanViewDownlineDailyRevenue:   req.CanViewDownlineDailyRevenue,
		CanViewDownlineRebateBalances: req.CanViewDownlineRebateBalances,
		CanManageDownlinePricing:      req.CanManageDownlinePricing,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, permissions)
}

// SetRebateBalance directly sets an agent's current rebate balance.
// PUT /api/v1/admin/affiliates/rebate-balances/:user_id
func (h *AffiliateHandler) SetRebateBalance(c *gin.Context) {
	h.setRebateBalance(c, c.Param("user_id"))
}

// SetRebateBalanceByBody directly sets an agent's current rebate balance using user_id in JSON.
func (h *AffiliateHandler) SetRebateBalanceByBody(c *gin.Context) {
	h.setRebateBalance(c, "")
}

func (h *AffiliateHandler) setRebateBalance(c *gin.Context, rawUserID string) {
	operator, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.requireRootAdmin(c.Request.Context(), operator.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req setRebateBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	var userID int64
	var err error
	if strings.TrimSpace(rawUserID) != "" {
		userID, err = parseAdminPositiveInt64(rawUserID, "user_id")
	} else if req.UserID > 0 {
		userID = req.UserID
	} else {
		err = infraerrors.BadRequest("INVALID_USER_ID", "invalid user_id")
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if strings.TrimSpace(req.Note) == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_NOTE", "note is required"))
		return
	}
	if math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) || req.Amount < 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_AMOUNT", "amount must be greater than or equal to 0"))
		return
	}

	result, err := h.affiliateService.AdminSetRebateBalance(c.Request.Context(), operator.UserID, userID, req.Amount, req.Note)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetDistributionTree returns the affiliate hierarchy for admin.
// GET /api/v1/admin/affiliates/tree
func (h *AffiliateHandler) GetDistributionTree(c *gin.Context) {
	filter := service.AgentTreeFilter{
		Search: strings.TrimSpace(c.Query("search")),
	}
	if root := strings.TrimSpace(c.Query("root_user_id")); root != "" {
		value, err := parseAdminPositiveInt64(root, "root_user_id")
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		filter.RootUserID = &value
	}

	items, err := h.affiliateService.GetDistributionTree(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

// GetUserDistributionPricing returns a target user's model pricing.
// GET /api/v1/admin/affiliates/users/:user_id/pricing
func (h *AffiliateHandler) GetUserDistributionPricing(c *gin.Context) {
	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	rates, err := h.affiliateService.GetUserDistributionPricing(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "model_rates": rates})
}

// UpdateUserDistributionPricing updates a target user's model pricing.
// PUT /api/v1/admin/affiliates/users/:user_id/pricing
func (h *AffiliateHandler) UpdateUserDistributionPricing(c *gin.Context) {
	operator, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rates, err := bindAdminModelRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.AdminUpdateUserDistributionPricing(c.Request.Context(), operator.UserID, userID, rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "model_rates": updated})
}

// ListMonthlyRebateArchives returns archived monthly rebate balances.
// GET /api/v1/admin/affiliates/monthly-archives
func (h *AffiliateHandler) ListMonthlyRebateArchives(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.AgentMonthlyArchiveFilter{
		Page:     page,
		PageSize: pageSize,
		Month:    strings.TrimSpace(c.Query("month")),
		Search:   strings.TrimSpace(c.Query("search")),
	}

	items, total, err := h.affiliateService.ListMonthlyRebateArchives(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func bindAdminModelRatesRequest(c *gin.Context) ([]service.AgentModelRateInput, error) {
	var req adminUpdateModelRatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, infraerrors.BadRequest("INVALID_REQUEST", "invalid request: "+err.Error())
	}
	if len(req.ModelRates) == 0 {
		return nil, infraerrors.BadRequest("INVALID_MODEL_RATES", "model_rates cannot be empty")
	}

	rates := make([]service.AgentModelRateInput, 0, len(req.ModelRates))
	seen := make(map[string]struct{}, len(req.ModelRates))
	for _, item := range req.ModelRates {
		model := strings.TrimSpace(item.Model)
		if model == "" {
			return nil, infraerrors.BadRequest("INVALID_MODEL_RATES", "model is required")
		}
		if _, exists := seen[model]; exists {
			return nil, infraerrors.BadRequest("INVALID_MODEL_RATES", "duplicate model: "+model)
		}
		if item.Multiplier <= 0 || math.IsNaN(item.Multiplier) || math.IsInf(item.Multiplier, 0) {
			return nil, infraerrors.BadRequest("INVALID_MODEL_RATES", "multiplier must be greater than 0")
		}
		seen[model] = struct{}{}
		rates = append(rates, service.AgentModelRateInput{
			Model:      model,
			Multiplier: item.Multiplier,
		})
	}
	return rates, nil
}

func parseAgentRankingFilter(c *gin.Context, page, pageSize int) (service.AgentRankingFilter, error) {
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

func parseAdminPositiveInt64(raw, field string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, infraerrors.BadRequest("INVALID_"+strings.ToUpper(field), "invalid "+field)
	}
	return value, nil
}

func (h *AffiliateHandler) requireRootAdmin(ctx context.Context, operatorUserID int64) error {
	if h.firstAdminLookup == nil {
		return infraerrors.Forbidden("ROOT_ADMIN_REQUIRED", "root admin access required")
	}
	firstAdmin, err := h.firstAdminLookup.GetFirstAdmin(ctx)
	if err != nil {
		return err
	}
	if firstAdmin == nil || firstAdmin.ID <= 0 || firstAdmin.ID != operatorUserID {
		return infraerrors.Forbidden("ROOT_ADMIN_REQUIRED", "root admin access required")
	}
	return nil
}
