import { describe, expect, it } from 'vitest'

import {
  apihubGuideDocument,
  apihubGuideHeadings,
  apihubGuideSidebarGroups,
} from '../apihubGuide'

function collectStrings(value: unknown, seen = new Set<unknown>()): string[] {
  if (typeof value === 'string') return [value]
  if (value == null || typeof value !== 'object') return []
  if (seen.has(value)) return []
  seen.add(value)
  if (Array.isArray(value)) return value.flatMap((item) => collectStrings(item, seen))
  return Object.entries(value as Record<string, unknown>).flatMap(([key, nestedValue]) => [key, ...collectStrings(nestedValue, seen)])
}
function normalize(value: unknown): string { return collectStrings(value).join('\n').replace(/\s+/g, ' ').toLowerCase() }

describe('apihub guide content integrity', () => {
  it('ships source-structured Chinese APIHub CodeX guide content', () => {
    const text = normalize(apihubGuideDocument)
    expect(apihubGuideDocument.title).toBe("apihub\u4f7f\u7528\u6559\u7a0b")
    expect(apihubGuideDocument.article.title).toBe("CodeX \u90e8\u7f72\u6307\u5357")
    expect(apihubGuideDocument.article.summary).toBe("\u4f01\u4e1a\u7ea7 AI \u7f16\u7801\u52a9\u624b - \u5b8c\u6574\u90e8\u7f72\u624b\u518c")
    expect(text).toContain("\u5feb\u901f\u5bfc\u822a")
    expect(text).toContain("\u4f7f\u7528 cc-switch \u5feb\u901f\u914d\u7f6e\uff08\u63a8\u8350\uff09")
    expect(text).toContain("linux \u5e73\u53f0")
    expect(text).toContain('wire_api = "responses"')
    expect(text).toContain('requires_openai_auth = true')
    expect(text).not.toContain(['i', 'k', 'u', 'n', 'c', 'o', 'd', 'e'].join(''))
    expect(text).not.toContain(['i', 'k', 'u', 'n', 'c', 'o', 'd', 'i', 'n', 'g'].join(''))
    expect(text).not.toContain(['i', 'k', 'u', 'n'].join(''))
    expect(text).not.toContain("\u552e\u524d\u552e\u540e")
    expect(text).not.toContain("\u5b98\u65b9\u4f18\u8d28\u9879\u76ee")
  })
  it('keeps every sidebar entry on an existing in-page anchor with Chinese labels', () => {
    const headingIds = new Set(apihubGuideHeadings.map((heading) => heading.id))
    const sidebarItems = apihubGuideSidebarGroups.flatMap((group) => group.items)
    expect(apihubGuideSidebarGroups.map((group) => group.title)).toEqual(["\u5feb\u901f\u5f00\u59cb", "\u4f7f\u7528\u6307\u5357", "Node.js \u73af\u5883\u5b89\u88c5", "\u5feb\u901f\u914d\u7f6e\u5de5\u5177", "\u7b2c\u4e09\u65b9\u5e94\u7528", "\u5176\u4ed6"])
    expect(sidebarItems.map((item) => item.text)).toContain("CC-Switch \u914d\u7f6e\u5de5\u5177")
    expect(sidebarItems.map((item) => item.text)).toContain("Hapi \u8fdc\u7a0b\u63a7\u5236")
    expect(sidebarItems.map((item) => item.text)).toContain("Alma \u5ba2\u6237\u7aef")
    expect(sidebarItems.map((item) => item.text)).toContain("CherryStudio \u5ba2\u6237\u7aef")
    expect(sidebarItems.map((item) => item.text)).not.toContain(['CC-Switch ', 'Config'].join(''))
    expect(sidebarItems.map((item) => item.text)).not.toContain(['Hapi Remote ', 'Control'].join(''))
    for (const item of sidebarItems) {
      expect(item.href.startsWith('#')).toBe(true)
      expect(headingIds.has(item.href.slice(1))).toBe(true)
    }
  })
  it('keeps the three CC-Switch guide images referenced', () => {
    const text = normalize(apihubGuideDocument)
    expect(text).toContain('tu9')
    expect(text).toContain('tu10')
    expect(text).toContain('tu11')
  })
})
