/**
 * Authentication Store
 * Manages user authentication state, login/logout, token refresh, and token persistence
 */

import { defineStore } from 'pinia'
import { computed, readonly, ref } from 'vue'
import { authAPI, isTotp2FARequired, type LoginResponse } from '@/api'
import { refreshAccessToken } from '@/api/client'
import {
  clearPersistedSession,
  getAccessToken,
  getPersistedAuthUser,
  getTokenExpiresAt,
  hasPersistedSessionHint,
  setAccessToken,
  setPersistedAuthUser,
  setTokenExpiresAt,
} from '@/api/tokenStorage'
import type { AuthResponse, LoginRequest, RegisterRequest, User } from '@/types'

const PENDING_AUTH_SESSION_KEY = 'pending_auth_session'
const PENDING_REGISTRATION_CHALLENGE_KEY = 'pending_registration_challenge'
const AUTO_REFRESH_INTERVAL = 60 * 1000
const TOKEN_REFRESH_BUFFER = 120 * 1000

type PendingAuthTokenField = 'pending_auth_token' | 'pending_oauth_token'

interface PendingAuthSessionSummary {
  token: string
  token_field: PendingAuthTokenField
  provider: string
  redirect?: string
  adoption_required?: boolean
  suggested_display_name?: string
  suggested_avatar_url?: string
}

type PersistedPendingAuthSessionSummary = Omit<PendingAuthSessionSummary, 'token'>

interface PendingRegistrationAdoptionDecision {
  adoptDisplayName?: boolean
  adoptAvatar?: boolean
}

interface PendingRegistrationChallengeSummary {
  ticket: string
  email: string
  turnstile_token?: string
  invitation_code?: string
  aff_code?: string
  pending_auth_token?: string
  pending_auth_token_field?: PendingAuthTokenField
  pending_provider?: string
  pending_redirect?: string
  pending_adoption_decision?: PendingRegistrationAdoptionDecision
}

interface PendingRegistrationChallengeInput {
  email: string
  password: string
  turnstile_token?: string
  invitation_code?: string
  aff_code?: string
  pending_auth_token?: string
  pending_auth_token_field?: PendingAuthTokenField
  pending_provider?: string
  pending_redirect?: string
  pending_adoption_decision?: PendingRegistrationAdoptionDecision
}

interface PendingRegistrationChallengePayload extends PendingRegistrationChallengeSummary {
  password: string
}

type PersistedPendingRegistrationChallengeSummary = Omit<
  PendingRegistrationChallengeSummary,
  'pending_auth_token'
>

let pendingRegistrationPassword: string | null = null

function normalizePendingAuthTokenField(value: unknown): PendingAuthTokenField {
  return value === 'pending_oauth_token' ? 'pending_oauth_token' : 'pending_auth_token'
}

function normalizePendingRegistrationAdoptionDecision(
  value: unknown
): PendingRegistrationAdoptionDecision | undefined {
  if (!value || typeof value !== 'object') {
    return undefined
  }

  const parsed = value as {
    adoptDisplayName?: unknown
    adoptAvatar?: unknown
    adopt_display_name?: unknown
    adopt_avatar?: unknown
  }
  const decision: PendingRegistrationAdoptionDecision = {}

  if (typeof parsed.adoptDisplayName === 'boolean') {
    decision.adoptDisplayName = parsed.adoptDisplayName
  } else if (typeof parsed.adopt_display_name === 'boolean') {
    decision.adoptDisplayName = parsed.adopt_display_name
  }

  if (typeof parsed.adoptAvatar === 'boolean') {
    decision.adoptAvatar = parsed.adoptAvatar
  } else if (typeof parsed.adopt_avatar === 'boolean') {
    decision.adoptAvatar = parsed.adopt_avatar
  }

  return Object.keys(decision).length > 0 ? decision : undefined
}

function sanitizePendingRegistrationChallengeSummary(
  value: Partial<PendingRegistrationChallengeSummary> | null | undefined
): PendingRegistrationChallengeSummary | null {
  const ticket = typeof value?.ticket === 'string' ? value.ticket.trim() : ''
  const email = typeof value?.email === 'string' ? value.email.trim() : ''

  if (!ticket || !email) {
    return null
  }

    return {
      ticket,
      email,
      turnstile_token: typeof value?.turnstile_token === 'string' ? value.turnstile_token : undefined,
      invitation_code: typeof value?.invitation_code === 'string' ? value.invitation_code : undefined,
      aff_code: typeof value?.aff_code === 'string' ? value.aff_code : undefined,
      pending_auth_token: undefined,
      pending_auth_token_field: value?.pending_auth_token_field
        ? normalizePendingAuthTokenField(value.pending_auth_token_field)
        : undefined,
    pending_provider: typeof value?.pending_provider === 'string' ? value.pending_provider : undefined,
    pending_redirect: typeof value?.pending_redirect === 'string' ? value.pending_redirect : undefined,
    pending_adoption_decision: normalizePendingRegistrationAdoptionDecision(value?.pending_adoption_decision)
  }
}

