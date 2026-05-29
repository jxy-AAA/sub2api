<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="grid flex-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            <SearchInput
              v-model="searchQuery"
              :placeholder="t('modelMarket.searchPlaceholder')"
              :debounce-ms="150"
            />

            <Select
              v-model="selectedProtocol"
              :options="protocolOptions"
              :placeholder="t('modelMarket.filters.protocol')"
            />

            <Select
              v-model="selectedProvider"
              :options="providerOptions"
              :placeholder="t('modelMarket.filters.provider')"
            />

            <Select
              v-model="selectedCapability"
              :options="capabilityOptions"
              :placeholder="t('modelMarket.filters.capability')"
            />
          </div>

          <div class="flex flex-wrap items-center justify-end gap-3">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('modelMarket.count', { filtered: filteredModels.length, total: models.length }) }}
            </span>
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadModels"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div class="flex h-full flex-col">
          <div
            v-if="loadError && models.length"
            class="border-b border-red-200 bg-red-50 px-6 py-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-300"
          >
            {{ loadError }}
          </div>

          <div v-if="loading && !models.length" class="flex h-full items-center justify-center p-10">
            <div class="flex flex-col items-center gap-3 text-sm text-gray-500 dark:text-dark-400">
              <LoadingSpinner size="lg" />
              <span>{{ t('common.loading') }}</span>
            </div>
          </div>

          <div
            v-else-if="loadError && !models.length"
            class="flex h-full items-center justify-center p-8"
          >
            <EmptyState
              :title="t('modelMarket.errorTitle')"
              :description="loadError"
              :action-text="t('common.refresh')"
              @action="loadModels"
            />
          </div>

          <div
            v-else-if="!filteredModels.length"
            class="flex h-full items-center justify-center p-8"
          >
            <EmptyState
              :title="searchActive ? t('modelMarket.emptyFilteredTitle') : t('modelMarket.emptyTitle')"
              :description="
                searchActive
                  ? t('modelMarket.emptyFilteredDescription')
                  : t('modelMarket.emptyDescription')
              "
              :action-text="searchActive ? t('common.reset') : t('common.refresh')"
              @action="handleEmptyAction"
            />
          </div>

          <div v-else class="h-full overflow-y-auto p-6">
            <div class="mb-4 flex flex-wrap gap-2">
              <span class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                {{ t('modelMarket.fields.channels') }}: {{ totalVisibleChannels }}
              </span>
              <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                {{ t('modelMarket.filters.provider') }}: {{ providerOptions.length - 1 }}
              </span>
              <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                {{ t('modelMarket.filters.protocol') }}: {{ protocolOptions.length - 1 }}
              </span>
            </div>

            <div class="grid gap-4 xl:grid-cols-2">
              <article
                v-for="model in filteredModels"
                :key="String(model.id ?? model.model_id)"
                class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md dark:border-dark-700 dark:bg-dark-800/80"
              >
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0">
                    <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
                      {{ model.display_name || model.model_id }}
                    </h2>
                    <p class="mt-1 break-all text-sm text-gray-500 dark:text-dark-400">
                      {{ model.model_id }}
                    </p>
                  </div>

                  <div class="flex flex-wrap justify-end gap-2">
                    <span :class="protocolBadgeClass(model.protocol)" class="model-market-badge">
                      <PlatformIcon :platform="resolveProtocolBadgeKey(model.protocol) || model.protocol" size="xs" />
                      {{ formatProtocolLabel(model.protocol) }}
                    </span>
                    <span v-if="model.recommended" class="model-market-badge model-market-badge-recommended">
                      {{ t('modelMarket.badges.recommended') }}
                    </span>
                    <span :class="statusBadgeClass(model)" class="model-market-badge">
                      {{ formatStatusLabel(model) }}
                    </span>
                  </div>
                </div>

                <p
                  v-if="model.description"
                  class="mt-3 text-sm leading-6 text-gray-600 dark:text-dark-300"
                >
                  {{ model.description }}
                </p>

                <dl class="mt-4 grid gap-4 text-sm md:grid-cols-2">
                  <div>
                    <dt class="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                      {{ t('modelMarket.fields.provider') }}
                    </dt>
                    <dd class="mt-1 flex flex-wrap gap-2">
                      <span
                        v-if="providerSourceKey(model)"
                        :class="['model-market-badge', providerBadgeClass(providerSourceKey(model), model.protocol)]"
                      >
                        <PlatformIcon :platform="resolveBadgeTone(providerSourceKey(model), model.protocol) || providerSourceKey(model)" size="xs" />
                        {{ formatProviderLabel(providerSourceKey(model)) }}
                      </span>
                      <span
                        v-if="model.platform && normalizeKey(model.platform) !== normalizeKey(providerSourceKey(model))"
                        :class="['model-market-badge', providerBadgeClass(model.platform, model.protocol)]"
                      >
                        <PlatformIcon :platform="resolveBadgeTone(model.platform, model.protocol) || model.platform" size="xs" />
                        {{ formatProviderLabel(model.platform) }}
                      </span>
                      <span v-if="!providerSourceKey(model) && !model.platform" class="text-gray-500 dark:text-dark-400">
                        {{ t('modelMarket.unknownProvider') }}
                      </span>
                    </dd>
                  </div>

                  <div>
                    <dt class="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                      {{ t('modelMarket.fields.contextWindow') }}
                    </dt>
                    <dd class="mt-1 text-gray-800 dark:text-dark-100">
                      {{ formatContextWindow(model.context_window) }}
                    </dd>
                  </div>
                </dl>

                <section class="mt-4 rounded-2xl bg-gray-50 p-4 dark:bg-dark-900/60">
                  <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                    {{ t('modelMarket.sections.pricing') }}
                  </div>

                  <div v-if="pricingBadges(model).length" class="flex flex-wrap gap-2">
                    <span
                      v-for="item in pricingBadges(model)"
                      :key="`${model.model_id}-${item.label}-${item.value}`"
                      class="rounded-full bg-white px-3 py-1 text-xs font-medium text-gray-700 shadow-sm dark:bg-dark-800 dark:text-dark-200"
                    >
                      {{ item.label }} {{ item.value }}
                    </span>
                  </div>

                  <p v-else class="text-sm text-gray-500 dark:text-dark-400">
                    {{ t('modelMarket.noPricing') }}
                  </p>
                </section>

                <section class="mt-4">
                  <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                    {{ t('modelMarket.sections.capabilities') }}
                  </div>

                  <div v-if="capabilityChips(model).length" class="flex flex-wrap gap-2">
                    <span
                      v-for="capability in capabilityChips(model)"
                      :key="`${model.model_id}-cap-${capability}`"
                      class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                    >
                      {{ capability }}
                    </span>
                  </div>

                  <p v-else class="text-sm text-gray-500 dark:text-dark-400">
                    {{ t('modelMarket.noCapabilities') }}
                  </p>
                </section>

                <section class="mt-4" v-if="model.tags.length">
                  <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                    {{ t('modelMarket.sections.tags') }}
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <span
                      v-for="tag in model.tags"
                      :key="`${model.model_id}-tag-${tag}`"
                      class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200"
                    >
                      {{ tag }}
                    </span>
                  </div>
                </section>

                <section class="mt-4">
                  <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                    {{ t('modelMarket.sections.channels') }}
                  </div>

                  <div v-if="model.channels.length" class="space-y-2">
                    <div
                      v-for="channel in model.channels"
                      :key="`${model.model_id}-channel-${channel.name}`"
                      class="rounded-xl border border-gray-200 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800/60"
                    >
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ channel.name }}</span>
                        <span
                          v-if="channelSourceKey(channel)"
                          :class="['model-market-badge', providerBadgeClass(channelSourceKey(channel), channel.platform || model.protocol)]"
                        >
                          <PlatformIcon :platform="resolveBadgeTone(channelSourceKey(channel), channel.platform || model.protocol) || channelSourceKey(channel)" size="xs" />
                          {{ formatChannelSource(channel) }}
                        </span>
                      </div>
                      <span
                        v-if="false"
                        class="text-gray-400 dark:text-dark-500"
                      >
                        · {{ formatChannelSource(channel) }}
                      </span>
                      <div v-if="channel.groups.length" class="mt-2 flex flex-wrap gap-1.5">
                        <span
                          v-for="group in channel.groups"
                          :key="`${model.model_id}-${channel.name}-group-${group.id}`"
                          :class="['rounded-full border px-2 py-0.5 text-[11px] font-medium', groupBadgeClass(group)]"
                        >
                          {{ group.name }}
                        </span>
                      </div>
                    </div>
                  </div>

                  <p v-else class="text-sm text-gray-500 dark:text-dark-400">
                    {{ t('modelMarket.noChannels') }}
                  </p>
                </section>
              </article>
            </div>
          </div>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import userModelMarketAPI, {
  type UserAvailableGroup,
  type UserModelMarketChannel,
  type UserModelMarketModel,
  type UserModelMarketPricing,
} from '@/api/modelMarket'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatNumberLocaleString } from '@/utils/format'
import { platformBadgeLightClass } from '@/utils/platformColors'
import { formatScaled } from '@/utils/pricing'
import {
  getPlatformDisplayName,
  getProtocolDisplayName,
  isCompatiblePlatform,
  normalizePlatformKey,
  resolvePlatformBadgeKey,
  resolveProtocolBadgeKey,
} from '@/utils/platforms'

