/**
 * Axios HTTP Client Configuration
 * Base client with interceptors for authentication, token refresh, and error handling
 */

import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import { getLocale } from '@/i18n'
import { navigateWithinApp } from '@/utils/spaNavigation'
import {
  clearPersistedSession,
  getAccessToken,
  hasPersistedSessionHint,
  setAccessToken,
  setTokenExpiresAt,
} from './tokenStorage'

// ==================== Axios Instance Configuration ====================

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// ==================== Token Refresh State ====================

export interface RefreshAccessTokenResult {
  accessToken: string
  refreshToken?: string
  expiresIn: number
}

let refreshPromise: Promise<RefreshAccessTokenResult> | null = null

function isAuthEndpoint(url: string): boolean {
  return (
    url.includes('/auth/login')
    || url.includes('/auth/register')
    || url.includes('/auth/refresh')
  )
}

function shouldAttemptTokenRefresh(url: string): boolean {
  if (isAuthEndpoint(url)) {
    return false
  }

  return Boolean(getAccessToken()) || hasPersistedSessionHint()
}

export async function refreshAccessToken(): Promise<RefreshAccessTokenResult> {
  if (refreshPromise) {
    return refreshPromise
  }

  refreshPromise = (async () => {
    const refreshResponse = await axios.post(
      `${API_BASE_URL}/auth/refresh`,
      {},
      { withCredentials: true, headers: { 'Content-Type': 'application/json' } }
    )

    const refreshData = refreshResponse.data as ApiResponse<{
      access_token: string
      refresh_token?: string
      expires_in: number
    }>

    if (refreshData.code !== 0 || !refreshData.data?.access_token || !refreshData.data.expires_in) {
      throw new Error('Token refresh failed')
    }

    const result: RefreshAccessTokenResult = {
      accessToken: refreshData.data.access_token,
      refreshToken: refreshData.data.refresh_token,
      expiresIn: refreshData.data.expires_in
    }

    setAccessToken(result.accessToken)
    setTokenExpiresAt(Date.now() + result.expiresIn * 1000)
    return result
  })()

  try {
    return await refreshPromise
  } finally {
    refreshPromise = null
  }
}

// ==================== Request Interceptor ====================

// Get user's timezone
const getUserTimezone = (): string => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

apiClient.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const requestUrl = String(config.url || '')

    if (refreshPromise && !isAuthEndpoint(requestUrl)) {
      try {
        await refreshPromise
      } catch {
        // Let the request continue and fail normally if refresh did not succeed.
      }
    }

    // Attach token from memory-first storage
    const token = getAccessToken()
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // Attach locale for backend translations
    if (config.headers) {
      config.headers['Accept-Language'] = getLocale()
    }

    // Attach timezone for all GET requests (backend may use it for default date ranges)
    if (config.method === 'get') {
      if (!config.params) {
        config.params = {}
      }
      config.params.timezone = getUserTimezone()
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// ==================== Response Interceptor ====================

apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    // Unwrap standard API response format { code, message, data }
    const apiResponse = response.data as ApiResponse<unknown>
    if (apiResponse && typeof apiResponse === 'object' && 'code' in apiResponse) {
      if (apiResponse.code === 0) {
        // Success - return the data portion
        response.data = apiResponse.data
      } else {
        // API error
        const resp = apiResponse as unknown as Record<string, unknown>
        return Promise.reject({
          status: response.status,
          code: apiResponse.code,
          message: apiResponse.message || 'Unknown error',
          reason: resp.reason,
          metadata: resp.metadata,
        })
      }
    }
    return response
  },
  async (error: AxiosError<ApiResponse<unknown>>) => {
    // Request cancellation: keep the original axios cancellation error so callers can ignore it.
    // Otherwise we'd misclassify it as a generic "network error".
    if (error.code === 'ERR_CANCELED' || axios.isCancel(error)) {
      return Promise.reject(error)
    }

    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    // Handle common errors
    if (error.response) {
      const { status, data } = error.response
      const url = String(error.config?.url || '')

      // Validate `data` shape to avoid HTML error pages breaking our error handling.
      const apiData = (typeof data === 'object' && data !== null ? data : {}) as Record<string, any>

      // Ops monitoring disabled: treat as feature-flagged 404, and proactively redirect away
      // from ops pages to avoid broken UI states.
      if (status === 404 && apiData.message === 'Ops monitoring is disabled') {
        try {
          localStorage.setItem('ops_monitoring_enabled_cached', 'false')
        } catch {
          // ignore localStorage failures
        }
        try {
          window.dispatchEvent(new CustomEvent('ops-monitoring-disabled'))
        } catch {
          // ignore event failures
        }

        if (window.location.pathname.startsWith('/admin/ops')) {
          await navigateWithinApp('/admin/settings', { replace: true })
        }

        return Promise.reject({
          status,
          code: 'OPS_DISABLED',
          message: apiData.message || error.message,
          url
        })
      }

      // 401: Try to refresh the token via refresh cookie
      // This handles TOKEN_EXPIRED, INVALID_TOKEN, TOKEN_REVOKED, etc.
      if (status === 401 && !originalRequest._retry) {
        const shouldRefresh = shouldAttemptTokenRefresh(url)

        if (shouldRefresh) {
          originalRequest._retry = true

          try {
            const refreshedSession = await refreshAccessToken()

            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${refreshedSession.accessToken}`
            }

            return apiClient(originalRequest)
          } catch {
            clearPersistedSession()
            sessionStorage.setItem('auth_expired', '1')

            if (!window.location.pathname.includes('/login')) {
              await navigateWithinApp('/login', { replace: true })
            }

            return Promise.reject({
              status: 401,
              code: 'TOKEN_REFRESH_FAILED',
              message: 'Session expired. Please log in again.'
            })
          }
        }

        // No refresh token or is auth endpoint - clear auth and redirect
        const hasToken = !!getAccessToken()
        const headers = error.config?.headers as Record<string, unknown> | undefined
        const authHeader = headers?.Authorization ?? headers?.authorization
        const sentAuth =
          typeof authHeader === 'string'
            ? authHeader.trim() !== ''
            : Array.isArray(authHeader)
              ? authHeader.length > 0
              : !!authHeader

        clearPersistedSession()
        if ((hasToken || sentAuth) && !isAuthEndpoint(url)) {
          sessionStorage.setItem('auth_expired', '1')
        }
        // Only redirect if not already on login page
        if (!window.location.pathname.includes('/login')) {
          await navigateWithinApp('/login', { replace: true })
        }
      }

      // Return structured error
      return Promise.reject({
        status,
        code: apiData.code,
        reason: apiData.reason,
        error: apiData.error,
        message: apiData.message || apiData.detail || error.message,
        metadata: apiData.metadata,
      })
    }

    // Network error
    return Promise.reject({
      status: 0,
      message: 'Network error. Please check your connection.'
    })
  }
)

export default apiClient
