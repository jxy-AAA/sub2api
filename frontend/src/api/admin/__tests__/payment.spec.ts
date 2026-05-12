import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
  },
}))

import { adminPaymentAPI } from '@/api/admin/payment'

describe('admin payment api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
  })

  it('unwraps config updates and forwards abort signal', async () => {
    const signal = new AbortController().signal
    const payload = { enabled: false, max_pending_orders: 10 }
    const updatedConfig = {
      enabled: false,
      min_amount: 1,
      max_amount: 100,
      daily_limit: 500,
      order_timeout_minutes: 30,
      max_pending_orders: 10,
      enabled_payment_types: ['stripe'],
      balance_disabled: false,
      balance_recharge_multiplier: 1,
      load_balance_strategy: 'round_robin',
      product_name_prefix: 'Sub',
      product_name_suffix: 'API',
      help_image_url: '',
      help_text: '',
    }
    put.mockResolvedValueOnce({ data: updatedConfig })

    const result = await adminPaymentAPI.updateConfig(payload, { signal })

    expect(put).toHaveBeenCalledWith('/admin/payment/config', payload, { signal })
    expect(result).toEqual(updatedConfig)
  })

  it('unwraps channel list and mutation responses', async () => {
    const signal = new AbortController().signal
    const channels = [
      {
        id: 3,
        name: 'Stripe',
        platform: 'stripe',
        rate_multiplier: 1.1,
        description: '',
        models: [],
        features: [],
        enabled: true,
      },
    ]
    const newChannel = channels[0]
    const updatedChannel = { ...channels[0], enabled: false }

    get.mockResolvedValueOnce({ data: channels })
    post.mockResolvedValueOnce({ data: newChannel })
    put.mockResolvedValueOnce({ data: updatedChannel })

    await expect(adminPaymentAPI.getChannels({ signal })).resolves.toEqual(channels)
    await expect(adminPaymentAPI.createChannel(newChannel, { signal })).resolves.toEqual(newChannel)
    await expect(adminPaymentAPI.updateChannel(3, { enabled: false }, { signal })).resolves.toEqual(updatedChannel)

    expect(get).toHaveBeenCalledWith('/admin/payment/channels', { signal })
    expect(post).toHaveBeenCalledWith('/admin/payment/channels', newChannel, { signal })
    expect(put).toHaveBeenCalledWith('/admin/payment/channels/3', { enabled: false }, { signal })
  })

  it('unwraps provider mutation responses', async () => {
    const signal = new AbortController().signal
    const newProvider = {
      id: 7,
      provider_key: 'stripe',
      name: 'Stripe CN',
      config: {},
      supported_types: ['stripe'],
      enabled: true,
      payment_mode: 'checkout',
      refund_enabled: true,
      allow_user_refund: true,
      limits: '',
      sort_order: 1,
    }
    const updatedProvider = { ...newProvider, enabled: false }

    post.mockResolvedValueOnce({ data: newProvider })
    put.mockResolvedValueOnce({ data: updatedProvider })

    await expect(adminPaymentAPI.createProvider(newProvider, { signal })).resolves.toEqual(newProvider)
    await expect(adminPaymentAPI.updateProvider(7, { enabled: false }, { signal })).resolves.toEqual(updatedProvider)

    expect(post).toHaveBeenCalledWith('/admin/payment/providers', newProvider, { signal })
    expect(put).toHaveBeenCalledWith('/admin/payment/providers/7', { enabled: false }, { signal })
  })
})
