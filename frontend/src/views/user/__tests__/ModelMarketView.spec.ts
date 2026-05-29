import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getModels: vi.fn(),
}))

const appStoreMock = vi.hoisted(() => ({
  showError: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

vi.mock('@/api/modelMarket', () => ({
  __esModule: true,
  default: {
    getModels: apiMocks.getModels,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

import ModelMarketView from '../ModelMarketView.vue'

const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="table" /></div>',
}

const EmptyStateStub = {
  props: ['title', 'description', 'actionText'],
  template: `
    <div>
      <div class="empty-title">{{ title }}</div>
      <div class="empty-description">{{ description }}</div>
      <button type="button" @click="$emit('action')">{{ actionText }}</button>
    </div>
  `,
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select />',
}

describe('User ModelMarketView', () => {
  beforeEach(() => {
    apiMocks.getModels.mockReset()
    appStoreMock.showError.mockReset()
  })

  it('loads and renders available models on mount', async () => {
    apiMocks.getModels.mockResolvedValue([
      {
        id: 1,
        model_id: 'gpt-5.4',
        display_name: 'GPT-5.4',
        provider_key: 'openai_compatible',
        platform: 'openai_compatible',
        protocol: 'openai',
        capabilities: ['chat', 'tools'],
        tags: ['recommended'],
        context_window: 128000,
        description: 'Flagship reasoning model',
        status: 'active',
        recommended: true,
        available: true,
        channels: [
          {
            name: 'Cursor Pool',
            provider_key: 'openai_compatible',
            platform: 'openai_compatible',
            groups: [],
            pricing: null,
          },
        ],
        price_summary: null,
      },
    ])

    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: TablePageLayoutStub,
          SearchInput: { template: '<input />' },
          Select: SelectStub,
          EmptyState: EmptyStateStub,
          LoadingSpinner: true,
          PlatformIcon: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(apiMocks.getModels).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('GPT-5.4')
    expect(wrapper.text()).toContain('Cursor Pool')
    expect(wrapper.text()).toContain('Flagship reasoning model')
  })

  it('shows load errors and retries from the empty state action', async () => {
    apiMocks.getModels
      .mockRejectedValueOnce(new Error('backend unavailable'))
      .mockResolvedValueOnce([
        {
          id: 2,
          model_id: 'claude-sonnet-4',
          display_name: 'Claude Sonnet 4',
          provider_key: 'anthropic_compatible',
          platform: 'anthropic_compatible',
          protocol: 'anthropic',
          capabilities: ['chat'],
          tags: [],
          channels: [],
          price_summary: null,
        },
      ])

    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: TablePageLayoutStub,
          SearchInput: { template: '<input />' },
          Select: SelectStub,
          EmptyState: EmptyStateStub,
          LoadingSpinner: true,
          PlatformIcon: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(appStoreMock.showError).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('backend unavailable')

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(apiMocks.getModels).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Claude Sonnet 4')
  })
})
