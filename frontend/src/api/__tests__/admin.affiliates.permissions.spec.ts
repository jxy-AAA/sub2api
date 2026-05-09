import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    put,
  },
}))

import {
  getUserDistributionPermissions,
  updateUserDistributionPermissions,
  type AffiliateAgentPermissions,
  type UpdateAffiliateAgentPermissionsRequest,
} from '@/api/admin/affiliates'

type Assert<T extends true> = T
type IsExact<T, U> = (
  (<G>() => G extends T ? 1 : 2) extends (<G>() => G extends U ? 1 : 2)
    ? ((<G>() => G extends U ? 1 : 2) extends (<G>() => G extends T ? 1 : 2) ? true : false)
    : false
)

type ExpectedPermissions = {
  user_id: number
  can_view_downline_daily_revenue: boolean
  can_view_downline_rebate_balances: boolean
  can_manage_downline_pricing: boolean
  granted_by_user_id?: number | null
  granted_by_email?: string | null
  updated_at?: string | null
  created_at?: string | null
}

type ExpectedUpdateRequest = {
  can_view_downline_daily_revenue: boolean
  can_view_downline_rebate_balances: boolean
  can_manage_downline_pricing: boolean
}

const permissionsContractExact: Assert<
  IsExact<AffiliateAgentPermissions, ExpectedPermissions>
> = true

const requestContractExact: Assert<
  IsExact<UpdateAffiliateAgentPermissionsRequest, ExpectedUpdateRequest>
> = true

describe('admin affiliates permissions api', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('loads permissions from the primary endpoint and normalizes the response', async () => {
    get.mockResolvedValue({
      data: {
        user_id: 18,
        can_view_downline_daily_revenue: true,
        can_view_downline_rebate_balances: false,
        can_manage_downline_pricing: true,
        granted_by_email: 'root@example.com',
      },
    })

    const result = await getUserDistributionPermissions(18)

    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/18/permissions')
    expect(result).toEqual({
      user_id: 18,
      can_view_downline_daily_revenue: true,
      can_view_downline_rebate_balances: false,
      can_manage_downline_pricing: true,
      granted_by_user_id: null,
      granted_by_email: 'root@example.com',
      updated_at: null,
      created_at: null,
    })
  })

  it('saves permissions and accepts wrapped backend responses', async () => {
    const payload: UpdateAffiliateAgentPermissionsRequest = {
      can_view_downline_daily_revenue: true,
      can_view_downline_rebate_balances: true,
      can_manage_downline_pricing: false,
    }

    put.mockResolvedValue({
      data: {
        user_id: 21,
        permissions: {
          user_id: 21,
          can_view_downline_daily_revenue: true,
          can_view_downline_rebate_balances: true,
          can_manage_downline_pricing: false,
          granted_by_user_id: 1,
          granted_by_email: 'root@example.com',
          updated_at: '2026-05-10T08:00:00Z',
        },
      },
    })

    const result = await updateUserDistributionPermissions(21, payload)

    expect(put).toHaveBeenCalledWith('/admin/affiliates/users/21/permissions', payload)
    expect(result).toEqual({
      user_id: 21,
      can_view_downline_daily_revenue: true,
      can_view_downline_rebate_balances: true,
      can_manage_downline_pricing: false,
      granted_by_user_id: 1,
      granted_by_email: 'root@example.com',
      updated_at: '2026-05-10T08:00:00Z',
      created_at: null,
    })
  })

  it('does not fallback when the primary endpoint reports a missing user', async () => {
    const missingUserError = {
      status: 404,
      reason: 'USER_NOT_FOUND',
      message: 'user not found',
    }
    get.mockRejectedValue(missingUserError)

    await expect(getUserDistributionPermissions(1475116430)).rejects.toBe(missingUserError)

    expect(get).toHaveBeenCalledTimes(1)
    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/1475116430/permissions')
  })

  it('keeps permission request and response types aligned with the backend contract', () => {
    expect(permissionsContractExact).toBe(true)
    expect(requestContractExact).toBe(true)
  })
})
