import type { AccountPlatform, GroupPlatform } from '@/types'

export type CompatiblePlatform = 'openai_compatible' | 'anthropic_compatible'
export type KnownPlatform = AccountPlatform | GroupPlatform
export type PlatformBadgeKey = KnownPlatform
export type CreateAccountEntryKey =
  | 'claude'
  | 'openai_official'
  | 'openai_compatible'
  | 'anthropic_official'
  | 'anthropic_compatible'
  | 'gemini'
  | 'antigravity'

export interface CreateAccountEntryOption {
  key: CreateAccountEntryKey
  title: string
  description: string
  platform: AccountPlatform
  badge?: string
  defaultCategory?: 'oauth-based' | 'apikey' | 'service_account'
}

export interface CreateAccountEntrySection {
  key: string
  title: string
  description?: string
  options: readonly CreateAccountEntryOption[]
}

const createAccountEntrySections: readonly CreateAccountEntrySection[] = [
  {
    key: 'claude',
    title: 'Claude',
    description: 'Claude Code, Claude Console, Bedrock, Vertex',
    options: [
      {
        key: 'claude',
        title: 'Claude',
        description: 'Claude Code / Setup Token / Console / Bedrock / Vertex',
        platform: 'anthropic',
        defaultCategory: 'oauth-based'
      }
    ]
  },
  {
    key: 'openai_interface',
    title: 'OpenAI \u63a5\u53e3',
    description: '\u5b98\u65b9 OpenAI API\uff0c\u6216\u517c\u5bb9 OpenAI \u534f\u8bae\u7684\u4e09\u65b9\u670d\u52a1',
    options: [
      {
        key: 'openai_official',
        title: 'OpenAI \u5b98\u65b9',
        description: 'OpenAI API Key / OAuth',
        platform: 'openai',
        badge: '\u5b98\u65b9',
        defaultCategory: 'apikey'
      },
      {
        key: 'openai_compatible',
        title: '\u56fd\u6a21 / \u4e09\u65b9\u517c\u5bb9',
        description: 'OpenRouter / SiliconFlow / Volcengine / Alibaba OpenAI-compatible APIs',
        platform: 'openai_compatible',
        badge: '\u517c\u5bb9'
      }
    ]
  },
  {
    key: 'anthropic_interface',
    title: 'Anthropic \u63a5\u53e3',
    description: '\u5b98\u65b9 Anthropic API\uff0c\u6216\u517c\u5bb9 Anthropic \u534f\u8bae\u7684\u4e09\u65b9\u670d\u52a1',
    options: [
      {
        key: 'anthropic_official',
        title: 'Anthropic \u5b98\u65b9',
        description: 'Claude Console API Key',
        platform: 'anthropic',
        badge: '\u5b98\u65b9',
        defaultCategory: 'apikey'
      },
      {
        key: 'anthropic_compatible',
        title: '\u56fd\u6a21 / \u4e09\u65b9\u517c\u5bb9',
        description: 'Anthropic-compatible Claude services',
        platform: 'anthropic_compatible',
        badge: '\u517c\u5bb9'
      }
    ]
  },
  {
    key: 'gemini',
    title: 'Gemini',
    description: 'Google AI Studio, Gemini Web, Vertex',
    options: [
      {
        key: 'gemini',
        title: 'Gemini',
        description: 'Google AI Studio / Gemini Web / Vertex',
        platform: 'gemini',
        defaultCategory: 'oauth-based'
      }
    ]
  },
  {
    key: 'special',
    title: '\u5176\u4ed6',
    options: [
      {
        key: 'antigravity',
        title: 'Antigravity',
        description: 'Special upstream / OAuth',
        platform: 'antigravity',
        defaultCategory: 'oauth-based'
      }
    ]
  }
] as const

const createAccountEntryOptions = createAccountEntrySections.flatMap((section) => [...section.options])
const createAccountEntryMap = Object.fromEntries(
  createAccountEntryOptions.map((option) => [option.key, option])
) as Record<CreateAccountEntryKey, CreateAccountEntryOption>

const KNOWN_PLATFORMS: ReadonlySet<KnownPlatform> = new Set([
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'openai_compatible',
  'anthropic_compatible'
])

