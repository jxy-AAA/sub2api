package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type affiliateDistributionRepository struct {
	client *dbent.Client
}

const (
	affiliateRateMultiplierFenPerUSD            = 10.0
	affiliateDistributionRateMultiplierMax      = 100.0
	affiliateDistributionUsageJobClaimTTL       = 15 * time.Minute
	affiliateDistributionSourceAdminOverride    = "admin_override"
	affiliateDistributionSourceUpstreamOverride = "upstream_override"
	affiliateDistributionSourceInviteCode       = "invite_code"
	affiliateDistributionSourceDefaultExplicit  = "default_explicit"
	affiliateDistributionSourceGroupInherited   = "group_inherited"
	affiliateDistributionSourceGroupDefault     = "group_default"
	affiliateDistributionSourceRootDefault      = "root_default"
	affiliateInviterSourceAffiliateCode         = "affiliate_code"
	affiliateInviterSourceDefaultRoot           = "default_root"
	affiliateDistributionPaidCreditTolerance    = 1e-9
)

type affiliateDistributionQueryExecer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type affiliateDistributionTreeNode struct {
	UserID    int64
	InviterID *int64
	Depth     int
}

func NewAffiliateDistributionRepository(client *dbent.Client) *affiliateDistributionRepository {
	return &affiliateDistributionRepository{client: client}
}

func affiliateDistributionRebateAmountRMB(usageAmountUSD, parentRate, childRate float64) float64 {
	if math.IsNaN(usageAmountUSD) || math.IsInf(usageAmountUSD, 0) || usageAmountUSD <= 0 {
		return 0
	}
	if math.IsNaN(parentRate) || math.IsInf(parentRate, 0) || parentRate <= 0 {
		return 0
	}
	if math.IsNaN(childRate) || math.IsInf(childRate, 0) || childRate <= 0 {
		return 0
	}
	rebateRate := childRate - parentRate
	if rebateRate <= 0 {
		return 0
	}
	rebateAmount := usageAmountUSD * rebateRate / affiliateRateMultiplierFenPerUSD
	if math.IsNaN(rebateAmount) || math.IsInf(rebateAmount, 0) || rebateAmount < 0 {
		return 0
	}
	return rebateAmount
}

