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

import tracesAPI from '@/api/admin/traces'

describe('admin traces api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('normalizes trace record list and detail responses', async () => {
    get
      .mockResolvedValueOnce({
        data: {
          items: [
            {
              id: '11',
              task_id: 'task-11',
              user_id: '42',
              api_key_id: 7,
              capture_rule_id: '5',
              protocol: 'openai.responses',
              model: 'gpt-5.4',
              input_tokens: '128',
              output_tokens: 32,
              total_tokens: 160,
              created_at: '2026-05-27T10:00:00Z',
            },
          ],
          total: '1',
          page: '2',
          page_size: '5',
          pages: '1',
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 11,
          task_id: 'task-11',
          protocol: 'openai.responses',
          model: 'gpt-5.4',
          raw_request_text: '{"incident":42}',
          dedupe_hash: 'abc',
          prompt_hash: 'def',
          created_at: '2026-05-27T10:00:00Z',
        },
      })

    const list = await tracesAPI.listRecords({
      page: 2,
      page_size: 5,
      model: 'gpt-5.4',
      keyword: 'incident',
    })
    const detail = await tracesAPI.getRecord(11)

    expect(get).toHaveBeenNthCalledWith(1, '/admin/traces', {
      params: {
        page: 2,
        page_size: 5,
        model: 'gpt-5.4',
        keyword: 'incident',
      },
      signal: undefined,
    })
    expect(list).toEqual({
      items: [
        expect.objectContaining({
          id: 11,
          task_id: 'task-11',
          user_id: 42,
          capture_rule_id: 5,
          total_tokens: 160,
        }),
      ],
      total: 1,
      page: 2,
      page_size: 5,
      pages: 1,
    })
    expect(detail).toEqual(
      expect.objectContaining({
        id: 11,
        task_id: 'task-11',
        raw_request_text: '{"incident":42}',
      }),
    )
  })

  it('normalizes rule mutations and delete helpers', async () => {
    post
      .mockResolvedValueOnce({ data: { deleted_count: '2' } })
      .mockResolvedValueOnce({
        data: {
          id: '9',
          name: 'capture-incident',
          enabled: true,
          model_patterns: ['gpt-5.4'],
          user_ids: ['42', 42],
          api_key_ids: ['7'],
          keywords: ['incident'],
          sampling_ratio: 1,
          created_at: '2026-05-27T10:00:00Z',
          updated_at: '2026-05-27T10:00:00Z',
        },
      })
    get.mockResolvedValueOnce({
      data: [
        {
          id: '9',
          name: 'capture-incident',
          enabled: true,
          model_patterns: ['gpt-5.4'],
          keywords: ['incident'],
          sampling_ratio: 1,
          created_at: '2026-05-27T10:00:00Z',
          updated_at: '2026-05-27T10:00:00Z',
        },
      ],
    })
    put.mockResolvedValueOnce({
      data: {
        id: 9,
        name: 'capture-incident-updated',
        enabled: false,
        model_patterns: ['gpt-*'],
        user_ids: [42],
        api_key_ids: [7],
        keywords: ['incident', 'sev1'],
        sampling_ratio: 0.5,
        created_at: '2026-05-27T10:00:00Z',
        updated_at: '2026-05-27T10:05:00Z',
      },
    })
    del
      .mockResolvedValueOnce({ data: { id: '12', deleted: true } })
      .mockResolvedValueOnce({ data: { message: 'rule deleted' } })

    await expect(tracesAPI.batchDeleteRecords([11, 12])).resolves.toEqual({ deleted_count: 2 })
    await expect(tracesAPI.listRules()).resolves.toEqual([
      expect.objectContaining({ id: 9, name: 'capture-incident' }),
    ])
    await expect(
      tracesAPI.createRule({
        name: 'capture-incident',
        model_patterns: ['gpt-5.4'],
        keywords: ['incident'],
      }),
    ).resolves.toEqual(
      expect.objectContaining({
        id: 9,
        user_ids: [42],
        api_key_ids: [7],
      }),
    )
    await expect(
      tracesAPI.updateRule(9, {
        enabled: false,
        model_patterns: ['gpt-*'],
      }),
    ).resolves.toEqual(
      expect.objectContaining({
        enabled: false,
        model_patterns: ['gpt-*'],
        keywords: ['incident', 'sev1'],
      }),
    )
    await expect(tracesAPI.deleteRecord(12)).resolves.toEqual({ id: 12, deleted: true })
    await expect(tracesAPI.deleteRule(9)).resolves.toEqual({ message: 'rule deleted' })
  })

  it('normalizes export task lifecycle and parses download filenames', async () => {
    const blob = new Blob(['[{"task_id":"task-11"}]'], { type: 'application/json' })

    get
      .mockResolvedValueOnce({
        data: {
          items: [
            {
              id: '21',
              status: 'running',
              format: 'json_array',
              filters: { keyword: 'incident', capture_rule_id: '5' },
              include_raw: true,
              requested_by: '1',
              target_records: '500',
              total_records: '1',
              processed_records: '0',
              created_at: '2026-05-27T10:00:00Z',
              updated_at: '2026-05-27T10:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          page_size: 10,
          pages: 1,
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 21,
          status: 'succeeded',
          format: 'json_array',
          filters: { keyword: 'incident', capture_rule_id: 5 },
          include_raw: true,
          requested_by: 1,
          download_filename: 'trace-export.json',
          file_size_bytes: '24',
          target_records: '500',
          total_records: '1',
          processed_records: '1',
          created_at: '2026-05-27T10:00:00Z',
          updated_at: '2026-05-27T10:01:00Z',
        },
      })
      .mockResolvedValueOnce({
        data: blob,
        headers: {
          'content-disposition': `attachment; filename*=UTF-8''trace-export%20final.json`,
          'content-type': 'application/json; charset=utf-8',
        },
      })
    post
      .mockResolvedValueOnce({
        data: {
          id: '21',
          status: 'pending',
          format: 'json_array',
          filters: { keyword: 'incident' },
          include_raw: true,
          requested_by: 1,
          target_records: 500,
          total_records: 0,
          processed_records: 0,
          created_at: '2026-05-27T10:00:00Z',
          updated_at: '2026-05-27T10:00:00Z',
        },
      })
      .mockResolvedValueOnce({ data: { id: '21', status: 'canceled' } })

    await expect(tracesAPI.listExportTasks({ page: 1, page_size: 10 })).resolves.toEqual({
      items: [
        expect.objectContaining({
          id: 21,
          status: 'running',
          filters: expect.objectContaining({ capture_rule_id: 5 }),
        }),
      ],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1,
    })
    await expect(
      tracesAPI.createExportTask({
        include_raw: true,
        target_records: 500,
        filters: { keyword: 'incident' },
      }),
    ).resolves.toEqual(expect.objectContaining({ id: 21, status: 'pending' }))
    await expect(tracesAPI.getExportTask(21)).resolves.toEqual(
      expect.objectContaining({
        id: 21,
        status: 'succeeded',
        file_size_bytes: 24,
      }),
    )
    await expect(tracesAPI.cancelExportTask(21)).resolves.toEqual({ id: 21, status: 'canceled' })
    await expect(tracesAPI.downloadExportTask(21)).resolves.toEqual({
      blob,
      filename: 'trace-export final.json',
      contentType: 'application/json; charset=utf-8',
    })
  })
})
