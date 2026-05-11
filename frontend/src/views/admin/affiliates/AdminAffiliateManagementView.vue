<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">代理管理</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            管理代理树、默认显式倍率、邀请码倍率，以及管理员自己的成本倍率与上游渠道。
          </p>
        </div>
        <div class="flex gap-2">
          <div class="relative min-w-[280px]">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="search"
              class="input pl-10"
              placeholder="搜索邮箱、用户名、邀请码"
              @keyup.enter="loadTree"
            />
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
          <div class="text-sm text-gray-500 dark:text-dark-400">普通用户数</div>
          <div class="mt-1 text-2xl font-semibold text-emerald-600">{{ customerCount }}</div>
        </div>
        <div class="card p-4">
          <div class="text-sm text-gray-500 dark:text-dark-400">返利余额</div>
          <div class="mt-1 text-2xl font-semibold text-amber-600">{{ formatRMB(totalBalanceRMB) }}</div>
        </div>
      </div>

      <div class="card p-5">
        <div class="mb-4 flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">默认显式分组倍率</h3>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              未使用邀请码注册、且尚未被单独编辑倍率的用户，将继承这里的显式分组成本倍率。
            </p>
          </div>
          <button class="btn btn-primary" :disabled="defaultPricingSaving || !canSaveDefaultPricing" @click="saveDefaultPricing">
            {{ defaultPricingSaving ? '保存中...' : '保存默认倍率' }}
          </button>
        </div>

        <div v-if="defaultPricingLoading" class="py-8 text-center text-sm text-gray-500">加载默认倍率中...</div>
        <div v-else-if="defaultPricingRows.length" class="space-y-3">
          <div
            v-for="row in defaultPricingRows"
            :key="`default-${row.group_id}`"
            class="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px]"
          >
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-500">分组</label>
              <div class="input flex items-center bg-gray-50 text-sm text-gray-600 dark:bg-dark-900 dark:text-gray-300">
                {{ pricingRowGroupLabel(row.group_id) }}
              </div>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-500">显式倍率</label>
              <input v-model.number="row.rate_multiplier" type="number" min="0.0001" step="0.0001" class="input" placeholder="1.0000" />
            </div>
          </div>

          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary btn-sm" :disabled="defaultPricingSaving" @click="loadDefaultPricing">重新加载</button>
          </div>
        </div>
        <div
          v-else
          class="rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700"
        >
          暂无可配置的启用分组
        </div>
      </div>

      <div class="card p-5">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">代理树</h3>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              管理员节点也可直接编辑自身成本倍率、邀请码倍率和上游渠道。
            </p>
          </div>
        </div>
        <div v-if="loading" class="py-10 text-center text-sm text-gray-500">加载中...</div>
        <div
          v-else-if="treeRoots.length === 0"
          class="rounded-xl border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500 dark:border-dark-700"
        >
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
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">代理 / 用户列表</h3>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            明确区分自身成本倍率与邀请码倍率，并可直接查看当前上级。
          </p>
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
              <option value="customer">仅普通用户</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">上级搜索</label>
            <input v-model="parentSearch" class="input" placeholder="邮箱、用户名或 ID" />
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
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">当前上级</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">邀请码</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">自身成本倍率</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">邀请码倍率</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">直属人数</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">今日返利</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">返利余额</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-for="row in filteredRows" :key="row.user_id">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ levelLabel(row) }}</td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap items-center gap-2">
                    <div class="text-sm font-medium text-gray-900 dark:text-white">{{ row.username || row.email || `#${row.user_id}` }}</div>
                    <span
                      v-if="currentUserId === row.user_id"
                      class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                    >
                      你自己
                    </span>
                    <span
                      v-if="row.is_root_admin"
                      class="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                    >
                      根管理员
                    </span>
                  </div>
                  <div class="text-xs text-gray-500">{{ row.email }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ parentLabel(row) }}</td>
                <td class="px-4 py-3 font-mono text-sm text-gray-700 dark:text-gray-300">{{ row.invite_code || '-' }}</td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div v-if="normalizeRateBadges(row.current_group_rates).length" class="flex flex-wrap gap-1.5">
                    <span
                      v-for="rate in normalizeRateBadges(row.current_group_rates)"
                      :key="`cost-${row.user_id}-${rate.group_id}`"
                      class="rounded-full bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                    >
                      {{ formatGroupRateBadge(rate) }}
                    </span>
                  </div>
                  <span v-else class="text-gray-400">未设置</span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div v-if="normalizeRateBadges(row.invite_group_rates).length" class="flex flex-wrap gap-1.5">
                    <span
                      v-for="rate in normalizeRateBadges(row.invite_group_rates)"
                      :key="`invite-${row.user_id}-${rate.group_id}`"
                      class="rounded-full bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300"
                    >
                      {{ formatGroupRateBadge(rate) }}
                    </span>
                  </div>
                  <span v-else class="text-gray-400">未设置</span>
                </td>
                <td class="px-4 py-3 text-right text-sm text-gray-700 dark:text-gray-300">{{ directChildrenLabel(row) }}</td>
                <td class="px-4 py-3 text-right text-sm text-gray-700 dark:text-gray-300">{{ formatRMB(readTodayRebateRMB(row), true) }}</td>
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
                    <button class="btn btn-secondary btn-sm" @click="openPricingDialog(row)">编辑倍率 / 上游</button>
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
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">允许该代理查看其全部下级的营业额与消耗统计。</p>
            </div>
            <Toggle v-model="permissionsForm.can_view_downline_daily_revenue" />
          </div>
          <div class="flex items-start justify-between gap-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">查看下级返利余额</div>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">允许该代理查看其全部下级的返利余额与归档统计。</p>
            </div>
            <Toggle v-model="permissionsForm.can_view_downline_rebate_balances" />
          </div>
          <div class="flex items-start justify-between gap-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">管理下级分组倍率</div>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">允许该代理编辑自己下级的分组倍率，不含其他分支。</p>
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

    <BaseDialog :show="showPricingDialog" title="编辑倍率与上游" width="wide" @close="closePricingDialog">
      <div class="space-y-5">
        <div v-if="selectedNode" class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900">
          <div class="flex flex-wrap items-center gap-2">
            <div class="text-sm text-gray-500">目标用户</div>
            <span
              v-if="selectedNode.is_root_admin"
              class="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
            >
              根管理员
            </span>
          </div>
          <div class="mt-1 font-semibold text-gray-900 dark:text-white">
            {{ selectedNode.username || selectedNode.email || `#${selectedNode.user_id}` }}
          </div>
          <div class="mt-1 text-sm text-gray-500">
            {{ selectedNode.email }} · {{ levelLabel(selectedNode) }} · 当前返利 {{ formatRMB(selectedNode.current_rebate_balance_rmb) }}
          </div>
        </div>

        <div v-if="pricingDialogLoading" class="py-8 text-center text-sm text-gray-500">加载倍率配置中...</div>

        <template v-else>
          <section class="space-y-3">
            <div>
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">自身成本分组倍率</h4>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                即当前用户自己的 `affiliate_distribution_user_group_rates`。
              </p>
            </div>
            <div
              v-for="row in currentPricingRows"
              :key="`current-${row.group_id}`"
              class="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px]"
            >
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-500">分组</label>
                <div class="input flex items-center bg-gray-50 text-sm text-gray-600 dark:bg-dark-900 dark:text-gray-300">
                  {{ pricingRowGroupLabel(row.group_id) }}
                </div>
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-500">倍率</label>
                <input v-model.number="row.rate_multiplier" type="number" min="0.0001" step="0.0001" class="input" placeholder="1.0000" />
              </div>
            </div>
          </section>

          <section class="space-y-3">
            <div>
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">邀请码分组倍率</h4>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                即当前用户邀请下级时发放的 `affiliate_distribution_invite_group_rates`。
              </p>
            </div>
            <div
              v-for="row in invitePricingRows"
              :key="`invite-${row.group_id}`"
              class="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px]"
            >
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-500">分组</label>
                <div class="input flex items-center bg-gray-50 text-sm text-gray-600 dark:bg-dark-900 dark:text-gray-300">
                  {{ pricingRowGroupLabel(row.group_id) }}
                </div>
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-500">倍率</label>
                <input v-model.number="row.rate_multiplier" type="number" min="0.0001" step="0.0001" class="input" placeholder="1.0000" />
              </div>
            </div>
          </section>

          <section class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div>
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">上级 / 上游渠道</h4>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                管理员可编辑当前用户的上级渠道；根管理员不允许设置上级。
              </p>
            </div>

            <div v-if="selectedNode?.is_root_admin" class="rounded-lg bg-amber-50 p-3 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
              根管理员是唯一根节点，已禁用上级编辑。
            </div>

            <template v-else>
              <div class="grid gap-3 md:grid-cols-2">
                <div>
                  <label class="mb-1 block text-xs font-medium text-gray-500">当前上级</label>
                  <div class="input flex items-center bg-gray-50 text-sm text-gray-600 dark:bg-dark-900 dark:text-gray-300">
                    {{ selectedNode ? parentLabel(selectedNode) : '-' }}
                  </div>
                </div>
                <div>
                  <label class="mb-1 block text-xs font-medium text-gray-500">目标上级 User ID</label>
                  <input
                    v-model="upstreamInviterIdInput"
                    class="input"
                    inputmode="numeric"
                    placeholder="留空则设为空上级"
                  />
                </div>
              </div>

              <div>
                <label class="mb-1 block text-xs font-medium text-gray-500">搜索用户</label>
                <input
                  v-model="upstreamSearch"
                  class="input"
                  placeholder="搜索邮箱、用户名或 ID"
                  @input="onUpstreamSearchInput"
                />
              </div>

              <div v-if="upstreamLookupLoading" class="text-sm text-gray-500">搜索中...</div>
              <div
                v-else-if="upstreamResults.length"
                class="max-h-48 space-y-2 overflow-y-auto rounded-xl border border-gray-200 p-3 dark:border-dark-700"
              >
                <button
                  v-for="user in upstreamResults"
                  :key="user.id"
                  class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left hover:bg-gray-50 dark:hover:bg-dark-900"
                  @click="selectUpstreamUser(user)"
                >
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-white">{{ user.username || user.email || `#${user.id}` }}</div>
                    <div class="text-xs text-gray-500">{{ user.email }}</div>
                  </div>
                  <span class="text-xs text-gray-500">ID {{ user.id }}</span>
                </button>
              </div>

              <div v-if="selectedUpstreamUser" class="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200">
                已选择上级：{{ selectedUpstreamUser.username || selectedUpstreamUser.email }}（ID {{ selectedUpstreamUser.id }}）
              </div>

              <div class="flex flex-wrap gap-2">
                <button class="btn btn-secondary btn-sm" @click="clearUpstreamSelection">清空选择</button>
                <span v-if="upstreamInputError" class="self-center text-xs text-red-500">{{ upstreamInputError }}</span>
              </div>
            </template>
          </section>

          <div class="rounded-lg bg-blue-50 p-3 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-200">
            自身成本倍率用于计算当前用户自己的成本；邀请码倍率用于该用户继续邀请下级时发放的倍率。
          </div>
        </template>

        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" :disabled="pricingSaving" @click="closePricingDialog">取消</button>
          <button class="btn btn-primary" :disabled="pricingSaving || pricingDialogLoading || !canSavePricingDialog" @click="savePricingDialog">
            {{ pricingSaving ? '保存中...' : '保存修改' }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onBeforeUnmount, onMounted, ref, type Component } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import {
  affiliatesAPI,
  type AffiliateAgentPermissions,
  type AffiliateDistributionTreeNode,
  type AffiliateGroupRate,
  type AffiliateGroupRateInput,
  type SimpleUser,
} from '@/api/admin/affiliates'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import groupsAPI from '@/api/admin/groups'
import type { AdminGroup } from '@/types'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDisplayDateTime } from '@/utils/format'

