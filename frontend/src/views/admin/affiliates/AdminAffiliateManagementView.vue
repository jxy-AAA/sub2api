<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">代理管理</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            查看管理员、各级代理、客户的多级分销树；管理员可授权代理查看其下级营业额/返利并管理其下级倍率。
          </p>
        </div>
        <div class="flex gap-2">
          <div class="relative min-w-[280px]">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="search" class="input pl-10" placeholder="搜索邮箱、用户名、邀请码" @keyup.enter="loadTree" />
          </div>
          <button class="btn btn-secondary" :disabled="loading" @click="loadTree">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            <span>刷新</span>
          </button>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <div class="card p-4">
          <div class="text-sm text-gray-500 dark:text-dark-400">总节点</div>
          <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ rows.length }}</div>
        </div>
        <div class="card p-4">
          <div class="text-sm text-gray-500 dark:text-dark-400">代理数</div>
          <div class="mt-1 text-2xl font-semibold text-primary-600">{{ agentCount }}</div>
        </div>
        <div class="card p-4">
          <div class="text-sm text-gray-500 dark:text-dark-400">客户数</div>
          <div class="mt-1 text-2xl font-semibold text-emerald-600">{{ customerCount }}</div>
        </div>
        <div class="card p-4">
          <div class="text-sm text-gray-500 dark:text-dark-400">返利余额</div>
          <div class="mt-1 text-2xl font-semibold text-amber-600">{{ formatRMB(totalBalanceRMB) }}</div>
        </div>
      </div>

      <div class="card p-5">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">树形层级</h3>
            <p class="text-sm text-gray-500 dark:text-dark-400">支持无限层级：管理员/根节点 → 第 1 层下级 → 第 2 层下级 → 更深层级</p>
          </div>
        </div>
        <div v-if="loading" class="py-10 text-center text-sm text-gray-500">加载中...</div>
        <div v-else-if="treeRoots.length === 0" class="rounded-xl border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500 dark:border-dark-700">
          暂无代理数据
        </div>
        <div v-else class="space-y-3">
          <TreeNode
            v-for="node in treeRoots"
            :key="node.user_id"
            :node="node"
            @edit="openPricingDialog"
            @authorize="openPermissionsDialog"
          />
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-200 p-5 dark:border-dark-700">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">代理/客户列表</h3>
          <p class="text-sm text-gray-500 dark:text-dark-400">平铺查看所有层级；管理员可授权代理查看其下级数据，并编辑其下级的模型倍率。</p>
        </div>
        <div class="grid gap-3 border-b border-gray-200 p-5 md:grid-cols-4 dark:border-dark-700">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">层级筛选</label>
            <select v-model="levelFilter" class="input">
              <option value="all">全部层级</option>
              <option v-for="depth in availableDepths" :key="depth" :value="String(depth)">{{ depthFilterLabel(depth) }}</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">身份筛选</label>
            <select v-model="agentFilter" class="input">
              <option value="all">全部身份</option>
              <option value="agent">仅代理</option>
              <option value="customer">仅客户</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">上级代理搜索</label>
            <input v-model="parentSearch" class="input" placeholder="搜索上级邮箱、用户名、ID" />
          </div>
          <div class="flex items-end">
            <button class="btn btn-secondary w-full" @click="resetLocalFilters">清空本地筛选</button>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">层级</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">用户</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">上级</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">直属人数</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">邀请码</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">今日营业额</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">今日返利</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">模型倍率</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">返利余额</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-for="row in filteredRows" :key="row.user_id">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ levelLabel(row) }}</td>
                <td class="px-4 py-3">
                  <div class="text-sm font-medium text-gray-900 dark:text-white">{{ row.username || row.email || `#${row.user_id}` }}</div>
                  <div class="text-xs text-gray-500">{{ row.email }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ parentLabel(row) }}</td>
                <td class="px-4 py-3 text-right text-sm text-gray-700 dark:text-gray-300">{{ directChildrenLabel(row) }}</td>
                <td class="px-4 py-3 font-mono text-sm text-gray-700 dark:text-gray-300">{{ row.invite_code || '-' }}</td>
                <td class="px-4 py-3 text-right text-sm text-gray-700 dark:text-gray-300">{{ formatRMB(readTodayRevenueRMB(row), true) }}</td>
                <td class="px-4 py-3 text-right text-sm text-gray-700 dark:text-gray-300">{{ formatRMB(readTodayRebateRMB(row), true) }}</td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div v-if="normalizedRates(row).length" class="flex flex-wrap gap-1.5">
                    <span v-for="rate in normalizedRates(row)" :key="`${row.user_id}-${rate.model}`" class="rounded-full bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
                      {{ rate.model }} {{ rate.multiplier.toFixed(4) }}x
                    </span>
                  </div>
                  <span v-else class="text-gray-400">未设置</span>
                </td>
                <td class="px-4 py-3 text-right text-sm font-semibold text-amber-600">{{ formatRMB(row.current_rebate_balance_rmb) }}</td>
                <td class="px-4 py-3 text-right">
                  <div class="flex justify-end gap-2">
                    <button
                      v-if="canAuthorizeRow(row)"
                      class="btn btn-secondary btn-sm"
                      @click="openPermissionsDialog(row)"
                    >
                      授权
                    </button>
                    <button class="btn btn-secondary btn-sm" @click="openPricingDialog(row)">编辑倍率</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <BaseDialog :show="showPermissionsDialog" title="代理权限授权" width="normal" @close="closePermissionsDialog">
      <div class="space-y-4">
        <div v-if="permissionTarget" class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900">
          <div class="text-sm text-gray-500">目标用户</div>
          <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ permissionTarget.email }}</div>
          <div class="mt-1 text-sm text-gray-500">{{ levelLabel(permissionTarget) }} · {{ permissionTarget.invite_code || '无邀请码' }}</div>
          <div v-if="permissionsMetaLabel" class="mt-2 text-xs text-gray-500">{{ permissionsMetaLabel }}</div>
        </div>

        <div v-if="permissionsLoading" class="py-6 text-center text-sm text-gray-500">加载权限中...</div>
        <div v-else class="space-y-3">
          <div class="flex items-start justify-between gap-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">查看下级每日营业额</div>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">允许该代理查看自己所有下级的每日营业额与消耗统计。</p>
            </div>
            <Toggle v-model="permissionsForm.can_view_downline_daily_revenue" />
          </div>
          <div class="flex items-start justify-between gap-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">查看下级返利额度</div>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">允许该代理查看自己所有下级的返利额度、返利摘要与归档相关统计。</p>
            </div>
            <Toggle v-model="permissionsForm.can_view_downline_rebate_balances" />
          </div>
          <div class="flex items-start justify-between gap-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">管理下级模型倍率</div>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">允许该代理编辑自己所有下级的模型倍率，不包含其他分支。</p>
            </div>
            <Toggle v-model="permissionsForm.can_manage_downline_pricing" />
          </div>
        </div>

        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" :disabled="permissionsSaving" @click="closePermissionsDialog">取消</button>
          <button class="btn btn-primary" :disabled="permissionsLoading || permissionsSaving || !permissionTarget" @click="savePermissions">
            {{ permissionsSaving ? '保存中...' : '保存授权' }}
          </button>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :show="showPricingDialog" title="编辑模型倍率" width="wide" @close="closePricingDialog">
      <div class="space-y-4">
        <div v-if="selectedNode" class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900">
          <div class="text-sm text-gray-500">目标用户</div>
          <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ selectedNode.email }}</div>
          <div class="mt-1 text-sm text-gray-500">{{ levelLabel(selectedNode) }} · 当前返利 {{ formatRMB(selectedNode.current_rebate_balance_rmb) }}</div>
        </div>

        <div class="space-y-3">
          <div v-for="(row, index) in pricingRows" :key="index" class="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-500">模型</label>
              <input v-model="row.model" class="input" placeholder="gpt-5.4" />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-500">倍率</label>
              <input v-model.number="row.multiplier" type="number" min="0.0001" step="0.0001" class="input" placeholder="1.6000" />
            </div>
            <div class="flex items-end">
              <button class="btn btn-secondary btn-sm w-full" @click="pricingRows.splice(index, 1)">删除</button>
            </div>
          </div>
        </div>

        <button class="btn btn-secondary btn-sm" @click="pricingRows.push({ model: 'gpt-5.4', multiplier: 1 })">新增模型倍率</button>

        <div class="rounded-lg bg-blue-50 p-3 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-200">
          说明：1.8x 表示 1.8 毛 / 1 刀。返利 = 每日营业额 ×（下级倍率 - 上级成本倍率）÷ 10。
        </div>

        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" :disabled="savingPricing" @click="closePricingDialog">取消</button>
          <button class="btn btn-primary" :disabled="savingPricing || !canSavePricing" @click="savePricing">
            {{ savingPricing ? '保存中...' : '保存倍率' }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, type Component } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDisplayDateTime } from '@/utils/format'
