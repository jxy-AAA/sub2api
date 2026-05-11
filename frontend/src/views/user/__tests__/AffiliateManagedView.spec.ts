import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import AffiliateManagedView from '@/views/user/AffiliateManagedView.vue'

const {
  getManagedAffiliatePermissionsMock,
  listManagedDailyRevenueRankingsMock,
  listManagedRebateBalanceRankingsMock,
  getManagedDistributionTreeMock,
  getManagedUserDistributionPricingMock,
  updateManagedUserDistributionPricingMock,
  getAvailableGroupsMock,
  showErrorMock,
  showSuccessMock,
} = vi.hoisted(() => ({
  getManagedAffiliatePermissionsMock: vi.fn(),
  listManagedDailyRevenueRankingsMock: vi.fn(),
  listManagedRebateBalanceRankingsMock: vi.fn(),
  getManagedDistributionTreeMock: vi.fn(),
  getManagedUserDistributionPricingMock: vi.fn(),
  updateManagedUserDistributionPricingMock: vi.fn(),
  getAvailableGroupsMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('@/api/userAffiliateManaged', () => ({
  emptyManagedAffiliatePermissions: () => ({
    can_view_downline_daily_revenue: false,
    can_view_downline_rebate_balances: false,
    can_manage_downline_pricing: false,
  }),
  hasManagedAffiliateAccess: (permissions: Record<string, boolean>) => Object.values(permissions).some(Boolean),
  getManagedAffiliatePermissions: getManagedAffiliatePermissionsMock,
  listManagedDailyRevenueRankings: listManagedDailyRevenueRankingsMock,
  listManagedRebateBalanceRankings: listManagedRebateBalanceRankingsMock,
  getManagedDistributionTree: getManagedDistributionTreeMock,
  getManagedUserDistributionPricing: getManagedUserDistributionPricingMock,
  updateManagedUserDistributionPricing: updateManagedUserDistributionPricingMock,
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

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const BaseDialogStub = defineComponent({
  props: { show: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () => props.show ? h('div', { 'data-testid': 'pricing-dialog' }, slots.default?.()) : null
  },
})

describe('AffiliateManagedView', () => {
  beforeEach(() => {
    getManagedAffiliatePermissionsMock.mockReset()
    listManagedDailyRevenueRankingsMock.mockReset()
    listManagedRebateBalanceRankingsMock.mockReset()
    getManagedDistributionTreeMock.mockReset()
    getManagedUserDistributionPricingMock.mockReset()
    updateManagedUserDistributionPricingMock.mockReset()
    getAvailableGroupsMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()

    getAvailableGroupsMock.mockResolvedValue([
      { id: 3, name: 'Pro', platform: 'openai', rate_multiplier: 1.2 },
      { id: 4, name: 'Fast', platform: 'anthropic', rate_multiplier: 1.4 },
    ])
    getManagedAffiliatePermissionsMock.mockResolvedValue({
      can_view_downline_daily_revenue: true,
      can_view_downline_rebate_balances: true,
      can_manage_downline_pricing: true,
    })
    listManagedDailyRevenueRankingsMock.mockResolvedValue({
      items: [
        {
          user_id: 12,
          email: 'agent@example.com',
          username: 'agent',
          date: '2026-05-10',
          daily_revenue_rmb: 120,
          direct_user_count: 1,
          direct_agent_count: 1,
        },
      ],
    })
    listManagedRebateBalanceRankingsMock.mockResolvedValue({
      items: [
        {
          user_id: 12,
          email: 'agent@example.com',
          username: 'agent',
          current_rebate_balance_rmb: 88,
          today_rebate_rmb: 12,
          monthly_rebate_rmb: 34,
        },
      ],
    })
    getManagedDistributionTreeMock.mockResolvedValue([
      {
        user_id: 18,
        inviter_id: 1,
        email: 'child@example.com',
        username: 'child',
        invite_code: 'CODE18',
        depth: 1,
        is_admin: false,
        is_agent: true,
        current_rebate_balance_rmb: 20,
        current_group_rates: [],
      },
    ])
    getManagedUserDistributionPricingMock.mockResolvedValue({
      user_id: 18,
      group_rates: [],
    })
    updateManagedUserDistributionPricingMock.mockResolvedValue({
      user_id: 18,
      group_rates: [
        { group_id: 3, group_name: 'Pro', rate_multiplier: 1.2 },
        { group_id: 4, group_name: 'Fast', rate_multiplier: 1.4 },
      ],
    })
  })

  it('defaults pricing rows from available groups and saves only group_rates', async () => {
    const wrapper = mount(AffiliateManagedView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: BaseDialogStub,
          AffiliateAdminUserCell: { template: '<div>user-cell</div>' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('下级每日营业额')
    expect(wrapper.text()).toContain('下级返利额度')
    expect(wrapper.text()).toContain('下级分组成本倍率管理')

    await wrapper.get('button.btn.btn-secondary.btn-sm').trigger('click')
    await flushPromises()

    const selects = wrapper.get('[data-testid="pricing-dialog"]').findAll('select')
    const inputs = wrapper.get('[data-testid="pricing-dialog"]').findAll('input[type="number"]')

    expect(selects).toHaveLength(2)
    expect((selects[0].element as HTMLSelectElement).value).toBe('3')
    expect((selects[1].element as HTMLSelectElement).value).toBe('4')
    expect((inputs[0].element as HTMLInputElement).value).toBe('1.2')
    expect((inputs[1].element as HTMLInputElement).value).toBe('1.4')

    await wrapper.get('button.btn.btn-primary').trigger('click')
    await flushPromises()

    expect(updateManagedUserDistributionPricingMock).toHaveBeenCalledWith(18, {
      group_rates: [
        { group_id: 3, rate_multiplier: 1.2 },
        { group_id: 4, rate_multiplier: 1.4 },
      ],
    })

    expect(showSuccessMock).toHaveBeenCalled()
  })

  it('shows no-access state when no permission is granted', async () => {
    getManagedAffiliatePermissionsMock.mockResolvedValueOnce({
      can_view_downline_daily_revenue: false,
      can_view_downline_rebate_balances: false,
      can_manage_downline_pricing: false,
    })

    const wrapper = mount(AffiliateManagedView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: BaseDialogStub,
          AffiliateAdminUserCell: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('暂未获得下级管理权限')
  })
})
