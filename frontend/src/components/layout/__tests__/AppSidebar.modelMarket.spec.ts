import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppSidebar model market navigation', () => {
  it('shows a user-facing model market entry in self navigation', () => {
    const buildSelfNavItemsBlock = componentSource.match(/function buildSelfNavItems\(withDashboard: boolean\): NavItem\[] \{[\s\S]*?return items\n\}/)

    expect(buildSelfNavItemsBlock?.[0]).toContain("path: '/model-market'")
  })

  it('shows an admin-facing model market entry in admin navigation', () => {
    const adminNavItemsBlock = componentSource.match(/const adminNavItems = computed\(\(\): NavItem\[] => \{[\s\S]*?return visible\n\}\)/)

    expect(adminNavItemsBlock?.[0]).toContain("path: '/admin/model-market'")
  })
})
