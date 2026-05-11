import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'
import type { AffiliateDistributionTreeNode } from '@/api/admin/affiliates'
import type {
  AffiliateGroupRate,
  AffiliateGroupRateInput,
  AffiliatePricingResponse,
} from '@/components/affiliate/types'

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

function normalizeGroupRates(data: unknown): AffiliateGroupRate[] {
  const payload = data as {
    group_rates?: AffiliateGroupRate[]
    current_group_rates?: AffiliateGroupRate[]
    invite_group_rates?: AffiliateGroupRate[]
  } | null

  const rawRates = payload?.group_rates
    ?? payload?.current_group_rates
    ?? payload?.invite_group_rates
    ?? []

  return rawRates
    .map((rate) => {
      const groupId = Number(rate?.group_id)
      const rateMultiplier = Number(rate?.rate_multiplier ?? Number.NaN)

      if (!Number.isInteger(groupId) || groupId <= 0 || !Number.isFinite(rateMultiplier) || rateMultiplier <= 0) {
        return null
      }

      return {
        ...rate,
        group_id: groupId,
        rate_multiplier: rateMultiplier,
      } satisfies AffiliateGroupRate
    })
    .filter((rate): rate is AffiliateGroupRate => rate !== null)
}

function normalizePricingResponse(
  data: unknown,
  userId: number,
): AffiliatePricingResponse {
  const payload = data as { user_id?: number } | null

  return {
    user_id: Number.isFinite(Number(payload?.user_id)) ? Number(payload?.user_id) : userId,
    group_rates: normalizeGroupRates(data),
  }
}

function createMissingGroupRatesError(path: string): Error {
  return Object.assign(new Error(`Expected non-empty group_rates from ${path}`), {
    code: 'INVALID_GROUP_RATES_RESPONSE',
  })
}

export async function getManagedAffiliatePermissions(): Promise<ManagedAffiliatePermissions> {
  const { data } = await apiClient.get<Partial<ManagedAffiliatePermissions>>('/user/aff/managed/permissions')

  return {
    can_view_downline_daily_revenue: Boolean(data.can_view_downline_daily_revenue),
    can_view_downline_rebate_balances: Boolean(data.can_view_downline_rebate_balances),
    can_manage_downline_pricing: Boolean(data.can_manage_downline_pricing),
  }
}

export async function listManagedDailyRevenueRankings(
  params: ManagedAffiliateLeaderboardQuery = {},
): Promise<ManagedAffiliateLeaderboardResponse<ManagedDailyRevenueRankingItem>> {
  const { data } = await apiClient.get<ManagedAffiliateLeaderboardResponse<ManagedDailyRevenueRankingItem>>('/user/aff/managed/daily-revenue', { params: normalizeQuery(params) })
  return data
}

export async function listManagedRebateBalanceRankings(
  params: ManagedAffiliateLeaderboardQuery = {},
): Promise<ManagedAffiliateLeaderboardResponse<ManagedRebateBalanceRankingItem>> {
  const { data } = await apiClient.get<ManagedAffiliateLeaderboardResponse<ManagedRebateBalanceRankingItem>>('/user/aff/managed/rebate-balances', { params: normalizeQuery(params) })
  return data
}

export async function getManagedDistributionTree(
  params: { search?: string } = {},
): Promise<AffiliateDistributionTreeNode[]> {
  const { data } = await apiClient.get<AffiliateDistributionTreeNode[]>('/user/aff/managed/tree', {
    params: { search: params.search || undefined },
  })
  return data
}

export async function getManagedUserDistributionPricing(
  userId: number,
): Promise<AffiliatePricingResponse> {
  const { data } = await apiClient.get<unknown>(`/user/aff/managed/users/${userId}/pricing`)

  return normalizePricingResponse(data, userId)
}

export async function updateManagedUserDistributionPricing(
  userId: number,
  payload: { group_rates: AffiliateGroupRateInput[] },
): Promise<AffiliatePricingResponse> {
  const groupRates = payload.group_rates.map((rate) => ({
    group_id: rate.group_id,
    rate_multiplier: rate.rate_multiplier,
  }))

  const { data } = await apiClient.put<unknown>(`/user/aff/managed/users/${userId}/pricing`, { group_rates: groupRates })

  const normalized = normalizePricingResponse(data, userId)
  if (normalized.group_rates.length > 0) {
    return normalized
  }

  const verified = await getManagedUserDistributionPricing(userId)
  if (verified.group_rates.length > 0) {
    return verified
  }

  throw createMissingGroupRatesError(`/user/aff/managed/users/${userId}/pricing`)
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
