import { apiClient } from '../client'
import type {
  ModelMarketImportResult,
  ModelMarketChannelRef,
  ModelMarketItem,
  ModelMarketProtocol,
  ModelMarketStatus,
} from '@/types'

export interface ModelMarketListFilters {
  keyword?: string
  protocol?: ModelMarketProtocol | ''
  provider_key?: string
  status?: ModelMarketStatus | ''
  capability?: string
}

export interface ModelMarketMutationPayload {
  model_id: string
  display_name: string
  provider_key: string
  protocol: ModelMarketProtocol
  capabilities?: string[]
  context_window?: number | null
  description?: string | null
  tags?: string[]
  status?: ModelMarketStatus
  sort_order?: number
  metadata?: Record<string, unknown>
}

type ModelMarketListResponse =
  | ModelMarketItem[]
  | {
      items?: unknown[]
    }

function normalizeProtocol(value: unknown, providerKey: unknown): ModelMarketProtocol {
  const normalized = String(value ?? '').trim().toLowerCase()
  if (normalized === 'anthropic' || normalized === 'anthropic_compatible') {
    return 'anthropic'
  }
  if (normalized === 'openai' || normalized === 'openai_compatible') {
    return 'openai'
  }

  const normalizedProviderKey = String(providerKey ?? '').trim().toLowerCase()
  if (normalizedProviderKey === 'anthropic' || normalizedProviderKey === 'anthropic_compatible') {
    return 'anthropic'
  }
  return 'openai'
}

function normalizeStringList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return Array.from(
      new Set(
        value
          .map((item) => String(item ?? '').trim())
          .filter((item) => item.length > 0)
      )
    )
  }
  if (typeof value === 'string') {
    return Array.from(
      new Set(
        value
          .split(/[,\n]/)
          .map((item) => item.trim())
          .filter((item) => item.length > 0)
      )
    )
  }
  return []
}

function normalizeMetadata(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return undefined
  }
  return value as Record<string, unknown>
}

function normalizeChannelRefs(value: unknown): ModelMarketChannelRef[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.map((item) => {
    const raw = (item || {}) as Record<string, unknown>
    const groupIds = Array.isArray(raw.group_ids)
      ? raw.group_ids.map((id) => Number(id)).filter((id) => Number.isFinite(id))
      : []
    return {
      channel_id: Number(raw.channel_id ?? 0),
      channel_name: String(raw.channel_name ?? ''),
      channel_status: String(raw.channel_status ?? ''),
      platform: String(raw.platform ?? ''),
      group_ids: groupIds,
      pricing: raw.pricing && typeof raw.pricing === 'object'
        ? (raw.pricing as ModelMarketChannelRef['pricing'])
        : null,
    }
  })
}

function normalizeContextWindow(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
    return Math.trunc(value)
  }
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    return Number.isFinite(parsed) && parsed > 0 ? Math.trunc(parsed) : null
  }
  return null
}

function normalizeItem(value: unknown): ModelMarketItem {
  const raw = (value || {}) as Record<string, unknown>
  const channelRefs = normalizeChannelRefs(raw.channel_refs)
  return {
    id: Number(raw.id ?? 0),
    model_id: String(raw.model_id ?? ''),
    display_name: String(raw.display_name ?? raw.model_id ?? ''),
    provider_key: String(raw.provider_key ?? ''),
    protocol: normalizeProtocol(raw.protocol, raw.provider_key),
    capabilities: normalizeStringList(raw.capabilities),
    context_window: normalizeContextWindow(raw.context_window),
    description: typeof raw.description === 'string' ? raw.description : null,
    tags: normalizeStringList(raw.tags),
    status:
      raw.status === 'hidden' || raw.status === 'disabled'
        ? raw.status
        : 'active',
    sort_order:
      typeof raw.sort_order === 'number' && Number.isFinite(raw.sort_order)
        ? Math.trunc(raw.sort_order)
        : 0,
    metadata: normalizeMetadata(raw.metadata),
    created_at: typeof raw.created_at === 'string' ? raw.created_at : undefined,
    updated_at: typeof raw.updated_at === 'string' ? raw.updated_at : undefined,
    channel_refs: channelRefs,
    available_channel_count:
      typeof raw.available_channel_count === 'number' && Number.isFinite(raw.available_channel_count)
        ? Math.trunc(raw.available_channel_count)
        : channelRefs.length,
  }
}

function normalizeListResponse(payload: ModelMarketListResponse): ModelMarketItem[] {
  if (Array.isArray(payload)) {
    return payload.map(normalizeItem)
  }
  if (payload && Array.isArray(payload.items)) {
    return payload.items.map(normalizeItem)
  }
  return []
}

export async function list(
  filters?: ModelMarketListFilters,
  options?: { signal?: AbortSignal }
): Promise<ModelMarketItem[]> {
  const { data } = await apiClient.get<ModelMarketListResponse>('/admin/model-market/models', {
    params: filters,
    signal: options?.signal,
  })
  return normalizeListResponse(data)
}

export async function create(payload: ModelMarketMutationPayload): Promise<ModelMarketItem> {
  const { data } = await apiClient.post<ModelMarketItem>('/admin/model-market/models', payload)
  return normalizeItem(data)
}

export async function update(
  id: number,
  payload: Partial<ModelMarketMutationPayload>
): Promise<ModelMarketItem> {
  const { data } = await apiClient.put<ModelMarketItem>(`/admin/model-market/models/${id}`, payload)
  return normalizeItem(data)
}

export async function remove(id: number): Promise<{ message?: string }> {
  const { data } = await apiClient.delete<{ message?: string }>(`/admin/model-market/models/${id}`)
  return data
}

export async function importFromChannels(): Promise<ModelMarketImportResult> {
  const { data } = await apiClient.post<ModelMarketImportResult>(
    '/admin/model-market/models/import-from-channels'
  )
  return data
}

const modelMarketAPI = {
  list,
  create,
  update,
  remove,
  importFromChannels,
}

export default modelMarketAPI
