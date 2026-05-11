import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, putMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    put: putMock,
  },
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
      },
    })

    const { getManagedAffiliatePermissions } = await import('@/api/userAffiliateManaged')
    const result = await getManagedAffiliatePermissions()

    expect(getMock).toHaveBeenCalledWith('/user/aff/managed/permissions')
    expect(result.can_view_downline_daily_revenue).toBe(true)
    expect(result.can_view_downline_rebate_balances).toBe(false)
    expect(result.can_manage_downline_pricing).toBe(true)
  })

  it('normalizes pricing responses from current_group_rates fallback', async () => {
    getMock.mockResolvedValueOnce({
      data: {
        user_id: 12,
        current_group_rates: [{ group_id: 3, group_name: 'Pro', rate_multiplier: 1.6 }],
      },
    })

    const { getManagedUserDistributionPricing } = await import('@/api/userAffiliateManaged')
    const result = await getManagedUserDistributionPricing(12)

    expect(result).toEqual({
      user_id: 12,
      group_rates: [{ group_id: 3, group_name: 'Pro', rate_multiplier: 1.6 }],
    })
  })

  it('verifies managed pricing through follow-up get when update response omits rates', async () => {
    putMock.mockResolvedValueOnce({
      data: {
        user_id: 12,
      },
    })
    getMock.mockResolvedValueOnce({
      data: {
        user_id: 12,
        group_rates: [{ group_id: 3, group_name: 'Pro', rate_multiplier: 1.6 }],
      },
    })

    const { updateManagedUserDistributionPricing } = await import('@/api/userAffiliateManaged')
    const result = await updateManagedUserDistributionPricing(12, {
      group_rates: [{ group_id: 3, rate_multiplier: 1.6 }],
    })

    expect(putMock).toHaveBeenCalledWith('/user/aff/managed/users/12/pricing', {
      group_rates: [{ group_id: 3, rate_multiplier: 1.6 }],
    })
    expect(getMock).toHaveBeenCalledWith('/user/aff/managed/users/12/pricing')
    expect(result).toEqual({
      user_id: 12,
      group_rates: [{ group_id: 3, group_name: 'Pro', rate_multiplier: 1.6 }],
    })
  })

  it('fails when managed pricing verification still lacks group_rates', async () => {
    putMock.mockResolvedValueOnce({
      data: {
        user_id: 12,
      },
    })
    getMock.mockResolvedValueOnce({
      data: {
        user_id: 12,
        group_rates: [],
      },
    })

    const { updateManagedUserDistributionPricing } = await import('@/api/userAffiliateManaged')

    await expect(updateManagedUserDistributionPricing(12, {
      group_rates: [{ group_id: 3, rate_multiplier: 1.6 }],
    })).rejects.toThrow('group_rates')
  })
})
