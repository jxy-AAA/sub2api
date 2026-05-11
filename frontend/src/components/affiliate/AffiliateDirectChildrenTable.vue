<template>
  <div class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
        <p v-if="description" class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ description }}
        </p>
      </div>
      <span class="rounded-full bg-gray-100 px-3 py-1 text-sm font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300">
        {{ countLabel }}
      </span>
    </div>

    <div v-if="children.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
      {{ emptyText }}
    </div>

    <div v-else class="mt-4 space-y-4">
      <div
        v-for="child in children"
        :key="child.user_id"
        class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                {{ child.username || child.email || `#${child.user_id}` }}
              </p>
              <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                {{ roleText(child.role) }}
              </span>
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ child.email || '-' }}</p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ joinedAtLabel }}：{{ formatDateTime(child.joined_at) || '-' }}
            </p>
          </div>

          <div class="grid gap-3 sm:grid-cols-3 lg:min-w-[460px]">
            <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900">
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ revenueLabel }}</p>
              <p class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatCurrency(child.today_revenue_usd) }}
              </p>
            </div>
            <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900">
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ rebateLabel }}</p>
              <p class="mt-1 text-sm font-semibold text-emerald-600 dark:text-emerald-400">
                {{ formatCurrency(child.today_rebate_rmb, 'CNY') }}
              </p>
            </div>
            <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900">
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ balanceLabel }}</p>
              <p class="mt-1 text-sm font-semibold text-amber-600 dark:text-amber-400">
                {{ formatCurrency(child.current_rebate_balance_rmb, 'CNY') }}
              </p>
            </div>
          </div>
        </div>

        <div class="mt-4 flex justify-end">
          <button type="button" class="btn btn-secondary btn-sm" @click="toggleExpanded(child.user_id)">
            <Icon :name="expandedId === child.user_id ? 'chevronUp' : 'chevronDown'" size="sm" />
            <span>{{ expandedId === child.user_id ? collapseLabel : expandLabel }}</span>
          </button>
        </div>

        <div v-if="expandedId === child.user_id" class="mt-4">
          <AffiliateGroupRateEditor
            :title="childRateTitle"
            :description="childRateDescription"
            :group-rates="child.group_rates"
            :group-options="groupOptions"
            :saving="savingUserId === child.user_id"
            :empty-text="emptyRatesText"
            :add-label="addRateLabel"
            :save-label="saveRateLabel"
            :saving-label="savingRateLabel"
            :group-label="groupLabel"
            :multiplier-label="multiplierLabel"
            :remove-label="removeLabel"
            :group-placeholder="groupPlaceholder"
            :duplicate-group-text="duplicateGroupText"
            @save="emit('save-child-rates', child.user_id, $event)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import AffiliateGroupRateEditor from './AffiliateGroupRateEditor.vue'
import type { AffiliateDirectChild, AffiliateGroupOption, AffiliateGroupRate } from './types'
import { formatCurrency, formatDateTime } from '@/utils/format'

interface Props {
  children: AffiliateDirectChild[]
  groupOptions: AffiliateGroupOption[]
  savingUserId?: number | null
  title: string
  description?: string
  countLabel: string
  emptyText: string
  joinedAtLabel: string
  revenueLabel: string
  rebateLabel: string
  balanceLabel: string
  expandLabel: string
  collapseLabel: string
  childRateTitle: string
  childRateDescription: string
  emptyRatesText: string
  addRateLabel: string
  saveRateLabel: string
  savingRateLabel: string
  groupLabel: string
  multiplierLabel: string
  removeLabel: string
  groupPlaceholder: string
  duplicateGroupText: string
  agentRoleLabel: string
  userRoleLabel: string
}

const props = withDefaults(defineProps<Props>(), {
  savingUserId: null,
  description: ''
})

const emit = defineEmits<{
  (e: 'save-child-rates', userId: number, value: AffiliateGroupRate[]): void
}>()

const expandedId = ref<number | null>(null)

function toggleExpanded(userId: number): void {
  expandedId.value = expandedId.value === userId ? null : userId
}

function roleText(role?: string): string {
  return role === 'agent' ? props.agentRoleLabel : props.userRoleLabel
}
</script>