func affiliateDistributionFiniteNonNegative(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

func affiliateDistributionBusinessAmountUSD(usageAmountUSD, childRate float64) float64 {
	if !affiliateDistributionFiniteNonNegative(usageAmountUSD) || usageAmountUSD <= 0 {
		return 0
	}
	return usageAmountUSD
}

func affiliateDistributionSettlementAmounts(cmd service.AffiliateDistributionUsageSettlementCommand) (rawUsageUSD, paidUsageUSD, consumerRate float64, hasConsumerRate bool) {
	if !math.IsNaN(cmd.TotalCost) && !math.IsInf(cmd.TotalCost, 0) && cmd.TotalCost > affiliateDistributionPaidCreditTolerance {
		rawUsageUSD = cmd.TotalCost
	}
	if !math.IsNaN(cmd.ActualCost) && !math.IsInf(cmd.ActualCost, 0) && cmd.ActualCost > affiliateDistributionPaidCreditTolerance {
		paidUsageUSD = cmd.ActualCost
	}
	if cmd.HasRateMultiplier && affiliateDistributionFiniteNonNegative(cmd.RateMultiplier) {
		consumerRate = cmd.RateMultiplier
		hasConsumerRate = true
		if paidUsageUSD <= affiliateDistributionPaidCreditTolerance && rawUsageUSD > affiliateDistributionPaidCreditTolerance {
			paidUsageUSD = rawUsageUSD * consumerRate
		}
	}
	if rawUsageUSD <= affiliateDistributionPaidCreditTolerance && paidUsageUSD > affiliateDistributionPaidCreditTolerance {
		rawUsageUSD = paidUsageUSD
	}
	if paidUsageUSD <= affiliateDistributionPaidCreditTolerance && !hasConsumerRate {
		paidUsageUSD = rawUsageUSD
	}
	return rawUsageUSD, paidUsageUSD, consumerRate, hasConsumerRate
}

func affiliateDistributionProratedRawUsage(rawUsageUSD, paidUsageUSD, eligiblePaidUsageUSD float64) float64 {
	if rawUsageUSD <= affiliateDistributionPaidCreditTolerance ||
		paidUsageUSD <= affiliateDistributionPaidCreditTolerance ||
		eligiblePaidUsageUSD <= affiliateDistributionPaidCreditTolerance {
		return 0
	}
	if math.IsNaN(rawUsageUSD) || math.IsInf(rawUsageUSD, 0) ||
		math.IsNaN(paidUsageUSD) || math.IsInf(paidUsageUSD, 0) ||
		math.IsNaN(eligiblePaidUsageUSD) || math.IsInf(eligiblePaidUsageUSD, 0) {
		return 0
	}
	if eligiblePaidUsageUSD >= paidUsageUSD-affiliateDistributionPaidCreditTolerance {
		return rawUsageUSD
	}
	return rawUsageUSD * (eligiblePaidUsageUSD / paidUsageUSD)
}

func scanAgentGroupRates(rows *sql.Rows, inheritedUpstreamUserID *int64) ([]service.AgentGroupRate, error) {
	items := make([]service.AgentGroupRate, 0)
	for rows.Next() {
		var item service.AgentGroupRate
		var upstreamUserID sql.NullInt64
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&item.GroupID,
			&item.GroupName,
			&item.GroupPlatform,
			&item.GroupDefaultRateMultiplier,
			&item.RateMultiplier,
			&item.SourceType,
			&item.SourceAffCode,
			&upstreamUserID,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if upstreamUserID.Valid {
			value := upstreamUserID.Int64
			item.UpstreamUserID = &value
		} else if inheritedUpstreamUserID != nil {
			value := *inheritedUpstreamUserID
			item.UpstreamUserID = &value
		}
		if updatedAt.Valid {
			value := updatedAt.Time
			item.UpdatedAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *affiliateDistributionRepository) GetDistributionOverview(ctx context.Context, userID int64) (*service.AgentDistributionOverview, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	summary, err := ensureUserAffiliateWithClient(ctx, clientFromContext(ctx, r.client), userID)
	if err != nil {
		return nil, err
	}
	today := truncateToShanghaiDate(time.Now())
	businessUSD, rebateRMB, err := r.getDailyTotals(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	balanceRMB, err := r.getCurrentRebateBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	inviteRates, err := r.ListInviteGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	currentRates, err := r.GetUserDistributionGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	isAdmin, err := r.isAdminUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &service.AgentDistributionOverview{
		UserID:                  summary.UserID,
		InviteCode:              summary.AffCode,
		InviterID:               summary.InviterID,
		IsAdmin:                 isAdmin,
		DirectMemberCount:       summary.AffCount,
		TodayBusinessUSD:        businessUSD,
		TodayRebateRMB:          rebateRMB,
		CurrentRebateBalanceRMB: balanceRMB,
		InviteGroupRates:        inviteRates,
		CurrentGroupRates:       currentRates,
		MyGroupRates:            currentRates,
		CanEditSubordinates:     isAdmin || summary.AffCount > 0,
		CanAdjustOwnRebate:      isAdmin,
		MonthlyResetDayOfUTC:    1,
	}, nil
}

func (r *affiliateDistributionRepository) ListInviteGroupRates(ctx context.Context, userID int64) ([]service.AgentGroupRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	return r.listInviteGroupRatesWithClient(ctx, clientFromContext(ctx, r.client), userID)
}

func (r *affiliateDistributionRepository) listInviteGroupRatesWithClient(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) ([]service.AgentGroupRate, error) {
	rows, err := client.QueryContext(ctx, `
SELECT g.id,
       g.name,
       g.platform,
       g.rate_multiplier::double precision,
       COALESCE(r.rate_multiplier, g.rate_multiplier)::double precision,
       CASE WHEN r.group_id IS NOT NULL THEN 'invite_code' ELSE 'group_default' END,
       ''::varchar,
       NULL::bigint,
       r.updated_at
FROM groups g
LEFT JOIN affiliate_distribution_invite_group_rates r
       ON r.group_id = g.id AND r.inviter_user_id = $1
WHERE g.deleted_at IS NULL AND g.status = 'active'
ORDER BY g.sort_order ASC, g.id ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list invite group rates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanAgentGroupRates(rows, nil)
}

func (r *affiliateDistributionRepository) SaveInviteGroupRates(ctx context.Context, userID int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	normalized, err := validateAndNormalizeDistributionGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	err = r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `DELETE FROM affiliate_distribution_invite_group_rates WHERE inviter_user_id = $1`, userID); err != nil {
			return fmt.Errorf("reset invite group rates: %w", err)
		}
		for _, rate := range normalized {
			if _, err := txClient.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_invite_group_rates (inviter_user_id, group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (inviter_user_id, group_id)
DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, updated_at = NOW()`,
				userID, rate.GroupID, rate.RateMultiplier,
			); err != nil {
				return fmt.Errorf("save invite group rate: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListInviteGroupRates(ctx, userID)
}

func (r *affiliateDistributionRepository) ListDirectSubordinates(ctx context.Context, userID int64) ([]service.AgentDirectMember, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
WITH child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
), direct_usage AS (
    SELECT s.direct_child_user_id AS user_id,
           COALESCE(SUM(s.usage_amount_usd), 0)::double precision AS direct_total_usage_usd,
           COALESCE(SUM(s.usage_amount_usd::double precision / $3::double precision), 0)::double precision AS direct_total_usage_rmb,
           COALESCE(SUM(s.rebate_amount), 0)::double precision AS today_rebate_rmb
    FROM affiliate_distribution_usage_settlements s
    WHERE s.beneficiary_user_id = $1
      AND s.settlement_day = $2
      AND s.direct_child_user_id IS NOT NULL
    GROUP BY s.direct_child_user_id
)
SELECT ua.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       (u.role = 'admin' OR COALESCE(cc.actual_children, 0) > 0) AS is_admin,
       ua.created_at,
       COALESCE(du.direct_total_usage_usd, 0)::double precision,
       COALESCE(du.direct_total_usage_rmb, 0)::double precision,
       COALESCE(du.today_rebate_rmb, 0)::double precision,
       COALESCE(du.direct_total_usage_usd, 0)::double precision,
       COALESCE(du.direct_total_usage_rmb, 0)::double precision,
       CASE WHEN u.role <> 'admin' AND COALESCE(cc.actual_children, 0) = 0 THEN COALESCE(du.direct_total_usage_usd, 0) ELSE 0 END::double precision,
       CASE WHEN u.role <> 'admin' AND COALESCE(cc.actual_children, 0) = 0 THEN COALESCE(du.direct_total_usage_rmb, 0) ELSE 0 END::double precision,
       CASE WHEN u.role = 'admin' OR COALESCE(cc.actual_children, 0) > 0 THEN COALESCE(du.direct_total_usage_usd, 0) ELSE 0 END::double precision,
       CASE WHEN u.role = 'admin' OR COALESCE(cc.actual_children, 0) > 0 THEN COALESCE(du.direct_total_usage_rmb, 0) ELSE 0 END::double precision,
       COALESCE(rb.current_amount, 0)::double precision
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
LEFT JOIN child_counts cc ON cc.user_id = ua.user_id
LEFT JOIN direct_usage du ON du.user_id = ua.user_id
LEFT JOIN affiliate_distribution_rebate_balances rb
       ON rb.user_id = ua.user_id
WHERE ua.inviter_id = $1
ORDER BY ua.created_at ASC`, userID, truncateToShanghaiDate(time.Now()), affiliateRateMultiplierFenPerUSD)
	if err != nil {
		return nil, fmt.Errorf("list direct subordinates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentDirectMember, 0)
	for rows.Next() {
		var item service.AgentDirectMember
		var createdAt sql.NullTime
		if err := rows.Scan(
			&item.UserID,
			&item.Email,
			&item.Username,
			&item.IsAgent,
			&createdAt,
			&item.TodayBusinessUSD,
			&item.TodayBusinessRMB,
			&item.TodayRebateRMB,
			&item.DirectTotalUsageUSD,
			&item.DirectTotalUsageRMB,
			&item.DirectUserUsageUSD,
			&item.DirectUserUsageRMB,
			&item.DirectAgentUsageUSD,
			&item.DirectAgentUsageRMB,
			&item.CurrentRebateBalanceRMB,
		); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			ts := createdAt.Time
			item.CreatedAt = &ts
		}
		rates, err := r.GetUserDistributionGroupRates(ctx, item.UserID)
		if err != nil {
			return nil, err
		}
		item.CurrentGroupRates = rates
		item.ParentCanEditRates = true
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *affiliateDistributionRepository) UpdateDirectSubordinateGroupRates(ctx context.Context, userID, subordinateUserID int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	if userID <= 0 || subordinateUserID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDirectSubordinate(ctx, userID, subordinateUserID); err != nil {
		return nil, err
	}
	return r.AdminUpdateUserDistributionGroupRates(ctx, userID, subordinateUserID, rates)
}

func (r *affiliateDistributionRepository) ListUserDistributionHistory(ctx context.Context, userID int64, filter service.AgentHistoryFilter) ([]service.AgentHistoryItem, int64, error) {
	if userID <= 0 {
		return nil, 0, service.ErrUserNotFound
	}
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	startDay := filter.StartAt
	endDay := filter.EndAt
	if startDay == nil {
		now := truncateToShanghaiDate(time.Now())
		startDay = &now
	}
	if endDay == nil {
		now := truncateToShanghaiDate(time.Now())
		endDay = &now
	}
	countSQL := `
SELECT COUNT(*)
FROM (
    SELECT settlement_day
    FROM affiliate_distribution_usage_settlements
    WHERE beneficiary_user_id = $1
      AND settlement_day BETWEEN $2 AND $3
    GROUP BY settlement_day
) t`
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), countSQL, userID, startDay, endDay)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
WITH child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
)
SELECT s.settlement_day::text,
       COALESCE(SUM(s.usage_amount_usd), 0)::double precision,
       COALESCE(SUM(s.rebate_amount), 0)::double precision,
       COUNT(DISTINCT s.direct_child_user_id) FILTER (
           WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
       )::integer,
       COUNT(DISTINCT s.direct_child_user_id) FILTER (
           WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
       )::integer,
       COALESCE(MAX(s.updated_at), MAX(s.created_at), NOW())
FROM affiliate_distribution_usage_settlements s
LEFT JOIN users child_u ON child_u.id = s.direct_child_user_id
LEFT JOIN child_counts child_cc ON child_cc.user_id = s.direct_child_user_id
WHERE s.beneficiary_user_id = $1
  AND s.settlement_day BETWEEN $2 AND $3
GROUP BY s.settlement_day
ORDER BY s.settlement_day DESC
LIMIT $4 OFFSET $5`, userID, startDay, endDay, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentHistoryItem, 0)
	for rows.Next() {
		var item service.AgentHistoryItem
		if err := rows.Scan(&item.StatDate, &item.BusinessUSD, &item.RebateRMB, &item.DirectUsers, &item.DirectAgents, &item.LastCalculated); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *affiliateDistributionRepository) ListDailyBusinessRanking(ctx context.Context, filter service.AgentRankingFilter) ([]service.AgentDailyBusinessRankingItem, int64, error) {
	statDate := truncateToShanghaiDate(time.Now())
	if filter.StatDate != nil {
		statDate = truncateToShanghaiDate(*filter.StatDate)
	}
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	search := "%" + strings.TrimSpace(filter.Search) + "%"
	scopeCTE, scopeArgs, nextArgIndex := buildAffiliateScopeCTE(filter.RootUserID, filter.OnlyDescendants, 1)
	searchArgIndex := nextArgIndex
	statDateArgIndex := nextArgIndex + 1
	limitArgIndex := nextArgIndex + 2
	offsetArgIndex := nextArgIndex + 3
	countSQL := fmt.Sprintf(`
WITH %s,
child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
),
agents AS (
    SELECT scope.user_id
    FROM scope_users scope
    JOIN users u ON u.id = scope.user_id
    LEFT JOIN child_counts cc ON cc.user_id = scope.user_id
    WHERE u.role = 'admin' OR COALESCE(cc.actual_children, 0) > 0
)
SELECT COUNT(*)
FROM agents a
JOIN users u ON u.id = a.user_id
WHERE u.email ILIKE $%d OR u.username ILIKE $%d`, scopeCTE, searchArgIndex, searchArgIndex)
	countArgs := append(append([]any{}, scopeArgs...), search)
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), countSQL, countArgs...)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, fmt.Sprintf(`
WITH %s,
child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
),
direct_counts AS (
    SELECT ua.inviter_id AS user_id,
           COUNT(*)::integer AS direct_children_count,
           COUNT(*) FILTER (WHERE child_u.role = 'admin' OR COALESCE(child_cc.actual_children, 0) > 0)::integer AS direct_agents,
           COUNT(*) FILTER (WHERE child_u.role <> 'admin' AND COALESCE(child_cc.actual_children, 0) = 0)::integer AS direct_users
    FROM user_affiliates ua
    JOIN users child_u ON child_u.id = ua.user_id
    LEFT JOIN child_counts child_cc ON child_cc.user_id = ua.user_id
    WHERE ua.inviter_id IS NOT NULL
    GROUP BY ua.inviter_id
),
agents AS (
    SELECT scope.user_id
    FROM scope_users scope
    JOIN users u ON u.id = scope.user_id
    LEFT JOIN direct_counts dc ON dc.user_id = scope.user_id
    WHERE u.role = 'admin' OR COALESCE(dc.direct_children_count, 0) > 0
),
business AS (
    SELECT s.beneficiary_user_id AS user_id,
           s.settlement_day::text AS stat_date,
           COALESCE(SUM(s.usage_amount_usd), 0)::double precision AS business_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f), 0)::double precision AS business_rmb,
           COALESCE(MAX(s.updated_at), MAX(s.created_at), NOW()) AS last_calculated_at
    FROM affiliate_distribution_usage_settlements s
    JOIN agents a ON a.user_id = s.beneficiary_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id, s.settlement_day
), direct_usage AS (
    SELECT s.beneficiary_user_id AS user_id,
           COALESCE(SUM(s.usage_amount_usd), 0)::double precision AS direct_total_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f), 0)::double precision AS direct_total_usage_rmb,
           COALESCE(SUM(s.usage_amount_usd) FILTER (
               WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
           ), 0)::double precision AS direct_user_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f) FILTER (
               WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
           ), 0)::double precision AS direct_user_usage_rmb,
           COALESCE(SUM(s.usage_amount_usd) FILTER (
               WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
           ), 0)::double precision AS direct_agent_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f) FILTER (
               WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
           ), 0)::double precision AS direct_agent_usage_rmb
    FROM affiliate_distribution_usage_settlements s
    JOIN agents a ON a.user_id = s.beneficiary_user_id
    LEFT JOIN users child_u ON child_u.id = s.direct_child_user_id
    LEFT JOIN child_counts child_cc ON child_cc.user_id = s.direct_child_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id
),
ranked AS (
    SELECT a.user_id,
           COALESCE(u.email, '') AS email,
           COALESCE(u.username, '') AS username,
           COALESCE(b.stat_date, $%d::date::text) AS stat_date,
           COALESCE(b.business_usd, 0)::double precision AS business_usd,
           COALESCE(b.business_rmb, 0)::double precision AS business_rmb,
           COALESCE(dc.direct_users, 0)::integer AS direct_users,
           COALESCE(dc.direct_agents, 0)::integer AS direct_agents,
           COALESCE(du.direct_total_usage_usd, 0)::double precision AS direct_total_usage_usd,
           COALESCE(du.direct_total_usage_rmb, 0)::double precision AS direct_total_usage_rmb,
           COALESCE(du.direct_user_usage_usd, 0)::double precision AS direct_user_usage_usd,
           COALESCE(du.direct_user_usage_rmb, 0)::double precision AS direct_user_usage_rmb,
           COALESCE(du.direct_agent_usage_usd, 0)::double precision AS direct_agent_usage_usd,
           COALESCE(du.direct_agent_usage_rmb, 0)::double precision AS direct_agent_usage_rmb,
           COALESCE(b.last_calculated_at, NOW()) AS last_calculated_at,
           ROW_NUMBER() OVER (ORDER BY COALESCE(b.business_rmb, 0) DESC, a.user_id ASC) AS rank
    FROM agents a
    JOIN users u ON u.id = a.user_id
    LEFT JOIN business b ON b.user_id = a.user_id
    LEFT JOIN direct_counts dc ON dc.user_id = a.user_id
    LEFT JOIN direct_usage du ON du.user_id = a.user_id
    WHERE u.email ILIKE $%d OR u.username ILIKE $%d
)
SELECT rank, user_id, email, username, stat_date, business_usd, business_rmb,
       direct_users, direct_agents,
       direct_total_usage_usd, direct_total_usage_rmb,
       direct_user_usage_usd, direct_user_usage_rmb,
       direct_agent_usage_usd, direct_agent_usage_rmb,
       last_calculated_at
FROM ranked
ORDER BY rank
LIMIT $%d OFFSET $%d`,
		scopeCTE,
		affiliateRateMultiplierFenPerUSD,
		statDateArgIndex,
		affiliateRateMultiplierFenPerUSD,
		affiliateRateMultiplierFenPerUSD,
		affiliateRateMultiplierFenPerUSD,
		statDateArgIndex,
		statDateArgIndex,
		searchArgIndex,
		searchArgIndex,
		limitArgIndex,
		offsetArgIndex),
		append(append([]any{}, scopeArgs...), search, statDate, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentDailyBusinessRankingItem, 0)
	for rows.Next() {
		var item service.AgentDailyBusinessRankingItem
		if err := rows.Scan(
			&item.Rank,
			&item.UserID,
			&item.Email,
			&item.Username,
			&item.StatDate,
			&item.BusinessUSD,
			&item.BusinessRMB,
			&item.DirectUsers,
			&item.DirectAgents,
			&item.DirectTotalUsageUSD,
			&item.DirectTotalUsageRMB,
			&item.DirectUserUsageUSD,
			&item.DirectUserUsageRMB,
			&item.DirectAgentUsageUSD,
			&item.DirectAgentUsageRMB,
			&item.LastCalculatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *affiliateDistributionRepository) ListRebateBalanceRanking(ctx context.Context, filter service.AgentRankingFilter) ([]service.AgentRebateBalanceRankingItem, int64, error) {
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	search := "%" + strings.TrimSpace(filter.Search) + "%"
	today := truncateToShanghaiDate(time.Now())
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	scopeCTE, scopeArgs, nextArgIndex := buildAffiliateScopeCTE(filter.RootUserID, filter.OnlyDescendants, 1)
	searchArgIndex := nextArgIndex
	todayArgIndex := nextArgIndex + 1
	monthStartArgIndex := nextArgIndex + 2
	limitArgIndex := nextArgIndex + 3
	offsetArgIndex := nextArgIndex + 4
	countSQL := fmt.Sprintf(`
WITH %s,
child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
),
agents AS (
    SELECT scope.user_id
    FROM scope_users scope
    JOIN users u ON u.id = scope.user_id
    LEFT JOIN child_counts cc ON cc.user_id = scope.user_id
    WHERE u.role = 'admin' OR COALESCE(cc.actual_children, 0) > 0
)
SELECT COUNT(*)
FROM agents a
JOIN users u ON u.id = a.user_id
WHERE u.email ILIKE $%d OR u.username ILIKE $%d`, scopeCTE, searchArgIndex, searchArgIndex)
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), countSQL, append(append([]any{}, scopeArgs...), search)...)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, fmt.Sprintf(`
WITH %s,
child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
),
latest_note AS (
    SELECT DISTINCT ON (user_id) user_id, reason, created_at
    FROM affiliate_distribution_rebate_adjustments
    ORDER BY user_id, created_at DESC
), today_metrics AS (
    SELECT s.beneficiary_user_id AS user_id,
           COALESCE(SUM(s.rebate_amount), 0)::double precision AS today_rebate_rmb
    FROM affiliate_distribution_usage_settlements s
    JOIN scope_users scope ON scope.user_id = s.beneficiary_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id
), monthly_metrics AS (
    SELECT s.beneficiary_user_id AS user_id,
           COALESCE(SUM(s.rebate_amount), 0)::double precision AS monthly_rebate_rmb
    FROM affiliate_distribution_usage_settlements s
    JOIN scope_users scope ON scope.user_id = s.beneficiary_user_id
    WHERE s.settlement_day >= $%d AND s.settlement_day < ($%d::date + INTERVAL '1 month')
    GROUP BY s.beneficiary_user_id
), direct_counts AS (
    SELECT ua.inviter_id AS user_id,
           COUNT(*)::integer AS direct_children_count,
           COUNT(*) FILTER (WHERE child_u.role = 'admin' OR COALESCE(child_cc.actual_children, 0) > 0)::integer AS direct_agents,
           COUNT(*) FILTER (WHERE child_u.role <> 'admin' AND COALESCE(child_cc.actual_children, 0) = 0)::integer AS direct_users
    FROM user_affiliates ua
    JOIN users child_u ON child_u.id = ua.user_id
    LEFT JOIN child_counts child_cc ON child_cc.user_id = ua.user_id
    WHERE ua.inviter_id IS NOT NULL
    GROUP BY ua.inviter_id
), agents AS (
    SELECT scope.user_id
    FROM scope_users scope
    JOIN users u ON u.id = scope.user_id
    LEFT JOIN direct_counts dc ON dc.user_id = scope.user_id
    WHERE u.role = 'admin' OR COALESCE(dc.direct_children_count, 0) > 0
), direct_usage AS (
    SELECT s.beneficiary_user_id AS user_id,
           COALESCE(SUM(s.usage_amount_usd), 0)::double precision AS direct_total_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f), 0)::double precision AS direct_total_usage_rmb,
           COALESCE(SUM(s.usage_amount_usd) FILTER (
               WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
           ), 0)::double precision AS direct_user_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f) FILTER (
               WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
           ), 0)::double precision AS direct_user_usage_rmb,
           COALESCE(SUM(s.usage_amount_usd) FILTER (
               WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
           ), 0)::double precision AS direct_agent_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f) FILTER (
               WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
           ), 0)::double precision AS direct_agent_usage_rmb
    FROM affiliate_distribution_usage_settlements s
    JOIN agents a ON a.user_id = s.beneficiary_user_id
    LEFT JOIN users child_u ON child_u.id = s.direct_child_user_id
    LEFT JOIN child_counts child_cc ON child_cc.user_id = s.direct_child_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id
),
ranked AS (
    SELECT a.user_id,
           COALESCE(u.email, '') AS email,
           COALESCE(u.username, '') AS username,
           COALESCE(rb.current_amount, 0)::double precision AS current_rebate_balance_rmb,
           COALESCE(tm.today_rebate_rmb, 0)::double precision AS today_rebate_rmb,
           COALESCE(mm.monthly_rebate_rmb, 0)::double precision AS monthly_rebate_rmb,
           COALESCE(dc.direct_users, 0)::integer AS direct_users,
           COALESCE(dc.direct_agents, 0)::integer AS direct_agents,
           COALESCE(du.direct_total_usage_usd, 0)::double precision AS direct_total_usage_usd,
           COALESCE(du.direct_total_usage_rmb, 0)::double precision AS direct_total_usage_rmb,
           COALESCE(du.direct_user_usage_usd, 0)::double precision AS direct_user_usage_usd,
           COALESCE(du.direct_user_usage_rmb, 0)::double precision AS direct_user_usage_rmb,
           COALESCE(du.direct_agent_usage_usd, 0)::double precision AS direct_agent_usage_usd,
           COALESCE(du.direct_agent_usage_rmb, 0)::double precision AS direct_agent_usage_rmb,
           COALESCE(ln.created_at, rb.updated_at, NOW()) AS last_adjusted_at,
           COALESCE(ln.reason, '') AS last_adjustment_note,
           ROW_NUMBER() OVER (ORDER BY COALESCE(rb.current_amount, 0) DESC, rb.updated_at DESC NULLS LAST, a.user_id ASC) AS rank
    FROM agents a
    JOIN users u ON u.id = a.user_id
    LEFT JOIN affiliate_distribution_rebate_balances rb ON rb.user_id = a.user_id
    LEFT JOIN latest_note ln ON ln.user_id = a.user_id
    LEFT JOIN today_metrics tm ON tm.user_id = a.user_id
    LEFT JOIN monthly_metrics mm ON mm.user_id = a.user_id
    LEFT JOIN direct_counts dc ON dc.user_id = a.user_id
    LEFT JOIN direct_usage du ON du.user_id = a.user_id
    WHERE u.email ILIKE $%d OR u.username ILIKE $%d
)
SELECT rank, user_id, email, username, current_rebate_balance_rmb,
       today_rebate_rmb, monthly_rebate_rmb, direct_users, direct_agents,
       direct_total_usage_usd, direct_total_usage_rmb,
       direct_user_usage_usd, direct_user_usage_rmb,
       direct_agent_usage_usd, direct_agent_usage_rmb,
       last_adjusted_at, last_adjustment_note
FROM ranked
ORDER BY rank
LIMIT $%d OFFSET $%d`,
		scopeCTE,
		todayArgIndex,
		monthStartArgIndex,
		monthStartArgIndex,
		affiliateRateMultiplierFenPerUSD,
		affiliateRateMultiplierFenPerUSD,
		affiliateRateMultiplierFenPerUSD,
		todayArgIndex,
		searchArgIndex,
		searchArgIndex,
		limitArgIndex,
		offsetArgIndex),
		append(append([]any{}, scopeArgs...), search, today, monthStart, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentRebateBalanceRankingItem, 0)
	for rows.Next() {
		var item service.AgentRebateBalanceRankingItem
		if err := rows.Scan(
			&item.Rank,
			&item.UserID,
			&item.Email,
			&item.Username,
			&item.CurrentRebateBalanceRMB,
			&item.TodayRebateRMB,
			&item.MonthlyRebateRMB,
			&item.DirectUsers,
			&item.DirectAgents,
			&item.DirectTotalUsageUSD,
			&item.DirectTotalUsageRMB,
			&item.DirectUserUsageUSD,
			&item.DirectUserUsageRMB,
			&item.DirectAgentUsageUSD,
			&item.DirectAgentUsageRMB,
			&item.LastAdjustedAt,
			&item.LastAdjustmentNote,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *affiliateDistributionRepository) GetAgentDistributionPermissions(ctx context.Context, userID int64) (*service.AgentDistributionPermission, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	return r.getAgentDistributionPermissionsWithClient(ctx, clientFromContext(ctx, r.client), userID)
}

func (r *affiliateDistributionRepository) UpdateAgentDistributionPermissions(ctx context.Context, operatorUserID, userID int64, input service.UpdateAgentDistributionPermissionInput) (*service.AgentDistributionPermission, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	var permission *service.AgentDistributionPermission
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := ensureUserExists(txCtx, txClient, operatorUserID); err != nil {
			return err
		}
		if err := ensureUserExists(txCtx, txClient, userID); err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_agent_permissions (
    user_id,
    can_view_downline_daily_revenue,
    can_view_downline_rebate_balances,
    can_manage_downline_pricing,
    granted_by_user_id,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (user_id)
DO UPDATE SET
    can_view_downline_daily_revenue = EXCLUDED.can_view_downline_daily_revenue,
    can_view_downline_rebate_balances = EXCLUDED.can_view_downline_rebate_balances,
    can_manage_downline_pricing = EXCLUDED.can_manage_downline_pricing,
    granted_by_user_id = EXCLUDED.granted_by_user_id,
    updated_at = NOW()`,
			userID,
			input.CanViewDownlineDailyRevenue,
			input.CanViewDownlineRebateBalances,
			input.CanManageDownlinePricing,
			operatorUserID,
		); err != nil {
			return fmt.Errorf("upsert agent distribution permissions: %w", err)
		}
		var err error
		permission, err = r.getAgentDistributionPermissionsWithClient(txCtx, txClient, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return permission, nil
}

func (r *affiliateDistributionRepository) AdminSetRebateBalance(ctx context.Context, operatorUserID, userID int64, amount float64, note string) (*service.AgentRebateBalanceAdjustment, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureRootAdmin(ctx, operatorUserID); err != nil {
		return nil, err
	}
	var out *service.AgentRebateBalanceAdjustment
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if err := r.ensureRebateBalanceRow(txCtx, txClient, userID); err != nil {
			return err
		}
		var previous float64
		if err := scanSingleRow(txCtx, txClient, `
SELECT current_amount::double precision
FROM affiliate_distribution_rebate_balances
WHERE user_id = $1
FOR UPDATE`, []any{userID}, &previous); err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE affiliate_distribution_rebate_balances
SET current_amount = $1,
    updated_at = NOW()
WHERE user_id = $2`, amount, userID); err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_rebate_adjustments (
    user_id, operator_user_id, adjustment_type, previous_amount, new_amount, delta_amount, reason, created_at
) VALUES ($1, $2, 'admin_set', $3, $4, $5, $6, NOW())`,
			userID, operatorUserID, previous, amount, amount-previous, note,
		); err != nil {
			return err
		}
		out = &service.AgentRebateBalanceAdjustment{
			UserID:             userID,
			OperatorUserID:     operatorUserID,
			PreviousBalanceRMB: previous,
			NewBalanceRMB:      amount,
			Note:               note,
			AdjustedAt:         time.Now().UTC(),
		}
		return nil
	})
	return out, err
}

func (r *affiliateDistributionRepository) GetDistributionTree(ctx context.Context, filter service.AgentTreeFilter) ([]service.AgentTreeNode, error) {
	search := "%" + strings.TrimSpace(filter.Search) + "%"
	rootPredicate := "ua.inviter_id IS NULL"
	args := make([]any, 0, 2)
	searchArgIndex := 1
	if filter.RootUserID != nil {
		rootPredicate = "ua.user_id = $1"
		args = append(args, *filter.RootUserID)
		searchArgIndex = 2
	}
	todayArgIndex := searchArgIndex + 1
	descendantClause := ""
	if filter.OnlyDescendants {
		descendantClause = "AND t.depth > 0"
	}
	query := fmt.Sprintf(`
WITH RECURSIVE tree AS (
    SELECT ua.user_id, ua.inviter_id, 0 AS depth
    FROM user_affiliates ua
    WHERE %s
    UNION ALL
    SELECT child.user_id, child.inviter_id, tree.depth + 1
    FROM user_affiliates child
    JOIN tree ON child.inviter_id = tree.user_id
), child_counts AS (
    SELECT inviter_id AS user_id, COUNT(*)::integer AS actual_children
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
), today_metrics AS (
    SELECT s.beneficiary_user_id AS user_id,
           COALESCE(SUM(s.usage_amount_usd), 0)::double precision AS today_business_usd,
           COALESCE(SUM(s.rebate_amount), 0)::double precision AS today_rebate_rmb
    FROM affiliate_distribution_usage_settlements s
    JOIN tree t ON t.user_id = s.beneficiary_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id
), direct_counts AS (
    SELECT ua.inviter_id AS user_id,
           COUNT(*)::integer AS direct_children_count,
           COUNT(*) FILTER (WHERE child_u.role = 'admin' OR COALESCE(child_cc.actual_children, 0) > 0)::integer AS direct_agent_count,
           COUNT(*) FILTER (WHERE child_u.role <> 'admin' AND COALESCE(child_cc.actual_children, 0) = 0)::integer AS direct_user_count
    FROM user_affiliates ua
    JOIN tree t ON t.user_id = ua.inviter_id
    JOIN users child_u ON child_u.id = ua.user_id
    LEFT JOIN child_counts child_cc ON child_cc.user_id = ua.user_id
    WHERE ua.inviter_id IS NOT NULL
    GROUP BY ua.inviter_id
), direct_usage AS (
    SELECT s.beneficiary_user_id AS user_id,
           COALESCE(SUM(s.usage_amount_usd), 0)::double precision AS direct_total_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f), 0)::double precision AS direct_total_usage_rmb,
           COALESCE(SUM(s.usage_amount_usd) FILTER (
               WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
           ), 0)::double precision AS direct_user_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f) FILTER (
               WHERE COALESCE(child_u.role <> 'admin', true) AND COALESCE(child_cc.actual_children, 0) = 0
           ), 0)::double precision AS direct_user_usage_rmb,
           COALESCE(SUM(s.usage_amount_usd) FILTER (
               WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
           ), 0)::double precision AS direct_agent_usage_usd,
           COALESCE(SUM(s.usage_amount_usd / %.1f) FILTER (
               WHERE COALESCE(child_u.role = 'admin', false) OR COALESCE(child_cc.actual_children, 0) > 0
           ), 0)::double precision AS direct_agent_usage_rmb
    FROM affiliate_distribution_usage_settlements s
    JOIN tree t ON t.user_id = s.beneficiary_user_id
    LEFT JOIN users child_u ON child_u.id = s.direct_child_user_id
    LEFT JOIN child_counts child_cc ON child_cc.user_id = s.direct_child_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id
)
SELECT t.user_id,
       t.inviter_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       ua.aff_code,
       t.depth,
       (u.role = 'admin')::boolean,
       (u.id = (SELECT id FROM users WHERE role = 'admin' AND status = 'active' ORDER BY id ASC LIMIT 1))::boolean,
       COALESCE(rb.current_amount, 0)::double precision,
       COALESCE(tm.today_business_usd, 0)::double precision,
       COALESCE(du.direct_total_usage_rmb, 0)::double precision,
       COALESCE(tm.today_rebate_rmb, 0)::double precision,
       COALESCE(dc.direct_children_count, 0)::integer,
       COALESCE(dc.direct_user_count, 0)::integer,
       COALESCE(dc.direct_agent_count, 0)::integer,
       COALESCE(du.direct_total_usage_usd, 0)::double precision,
       COALESCE(du.direct_total_usage_rmb, 0)::double precision,
       COALESCE(du.direct_user_usage_usd, 0)::double precision,
       COALESCE(du.direct_user_usage_rmb, 0)::double precision,
       COALESCE(du.direct_agent_usage_usd, 0)::double precision,
       COALESCE(du.direct_agent_usage_rmb, 0)::double precision
FROM tree t
JOIN users u ON u.id = t.user_id
JOIN user_affiliates ua ON ua.user_id = t.user_id
LEFT JOIN affiliate_distribution_rebate_balances rb ON rb.user_id = t.user_id
LEFT JOIN today_metrics tm ON tm.user_id = t.user_id
LEFT JOIN direct_counts dc ON dc.user_id = t.user_id
LEFT JOIN direct_usage du ON du.user_id = t.user_id
WHERE (u.email ILIKE $%d OR u.username ILIKE $%d OR ua.aff_code ILIKE $%d)
  %s
ORDER BY t.depth ASC, t.user_id ASC`,
		rootPredicate,
		todayArgIndex,
		affiliateRateMultiplierFenPerUSD,
		affiliateRateMultiplierFenPerUSD,
		affiliateRateMultiplierFenPerUSD,
		todayArgIndex,
		searchArgIndex,
		searchArgIndex,
		searchArgIndex,
		descendantClause)
	args = append(args, search, truncateToShanghaiDate(time.Now()))
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentTreeNode, 0)
	for rows.Next() {
		var node service.AgentTreeNode
		var inviterID sql.NullInt64
		if err := rows.Scan(
			&node.UserID,
			&inviterID,
			&node.Email,
			&node.Username,
			&node.InviteCode,
			&node.Depth,
			&node.IsAdmin,
			&node.IsRootAdmin,
			&node.CurrentRebateBalanceRMB,
			&node.TodayBusinessUSD,
			&node.TodayBusinessRMB,
			&node.TodayRebateRMB,
			&node.DirectChildrenCount,
			&node.DirectUserCount,
			&node.DirectAgentCount,
			&node.DirectTotalUsageUSD,
			&node.DirectTotalUsageRMB,
			&node.DirectUserUsageUSD,
			&node.DirectUserUsageRMB,
			&node.DirectAgentUsageUSD,
			&node.DirectAgentUsageRMB,
		); err != nil {
			return nil, err
		}
		if inviterID.Valid {
			node.InviterID = &inviterID.Int64
		}
		rates, err := r.GetUserDistributionGroupRates(ctx, node.UserID)
		if err != nil {
			return nil, err
		}
		node.CurrentGroupRates = rates
		inviteRates, err := r.ListInviteGroupRates(ctx, node.UserID)
		if err != nil {
			return nil, err
		}
		node.InviteGroupRates = inviteRates
		items = append(items, node)
	}
	return items, rows.Err()
}

func (r *affiliateDistributionRepository) GetUserDistributionGroupRates(ctx context.Context, userID int64) ([]service.AgentGroupRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	return r.getUserDistributionGroupRatesWithClient(ctx, clientFromContext(ctx, r.client), userID, make(map[int64][]service.AgentGroupRate))
}

func (r *affiliateDistributionRepository) AdminUpdateUserDistributionGroupRates(ctx context.Context, operatorUserID, userID int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDirectOrAdmin(ctx, operatorUserID, userID); err != nil {
		return nil, err
	}
	if _, err := r.saveUserDistributionGroupRates(ctx, userID, rates, "admin_override"); err != nil {
		return nil, err
	}
	return r.GetUserDistributionGroupRates(ctx, userID)
}

func (r *affiliateDistributionRepository) ListDailyBusinessRankingScoped(ctx context.Context, operatorUserID int64, filter service.AgentRankingFilter) ([]service.AgentDailyBusinessRankingItem, int64, error) {
	if operatorUserID <= 0 {
		return nil, 0, service.ErrUserNotFound
	}
	filter.RootUserID = &operatorUserID
	filter.OnlyDescendants = true
	return r.ListDailyBusinessRanking(ctx, filter)
}

func (r *affiliateDistributionRepository) ListRebateBalanceRankingScoped(ctx context.Context, operatorUserID int64, filter service.AgentRankingFilter) ([]service.AgentRebateBalanceRankingItem, int64, error) {
	if operatorUserID <= 0 {
		return nil, 0, service.ErrUserNotFound
	}
	filter.RootUserID = &operatorUserID
	filter.OnlyDescendants = true
	return r.ListRebateBalanceRanking(ctx, filter)
}

func (r *affiliateDistributionRepository) GetDistributionTreeScoped(ctx context.Context, operatorUserID int64, filter service.AgentTreeFilter) ([]service.AgentTreeNode, error) {
	if operatorUserID <= 0 {
		return nil, service.ErrUserNotFound
	}
	filter.RootUserID = &operatorUserID
	filter.OnlyDescendants = true
	return r.GetDistributionTree(ctx, filter)
}

func (r *affiliateDistributionRepository) GetUserDistributionGroupRatesScoped(ctx context.Context, operatorUserID, userID int64) ([]service.AgentGroupRate, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDescendant(ctx, operatorUserID, userID); err != nil {
		return nil, err
	}
	return r.GetUserDistributionGroupRates(ctx, userID)
}

func (r *affiliateDistributionRepository) UpdateUserDistributionGroupRatesScoped(ctx context.Context, operatorUserID, userID int64, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDescendant(ctx, operatorUserID, userID); err != nil {
		return nil, err
	}
	if _, err := r.saveUserDistributionGroupRates(ctx, userID, rates, "upstream_override"); err != nil {
		return nil, err
	}
	return r.GetUserDistributionGroupRates(ctx, userID)
}

func (r *affiliateDistributionRepository) ListDefaultUserGroupRates(ctx context.Context) ([]service.AgentGroupRate, error) {
	return r.listDefaultUserGroupRatesWithClient(ctx, clientFromContext(ctx, r.client))
}

func (r *affiliateDistributionRepository) SaveDefaultUserGroupRates(ctx context.Context, rates []service.AgentGroupRateInput) ([]service.AgentGroupRate, error) {
	normalized, err := validateAndNormalizeDistributionGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	err = r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := txClient.ExecContext(txCtx, `DELETE FROM affiliate_distribution_default_user_group_rates`); err != nil {
			return fmt.Errorf("reset default user group rates: %w", err)
		}
		for _, rate := range normalized {
			if _, err := txClient.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_default_user_group_rates (group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (group_id)
DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, updated_at = NOW()`,
				rate.GroupID, rate.RateMultiplier,
			); err != nil {
				return fmt.Errorf("save default user group rate: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListDefaultUserGroupRates(ctx)
}

func (r *affiliateDistributionRepository) EnsureUserDistributionPricing(ctx context.Context, userID int64) ([]service.AgentGroupRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		return r.ensureUserDistributionPricingWithClient(txCtx, txClient, userID)
	}); err != nil {
		return nil, err
	}
	return r.GetUserDistributionGroupRates(ctx, userID)
}

