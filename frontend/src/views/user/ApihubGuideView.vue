<template>
  <AppLayout>
    <div class="mx-auto max-w-[1680px]">
      <div class="rounded-3xl border border-gray-200 bg-white/90 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/90">
        <div class="border-b border-gray-200 px-4 py-4 dark:border-dark-700 sm:px-6 lg:hidden">
          <div class="flex flex-wrap items-center gap-3">
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-xl border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
              @click="showMobileSidebar = !showMobileSidebar"
            >
              <Icon name="book" size="sm" />
              <span>文档导航</span>
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-xl border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
              @click="showMobileToc = !showMobileToc"
            >
              <Icon name="document" size="sm" />
              <span>本页目录</span>
            </button>
          </div>
        </div>

        <div class="grid min-h-[calc(100vh-12rem)] grid-cols-1 lg:grid-cols-[280px_minmax(0,1fr)] xl:grid-cols-[280px_minmax(0,1fr)_240px]">
          <aside
            class="border-b border-gray-200 bg-gray-50/80 dark:border-dark-700 dark:bg-dark-950/40 lg:border-b-0 lg:border-r"
            :class="showMobileSidebar ? 'block' : 'hidden lg:block'"
          >
            <div class="sticky top-0 max-h-[calc(100vh-4rem)] overflow-y-auto px-4 py-6 sm:px-5">
              <div class="mb-5">
                <p class="text-xs font-semibold uppercase tracking-[0.24em] text-primary-600 dark:text-primary-300">
                  {{ article.providerName }} 文档
                </p>
                <h2 class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                  apihub使用教程
                </h2>
              </div>

              <nav class="space-y-5">
                <div
                  v-for="group in sidebarGroups"
                  :key="group.title"
                  class="space-y-2"
                >
                  <h3 class="px-2 text-xs font-semibold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
                    {{ group.title }}
                  </h3>
                  <ul class="space-y-1.5">
                    <li
                      v-for="item in group.items"
                      :key="`${group.title}-${item.text}-${item.href}`"
                    >
                      <button
                        type="button"
                        class="flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-sm transition"
                        :class="navItemClass(item.href)"
                        @click="handleNavClick(item.href)"
                      >
                        <span class="truncate">{{ item.text }}</span>
                        <span
                          v-if="'badge' in item && item.badge"
                          class="ml-3 rounded-full bg-primary-50 px-2 py-0.5 text-[11px] font-semibold text-primary-700 dark:bg-primary-500/10 dark:text-primary-300"
                        >
                          {{ item.badge }}
                        </span>
                      </button>
                    </li>
                  </ul>
                </div>
              </nav>
            </div>
          </aside>

          <main class="min-w-0 px-4 py-6 sm:px-6 lg:px-10 lg:py-10">
            <div
              v-if="showMobileToc"
              class="mb-6 rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-950/50 xl:hidden"
            >
              <div class="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <Icon name="document" size="sm" />
                <span>本页目录</span>
              </div>
              <nav class="space-y-1.5">
                <button
                  v-for="entry in tocEntries"
                  :key="entry.id"
                  type="button"
                  class="block w-full rounded-lg px-3 py-2 text-left text-sm transition"
                  :class="tocItemClass(entry.id, entry.depth)"
                  @click="scrollToHeading(entry.id)"
                >
                  {{ entry.text }}
                </button>
              </nav>
            </div>

            <article ref="articleRef" class="mx-auto max-w-4xl">
              <header class="mb-10 border-b border-gray-200 pb-8 dark:border-dark-700">
                <div class="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">
                  <Icon name="sparkles" size="xs" />
                  <span>{{ article.kicker || article.description || 'APIHub 部署文档' }}</span>
                </div>
                <h1 class="mt-4 text-3xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-4xl">
                  {{ article.title }}
                </h1>
                <p
                  v-if="article.summary"
                  class="mt-4 max-w-3xl text-base leading-7 text-gray-600 dark:text-gray-300 sm:text-lg"
                >
                  {{ article.summary }}
                </p>
                <div
                  v-if="article.meta?.length"
                  class="mt-5 flex flex-wrap gap-3 text-sm text-gray-500 dark:text-gray-400"
                >
                  <span
                    v-for="meta in article.meta"
                    :key="`${meta.label}-${meta.value}`"
                    class="inline-flex items-center gap-2 rounded-full border border-gray-200 px-3 py-1 dark:border-dark-600"
                  >
                    <span class="font-medium text-gray-700 dark:text-gray-200">{{ meta.label }}</span>
                    <span>{{ meta.value }}</span>
                  </span>
                </div>
              </header>

              <section
                v-for="section in article.sections"
                :id="section.id"
                :key="section.id"
                class="scroll-mt-24 border-b border-gray-100 py-8 last:border-b-0 dark:border-dark-800"
              >
                <component
                  :is="headingTag(section.level)"
                  v-if="section.renderTitle !== false"
                  class="group mb-5 flex items-start gap-3 font-semibold tracking-tight text-gray-900 dark:text-white"
                  :class="headingClass(section.level)"
                >
                  <button
                    type="button"
                    class="text-left transition hover:text-primary-600 dark:hover:text-primary-300"
                    @click="scrollToHeading(section.id)"
                  >
                    {{ section.title }}
                  </button>
                  <button
                    type="button"
                    class="mt-1 opacity-0 transition group-hover:opacity-100"
                    :title="copiedAnchor === section.id ? '链接已复制' : '复制本节链接'"
                    @click="copyAnchor(section.id)"
                  >
                    <Icon
                      :name="copiedAnchor === section.id ? 'checkCircle' : 'link'"
                      size="sm"
                      class="text-gray-400 hover:text-primary-600 dark:hover:text-primary-300"
                    />
                  </button>
                </component>

                <div class="space-y-6">
                  <template
                    v-for="(block, blockIndex) in section.blocks"
                    :key="`${section.id}-${blockIndex}`"
                  >
                    <p
                      v-if="block.type === 'paragraph'"
                      class="text-[15px] leading-7 text-gray-700 dark:text-gray-300 sm:text-base"
                    >
                      <template
                        v-for="inline in normalizeInlineContent(block.content)"
                        :key="`${section.id}-${blockIndex}-${inline.text}-${inline.link || 'plain'}`"
                      >
                        <component
                          :is="inline.link ? 'a' : 'span'"
                          :href="inline.link ? resolveLinkHref(inline.link) : undefined"
                          :target="inline.link ? linkTarget(inline.link, inline.external) : undefined"
                          :rel="inline.link ? linkRel(inline.link, inline.external) : undefined"
                          class="break-words"
                          :class="inlineClass(inline)"
                          @click="inline.link ? handleInlineLinkClick($event, inline.link, inline.external) : undefined"
                        >
                          <code
                            v-if="inline.code"
                            class="rounded-md bg-gray-100 px-1.5 py-0.5 text-[0.92em] text-primary-700 dark:bg-dark-800 dark:text-primary-300"
                          >
                            {{ inline.text }}
                          </code>
                          <template v-else>{{ inline.text }}</template>
                        </component>
                      </template>
                    </p>

                    <div
                      v-else-if="block.type === 'html'"
                      class="guide-html text-[15px] leading-7 text-gray-700 dark:text-gray-300"
                      v-html="block.html"
                    />

                    <div
                      v-else-if="block.type === 'callout'"
                      class="rounded-2xl border p-4 sm:p-5"
                      :class="calloutClass(block.kind)"
                    >
                      <div class="flex items-start gap-3">
                        <div class="mt-0.5 shrink-0">
                          <Icon :name="calloutIcon(block.kind)" size="sm" />
                        </div>
                        <div class="min-w-0 space-y-2">
                          <h4
                            v-if="block.title"
                            class="text-sm font-semibold sm:text-base"
                          >
                            {{ block.title }}
                          </h4>
                          <p
                            v-for="(line, lineIndex) in block.content"
                            :key="`${section.id}-${blockIndex}-${lineIndex}`"
                            class="text-[15px] leading-7"
                          >
                            {{ line }}
                          </p>
                        </div>
                      </div>
                    </div>

                    <ul
                      v-else-if="block.type === 'list' && block.ordered !== true"
                      class="space-y-3 pl-6 text-[15px] leading-7 text-gray-700 marker:text-primary-500 dark:text-gray-300"
                    >
                      <li
                        v-for="(item, itemIndex) in block.items"
                        :key="`${section.id}-${blockIndex}-${itemIndex}`"
                        class="list-disc"
                      >
                        {{ item }}
                      </li>
                    </ul>

                    <ol
                      v-else-if="block.type === 'list' && block.ordered === true"
                      class="space-y-3 pl-6 text-[15px] leading-7 text-gray-700 marker:font-semibold marker:text-primary-500 dark:text-gray-300"
                    >
                      <li
                        v-for="(item, itemIndex) in block.items"
                        :key="`${section.id}-${blockIndex}-${itemIndex}`"
                        class="list-decimal"
                      >
                        {{ item }}
                      </li>
                    </ol>

                    <div
                      v-else-if="block.type === 'steps'"
                      class="space-y-4"
                    >
                      <div
                        v-for="(step, stepIndex) in block.items"
                        :key="`${section.id}-${blockIndex}-${stepIndex}`"
                        class="flex gap-4 rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-950/40"
                      >
                        <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-600 text-sm font-semibold text-white">
                          {{ stepIndex + 1 }}
                        </div>
                        <div class="min-w-0">
                          <h4 class="text-sm font-semibold text-gray-900 dark:text-white sm:text-base">
                            {{ step.title }}
                          </h4>
                          <p
                            v-if="step.body"
                            class="mt-1 text-[15px] leading-7 text-gray-700 dark:text-gray-300"
                          >
                            {{ step.body }}
                          </p>
                        </div>
                      </div>
                    </div>

                    <figure
                      v-else-if="block.type === 'image'"
                      class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 shadow-sm dark:border-dark-700 dark:bg-dark-950/50"
                    >
                      <img
                        :src="block.src"
                        :alt="block.alt"
                        class="w-full object-cover"
                        :loading="block.eager ? 'eager' : 'lazy'"
                      >
                      <figcaption
                        v-if="block.caption"
                        class="border-t border-gray-200 px-4 py-3 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
                      >
                        {{ block.caption }}
                      </figcaption>
                    </figure>

                    <div
                      v-else-if="block.type === 'code'"
                      class="overflow-hidden rounded-2xl border border-gray-200 bg-[#0b1020] shadow-sm dark:border-dark-700"
                    >
                      <div class="flex items-center justify-between gap-4 border-b border-white/10 px-4 py-3 text-sm text-gray-200">
                        <div class="min-w-0">
                          <p
                            v-if="block.title"
                            class="truncate font-medium text-white"
                          >
                            {{ block.title }}
                          </p>
                          <p
                            v-if="block.language"
                            class="text-xs uppercase tracking-[0.2em] text-gray-400"
                          >
                            {{ block.language }}
                          </p>
                        </div>
                        <button
                          type="button"
                          class="inline-flex shrink-0 items-center gap-2 rounded-lg border border-white/10 px-3 py-1.5 text-xs font-medium text-gray-100 transition hover:border-primary-400/60 hover:text-white"
                          @click="copyCode(`${section.id}-${blockIndex}`, block.code)"
                        >
                          <Icon :name="copiedCodeKey === `${section.id}-${blockIndex}` ? 'check' : 'copy'" size="xs" />
                          <span>{{ copiedCodeKey === `${section.id}-${blockIndex}` ? '已复制' : '复制' }}</span>
                        </button>
                      </div>
                      <pre class="overflow-x-auto px-4 py-4 text-sm leading-6 text-gray-100"><code>{{ block.code }}</code></pre>
                    </div>

                    <div
                      v-else-if="block.type === 'table'"
                      class="overflow-hidden rounded-2xl border border-gray-200 dark:border-dark-700"
                    >
                      <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
                          <thead class="bg-gray-50 dark:bg-dark-900">
                            <tr>
                              <th
                                v-for="header in block.headers"
                                :key="header"
                                class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400"
                              >
                                {{ header }}
                              </th>
                            </tr>
                          </thead>
                          <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-800 dark:bg-dark-950/40">
                            <tr
                              v-for="(row, rowIndex) in block.rows"
                              :key="`${section.id}-${blockIndex}-${rowIndex}`"
                            >
                              <td
                                v-for="(cell, cellIndex) in row"
                                :key="`${section.id}-${blockIndex}-${rowIndex}-${cellIndex}`"
                                class="px-4 py-3 text-sm leading-6 text-gray-700 dark:text-gray-300"
                              >
                                {{ cell }}
                              </td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </template>
                </div>
              </section>
            </article>
          </main>

          <aside class="hidden border-l border-gray-200 bg-gray-50/80 xl:block dark:border-dark-700 dark:bg-dark-950/40">
            <div class="sticky top-0 max-h-[calc(100vh-4rem)] overflow-y-auto px-5 py-8">
              <div class="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <Icon name="document" size="sm" />
                <span>本页目录</span>
              </div>
              <nav class="space-y-1.5">
                <button
                  v-for="entry in tocEntries"
                  :key="entry.id"
                  type="button"
                  class="block w-full rounded-lg px-3 py-2 text-left text-sm transition"
                  :class="tocItemClass(entry.id, entry.depth)"
                  @click="scrollToHeading(entry.id)"
                >
                  {{ entry.text }}
                </button>
              </nav>
            </div>
          </aside>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Icon } from '@/components/icons'
