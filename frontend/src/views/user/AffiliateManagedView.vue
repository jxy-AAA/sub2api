<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ tt('affiliateManaged.title', '下级管理') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ tt('affiliateManaged.description', '在授权后查看自己的下级营业额、返利额度，并管理下级分组成本倍率。') }}
        </p>
      </div>

      <div v-if="loadingPermissions" class="card p-8 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ tt('common.loading', '加载中...') }}
      </div>
      <div v-else-if="!hasAnyPermission" class="card rounded-xl border border-dashed border-gray-300 p-8 text-center dark:border-dark-700">
        <div class="text-base font-medium text-gray-900 dark:text-white">{{ tt('affiliateManaged.noAccess', '暂未获得下级管理权限') }}</div>
        <div class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ tt('affiliateManaged.noAccessHint', '请联系管理员为你授权查看营业额、返利额度或管理分组倍率。') }}
        </div>
      </div>
      <div
        v-else-if="permissionNotice"
        class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200"
      >
        {{ permissionNotice }}
      </div>

      <section v-if="permissions.can_view_downline_daily_revenue" class="card overflow-hidden">
        <div class="border-b border-gray-200 p-5 dark:border-dark-700">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ tt('affiliateManaged.dailyRevenueTitle', '下级每日营业额') }}</h3>
              <p class="text-sm text-gray-500 dark:text-dark-400">
                {{ tt('affiliateManaged.dailyRevenueDescription', '仅展示你自己的下级营业额，金额统一按 RMB 显示。') }}
              </p>
            </div>
            <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]">
              <input
                v-model="daily.search"
                class="input"
                :placeholder="tt('affiliateManaged.searchPlaceholder', '搜索邮箱、用户名或邀请码')"
                @keyup.enter="loadDailyRevenue"
              />
              <input v-model="daily.date" type="date" class="input" @change="loadDailyRevenue" />
              <button class="btn btn-secondary" :disabled="daily.loading" @click="loadDailyRevenue">
                {{ tt('affiliateManaged.refresh', '刷新') }}
              </button>
            </div>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.rank', '排名') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.agent', '用户') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.date', '日期') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.revenueRmb', '营业额 (RMB)') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.downlineSummary', '直属下级摘要') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-if="!daily.loading && daily.items.length === 0">
                <td colspan="5" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ tt('affiliateManaged.noData', '暂无数据') }}
                </td>
              </tr>
              <tr v-for="row in daily.items" :key="`${readUserId(row)}-${readDate(row)}`">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ row.rank ?? '-' }}</td>
                <td class="px-4 py-3">
                  <AffiliateAdminUserCell :user-id="readUserId(row)" :email="readEmail(row)" :username="readUsername(row)" />
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatDateOnly(readDate(row)) || '-' }}</td>
                <td class="px-4 py-3 text-right text-sm font-semibold text-emerald-600 dark:text-emerald-400">
                  {{ formatCurrency(readRevenueRMB(row), 'CNY') }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  {{ tt('affiliateManaged.directUsers', '直属用户') }} {{ readDirectUserCount(row) }}
                  /
                  {{ tt('affiliateManaged.directAgents', '直属代理') }} {{ readDirectAgentCount(row) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-if="permissions.can_view_downline_rebate_balances" class="card overflow-hidden">
        <div class="border-b border-gray-200 p-5 dark:border-dark-700">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ tt('affiliateManaged.rebateTitle', '下级返利额度') }}</h3>
              <p class="text-sm text-gray-500 dark:text-dark-400">
                {{ tt('affiliateManaged.rebateDescription', '仅展示你自己的下级返利余额与返利摘要。') }}
              </p>
            </div>
            <div class="flex gap-3">
              <input
                v-model="rebate.search"
                class="input min-w-[220px]"
                :placeholder="tt('affiliateManaged.searchPlaceholder', '搜索邮箱、用户名或邀请码')"
                @keyup.enter="loadRebateBalances"
              />
              <button class="btn btn-secondary" :disabled="rebate.loading" @click="loadRebateBalances">
                {{ tt('affiliateManaged.refresh', '刷新') }}
              </button>
            </div>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.rank', '排名') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.agent', '用户') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.balanceRmb', '返利额度 (RMB)') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.rebateSummary', '返利摘要') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-if="!rebate.loading && rebate.items.length === 0">
                <td colspan="4" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ tt('affiliateManaged.noData', '暂无数据') }}
                </td>
              </tr>
              <tr v-for="row in rebate.items" :key="`${readUserId(row)}-${row.rank ?? 0}`">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ row.rank ?? '-' }}</td>
                <td class="px-4 py-3">
                  <AffiliateAdminUserCell :user-id="readUserId(row)" :email="readEmail(row)" :username="readUsername(row)" />
                </td>
                <td class="px-4 py-3 text-right text-sm font-semibold text-amber-600 dark:text-amber-400">
                  {{ formatCurrency(readRebateBalanceRMB(row), 'CNY') }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  {{ tt('affiliateManaged.todayRebate', '今日返利') }} {{ formatCurrency(readTodayRebateRMB(row), 'CNY') }}
                  /
                  {{ tt('affiliateManaged.monthlyRebate', '本月返利') }} {{ formatCurrency(readMonthlyRebateRMB(row), 'CNY') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-if="permissions.can_manage_downline_pricing" class="card overflow-hidden">
        <div class="border-b border-gray-200 p-5 dark:border-dark-700">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ tt('affiliateManaged.pricingTitle', '下级分组成本倍率管理') }}</h3>
              <p class="text-sm text-gray-500 dark:text-dark-400">
                {{ tt('affiliateManaged.pricingDescription', '仅可编辑你邀请树中的下级分组成本倍率，不包含全局数据。') }}
              </p>
            </div>
            <div class="flex gap-3">
              <input
                v-model="tree.search"
                class="input min-w-[220px]"
                :placeholder="tt('affiliateManaged.searchPlaceholder', '搜索邮箱、用户名或邀请码')"
                @keyup.enter="loadTree"
              />
              <button class="btn btn-secondary" :disabled="tree.loading" @click="loadTree">
                {{ tt('affiliateManaged.refresh', '刷新') }}
              </button>
            </div>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.level', '层级') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.agent', '用户') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.parent', '上级') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.todayRevenue', '今日营业额 (RMB)') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.groupRates', '分组成本倍率') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{{ tt('affiliateManaged.actions', '操作') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-if="!tree.loading && tree.rows.length === 0">
                <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ tt('affiliateManaged.noData', '暂无数据') }}
                </td>
              </tr>
              <tr v-for="row in tree.rows" :key="row.user_id">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ depthLabel(row.depth) }}</td>
                <td class="px-4 py-3">
                  <div class="text-sm font-medium text-gray-900 dark:text-white">{{ row.username || row.email || `#${row.user_id}` }}</div>
                  <div class="text-xs text-gray-500">{{ row.email }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ parentLabel(row) }}</td>
                <td class="px-4 py-3 text-right text-sm text-gray-700 dark:text-gray-300">{{ formatCurrency(readTreeRevenueRMB(row), 'CNY') }}</td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  {{ normalizedRates(row).map(formatRateBadge).join(' / ') || tt('affiliateManaged.noData', '暂无数据') }}
                </td>
                <td class="px-4 py-3 text-right">
                  <button class="btn btn-secondary btn-sm" @click="openPricingDialog(row)">
                    {{ tt('affiliateManaged.editRates', '编辑分组成本倍率') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <BaseDialog
      :show="showPricingDialog"
      :title="tt('affiliateManaged.rateDialogTitle', '编辑下级分组成本倍率')"
      width="wide"
      @close="closePricingDialog"
    >
      <div class="space-y-4">
        <div v-for="(row, index) in pricingRows" :key="`${row.group_id}-${index}`" class="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]">
          <select v-model.number="row.group_id" class="input">
            <option :value="0" disabled>{{ tt('affiliateManaged.selectGroup', '请选择分组') }}</option>
            <option
              v-for="group in normalizedGroupOptions"
              :key="group.id"
              :value="group.id"
              :disabled="isGroupSelected(index, Number(group.id))"
            >
              {{ groupOptionLabel(group) }}
            </option>
          </select>
          <input v-model.number="row.rate_multiplier" type="number" min="0.0001" step="0.0001" class="input" placeholder="1.6000" />
          <button class="btn btn-secondary btn-sm" @click="pricingRows.splice(index, 1)">
            {{ tt('affiliateManaged.remove', '删除') }}
          </button>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="!canAppendPricingRow" @click="appendPricingRow">
          {{ tt('affiliateManaged.addRate', '添加分组成本倍率') }}
        </button>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" :disabled="savingPricing" @click="closePricingDialog">{{ tt('affiliateManaged.cancel', '取消') }}</button>
          <button class="btn btn-primary" :disabled="savingPricing || !canSavePricing" @click="savePricing">
            {{ savingPricing ? tt('affiliateManaged.saving', '保存中...') : tt('affiliateManaged.save', '保存分组成本倍率') }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import type { AxiosError } from 'axios'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AffiliateAdminUserCell from '@/components/admin/affiliate/AffiliateAdminUserCell.vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateOnly } from '@/utils/format'
import type { AffiliateDistributionTreeNode, AffiliateGroupRate } from '@/api/admin/affiliates'
import userGroupsAPI from '@/api/groups'
import type { Group } from '@/types'
import {
  emptyManagedAffiliatePermissions,
  getManagedAffiliatePermissions,
  getManagedDistributionTree,
  getManagedUserDistributionPricing,
  hasManagedAffiliateAccess,
  listManagedDailyRevenueRankings,
  listManagedRebateBalanceRankings,
  updateManagedUserDistributionPricing,
  type ManagedAffiliatePermissions,
  type ManagedDailyRevenueRankingItem,
  type ManagedRebateBalanceRankingItem,
} from '@/api/userAffiliateManaged'

interface PricingRow {
  group_id: number
  rate_multiplier: number
}

const { t } = useI18n()
const appStore = useAppStore()
const loadingPermissions = ref(true)
const permissions = ref<ManagedAffiliatePermissions>(emptyManagedAffiliatePermissions())
const permissionNotice = ref('')
const showPricingDialog = ref(false)
const selectedNode = ref<AffiliateDistributionTreeNode | null>(null)
const pricingRows = ref<PricingRow[]>([])
const groupOptions = ref<Group[]>([])
const savingPricing = ref(false)

const daily = reactive({
  loading: false,
  items: [] as ManagedDailyRevenueRankingItem[],
  search: '',
  date: new Date().toISOString().slice(0, 10),
})

const rebate = reactive({
  loading: false,
  items: [] as ManagedRebateBalanceRankingItem[],
  search: '',
})

const tree = reactive({
  loading: false,
  rows: [] as AffiliateDistributionTreeNode[],
  search: '',
})

const normalizedGroupOptions = computed(() => (
  groupOptions.value
    .map((group) => ({
      ...group,
      id: Number(group.id),
      rate_multiplier: toFiniteNumber(group.rate_multiplier, 1),
    }))
    .filter((group) => Number.isInteger(group.id) && group.id > 0)
))

const hasAnyPermission = computed(() => hasManagedAffiliateAccess(permissions.value))
const canSavePricing = computed(() => (
  pricingRows.value.length > 0
  && pricingRows.value.every((row) => row.group_id > 0 && Number(row.rate_multiplier) > 0)
  && !hasDuplicateGroups()
))
const canAppendPricingRow = computed(() => (
  normalizedGroupOptions.value.length === 0
    || normalizedGroupOptions.value.some((group) => !pricingRows.value.some((row) => row.group_id === Number(group.id)))
))

function tt(key: string, fallback: string, params?: Record<string, unknown>) {
  const translated = t(key, params ?? {})
  return translated === key ? fallback : translated
}

function isForbidden(error: unknown) {
  const axiosError = error as AxiosError
  const status = axiosError?.response?.status ?? (error as { status?: number })?.status
  return status === 403
}

function toFiniteNumber(value: unknown, fallback = 0) {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : fallback
}

function readUserId(row: { agent_user_id?: number; user_id?: number; id?: number }) {
  return toFiniteNumber(row.user_id ?? row.agent_user_id ?? row.id)
}

function readEmail(row: { agent_email?: string; user_email?: string; email?: string }) {
  return row.email || row.user_email || row.agent_email || ''
}

function readUsername(row: { agent_username?: string; username?: string }) {
  return row.username || row.agent_username || ''
}

function readDate(row: ManagedDailyRevenueRankingItem) {
  return row.date || row.revenue_date || ''
}

function readRevenueRMB(row: ManagedDailyRevenueRankingItem) {
  return toFiniteNumber(row.business_rmb ?? row.daily_revenue_rmb ?? row.revenue_rmb ?? row.total_revenue_rmb)
}

function readRebateBalanceRMB(row: ManagedRebateBalanceRankingItem) {
  return toFiniteNumber(row.current_rebate_balance_rmb ?? row.rebate_balance_rmb ?? row.total_rebate_balance_rmb)
}

function readTodayRebateRMB(row: ManagedRebateBalanceRankingItem) {
  return toFiniteNumber(row.today_rebate_rmb)
}

function readMonthlyRebateRMB(row: ManagedRebateBalanceRankingItem) {
  return toFiniteNumber(row.monthly_rebate_rmb)
}

function readDirectUserCount(row: Partial<ManagedDailyRevenueRankingItem & ManagedRebateBalanceRankingItem>) {
  return toFiniteNumber(row.direct_user_count ?? row.direct_users)
}

function readDirectAgentCount(row: Partial<ManagedDailyRevenueRankingItem & ManagedRebateBalanceRankingItem>) {
  return toFiniteNumber(row.direct_agent_count ?? row.direct_agents)
}

function normalizedRates(row: Pick<AffiliateDistributionTreeNode, 'current_group_rates'>) {
  return (row.current_group_rates || [])
    .map((rate: AffiliateGroupRate) => ({
      group_id: Number(rate.group_id),
      group_name: rate.group_name || groupNameById(Number(rate.group_id)),
      rate_multiplier: Number(rate.rate_multiplier),
    }))
    .filter((rate) => rate.group_id > 0 && Number.isFinite(rate.rate_multiplier) && rate.rate_multiplier > 0)
}

function createEmptyPricingRow(group?: Pick<Group, 'id' | 'rate_multiplier'>): PricingRow {
  return {
    group_id: Number(group?.id) || 0,
    rate_multiplier: toFiniteNumber(group?.rate_multiplier, 1),
  }
}

function buildPricingRows(rates: AffiliateGroupRate[] = []): PricingRow[] {
  const explicitRows = rates
    .map((rate) => ({
      group_id: Number(rate.group_id),
      rate_multiplier: toFiniteNumber(rate.rate_multiplier, Number.NaN),
    }))
    .filter((rate) => rate.group_id > 0 && Number.isFinite(rate.rate_multiplier) && rate.rate_multiplier > 0)

  const explicitMap = new Map(explicitRows.map((rate) => [rate.group_id, rate.rate_multiplier]))
  const mergedRows: PricingRow[] = normalizedGroupOptions.value.map((group) => ({
    group_id: Number(group.id),
    rate_multiplier: explicitMap.get(Number(group.id)) ?? toFiniteNumber(group.rate_multiplier, 1),
  }))

  explicitRows.forEach((rate) => {
    if (!mergedRows.some((row) => row.group_id === rate.group_id)) {
      mergedRows.push(rate)
    }
  })

  if (mergedRows.length > 0) return mergedRows
  if (explicitRows.length > 0) return explicitRows
  return [createEmptyPricingRow()]
}

function appendPricingRow() {
  const nextGroup = normalizedGroupOptions.value.find(
    (group) => !pricingRows.value.some((row) => row.group_id === Number(group.id)),
  )

  pricingRows.value.push(createEmptyPricingRow(nextGroup))
}

function groupNameById(groupId: number) {
  return normalizedGroupOptions.value.find((group) => Number(group.id) === groupId)?.name || `#${groupId}`
}

function groupOptionLabel(group: Group) {
  const parts = [group.name || `#${group.id}`]
  if (group.platform) {
    parts.push(String(group.platform))
  }
  parts.push(`默认分组成本倍率 ${toFiniteNumber(group.rate_multiplier, 1).toFixed(4)}x`)
  return parts.join(' · ')
}

function formatRateBadge(rate: { group_id: number; group_name?: string; rate_multiplier: number }) {
  return `${rate.group_name || groupNameById(rate.group_id)} ${rate.rate_multiplier.toFixed(4)}x`
}

function isGroupSelected(index: number, groupId: number) {
  return pricingRows.value.some((row, rowIndex) => rowIndex !== index && row.group_id === groupId)
}

function hasDuplicateGroups() {
  const seen = new Set<number>()

  for (const row of pricingRows.value) {
    if (row.group_id <= 0) continue
    if (seen.has(row.group_id)) return true
    seen.add(row.group_id)
  }

  return false
}

function parentLabel(row: AffiliateDistributionTreeNode) {
  const parent = tree.rows.find((item) => item.user_id === row.inviter_id)
  return parent?.email || row.inviter_email || row.inviter_username || '-'
}

function readTreeRevenueRMB(row: AffiliateDistributionTreeNode) {
  const record = row as unknown as Record<string, unknown>
  return toFiniteNumber(record.today_revenue_rmb ?? record.today_business_rmb)
}

function depthLabel(depth: number) {
  return depth === 0
    ? tt('affiliateManaged.rootNode', '根节点')
    : tt('affiliateManaged.depthLevel', `第 ${depth} 层下级`, { depth })
}

function hidePermission(permissionKey: keyof ManagedAffiliatePermissions) {
  permissions.value = { ...permissions.value, [permissionKey]: false }
  permissionNotice.value = tt('affiliateManaged.permissionsUpdated', '加载期间权限发生变化，未授权的版块已自动隐藏。')
}

async function loadGroupOptions() {
  try {
    groupOptions.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    groupOptions.value = []
    appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.groupsLoadFailed', '加载可用分组失败')))
  }
}

async function loadPermissions() {
  loadingPermissions.value = true

  try {
    permissions.value = await getManagedAffiliatePermissions()
  } catch (error) {
    permissions.value = emptyManagedAffiliatePermissions()
    if (!isForbidden(error)) {
      appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.loadFailed', '加载下级管理数据失败')))
    }
  } finally {
    loadingPermissions.value = false
  }
}

async function loadDailyRevenue() {
  if (!permissions.value.can_view_downline_daily_revenue) return
  daily.loading = true

  try {
    daily.items = (await listManagedDailyRevenueRankings({
      search: daily.search.trim() || undefined,
      date: daily.date || undefined,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    })).items ?? []
  } catch (error) {
    if (isForbidden(error)) return void hidePermission('can_view_downline_daily_revenue')
    appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.loadFailed', '加载下级管理数据失败')))
  } finally {
    daily.loading = false
  }
}

async function loadRebateBalances() {
  if (!permissions.value.can_view_downline_rebate_balances) return
  rebate.loading = true

  try {
    rebate.items = (await listManagedRebateBalanceRankings({
      search: rebate.search.trim() || undefined,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    })).items ?? []
  } catch (error) {
    if (isForbidden(error)) return void hidePermission('can_view_downline_rebate_balances')
    appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.loadFailed', '加载下级管理数据失败')))
  } finally {
    rebate.loading = false
  }
}

async function loadTree() {
  if (!permissions.value.can_manage_downline_pricing) return
  tree.loading = true

  try {
    tree.rows = await getManagedDistributionTree({ search: tree.search.trim() || undefined })
  } catch (error) {
    if (isForbidden(error)) return void hidePermission('can_manage_downline_pricing')
    appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.loadFailed', '加载下级管理数据失败')))
  } finally {
    tree.loading = false
  }
}

async function openPricingDialog(row: AffiliateDistributionTreeNode) {
  selectedNode.value = row
  pricingRows.value = buildPricingRows(normalizedRates(row))
  showPricingDialog.value = true

  try {
    const response = await getManagedUserDistributionPricing(row.user_id)
    pricingRows.value = buildPricingRows(response.group_rates)
  } catch (error) {
    if (isForbidden(error)) {
      hidePermission('can_manage_downline_pricing')
      closePricingDialog()
      return
    }

    appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.loadFailed', '加载下级管理数据失败')))
  }
}

function closePricingDialog() {
  if (savingPricing.value) return
  showPricingDialog.value = false
  selectedNode.value = null
}

async function savePricing() {
  if (!selectedNode.value || savingPricing.value) return
  savingPricing.value = true

  try {
    await updateManagedUserDistributionPricing(selectedNode.value.user_id, {
      group_rates: pricingRows.value
        .filter((row) => row.group_id > 0 && Number(row.rate_multiplier) > 0)
        .map((row) => ({
          group_id: row.group_id,
          rate_multiplier: Number(row.rate_multiplier),
        })),
    })
    appStore.showSuccess(tt('affiliateManaged.saveSuccess', '下级分组成本倍率已更新'))
    closePricingDialog()
    await loadTree()
  } catch (error) {
    if (isForbidden(error)) {
      hidePermission('can_manage_downline_pricing')
      closePricingDialog()
      return
    }

    appStore.showError(extractApiErrorMessage(error, tt('affiliateManaged.saveFailed', '更新下级分组成本倍率失败')))
  } finally {
    savingPricing.value = false
  }
}

onMounted(async () => {
  await loadGroupOptions()
  await loadPermissions()
  await Promise.all([
    permissions.value.can_view_downline_daily_revenue ? loadDailyRevenue() : Promise.resolve(),
    permissions.value.can_view_downline_rebate_balances ? loadRebateBalances() : Promise.resolve(),
    permissions.value.can_manage_downline_pricing ? loadTree() : Promise.resolve(),
  ])
})
</script>