func (r *affiliateDistributionRepository) AdminSetUserInviter(ctx context.Context, operatorUserID, userID int64, inviterID *int64) (*service.AffiliateSummary, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	isAdmin, err := r.isAdminUser(ctx, operatorUserID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, infraerrors.Forbidden("AFFILIATE_DISTRIBUTION_ADMIN_REQUIRED", "admin access required")
	}
	var updated *service.AffiliateSummary
	err = r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		rootAdminID, hasRootAdmin, err := queryFirstActiveAdminID(txCtx, txClient)
		if err != nil {
			return err
		}
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		current, err := queryAffiliateByUserID(txCtx, txClient, userID)
		if err != nil {
			return err
		}
		if hasRootAdmin && userID == rootAdminID {
			if inviterID != nil {
				return infraerrors.Forbidden("AFFILIATE_DISTRIBUTION_ROOT_ADMIN_UPSTREAM_FORBIDDEN", "root admin cannot set inviter")
			}
			updated = current
			return nil
		}

		targetInviterID := inviterID
		inviterSource := "admin_override"
		if targetInviterID == nil && hasRootAdmin {
			targetInviterID = &rootAdminID
			inviterSource = "default_root"
		} else if targetInviterID == nil {
			inviterSource = "none"
		}
		if targetInviterID != nil {
			if *targetInviterID == userID {
				return infraerrors.BadRequest("AFFILIATE_DISTRIBUTION_UPSTREAM_SELF_REFERENCE", "user cannot set self as inviter")
			}
			if _, err := ensureUserAffiliateWithClient(txCtx, txClient, *targetInviterID); err != nil {
				return err
			}
			if err := r.ensureNoInviterCycle(txCtx, txClient, userID, *targetInviterID); err != nil {
				return err
			}
		}

		if sameNullableInt64(current.InviterID, targetInviterID) {
			updated = current
			return nil
		}

		if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET inviter_id = $1,
    inviter_source = $2,
    updated_at = NOW()
WHERE user_id = $3`, nullableInt64(targetInviterID), inviterSource, userID); err != nil {
			return fmt.Errorf("update affiliate inviter: %w", err)
		}
		if current.InviterID != nil {
			if err := refreshAffiliateCountWithClient(txCtx, txClient, *current.InviterID); err != nil {
				return err
			}
		}
		if targetInviterID != nil {
			if err := refreshAffiliateCountWithClient(txCtx, txClient, *targetInviterID); err != nil {
				return err
			}
		}

		sourceCode := ""
		if targetInviterID != nil {
			parentSummary, err := queryAffiliateByUserID(txCtx, txClient, *targetInviterID)
			if err != nil {
				return err
			}
			sourceCode = parentSummary.AffCode
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE affiliate_distribution_user_group_rates
SET upstream_user_id = $1,
    source_aff_code = $2,
    updated_at = NOW()
WHERE user_id = $3`, nullableInt64(targetInviterID), nullableString(sourceCode), userID); err != nil {
			return fmt.Errorf("sync user group rate upstream: %w", err)
		}
		updated, err = queryAffiliateByUserID(txCtx, txClient, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *affiliateDistributionRepository) AdminUpdateUserUpstream(ctx context.Context, operatorUserID, userID int64, upstreamUserID *int64) (*service.AgentUserUpstream, error) {
	summary, err := r.AdminSetUserInviter(ctx, operatorUserID, userID, upstreamUserID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, service.ErrUserNotFound
	}
	updatedAt := summary.UpdatedAt
	return &service.AgentUserUpstream{
		UserID:         summary.UserID,
		UpstreamUserID: summary.InviterID,
		InviterID:      summary.InviterID,
		UpdatedAt:      &updatedAt,
	}, nil
}

