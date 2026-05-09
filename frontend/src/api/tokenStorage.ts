let inMemoryAccessToken: string | null = null

export function setAccessToken(token: string): void {
  const value = token.trim()
  inMemoryAccessToken = value || null
}

export function getAccessToken(): string | null {
  return inMemoryAccessToken
}

export function clearAccessToken(): void {
  inMemoryAccessToken = null
  localStorage.removeItem('auth_token')
}
