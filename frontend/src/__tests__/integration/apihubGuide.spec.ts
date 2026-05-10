import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import router from '@/router'
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ checkAuth: vi.fn(), isAuthenticated: true, isAdmin: false, isSimpleMode: false, hasPendingAuthSession: false }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ siteName: 'Sub2API', backendModeEnabled: false, cachedPublicSettings: null }) }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({ customMenuItems: [] }) }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { name: 'AppLayout', template: '<div data-testid="app-layout"><slot /></div>' } }))
vi.mock('@/components/icons', () => ({ Icon: { name: 'Icon', props: ['name', 'size'], template: '<span :data-icon="name"></span>' } }))

let scrollIntoView: ReturnType<typeof vi.fn>
beforeEach(() => {
  scrollIntoView = vi.fn()
  Object.defineProperty(window.HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: scrollIntoView })
  vi.stubGlobal('IntersectionObserver', vi.fn(() => ({ observe: vi.fn(), disconnect: vi.fn(), unobserve: vi.fn(), takeRecords: vi.fn(() => []) })))
})

describe('APIHub guide route integration', () => {
  it('registers the guide as an authenticated user page with a Chinese navigation title', () => {
    const resolved = router.resolve('/apihub-guide')
    const route = router.getRoutes().find((record) => record.name === 'ApihubGuide')
    expect(resolved.name).toBe('ApihubGuide')
    expect(route?.path).toBe('/apihub-guide')
    expect(route?.meta.requiresAuth).toBe(true)
    expect(route?.meta.requiresAdmin).toBe(false)
    expect(route?.meta.title).toBe("apihub\u4f7f\u7528\u6559\u7a0b")
    expect(route?.meta.titleKey).toBe('nav.apihubGuide')
  })
  it('renders source-structured Chinese CodeX guide content after login', async () => {
    const { default: ApihubGuideView } = await import('@/views/user/ApihubGuideView.vue')
    const wrapper = mount(ApihubGuideView, { attachTo: document.body })
    const text = wrapper.text()
    expect(wrapper.find('h1').text()).toContain("CodeX \u90e8\u7f72\u6307\u5357")
    expect(text).toContain("\u4f01\u4e1a\u7ea7 AI \u7f16\u7801\u52a9\u624b - \u5b8c\u6574\u90e8\u7f72\u624b\u518c")
    expect(text).toContain("\u5feb\u901f\u5bfc\u822a")
    expect(text).toContain("Linux \u5e73\u53f0")
    expect(text).toContain("\u4f7f\u7528 CC-Switch \u5feb\u901f\u914d\u7f6e\uff08\u63a8\u8350\uff09")
    expect(text).not.toContain(['I', 'K', 'u', 'n', 'C', 'o', 'd', 'e'].join(''))
    expect(text).not.toContain("\u552e\u524d\u552e\u540e")
    expect(text).not.toContain("\u5b98\u65b9\u4f18\u8d28\u9879\u76ee")
  })
  it('scrolls to the matching section when a left navigation item is clicked', async () => {
    const { default: ApihubGuideView } = await import('@/views/user/ApihubGuideView.vue')
    const wrapper = mount(ApihubGuideView, { attachTo: document.body })
    const target = wrapper.findAll('button').find((button) => button.text().includes("CC-Switch \u914d\u7f6e\u5de5\u5177"))
    expect(target).toBeDefined()
    await target?.trigger('click')
    expect(scrollIntoView).toHaveBeenCalled()
    expect(window.location.hash).toBe('#cc-switch-config')
  })
})
