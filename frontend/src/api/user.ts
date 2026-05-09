/**
 * User API endpoints
 * Handles user profile management and password changes
 */

import { apiClient } from './client'
import {
  resolveWeChatOAuthStartStrict,
  prepareOAuthBindAccessTokenCookie,
  type WeChatOAuthPublicSettings,
} from './auth'
import type {
  User,
  ChangePasswordRequest,
  NotifyEmailEntry,
  UserAuthProvider,
} from '@/types'

export interface AffiliateModelRate {
  model_name: string
  multiplier: number
  parent_multiplier?: number | null
  source?: 'invite_code' | 'user_override' | 'default' | string
  updated_at?: string | null
}

export interface AffiliateInviteCodePricing {
  invite_code: string
  model_rates: AffiliateModelRate[]
  updated_at?: string | null
}

export interface AffiliateDirectChild {
  user_id: number
  email: string
  username: string
  inviter_id?: number | null
  role?: 'agent' | 'user' | string
  today_revenue_usd?: number
  today_rebate_rmb?: number
  current_rebate_balance_rmb?: number
  model_rates?: AffiliateModelRate[]
  created_at?: string | null
  updated_at?: string | null
}

export interface AffiliateDailyRevenueEntry {
  stat_date: string
  agent_id: number
  agent_email: string
  agent_username: string
  direct_user_count?: number
  direct_agent_count?: number
  revenue_usd: number
  rank?: number
}

export interface AffiliateRebateBalanceEntry {
  agent_id: number
  agent_email: string
  agent_username: string
  current_rebate_balance_rmb: number
  monthly_rebate_rmb?: number
  rank?: number
  updated_at?: string | null
}

export interface AffiliateMonthlyArchiveEntry {
  archive_month: string
  agent_id: number
  agent_email: string
  agent_username: string
  archived_rebate_rmb: number
  created_at: string
}

export interface ListAffiliateStatsParams {
  page?: number
  page_size?: number
  search?: string
  start_date?: string
  end_date?: string
  stat_date?: string
  month?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  timezone?: string
}

export interface UpdateInviteCodeModelRatesRequest {
  invite_code?: string
  model_rates: Array<{
    model_name: string
    multiplier: number
  }>
}

export interface UpdateDirectChildModelRatesRequest {
  model_rates: Array<{
    model_name: string
    multiplier: number
  }>
}

export interface UserAffiliateOverview {
  user_id: number
  aff_code: string
  inviter_id?: number | null
  inviter_email?: string | null
  inviter_username?: string | null
  invite_code_pricing: AffiliateInviteCodePricing[]
  direct_children: AffiliateDirectChild[]
  direct_children_count: number
  today_revenue_usd: number
  today_rebate_rmb: number
  current_rebate_balance_rmb: number
  monthly_rebate_rmb?: number
}

export type UserAffiliateDetail = UserAffiliateOverview

export interface AffiliateTransferResponse {
  transferred_quota: number
  balance: number
}

/**
 * Get current user profile
 * @returns User profile data
 */
export async function getProfile(): Promise<User> {
  const { data } = await apiClient.get<User>('/user/profile')
  return data
}

/**
 * Update current user profile
 * @param profile - Profile data to update
 * @returns Updated user profile data
 */
export async function updateProfile(profile: {
  username?: string
  avatar_url?: string | null
  balance_notify_enabled?: boolean
  balance_notify_threshold?: number | null
  balance_notify_extra_emails?: NotifyEmailEntry[]
}): Promise<User> {
  const { data } = await apiClient.put<User>('/user', profile)
  return data
}

/**
 * Change current user password
 * @param passwords - Old and new password
 * @returns Success message
 */
export async function changePassword(
  oldPassword: string,
  newPassword: string
): Promise<{ message: string }> {
  const payload: ChangePasswordRequest = {
    old_password: oldPassword,
    new_password: newPassword
  }

  const { data } = await apiClient.put<{ message: string }>('/user/password', payload)
  return data
}

/**
 * Send verification code for adding a notify email
 * @param email - Email address to verify
 */
export async function sendNotifyEmailCode(email: string): Promise<void> {
  await apiClient.post('/user/notify-email/send-code', { email })
}

/**
 * Verify and add a notify email
 * @param email - Email address to add
 * @param code - Verification code
 */
export async function verifyNotifyEmail(email: string, code: string): Promise<void> {
  await apiClient.post('/user/notify-email/verify', { email, code })
}

/**
 * Remove a notify email
 * @param email - Email address to remove
 */
export async function removeNotifyEmail(email: string): Promise<void> {
  await apiClient.delete('/user/notify-email', { data: { email } })
}

/**
 * Toggle a notify email's disabled state
 * @param email - Email address (empty string for primary email placeholder)
 * @param disabled - Whether to disable the email
 */
export async function toggleNotifyEmail(email: string, disabled: boolean): Promise<User> {
  const { data } = await apiClient.put<User>('/user/notify-email/toggle', { email, disabled })
  return data
}

export async function sendEmailBindingCode(email: string): Promise<void> {
  await apiClient.post('/user/account-bindings/email/send-code', { email })
}

export async function bindEmailIdentity(payload: {
  email: string
  verify_code: string
  password: string
}): Promise<User> {
  const { data } = await apiClient.post<User>('/user/account-bindings/email', payload)
  return data
}

