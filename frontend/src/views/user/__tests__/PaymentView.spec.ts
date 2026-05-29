import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import PaymentView from '../PaymentView.vue'
import { PAYMENT_RECOVERY_STORAGE_KEY } from '@/components/payment/paymentFlow'

const routeState = vi.hoisted(() => ({
  path: '/purchase',
  query: {} as Record<string, unknown>,
}))

const routerReplace = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())
const routerResolve = vi.hoisted(() => vi.fn(() => ({ href: '/payment/stripe?mock=1' })))
const createOrder = vi.hoisted(() => vi.fn())
const refreshUser = vi.hoisted(() => vi.fn())
const fetchActiveSubscriptions = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
const showError = vi.hoisted(() => vi.fn())
const showInfo = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())
const getCheckoutInfo = vi.hoisted(() => vi.fn())
const bridgeInvoke = vi.hoisted(() => vi.fn())
const isMobileDeviceMock = vi.hoisted(() => vi.fn(() => true))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      replace: routerReplace,
      push: routerPush,
      resolve: routerResolve,
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      username: 'demo-user',
      balance: 0,
    },
    refreshUser,
  }),
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({
    createOrder,
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    activeSubscriptions: [],
    fetchActiveSubscriptions,
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showInfo,
    showWarning,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo,
  },
}))

vi.mock('@/utils/device', () => ({
  isMobileDevice: isMobileDeviceMock,
}))

function checkoutInfoFixture() {
  return {
    data: {
      methods: {
        wxpay: {
          daily_limit: 0,
          daily_used: 0,
          daily_remaining: 0,
          single_min: 0,
          single_max: 0,
          fee_rate: 0,
          available: true,
        },
      },
      global_min: 0,
      global_max: 0,
      plans: [],
      balance_disabled: false,
      balance_recharge_multiplier: 1,
      recharge_fee_rate: 0,
      help_text: '',
      help_image_url: '',
      stripe_publishable_key: '',
    },
  }
}

function checkoutInfoWithPlansFixture() {
  return {
    data: {
      ...checkoutInfoFixture().data,
      plans: [
        {
          id: 7,
          group_id: 3,
          name: 'Starter',
          description: '',
          price: 128,
          original_price: 0,
          validity_days: 30,
          validity_unit: 'day',
          rate_multiplier: 1,
          daily_limit_usd: null,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
          features: [],
          group_platform: 'openai',
          sort_order: 1,
          for_sale: true,
          group_name: 'OpenAI',
        },
      ],
    },
  }
}

function jsapiOrderFixture(resumeToken: string) {
  return {
    order_id: 123,
    amount: 88,
    pay_amount: 88,
    fee_rate: 0,
    expires_at: '2099-01-01T00:10:00.000Z',
    payment_type: 'wxpay',
    out_trade_no: 'sub2_jsapi_123',
    result_type: 'jsapi_ready' as const,
    resume_token: resumeToken,
    jsapi: {
      appId: 'wx123',
      timeStamp: '1712345678',
      nonceStr: 'nonce',
      package: 'prepay_id=wx123',
      signType: 'RSA',
      paySign: 'signed',
    },
  }
}

function oauthOrderFixture() {
  return {
    order_id: 456,
    amount: 128,
    pay_amount: 128,
    fee_rate: 0,
    expires_at: '2099-01-01T00:10:00.000Z',
    payment_type: 'wxpay',
    result_type: 'oauth_required' as const,
    oauth: {
      authorize_url: '/api/v1/auth/oauth/wechat/payment/start?payment_type=wxpay&redirect=%2Fpurchase%3Ffrom%3Dwechat',
      appid: 'wx123',
      scope: 'snsapi_base',
      redirect_url: '/auth/wechat/payment/callback',
    },
  }
}