func (r *affiliateDistributionRepository) ListMonthlyRebateArchives(ctx context.Context, filter service.AgentMonthlyArchiveFilter) ([]service.AgentMonthlyArchiveItem, int64, error) {
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	search := "%" + strings.TrimSpace(filter.Search) + "%"
	month := strings.TrimSpace(filter.Month)
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), `
SELECT COUNT(*)
FROM affiliate_distribution_monthly_archives a
JOIN users u ON u.id = a.user_id
WHERE a.archive_month::text LIKE $1 || '%'
  AND (u.email ILIKE $2 OR u.username ILIKE $2)`, month, search)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT a.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       to_char(a.archive_month, 'YYYY-MM') AS month,
       a.archived_amount::double precision,
       a.snapshot_at
FROM affiliate_distribution_monthly_archives a
JOIN users u ON u.id = a.user_id
WHERE a.archive_month::text LIKE $1 || '%'
  AND (u.email ILIKE $2 OR u.username ILIKE $2)
ORDER BY a.archive_month DESC, a.archived_amount DESC
LIMIT $3 OFFSET $4`, month, search, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentMonthlyArchiveItem, 0)
	for rows.Next() {
		var item service.AgentMonthlyArchiveItem
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.Month, &item.ArchivedRebateRMB, &item.ArchivedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *affiliateDistributionRepository) ArchiveMonthlyRebateBalances(ctx context.Context, archiveMonth time.Time, operatorUserID *int64, operatorName string) (int64, error) {
	archiveMonth = time.Date(archiveMonth.UTC().Year(), archiveMonth.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	var archived int64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		rows, err := txClient.QueryContext(txCtx, `
WITH claimed AS (
    INSERT INTO affiliate_distribution_monthly_reset_runs (
        archive_month, operator_user_id, operator_name, archived_count, created_at, updated_at
    )
    VALUES ($1::date, $2, NULLIF($3, ''), 0, NOW(), NOW())
    ON CONFLICT (archive_month) DO NOTHING
    RETURNING archive_month
), active AS (
    SELECT user_id, current_amount
    FROM affiliate_distribution_rebate_balances
    WHERE current_amount <> 0
      AND EXISTS (SELECT 1 FROM claimed)
), inserted AS (
    INSERT INTO affiliate_distribution_monthly_archives (
        user_id, archive_month, archived_amount, operator_user_id, operator_name, snapshot_at, created_at, updated_at
    )
    SELECT user_id, $1::date, current_amount, $2, NULLIF($3, ''), NOW(), NOW(), NOW()
    FROM active
    ON CONFLICT (user_id, archive_month) DO NOTHING
    RETURNING user_id, archived_amount
), adjusted AS (
    INSERT INTO affiliate_distribution_rebate_adjustments (
        user_id, operator_user_id, adjustment_type, previous_amount, new_amount, delta_amount, reason, created_at
    )
    SELECT user_id, $2, 'monthly_reset', archived_amount, 0, -archived_amount,
           'monthly archive reset: ' || to_char($1::date, 'YYYY-MM'), NOW()
    FROM inserted
    RETURNING user_id
), reset AS (
    UPDATE affiliate_distribution_rebate_balances b
    SET current_amount = 0,
        last_reset_month = $1::date,
        updated_at = NOW()
    FROM inserted i
    WHERE b.user_id = i.user_id
    RETURNING b.user_id
), finalized AS (
    UPDATE affiliate_distribution_monthly_reset_runs r
    SET archived_count = (SELECT COUNT(*)::integer FROM reset),
        updated_at = NOW()
    WHERE r.archive_month = $1::date
      AND EXISTS (SELECT 1 FROM claimed)
    RETURNING archived_count
)
SELECT COALESCE((SELECT archived_count::bigint FROM finalized), 0)::bigint`, archiveMonth, nullableInt64Arg(operatorUserID), strings.TrimSpace(operatorName))
		if err != nil {
			return err
		}
		defer rows.Close()
		if rows.Next() {
			if err := rows.Scan(&archived); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	return archived, err
}

func (r *affiliateDistributionRepository) TryBeginUsageSettlement(ctx context.Context, usageLogID int64) (bool, error) {
	retryBefore := time.Now().UTC().Add(-affiliateDistributionUsageJobClaimTTL)
	res, err := clientFromContext(ctx, r.client).ExecContext(ctx, `
INSERT INTO affiliate_distribution_usage_jobs (usage_log_id, status, claimed_at, updated_at, last_error)
VALUES ($1, 'processing', NOW(), NOW(), NULL)
ON CONFLICT (usage_log_id) DO UPDATE
SET status = 'processing',
    claimed_at = NOW(),
    updated_at = NOW(),
    last_error = NULL
WHERE affiliate_distribution_usage_jobs.status <> 'done'
  AND (
      affiliate_distribution_usage_jobs.status = 'failed'
      OR affiliate_distribution_usage_jobs.claimed_at < $2
      OR affiliate_distribution_usage_jobs.updated_at < $2
  )`, usageLogID, retryBefore)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (r *affiliateDistributionRepository) MarkUsageSettlementDone(ctx context.Context, usageLogID int64) error {
	_, err := clientFromContext(ctx, r.client).ExecContext(ctx, `
UPDATE affiliate_distribution_usage_jobs
SET status = 'done', last_error = NULL, updated_at = NOW()
WHERE usage_log_id = $1`, usageLogID)
	return err
}

func (r *affiliateDistributionRepository) MarkUsageSettlementFailed(ctx context.Context, usageLogID int64) error {
	_, err := clientFromContext(ctx, r.client).ExecContext(ctx, `
UPDATE affiliate_distribution_usage_jobs
SET status = 'failed',
    last_error = COALESCE(last_error, 'retryable settlement failure'),
    updated_at = NOW()
