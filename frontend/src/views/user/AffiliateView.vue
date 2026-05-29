<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="detail">
        <div class="card p-6">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ tt('affiliate.title', '代理分销') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                {{ tt('affiliate.description', '查看我的邀请码、分组倍率、直属下级和营业数据。') }}
              </p>
            </div>
            <div class="rounded-2xl bg-primary-50 px-4 py-3 text-sm text-primary-700 dark:bg-primary-900/20 dark:text-primary-200">
              {{ tt('affiliate.noSelfAdjust', '返利额度仅管理员可调整，普通代理无法修改自己的额度。') }}
            </div>
          </div>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ tt('affiliate.yourCode', '我的邀请码') }}</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ tt('affiliate.copyCode', '复制邀请码') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ tt('affiliate.inviteLink', '邀请链接') }}</p>
              <div class="space-y-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <button
                  class="btn btn-secondary btn-sm"
                  data-testid="generate-invite-link"
                  @click="generateInviteLink"
                >
                  <Icon name="link" size="sm" />
                  <span>{{ tt('affiliate.generateInviteLink', '生成邀请链接') }}</span>
                </button>
                <code
                  v-if="generatedInviteLink"
                  class="block truncate text-sm text-gray-700 dark:text-gray-300"
                  data-testid="generated-invite-link"
                >
                  {{ generatedInviteLink }}
                </code>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="chartBar" size="sm" class="text-primary-500" />
              {{ tt('affiliate.stats.todayRevenue', '今日营业额') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.today_revenue_usd) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="sparkles" size="sm" class="text-emerald-500" />
              {{ tt('affiliate.stats.todayRebate', '今日返利') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.today_rebate_rmb, 'CNY') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="creditCard" size="sm" class="text-amber-500" />
              {{ tt('affiliate.stats.currentBalance', '当前可结算返利额度') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-amber-600 dark:text-amber-400">
              {{ formatCurrency(detail.current_rebate_balance_rmb, 'CNY') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="users" size="sm" class="text-primary-500" />
              {{ tt('affiliate.stats.directChildren', '直属下级') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.direct_children_count) }}
            </p>
          </div>
        </div>

        <AffiliateGroupRateEditor
          :title="tt('affiliate.myRates.title', '我的当前分组倍率')"
          :description="tt('affiliate.myRates.description', '当前账号对外结算时使用的分组倍率，仅展示不可自行修改。')"
          :group-rates="detail.my_group_rates"
          :group-options="groupOptions"
          :editable="false"
          :empty-text="tt('affiliate.myRates.empty', '暂无当前分组倍率数据。')"
          :add-label="tt('affiliate.rateConfig.add', '添加分组倍率')"
          :save-label="tt('affiliate.rateConfig.save', '保存邀请码分组倍率')"
          :saving-label="tt('affiliate.rateConfig.saving', '保存中...')"
          :group-label="tt('affiliate.rateConfig.group', '分组')"
          :multiplier-label="tt('affiliate.rateConfig.multiplier', '分组倍率')"
          :remove-label="tt('affiliate.rateConfig.remove', '删除')"
          :group-placeholder="tt('affiliate.rateConfig.placeholder', '请选择分组')"
          :duplicate-group-text="tt('affiliate.rateConfig.duplicate', '同一个分组只能配置一次。')"
        />

        <AffiliateGroupRateEditor
          :title="tt('affiliate.rateConfig.title', '邀请码分组倍率')"
          :description="tt('affiliate.rateConfig.description', '新用户填写你的邀请码后，会继承这里配置的分组倍率。')"
          :group-rates="detail.invite_group_rates"
          :group-options="groupOptions"
          :saving="savingInviteCodeRates"
          :empty-text="tt('affiliate.rateConfig.empty', '暂无分组倍率，点击添加后保存。')"
          :add-label="tt('affiliate.rateConfig.add', '添加分组倍率')"
          :save-label="tt('affiliate.rateConfig.save', '保存邀请码分组倍率')"
          :saving-label="tt('affiliate.rateConfig.saving', '保存中...')"
          :group-label="tt('affiliate.rateConfig.group', '分组')"
          :multiplier-label="tt('affiliate.rateConfig.multiplier', '邀请码分组倍率')"
          :remove-label="tt('affiliate.rateConfig.remove', '删除')"
          :group-placeholder="tt('affiliate.rateConfig.placeholder', '请选择分组')"
          :duplicate-group-text="tt('affiliate.rateConfig.duplicate', '同一个分组只能配置一次。')"
          @save="saveInviteCodeRates"
        />

        <AffiliateDirectChildrenTable
          :children="detail.direct_children"
          :group-options="groupOptions"
          :saving-user-id="savingChildId"
          :title="tt('affiliate.children.title', '直属下级列表')"
          :description="tt('affiliate.children.description', '你可以调整直属下级的分组成本倍率，但不能调整自己的返利额度。')"
          :count-label="tt('affiliate.children.count', '共 {count} 人', { count: detail.direct_children_count })"
          :empty-text="directChildrenEmptyText"
          :joined-at-label="tt('affiliate.children.joinedAt', '加入时间')"
          :revenue-label="tt('affiliate.children.revenue', '今日营业额')"
          :rebate-label="tt('affiliate.children.rebate', '今日返利')"
          :balance-label="tt('affiliate.children.balance', '当前返利额度')"
          :expand-label="tt('affiliate.children.expand', '编辑分组成本倍率')"
          :collapse-label="tt('affiliate.children.collapse', '收起分组成本倍率')"
          :child-rate-title="tt('affiliate.children.rateTitle', '直属下级分组成本倍率')"
          :child-rate-description="tt('affiliate.children.rateDescription', '修改后该下级用户的分组计费倍率立即生效。')"
          :empty-rates-text="tt('affiliate.children.rateEmpty', '暂无分组成本倍率，点击添加后保存。')"
          :add-rate-label="tt('affiliate.children.addRate', '添加分组成本倍率')"
          :save-rate-label="tt('affiliate.children.saveRate', '保存下级分组成本倍率')"
          :saving-rate-label="tt('affiliate.children.savingRate', '保存中...')"
          :group-label="tt('affiliate.children.group', '分组')"
          :multiplier-label="tt('affiliate.children.multiplier', '分组成本倍率')"
          :remove-label="tt('affiliate.children.remove', '删除')"
          :group-placeholder="tt('affiliate.children.placeholder', '请选择分组')"
          :duplicate-group-text="tt('affiliate.children.duplicate', '同一个分组只能配置一次。')"
          :agent-role-label="tt('affiliate.children.agent', '代理')"
          :user-role-label="tt('affiliate.children.user', '普通用户')"
          @save-child-rates="saveChildRates"
        />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import userGroupsAPI from '@/api/groups'
import {
  getAffiliateOverview,
  updateAffiliateDirectChildGroupRates,
  updateInviteCodeGroupRates,
  type AffiliateGroupRate,
  type UserAffiliateDetail,
} from '@/api/user'
import type { Group } from '@/types'
import type { AffiliateGroupOption } from '@/components/affiliate/types'
import AffiliateDirectChildrenTable from '@/components/affiliate/AffiliateDirectChildrenTable.vue'
import AffiliateGroupRateEditor from '@/components/affiliate/AffiliateGroupRateEditor.vue'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const savingInviteCodeRates = ref(false)
const savingChildId = ref<number | null>(null)
const detail = ref<UserAffiliateDetail | null>(null)
const groupOptions = ref<AffiliateGroupOption[]>([])
const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})
const generatedInviteLink = ref('')

const directChildrenEmptyText = computed(() => tt('affiliate.children.empty', '暂无直属下级'))

function tt(key: string, fallback: string, params?: Record<string, unknown>): string {
  const translated = t(key, params ?? {})
  if (translated !== key) return translated
  return fallback.replace(/\{(\w+)\}/g, (_, name: string) => String(params?.[name] ?? `{${name}}`))
}

function formatCount(value: number): string {
  return value.toLocaleString()
}

function toFiniteNumber(value: unknown, fallback = 0): number {
  const amount = Number(value)
  return Number.isFinite(amount) ? amount : fallback
}

function normalizeGroupOptions(groups: Group[]): AffiliateGroupOption[] {
  return groups
    .map((group) => ({
      id: Number(group.id),
      name: group.name || `#${group.id}`,
      platform: group.platform,
      rate_multiplier: toFiniteNumber(group.rate_multiplier, 1),
    }))
    .filter((group) => group.id > 0)
}

async function loadGroupOptions(): Promise<void> {
  try {
    groupOptions.value = normalizeGroupOptions(await userGroupsAPI.getAvailable())
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, tt('affiliate.groupsLoadFailed', '加载可用分组失败')))
  }
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }

  try {
    detail.value = await getAffiliateOverview()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, tt('affiliate.loadFailed', '加载代理分销数据失败')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, tt('affiliate.codeCopied', '邀请码已复制'))
}

