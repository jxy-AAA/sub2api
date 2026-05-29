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

describe('model market router contracts', () => {
  it('registers the user model market route as an authenticated user page', () => {
    const routeBlock = extractRouteBlock('/model-market')

    expect(routeBlock).toContain("path: '/model-market'")
    expect(routeBlock).toContain('requiresAuth: true')
    expect(routeBlock).toContain('requiresAdmin: false')
  })

  it('registers the admin model market route as an authenticated admin page', () => {
    const routeBlock = extractRouteBlock('/admin/model-market')

    expect(routeBlock).toContain("path: '/admin/model-market'")
    expect(routeBlock).toContain('requiresAuth: true')
    expect(routeBlock).toContain('requiresAdmin: true')
  })
})