const PLATFORM_DISPLAY_NAMES: Record<string, string> = {
  anthropic: 'Anthropic',
  anthropic_compatible: 'Anthropic 接口（兼容）',
  antigravity: 'Antigravity',
  baidu: 'ERNIE',
  claude: 'Claude',
  cohere: 'Cohere',
  deepseek: 'DeepSeek',
  doubao: 'Doubao',
  ernie: 'ERNIE',
  gemini: 'Gemini',
  glm: 'Zhipu GLM',
  grok: 'Grok',
  hunyuan: 'Hunyuan',
  kimi: 'Kimi',
  meta: 'Meta Llama',
  minimax: 'MiniMax',
  mistral: 'Mistral',
  moonshot: 'Moonshot',
  newapi: 'NewAPI',
  'new-api': 'NewAPI',
  ollama: 'Ollama',
  oneapi: 'OneAPI',
  'one-api': 'OneAPI',
  openai: 'OpenAI',
  openai_compatible: 'OpenAI 接口 / 国模',
  openrouter: 'OpenRouter',
  perplexity: 'Perplexity',
  qwen: 'Qwen',
  siliconflow: 'SiliconFlow',
  spark: 'Spark',
  xai: 'xAI',
  yi: 'Yi',
  zhipu: 'Zhipu GLM'
}

const OPENAI_COMPATIBLE_PROVIDER_ALIASES = new Set([
  'baidu',
  'cohere',
  'deepseek',
  'doubao',
  'ernie',
  'glm',
  'grok',
  'hunyuan',
  'kimi',
  'meta',
  'minimax',
  'mistral',
  'moonshot',
  'newapi',
  'new-api',
  'ollama',
  'oneapi',
  'one-api',
  'openrouter',
  'perplexity',
  'qwen',
  'siliconflow',
  'spark',
  'xai',
  'yi',
  'zhipu'
])

const ANTHROPIC_COMPATIBLE_PROVIDER_ALIASES = new Set([
  'anthropic_compatible',
  'claude'
])

export function normalizePlatformKey(platform?: string | null): string {
  return (platform ?? '').trim().toLowerCase()
}

export function isKnownPlatform(platform?: string | null): platform is KnownPlatform {
  return KNOWN_PLATFORMS.has(normalizePlatformKey(platform) as KnownPlatform)
}

export function isCompatiblePlatform(platform?: string | null): platform is CompatiblePlatform {
  const normalized = normalizePlatformKey(platform)
  return normalized === 'openai_compatible' || normalized === 'anthropic_compatible'
}

export function getCreateAccountEntrySections(): readonly CreateAccountEntrySection[] {
  return createAccountEntrySections
}

export function getCreateAccountEntryOption(key: CreateAccountEntryKey): CreateAccountEntryOption {
  return createAccountEntryMap[key]
}

export function resolveCreateAccountEntryKey(
  platform?: string | null,
  accountCategory?: string | null
): CreateAccountEntryKey {
  switch (normalizePlatformKey(platform)) {
    case 'openai':
      return 'openai_official'
    case 'openai_compatible':
      return 'openai_compatible'
    case 'anthropic_compatible':
      return 'anthropic_compatible'
    case 'gemini':
      return 'gemini'
    case 'antigravity':
      return 'antigravity'
    case 'anthropic':
    default:
      return accountCategory === 'apikey' ? 'anthropic_official' : 'claude'
  }
}

export function isOpenAIProtocolPlatform(platform?: string | null): boolean {
  const normalized = normalizePlatformKey(platform)
  return normalized === 'openai' || normalized === 'openai_compatible'
}

export function isAnthropicProtocolPlatform(platform?: string | null): boolean {
  const normalized = normalizePlatformKey(platform)
  return normalized === 'anthropic' || normalized === 'anthropic_compatible'
}

export function resolvePlatformBadgeKey(
  platform?: string | null,
  protocol?: string | null
): PlatformBadgeKey | null {
  const normalizedPlatform = normalizePlatformKey(platform)
  if (isKnownPlatform(normalizedPlatform)) {
    return normalizedPlatform
  }
  if (OPENAI_COMPATIBLE_PROVIDER_ALIASES.has(normalizedPlatform)) {
    return 'openai_compatible'
  }
  if (ANTHROPIC_COMPATIBLE_PROVIDER_ALIASES.has(normalizedPlatform)) {
    return 'anthropic_compatible'
  }

  const normalizedProtocol = normalizePlatformKey(protocol)
  if (normalizedProtocol === 'openai' || normalizedProtocol === 'openai_compatible') {
    return 'openai_compatible'
  }
  if (normalizedProtocol === 'anthropic' || normalizedProtocol === 'anthropic_compatible') {
    return 'anthropic_compatible'
  }
  if (isKnownPlatform(normalizedProtocol)) {
    return normalizedProtocol
  }

  return null
}

