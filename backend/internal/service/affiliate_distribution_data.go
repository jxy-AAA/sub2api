package service

import "time"

type RecordAffiliateUsageSettlementInput struct {
	UsageLogID                int64
	ConsumerUserID            int64
	GroupID                   int64
	ModelKey                  string
	UsageAmountUSD            float64
	SettlementAt              time.Time
	RootRate                  float64
	ConsumerRateMultiplier    float64
	HasConsumerRateMultiplier bool
}

type AffiliateDistributionSettlementEntry struct {
	BeneficiaryUserID    int64   `json:"beneficiary_user_id"`
	DirectChildUserID    int64   `json:"direct_child_user_id"`
	ConsumerUserID       int64   `json:"consumer_user_id"`
	ModelKey             string  `json:"model_key"`
	UsageAmountUSD       float64 `json:"usage_amount_usd"`
	RevenueAmountUSD     float64 `json:"revenue_amount_usd"`
	ParentRateMultiplier float64 `json:"parent_rate_multiplier"`
	ChildRateMultiplier  float64 `json:"child_rate_multiplier"`
	RebateAmountRMB      float64 `json:"rebate_amount_rmb"`
	Depth                int     `json:"depth"`
	Applied              bool    `json:"applied"`
}

type AffiliateDistributionSettlementResult struct {
	UsageLogID     int64                                  `json:"usage_log_id"`
	ConsumerUserID int64                                  `json:"consumer_user_id"`
	ModelKey       string                                 `json:"model_key"`
	SettlementDay  time.Time                              `json:"settlement_day"`
	AppliedEntries []AffiliateDistributionSettlementEntry `json:"applied_entries"`
}

type AffiliateDistributionLeaderboardFilter struct {
	Search   string
	Day      *time.Time
	Page     int
	PageSize int
}

type AffiliateDistributionRevenueLeaderboardEntry struct {
	Rank             int64   `json:"rank"`
	UserID           int64   `json:"user_id"`
	Email            string  `json:"email"`
	Username         string  `json:"username"`
	AffCode          string  `json:"aff_code"`
	RevenueAmountUSD float64 `json:"revenue_amount_usd"`
	RebateAmountRMB  float64 `json:"rebate_amount_rmb"`
	UsageCount       int64   `json:"usage_count"`
}

type AffiliateDistributionRebateLeaderboardEntry struct {
	Rank              int64   `json:"rank"`
	UserID            int64   `json:"user_id"`
	Email             string  `json:"email"`
	Username          string  `json:"username"`
	AffCode           string  `json:"aff_code"`
	CurrentAmountRMB  float64 `json:"current_amount_rmb"`
	LifetimeAmountRMB float64 `json:"lifetime_amount_rmb"`
}

type SetAffiliateCurrentRebateBalanceInput struct {
	UserID         int64
	Amount         float64
	OperatorUserID *int64
	Reason         string
}

type AffiliateDistributionBalanceSnapshot struct {
	UserID            int64     `json:"user_id"`
	CurrentAmountRMB  float64   `json:"current_amount_rmb"`
	LifetimeAmountRMB float64   `json:"lifetime_amount_rmb"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ArchiveAffiliateMonthlyBalancesInput struct {
	ArchiveMonth   time.Time
	OperatorUserID *int64
	OperatorName   string
}
