import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  listRecords: vi.fn(),
  listRules: vi.fn(),
  listExportTasks: vi.fn(),
  createExportTask: vi.fn(),
  getRecord: vi.fn(),
  deleteRecord: vi.fn(),
  batchDeleteRecords: vi.fn(),
  deleteRule: vi.fn(),
  cancelExportTask: vi.fn(),
  downloadExportTask: vi.fn(),
}))

const appStoreMock = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

const authStoreMock = vi.hoisted(() => ({
  user: {
    role: 'admin',
    is_root_admin: true,
  },
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
    traces: apiMocks,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStoreMock,
  useAuthStore: () => authStoreMock,
}))

import TracesView from '../TracesView.vue'

const DataTableStub = {
  props: ['data', 'loading'],
  template: `
    <div>
      <slot v-if="!loading && (!data || !data.length)" name="empty" />
      <div v-for="row in data" :key="row.id">
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `,
}

function defaultExportTask() {
  return {
    id: 21,
    status: 'pending',
    format: 'json_array',
    filters: {},
    include_raw: true,
    requested_by: 1,
    file_size_bytes: 0,
    target_records: 500,
    total_records: 0,
    processed_records: 0,
    created_at: '2026-05-27T10:00:00Z',
    updated_at: '2026-05-27T10:00:00Z',
  }
}

async function mountView() {
  apiMocks.listRecords.mockResolvedValue({
    items: [],
    total: 0,
    page: 1,
    page_size: 20,
    pages: 0,
  })
  apiMocks.listRules.mockResolvedValue([])
  apiMocks.listExportTasks.mockResolvedValue({
    items: [],
    total: 0,
    page: 1,
    page_size: 10,
    pages: 0,
  })
  apiMocks.createExportTask.mockResolvedValue(defaultExportTask())

  const wrapper = mount(TracesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: true,
        ConfirmDialog: true,
        Icon: true,
      },
    },
  })

  await flushPromises()
  return wrapper
}

async function openExportsTab(wrapper: ReturnType<typeof mount>) {
  const tab = wrapper
    .findAll('button')
    .find((button) => button.text().includes('admin.traces.tabs.exports'))
  expect(tab).toBeDefined()
  await tab!.trigger('click')
  await flushPromises()
}

describe('Admin TracesView export form', () => {
  beforeEach(() => {
    for (const mock of Object.values(apiMocks)) {
      mock.mockReset()
    }
    appStoreMock.showError.mockReset()
    appStoreMock.showSuccess.mockReset()
    appStoreMock.showWarning.mockReset()
    authStoreMock.user = {
      role: 'admin',
      is_root_admin: true,
    }
  })

  it('normalizes number-backed export fields before creating a task', async () => {
    const wrapper = await mountView()
    await openExportsTab(wrapper)

    await wrapper.find('input[placeholder="42"]').setValue(42)
    await wrapper.find('input[placeholder="108"]').setValue(108)
    await wrapper.find('input[placeholder="50000"]').setValue(1000)
    await wrapper.find('input[placeholder="500"]').setValue(500)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(apiMocks.createExportTask).toHaveBeenCalledTimes(1)
    expect(apiMocks.createExportTask).toHaveBeenCalledWith({
      filters: expect.objectContaining({
        user_id: 42,
        api_key_id: 108,
        max_total_tokens: 1000,
        timezone: expect.any(String),
      }),
      include_raw: true,
      target_records: 500,
    })
    expect(appStoreMock.showError).not.toHaveBeenCalled()
  })

  it('shows one validation toast for an invalid number-backed target record count', async () => {
    const wrapper = await mountView()
    await openExportsTab(wrapper)

    await wrapper.find('input[placeholder="500"]').setValue(0)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(apiMocks.createExportTask).not.toHaveBeenCalled()
    expect(appStoreMock.showError).toHaveBeenCalledTimes(1)
    expect(appStoreMock.showError).toHaveBeenCalledWith('admin.traces.validation.invalidPositiveInteger')
  })

  it('submits safely after copying number-backed record filters to export filters', async () => {
    const wrapper = await mountView()

    await wrapper.find('input[placeholder="42"]').setValue(42)
    await wrapper.find('input[placeholder="50000"]').setValue(1000)
    await openExportsTab(wrapper)

    const copyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.traces.export.copyFromRecords'))
    expect(copyButton).toBeDefined()
    await copyButton!.trigger('click')
    await wrapper.find('input[placeholder="500"]').setValue(500)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(apiMocks.createExportTask).toHaveBeenCalledTimes(1)
    expect(apiMocks.createExportTask).toHaveBeenCalledWith({
      filters: expect.objectContaining({
        user_id: 42,
        max_total_tokens: 1000,
        timezone: expect.any(String),
      }),
      include_raw: true,
      target_records: 500,
    })
  })
})