async function generateInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  generatedInviteLink.value = inviteLink.value
  await copyToClipboard(inviteLink.value, tt('affiliate.linkCopied', '邀请链接已复制'))
}

async function saveInviteCodeRates(rates: AffiliateGroupRate[]): Promise<void> {
  if (!detail.value || savingInviteCodeRates.value) return
  savingInviteCodeRates.value = true

  try {
    const result = await updateInviteCodeGroupRates({
      group_rates: rates.map((rate) => ({
        group_id: rate.group_id,
        rate_multiplier: rate.rate_multiplier,
      })),
    })
    detail.value = {
      ...detail.value,
      invite_group_rates: result.group_rates,
    }
    appStore.showSuccess(tt('affiliate.rateConfig.saved', '邀请码分组倍率已保存'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, tt('affiliate.rateConfig.saveFailed', '保存邀请码分组倍率失败')))
  } finally {
    savingInviteCodeRates.value = false
  }
}

async function saveChildRates(userId: number, rates: AffiliateGroupRate[]): Promise<void> {
  if (!detail.value || savingChildId.value !== null) return
  savingChildId.value = userId

  try {
    const result = await updateAffiliateDirectChildGroupRates(userId, {
      group_rates: rates.map((rate) => ({
        group_id: rate.group_id,
        rate_multiplier: rate.rate_multiplier,
      })),
    })
    detail.value = {
      ...detail.value,
      direct_children: detail.value.direct_children.map((child) => (
        child.user_id === userId
          ? { ...child, group_rates: result.group_rates }
          : child
      )),
    }
    appStore.showSuccess(tt('affiliate.children.saved', '直属下级分组成本倍率已保存'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, tt('affiliate.children.saveFailed', '保存直属下级分组成本倍率失败')))
  } finally {
    savingChildId.value = null
  }
}

onMounted(() => {
  void Promise.all([loadGroupOptions(), loadAffiliateDetail()])
})
</script>
