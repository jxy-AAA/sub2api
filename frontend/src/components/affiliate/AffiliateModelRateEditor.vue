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
        :key="`${row.model}-${index}`"
        class="grid gap-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-[minmax(0,1fr)_180px_auto]"
      >
        <div>
          <label class="mb-1.5 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
            {{ modelLabel }}
          </label>
          <Input
            :model-value="row.model"
            :disabled="!editable"
            :placeholder="modelPlaceholder"
            @update:model-value="updateRow(index, $event)"
          />
        </div>
        <div>
          <label class="mb-1.5 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
            {{ multiplierLabel }}
          </label>
          <Input
            :model-value="String(row.multiplier ?? '')"
            :disabled="!editable"
            type="number"
            placeholder="1.60"
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
import type { AffiliateModelRate } from './types'

interface Props {
  title: string
  description?: string
  modelRates: AffiliateModelRate[]
  saving?: boolean
  editable?: boolean
  emptyText: string
  addLabel: string
  saveLabel: string
  savingLabel: string
  modelLabel: string
  multiplierLabel: string
  removeLabel: string
  modelPlaceholder: string
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  saving: false,
  editable: true
})

const emit = defineEmits<{
  (e: 'save', value: AffiliateModelRate[]): void
}>()

const draftRows = ref<AffiliateModelRate[]>([])

watch(
  () => props.modelRates,
  (value) => {
    draftRows.value = value.map((item) => ({
      model: item.model,
      multiplier: item.multiplier
    }))
  },
  { immediate: true, deep: true }
)

const normalizedRows = computed(() =>
  draftRows.value
    .map((row) => ({
      model: row.model.trim(),
      multiplier: Number(row.multiplier)
    }))
    .filter((row) => row.model && Number.isFinite(row.multiplier) && row.multiplier >= 0)
)

const hasValidRows = computed(() => normalizedRows.value.length > 0)

function appendRow(): void {
  draftRows.value.push({ model: '', multiplier: 1 })
}

function updateRow(index: number, value: string): void {
  draftRows.value[index] = {
    ...draftRows.value[index],
    model: value
  }
}

function updateMultiplier(index: number, value: string): void {
  draftRows.value[index] = {
    ...draftRows.value[index],
    multiplier: Number(value)
  }
}

function removeRow(index: number): void {
  draftRows.value.splice(index, 1)
}

function submit(): void {
  emit('save', normalizedRows.value)
}
</script>
