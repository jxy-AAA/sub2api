import { describe, expect, it } from 'vitest'
import {
  buildAdminOAuthRedirectUrl,
  isValidHttpUrl,
  resolveBrowserOrigin
} from '@/utils/adminSettingsUrl'

describe('adminSettingsUrl', () => {
  it('resolves the current browser origin', () => {
    expect(resolveBrowserOrigin()).toBe(window.location.origin)
  })

  it('builds provider callback URLs from the current origin', () => {
    expect(buildAdminOAuthRedirectUrl('github')).toBe(`${window.location.origin}/api/v1/auth/oauth/github/callback`)
    expect(buildAdminOAuthRedirectUrl('oidc')).toBe(`${window.location.origin}/api/v1/auth/oauth/oidc/callback`)
  })

  it('accepts empty values and validates http/https URLs', () => {
    expect(isValidHttpUrl('')).toBe(true)
    expect(isValidHttpUrl('https://example.com/docs')).toBe(true)
    expect(isValidHttpUrl('ftp://example.com/docs')).toBe(false)
  })
})
