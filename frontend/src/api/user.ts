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
import type {
  AffiliateDistributionDetail,
  AffiliateDistributionDetailResponse,
  AffiliateDirectChild,
  AffiliateDirectChildResponse,
  AffiliateGroupRate,
  AffiliateGroupRateInput,
  AffiliatePricingResponse,
  AffiliateRawGroupRate,
} from '@/components/affiliate/types'

export type {
  AffiliateDistributionDetail,
  AffiliateDirectChild,
  AffiliateGroupRate,
  AffiliateGroupRateInput,
  AffiliatePricingResponse,
} from '@/components/affiliate/types'

export interface AffiliateInviteCodePricing {
  group_rates: AffiliateGroupRate[]
  updated_at?: string | null
}

export interface AffiliateDailyRevenueEntry {
  stat_date: string
  user_id?: number
  email?: string
  username?: string
  agent_id: number
  agent_email: string
  agent_username: string
  business_usd?: number
  business_rmb?: number
  direct_user_count?: number
  direct_agent_count?: number
  direct_users?: number
  direct_agents?: number
  direct_total_usage_usd?: number
  direct_total_usage_rmb?: number
  direct_user_usage_usd?: number
  direct_user_usage_rmb?: number
  direct_agent_usage_usd?: number
  direct_agent_usage_rmb?: number
  revenue_usd: number
  rank?: number
}

