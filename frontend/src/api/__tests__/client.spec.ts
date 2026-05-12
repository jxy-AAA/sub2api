import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import axios from 'axios'
import type { AxiosInstance } from 'axios'

const { navigateWithinAppMock } = vi.hoisted(() => ({
  navigateWithinAppMock: vi.fn().mockResolvedValue(true)
}))

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN',
}))

vi.mock('@/utils/spaNavigation', () => ({
  navigateWithinApp: navigateWithinAppMock
}))

describe('apiClient', () => {
  let apiClient: AxiosInstance
  let refreshAccessToken: typeof import('@/api/client').refreshAccessToken

  beforeEach(async () => {
    localStorage.clear()
    sessionStorage.clear()
    vi.resetModules()
    navigateWithinAppMock.mockClear()

    const tokenStorage = await import('@/api/tokenStorage')
    tokenStorage.clearPersistedSession()

    const clientModule = await import('@/api/client')
    apiClient = clientModule.apiClient
    refreshAccessToken = clientModule.refreshAccessToken
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('request interceptor', () => {
    it('attaches the Authorization header when a token exists', async () => {
      const tokenStorage = await import('@/api/tokenStorage')
      tokenStorage.setAccessToken('my-jwt-token')

      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBe('Bearer my-jwt-token')
    })

    it('does not attach Authorization without a token', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBeFalsy()
    })

    it('adds timezone params to GET requests', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.params).toHaveProperty('timezone')
    })

    it('does not add timezone params to POST requests', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/test', { foo: 'bar' })

      const config = adapter.mock.calls[0][0]
      expect(config.params?.timezone).toBeUndefined()
    })

    it('keeps withCredentials enabled for cookie-backed auth', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/auth/oauth/bind-token')

      const config = adapter.mock.calls[0][0]
      expect(config.withCredentials).toBe(true)
    })
  })

  describe('response interceptor', () => {
    it('unwraps the data field when code is 0', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: { name: 'test' }, message: 'ok' },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      const response = await apiClient.get('/test')
      expect(response.data).toEqual({ name: 'test' })
    })

    it('rejects with a structured error when code is not 0', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 1001, message: '鍙傛暟閿欒', data: null },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toEqual(
        expect.objectContaining({
          code: 1001,
          message: '鍙傛暟閿欒',
        })
      )
    })

    it('uses internal SPA navigation when ops pages are disabled', async () => {
      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 404,
          data: { message: 'Ops monitoring is disabled' },
        },
        config: {
          url: '/admin/ops/config',
          headers: {},
        },
      })
      apiClient.defaults.adapter = adapter

      const originalLocation = window.location
      Object.defineProperty(window, 'location', {
        value: { ...originalLocation, pathname: '/admin/ops' },
        writable: true,
      })

      await expect(apiClient.get('/admin/ops/config')).rejects.toEqual(
        expect.objectContaining({
          code: 'OPS_DISABLED',
        })
      )

      expect(navigateWithinAppMock).toHaveBeenCalledWith('/admin/settings', { replace: true })

      Object.defineProperty(window, 'location', {
        value: originalLocation,
        writable: true,
      })
    })
  })

  describe('token refresh', () => {
    it('retries a protected request after refreshing from persisted session hints', async () => {
      localStorage.setItem('auth_user', JSON.stringify({ id: 1, role: 'user' }))
      localStorage.setItem('token_expires_at', String(Date.now() + 3600_000))

      const refreshSpy = vi.spyOn(axios, 'post').mockResolvedValue({
        data: {
          code: 0,
          data: {
            access_token: 'refreshed-token',
            expires_in: 3600,
          },
        },
      } as any)

      const adapter = vi
        .fn()
        .mockRejectedValueOnce({
          response: {
            status: 401,
            data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
          },
          config: {
            url: '/protected',
            headers: {},
          },
          code: 'ERR_BAD_REQUEST',
        })
        .mockResolvedValueOnce({
          status: 200,
          data: { code: 0, data: { ok: true } },
          headers: {},
          config: {},
          statusText: 'OK',
        })
      apiClient.defaults.adapter = adapter

      const response = await apiClient.get('/protected')

      expect(response.data).toEqual({ ok: true })
      expect(refreshSpy).toHaveBeenCalledTimes(1)
      expect(adapter).toHaveBeenCalledTimes(2)
      const retriedConfig = adapter.mock.calls[1][0]
      expect(retriedConfig.headers.get('Authorization')).toBe('Bearer refreshed-token')
    })

    it('clears persisted auth state when refresh fails', async () => {
      localStorage.setItem('auth_user', JSON.stringify({ id: 1, role: 'user' }))
      localStorage.setItem('token_expires_at', String(Date.now() + 3600_000))
      vi.spyOn(axios, 'post').mockRejectedValue(new Error('refresh failed'))

      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config: {
          url: '/protected',
          headers: {},
        },
        code: 'ERR_BAD_REQUEST',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/protected')).rejects.toEqual(
        expect.objectContaining({
          code: 'TOKEN_REFRESH_FAILED',
        })
      )

      expect(localStorage.getItem('auth_user')).toBeNull()
      expect(localStorage.getItem('token_expires_at')).toBeNull()
      expect(navigateWithinAppMock).toHaveBeenCalledWith('/login', { replace: true })
    })

    it('deduplicates concurrent refresh calls', async () => {
      let resolveRefresh: ((value: any) => void) | null = null
      const refreshSpy = vi.spyOn(axios, 'post').mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveRefresh = resolve
          }) as any
      )

      const firstCall = refreshAccessToken()
      const secondCall = refreshAccessToken()

      expect(refreshSpy).toHaveBeenCalledTimes(1)

      resolveRefresh?.({
        data: {
          code: 0,
          data: {
            access_token: 'deduped-token',
            expires_in: 3600,
          },
        },
      })

      const [firstResult, secondResult] = await Promise.all([firstCall, secondCall])

      expect(firstResult).toEqual({
        accessToken: 'deduped-token',
        refreshToken: undefined,
        expiresIn: 3600,
      })
      expect(secondResult).toEqual(firstResult)
    })
  })

  describe('network errors', () => {
    it('maps network errors to status 0', async () => {
      const adapter = vi.fn().mockRejectedValue({
        code: 'ERR_NETWORK',
        message: 'Network Error',
        config: { url: '/test' },
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toEqual(
        expect.objectContaining({
          status: 0,
          message: 'Network error. Please check your connection.',
        })
      )
    })
  })

  describe('request cancellation', () => {
    it('preserves axios cancellation errors', async () => {
      const source = axios.CancelToken.source()
      const adapter = vi.fn().mockRejectedValue(new axios.Cancel('Operation canceled'))
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test', { cancelToken: source.token })).rejects.toBeDefined()
    })
  })
})
