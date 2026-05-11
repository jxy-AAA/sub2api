export interface AffiliateGroupRate {
  group_id: number
  group_name?: string
  group_platform?: string
  group_rate_multiplier?: number
  rate_multiplier: number
  source_type?: string
  source_aff_code?: string
  upstream_user_id?: number | null
  updated_at?: string | null
}

export interface AffiliateGroupRateInput {
  group_id: number
  rate_multiplier: number
}

export interface AffiliatePricingResponse {
  user_id?: number
  group_rates: AffiliateGroupRate[]
  updated_at?: string | null
}

export interface AffiliateDefaultPricingResponse {
  group_rates: AffiliateGroupRate[]
  updated_at?: string | null
}

export interface AffiliateUserUpstreamRequest {
  inviter_id?: number | null
  upstream_user_id?: number | null
}

export interface AffiliateUserUpstreamResponse {
  user_id: number
  inviter_id?: number | null
  upstream_user_id?: number | null
  updated_at?: string | null
}

export interface AffiliateRawGroupRate {
  group_id?: number | null
  group_name?: string | null
  group_platform?: string | null
  group_rate_multiplier?: number | null
  rate_multiplier?: number | null
  source_type?: string | null
  source_aff_code?: string | null
  upstream_user_id?: number | null
  updated_at?: string | null
}

export interface AffiliateGroupOption {
  id: number
  name: string
  platform?: string
  rate_multiplier?: number
}

export interface AffiliateDirectChild {
  user_id: number
  email: string
  username: string
  role?: 'agent' | 'user' | string
  joined_at?: string | null
  today_revenue_usd: number
  today_rebate_rmb: number
  current_rebate_balance_rmb: number
  group_rates: AffiliateGroupRate[]
}

export interface AffiliateDirectChildResponse {
  user_id: number
  email?: string | null
  username?: string | null
  role?: 'agent' | 'user' | string
  is_agent?: boolean | null
  joined_at?: string | null
  created_at?: string | null
  today_revenue_usd?: number | null
  today_business_usd?: number | null
  today_rebate_rmb?: number | null
  current_rebate_balance_rmb?: number | null
  group_rates?: AffiliateRawGroupRate[] | null
  current_group_rates?: AffiliateRawGroupRate[] | null
}

export interface AffiliateDistributionDetail {
  user_id: number
  aff_code: string
  inviter_id?: number | null
  invite_group_rates: AffiliateGroupRate[]
  my_group_rates: AffiliateGroupRate[]
  today_revenue_usd: number
  today_rebate_rmb: number
  current_rebate_balance_rmb: number
  direct_children: AffiliateDirectChild[]
  direct_children_count: number
}

export interface AffiliateDistributionDetailResponse {
  user_id: number
  aff_code?: string | null
  invite_code?: string | null
  inviter_id?: number | null
  invite_group_rates?: AffiliateRawGroupRate[] | null
  my_group_rates?: AffiliateRawGroupRate[] | null
  current_group_rates?: AffiliateRawGroupRate[] | null
  group_rates?: AffiliateRawGroupRate[] | null
  today_revenue_usd?: number | null
  today_business_usd?: number | null
  today_rebate_rmb?: number | null
  current_rebate_balance_rmb?: number | null
  direct_children?: AffiliateDirectChildResponse[] | null
  direct_children_count?: number | null
  direct_member_count?: number | null
}
