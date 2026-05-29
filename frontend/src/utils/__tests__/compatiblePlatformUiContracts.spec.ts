import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const rootDir = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const utilsPlatformsPath = resolve(rootDir, '../utils/platforms.ts')
const badgePath = resolve(rootDir, '../components/common/PlatformTypeBadge.vue')
const userModelMarketPath = resolve(rootDir, '../views/user/ModelMarketView.vue')
const zhLocalePath = resolve(rootDir, '../i18n/locales/zh.ts')
const enLocalePath = resolve(rootDir, '../i18n/locales/en.ts')

const utilsPlatformsSource = readFileSync(utilsPlatformsPath, 'utf8')
const badgeSource = readFileSync(badgePath, 'utf8')
const userModelMarketSource = readFileSync(userModelMarketPath, 'utf8')
const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')
const enLocaleSource = readFileSync(enLocalePath, 'utf8')

describe('compatible platform UI contracts', () => {
  it('ships locale entries for compatible account/group labels and upstream copy', () => {
    for (const source of [zhLocaleSource, enLocaleSource]) {
      expect(source).toContain('openai_compatible')
      expect(source).toContain('anthropic_compatible')
      expect(source).toContain('compatibleProviders')
      expect(source).toContain('openaiDescription')
      expect(source).toContain('anthropicDescription')
      expect(source).toContain('headersPlaceholder')
      expect(source).toContain('headersHint')
      expect(source).toContain('invalidHeaders')
    }

    expect(zhLocaleSource).toContain("openai_compatible: 'OpenAI 接口 / 国模'")
    expect(zhLocaleSource).toContain("anthropic_compatible: 'Anthropic 接口'")
    expect(zhLocaleSource).toContain('国模/三方兼容')
    expect(enLocaleSource).toContain("openai_compatible: 'OpenAI API'")
    expect(enLocaleSource).toContain("anthropic_compatible: 'Anthropic API'")
  })

  it('keeps account badge labels translation-backed with friendly compatible fallbacks', () => {
    expect(badgeSource).toContain('const key = `admin.accounts.platforms.${props.platform}`')
    expect(badgeSource).toContain('if (translated !== key)')
    expect(badgeSource).toContain('isCompatiblePlatform(props.platform)')
    expect(badgeSource).toContain('getProtocolDisplayName(props.platform)')
    expect(utilsPlatformsSource).toContain("case 'openai_compatible'")
    expect(utilsPlatformsSource).toContain("case 'anthropic_compatible'")
    expect(utilsPlatformsSource).not.toContain("case 'internal'")
    expect(utilsPlatformsSource).not.toContain("case 'admin'")
  })

  it('uses model-market protocol translations for friendly protocol labels', () => {
    expect(userModelMarketSource).toContain("t('modelMarket.protocolLabels.openai')")
    expect(userModelMarketSource).toContain("t('modelMarket.protocolLabels.anthropic')")
    expect(zhLocaleSource).toContain("openai: 'OpenAI 接口'")
    expect(zhLocaleSource).toContain("anthropic: 'Anthropic 接口'")
    expect(enLocaleSource).toContain("openai: 'OpenAI API'")
    expect(enLocaleSource).toContain("anthropic: 'Anthropic API'")
  })
})
