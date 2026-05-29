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

import backupAPI from '@/api/admin/backup'

describe('admin backup api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('downloads requirement export from the trace endpoint', async () => {
    const blob = new Blob(['{"task_id":"task_123"}'], {
      type: 'application/json',
    })

    get.mockResolvedValueOnce({ data: blob })

    await expect(backupAPI.exportRequirementJson()).resolves.toBe(blob)
    expect(get).toHaveBeenCalledTimes(1)
    expect(get).toHaveBeenCalledWith('/admin/traces/export', {
      responseType: 'blob',
    })
  })

  it('does not swallow non-404 requirement export failures', async () => {
    const error = {
      isAxiosError: true,
      response: { status: 500 },
      message: 'boom',
    }
    get.mockRejectedValueOnce(error)

    await expect(backupAPI.exportRequirementJson()).rejects.toBe(error)
    expect(get).toHaveBeenCalledTimes(1)
  })
})
