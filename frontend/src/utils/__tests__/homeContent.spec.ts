import { describe, expect, it } from 'vitest'
import { renderSafeHomeContent, resolveSafeHomeIframeUrl } from '@/utils/homeContent'

describe('homeContent', () => {
  it('accepts only http/https iframe URLs', () => {
    expect(resolveSafeHomeIframeUrl('https://example.com/page')).toBe('https://example.com/page')
    expect(resolveSafeHomeIframeUrl('javascript:alert(1)')).toBeNull()
    expect(resolveSafeHomeIframeUrl('data:text/html,test')).toBeNull()
  })

  it('sanitizes script payloads in markdown/html mode', () => {
    const html = renderSafeHomeContent('# title\n<script>alert(1)</script><img src=x onerror=alert(1)>')
    expect(html).not.toContain('<script')
    expect(html).not.toContain('onerror=')
  })

  it('drops unsafe iframe sources while keeping safe ones', () => {
    const html = renderSafeHomeContent(
      '<iframe src="javascript:alert(1)"></iframe><iframe src="https://example.com/embed"></iframe>',
    )
    expect(html).not.toContain('javascript:')
    expect(html).toContain('https://example.com/embed')
  })
})
