<template>
  <div class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
    <div class="text-sm text-gray-500 dark:text-dark-400">{{ label }}</div>
    <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ value }}</div>
    <div v-if="hint" class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ hint }}</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  value: string
  hint?: string
}>()
</script>
