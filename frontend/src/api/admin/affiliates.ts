/**
 * Admin affiliate distribution API endpoints.
 * Covers agent hierarchy management, daily revenue rankings,
 * rebate balance rankings and monthly archives.
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

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
  today_rebate_amount?: number
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
  today_rebate_amount?: number | null
  today_rebate_rmb?: number | null
  direct_children_count?: number | null
  direct_member_count?: number | null
  direct_user_count?: number | null
  direct_agent_count?: number | null
  direct_count?: number | null
  invite_model_rates?: AffiliateModelRate[]
  current_model_rates?: AffiliateModelRate[]
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

export interface AffiliateModelRate {
  model?: string
  model_name: string
  multiplier: number
  parent_multiplier?: number | null
  source?: 'invite_code' | 'user_override' | 'default' | string
  updated_at?: string | null
}

export interface AffiliateInviteCodeRateConfig {
  invite_code: string
  model_rates: AffiliateModelRate[]
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
  today_rebate_amount?: number
  current_rebate_balance?: number
  model_rates?: AffiliateModelRate[]
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
  today_rebate_amount?: number
  current_rebate_balance?: number
  monthly_rebate_amount?: number
}

export interface UpdateAffiliateUserRequest {
  aff_code?: string
  inviter_id?: number | null
  invite_code_rates?: Array<{
    model_name: string
    multiplier: number
  }>
  model_rates?: Array<{
    model_name: string
    multiplier: number
  }>
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

async function getWithFallback<T>(paths: string[]): Promise<T> {
  let lastError: unknown
  for (const path of paths) {
    try {
      const { data } = await apiClient.get<T>(path)
      return data
    } catch (error: any) {
      lastError = error
      if (!isRouteFallback404(error)) {
        throw error
      }
    }
  }
  throw lastError
}

async function putWithFallback<T>(paths: string[], payload: object): Promise<T> {
  let lastError: unknown
  for (const path of paths) {
    try {
      const { data } = await apiClient.put<T>(path, payload)
      return data
    } catch (error: any) {
      lastError = error
      if (!isRouteFallback404(error)) {
        throw error
      }
    }
  }
  throw lastError
}

function isRouteFallback404(error: any): boolean {
  if (error?.status !== 404) return false
  const reason = String(error?.reason || error?.code || '')
  return reason === '' || reason === 'NOT_FOUND' || reason === 'ROUTE_NOT_FOUND'
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

export async function getUserDistributionPricing(
  userId: number,
): Promise<{ user_id: number; model_rates: AffiliateModelRate[] }> {
  const { data } = await apiClient.get<{ user_id: number; model_rates: AffiliateModelRate[] }>(
    `/admin/affiliates/users/${userId}/pricing`,
  )
  return data
}

export async function updateUserDistributionPricing(
  userId: number,
  payload: { model_rates: Array<{ model_name?: string; model?: string; multiplier: number }> },
): Promise<{ user_id: number; model_rates: AffiliateModelRate[] }> {
  const modelRates = payload.model_rates.map((rate) => ({
    model: rate.model || rate.model_name || '',
    multiplier: rate.multiplier,
  }))
  const { data } = await apiClient.put<{ user_id: number; model_rates: AffiliateModelRate[] }>(
    `/admin/affiliates/users/${userId}/pricing`,
    { model_rates: modelRates },
  )
  return data
}

export async function getUserDistributionPermissions(
  userId: number,
): Promise<AffiliateAgentPermissions> {
  const data = await getWithFallback<AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number }>([
    `/admin/affiliates/users/${userId}/permissions`,
    `/admin/affiliates/users/${userId}/agent-permissions`,
  ])
  return normalizePermissionsResponse(data, userId)
}

export async function updateUserDistributionPermissions(
  userId: number,
  payload: UpdateAffiliateAgentPermissionsRequest,
): Promise<AffiliateAgentPermissions> {
  const data = await putWithFallback<AffiliateAgentPermissions | { permissions?: AffiliateAgentPermissions; user_id?: number }>(
    [
      `/admin/affiliates/users/${userId}/permissions`,
      `/admin/affiliates/users/${userId}/agent-permissions`,
    ],
    payload,
  )
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
  getUserDistributionPricing,
  updateUserDistributionPricing,
  getUserDistributionPermissions,
  updateUserDistributionPermissions,
}

export default affiliatesAPI
