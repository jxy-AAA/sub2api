import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { clearAccessToken, getAccessToken, setAccessToken } from '@/api/tokenStorage'

const mockLogin = vi.fn()
const mockLogin2FA = vi.fn()
const mockLogout = vi.fn()
const mockGetCurrentUser = vi.fn()
const mockRegister = vi.fn()
const mockRefreshToken = vi.fn()
const mockRefreshAccessToken = vi.fn()

vi.mock('@/api', () => ({
  authAPI: {
    login: (...args: any[]) => mockLogin(...args),
    login2FA: (...args: any[]) => mockLogin2FA(...args),
    logout: (...args: any[]) => mockLogout(...args),
    getCurrentUser: (...args: any[]) => mockGetCurrentUser(...args),
    register: (...args: any[]) => mockRegister(...args),
    refreshToken: (...args: any[]) => mockRefreshToken(...args),
  },
  isTotp2FARequired: (response: any) => response?.requires_2fa === true,
}))

vi.mock('@/api/client', () => ({
  refreshAccessToken: (...args: any[]) => mockRefreshAccessToken(...args),
}))

const fakeUser = {
  id: 1,
  username: 'testuser',
  email: 'test@example.com',
  role: 'user' as const,
  balance: 100,
  concurrency: 5,
  status: 'active' as const,
  allowed_groups: null,
  created_at: '2024-01-01',
  updated_at: '2024-01-01',
}

const fakeAdminUser = {
  ...fakeUser,
  id: 2,
  username: 'admin',
  email: 'admin@example.com',
  role: 'admin' as const,
}

