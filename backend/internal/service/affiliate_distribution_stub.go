package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrAffiliateDistributionUnavailable = infraerrors.ServiceUnavailable("AFFILIATE_DISTRIBUTION_UNAVAILABLE", "affiliate distribution service unavailable")

type AgentModelRate struct {
	Model      string  `json:"model"`
	Multiplier float64 `json:"multiplier"`
}

type AgentModelRateInput struct {
	Model      string  `json:"model"`
	Multiplier float64 `json:"multiplier"`
}

type AgentDistributionOverview struct {
	UserID                  int64            `json:"user_id"`
	InviteCode              string           `json:"invite_code"`
	InviterID               *int64           `json:"inviter_id,omitempty"`
	IsAdmin                 bool             `json:"is_admin"`
	DirectMemberCount       int              `json:"direct_member_count"`
	TodayBusinessUSD        float64          `json:"today_business_usd"`
	TodayRebateRMB          float64          `json:"today_rebate_rmb"`
	CurrentRebateBalanceRMB float64          `json:"current_rebate_balance_rmb"`
	InviteModelRates        []AgentModelRate `json:"invite_model_rates"`
	CanEditSubordinates     bool             `json:"can_edit_subordinates"`
	CanAdjustOwnRebate      bool             `json:"can_adjust_own_rebate"`
	MonthlyResetDayOfUTC    int              `json:"monthly_reset_day_of_utc"`
}

type AgentDirectMember struct {
	UserID                  int64            `json:"user_id"`
	Email                   string           `json:"email"`
	Username                string           `json:"username"`
	IsAgent                 bool             `json:"is_agent"`
	CreatedAt               *time.Time       `json:"created_at,omitempty"`
	TodayBusinessUSD        float64          `json:"today_business_usd"`
	TodayRebateRMB          float64          `json:"today_rebate_rmb"`
	CurrentRebateBalanceRMB float64          `json:"current_rebate_balance_rmb"`
	CurrentModelRates       []AgentModelRate `json:"current_model_rates"`
	ParentCanEditRates      bool             `json:"parent_can_edit_rates"`
}

type AgentHistoryFilter struct {
	Page     int
	PageSize int
	StartAt  *time.Time
	EndAt    *time.Time
}

type AgentHistoryItem struct {
	StatDate                string    `json:"stat_date"`
	BusinessUSD             float64   `json:"business_usd"`
	RebateRMB               float64   `json:"rebate_rmb"`
	DirectUsers             int       `json:"direct_users"`
	DirectAgents            int       `json:"direct_agents"`
	LastCalculated          time.Time `json:"last_calculated_at"`
	PeriodLabel             string    `json:"period_label,omitempty"`
	CurrentRebateBalanceRMB float64   `json:"current_rebate_balance_rmb,omitempty"`
}

type AgentRankingFilter struct {
	Page            int
	PageSize        int
	StatDate        *time.Time
	Search          string
	RootUserID      *int64
	OnlyDescendants bool
}

type AgentDailyBusinessRankingItem struct {
	Rank             int       `json:"rank"`
	UserID           int64     `json:"user_id"`
	Email            string    `json:"email"`
	Username         string    `json:"username"`
	StatDate         string    `json:"stat_date"`
	BusinessUSD      float64   `json:"business_usd"`
	BusinessRMB      float64   `json:"business_rmb"`
	DirectUsers      int       `json:"direct_users"`
	DirectAgents     int       `json:"direct_agents"`
	LastCalculatedAt time.Time `json:"last_calculated_at"`
}

type AgentRebateBalanceRankingItem struct {
	Rank                    int       `json:"rank"`
	UserID                  int64     `json:"user_id"`
	Email                   string    `json:"email"`
	Username                string    `json:"username"`
	CurrentRebateBalanceRMB float64   `json:"current_rebate_balance_rmb"`
	TodayRebateRMB          float64   `json:"today_rebate_rmb"`
	MonthlyRebateRMB        float64   `json:"monthly_rebate_rmb"`
	DirectUsers             int       `json:"direct_users"`
	DirectAgents            int       `json:"direct_agents"`
	LastAdjustedAt          time.Time `json:"last_adjusted_at"`
	LastAdjustmentNote      string    `json:"last_adjustment_note,omitempty"`
}

