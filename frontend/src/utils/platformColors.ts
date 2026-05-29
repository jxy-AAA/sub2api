import { getPlatformDisplayName, resolvePlatformBadgeKey } from './platforms'

/**
 * Centralized platform color definitions.
 *
 * All components that need platform-specific styling should import from here
 * instead of defining their own color mappings.
 */

export type Platform =
  | 'anthropic'
  | 'openai'
  | 'antigravity'
  | 'gemini'
  | 'openai_compatible'
  | 'anthropic_compatible'

// ── Badge (bg + text + border, for inline badges with border) ───────
const BADGE: Record<Platform, string> = {
  anthropic: 'bg-orange-500/10 text-orange-600 border-orange-500/30 dark:text-orange-400',
  openai: 'bg-green-500/10 text-green-600 border-green-500/30 dark:text-green-400',
  antigravity: 'bg-purple-500/10 text-purple-600 border-purple-500/30 dark:text-purple-400',
  gemini: 'bg-blue-500/10 text-blue-600 border-blue-500/30 dark:text-blue-400',
  openai_compatible: 'bg-teal-500/10 text-teal-600 border-teal-500/30 dark:text-teal-400',
  anthropic_compatible: 'bg-amber-500/10 text-amber-600 border-amber-500/30 dark:text-amber-400',
}
const BADGE_DEFAULT = 'bg-slate-500/10 text-slate-600 border-slate-500/30 dark:text-slate-400'

// ── Light badge (softer bg, no border) ──────────────────────────────
const BADGE_LIGHT: Record<Platform, string> = {
  anthropic: 'bg-orange-500/10 text-orange-600 dark:bg-orange-500/10 dark:text-orange-300',
  openai: 'bg-green-500/10 text-green-600 dark:bg-green-500/10 dark:text-green-300',
  antigravity: 'bg-purple-500/10 text-purple-600 dark:bg-purple-500/10 dark:text-purple-300',
  gemini: 'bg-blue-500/10 text-blue-600 dark:bg-blue-500/10 dark:text-blue-300',
  openai_compatible: 'bg-teal-500/10 text-teal-600 dark:bg-teal-500/10 dark:text-teal-300',
  anthropic_compatible: 'bg-amber-500/10 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300',
}

// ── Border ──────────────────────────────────────────────────────────
const BORDER: Record<Platform, string> = {
  anthropic: 'border-orange-500/20 dark:border-orange-500/20',
  openai: 'border-green-500/20 dark:border-green-500/20',
  antigravity: 'border-purple-500/20 dark:border-purple-500/20',
  gemini: 'border-blue-500/20 dark:border-blue-500/20',
  openai_compatible: 'border-teal-500/20 dark:border-teal-500/20',
  anthropic_compatible: 'border-amber-500/20 dark:border-amber-500/20',
}
const BORDER_DEFAULT = 'border-gray-200 dark:border-dark-700'

// ── Accent bar (gradient) ───────────────────────────────────────────
const ACCENT_BAR: Record<Platform, string> = {
  anthropic: 'bg-gradient-to-r from-orange-400 to-orange-500',
  openai: 'bg-gradient-to-r from-emerald-400 to-emerald-500',
  antigravity: 'bg-gradient-to-r from-purple-400 to-purple-500',
  gemini: 'bg-gradient-to-r from-blue-400 to-blue-500',
  openai_compatible: 'bg-gradient-to-r from-teal-400 to-teal-500',
  anthropic_compatible: 'bg-gradient-to-r from-amber-400 to-amber-500',
}
const ACCENT_BAR_DEFAULT = 'bg-gradient-to-r from-primary-400 to-primary-500'

// ── Text (price, icon) ─────────────────────────────────────────────
const TEXT: Record<Platform, string> = {
  anthropic: 'text-orange-600 dark:text-orange-400',
  openai: 'text-emerald-600 dark:text-emerald-400',
  antigravity: 'text-purple-600 dark:text-purple-400',
  gemini: 'text-blue-600 dark:text-blue-400',
  openai_compatible: 'text-teal-600 dark:text-teal-400',
  anthropic_compatible: 'text-amber-600 dark:text-amber-400',
}
const TEXT_DEFAULT = 'text-primary-600 dark:text-primary-400'

// ── Icon (check mark etc.) ──────────────────────────────────────────
const ICON: Record<Platform, string> = {
  anthropic: 'text-orange-500 dark:text-orange-400',
  openai: 'text-emerald-500 dark:text-emerald-400',
  antigravity: 'text-purple-500 dark:text-purple-400',
  gemini: 'text-blue-500 dark:text-blue-400',
  openai_compatible: 'text-teal-500 dark:text-teal-400',
  anthropic_compatible: 'text-amber-500 dark:text-amber-400',
}
const ICON_DEFAULT = 'text-primary-500 dark:text-primary-400'

