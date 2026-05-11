<template>
  <div class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
        <p v-if="description" class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ description }}
        </p>
      </div>
      <button
        v-if="editable"
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="availableGroupsForNewRow.length === 0"
        @click="appendRow"
      >
        <Icon name="plus" size="sm" />
        <span>{{ addLabel }}</span>
      </button>
    </div>

    <div v-if="draftRows.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
      {{ emptyText }}
    </div>

    <div v-else class="mt-4 space-y-3">
      <div
        v-for="(row, index) in draftRows"
        :key="`${row.group_id}-${index}`"
        class="grid gap-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-[minmax(0,1fr)_180px_auto]"
      >
        <div>
          <label class="mb-1.5 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
            {{ groupLabel }}
          </label>
          <select
            :model-value="String(row.group_id || 0)"
            :disabled="!editable"
            class="input"
            @change="updateGroup(index, ($event.target as HTMLSelectElement).value)"
          >
            <option :value="0" disabled>{{ groupPlaceholder }}</option>
            <option
              v-for="option in mergedGroupOptions"
              :key="option.id"
              :value="option.id"
              :disabled="isGroupSelectedByOtherRow(option.id, index)"
            >
              {{ formatGroupOption(option) }}
            </option>
          </select>
        </div>
        <div>
          <label class="mb-1.5 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
            {{ multiplierLabel }}
          </label>
          <Input
            :model-value="String(row.rate_multiplier ?? '')"
            :disabled="!editable"
            type="number"
            min="0.0001"
            step="0.0001"
            placeholder="1.0000"
            @update:model-value="updateMultiplier(index, $event)"
          />
        </div>
        <div v-if="editable" class="flex items-end">
          <button type="button" class="btn btn-secondary btn-sm w-full" @click="removeRow(index)">
            <Icon name="trash" size="sm" />
            <span>{{ removeLabel }}</span>
          </button>
        </div>
      </div>
    </div>

    <p v-if="editable && hasDuplicateGroups" class="mt-3 text-sm text-red-600 dark:text-red-400">
      {{ duplicateGroupText }}
    </p>

    <div v-if="editable" class="mt-4 flex justify-end">
      <button
        type="button"
        class="btn btn-primary"
        :disabled="saving || !hasValidRows"
        @click="submit"
      >
        <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />
        <Icon v-else name="check" size="sm" />
        <span>{{ saving ? savingLabel : saveLabel }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Input from '@/components/common/Input.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AffiliateGroupOption, AffiliateGroupRate } from './types'

interface Props {
  title: string
  description?: string
  groupRates: AffiliateGroupRate[]
  groupOptions: AffiliateGroupOption[]
  saving?: boolean
  editable?: boolean
  emptyText: string
  addLabel: string
  saveLabel: string
  savingLabel: string
  groupLabel: string
  multiplierLabel: string
  removeLabel: string
  groupPlaceholder: string
  duplicateGroupText: string
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  saving: false,
  editable: true
})

const emit = defineEmits<{
  (e: 'save', value: AffiliateGroupRate[]): void
}>()

const draftRows = ref<AffiliateGroupRate[]>([])

watch(
  () => props.groupRates,
  (value) => {
    draftRows.value = value.map((item) => ({
      ...item,
      group_id: Number(item.group_id) || 0,
      rate_multiplier: Number(item.rate_multiplier)
    }))
  },
  { immediate: true, deep: true }
)

const mergedGroupOptions = computed<AffiliateGroupOption[]>(() => {
  const options = new Map<number, AffiliateGroupOption>()
  props.groupOptions.forEach((group) => {
    if (Number.isFinite(Number(group.id)) && Number(group.id) > 0) {
      options.set(Number(group.id), {
        id: Number(group.id),
        name: group.name || `#${group.id}`,
        platform: group.platform,
        rate_multiplier: group.rate_multiplier
      })
    }
  })
  draftRows.value.forEach((row) => {
    if (row.group_id > 0 && !options.has(row.group_id)) {
      options.set(row.group_id, {
        id: row.group_id,
        name: row.group_name || `#${row.group_id}`,
        platform: row.group_platform,
        rate_multiplier: row.group_rate_multiplier
      })
    }
  })
  return [...options.values()].sort((a, b) => a.name.localeCompare(b.name) || a.id - b.id)
})

const selectedGroupIds = computed(() => new Set(draftRows.value.map((row) => row.group_id).filter((id) => id > 0)))
const availableGroupsForNewRow = computed(() => mergedGroupOptions.value.filter((option) => !selectedGroupIds.value.has(option.id)))

const normalizedRows = computed(() =>
  draftRows.value
    .map((row) => {
      const option = mergedGroupOptions.value.find((item) => item.id === row.group_id)
      return {
        group_id: Number(row.group_id),
        group_name: row.group_name || option?.name,
        group_platform: row.group_platform || option?.platform,
        group_rate_multiplier: row.group_rate_multiplier ?? option?.rate_multiplier,
        rate_multiplier: Number(row.rate_multiplier),
        source_type: row.source_type,
        updated_at: row.updated_at ?? null
      } satisfies AffiliateGroupRate
    })
    .filter((row) => row.group_id > 0 && Number.isFinite(row.rate_multiplier) && row.rate_multiplier > 0)
)

const hasDuplicateGroups = computed(() => {
  const seen = new Set<number>()
  for (const row of draftRows.value) {
    if (row.group_id <= 0) continue
    if (seen.has(row.group_id)) return true
    seen.add(row.group_id)
  }
  return false
})

const hasValidRows = computed(() =>
  draftRows.value.length > 0
    && normalizedRows.value.length === draftRows.value.length
    && !hasDuplicateGroups.value
)

function appendRow(): void {
  const nextGroup = availableGroupsForNewRow.value[0]
  draftRows.value.push({
    group_id: nextGroup?.id ?? 0,
    group_name: nextGroup?.name,
    group_platform: nextGroup?.platform,
    group_rate_multiplier: nextGroup?.rate_multiplier,
    rate_multiplier: 1
  })
}

function updateGroup(index: number, value: string): void {
  const groupID = Number(value)
  const option = mergedGroupOptions.value.find((item) => item.id === groupID)
  draftRows.value[index] = {
    ...draftRows.value[index],
    group_id: Number.isFinite(groupID) ? groupID : 0,
    group_name: option?.name,
    group_platform: option?.platform,
    group_rate_multiplier: option?.rate_multiplier
  }
}

function updateMultiplier(index: number, value: string): void {
  draftRows.value[index] = {
    ...draftRows.value[index],
    rate_multiplier: Number(value)
  }
}

function removeRow(index: number): void {
  draftRows.value.splice(index, 1)
}

function isGroupSelectedByOtherRow(groupID: number, rowIndex: number): boolean {
  return draftRows.value.some((row, index) => index !== rowIndex && row.group_id === groupID)
}

function formatGroupOption(option: AffiliateGroupOption): string {
  const platform = option.platform ? ` ? ${option.platform}` : ''
  const rate = Number.isFinite(Number(option.rate_multiplier)) ? ` ? ?? ${Number(option.rate_multiplier).toFixed(4)}x` : ''
  return `${option.name}${platform}${rate}`
}

function submit(): void {
  if (!hasValidRows.value) return
  emit('save', normalizedRows.value)
}
</script>