import {
  affiliatesAPI,
  type AffiliateAgentPermissions,
  type AffiliateDistributionTreeNode,
  type AffiliateModelRate,
} from '@/api/admin/affiliates'

interface TreeDisplayNode extends AffiliateDistributionTreeNode {
  children: TreeDisplayNode[]
}

interface PricingRow {
  model: string
  multiplier: number
}

const appStore = useAppStore()
const loading = ref(false)
const savingPricing = ref(false)
const permissionsLoading = ref(false)
const permissionsSaving = ref(false)
const search = ref('')
const levelFilter = ref('all')
const agentFilter = ref<'all' | 'agent' | 'customer'>('all')
const parentSearch = ref('')
const rows = ref<AffiliateDistributionTreeNode[]>([])
const showPricingDialog = ref(false)
const showPermissionsDialog = ref(false)
const selectedNode = ref<AffiliateDistributionTreeNode | null>(null)
const permissionTarget = ref<AffiliateDistributionTreeNode | null>(null)
const permissionsMeta = ref<AffiliateAgentPermissions | null>(null)
const pricingRows = ref<PricingRow[]>([])
const permissionsForm = ref({
  can_view_downline_daily_revenue: false,
  can_view_downline_rebate_balances: false,
  can_manage_downline_pricing: false,
})

