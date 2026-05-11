import { apiClient } from '@/api/client'
import type { PaginatedResponse } from '@/types'

export type AdminAffiliateRecordType = 'invites' | 'rebates' | 'transfers'

export interface AffiliateLeaderboardQuery {
  page?: number
  page_size?: number
  search?: string
  date?: string
  month?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  timezone?: string
}

export interface AffiliateAgentIdentity {
  agent_user_id?: number
  user_id?: number
  id?: number
  agent_email?: string
  user_email?: string
  email?: string
  agent_username?: string
  username?: string
}

export interface DailyRevenueRankingItem extends AffiliateAgentIdentity {
  rank?: number
  revenue_date?: string
  date?: string
  business_rmb?: number
  total_business_rmb?: number
  revenue_rmb?: number
  daily_revenue_rmb?: number
  total_revenue_usd?: number
  revenue_usd?: number
  daily_revenue_usd?: number
  direct_downline_count?: number
  direct_user_count?: number
  direct_agent_count?: number
  direct_total_usage_rmb?: number
  direct_user_usage_rmb?: number
  direct_agent_usage_rmb?: number
  direct_total_usage_usd?: number
  direct_user_usage_usd?: number
  direct_agent_usage_usd?: number
  updated_at?: string
}

export interface RebateBalanceRankingItem extends AffiliateAgentIdentity {
  rank?: number
  current_rebate_balance_rmb?: number
  rebate_balance_rmb?: number
  total_rebate_balance_rmb?: number
  today_rebate_rmb?: number
  monthly_rebate_rmb?: number
  direct_users?: number
  direct_agents?: number
  direct_user_count?: number
  direct_agent_count?: number
  direct_downline_count?: number
  updated_at?: string
}

export interface MonthlyArchiveItem extends AffiliateAgentIdentity {
  archive_month?: string
  month?: string
  archived_rebate_rmb?: number
  archived_balance_rmb?: number
  opening_balance_rmb?: number
  closing_balance_rmb?: number
  reset_status?: string
  cleared_to_zero?: boolean
  cleared_at?: string
  created_at?: string
  operator_email?: string
  remark?: string
  note?: string
}

export interface AffiliateLeaderboardSummary {
  total_business_rmb?: number
  total_revenue_rmb?: number
  total_revenue_usd?: number
  total_rebate_balance_rmb?: number
  total_archived_rebate_rmb?: number
  total_agents?: number
  total_direct_downlines?: number
  total_direct_usage_rmb?: number
  total_direct_usage_usd?: number
  total_direct_users?: number
  total_direct_agents?: number
  cleared_count?: number
  pending_count?: number
}

export interface AffiliateAgentPermissions {
  user_id: number
  can_view_downline_daily_revenue: boolean
  can_view_downline_rebate_balances: boolean
  can_manage_downline_pricing: boolean
  granted_by_user_id?: number | null
  granted_by_email?: string | null
  updated_at?: string | null
  created_at?: string | null
}

export interface UpdateAffiliateAgentPermissionsPayload {
  can_view_downline_daily_revenue: boolean
  can_view_downline_rebate_balances: boolean
  can_manage_downline_pricing: boolean
}

export interface AffiliateLeaderboardResponse<T> extends PaginatedResponse<T> {
  summary?: AffiliateLeaderboardSummary
}

export interface RebateBalanceAdjustmentPayload {
  agent_user_id: number
  rebate_balance_rmb: number
  remark: string
}

async function getWithFallback<T>(paths: string[], params: Record<string, unknown>) {
  let lastError: unknown
  for (const path of paths) {
    try {
      const { data } = await apiClient.get<T>(path, { params })
      return data
    } catch (error: any) {
      lastError = error
      if (error?.status !== 404) {
        throw error
      }
    }
  }
  throw lastError
}

async function postWithFallback<T>(paths: string[], payload: Record<string, any>) {
  let lastError: unknown
  for (const path of paths) {
    try {
      const { data } = await apiClient.post<T>(path, payload)
      return data
    } catch (error: any) {
      lastError = error
      if (error?.status !== 404) {
        throw error
      }
    }
  }
  throw lastError
}

