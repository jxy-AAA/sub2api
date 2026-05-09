import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, putMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    put: putMock,
  }
}))

describe('userAffiliateManaged api', () => {
  beforeEach(() => {
    getMock.mockReset()
    putMock.mockReset()
  })

  it('loads managed affiliate permissions', async () => {
    getMock.mockResolvedValueOnce({
      data: {
        can_view_downline_daily_revenue: true,
        can_view_downline_rebate_balances: false,
        can_manage_downline_pricing: true,
      }
    })

    const { getManagedAffiliatePermissions } = await import('@/api/userAffiliateManaged')
    const result = await getManagedAffiliatePermissions()

    expect(getMock).toHaveBeenCalledWith('/user/aff/managed/permissions', { params: undefined })
    expect(result.can_view_downline_daily_revenue).toBe(true)
    expect(result.can_view_downline_rebate_balances).toBe(false)
    expect(result.can_manage_downline_pricing).toBe(true)
  })

  it('normalizes managed pricing payloads to the user managed endpoint', async () => {
    putMock.mockResolvedValueOnce({
      data: {
        user_id: 12,
        model_rates: [{ model_name: 'gpt-5.4', multiplier: 1.6 }]
      }
    })

    const { updateManagedUserDistributionPricing } = await import('@/api/userAffiliateManaged')
    await updateManagedUserDistributionPricing(12, {
      model_rates: [{ model_name: 'gpt-5.4', multiplier: 1.6 }]
    })

    expect(putMock).toHaveBeenCalledWith('/user/aff/managed/users/12/pricing', {
      model_rates: [{ model: 'gpt-5.4', multiplier: 1.6 }]
    })
  })
})
