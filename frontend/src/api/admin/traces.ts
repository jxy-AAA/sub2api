import { apiClient } from '../client'
import type {
  CreateTraceExportTaskRequest,
  CreateTraceRuleRequest,
  PaginatedResponse,
  TraceExportTask,
  TraceRecord,
  TraceRecordFilters,
  TraceRule,
  UpdateTraceRuleRequest,
} from '@/types'

export interface TraceRecordListParams extends TraceRecordFilters {
  page?: number
  page_size?: number
}

export interface TraceExportTaskListParams {
  page?: number
  page_size?: number
}

export interface TraceDownloadResult {
  blob: Blob
  filename?: string
  contentType?: string
}

function normalizeInteger(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Math.trunc(value)
  }
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? Math.trunc(parsed) : null
  }
  return null
}

function normalizeNullableInteger(value: unknown): number | null {
  return normalizeInteger(value)
}

function normalizeStringList(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }
  return Array.from(
    new Set(
      value
        .map((item) => String(item ?? '').trim())
        .filter((item) => item.length > 0)
    )
  )
}

function normalizeIntegerList(value: unknown): number[] {
  if (!Array.isArray(value)) {
    return []
  }
  return Array.from(
    new Set(
      value
        .map((item) => normalizeInteger(item))
        .filter((item): item is number => item !== null && item > 0)
    )
  )
}

function normalizeTraceFilters(value: unknown): TraceRecordFilters {
  const raw = (value || {}) as Record<string, unknown>
  return {
    model: typeof raw.model === 'string' ? raw.model : '',
    user_id: normalizeNullableInteger(raw.user_id),
    api_key_id: normalizeNullableInteger(raw.api_key_id),
    capture_rule_id: normalizeNullableInteger(raw.capture_rule_id),
    start_time: typeof raw.start_time === 'string' ? raw.start_time : null,
    end_time: typeof raw.end_time === 'string' ? raw.end_time : null,
    start_date: typeof raw.start_date === 'string' ? raw.start_date : null,
    end_date: typeof raw.end_date === 'string' ? raw.end_date : null,
    timezone: typeof raw.timezone === 'string' ? raw.timezone : null,
    keyword: typeof raw.keyword === 'string' ? raw.keyword : '',
    min_input_tokens: normalizeNullableInteger(raw.min_input_tokens),
    max_input_tokens: normalizeNullableInteger(raw.max_input_tokens),
    min_output_tokens: normalizeNullableInteger(raw.min_output_tokens),
    max_output_tokens: normalizeNullableInteger(raw.max_output_tokens),
    min_total_tokens: normalizeNullableInteger(raw.min_total_tokens),
    max_total_tokens: normalizeNullableInteger(raw.max_total_tokens),
  }
}

function normalizeTraceRecord(value: unknown): TraceRecord {
  const raw = (value || {}) as Record<string, unknown>
  return {
    id: normalizeInteger(raw.id) ?? 0,
    task_id: String(raw.task_id ?? ''),
    request_id: typeof raw.request_id === 'string' ? raw.request_id : null,
    response_id: typeof raw.response_id === 'string' ? raw.response_id : null,
    user_id: normalizeNullableInteger(raw.user_id),
    api_key_id: normalizeNullableInteger(raw.api_key_id),
    group_id: normalizeNullableInteger(raw.group_id),
    account_id: normalizeNullableInteger(raw.account_id),
    capture_rule_id: normalizeNullableInteger(raw.capture_rule_id),
    protocol: String(raw.protocol ?? ''),
    model: String(raw.model ?? ''),
    requested_model: typeof raw.requested_model === 'string' ? raw.requested_model : null,
    upstream_model: typeof raw.upstream_model === 'string' ? raw.upstream_model : null,
    request_content_type:
      typeof raw.request_content_type === 'string' ? raw.request_content_type : null,
    response_content_type:
      typeof raw.response_content_type === 'string' ? raw.response_content_type : null,
    input_tokens: normalizeNullableInteger(raw.input_tokens),
    output_tokens: normalizeNullableInteger(raw.output_tokens),
    total_tokens: normalizeNullableInteger(raw.total_tokens),
    upstream_status_code: normalizeNullableInteger(raw.upstream_status_code),
    scaffold: String(raw.scaffold ?? ''),
    scaffold_version: String(raw.scaffold_version ?? ''),
    prompt: raw.prompt,
    candidates: raw.candidates,
    tools: raw.tools,
    signature: raw.signature,
    meta: raw.meta,
    raw_request: raw.raw_request,
    raw_response: raw.raw_response,
    raw_request_text: typeof raw.raw_request_text === 'string' ? raw.raw_request_text : null,
    raw_response_text: typeof raw.raw_response_text === 'string' ? raw.raw_response_text : null,
    dedupe_hash: String(raw.dedupe_hash ?? ''),
    prompt_hash: String(raw.prompt_hash ?? ''),
    created_at: String(raw.created_at ?? ''),
  }
}

