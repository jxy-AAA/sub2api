package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type affiliateDistributionRepository struct {
	client *dbent.Client
}

const affiliateRateMultiplierFenPerUSD = 10.0

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
	rebateRate := childRate - parentRate
	if rebateRate < 0 {
		rebateRate = 0
	}
	return usageAmountUSD * rebateRate / affiliateRateMultiplierFenPerUSD
}

func (r *affiliateDistributionRepository) GetDistributionOverview(ctx context.Context, userID int64) (*service.AgentDistributionOverview, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	summary, err := ensureUserAffiliateWithClient(ctx, clientFromContext(ctx, r.client), userID)
	if err != nil {
		return nil, err
	}
	today := truncateToUTCDate(time.Now())
	businessUSD, rebateRMB, err := r.getDailyTotals(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	balanceRMB, err := r.getCurrentRebateBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	inviteRates, err := r.ListInviteModelRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	isAdmin, err := r.isAdminUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	agentRates := make([]service.AgentModelRate, 0, len(inviteRates))
	for _, rate := range inviteRates {
		agentRates = append(agentRates, service.AgentModelRate{Model: rate.Model, Multiplier: rate.Multiplier})
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
		InviteModelRates:        agentRates,
		CanEditSubordinates:     isAdmin || summary.AffCount > 0,
		CanAdjustOwnRebate:      isAdmin,
		MonthlyResetDayOfUTC:    1,
	}, nil
}

func (r *affiliateDistributionRepository) ListInviteModelRates(ctx context.Context, userID int64) ([]service.AgentModelRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT model_key, rate_multiplier::double precision
FROM affiliate_distribution_invite_model_rates
WHERE inviter_user_id = $1
ORDER BY model_key ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list invite model rates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.AgentModelRate, 0)
	for rows.Next() {
		var item service.AgentModelRate
		if err := rows.Scan(&item.Model, &item.Multiplier); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *affiliateDistributionRepository) SaveInviteModelRates(ctx context.Context, userID int64, rates []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	normalized := normalizeAgentModelRateInputs(rates)
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		for _, rate := range normalized {
			if _, err := txClient.ExecContext(txCtx, `
INSERT INTO affiliate_distribution_invite_model_rates (inviter_user_id, model_key, rate_multiplier, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (inviter_user_id, model_key)
DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, updated_at = NOW()`,
				userID, rate.Model, rate.Multiplier,
			); err != nil {
				return fmt.Errorf("save invite model rate: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListInviteModelRates(ctx, userID)
}

func (r *affiliateDistributionRepository) ListDirectSubordinates(ctx context.Context, userID int64) ([]service.AgentDirectMember, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT ua.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       (u.role = 'admin') AS is_admin,
       ua.created_at,
       COALESCE(dm.revenue_amount_usd, 0)::double precision,
       COALESCE(dm.rebate_amount, 0)::double precision,
       COALESCE(rb.current_amount, 0)::double precision
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
LEFT JOIN affiliate_distribution_daily_metrics dm
       ON dm.user_id = ua.user_id AND dm.metric_date = $2
LEFT JOIN affiliate_distribution_rebate_balances rb
       ON rb.user_id = ua.user_id
WHERE ua.inviter_id = $1
ORDER BY ua.created_at ASC`, userID, truncateToUTCDate(time.Now()))
	if err != nil {
		return nil, fmt.Errorf("list direct subordinates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentDirectMember, 0)
	for rows.Next() {
		var item service.AgentDirectMember
		var createdAt sql.NullTime
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.IsAgent, &createdAt, &item.TodayBusinessUSD, &item.TodayRebateRMB, &item.CurrentRebateBalanceRMB); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			ts := createdAt.Time
			item.CreatedAt = &ts
		}
		rates, err := r.GetUserDistributionPricing(ctx, item.UserID)
		if err != nil {
			return nil, err
		}
		item.CurrentModelRates = rates
		item.ParentCanEditRates = true
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *affiliateDistributionRepository) UpdateDirectSubordinateModelRates(ctx context.Context, userID, subordinateUserID int64, rates []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	if userID <= 0 || subordinateUserID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDirectSubordinate(ctx, userID, subordinateUserID); err != nil {
		return nil, err
	}
	return r.AdminUpdateUserDistributionPricing(ctx, userID, subordinateUserID, rates)
}

func (r *affiliateDistributionRepository) ListUserDistributionHistory(ctx context.Context, userID int64, filter service.AgentHistoryFilter) ([]service.AgentHistoryItem, int64, error) {
	if userID <= 0 {
		return nil, 0, service.ErrUserNotFound
	}
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	startDay := filter.StartAt
	endDay := filter.EndAt
	if startDay == nil {
		now := truncateToUTCDate(time.Now())
		startDay = &now
	}
	if endDay == nil {
		now := truncateToUTCDate(time.Now())
		endDay = &now
	}
	countSQL := `
SELECT COUNT(*)
FROM (
    SELECT metric_date
    FROM affiliate_distribution_daily_metrics
    WHERE user_id = $1
      AND metric_date BETWEEN $2 AND $3
    GROUP BY metric_date
) t`
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), countSQL, userID, startDay, endDay)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT metric_date::text,
       COALESCE(revenue_amount_usd, 0)::double precision,
       COALESCE(rebate_amount, 0)::double precision,
       0::integer,
       0::integer,
       COALESCE(last_usage_at, NOW())
FROM affiliate_distribution_daily_metrics
WHERE user_id = $1
  AND metric_date BETWEEN $2 AND $3
ORDER BY metric_date DESC
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
	statDate := truncateToUTCDate(time.Now())
	if filter.StatDate != nil {
		statDate = truncateToUTCDate(*filter.StatDate)
	}
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	search := "%" + strings.TrimSpace(filter.Search) + "%"
	scopeCTE, scopeArgs, nextArgIndex := buildAffiliateScopeCTE(filter.RootUserID, filter.OnlyDescendants, 1)
	statDateArgIndex := nextArgIndex
	searchArgIndex := nextArgIndex + 1
	limitArgIndex := nextArgIndex + 2
	offsetArgIndex := nextArgIndex + 3
	countSQL := fmt.Sprintf(`
WITH %s,
business AS (
    SELECT s.beneficiary_user_id AS user_id,
           s.settlement_day::text AS stat_date,
           COALESCE(SUM(s.revenue_amount_usd), 0)::double precision AS business_usd,
           COALESCE(SUM((s.usage_amount_usd * s.child_rate_multiplier) / %.1f), 0)::double precision AS business_rmb,
           COALESCE(MAX(s.updated_at), MAX(s.created_at), NOW()) AS last_calculated_at
    FROM affiliate_distribution_usage_settlements s
    JOIN scope_users scope ON scope.user_id = s.beneficiary_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id, s.settlement_day
)
SELECT COUNT(*)
FROM business b
JOIN users u ON u.id = b.user_id
WHERE u.email ILIKE $%d OR u.username ILIKE $%d`, scopeCTE, affiliateRateMultiplierFenPerUSD, statDateArgIndex, searchArgIndex, searchArgIndex)
	countArgs := append(append([]any{}, scopeArgs...), statDate, search)
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), countSQL, countArgs...)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, fmt.Sprintf(`
WITH %s,
direct_counts AS (
    SELECT ua.inviter_id AS user_id,
           COUNT(*) FILTER (WHERE child_u.role = 'admin' OR COALESCE(child_ua.aff_count, 0) > 0)::integer AS direct_agents,
           COUNT(*) FILTER (WHERE child_u.role <> 'admin' AND COALESCE(child_ua.aff_count, 0) = 0)::integer AS direct_users
    FROM user_affiliates ua
    JOIN users child_u ON child_u.id = ua.user_id
    JOIN user_affiliates child_ua ON child_ua.user_id = ua.user_id
    WHERE ua.inviter_id IS NOT NULL
    GROUP BY ua.inviter_id
),
business AS (
    SELECT s.beneficiary_user_id AS user_id,
           s.settlement_day::text AS stat_date,
           COALESCE(SUM(s.revenue_amount_usd), 0)::double precision AS business_usd,
           COALESCE(SUM((s.usage_amount_usd * s.child_rate_multiplier) / %.1f), 0)::double precision AS business_rmb,
           COALESCE(MAX(s.updated_at), MAX(s.created_at), NOW()) AS last_calculated_at
    FROM affiliate_distribution_usage_settlements s
    JOIN scope_users scope ON scope.user_id = s.beneficiary_user_id
    WHERE s.settlement_day = $%d
    GROUP BY s.beneficiary_user_id, s.settlement_day
),
ranked AS (
    SELECT b.user_id,
           COALESCE(u.email, '') AS email,
           COALESCE(u.username, '') AS username,
           b.stat_date,
           b.business_usd,
           b.business_rmb,
           COALESCE(dc.direct_users, 0)::integer AS direct_users,
           COALESCE(dc.direct_agents, 0)::integer AS direct_agents,
           b.last_calculated_at,
           ROW_NUMBER() OVER (ORDER BY b.business_rmb DESC, b.user_id ASC) AS rank
    FROM business b
    JOIN users u ON u.id = b.user_id
    LEFT JOIN direct_counts dc ON dc.user_id = b.user_id
    WHERE u.email ILIKE $%d OR u.username ILIKE $%d
)
SELECT rank, user_id, email, username, stat_date, business_usd, business_rmb, direct_users, direct_agents, last_calculated_at
FROM ranked
ORDER BY rank
LIMIT $%d OFFSET $%d`, scopeCTE, affiliateRateMultiplierFenPerUSD, statDateArgIndex, searchArgIndex, searchArgIndex, limitArgIndex, offsetArgIndex),
		append(append([]any{}, scopeArgs...), statDate, search, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentDailyBusinessRankingItem, 0)
	for rows.Next() {
		var item service.AgentDailyBusinessRankingItem
		if err := rows.Scan(&item.Rank, &item.UserID, &item.Email, &item.Username, &item.StatDate, &item.BusinessUSD, &item.BusinessRMB, &item.DirectUsers, &item.DirectAgents, &item.LastCalculatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *affiliateDistributionRepository) ListRebateBalanceRanking(ctx context.Context, filter service.AgentRankingFilter) ([]service.AgentRebateBalanceRankingItem, int64, error) {
	_, pageSize, offset := normalizePage(filter.Page, filter.PageSize)
	search := "%" + strings.TrimSpace(filter.Search) + "%"
	today := truncateToUTCDate(time.Now())
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	scopeCTE, scopeArgs, nextArgIndex := buildAffiliateScopeCTE(filter.RootUserID, filter.OnlyDescendants, 1)
	searchArgIndex := nextArgIndex
	todayArgIndex := nextArgIndex + 1
	monthStartArgIndex := nextArgIndex + 2
	limitArgIndex := nextArgIndex + 3
	offsetArgIndex := nextArgIndex + 4
	countSQL := fmt.Sprintf(`
WITH %s
SELECT COUNT(*)
FROM affiliate_distribution_rebate_balances rb
JOIN scope_users scope ON scope.user_id = rb.user_id
JOIN users u ON u.id = rb.user_id
WHERE u.email ILIKE $%d OR u.username ILIKE $%d`, scopeCTE, searchArgIndex, searchArgIndex)
	total, err := scanInt64(ctx, clientFromContext(ctx, r.client), countSQL, append(append([]any{}, scopeArgs...), search)...)
	if err != nil {
		return nil, 0, err
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, fmt.Sprintf(`
WITH %s,
latest_note AS (
    SELECT DISTINCT ON (user_id) user_id, reason, created_at
    FROM affiliate_distribution_rebate_adjustments
    ORDER BY user_id, created_at DESC
), today_metrics AS (
    SELECT user_id, COALESCE(SUM(rebate_amount), 0)::double precision AS today_rebate_rmb
    FROM affiliate_distribution_daily_metrics
    WHERE metric_date = $%d
    GROUP BY user_id
), monthly_metrics AS (
    SELECT user_id, COALESCE(SUM(rebate_amount), 0)::double precision AS monthly_rebate_rmb
    FROM affiliate_distribution_daily_metrics
    WHERE metric_date >= $%d AND metric_date < ($%d::date + INTERVAL '1 month')
    GROUP BY user_id
), direct_counts AS (
    SELECT ua.inviter_id AS user_id,
           COUNT(*) FILTER (WHERE child_u.role = 'admin' OR COALESCE(child_ua.aff_count, 0) > 0)::integer AS direct_agents,
           COUNT(*) FILTER (WHERE child_u.role <> 'admin' AND COALESCE(child_ua.aff_count, 0) = 0)::integer AS direct_users
    FROM user_affiliates ua
    JOIN users child_u ON child_u.id = ua.user_id
    JOIN user_affiliates child_ua ON child_ua.user_id = ua.user_id
    WHERE ua.inviter_id IS NOT NULL
    GROUP BY ua.inviter_id
),
ranked AS (
    SELECT rb.user_id,
           COALESCE(u.email, '') AS email,
           COALESCE(u.username, '') AS username,
           rb.current_amount::double precision AS current_rebate_balance_rmb,
           COALESCE(tm.today_rebate_rmb, 0)::double precision AS today_rebate_rmb,
           COALESCE(mm.monthly_rebate_rmb, 0)::double precision AS monthly_rebate_rmb,
           COALESCE(dc.direct_users, 0)::integer AS direct_users,
           COALESCE(dc.direct_agents, 0)::integer AS direct_agents,
           COALESCE(ln.created_at, rb.updated_at) AS last_adjusted_at,
           COALESCE(ln.reason, '') AS last_adjustment_note,
           ROW_NUMBER() OVER (ORDER BY rb.current_amount DESC, rb.updated_at DESC, rb.user_id ASC) AS rank
    FROM affiliate_distribution_rebate_balances rb
    JOIN scope_users scope ON scope.user_id = rb.user_id
    JOIN users u ON u.id = rb.user_id
    LEFT JOIN latest_note ln ON ln.user_id = rb.user_id
    LEFT JOIN today_metrics tm ON tm.user_id = rb.user_id
    LEFT JOIN monthly_metrics mm ON mm.user_id = rb.user_id
    LEFT JOIN direct_counts dc ON dc.user_id = rb.user_id
    WHERE u.email ILIKE $%d OR u.username ILIKE $%d
)
SELECT rank, user_id, email, username, current_rebate_balance_rmb, today_rebate_rmb, monthly_rebate_rmb, direct_users, direct_agents, last_adjusted_at, last_adjustment_note
FROM ranked
ORDER BY rank
LIMIT $%d OFFSET $%d`, scopeCTE, todayArgIndex, monthStartArgIndex, monthStartArgIndex, searchArgIndex, searchArgIndex, limitArgIndex, offsetArgIndex),
		append(append([]any{}, scopeArgs...), search, today, monthStart, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentRebateBalanceRankingItem, 0)
	for rows.Next() {
		var item service.AgentRebateBalanceRankingItem
		if err := rows.Scan(&item.Rank, &item.UserID, &item.Email, &item.Username, &item.CurrentRebateBalanceRMB, &item.TodayRebateRMB, &item.MonthlyRebateRMB, &item.DirectUsers, &item.DirectAgents, &item.LastAdjustedAt, &item.LastAdjustmentNote); err != nil {
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
)
SELECT t.user_id, t.inviter_id, COALESCE(u.email, ''), COALESCE(u.username, ''), ua.aff_code, t.depth, (u.role = 'admin')::boolean, COALESCE(rb.current_amount, 0)::double precision
FROM tree t
JOIN users u ON u.id = t.user_id
JOIN user_affiliates ua ON ua.user_id = t.user_id
LEFT JOIN affiliate_distribution_rebate_balances rb ON rb.user_id = t.user_id
WHERE (u.email ILIKE $%d OR u.username ILIKE $%d OR ua.aff_code ILIKE $%d)
  %s
ORDER BY t.depth ASC, t.user_id ASC`, rootPredicate, searchArgIndex, searchArgIndex, searchArgIndex, descendantClause)
	args = append(args, search)
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentTreeNode, 0)
	for rows.Next() {
		var node service.AgentTreeNode
		var inviterID sql.NullInt64
		if err := rows.Scan(&node.UserID, &inviterID, &node.Email, &node.Username, &node.InviteCode, &node.Depth, &node.IsAdmin, &node.CurrentRebateBalanceRMB); err != nil {
			return nil, err
		}
		if inviterID.Valid {
			node.InviterID = &inviterID.Int64
		}
		rates, err := r.GetUserDistributionPricing(ctx, node.UserID)
		if err != nil {
			return nil, err
		}
		node.CurrentModelRates = rates
		items = append(items, node)
	}
	return items, rows.Err()
}

func (r *affiliateDistributionRepository) GetUserDistributionPricing(ctx context.Context, userID int64) ([]service.AgentModelRate, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT model_key, rate_multiplier::double precision
FROM affiliate_distribution_user_model_rates
WHERE user_id = $1
ORDER BY model_key ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AgentModelRate, 0)
	for rows.Next() {
		var item service.AgentModelRate
		if err := rows.Scan(&item.Model, &item.Multiplier); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *affiliateDistributionRepository) AdminUpdateUserDistributionPricing(ctx context.Context, operatorUserID, userID int64, rates []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDirectOrAdmin(ctx, operatorUserID, userID); err != nil {
		return nil, err
	}
	_, err := r.SaveInviteModelRates(ctx, userID, rates)
	if err != nil {
		return nil, err
	}
	return r.GetUserDistributionPricing(ctx, userID)
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

func (r *affiliateDistributionRepository) GetUserDistributionPricingScoped(ctx context.Context, operatorUserID, userID int64) ([]service.AgentModelRate, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDescendant(ctx, operatorUserID, userID); err != nil {
		return nil, err
	}
	return r.GetUserDistributionPricing(ctx, userID)
}

func (r *affiliateDistributionRepository) UpdateUserDistributionPricingScoped(ctx context.Context, operatorUserID, userID int64, rates []service.AgentModelRateInput) ([]service.AgentModelRate, error) {
	if operatorUserID <= 0 || userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if err := r.ensureDescendant(ctx, operatorUserID, userID); err != nil {
		return nil, err
	}
	_, err := r.SaveInviteModelRates(ctx, userID, rates)
	if err != nil {
		return nil, err
	}
	return r.GetUserDistributionPricing(ctx, userID)
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
	res, err := clientFromContext(ctx, r.client).ExecContext(ctx, `
INSERT INTO affiliate_distribution_usage_jobs (usage_log_id, status, claimed_at, updated_at)
VALUES ($1, 'processing', NOW(), NOW())
ON CONFLICT (usage_log_id) DO NOTHING`, usageLogID)
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
SET status = 'failed', updated_at = NOW()
WHERE usage_log_id = $1`, usageLogID)
	return err
}

func (r *affiliateDistributionRepository) SettleUsageDistribution(ctx context.Context, cmd service.AffiliateDistributionUsageSettlementCommand) error {
	model := strings.TrimSpace(cmd.RequestedModel)
	if model == "" {
		model = strings.TrimSpace(cmd.Model)
	}
	if model == "" {
		return infraerrors.BadRequest("AFFILIATE_DISTRIBUTION_MODEL_REQUIRED", "model is required")
	}
	usageUSD := cmd.TotalCost
	if usageUSD <= 0 {
		usageUSD = cmd.ActualCost
	}
	if usageUSD <= 0 {
		return nil
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		resolvedRate, err := r.resolveModelRateTx(txCtx, txClient, cmd.UserID, model, 0, 1.0)
		if err != nil {
			return err
		}
		_ = resolvedRate
		result, err := r.recordUsageSettlementWithTx(txCtx, txClient, service.RecordAffiliateUsageSettlementInput{
			UsageLogID:     cmd.UsageLogID,
			ConsumerUserID: cmd.UserID,
			ModelKey:       model,
			UsageAmountUSD: usageUSD,
			SettlementAt:   cmd.UsageCreatedAt,
			DefaultMarkup:  0,
			RootRate:       1.0,
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
		SettlementDay:  truncateToUTCDate(input.SettlementAt),
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
		parentRate, err := r.resolveModelRateTx(ctx, txClient, parent.UserID, input.ModelKey, input.DefaultMarkup, input.RootRate)
		if err != nil {
			return nil, err
		}
		childRate, err := r.resolveModelRateTx(ctx, txClient, child.UserID, input.ModelKey, input.DefaultMarkup, input.RootRate)
		if err != nil {
			return nil, err
		}
		rebateAmountRMB := affiliateDistributionRebateAmountRMB(input.UsageAmountUSD, parentRate, childRate)
		applied, err := r.insertSettlementRow(ctx, txClient, input, parent.UserID, child.UserID, int64(depth), parentRate, childRate, rebateAmountRMB)
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
				RevenueAmountUSD:     input.UsageAmountUSD,
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

func (r *affiliateDistributionRepository) resolveModelRateTx(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, model string, defaultMarkup, rootRate float64) (float64, error) {
	var rate float64
	err := scanSingleRow(ctx, client, `
SELECT rate_multiplier::double precision
FROM affiliate_distribution_user_model_rates
WHERE user_id = $1 AND model_key = $2
LIMIT 1`, []any{userID, model}, &rate)
	if err == nil {
		return rate, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	summary, err := queryAffiliateByUserID(ctx, client, userID)
	if err != nil {
		return 0, err
	}
	if summary.InviterID == nil {
		if rootRate <= 0 {
			rootRate = 1
		}
		return rootRate, r.upsertUserModelRateTx(ctx, client, userID, nil, model, rootRate, "", "root_default")
	}
	parentRate, err := r.resolveModelRateTx(ctx, client, *summary.InviterID, model, defaultMarkup, rootRate)
	if err != nil {
		return 0, err
	}
	err = scanSingleRow(ctx, client, `
SELECT rate_multiplier::double precision
FROM affiliate_distribution_invite_model_rates
WHERE inviter_user_id = $1 AND model_key = $2
LIMIT 1`, []any{*summary.InviterID, model}, &rate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	sourceType := "invite_code"
	if errors.Is(err, sql.ErrNoRows) {
		markup, found, markupErr := r.lookupDefaultMarkup(ctx, client, model)
		if markupErr != nil {
			return 0, markupErr
		}
		if found {
			defaultMarkup = markup
		}
		rate = parentRate + defaultMarkup
		sourceType = "default_markup"
	}
	parentSummary, _ := queryAffiliateByUserID(ctx, client, *summary.InviterID)
	sourceCode := ""
	if parentSummary != nil {
		sourceCode = parentSummary.AffCode
	}
	return rate, r.upsertUserModelRateTx(ctx, client, userID, summary.InviterID, model, rate, sourceCode, sourceType)
}

func (r *affiliateDistributionRepository) insertSettlementRow(ctx context.Context, client affiliateDistributionQueryExecer, input service.RecordAffiliateUsageSettlementInput, parentUserID, childUserID, depth int64, parentRate, childRate, rebateAmount float64) (bool, error) {
	res, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_usage_settlements (
    usage_log_id, settlement_key, beneficiary_user_id, direct_child_user_id, consumer_user_id, model_key,
    usage_amount_usd, revenue_amount_usd, parent_rate_multiplier, child_rate_multiplier, rebate_amount, settlement_day, depth, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, $9, $10, $11, $12, NOW(), NOW())
ON CONFLICT (usage_log_id, beneficiary_user_id) DO NOTHING`,
		input.UsageLogID,
		fmt.Sprintf("%d:%d:%s", input.UsageLogID, parentUserID, input.ModelKey),
		parentUserID,
		childUserID,
		input.ConsumerUserID,
		input.ModelKey,
		input.UsageAmountUSD,
		parentRate,
		childRate,
		rebateAmount,
		truncateToUTCDate(input.SettlementAt),
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
		parentUserID, truncateToUTCDate(input.SettlementAt), input.UsageAmountUSD, rebateAmount, input.SettlementAt,
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

func (r *affiliateDistributionRepository) getDailyTotals(ctx context.Context, userID int64, day time.Time) (float64, float64, error) {
	var revenue float64
	var rebate float64
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
SELECT COALESCE(revenue_amount_usd, 0)::double precision, COALESCE(rebate_amount, 0)::double precision
FROM affiliate_distribution_daily_metrics
WHERE user_id = $1 AND metric_date = $2
LIMIT 1`, []any{userID, day}, &revenue, &rebate)
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

func normalizeAgentModelRateInputs(rates []service.AgentModelRateInput) []service.AgentModelRateInput {
	if len(rates) == 0 {
		return []service.AgentModelRateInput{}
	}
	normalized := make([]service.AgentModelRateInput, 0, len(rates))
	for _, rate := range rates {
		model := strings.TrimSpace(rate.Model)
		if model == "" {
			continue
		}
		normalized = append(normalized, service.AgentModelRateInput{Model: model, Multiplier: rate.Multiplier})
	}
	return normalized
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func (r *affiliateDistributionRepository) lookupDefaultMarkup(ctx context.Context, client affiliateDistributionQueryExecer, model string) (float64, bool, error) {
	var markup float64
	err := scanSingleRow(ctx, client, `
SELECT default_markup::double precision
FROM affiliate_distribution_default_model_rates
WHERE model_key = $1 OR model_key = '*'
ORDER BY CASE WHEN model_key = $1 THEN 0 ELSE 1 END
LIMIT 1`, []any{model}, &markup)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return markup, err == nil, err
}

func (r *affiliateDistributionRepository) upsertUserModelRateTx(ctx context.Context, client affiliateDistributionQueryExecer, userID int64, upstreamUserID *int64, model string, multiplier float64, sourceAffCode, sourceType string) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_distribution_user_model_rates (
    user_id, upstream_user_id, model_key, rate_multiplier, source_aff_code, source_type, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (user_id, model_key)
DO UPDATE SET
    upstream_user_id = EXCLUDED.upstream_user_id,
    rate_multiplier = EXCLUDED.rate_multiplier,
    source_aff_code = EXCLUDED.source_aff_code,
    source_type = EXCLUDED.source_type,
    updated_at = NOW()`,
		userID, nullableInt64Arg(upstreamUserID), model, multiplier, nullableString(sourceAffCode), sourceType,
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

func truncateToUTCDate(value time.Time) time.Time {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
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