interface TreeDisplayNode extends AffiliateDistributionTreeNode {
  children: TreeDisplayNode[]
}

interface PricingRow {
  group_id: number
  rate_multiplier: number
}

const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(false)
const defaultPricingLoading = ref(false)
const defaultPricingSaving = ref(false)
const pricingDialogLoading = ref(false)
const pricingSaving = ref(false)
const permissionsLoading = ref(false)
const permissionsSaving = ref(false)

const search = ref('')
const levelFilter = ref('all')
const agentFilter = ref<'all' | 'agent' | 'customer'>('all')
const parentSearch = ref('')

const rows = ref<AffiliateDistributionTreeNode[]>([])
const groupOptions = ref<AdminGroup[]>([])
const defaultPricingRows = ref<PricingRow[]>([])
const currentPricingRows = ref<PricingRow[]>([])
const invitePricingRows = ref<PricingRow[]>([])

const showPricingDialog = ref(false)
const showPermissionsDialog = ref(false)

const selectedNode = ref<AffiliateDistributionTreeNode | null>(null)
const permissionTarget = ref<AffiliateDistributionTreeNode | null>(null)
const permissionsMeta = ref<AffiliateAgentPermissions | null>(null)
const permissionsForm = ref({
  can_view_downline_daily_revenue: false,
  can_view_downline_rebate_balances: false,
  can_manage_downline_pricing: false,
})

