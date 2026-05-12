import { marked } from 'marked'
import DOMPurify from 'dompurify'

type SanitizeConfig = Parameters<typeof DOMPurify.sanitize>[1]

marked.setOptions({
  breaks: true,
  gfm: true,
})

export function renderMarkdown(content: string, sanitizeConfig?: SanitizeConfig): string {
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html, sanitizeConfig)
}