import {
  apihubGuideArticle,
  apihubGuideDocument,
  apihubGuideSidebarGroups,
  type ApihubGuideDocument,
  type ApihubGuideHeading,
  type ApihubGuideSidebarGroup,
} from '@/content/apihubGuide'

type InlineToken = {
  text: string
  link?: string
  external?: boolean
  code?: boolean
  strong?: boolean
}

type ParagraphBlock = {
  type: 'paragraph'
  content: string | InlineToken[]
}

type HtmlBlock = {
  type: 'html'
  html: string
}

type CalloutBlock = {
  type: 'callout'
  kind: 'info' | 'tip' | 'warning' | 'danger'
  title?: string
  content: string[]
}

type ListBlock = {
  type: 'list'
  ordered?: boolean
  items: string[]
}

type StepsBlock = {
  type: 'steps'
  items: Array<{ title: string; body?: string }>
}

type ImageBlock = {
  type: 'image'
  src: string
  alt: string
  caption?: string
  eager?: boolean
}

type CodeBlock = {
  type: 'code'
  code: string
  language?: string
  title?: string
}

type TableBlock = {
  type: 'table'
  headers: string[]
  rows: string[][]
}

type GuideBlock = ParagraphBlock | HtmlBlock | CalloutBlock | ListBlock | StepsBlock | ImageBlock | CodeBlock | TableBlock