WHERE usage_log_id = $1`, usageLogID)
	return err
}

func (r *affiliateDistributionRepository) RecordPaidCredit(ctx context.Context, userID, sourceOrderID int64, amountUSD float64, creditedAt time.Time) (bool, error) {
	if userID <= 0 || sourceOrderID <= 0 || amountUSD <= 0 || math.IsNaN(amountUSD) || math.IsInf(amountUSD, 0) {
		return false, service.ErrAffiliateDistributionPaidCreditInvalidInput
	}
	if creditedAt.IsZero() {
		creditedAt = time.Now().UTC()
	} else {
		creditedAt = creditedAt.UTC()
	}
	var applied bool
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		existing, found, err := r.loadPaidCreditEventBySourceOrderID(txCtx, txClient, sourceOrderID)
		if err != nil {
			return err
		}
		if found {
			if sameAffiliateDistributionPaidCreditEvent(existing, userID, amountUSD, creditedAt) {
				return nil
			}
			return infraerrors.Conflict(
				"AFFILIATE_DISTRIBUTION_PAID_CREDIT_CONFLICT",
				fmt.Sprintf("source_order_id %d conflicts with existing paid credit event", sourceOrderID),
			)
		}
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if err := r.ensurePaidCreditBalanceRow(txCtx, txClient, userID); err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_paid_credit_events (
    user_id, source_order_id, amount_usd, credited_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, NOW(), NOW())`, userID, sourceOrderID, amountUSD, creditedAt); err != nil {
			if !isAffiliateDistributionUniqueViolation(err) {
				return err
			}
			existing, found, lookupErr := r.loadPaidCreditEventBySourceOrderID(txCtx, txClient, sourceOrderID)
			if lookupErr != nil {
				return lookupErr
			}
			if found && sameAffiliateDistributionPaidCreditEvent(existing, userID, amountUSD, creditedAt) {
				return nil
			}
			return infraerrors.Conflict(
				"AFFILIATE_DISTRIBUTION_PAID_CREDIT_CONFLICT",
				fmt.Sprintf("source_order_id %d conflicts with existing paid credit event", sourceOrderID),
			)
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE affiliate_distribution_paid_credit_balances
SET paid_credit_usd = paid_credit_usd + $1,
    last_credit_at = GREATEST(COALESCE(last_credit_at, $2), $2),
    updated_at = NOW()
WHERE user_id = $3`, amountUSD, creditedAt, userID); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

func (r *affiliateDistributionRepository) ReversePaidCredit(ctx context.Context, userID, sourceOrderID int64, amountUSD float64, reversedAt time.Time) (bool, error) {
	if userID <= 0 || sourceOrderID <= 0 || amountUSD <= 0 || math.IsNaN(amountUSD) || math.IsInf(amountUSD, 0) {
		return false, service.ErrAffiliateDistributionPaidCreditInvalidInput
	}
	if reversedAt.IsZero() {
		reversedAt = time.Now().UTC()
	} else {
		reversedAt = reversedAt.UTC()
	}

	var reversed bool
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		event, found, err := r.loadPaidCreditEventBySourceOrderID(txCtx, txClient, sourceOrderID)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		if event.UserID != userID {
			return infraerrors.Conflict(
				"AFFILIATE_DISTRIBUTION_PAID_CREDIT_CONFLICT",
				fmt.Sprintf("source_order_id %d belongs to a different paid credit user", sourceOrderID),
			)
		}
		if err := r.ensurePaidCreditBalanceRow(txCtx, txClient, userID); err != nil {
			return err
		}
		if err := r.lockPaidCreditBalanceRow(txCtx, txClient, userID); err != nil {
			return err
		}
		alreadyReversed, err := r.sumPaidCreditReversalsWithTx(txCtx, txClient, event.ID)
		if err != nil {
			return err
		}
		remainingCredit := event.AmountUSD - alreadyReversed
		if remainingCredit <= affiliateDistributionPaidCreditTolerance {
			return nil
		}
		reverseAmount := math.Min(amountUSD, remainingCredit)
		if reverseAmount <= affiliateDistributionPaidCreditTolerance || math.IsNaN(reverseAmount) || math.IsInf(reverseAmount, 0) {
			return nil
		}
		settledReversed, err := r.reversePaidCreditAllocationsWithTx(txCtx, txClient, event.ID, reverseAmount)
		if err != nil {
			return err
		}
		if err := r.insertPaidCreditReversalWithTx(txCtx, txClient, event.ID, sourceOrderID, userID, reverseAmount, reversedAt); err != nil {
			return err
		}
		if err := r.decrementPaidCreditBalanceForReversalWithTx(txCtx, txClient, userID, reverseAmount, settledReversed); err != nil {
			return err
		}
		reversed = true
		return nil
	})
	return reversed, err
}

func (r *affiliateDistributionRepository) SettleUsageDistribution(ctx context.Context, cmd service.AffiliateDistributionUsageSettlementCommand) error {
	if cmd.UsageLogID <= 0 || cmd.UserID <= 0 {
		return service.ErrAffiliateDistributionSettlementInvalidInput
	}
	if cmd.GroupID <= 0 {
		return service.ErrAffiliateDistributionGroupIDRequired
	}
	model := strings.TrimSpace(cmd.RequestedModel)
	if model == "" {
		model = strings.TrimSpace(cmd.Model)
	}
	if model == "" {
		return infraerrors.BadRequest("AFFILIATE_DISTRIBUTION_MODEL_REQUIRED", "model is required")
	}
	rawUsageUSD, paidUsageUSD, consumerRate, hasConsumerRate := affiliateDistributionSettlementAmounts(cmd)
	if math.IsNaN(rawUsageUSD) || math.IsInf(rawUsageUSD, 0) || math.IsNaN(paidUsageUSD) || math.IsInf(paidUsageUSD, 0) {
		return infraerrors.BadRequest("AFFILIATE_DISTRIBUTION_USAGE_INVALID", "usage amount must be finite")
	}
	if rawUsageUSD <= 0 || paidUsageUSD <= 0 {
		return nil
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		eligiblePaidUsageUSD, err := r.reservePaidUsageForSettlementWithTx(txCtx, txClient, cmd.UserID, cmd.UsageLogID, paidUsageUSD, cmd.UsageCreatedAt)
		if err != nil {
			return err
		}
		eligibleUsageUSD := affiliateDistributionProratedRawUsage(rawUsageUSD, paidUsageUSD, eligiblePaidUsageUSD)
		if eligibleUsageUSD <= 0 {
			logger.LegacyPrintf(
				"affiliate_distribution",
				"Usage settlement skipped: no paid credit available usage_log_id=%d user_id=%d group_id=%d model=%s raw_usage_usd=%.8f paid_usage_usd=%.8f eligible_paid_usage_usd=%.8f usage_created_at=%s",
				cmd.UsageLogID,
				cmd.UserID,
				cmd.GroupID,
				model,
				rawUsageUSD,
				paidUsageUSD,
				eligiblePaidUsageUSD,
				cmd.UsageCreatedAt.UTC().Format(time.RFC3339),
			)
			return nil
		}
		result, err := r.recordUsageSettlementWithTx(txCtx, txClient, service.RecordAffiliateUsageSettlementInput{
			UsageLogID:                cmd.UsageLogID,
			ConsumerUserID:            cmd.UserID,
			GroupID:                   cmd.GroupID,
			ModelKey:                  model,
			UsageAmountUSD:            eligibleUsageUSD,
			SettlementAt:              cmd.UsageCreatedAt,
			RootRate:                  0.0,
			ConsumerRateMultiplier:    consumerRate,
			HasConsumerRateMultiplier: hasConsumerRate,
		})
		if err != nil {
			return err
		}
		if result != nil && len(result.AppliedEntries) == 0 {
			return nil
		}
		return nil
	})
}

func (r *affiliateDistributionRepository) recordUsageSettlementWithTx(ctx context.Context, txClient *dbent.Client, input service.RecordAffiliateUsageSettlementInput) (*service.AffiliateDistributionSettlementResult, error) {
	result := &service.AffiliateDistributionSettlementResult{
		UsageLogID:     input.UsageLogID,
		ConsumerUserID: input.ConsumerUserID,
		ModelKey:       input.ModelKey,
		SettlementDay:  truncateToShanghaiDate(input.SettlementAt),
	}
	if _, err := ensureUserAffiliateWithClient(ctx, txClient, input.ConsumerUserID); err != nil {
		return nil, err
	}
	chain, err := r.loadAffiliateChain(ctx, txClient, input.ConsumerUserID)
	if err != nil {
		return nil, err
	}
	for depth := 1; depth < len(chain); depth++ {
		parent := chain[depth]
		child := chain[depth-1]
		parentRate, err := r.resolveGroupRateTx(ctx, txClient, parent.UserID, input.GroupID, input.RootRate)
		if err != nil {
			return nil, err
		}
		childRate, err := r.resolveGroupRateTx(ctx, txClient, child.UserID, input.GroupID, input.RootRate)
		if err != nil {
			return nil, err
		}
		if depth == 1 && input.HasConsumerRateMultiplier {
			childRate, err = r.resolveDirectConsumerChildRate(ctx, txClient, parent.UserID, child.UserID, input.GroupID, childRate, input.ConsumerRateMultiplier)
			if err != nil {
				return nil, err
			}
		}
		revenueAmountUSD := affiliateDistributionBusinessAmountUSD(input.UsageAmountUSD, childRate)
		rebateAmountRMB := affiliateDistributionRebateAmountRMB(input.UsageAmountUSD, parentRate, childRate)
		applied, err := r.insertSettlementRow(ctx, txClient, input, parent.UserID, child.UserID, int64(depth), parentRate, childRate, revenueAmountUSD, rebateAmountRMB)
		if err != nil {
			return nil, err
		}
		if applied {
			result.AppliedEntries = append(result.AppliedEntries, service.AffiliateDistributionSettlementEntry{
				BeneficiaryUserID:    parent.UserID,
				DirectChildUserID:    child.UserID,
				ConsumerUserID:       input.ConsumerUserID,
				ModelKey:             input.ModelKey,
				UsageAmountUSD:       input.UsageAmountUSD,
				RevenueAmountUSD:     revenueAmountUSD,
				ParentRateMultiplier: parentRate,
				ChildRateMultiplier:  childRate,
				RebateAmountRMB:      rebateAmountRMB,
				Depth:                depth,
				Applied:              true,
			})
		}
	}
	return result, nil
}

func (r *affiliateDistributionRepository) loadAffiliateChain(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) ([]affiliateDistributionTreeNode, error) {
	rows, err := client.QueryContext(ctx, `
WITH RECURSIVE chain AS (
    SELECT user_id, inviter_id, 0 AS depth
    FROM user_affiliates
    WHERE user_id = $1
    UNION ALL
    SELECT parent.user_id, parent.inviter_id, chain.depth + 1
    FROM user_affiliates parent
    JOIN chain ON chain.inviter_id = parent.user_id
)
SELECT user_id, inviter_id, depth
FROM chain
ORDER BY depth ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	nodes := make([]affiliateDistributionTreeNode, 0)
	for rows.Next() {
		var node affiliateDistributionTreeNode
		var inviterID sql.NullInt64
		if err := rows.Scan(&node.UserID, &inviterID, &node.Depth); err != nil {
			return nil, err
		}
		if inviterID.Valid {
			node.InviterID = &inviterID.Int64
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *affiliateDistributionRepository) resolveGroupRateTx(ctx context.Context, client affiliateDistributionQueryExecer, userID, groupID int64, rootRate float64) (float64, error) {
	return r.resolveGroupRateWithMemo(ctx, client, userID, groupID, rootRate, make(map[[2]int64]float64))
}

func (r *affiliateDistributionRepository) resolveDirectConsumerChildRate(ctx context.Context, client affiliateDistributionQueryExecer, parentUserID, childUserID, groupID int64, resolvedChildRate, consumerRate float64) (float64, error) {
	if rate, found, err := r.lookupExplicitUserGroupRate(ctx, client, childUserID, groupID); err != nil {
		return 0, err
	} else if found {
		return rate, nil
	}
	if rate, found, err := r.lookupInviteGroupRate(ctx, client, parentUserID, groupID); err != nil {
		return 0, err
	} else if found {
		return rate, nil
	}
	if rate, found, err := r.lookupDefaultUserGroupRate(ctx, client, groupID); err != nil {
		return 0, err
	} else if found {
		return rate, nil
	}
	if affiliateDistributionFiniteNonNegative(consumerRate) && consumerRate > 0 {
		return consumerRate, nil
	}
	return resolvedChildRate, nil
}

func (r *affiliateDistributionRepository) insertSettlementRow(ctx context.Context, client affiliateDistributionQueryExecer, input service.RecordAffiliateUsageSettlementInput, parentUserID, childUserID, depth int64, parentRate, childRate, revenueAmountUSD, rebateAmount float64) (bool, error) {
	res, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_usage_settlements (
    usage_log_id, settlement_key, beneficiary_user_id, direct_child_user_id, consumer_user_id, model_key,
    usage_amount_usd, revenue_amount_usd, parent_rate_multiplier, child_rate_multiplier, rebate_amount, settlement_day, depth, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
ON CONFLICT (usage_log_id, beneficiary_user_id) DO NOTHING`,
		input.UsageLogID,
		fmt.Sprintf("%d:%d:%s", input.UsageLogID, parentUserID, input.ModelKey),
		parentUserID,
		childUserID,
		input.ConsumerUserID,
		input.ModelKey,
		input.UsageAmountUSD,
		revenueAmountUSD,
		parentRate,
		childRate,
		rebateAmount,
		truncateToShanghaiDate(input.SettlementAt),
		depth,
	)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return false, nil
	}
	if _, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_daily_metrics (
    user_id, metric_date, revenue_amount_usd, rebate_amount, usage_count, last_usage_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, 1, $5, NOW(), NOW())
ON CONFLICT (user_id, metric_date)
DO UPDATE SET
    revenue_amount_usd = affiliate_distribution_daily_metrics.revenue_amount_usd + EXCLUDED.revenue_amount_usd,
    rebate_amount = affiliate_distribution_daily_metrics.rebate_amount + EXCLUDED.rebate_amount,
    usage_count = affiliate_distribution_daily_metrics.usage_count + 1,
    last_usage_at = GREATEST(affiliate_distribution_daily_metrics.last_usage_at, EXCLUDED.last_usage_at),
    updated_at = NOW()`,
		parentUserID, truncateToShanghaiDate(input.SettlementAt), revenueAmountUSD, rebateAmount, input.SettlementAt,
	); err != nil {
		return false, err
	}
	if err := r.ensureRebateBalanceRow(ctx, client, parentUserID); err != nil {
		return false, err
	}
	if _, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_rebate_balances
SET current_amount = current_amount + $1,
    lifetime_amount = lifetime_amount + $1,
    updated_at = NOW()
WHERE user_id = $2`, rebateAmount, parentUserID); err != nil {
		return false, err
	}
	return true, nil
}

func (r *affiliateDistributionRepository) reservePaidUsageForSettlementWithTx(ctx context.Context, client affiliateDistributionQueryExecer, userID, usageLogID int64, usageAmountUSD float64, settlementAt time.Time) (float64, error) {
	if userID <= 0 || usageLogID <= 0 || usageAmountUSD <= 0 || math.IsNaN(usageAmountUSD) || math.IsInf(usageAmountUSD, 0) {
		return 0, service.ErrAffiliateDistributionSettlementInvalidInput
	}
	if settlementAt.IsZero() {
		settlementAt = time.Now().UTC()
	} else {
		settlementAt = settlementAt.UTC()
	}
	if err := r.ensurePaidCreditBalanceRow(ctx, client, userID); err != nil {
		return 0, err
	}

	inserted, existing, err := r.insertPaidUsageSettlementPlaceholder(ctx, client, userID, usageLogID, settlementAt)
	if err != nil {
		return 0, err
	}
	if !inserted {
		return existing, nil
	}
	if err := r.lockPaidCreditBalanceRow(ctx, client, userID); err != nil {
		return 0, err
	}

	availableCredits, err := r.listAvailablePaidCreditEventsForSettlement(ctx, client, userID, settlementAt)
	if err != nil {
		return 0, err
	}
	remaining := usageAmountUSD
	eligible := 0.0
	for _, credit := range availableCredits {
		if remaining <= 0 {
			break
		}
		allocateUSD := math.Min(remaining, credit.RemainingUSD)
		if allocateUSD <= 0 || math.IsNaN(allocateUSD) || math.IsInf(allocateUSD, 0) {
			continue
		}
		if err := r.insertPaidCreditAllocationWithTx(ctx, client, credit.CreditEventID, usageLogID, userID, allocateUSD); err != nil {
			return 0, err
		}
		eligible += allocateUSD
		remaining -= allocateUSD
	}
	if eligible > 0 {
		if err := r.updatePaidUsageSettlementAmountWithTx(ctx, client, usageLogID, eligible); err != nil {
			return 0, err
		}
		if err := r.bumpPaidCreditSettlementBalanceWithTx(ctx, client, userID, eligible, settlementAt); err != nil {
			return 0, err
		}
	}
	if eligible <= 0 || math.IsNaN(eligible) || math.IsInf(eligible, 0) {
		return 0, nil
	}
	return eligible, nil
}

func (r *affiliateDistributionRepository) sumPaidCreditReversalsWithTx(ctx context.Context, client affiliateDistributionQueryExecer, creditEventID int64) (float64, error) {
	var total float64
	err := scanSingleRow(ctx, client, `
SELECT COALESCE(SUM(amount_usd), 0)::double precision
FROM affiliate_distribution_paid_credit_reversals
WHERE credit_event_id = $1`, []any{creditEventID}, &total)
	return total, err
}

func (r *affiliateDistributionRepository) insertPaidCreditReversalWithTx(ctx context.Context, client affiliateDistributionQueryExecer, creditEventID, sourceOrderID, userID int64, amountUSD float64, reversedAt time.Time) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_paid_credit_reversals (
    credit_event_id, source_order_id, user_id, amount_usd, reason, reversed_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, 'refund', $5, NOW(), NOW())`, creditEventID, sourceOrderID, userID, amountUSD, reversedAt)
	return err
}

func (r *affiliateDistributionRepository) decrementPaidCreditBalanceForReversalWithTx(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, creditReversedUSD, settledReversedUSD float64) error {
	_, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_paid_credit_balances
SET paid_credit_usd = GREATEST(paid_credit_usd - $1, 0),
    settled_paid_usage_usd = GREATEST(settled_paid_usage_usd - $2, 0),
    updated_at = NOW()
WHERE user_id = $3`, creditReversedUSD, settledReversedUSD, userID)
	return err
}

func (r *affiliateDistributionRepository) reversePaidCreditAllocationsWithTx(ctx context.Context, client affiliateDistributionQueryExecer, creditEventID int64, amountUSD float64) (float64, error) {
	allocations, err := r.listPaidCreditAllocationsForReverseWithTx(ctx, client, creditEventID)
	if err != nil {
		return 0, err
	}
	remaining := amountUSD
	settledReversed := 0.0
	for _, allocation := range allocations {
		if remaining <= affiliateDistributionPaidCreditTolerance {
			break
		}
		allocationReverse := math.Min(remaining, allocation.AmountUSD)
		if allocationReverse <= affiliateDistributionPaidCreditTolerance || math.IsNaN(allocationReverse) || math.IsInf(allocationReverse, 0) {
			continue
		}
		actualSettledReversed, err := r.reducePaidUsageSettlementForReversalWithTx(ctx, client, allocation.UsageLogID, allocationReverse)
		if err != nil {
			return 0, err
		}
		if err := r.reducePaidCreditAllocationWithTx(ctx, client, creditEventID, allocation.UsageLogID, allocationReverse); err != nil {
			return 0, err
		}
		settledReversed += actualSettledReversed
		remaining -= allocationReverse
	}
	return settledReversed, nil
}

func (r *affiliateDistributionRepository) listPaidCreditAllocationsForReverseWithTx(ctx context.Context, client affiliateDistributionQueryExecer, creditEventID int64) ([]affiliateDistributionPaidCreditAllocation, error) {
	rows, err := client.QueryContext(ctx, `
SELECT usage_log_id, amount_usd::double precision
FROM affiliate_distribution_paid_credit_allocations
WHERE credit_event_id = $1
ORDER BY created_at DESC, usage_log_id DESC
FOR UPDATE`, creditEventID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	allocations := make([]affiliateDistributionPaidCreditAllocation, 0)
	for rows.Next() {
		var allocation affiliateDistributionPaidCreditAllocation
		if err := rows.Scan(&allocation.UsageLogID, &allocation.AmountUSD); err != nil {
			return nil, err
		}
		allocations = append(allocations, allocation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allocations, nil
}

func (r *affiliateDistributionRepository) reducePaidCreditAllocationWithTx(ctx context.Context, client affiliateDistributionQueryExecer, creditEventID, usageLogID int64, amountUSD float64) error {
	var current float64
	err := scanSingleRow(ctx, client, `
SELECT amount_usd::double precision
FROM affiliate_distribution_paid_credit_allocations
WHERE credit_event_id = $1 AND usage_log_id = $2
FOR UPDATE`, []any{creditEventID, usageLogID}, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if current-amountUSD <= affiliateDistributionPaidCreditTolerance {
		_, err = client.ExecContext(ctx, `
DELETE FROM affiliate_distribution_paid_credit_allocations
WHERE credit_event_id = $1 AND usage_log_id = $2`, creditEventID, usageLogID)
		return err
	}
	_, err = client.ExecContext(ctx, `
UPDATE affiliate_distribution_paid_credit_allocations
SET amount_usd = amount_usd - $1,
    updated_at = NOW()
WHERE credit_event_id = $2 AND usage_log_id = $3`, amountUSD, creditEventID, usageLogID)
	return err
}

func (r *affiliateDistributionRepository) reducePaidUsageSettlementForReversalWithTx(ctx context.Context, client affiliateDistributionQueryExecer, usageLogID int64, amountUSD float64) (float64, error) {
	var current float64
	err := scanSingleRow(ctx, client, `
SELECT usage_amount_usd::double precision
FROM affiliate_distribution_paid_usage_settlements
WHERE usage_log_id = $1
FOR UPDATE`, []any{usageLogID}, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if current <= affiliateDistributionPaidCreditTolerance {
		return 0, nil
	}
	actualReversal := math.Min(amountUSD, current)
	if actualReversal <= affiliateDistributionPaidCreditTolerance || math.IsNaN(actualReversal) || math.IsInf(actualReversal, 0) {
		return 0, nil
	}
	if err := r.applyUsageSettlementReversalWithTx(ctx, client, usageLogID, actualReversal, current); err != nil {
		return 0, err
	}
	_, err = client.ExecContext(ctx, `
UPDATE affiliate_distribution_paid_usage_settlements
SET usage_amount_usd = GREATEST(usage_amount_usd - $1, 0),
    updated_at = NOW()
WHERE usage_log_id = $2`, actualReversal, usageLogID)
	if err != nil {
		return 0, err
	}
	return actualReversal, nil
}

func (r *affiliateDistributionRepository) applyUsageSettlementReversalWithTx(ctx context.Context, client affiliateDistributionQueryExecer, usageLogID int64, amountUSD, previousPaidUsageUSD float64) error {
	if previousPaidUsageUSD <= affiliateDistributionPaidCreditTolerance {
		return nil
	}
	adjustments, err := r.loadSettlementAdjustmentsForReversalWithTx(ctx, client, usageLogID)
	if err != nil {
		return err
	}
	if len(adjustments) == 0 {
		return nil
	}
	ratio := amountUSD / previousPaidUsageUSD
	if ratio > 1 {
		ratio = 1
	}
	if ratio <= affiliateDistributionPaidCreditTolerance || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return nil
	}
	for _, adjustment := range adjustments {
		usageDelta := math.Min(adjustment.UsageAmountUSD, adjustment.UsageAmountUSD*ratio)
		revenueDelta := math.Min(adjustment.RevenueAmountUSD, adjustment.RevenueAmountUSD*ratio)
		rebateDelta := math.Min(adjustment.RebateAmountRMB, adjustment.RebateAmountRMB*ratio)
		fullReversal := ratio >= 1-affiliateDistributionPaidCreditTolerance || adjustment.UsageAmountUSD-usageDelta <= affiliateDistributionPaidCreditTolerance
		if err := r.applySettlementAdjustmentReversalWithTx(ctx, client, usageLogID, adjustment, usageDelta, revenueDelta, rebateDelta, fullReversal); err != nil {
			return err
		}
	}
	return nil
}

func (r *affiliateDistributionRepository) loadSettlementAdjustmentsForReversalWithTx(ctx context.Context, client affiliateDistributionQueryExecer, usageLogID int64) ([]affiliateDistributionSettlementAdjustment, error) {
	rows, err := client.QueryContext(ctx, `
SELECT beneficiary_user_id,
       settlement_day,
       usage_amount_usd::double precision,
       revenue_amount_usd::double precision,
       rebate_amount::double precision
FROM affiliate_distribution_usage_settlements
WHERE usage_log_id = $1
FOR UPDATE`, usageLogID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	adjustments := make([]affiliateDistributionSettlementAdjustment, 0)
	for rows.Next() {
		var adjustment affiliateDistributionSettlementAdjustment
		if err := rows.Scan(
			&adjustment.BeneficiaryUserID,
			&adjustment.SettlementDay,
			&adjustment.UsageAmountUSD,
			&adjustment.RevenueAmountUSD,
			&adjustment.RebateAmountRMB,
		); err != nil {
			return nil, err
		}
		adjustments = append(adjustments, adjustment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return adjustments, nil
}

func (r *affiliateDistributionRepository) applySettlementAdjustmentReversalWithTx(ctx context.Context, client affiliateDistributionQueryExecer, usageLogID int64, adjustment affiliateDistributionSettlementAdjustment, usageDelta, revenueDelta, rebateDelta float64, fullReversal bool) error {
	if fullReversal {
		if _, err := client.ExecContext(ctx, `
DELETE FROM affiliate_distribution_usage_settlements
WHERE usage_log_id = $1 AND beneficiary_user_id = $2`, usageLogID, adjustment.BeneficiaryUserID); err != nil {
			return err
		}
	} else {
		if _, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_usage_settlements
SET usage_amount_usd = GREATEST(usage_amount_usd - $1, 0),
    revenue_amount_usd = GREATEST(revenue_amount_usd - $2, 0),
    rebate_amount = GREATEST(rebate_amount - $3, 0),
    updated_at = NOW()
WHERE usage_log_id = $4 AND beneficiary_user_id = $5`, usageDelta, revenueDelta, rebateDelta, usageLogID, adjustment.BeneficiaryUserID); err != nil {
			return err
		}
	}
	usageCountDelta := 0
	if fullReversal {
		usageCountDelta = 1
	}
	if _, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_daily_metrics