const { t } = useI18n()
const appStore = useAppStore()

const models = ref<UserModelMarketModel[]>([])
const loading = ref(false)
const loadError = ref('')

const searchQuery = ref('')
const selectedProtocol = ref('')
const selectedProvider = ref('')
const selectedCapability = ref('')

function normalizeKey(value?: string | null): string {
  return normalizePlatformKey(value)
}

function uniqueSorted(values: string[]): string[] {
  return Array.from(new Set(values.filter(Boolean))).sort((left, right) =>
    left.localeCompare(right, undefined, { sensitivity: 'base' }),
  )
}

function formatProviderLabel(value?: string | null): string {
  const normalized = normalizeKey(value)
  if (!normalized) return t('modelMarket.unknownProvider')
  return isCompatiblePlatform(normalized)
    ? getProtocolDisplayName(normalized)
    : getPlatformDisplayName(normalized)
}

function formatProtocolLabel(value?: string | null): string {
  const normalized = normalizeKey(value)
  if (!normalized) return t('modelMarket.unknownProtocol')
  if (normalized === 'openai' || normalized === 'openai_compatible') {
    return t('modelMarket.protocolLabels.openai')
  }
  if (normalized === 'anthropic' || normalized === 'anthropic_compatible') {
    return t('modelMarket.protocolLabels.anthropic')
  }
  return getProtocolDisplayName(normalized)
}

