export interface AffiliateModelRate {
  model: string
  multiplier: number
}

export interface AffiliateRawModelRate {
  model?: string
  model_name?: string
  multiplier?: number | null
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
  model_rates: AffiliateModelRate[]
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
  model_rates?: AffiliateRawModelRate[] | null
  current_model_rates?: AffiliateRawModelRate[] | null
}

export interface AffiliateDistributionDetail {
  user_id: number
  aff_code: string
  inviter_id?: number | null
  invite_code_model_rates: AffiliateModelRate[]
  my_model_rates: AffiliateModelRate[]
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
  invite_code_model_rates?: AffiliateRawModelRate[] | null
  invite_model_rates?: AffiliateRawModelRate[] | null
  my_model_rates?: AffiliateRawModelRate[] | null
  today_revenue_usd?: number | null
  today_business_usd?: number | null
  today_rebate_rmb?: number | null
  current_rebate_balance_rmb?: number | null
  direct_children?: AffiliateDirectChildResponse[] | null
  direct_children_count?: number | null
  direct_member_count?: number | null
}