export function resolveProtocolBadgeKey(protocol?: string | null): CompatiblePlatform | null {
  const normalized = normalizePlatformKey(protocol)
  if (normalized === 'openai' || normalized === 'openai_compatible') {
    return 'openai_compatible'
  }
  if (normalized === 'anthropic' || normalized === 'anthropic_compatible') {
    return 'anthropic_compatible'
  }
  return null
}

export function getProtocolCompatibleGroupPlatforms(platform?: string | null): GroupPlatform[] {
  const normalized = normalizePlatformKey(platform)
  if (isOpenAIProtocolPlatform(normalized)) {
    return ['openai', 'openai_compatible']
  }
  if (isAnthropicProtocolPlatform(normalized)) {
    return ['anthropic', 'anthropic_compatible']
  }
  if (normalized === 'gemini' || normalized === 'antigravity') {
    return [normalized]
  }
  return []
}

export function getPlatformModelPresetPlatform(platform?: string | null): string {
  const normalized = normalizePlatformKey(platform)
  if (normalized === 'openai_compatible') {
    return 'openai'
  }
  if (normalized === 'anthropic_compatible') {
    return 'anthropic'
  }
  return normalized || 'anthropic'
}

export function getPlatformDefaultBaseUrl(platform?: string | null): string {
  const normalized = normalizePlatformKey(platform)
  if (normalized === 'openai_compatible' || normalized === 'anthropic_compatible') {
    return ''
  }
  if (normalized === 'openai') {
    return 'https://api.openai.com'
  }
  if (normalized === 'gemini') {
    return 'https://generativelanguage.googleapis.com'
  }
  if (normalized === 'antigravity') {
    return 'https://cloudcode-pa.googleapis.com'
  }
  if (normalized === 'anthropic') {
    return 'https://api.anthropic.com'
  }
  return ''
}

export function getPlatformBaseUrlPlaceholder(platform?: string | null): string {
  const normalized = normalizePlatformKey(platform)
  if (normalized === 'openai_compatible') {
    return 'https://api.example.com/v1'
  }
  if (normalized === 'anthropic_compatible') {
    return 'https://api.example.com'
  }
  return getPlatformDefaultBaseUrl(platform)
}

export function getPlatformApiKeyPlaceholder(platform?: string | null): string {
  const normalized = normalizePlatformKey(platform)
  if (normalized === 'gemini') {
    return 'AIza...'
  }
  if (normalized === 'openai') {
    return 'sk-proj-...'
  }
  if (isAnthropicProtocolPlatform(normalized)) {
    return 'sk-ant-...'
  }
  return 'sk-...'
}

export function getPlatformDisplayName(platform?: string | null): string {
  const normalized = normalizePlatformKey(platform)
  if (!normalized) {
    return 'API'
  }
  if (PLATFORM_DISPLAY_NAMES[normalized]) {
    return PLATFORM_DISPLAY_NAMES[normalized]
  }
  return normalized
    .split(/[_\-\s]+/)
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ')
}

export function getCompatiblePlatformDisplayName(platform?: string | null): string {
  const normalized = normalizePlatformKey(platform)
  if (normalized === 'openai_compatible') {
    return 'OpenAI \u63a5\u53e3 / \u56fd\u6a21\u517c\u5bb9'
  }
  if (normalized === 'anthropic_compatible') {
    return 'Anthropic \u63a5\u53e3 / \u4e09\u65b9\u517c\u5bb9'
  }
  return getPlatformDisplayName(normalized)
}

export function getProtocolDisplayName(protocol?: string | null): string {
  const normalized = normalizePlatformKey(protocol)
  if (normalized === 'openai' || normalized === 'openai_compatible') {
    return 'OpenAI Compatible'
  }
  if (normalized === 'anthropic' || normalized === 'anthropic_compatible') {
    return 'Anthropic Compatible'
  }
  return getPlatformDisplayName(normalized)
}
