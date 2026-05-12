import { describe, expect, it } from 'vitest'
import {
  buildTempUnschedRules,
  buildTempUnschedValidationResult,
  isEditableAccountStatus,
  validateApiKeyRequirement,
  validateCustomErrorCodeCandidate,
  validateVertexServiceAccount
} from '@/validation/accountForm'

describe('accountForm validation helpers', () => {
  it('builds valid temp unschedulable rules only', () => {
    expect(
      buildTempUnschedRules([
        {
          error_code: 429,
          keywords: 'quota, burst',
          duration_minutes: 30,
          description: 'limit'
        },
        {
          error_code: 99,
          keywords: '',
          duration_minutes: 0,
          description: 'invalid'
        }
      ])
    ).toEqual([
      {
        error_code: 429,
        keywords: ['quota', 'burst'],
        duration_minutes: 30,
        description: 'limit'
      }
    ])
  })

  it('reports an error key when enabled temp unsched rules are invalid', () => {
    expect(
      buildTempUnschedValidationResult(true, [
        {
          error_code: null,
          keywords: '',
          duration_minutes: null,
          description: ''
        }
      ])
    ).toEqual({
      valid: false,
      rules: [],
      errorKey: 'admin.accounts.tempUnschedulable.rulesInvalid'
    })
  })

  it('validates duplicate and warning custom error codes', () => {
    expect(validateCustomErrorCodeCandidate(429, [])).toEqual({
      ok: true,
      warningCode: 429
    })
    expect(validateCustomErrorCodeCandidate(401, [401])).toEqual({
      ok: false,
      messageKey: 'admin.accounts.errorCodeExists',
      messageLevel: 'info'
    })
  })

  it('checks reusable api key and vertex service account validation', () => {
    expect(validateApiKeyRequirement('', undefined)).toBe('admin.accounts.apiKeyIsRequired')
    expect(validateApiKeyRequirement('', 'existing-key')).toBeNull()
    expect(
      validateVertexServiceAccount({
        projectId: 'project-id',
        clientEmail: 'svc@example.com',
        location: '',
        hasServiceAccount: true
      })
    ).toBe('admin.accounts.vertexLocationRequired')
  })

  it('accepts only editable account statuses', () => {
    expect(isEditableAccountStatus('active')).toBe(true)
    expect(isEditableAccountStatus('paused')).toBe(false)
  })
})
