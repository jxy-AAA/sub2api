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
  getDefaultDistributionPricing,
  getUserDistributionPricing,
  getUserInvitePricing,
  updateDefaultDistributionPricing,
  updateUserDistributionPricing,
  updateUserInvitePricing,
  updateUserUpstream,
} from '@/api/admin/affiliates'

describe('admin affiliates group rates api', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('normalizes group pricing responses', async () => {
    get.mockResolvedValueOnce({
      data: {
        user_id: 18,
        group_rates: [{ group_id: 3, group_name: 'Pro', rate_multiplier: 1.9 }],
        updated_at: '2026-05-11T08:00:00Z',
      },
    })

    const result = await getUserDistributionPricing(18)

    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/18/pricing')
    expect(result.user_id).toBe(18)
    expect(result.updated_at).toBe('2026-05-11T08:00:00Z')
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 3,
        group_name: 'Pro',
        rate_multiplier: 1.9,
      }),
    ])
  })

  it('normalizes default pricing responses', async () => {
    get.mockResolvedValueOnce({
      data: {
        group_rates: [{ group_id: 1, group_name: 'Default', rate_multiplier: 1.1 }],
      },
    })

    const result = await getDefaultDistributionPricing()

    expect(get).toHaveBeenCalledWith('/admin/affiliates/default-pricing')
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 1,
        group_name: 'Default',
        rate_multiplier: 1.1,
      }),
    ])
  })

  it('normalizes invite pricing responses', async () => {
    get.mockResolvedValueOnce({
      data: {
        user_id: 18,
        invite_group_rates: [{ group_id: 4, group_name: 'Invite', rate_multiplier: 1.7 }],
      },
    })

    const result = await getUserInvitePricing(18)

    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/18/invite-pricing')
    expect(result.user_id).toBe(18)
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 4,
        group_name: 'Invite',
        rate_multiplier: 1.7,
      }),
    ])
  })

  it('verifies default pricing when update response omits group_rates', async () => {
    put.mockResolvedValueOnce({
      data: {},
    })
    get.mockResolvedValueOnce({
      data: {
        group_rates: [{ group_id: 1, group_name: 'Default', rate_multiplier: 1.2 }],
      },
    })

    const result = await updateDefaultDistributionPricing({
      group_rates: [{ group_id: 1, rate_multiplier: 1.2 }],
    })

    expect(put).toHaveBeenCalledWith('/admin/affiliates/default-pricing', {
      group_rates: [{ group_id: 1, rate_multiplier: 1.2 }],
    })
    expect(get).toHaveBeenCalledWith('/admin/affiliates/default-pricing')
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 1,
        group_name: 'Default',
        rate_multiplier: 1.2,
      }),
    ])
  })

  it('verifies current pricing when update response omits group_rates', async () => {
    put.mockResolvedValueOnce({
      data: {
        user_id: 23,
      },
    })
    get.mockResolvedValueOnce({
      data: {
        user_id: 23,
        group_rates: [{ group_id: 3, group_name: 'Pro', rate_multiplier: 1.6 }],
      },
    })

    const result = await updateUserDistributionPricing(23, {
      group_rates: [{ group_id: 3, rate_multiplier: 1.6 }],
    })

    expect(put).toHaveBeenCalledWith('/admin/affiliates/users/23/pricing', {
      group_rates: [{ group_id: 3, rate_multiplier: 1.6 }],
    })
    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/23/pricing')
    expect(result.user_id).toBe(23)
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 3,
        group_name: 'Pro',
        rate_multiplier: 1.6,
      }),
    ])
  })

  it('verifies invite pricing when update response omits group_rates', async () => {
    put.mockResolvedValueOnce({
      data: {
        user_id: 23,
      },
    })
    get.mockResolvedValueOnce({
      data: {
        user_id: 23,
        group_rates: [{ group_id: 7, group_name: 'Invite', rate_multiplier: 1.4 }],
      },
    })

    const result = await updateUserInvitePricing(23, {
      group_rates: [{ group_id: 7, rate_multiplier: 1.4 }],
    })

    expect(put).toHaveBeenCalledWith('/admin/affiliates/users/23/invite-pricing', {
      group_rates: [{ group_id: 7, rate_multiplier: 1.4 }],
    })
    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/23/invite-pricing')
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 7,
        group_name: 'Invite',
        rate_multiplier: 1.4,
      }),
    ])
  })

  it('fails when verified current pricing still lacks group_rates', async () => {
    put.mockResolvedValueOnce({
      data: {
        user_id: 23,
      },
    })
    get.mockResolvedValueOnce({
      data: {
        user_id: 23,
        group_rates: [],
      },
    })

    await expect(updateUserDistributionPricing(23, {
      group_rates: [{ group_id: 3, rate_multiplier: 1.6 }],
    })).rejects.toThrow('group_rates')
  })

  it('normalizes upstream updates with inviter and upstream ids', async () => {
    put.mockResolvedValueOnce({
      data: {
        user_id: 23,
        inviter_id: 9,
        upstream_user_id: 9,
        updated_at: '2026-05-11T08:30:00Z',
      },
    })

    const result = await updateUserUpstream(23, {
      inviter_id: 9,
    })

    expect(put).toHaveBeenCalledWith('/admin/affiliates/users/23/upstream', {
      inviter_id: 9,
    })
    expect(result).toEqual({
      user_id: 23,
      inviter_id: 9,
      upstream_user_id: 9,
      updated_at: '2026-05-11T08:30:00Z',
    })
  })
})