type GuideSection = {
  id: string
  title: string
  level: 2 | 3 | 4
  blocks: GuideBlock[]
  renderTitle?: boolean
}

type GuideArticle = {
  title: string
  kicker?: string
  summary?: string
  description?: string
  providerName: string
  meta?: Array<{ label: string; value: string }>
  sections: GuideSection[]
}

const showMobileSidebar = ref(false)
const showMobileToc = ref(false)
const articleRef = ref<HTMLElement | null>(null)
const copiedCodeKey = ref('')
const copiedAnchor = ref('')
const activeSectionId = ref('')
const htmlInteractionHandlers = new Map<Element, EventListener>()
let copyCodeTimer: ReturnType<typeof setTimeout> | null = null
let copyAnchorTimer: ReturnType<typeof setTimeout> | null = null
let headingObserver: IntersectionObserver | null = null

function isGuideArticle(value: unknown): value is GuideArticle {
  if (!value || typeof value !== 'object') return false

  const candidate = value as Partial<GuideArticle>
  return typeof candidate.title === 'string'
    && typeof candidate.providerName === 'string'
    && Array.isArray(candidate.sections)
}

function normalizeLevel(level: number): 2 | 3 | 4 {
  if (level <= 2) return 2
  if (level === 3) return 3
  return 4
}

