import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

import PlatformTypeBadge from '../PlatformTypeBadge.vue'

describe('PlatformTypeBadge', () => {
  it('renders compatible platform labels with compatible color tone', () => {
    const wrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai_compatible',
        type: 'apikey',
      },
      global: {
        stubs: {
          Icon: {
            template: '<span />',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('OpenAI Compatible')
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.html()).toContain('text-teal-600')
  })
})