export async function unbindAuthIdentity(provider: BindableOAuthProvider): Promise<User> {
  const { data } = await apiClient.delete<User>(`/user/account-bindings/${provider}`)
  return data
}

export type BindableOAuthProvider = Exclude<UserAuthProvider, 'email'>

interface BuildOAuthBindingStartURLOptions {
  redirectTo?: string
  wechatOAuthSettings?: WeChatOAuthPublicSettings | null
}

export function resolveWeChatOAuthMode(): 'open' | 'mp' {
  if (typeof navigator === 'undefined') {
    return 'open'
  }
  return /MicroMessenger/i.test(navigator.userAgent) ? 'mp' : 'open'
}

function resolveWeChatOAuthBindingMode(
  settings?: WeChatOAuthPublicSettings | null
): 'open' | 'mp' | null {
  if (settings) {
    return resolveWeChatOAuthStartStrict(settings).mode
  }
  return resolveWeChatOAuthMode()
}

export function buildOAuthBindingStartURL(
  provider: BindableOAuthProvider,
  options: BuildOAuthBindingStartURLOptions = {}
): string | null {
  const redirectTo = options.redirectTo?.trim() || '/profile'
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
  const normalized = apiBase.replace(/\/$/, '')
  const params = new URLSearchParams({
    redirect: redirectTo,
    intent: 'bind_current_user'
  })

  if (provider === 'wechat') {
    const mode = resolveWeChatOAuthBindingMode(options.wechatOAuthSettings)
    if (!mode) {
      return null
    }
    params.set('mode', mode)
  }

  return `${normalized}/auth/oauth/${provider}/bind/start?${params.toString()}`
}

export async function startOAuthBinding(
  provider: BindableOAuthProvider,
  options: BuildOAuthBindingStartURLOptions = {}
): Promise<void> {
  if (typeof window === 'undefined') {
    return
  }
  const startURL = buildOAuthBindingStartURL(provider, options)
  if (!startURL) {
    return
  }
  await prepareOAuthBindAccessTokenCookie()
  window.location.href = startURL
}

export async function getAffiliateOverview(): Promise<UserAffiliateOverview> {
  const { data } = await apiClient.get<UserAffiliateOverview>('/user/aff')
  return data
}

export async function getAffiliateDetail(): Promise<UserAffiliateDetail> {
  return getAffiliateOverview()
}

export async function updateInviteCodeModelRates(
  payload: UpdateInviteCodeModelRatesRequest,
): Promise<AffiliateInviteCodePricing[]> {
  const { data } = await apiClient.put<AffiliateInviteCodePricing[]>(
    '/user/aff/invite-pricing',
    payload,
  )
  return data
}

export async function listAffiliateDirectChildren(
  params: ListAffiliateStatsParams = {},
): Promise<AffiliateDirectChild[]> {
  const { data } = await apiClient.get<AffiliateDirectChild[]>('/user/aff/direct-members', { params })
  return data
}

export async function updateAffiliateDirectChildModelRates(
  userId: number,
  payload: UpdateDirectChildModelRatesRequest,
): Promise<{ user_id: number; model_rates: AffiliateModelRate[] }> {
  const { data } = await apiClient.put<{ user_id: number; model_rates: AffiliateModelRate[] }>(
    `/user/aff/direct-members/${userId}/pricing`,
    payload,
  )
  return data
}

export async function listAffiliateDailyRevenue(
  params: ListAffiliateStatsParams = {},
): Promise<AffiliateDailyRevenueEntry[]> {
  const { data } = await apiClient.get<AffiliateDailyRevenueEntry[]>('/user/aff/daily-revenue', { params })
  return data
}

export async function listAffiliateRebateBalances(
  params: ListAffiliateStatsParams = {},
): Promise<AffiliateRebateBalanceEntry[]> {
  const { data } = await apiClient.get<AffiliateRebateBalanceEntry[]>('/user/aff/rebate-balances', { params })
  return data
}

export async function listAffiliateMonthlyArchives(
  params: ListAffiliateStatsParams = {},
): Promise<AffiliateMonthlyArchiveEntry[]> {
  const { data } = await apiClient.get<AffiliateMonthlyArchiveEntry[]>('/user/aff/monthly-archives', { params })
  return data
}

export async function transferAffiliateQuota(): Promise<AffiliateTransferResponse> {
  const { data } = await apiClient.post<AffiliateTransferResponse>('/user/aff/transfer')
  return data
}

export const userAPI = {
  getProfile,
  updateProfile,
  changePassword,
  sendNotifyEmailCode,
  verifyNotifyEmail,
  removeNotifyEmail,
  toggleNotifyEmail,
  sendEmailBindingCode,
  bindEmailIdentity,
  unbindAuthIdentity,
  buildOAuthBindingStartURL,
  startOAuthBinding,
  getAffiliateOverview,
  getAffiliateDetail,
  updateInviteCodeModelRates,
  listAffiliateDirectChildren,
  updateAffiliateDirectChildModelRates,
  listAffiliateDailyRevenue,
  listAffiliateRebateBalances,
  listAffiliateMonthlyArchives,
  transferAffiliateQuota,
}

export default userAPI
