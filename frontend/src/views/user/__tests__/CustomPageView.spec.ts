import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CustomPageView from '../CustomPageView.vue'

const routeState = vi.hoisted(() => ({
  params: { id: 'billing' },
}))

const appStoreState = vi.hoisted(() => ({
  cachedPublicSettings: {
    custom_menu_items: [
      {
        id: 'billing',
        label: 'Billing',
        icon_svg: '',
        url: 'https://portal.example.com/embed?plan=pro',
        visibility: 'user',
        sort_order: 1,
      },
    ],
  },
  publicSettingsLoaded: true,
  fetchPublicSettings: vi.fn().mockResolvedValue(undefined),
}))

const authStoreState = vi.hoisted(() => ({
  user: {
    id: 7,
    role: 'user',
  },
  token: 'jwt-real-user-token',
  isAdmin: false,
}))

const adminSettingsStoreState = vi.hoisted(() => ({
  customMenuItems: [],
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' },
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStoreState,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreState,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => adminSettingsStoreState,
}))

vi.mock('@/utils/markdown', () => ({
  renderMarkdown: (value: string) => value,
}))

describe('CustomPageView', () => {
  const originalLocation = window.location

  beforeEach(() => {
    routeState.params = { id: 'billing' }
    appStoreState.cachedPublicSettings = {
      custom_menu_items: [
        {
          id: 'billing',
          label: 'Billing',
          icon_svg: '',
          url: 'https://portal.example.com/embed?plan=pro',
          visibility: 'user',
          sort_order: 1,
        },
      ],
    }
    appStoreState.publicSettingsLoaded = true
    appStoreState.fetchPublicSettings.mockClear()
    authStoreState.user = { id: 7, role: 'user' }
    authStoreState.token = 'jwt-real-user-token'
    authStoreState.isAdmin = false

    Object.defineProperty(window, 'location', {
      value: {
        origin: 'https://app.example.com',
        href: 'https://app.example.com/custom/billing?from=dashboard&token=jwt-real-user-token&resume_token=resume-42',
      },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
  })

  it('renders iframe and new-tab URL without leaking real auth tokens', async () => {
    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
        },
      },
    })

    await flushPromises()

    const openLink = wrapper.get('a.custom-open-fab')
    const iframe = wrapper.get('iframe.custom-embed-frame')
    const href = openLink.attributes('href')
    const src = iframe.attributes('src')

    expect(href).toBe(src)
    expect(href).toContain('plan=pro')
    expect(href).toContain('user_id=7')
    expect(href).toContain('theme=light')
    expect(href).toContain('lang=zh-CN')
    expect(href).toContain('ui_mode=embedded')
    expect(href).toContain('src_host=https%3A%2F%2Fapp.example.com')
    expect(href).toContain('src_url=https%3A%2F%2Fapp.example.com%2Fcustom%2Fbilling%3Ffrom%3Ddashboard')
    expect(href).not.toContain('jwt-real-user-token')
    expect(href).not.toContain('resume-42')
    expect(href).not.toContain('token=')
    expect(iframe.attributes('referrerpolicy')).toBe('no-referrer')
  })
})
