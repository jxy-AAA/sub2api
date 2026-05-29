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

  it('uses the referral link code without rendering an agent invitation field', async () => {
    const wrapper = mountRegisterView()

    await flushPromises()

    expect(wrapper.find('#agent_invitation_code').exists()).toBe(false)

    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret-123',
      turnstile_token: undefined,
      invitation_code: undefined,
      aff_code: 'QUERY-CODE',
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
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(setPendingRegistrationChallengeMock).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret-123',
      turnstile_token: '',
      invitation_code: undefined,
      aff_code: 'QUERY-CODE',
    })
    expect(sessionStorage.getItem('register_data')).toBeNull()
    expect(registerMock).not.toHaveBeenCalled()
    expect(pushMock).toHaveBeenCalledWith('/email-verify')
  })

  it('blocks registration when the hidden referral link code is invalid', async () => {
    validateAffiliateCodeMock.mockResolvedValue({ valid: false, error_code: 'AFFILIATE_CODE_INVALID' })

    const wrapper = mountRegisterView()
    await flushPromises()

    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(validateAffiliateCodeMock).toHaveBeenCalledWith('QUERY-CODE')
    expect(registerMock).not.toHaveBeenCalled()
  })
})

