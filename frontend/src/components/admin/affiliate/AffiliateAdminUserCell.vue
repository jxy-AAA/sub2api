<template>
  <div class="space-y-0.5">
    <div class="font-mono text-sm text-gray-900 dark:text-white">#{{ userId || '-' }}</div>
    <div class="max-w-56 truncate text-sm font-medium text-gray-900 dark:text-white">{{ email || '-' }}</div>
    <div class="max-w-56 truncate text-sm text-gray-500 dark:text-dark-400">{{ username || '-' }}</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  userId?: number | null
  email?: string | null
  username?: string | null
}>()
</script>
