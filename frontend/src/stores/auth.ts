/**
 * Authentication Store
 * Manages user authentication state, login/logout, token refresh, and token persistence
 */

import { defineStore } from 'pinia'
import { computed, readonly, ref } from 'vue'
import { authAPI, isTotp2FARequired, type LoginResponse } from '@/api'
import { clearAccessToken, getAccessToken, setAccessToken } from '@/api/tokenStorage'
import type { AuthResponse, LoginRequest, RegisterRequest, User } from '@/types'

const AUTH_USER_KEY = 'auth_user'
const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'
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
  const pendingAuthSession = ref<PendingAuthSessionSummary | null>(null)
  const pendingRegistrationChallenge = ref<PendingRegistrationChallengeSummary | null>(null)

  let refreshIntervalId: ReturnType<typeof setInterval> | null = null
  let tokenRefreshTimeoutId: ReturnType<typeof setTimeout> | null = null

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isSimpleMode = computed(() => runMode.value === 'simple')
  const hasPendingAuthSession = computed(() => pendingAuthSession.value !== null)
  const hasPendingRegistrationChallenge = computed(
    () => pendingRegistrationChallenge.value !== null
  )

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

  function checkAuth(): void {
    const savedUser = localStorage.getItem(AUTH_USER_KEY)
    const savedExpiresAt = localStorage.getItem(TOKEN_EXPIRES_AT_KEY)
    const savedAccessToken = getAccessToken()
    pendingAuthSession.value = getPersistedPendingAuthSession()
    pendingRegistrationChallenge.value = getPersistedPendingRegistrationChallenge()

    if (savedAccessToken) {
      token.value = savedAccessToken
    }

    if (savedUser && savedAccessToken) {
      try {
        user.value = JSON.parse(savedUser)
        tokenExpiresAt.value = savedExpiresAt ? parseInt(savedExpiresAt, 10) : null
        clearPendingAuthSession()
        clearPendingRegistrationChallenge()

        refreshUser().catch((error) => {
          console.error('Failed to refresh user on init:', error)
        })
        startAutoRefresh()

        if (tokenExpiresAt.value !== null) {
          scheduleTokenRefreshAt(tokenExpiresAt.value)
        }
      } catch (error) {
        console.error('Failed to parse saved user data:', error)
        clearAuth({
          preservePendingAuthSession: true,
          preservePendingRegistrationChallenge: true
        })
      }
    }
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
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, String(expiresAtMs))
    scheduleTokenRefreshAt(expiresAtMs)
  }

  async function performTokenRefresh(): Promise<void> {
    try {
      const response = await authAPI.refreshToken()
      token.value = response.access_token
      if (response.refresh_token) {
        refreshTokenValue.value = response.refresh_token
      }
      scheduleTokenRefresh(response.expires_in)
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
    token.value = response.access_token

    if (response.refresh_token) {
      refreshTokenValue.value = response.refresh_token
    }

    if (response.user.run_mode) {
      runMode.value = response.user.run_mode
    }

    const { run_mode: _run_mode, ...userData } = response.user
    user.value = userData

    setAccessToken(response.access_token)
    localStorage.setItem(AUTH_USER_KEY, JSON.stringify(userData))
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
    token.value = null
    user.value = null

    token.value = newToken
    setAccessToken(newToken)

    const savedExpiresAt = localStorage.getItem(TOKEN_EXPIRES_AT_KEY)
    if (savedExpiresAt) {
      tokenExpiresAt.value = parseInt(savedExpiresAt, 10)
    }

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
      throw new Error('Not authenticated')
    }

    try {
      const response = await authAPI.getCurrentUser()
      if (response.data.run_mode) {
        runMode.value = response.data.run_mode
      }
      const { run_mode: _run_mode, ...userData } = response.data
      user.value = userData
      localStorage.setItem(AUTH_USER_KEY, JSON.stringify(userData))
      return userData
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

    token.value = null
    refreshTokenValue.value = null
    tokenExpiresAt.value = null
    user.value = null
    clearAccessToken()
    localStorage.removeItem(AUTH_USER_KEY)
    localStorage.removeItem('refresh_token')
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY)

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