function guideArticleFromDocument(document: ApihubGuideDocument): GuideArticle {
  return {
    title: document.article?.title || document.title,
    description: document.article?.summary || document.description,
    providerName: document.providerName,
    sections: document.contentBlocks.map((block) => ({
      id: block.id,
      title: block.title,
      level: normalizeLevel(block.level),
      blocks: [{ type: 'html', html: block.html }],
      renderTitle: false,
    })),
  }
}

const article = computed<GuideArticle>(() =>
  isGuideArticle(apihubGuideArticle)
    ? apihubGuideArticle
    : guideArticleFromDocument(apihubGuideDocument)
)

const sidebarGroups = computed<ApihubGuideSidebarGroup[]>(() => apihubGuideSidebarGroups)

const fallbackToc = computed<ApihubGuideHeading[]>(() =>
  article.value.sections.map((section) => ({
    id: section.id,
    text: section.title,
    level: section.level,
  }))
)

const tocEntries = computed(() => {
  const source = apihubGuideDocument.toc?.length ? apihubGuideDocument.toc : fallbackToc.value
  return source.map((entry) => ({
    id: entry.id,
    text: entry.text,
    depth: normalizeLevel(entry.level),
  }))
})

function headingTag(level: GuideSection['level']) {
  if (level === 2) return 'h2'
  if (level === 3) return 'h3'
  return 'h4'
}

