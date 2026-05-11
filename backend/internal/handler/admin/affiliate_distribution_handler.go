package admin

import (
	"encoding/json"
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

type adminGroupRatePayload struct {
	GroupID        int64   `json:"group_id"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type adminUpdateGroupRatesRequest struct {
	GroupRates []adminGroupRatePayload `json:"group_rates" binding:"required"`
}

type updateUserUpstreamRequest struct {
	UpstreamUserID    *int64 `json:"-"`
	InviterID         *int64 `json:"-"`
	ClearUpstream     *bool  `json:"-"`
	hasUpstreamUserID bool
	hasInviterID      bool
	hasClearUpstream  bool
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

func (r *updateUserUpstreamRequest) UnmarshalJSON(data []byte) error {
	type rawUpdateUserUpstreamRequest struct {
		UpstreamUserID json.RawMessage `json:"upstream_user_id"`
		InviterID      json.RawMessage `json:"inviter_id"`
		ClearUpstream  json.RawMessage `json:"clear_upstream"`
	}

	var raw rawUpdateUserUpstreamRequest
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = updateUserUpstreamRequest{}

	if raw.UpstreamUserID != nil {
		r.hasUpstreamUserID = true
		value, err := decodeOptionalInt64(raw.UpstreamUserID, "upstream_user_id")
		if err != nil {
			return err
		}
		r.UpstreamUserID = value
	}

	if raw.InviterID != nil {
		r.hasInviterID = true
		value, err := decodeOptionalInt64(raw.InviterID, "inviter_id")
		if err != nil {
			return err
		}
		r.InviterID = value
	}

	if raw.ClearUpstream != nil {
		r.hasClearUpstream = true
		var clear bool
		if err := json.Unmarshal(raw.ClearUpstream, &clear); err != nil {
			return err
		}
		r.ClearUpstream = &clear
	}

	return nil
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
	if err := h.requireRootAdmin(c, operator.UserID); err != nil {
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

// GetUserDistributionGroupRates returns a target user's current group pricing.
// GET /api/v1/admin/affiliates/users/:user_id/pricing
func (h *AffiliateHandler) GetUserDistributionGroupRates(c *gin.Context) {
	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	rates, err := h.affiliateService.GetUserDistributionGroupRates(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "group_rates": rates})
}

// UpdateUserDistributionPricing updates a target user's current group pricing.
// PUT /api/v1/admin/affiliates/users/:user_id/pricing
func (h *AffiliateHandler) UpdateUserDistributionGroupRates(c *gin.Context) {
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
	rates, err := bindAdminGroupRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.SaveUserCurrentGroupRates(c.Request.Context(), operator.UserID, userID, rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "group_rates": updated})
}

// GetDefaultDistributionPricing returns the default explicit group pricing for users without an affiliate code.
// GET /api/v1/admin/affiliates/default-pricing
func (h *AffiliateHandler) GetDefaultDistributionGroupRates(c *gin.Context) {
	rates, err := h.affiliateService.ListDefaultUserGroupRates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_rates": rates})
}

// UpdateDefaultDistributionPricing updates the default explicit group pricing for users without an affiliate code.
// PUT /api/v1/admin/affiliates/default-pricing
func (h *AffiliateHandler) UpdateDefaultDistributionGroupRates(c *gin.Context) {
	rates, err := bindAdminGroupRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.SaveDefaultUserGroupRates(c.Request.Context(), rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_rates": updated})
}

// GetUserInvitePricing returns the invite-code group pricing a user gives to new direct downstream members.
// GET /api/v1/admin/affiliates/users/:user_id/invite-pricing
func (h *AffiliateHandler) GetUserInviteGroupRates(c *gin.Context) {
	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	rates, err := h.affiliateService.ListUserInviteGroupRates(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "group_rates": rates})
}

// UpdateUserInvitePricing updates the invite-code group pricing a user gives to new direct downstream members.
// PUT /api/v1/admin/affiliates/users/:user_id/invite-pricing
func (h *AffiliateHandler) UpdateUserInviteGroupRates(c *gin.Context) {
	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rates, err := bindAdminGroupRatesRequest(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.SaveUserInviteGroupRates(c.Request.Context(), userID, rates)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "group_rates": updated})
}

// UpdateUserUpstream updates a user's upstream/inviter relationship.
// PUT /api/v1/admin/affiliates/users/:user_id/upstream
func (h *AffiliateHandler) UpdateUserUpstream(c *gin.Context) {
	operator, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := rejectAdminAPIKey(c, "ADMIN_JWT_REQUIRED", "admin JWT required for upstream updates"); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	userID, err := parseAdminPositiveInt64(c.Param("user_id"), "user_id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req updateUserUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REQUEST", "invalid request: "+err.Error()))
		return
	}

	upstreamUserID, err := resolveUpstreamUserID(req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.affiliateService.AdminUpdateUserUpstream(c.Request.Context(), operator.UserID, userID, upstreamUserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
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

func bindAdminGroupRatesRequest(c *gin.Context) ([]service.AgentGroupRateInput, error) {
	var req adminUpdateGroupRatesRequest
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

func decodeOptionalInt64(raw json.RawMessage, field string) (*int64, error) {
	if strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil, nil
	}

	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, infraerrors.BadRequest("INVALID_"+strings.ToUpper(field), "invalid "+field)
	}
	return &value, nil
}

func resolveUpstreamUserID(req updateUserUpstreamRequest) (*int64, error) {
	if req.hasUpstreamUserID && req.hasInviterID {
		return nil, infraerrors.BadRequest("AMBIGUOUS_UPSTREAM_REQUEST", "specify only one of upstream_user_id or inviter_id")
	}
	if req.hasClearUpstream {
		if req.ClearUpstream == nil {
			return nil, infraerrors.BadRequest("INVALID_CLEAR_UPSTREAM", "invalid clear_upstream")
		}
		if *req.ClearUpstream {
			if req.hasUpstreamUserID || req.hasInviterID {
				return nil, infraerrors.BadRequest("AMBIGUOUS_UPSTREAM_REQUEST", "cannot set upstream_user_id or inviter_id when clear_upstream is true")
			}
			return nil, nil
		}
		if !req.hasUpstreamUserID && !req.hasInviterID {
			return nil, infraerrors.BadRequest("EXPLICIT_UPSTREAM_REQUIRED", "request must explicitly set or clear upstream")
		}
	}
	if req.hasUpstreamUserID {
		if req.UpstreamUserID == nil {
			return nil, nil
		}
		if *req.UpstreamUserID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_UPSTREAM_USER_ID", "invalid upstream_user_id")
		}
		return req.UpstreamUserID, nil
	}
	if req.hasInviterID {
		if req.InviterID == nil {
			return nil, nil
		}
		if *req.InviterID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_INVITER_ID", "invalid inviter_id")
		}
		return req.InviterID, nil
	}
	return nil, infraerrors.BadRequest("EXPLICIT_UPSTREAM_REQUIRED", "request must explicitly set upstream_user_id, inviter_id, or clear_upstream")
}

func rejectAdminAPIKey(c *gin.Context, reason, message string) error {
	if c == nil {
		return nil
	}
	authMethod, exists := c.Get("auth_method")
	if !exists {
		return nil
	}
	method, ok := authMethod.(string)
	if ok && method == "admin_api_key" {
		return infraerrors.Forbidden(reason, message)
	}
	return nil
}

func (h *AffiliateHandler) requireRootAdmin(c *gin.Context, operatorUserID int64) error {
	if err := rejectAdminAPIKey(c, "ROOT_ADMIN_JWT_REQUIRED", "root admin JWT required for this operation"); err != nil {
		return err
	}
	if h.firstAdminLookup == nil {
		return infraerrors.Forbidden("ROOT_ADMIN_REQUIRED", "root admin access required")
	}
	firstAdmin, err := h.firstAdminLookup.GetFirstAdmin(c.Request.Context())
	if err != nil {
		return err
	}
	if firstAdmin == nil || firstAdmin.ID <= 0 || firstAdmin.ID != operatorUserID {
		return infraerrors.Forbidden("ROOT_ADMIN_REQUIRED", "root admin access required")
	}
	return nil
}
