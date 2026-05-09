import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'
import type { AffiliateDistributionTreeNode, AffiliateModelRate } from '@/api/admin/affiliates'

export interface ManagedAffiliatePermissions {
  can_view_downline_daily_revenue: boolean
  can_view_downline_rebate_balances: boolean
  can_manage_downline_pricing: boolean
}

export interface ManagedAffiliateLeaderboardQuery {
  page?: number
  page_size?: number
  search?: string
  date?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  timezone?: string
}

export interface ManagedAffiliateLeaderboardSummary {
  total_revenue_rmb?: number
  total_rebate_balance_rmb?: number
  total_agents?: number
  total_direct_downlines?: number
  total_direct_usage_rmb?: number
  total_direct_users?: number
  total_direct_agents?: number
}

export interface ManagedAffiliateLeaderboardResponse<T> extends PaginatedResponse<T> {
  summary?: ManagedAffiliateLeaderboardSummary
}

export interface ManagedDailyRevenueRankingItem {
  rank?: number
  agent_user_id?: number
  user_id?: number
  id?: number
  agent_email?: string
  user_email?: string
  email?: string
  agent_username?: string
  username?: string
  revenue_date?: string
  date?: string
  total_revenue_rmb?: number
  revenue_rmb?: number
  daily_revenue_rmb?: number
  business_rmb?: number
  direct_downline_count?: number
  direct_user_count?: number
  direct_agent_count?: number
  direct_total_usage_rmb?: number
  direct_user_usage_rmb?: number
  direct_agent_usage_rmb?: number
  updated_at?: string
}

export interface ManagedRebateBalanceRankingItem {
  rank?: number
  agent_user_id?: number
  user_id?: number
  id?: number
  agent_email?: string
  user_email?: string
  email?: string
  agent_username?: string
  username?: string
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

export function emptyManagedAffiliatePermissions(): ManagedAffiliatePermissions {
  return {
    can_view_downline_daily_revenue: false,
    can_view_downline_rebate_balances: false,
    can_manage_downline_pricing: false,
  }
}

export function hasManagedAffiliateAccess(permissions?: ManagedAffiliatePermissions | null): boolean {
  return Boolean(
    permissions?.can_view_downline_daily_revenue
      || permissions?.can_view_downline_rebate_balances
      || permissions?.can_manage_downline_pricing,
  )
}

function normalizeQuery(params: ManagedAffiliateLeaderboardQuery = {}) {
  return {
    page: params.page ?? 1,
    page_size: params.page_size ?? 20,
    search: params.search ?? '',
    date: params.date || undefined,
    sort_by: params.sort_by || undefined,
    sort_order: params.sort_order || undefined,
    timezone: params.timezone || undefined,
  }
}

async function getWithFallback<T>(paths: string[], params?: Record<string, unknown>) {
  let lastError: unknown
  for (const path of paths) {
    try {
      const { data } = await apiClient.get<T>(path, { params })
      return data
    } catch (error: any) {
      lastError = error
      const status = error?.response?.status ?? error?.status
      if (status !== 404) {
        throw error
      }
    }
  }
  throw lastError
}

async function putWithFallback<T>(paths: string[], payload: Record<string, unknown>) {
  let lastError: unknown
  for (const path of paths) {
    try {
      const { data } = await apiClient.put<T>(path, payload)
      return data
    } catch (error: any) {
      lastError = error
      const status = error?.response?.status ?? error?.status
      if (status !== 404) {
        throw error
      }
    }
  }
  throw lastError
}

export async function getManagedAffiliatePermissions(): Promise<ManagedAffiliatePermissions> {
  const data = await getWithFallback<Partial<ManagedAffiliatePermissions>>(
    [
      '/user/aff/managed/permissions',
      '/user/aff/manage/permissions',
    ],
  )

  return {
    can_view_downline_daily_revenue: Boolean(data.can_view_downline_daily_revenue),
    can_view_downline_rebate_balances: Boolean(data.can_view_downline_rebate_balances),
    can_manage_downline_pricing: Boolean(data.can_manage_downline_pricing),
  }
}

export async function listManagedDailyRevenueRankings(
  params: ManagedAffiliateLeaderboardQuery = {},
): Promise<ManagedAffiliateLeaderboardResponse<ManagedDailyRevenueRankingItem>> {
  return getWithFallback<ManagedAffiliateLeaderboardResponse<ManagedDailyRevenueRankingItem>>(
    [
      '/user/aff/managed/daily-revenue',
      '/user/aff/manage/daily-revenue',
    ],
    normalizeQuery(params),
  )
}

export async function listManagedRebateBalanceRankings(
  params: ManagedAffiliateLeaderboardQuery = {},
): Promise<ManagedAffiliateLeaderboardResponse<ManagedRebateBalanceRankingItem>> {
  return getWithFallback<ManagedAffiliateLeaderboardResponse<ManagedRebateBalanceRankingItem>>(
    [
      '/user/aff/managed/rebate-balances',
      '/user/aff/manage/rebate-balances',
    ],
    normalizeQuery(params),
  )
}

export async function getManagedDistributionTree(
  params: { search?: string } = {},
): Promise<AffiliateDistributionTreeNode[]> {
  return getWithFallback<AffiliateDistributionTreeNode[]>(
    [
      '/user/aff/managed/tree',
      '/user/aff/manage/tree',
    ],
    {
      search: params.search || undefined,
    },
  )
}

export async function getManagedUserDistributionPricing(
  userId: number,
): Promise<{ user_id: number; model_rates: AffiliateModelRate[] }> {
  return getWithFallback<{ user_id: number; model_rates: AffiliateModelRate[] }>(
    [
      `/user/aff/managed/users/${userId}/pricing`,
      `/user/aff/manage/users/${userId}/pricing`,
    ],
  )
}

export async function updateManagedUserDistributionPricing(
  userId: number,
  payload: { model_rates: Array<{ model_name?: string; model?: string; multiplier: number }> },
): Promise<{ user_id: number; model_rates: AffiliateModelRate[] }> {
  const modelRates = payload.model_rates.map((rate) => ({
    model: rate.model || rate.model_name || '',
    multiplier: rate.multiplier,
  }))

  return putWithFallback<{ user_id: number; model_rates: AffiliateModelRate[] }>(
    [
      `/user/aff/managed/users/${userId}/pricing`,
      `/user/aff/manage/users/${userId}/pricing`,
    ],
    { model_rates: modelRates },
  )
}

const userAffiliateManagedAPI = {
  emptyManagedAffiliatePermissions,
  hasManagedAffiliateAccess,
  getManagedAffiliatePermissions,
  listManagedDailyRevenueRankings,
  listManagedRebateBalanceRankings,
  getManagedDistributionTree,
  getManagedUserDistributionPricing,
  updateManagedUserDistributionPricing,
}

export default userAffiliateManagedAPI
