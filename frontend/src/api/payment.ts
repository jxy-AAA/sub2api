/**
 * User Payment API endpoints
 * Handles payment operations for regular users
 */

import { apiClient } from './client'
import type {
  PaymentConfig,
  SubscriptionPlan,
  PaymentChannel,
  MethodLimitsResponse,
  CheckoutInfoResponse,
  CreateOrderRequest,
  CreateOrderResult,
  PaymentOrder
} from '@/types/payment'
import type { BasePaginationResponse, FetchOptions } from '@/types'

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

export const paymentAPI = {
  /** Get payment configuration (enabled types, limits, etc.) */
  getConfig() {
    return apiClient.get<PaymentConfig>('/payment/config')
  },

  /** Get available subscription plans */
  getPlans() {
    return apiClient.get<SubscriptionPlan[]>('/payment/plans')
  },

  /** Get available payment channels */
  getChannels(options?: FetchOptions) {
    return getData<PaymentChannel[]>('/payment/channels', options)
  },

  /** Get all checkout page data in a single call */
  getCheckoutInfo() {
    return apiClient.get<CheckoutInfoResponse>('/payment/checkout-info')
  },

  /** Get payment method limits and fee rates */
  getLimits(options?: FetchOptions) {
    return getData<MethodLimitsResponse>('/payment/limits', options)
  },

  /** Create a new payment order */
  createOrder(data: CreateOrderRequest) {
    return apiClient.post<CreateOrderResult>('/payment/orders', data)
  },

  /** Get current user's orders */
  getMyOrders(params?: { page?: number; page_size?: number; status?: string }) {
    return apiClient.get<BasePaginationResponse<PaymentOrder>>('/payment/orders/my', { params })
  },

  /** Get a specific order by ID */
  getOrder(id: number) {
    return apiClient.get<PaymentOrder>(`/payment/orders/${id}`)
  },

  /** Cancel a pending order */
  cancelOrder(id: number) {
    return apiClient.post(`/payment/orders/${id}/cancel`)
  },

  /** Verify order payment status with upstream provider */
  verifyOrder(outTradeNo: string, options?: FetchOptions) {
    return postData<PaymentOrder>('/payment/orders/verify', { out_trade_no: outTradeNo }, options)
  },

  /** Legacy-compatible public order lookup by out_trade_no */
  verifyOrderPublic(outTradeNo: string) {
    return apiClient.post<PaymentOrder>('/payment/public/orders/verify', { out_trade_no: outTradeNo })
  },

  /** Resolve an order from a signed resume token without auth */
  resolveOrderPublicByResumeToken(resumeToken: string) {
    return apiClient.post<PaymentOrder>('/payment/public/orders/resolve', { resume_token: resumeToken })
  },

  /** Request a refund for a completed order */
  requestRefund(id: number, data: { reason: string }) {
    return apiClient.post(`/payment/orders/${id}/refund-request`, data)
  },

  /** Get provider instance IDs that allow user refund */
  getRefundEligibleProviders() {
    return apiClient.get<{ provider_instance_ids: string[] }>('/payment/orders/refund-eligible-providers')
  }
}
