import DOMPurify from 'dompurify'
import { marked } from 'marked'

const SAFE_IFRAME_PROTOCOLS = new Set(['http:', 'https:'])

export function resolveSafeHomeIframeUrl(content: string): string | null {
  const raw = content.trim()
  if (!raw) return null
  if (!raw.startsWith('http://') && !raw.startsWith('https://')) {
    return null
  }

  try {
    const parsed = new URL(raw)
    if (!SAFE_IFRAME_PROTOCOLS.has(parsed.protocol)) {
      return null
    }
    return parsed.toString()
  } catch {
    return null
  }
}

function isSafeIframeSource(source: string): boolean {
  try {
    const parsed = new URL(source, 'http://localhost')
    return SAFE_IFRAME_PROTOCOLS.has(parsed.protocol)
  } catch {
    return false
  }
}

export function renderSafeHomeContent(content: string): string {
  const raw = content.trim()
  if (!raw) return ''

  const html = marked.parse(raw) as string
  const sanitized = DOMPurify.sanitize(html, {
    ADD_TAGS: ['iframe'],
    ADD_ATTR: ['allow', 'allowfullscreen', 'frameborder', 'referrerpolicy', 'sandbox', 'src'],
    FORBID_TAGS: ['script', 'style'],
  })

  const doc = new DOMParser().parseFromString(sanitized, 'text/html')
  doc.querySelectorAll('iframe').forEach((iframe) => {
    const src = iframe.getAttribute('src')?.trim() || ''
    if (!src || !isSafeIframeSource(src)) {
      iframe.remove()
      return
    }
    iframe.setAttribute('referrerpolicy', 'no-referrer')
    if (!iframe.hasAttribute('sandbox')) {
      iframe.setAttribute('sandbox', 'allow-same-origin allow-scripts allow-popups allow-forms')
    }
  })

  return doc.body.innerHTML
}
