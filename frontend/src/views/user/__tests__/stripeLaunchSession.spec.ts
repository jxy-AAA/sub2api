import { beforeEach, describe, expect, it } from 'vitest'
import {
  STRIPE_LAUNCH_SESSION_MAX_AGE_MS,
  consumeStripeLaunchSession,
  createStripeLaunchSession,
} from '../stripeLaunchSession'

describe('stripeLaunchSession', () => {
  beforeEach(() => {
    window.sessionStorage.clear()
  })

  it('stores and consumes a secure launch session', () => {
    const sessionId = createStripeLaunchSession({
      orderId: 42,
      clientSecret: 'cs_test_42',
      method: 'alipay',
    }, {
      now: 1_000,
    })

    const raw = window.sessionStorage.getItem(`payment.stripe.launch.${sessionId}`)
    expect(raw).toContain('cs_test_42')

    const restored = consumeStripeLaunchSession(sessionId, {
      expectedOrderId: 42,
      expectedMethod: 'alipay',
      now: 2_000,
    })

    expect(restored).toEqual({
      orderId: 42,
      clientSecret: 'cs_test_42',
      method: 'alipay',
      createdAt: 1_000,
    })
    expect(window.sessionStorage.getItem(`payment.stripe.launch.${sessionId}`)).toBeNull()
  })

  it('rejects mismatched or expired launch sessions', () => {
    const mismatchedSessionId = createStripeLaunchSession({
      orderId: 7,
      clientSecret: 'cs_test_7',
      method: 'wechat_pay',
    }, {
      now: 10,
    })

    expect(consumeStripeLaunchSession(mismatchedSessionId, {
      expectedOrderId: 8,
      now: 100,
    })).toBeNull()

    const expiredSessionId = createStripeLaunchSession({
      orderId: 9,
      clientSecret: 'cs_test_9',
      method: '',
    }, {
      now: 10,
    })

    expect(consumeStripeLaunchSession(expiredSessionId, {
      now: STRIPE_LAUNCH_SESSION_MAX_AGE_MS + 11,
    })).toBeNull()
  })
})
