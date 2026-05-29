import { describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

import AccountTableFilters from '../AccountTableFilters.vue'

describe('AccountTableFilters platform options', () => {
  it('exposes compatible platform filters and groups them by platform label', () => {
    const wrapper = shallowMount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          type: '',
          status: '',
          privacy_mode: '',
          group: '',
        },
        groups: [
          {
            id: 1,
            name: 'Claude 兼容组',
            platform: 'anthropic_compatible',
          },
          {
            id: 2,
            name: '国模兼容组',
            platform: 'openai_compatible',
          },
        ],
      },
    })

    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(5)

    const platformOptions = selects[0].props('options') as Array<{ value: string; label: string }>
    expect(platformOptions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ value: 'openai_compatible', label: 'OpenAI Compatible' }),
        expect.objectContaining({ value: 'anthropic_compatible', label: 'Anthropic Compatible' }),
      ]),
    )

    const groupOptions = selects[4].props('options') as Array<Record<string, unknown>>
    expect(groupOptions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ kind: 'group', label: 'Anthropic Compatible' }),
        expect.objectContaining({ kind: 'group', label: 'OpenAI Compatible' }),
        expect.objectContaining({ value: '1', label: 'Claude 兼容组 (Anthropic Compatible)' }),
        expect.objectContaining({ value: '2', label: '国模兼容组 (OpenAI Compatible)' }),
      ]),
    )
  })
})
