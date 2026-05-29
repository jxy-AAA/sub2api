import { beforeEach, describe, expect, it } from 'vitest'

describe('setup api helpers', () => {
  beforeEach(() => {
    window.sessionStorage.clear()
    window.history.replaceState({}, '', '/setup')
  })

  it('stores trimmed setup token in session storage', async () => {
    const { getSetupToken, setSetupToken } = await import('@/api/setup')

    setSetupToken('  token-123  ')

    expect(getSetupToken()).toBe('token-123')
  })

  it('clears setup token when blank value is stored', async () => {
    const { getSetupToken, setSetupToken } = await import('@/api/setup')

    setSetupToken('token-abc')
    expect(getSetupToken()).toBe('token-abc')

    setSetupToken('   ')
    expect(getSetupToken()).toBe('')
  })
})
