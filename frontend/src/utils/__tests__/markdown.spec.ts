import { describe, expect, it } from 'vitest'

import { renderMarkdown } from '../markdown'

describe('renderMarkdown', () => {
  it('renders markdown and strips unsafe markup', () => {
    const rendered = renderMarkdown('hello\nworld<script>alert(1)</script>')

    expect(rendered).toContain('hello<br>world')
    expect(rendered).not.toContain('<script>')
  })

  it('allows caller-provided sanitize extensions', () => {
    const rendered = renderMarkdown('<iframe src="https://example.com"></iframe>', {
      ADD_TAGS: ['iframe'],
      ADD_ATTR: ['src'],
    })

    expect(rendered).toContain('<iframe')
    expect(rendered).toContain('src="https://example.com"')
  })
})
