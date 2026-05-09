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
                {{ tt('affiliate.description', '查看我的邀请码、模型倍率、直属下级和营业数据。') }}
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
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm text-gray-700 dark:text-gray-300">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ tt('affiliate.copyLink', '复制链接') }}</span>
                </button>
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
              {{ formatCount(detail.direct_children.length) }}
            </p>
          </div>
        </div>

        <AffiliateModelRateEditor
          :title="tt('affiliate.myRates.title', '我的当前模型倍率')"
          :description="tt('affiliate.myRates.description', '当前账号对外结算时使用的模型倍率，仅展示不可自行修改。')"
          :model-rates="detail.my_model_rates"
          :editable="false"
          :empty-text="tt('affiliate.myRates.empty', '暂未返回当前模型倍率数据。')"
          :add-label="tt('affiliate.rateConfig.add', '添加模型倍率')"
          :save-label="tt('affiliate.rateConfig.save', '保存邀请码倍率')"
          :saving-label="tt('affiliate.rateConfig.saving', '保存中...')"
          :model-label="tt('affiliate.rateConfig.model', '模型')"
          :multiplier-label="tt('affiliate.rateConfig.multiplier', '倍率')"
          :remove-label="tt('affiliate.rateConfig.remove', '删除')"
          :model-placeholder="tt('affiliate.rateConfig.placeholder', '例如 gpt-4.1')"
        />

        <AffiliateModelRateEditor
          :title="tt('affiliate.rateConfig.title', '邀请码模型倍率配置')"
          :description="tt('affiliate.rateConfig.description', '新用户填写你的邀请码后，会继承这里配置的模型倍率。')"
          :model-rates="detail.invite_code_model_rates"
          :saving="savingInviteCodeRates"
          :empty-text="tt('affiliate.rateConfig.empty', '暂无模型倍率，点击添加后保存。')"
          :add-label="tt('affiliate.rateConfig.add', '添加模型倍率')"
          :save-label="tt('affiliate.rateConfig.save', '保存邀请码倍率')"
          :saving-label="tt('affiliate.rateConfig.saving', '保存中...')"
          :model-label="tt('affiliate.rateConfig.model', '模型')"
          :multiplier-label="tt('affiliate.rateConfig.multiplier', '倍率')"
          :remove-label="tt('affiliate.rateConfig.remove', '删除')"
          :model-placeholder="tt('affiliate.rateConfig.placeholder', '例如 gpt-4.1')"
          @save="saveInviteCodeRates"
        />

        <AffiliateDirectChildrenTable
          :children="detail.direct_children"
          :saving-user-id="savingChildId"
          :title="tt('affiliate.children.title', '直属下级列表')"
          :description="tt('affiliate.children.description', '你可以调整直属下级的模型倍率，但不能调整自己的返利额度。')"
          :count-label="`共 ${detail.direct_children.length} 人`"
          :empty-text="tt('affiliate.children.empty', '暂无直属下级')"
          :joined-at-label="tt('affiliate.children.joinedAt', '加入时间')"
          :revenue-label="tt('affiliate.children.revenue', '今日营业额')"
          :rebate-label="tt('affiliate.children.rebate', '今日返利')"
          :balance-label="tt('affiliate.children.balance', '当前返利额度')"
          :expand-label="tt('affiliate.children.expand', '编辑模型倍率')"
          :collapse-label="tt('affiliate.children.collapse', '收起倍率编辑')"
          :child-rate-title="tt('affiliate.children.rateTitle', '直属下级模型倍率')"
          :child-rate-description="tt('affiliate.children.rateDescription', '修改后该下级用户的模型计费倍率即时生效。')"
          :empty-rates-text="tt('affiliate.children.rateEmpty', '暂无模型倍率，点击添加后保存。')"
          :add-rate-label="tt('affiliate.children.addRate', '添加模型倍率')"
          :save-rate-label="tt('affiliate.children.saveRate', '保存下级倍率')"
          :saving-rate-label="tt('affiliate.children.savingRate', '保存中...')"
          :model-label="tt('affiliate.children.model', '模型')"
          :multiplier-label="tt('affiliate.children.multiplier', '倍率')"
          :remove-label="tt('affiliate.children.remove', '删除')"
          :model-placeholder="tt('affiliate.children.placeholder', '例如 claude-3.7-sonnet')"
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
import type { AxiosError } from 'axios'
import { apiClient } from '@/api/client'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import type {
  AffiliateDirectChildResponse,
  AffiliateDistributionDetail,
  AffiliateDistributionDetailResponse,
  AffiliateModelRate,
  AffiliateRawModelRate
} from '@/components/affiliate/types'
import AffiliateDirectChildrenTable from '@/components/affiliate/AffiliateDirectChildrenTable.vue'
import AffiliateModelRateEditor from '@/components/affiliate/AffiliateModelRateEditor.vue'
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
const detail = ref<AffiliateDistributionDetail | null>(null)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

function tt(key: string, fallback: string, params?: Record<string, unknown>): string {
  const translated = t(key, params ?? {})
  return translated === key ? fallback : translated
}

function formatCount(value: number): string {
  return value.toLocaleString()
}

function isRetryableMissingEndpoint(error: unknown): boolean {
  const status = (error as AxiosError | undefined)?.response?.status
  return status === 404 || status === 405
}

function toFiniteNumber(value: unknown, fallback = 0): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function normalizeModelRates(rates?: AffiliateRawModelRate[] | null): AffiliateModelRate[] {
  return (rates ?? [])
    .map((rate) => ({
      model: (typeof rate.model === 'string' && rate.model.trim()) || (typeof rate.model_name === 'string' && rate.model_name.trim()) || '',
      multiplier: toFiniteNumber(rate.multiplier, 0)
    }))
    .filter((rate) => rate.model.length > 0)
}

