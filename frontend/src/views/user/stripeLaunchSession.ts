export interface StripeLaunchSession {
  orderId: number
  clientSecret: string
  method: string
  createdAt: number
}

interface StripeLaunchSessionPayload {
  orderId: number
  clientSecret: string
  method?: string
}

interface StripeLaunchSessionReadOptions {
  expectedOrderId?: number
  expectedMethod?: string
  now?: number
}

const STRIPE_LAUNCH_SESSION_PREFIX = 'payment.stripe.launch.'
export const STRIPE_LAUNCH_SESSION_MAX_AGE_MS = 30 * 60 * 1000

const memoryStripeLaunchSessions = new Map<string, StripeLaunchSession>()

function createSessionId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  return `stripe_${Date.now()}_${Math.random().toString(36).slice(2, 12)}`
}

function getStorageKey(sessionId: string): string {
  return `${STRIPE_LAUNCH_SESSION_PREFIX}${sessionId}`
}

function isStoredSession(value: unknown): value is StripeLaunchSession {
  if (!value || typeof value !== 'object') {
    return false
  }

  const candidate = value as Partial<StripeLaunchSession>
  return Number.isFinite(candidate.orderId)
    && typeof candidate.clientSecret === 'string'
    && typeof candidate.method === 'string'
    && Number.isFinite(candidate.createdAt)
}

function writeStoredSession(sessionId: string, session: StripeLaunchSession): void {
  memoryStripeLaunchSessions.set(sessionId, session)

  try {
    sessionStorage.setItem(getStorageKey(sessionId), JSON.stringify(session))
  } catch {
    // Ignore storage failures; the in-memory session still supports SPA navigation.
  }
}

function readStoredSession(sessionId: string): StripeLaunchSession | null {
  const inMemory = memoryStripeLaunchSessions.get(sessionId)
  if (inMemory) {
    return inMemory
  }

  try {
    const raw = sessionStorage.getItem(getStorageKey(sessionId))
    if (!raw) {
      return null
    }

    const parsed = JSON.parse(raw) as unknown
    return isStoredSession(parsed) ? parsed : null
  } catch {
    return null
  }
}

function removeStoredSession(sessionId: string): void {
  memoryStripeLaunchSessions.delete(sessionId)

  try {
    sessionStorage.removeItem(getStorageKey(sessionId))
  } catch {
    // Ignore storage failures.
  }
}

function pruneExpiredSessions(now: number): void {
  for (const [sessionId, session] of memoryStripeLaunchSessions.entries()) {
    if (session.createdAt + STRIPE_LAUNCH_SESSION_MAX_AGE_MS <= now) {
      memoryStripeLaunchSessions.delete(sessionId)
    }
  }

  try {
    for (let index = sessionStorage.length - 1; index >= 0; index -= 1) {
      const key = sessionStorage.key(index)
      if (!key || !key.startsWith(STRIPE_LAUNCH_SESSION_PREFIX)) {
        continue
      }

      const raw = sessionStorage.getItem(key)
      if (!raw) {
        sessionStorage.removeItem(key)
        continue
      }

      try {
        const parsed = JSON.parse(raw) as unknown
        if (!isStoredSession(parsed) || parsed.createdAt + STRIPE_LAUNCH_SESSION_MAX_AGE_MS <= now) {
          sessionStorage.removeItem(key)
        }
      } catch {
        sessionStorage.removeItem(key)
      }
    }
  } catch {
    // Ignore storage failures.
  }
}

export function createStripeLaunchSession(
  payload: StripeLaunchSessionPayload,
  options: { now?: number } = {},
): string {
  const now = options.now ?? Date.now()
  const clientSecret = payload.clientSecret.trim()
  if (!Number.isFinite(payload.orderId) || payload.orderId <= 0 || clientSecret === '') {
    throw new Error('Invalid Stripe launch session payload')
  }

  pruneExpiredSessions(now)

  const sessionId = createSessionId()
  writeStoredSession(sessionId, {
    orderId: payload.orderId,
    clientSecret,
    method: (payload.method || '').trim(),
    createdAt: now,
  })
  return sessionId
}

export function consumeStripeLaunchSession(
  sessionId: string,
  options: StripeLaunchSessionReadOptions = {},
): StripeLaunchSession | null {
  const normalizedSessionId = sessionId.trim()
  if (!normalizedSessionId) {
    return null
  }

  const session = readStoredSession(normalizedSessionId)
  removeStoredSession(normalizedSessionId)
  if (!session) {
    return null
  }

  const now = options.now ?? Date.now()
  if (session.createdAt + STRIPE_LAUNCH_SESSION_MAX_AGE_MS <= now) {
    return null
  }
  if (options.expectedOrderId && session.orderId !== options.expectedOrderId) {
    return null
  }
  if (options.expectedMethod != null && options.expectedMethod.trim() !== '' && session.method !== options.expectedMethod.trim()) {
    return null
  }

  return session
}

export function clearStripeLaunchSession(sessionId: string): void {
  const normalizedSessionId = sessionId.trim()
  if (!normalizedSessionId) {
    return
  }

  removeStoredSession(normalizedSessionId)
}
