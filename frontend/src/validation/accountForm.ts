export type EditableAccountStatus = 'active' | 'inactive' | 'error'

export interface TempUnschedRuleFormInput {
  error_code: number | null
  keywords: string
  duration_minutes: number | null
  description: string
}

export interface TempUnschedRulePayload {
  error_code: number
  keywords: string[]
  duration_minutes: number
  description: string
}

export interface CustomErrorCodeValidationResult {
  ok: boolean
  messageKey?: string
  messageLevel?: 'error' | 'info'
  warningCode?: 429 | 529
}

export interface TempUnschedValidationResult {
  valid: boolean
  rules: TempUnschedRulePayload[]
  errorKey?: string
}

export interface VertexServiceAccountValidationInput {
  projectId: string
  clientEmail: string
  location: string
  hasServiceAccount: boolean
}

export function validateAccountName(
  name: string,
  messageKey = 'admin.accounts.pleaseEnterAccountName'
): string | null {
  return name.trim() ? null : messageKey
}

export function validateCustomErrorCodeCandidate(
  code: number | null,
  selectedCodes: ReadonlyArray<number>
): CustomErrorCodeValidationResult {
  if (code === null || code < 100 || code > 599) {
    return {
      ok: false,
      messageKey: 'admin.accounts.invalidErrorCode',
      messageLevel: 'error'
    }
  }

  if (selectedCodes.includes(code)) {
    return {
      ok: false,
      messageKey: 'admin.accounts.errorCodeExists',
      messageLevel: 'info'
    }
  }

  if (code === 429 || code === 529) {
    return {
      ok: true,
      warningCode: code
    }
  }

  return { ok: true }
}

export function splitTempUnschedKeywords(value: string): string[] {
  return value
    .split(/[,;]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

export function buildTempUnschedRules(
  rules: ReadonlyArray<TempUnschedRuleFormInput>
): TempUnschedRulePayload[] {
  const nextRules: TempUnschedRulePayload[] = []

  for (const rule of rules) {
    const errorCode = Number(rule.error_code)
    const duration = Number(rule.duration_minutes)
    const keywords = splitTempUnschedKeywords(rule.keywords)

    if (!Number.isFinite(errorCode) || errorCode < 100 || errorCode > 599) {
      continue
    }

    if (!Number.isFinite(duration) || duration <= 0) {
      continue
    }

    if (keywords.length === 0) {
      continue
    }

    nextRules.push({
      error_code: Math.trunc(errorCode),
      keywords,
      duration_minutes: Math.trunc(duration),
      description: rule.description.trim()
    })
  }

  return nextRules
}

export function buildTempUnschedValidationResult(
  enabled: boolean,
  rules: ReadonlyArray<TempUnschedRuleFormInput>
): TempUnschedValidationResult {
  if (!enabled) {
    return {
      valid: true,
      rules: []
    }
  }

  const parsedRules = buildTempUnschedRules(rules)
  if (parsedRules.length === 0) {
    return {
      valid: false,
      rules: [],
      errorKey: 'admin.accounts.tempUnschedulable.rulesInvalid'
    }
  }

  return {
    valid: true,
    rules: parsedRules
  }
}

export function isEditableAccountStatus(status: string): status is EditableAccountStatus {
  return status === 'active' || status === 'inactive' || status === 'error'
}

function hasNonEmptyCredential(value: unknown): boolean {
  return typeof value === 'string' && value.trim().length > 0
}

export function validateApiKeyRequirement(
  apiKey: string,
  fallbackApiKey?: unknown,
  messageKey = 'admin.accounts.apiKeyIsRequired'
): string | null {
  return apiKey.trim() || hasNonEmptyCredential(fallbackApiKey) ? null : messageKey
}

export function validateVertexServiceAccount(
  input: VertexServiceAccountValidationInput
): string | null {
  if (!input.projectId.trim()) {
    return 'admin.accounts.vertexSaJsonMissingProjectId'
  }

  if (!input.clientEmail.trim()) {
    return 'admin.accounts.vertexSaJsonMissingClientEmail'
  }

  if (!input.location.trim()) {
    return 'admin.accounts.vertexLocationRequired'
  }

  if (!input.hasServiceAccount) {
    return 'admin.accounts.vertexSaJsonRequired'
  }

  return null
}