type AgentRebateBalanceAdjustment struct {
	UserID             int64     `json:"user_id"`
	OperatorUserID     int64     `json:"operator_user_id"`
	PreviousBalanceRMB float64   `json:"previous_balance_rmb"`
	NewBalanceRMB      float64   `json:"new_balance_rmb"`
	Note               string    `json:"note"`
	AdjustedAt         time.Time `json:"adjusted_at"`
}

type AgentDistributionPermission struct {
	UserID                        int64      `json:"user_id"`
	CanViewDownlineDailyRevenue   bool       `json:"can_view_downline_daily_revenue"`
	CanViewDownlineRebateBalances bool       `json:"can_view_downline_rebate_balances"`
	CanManageDownlinePricing      bool       `json:"can_manage_downline_pricing"`
	GrantedByUserID               *int64     `json:"granted_by_user_id,omitempty"`
	CreatedAt                     *time.Time `json:"created_at,omitempty"`
	UpdatedAt                     *time.Time `json:"updated_at,omitempty"`
}

type UpdateAgentDistributionPermissionInput struct {
	CanViewDownlineDailyRevenue   bool `json:"can_view_downline_daily_revenue"`
	CanViewDownlineRebateBalances bool `json:"can_view_downline_rebate_balances"`
	CanManageDownlinePricing      bool `json:"can_manage_downline_pricing"`
}

type AgentTreeFilter struct {
	RootUserID      *int64
	Search          string
	OnlyDescendants bool
}

type AgentTreeNode struct {
	UserID                  int64            `json:"user_id"`
	InviterID               *int64           `json:"inviter_id,omitempty"`
	Email                   string           `json:"email"`
	Username                string           `json:"username"`
	InviteCode              string           `json:"invite_code"`
	Depth                   int              `json:"depth"`
	IsAdmin                 bool             `json:"is_admin"`
	CurrentRebateBalanceRMB float64          `json:"current_rebate_balance_rmb"`
	InviteModelRates        []AgentModelRate `json:"invite_model_rates,omitempty"`
	CurrentModelRates       []AgentModelRate `json:"current_model_rates,omitempty"`
}

type AgentMonthlyArchiveFilter struct {
	Page     int
	PageSize int
	Month    string
	Search   string
}