function providerSourceKey(model: UserModelMarketModel): string {
  const provider = model.provider_key || model.provider || model.platform || ''
  return provider /*
  return `${provider} · ${platform}`
  */
}

function formatChannelSource(channel: UserModelMarketChannel): string {
  return formatProviderLabel(channel.provider_key || channel.provider || channel.platform)
}

function channelSourceKey(channel: UserModelMarketChannel): string {
  return channel.provider_key || channel.provider || channel.platform || ''
}

function resolveBadgeTone(value?: string | null, protocol?: string | null): string {
  return resolvePlatformBadgeKey(value, protocol) ?? resolveProtocolBadgeKey(protocol) ?? ''
}

function providerBadgeClass(value?: string | null, protocol?: string | null): string {
  const badgeTone = resolveBadgeTone(value, protocol)
  return badgeTone ? platformBadgeLightClass(badgeTone) : 'model-market-badge-neutral'
}

function protocolBadgeClass(value?: string | null): string {
  const badgeTone = resolveProtocolBadgeKey(value)
  return badgeTone ? platformBadgeLightClass(badgeTone) : 'model-market-badge-neutral'
}

function groupBadgeClass(group: UserAvailableGroup): string {
  return providerBadgeClass(group.platform, group.platform)
}

function formatContextWindow(value?: number | null): string {
  if (!value) return t('modelMarket.unknownContext')
  return formatNumberLocaleString(value)
}

