<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-3">
          <div class="grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto_auto_auto]">
            <div class="relative">
              <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                v-model="filters.search"
                type="text"
                class="input pl-10"
                :placeholder="pageMeta.searchPlaceholder"
                @input="debounceLoad"
              />
            </div>
            <input
              v-if="props.type === 'invites'"
              v-model="filters.date"
              type="date"
              class="input w-full sm:w-44"
              title="营业额日期"
              @change="reloadFromFirstPage"
            />
            <input
              v-if="props.type === 'transfers'"
              v-model="filters.month"
              type="month"
              class="input w-full sm:w-44"
              title="归档月份"
              @change="reloadFromFirstPage"
            />
            <button class="btn btn-secondary px-3" :disabled="loading" title="刷新" @click="loadRecords">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>

          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <AffiliateAdminSummaryCard
              v-for="card in summaryCards"
              :key="card.label"
              :label="card.label"
              :value="card.value"
              :hint="card.hint"
            />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="records"
          :loading="loading"
          :row-key="resolveRowKey"
          :server-side-sort="true"
          :default-sort-key="pageMeta.defaultSortKey"
          :default-sort-order="pageMeta.defaultSortOrder"
          :sort-storage-key="pageMeta.sortStorageKey"
          @sort="handleSort"
        >
          <template #cell-rank="{ row }">
            <div class="flex items-center gap-2">
              <span class="inline-flex h-8 w-8 items-center justify-center rounded-full bg-primary-50 text-sm font-semibold text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">
                {{ displayRank(row) }}
              </span>
            </div>
          </template>

          <template #cell-agent="{ row }">
            <AffiliateAdminUserCell
              :user-id="getUserId(row)"
              :email="getEmail(row)"
              :username="getUsername(row)"
            />
          </template>

          <template #cell-date="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDate(getDateValue(row)) }}</span>
          </template>

          <template #cell-month="{ row }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ getMonthValue(row) || '-' }}</span>
          </template>

          <template #cell-revenue="{ row }">
            <span class="text-sm font-semibold text-emerald-600 dark:text-emerald-400">{{ formatRMB(getRevenueAmount(row)) }}</span>
          </template>

          <template #cell-balance="{ row }">
            <span class="text-sm font-semibold text-amber-600 dark:text-amber-400">{{ formatRMB(getRebateBalance(row)) }}</span>
          </template>

          <template #cell-archived_rebate="{ row }">
            <span class="text-sm font-semibold text-emerald-600 dark:text-emerald-400">{{ formatRMB(getArchivedAmount(row)) }}</span>
          </template>

          <template #cell-downlines="{ row }">
            <div class="space-y-0.5 text-sm text-gray-700 dark:text-gray-300">
              <div>直属总数：{{ getDownlineCount(row) }}</div>
              <div>直属用户：{{ getDirectUserCount(row) }}</div>
              <div>直属代理：{{ getDirectAgentCount(row) }}</div>
            </div>
          </template>

          <template #cell-usage_summary="{ row }">
            <div class="space-y-0.5 text-sm text-gray-700 dark:text-gray-300">
              <div>直属消耗合计：{{ formatRMB(getDirectUsageAmount(row)) }}</div>
              <div v-if="props.type === 'invites'">直属用户消耗：{{ formatRMB(getDirectUserUsageAmount(row)) }}</div>
              <div v-if="props.type === 'invites'">直属代理消耗：{{ formatRMB(getDirectAgentUsageAmount(row)) }}</div>
              <div v-if="props.type === 'rebates'">今日返利：{{ formatRMB(getTodayRebateAmount(row)) }}</div>
              <div v-if="props.type === 'rebates'">本月返利：{{ formatRMB(getMonthlyRebateAmount(row)) }}</div>
            </div>
          </template>

          <template #cell-status="{ row }">
            <span
              class="inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
              :class="getArchiveStatus(row).cleared ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'"
            >
              {{ getArchiveStatus(row).label }}
            </span>
          </template>

          <template #cell-remark="{ row }">
            <div class="max-w-64 whitespace-normal text-sm text-gray-700 dark:text-gray-300">
              {{ getArchiveRemark(row) || '-' }}
            </div>
          </template>

          <template #cell-updated_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(getUpdatedAt(row)) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div v-if="props.type === 'rebates'" class="flex justify-end">
              <button
                v-if="canAdjustRebateBalance"
                class="btn btn-secondary btn-sm"
                :disabled="adjustSubmitting"
                @click="openAdjustDialog(row)"
              >
                调整额度
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showAdjustDialog"
      title="调整返利额度"
      width="normal"
      @close="closeAdjustDialog"
    >
      <div class="space-y-4">
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-sm text-gray-500 dark:text-dark-400">目标代理</div>
          <AffiliateAdminUserCell
            class="mt-2"
            :user-id="adjustForm.agent_user_id"
            :email="adjustForm.email"
            :username="adjustForm.username"
          />
          <div class="mt-3 text-sm text-gray-600 dark:text-dark-300">
            当前额度：<span class="font-semibold text-amber-600 dark:text-amber-400">{{ formatRMB(adjustForm.current_balance) }}</span>
          </div>
        </div>

        <div>
          <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">设置新的返利额度</label>
          <input
            v-model="adjustForm.rebate_balance_rmb"
            type="number"
            min="0"
            step="0.01"
            class="input"
            placeholder="例如 120.50"
          />
        </div>

        <div>
          <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">调整备注</label>
          <textarea
            v-model="adjustForm.remark"
            class="input min-h-[108px]"
            placeholder="请填写结算说明或调整原因"
          />
        </div>

        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" :disabled="adjustSubmitting" @click="closeAdjustDialog">取消</button>
          <button class="btn btn-primary" :disabled="adjustSubmitting" @click="submitAdjustment">
            {{ adjustSubmitting ? '提交中...' : '确认调整' }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import AffiliateAdminSummaryCard from '@/components/admin/affiliate/AffiliateAdminSummaryCard.vue'
import AffiliateAdminUserCell from '@/components/admin/affiliate/AffiliateAdminUserCell.vue'
import {
  adjustRebateBalance,
  listDailyRevenueRankings,
  listMonthlyArchives,
  listRebateBalanceRankings,
  type AdminAffiliateRecordType,
  type AffiliateLeaderboardResponse,
  type DailyRevenueRankingItem,
  type MonthlyArchiveItem,
  type RebateBalanceRankingItem,
} from '@/components/admin/affiliate/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDisplayDateTime } from '@/utils/format'

type RowItem = DailyRevenueRankingItem | RebateBalanceRankingItem | MonthlyArchiveItem

const props = defineProps<{
  type: AdminAffiliateRecordType
}>()

const appStore = useAppStore()
const authStore = useAuthStore()
const loading = ref(false)
const adjustSubmitting = ref(false)
const showAdjustDialog = ref(false)
const records = ref<RowItem[]>([])
const summary = ref<Record<string, unknown>>({})
const pagination = reactive({ page: 1, page_size: 20, total: 0, pages: 0 })
const filters = reactive({
  search: '',
  date: todayDate(),
  month: currentMonth(),
})
const sortState = reactive<{ sort_by: string; sort_order: 'asc' | 'desc' }>({
  sort_by: props.type === 'rebates'
    ? 'current_rebate_balance_rmb'
    : props.type === 'transfers'
      ? 'archive_month'
      : 'business_rmb',
  sort_order: 'desc',
})
const adjustForm = reactive({
  agent_user_id: 0,
  email: '',
  username: '',
  current_balance: 0,
  rebate_balance_rmb: '',
  remark: '',
})

const canAdjustRebateBalance = computed(() => authStore.user?.is_root_admin === true)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

const pageMeta = computed(() => {
  if (props.type === 'invites') {
    return {
      searchPlaceholder: '搜索代理 ID / 邮箱 / 用户名',
      defaultSortKey: 'revenue',
      defaultSortOrder: 'desc' as const,
      sortStorageKey: 'admin-affiliate-daily-revenue-sort',
    }
  }
  if (props.type === 'rebates') {
    return {
      searchPlaceholder: '搜索代理 ID / 邮箱 / 用户名',
      defaultSortKey: 'balance',
      defaultSortOrder: 'desc' as const,
      sortStorageKey: 'admin-affiliate-rebate-balance-sort',
    }
  }
  return {
    searchPlaceholder: '搜索代理 ID / 邮箱 / 用户名',
    defaultSortKey: 'month',
    defaultSortOrder: 'desc' as const,
    sortStorageKey: 'admin-affiliate-monthly-archive-sort',
  }
})

const columns = computed<Column[]>(() => {
  if (props.type === 'invites') {
    return [
      { key: 'rank', label: '排名', sortable: true, class: 'min-w-[92px]' },
      { key: 'agent', label: '代理', sortable: true, class: 'min-w-[240px]' },
      { key: 'date', label: '日期', sortable: true, class: 'min-w-[140px]' },
      { key: 'revenue', label: '营业额', sortable: true, class: 'min-w-[160px]' },
      { key: 'downlines', label: '直属下级摘要', sortable: false, class: 'min-w-[190px] whitespace-normal' },
      { key: 'usage_summary', label: '消耗摘要', sortable: false, class: 'min-w-[220px] whitespace-normal' },
      { key: 'updated_at', label: '更新时间', sortable: true, class: 'min-w-[180px]' },
    ]
  }
  if (props.type === 'rebates') {
    const baseColumns: Column[] = [
      { key: 'rank', label: '排名', sortable: true, class: 'min-w-[92px]' },
      { key: 'agent', label: '代理', sortable: true, class: 'min-w-[240px]' },
      { key: 'balance', label: '当前返利额度', sortable: true, class: 'min-w-[180px]' },
      { key: 'downlines', label: '直属下级摘要', sortable: false, class: 'min-w-[190px] whitespace-normal' },
      { key: 'usage_summary', label: '返利摘要', sortable: false, class: 'min-w-[220px] whitespace-normal' },
      { key: 'updated_at', label: '更新时间', sortable: true, class: 'min-w-[180px]' },
    ]
    if (canAdjustRebateBalance.value) {
      baseColumns.push({ key: 'actions', label: '操作', sortable: false, class: 'min-w-[130px]' })
    }
    return baseColumns
  }
  return [
    { key: 'month', label: '归档月份', sortable: true, class: 'min-w-[140px]' },
    { key: 'agent', label: '代理', sortable: true, class: 'min-w-[240px]' },
    { key: 'archived_rebate', label: '归档返利额度', sortable: true, class: 'min-w-[180px]' },
    { key: 'status', label: '清零状态', sortable: false, class: 'min-w-[140px]' },
    { key: 'remark', label: '备注', sortable: false, class: 'min-w-[240px] whitespace-normal' },
    { key: 'updated_at', label: '归档时间', sortable: true, class: 'min-w-[180px]' },
  ]
})

const summaryCards = computed(() => {
  if (props.type === 'invites') {
    return [
      {
        label: '营业额合计',
        value: formatRMB(readNumber(summary.value, ['total_business_rmb', 'total_revenue_rmb', 'total_revenue_usd']) || sumBy(records.value, getRevenueAmount)),
        hint: '当前筛选范围内所有代理营业额总和',
      },
      {
        label: '代理数量',
        value: String(readNumber(summary.value, ['total_agents']) || pagination.total || records.value.length),
        hint: '参与排名的代理总数',
      },
      {
        label: '直属下级合计',
        value: String(readNumber(summary.value, ['total_direct_downlines']) || sumBy(records.value, getDownlineCount)),
        hint: '所有代理直属用户与代理数量总和',
      },
      {
        label: '直属消耗合计',
        value: formatRMB(readNumber(summary.value, ['total_direct_usage_rmb', 'total_direct_usage_usd']) || sumBy(records.value, getDirectUsageAmount)),
        hint: '直属客户与下级代理消耗总和',
      },
    ]
  }
  if (props.type === 'rebates') {
    return [
      {
        label: '返利额度合计',
        value: formatRMB(readNumber(summary.value, ['total_rebate_balance_rmb']) || sumBy(records.value, getRebateBalance)),
        hint: '当前可结算返利额度总和',
      },
      {
        label: '代理数量',
        value: String(readNumber(summary.value, ['total_agents']) || pagination.total || records.value.length),
        hint: '参与返利排行的代理总数',
      },
      {
        label: '直属下级合计',
        value: String(readNumber(summary.value, ['total_direct_downlines']) || sumBy(records.value, getDownlineCount)),
        hint: '所有代理直属用户与代理数量总和',
      },
      {
        label: '本月返利合计',
        value: formatRMB(sumBy(records.value, getMonthlyRebateAmount)),
        hint: '当前页统计的本月返利总额',
      },
    ]
  }
  return [
    {
      label: '归档返利合计',
      value: formatRMB(readNumber(summary.value, ['total_archived_rebate_rmb']) || sumBy(records.value, getArchivedAmount)),
      hint: '每月归档返利总额',
    },
    {
      label: '已清零记录',
      value: String(readNumber(summary.value, ['cleared_count']) || records.value.filter((item) => getArchiveStatus(item).cleared).length),
      hint: '已完成归档并清零的代理数',
    },
    {
      label: '待处理记录',
      value: String(readNumber(summary.value, ['pending_count']) || records.value.filter((item) => !getArchiveStatus(item).cleared).length),
      hint: '归档后仍待确认的记录数',
    },
    {
      label: '归档代理数量',
      value: String(readNumber(summary.value, ['total_agents']) || pagination.total || records.value.length),
      hint: '当前筛选月份内归档代理总数',
    },
  ]
})

function userTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

function todayDate() {
  return new Date().toISOString().slice(0, 10)
}

function currentMonth() {
  return new Date().toISOString().slice(0, 7)
}

function buildParams() {
  return {
    page: pagination.page,
    page_size: pagination.page_size,
    search: filters.search.trim() || undefined,
    date: props.type === 'invites' ? filters.date || undefined : undefined,
    month: props.type === 'transfers' ? filters.month || undefined : undefined,
    sort_by: mapSortField(sortState.sort_by),
    sort_order: sortState.sort_order,
    timezone: userTimezone(),
  }
}

function mapSortField(key: string) {
  if (props.type === 'invites') {
    if (key === 'rank') return 'rank'
    if (key === 'agent') return 'agent_user_id'
    if (key === 'date') return 'revenue_date'
    if (key === 'revenue') return 'business_rmb'
    if (key === 'updated_at') return 'updated_at'
  }
  if (props.type === 'rebates') {
    if (key === 'rank') return 'rank'
    if (key === 'agent') return 'agent_user_id'
    if (key === 'balance') return 'current_rebate_balance_rmb'
    if (key === 'updated_at') return 'updated_at'
  }
  if (key === 'month') return 'archive_month'
  if (key === 'archived_rebate') return 'archived_rebate_rmb'
  if (key === 'updated_at') return 'cleared_at'
  return key
}

async function loadRecords() {
  loading.value = true
  try {
    let response: AffiliateLeaderboardResponse<RowItem>
    if (props.type === 'invites') {
      response = await listDailyRevenueRankings(buildParams())
    } else if (props.type === 'rebates') {
      response = await listRebateBalanceRankings(buildParams())
    } else {
      response = await listMonthlyArchives(buildParams())
    }
    records.value = response.items || []
    summary.value = (response.summary || {}) as Record<string, unknown>
    pagination.total = response.total || 0
    pagination.page = response.page || pagination.page
    pagination.page_size = response.page_size || pagination.page_size
    pagination.pages = response.pages || 0
  } catch (error) {
    appStore.showError(
      extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '加载代理数据失败'),
    )
  } finally {
    loading.value = false
  }
}

