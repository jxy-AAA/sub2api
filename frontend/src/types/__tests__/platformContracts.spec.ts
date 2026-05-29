import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const typesPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const typesSource = readFileSync(typesPath, 'utf8')
const platformColorsPath = resolve(dirname(fileURLToPath(import.meta.url)), '../../utils/platformColors.ts')
const platformColorsSource = readFileSync(platformColorsPath, 'utf8')

function extractTypeAlias(source: string, typeName: string): string {
  const match = source.match(new RegExp(`export type ${typeName} =[\\s\\S]*?(?=\\nexport |\\ninterface |\\ntype |$)`))
  return match?.[0] ?? ''
}

function extractInterface(source: string, interfaceName: string): string {
  const match = source.match(new RegExp(`export interface ${interfaceName} \\{[\\s\\S]*?\\n\\}`))
  return match?.[0] ?? ''
}

describe('frontend platform contracts', () => {
  it('extends shared account and group platform unions for compatible providers', () => {
    const accountPlatformAlias = extractTypeAlias(typesSource, 'AccountPlatform')
    const groupPlatformAlias = extractTypeAlias(typesSource, 'GroupPlatform')

    for (const platform of [
      "'anthropic'",
      "'openai'",
      "'gemini'",
      "'antigravity'",
      "'openai_compatible'",
      "'anthropic_compatible'",
    ]) {
      expect(accountPlatformAlias).toContain(platform)
      expect(groupPlatformAlias).toContain(platform)
    }
  })

  it('keeps platform color helpers in sync with compatible providers', () => {
    const platformAlias = extractTypeAlias(platformColorsSource, 'Platform')
    const isPlatformBlock = platformColorsSource.match(/function isPlatform\(p: string\): p is Platform \{[\s\S]*?\n\}/)?.[0] ?? ''

    for (const platform of [
      "'anthropic'",
      "'openai'",
      "'antigravity'",
      "'gemini'",
      "'openai_compatible'",
      "'anthropic_compatible'",
    ]) {
      expect(platformAlias).toContain(platform)
      expect(isPlatformBlock).toContain(platform)
    }
  })

  it('keeps account request payload contracts compatible-provider ready', () => {
    const createRequest = extractInterface(typesSource, 'CreateAccountRequest')
    const updateRequest = extractInterface(typesSource, 'UpdateAccountRequest')
    const mixedChannelRequest = extractInterface(typesSource, 'CheckMixedChannelRequest')

    expect(createRequest).toContain('platform: AccountPlatform')
    expect(createRequest).toContain('group_ids?: number[]')

    expect(updateRequest).toContain('group_ids?: number[]')

    expect(mixedChannelRequest).toContain('platform: AccountPlatform')
    expect(mixedChannelRequest).toContain('group_ids: number[]')
  })
})