const flatRows = computed(() => [...rows.value].sort((a, b) => a.depth - b.depth || a.user_id - b.user_id))
const availableDepths = computed(() => {
  const depths = new Set<number>()
  rows.value.forEach((row) => {
    const depth = Number(row.depth)
    if (Number.isInteger(depth) && depth >= 0) depths.add(depth)
  })
  return [...depths].sort((a, b) => a - b)
})
const filteredRows = computed(() => flatRows.value.filter((row) => {
  if (levelFilter.value !== 'all' && String(row.depth) !== levelFilter.value) {
    return false
  }
  if (agentFilter.value === 'agent' && !isAgentRow(row)) return false
  if (agentFilter.value === 'customer' && isAgentRow(row)) return false
  const keyword = parentSearch.value.trim().toLowerCase()
  if (keyword) {
    const parent = resolveParent(row)
    const haystack = [
      parent?.email,
      parent?.username,
      parent?.user_id ? String(parent.user_id) : '',
      row.inviter_email,
      row.inviter_username,
      row.inviter_id ? String(row.inviter_id) : '',
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    if (!haystack.includes(keyword)) return false
  }
  return true
}))
const agentCount = computed(() => rows.value.filter((row) => row.is_admin || hasChildren(row.user_id)).length)
const customerCount = computed(() => rows.value.length - agentCount.value)
const totalBalanceRMB = computed(() => rows.value.reduce((sum, row) => sum + Number(row.current_rebate_balance_rmb || 0), 0))
const permissionsMetaLabel = computed(() => {
  if (!permissionsMeta.value) return ''
  const segments: string[] = []
  if (permissionsMeta.value.granted_by_email) {
    segments.push(`授权人：${permissionsMeta.value.granted_by_email}`)
  }
  if (permissionsMeta.value.updated_at) {
    segments.push(`更新时间：${formatDisplayDateTime(permissionsMeta.value.updated_at)}`)
  }
  return segments.join(' · ')
})

const treeRoots = computed<TreeDisplayNode[]>(() => {
  const map = new Map<number, TreeDisplayNode>()
  rows.value.forEach((row) => map.set(row.user_id, { ...row, children: [] }))
  const roots: TreeDisplayNode[] = []
  map.forEach((node) => {
    if (node.inviter_id && map.has(node.inviter_id)) {
      map.get(node.inviter_id)!.children.push(node)
    } else {
      roots.push(node)
    }
  })
  const sortTree = (nodes: TreeDisplayNode[]) => {
    nodes.sort((a, b) => a.depth - b.depth || a.user_id - b.user_id)
    nodes.forEach((node) => sortTree(node.children))
  }
  sortTree(roots)
  return roots
})

const canSavePricing = computed(() => pricingRows.value.some((row) => row.model.trim() && Number(row.multiplier) > 0))

const TreeNode: Component = defineComponent({
  name: 'TreeNode',
  props: {
    node: { type: Object as () => TreeDisplayNode, required: true },
  },
  emits: ['edit', 'authorize'],
  setup(props, { emit }) {
    const expanded = ref(true)
    return (): ReturnType<typeof h> => h('div', { class: 'rounded-xl border border-gray-200 p-3 dark:border-dark-700' }, [
      h('div', { class: 'flex flex-col gap-3 md:flex-row md:items-center md:justify-between' }, [
        h('div', { class: 'flex items-start gap-3' }, [
          props.node.children.length
            ? h('button', { class: 'mt-0.5 rounded p-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700', onClick: () => { expanded.value = !expanded.value } }, expanded.value ? '−' : '+')
            : h('span', { class: 'mt-0.5 w-6' }, ''),
          h('div', [
            h('div', { class: 'flex flex-wrap items-center gap-2' }, [
              h('span', { class: 'font-medium text-gray-900 dark:text-white' }, props.node.email || `#${props.node.user_id}`),
              h('span', { class: 'rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300' }, levelLabel(props.node)),
            ]),
            h('div', { class: 'mt-1 text-xs text-gray-500' }, `邀请码：${props.node.invite_code || '-'} · 返利余额：${formatRMB(props.node.current_rebate_balance_rmb)}`),
            h('div', { class: 'mt-1 text-xs text-gray-500' }, normalizedRates(props.node).map((rate) => `${rate.model} ${rate.multiplier.toFixed(4)}x`).join('，') || '未设置模型倍率'),
          ]),
        ]),
        h('div', { class: 'flex flex-wrap justify-end gap-2' }, [
          canAuthorizeRow(props.node)
            ? h('button', { class: 'btn btn-secondary btn-sm', onClick: () => emit('authorize', props.node) }, '授权')
            : null,
          h('button', { class: 'btn btn-secondary btn-sm', onClick: () => emit('edit', props.node) }, '编辑倍率'),
        ]),
      ]),
      expanded.value && props.node.children.length
        ? h('div', { class: 'mt-3 space-y-3 pl-6' }, props.node.children.map((child) => h(TreeNode, { node: child, onEdit: (node: TreeDisplayNode) => emit('edit', node), onAuthorize: (node: TreeDisplayNode) => emit('authorize', node) })))
        : null,
    ])
  },
})

onMounted(loadTree)

async function loadTree() {
  loading.value = true
  try {
    rows.value = await affiliatesAPI.getDistributionTree({ search: search.value.trim() })
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '加载代理管理数据失败'))
  } finally {
    loading.value = false
  }
}