const upstreamSearch = ref('')
const upstreamResults = ref<SimpleUser[]>([])
const upstreamLookupLoading = ref(false)
const selectedUpstreamUser = ref<SimpleUser | null>(null)
const upstreamInviterIdInput = ref('')
const upstreamLookupTimer = ref<number | null>(null)

const initialDefaultPricingSignature = ref('')
const initialCurrentPricingSignature = ref('')
const initialInvitePricingSignature = ref('')
const initialUpstreamInviterId = ref<number | null>(null)

const currentUserId = computed(() => authStore.user?.id ?? null)
const enabledGroupOptions = computed(() => groupOptions.value.filter((group) => group.status !== 'inactive'))
const groupOptionMap = computed(() => new Map(enabledGroupOptions.value.map((group) => [Number(group.id), group])))

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
  if (levelFilter.value !== 'all' && String(row.depth) !== levelFilter.value) return false
  if (agentFilter.value === 'agent' && !isAgentRow(row)) return false
  if (agentFilter.value === 'customer' && isAgentRow(row)) return false

  const keyword = parentSearch.value.trim().toLowerCase()
  if (!keyword) return true

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

  return haystack.includes(keyword)
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

const defaultPricingDirty = computed(() => serializePricingRows(defaultPricingRows.value) !== initialDefaultPricingSignature.value)
const hasValidDefaultPricing = computed(() => sanitizePricingRows(defaultPricingRows.value).length > 0)
const canSaveDefaultPricing = computed(() => defaultPricingDirty.value && hasValidDefaultPricing.value)

