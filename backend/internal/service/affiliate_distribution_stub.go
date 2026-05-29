package service

import (
	"context"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrAffiliateDistributionUnavailable = infraerrors.ServiceUnavailable("AFFILIATE_DISTRIBUTION_UNAVAILABLE", "affiliate distribution service unavailable")

const affiliateDistributionRateMultiplierMax = 100.0

type AgentGroupRate struct {
	GroupID                    int64      `json:"group_id"`
	GroupName                  string     `json:"group_name"`
	GroupPlatform              string     `json:"group_platform,omitempty"`
	GroupDefaultRateMultiplier float64    `json:"group_rate_multiplier"`
	RateMultiplier             float64    `json:"rate_multiplier"`
	SourceType                 string     `json:"source_type,omitempty"`
	SourceAffCode              string     `json:"source_aff_code,omitempty"`
	UpstreamUserID             *int64     `json:"upstream_user_id,omitempty"`
	UpdatedAt                  *time.Time `json:"updated_at,omitempty"`
}

type AgentGroupRateInput struct {
	GroupID        int64   `json:"group_id"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type AgentDistributionOverview struct {
	UserID                  int64               `json:"user_id"`
	InviteCode              string              `json:"invite_code"`
	InviterID               *int64              `json:"inviter_id,omitempty"`
	IsAdmin                 bool                `json:"is_admin"`
	IsRootAdmin             bool                `json:"is_root_admin"`
	DirectMemberCount       int                 `json:"direct_member_count"`
	DirectChildrenCount     int                 `json:"direct_children_count"`
	DirectChildren          []AgentDirectMember `json:"direct_children"`
	TodayBusinessUSD        float64             `json:"today_business_usd"`
	TodayRebateRMB          float64             `json:"today_rebate_rmb"`
	CurrentRebateBalanceRMB float64             `json:"current_rebate_balance_rmb"`
	InviteGroupRates        []AgentGroupRate    `json:"invite_group_rates"`
	CurrentGroupRates       []AgentGroupRate    `json:"current_group_rates"`
	MyGroupRates            []AgentGroupRate    `json:"my_group_rates,omitempty"`
	CanEditSubordinates     bool                `json:"can_edit_subordinates"`
	CanAdjustOwnRebate      bool                `json:"can_adjust_own_rebate"`
	MonthlyResetDayOfUTC    int                 `json:"monthly_reset_day_of_utc"`
}

type AgentDirectMember struct {
	UserID                  int64            `json:"user_id"`
	Email                   string           `json:"email"`
	Username                string           `json:"username"`
	IsAgent                 bool             `json:"is_agent"`
	CreatedAt               *time.Time       `json:"created_at,omitempty"`
	TodayBusinessUSD        float64          `json:"today_business_usd"`
	TodayBusinessRMB        float64          `json:"today_business_rmb"`
	TodayRebateRMB          float64          `json:"today_rebate_rmb"`
	DirectTotalUsageUSD     float64          `json:"direct_total_usage_usd"`
	DirectTotalUsageRMB     float64          `json:"direct_total_usage_rmb"`
	DirectUserUsageUSD      float64          `json:"direct_user_usage_usd"`
	DirectUserUsageRMB      float64          `json:"direct_user_usage_rmb"`
	DirectAgentUsageUSD     float64          `json:"direct_agent_usage_usd"`
	DirectAgentUsageRMB     float64          `json:"direct_agent_usage_rmb"`
	CurrentRebateBalanceRMB float64          `json:"current_rebate_balance_rmb"`
	CurrentGroupRates       []AgentGroupRate `json:"current_group_rates"`
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
	Rank                int       `json:"rank"`
	UserID              int64     `json:"user_id"`
	Email               string    `json:"email"`
	Username            string    `json:"username"`
	StatDate            string    `json:"stat_date"`
	BusinessUSD         float64   `json:"business_usd"`
	BusinessRMB         float64   `json:"business_rmb"`
	DirectUsers         int       `json:"direct_users"`
	DirectAgents        int       `json:"direct_agents"`
	DirectTotalUsageUSD float64   `json:"direct_total_usage_usd"`
	DirectTotalUsageRMB float64   `json:"direct_total_usage_rmb"`
	DirectUserUsageUSD  float64   `json:"direct_user_usage_usd"`
	DirectUserUsageRMB  float64   `json:"direct_user_usage_rmb"`
	DirectAgentUsageUSD float64   `json:"direct_agent_usage_usd"`
	DirectAgentUsageRMB float64   `json:"direct_agent_usage_rmb"`
	LastCalculatedAt    time.Time `json:"last_calculated_at"`
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
	DirectTotalUsageUSD     float64   `json:"direct_total_usage_usd"`
	DirectTotalUsageRMB     float64   `json:"direct_total_usage_rmb"`
	DirectUserUsageUSD      float64   `json:"direct_user_usage_usd"`
	DirectUserUsageRMB      float64   `json:"direct_user_usage_rmb"`
	DirectAgentUsageUSD     float64   `json:"direct_agent_usage_usd"`
	DirectAgentUsageRMB     float64   `json:"direct_agent_usage_rmb"`
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
	IsRootAdmin             bool             `json:"is_root_admin"`
	CurrentRebateBalanceRMB float64          `json:"current_rebate_balance_rmb"`
	TodayBusinessUSD        float64          `json:"today_business_usd"`
	TodayBusinessRMB        float64          `json:"today_business_rmb"`
	TodayRebateRMB          float64          `json:"today_rebate_rmb"`
	DirectChildrenCount     int              `json:"direct_children_count"`
	DirectUserCount         int              `json:"direct_user_count"`
	DirectAgentCount        int              `json:"direct_agent_count"`
	DirectTotalUsageUSD     float64          `json:"direct_total_usage_usd"`
	DirectTotalUsageRMB     float64          `json:"direct_total_usage_rmb"`
	DirectUserUsageUSD      float64          `json:"direct_user_usage_usd"`
	DirectUserUsageRMB      float64          `json:"direct_user_usage_rmb"`
	DirectAgentUsageUSD     float64          `json:"direct_agent_usage_usd"`
	DirectAgentUsageRMB     float64          `json:"direct_agent_usage_rmb"`
	InviteGroupRates        []AgentGroupRate `json:"invite_group_rates,omitempty"`
	CurrentGroupRates       []AgentGroupRate `json:"current_group_rates,omitempty"`
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

type AgentUserUpstream struct {
	UserID         int64      `json:"user_id"`
	UpstreamUserID *int64     `json:"upstream_user_id,omitempty"`
	InviterID      *int64     `json:"inviter_id,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type AffiliateDistributionRepository interface {
	GetDistributionOverview(ctx context.Context, userID int64) (*AgentDistributionOverview, error)
	ListInviteGroupRates(ctx context.Context, userID int64) ([]AgentGroupRate, error)
	SaveInviteGroupRates(ctx context.Context, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error)
	ListDirectSubordinates(ctx context.Context, userID int64) ([]AgentDirectMember, error)
	UpdateDirectSubordinateGroupRates(ctx context.Context, userID, subordinateUserID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error)
	ListUserDistributionHistory(ctx context.Context, userID int64, filter AgentHistoryFilter) ([]AgentHistoryItem, int64, error)
	ListDailyBusinessRanking(ctx context.Context, filter AgentRankingFilter) ([]AgentDailyBusinessRankingItem, int64, error)
	ListRebateBalanceRanking(ctx context.Context, filter AgentRankingFilter) ([]AgentRebateBalanceRankingItem, int64, error)
	GetAgentDistributionPermissions(ctx context.Context, userID int64) (*AgentDistributionPermission, error)
	UpdateAgentDistributionPermissions(ctx context.Context, operatorUserID, userID int64, input UpdateAgentDistributionPermissionInput) (*AgentDistributionPermission, error)
	AdminSetRebateBalance(ctx context.Context, operatorUserID, userID int64, amount float64, note string) (*AgentRebateBalanceAdjustment, error)
	GetDistributionTree(ctx context.Context, filter AgentTreeFilter) ([]AgentTreeNode, error)
	GetUserDistributionGroupRates(ctx context.Context, userID int64) ([]AgentGroupRate, error)
	AdminUpdateUserDistributionGroupRates(ctx context.Context, operatorUserID, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error)
	ListMonthlyRebateArchives(ctx context.Context, filter AgentMonthlyArchiveFilter) ([]AgentMonthlyArchiveItem, int64, error)
	ArchiveMonthlyRebateBalances(ctx context.Context, archiveMonth time.Time, operatorUserID *int64, operatorName string) (int64, error)
}

type affiliateDistributionScopedRepository interface {
	ListDailyBusinessRankingScoped(ctx context.Context, operatorUserID int64, filter AgentRankingFilter) ([]AgentDailyBusinessRankingItem, int64, error)
	ListRebateBalanceRankingScoped(ctx context.Context, operatorUserID int64, filter AgentRankingFilter) ([]AgentRebateBalanceRankingItem, int64, error)
	GetDistributionTreeScoped(ctx context.Context, operatorUserID int64, filter AgentTreeFilter) ([]AgentTreeNode, error)
	GetUserDistributionGroupRatesScoped(ctx context.Context, operatorUserID, userID int64) ([]AgentGroupRate, error)
	UpdateUserDistributionGroupRatesScoped(ctx context.Context, operatorUserID, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error)
}

type affiliateDistributionDefaultPricingRepository interface {
	ListDefaultUserGroupRates(ctx context.Context) ([]AgentGroupRate, error)
	SaveDefaultUserGroupRates(ctx context.Context, rates []AgentGroupRateInput) ([]AgentGroupRate, error)
}

type affiliateDistributionUpstreamRepository interface {
	AdminUpdateUserUpstream(ctx context.Context, operatorUserID, userID int64, upstreamUserID *int64) (*AgentUserUpstream, error)
}

func (s *AffiliateService) RecordDistributionPaidCredit(ctx context.Context, userID, sourceOrderID int64, amountUSD float64, creditedAt time.Time) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrAffiliateDistributionUnavailable
	}
	if userID <= 0 || sourceOrderID <= 0 || amountUSD <= 0 || math.IsNaN(amountUSD) || math.IsInf(amountUSD, 0) {
		return false, ErrAffiliateDistributionPaidCreditInvalidInput
	}
	repo, ok := s.repo.(AffiliateDistributionPaidCreditRecorder)
	if !ok {
		return false, ErrAffiliateDistributionUnavailable
	}
	return repo.RecordPaidCredit(ctx, userID, sourceOrderID, amountUSD, creditedAt)
}

