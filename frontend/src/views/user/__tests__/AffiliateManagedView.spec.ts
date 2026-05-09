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
  showErrorMock,
  showSuccessMock,
} = vi.hoisted(() => ({
  getManagedAffiliatePermissionsMock: vi.fn(),
  listManagedDailyRevenueRankingsMock: vi.fn(),
  listManagedRebateBalanceRankingsMock: vi.fn(),
  getManagedDistributionTreeMock: vi.fn(),
  getManagedUserDistributionPricingMock: vi.fn(),
  updateManagedUserDistributionPricingMock: vi.fn(),
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

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock,
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  props: { show: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () => props.show ? h('div', slots.default?.()) : null
  }
})

describe('AffiliateManagedView', () => {
  beforeEach(() => {
    getManagedAffiliatePermissionsMock.mockReset()
    listManagedDailyRevenueRankingsMock.mockReset()
    listManagedRebateBalanceRankingsMock.mockReset()
    getManagedDistributionTreeMock.mockReset()
    getManagedUserDistributionPricingMock.mockReset()
    updateManagedUserDistributionPricingMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()

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
        }
      ]
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
        }
      ]
    })
    getManagedDistributionTreeMock.mockResolvedValue([
      {
        user_id: 18,
        inviter_id: 1,
        email: 'child@example.com',
        username: 'child',
        invite_code: 'CODE18',
        depth: 1,
        is_agent: true,
        current_rebate_balance_rmb: 20,
        current_model_rates: [{ model_name: 'gpt-5.4', multiplier: 1.7 }],
      }
    ])
    getManagedUserDistributionPricingMock.mockResolvedValue({
      user_id: 18,
      model_rates: [{ model_name: 'gpt-5.4', multiplier: 1.8 }]
    })
    updateManagedUserDistributionPricingMock.mockResolvedValue({
      user_id: 18,
      model_rates: [{ model_name: 'gpt-5.4', multiplier: 1.8 }]
    })
  })

  it('renders authorized sections and saves descendant pricing', async () => {
    const wrapper = mount(AffiliateManagedView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: BaseDialogStub,
          AffiliateAdminUserCell: { template: '<div><slot />user-cell</div>' },
          Icon: true,
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('下级每日营业额')
    expect(wrapper.text()).toContain('下级返利额度')
    expect(wrapper.text()).toContain('下级模型倍率管理')

    await wrapper.find('button.btn.btn-secondary.btn-sm').trigger('click')
    await flushPromises()
    await wrapper.find('button.btn.btn-primary').trigger('click')
    await flushPromises()

    expect(updateManagedUserDistributionPricingMock).toHaveBeenCalledWith(18, {
      model_rates: [{ model: 'gpt-5.4', multiplier: 1.8 }]
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
          Icon: true,
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('暂未获得下级管理权限')
  })
})