const currentPricingDirty = computed(() => serializePricingRows(currentPricingRows.value) !== initialCurrentPricingSignature.value)
const invitePricingDirty = computed(() => serializePricingRows(invitePricingRows.value) !== initialInvitePricingSignature.value)
const parsedUpstreamInviterId = computed<number | null>(() => {
  const raw = upstreamInviterIdInput.value.trim()
  if (!raw) return null
  const value = Number(raw)
  if (!Number.isInteger(value) || value <= 0) return Number.NaN
  return value
})
const upstreamDirty = computed(() => parsedUpstreamInviterId.value !== initialUpstreamInviterId.value)
const upstreamInputError = computed(() => {
  if (selectedNode.value?.is_root_admin) return ''
  if (!upstreamInviterIdInput.value.trim()) return ''
  if (Number.isNaN(parsedUpstreamInviterId.value)) return '上级 User ID 必须是正整数'
  if (selectedNode.value && parsedUpstreamInviterId.value === selectedNode.value.user_id) return '不能把用户自己设为上级'
  return ''
})
const canSavePricingDialog = computed(() => {
  if (!selectedNode.value) return false
  if (!currentPricingDirty.value && !invitePricingDirty.value && !upstreamDirty.value) return false
  if (upstreamInputError.value) return false
  if (currentPricingDirty.value && !hasValidPricingRows(currentPricingRows.value)) return false
  if (invitePricingDirty.value && !hasValidPricingRows(invitePricingRows.value)) return false
  return true
})

