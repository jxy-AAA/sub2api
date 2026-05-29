import { apiClient } from './client'
import type { UserAvailableGroup } from './channels'
import {
  isAnthropicProtocolPlatform,
  isOpenAIProtocolPlatform,
  normalizePlatformKey,
} from '@/utils/platforms'

export interface UserModelMarketPricingInterval {
  min_tokens: number
  max_tokens: number | null
  tier_label?: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
}

export interface UserModelMarketPricing {
  billing_mode?: string | null
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: UserModelMarketPricingInterval[]
}

export interface UserModelMarketChannel {
  id?: number | string
  name: string
  provider?: string
  provider_key?: string
  platform?: string
  status?: string
  available?: boolean
  groups: UserAvailableGroup[]
  pricing: UserModelMarketPricing | null
}

export interface UserModelMarketModel {
  id?: number | string
  model_id: string
  display_name?: string
  provider?: string
  provider_key?: string
  platform?: string
  protocol?: string
  capabilities: string[]
  tags: string[]
  context_window?: number | null
  description?: string
  status?: string
  recommended?: boolean
  available?: boolean
  channels: UserModelMarketChannel[]
  price_summary: UserModelMarketPricing | null
  metadata?: Record<string, unknown>
}

type RawRecord = Record<string, unknown>

function asRecord(value: unknown): RawRecord | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as RawRecord) : null
}

function normalizeString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

function inferProtocol(value?: string | null): string | undefined {
  const normalized = normalizePlatformKey(value)
  if (!normalized) return undefined
  if (isOpenAIProtocolPlatform(normalized)) return 'openai'
  if (isAnthropicProtocolPlatform(normalized)) return 'anthropic'
  return undefined
}

function normalizeNumber(value: unknown): number | null {
  if (value === null || value === undefined || value === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

function normalizeBoolean(value: unknown): boolean | undefined {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (['true', '1', 'yes', 'enabled', 'active'].includes(normalized)) return true
    if (['false', '0', 'no', 'disabled', 'inactive'].includes(normalized)) return false
  }
  return undefined
}

function normalizeStringList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return Array.from(
      new Set(
        value
          .map((item) => normalizeString(item))
          .filter((item): item is string => Boolean(item)),
      ),
    )
  }

  const single = normalizeString(value)
  if (!single) return []

  return Array.from(
    new Set(
      single
        .split(/[,\n|]/)
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  )
}

function normalizeGroup(value: unknown): UserAvailableGroup | null {
  const payload = asRecord(value)
  if (!payload) return null

  const id = normalizeNumber(payload.id)
  const name = normalizeString(payload.name)
  if (id === null || !name) {
    return null
  }

  return {
    id,
    name,
    platform: normalizeString(payload.platform) ?? '',
    subscription_type: normalizeString(payload.subscription_type) ?? 'standard',
    rate_multiplier: normalizeNumber(payload.rate_multiplier) ?? 1,
    is_exclusive: normalizeBoolean(payload.is_exclusive) ?? false,
  }
}

function normalizeGroupList(value: unknown): UserAvailableGroup[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map(normalizeGroup)
    .filter((item): item is UserAvailableGroup => item !== null)
}

function normalizePricingInterval(value: unknown): UserModelMarketPricingInterval | null {
  const payload = asRecord(value)
  if (!payload) return null

  return {
    min_tokens: normalizeNumber(payload.min_tokens) ?? 0,
    max_tokens: normalizeNumber(payload.max_tokens),
    tier_label: normalizeString(payload.tier_label),
    input_price: normalizeNumber(payload.input_price),
    output_price: normalizeNumber(payload.output_price),
    cache_write_price: normalizeNumber(payload.cache_write_price),
    cache_read_price: normalizeNumber(payload.cache_read_price),
    per_request_price: normalizeNumber(payload.per_request_price),
  }
}

function normalizePricing(value: unknown): UserModelMarketPricing | null {
  const payload = asRecord(value)
  if (!payload) return null

  const pricing: UserModelMarketPricing = {
    billing_mode: normalizeString(payload.billing_mode) ?? normalizeString(payload.billingMode) ?? null,
    input_price: normalizeNumber(payload.input_price),
    output_price: normalizeNumber(payload.output_price),
    cache_write_price: normalizeNumber(payload.cache_write_price),
    cache_read_price: normalizeNumber(payload.cache_read_price),
    image_output_price: normalizeNumber(payload.image_output_price),
    per_request_price: normalizeNumber(payload.per_request_price),
    intervals: Array.isArray(payload.intervals)
      ? payload.intervals
          .map(normalizePricingInterval)
          .filter((item): item is UserModelMarketPricingInterval => item !== null)
      : [],
  }

  const hasValue = [
    pricing.billing_mode,
    pricing.input_price,
    pricing.output_price,
    pricing.cache_write_price,
    pricing.cache_read_price,
    pricing.image_output_price,
    pricing.per_request_price,
  ].some((item) => item !== null && item !== undefined)

  return hasValue || pricing.intervals.length > 0 ? pricing : null
}

