import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import AffiliateView from '@/views/user/AffiliateView.vue'

const {
  getAffiliateOverviewMock,
  updateInviteCodeGroupRatesMock,
  getAvailableGroupsMock,
  copyToClipboardMock,
  showErrorMock,
  showSuccessMock,
} = vi.hoisted(() => ({
  getAffiliateOverviewMock: vi.fn(),
  updateInviteCodeGroupRatesMock: vi.fn(),
  getAvailableGroupsMock: vi.fn(),
  copyToClipboardMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  getAffiliateOverview: getAffiliateOverviewMock,
  updateInviteCodeGroupRates: updateInviteCodeGroupRatesMock,
  updateAffiliateDirectChildGroupRates: vi.fn(),
}))

vi.mock('@/api/groups', () => ({
  default: {
    getAvailable: getAvailableGroupsMock,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: copyToClipboardMock,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const AffiliateGroupRateEditorStub = defineComponent({
  inheritAttrs: false,
  props: {
    title: { type: String, required: true },
  },
  emits: ['save'],
  setup(props, { emit }) {
    return () => h('div', [
      h('div', props.title),
      h(
        'button',
        {
          'data-testid': 'save-rate',
          onClick: () => emit('save', [{ group_id: 1, rate_multiplier: 1.7 }]),
        },
        'save',
      ),
    ])
  },
})

const AffiliateDirectChildrenTableStub = defineComponent({
  inheritAttrs: false,
  props: {
    title: { type: String, required: true },
    children: { type: Array, required: true },
    emptyText: { type: String, required: true },
    countLabel: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', { 'data-testid': 'direct-children' }, [
      h('div', props.title),
      h('div', { 'data-testid': 'children-count' }, props.countLabel),
      h('div', { 'data-testid': 'children-empty' }, props.emptyText),
      h('pre', JSON.stringify(props.children)),
    ])
  },
})

function mountAffiliateView() {
  return mount(AffiliateView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        AffiliateGroupRateEditor: AffiliateGroupRateEditorStub,
        AffiliateDirectChildrenTable: AffiliateDirectChildrenTableStub,
      },
    },
  })
}

describe('AffiliateView', () => {
  beforeEach(() => {
    getAffiliateOverviewMock.mockReset()
    updateInviteCodeGroupRatesMock.mockReset()
    getAvailableGroupsMock.mockReset()
    copyToClipboardMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()

    getAvailableGroupsMock.mockResolvedValue([
      { id: 1, name: 'Default', platform: 'openai', rate_multiplier: 1 },
    ])
    getAffiliateOverviewMock.mockResolvedValue({
      user_id: 1,
      aff_code: 'ALICE888',
      inviter_id: null,
      invite_group_rates: [{ group_id: 1, group_name: 'Default', rate_multiplier: 1.6 }],
      my_group_rates: [{ group_id: 1, group_name: 'Default', rate_multiplier: 1.4 }],
      today_revenue_usd: 200,
      today_rebate_rmb: 40,
      current_rebate_balance_rmb: 120,
      direct_children_count: 1,
      direct_children: [{
        user_id: 8,
        email: 'b@example.com',
        username: 'bob',
        role: 'agent',
        joined_at: '2026-05-08T10:00:00Z',
        today_revenue_usd: 200,
        today_rebate_rmb: 40,
        current_rebate_balance_rmb: 80,
        group_rates: [{ group_id: 1, group_name: 'Default', rate_multiplier: 1.6 }],
      }],
    })
    updateInviteCodeGroupRatesMock.mockResolvedValue({
      group_rates: [{ group_id: 1, group_name: 'Default', rate_multiplier: 1.7 }],
    })
  })

  it('renders normalized distribution data and saves invite group_rates only', async () => {
    const wrapper = mountAffiliateView()

    await flushPromises()

    expect(wrapper.text()).toContain('ALICE888')
    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('b@example.com')

    await wrapper.findAll('[data-testid="save-rate"]')[1].trigger('click')
    await flushPromises()

    expect(updateInviteCodeGroupRatesMock).toHaveBeenCalledWith({
      group_rates: [{ group_id: 1, rate_multiplier: 1.7 }],
    })
    expect(showSuccessMock).toHaveBeenCalled()
  })

  it('generates and copies a fixed referral registration link', async () => {
    const wrapper = mountAffiliateView()
    await flushPromises()

    await wrapper.get('[data-testid="generate-invite-link"]').trigger('click')
    await flushPromises()

    const expectedLink = `${window.location.origin}/register?aff=ALICE888`
    expect(copyToClipboardMock).toHaveBeenCalledWith(expectedLink, '邀请链接已复制')
    expect(wrapper.get('[data-testid="generated-invite-link"]').text()).toBe(expectedLink)
  })

  it('renders direct children provided by the wrapper response', async () => {
    getAvailableGroupsMock.mockResolvedValueOnce([
      { id: 2, name: 'Fallback', platform: 'anthropic', rate_multiplier: 1.2 },
    ])
    getAffiliateOverviewMock.mockResolvedValueOnce({
      user_id: 1,
      aff_code: 'DIRECT88',
      inviter_id: null,
      invite_group_rates: [],
      my_group_rates: [],
      today_revenue_usd: 120,
      today_rebate_rmb: 30,
      current_rebate_balance_rmb: 50,
      direct_children_count: 1,
      direct_children: [{
        user_id: 18,
        email: 'direct@example.com',
        username: 'direct-user',
        role: 'user',
        joined_at: '2026-05-07T00:00:00Z',
        today_revenue_usd: 66,
        today_rebate_rmb: 12,
        current_rebate_balance_rmb: 20,
        group_rates: [{ group_id: 2, group_name: 'Fallback', rate_multiplier: 1.9 }],
      }],
    })

    const wrapper = mountAffiliateView()
    await flushPromises()

    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('direct@example.com')
    expect(wrapper.get('[data-testid="children-count"]').text()).toContain('1')
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('shows an empty state when the wrapper returns no direct children', async () => {
    getAffiliateOverviewMock.mockResolvedValueOnce({
      user_id: 1,
      aff_code: 'EMPTY',
      inviter_id: null,
      invite_group_rates: [],
      my_group_rates: [],
      today_revenue_usd: 0,
      today_rebate_rmb: 0,
      current_rebate_balance_rmb: 0,
      direct_children_count: 0,
      direct_children: [],
    })

    const wrapper = mountAffiliateView()
    await flushPromises()

    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('[]')
    expect(showErrorMock).not.toHaveBeenCalled()
  })
})