const TreeNode: Component = defineComponent({
  name: 'TreeNode',
  props: {
    node: { type: Object as () => TreeDisplayNode, required: true },
  },
  emits: ['edit', 'authorize'],
  setup(props, { emit }) {
    const expanded = ref(true)

    return (): ReturnType<typeof h> => h('div', { class: 'rounded-xl border border-gray-200 p-3 dark:border-dark-700' }, [
      h('div', { class: 'flex flex-col gap-3 md:flex-row md:items-start md:justify-between' }, [
        h('div', { class: 'flex items-start gap-3' }, [
          props.node.children.length
            ? h(
              'button',
              {
                class: 'mt-0.5 rounded p-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700',
                onClick: () => {
                  expanded.value = !expanded.value
                },
              },
              expanded.value ? '－' : '+',
            )
            : h('span', { class: 'mt-0.5 w-6' }, ''),
          h('div', [
            h('div', { class: 'flex flex-wrap items-center gap-2' }, [
              h('span', { class: 'font-medium text-gray-900 dark:text-white' }, props.node.username || props.node.email || `#${props.node.user_id}`),
              h('span', { class: 'rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300' }, levelLabel(props.node)),
              props.node.is_root_admin
                ? h('span', { class: 'rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300' }, '根管理员')
                : null,
              currentUserId.value === props.node.user_id
                ? h('span', { class: 'rounded-full bg-primary-50 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/20 dark:text-primary-300' }, '你自己')
                : null,
            ]),
            h('div', { class: 'mt-1 text-xs text-gray-500' }, `邀请码：${props.node.invite_code || '-'} · 上级：${parentLabel(props.node)} · 返利余额：${formatRMB(props.node.current_rebate_balance_rmb)}`),
            h('div', { class: 'mt-1 text-xs text-gray-500' }, [
              h('span', { class: 'font-medium text-gray-700 dark:text-gray-300' }, '自身成本倍率：'),
              normalizeRateBadges(props.node.current_group_rates).map((rate) => `${rate.group_name} ${rate.rate_multiplier.toFixed(4)}x`).join('，') || '未设置',
            ]),
            h('div', { class: 'mt-1 text-xs text-gray-500' }, [
              h('span', { class: 'font-medium text-gray-700 dark:text-gray-300' }, '邀请码倍率：'),
              normalizeRateBadges(props.node.invite_group_rates).map((rate) => `${rate.group_name} ${rate.rate_multiplier.toFixed(4)}x`).join('，') || '未设置',
            ]),
          ]),
        ]),
        h('div', { class: 'flex flex-wrap justify-end gap-2' }, [
          canAuthorizeRow(props.node)
            ? h('button', { class: 'btn btn-secondary btn-sm', onClick: () => emit('authorize', props.node) }, '授权')
            : null,
          h('button', { class: 'btn btn-secondary btn-sm', onClick: () => emit('edit', props.node) }, '编辑倍率 / 上游'),
        ]),
      ]),
      expanded.value && props.node.children.length
        ? h(
          'div',
          { class: 'mt-3 space-y-3 pl-6' },
          props.node.children.map((child) => h(TreeNode, {
            node: child,
            onEdit: (node: TreeDisplayNode) => emit('edit', node),
            onAuthorize: (node: TreeDisplayNode) => emit('authorize', node),
          })),
        )
        : null,
    ])
  },
})

