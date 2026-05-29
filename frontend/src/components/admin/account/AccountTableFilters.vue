<template>
  <div class="flex flex-wrap items-center gap-3">
    <SearchInput
      :model-value="searchQuery"
      :placeholder="t('admin.accounts.searchAccounts')"
      class="w-full sm:w-64"
      @update:model-value="$emit('update:searchQuery', $event)"
      @search="$emit('change')"
    />
    <Select :model-value="filters.platform" class="w-40" :options="pOpts" @update:model-value="updatePlatform" @change="$emit('change')" />
    <Select :model-value="filters.type" class="w-40" :options="tOpts" @update:model-value="updateType" @change="$emit('change')" />
    <Select :model-value="filters.status" class="w-40" :options="sOpts" @update:model-value="updateStatus" @change="$emit('change')" />
    <Select :model-value="filters.privacy_mode" class="w-40" :options="privacyOpts" @update:model-value="updatePrivacyMode" @change="$emit('change')" />
    <Select :model-value="filters.group" class="w-40" :options="gOpts" @update:model-value="updateGroup" @change="$emit('change')" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import type { AdminGroup } from '@/types'
import { getPlatformDisplayName, getProtocolDisplayName, isCompatiblePlatform } from '@/utils/platforms'

const PLATFORM_ORDER = [
  'anthropic',
  'anthropic_compatible',
  'openai',
  'openai_compatible',
  'gemini',
  'antigravity',
] as const

const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change'])
const { t } = useI18n()

const updatePlatform = (value: string | number | boolean | null) => {
  emit('update:filters', { ...props.filters, platform: value })
}
const updateType = (value: string | number | boolean | null) => {
  emit('update:filters', { ...props.filters, type: value })
}
const updateStatus = (value: string | number | boolean | null) => {
  emit('update:filters', { ...props.filters, status: value })
}
const updatePrivacyMode = (value: string | number | boolean | null) => {
  emit('update:filters', { ...props.filters, privacy_mode: value })
}
const updateGroup = (value: string | number | boolean | null) => {
  emit('update:filters', { ...props.filters, group: value })
}

const platformRank = (platform: string) => {
  const index = PLATFORM_ORDER.indexOf(platform as (typeof PLATFORM_ORDER)[number])
  return index === -1 ? PLATFORM_ORDER.length : index
}

const formatPlatformLabel = (platform: string) =>
  isCompatiblePlatform(platform) ? getProtocolDisplayName(platform) : getPlatformDisplayName(platform)

const formatGroupLabel = (group: AdminGroup) => `${group.name} (${formatPlatformLabel(group.platform)})`

const pOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  ...PLATFORM_ORDER.map((platform) => ({
    value: platform,
    label: formatPlatformLabel(platform),
  })),
])

const tOpts = computed(() => [
  { value: '', label: t('admin.accounts.allTypes') },
  { value: 'oauth', label: t('admin.accounts.oauthType') },
  { value: 'setup-token', label: t('admin.accounts.setupToken') },
  { value: 'apikey', label: t('admin.accounts.apiKey') },
  { value: 'upstream', label: t('admin.accounts.types.upstream') },
  { value: 'bedrock', label: 'AWS Bedrock' },
])

const sOpts = computed(() => [
  { value: '', label: t('admin.accounts.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') },
  { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') },
  { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') },
  { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') },
])

const privacyOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || [])
    .slice()
    .sort((left, right) => {
      const platformDiff = platformRank(left.platform) - platformRank(right.platform)
      if (platformDiff !== 0) {
        return platformDiff
      }
      return left.name.localeCompare(right.name, undefined, { sensitivity: 'base' })
    })
    .flatMap((group, index, allGroups) => {
      const options: Array<Record<string, unknown>> = []
      const previousPlatform = index > 0 ? allGroups[index - 1].platform : null
      if (group.platform !== previousPlatform) {
        options.push({
          kind: 'group',
          value: `__group__${group.platform}`,
          label: formatPlatformLabel(group.platform),
          disabled: true,
        })
      }
      options.push({ value: String(group.id), label: formatGroupLabel(group) })
      return options
    })
])
</script>