function debounceLoad() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => reloadFromFirstPage(), 300)
}

function reloadFromFirstPage() {
  pagination.page = 1
  void loadRecords()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadRecords()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  void loadRecords()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadRecords()
}

function resolveRowKey(row: RowItem) {
  return `${getMonthValue(row) || getDateValue(row) || 'current'}-${getUserId(row)}`
}

function displayRank(row: RowItem) {
  const rank = readNumber(row as Record<string, unknown>, ['rank'])
  if (rank > 0) return rank
  return (pagination.page - 1) * pagination.page_size + records.value.indexOf(row) + 1
}

function getUserId(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['agent_user_id', 'user_id', 'id'])
}

function getEmail(row: RowItem) {
  return readString(row as Record<string, unknown>, ['agent_email', 'user_email', 'email'])
}

function getUsername(row: RowItem) {
  return readString(row as Record<string, unknown>, ['agent_username', 'username'])
}

function getDateValue(row: RowItem) {
  return readString(row as Record<string, unknown>, ['revenue_date', 'stat_date', 'date'])
}

function getMonthValue(row: RowItem) {
  return readString(row as Record<string, unknown>, ['archive_month', 'month'])
}

function getRevenueAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, [
    'business_rmb',
    'total_business_rmb',
    'revenue_rmb',
    'daily_revenue_rmb',
    'total_revenue_usd',
    'business_usd',
    'revenue_usd',
    'daily_revenue_usd',
  ])
}

