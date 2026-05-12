const ACCESS_TOKEN_KEY = 'auth_token'
const AUTH_USER_KEY = 'auth_user'
const REFRESH_TOKEN_KEY = 'refresh_token'
const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'

let inMemoryAccessToken: string | null = null
let inMemoryRefreshToken: string | null = null

function readSessionStorage(key: string): string | null {
  try {
    return sessionStorage.getItem(key)
  } catch {
    return null
  }
}

function writeSessionStorage(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value)
  } catch {
    // Ignore storage failures
  }
}

function removeSessionStorage(key: string): void {
  try {
    sessionStorage.removeItem(key)
  } catch {
    // Ignore storage failures
  }
}

function readLocalStorage(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeLocalStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    // Ignore storage failures
  }
}

function removeLocalStorage(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    // Ignore storage failures
  }
}

export function setAccessToken(token: string): void {
  const value = token.trim()
  inMemoryAccessToken = value || null

  if (value) {
    writeSessionStorage(ACCESS_TOKEN_KEY, value)
  } else {
    removeSessionStorage(ACCESS_TOKEN_KEY)
  }

  removeLocalStorage(ACCESS_TOKEN_KEY)
}

export function getAccessToken(): string | null {
  if (inMemoryAccessToken) {
    return inMemoryAccessToken
  }

  const persistedToken = readSessionStorage(ACCESS_TOKEN_KEY)?.trim()
  if (!persistedToken) {
    return null
  }

  inMemoryAccessToken = persistedToken
  return persistedToken
}

export function clearAccessToken(): void {
  inMemoryAccessToken = null
  removeSessionStorage(ACCESS_TOKEN_KEY)
  removeLocalStorage(ACCESS_TOKEN_KEY)
}

export function setRefreshToken(token: string): void {
  const value = token.trim()
  inMemoryRefreshToken = value || null
  removeLocalStorage(REFRESH_TOKEN_KEY)
}

export function getRefreshToken(): string | null {
  return inMemoryRefreshToken
}

export function clearRefreshToken(): void {
  inMemoryRefreshToken = null
  removeLocalStorage(REFRESH_TOKEN_KEY)
}

export function setPersistedAuthUser(user: unknown): void {
  writeLocalStorage(AUTH_USER_KEY, JSON.stringify(user))
}

export function getPersistedAuthUser(): string | null {
  return readLocalStorage(AUTH_USER_KEY)
}

export function clearPersistedAuthUser(): void {
  removeLocalStorage(AUTH_USER_KEY)
}

export function setTokenExpiresAt(timestampMs: number): void {
  if (!Number.isFinite(timestampMs) || timestampMs <= 0) {
    removeLocalStorage(TOKEN_EXPIRES_AT_KEY)
    return
  }

  writeLocalStorage(TOKEN_EXPIRES_AT_KEY, String(Math.trunc(timestampMs)))
}

export function setTokenExpiresIn(expiresInSeconds: number): void {
  if (!Number.isFinite(expiresInSeconds) || expiresInSeconds <= 0) {
    removeLocalStorage(TOKEN_EXPIRES_AT_KEY)
    return
  }

  setTokenExpiresAt(Date.now() + expiresInSeconds * 1000)
}

export function getTokenExpiresAt(): number | null {
  const rawValue = readLocalStorage(TOKEN_EXPIRES_AT_KEY)
  if (!rawValue) {
    return null
  }

  const parsedValue = Number.parseInt(rawValue, 10)
  return Number.isFinite(parsedValue) ? parsedValue : null
}

export function hasPersistedSessionHint(): boolean {
  return Boolean(getPersistedAuthUser()) && getTokenExpiresAt() !== null
}

export function clearPersistedSession(): void {
  clearAccessToken()
  clearRefreshToken()
  clearPersistedAuthUser()
  removeLocalStorage(TOKEN_EXPIRES_AT_KEY)
}
