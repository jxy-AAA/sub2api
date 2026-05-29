import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
    delete: del,
  },
}))

import modelMarketAPI from '@/api/admin/modelMarket'

describe('admin model market api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('unwraps paginated list responses and forwards filters', async () => {
    get.mockResolvedValueOnce({
      data: {
        items: [
          {
            id: 1,
            model_id: 'gpt-4.1',
            display_name: 'GPT-4.1',
            provider_key: 'openai_compatible',
            protocol: 'openai',
            capabilities: 'chat, vision',
            tags: ['recommended'],
            status: 'active',
            sort_order: 10,
          },
        ],
      },
    })

    const result = await modelMarketAPI.list({
      protocol: 'openai',
      provider_key: 'openai_compatible',
    })

    expect(get).toHaveBeenCalledWith('/admin/model-market/models', {
      params: {
        protocol: 'openai',
        provider_key: 'openai_compatible',
      },
      signal: undefined,
    })
    expect(result).toEqual([
      expect.objectContaining({
        model_id: 'gpt-4.1',
        capabilities: ['chat', 'vision'],
        provider_key: 'openai_compatible',
      }),
    ])
  })

  it('normalizes create update and import responses', async () => {
    post
      .mockResolvedValueOnce({
        data: {
          id: 2,
          model_id: 'claude-sonnet-4',
          display_name: 'Claude Sonnet 4',
          provider_key: 'anthropic_compatible',
          protocol: 'anthropic',
          capabilities: ['chat'],
          tags: 'recommended, long-context',
          status: 'active',
          sort_order: 5,
        },
      })
      .mockResolvedValueOnce({
        data: {
          message: 'ok',
          imported_count: 4,
          updated_count: 2,
        },
      })
    put.mockResolvedValueOnce({
      data: {
        id: 2,
        model_id: 'claude-sonnet-4',
        display_name: 'Claude Sonnet 4.1',
        provider_key: 'anthropic_compatible',
        protocol: 'anthropic',
        capabilities: ['chat'],
        tags: ['recommended'],
        status: 'hidden',
        sort_order: 6,
      },
    })

    await expect(
      modelMarketAPI.create({
        model_id: 'claude-sonnet-4',
        display_name: 'Claude Sonnet 4',
        provider_key: 'anthropic_compatible',
        protocol: 'anthropic',
      })
    ).resolves.toEqual(
      expect.objectContaining({
        tags: ['recommended', 'long-context'],
      })
    )

    await expect(
      modelMarketAPI.update(2, {
        display_name: 'Claude Sonnet 4.1',
        status: 'hidden',
      })
    ).resolves.toEqual(
      expect.objectContaining({
        display_name: 'Claude Sonnet 4.1',
        status: 'hidden',
      })
    )

    await expect(modelMarketAPI.importFromChannels()).resolves.toEqual({
      message: 'ok',
      imported_count: 4,
      updated_count: 2,
    })
  })

  it('forwards delete requests', async () => {
    del.mockResolvedValueOnce({ data: { message: 'deleted' } })

    await expect(modelMarketAPI.remove(9)).resolves.toEqual({ message: 'deleted' })
    expect(del).toHaveBeenCalledWith('/admin/model-market/models/9')
  })
})