onMounted(() => {
  void (async () => {
    await loadGroupOptions()
    await Promise.all([loadTree(), loadDefaultPricing()])
  })()
})

onBeforeUnmount(() => {
  if (upstreamLookupTimer.value !== null) {
    window.clearTimeout(upstreamLookupTimer.value)
  }
})

async function loadGroupOptions() {
  try {
    groupOptions.value = await groupsAPI.getAll()
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '加载分组选项失败'))
  }
}

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

async function loadDefaultPricing() {
  defaultPricingLoading.value = true
  try {
    const result = await affiliatesAPI.getDefaultDistributionPricing()
    defaultPricingRows.value = buildPricingRows(result.group_rates)
    initialDefaultPricingSignature.value = serializePricingRows(defaultPricingRows.value)
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '加载默认显式倍率失败'))
  } finally {
    defaultPricingLoading.value = false
  }
}

async function saveDefaultPricing() {
  const groupRates = sanitizePricingRows(defaultPricingRows.value)
  if (groupRates.length === 0) {
    appStore.showError('请至少填写一条有效的默认显式倍率')
    return
  }

  defaultPricingSaving.value = true
  try {
    const result = await affiliatesAPI.updateDefaultDistributionPricing({ group_rates: groupRates })
    defaultPricingRows.value = buildPricingRows(result.group_rates)
    initialDefaultPricingSignature.value = serializePricingRows(defaultPricingRows.value)
    appStore.showSuccess('默认显式倍率已更新')
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '保存默认显式倍率失败'))
  } finally {
    defaultPricingSaving.value = false
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
  if (!row.inviter_id) return row.is_root_admin ? '根节点' : '-'
  const parent = resolveParent(row)
  return parent?.email || row.inviter_email || row.inviter_username || `#${row.inviter_id}`
}

