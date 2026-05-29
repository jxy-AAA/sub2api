import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import StripePaymentView from '../StripePaymentView.vue'
import { createStripeLaunchSession } from '../stripeLaunchSession'

const routeState = vi.hoisted(() => ({
  path: '/payment/stripe',
  query: {} as Record<string, unknown>,
}))

const routerPush = vi.hoisted(() => vi.fn())
const routerReplace = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
const fetchConfig = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
const pollOrderStatus = vi.hoisted(() => vi.fn())
const getOrder = vi.hoisted(() => vi.fn())
const loadStripe = vi.hoisted(() => vi.fn())
const confirmAlipayPayment = vi.hoisted(() => vi.fn().mockResolvedValue({}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      push: routerPush,
      replace: routerReplace,
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

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({
    config: {
      stripe_publishable_key: 'pk_test_route',
    },
    fetchConfig,
    pollOrderStatus,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getOrder,
  },
}))

vi.mock('@/utils/device', () => ({
  isMobileDevice: () => false,
}))

vi.mock('@stripe/stripe-js', () => ({
  loadStripe,
}))

describe('StripePaymentView', () => {
  beforeEach(() => {
    routeState.query = {}
    routerPush.mockReset()
    routerReplace.mockReset().mockResolvedValue(undefined)
    fetchConfig.mockReset().mockResolvedValue(undefined)
    pollOrderStatus.mockReset()
    getOrder.mockReset().mockResolvedValue({
      data: {
        id: 42,
        user_id: 7,
        amount: 30,
        pay_amount: 30,
        fee_rate: 0,
        payment_type: 'alipay',
        out_trade_no: 'sub2_42',
        status: 'PENDING',
        order_type: 'balance',
        created_at: '2099-01-01T00:00:00.000Z',
        expires_at: '2099-01-01T00:10:00.000Z',
        refund_amount: 0,
      },
    })
    confirmAlipayPayment.mockReset().mockResolvedValue({})
    loadStripe.mockReset().mockResolvedValue({
      confirmAlipayPayment,
    })
    window.sessionStorage.clear()
  })

  it('consumes secure launch sessions and strips the opaque session_id from the URL', async () => {
    const sessionId = createStripeLaunchSession({
      orderId: 42,
      clientSecret: 'cs_test_secure_42',
      method: 'alipay',
    })
    routeState.query = {
      order_id: '42',
      method: 'alipay',
      session_id: sessionId,
    }

    shallowMount(StripePaymentView, {
      global: {
        stubs: {
          AppLayout: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({
      path: '/payment/stripe',
      query: {
        order_id: '42',
        method: 'alipay',
      },
    })
    expect(confirmAlipayPayment).toHaveBeenCalledWith(
      'cs_test_secure_42',
      expect.objectContaining({
        return_url: expect.stringContaining('/payment/result?order_id=42&status=success'),
      }),
    )
    expect(window.sessionStorage.getItem(`payment.stripe.launch.${sessionId}`)).toBeNull()
  })

  it('fails closed for legacy client_secret deep links and never consumes URL secrets', async () => {
    routeState.query = {
      order_id: '42',
      method: 'alipay',
      client_secret: 'cs_leaked_legacy_secret',
    }

    const wrapper = shallowMount(StripePaymentView, {
      global: {
        stubs: {
          AppLayout: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({
      path: '/payment/stripe',
      query: {
        order_id: '42',
        method: 'alipay',
      },
    })
    expect(confirmAlipayPayment).not.toHaveBeenCalled()
    expect(getOrder).not.toHaveBeenCalled()
    expect(wrapper.html()).toContain('payment.stripeMissingParams')
  })
})