func (s *AffiliateService) ReverseDistributionPaidCredit(ctx context.Context, userID, sourceOrderID int64, amountUSD float64, reversedAt time.Time) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrAffiliateDistributionUnavailable
	}
	if userID <= 0 || sourceOrderID <= 0 || amountUSD <= 0 || math.IsNaN(amountUSD) || math.IsInf(amountUSD, 0) {
		return false, ErrAffiliateDistributionPaidCreditInvalidInput
	}
	repo, ok := s.repo.(AffiliateDistributionPaidCreditRecorder)
	if !ok {
		return false, ErrAffiliateDistributionUnavailable
	}
	return repo.ReversePaidCredit(ctx, userID, sourceOrderID, amountUSD, reversedAt)
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
	overview, err := repo.GetDistributionOverview(ctx, userID)
	if err != nil || overview == nil {
		return overview, err
	}

	directChildren, err := repo.ListDirectSubordinates(ctx, userID)
	if err != nil {
		return nil, err
	}
	return normalizeAgentDistributionOverview(overview, directChildren), nil
}

func (s *AffiliateService) ListInviteGroupRates(ctx context.Context, userID int64) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	rates, err := repo.ListInviteGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(rates), nil
}

func (s *AffiliateService) SaveInviteGroupRates(ctx context.Context, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	normalized, err := validateAndNormalizeAgentGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	updated, err := repo.SaveInviteGroupRates(ctx, userID, normalized)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(updated), nil
}