function normalizeQuery(params: AffiliateLeaderboardQuery = {}) {
  return {
    page: params.page ?? 1,
    page_size: params.page_size ?? 20,
    search: params.search ?? '',
    date: params.date || undefined,
    month: params.month || undefined,
    sort_by: params.sort_by || undefined,
    sort_order: params.sort_order || undefined,
    timezone: params.timezone || undefined,
  }
}

export async function listDailyRevenueRankings(
  params: AffiliateLeaderboardQuery = {},
): Promise<AffiliateLeaderboardResponse<DailyRevenueRankingItem>> {
  return getWithFallback<AffiliateLeaderboardResponse<DailyRevenueRankingItem>>(
    [
      '/admin/affiliates/daily-revenue-rankings',
      '/admin/affiliates/daily-revenues',
      '/admin/affiliates/daily-revenue',
    ],
    normalizeQuery(params),
  )
}

export async function listRebateBalanceRankings(
  params: AffiliateLeaderboardQuery = {},
): Promise<AffiliateLeaderboardResponse<RebateBalanceRankingItem>> {
  return getWithFallback<AffiliateLeaderboardResponse<RebateBalanceRankingItem>>(
    [
      '/admin/affiliates/rebate-balance-rankings',
      '/admin/affiliates/rebate-balances',
      '/admin/affiliates/rebate-balance',
    ],
    normalizeQuery(params),
  )
}

export async function listMonthlyArchives(
  params: AffiliateLeaderboardQuery = {},
): Promise<AffiliateLeaderboardResponse<MonthlyArchiveItem>> {
  return getWithFallback<AffiliateLeaderboardResponse<MonthlyArchiveItem>>(
    [
      '/admin/affiliates/monthly-archives',
      '/admin/affiliates/rebate-archives',
      '/admin/affiliates/monthly-rebate-archives',
    ],
    normalizeQuery(params),
  )
}

export async function adjustRebateBalance(
  payload: RebateBalanceAdjustmentPayload,
): Promise<{ agent_user_id: number; rebate_balance_rmb: number }> {
  const requestPayload = {
    user_id: payload.agent_user_id,
    amount: payload.rebate_balance_rmb,
    note: payload.remark,
  }
  return postWithFallback<{ agent_user_id: number; rebate_balance_rmb: number }>(
    [
      '/admin/affiliates/rebate-balances/adjust',
      '/admin/affiliates/rebate-balance-adjustments',
      '/admin/affiliates/rebates/adjust-balance',
    ],
    requestPayload,
  )
}

function normalizePermissionsResponse(
  data: AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number },
  userId: number,
): AffiliateAgentPermissions {
  const permissions: Partial<AffiliateAgentPermissions> = 'permissions' in data && data.permissions
    ? data.permissions
    : (data as AffiliateAgentPermissions)
  return {
    user_id: permissions.user_id ?? ('user_id' in data && typeof data.user_id === 'number' ? data.user_id : userId),
    can_view_downline_daily_revenue: Boolean(permissions.can_view_downline_daily_revenue),
    can_view_downline_rebate_balances: Boolean(permissions.can_view_downline_rebate_balances),
    can_manage_downline_pricing: Boolean(permissions.can_manage_downline_pricing),
    granted_by_user_id: permissions.granted_by_user_id ?? null,
    granted_by_email: permissions.granted_by_email ?? null,
    updated_at: permissions.updated_at ?? null,
    created_at: permissions.created_at ?? null,
  }
}

export async function getAffiliateAgentPermissions(
  userId: number,
): Promise<AffiliateAgentPermissions> {
  const { data } = await apiClient.get<AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number }>(`/admin/affiliates/users/${userId}/permissions`)
  return normalizePermissionsResponse(data, userId)
}

export async function updateAffiliateAgentPermissions(
  userId: number,
  payload: UpdateAffiliateAgentPermissionsPayload,
): Promise<AffiliateAgentPermissions> {
  const { data } = await apiClient.put<AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number }>(`/admin/affiliates/users/${userId}/permissions`, payload)
  return normalizePermissionsResponse(data, userId)
}