SET revenue_amount_usd = GREATEST(revenue_amount_usd - $1, 0),
    rebate_amount = GREATEST(rebate_amount - $2, 0),
    usage_count = GREATEST(usage_count - $3, 0),
    updated_at = NOW()
WHERE user_id = $4 AND metric_date = $5`, revenueDelta, rebateDelta, usageCountDelta, adjustment.BeneficiaryUserID, adjustment.SettlementDay); err != nil {
		return err
	}
	if rebateDelta <= affiliateDistributionPaidCreditTolerance {
		return nil
	}
	_, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_rebate_balances
SET current_amount = GREATEST(current_amount - $1, 0),
    lifetime_amount = GREATEST(lifetime_amount - $1, 0),
    updated_at = NOW()
WHERE user_id = $2`, rebateDelta, adjustment.BeneficiaryUserID)
	return err
}

func (r *affiliateDistributionRepository) getDailyTotals(ctx context.Context, userID int64, day time.Time) (float64, float64, error) {
	var revenue float64
	var rebate float64
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
SELECT COALESCE(SUM(usage_amount_usd), 0)::double precision,
       COALESCE(SUM(rebate_amount), 0)::double precision
FROM affiliate_distribution_usage_settlements
WHERE beneficiary_user_id = $1 AND settlement_day = $2`, []any{userID, day}, &revenue, &rebate)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return revenue, rebate, err
}

func (r *affiliateDistributionRepository) getCurrentRebateBalance(ctx context.Context, userID int64) (float64, error) {
	var balance float64
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
SELECT current_amount::double precision
FROM affiliate_distribution_rebate_balances
WHERE user_id = $1
LIMIT 1`, []any{userID}, &balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return balance, err
}