function resolvePricing(model: UserModelMarketModel): UserModelMarketPricing | null {
  if (model.price_summary) return model.price_summary
  return model.channels.find((channel) => channel.pricing)?.pricing ?? null
}

function billingModeLabel(pricing: UserModelMarketPricing): string {
  const mode = normalizeKey(pricing.billing_mode)
  switch (mode) {
    case 'token':
    case 'per_token':
      return t('availableChannels.pricing.billingModeToken')
    case 'per_request':
    case 'request':
      return t('availableChannels.pricing.billingModePerRequest')
    case 'image':
    case 'per_image':
      return t('availableChannels.pricing.billingModeImage')
    default:
      return pricing.billing_mode || t('modelMarket.unknownProtocol')
  }
}

function pricingBadges(model: UserModelMarketModel): Array<{ label: string; value: string }> {
  const pricing = resolvePricing(model)
  if (!pricing) return []

  const badges: Array<{ label: string; value: string }> = [
    {
      label: t('availableChannels.pricing.billingMode'),
      value: billingModeLabel(pricing),
    },
  ]

  if (pricing.input_price != null) {
    badges.push({
      label: t('availableChannels.pricing.inputPrice'),
      value: `${formatScaled(pricing.input_price, 1_000_000)} ${t('availableChannels.pricing.unitPerMillion')}`,
    })
  }

  if (pricing.output_price != null) {
    badges.push({
      label: t('availableChannels.pricing.outputPrice'),
      value: `${formatScaled(pricing.output_price, 1_000_000)} ${t('availableChannels.pricing.unitPerMillion')}`,
    })
  }

  if (pricing.cache_write_price != null) {
    badges.push({
      label: t('availableChannels.pricing.cacheWritePrice'),
      value: `${formatScaled(pricing.cache_write_price, 1_000_000)} ${t('availableChannels.pricing.unitPerMillion')}`,
    })
  }

  if (pricing.cache_read_price != null) {
    badges.push({
      label: t('availableChannels.pricing.cacheReadPrice'),
      value: `${formatScaled(pricing.cache_read_price, 1_000_000)} ${t('availableChannels.pricing.unitPerMillion')}`,
    })
  }

  if (pricing.image_output_price != null) {
    badges.push({
      label: t('availableChannels.pricing.imageOutputPrice'),
      value: `${formatScaled(pricing.image_output_price, 1)} ${t('availableChannels.pricing.unitPerRequest')}`,
    })
  }

  if (pricing.per_request_price != null) {
    badges.push({
      label: t('availableChannels.pricing.perRequestPrice'),
      value: `${formatScaled(pricing.per_request_price, 1)} ${t('availableChannels.pricing.unitPerRequest')}`,
    })
  }

  if (pricing.intervals.length > 0) {
    badges.push({
      label: t('availableChannels.pricing.intervals'),
      value: t('modelMarket.badges.tieredPricing'),
    })
  }

  return badges
}

function capabilityChips(model: UserModelMarketModel): string[] {
  return uniqueSorted([...model.capabilities, ...model.tags.filter((tag) => !model.capabilities.includes(tag))])
}

