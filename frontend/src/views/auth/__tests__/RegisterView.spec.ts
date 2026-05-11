import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '@/views/auth/RegisterView.vue'

const {
  pushMock,
  registerMock,
  setPendingRegistrationChallengeMock,
  showSuccessMock,
  showErrorMock,
  showWarningMock,
  routeState,
  getPublicSettingsMock,
  validateAffiliateCodeMock,
} = vi.hoisted(() => ({
  pushMock: vi.fn(),
  registerMock: vi.fn(),
  setPendingRegistrationChallengeMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  showWarningMock: vi.fn(),
  routeState: {
    query: {
      aff: 'QUERY-CODE',
    } as Record<string, string>,
  },
  getPublicSettingsMock: vi.fn(),
  validateAffiliateCodeMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
  }),
  useRoute: () => routeState,
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key,
    },
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'auth.accountCreatedSuccess') {
        return `Account created for ${params?.siteName ?? 'Sub2API'}`
      }
      return key
    },
    locale: { value: 'en' },
  }),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register: (...args: any[]) => registerMock(...args),
    setPendingRegistrationChallenge: (...args: any[]) => setPendingRegistrationChallengeMock(...args),
  }),
  useAppStore: () => ({
    showSuccess: (...args: any[]) => showSuccessMock(...args),
    showError: (...args: any[]) => showErrorMock(...args),
    showWarning: (...args: any[]) => showWarningMock(...args),
  }),
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    getPublicSettings: (...args: any[]) => getPublicSettingsMock(...args),
    validateAffiliateCode: (...args: any[]) => validateAffiliateCodeMock(...args),
    validateInvitationCode: vi.fn().mockResolvedValue({ valid: true }),
  }
})

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function mountRegisterView() {
  return mount(RegisterView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        Icon: true,
        TurnstileWidget: true,
        LoginAgreementPrompt: true,
        EmailOAuthButtons: true,
        LinuxDoOAuthSection: true,
        OidcOAuthSection: true,
        WechatOAuthSection: true,
        'router-link': true,
        transition: false,
      },
    },
  })
}

describe('RegisterView', () => {
  beforeEach(() => {
    pushMock.mockReset()
    registerMock.mockReset()
    setPendingRegistrationChallengeMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    showWarningMock.mockReset()
    getPublicSettingsMock.mockReset()
    validateAffiliateCodeMock.mockReset()
    routeState.query = { aff: 'QUERY-CODE' }
    localStorage.clear()
    sessionStorage.clear()

    getPublicSettingsMock.mockResolvedValue({
      registration_enabled: true,
      email_verify_enabled: false,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: 'Sub2API',
      linuxdo_oauth_enabled: false,
      wechat_oauth_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      github_oauth_enabled: false,
      google_oauth_enabled: false,
      registration_email_suffix_whitelist: [],
      login_agreement_enabled: false,
      login_agreement_documents: [],
    })
    registerMock.mockResolvedValue({})
    validateAffiliateCodeMock.mockResolvedValue({ valid: true })
  })

  it('prefills the agent invitation field from referral code and submits aff_code only', async () => {
    const wrapper = mountRegisterView()

    await flushPromises()

    const affiliateInput = wrapper.get('#agent_invitation_code')
    expect((affiliateInput.element as HTMLInputElement).value).toBe('QUERY-CODE')

    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('secret-123')
    await affiliateInput.setValue('MANUAL-CODE')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret-123',
      turnstile_token: undefined,
      invitation_code: undefined,
      aff_code: 'MANUAL-CODE',
    })
    expect(registerMock.mock.calls[0][0]).not.toHaveProperty('promo_code')
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
  })

  it('stages an in-memory registration challenge instead of storing password in sessionStorage', async () => {
    getPublicSettingsMock.mockResolvedValue({
      registration_enabled: true,
      email_verify_enabled: true,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: 'Sub2API',
      linuxdo_oauth_enabled: false,
      wechat_oauth_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      github_oauth_enabled: false,
      google_oauth_enabled: false,
      registration_email_suffix_whitelist: [],
      login_agreement_enabled: false,
      login_agreement_documents: [],
    })

    const wrapper = mountRegisterView()

    await flushPromises()
    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('#agent_invitation_code').setValue('AFF-EMAIL')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(setPendingRegistrationChallengeMock).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret-123',
      turnstile_token: '',
      invitation_code: undefined,
      aff_code: 'AFF-EMAIL',
    })
    expect(sessionStorage.getItem('register_data')).toBeNull()
    expect(registerMock).not.toHaveBeenCalled()
    expect(pushMock).toHaveBeenCalledWith('/email-verify')
  })

  it('ignores stale affiliate validation responses when newer input has already been validated', async () => {
    vi.useFakeTimers()
    routeState.query = {}

    const staleValidation = createDeferred<{ valid: boolean; error_code?: string }>()
    const freshValidation = createDeferred<{ valid: boolean; error_code?: string }>()
    validateAffiliateCodeMock
      .mockImplementationOnce(() => staleValidation.promise)
      .mockImplementationOnce(() => freshValidation.promise)

    try {
      const wrapper = mountRegisterView()
      await flushPromises()

      await wrapper.get('#email').setValue('user@example.com')
      await wrapper.get('#password').setValue('secret-123')

      const affiliateInput = wrapper.get('#agent_invitation_code')
      await affiliateInput.setValue('STALE-CODE')
      await vi.advanceTimersByTimeAsync(500)

      await affiliateInput.setValue('FRESH-CODE')
      await vi.advanceTimersByTimeAsync(500)

      freshValidation.resolve({ valid: true })
      await flushPromises()

      staleValidation.resolve({ valid: false, error_code: 'AFFILIATE_CODE_INVALID' })
      await flushPromises()

      await wrapper.get('form').trigger('submit.prevent')
      await flushPromises()

      expect(validateAffiliateCodeMock).toHaveBeenCalledTimes(2)
      expect(registerMock).toHaveBeenCalledWith({
        email: 'user@example.com',
        password: 'secret-123',
        turnstile_token: undefined,
        invitation_code: undefined,
        aff_code: 'FRESH-CODE',
      })
      expect(showErrorMock).not.toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })
})