func (r *affiliateDistributionRepository) isAdminUser(ctx context.Context, userID int64) (bool, error) {
	var isAdmin bool
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
SELECT (role = 'admin')
FROM users
WHERE id = $1
LIMIT 1`, []any{userID}, &isAdmin)
	return isAdmin, err
}

func (r *affiliateDistributionRepository) ensureRootAdmin(ctx context.Context, operatorUserID int64) error {
	var rootAdminID int64
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
SELECT id
FROM users
WHERE role = $1 AND status = $2
ORDER BY id ASC
LIMIT 1`, []any{service.RoleAdmin, service.StatusActive}, &rootAdminID)
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.Forbidden("ROOT_ADMIN_REQUIRED", "root admin access required")
	}
	if err != nil {
		return err
	}
	if rootAdminID != operatorUserID {
		return infraerrors.Forbidden("ROOT_ADMIN_REQUIRED", "root admin access required")
	}
	return nil
}

func (r *affiliateDistributionRepository) ensureDirectSubordinate(ctx context.Context, userID, subordinateUserID int64) error {
	var exists bool
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
SELECT TRUE
FROM user_affiliates
WHERE user_id = $1 AND inviter_id = $2
LIMIT 1`, []any{subordinateUserID, userID}, &exists)
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.Forbidden("AFFILIATE_DISTRIBUTION_NOT_DIRECT_SUBORDINATE", "subordinate is not directly bound to current agent")
	}
	return err
}

func (r *affiliateDistributionRepository) ensureDirectOrAdmin(ctx context.Context, operatorUserID, targetUserID int64) error {
	isAdmin, err := r.isAdminUser(ctx, operatorUserID)
	if err != nil {
		return err
	}
	if isAdmin {
		return nil
	}
	return r.ensureDescendant(ctx, operatorUserID, targetUserID)
}

func (r *affiliateDistributionRepository) ensureDescendant(ctx context.Context, operatorUserID, targetUserID int64) error {
	if operatorUserID == targetUserID {
		return infraerrors.Forbidden("AFFILIATE_DISTRIBUTION_NOT_DESCENDANT", "target user is not a descendant of current agent")
	}
	var exists bool
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
WITH RECURSIVE descendants AS (
    SELECT ua.user_id, 0 AS depth
    FROM user_affiliates ua
    WHERE ua.user_id = $1
    UNION ALL
    SELECT child.user_id, descendants.depth + 1
    FROM user_affiliates child
    JOIN descendants ON child.inviter_id = descendants.user_id
)
SELECT TRUE
FROM descendants
WHERE user_id = $2
  AND depth > 0
LIMIT 1`, []any{operatorUserID, targetUserID}, &exists)
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.Forbidden("AFFILIATE_DISTRIBUTION_NOT_DESCENDANT", "target user is not a descendant of current agent")
	}
	return err
}