function getDownlineCount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['direct_downline_count', 'direct_users']) + readNumber(row as Record<string, unknown>, ['direct_agents'])
}

function getDirectUserCount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['direct_user_count', 'direct_users'])
}

function getDirectAgentCount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['direct_agent_count', 'direct_agents'])
}

function getDirectUsageAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['direct_total_usage_rmb', 'direct_total_usage_usd', 'business_rmb', 'business_usd'])
}

function getDirectUserUsageAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['direct_user_usage_rmb', 'direct_user_usage_usd'])
}

function getDirectAgentUsageAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['direct_agent_usage_rmb', 'direct_agent_usage_usd'])
}

function getRebateBalance(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['current_rebate_balance_rmb', 'rebate_balance_rmb', 'rebate_balance', 'total_rebate_balance_rmb'])
}

function getTodayRebateAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['today_rebate_rmb'])
}

function getMonthlyRebateAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['monthly_rebate_rmb'])
}

function getArchivedAmount(row: RowItem) {
  return readNumber(row as Record<string, unknown>, ['archived_rebate_rmb', 'archived_rebate_balance', 'archived_balance_rmb', 'opening_balance_rmb'])
}

function getArchiveStatus(row: RowItem) {
  const cleared = Boolean((row as MonthlyArchiveItem).cleared_to_zero) || readString(row as Record<string, unknown>, ['reset_status']) === 'cleared'
  return {
    cleared,
    label: cleared ? '已归档并清零' : '待确认',
  }
}