func (s *AffiliateService) ListUserInviteGroupRates(ctx context.Context, userID int64) ([]AgentGroupRate, error) {
	return s.ListInviteGroupRates(ctx, userID)
}

func (s *AffiliateService) SaveUserInviteGroupRates(ctx context.Context, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	return s.SaveInviteGroupRates(ctx, userID, rates)
}

func (s *AffiliateService) ListDirectSubordinates(ctx context.Context, userID int64) ([]AgentDirectMember, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	members, err := repo.ListDirectSubordinates(ctx, userID)
	if err != nil {
		return nil, err
	}
	return normalizeAgentDirectMembers(members), nil
}

func (s *AffiliateService) UpdateDirectSubordinateGroupRates(ctx context.Context, userID, subordinateUserID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	normalized, err := validateAndNormalizeAgentGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	updated, err := repo.UpdateDirectSubordinateGroupRates(ctx, userID, subordinateUserID, normalized)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(updated), nil
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

func (s *AffiliateService) GetUserDistributionGroupRates(ctx context.Context, userID int64) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	rates, err := repo.GetUserDistributionGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(rates), nil
}

func (s *AffiliateService) GetUserCurrentGroupRates(ctx context.Context, userID int64) ([]AgentGroupRate, error) {
	return s.GetUserDistributionGroupRates(ctx, userID)
}

func (s *AffiliateService) GetUserDistributionGroupRatesScoped(ctx context.Context, operatorUserID, userID int64) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	rates, err := scopedRepo.GetUserDistributionGroupRatesScoped(ctx, operatorUserID, userID)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(rates), nil
}

func (s *AffiliateService) AdminUpdateUserDistributionGroupRates(ctx context.Context, operatorUserID, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	normalized, err := validateAndNormalizeAgentGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	updated, err := repo.AdminUpdateUserDistributionGroupRates(ctx, operatorUserID, userID, normalized)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(updated), nil
}

func (s *AffiliateService) SaveUserCurrentGroupRates(ctx context.Context, operatorUserID, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	return s.AdminUpdateUserDistributionGroupRates(ctx, operatorUserID, userID, rates)
}