function headingClass(level: GuideSection['level']) {
  if (level === 2) return 'text-2xl sm:text-3xl'
  if (level === 3) return 'text-xl sm:text-2xl'
  return 'text-lg sm:text-xl'
}

function normalizeInlineContent(content: ParagraphBlock['content']): InlineToken[] {
  if (typeof content === 'string') {
    return [{ text: content }]
  }
  return content
}

function resolveHashId(href: string) {
  if (href.startsWith('#')) {
    return href.slice(1)
  }

  try {
    const url = new URL(href, window.location.href)
    if (url.origin === window.location.origin && url.pathname === window.location.pathname && url.hash) {
      return url.hash.slice(1)
    }
  } catch {
    return ''
  }

  return ''
}

function isExternalLink(href: string, external?: boolean) {
  if (external) return true
  if (!href || href.startsWith('#')) return false

  try {
    const url = new URL(href, window.location.href)
    if (url.origin !== window.location.origin) return true
    return !(url.pathname === window.location.pathname && url.hash)
  } catch {
    return false
  }
}

function resolveLinkHref(href: string) {
  const hashId = resolveHashId(href)
  return hashId ? `#${hashId}` : href
}

function linkTarget(href: string, external?: boolean) {
  return isExternalLink(href, external) ? '_blank' : undefined
}

function linkRel(href: string, external?: boolean) {
  return isExternalLink(href, external) ? 'noreferrer noopener' : undefined
}

function inlineClass(inline: InlineToken) {
  return {
    'font-semibold text-gray-900 dark:text-white': inline.strong,
    'text-primary-600 underline decoration-primary-300 underline-offset-4 hover:text-primary-700 dark:text-primary-300 dark:decoration-primary-500 dark:hover:text-primary-200': !!inline.link,
    'mr-1.5': true,
  }
}

function calloutClass(kind: CalloutBlock['kind']) {
  return {
    'border-sky-200 bg-sky-50/80 text-sky-900 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-100': kind === 'info',
    'border-emerald-200 bg-emerald-50/80 text-emerald-900 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-100': kind === 'tip',
    'border-amber-200 bg-amber-50/80 text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100': kind === 'warning',
    'border-rose-200 bg-rose-50/80 text-rose-900 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-100': kind === 'danger',
  }
}

function calloutIcon(kind: CalloutBlock['kind']): 'infoCircle' | 'lightbulb' | 'exclamationTriangle' | 'xCircle' {
  if (kind === 'tip') return 'lightbulb'
  if (kind === 'warning') return 'exclamationTriangle'
  if (kind === 'danger') return 'xCircle'
  return 'infoCircle'
}

