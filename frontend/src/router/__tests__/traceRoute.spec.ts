import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

function extractRouteBlock(path: string): string {
  const escapedPath = path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = routerSource.match(new RegExp(`\\{\\s*path: '${escapedPath}'[\\s\\S]*?\\n  \\},`))
  return match?.[0] ?? ''
}

describe('trace router contracts', () => {
  it('registers the admin traces route as an authenticated admin page', () => {
    const routeBlock = extractRouteBlock('/admin/traces')

    expect(routeBlock).toContain("path: '/admin/traces'")
    expect(routeBlock).toContain("name: 'AdminTraces'")
    expect(routeBlock).toContain("import('@/views/admin/TracesView.vue')")
    expect(routeBlock).toContain('requiresAuth: true')
    expect(routeBlock).toContain('requiresAdmin: true')
    expect(routeBlock).toContain("titleKey: 'admin.traces.title'")
    expect(routeBlock).toContain("descriptionKey: 'admin.traces.description'")
  })
})
