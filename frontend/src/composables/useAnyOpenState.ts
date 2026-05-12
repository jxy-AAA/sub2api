import { computed, toValue, type MaybeRefOrGetter } from 'vue'

export function useAnyOpenState(sources: ReadonlyArray<MaybeRefOrGetter<boolean>>) {
  return computed(() => sources.some((source) => Boolean(toValue(source))))
}
