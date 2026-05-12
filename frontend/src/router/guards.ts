import type { PublicSettings } from '@/types'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

export interface GuardRouteMeta {
  readonly requiresAuth?: boolean
  readonly requiresAdmin?: boolean
  readonly requiresPayment?: boolean
  readonly requiresRiskControl?: boolean
}

export interface GuardRouteTarget {
  readonly path: string
  readonly fullPath: string
  readonly meta: GuardRouteMeta
}

export interface GuardAuthState {
  readonly isAuthenticated: boolean
  readonly isAdmin: boolean
  readonly isSimpleMode: boolean
  readonly hasPendingAuthSession: boolean
  readonly hasPendingRegistrationChallenge: boolean
}

export interface GuardAppState {
  readonly backendModeEnabled: boolean
  readonly publicSettings?: Partial<PublicSettings> | null
}

export type GuardRedirect =
  | string
  | {
    path: string
    query: Record<string, string>
  }
  | null

const BACKEND_MODE_ALLOWED_PATHS = [
  '/login',
  '/key-usage',
  '/setup',
  '/payment/result',
  '/legal',
] as const

const BACKEND_MODE_CALLBACK_PATHS = [
  '/auth/callback',
  '/auth/linuxdo/callback',
  '/auth/oidc/callback',
  '/auth/wechat/callback',
  '/auth/wechat/payment/callback',
] as const

const BACKEND_MODE_PENDING_AUTH_PATHS = ['/register', '/email-verify'] as const

const SIMPLE_MODE_RESTRICTED_PATHS = [
  '/admin/groups',
  '/admin/subscriptions',
  '/admin/redeem',
  '/subscriptions',
  '/redeem',
] as const

function matchesPath(path: string, candidate: string): boolean {
  return path === candidate || path.startsWith(`${candidate}/`)
}

function getSignedInHomePath(isAdmin: boolean): string {
  return isAdmin ? '/admin/dashboard' : '/dashboard'
}

export function isBackendModePublicRouteAllowed(
  path: string,
  hasPendingAuthSession: boolean,
  hasPendingRegistrationChallenge: boolean,
): boolean {
  if (BACKEND_MODE_ALLOWED_PATHS.some((allowedPath) => matchesPath(path, allowedPath))) {
    return true
  }

  if (BACKEND_MODE_CALLBACK_PATHS.some((callbackPath) => path === callbackPath)) {
    return true
  }

  if (
    (hasPendingAuthSession || hasPendingRegistrationChallenge)
    && BACKEND_MODE_PENDING_AUTH_PATHS.some((allowedPath) => path === allowedPath)
  ) {
    return true
  }

  return false
}

function resolvePublicRouteRedirect(
  route: GuardRouteTarget,
  authState: GuardAuthState,
  appState: GuardAppState,
): GuardRedirect {
  if (authState.isAuthenticated && route.path === '/email-verify') {
    if (appState.backendModeEnabled && !authState.isAdmin) {
      return '/login'
    }

    return getSignedInHomePath(authState.isAdmin)
  }

  if (authState.isAuthenticated && (route.path === '/login' || route.path === '/register')) {
    if (appState.backendModeEnabled && !authState.isAdmin) {
      return null
    }

    return getSignedInHomePath(authState.isAdmin)
  }

  if (appState.backendModeEnabled && !authState.isAuthenticated) {
    const isAllowed = isBackendModePublicRouteAllowed(
      route.path,
      authState.hasPendingAuthSession,
      authState.hasPendingRegistrationChallenge,
    )

    if (!isAllowed) {
      return '/login'
    }
  }

  return null
}

function resolveFeatureGateRedirect(
  route: GuardRouteTarget,
  authState: GuardAuthState,
  appState: GuardAppState,
): GuardRedirect {
  if (
    route.meta.requiresPayment
    && !resolveFeatureFlag(FeatureFlags.payment, appState.publicSettings)
  ) {
    return getSignedInHomePath(authState.isAdmin)
  }

  if (
    route.meta.requiresRiskControl
    && !resolveFeatureFlag(FeatureFlags.riskControl, appState.publicSettings)
  ) {
    return authState.isAdmin ? '/admin/settings' : '/dashboard'
  }

  return null
}

function resolveSimpleModeRedirect(
  route: GuardRouteTarget,
  authState: GuardAuthState,
): GuardRedirect {
  if (!authState.isSimpleMode) {
    return null
  }

  if (SIMPLE_MODE_RESTRICTED_PATHS.some((restrictedPath) => matchesPath(route.path, restrictedPath))) {
    return getSignedInHomePath(authState.isAdmin)
  }

  return null
}

function resolveBackendModeProtectedRedirect(
  route: GuardRouteTarget,
  authState: GuardAuthState,
  appState: GuardAppState,
): GuardRedirect {
  if (!appState.backendModeEnabled) {
    return null
  }

  if (authState.isAuthenticated && authState.isAdmin) {
    return null
  }

  const isAllowed = isBackendModePublicRouteAllowed(
    route.path,
    authState.hasPendingAuthSession,
    authState.hasPendingRegistrationChallenge,
  )

  return isAllowed ? null : '/login'
}

function resolveProtectedRouteRedirect(
  route: GuardRouteTarget,
  authState: GuardAuthState,
  appState: GuardAppState,
): GuardRedirect {
  if (!authState.isAuthenticated) {
    return {
      path: '/login',
      query: { redirect: route.fullPath },
    }
  }

  if (route.meta.requiresAdmin && !authState.isAdmin) {
    return '/dashboard'
  }

  const featureRedirect = resolveFeatureGateRedirect(route, authState, appState)
  if (featureRedirect) {
    return featureRedirect
  }

  const simpleModeRedirect = resolveSimpleModeRedirect(route, authState)
  if (simpleModeRedirect) {
    return simpleModeRedirect
  }

  return resolveBackendModeProtectedRedirect(route, authState, appState)
}

export function resolveNavigationGuardRedirect(
  route: GuardRouteTarget,
  authState: GuardAuthState,
  appState: GuardAppState,
): GuardRedirect {
  const requiresAuth = route.meta.requiresAuth !== false

  if (!requiresAuth) {
    return resolvePublicRouteRedirect(route, authState, appState)
  }

  return resolveProtectedRouteRedirect(route, authState, appState)
}