function getArchiveRemark(row: RowItem) {
  return readString(row as Record<string, unknown>, ['remark', 'note', 'operator_email'])
}

function getUpdatedAt(row: RowItem) {
  return readString(row as Record<string, unknown>, ['updated_at', 'last_calculated_at', 'last_adjusted_at', 'archived_at', 'cleared_at', 'created_at'])
}

function openAdjustDialog(row: RowItem) {
  adjustForm.agent_user_id = getUserId(row)
  adjustForm.email = getEmail(row)
  adjustForm.username = getUsername(row)
  adjustForm.current_balance = getRebateBalance(row)
  adjustForm.rebate_balance_rmb = getRebateBalance(row).toFixed(2)
  adjustForm.remark = ''
  showAdjustDialog.value = true
}

function closeAdjustDialog() {
  if (adjustSubmitting.value) return
  showAdjustDialog.value = false
}

async function submitAdjustment() {
  const targetAmount = Number(adjustForm.rebate_balance_rmb)
  if (!adjustForm.agent_user_id) {
    appStore.showError('未选择代理')
    return
  }
  if (!Number.isFinite(targetAmount) || targetAmount < 0) {
    appStore.showError('请输入有效的返利额度')
    return
  }
  if (!adjustForm.remark.trim()) {
    appStore.showError('请填写调整备注')
    return
  }

  adjustSubmitting.value = true
  try {
    await adjustRebateBalance({
      agent_user_id: adjustForm.agent_user_id,
      rebate_balance_rmb: targetAmount,
      remark: adjustForm.remark.trim(),
    })
    appStore.showSuccess('返利额度已更新')
    showAdjustDialog.value = false
    await loadRecords()
  } catch (error) {
    appStore.showError(
      extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '调整返利额度失败'),
    )
  } finally {
    adjustSubmitting.value = false
  }
}

function readString(source: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = source[key]
    if (typeof value === 'string' && value.trim()) return value
  }
  return ''
}

function readNumber(source: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = source[key]
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return 0
}

function sumBy<T>(items: T[], getter: (item: T) => number) {
  return items.reduce((sum, item) => sum + getter(item), 0)
}

function formatRMB(value: number | null | undefined) {
  const amount = Number(value || 0)
  return `¥${amount.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

function formatDate(value: string | null | undefined) {
  if (!value) return '-'
  if (/^\d{4}-\d{2}$/.test(value)) return value
  return value.slice(0, 10)
}

function formatDateTime(value: string | null | undefined) {
  return value ? formatDisplayDateTime(value) : '-'
}

onMounted(() => {
  void loadRecords()
})
</script>