func (s *AffiliateService) UpdateUserDistributionGroupRatesScoped(ctx context.Context, operatorUserID, userID int64, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	repo, err := s.distributionRepo()
	if err != nil {
		return nil, err
	}
	scopedRepo, ok := repo.(affiliateDistributionScopedRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	normalized, err := validateAndNormalizeAgentGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	updated, err := scopedRepo.UpdateUserDistributionGroupRatesScoped(ctx, operatorUserID, userID, normalized)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(updated), nil
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

func (s *AffiliateService) ListDefaultUserGroupRates(ctx context.Context) ([]AgentGroupRate, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAffiliateDistributionUnavailable
	}
	repo, ok := s.repo.(affiliateDistributionDefaultPricingRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	rates, err := repo.ListDefaultUserGroupRates(ctx)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(rates), nil
}

func (s *AffiliateService) SaveDefaultUserGroupRates(ctx context.Context, rates []AgentGroupRateInput) ([]AgentGroupRate, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAffiliateDistributionUnavailable
	}
	repo, ok := s.repo.(affiliateDistributionDefaultPricingRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	normalized, err := validateAndNormalizeAgentGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	updated, err := repo.SaveDefaultUserGroupRates(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return normalizeAgentGroupRates(updated), nil
}

func validateAndNormalizeAgentGroupRateInputs(rates []AgentGroupRateInput) ([]AgentGroupRateInput, error) {
	if len(rates) == 0 {
		return []AgentGroupRateInput{}, nil
	}
	result := make([]AgentGroupRateInput, 0, len(rates))
	for _, rate := range rates {
		if rate.GroupID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_GROUP_RATES", "group_id must be greater than 0")
		}
		if err := validateAffiliateDistributionRateMultiplier(rate.RateMultiplier); err != nil {
			return nil, err
		}
		result = append(result, AgentGroupRateInput{
			GroupID:        rate.GroupID,
			RateMultiplier: rate.RateMultiplier,
		})
	}
	return result, nil
}

func validateAffiliateDistributionRateMultiplier(multiplier float64) error {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 || multiplier > affiliateDistributionRateMultiplierMax {
		return infraerrors.BadRequest("INVALID_GROUP_RATES", "rate_multiplier must be a finite number greater than 0 and at most 100")
	}
	return nil
}

func (s *AffiliateService) AdminUpdateUserUpstream(ctx context.Context, operatorUserID, userID int64, upstreamUserID *int64) (*AgentUserUpstream, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAffiliateDistributionUnavailable
	}
	repo, ok := s.repo.(affiliateDistributionUpstreamRepository)
	if !ok {
		return nil, ErrAffiliateDistributionUnavailable
	}
	return repo.AdminUpdateUserUpstream(ctx, operatorUserID, userID, upstreamUserID)
}

func normalizeAgentGroupRateInputs(rates []AgentGroupRateInput) []AgentGroupRateInput {
	if len(rates) == 0 {
		return []AgentGroupRateInput{}
	}
	result := make([]AgentGroupRateInput, 0, len(rates))
	for _, rate := range rates {
		if rate.GroupID <= 0 {
			continue
		}
		result = append(result, AgentGroupRateInput{
			GroupID:        rate.GroupID,
			RateMultiplier: rate.RateMultiplier,
		})
	}
	return result
}

func normalizeAgentGroupRates(rates []AgentGroupRate) []AgentGroupRate {
	if len(rates) == 0 {
		return []AgentGroupRate{}
	}
	return append([]AgentGroupRate(nil), rates...)
}

func normalizeAgentDirectMembers(members []AgentDirectMember) []AgentDirectMember {
	if len(members) == 0 {
		return []AgentDirectMember{}
	}

	result := append([]AgentDirectMember(nil), members...)
	for index := range result {
		result[index].CurrentGroupRates = normalizeAgentGroupRates(result[index].CurrentGroupRates)
	}
	return result
}

func normalizeAgentDistributionOverview(overview *AgentDistributionOverview, directChildren []AgentDirectMember) *AgentDistributionOverview {
	if overview == nil {
		return nil
	}

	result := *overview
	result.InviteGroupRates = normalizeAgentGroupRates(result.InviteGroupRates)
	result.CurrentGroupRates = normalizeAgentGroupRates(result.CurrentGroupRates)
	result.MyGroupRates = normalizeAgentGroupRates(result.MyGroupRates)
	result.DirectChildren = normalizeAgentDirectMembers(directChildren)
	result.DirectChildrenCount = len(result.DirectChildren)
	if result.DirectMemberCount == 0 {
		result.DirectMemberCount = result.DirectChildrenCount
	}
	return &result
}
