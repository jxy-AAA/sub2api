export type AdminOAuthRedirectProvider =
  | 'linuxdo'
  | 'github'
  | 'google'
  | 'wechat'
  | 'oidc'

const redirectPathByProvider: Record<AdminOAuthRedirectProvider, string> = {
  linuxdo: '/api/v1/auth/oauth/linuxdo/callback',
  github: '/api/v1/auth/oauth/github/callback',
  google: '/api/v1/auth/oauth/google/callback',
  wechat: '/api/v1/auth/oauth/wechat/callback',
  oidc: '/api/v1/auth/oauth/oidc/callback'
}

export function resolveBrowserOrigin(): string {
  if (typeof window === 'undefined') {
    return ''
  }

  return window.location.origin || `${window.location.protocol}//${window.location.host}`
}

export function buildAdminOAuthRedirectUrl(provider: AdminOAuthRedirectProvider): string {
  const origin = resolveBrowserOrigin()
  return origin ? `${origin}${redirectPathByProvider[provider]}` : ''
}

export function isValidHttpUrl(url: string): boolean {
  if (!url) {
    return true
  }

  try {
    const parsedUrl = new URL(url)
    return parsedUrl.protocol === 'http:' || parsedUrl.protocol === 'https:'
  } catch {
    return false
  }
}