function hasChildren(userId: number) {
  return rows.value.some((row) => row.inviter_id === userId)
}

function isAgentRow(row: AffiliateDistributionTreeNode) {
  if (typeof row.is_agent === 'boolean') return row.is_agent
  return row.is_admin || hasChildren(row.user_id)
}

function canAuthorizeRow(row: AffiliateDistributionTreeNode) {
  return !row.is_admin
}

function resolveParent(row: AffiliateDistributionTreeNode) {
  if (!row.inviter_id) return null
  return rows.value.find((item) => item.user_id === row.inviter_id) ?? null
}

function parentLabel(row: AffiliateDistributionTreeNode) {
  if (!row.inviter_id) return '-'
  const parent = resolveParent(row)
  return parent?.email || row.inviter_email || row.inviter_username || `#${row.inviter_id}`
}

function directChildrenLabel(row: AffiliateDistributionTreeNode) {
  const directCount = readDirectChildrenCount(row)
  if (directCount === null) return '—'
  const agentCountValue = toFiniteNumber(row.direct_agent_count)
  const userCountValue = toFiniteNumber(row.direct_user_count)
  if (agentCountValue !== null || userCountValue !== null) {
    return `${directCount}（代${agentCountValue ?? 0}/客${userCountValue ?? 0}）`
  }
  return String(directCount)
}

function levelLabel(row: Pick<AffiliateDistributionTreeNode, 'depth' | 'is_admin' | 'is_root_admin'>) {
  if (row.is_root_admin) return `根管理员 · ${depthLabel(row.depth)}`
  if (row.is_admin) return `管理员 · ${depthLabel(row.depth)}`
  return depthLabel(row.depth)
}

function depthLabel(depth: number) {
  const normalizedDepth = Number.isFinite(Number(depth)) && Number(depth) >= 0 ? Number(depth) : 0
  if (normalizedDepth === 0) return '根节点'
  return `第 ${normalizedDepth} 层下级`
}

function depthFilterLabel(depth: number) {
  return depth === 0 ? '根节点/管理员' : `第 ${depth} 层下级`
}

function normalizedRates(row: Pick<AffiliateDistributionTreeNode, 'current_model_rates'>): Array<{ model: string; multiplier: number }> {
  return (row.current_model_rates || [])
    .map((rate: AffiliateModelRate) => ({
      model: (rate.model || rate.model_name || '').trim(),
      multiplier: Number(rate.multiplier),
    }))
    .filter((rate) => rate.model && Number.isFinite(rate.multiplier))
}

function readTodayRevenueRMB(row: AffiliateDistributionTreeNode) {
  return toFiniteNumber(row.today_revenue_rmb ?? row.today_business_rmb ?? row.today_revenue_usd ?? row.today_business_usd)
}