function scrollToHeading(id: string) {
  activeSectionId.value = id
  showMobileSidebar.value = false
  showMobileToc.value = false
  history.replaceState(null, '', `#${id}`)
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function handleNavClick(href: string) {
  const hashId = resolveHashId(href)
  if (hashId) {
    scrollToHeading(hashId)
    return
  }
  window.open(href, '_blank', 'noopener,noreferrer')
}

function handleInlineLinkClick(event: Event, href: string, external?: boolean) {
  const hashId = resolveHashId(href)
  if (hashId) {
    event.preventDefault()
    scrollToHeading(hashId)
    return
  }

  if (isExternalLink(href, external)) {
    event.preventDefault()
    window.open(href, '_blank', 'noopener,noreferrer')
  }
}

function navItemClass(href: string) {
  const sectionId = href.startsWith('#') ? href.slice(1) : ''
  const isActive = !!sectionId && activeSectionId.value === sectionId

  return {
    'bg-primary-50 text-primary-700 shadow-sm dark:bg-primary-500/10 dark:text-primary-200': isActive,
    'text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-800 dark:hover:text-white': !isActive,
  }
}

function tocItemClass(id: string, depth: number) {
  const isActive = activeSectionId.value === id
  return {
    'bg-primary-50 font-medium text-primary-700 dark:bg-primary-500/10 dark:text-primary-200': isActive,
    'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-white': !isActive,
    'pl-3': depth === 2,
    'pl-6': depth === 3,
    'pl-9': depth >= 4,
  }
}

async function copyCode(key: string, code: string) {
  await navigator.clipboard.writeText(code)
  copiedCodeKey.value = key
  if (copyCodeTimer) clearTimeout(copyCodeTimer)
  copyCodeTimer = setTimeout(() => {
    copiedCodeKey.value = ''
  }, 1800)
}

async function copyAnchor(id: string) {
  const hash = `${window.location.origin}${window.location.pathname}#${id}`
  await navigator.clipboard.writeText(hash)
  copiedAnchor.value = id
  history.replaceState(null, '', `#${id}`)
  if (copyAnchorTimer) clearTimeout(copyAnchorTimer)
  copyAnchorTimer = setTimeout(() => {
    copiedAnchor.value = ''
  }, 1600)
}

function cleanupHtmlCopyHandlers() {
  htmlInteractionHandlers.forEach((handler, element) => {
    element.removeEventListener('click', handler)
  })
  htmlInteractionHandlers.clear()
}

function enhanceHtmlBlocks() {
  cleanupHtmlCopyHandlers()

  if (!articleRef.value) return

  articleRef.value.querySelectorAll<HTMLButtonElement>('.guide-html button.copy').forEach((button) => {
    const codeElement = button.parentElement?.querySelector('pre code')
    if (!codeElement) return

    button.type = 'button'
    button.classList.add('guide-copy-button')
    button.textContent = '复制'
    button.title = '复制代码'

    const handler = async () => {
      await navigator.clipboard.writeText(codeElement.textContent || '')
      const originalTitle = button.title || '复制代码'
      const originalText = button.textContent || ''
      button.textContent = '已复制'
      button.title = '已复制'
      window.setTimeout(() => {
        button.textContent = originalText
        button.title = originalTitle
      }, 1600)
    }

    button.addEventListener('click', handler)
    htmlInteractionHandlers.set(button, handler)
  })

  articleRef.value.querySelectorAll<HTMLAnchorElement>('.guide-html a[href]').forEach((anchor) => {
    const href = anchor.getAttribute('href')
    if (!href) return

    const hashId = resolveHashId(href)
    if (hashId) {
      anchor.setAttribute('href', `#${hashId}`)
      anchor.removeAttribute('target')
      anchor.removeAttribute('rel')

      const handler = (event: Event) => {
        event.preventDefault()
        scrollToHeading(hashId)
      }

      anchor.addEventListener('click', handler)
      htmlInteractionHandlers.set(anchor, handler)
      return
    }

    if (isExternalLink(href)) {
      anchor.setAttribute('target', '_blank')
      anchor.setAttribute('rel', 'noreferrer noopener')
    }
  })
}

function setupHeadingObserver() {
  headingObserver?.disconnect()

  headingObserver = new IntersectionObserver(
    (entries) => {
      const visibleEntries = entries
        .filter((entry) => entry.isIntersecting)
        .sort((left, right) => right.intersectionRatio - left.intersectionRatio)

      if (visibleEntries[0]?.target?.id) {
        activeSectionId.value = visibleEntries[0].target.id
      }
    },
    {
      rootMargin: '-96px 0px -60% 0px',
      threshold: [0.1, 0.3, 0.6],
    }
  )

  tocEntries.value.forEach((entry) => {
    const element = document.getElementById(entry.id)
    if (element) headingObserver?.observe(element)
  })

  if (!activeSectionId.value && tocEntries.value[0]) {
    activeSectionId.value = tocEntries.value[0].id
  }
}

onMounted(async () => {
  await nextTick()
  enhanceHtmlBlocks()
  setupHeadingObserver()

  if (window.location.hash) {
    const hashId = window.location.hash.slice(1)
    if (tocEntries.value.some((entry) => entry.id === hashId)) {
      activeSectionId.value = hashId
      window.setTimeout(() => scrollToHeading(hashId), 60)
    }
  }
})

onBeforeUnmount(() => {
  cleanupHtmlCopyHandlers()
  headingObserver?.disconnect()
  if (copyCodeTimer) clearTimeout(copyCodeTimer)
  if (copyAnchorTimer) clearTimeout(copyAnchorTimer)
})
</script>

<style scoped>
.guide-html :deep(h1),
.guide-html :deep(h2),
.guide-html :deep(h3),
.guide-html :deep(h4) {
  color: rgb(17 24 39);
  font-weight: 700;
  letter-spacing: -0.02em;
  margin-bottom: 1rem;
  scroll-margin-top: 6rem;
}

.guide-html :deep(h1) {
  font-size: 2rem;
  line-height: 2.5rem;
}

.guide-html :deep(h2) {
  font-size: 1.5rem;
  line-height: 2rem;
}

.guide-html :deep(h3) {
  font-size: 1.25rem;
  line-height: 1.75rem;
}

.guide-html :deep(h4) {
  font-size: 1.125rem;
  line-height: 1.75rem;
}

.guide-html :deep(p),
.guide-html :deep(ul),
.guide-html :deep(ol),
.guide-html :deep(table),
.guide-html :deep(.custom-block),
.guide-html :deep([class*='language-']) {
  margin-bottom: 1rem;
}

.guide-html :deep(a) {
  color: rgb(37 99 235);
  text-decoration: underline;
  text-underline-offset: 4px;
}

.guide-html :deep(code) {
  border-radius: 0.375rem;
  background: rgb(243 244 246);
  padding: 0.125rem 0.375rem;
  color: rgb(29 78 216);
}

.guide-html :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}