function getPersistedPendingAuthSession(): PendingAuthSessionSummary | null {
  const raw = sessionStorage.getItem(PENDING_AUTH_SESSION_KEY)
  if (!raw) {
    return null
  }

  try {
    const parsed = JSON.parse(raw) as Partial<PersistedPendingAuthSessionSummary> | null
    const provider = typeof parsed?.provider === 'string' ? parsed.provider.trim() : ''
    if (!provider) {
      sessionStorage.removeItem(PENDING_AUTH_SESSION_KEY)
      return null
    }

    return {
      token: '',
      token_field: normalizePendingAuthTokenField(parsed?.token_field),
      provider,
      redirect: typeof parsed?.redirect === 'string' ? parsed.redirect : undefined,
      adoption_required: typeof parsed?.adoption_required === 'boolean' ? parsed.adoption_required : undefined,
      suggested_display_name: typeof parsed?.suggested_display_name === 'string' ? parsed.suggested_display_name : undefined,
      suggested_avatar_url: typeof parsed?.suggested_avatar_url === 'string' ? parsed.suggested_avatar_url : undefined
    }
  } catch {
    sessionStorage.removeItem(PENDING_AUTH_SESSION_KEY)
    return null
  }
}

function persistPendingAuthSession(session: PendingAuthSessionSummary): void {
  const safeSession: PersistedPendingAuthSessionSummary = {
    token_field: session.token_field,
    provider: session.provider,
    redirect: session.redirect,
    adoption_required: session.adoption_required,
    suggested_display_name: session.suggested_display_name,
    suggested_avatar_url: session.suggested_avatar_url
  }
  sessionStorage.setItem(PENDING_AUTH_SESSION_KEY, JSON.stringify(safeSession))
}

function clearPendingAuthSessionStorage(): void {
  sessionStorage.removeItem(PENDING_AUTH_SESSION_KEY)
}

function getPersistedPendingRegistrationChallenge(): PendingRegistrationChallengeSummary | null {
  const raw = sessionStorage.getItem(PENDING_REGISTRATION_CHALLENGE_KEY)
  if (!raw) {
    return null
  }

  try {
    const parsed = JSON.parse(raw) as Partial<PendingRegistrationChallengeSummary> | null
    const summary = sanitizePendingRegistrationChallengeSummary(parsed)
    if (!summary) {
      sessionStorage.removeItem(PENDING_REGISTRATION_CHALLENGE_KEY)
      return null
    }
    return summary
  } catch {
    sessionStorage.removeItem(PENDING_REGISTRATION_CHALLENGE_KEY)
    return null
  }
}

function persistPendingRegistrationChallenge(summary: PendingRegistrationChallengeSummary): void {
  const safeSummary: PersistedPendingRegistrationChallengeSummary = {
    ticket: summary.ticket,
    email: summary.email,
    turnstile_token: summary.turnstile_token,
    invitation_code: summary.invitation_code,
    aff_code: summary.aff_code,
    pending_auth_token_field: summary.pending_auth_token_field,
    pending_provider: summary.pending_provider,
    pending_redirect: summary.pending_redirect,
    pending_adoption_decision: summary.pending_adoption_decision
  }
  sessionStorage.setItem(PENDING_REGISTRATION_CHALLENGE_KEY, JSON.stringify(safeSummary))
}

function clearPendingRegistrationChallengeStorage(): void {
  sessionStorage.removeItem(PENDING_REGISTRATION_CHALLENGE_KEY)
}