export interface AffiliateRebateBalanceEntry {
  user_id?: number
  email?: string
  username?: string
  agent_id: number
  agent_email: string
  agent_username: string
  current_rebate_balance_rmb: number
  today_rebate_rmb?: number
  monthly_rebate_rmb?: number
  direct_user_count?: number
  direct_agent_count?: number
  direct_users?: number
  direct_agents?: number
  direct_total_usage_usd?: number
  direct_total_usage_rmb?: number
  direct_user_usage_usd?: number
  direct_user_usage_rmb?: number
  direct_agent_usage_usd?: number
  direct_agent_usage_rmb?: number
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

export interface UpdateInviteCodeGroupRatesRequest {
  invite_code?: string
  group_rates: AffiliateGroupRateInput[]
}

export interface UpdateDirectChildGroupRatesRequest {
  group_rates: AffiliateGroupRateInput[]
}

export type UserAffiliateOverview = AffiliateDistributionDetail
export type UserAffiliateDetail = AffiliateDistributionDetail

export interface AffiliateTransferResponse {
  transferred_quota: number
  balance: number
}

function toFiniteAffiliateNumber(value: unknown, fallback = 0): number {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : fallback
}

function normalizeAffiliateGroupRates(rates?: AffiliateRawGroupRate[] | null): AffiliateGroupRate[] {
  return (rates ?? [])
    .map((rate) => {
      const groupId = Number(rate.group_id)
      const rateMultiplier = toFiniteAffiliateNumber(rate.rate_multiplier, Number.NaN)

      return {
        group_id: groupId,
        group_name: rate.group_name ?? undefined,
        group_platform: rate.group_platform ?? undefined,
        group_rate_multiplier: rate.group_rate_multiplier ?? undefined,
        rate_multiplier: rateMultiplier,
        source_type: rate.source_type ?? undefined,
        source_aff_code: rate.source_aff_code ?? undefined,
        upstream_user_id: rate.upstream_user_id ?? null,
        updated_at: rate.updated_at ?? null,
      } satisfies AffiliateGroupRate
    })
    .filter((rate) => rate.group_id > 0 && Number.isFinite(rate.rate_multiplier) && rate.rate_multiplier > 0)
}

function normalizeAffiliateDirectChild(child: AffiliateDirectChildResponse): AffiliateDirectChild {
  return {
    user_id: child.user_id,
    email: child.email ?? '',
    username: child.username ?? '',
    role: child.role ?? (child.is_agent ? 'agent' : 'user'),
    joined_at: child.joined_at ?? child.created_at ?? null,
    today_revenue_usd: toFiniteAffiliateNumber(child.today_revenue_usd ?? child.today_business_usd, 0),
    today_business_rmb: toFiniteAffiliateNumber(child.today_business_rmb ?? child.direct_total_usage_rmb, 0),
    today_rebate_rmb: toFiniteAffiliateNumber(child.today_rebate_rmb, 0),
    direct_total_usage_usd: toFiniteAffiliateNumber(child.direct_total_usage_usd ?? child.today_revenue_usd ?? child.today_business_usd, 0),
    direct_total_usage_rmb: toFiniteAffiliateNumber(child.direct_total_usage_rmb ?? child.today_business_rmb, 0),
    direct_user_usage_usd: toFiniteAffiliateNumber(child.direct_user_usage_usd, 0),
    direct_user_usage_rmb: toFiniteAffiliateNumber(child.direct_user_usage_rmb, 0),
    direct_agent_usage_usd: toFiniteAffiliateNumber(child.direct_agent_usage_usd, 0),
    direct_agent_usage_rmb: toFiniteAffiliateNumber(child.direct_agent_usage_rmb, 0),
    current_rebate_balance_rmb: toFiniteAffiliateNumber(child.current_rebate_balance_rmb, 0),
    group_rates: normalizeAffiliateGroupRates(child.current_group_rates ?? child.group_rates),
  }
}

function normalizeAffiliateOverview(raw: AffiliateDistributionDetailResponse): UserAffiliateOverview {
  const directChildren = (raw.direct_children ?? []).map(normalizeAffiliateDirectChild)

  return {
    user_id: raw.user_id,
    aff_code: raw.aff_code ?? raw.invite_code ?? '',
    inviter_id: raw.inviter_id ?? null,
    invite_group_rates: normalizeAffiliateGroupRates(raw.invite_group_rates),
    my_group_rates: normalizeAffiliateGroupRates(raw.my_group_rates ?? raw.current_group_rates ?? raw.group_rates),
    today_revenue_usd: toFiniteAffiliateNumber(raw.today_revenue_usd ?? raw.today_business_usd, 0),
    today_business_rmb: toFiniteAffiliateNumber(raw.today_business_rmb, 0),
    today_rebate_rmb: toFiniteAffiliateNumber(raw.today_rebate_rmb, 0),
    current_rebate_balance_rmb: toFiniteAffiliateNumber(raw.current_rebate_balance_rmb, 0),
    direct_children: directChildren,
    direct_children_count: toFiniteAffiliateNumber(raw.direct_children_count ?? raw.direct_member_count, directChildren.length),
  }
}

function normalizeAffiliatePricingResponse(data: unknown, fallbackUserId?: number): AffiliatePricingResponse {
  const payload = data as {
    user_id?: number
    group_rates?: AffiliateRawGroupRate[]
    current_group_rates?: AffiliateRawGroupRate[]
    invite_group_rates?: AffiliateRawGroupRate[]
  } | null

  return {
    user_id: Number.isFinite(Number(payload?.user_id)) ? Number(payload?.user_id) : fallbackUserId,
    group_rates: normalizeAffiliateGroupRates(
      payload?.group_rates
        ?? payload?.current_group_rates
        ?? payload?.invite_group_rates
        ?? [],
    ),
    updated_at: typeof (data as any)?.updated_at === 'string' ? (data as any).updated_at : null,
  }
}

function createMissingGroupRatesError(path: string): Error {
  return Object.assign(new Error(`Expected non-empty group_rates from ${path}`), {
    code: 'INVALID_GROUP_RATES_RESPONSE',
  })
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
  const { data } = await apiClient.get<AffiliateDistributionDetailResponse>('/user/aff/distribution')
  return normalizeAffiliateOverview(data)
}

export async function getAffiliateDetail(): Promise<UserAffiliateDetail> {
  return getAffiliateOverview()
}

export async function getInviteCodeGroupRates(): Promise<AffiliatePricingResponse> {
  const { data } = await apiClient.get<unknown>('/user/aff/invite-pricing')
  return normalizeAffiliatePricingResponse(data)
}

export async function updateInviteCodeGroupRates(
  payload: UpdateInviteCodeGroupRatesRequest,
): Promise<AffiliatePricingResponse> {
  const { data } = await apiClient.put<unknown>(
    '/user/aff/invite-pricing',
    {
      ...(payload.invite_code ? { invite_code: payload.invite_code } : {}),
      group_rates: payload.group_rates.map((rate) => ({
        group_id: rate.group_id,
        rate_multiplier: rate.rate_multiplier,
      })),
    },
  )

  const normalized = normalizeAffiliatePricingResponse(data)
  if (normalized.group_rates.length > 0) {
    return normalized
  }

  const verified = await getInviteCodeGroupRates()
  if (verified.group_rates.length > 0) {
    return verified
  }

  throw createMissingGroupRatesError('/user/aff/invite-pricing')
}

export async function listAffiliateDirectChildren(
  params: ListAffiliateStatsParams = {},
): Promise<AffiliateDirectChild[]> {
  const { data } = await apiClient.get<AffiliateDirectChildResponse[]>('/user/aff/direct-members', { params })
  return data.map(normalizeAffiliateDirectChild)
}

export async function updateAffiliateDirectChildGroupRates(
  userId: number,
  payload: UpdateDirectChildGroupRatesRequest,
): Promise<AffiliatePricingResponse> {
  const { data } = await apiClient.put<unknown>(
    `/user/aff/direct-members/${userId}/pricing`,
    {
      group_rates: payload.group_rates.map((rate) => ({
        group_id: rate.group_id,
        rate_multiplier: rate.rate_multiplier,
      })),
    },
  )

  const normalized = normalizeAffiliatePricingResponse(data, userId)
  if (normalized.group_rates.length > 0) {
    return normalized
  }

  const verifiedChild = (await listAffiliateDirectChildren()).find((child) => child.user_id === userId)
  if (verifiedChild && verifiedChild.group_rates.length > 0) {
    return {
      user_id: userId,
      group_rates: verifiedChild.group_rates,
    }
  }

  throw createMissingGroupRatesError(`/user/aff/direct-members/${userId}/pricing`)
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
  getInviteCodeGroupRates,
  updateInviteCodeGroupRates,
  listAffiliateDirectChildren,
  updateAffiliateDirectChildGroupRates,
  listAffiliateDailyRevenue,
  listAffiliateRebateBalances,
  listAffiliateMonthlyArchives,
  transferAffiliateQuota,
}

export default userAPI