const fakeAuthResponse = {
  access_token: 'test-token-123',
  refresh_token: 'refresh-token-456',
  expires_in: 3600,
  token_type: 'Bearer',
  user: { ...fakeUser },
}

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    sessionStorage.clear()
    clearAccessToken()
    vi.useFakeTimers()
    vi.clearAllMocks()
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  describe('login', () => {
    it('sets token and user after a successful login', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.token).toBe('test-token-123')
      expect(store.user).toEqual(fakeUser)
      expect(store.isAuthenticated).toBe(true)
      expect(getAccessToken()).toBe('test-token-123')
      expect(localStorage.getItem('auth_token')).toBeNull()
      expect(localStorage.getItem('auth_user')).toBe(JSON.stringify(fakeUser))
    })

    it('clears auth state when login fails', async () => {
      mockLogin.mockRejectedValue(new Error('Invalid credentials'))
      const store = useAuthStore()

      await expect(store.login({ email: 'test@example.com', password: 'wrong' })).rejects.toThrow(
        'Invalid credentials'
      )

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('returns a 2FA challenge without authenticating the user', async () => {
      const twoFactorResponse = { requires_2fa: true, temp_token: 'temp-123' }
      mockLogin.mockResolvedValue(twoFactorResponse)
      const store = useAuthStore()

      const result = await store.login({ email: 'test@example.com', password: '123456' })

      expect(result).toEqual(twoFactorResponse)
      expect(store.token).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('login2FA', () => {
    it('authenticates the user after a successful 2FA login', async () => {
      mockLogin2FA.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()

      const user = await store.login2FA('temp-123', '654321')

      expect(store.token).toBe('test-token-123')
      expect(store.user).toEqual(fakeUser)
      expect(user).toEqual(fakeUser)
      expect(mockLogin2FA).toHaveBeenCalledWith({
        temp_token: 'temp-123',
        totp_code: '654321',
      })
    })

    it('clears auth state when 2FA verification fails', async () => {
      mockLogin2FA.mockRejectedValue(new Error('Invalid TOTP'))
      const store = useAuthStore()

      await expect(store.login2FA('temp-123', '000000')).rejects.toThrow('Invalid TOTP')
      expect(store.token).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('logout', () => {
    it('clears persisted auth state on logout', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      mockLogout.mockResolvedValue(undefined)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })
      expect(store.isAuthenticated).toBe(true)

      await store.logout()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem('auth_token')).toBeNull()
      expect(localStorage.getItem('auth_user')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
      expect(localStorage.getItem('token_expires_at')).toBeNull()
    })
  })

  describe('checkAuth', () => {
    it('restores persisted auth state when an access token is still available', async () => {
      setAccessToken('saved-token')
      localStorage.setItem('auth_user', JSON.stringify(fakeUser))
      mockGetCurrentUser.mockResolvedValue({ data: fakeUser })

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.token).toBe('saved-token')
      expect(store.user).toEqual(fakeUser)
      expect(store.isAuthenticated).toBe(true)
    })

    it('keeps the store unauthenticated when nothing is persisted', async () => {
      const store = useAuthStore()
      await store.checkAuth()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('clears auth state when persisted user data is invalid', async () => {
      setAccessToken('saved-token')
      localStorage.setItem('auth_user', 'invalid-json{{{')

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(localStorage.getItem('auth_token')).toBeNull()
    })

    it('restores token expiry metadata from persistence', async () => {
      const futureTs = String(Date.now() + 3600_000)
      setAccessToken('saved-token')
      localStorage.setItem('auth_user', JSON.stringify(fakeUser))
      localStorage.setItem('token_expires_at', futureTs)
      mockGetCurrentUser.mockResolvedValue({ data: fakeUser })

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.isAuthenticated).toBe(true)
    })

    it('restores pending auth metadata without restoring a pending token', async () => {
      sessionStorage.setItem(
        'pending_auth_session',
        JSON.stringify({
          token: 'pending-token',
          token_field: 'pending_auth_token',
          provider: 'wechat',
          redirect: '/profile',
        })
      )

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.hasPendingAuthSession).toBe(true)
      expect(store.pendingAuthSession).toEqual({
        token: '',
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
        adoption_required: undefined,
        suggested_display_name: undefined,
        suggested_avatar_url: undefined,
      })
    })

    it('restores the access token through refresh when only session hints remain', async () => {
      localStorage.setItem('auth_user', JSON.stringify(fakeUser))
      localStorage.setItem('token_expires_at', String(Date.now() + 3600_000))
      mockRefreshAccessToken.mockImplementation(async () => {
        setAccessToken('restored-token')
        return {
          accessToken: 'restored-token',
          expiresIn: 3600,
        }
      })
      mockGetCurrentUser.mockResolvedValue({ data: fakeUser })

      const store = useAuthStore()
      await store.checkAuth()

      expect(mockRefreshAccessToken).toHaveBeenCalledTimes(1)
      expect(store.token).toBe('restored-token')
      expect(getAccessToken()).toBe('restored-token')
      expect(store.user).toEqual(fakeUser)
      expect(store.isAuthenticated).toBe(true)
    })

    it('clears persisted auth state when restore-via-refresh fails', async () => {
      localStorage.setItem('auth_user', JSON.stringify(fakeUser))
      localStorage.setItem('token_expires_at', String(Date.now() + 3600_000))
      mockRefreshAccessToken.mockRejectedValue(new Error('refresh failed'))

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem('auth_user')).toBeNull()
      expect(localStorage.getItem('token_expires_at')).toBeNull()
    })
  })

  describe('pending auth session', () => {
    it('persists and clears pending auth session state', () => {
      const store = useAuthStore()

      store.setPendingAuthSession({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
      })

      expect(store.hasPendingAuthSession).toBe(true)
      expect(localStorage.getItem('pending_auth_session')).toBeNull()
      expect(store.pendingAuthSession).toEqual({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
      })

      const persisted = JSON.parse(sessionStorage.getItem('pending_auth_session') || 'null')
      expect(persisted).toEqual({
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
      })
      expect(sessionStorage.getItem('pending_auth_session')).not.toContain('pending-token')

      store.clearPendingAuthSession()

      expect(store.hasPendingAuthSession).toBe(false)
      expect(sessionStorage.getItem('pending_auth_session')).toBeNull()
    })

    it('restores a persisted pending oauth session without requiring a token value', async () => {
      const firstStore = useAuthStore()

      firstStore.setPendingAuthSession({
        token: '',
        token_field: 'pending_oauth_token',
        provider: 'oidc',
        redirect: '/welcome',
        adoption_required: true,
        suggested_display_name: 'OIDC Nick',
      })

      setActivePinia(createPinia())
      const restoredStore = useAuthStore()
      await restoredStore.checkAuth()

      expect(restoredStore.isAuthenticated).toBe(false)
      expect(restoredStore.hasPendingAuthSession).toBe(true)
      expect(restoredStore.pendingAuthSession).toEqual({
        token: '',
        token_field: 'pending_oauth_token',
        provider: 'oidc',
        redirect: '/welcome',
        adoption_required: true,
        suggested_display_name: 'OIDC Nick',
        suggested_avatar_url: undefined,
      })
    })

    it('preserves the pending auth session when registration fails', async () => {
      const store = useAuthStore()
      store.setPendingAuthSession({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'oidc',
        redirect: '/register',
      })
      mockRegister.mockRejectedValue(new Error('Register failed'))

      await expect(
        store.register({ email: 'user@example.com', password: 'secret-123' })
      ).rejects.toThrow('Register failed')

      expect(store.hasPendingAuthSession).toBe(true)
      expect(store.pendingAuthSession).toEqual({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'oidc',
        redirect: '/register',
      })
    })

    it('stages a pending registration challenge without persisting the password', () => {
      const store = useAuthStore()

      store.setPendingRegistrationChallenge({
        email: 'user@example.com',
        password: 'secret-123',
        turnstile_token: 'turnstile-token',
        invitation_code: 'INVITE123',
        aff_code: 'AFF123',
        pending_auth_token: 'pending-token',
        pending_auth_token_field: 'pending_oauth_token',
      })

      expect(store.hasPendingRegistrationChallenge).toBe(true)
      expect(localStorage.getItem('pending_registration_challenge')).toBeNull()
      expect(
        JSON.parse(sessionStorage.getItem('pending_registration_challenge') || 'null')
      ).toMatchObject({
        email: 'user@example.com',
        turnstile_token: 'turnstile-token',
        invitation_code: 'INVITE123',
        aff_code: 'AFF123',
      })
      expect(sessionStorage.getItem('pending_registration_challenge')).not.toContain('secret-123')
      expect(sessionStorage.getItem('pending_registration_challenge')).not.toContain('pending-token')
      expect(store.getPendingRegistrationChallengePayload()).toMatchObject({
        email: 'user@example.com',
        password: 'secret-123',
        turnstile_token: 'turnstile-token',
        invitation_code: 'INVITE123',
        aff_code: 'AFF123',
        pending_auth_token: 'pending-token',
        pending_auth_token_field: 'pending_oauth_token',
      })
    })

    it('clears the pending registration challenge after a successful login', async () => {
      const store = useAuthStore()
      store.setPendingRegistrationChallenge({
        email: 'user@example.com',
        password: 'secret-123',
      })
      mockLogin.mockResolvedValue(fakeAuthResponse)

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.hasPendingRegistrationChallenge).toBe(false)
      expect(store.getPendingRegistrationChallengePayload()).toBeNull()
      expect(sessionStorage.getItem('pending_registration_challenge')).toBeNull()
    })
  })

  describe('isAdmin', () => {
    it('returns true for admin users', async () => {
      const adminResponse = { ...fakeAuthResponse, user: { ...fakeAdminUser } }
      mockLogin.mockResolvedValue(adminResponse)
      const store = useAuthStore()

      await store.login({ email: 'admin@example.com', password: '123456' })

      expect(store.isAdmin).toBe(true)
    })

    it('returns false for regular users', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.isAdmin).toBe(false)
    })

    it('returns false when unauthenticated', () => {
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })
  })

  describe('refreshUser', () => {
    it('refreshes the user profile and updates persistence', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()
      await store.login({ email: 'test@example.com', password: '123456' })

      const updatedUser = { ...fakeUser, username: 'updated-name' }
      mockGetCurrentUser.mockResolvedValue({ data: updatedUser })

      const result = await store.refreshUser()

      expect(result).toEqual(updatedUser)
      expect(store.user).toEqual(updatedUser)
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(updatedUser)
    })

    it('throws when the user is not authenticated', async () => {
      const store = useAuthStore()
      await expect(store.refreshUser()).rejects.toThrow('Not authenticated')
    })
  })

  describe('isSimpleMode', () => {
    it('returns true when run_mode is simple', async () => {
      const simpleResponse = {
        ...fakeAuthResponse,
        user: { ...fakeUser, run_mode: 'simple' as const },
      }
      mockLogin.mockResolvedValue(simpleResponse)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.isSimpleMode).toBe(true)
    })

    it('defaults to standard mode', () => {
      const store = useAuthStore()
      expect(store.isSimpleMode).toBe(false)
    })
  })
})