function createPendingRegistrationTicket(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `reg_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)
  const refreshTokenValue = ref<string | null>(null)
  const tokenExpiresAt = ref<number | null>(null)
  const runMode = ref<'standard' | 'simple'>('standard')
  const isSessionRestoring = ref(false)
  const pendingAuthSession = ref<PendingAuthSessionSummary | null>(null)
  const pendingRegistrationChallenge = ref<PendingRegistrationChallengeSummary | null>(null)

  let refreshIntervalId: ReturnType<typeof setInterval> | null = null
  let tokenRefreshTimeoutId: ReturnType<typeof setTimeout> | null = null
  let restoreSessionPromise: Promise<User | null> | null = null

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isSimpleMode = computed(() => runMode.value === 'simple')
  const hasPendingAuthSession = computed(() => pendingAuthSession.value !== null)
  const hasPendingRegistrationChallenge = computed(
    () => pendingRegistrationChallenge.value !== null
  )

  function applyUserState(nextUser: User & { run_mode?: 'standard' | 'simple' }): User {
    if (nextUser.run_mode) {
      runMode.value = nextUser.run_mode
    }

    const { run_mode: _runMode, ...userData } = nextUser
    user.value = userData
    setPersistedAuthUser(userData)
    return userData
  }

  function keepPendingAuthSessionIfRequested(shouldPreserve: boolean): void {
    if (shouldPreserve) {
      pendingAuthSession.value = pendingAuthSession.value ?? getPersistedPendingAuthSession()
      return
    }

    pendingAuthSession.value = null
    clearPendingAuthSessionStorage()
  }

  function keepPendingRegistrationChallengeIfRequested(shouldPreserve: boolean): void {
    if (shouldPreserve) {
      pendingRegistrationChallenge.value =
        pendingRegistrationChallenge.value ?? getPersistedPendingRegistrationChallenge()
      return
    }

    pendingRegistrationChallenge.value = null
    pendingRegistrationPassword = null
    clearPendingRegistrationChallengeStorage()
  }

  function isSessionAuthFailure(error: unknown): boolean {
    const apiError = error as { status?: number; code?: string }
    return (
      apiError.status === 401
      || apiError.code === 'TOKEN_REFRESH_FAILED'
      || apiError.code === 'TOKEN_EXPIRED'
      || apiError.code === 'TOKEN_REVOKED'
      || apiError.code === 'INVALID_TOKEN'
    )
  }

  function clearAuthAfterSessionFailure(): void {
    clearAuth({
      preservePendingAuthSession: pendingAuthSession.value !== null,
      preservePendingRegistrationChallenge: pendingRegistrationChallenge.value !== null
    })
  }

  async function restoreSession(): Promise<User | null> {
    if (token.value && user.value) {
      return user.value
    }

    if (!hasPersistedSessionHint()) {
      return null
    }

    if (restoreSessionPromise) {
      return restoreSessionPromise
    }

    isSessionRestoring.value = true

    restoreSessionPromise = (async () => {
      try {
        const refreshedSession = await refreshAccessToken()
        token.value = refreshedSession.accessToken
        if (refreshedSession.refreshToken) {
          refreshTokenValue.value = refreshedSession.refreshToken
        }
        scheduleTokenRefresh(refreshedSession.expiresIn)
      } catch (error) {
        clearAuthAfterSessionFailure()
        throw error
      }

      try {
        const restoredUser = await refreshUser()
        clearPendingAuthSession()
        clearPendingRegistrationChallenge()
        startAutoRefresh()
        return restoredUser
      } catch (error) {
        if (isSessionAuthFailure(error)) {
          clearAuthAfterSessionFailure()
          throw error
        }

        console.error('Failed to refresh user while restoring session:', error)
        startAutoRefresh()
        return user.value
      } finally {
        isSessionRestoring.value = false
        restoreSessionPromise = null
      }
    })()

    return restoreSessionPromise
  }

  async function checkAuth(): Promise<void> {
    const savedUser = getPersistedAuthUser()
    const savedExpiresAt = getTokenExpiresAt()
    const savedAccessToken = getAccessToken()
    pendingAuthSession.value = getPersistedPendingAuthSession()
    pendingRegistrationChallenge.value = getPersistedPendingRegistrationChallenge()

    if (savedAccessToken) {
      token.value = savedAccessToken
    }

    if (!savedUser) {
      if (savedAccessToken) {
        try {
          await refreshUser()
          startAutoRefresh()
          if (savedExpiresAt !== null) {
            tokenExpiresAt.value = savedExpiresAt
            scheduleTokenRefreshAt(savedExpiresAt)
          }
        } catch (error) {
          console.error('Failed to restore user from saved access token:', error)
          if (isSessionAuthFailure(error)) {
            clearAuthAfterSessionFailure()
          }
        }
      }
      return
    }

    try {
      applyUserState(JSON.parse(savedUser) as User)
      tokenExpiresAt.value = savedExpiresAt
    } catch (error) {
      console.error('Failed to parse saved user data:', error)
      clearAuth({
        preservePendingAuthSession: true,
        preservePendingRegistrationChallenge: true
      })
      return
    }

    if (savedAccessToken) {
      clearPendingAuthSession()
      clearPendingRegistrationChallenge()

      try {
        await refreshUser()
      } catch (error) {
        console.error('Failed to refresh user on init:', error)
        if (isSessionAuthFailure(error)) {
          clearAuthAfterSessionFailure()
          return
        }
      }

      startAutoRefresh()

      if (tokenExpiresAt.value !== null) {
        scheduleTokenRefreshAt(tokenExpiresAt.value)
      }

      return
    }

    if (savedExpiresAt !== null) {
      try {
        await restoreSession()
      } catch (error) {
        console.error('Failed to restore session on init:', error)
      }
      return
    }

    clearAuth({
      preservePendingAuthSession: true,
      preservePendingRegistrationChallenge: true
    })
  }

  function startAutoRefresh(): void {
    stopAutoRefresh()

    refreshIntervalId = setInterval(() => {
      if (token.value) {
        refreshUser().catch((error) => {
          console.error('Auto-refresh user failed:', error)
        })
      }
    }, AUTO_REFRESH_INTERVAL)
  }

  function stopAutoRefresh(): void {
    if (refreshIntervalId) {
      clearInterval(refreshIntervalId)
      refreshIntervalId = null
    }
  }

  function scheduleTokenRefreshAt(expiresAtMs: number): void {
    if (tokenRefreshTimeoutId) {
      clearTimeout(tokenRefreshTimeoutId)
      tokenRefreshTimeoutId = null
    }

    const now = Date.now()
    const refreshInMs = Math.max(0, expiresAtMs - now - TOKEN_REFRESH_BUFFER)

    if (refreshInMs <= 0) {
      void performTokenRefresh()
      return
    }

    tokenRefreshTimeoutId = setTimeout(() => {
      void performTokenRefresh()
    }, refreshInMs)
  }

  function scheduleTokenRefresh(expiresInSeconds: number): void {
    const expiresAtMs = Date.now() + expiresInSeconds * 1000
    tokenExpiresAt.value = expiresAtMs
    setTokenExpiresAt(expiresAtMs)
    scheduleTokenRefreshAt(expiresAtMs)
  }

  async function performTokenRefresh(): Promise<void> {
    try {
      const refreshedSession = await refreshAccessToken()
      token.value = refreshedSession.accessToken
      if (refreshedSession.refreshToken) {
        refreshTokenValue.value = refreshedSession.refreshToken
      }
      scheduleTokenRefresh(refreshedSession.expiresIn)
    } catch (error) {
      console.error('Token refresh failed:', error)
    }
  }

  function stopTokenRefresh(): void {
    if (tokenRefreshTimeoutId) {
      clearTimeout(tokenRefreshTimeoutId)
      tokenRefreshTimeoutId = null
    }
  }

  async function login(credentials: LoginRequest): Promise<LoginResponse> {
    try {
      const response = await authAPI.login(credentials)

      if (isTotp2FARequired(response)) {
        return response
      }

      setAuthFromResponse(response)
      return response
    } catch (error) {
      clearAuth({
        preservePendingAuthSession: pendingAuthSession.value !== null,
        preservePendingRegistrationChallenge: pendingRegistrationChallenge.value !== null
      })
      throw error
    }
  }

  async function login2FA(tempToken: string, totpCode: string): Promise<User> {
    try {
      const response = await authAPI.login2FA({ temp_token: tempToken, totp_code: totpCode })
      setAuthFromResponse(response)
      return user.value!
    } catch (error) {
      clearAuth({
        preservePendingAuthSession: pendingAuthSession.value !== null,
        preservePendingRegistrationChallenge: pendingRegistrationChallenge.value !== null
      })
      throw error
    }
  }

  function setAuthFromResponse(response: AuthResponse): void {
    isSessionRestoring.value = false
    token.value = response.access_token

    if (response.refresh_token) {
      refreshTokenValue.value = response.refresh_token
    }

    setAccessToken(response.access_token)
    applyUserState(response.user)
    clearPendingAuthSession()
    clearPendingRegistrationChallenge()
    startAutoRefresh()

    if (response.expires_in) {
      scheduleTokenRefresh(response.expires_in)
    }
  }

  async function register(userData: RegisterRequest): Promise<User> {
    try {
      const response = await authAPI.register(userData)
      setAuthFromResponse(response)
      return user.value!
    } catch (error) {
      clearAuth({
        preservePendingAuthSession: pendingAuthSession.value !== null,
        preservePendingRegistrationChallenge: pendingRegistrationChallenge.value !== null
      })
      throw error
    }
  }

  async function setToken(newToken: string): Promise<User> {
    stopAutoRefresh()
    stopTokenRefresh()
    isSessionRestoring.value = false
    token.value = null
    user.value = null

    token.value = newToken
    setAccessToken(newToken)

    tokenExpiresAt.value = getTokenExpiresAt()

    try {
      const userData = await refreshUser()
      startAutoRefresh()

      if (tokenExpiresAt.value !== null) {
        scheduleTokenRefreshAt(tokenExpiresAt.value)
      }

      clearPendingAuthSession()
      clearPendingRegistrationChallenge()
      return userData
    } catch (error) {
      clearAuth({
        preservePendingAuthSession: pendingAuthSession.value !== null,
        preservePendingRegistrationChallenge: pendingRegistrationChallenge.value !== null
      })
      throw error
    }
  }

  function setPendingAuthSession(session: PendingAuthSessionSummary | null): void {
    pendingAuthSession.value = session

    if (session) {
      persistPendingAuthSession(session)
      return
    }

    clearPendingAuthSessionStorage()
  }

  function clearPendingAuthSession(): void {
    setPendingAuthSession(null)
  }

  function setPendingRegistrationChallenge(
    input: PendingRegistrationChallengeInput
  ): PendingRegistrationChallengeSummary {
    const summary: PendingRegistrationChallengeSummary = {
      ticket: createPendingRegistrationTicket(),
      email: input.email.trim(),
      turnstile_token: input.turnstile_token,
      invitation_code: input.invitation_code,
      aff_code: input.aff_code,
      pending_auth_token: input.pending_auth_token,
      pending_auth_token_field: input.pending_auth_token_field,
      pending_provider: input.pending_provider,
      pending_redirect: input.pending_redirect,
      pending_adoption_decision: input.pending_adoption_decision
    }

    pendingRegistrationChallenge.value = summary
    pendingRegistrationPassword = input.password
    persistPendingRegistrationChallenge(summary)
    return summary
  }

  function getPendingRegistrationChallengePayload():
    | PendingRegistrationChallengePayload
    | null {
    if (!pendingRegistrationChallenge.value || !pendingRegistrationPassword) {
      return null
    }

    return {
      ...pendingRegistrationChallenge.value,
      password: pendingRegistrationPassword
    }
  }

  function clearPendingRegistrationChallenge(): void {
    pendingRegistrationChallenge.value = null
    pendingRegistrationPassword = null
    clearPendingRegistrationChallengeStorage()
  }

  async function logout(): Promise<void> {
    await authAPI.logout()
    clearAuth()
  }

  async function refreshUser(): Promise<User> {
    if (!token.value) {
      try {
        await restoreSession()
      } catch {
        // Fall through to a normalized "not authenticated" error below.
      }
    }

    if (!token.value) {
      throw new Error('Not authenticated')
    }

    try {
      const response = await authAPI.getCurrentUser()
      return applyUserState(response.data)
    } catch (error) {
      if ((error as { status?: number }).status === 401) {
        clearAuth({
          preservePendingAuthSession: pendingAuthSession.value !== null,
          preservePendingRegistrationChallenge: pendingRegistrationChallenge.value !== null
        })
      }
      throw error
    }
  }

  function clearAuth(options?: {
    preservePendingAuthSession?: boolean
    preservePendingRegistrationChallenge?: boolean
  }): void {
    stopAutoRefresh()
    stopTokenRefresh()

    isSessionRestoring.value = false
    restoreSessionPromise = null
    token.value = null
    refreshTokenValue.value = null
    tokenExpiresAt.value = null
    user.value = null
    clearPersistedSession()

    keepPendingAuthSessionIfRequested(options?.preservePendingAuthSession === true)
    keepPendingRegistrationChallengeIfRequested(
      options?.preservePendingRegistrationChallenge === true
    )
  }

  return {
    user,
    token,
    runMode: readonly(runMode),
    pendingAuthSession: readonly(pendingAuthSession),
    pendingRegistrationChallenge: readonly(pendingRegistrationChallenge),

    isAuthenticated,
    isAdmin,
    isSimpleMode,
    hasPendingAuthSession,
    hasPendingRegistrationChallenge,

    login,
    login2FA,
    register,
    setToken,
    logout,
    checkAuth,
    refreshUser,
    setPendingAuthSession,
    clearPendingAuthSession,
    setPendingRegistrationChallenge,
    getPendingRegistrationChallengePayload,
    clearPendingRegistrationChallenge
  }
})