function normalizeChannel(value: unknown): UserModelMarketChannel | null {
  if (typeof value === 'string') {
    const name = normalizeString(value)
    return name ? { name, groups: [], pricing: null } : null
  }

  const payload = asRecord(value)
  if (!payload) return null

  const name =
    normalizeString(payload.name) ??
    normalizeString(payload.display_name) ??
    normalizeString(payload.channel_name)

  if (!name) return null

  return {
    id: payload.id as number | string | undefined,
    name,
    provider: normalizeString(payload.provider) ?? normalizeString(payload.provider_name),
    provider_key:
      normalizeString(payload.provider_key) ??
      normalizeString(payload.providerKey) ??
      normalizeString(payload.platform) ??
      normalizeString(payload.provider) ??
      normalizeString(payload.provider_name),
    platform:
      normalizeString(payload.platform) ??
      normalizeString(payload.provider_key) ??
      normalizeString(payload.providerKey) ??
      normalizeString(payload.provider) ??
      normalizeString(payload.provider_name),
    status: normalizeString(payload.status),
    available:
      normalizeBoolean(payload.available) ??
      normalizeBoolean(payload.is_available) ??
      normalizeBoolean(payload.enabled),
    groups: normalizeGroupList(payload.groups ?? payload.available_groups ?? payload.group_refs),
    pricing:
      normalizePricing(payload.pricing) ??
      normalizePricing(payload.price_summary) ??
      normalizePricing(payload.pricing_summary),
  }
}

function pickArrayPayload(value: unknown): unknown[] {
  if (Array.isArray(value)) return value

  const payload = asRecord(value)
  if (!payload) return []

  for (const key of ['items', 'models', 'list', 'data']) {
    if (Array.isArray(payload[key])) {
      return payload[key] as unknown[]
    }
  }

  if (payload.data) {
    return pickArrayPayload(payload.data)
  }

  return []
}

function normalizeModel(value: unknown): UserModelMarketModel | null {
  const payload = asRecord(value)
  if (!payload) return null

  const modelId =
    normalizeString(payload.model_id) ??
    normalizeString(payload.model) ??
    normalizeString(payload.name) ??
    normalizeString(payload.id)

  if (!modelId) return null

  const channels = pickArrayPayload(payload.channels ?? payload.available_channels)
    .map(normalizeChannel)
    .filter((item): item is UserModelMarketChannel => item !== null)

  const providerKey =
    normalizeString(payload.provider_key) ??
    normalizeString(payload.providerKey) ??
    normalizeString(payload.platform) ??
    normalizeString(payload.provider) ??
    normalizeString(payload.provider_name)

  const platform =
    normalizeString(payload.platform) ??
    providerKey ??
    normalizeString(payload.provider) ??
    normalizeString(payload.provider_name)

  const protocol =
    normalizeString(payload.protocol) ??
    normalizeString(payload.compatibility) ??
    normalizeString(payload.protocol_type) ??
    inferProtocol(platform) ??
    inferProtocol(providerKey)

  const recommended =
    normalizeBoolean(payload.recommended) ??
    normalizeBoolean(payload.is_recommended) ??
    normalizeBoolean(payload.featured)

  const status = normalizeString(payload.status)
  const available =
    normalizeBoolean(payload.available) ??
    normalizeBoolean(payload.is_available) ??
    normalizeBoolean(payload.enabled) ??
    (status === 'disabled' ? false : undefined)

  return {
    id: payload.id as number | string | undefined,
    model_id: modelId,
    display_name:
      normalizeString(payload.display_name) ??
      normalizeString(payload.displayName) ??
      normalizeString(payload.title),
    provider:
      normalizeString(payload.provider) ??
      normalizeString(payload.provider_name),
    provider_key: providerKey,
    platform,
    protocol,
    capabilities: normalizeStringList(payload.capabilities),
    tags: normalizeStringList(payload.tags),
    context_window:
      normalizeNumber(payload.context_window) ??
      normalizeNumber(payload.contextWindow) ??
      normalizeNumber(payload.max_context_tokens),
    description:
      normalizeString(payload.description) ??
      normalizeString(payload.summary),
    status,
    recommended,
    available,
    channels,
    price_summary:
      normalizePricing(payload.price_summary) ??
      normalizePricing(payload.pricing) ??
      normalizePricing(payload.pricing_summary),
    metadata: asRecord(payload.metadata) ?? undefined,
  }
}

export function normalizeUserModelMarketResponse(value: unknown): UserModelMarketModel[] {
  return pickArrayPayload(value)
    .map(normalizeModel)
    .filter((item): item is UserModelMarketModel => item !== null)
}

export async function getModels(options?: { signal?: AbortSignal }): Promise<UserModelMarketModel[]> {
  const { data } = await apiClient.get<unknown>('/model-market/models', {
    signal: options?.signal,
  })

  return normalizeUserModelMarketResponse(data)
}

export const userModelMarketAPI = { getModels }

export default userModelMarketAPI
export type { UserAvailableGroup } from './channels'
