/**
 * Shared URL builder for iframe-embedded pages.
 * Keeps only non-sensitive context in outbound iframe/new-tab URLs.
 */

const EMBEDDED_USER_ID_QUERY_KEY = 'user_id'
const EMBEDDED_THEME_QUERY_KEY = 'theme'
const EMBEDDED_LANG_QUERY_KEY = 'lang'
const EMBEDDED_UI_MODE_QUERY_KEY = 'ui_mode'
const EMBEDDED_UI_MODE_VALUE = 'embedded'
const EMBEDDED_SRC_HOST_QUERY_KEY = 'src_host'
const EMBEDDED_SRC_QUERY_KEY = 'src_url'
const SENSITIVE_SOURCE_QUERY_KEY_PARTS = [
  'token',
  'secret',
  'session',
  'jwt',
  'signature',
  'authorization',
]
const SENSITIVE_SOURCE_QUERY_KEYS = new Set(['code', 'state', 'sig'])

function isSensitiveSourceQueryKey(key: string): boolean {
  const normalized = key.trim().toLowerCase()
  if (!normalized) return true
  if (SENSITIVE_SOURCE_QUERY_KEYS.has(normalized)) return true
  return SENSITIVE_SOURCE_QUERY_KEY_PARTS.some((part) => normalized.includes(part))
}

function sanitizeCurrentLocationHref(rawHref: string): string | null {
  try {
    const currentUrl = new URL(rawHref)
    const sanitizedUrl = new URL(currentUrl.origin + currentUrl.pathname)
    currentUrl.searchParams.forEach((value, key) => {
      if (!isSensitiveSourceQueryKey(key)) {
        sanitizedUrl.searchParams.append(key, value)
      }
    })
    return sanitizedUrl.toString()
  } catch {
    return null
  }
}

export function buildEmbeddedUrl(
  baseUrl: string,
  userId?: number,
  theme: 'light' | 'dark' = 'light',
  lang?: string,
): string {
  if (!baseUrl) return baseUrl
  try {
    const url = new URL(baseUrl)
    if (userId) {
      url.searchParams.set(EMBEDDED_USER_ID_QUERY_KEY, String(userId))
    }
    url.searchParams.set(EMBEDDED_THEME_QUERY_KEY, theme)
    if (lang) {
      url.searchParams.set(EMBEDDED_LANG_QUERY_KEY, lang)
    }
    url.searchParams.set(EMBEDDED_UI_MODE_QUERY_KEY, EMBEDDED_UI_MODE_VALUE)
    // Source tracking: let the embedded page know where it's being loaded from
    if (typeof window !== 'undefined') {
      url.searchParams.set(EMBEDDED_SRC_HOST_QUERY_KEY, window.location.origin)
      const sanitizedSourceUrl = sanitizeCurrentLocationHref(window.location.href)
      if (sanitizedSourceUrl) {
        url.searchParams.set(EMBEDDED_SRC_QUERY_KEY, sanitizedSourceUrl)
      }
    }
    return url.toString()
  } catch {
    return baseUrl
  }
}

export function detectTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'light'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}
