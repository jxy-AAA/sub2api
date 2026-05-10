import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

const authStore = vi.hoisted(() => ({
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  sidebarCollapsed: false,
  mobileOpen: false,
  backendModeEnabled: false,
  cachedPublicSettings: { custom_menu_items: [] },
  publicSettingsLoaded: true,
  siteName: 'Sub2API',
  siteLogo: '',
  siteVersion: '',
}))

const adminSettingsStore = vi.hoisted(() => ({
  customMenuItems: [] as unknown[],
  opsMonitoringEnabled: false,
  paymentEnabled: false,
  fetch: vi.fn(),
}))

const onboardingStore = vi.hoisted(() => ({
  isCurrentStep: vi.fn(() => false),
  nextStep: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => authStore,
  useAppStore: () => appStore,
  useAdminSettingsStore: () => adminSettingsStore,
  useOnboardingStore: () => onboardingStore,
}))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'nav.dashboard': "\u4eea\u8868\u76d8",
      'nav.apiKeys': "API \u5bc6\u94a5",
      'nav.apihubGuide': "apihub\u4f7f\u7528\u6559\u7a0b",
      'nav.usage': "\u4f7f\u7528\u8bb0\u5f55",
      'nav.myAccount': "\u6211\u7684\u8d26\u6237",
    }[key] ?? key),
  }),
}))
vi.mock('@/utils/featureFlags', () => ({
  FeatureFlags: {},
  makeSidebarFlag: () => () => false,
}))
vi.mock('@/components/common/VersionBadge.vue', () => ({
  default: { name: 'VersionBadge', template: '<span />' },
}))
vi.mock('@/api/userAffiliateManaged', () => ({
  emptyManagedAffiliatePermissions: () => ({}),
  getManagedAffiliatePermissions: vi.fn(async () => ({})),
  hasManagedAffiliateAccess: () => false,
}))
vi.mock('@/utils/sanitize', () => ({ sanitizeSvg: (value: string) => value }))

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: { template: '<div>dashboard</div>' } },
      { path: '/admin/dashboard', component: { template: '<div>admin dashboard</div>' } },
      { path: '/keys', component: { template: '<div>keys</div>' } },
      { path: '/apihub-guide', component: { template: '<div>guide</div>' } },
      { path: '/usage', component: { template: '<div>usage</div>' } },
    ],
  })
}

async function renderSidebar(initialPath: string) {
  vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
    media: '',
    onchange: null,
  }))

  const { default: AppSidebar } = await import('@/components/layout/AppSidebar.vue')
  const router = createTestRouter()
  await router.push(initialPath)
  await router.isReady()

  const wrapper = mount(AppSidebar, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

describe('AppSidebar APIHub guide navigation', () => {
  beforeEach(() => {
    authStore.isAuthenticated = true
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    appStore.sidebarCollapsed = false
    appStore.mobileOpen = false
    appStore.backendModeEnabled = false
    appStore.cachedPublicSettings = { custom_menu_items: [] }
    adminSettingsStore.customMenuItems = []
    adminSettingsStore.fetch.mockReset()
    onboardingStore.isCurrentStep.mockReset()
    onboardingStore.isCurrentStep.mockReturnValue(false)
    localStorage.clear()
  })

  it('shows the Chinese APIHub guide entry after API keys for regular users', async () => {
    const wrapper = await renderSidebar('/dashboard')
    const links = wrapper.findAll('a')
    const linkTexts = links.map((link) => link.text())
    const guideLink = links.find((link) => link.text().includes("apihub\u4f7f\u7528\u6559\u7a0b"))
    const apiKeysIndex = linkTexts.findIndex((text) => text.includes("API \u5bc6\u94a5"))
    const guideIndex = linkTexts.findIndex((text) => text.includes("apihub\u4f7f\u7528\u6559\u7a0b"))

    expect(guideLink).toBeDefined()
    expect(guideLink?.attributes('href')).toBe('/apihub-guide')
    expect(apiKeysIndex).toBeGreaterThanOrEqual(0)
    expect(guideIndex).toBe(apiKeysIndex + 1)
    expect(wrapper.text()).not.toContain('APIHub Guide')
  })

  it('shows the Chinese APIHub guide entry in the admin personal section', async () => {
    authStore.isAdmin = true

    const wrapper = await renderSidebar('/admin/dashboard')
    const links = wrapper.findAll('a')
    const guideLinks = links.filter((link) => link.text().includes("apihub\u4f7f\u7528\u6559\u7a0b"))

    expect(wrapper.text()).toContain("\u6211\u7684\u8d26\u6237")
    expect(guideLinks).toHaveLength(1)
    expect(guideLinks[0]?.attributes('href')).toBe('/apihub-guide')
    expect(wrapper.text()).not.toContain('APIHub Guide')
  })
})
