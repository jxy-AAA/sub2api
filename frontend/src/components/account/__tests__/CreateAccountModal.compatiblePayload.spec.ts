import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const { createAccountMock } = vi.hoisted(() => ({
  createAccountMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: false
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: vi.fn()
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({})
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([])
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/composables/useQuotaNotifyState', () => ({
  useQuotaNotifyState: () => ({
    loadGlobalState: vi.fn(),
    writeToExtra: vi.fn()
  })
}))

const buildOAuthStub = () => ({
  authUrl: ref(''),
  authUrlReady: ref(false),
  loading: ref(false),
  error: ref(''),
  sessionId: ref(''),
  state: ref(''),
  oauthState: ref(''),
  resetState: vi.fn(),
  buildCredentials: vi.fn(() => ({})),
  buildExtraInfo: vi.fn(() => ({})),
  validateRefreshToken: vi.fn(),
  exchangeAuthCode: vi.fn(),
  parseSessionKeys: vi.fn(() => []),
  getCapabilities: vi.fn().mockResolvedValue({ ai_studio_oauth_enabled: false })
})

vi.mock('@/composables/useAccountOAuth', () => ({
  useAccountOAuth: () => buildOAuthStub()
}))

vi.mock('@/composables/useOpenAIOAuth', () => ({
  useOpenAIOAuth: () => buildOAuthStub()
}))

vi.mock('@/composables/useGeminiOAuth', () => ({
  useGeminiOAuth: () => buildOAuthStub()
}))

vi.mock('@/composables/useAntigravityOAuth', () => ({
  useAntigravityOAuth: () => buildOAuthStub()
}))

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const GroupSelectorStub = defineComponent({
  name: 'GroupSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <div>
      <button
        type="button"
        data-testid="select-groups"
        @click="$emit('update:modelValue', [101, 202])"
      >
        select groups
      </button>
    </div>
  `
})

function mountModal() {
  return mount(CreateAccountModal, {
    props: {
      show: true,
      proxies: [],
      groups: [
        { id: 101, name: 'Compatibility A', platform: 'openai_compatible' },
        { id: 202, name: 'Compatibility B', platform: 'openai_compatible' }
      ]
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        Select: true,
        Icon: true,
        ProxySelector: true,
        GroupSelector: GroupSelectorStub,
        ModelWhitelistSelector: true,
        QuotaLimitCard: true,
        OAuthAuthorizationFlow: true
      }
    }
  })
}

function findButtonByText(wrapper: ReturnType<typeof mountModal>, text: string) {
  return wrapper.findAll('button').find((button) => button.text().includes(text))
}

describe('CreateAccountModal compatible upstream payload contract', () => {
  it('opens OpenAI and Anthropic compatible entries without entering OAuth flow', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const openAICompatibleButton = findButtonByText(wrapper, 'OpenRouter')
    expect(openAICompatibleButton).toBeTruthy()
    await openAICompatibleButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('form#create-account-form').exists()).toBe(true)
    expect(wrapper.find('oauth-authorization-flow-stub').exists()).toBe(false)
    expect(wrapper.find('input[placeholder="https://api.example.com/v1"]').exists()).toBe(true)

    const anthropicCompatibleButton = findButtonByText(wrapper, 'Anthropic-compatible')
    expect(anthropicCompatibleButton).toBeTruthy()
    await anthropicCompatibleButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('form#create-account-form').exists()).toBe(true)
    expect(wrapper.find('oauth-authorization-flow-stub').exists()).toBe(false)
    expect(wrapper.find('input[placeholder="https://api.example.com"]').exists()).toBe(true)
  })

  it('submits compatible upstream headers and selected group ids through component behavior', async () => {
    createAccountMock.mockReset()
    createAccountMock.mockResolvedValue({})

    const wrapper = mountModal()
    await flushPromises()

    const compatibleButton = findButtonByText(wrapper, 'OpenRouter')

    expect(compatibleButton).toBeTruthy()
    await compatibleButton!.trigger('click')
    await flushPromises()

    await wrapper.get('input[placeholder="admin.accounts.enterAccountName"]').setValue('Compatible upstream')
    await wrapper.get('[data-testid="select-groups"]').trigger('click')
    await wrapper.get('input[placeholder="https://api.example.com/v1"]').setValue('https://compat.example.com/v1')
    await wrapper.get('input[placeholder="sk-..."]').setValue('sk-compatible-new')
    await wrapper
      .get('textarea[placeholder="admin.accounts.upstream.headersPlaceholder"]')
      .setValue('{\n  "x-api-version": "2024-01-01",\n  "x-provider": "openrouter"\n}')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]).toMatchObject({
      name: 'Compatible upstream',
      platform: 'openai_compatible',
      type: 'upstream',
      group_ids: [101, 202],
      credentials: {
        base_url: 'https://compat.example.com/v1',
        api_key: 'sk-compatible-new',
        headers: {
          'x-api-version': '2024-01-01',
          'x-provider': 'openrouter'
        }
      }
    })
  })

  it('submits Anthropic-compatible upstream payload through component behavior', async () => {
    createAccountMock.mockReset()
    createAccountMock.mockResolvedValue({})

    const wrapper = mountModal()
    await flushPromises()

    const compatibleButton = findButtonByText(wrapper, 'Anthropic-compatible')
    expect(compatibleButton).toBeTruthy()
    await compatibleButton!.trigger('click')
    await flushPromises()

    await wrapper.get('input[placeholder="admin.accounts.enterAccountName"]').setValue('Anthropic compatible upstream')
    await wrapper.get('input[placeholder="https://api.example.com"]').setValue('https://claude.compat.example.com')
    await wrapper.get('input[placeholder="sk-ant-..."]').setValue('sk-ant-compatible-new')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]).toMatchObject({
      name: 'Anthropic compatible upstream',
      platform: 'anthropic_compatible',
      type: 'upstream',
      credentials: {
        base_url: 'https://claude.compat.example.com',
        api_key: 'sk-ant-compatible-new'
      }
    })
  })
})