function formatStatusLabel(model: UserModelMarketModel): string {
  if (model.available === false) return t('modelMarket.status.unavailable')

  const status = normalizeKey(model.status)
  if (status === 'active') return t('modelMarket.status.active')
  if (status === 'hidden') return t('modelMarket.status.hidden')
  if (status === 'disabled') return t('modelMarket.status.disabled')

  return status ? formatProviderLabel(status) : t('modelMarket.status.active')
}

function statusBadgeClass(model: UserModelMarketModel): string {
  if (model.available === false) return 'model-market-badge-disabled'

  const status = normalizeKey(model.status)
  if (status === 'disabled') return 'model-market-badge-disabled'
  if (status === 'hidden') return 'model-market-badge-hidden'
  return 'model-market-badge-active'
}

const protocolOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('modelMarket.filters.allProtocols') },
  ...uniqueSorted(
    models.value
      .map((model) => normalizeKey(model.protocol))
      .filter(Boolean),
  ).map((value) => ({
    value,
    label: formatProtocolLabel(value),
  })),
])

const providerOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('modelMarket.filters.allProviders') },
  ...uniqueSorted(
    models.value
      .map((model) => normalizeKey(model.provider_key || model.provider || model.platform))
      .filter(Boolean),
  ).map((value) => ({
    value,
    label: formatProviderLabel(value),
  })),
])

const capabilityOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('modelMarket.filters.allCapabilities') },
  ...uniqueSorted(
    models.value.flatMap((model) =>
      capabilityChips(model).map((value) => normalizeKey(value)).filter(Boolean),
    ),
  ).map((value) => ({
    value,
    label: value,
  })),
])

const searchActive = computed(
  () =>
    Boolean(searchQuery.value.trim()) ||
    Boolean(selectedProtocol.value) ||
    Boolean(selectedProvider.value) ||
    Boolean(selectedCapability.value),
)

const filteredModels = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()

  return models.value.filter((model) => {
    const matchesProtocol =
      !selectedProtocol.value || normalizeKey(model.protocol) === selectedProtocol.value
    const matchesProvider =
      !selectedProvider.value ||
      normalizeKey(model.provider_key || model.provider || model.platform) === selectedProvider.value
    const matchesCapability =
      !selectedCapability.value ||
      capabilityChips(model).some((capability) => normalizeKey(capability) === selectedCapability.value)

    if (!matchesProtocol || !matchesProvider || !matchesCapability) return false
    if (!query) return true

    const haystack = [
      model.display_name,
      model.model_id,
      model.description,
      model.provider_key,
      model.provider,
      model.platform,
      model.protocol,
      ...model.capabilities,
      ...model.tags,
      ...model.channels.flatMap((channel) => [channel.name, channel.provider_key, channel.provider, channel.platform]),
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()

    return haystack.includes(query)
  })
})

const totalVisibleChannels = computed(() =>
  filteredModels.value.reduce((count, model) => count + model.channels.length, 0),
)

function resetFilters() {
  searchQuery.value = ''
  selectedProtocol.value = ''
  selectedProvider.value = ''
  selectedCapability.value = ''
}

function handleEmptyAction() {
  if (searchActive.value) {
    resetFilters()
    return
  }

  void loadModels()
}

async function loadModels() {
  loading.value = true
  loadError.value = ''

  try {
    models.value = await userModelMarketAPI.getModels()
  } catch (error: unknown) {
    loadError.value = extractApiErrorMessage(error, t('common.error'))
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadModels()
})
</script>

<style scoped>
.model-market-badge {
  @apply inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold;
}

.model-market-badge-openai {
  @apply bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300;
}

.model-market-badge-anthropic {
  @apply bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300;
}

.model-market-badge-neutral {
  @apply bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200;
}

.model-market-badge-recommended {
  @apply bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300;
}

.model-market-badge-active {
  @apply bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300;
}

.model-market-badge-hidden {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300;
}

.model-market-badge-disabled {
  @apply bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300;
}
</style>
