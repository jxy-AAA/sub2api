<template>
  <div class="flex min-h-screen items-center justify-center bg-slate-50 p-4 dark:bg-slate-950">
    <div
      class="w-full max-w-md space-y-4 rounded-2xl border border-slate-200 bg-white p-6 shadow-lg dark:border-slate-700 dark:bg-slate-900"
    >
      <!-- Amount + Order ID -->
      <div v-if="amount" class="text-center">
        <p class="text-3xl font-bold" :style="{ color: methodColor }">¥{{ amount }}</p>
        <p v-if="orderId" class="mt-1 text-sm text-gray-500 dark:text-slate-400">
          {{ t('payment.orders.orderId') }}: {{ orderId }}
        </p>
      </div>

      <!-- Error -->
      <div v-if="error" class="space-y-3">
        <div
          class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-700 dark:bg-red-900/30 dark:text-red-400"
        >
          {{ error }}
        </div>
        <button
          class="w-full text-sm underline dark:text-blue-400 dark:hover:text-blue-300"
          :style="{ color: methodColor }"
          @click="closeWindow"
        >
          {{ t('common.close') }}
        </button>
      </div>

      <!-- Success -->
      <div v-else-if="success" class="space-y-3 py-4 text-center">
        <div class="text-5xl text-green-600 dark:text-green-400">✓</div>
        <p class="text-sm text-gray-500 dark:text-slate-400">{{ t('payment.result.success') }}</p>
        <button
          class="text-sm underline dark:text-blue-400 dark:hover:text-blue-300"
          :style="{ color: methodColor }"
          @click="closeWindow"
        >
          {{ t('common.close') }}
        </button>
      </div>

      <!-- Loading / Redirecting -->
      <div v-else class="flex items-center justify-center py-8">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-t-transparent"
          :style="{ borderColor: methodColor, borderTopColor: 'transparent' }"
        />
        <span class="ml-3 text-sm text-gray-500 dark:text-slate-400">{{ hint }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { isMobileDevice } from '@/utils/device'
import { consumeStripeLaunchSession } from './stripeLaunchSession'

interface StripeWithWechatPay {
  confirmWechatPayPayment(clientSecret: string, options: Record<string, unknown>): Promise<{ error?: { message?: string }; paymentIntent?: { status: string } }>
}

const METHOD_COLORS: Record<string, string> = {
  alipay: '#00AEEF',
  wechat_pay: '#07C160',
}
const DEFAULT_METHOD_COLOR = '#635bff'

const { t } = useI18n()
const route = useRoute()

const orderId = String(route.query.order_id || '')
const method = String(route.query.method || 'alipay')
const amount = String(route.query.amount || '')
const sessionId = String(route.query.session_id || '')

const methodColor = computed(() => METHOD_COLORS[method] || DEFAULT_METHOD_COLOR)

const error = ref('')
const success = ref(false)
const hint = ref(t('payment.stripePopup.redirecting'))
const resumeToken = ref('')

let pollTimer: ReturnType<typeof setInterval> | null = null
let initStarted = false
let sessionBootstrapTimer: ReturnType<typeof setTimeout> | null = null

function closeWindow() { window.close() }

onMounted(() => {
  const handler = (event: MessageEvent) => {
    if (event.origin !== window.location.origin) return
    if (window.opener && event.source !== window.opener) return
    if (event.data?.type !== 'STRIPE_POPUP_INIT') return
    window.removeEventListener('message', handler)
    if (sessionBootstrapTimer) {
      clearTimeout(sessionBootstrapTimer)
      sessionBootstrapTimer = null
    }
    initStripe(event.data.clientSecret, event.data.publishableKey, event.data.resumeToken)
  }
  window.addEventListener('message', handler)

  if (sessionId) {
    sessionBootstrapTimer = setTimeout(() => {
      if (initStarted) return
      const launchSession = consumeStripeLaunchSession(sessionId, {
        expectedOrderId: Number(orderId) || undefined,
        expectedMethod: method || undefined,
      })
      if (launchSession) {
        void initStripe(launchSession.clientSecret, '', '')
      }
    }, 500)
  }

  if (window.opener) {
    window.opener.postMessage({ type: 'STRIPE_POPUP_READY' }, window.location.origin)
  }

  setTimeout(() => {
    if (!error.value && !success.value) {
      error.value = t('payment.stripePopup.timeout')
    }
  }, 15000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (sessionBootstrapTimer) clearTimeout(sessionBootstrapTimer)
})

async function initStripe(clientSecret: string, publishableKey: string, nextResumeToken?: string) {
  if (initStarted) {
    return
  }
  initStarted = true
  const normalizedClientSecret = String(clientSecret || '').trim()
  let resolvedPublishableKey = String(publishableKey || '').trim()
  if (!resolvedPublishableKey) {
    try {
      const config = await paymentAPI.getConfig()
      resolvedPublishableKey = String(config.data?.stripe_publishable_key || '').trim()
    } catch {
      resolvedPublishableKey = ''
    }
  }

  if (!normalizedClientSecret || !resolvedPublishableKey) {
    error.value = t('payment.stripeMissingParams')
    return
  }
  resumeToken.value = typeof nextResumeToken === 'string' ? nextResumeToken : ''
  try {
    const { loadStripe } = await import('@stripe/stripe-js')
    const stripe = await loadStripe(resolvedPublishableKey)
    if (!stripe) { error.value = t('payment.stripeLoadFailed'); return }

    const returnUrl = window.location.origin + '/payment/result?order_id=' + orderId + '&status=success'

    if (method === 'alipay') {
      // Alipay: redirect this popup to Alipay payment page
      const { error: err } = await stripe.confirmAlipayPayment(normalizedClientSecret, { return_url: returnUrl })
      if (err) error.value = err.message || t('payment.result.failed')
    } else if (method === 'wechat_pay') {
      // WeChat: Stripe shows its built-in QR dialog, user scans, promise resolves
      hint.value = t('payment.stripePopup.loadingQr')
      const result = await (stripe as unknown as StripeWithWechatPay).confirmWechatPayPayment(normalizedClientSecret, {
        payment_method_options: { wechat_pay: { client: isMobileDevice() ? 'mobile_web' : 'web' } },
      })
      if (result.error) {
        error.value = result.error.message || t('payment.result.failed')
      } else if (result.paymentIntent?.status === 'succeeded') {
        success.value = true
        setTimeout(closeWindow, 2000)
      } else {
        // Payment not completed (user closed QR dialog)
        startPolling()
      }
    }
  } catch (err: unknown) {
    error.value = extractI18nErrorMessage(err, t, 'payment.errors', t('payment.stripeLoadFailed'))
  }
}

function startPolling() {
  pollTimer = setInterval(async () => {
    try {
      const response = resumeToken.value
        ? await paymentAPI.resolveOrderPublicByResumeToken(resumeToken.value)
        : await paymentAPI.getOrder(Number(orderId))
      const status = response.data?.status
      if (status === 'COMPLETED' || status === 'PAID') {
        if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
        success.value = true
        setTimeout(closeWindow, 2000)
      }
    } catch { /* ignore */ }
  }, 3000)
}
</script>
