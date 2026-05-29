import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppSidebar trace navigation', () => {
  it('shows an admin-facing trace entry in admin navigation', () => {
    const adminNavItemsBlock = componentSource.match(/const adminNavItems = computed\(\(\): NavItem\[] => \{[\s\S]*?return visible\n\}\)/)

    expect(adminNavItemsBlock?.[0]).toContain("path: '/admin/traces'")
    expect(adminNavItemsBlock?.[0]).toContain("label: t('nav.traces')")
  })
})