function directChildrenLabel(row: AffiliateDistributionTreeNode) {
  const directCount = readDirectChildrenCount(row)
  if (directCount === null) return '—'

  const agentCountValue = toFiniteNumber(row.direct_agent_count)
  const userCountValue = toFiniteNumber(row.direct_user_count)
  if (agentCountValue !== null || userCountValue !== null) {
    return `${directCount}（代${agentCountValue ?? 0} / 普${userCountValue ?? 0}）`
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
  return depth === 0 ? '根节点 / 管理员' : `第 ${depth} 层下级`
}

function normalizeRateBadges(groupRates?: AffiliateGroupRate[]) {
  return (groupRates || [])
    .map((rate) => ({
      group_id: Number(rate.group_id),
      group_name: rate.group_name || groupNameById(Number(rate.group_id)),
      rate_multiplier: Number(rate.rate_multiplier),
    }))
    .filter((rate) => rate.group_id > 0 && Number.isFinite(rate.rate_multiplier))
}

function formatGroupRateBadge(rate: { group_id: number; group_name?: string; rate_multiplier: number }) {
  return `${rate.group_name || groupNameById(rate.group_id)} ${rate.rate_multiplier.toFixed(4)}x`
}

function groupNameById(groupID: number) {
  const group = groupOptionMap.value.get(Number(groupID))
  return group?.name || `#${groupID}`
}

function groupOptionLabel(group: AdminGroup) {
  const platform = group.platform ? ` · ${group.platform}` : ''
  const multiplier = Number.isFinite(Number(group.rate_multiplier)) ? ` · 默认 ${Number(group.rate_multiplier).toFixed(4)}x` : ''
  return `${group.name || `#${group.id}`}${platform}${multiplier}`
}

function pricingRowGroupLabel(groupID: number) {
  const group = groupOptionMap.value.get(Number(groupID))
  if (!group) return groupNameById(groupID)
  return groupOptionLabel(group)
}

function defaultRateMultiplierForGroup(group: AdminGroup) {
  const rateMultiplier = Number(group.rate_multiplier)
  return Number.isFinite(rateMultiplier) && rateMultiplier > 0 ? rateMultiplier : 1
}

function readTodayRebateRMB(row: AffiliateDistributionTreeNode) {
  return toFiniteNumber(row.today_rebate_rmb)
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

function toPricingRows(groupRates?: AffiliateGroupRate[] | AffiliateGroupRateInput[]) {
  return (groupRates || [])
    .map((rate) => ({
      group_id: Number(rate.group_id),
      rate_multiplier: Number(rate.rate_multiplier),
    }))
    .filter((rate) => Number.isInteger(rate.group_id) && rate.group_id > 0 && Number.isFinite(rate.rate_multiplier) && rate.rate_multiplier > 0)
}

function buildPricingRows(groupRates?: AffiliateGroupRate[] | AffiliateGroupRateInput[]) {
  const normalizedRows = toPricingRows(groupRates)
  if (!enabledGroupOptions.value.length) return normalizedRows

  const rateByGroupID = new Map<number, number>(
    normalizedRows.map((row) => [row.group_id, row.rate_multiplier]),
  )

  return enabledGroupOptions.value.map((group) => ({
    group_id: Number(group.id),
    rate_multiplier: rateByGroupID.get(Number(group.id)) ?? defaultRateMultiplierForGroup(group),
  }))
}

function sanitizePricingRows(rowsInput: PricingRow[]): AffiliateGroupRateInput[] {
  const enabledGroupIDs = new Set(enabledGroupOptions.value.map((group) => Number(group.id)))
  const shouldFilterByEnabledGroup = enabledGroupIDs.size > 0

  return rowsInput
    .map((row) => ({
      group_id: Number(row.group_id),
      rate_multiplier: Number(row.rate_multiplier),
    }))
    .filter((row) => (
      Number.isInteger(row.group_id)
      && row.group_id > 0
      && Number.isFinite(row.rate_multiplier)
      && row.rate_multiplier > 0
      && (!shouldFilterByEnabledGroup || enabledGroupIDs.has(row.group_id))
    ))
}

function hasValidPricingRows(rowsInput: PricingRow[]) {
  return sanitizePricingRows(rowsInput).length > 0
}

function serializePricingRows(rowsInput: PricingRow[]) {
  return JSON.stringify(
    sanitizePricingRows(rowsInput)
      .slice()
      .sort((a, b) => a.group_id - b.group_id || a.rate_multiplier - b.rate_multiplier),
  )
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

async function openPricingDialog(row: AffiliateDistributionTreeNode) {
  selectedNode.value = row
  showPricingDialog.value = true
  pricingDialogLoading.value = true

  currentPricingRows.value = buildPricingRows(row.current_group_rates)
  invitePricingRows.value = buildPricingRows(row.invite_group_rates)

  upstreamSearch.value = ''
  upstreamResults.value = []
  selectedUpstreamUser.value = null
  upstreamInviterIdInput.value = row.inviter_id ? String(row.inviter_id) : ''
  initialUpstreamInviterId.value = row.inviter_id ?? null

  try {
    const [currentResult, inviteResult] = await Promise.allSettled([
      affiliatesAPI.getUserDistributionPricing(row.user_id),
      affiliatesAPI.getUserInvitePricing(row.user_id),
    ])

    if (currentResult.status === 'fulfilled') {
      currentPricingRows.value = buildPricingRows(currentResult.value.group_rates)
    }

    if (inviteResult.status === 'fulfilled') {
      invitePricingRows.value = buildPricingRows(inviteResult.value.group_rates)
    }

    if (currentResult.status === 'rejected' || inviteResult.status === 'rejected') {
      appStore.showError('部分倍率数据加载失败，已展示当前页面可用数据')
    }
  } finally {
    initialCurrentPricingSignature.value = serializePricingRows(currentPricingRows.value)
    initialInvitePricingSignature.value = serializePricingRows(invitePricingRows.value)
    pricingDialogLoading.value = false
  }
}

function closePricingDialog() {
  if (pricingSaving.value) return
  showPricingDialog.value = false
}

function onUpstreamSearchInput() {
  const keyword = upstreamSearch.value.trim()
  if (!keyword) {
    upstreamResults.value = []
    upstreamLookupLoading.value = false
    return
  }

  if (upstreamLookupTimer.value !== null) {
    window.clearTimeout(upstreamLookupTimer.value)
  }

  upstreamLookupTimer.value = window.setTimeout(async () => {
    upstreamLookupLoading.value = true
    try {
      upstreamResults.value = await affiliatesAPI.lookupUsers(keyword)
    } catch (error) {
      appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '搜索上级用户失败'))
    } finally {
      upstreamLookupLoading.value = false
    }
  }, 250)
}

function selectUpstreamUser(user: SimpleUser) {
  selectedUpstreamUser.value = user
  upstreamInviterIdInput.value = String(user.id)
  upstreamSearch.value = ''
  upstreamResults.value = []
}

function clearUpstreamSelection() {
  selectedUpstreamUser.value = null
  upstreamSearch.value = ''
  upstreamResults.value = []
  upstreamInviterIdInput.value = ''
}

async function savePricingDialog() {
  if (!selectedNode.value) return
  if (upstreamInputError.value) {
    appStore.showError(upstreamInputError.value)
    return
  }

  const currentRates = sanitizePricingRows(currentPricingRows.value)
  const inviteRates = sanitizePricingRows(invitePricingRows.value)

  if (currentPricingDirty.value && currentRates.length === 0) {
    appStore.showError('请至少填写一条有效的自身成本倍率')
    return
  }
  if (invitePricingDirty.value && inviteRates.length === 0) {
    appStore.showError('请至少填写一条有效的邀请码倍率')
    return
  }

  pricingSaving.value = true
  try {
    let savedUpstreamInviterId = parsedUpstreamInviterId.value

    if (currentPricingDirty.value) {
      const result = await affiliatesAPI.updateUserDistributionPricing(selectedNode.value.user_id, { group_rates: currentRates })
      currentPricingRows.value = buildPricingRows(result.group_rates)
    }

    if (invitePricingDirty.value) {
      const result = await affiliatesAPI.updateUserInvitePricing(selectedNode.value.user_id, { group_rates: inviteRates })
      invitePricingRows.value = buildPricingRows(result.group_rates)
    }

    if (upstreamDirty.value && !selectedNode.value.is_root_admin) {
      const result = await affiliatesAPI.updateUserUpstream(selectedNode.value.user_id, {
        inviter_id: parsedUpstreamInviterId.value,
      })
      savedUpstreamInviterId = result.upstream_user_id ?? result.inviter_id ?? null
      upstreamInviterIdInput.value = savedUpstreamInviterId ? String(savedUpstreamInviterId) : ''
    }

    await loadTree()
    const freshNode = rows.value.find((row) => row.user_id === selectedNode.value?.user_id) ?? selectedNode.value
    selectedNode.value = freshNode
    initialCurrentPricingSignature.value = serializePricingRows(currentPricingRows.value)
    initialInvitePricingSignature.value = serializePricingRows(invitePricingRows.value)
    initialUpstreamInviterId.value = savedUpstreamInviterId
    appStore.showSuccess('代理倍率与上游配置已更新')
    showPricingDialog.value = false
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, (value: string) => value, 'admin.affiliates.errors', '保存倍率或上游配置失败'))
  } finally {
    pricingSaving.value = false
  }
}

function formatRMB(value: number | null | undefined, allowEmpty = false) {
  const amount = toFiniteNumber(value)
  if (amount === null) return allowEmpty ? '—' : '¥0.00'
  return `¥${amount.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}
</script>