describe('PaymentView WeChat JSAPI flow', () => {
  beforeEach(() => {
    routeState.path = '/purchase'
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
    }
    routerReplace.mockReset().mockResolvedValue(undefined)
    routerPush.mockReset().mockResolvedValue(undefined)
    routerResolve.mockClear()
    createOrder.mockReset()
    refreshUser.mockReset()
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    showError.mockReset()
    showInfo.mockReset()
    showWarning.mockReset()
    getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoFixture())
    bridgeInvoke.mockReset()
    isMobileDeviceMock.mockReset()
    isMobileDeviceMock.mockReturnValue(true)
    window.localStorage.clear()
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = {
      invoke: bridgeInvoke,
    }
  })

  it('resets payment state and redirects to /payment/result after JSAPI reports success', async () => {
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-123'))
    bridgeInvoke.mockImplementation((_action, _payload, callback) => {
      callback({ err_msg: 'get_brand_wcpay_request:ok' })
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({ path: '/purchase', query: {} })
    expect(routerPush).toHaveBeenCalledWith({
      path: '/payment/result',
      query: {
        order_id: '123',
        out_trade_no: 'sub2_jsapi_123',
      },
    })
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('resets payment state when JSAPI reports cancellation', async () => {
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-cancel'))
    bridgeInvoke.mockImplementation((_action, _payload, callback) => {
      callback({ err_msg: 'get_brand_wcpay_request:cancel' })
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(showInfo).toHaveBeenCalledWith('payment.qr.cancelled')
    expect(routerPush).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('clears stale recovery state when JSAPI never becomes available', async () => {
    vi.useFakeTimers()
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-missing-bridge'))
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })

    await flushPromises()
    await vi.advanceTimersByTimeAsync(4000)
    await flushPromises()
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(
      'payment.errors.wechatJsapiUnavailable payment.errors.wechatOpenInWeChatHint',
    )
    expect(routerPush).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
    expect(wrapper.html()).not.toContain('payment-status-panel-stub')
  })

  it('clears a stale recovery snapshot before handling wechat resume callback params', async () => {
    createOrder.mockRejectedValueOnce(new Error('resume failed'))
    window.localStorage.setItem(PAYMENT_RECOVERY_STORAGE_KEY, JSON.stringify({
      orderId: 999,
      amount: 66,
      qrCode: 'stale-qr',
      expiresAt: '2099-01-01T00:10:00.000Z',
      paymentType: 'alipay',
      payUrl: 'https://pay.example.com/stale',
      outTradeNo: 'stale-out-trade-no',
      clientSecret: '',
      payAmount: 66,
      orderType: 'balance',
      paymentMode: 'popup',
      resumeToken: '',
      createdAt: Date.UTC(2099, 0, 1, 0, 0, 0),
    }))

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      wechat_resume_token: 'resume-token-123',
    }))
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('keeps subscription resume context for token-only WeChat callbacks', async () => {
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-subscription-7',
      payment_type: 'wxpay_direct',
      order_type: 'subscription',
      plan_id: '7',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())
    createOrder.mockResolvedValue(oauthOrderFixture())

    const originalLocation = window.location
    const locationState = {
      href: 'http://localhost/purchase',
      origin: 'http://localhost',
    }
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState,
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({ path: '/purchase', query: {} })
    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      payment_type: 'wxpay',
      order_type: 'subscription',
      plan_id: 7,
      wechat_resume_token: 'resume-subscription-7',
    }))
    expect(locationState.href).toContain('/api/v1/auth/oauth/wechat/payment/start?')
    expect(new URL(locationState.href, 'http://localhost').searchParams.get('redirect')).toBe(
      '/purchase?from=wechat&payment_type=wxpay&order_type=subscription&plan_id=7',
    )

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
  })

  it('falls back to QR flow when mobile WeChat payment is unavailable', async () => {
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-h5',
      payment_type: 'wxpay_direct',
    }
    createOrder
      .mockRejectedValueOnce({ reason: 'WECHAT_H5_NOT_AUTHORIZED' })
      .mockResolvedValueOnce({
        order_id: 778,
        amount: 88,
        pay_amount: 88,
        fee_rate: 0,
        expires_at: '2099-01-01T00:10:00.000Z',
        payment_type: 'wxpay',
        qr_code: 'weixin://wxpay/bizpayurl?pr=fallback-native',
        out_trade_no: 'sub2_qr_778',
      })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).toHaveBeenNthCalledWith(1, expect.objectContaining({
      payment_type: 'wxpay',
      is_mobile: true,
      wechat_resume_token: 'resume-token-h5',
    }))
    expect(createOrder).toHaveBeenNthCalledWith(2, expect.objectContaining({
      payment_type: 'wxpay',
      is_mobile: false,
      payment_source: 'hosted_redirect',
    }))
    expect(showWarning).toHaveBeenCalledWith('payment.errors.mobilePaymentFallbackToQr')
    expect(showError).not.toHaveBeenCalled()
    const recoveryRaw = window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY) || ''
    expect(recoveryRaw).toContain('"payUrl":""')
    expect(recoveryRaw).toContain('"clientSecret":""')
    expect(recoveryRaw).not.toContain('resume-token-h5')
  })

  it('routes dedicated Stripe payments without leaking client_secret or resume_token into the URL', async () => {
    routeState.query = {}
    getCheckoutInfo.mockResolvedValue({
      data: {
        methods: {
          stripe: {
            daily_limit: 0,
            daily_used: 0,
            daily_remaining: 0,
            single_min: 0,
            single_max: 0,
            fee_rate: 0,
            available: true,
          },
        },
        global_min: 0,
        global_max: 0,
        plans: [],
        balance_disabled: false,
        balance_recharge_multiplier: 1,
        recharge_fee_rate: 0,
        help_text: '',
        help_image_url: '',
        stripe_publishable_key: 'pk_test_123',
      },
    })
    createOrder.mockResolvedValue({
      order_id: 901,
      amount: 25,
      pay_amount: 25,
      fee_rate: 0,
      expires_at: '2099-01-01T00:10:00.000Z',
      payment_type: 'stripe',
      client_secret: 'cs_test_secret_901',
      resume_token: 'resume-token-901',
    })

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    ;(wrapper.vm as unknown as { amount: number }).amount = 25
    await flushPromises()
    await (wrapper.vm as unknown as { handleSubmitRecharge: () => Promise<void> }).handleSubmitRecharge()
    await flushPromises()

    const stripeRouteLocation = routerResolve.mock.calls.find(([arg]) => arg?.path === '/payment/stripe')?.[0]
    expect(stripeRouteLocation.path).toBe('/payment/stripe')
    expect(stripeRouteLocation.query.order_id).toBe('901')
    expect(stripeRouteLocation.query.session_id).toEqual(expect.any(String))
    expect(stripeRouteLocation.query.client_secret).toBeUndefined()
    expect(stripeRouteLocation.query.resume_token).toBeUndefined()
    expect(routerPush).toHaveBeenCalledWith('/payment/stripe?mock=1')

    const recoveryRaw = window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY) || ''
    expect(recoveryRaw).not.toContain('cs_test_secret_901')
    expect(recoveryRaw).not.toContain('resume-token-901')
  })

  it('opens the Stripe popup with postMessage init and keeps secrets out of popup URLs', async () => {
    isMobileDeviceMock.mockReturnValue(false)
    routeState.query = {}
    getCheckoutInfo.mockResolvedValue({
      data: {
        methods: {
          alipay: {
            daily_limit: 0,
            daily_used: 0,
            daily_remaining: 0,
            single_min: 0,
            single_max: 0,
            fee_rate: 0,
            available: true,
          },
        },
        global_min: 0,
        global_max: 0,
        plans: [],
        balance_disabled: false,
        balance_recharge_multiplier: 1,
        recharge_fee_rate: 0,
        help_text: '',
        help_image_url: '',
        stripe_publishable_key: 'pk_test_popup',
      },
    })
    createOrder.mockResolvedValue({
      order_id: 902,
      amount: 30,
      pay_amount: 30,
      fee_rate: 0,
      expires_at: '2099-01-01T00:10:00.000Z',
      payment_type: 'alipay',
      client_secret: 'cs_test_secret_902',
      resume_token: 'resume-token-902',
    })
    routerResolve
      .mockReset()
      .mockReturnValueOnce({ href: '/payment/stripe-popup?opaque=1' })
      .mockReturnValueOnce({ href: '/payment/stripe?opaque=2' })

    const popupPostMessage = vi.fn()
    const popupClose = vi.fn()
    const fakePopup = {
      closed: false,
      postMessage: popupPostMessage,
      close: popupClose,
    }
    const windowOpen = vi.spyOn(window, 'open').mockReturnValue(fakePopup as unknown as Window)

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    ;(wrapper.vm as unknown as { amount: number }).amount = 30
    await flushPromises()
    await (wrapper.vm as unknown as { handleSubmitRecharge: () => Promise<void> }).handleSubmitRecharge()
    await flushPromises()

    const popupLocation = routerResolve.mock.calls[0]?.[0]
    const fallbackLocation = routerResolve.mock.calls[1]?.[0]
    expect(popupLocation.path).toBe('/payment/stripe-popup')
    expect(popupLocation.query.client_secret).toBeUndefined()
    expect(popupLocation.query.resume_token).toBeUndefined()
    expect(fallbackLocation.path).toBe('/payment/stripe')
    expect(fallbackLocation.query.client_secret).toBeUndefined()
    expect(fallbackLocation.query.resume_token).toBeUndefined()

    window.dispatchEvent(new MessageEvent('message', {
      origin: window.location.origin,
      data: { type: 'STRIPE_POPUP_READY' },
      source: fakePopup as unknown as MessageEventSource,
    }))
    await flushPromises()

    expect(windowOpen).toHaveBeenCalledWith('/payment/stripe-popup?opaque=1', 'paymentPopup', expect.any(String))
    expect(popupPostMessage).toHaveBeenCalledWith({
      type: 'STRIPE_POPUP_INIT',
      clientSecret: 'cs_test_secret_902',
      publishableKey: 'pk_test_popup',
      resumeToken: 'resume-token-902',
    }, window.location.origin)
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY) || '').not.toContain('cs_test_secret_902')
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY) || '').not.toContain('resume-token-902')

    windowOpen.mockRestore()
  })
})
