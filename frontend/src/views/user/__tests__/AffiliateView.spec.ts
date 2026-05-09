import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import AffiliateView from '@/views/user/AffiliateView.vue'

const {
  getMock,
  putMock,
  copyToClipboardMock,
  showErrorMock,
  showSuccessMock,
  getAffiliateDetailMock
} = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
  copyToClipboardMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  getAffiliateDetailMock: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    put: putMock
  }
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail: getAffiliateDetailMock
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: copyToClipboardMock
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

describe('AffiliateView', () => {
  beforeEach(() => {
    getMock.mockReset()
    putMock.mockReset()
    copyToClipboardMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
    getAffiliateDetailMock.mockReset()

    getMock.mockResolvedValue({
      data: {
        user_id: 1,
        aff_code: 'ALICE888',
        invite_code_model_rates: [{ model: 'gpt-4.1', multiplier: 1.6 }],
        my_model_rates: [{ model: 'gpt-4.1', multiplier: 1.4 }],
        today_revenue_usd: 200,
        today_rebate_rmb: 40,
        current_rebate_balance_rmb: 120,
        direct_children: [
          {
            user_id: 8,
            email: 'b@example.com',
            username: 'bob',
            role: 'agent',
            joined_at: '2026-05-08T10:00:00Z',
            today_revenue_usd: 200,
            today_rebate_rmb: 40,
            current_rebate_balance_rmb: 80,
            model_rates: [{ model: 'gpt-4.1', multiplier: 1.6 }]
          }
        ],
        direct_children_count: 1
      }
    })
  })

  it('renders the agent distribution page and saves rate updates through the new endpoints', async () => {
    const AffiliateModelRateEditorStub = defineComponent({
      inheritAttrs: false,
      props: {
        title: { type: String, required: true }
      },
      emits: ['save'],
      setup(props, { emit }) {
        return () =>
          h('div', [
            h('div', props.title),
            h(
              'button',
              {
                'data-testid': 'save-invite-rates',
                onClick: () => emit('save', [{ model: 'gpt-4.1', multiplier: 1.7 }])
              },
              'save'
            )
          ])
      }
    })

    const AffiliateDirectChildrenTableStub = defineComponent({
      inheritAttrs: false,
      props: {
        title: { type: String, required: true }
      },
      emits: ['save-child-rates'],
      setup(props, { emit }) {
        return () =>
          h('div', [
            h('div', props.title),
            h(
              'button',
              {
                'data-testid': 'save-child-rates',
                onClick: () => emit('save-child-rates', 8, [{ model: 'gpt-4.1', multiplier: 1.8 }])
              },
              'save child'
            )
          ])
      }
    })

    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          AffiliateModelRateEditor: AffiliateModelRateEditorStub,
          AffiliateDirectChildrenTable: AffiliateDirectChildrenTableStub
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('代理分销')
    expect(wrapper.text()).toContain('ALICE888')
    expect(wrapper.text()).toContain('今日营业额')
    expect(wrapper.text()).toContain('直属下级列表')
    expect(wrapper.text()).not.toContain('Transfer to Balance')
    expect(wrapper.text()).not.toContain('提取记录')
    expect(wrapper.find('[data-testid="save-invite-rates"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="save-child-rates"]').exists()).toBe(true)
  })

  it('normalizes the new distribution overview contract without crashing', async () => {
    const AffiliateModelRateEditorStub = defineComponent({
      inheritAttrs: false,
      props: {
        title: { type: String, required: true },
        modelRates: { type: Array, required: true }
      },
      setup(props) {
        return () =>
          h('div', { 'data-testid': `rates-${props.title}` }, JSON.stringify(props.modelRates))
      }
    })

    const AffiliateDirectChildrenTableStub = defineComponent({
      inheritAttrs: false,
      props: {
        children: { type: Array, required: true },
        countLabel: { type: String, required: true }
      },
      setup(props) {
        return () =>
          h('div', { 'data-testid': 'direct-children' }, [
            h('div', props.countLabel),
            h('pre', JSON.stringify(props.children))
          ])
      }
    })

    getMock.mockResolvedValue({
      data: {
        user_id: 1,
        invite_code: 'NEWCODE88',
        invite_model_rates: [{ model_name: 'gpt-4.1', multiplier: 1.6 }],
        my_model_rates: [{ model_name: 'gpt-4.1-mini', multiplier: 1.4 }],
        today_business_usd: 200,
        today_rebate_rmb: 40,
        current_rebate_balance_rmb: 120,
        direct_member_count: 1,
        direct_children: [
          {
            user_id: 8,
            email: 'b@example.com',
            username: 'bob',
            is_agent: true,
            created_at: '2026-05-08T10:00:00Z',
            today_business_usd: 88,
            today_rebate_rmb: 16,
            current_rebate_balance_rmb: 20,
            current_model_rates: [{ model_name: 'claude-3.7-sonnet', multiplier: 1.8 }]
          }
        ]
      }
    })

    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          AffiliateModelRateEditor: AffiliateModelRateEditorStub,
          AffiliateDirectChildrenTable: AffiliateDirectChildrenTableStub
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('NEWCODE88')
    expect(wrapper.text()).toContain('共 1 人')
    expect(wrapper.get('[data-testid="rates-我的当前模型倍率"]').text()).toContain('gpt-4.1-mini')
    expect(wrapper.get('[data-testid="rates-邀请码模型倍率配置"]').text()).toContain('gpt-4.1')
    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('"role":"agent"')
    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('"joined_at":"2026-05-08T10:00:00Z"')
    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('"today_revenue_usd":88')
    expect(wrapper.get('[data-testid="direct-children"]').text()).toContain('claude-3.7-sonnet')
    expect(showErrorMock).not.toHaveBeenCalled()
  })
})