type AgentMonthlyArchiveItem struct {
	UserID            int64     `json:"user_id"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	Month             string    `json:"month"`
	ArchivedRebateRMB float64   `json:"archived_rebate_rmb"`
	ArchivedAt        time.Time `json:"archived_at"`
}

type AffiliateDistributionRepository interface {
	GetDistributionOverview(ctx context.Context, userID int64) (*AgentDistributionOverview, error)
	ListInviteModelRates(ctx context.Context, userID int64) ([]AgentModelRate, error)
	SaveInviteModelRates(ctx context.Context, userID int64, rates []AgentModelRateInput) ([]AgentModelRate, error)
	ListDirectSubordinates(ctx context.Context, userID int64) ([]AgentDirectMember, error)
	UpdateDirectSubordinateModelRates(ctx context.Context, userID, subordinateUserID int64, rates []AgentModelRateInput) ([]AgentModelRate, error)
	ListUserDistributionHistory(ctx context.Context, userID int64, filter AgentHistoryFilter) ([]AgentHistoryItem, int64, error)
	ListDailyBusinessRanking(ctx context.Context, filter AgentRankingFilter) ([]AgentDailyBusinessRankingItem, int64, error)
	ListRebateBalanceRanking(ctx context.Context, filter AgentRankingFilter) ([]AgentRebateBalanceRankingItem, int64, error)
	GetAgentDistributionPermissions(ctx context.Context, userID int64) (*AgentDistributionPermission, error)
	UpdateAgentDistributionPermissions(ctx context.Context, operatorUserID, userID int64, input UpdateAgentDistributionPermissionInput) (*AgentDistributionPermission, error)
	AdminSetRebateBalance(ctx context.Context, operatorUserID, userID int64, amount float64, note string) (*AgentRebateBalanceAdjustment, error)
	GetDistributionTree(ctx context.Context, filter AgentTreeFilter) ([]AgentTreeNode, error)
	GetUserDistributionPricing(ctx context.Context, userID int64) ([]AgentModelRate, error)
	AdminUpdateUserDistributionPricing(ctx context.Context, operatorUserID, userID int64, rates []AgentModelRateInput) ([]AgentModelRate, error)
	ListMonthlyRebateArchives(ctx context.Context, filter AgentMonthlyArchiveFilter) ([]AgentMonthlyArchiveItem, int64, error)
	ArchiveMonthlyRebateBalances(ctx context.Context, archiveMonth time.Time, operatorUserID *int64, operatorName string) (int64, error)
}

type affiliateDistributionScopedRepository interface {
	ListDailyBusinessRankingScoped(ctx context.Context, operatorUserID int64, filter AgentRankingFilter) ([]AgentDailyBusinessRankingItem, int64, error)
	ListRebateBalanceRankingScoped(ctx context.Context, operatorUserID int64, filter AgentRankingFilter) ([]AgentRebateBalanceRankingItem, int64, error)
	GetDistributionTreeScoped(ctx context.Context, operatorUserID int64, filter AgentTreeFilter) ([]AgentTreeNode, error)
	GetUserDistributionPricingScoped(ctx context.Context, operatorUserID, userID int64) ([]AgentModelRate, error)
	UpdateUserDistributionPricingScoped(ctx context.Context, operatorUserID, userID int64, rates []AgentModelRateInput) ([]AgentModelRate, error)
}

func (s *AffiliateService) distributionRepo() (AffiliateDistributionRepository, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAffiliateDistributionUnavailable
	}
	repo, ok := s.repo.(AffiliateDistributionRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	return repo, nil
}

func (s *AffiliateService) GetDistributionOverview(ctx context.Context, userID int64) (*AgentDistributionOverview, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetDistributionOverview(ctx, userID)
}

func (s *AffiliateService) ListInviteModelRates(ctx context.Context, userID int64) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.ListInviteModelRates(ctx, userID)
}

func (s *AffiliateService) SaveInviteModelRates(ctx context.Context, userID int64, rates []AgentModelRateInput) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.SaveInviteModelRates(ctx, userID, normalizeAgentModelRateInputs(rates))
}

func (s *AffiliateService) ListDirectSubordinates(ctx context.Context, userID int64) ([]AgentDirectMember, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.ListDirectSubordinates(ctx, userID)
}

func (s *AffiliateService) UpdateDirectSubordinateModelRates(ctx context.Context, userID, subordinateUserID int64, rates []AgentModelRateInput) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.UpdateDirectSubordinateModelRates(ctx, userID, subordinateUserID, normalizeAgentModelRateInputs(rates))
}

func (s *AffiliateService) ListUserDistributionHistory(ctx context.Context, userID int64, filter AgentHistoryFilter) ([]AgentHistoryItem, int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, 0, err
	}
	return repo.ListUserDistributionHistory(ctx, userID, filter)
}

func (s *AffiliateService) ListDailyBusinessRanking(ctx context.Context, filter AgentRankingFilter) ([]AgentDailyBusinessRankingItem, int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, 0, err
	}
	return repo.ListDailyBusinessRanking(ctx, filter)
}

func (s *AffiliateService) ListDailyBusinessRankingScoped(ctx context.Context, operatorUserID int64, filter AgentRankingFilter) ([]AgentDailyBusinessRankingItem, int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, 0, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, 0, ErrAffiliateDistributionUnavailable
	}
	return scopedRepo.ListDailyBusinessRankingScoped(ctx, operatorUserID, filter)
}

func (s *AffiliateService) ListRebateBalanceRanking(ctx context.Context, filter AgentRankingFilter) ([]AgentRebateBalanceRankingItem, int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, 0, err
	}
	return repo.ListRebateBalanceRanking(ctx, filter)
}

func (s *AffiliateService) ListRebateBalanceRankingScoped(ctx context.Context, operatorUserID int64, filter AgentRankingFilter) ([]AgentRebateBalanceRankingItem, int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, 0, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, 0, ErrAffiliateDistributionUnavailable
	}
	return scopedRepo.ListRebateBalanceRankingScoped(ctx, operatorUserID, filter)
}

func (s *AffiliateService) GetAgentDistributionPermissions(ctx context.Context, userID int64) (*AgentDistributionPermission, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetAgentDistributionPermissions(ctx, userID)
}

func (s *AffiliateService) UpdateAgentDistributionPermissions(ctx context.Context, operatorUserID, userID int64, input UpdateAgentDistributionPermissionInput) (*AgentDistributionPermission, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.UpdateAgentDistributionPermissions(ctx, operatorUserID, userID, input)
}

func (s *AffiliateService) AdminSetRebateBalance(ctx context.Context, operatorUserID, userID int64, amount float64, note string) (*AgentRebateBalanceAdjustment, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.AdminSetRebateBalance(ctx, operatorUserID, userID, amount, strings.TrimSpace(note))
}

func (s *AffiliateService) GetDistributionTree(ctx context.Context, filter AgentTreeFilter) ([]AgentTreeNode, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetDistributionTree(ctx, filter)
}

func (s *AffiliateService) GetDistributionTreeScoped(ctx context.Context, operatorUserID int64, filter AgentTreeFilter) ([]AgentTreeNode, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	return scopedRepo.GetDistributionTreeScoped(ctx, operatorUserID, filter)
}

func (s *AffiliateService) GetUserDistributionPricing(ctx context.Context, userID int64) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetUserDistributionPricing(ctx, userID)
}

func (s *AffiliateService) GetUserDistributionPricingScoped(ctx context.Context, operatorUserID, userID int64) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	return scopedRepo.GetUserDistributionPricingScoped(ctx, operatorUserID, userID)
}

func (s *AffiliateService) AdminUpdateUserDistributionPricing(ctx context.Context, operatorUserID, userID int64, rates []AgentModelRateInput) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	return repo.AdminUpdateUserDistributionPricing(ctx, operatorUserID, userID, normalizeAgentModelRateInputs(rates))
}

func (s *AffiliateService) UpdateUserDistributionPricingScoped(ctx context.Context, operatorUserID, userID int64, rates []AgentModelRateInput) ([]AgentModelRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	return scopedRepo.UpdateUserDistributionPricingScoped(ctx, operatorUserID, userID, normalizeAgentModelRateInputs(rates))
}

func (s *AffiliateService) ListMonthlyRebateArchives(ctx context.Context, filter AgentMonthlyArchiveFilter) ([]AgentMonthlyArchiveItem, int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, 0, err
	}
	return repo.ListMonthlyRebateArchives(ctx, filter)
}

func (s *AffiliateService) ArchiveMonthlyRebateBalances(ctx context.Context, archiveMonth time.Time, operatorUserID *int64, operatorName string) (int64, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return 0, err
	}
	return repo.ArchiveMonthlyRebateBalances(ctx, archiveMonth, operatorUserID, operatorName)
}

func normalizeAgentModelRateInputs(rates []AgentModelRateInput) []AgentModelRateInput {
	if len(rates) == 0 {
		return []AgentModelRateInput{}
	}
	result := make([]AgentModelRateInput, 0, len(rates))
	for _, rate := range rates {
		model := strings.TrimSpace(rate.Model)
		if model == "" {
			continue
		}
		result = append(result, AgentModelRateInput{
			Model:      model,
			Multiplier: rate.Multiplier,
		})
	}
	return result
}
