import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routerMock } = vi.hoisted(() => ({
  routerMock: {
    push: vi.fn().mockResolvedValue(undefined),
    replace: vi.fn().mockResolvedValue(undefined)
  }
}))

vi.mock('@/router', () => ({
  default: routerMock
}))

describe('spaNavigation', () => {
  beforeEach(() => {
    routerMock.push.mockClear()
    routerMock.replace.mockClear()
  })

  it('detects internal app paths', async () => {
    const { isInternalNavigationTarget } = await import('@/utils/spaNavigation')

    expect(isInternalNavigationTarget('/login')).toBe(true)
    expect(isInternalNavigationTarget('https://example.com')).toBe(false)
    expect(isInternalNavigationTarget('//cdn.example.com/app')).toBe(false)
  })

  it('navigates with router.push for internal routes', async () => {
    const { navigateWithinApp } = await import('@/utils/spaNavigation')

    await expect(navigateWithinApp('/admin/settings')).resolves.toBe(true)
    expect(routerMock.push).toHaveBeenCalledWith('/admin/settings')
  })

  it('navigates with router.replace when requested', async () => {
    const { navigateWithinApp } = await import('@/utils/spaNavigation')

    await expect(navigateWithinApp('/login', { replace: true })).resolves.toBe(true)
    expect(routerMock.replace).toHaveBeenCalledWith('/login')
  })
})