.guide-html :deep(pre) {
  overflow-x: auto;
}

.guide-html :deep(img) {
  width: 100%;
  border-radius: 1rem;
  border: 1px solid rgb(229 231 235);
}

.guide-html :deep(table) {
  width: 100%;
  overflow: hidden;
  border-collapse: collapse;
  border-radius: 1rem;
  border: 1px solid rgb(229 231 235);
}

.guide-html :deep(th),
.guide-html :deep(td) {
  border-bottom: 1px solid rgb(229 231 235);
  padding: 0.75rem 1rem;
  text-align: left;
}

.guide-html :deep(th) {
  background: rgb(249 250 251);
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: rgb(107 114 128);
}

.guide-html :deep(.custom-block) {
  border-radius: 1rem;
  border: 1px solid rgb(229 231 235);
  padding: 1rem 1.25rem;
}

.guide-html :deep(.tip.custom-block) {
  background: rgb(236 253 245);
}

.guide-html :deep(.warning.custom-block) {
  background: rgb(255 251 235);
}

.guide-html :deep(.danger.custom-block) {
  background: rgb(255 241 242);
}

.guide-html :deep(.copy),
.guide-html :deep(.guide-copy-button) {
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.18);
  background: rgba(15, 23, 42, 0.88);
  color: rgb(243 244 246);
  cursor: pointer;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.375rem 0.75rem;
}

.dark .guide-html :deep(h1),
.dark .guide-html :deep(h2),
.dark .guide-html :deep(h3),
.dark .guide-html :deep(h4) {
  color: rgb(255 255 255);
}

.dark .guide-html :deep(a) {
  color: rgb(147 197 253);
}

.dark .guide-html :deep(code) {
  background: rgb(31 41 55);
  color: rgb(147 197 253);
}

.dark .guide-html :deep(img),
.dark .guide-html :deep(table),
.dark .guide-html :deep(th),
.dark .guide-html :deep(td),
.dark .guide-html :deep(.custom-block) {
  border-color: rgb(55 65 81);
}

.dark .guide-html :deep(th) {
  background: rgb(17 24 39);
  color: rgb(156 163 175);
}

.dark .guide-html :deep(.tip.custom-block) {
  background: rgba(6, 78, 59, 0.3);
}

.dark .guide-html :deep(.warning.custom-block) {
  background: rgba(120, 53, 15, 0.3);
}

.dark .guide-html :deep(.danger.custom-block) {
  background: rgba(136, 19, 55, 0.3);
}
</style>