function readTodayRebateRMB(row: AffiliateDistributionTreeNode) {
  return toFiniteNumber(row.today_rebate_rmb ?? row.today_rebate_amount)
}

function readDirectChildrenCount(row: AffiliateDistributionTreeNode) {
  const directCount = toFiniteNumber(row.direct_children_count ?? row.direct_count ?? row.direct_member_count)
  if (directCount !== null) return directCount
  const agentCountValue = toFiniteNumber(row.direct_agent_count)
  const userCountValue = toFiniteNumber(row.direct_user_count)
  if (agentCountValue !== null || userCountValue !== null) {
    return (agentCountValue ?? 0) + (userCountValue ?? 0)
  }
  if (isAgentRow(row)) {
    return rows.value.filter((item) => item.inviter_id === row.user_id).length
  }
  return null
}

function toFiniteNumber(value: unknown) {
  const amount = Number(value)
  return Number.isFinite(amount) ? amount : null
}

function resetLocalFilters() {
  levelFilter.value = 'all'
  agentFilter.value = 'all'
  parentSearch.value = ''
}

async function openPermissionsDialog(row: AffiliateDistributionTreeNode) {
  permissionTarget.value = row
  permissionsMeta.value = null
  permissionsForm.value = {
    can_view_downline_daily_revenue: false,
    can_view_downline_rebate_balances: false,
    can_manage_downline_pricing: false,
  }
  showPermissionsDialog.value = true
  permissionsLoading.value = true
  try {
    const result = await affiliatesAPI.getUserDistributionPermissions(row.user_id)
    permissionsMeta.value = result
    permissionsForm.value = {
      can_view_downline_daily_revenue: result.can_view_downline_daily_revenue,
      can_view_downline_rebate_balances: result.can_view_downline_rebate_balances,
      can_manage_downline_pricing: result.can_manage_downline_pricing,
    }
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '加载代理权限失败'))
    showPermissionsDialog.value = false
    permissionTarget.value = null
  } finally {
    permissionsLoading.value = false
  }
}

function closePermissionsDialog() {
  if (permissionsSaving.value) return
  showPermissionsDialog.value = false
}

async function savePermissions() {
  if (!permissionTarget.value) return
  permissionsSaving.value = true
  try {
    const result = await affiliatesAPI.updateUserDistributionPermissions(permissionTarget.value.user_id, {
      ...permissionsForm.value,
    })
    permissionsMeta.value = result
    permissionsForm.value = {
      can_view_downline_daily_revenue: result.can_view_downline_daily_revenue,
      can_view_downline_rebate_balances: result.can_view_downline_rebate_balances,
      can_manage_downline_pricing: result.can_manage_downline_pricing,
    }
    appStore.showSuccess('代理权限已更新')
    showPermissionsDialog.value = false
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '保存代理权限失败'))
  } finally {
    permissionsSaving.value = false
  }
}

function openPricingDialog(row: AffiliateDistributionTreeNode) {
  selectedNode.value = row
  pricingRows.value = normalizedRates(row).map((rate) => ({ ...rate }))
  if (pricingRows.value.length === 0) {
    pricingRows.value = [{ model: 'gpt-5.4', multiplier: 1 }]
  }
  showPricingDialog.value = true
}

function closePricingDialog() {
  if (savingPricing.value) return
  showPricingDialog.value = false
}

async function savePricing() {
  if (!selectedNode.value) return
  const modelRates = pricingRows.value
    .map((row) => ({ model: row.model.trim(), multiplier: Number(row.multiplier) }))
    .filter((row) => row.model && Number.isFinite(row.multiplier) && row.multiplier > 0)
  if (modelRates.length === 0) {
    appStore.showError('请至少填写一个有效模型倍率')
    return
  }
  savingPricing.value = true
  try {
    const result = await affiliatesAPI.updateUserDistributionPricing(selectedNode.value.user_id, { model_rates: modelRates })
    const index = rows.value.findIndex((row) => row.user_id === selectedNode.value?.user_id)
    if (index >= 0) {
      rows.value[index] = {
        ...rows.value[index],
        current_model_rates: result.model_rates,
      }
    }
    appStore.showSuccess('模型倍率已更新')
    showPricingDialog.value = false
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '保存模型倍率失败'))
  } finally {
    savingPricing.value = false
  }
}

function formatRMB(value: number | null | undefined, allowEmpty = false) {
  const amount = toFiniteNumber(value)
  if (amount === null) return allowEmpty ? '—' : '¥0.00'
  return `¥${amount.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}
</script>
