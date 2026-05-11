import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'

import AdminAffiliateManagementView from '@/views/admin/affiliates/AdminAffiliateManagementView.vue'

const {
  getDistributionTreeMock,
  getDefaultDistributionPricingMock,
  updateDefaultDistributionPricingMock,
  getUserDistributionPricingMock,
  updateUserDistributionPricingMock,
  getUserInvitePricingMock,
  updateUserInvitePricingMock,
  updateUserUpstreamMock,
  lookupUsersMock,
  getUserDistributionPermissionsMock,
  updateUserDistributionPermissionsMock,
  getAllGroupsMock,
  showErrorMock,
  showSuccessMock,
} = vi.hoisted(() => ({
  getDistributionTreeMock: vi.fn(),
  getDefaultDistributionPricingMock: vi.fn(),
  updateDefaultDistributionPricingMock: vi.fn(),
  getUserDistributionPricingMock: vi.fn(),
  updateUserDistributionPricingMock: vi.fn(),
  getUserInvitePricingMock: vi.fn(),
  updateUserInvitePricingMock: vi.fn(),
  updateUserUpstreamMock: vi.fn(),
  lookupUsersMock: vi.fn(),
  getUserDistributionPermissionsMock: vi.fn(),
  updateUserDistributionPermissionsMock: vi.fn(),
  getAllGroupsMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('@/api/admin/affiliates', () => {
  const api = {
    getDistributionTree: getDistributionTreeMock,
    getDefaultDistributionPricing: getDefaultDistributionPricingMock,
    updateDefaultDistributionPricing: updateDefaultDistributionPricingMock,
    getUserDistributionPricing: getUserDistributionPricingMock,
    updateUserDistributionPricing: updateUserDistributionPricingMock,
    getUserInvitePricing: getUserInvitePricingMock,
    updateUserInvitePricing: updateUserInvitePricingMock,
    updateUserUpstream: updateUserUpstreamMock,
    lookupUsers: lookupUsersMock,
    getUserDistributionPermissions: getUserDistributionPermissionsMock,
    updateUserDistributionPermissions: updateUserDistributionPermissionsMock,
  }

  return {
    default: api,
    affiliatesAPI: api,
  }
})

vi.mock('@/api/admin/groups', () => ({
  default: {
    getAll: getAllGroupsMock,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { id: 1 },
  }),
}))

const BaseDialogStub = defineComponent({
  props: {
    show: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    return () => props.show ? h('div', { 'data-test': 'dialog' }, slots.default?.()) : null
  },
})

const buildGroup = (id: number, name: string, rateMultiplier: number, platform = 'openai') => ({
  id,
  name,
  description: null,
  platform,
  rate_multiplier: rateMultiplier,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'free',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '2026-05-11T00:00:00Z',
  updated_at: '2026-05-11T00:00:00Z',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: false,
}) as any

describe('AdminAffiliateManagementView', () => {
  beforeEach(() => {
    getDistributionTreeMock.mockReset()
    getDefaultDistributionPricingMock.mockReset()
    updateDefaultDistributionPricingMock.mockReset()
    getUserDistributionPricingMock.mockReset()
    updateUserDistributionPricingMock.mockReset()
    getUserInvitePricingMock.mockReset()
    updateUserInvitePricingMock.mockReset()
    updateUserUpstreamMock.mockReset()
    lookupUsersMock.mockReset()
    getUserDistributionPermissionsMock.mockReset()
    updateUserDistributionPermissionsMock.mockReset()
    getAllGroupsMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()

    getAllGroupsMock.mockResolvedValue([
      buildGroup(1, '基础组', 1.1),
      buildGroup(2, '高级组', 1.2, 'anthropic'),
    ])
    getDistributionTreeMock.mockResolvedValue([])
    getDefaultDistributionPricingMock.mockResolvedValue({
      group_rates: [{ group_id: 2, rate_multiplier: 1.8 }],
    })
    updateDefaultDistributionPricingMock.mockImplementation(async (payload: { group_rates: Array<{ group_id: number; rate_multiplier: number }> }) => payload)
    getUserDistributionPricingMock.mockResolvedValue({
      user_id: 18,
      group_rates: [{ group_id: 2, rate_multiplier: 1.9 }],
    })
    updateUserDistributionPricingMock.mockResolvedValue({
      user_id: 18,
      group_rates: [{ group_id: 1, rate_multiplier: 1.6 }, { group_id: 2, rate_multiplier: 1.9 }],
    })
    getUserInvitePricingMock.mockResolvedValue({
      user_id: 18,
      group_rates: [{ group_id: 1, rate_multiplier: 1.7 }],
    })
    updateUserInvitePricingMock.mockResolvedValue({
      user_id: 18,
      group_rates: [{ group_id: 1, rate_multiplier: 1.7 }, { group_id: 2, rate_multiplier: 1.4 }],
    })
    updateUserUpstreamMock.mockResolvedValue({
      user_id: 18,
      inviter_id: 1,
    })
    lookupUsersMock.mockResolvedValue([])
    getUserDistributionPermissionsMock.mockResolvedValue({
      user_id: 18,
      can_view_downline_daily_revenue: true,
      can_view_downline_rebate_balances: true,
      can_manage_downline_pricing: true,
    })
    updateUserDistributionPermissionsMock.mockResolvedValue({
      user_id: 18,
      can_view_downline_daily_revenue: true,
      can_view_downline_rebate_balances: true,
      can_manage_downline_pricing: true,
    })
  })

  it('prefills enabled groups with group defaults and saved default overrides', async () => {
    const wrapper = mount(AdminAffiliateManagementView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: BaseDialogStub,
          Icon: true,
          Toggle: true,
        },
      },
    })

    await flushPromises()

    const inputs = wrapper.findAll('input[type="number"]')
    expect(inputs).toHaveLength(2)
    expect((inputs[0].element as HTMLInputElement).value).toBe('1.1')
    expect((inputs[1].element as HTMLInputElement).value).toBe('1.8')

    await inputs[0].setValue('1.3')
    await flushPromises()

    const saveButton = wrapper.findAll('button').find((button) => button.text().includes('保存默认倍率'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(updateDefaultDistributionPricingMock).toHaveBeenCalledWith({
      group_rates: [
        { group_id: 1, rate_multiplier: 1.3 },
        { group_id: 2, rate_multiplier: 1.8 },
      ],
    })
  })

  it('edits user pricing with group-only payloads and no model wording', async () => {
    getDistributionTreeMock.mockResolvedValue([
      {
        user_id: 18,
        inviter_id: 1,
        email: 'child@example.com',
        username: 'child',
        invite_code: 'CODE18',
        depth: 1,
        is_admin: false,
        is_root_admin: false,
        is_agent: true,
        current_rebate_balance_rmb: 20,
        current_group_rates: [{ group_id: 2, group_name: '高级组', rate_multiplier: 1.9 }],
        invite_group_rates: [{ group_id: 1, group_name: '基础组', rate_multiplier: 1.7 }],
      },
    ])

    const wrapper = mount(AdminAffiliateManagementView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: BaseDialogStub,
          Icon: true,
          Toggle: true,
        },
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find((button) => button.text().includes('编辑倍率 / 上游'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('模型倍率')

    const dialog = wrapper.get('[data-test="dialog"]')
    const dialogInputs = dialog.findAll('input[type="number"]')
    expect(dialogInputs).toHaveLength(4)

    await dialogInputs[0].setValue('1.6')
    await dialogInputs[3].setValue('1.4')
    await flushPromises()

    const saveButton = dialog.findAll('button').find((button) => button.text().includes('保存修改'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(updateUserDistributionPricingMock).toHaveBeenCalledWith(18, {
      group_rates: [
        { group_id: 1, rate_multiplier: 1.6 },
        { group_id: 2, rate_multiplier: 1.9 },
      ],
    })
    expect(updateUserInvitePricingMock).toHaveBeenCalledWith(18, {
      group_rates: [
        { group_id: 1, rate_multiplier: 1.7 },
        { group_id: 2, rate_multiplier: 1.4 },
      ],
    })
    expect(updateUserUpstreamMock).not.toHaveBeenCalled()
    expect(showSuccessMock).toHaveBeenCalledWith('代理倍率与上游配置已更新')
  })
})