// ── Button (solid bg) ───────────────────────────────────────────────
const BUTTON: Record<Platform, string> = {
  anthropic: 'bg-orange-500 text-white hover:bg-orange-600 active:bg-orange-700 dark:bg-orange-500/80 dark:hover:bg-orange-500',
  openai: 'bg-green-600 text-white hover:bg-green-700 active:bg-green-800 dark:bg-green-600/80 dark:hover:bg-green-600',
  antigravity: 'bg-purple-500 text-white hover:bg-purple-600 active:bg-purple-700 dark:bg-purple-500/80 dark:hover:bg-purple-500',
  gemini: 'bg-blue-500 text-white hover:bg-blue-600 active:bg-blue-700 dark:bg-blue-500/80 dark:hover:bg-blue-500',
  openai_compatible: 'bg-teal-600 text-white hover:bg-teal-700 active:bg-teal-800 dark:bg-teal-600/80 dark:hover:bg-teal-600',
  anthropic_compatible: 'bg-amber-500 text-white hover:bg-amber-600 active:bg-amber-700 dark:bg-amber-500/80 dark:hover:bg-amber-500',
}
const BUTTON_DEFAULT = 'bg-primary-500 text-white hover:bg-primary-600 dark:bg-primary-600 dark:hover:bg-primary-500'

// ── Discount badge ──────────────────────────────────────────────────
const DISCOUNT: Record<Platform, string> = {
  anthropic: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300',
  openai: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300',
  antigravity: 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
  gemini: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  openai_compatible: 'bg-teal-100 text-teal-700 dark:bg-teal-900/40 dark:text-teal-300',
  anthropic_compatible: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
}
const DISCOUNT_DEFAULT = 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'

// ── Header gradient (subscription confirm) ─────────────────────────
const GRADIENT: Record<Platform, string> = {
  anthropic: 'from-orange-500 to-orange-600',
  openai: 'from-emerald-500 to-emerald-600',
  antigravity: 'from-purple-500 to-purple-600',
  gemini: 'from-blue-500 to-blue-600',
  openai_compatible: 'from-teal-500 to-teal-600',
  anthropic_compatible: 'from-amber-500 to-amber-600',
}
const GRADIENT_DEFAULT = 'from-primary-500 to-primary-600'

// ── Header text (light text on gradient bg) ────────────────────────
const GRADIENT_TEXT: Record<Platform, string> = {
  anthropic: 'text-orange-100',
  openai: 'text-emerald-100',
  antigravity: 'text-purple-100',
  gemini: 'text-blue-100',
  openai_compatible: 'text-teal-100',
  anthropic_compatible: 'text-amber-100',
}
const GRADIENT_TEXT_DEFAULT = 'text-primary-100'

const GRADIENT_SUBTEXT: Record<Platform, string> = {
  anthropic: 'text-orange-200',
  openai: 'text-emerald-200',
  antigravity: 'text-purple-200',
  gemini: 'text-blue-200',
  openai_compatible: 'text-teal-200',
  anthropic_compatible: 'text-amber-200',
}
const GRADIENT_SUBTEXT_DEFAULT = 'text-primary-200'

// ── Public API ──────────────────────────────────────────────────────

function isPlatform(p: string): p is Platform {
  return (
    p === 'anthropic' ||
    p === 'openai' ||
    p === 'antigravity' ||
    p === 'gemini' ||
    p === 'openai_compatible' ||
    p === 'anthropic_compatible'
  )
}

function resolvePlatform(p: string): Platform | null {
  const resolved = resolvePlatformBadgeKey(p)
  return resolved && isPlatform(resolved) ? resolved : null
}

export function platformBadgeClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? BADGE[resolved] : BADGE_DEFAULT
}

export function platformBadgeLightClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? BADGE_LIGHT[resolved] : BADGE_DEFAULT
}

export function platformBorderClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? BORDER[resolved] : BORDER_DEFAULT
}

export function platformAccentBarClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? ACCENT_BAR[resolved] : ACCENT_BAR_DEFAULT
}

export function platformTextClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? TEXT[resolved] : TEXT_DEFAULT
}

export function platformIconClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? ICON[resolved] : ICON_DEFAULT
}

export function platformButtonClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? BUTTON[resolved] : BUTTON_DEFAULT
}

export function platformDiscountClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? DISCOUNT[resolved] : DISCOUNT_DEFAULT
}

export function platformGradientClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? GRADIENT[resolved] : GRADIENT_DEFAULT
}

export function platformGradientTextClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? GRADIENT_TEXT[resolved] : GRADIENT_TEXT_DEFAULT
}

export function platformGradientSubtextClass(p: string): string {
  const resolved = resolvePlatform(p)
  return resolved ? GRADIENT_SUBTEXT[resolved] : GRADIENT_SUBTEXT_DEFAULT
}

export function platformLabel(p: string): string {
  return getPlatformDisplayName(p)
}
