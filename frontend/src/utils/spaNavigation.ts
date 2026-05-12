export interface SpaNavigationOptions {
  replace?: boolean
}

function performBrowserNavigation(target: string, replace = false): void {
  if (typeof window === 'undefined') {
    return
  }

  if (replace) {
    window.location.replace(target)
    return
  }

  window.location.assign(target)
}

export function isInternalNavigationTarget(target: string): boolean {
  const normalizedTarget = target.trim()
  return normalizedTarget.startsWith('/') && !normalizedTarget.startsWith('//')
}

export async function navigateWithinApp(
  target: string,
  options: SpaNavigationOptions = {}
): Promise<boolean> {
  const normalizedTarget = target.trim()
  if (!normalizedTarget) {
    return false
  }

  if (!isInternalNavigationTarget(normalizedTarget)) {
    performBrowserNavigation(normalizedTarget, options.replace)
    return false
  }

  try {
    const { default: router } = await import('@/router')
    const currentRoute = `${window.location.pathname}${window.location.search}${window.location.hash}`

    if (currentRoute === normalizedTarget) {
      return true
    }

    if (options.replace) {
      await router.replace(normalizedTarget)
    } else {
      await router.push(normalizedTarget)
    }

    return true
  } catch {
    performBrowserNavigation(normalizedTarget, options.replace)
    return false
  }
}
