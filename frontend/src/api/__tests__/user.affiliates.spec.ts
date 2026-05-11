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

vi.mock('@/api/auth', () => ({
  resolveWeChatOAuthStartStrict: () => ({ mode: 'open' }),
  prepareOAuthBindAccessTokenCookie: vi.fn(),
}))

describe('user affiliate api', () => {
  beforeEach(() => {
    getMock.mockReset()
    putMock.mockReset()
  })

  it('normalizes affiliate overview to the new distribution shape', async () => {
    getMock.mockResolvedValueOnce({
      data: {
        user_id: 5,
        invite_code: 'AGENT-5',
        inviter_id: 1,
        invite_group_rates: [{ group_id: 2, group_name: 'Invite', rate_multiplier: 1.6 }],
        current_group_rates: [{ group_id: 3, group_name: 'Current', rate_multiplier: 1.4 }],
        today_business_usd: 18,
        today_rebate_rmb: 9,
        current_rebate_balance_rmb: 27,
        direct_member_count: 1,
        direct_children: [
          {
            user_id: 8,
            email: 'child@example.com',
            username: 'child',
            is_agent: true,
            created_at: '2026-05-11T08:00:00Z',
            today_business_usd: 11,
            today_rebate_rmb: 2,
            current_rebate_balance_rmb: 4,
            current_group_rates: [{ group_id: 4, group_name: 'Child', rate_multiplier: 1.9 }],
          },
        ],
      },
    })

    const { getAffiliateOverview } = await import('@/api/user')
    const result = await getAffiliateOverview()

    expect(getMock).toHaveBeenCalledWith('/user/aff/distribution')
    expect(result).toEqual({
      user_id: 5,
      aff_code: 'AGENT-5',
      inviter_id: 1,
      invite_group_rates: [{ group_id: 2, group_name: 'Invite', group_platform: undefined, group_rate_multiplier: undefined, rate_multiplier: 1.6, source_type: undefined, source_aff_code: undefined, upstream_user_id: null, updated_at: null }],
      my_group_rates: [{ group_id: 3, group_name: 'Current', group_platform: undefined, group_rate_multiplier: undefined, rate_multiplier: 1.4, source_type: undefined, source_aff_code: undefined, upstream_user_id: null, updated_at: null }],
      today_revenue_usd: 18,
      today_rebate_rmb: 9,
      current_rebate_balance_rmb: 27,
      direct_children: [{
        user_id: 8,
        email: 'child@example.com',
        username: 'child',
        role: 'agent',
        joined_at: '2026-05-11T08:00:00Z',
        today_revenue_usd: 11,
        today_rebate_rmb: 2,
        current_rebate_balance_rmb: 4,
        group_rates: [{ group_id: 4, group_name: 'Child', group_platform: undefined, group_rate_multiplier: undefined, rate_multiplier: 1.9, source_type: undefined, source_aff_code: undefined, upstream_user_id: null, updated_at: null }],
      }],
      direct_children_count: 1,
    })
  })

  it('verifies invite pricing when update response omits group_rates', async () => {
    putMock.mockResolvedValueOnce({
      data: {},
    })
    getMock.mockResolvedValueOnce({
      data: {
        group_rates: [{ group_id: 2, group_name: 'Invite', rate_multiplier: 1.6 }],
      },
    })

    const { updateInviteCodeGroupRates } = await import('@/api/user')
    const result = await updateInviteCodeGroupRates({
      group_rates: [{ group_id: 2, rate_multiplier: 1.6 }],
    })

    expect(putMock).toHaveBeenCalledWith('/user/aff/invite-pricing', {
      group_rates: [{ group_id: 2, rate_multiplier: 1.6 }],
    })
    expect(getMock).toHaveBeenCalledWith('/user/aff/invite-pricing')
    expect(result.group_rates).toEqual([
      expect.objectContaining({
        group_id: 2,
        group_name: 'Invite',
        rate_multiplier: 1.6,
      }),
    ])
  })
})
