/**
 * Admin affiliate distribution API endpoints.
 * Covers agent hierarchy management, pricing, permissions and rankings.
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type {
  AffiliateDefaultPricingResponse,
  AffiliateGroupRate,
  AffiliateGroupRateInput,
  AffiliatePricingResponse,
  AffiliateUserUpstreamRequest,
  AffiliateUserUpstreamResponse,
} from '@/components/affiliate/types'

export type {
  AffiliateDefaultPricingResponse as DefaultDistributionPricingResponse,
  AffiliateGroupRate,
  AffiliateGroupRateInput,
  AffiliatePricingResponse,
  AffiliateUserUpstreamRequest as UpdateUserUpstreamRequest,
  AffiliateUserUpstreamResponse as UserUpstreamResponse,
} from '@/components/affiliate/types'

type DefaultDistributionPricingResponse = AffiliateDefaultPricingResponse
type UpdateUserUpstreamRequest = AffiliateUserUpstreamRequest
type UserUpstreamResponse = AffiliateUserUpstreamResponse

export interface AffiliateAdminEntry {
  user_id: number
  email: string
  username: string
  aff_code: string
  aff_code_custom?: boolean
  inviter_id?: number | null
  inviter_email?: string | null
  inviter_username?: string | null
  aff_count: number
  today_revenue_usd?: number
  current_rebate_balance?: number
  rank?: number
}

export interface AffiliateDistributionTreeNode {
  user_id: number
  inviter_id?: number | null
  inviter_email?: string | null
  inviter_username?: string | null
  email: string
  username: string
  invite_code: string
  depth: number
  is_admin: boolean
  is_root_admin?: boolean | null
  is_agent?: boolean | null
  current_rebate_balance_rmb: number
  today_revenue_usd?: number | null
  today_business_usd?: number | null
  today_revenue_rmb?: number | null
  today_business_rmb?: number | null
  today_rebate_rmb?: number | null
  direct_children_count?: number | null
  direct_member_count?: number | null
  direct_user_count?: number | null
  direct_agent_count?: number | null
  direct_count?: number | null
  invite_group_rates?: AffiliateGroupRate[]
  current_group_rates?: AffiliateGroupRate[]
}

export interface ListAffiliateUsersParams {
  page?: number
  page_size?: number
  search?: string
}

export interface ListAffiliateRecordsParams {
  page?: number
  page_size?: number
  search?: string
  stat_date?: string
  month?: string
  start_at?: string
  end_at?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  timezone?: string
}

export interface AffiliateInviteCodeRateConfig {
  invite_code: string
  group_rates: AffiliateGroupRate[]
  created_at?: string | null
  updated_at?: string | null
}

export interface AffiliateChildAgent {
  user_id: number
  email: string
  username: string
  inviter_id?: number | null
  role?: 'agent' | 'user' | string
  today_revenue_usd?: number
  current_rebate_balance?: number
  group_rates?: AffiliateGroupRate[]
  created_at?: string | null
  updated_at?: string | null
}

export interface AffiliateRevenueRecord {
  stat_date: string
  agent_id: number
  agent_email: string
  agent_username: string
  direct_user_count?: number
  direct_agent_count?: number
  revenue_usd: number
  rank?: number
}

export interface AffiliateRebateRecord {
  agent_id: number
  agent_email: string
  agent_username: string
  current_rebate_balance: number
  monthly_rebate_amount?: number
  rank?: number
  updated_at?: string | null
}

export interface AffiliateMonthlyArchiveRecord {
  archive_month: string
  agent_id: number
  agent_email: string
  agent_username: string
  archived_rebate_balance: number
  created_at: string
}

export interface AffiliateBalanceAdjustmentRecord {
  id: number
  agent_id: number
  agent_email: string
  agent_username: string
  before_amount: number
  after_amount: number
  remark?: string | null
  operator_id: number
  operator_email?: string | null
  created_at: string
}

export interface AffiliateUserOverview {
  user_id: number
  email: string
  username: string
  aff_code: string
  inviter_id?: number | null
  inviter_email?: string | null
  inviter_username?: string | null
  invite_code_configs?: AffiliateInviteCodeRateConfig[]
  direct_children?: AffiliateChildAgent[]
  invited_count: number
  today_revenue_usd?: number
  current_rebate_balance?: number
  monthly_rebate_amount?: number
}

export interface UpdateAffiliateUserRequest {
  aff_code?: string
  inviter_id?: number | null
  invite_code_rates?: AffiliateGroupRateInput[]
  group_rates?: AffiliateGroupRateInput[]
}

export interface SetAffiliateRebateBalanceRequest {
  amount: number
  remark?: string
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

export interface UpdateAffiliateAgentPermissionsRequest {
  can_view_downline_daily_revenue: boolean
  can_view_downline_rebate_balances: boolean
  can_manage_downline_pricing: boolean
}

export interface SimpleUser {
  id: number
  email: string
  username: string
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

function normalizeGroupRates(data: unknown): AffiliateGroupRate[] {
  const rawRates = Array.isArray(data)
    ? data
    : Array.isArray((data as any)?.group_rates)
      ? (data as any).group_rates
      : Array.isArray((data as any)?.current_group_rates)
        ? (data as any).current_group_rates
        : Array.isArray((data as any)?.invite_group_rates)
          ? (data as any).invite_group_rates
          : Array.isArray((data as any)?.default_group_rates)
            ? (data as any).default_group_rates
            : []

  return rawRates
    .map((raw: any) => {
      const groupID = Number(raw?.group_id)
      const rateMultiplier = Number(raw?.rate_multiplier ?? NaN)
      if (!Number.isInteger(groupID) || groupID <= 0 || !Number.isFinite(rateMultiplier)) return null
      return {
        group_id: groupID,
        group_name: typeof raw?.group_name === 'string' ? raw.group_name : undefined,
        group_platform: typeof raw?.group_platform === 'string' ? raw.group_platform : undefined,
        group_rate_multiplier: raw?.group_rate_multiplier != null && Number.isFinite(Number(raw.group_rate_multiplier))
          ? Number(raw.group_rate_multiplier)
          : undefined,
        rate_multiplier: rateMultiplier,
        source_type: typeof raw?.source_type === 'string' ? raw.source_type : undefined,
        source_aff_code: typeof raw?.source_aff_code === 'string' ? raw.source_aff_code : undefined,
        upstream_user_id: raw?.upstream_user_id != null && Number.isInteger(Number(raw.upstream_user_id)) && Number(raw.upstream_user_id) > 0
          ? Number(raw.upstream_user_id)
          : null,
        updated_at: typeof raw?.updated_at === 'string' ? raw.updated_at : null,
      } satisfies AffiliateGroupRate
    })
    .filter((item: AffiliateGroupRate | null): item is AffiliateGroupRate => item !== null)
}

function normalizePricingResponse(data: unknown, fallbackUserId?: number): AffiliatePricingResponse {
  return {
    user_id: Number.isFinite(Number((data as any)?.user_id)) ? Number((data as any).user_id) : fallbackUserId,
    group_rates: normalizeGroupRates(data),
    updated_at: typeof (data as any)?.updated_at === 'string' ? (data as any).updated_at : null,
  }
}

function normalizeDefaultPricingResponse(data: unknown): DefaultDistributionPricingResponse {
  return {
    group_rates: normalizeGroupRates(data),
    updated_at: typeof (data as any)?.updated_at === 'string' ? (data as any).updated_at : null,
  }
}

function normalizeUpstreamResponse(
  data: unknown,
  userId: number,
  payload: UpdateUserUpstreamRequest,
): UserUpstreamResponse {
  const fallbackUpstreamUserId = payload.upstream_user_id ?? payload.inviter_id ?? null
  const upstreamUserID = Number.isFinite(Number((data as any)?.upstream_user_id))
    ? Number((data as any).upstream_user_id)
    : fallbackUpstreamUserId
  const inviterID = Number.isFinite(Number((data as any)?.inviter_id))
    ? Number((data as any).inviter_id)
    : upstreamUserID

  return {
    user_id: Number.isFinite(Number((data as any)?.user_id)) ? Number((data as any).user_id) : userId,
    inviter_id: inviterID,
    upstream_user_id: upstreamUserID,
    updated_at: typeof (data as any)?.updated_at === 'string' ? (data as any).updated_at : null,
  }
}

function createMissingGroupRatesError(path: string): Error {
  return Object.assign(new Error(`Expected non-empty group_rates from ${path}`), {
    code: 'INVALID_GROUP_RATES_RESPONSE',
  })
}

async function ensurePricingUpdateResult(
  path: string,
  data: unknown,
  fallback: () => Promise<AffiliatePricingResponse>,
  fallbackUserId?: number,
): Promise<AffiliatePricingResponse> {
  const normalized = normalizePricingResponse(data, fallbackUserId)
  if (normalized.group_rates.length > 0) {
    return normalized
  }

  const verified = await fallback()
  if (verified.group_rates.length > 0) {
    return verified
  }

  throw createMissingGroupRatesError(path)
}

async function ensureDefaultPricingUpdateResult(
  path: string,
  data: unknown,
  fallback: () => Promise<DefaultDistributionPricingResponse>,
): Promise<DefaultDistributionPricingResponse> {
  const normalized = normalizeDefaultPricingResponse(data)
  if (normalized.group_rates.length > 0) {
    return normalized
  }

  const verified = await fallback()
  if (verified.group_rates.length > 0) {
    return verified
  }

  throw createMissingGroupRatesError(path)
}

export async function listUsers(
  params: ListAffiliateUsersParams = {},
): Promise<PaginatedResponse<AffiliateAdminEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateAdminEntry>>(
    '/admin/affiliates/users',
    {
      params: {
        page: params.page ?? 1,
        page_size: params.page_size ?? 20,
        search: params.search ?? '',
      },
    },
  )
  return data
}

export async function lookupUsers(q: string): Promise<SimpleUser[]> {
  const { data } = await apiClient.get<SimpleUser[]>(
    '/admin/affiliates/users/lookup',
    { params: { q } },
  )
  return data
}

export async function updateUserSettings(
  userId: number,
  payload: UpdateAffiliateUserRequest,
): Promise<{ user_id: number }> {
  const { data } = await apiClient.put<{ user_id: number }>(
    `/admin/affiliates/users/${userId}`,
    payload,
  )
  return data
}

export async function clearUserSettings(
  userId: number,
): Promise<{ user_id: number }> {
  const { data } = await apiClient.delete<{ user_id: number }>(
    `/admin/affiliates/users/${userId}`,
  )
  return data
}

function recordParams(params: ListAffiliateRecordsParams = {}) {
  return {
    page: params.page ?? 1,
    page_size: params.page_size ?? 20,
    search: params.search ?? '',
    stat_date: params.stat_date || undefined,
    month: params.month || undefined,
    start_at: params.start_at || undefined,
    end_at: params.end_at || undefined,
    sort_by: params.sort_by || undefined,
    sort_order: params.sort_order || undefined,
    timezone: params.timezone || undefined,
  }
}

export async function listInviteRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateRevenueRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateRevenueRecord>>(
    '/admin/affiliates/daily-revenue',
    { params: recordParams(params) },
  )
  return data
}

export async function listRebateRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateRebateRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateRebateRecord>>(
    '/admin/affiliates/rebate-balances',
    { params: recordParams(params) },
  )
  return data
}

export async function listTransferRecords(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateMonthlyArchiveRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<AffiliateMonthlyArchiveRecord>>(
    '/admin/affiliates/monthly-archives',
    { params: recordParams(params) },
  )
  return data
}

export async function listDailyRevenueLeaderboard(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateRevenueRecord>> {
  return listInviteRecords(params)
}

export async function listRebateBalanceLeaderboard(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateRebateRecord>> {
  return listRebateRecords(params)
}

export async function listMonthlyArchives(
  params: ListAffiliateRecordsParams = {},
): Promise<PaginatedResponse<AffiliateMonthlyArchiveRecord>> {
  return listTransferRecords(params)
}

export async function getUserOverview(
  userId: number,
): Promise<AffiliateUserOverview> {
  const { data } = await apiClient.get<AffiliateUserOverview>(
    `/admin/affiliates/users/${userId}/overview`,
  )
  return data
}

export async function setUserRebateBalance(
  userId: number,
  payload: SetAffiliateRebateBalanceRequest,
): Promise<{ user_id: number; amount: number }> {
  const { data } = await apiClient.put<{ user_id: number; amount: number }>(
    `/admin/affiliates/users/${userId}/rebate-balance`,
    payload,
  )
  return data
}

export async function getDistributionTree(params: { search?: string; root_user_id?: number } = {}): Promise<AffiliateDistributionTreeNode[]> {
  const { data } = await apiClient.get<AffiliateDistributionTreeNode[]>('/admin/affiliates/tree', {
    params: {
      search: params.search || undefined,
      root_user_id: params.root_user_id || undefined,
    },
  })
  return data
}

export async function getDefaultDistributionPricing(): Promise<DefaultDistributionPricingResponse> {
  const { data } = await apiClient.get<unknown>('/admin/affiliates/default-pricing')
  return normalizeDefaultPricingResponse(data)
}

export async function updateDefaultDistributionPricing(
  payload: { group_rates: AffiliateGroupRateInput[] },
): Promise<DefaultDistributionPricingResponse> {
  const groupRates = payload.group_rates.map((rate) => ({
    group_id: rate.group_id,
    rate_multiplier: rate.rate_multiplier,
  }))
  const { data } = await apiClient.put<unknown>('/admin/affiliates/default-pricing', { group_rates: groupRates })
  return ensureDefaultPricingUpdateResult('/admin/affiliates/default-pricing', data, () => getDefaultDistributionPricing())
}

export async function getUserDistributionPricing(
  userId: number,
): Promise<AffiliatePricingResponse> {
  const { data } = await apiClient.get<unknown>(`/admin/affiliates/users/${userId}/pricing`)
  return normalizePricingResponse(data, userId)
}

export async function updateUserDistributionPricing(
  userId: number,
  payload: { group_rates: AffiliateGroupRateInput[] },
): Promise<AffiliatePricingResponse> {
  const groupRates = payload.group_rates.map((rate) => ({
    group_id: rate.group_id,
    rate_multiplier: rate.rate_multiplier,
  }))
  const { data } = await apiClient.put<unknown>(`/admin/affiliates/users/${userId}/pricing`, { group_rates: groupRates })
  return ensurePricingUpdateResult(
    `/admin/affiliates/users/${userId}/pricing`,
    data,
    () => getUserDistributionPricing(userId),
    userId,
  )
}

export async function getUserInvitePricing(
  userId: number,
): Promise<AffiliatePricingResponse> {
  const { data } = await apiClient.get<unknown>(`/admin/affiliates/users/${userId}/invite-pricing`)
  return normalizePricingResponse(data, userId)
}

export async function updateUserInvitePricing(
  userId: number,
  payload: { group_rates: AffiliateGroupRateInput[] },
): Promise<AffiliatePricingResponse> {
  const groupRates = payload.group_rates.map((rate) => ({
    group_id: rate.group_id,
    rate_multiplier: rate.rate_multiplier,
  }))
  const { data } = await apiClient.put<unknown>(`/admin/affiliates/users/${userId}/invite-pricing`, { group_rates: groupRates })
  return ensurePricingUpdateResult(
    `/admin/affiliates/users/${userId}/invite-pricing`,
    data,
    () => getUserInvitePricing(userId),
    userId,
  )
}

export async function updateUserUpstream(
  userId: number,
  payload: UpdateUserUpstreamRequest,
): Promise<UserUpstreamResponse> {
  const { data } = await apiClient.put<unknown>(`/admin/affiliates/users/${userId}/upstream`, payload)
  return normalizeUpstreamResponse(data, userId, payload)
}

export async function getUserDistributionPermissions(
  userId: number,
): Promise<AffiliateAgentPermissions> {
  const { data } = await apiClient.get<AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number }>(`/admin/affiliates/users/${userId}/permissions`)
  return normalizePermissionsResponse(data, userId)
}

export async function updateUserDistributionPermissions(
  userId: number,
  payload: UpdateAffiliateAgentPermissionsRequest,
): Promise<AffiliateAgentPermissions> {
  const { data } = await apiClient.put<AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number }>(`/admin/affiliates/users/${userId}/permissions`, payload)
  return normalizePermissionsResponse(data, userId)
}

export const affiliatesAPI = {
  listUsers,
  lookupUsers,
  updateUserSettings,
  clearUserSettings,
  listDailyRevenueLeaderboard,
  listRebateBalanceLeaderboard,
  listMonthlyArchives,
  listInviteRecords,
  listRebateRecords,
  listTransferRecords,
  getUserOverview,
  setUserRebateBalance,
  getDistributionTree,
  getDefaultDistributionPricing,
  updateDefaultDistributionPricing,
  getUserDistributionPricing,
  updateUserDistributionPricing,
  getUserInvitePricing,
  updateUserInvitePricing,
  updateUserUpstream,
  getUserDistributionPermissions,
  updateUserDistributionPermissions,
}

export default affiliatesAPI