function normalizeDirectChild(child: AffiliateDirectChildResponse) {
  return {
    user_id: child.user_id,
    email: child.email ?? '',
    username: child.username ?? '',
    role: child.role ?? (child.is_agent ? 'agent' : 'user'),
    joined_at: child.joined_at ?? child.created_at ?? null,
    today_revenue_usd: toFiniteNumber(child.today_revenue_usd ?? child.today_business_usd, 0),
    today_rebate_rmb: toFiniteNumber(child.today_rebate_rmb, 0),
    current_rebate_balance_rmb: toFiniteNumber(child.current_rebate_balance_rmb, 0),
    model_rates: normalizeModelRates(child.model_rates ?? child.current_model_rates)
  }
}

function normalizeDistributionDetail(raw: AffiliateDistributionDetailResponse): AffiliateDistributionDetail {
  const directChildren = (raw.direct_children ?? []).map(normalizeDirectChild)
  const directChildrenCount = toFiniteNumber(raw.direct_children_count ?? raw.direct_member_count, directChildren.length)

  return {
    user_id: raw.user_id,
    aff_code: raw.aff_code ?? raw.invite_code ?? '',
    inviter_id: raw.inviter_id ?? null,
    invite_code_model_rates: normalizeModelRates(raw.invite_code_model_rates ?? raw.invite_model_rates),
    my_model_rates: normalizeModelRates(raw.my_model_rates),
    today_revenue_usd: toFiniteNumber(raw.today_revenue_usd ?? raw.today_business_usd, 0),
    today_rebate_rmb: toFiniteNumber(raw.today_rebate_rmb, 0),
    current_rebate_balance_rmb: toFiniteNumber(raw.current_rebate_balance_rmb, 0),
    direct_children: directChildren,
    direct_children_count: directChildrenCount
  }
}

async function requestFirstAvailable<T>(requests: Array<() => Promise<T>>): Promise<T> {
  let lastError: unknown
  for (const request of requests) {
    try {
      return await request()
    } catch (error) {
      lastError = error
      if (!isRetryableMissingEndpoint(error)) {
        throw error
      }
    }
  }
  throw lastError
}

function normalizeLegacyAffiliateDetail(legacy: UserAffiliateDetail): AffiliateDistributionDetail {
  return {
    user_id: legacy.user_id,
    aff_code: legacy.aff_code,
    inviter_id: legacy.inviter_id ?? null,
    invite_code_model_rates: [],
    my_model_rates: [],
    today_revenue_usd: 0,
    today_rebate_rmb: 0,
    current_rebate_balance_rmb: legacy.aff_quota ?? 0,
    direct_children: (legacy.invitees ?? []).map((invitee) => ({
      user_id: invitee.user_id,
      email: invitee.email,
      username: invitee.username,
      role: 'user',
      joined_at: invitee.created_at ?? null,
      today_revenue_usd: 0,
      today_rebate_rmb: invitee.total_rebate ?? 0,
      current_rebate_balance_rmb: invitee.total_rebate ?? 0,
      model_rates: []
    })),
    direct_children_count: legacy.invitees?.length ?? 0
  }
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await requestFirstAvailable<AffiliateDistributionDetail>([
      async () => {
        const { data } = await apiClient.get<AffiliateDistributionDetailResponse>('/user/aff/distribution')
        return normalizeDistributionDetail(data)
      },
      async () => {
        const { data } = await apiClient.get<AffiliateDistributionDetailResponse>('/user/aff/agent')
        return normalizeDistributionDetail(data)
      },
      async () => normalizeLegacyAffiliateDetail((await userAPI.getAffiliateDetail()) as unknown as UserAffiliateDetail)
    ])
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

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, tt('affiliate.linkCopied', '邀请链接已复制'))
}

async function saveInviteCodeRates(rates: AffiliateModelRate[]): Promise<void> {
  if (!detail.value || savingInviteCodeRates.value) return
  savingInviteCodeRates.value = true
  try {
    const payload = { model_rates: rates }
    await requestFirstAvailable([
      () => apiClient.put('/user/aff/invite-code-model-rates', payload),
      () => apiClient.put('/user/aff/invite-code-pricing', payload),
      () => apiClient.put('/user/aff/model-rates', payload)
    ])
    appStore.showSuccess(tt('affiliate.rateConfig.saved', '邀请码模型倍率已保存'))
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, tt('affiliate.rateConfig.saveFailed', '保存邀请码模型倍率失败')))
  } finally {
    savingInviteCodeRates.value = false
  }
}

async function saveChildRates(userId: number, rates: AffiliateModelRate[]): Promise<void> {
  if (!detail.value || savingChildId.value !== null) return
  savingChildId.value = userId
  try {
    const payload = { model_rates: rates }
    await requestFirstAvailable([
      () => apiClient.put(`/user/aff/downlines/${userId}/model-rates`, payload),
      () => apiClient.put(`/user/aff/direct-invitees/${userId}/model-rates`, payload),
      () => apiClient.put(`/user/aff/children/${userId}/model-rates`, payload)
    ])
    appStore.showSuccess(tt('affiliate.children.saved', '直属下级模型倍率已保存'))
    await loadAffiliateDetail(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, tt('affiliate.children.saveFailed', '保存直属下级模型倍率失败')))
  } finally {
    savingChildId.value = null
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
