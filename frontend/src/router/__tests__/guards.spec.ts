import { describe, expect, it, vi } from 'vitest'
import router from '../index'
import {
  isBackendModePublicRouteAllowed,
  resolveNavigationGuardRedirect,
  type GuardAppState,
  type GuardAuthState,
  type GuardRouteTarget,
} from '../guards'

vi.mock('@/composables/useNavigationLoading', () => {
  const startNavigation = vi.fn()
  const endNavigation = vi.fn()

  return {
    useNavigationLoadingState: () => ({
      startNavigation,
      endNavigation,
      isLoading: { value: false },
    }),
    useNavigationLoading: () => ({
      startNavigation,
      endNavigation,
      isLoading: { value: false },
    }),
  }
})

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

function createAuthState(overrides: Partial<GuardAuthState> = {}): GuardAuthState {
  return {
    isAuthenticated: false,
    isAdmin: false,
    isSimpleMode: false,
    hasPendingAuthSession: false,
    hasPendingRegistrationChallenge: false,
    ...overrides,
  }
}

function createAppState(overrides: Partial<GuardAppState> = {}): GuardAppState {
  return {
    backendModeEnabled: false,
    publicSettings: undefined,
    ...overrides,
  }
}

function resolveRoute(path: string): GuardRouteTarget {
  const resolved = router.resolve(path)

  return {
    path: resolved.path,
    fullPath: resolved.fullPath,
    meta: resolved.meta,
  }
}

describe('router guard helpers', () => {
  describe('backend mode public allowlist', () => {
    it('allows legal documents in backend mode', () => {
      expect(isBackendModePublicRouteAllowed('/legal/privacy', false, false)).toBe(true)
    })

    it('allows register only with pending auth state', () => {
      expect(isBackendModePublicRouteAllowed('/register', false, false)).toBe(false)
      expect(isBackendModePublicRouteAllowed('/register', true, false)).toBe(true)
    })
  })

  describe('public route redirects', () => {
    it('blocks unauthenticated public pages outside backend allowlist', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/home'),
        createAuthState(),
        createAppState({ backendModeEnabled: true }),
      )

      expect(redirect).toBe('/login')
    })

    it('keeps legal pages reachable in backend mode', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/legal/privacy'),
        createAuthState(),
        createAppState({ backendModeEnabled: true }),
      )

      expect(redirect).toBeNull()
    })

    it('avoids backend-mode login redirect loops for authenticated non-admin users', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/login'),
        createAuthState({ isAuthenticated: true }),
        createAppState({ backendModeEnabled: true }),
      )

      expect(redirect).toBeNull()
    })
  })

  describe('protected route redirects', () => {
    it('keeps login redirect target for protected routes', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/dashboard?tab=usage'),
        createAuthState(),
        createAppState(),
      )

      expect(redirect).toEqual({
        path: '/login',
        query: { redirect: '/dashboard?tab=usage' },
      })
    })

    it('redirects non-admin users away from admin routes', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/admin/settings'),
        createAuthState({ isAuthenticated: true }),
        createAppState(),
      )

      expect(redirect).toBe('/dashboard')
    })
  })

  describe('feature gate redirects', () => {
    it('treats payment routes as enabled until settings explicitly disable them', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/purchase'),
        createAuthState({ isAuthenticated: true }),
        createAppState(),
      )

      expect(redirect).toBeNull()
    })

    it('redirects when payment is explicitly disabled', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/purchase'),
        createAuthState({ isAuthenticated: true }),
        createAppState({ publicSettings: { payment_enabled: false } }),
      )

      expect(redirect).toBe('/dashboard')
    })

    it('uses shared risk-control flag semantics for admin routes', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/admin/risk-control'),
        createAuthState({ isAuthenticated: true, isAdmin: true }),
        createAppState({ publicSettings: { risk_control_enabled: false } }),
      )

      expect(redirect).toBe('/admin/settings')
    })
  })

  describe('simple mode restrictions', () => {
    it('redirects simple-mode users away from restricted routes', () => {
      const redirect = resolveNavigationGuardRedirect(
        resolveRoute('/subscriptions'),
        createAuthState({ isAuthenticated: true, isSimpleMode: true }),
        createAppState(),
      )

      expect(redirect).toBe('/dashboard')
    })
  })
})