function normalizeTraceRule(value: unknown): TraceRule {
  const raw = (value || {}) as Record<string, unknown>
  return {
    id: normalizeInteger(raw.id) ?? 0,
    name: String(raw.name ?? ''),
    enabled: raw.enabled !== false,
    priority: normalizeInteger(raw.priority) ?? 0,
    model_patterns: normalizeStringList(raw.model_patterns),
    user_ids: normalizeIntegerList(raw.user_ids),
    api_key_ids: normalizeIntegerList(raw.api_key_ids),
    keywords: normalizeStringList(raw.keywords),
    min_tokens: normalizeNullableInteger(raw.min_tokens),
    max_tokens: normalizeNullableInteger(raw.max_tokens),
    sampling_ratio:
      typeof raw.sampling_ratio === 'number' && Number.isFinite(raw.sampling_ratio)
        ? raw.sampling_ratio
        : 1,
    active_from: typeof raw.active_from === 'string' ? raw.active_from : null,
    active_to: typeof raw.active_to === 'string' ? raw.active_to : null,
    created_at: String(raw.created_at ?? ''),
    updated_at: String(raw.updated_at ?? ''),
  }
}

function normalizeTraceExportTask(value: unknown): TraceExportTask {
  const raw = (value || {}) as Record<string, unknown>
  return {
    id: normalizeInteger(raw.id) ?? 0,
    status:
      raw.status === 'running'
      || raw.status === 'succeeded'
      || raw.status === 'failed'
      || raw.status === 'canceled'
        ? raw.status
        : 'pending',
    format: String(raw.format ?? ''),
    filters: normalizeTraceFilters(raw.filters),
    include_raw: raw.include_raw === true,
    requested_by: normalizeInteger(raw.requested_by) ?? 0,
    download_filename:
      typeof raw.download_filename === 'string' ? raw.download_filename : null,
    file_size_bytes: normalizeInteger(raw.file_size_bytes) ?? 0,
    target_records: normalizeInteger(raw.target_records) ?? 500,
    total_records: normalizeInteger(raw.total_records) ?? 0,
    processed_records: normalizeInteger(raw.processed_records) ?? 0,
    error_message: typeof raw.error_message === 'string' ? raw.error_message : null,
    canceled_by: normalizeNullableInteger(raw.canceled_by),
    canceled_at: typeof raw.canceled_at === 'string' ? raw.canceled_at : null,
    started_at: typeof raw.started_at === 'string' ? raw.started_at : null,
    finished_at: typeof raw.finished_at === 'string' ? raw.finished_at : null,
    created_at: String(raw.created_at ?? ''),
    updated_at: String(raw.updated_at ?? ''),
  }
}

function normalizePaginatedResponse<T>(
  payload: unknown,
  normalizeItem: (value: unknown) => T
): PaginatedResponse<T> {
  const raw = (payload || {}) as Record<string, unknown>
  const items = Array.isArray(raw.items) ? raw.items.map(normalizeItem) : []
  return {
    items,
    total: normalizeInteger(raw.total) ?? items.length,
    page: normalizeInteger(raw.page) ?? 1,
    page_size: normalizeInteger(raw.page_size) ?? items.length,
    pages: normalizeInteger(raw.pages) ?? 1,
  }
}

function parseFilenameFromDisposition(contentDisposition?: string): string | undefined {
  if (!contentDisposition) {
    return undefined
  }

  const encodedMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (encodedMatch?.[1]) {
    try {
      return decodeURIComponent(encodedMatch[1])
    } catch {
      return encodedMatch[1]
    }
  }

  const quotedMatch = contentDisposition.match(/filename="([^"]+)"/i)
  if (quotedMatch?.[1]) {
    return quotedMatch[1]
  }

  const plainMatch = contentDisposition.match(/filename=([^;]+)/i)
  return plainMatch?.[1]?.trim()
}

