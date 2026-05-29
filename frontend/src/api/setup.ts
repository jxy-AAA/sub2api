/**
 * Setup API endpoints
 */
import axios from 'axios'

// Create a separate client for setup endpoints (not under /api/v1)
const setupClient = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

const SETUP_TOKEN_STORAGE_KEY = 'sub2api_setup_token'
const SETUP_TOKEN_HEADER = 'X-Setup-Token'

export interface SetupStatus {
  needs_setup: boolean
  step: string
  setup_token_required?: boolean
  setup_token_available?: boolean
}

export interface DatabaseConfig {
  host: string
  port: number
  user: string
  password: string
  dbname: string
  sslmode: string
}

export interface RedisConfig {
  host: string
  port: number
  password: string
  db: number
  enable_tls: boolean
}

export interface AdminConfig {
  email: string
  password: string
}

export interface ServerConfig {
  host: string
  port: number
  mode: string
}

export interface InstallRequest {
  database: DatabaseConfig
  redis: RedisConfig
  admin: AdminConfig
  server: ServerConfig
}

export interface InstallResponse {
  message: string
  restart: boolean
}

export function getSetupToken(): string {
  if (typeof window === 'undefined') return ''
  return window.sessionStorage.getItem(SETUP_TOKEN_STORAGE_KEY) || ''
}

export function setSetupToken(token: string): void {
  if (typeof window === 'undefined') return
  const normalized = token.trim()
  if (!normalized) {
    window.sessionStorage.removeItem(SETUP_TOKEN_STORAGE_KEY)
    return
  }
  window.sessionStorage.setItem(SETUP_TOKEN_STORAGE_KEY, normalized)
}

setupClient.interceptors.request.use((config) => {
  const token = getSetupToken()
  if (token) {
    config.headers = config.headers ?? {}
    config.headers[SETUP_TOKEN_HEADER] = token
  }
  return config
})

/**
 * Get setup status
 */
export async function getSetupStatus(): Promise<SetupStatus> {
  const response = await setupClient.get('/setup/status')
  return response.data.data
}

/**
 * Test database connection
 */
export async function testDatabase(config: DatabaseConfig): Promise<void> {
  await setupClient.post('/setup/test-db', config)
}

/**
 * Test Redis connection
 */
export async function testRedis(config: RedisConfig): Promise<void> {
  await setupClient.post('/setup/test-redis', config)
}

/**
 * Perform installation
 */
export async function install(config: InstallRequest): Promise<InstallResponse> {
  const response = await setupClient.post('/setup/install', config)
  return response.data.data
}
