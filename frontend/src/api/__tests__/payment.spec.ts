import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { paymentAPI } from '@/api/payment'

describe('payment api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({ data: {} })
    post.mockResolvedValue({ data: {} })
  })

  it('unwraps channels responses and forwards abort signal', async () => {
    const signal = new AbortController().signal
    const channels = [{ id: 1, name: 'Stripe', platform: 'stripe', rate_multiplier: 1, description: '', models: [], features: [], enabled: true }]
    get.mockResolvedValueOnce({ data: channels })

    const result = await paymentAPI.getChannels({ signal })

    expect(get).toHaveBeenCalledWith('/payment/channels', { signal })
    expect(result).toEqual(channels)
  })

  it('unwraps limits responses and forwards abort signal', async () => {
    const signal = new AbortController().signal
    const limits = {
      methods: {
        alipay: {
          daily_limit: 100,
          daily_used: 10,
          daily_remaining: 90,
          single_min: 1,
          single_max: 20,
          fee_rate: 0.02,
          available: true,
        },
      },
      global_min: 1,
      global_max: 20,
    }
    get.mockResolvedValueOnce({ data: limits })

    const result = await paymentAPI.getLimits({ signal })

    expect(get).toHaveBeenCalledWith('/payment/limits', { signal })
    expect(result).toEqual(limits)
  })

  it('unwraps internal verify responses and forwards abort signal', async () => {
    const signal = new AbortController().signal
    const order = { id: 9, out_trade_no: 'verify-9', status: 'PAID' }
    post.mockResolvedValueOnce({ data: order })

    const result = await paymentAPI.verifyOrder('verify-9', { signal })

    expect(post).toHaveBeenCalledWith(
      '/payment/orders/verify',
      { out_trade_no: 'verify-9' },
      { signal },
    )
    expect(result).toEqual(order)
  })

  it('keeps legacy public out_trade_no verification for upgrade compatibility', async () => {
    await paymentAPI.verifyOrderPublic('legacy-order-no', 'lookup-token-123')

    expect(post).toHaveBeenCalledWith('/payment/public/orders/verify', {
      out_trade_no: 'legacy-order-no',
      lookup_token: 'lookup-token-123',
    })
  })

  it('keeps signed public resume-token resolve endpoint', async () => {
    await paymentAPI.resolveOrderPublicByResumeToken('resume-token-123')

    expect(post).toHaveBeenCalledWith('/payment/public/orders/resolve', {
      resume_token: 'resume-token-123',
    })
  })
})