func (r *affiliateDistributionRepository) ensureRebateBalanceRow(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_rebate_balances (user_id, current_amount, lifetime_amount, created_at, updated_at)
VALUES ($1, 0, 0, NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

func (r *affiliateDistributionRepository) ensurePaidCreditBalanceRow(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_paid_credit_balances (user_id, paid_credit_usd, settled_paid_usage_usd, created_at, updated_at)
VALUES ($1, 0, 0, NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

type affiliateDistributionPaidCreditEvent struct {
	ID         int64
	UserID     int64
	AmountUSD  float64
	CreditedAt time.Time
}

type affiliateDistributionPaidCreditAvailability struct {
	CreditEventID int64
	RemainingUSD  float64
}

type affiliateDistributionPaidCreditAllocation struct {
	UsageLogID int64
	AmountUSD  float64
}

type affiliateDistributionSettlementAdjustment struct {
	BeneficiaryUserID int64
	SettlementDay     time.Time
	UsageAmountUSD    float64
	RevenueAmountUSD  float64
	RebateAmountRMB   float64
}

func (r *affiliateDistributionRepository) loadPaidCreditEventBySourceOrderID(ctx context.Context, client affiliateDistributionQueryExecer, sourceOrderID int64) (affiliateDistributionPaidCreditEvent, bool, error) {
	var existing affiliateDistributionPaidCreditEvent
	err := scanSingleRow(ctx, client, `
SELECT id, user_id, amount_usd::double precision, credited_at
FROM affiliate_distribution_paid_credit_events
WHERE source_order_id = $1
LIMIT 1`, []any{sourceOrderID}, &existing.ID, &existing.UserID, &existing.AmountUSD, &existing.CreditedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return affiliateDistributionPaidCreditEvent{}, false, nil
	}
	if err != nil {
		return affiliateDistributionPaidCreditEvent{}, false, err
	}
	return existing, true, nil
}

func (r *affiliateDistributionRepository) insertPaidUsageSettlementPlaceholder(ctx context.Context, client affiliateDistributionQueryExecer, userID, usageLogID int64, settlementAt time.Time) (bool, float64, error) {
	res, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_paid_usage_settlements (
    usage_log_id, user_id, usage_amount_usd, settlement_day, settled_at, created_at, updated_at
) VALUES ($1, $2, 0, $3, $4, NOW(), NOW())
ON CONFLICT (usage_log_id) DO NOTHING`, usageLogID, userID, truncateToShanghaiDate(settlementAt), settlementAt)
	if err != nil {
		return false, 0, err
	}
	affected, _ := res.RowsAffected()
	if affected > 0 {
		return true, 0, nil
	}

	var existing float64
	if err := scanSingleRow(ctx, client, `
SELECT COALESCE(usage_amount_usd, 0)::double precision
FROM affiliate_distribution_paid_usage_settlements
WHERE usage_log_id = $1
LIMIT 1`, []any{usageLogID}, &existing); err != nil {
		return false, 0, err
	}
	return false, existing, nil
}

func (r *affiliateDistributionRepository) lockPaidCreditBalanceRow(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) error {
	var locked bool
	if err := scanSingleRow(ctx, client, `
SELECT TRUE
FROM affiliate_distribution_paid_credit_balances
WHERE user_id = $1
FOR UPDATE`, []any{userID}, &locked); err != nil {
		return err
	}
	return nil
}

func (r *affiliateDistributionRepository) listAvailablePaidCreditEventsForSettlement(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, settlementAt time.Time) ([]affiliateDistributionPaidCreditAvailability, error) {
	rows, err := client.QueryContext(ctx, `
SELECT e.id,
       GREATEST(
           e.amount_usd - COALESCE(rev.reversed_usd, 0) - COALESCE(alloc.allocated_usd, 0),
           0
       )::double precision AS remaining_usd
FROM affiliate_distribution_paid_credit_events e
LEFT JOIN (
    SELECT credit_event_id, COALESCE(SUM(amount_usd), 0)::double precision AS reversed_usd
    FROM affiliate_distribution_paid_credit_reversals
    GROUP BY credit_event_id
) rev ON rev.credit_event_id = e.id
LEFT JOIN (
    SELECT credit_event_id, COALESCE(SUM(amount_usd), 0)::double precision AS allocated_usd
    FROM affiliate_distribution_paid_credit_allocations
    GROUP BY credit_event_id
) alloc ON alloc.credit_event_id = e.id
WHERE e.user_id = $1
  AND e.credited_at <= $2
  AND GREATEST(e.amount_usd - COALESCE(rev.reversed_usd, 0) - COALESCE(alloc.allocated_usd, 0), 0) > 0
ORDER BY e.credited_at ASC, e.id ASC`, userID, settlementAt)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	credits := make([]affiliateDistributionPaidCreditAvailability, 0)
	for rows.Next() {
		var item affiliateDistributionPaidCreditAvailability
		if err := rows.Scan(&item.CreditEventID, &item.RemainingUSD); err != nil {
			return nil, err
		}
		credits = append(credits, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return credits, nil
}

func (r *affiliateDistributionRepository) insertPaidCreditAllocationWithTx(ctx context.Context, client affiliateDistributionQueryExecer, creditEventID, usageLogID, userID int64, amountUSD float64) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_paid_credit_allocations (
    credit_event_id, usage_log_id, user_id, amount_usd, created_at, updated_at
) VALUES ($1, $2, $3, $4, NOW(), NOW())`, creditEventID, usageLogID, userID, amountUSD)
	return err
}

func (r *affiliateDistributionRepository) updatePaidUsageSettlementAmountWithTx(ctx context.Context, client affiliateDistributionQueryExecer, usageLogID int64, amountUSD float64) error {
	_, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_paid_usage_settlements
SET usage_amount_usd = $1,
    updated_at = NOW()
WHERE usage_log_id = $2`, amountUSD, usageLogID)
	return err
}

func (r *affiliateDistributionRepository) bumpPaidCreditSettlementBalanceWithTx(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, amountUSD float64, settlementAt time.Time) error {
	_, err := client.ExecContext(ctx, `
UPDATE affiliate_distribution_paid_credit_balances
SET settled_paid_usage_usd = settled_paid_usage_usd + $1,
    last_settlement_at = GREATEST(COALESCE(last_settlement_at, $2), $2),
    updated_at = NOW()
WHERE user_id = $3`, amountUSD, settlementAt, userID)
	return err
}

func sameAffiliateDistributionPaidCreditEvent(existing affiliateDistributionPaidCreditEvent, userID int64, amountUSD float64, creditedAt time.Time) bool {
	if existing.UserID != userID {
		return false
	}
	if math.Abs(existing.AmountUSD-amountUSD) > affiliateDistributionPaidCreditTolerance {
		return false
	}
	return existing.CreditedAt.UTC().Equal(creditedAt.UTC())
}

func isAffiliateDistributionUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	return false
}

func normalizeAgentGroupRateInputs(rates []service.AgentGroupRateInput) []service.AgentGroupRateInput {
	if len(rates) == 0 {
		return []service.AgentGroupRateInput{}
	}
	normalized := make([]service.AgentGroupRateInput, 0, len(rates))
	for _, rate := range rates {
		if rate.GroupID <= 0 {
			continue
		}
		normalized = append(normalized, service.AgentGroupRateInput{GroupID: rate.GroupID, RateMultiplier: rate.RateMultiplier})
	}
	return normalized
}

func validateAndNormalizeDistributionGroupRateInputs(rates []service.AgentGroupRateInput) ([]service.AgentGroupRateInput, error) {
	if len(rates) == 0 {
		return []service.AgentGroupRateInput{}, nil
	}
	normalized := make([]service.AgentGroupRateInput, 0, len(rates))
	for _, rate := range rates {
		if rate.GroupID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_GROUP_RATES", "group_id must be greater than 0")
		}
		if err := validateAffiliateDistributionRateMultiplierValue(rate.RateMultiplier); err != nil {
			return nil, err
		}
		normalized = append(normalized, service.AgentGroupRateInput{
			GroupID:        rate.GroupID,
			RateMultiplier: rate.RateMultiplier,
		})
	}
	return normalized, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func (r *affiliateDistributionRepository) lookupDefaultUserGroupRate(ctx context.Context, client affiliateDistributionQueryExecer, groupID int64) (float64, bool, error) {
	var rate float64
	err := scanSingleRow(ctx, client, `
SELECT rate_multiplier::double precision
FROM affiliate_distribution_default_user_group_rates
WHERE group_id = $1
LIMIT 1`, []any{groupID}, &rate)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if err := validateAffiliateDistributionRateMultiplierValue(rate); err != nil {
		return 0, false, err
	}
	return rate, true, nil
}

func (r *affiliateDistributionRepository) lookupGroupDefaultRate(ctx context.Context, client affiliateDistributionQueryExecer, groupID int64) (float64, bool, error) {
	var rate float64
	err := scanSingleRow(ctx, client, `
SELECT rate_multiplier::double precision
FROM groups
WHERE id = $1
  AND deleted_at IS NULL
  AND status = 'active'
LIMIT 1`, []any{groupID}, &rate)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if err := validateAffiliateDistributionRateMultiplierValue(rate); err != nil {
		return 0, false, err
	}
	return rate, true, nil
}

func (r *affiliateDistributionRepository) listDefaultUserGroupRatesWithClient(ctx context.Context, client affiliateDistributionQueryExecer) ([]service.AgentGroupRate, error) {
	rows, err := client.QueryContext(ctx, `
SELECT g.id,
       g.name,
       g.platform,
       g.rate_multiplier::double precision,
       COALESCE(r.rate_multiplier, g.rate_multiplier)::double precision,
       CASE WHEN r.group_id IS NOT NULL THEN 'default_explicit' ELSE 'group_default' END,
       ''::varchar,
       NULL::bigint,
       r.updated_at
FROM groups g
LEFT JOIN affiliate_distribution_default_user_group_rates r
       ON r.group_id = g.id
WHERE g.deleted_at IS NULL AND g.status = 'active'
ORDER BY g.sort_order ASC, g.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list default user group rates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanAgentGroupRates(rows, nil)
}

func (r *affiliateDistributionRepository) ensureUserDistributionPricingWithClient(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) error {
	_, err := ensureUserAffiliateWithClient(ctx, client, userID)
	return err
}

func validateAffiliateDistributionRateMultiplierValue(multiplier float64) error {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 || multiplier > affiliateDistributionRateMultiplierMax {
		return infraerrors.BadRequest("INVALID_GROUP_RATES", "rate_multiplier must be a finite number greater than 0 and at most 100")
	}
	return nil
}

func isExplicitDistributionUserGroupRateSource(sourceType string) bool {
	return sourceType == affiliateDistributionSourceAdminOverride || sourceType == affiliateDistributionSourceUpstreamOverride
}

func (r *affiliateDistributionRepository) getUserDistributionGroupRatesWithClient(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, memo map[int64][]service.AgentGroupRate) ([]service.AgentGroupRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if cached, ok := memo[userID]; ok {
		return append([]service.AgentGroupRate(nil), cached...), nil
	}

	summary, err := queryAffiliateByUserID(ctx, client, userID)
	if err != nil {
		return nil, err
	}

	var inviterArg any
	parentAffCode := ""
	parentRatesByGroup := make(map[int64]service.AgentGroupRate)
	if summary.InviterID != nil {
		inviterArg = *summary.InviterID
		parentSummary, err := queryAffiliateByUserID(ctx, client, *summary.InviterID)
		if err != nil {
			return nil, err
		}
		parentAffCode = parentSummary.AffCode
		parentRates, err := r.getUserDistributionGroupRatesWithClient(ctx, client, *summary.InviterID, memo)
		if err != nil {
			return nil, err
		}
		for _, rate := range parentRates {
			parentRatesByGroup[rate.GroupID] = rate
		}
	}

	rows, err := client.QueryContext(ctx, `
SELECT g.id,
       g.name,
       g.platform,
       g.rate_multiplier::double precision,
       ug.rate_multiplier::double precision,
       ug.source_type,
       COALESCE(ug.source_aff_code, ''),
       ug.upstream_user_id,
       ug.updated_at,
       ig.rate_multiplier::double precision,
       dg.rate_multiplier::double precision
FROM groups g
LEFT JOIN affiliate_distribution_user_group_rates ug
       ON ug.group_id = g.id
      AND ug.user_id = $1
      AND ug.source_type IN ('admin_override', 'upstream_override')
LEFT JOIN affiliate_distribution_invite_group_rates ig
       ON ig.group_id = g.id
      AND ig.inviter_user_id = $2
LEFT JOIN affiliate_distribution_default_user_group_rates dg
       ON dg.group_id = g.id
WHERE g.deleted_at IS NULL AND g.status = 'active'
ORDER BY g.sort_order ASC, g.id ASC`, userID, inviterArg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AgentGroupRate, 0)
	for rows.Next() {
		var item service.AgentGroupRate
		var explicitRate sql.NullFloat64
		var explicitSourceType sql.NullString
		var explicitSourceCode string
		var explicitUpstreamUserID sql.NullInt64
		var explicitUpdatedAt sql.NullTime
		var inviteRate sql.NullFloat64
		var defaultRate sql.NullFloat64

		if err := rows.Scan(
			&item.GroupID,
			&item.GroupName,
			&item.GroupPlatform,
			&item.GroupDefaultRateMultiplier,
			&explicitRate,
			&explicitSourceType,
			&explicitSourceCode,
			&explicitUpstreamUserID,
			&explicitUpdatedAt,
			&inviteRate,
			&defaultRate,
		); err != nil {
			return nil, err
		}
		if err := validateAffiliateDistributionRateMultiplierValue(item.GroupDefaultRateMultiplier); err != nil {
			return nil, err
		}

		switch {
		case explicitRate.Valid:
			if err := validateAffiliateDistributionRateMultiplierValue(explicitRate.Float64); err != nil {
				return nil, err
			}
			item.RateMultiplier = explicitRate.Float64
			item.SourceType = explicitSourceType.String
			item.SourceAffCode = explicitSourceCode
			if explicitUpstreamUserID.Valid {
				upstreamUserID := explicitUpstreamUserID.Int64
				item.UpstreamUserID = &upstreamUserID
			}
			if explicitUpdatedAt.Valid {
				updatedAt := explicitUpdatedAt.Time
				item.UpdatedAt = &updatedAt
			}
		case inviteRate.Valid:
			if err := validateAffiliateDistributionRateMultiplierValue(inviteRate.Float64); err != nil {
				return nil, err
			}
			item.RateMultiplier = inviteRate.Float64
			item.SourceType = affiliateDistributionSourceInviteCode
			item.SourceAffCode = parentAffCode
			if summary.InviterID != nil {
				item.UpstreamUserID = summary.InviterID
			}
		case defaultRate.Valid:
			if err := validateAffiliateDistributionRateMultiplierValue(defaultRate.Float64); err != nil {
				return nil, err
			}
			item.RateMultiplier = defaultRate.Float64
			item.SourceType = affiliateDistributionSourceDefaultExplicit
			item.SourceAffCode = parentAffCode
			if summary.InviterID != nil {
				item.UpstreamUserID = summary.InviterID
			}
		case summary.InviterID != nil:
			parentRate, ok := parentRatesByGroup[item.GroupID]
			if ok {
				item.RateMultiplier = parentRate.RateMultiplier
			} else {
				item.RateMultiplier = item.GroupDefaultRateMultiplier
			}
			if err := validateAffiliateDistributionRateMultiplierValue(item.RateMultiplier); err != nil {
				return nil, err
			}
			item.SourceType = affiliateDistributionSourceGroupInherited
			item.SourceAffCode = parentAffCode
			item.UpstreamUserID = summary.InviterID
		default:
			item.RateMultiplier = item.GroupDefaultRateMultiplier
			item.SourceType = affiliateDistributionSourceGroupDefault
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	memo[userID] = append([]service.AgentGroupRate(nil), items...)
	return append([]service.AgentGroupRate(nil), items...), nil
}

func (r *affiliateDistributionRepository) lookupExplicitUserGroupRate(ctx context.Context, client affiliateDistributionQueryExecer, userID, groupID int64) (float64, bool, error) {
	var rate float64
	var sourceType string
	err := scanSingleRow(ctx, client, `
SELECT rate_multiplier::double precision,
       source_type
FROM affiliate_distribution_user_group_rates
WHERE user_id = $1
  AND group_id = $2
LIMIT 1`, []any{userID, groupID}, &rate, &sourceType)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !isExplicitDistributionUserGroupRateSource(sourceType) {
		return 0, false, nil
	}
	if err := validateAffiliateDistributionRateMultiplierValue(rate); err != nil {
		return 0, false, err
	}
	return rate, true, nil
}

func (r *affiliateDistributionRepository) lookupInviteGroupRate(ctx context.Context, client affiliateDistributionQueryExecer, inviterUserID, groupID int64) (float64, bool, error) {
	var rate float64
	err := scanSingleRow(ctx, client, `
SELECT rate_multiplier::double precision
FROM affiliate_distribution_invite_group_rates
WHERE inviter_user_id = $1
  AND group_id = $2
LIMIT 1`, []any{inviterUserID, groupID}, &rate)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if err := validateAffiliateDistributionRateMultiplierValue(rate); err != nil {
		return 0, false, err
	}
	return rate, true, nil
}

func (r *affiliateDistributionRepository) resolveGroupRateWithMemo(ctx context.Context, client affiliateDistributionQueryExecer, userID, groupID int64, rootRate float64, memo map[[2]int64]float64) (float64, error) {
	key := [2]int64{userID, groupID}
	if rate, ok := memo[key]; ok {
		return rate, nil
	}
	if err := r.ensureUserDistributionPricingWithClient(ctx, client, userID); err != nil {
		return 0, err
	}

	if rate, found, err := r.lookupExplicitUserGroupRate(ctx, client, userID, groupID); err != nil {
		return 0, err
	} else if found {
		memo[key] = rate
		return rate, nil
	}

	summary, err := queryAffiliateByUserID(ctx, client, userID)
	if err != nil {
		return 0, err
	}
	if summary.InviterID == nil {
		if rate, found, err := r.lookupDefaultUserGroupRate(ctx, client, groupID); err != nil {
			return 0, err
		} else if found {
			memo[key] = rate
			return rate, nil
		}
		if rate, found, err := r.lookupGroupDefaultRate(ctx, client, groupID); err != nil {
			return 0, err
		} else if found {
			memo[key] = rate
			return rate, nil
		}
		if !affiliateDistributionFiniteNonNegative(rootRate) || rootRate <= 0 {
			rootRate = 1
		}
		if err := validateAffiliateDistributionRateMultiplierValue(rootRate); err != nil {
			return 0, err
		}
		memo[key] = rootRate
		return rootRate, nil
	}

	if rate, found, err := r.lookupInviteGroupRate(ctx, client, *summary.InviterID, groupID); err != nil {
		return 0, err
	} else if found {
		memo[key] = rate
		return rate, nil
	}

	if rate, found, err := r.lookupDefaultUserGroupRate(ctx, client, groupID); err != nil {
		return 0, err
	} else if found {
		memo[key] = rate
		return rate, nil
	}

	rate, err := r.resolveGroupRateWithMemo(ctx, client, *summary.InviterID, groupID, rootRate, memo)
	if err != nil {
		return 0, err
	}
	memo[key] = rate
	return rate, nil
}

func (r *affiliateDistributionRepository) saveUserDistributionGroupRates(ctx context.Context, userID int64, rates []service.AgentGroupRateInput, sourceType string) ([]service.AgentGroupRate, error) {
	normalized, err := validateAndNormalizeDistributionGroupRateInputs(rates)
	if err != nil {
		return nil, err
	}
	err = r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		summary, err := queryAffiliateByUserID(txCtx, txClient, userID)
		if err != nil {
			return err
		}
		sourceCode := ""
		if _, err := txClient.ExecContext(txCtx, `DELETE FROM affiliate_distribution_user_group_rates WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("reset user distribution group rates: %w", err)
		}
		if summary.InviterID != nil {
			parentSummary, err := queryAffiliateByUserID(txCtx, txClient, *summary.InviterID)
			if err != nil {
				return err
			}
			sourceCode = parentSummary.AffCode
		}
		for _, rate := range normalized {
			if err := r.upsertUserGroupRateTx(txCtx, txClient, userID, summary.InviterID, rate.GroupID, rate.RateMultiplier, sourceCode, sourceType); err != nil {
				return fmt.Errorf("save user distribution group rate: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetUserDistributionGroupRates(ctx, userID)
}

func (r *affiliateDistributionRepository) ensureNoInviterCycle(ctx context.Context, client affiliateDistributionQueryExecer, userID, inviterID int64) error {
	var exists bool
	err := scanSingleRow(ctx, client, `
WITH RECURSIVE descendants AS (
    SELECT ua.user_id
    FROM user_affiliates ua
    WHERE ua.user_id = $1
    UNION ALL
    SELECT child.user_id
    FROM user_affiliates child
    JOIN descendants ON child.inviter_id = descendants.user_id
)
SELECT TRUE
FROM descendants
WHERE user_id = $2
LIMIT 1`, []any{userID, inviterID}, &exists)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return infraerrors.BadRequest("AFFILIATE_DISTRIBUTION_UPSTREAM_CYCLE", "inviter assignment would create a cycle")
}

func sameNullableInt64(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (r *affiliateDistributionRepository) upsertUserGroupRateTx(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, upstreamUserID *int64, groupID int64, multiplier float64, sourceAffCode, sourceType string) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_user_group_rates (
    user_id, upstream_user_id, group_id, rate_multiplier, source_aff_code, source_type, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (user_id, group_id)
DO UPDATE SET
    upstream_user_id = EXCLUDED.upstream_user_id,
    rate_multiplier = EXCLUDED.rate_multiplier,
    source_aff_code = EXCLUDED.source_aff_code,
    source_type = EXCLUDED.source_type,
    updated_at = NOW()`,
		userID, nullableInt64Arg(upstreamUserID), groupID, multiplier, nullableString(sourceAffCode), sourceType,
	)
	return err
}

func (r *affiliateDistributionRepository) getAgentDistributionPermissionsWithClient(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) (*service.AgentDistributionPermission, error) {
	if err := ensureUserExists(ctx, client, userID); err != nil {
		return nil, err
	}
	permission := &service.AgentDistributionPermission{UserID: userID}
	var grantedByUserID sql.NullInt64
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := scanSingleRow(ctx, client, `
SELECT can_view_downline_daily_revenue,
       can_view_downline_rebate_balances,
       can_manage_downline_pricing,
       granted_by_user_id,
       created_at,
       updated_at
FROM affiliate_distribution_agent_permissions
WHERE user_id = $1
LIMIT 1`, []any{userID},
		&permission.CanViewDownlineDailyRevenue,
		&permission.CanViewDownlineRebateBalances,
		&permission.CanManageDownlinePricing,
		&grantedByUserID,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return permission, nil
	}
	if err != nil {
		return nil, err
	}
	if grantedByUserID.Valid {
		permission.GrantedByUserID = &grantedByUserID.Int64
	}
	if createdAt.Valid {
		ts := createdAt.Time
		permission.CreatedAt = &ts
	}
	if updatedAt.Valid {
		ts := updatedAt.Time
		permission.UpdatedAt = &ts
	}
	return permission, nil
}

func buildAffiliateScopeCTE(rootUserID *int64, onlyDescendants bool, placeholderIndex int) (string, []any, int) {
	if rootUserID == nil {
		return `scope_users AS (
    SELECT ua.user_id, 0 AS depth
    FROM user_affiliates ua
)`, nil, placeholderIndex
	}
	descendantFilter := ""
	if onlyDescendants {
		descendantFilter = "WHERE scope.depth > 0"
	}
	return fmt.Sprintf(`scope_users AS (
    WITH RECURSIVE scope AS (
        SELECT ua.user_id, ua.inviter_id, 0 AS depth
        FROM user_affiliates ua
        WHERE ua.user_id = $%d
        UNION ALL
        SELECT child.user_id, child.inviter_id, scope.depth + 1
        FROM user_affiliates child
        JOIN scope ON child.inviter_id = scope.user_id
    )
    SELECT scope.user_id, scope.depth
    FROM scope
    %s
)`, placeholderIndex, descendantFilter), []any{*rootUserID}, placeholderIndex + 1
}

func ensureUserExists(ctx context.Context, client affiliateDistributionQueryExecer, userID int64) error {
	var exists bool
	err := scanSingleRow(ctx, client, `
SELECT TRUE
FROM users
WHERE id = $1
LIMIT 1`, []any{userID}, &exists)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrUserNotFound
	}
	return err
}

func (r *affiliateDistributionRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}

func truncateToShanghaiDate(value time.Time) time.Time {
	loc := shanghaiLocation()
	if value.IsZero() {
		value = time.Now().In(loc)
	}
	local := value.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func truncateToUTCDate(value time.Time) time.Time {
	return truncateToShanghaiDate(value)
}

func shanghaiLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return loc
}

func normalizePage(page, pageSize int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	return page, pageSize, (page - 1) * pageSize
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
