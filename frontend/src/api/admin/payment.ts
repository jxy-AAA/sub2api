/**
 * Admin Payment API endpoints
 * Handles payment management operations for administrators
 */

import { apiClient } from '../client'
import type {
  DashboardStats,
  PaymentOrder,
  PaymentChannel,
  SubscriptionPlan,
  ProviderInstance
} from '@/types/payment'
import type { BasePaginationResponse, FetchOptions } from '@/types'

/** Admin-facing payment config returned by GET /admin/payment/config */
export interface AdminPaymentConfig {
  enabled: boolean
  min_amount: number
  max_amount: number
  daily_limit: number
  order_timeout_minutes: number
  max_pending_orders: number
  enabled_payment_types: string[]
  balance_disabled: boolean
  balance_recharge_multiplier: number
  load_balance_strategy: string
  product_name_prefix: string
  product_name_suffix: string
  help_image_url: string
  help_text: string
}

/** Fields accepted by PUT /admin/payment/config (all optional via pointer semantics) */
export interface UpdatePaymentConfigRequest {
  enabled?: boolean
  min_amount?: number
  max_amount?: number
  daily_limit?: number
  order_timeout_minutes?: number
  max_pending_orders?: number
  enabled_payment_types?: string[]
  balance_disabled?: boolean
  balance_recharge_multiplier?: number
  load_balance_strategy?: string
  product_name_prefix?: string
  product_name_suffix?: string
  help_image_url?: string
  help_text?: string
}

async function getData<T>(url: string, options?: FetchOptions): Promise<T> {
  const { data } = await apiClient.get<T>(url, {
    signal: options?.signal
  })
  return data
}

async function postData<T>(url: string, payload?: unknown, options?: FetchOptions): Promise<T> {
  const { data } = await apiClient.post<T>(url, payload, {
    signal: options?.signal
  })
  return data
}

async function putData<T>(url: string, payload?: unknown, options?: FetchOptions): Promise<T> {
  const { data } = await apiClient.put<T>(url, payload, {
    signal: options?.signal
  })
  return data
}

export const adminPaymentAPI = {
  // ==================== Config ====================

  /** Get payment configuration (admin view) */
  getConfig() {
    return apiClient.get<AdminPaymentConfig>('/admin/payment/config')
  },

  /** Update payment configuration */
  updateConfig(data: UpdatePaymentConfigRequest, options?: FetchOptions) {
    return putData<AdminPaymentConfig>('/admin/payment/config', data, options)
  },

  // ==================== Dashboard ====================

  /** Get payment dashboard statistics */
  getDashboard(days?: number) {
    return apiClient.get<DashboardStats>('/admin/payment/dashboard', {
      params: days ? { days } : undefined
    })
  },

  // ==================== Orders ====================

  /** Get all orders (paginated, with filters) */
  getOrders(params?: {
    page?: number
    page_size?: number
    status?: string
    payment_type?: string
    user_id?: number
    keyword?: string
    start_date?: string
    end_date?: string
    order_type?: string
  }) {
    return apiClient.get<BasePaginationResponse<PaymentOrder>>('/admin/payment/orders', { params })
  },

  /** Get a specific order by ID */
  getOrder(id: number) {
    return apiClient.get<PaymentOrder>(`/admin/payment/orders/${id}`)
  },

  /** Cancel an order (admin) */
  cancelOrder(id: number) {
    return apiClient.post(`/admin/payment/orders/${id}/cancel`)
  },

  /** Retry recharge for a failed order */
  retryRecharge(id: number) {
    return apiClient.post(`/admin/payment/orders/${id}/retry`)
  },

  /** Process a refund */
  refundOrder(id: number, data: { amount: number; reason: string; deduct_balance?: boolean; force?: boolean }) {
    return apiClient.post(`/admin/payment/orders/${id}/refund`, data)
  },

  // ==================== Channels ====================

  /** Get all payment channels */
  getChannels(options?: FetchOptions) {
    return getData<PaymentChannel[]>('/admin/payment/channels', options)
  },

  /** Create a payment channel */
  createChannel(data: Partial<PaymentChannel>, options?: FetchOptions) {
    return postData<PaymentChannel>('/admin/payment/channels', data, options)
  },

  /** Update a payment channel */
  updateChannel(id: number, data: Partial<PaymentChannel>, options?: FetchOptions) {
    return putData<PaymentChannel>(`/admin/payment/channels/${id}`, data, options)
  },

  /** Delete a payment channel */
  deleteChannel(id: number) {
    return apiClient.delete(`/admin/payment/channels/${id}`)
  },

  // ==================== Subscription Plans ====================

  /** Get all subscription plans */
  getPlans() {
    return apiClient.get<SubscriptionPlan[]>('/admin/payment/plans')
  },

  /** Create a subscription plan */
  createPlan(data: Record<string, unknown>) {
    return apiClient.post<SubscriptionPlan>('/admin/payment/plans', data)
  },

  /** Update a subscription plan */
  updatePlan(id: number, data: Record<string, unknown>) {
    return apiClient.put<SubscriptionPlan>(`/admin/payment/plans/${id}`, data)
  },

  /** Delete a subscription plan */
  deletePlan(id: number) {
    return apiClient.delete(`/admin/payment/plans/${id}`)
  },

  // ==================== Provider Instances ====================

  /** Get all provider instances */
  getProviders() {
    return apiClient.get<ProviderInstance[]>('/admin/payment/providers')
  },

  /** Create a provider instance */
  createProvider(data: Partial<ProviderInstance>, options?: FetchOptions) {
    return postData<ProviderInstance>('/admin/payment/providers', data, options)
  },

  /** Update a provider instance */
  updateProvider(id: number, data: Partial<ProviderInstance>, options?: FetchOptions) {
    return putData<ProviderInstance>(`/admin/payment/providers/${id}`, data, options)
  },

  /** Delete a provider instance */
  deleteProvider(id: number) {
    return apiClient.delete(`/admin/payment/providers/${id}`)
  }
}

export default adminPaymentAPI
