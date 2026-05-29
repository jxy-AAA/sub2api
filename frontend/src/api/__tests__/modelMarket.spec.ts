import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { AxiosInstance } from 'axios'

describe('model market api', () => {
  let apiClient: AxiosInstance
  let getModels: typeof import('@/api/modelMarket').getModels
  let normalizeUserModelMarketResponse: typeof import('@/api/modelMarket').normalizeUserModelMarketResponse

  beforeEach(async () => {
    vi.resetModules()
    const modelMarketModule = await import('@/api/modelMarket')
    ;({ getModels, normalizeUserModelMarketResponse } = modelMarketModule)
    ;({ apiClient } = await import('@/api/client'))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('normalizes nested response payloads with alias fields', () => {
    const models = normalizeUserModelMarketResponse({
      items: [
        {
          model_id: 'gpt-4.1',
          display_name: 'GPT-4.1',
          provider_key: 'openai_compatible',
          protocol: 'openai',
          context_window: 128000,
          capabilities: 'chat,tools',
          tags: ['recommended', 'reasoning'],
          recommended: true,
          available_channels: [
            {
              channel_name: 'Cursor Pool',
              platform: 'openai_compatible',
              groups: [
                {
                  id: 7,
                  name: '国模兼容',
                  platform: 'openai_compatible',
                  subscription_type: 'standard',
                  rate_multiplier: 1,
                  is_exclusive: false,
                },
              ],
              price_summary: {
                billing_mode: 'token',
                input_price: 0.000002,
                output_price: 0.000008,
              },
            },
          ],
        },
      ],
    })

    expect(models).toHaveLength(1)
    expect(models[0]).toMatchObject({
      model_id: 'gpt-4.1',
      display_name: 'GPT-4.1',
      provider_key: 'openai_compatible',
      protocol: 'openai',
      context_window: 128000,
      capabilities: ['chat', 'tools'],
      tags: ['recommended', 'reasoning'],
      recommended: true,
    })
    expect(models[0].channels[0]).toMatchObject({
      name: 'Cursor Pool',
      platform: 'openai_compatible',
      groups: [
        expect.objectContaining({
          id: 7,
          name: '国模兼容',
          platform: 'openai_compatible',
        }),
      ],
    })
    expect(models[0].channels[0].pricing?.input_price).toBe(0.000002)
  })

  it('requests the model market endpoint and normalizes string channels', async () => {
    const adapter = vi.fn().mockResolvedValue({
      status: 200,
      data: {
        code: 0,
        data: {
          models: [
            {
              id: 'claude-sonnet-4',
              display_name: 'Claude Sonnet 4',
              provider: 'anthropic_compatible',
              protocol: 'anthropic',
              channels: ['Claude Code'],
            },
          ],
        },
      },
      headers: {},
      config: {},
      statusText: 'OK',
    })

    apiClient.defaults.adapter = adapter

    const models = await getModels()

    expect(adapter.mock.calls[0][0].url).toBe('/model-market/models')
    expect(models).toHaveLength(1)
    expect(models[0]).toMatchObject({
      model_id: 'claude-sonnet-4',
      display_name: 'Claude Sonnet 4',
      provider: 'anthropic_compatible',
      protocol: 'anthropic',
    })
    expect(models[0].channels).toEqual([{ name: 'Claude Code', groups: [], pricing: null }])
  })
})
