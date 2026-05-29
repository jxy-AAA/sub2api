import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, beforeEach, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getAllGroups: vi.fn(),
  listModels: vi.fn(),
  importFromChannels: vi.fn(),
  createModel: vi.fn(),
  updateModel: vi.fn(),
  removeModel: vi.fn(),
}))

const appStoreMock = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      getAll: apiMocks.getAllGroups,
    },
    modelMarket: {
      list: apiMocks.listModels,
      importFromChannels: apiMocks.importFromChannels,
      create: apiMocks.createModel,
      update: apiMocks.updateModel,
      remove: apiMocks.removeModel,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

import ModelMarketView from '../ModelMarketView.vue'

const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="table" /></div>',
}

const DataTableStub = {
  props: ['data', 'loading'],
  template: `
    <div>
      <div data-testid="table-loading">{{ loading ? 'loading' : 'idle' }}</div>
      <div v-for="row in data" :key="row.id" class="row">
        <slot name="cell-display_name" :row="row">{{ row.display_name || row.model_id }}</slot>
        <slot name="cell-provider_key" :row="row" :value="row.provider_key">{{ row.provider_key }}</slot>
      </div>
      <slot v-if="!loading && (!data || !data.length)" name="empty" />
    </div>
  `,
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select />',
}

describe('Admin ModelMarketView', () => {
  beforeEach(() => {
    apiMocks.getAllGroups.mockReset()
    apiMocks.listModels.mockReset()
    apiMocks.importFromChannels.mockReset()
    apiMocks.createModel.mockReset()
    apiMocks.updateModel.mockReset()
    apiMocks.removeModel.mockReset()
    appStoreMock.showError.mockReset()
    appStoreMock.showSuccess.mockReset()
  })

  it('loads groups and models on mount and refreshes after import', async () => {
    apiMocks.getAllGroups.mockResolvedValue([{ id: 1, name: 'Ops', platform: 'openai' }])
    apiMocks.listModels
      .mockResolvedValueOnce([
        {
          id: 11,
          model_id: 'gpt-5.4',
          display_name: 'GPT-5.4',
          provider_key: 'openai_compatible',
          protocol: 'openai',
          capabilities: ['chat'],
          tags: ['recommended'],
          status: 'active',
          sort_order: 1,
          channel_refs: [],
          available_channel_count: 0,
        },
      ])
      .mockResolvedValueOnce([
        {
          id: 11,
          model_id: 'gpt-5.4',
          display_name: 'GPT-5.4',
          provider_key: 'openai_compatible',
          protocol: 'openai',
          capabilities: ['chat'],
          tags: ['recommended'],
          status: 'active',
          sort_order: 1,
          channel_refs: [],
          available_channel_count: 0,
        },
      ])
    apiMocks.importFromChannels.mockResolvedValue({ message: 'imported ok' })

    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          SearchInput: { template: '<input />' },
          Select: SelectStub,
          PlatformIcon: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(apiMocks.getAllGroups).toHaveBeenCalledTimes(1)
    expect(apiMocks.listModels).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('GPT-5.4')

    const importButton = wrapper
      .findAll('button')
      .find((candidate) => candidate.text().includes('admin.modelMarket.importFromChannels'))
    expect(importButton).toBeDefined()

    await importButton!.trigger('click')
    await flushPromises()

    expect(apiMocks.importFromChannels).toHaveBeenCalledTimes(1)
    expect(apiMocks.listModels).toHaveBeenCalledTimes(2)
    expect(appStoreMock.showSuccess).toHaveBeenCalledWith('imported ok')
  })

  it('surfaces load failures through the app store', async () => {
    apiMocks.getAllGroups.mockResolvedValue([])
    apiMocks.listModels.mockRejectedValue(new Error('boom'))

    mount(ModelMarketView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          SearchInput: { template: '<input />' },
          Select: SelectStub,
          PlatformIcon: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(appStoreMock.showError).toHaveBeenCalledWith('admin.modelMarket.loadFailed')
  })

  it('filters channel group display by channel platform', async () => {
    apiMocks.getAllGroups.mockResolvedValue([
      { id: 1, name: 'OpenAI Ops', platform: 'openai' },
      { id: 2, name: 'Gemini Ops', platform: 'gemini' },
    ])
    apiMocks.listModels.mockResolvedValue([
      {
        id: 11,
        model_id: 'gpt-5.4',
        display_name: 'GPT-5.4',
        provider_key: 'openai_compatible',
        protocol: 'openai',
        capabilities: ['chat'],
        tags: [],
        status: 'active',
        sort_order: 1,
        channel_refs: [
          {
            channel_id: 9,
            channel_name: 'shared-channel',
            channel_status: 'active',
            platform: 'openai',
            group_ids: [1, 2],
            pricing: null,
          },
        ],
        available_channel_count: 1,
      },
    ])

    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          SearchInput: { template: '<input />' },
          Select: SelectStub,
          PlatformIcon: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI Ops')
    expect(wrapper.text()).not.toContain('Gemini Ops')
  })
})