export async function listRecords(
  params: TraceRecordListParams,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<TraceRecord>> {
  const { data } = await apiClient.get<PaginatedResponse<TraceRecord>>('/admin/traces', {
    params,
    signal: options?.signal,
  })
  return normalizePaginatedResponse(data, normalizeTraceRecord)
}

export async function getRecord(id: number): Promise<TraceRecord> {
  const { data } = await apiClient.get<TraceRecord>(`/admin/traces/${id}`)
  return normalizeTraceRecord(data)
}

export async function deleteRecord(id: number): Promise<{ id: number; deleted: boolean }> {
  const { data } = await apiClient.delete<{ id: number; deleted: boolean }>(`/admin/traces/${id}`)
  return {
    id: normalizeInteger((data as Record<string, unknown>)?.id) ?? id,
    deleted: (data as Record<string, unknown>)?.deleted === true,
  }
}

export async function batchDeleteRecords(ids: number[]): Promise<{ deleted_count: number }> {
  const { data } = await apiClient.post<{ deleted_count: number }>('/admin/traces/batch-delete', {
    ids,
  })
  return {
    deleted_count: normalizeInteger((data as Record<string, unknown>)?.deleted_count) ?? 0,
  }
}

export async function listRules(): Promise<TraceRule[]> {
  const { data } = await apiClient.get<TraceRule[]>('/admin/traces/rules')
  return Array.isArray(data) ? data.map(normalizeTraceRule) : []
}

export async function getRuleById(id: number): Promise<TraceRule> {
  const { data } = await apiClient.get<TraceRule>(`/admin/traces/rules/${id}`)
  return normalizeTraceRule(data)
}

export async function createRule(payload: CreateTraceRuleRequest): Promise<TraceRule> {
  const { data } = await apiClient.post<TraceRule>('/admin/traces/rules', payload)
  return normalizeTraceRule(data)
}

export async function updateRule(
  id: number,
  payload: UpdateTraceRuleRequest
): Promise<TraceRule> {
  const { data } = await apiClient.put<TraceRule>(`/admin/traces/rules/${id}`, payload)
  return normalizeTraceRule(data)
}

export async function deleteRule(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/traces/rules/${id}`)
  return {
    message: String((data as Record<string, unknown>)?.message ?? ''),
  }
}

export async function listExportTasks(
  params: TraceExportTaskListParams,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<TraceExportTask>> {
  const { data } = await apiClient.get<PaginatedResponse<TraceExportTask>>(
    '/admin/traces/export-tasks',
    {
      params,
      signal: options?.signal,
    }
  )
  return normalizePaginatedResponse(data, normalizeTraceExportTask)
}

export async function createExportTask(
  payload: CreateTraceExportTaskRequest
): Promise<TraceExportTask> {
  const { data } = await apiClient.post<TraceExportTask>('/admin/traces/export-tasks', payload)
  return normalizeTraceExportTask(data)
}

export async function getExportTask(id: number): Promise<TraceExportTask> {
  const { data } = await apiClient.get<TraceExportTask>(`/admin/traces/export-tasks/${id}`)
  return normalizeTraceExportTask(data)
}

export async function cancelExportTask(id: number): Promise<{ id: number; status: string }> {
  const { data } = await apiClient.post<{ id: number; status: string }>(
    `/admin/traces/export-tasks/${id}/cancel`
  )
  return {
    id: normalizeInteger((data as Record<string, unknown>)?.id) ?? id,
    status: String((data as Record<string, unknown>)?.status ?? ''),
  }
}

export async function downloadExportTask(id: number): Promise<TraceDownloadResult> {
  const response = await apiClient.get<Blob>(`/admin/traces/export-tasks/${id}/download`, {
    responseType: 'blob',
  })
  const contentTypeHeader = response.headers['content-type']
  return {
    blob: response.data,
    filename: parseFilenameFromDisposition(response.headers['content-disposition']),
    contentType: typeof contentTypeHeader === 'string' ? contentTypeHeader : undefined,
  }
}

const tracesAPI = {
  listRecords,
  getRecord,
  deleteRecord,
  batchDeleteRecords,
  listRules,
  getRuleById,
  createRule,
  updateRule,
  deleteRule,
  listExportTasks,
  createExportTask,
  getExportTask,
  cancelExportTask,
  downloadExportTask,
}

export default tracesAPI
