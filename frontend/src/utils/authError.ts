import { extractApiErrorMessage } from './apiError'

export function buildAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
  }
): string {
  const { fallback } = options
  return extractApiErrorMessage(error, fallback)
}
