<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <SearchInput
              :model-value="filters.keyword"
              :placeholder="t('admin.modelMarket.searchPlaceholder')"
              class="w-full sm:w-72"
              @update:model-value="filters.keyword = String($event || '')"
            />
            <Select v-model="filters.protocol" :options="protocolOptions" class="w-40" />
            <Select v-model="filters.provider_key" :options="providerOptions" class="w-48" />
            <Select v-model="filters.status" :options="statusOptions" class="w-40" />
            <Select v-model="filters.capability" :options="capabilityOptions" class="w-44" />
          </div>

          <div class="flex w-full flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading || importing"
              @click="loadItems"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading || importing"
              @click="handleImport"
            >
              <Icon name="arrowPath" size="sm" class="mr-2" :class="importing ? 'animate-spin' : ''" />
              {{ t('admin.modelMarket.importFromChannels') }}
            </button>
            <button type="button" class="btn btn-primary" @click="openCreateDialog">
              <Icon name="plus" size="sm" class="mr-2" />
              {{ t('admin.modelMarket.createModel') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredItems" :loading="loading">
          <template #cell-display_name="{ row }">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">
                {{ row.display_name || row.model_id }}
              </div>
              <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="font-mono">{{ row.model_id }}</span>
                <span v-if="row.tags.length > 0" class="flex flex-wrap gap-1">
                  <span
                    v-for="tag in row.tags"
                    :key="`${row.id}-tag-${tag}`"
                    class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                  >
                    {{ tag }}
                  </span>
                </span>
              </div>
            </div>
          </template>

          <template #cell-provider_key="{ row }">
            <div class="space-y-2">
              <div class="flex flex-wrap items-center gap-2">
                <span
                  class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="providerBadgeClass(row.provider_key, row.protocol)"
                >
                  <PlatformIcon :platform="providerBadgeTone(row.provider_key, row.protocol) || row.provider_key" size="xs" />
                  {{ providerLabel(row.provider_key) }}
                </span>
                <span
                  class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="protocolBadgeClass(row.protocol)"
                >
                  <PlatformIcon :platform="resolveProtocolBadgeKey(row.protocol) || row.protocol" size="xs" />
                  {{ protocolLabel(row.protocol) }}
                </span>
                <span class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ row.provider_key }}</span>
              </div>
              <div v-if="row.available_channel_count != null" class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.modelMarket.availableChannelCount', { count: row.available_channel_count }) }}
              </div>
              <div v-if="row.channel_refs?.length" class="space-y-1">
                <div
                  v-for="ref in row.channel_refs.slice(0, 3)"
                  :key="`${row.id}-channel-${ref.channel_id}`"
                  class="flex max-w-xs flex-wrap items-center gap-1 text-[11px]"
                >
                  <span class="rounded-full bg-gray-100 px-2 py-0.5 text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                    {{ ref.channel_name || `#${ref.channel_id}` }}
                  </span>
                  <span
                    :class="['rounded-full px-2 py-0.5', providerBadgeClass(ref.platform, row.protocol)]"
                  >
                    {{ providerLabel(ref.platform) }}
                  </span>
                  <span
                    v-if="ref.group_ids.length"
                    class="rounded-full bg-gray-100 px-2 py-0.5 text-gray-500 dark:bg-dark-700 dark:text-gray-400"
                  >
                    {{ formatVisibleGroups(ref.group_ids, ref.platform) }}
                  </span>
                </div>
                <span v-if="row.channel_refs.length > 3" class="text-[11px] text-gray-400">
                  +{{ row.channel_refs.length - 3 }}
                </span>
              </div>
            </div>
          </template>

          <template #cell-capabilities="{ row }">
            <div class="flex max-w-xs flex-wrap gap-1">
              <span
                v-for="capability in row.capabilities"
                :key="`${row.id}-cap-${capability}`"
                class="rounded-full bg-primary-50 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
              >
                {{ capability }}
              </span>
              <span v-if="row.capabilities.length === 0" class="text-sm text-gray-400">-</span>
            </div>
          </template>

          <template #cell-context_window="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ formatContextWindow(value) }}
            </span>
          </template>

          <template #cell-status="{ value }">
            <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="statusClass(value)">
              {{ statusLabel(value) }}
            </span>
          </template>

          <template #cell-sort_order="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ value ?? 0 }}</span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ value ? formatDateTime(value) : '-' }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                :title="t('common.edit')"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-300"
                :title="t('common.delete')"
                @click="openDeleteDialog(row)"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.modelMarket.emptyTitle')"
              :description="t('admin.modelMarket.emptyDescription')"
              :action-text="t('admin.modelMarket.createModel')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showDialog"
      :title="editingItem ? t('admin.modelMarket.editModel') : t('admin.modelMarket.createModel')"
      width="wide"
      @close="closeDialog"
    >
      <form id="model-market-form" class="space-y-4" @submit.prevent="handleSave">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.displayName') }}</label>
            <input
              v-model="form.display_name"
              type="text"
              class="input"
              :placeholder="t('admin.modelMarket.form.displayNamePlaceholder')"
              required
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.modelId') }}</label>
            <input
              v-model="form.model_id"
              type="text"
              class="input font-mono"
              :placeholder="t('admin.modelMarket.form.modelIdPlaceholder')"
              required
            />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.providerKey') }}</label>
            <input
              v-model="form.provider_key"
              type="text"
              class="input"
              :placeholder="t('admin.modelMarket.form.providerKeyPlaceholder')"
              required
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.protocol') }}</label>
            <Select v-model="form.protocol" :options="protocolOptions.slice(1)" />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.capabilities') }}</label>
            <input
              v-model="form.capabilitiesText"
              type="text"
              class="input"
              :placeholder="t('admin.modelMarket.form.capabilitiesPlaceholder')"
            />
            <p class="input-hint">{{ t('admin.modelMarket.form.capabilitiesHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.tags') }}</label>
            <input
              v-model="form.tagsText"
              type="text"
              class="input"
              :placeholder="t('admin.modelMarket.form.tagsPlaceholder')"
            />
            <p class="input-hint">{{ t('admin.modelMarket.form.tagsHint') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.contextWindow') }}</label>
            <input
              v-model="form.context_window"
              type="number"
              min="0"
              step="1"
              class="input"
              :placeholder="t('admin.modelMarket.form.contextWindowPlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.status') }}</label>
            <Select v-model="form.status" :options="statusOptions.slice(1)" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelMarket.form.sortOrder') }}</label>
            <input v-model="form.sort_order" type="number" step="1" class="input" />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelMarket.form.description') }}</label>
          <textarea
            v-model="form.description"
            rows="3"
            class="input"
            :placeholder="t('admin.modelMarket.form.descriptionPlaceholder')"
          ></textarea>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelMarket.form.metadata') }}</label>
          <textarea
            v-model="form.metadataText"
            rows="5"
            class="input font-mono"
            :placeholder="t('admin.modelMarket.form.metadataPlaceholder')"
          ></textarea>
          <p class="input-hint">{{ t('admin.modelMarket.form.metadataHint') }}</p>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="model-market-form" class="btn btn-primary" :disabled="saving">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.modelMarket.deleteModel')"
      :message="deleteMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import { platformBadgeLightClass } from '@/utils/platformColors'
import type {
  AdminGroup,
  ModelMarketItem,
  ModelMarketProtocol,
  ModelMarketStatus,
} from '@/types'
import type { Column } from '@/components/common/types'
import {
  getPlatformDisplayName,
  getProtocolDisplayName,
  isCompatiblePlatform,
  resolvePlatformBadgeKey,
  resolveProtocolBadgeKey,
} from '@/utils/platforms'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<ModelMarketItem[]>([])
const groups = ref<AdminGroup[]>([])
const loading = ref(false)
const importing = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
const editingItem = ref<ModelMarketItem | null>(null)
const deletingItem = ref<ModelMarketItem | null>(null)

const filters = reactive({
  keyword: '',
  protocol: '',
  provider_key: '',
  status: '',
  capability: '',
})

const form = reactive({
  model_id: '',
  display_name: '',
  provider_key: '',
  protocol: 'openai' as ModelMarketProtocol,
  capabilitiesText: '',
  context_window: '',
  description: '',
  tagsText: '',
  status: 'active' as ModelMarketStatus,
  sort_order: '0',
  metadataText: '',
})

const protocolOptions = computed(() => [
  { value: '', label: t('admin.modelMarket.allProtocols') },
  { value: 'openai', label: t('admin.modelMarket.protocols.openai') },
  { value: 'anthropic', label: t('admin.modelMarket.protocols.anthropic') },
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.modelMarket.allStatuses') },
  { value: 'active', label: t('admin.modelMarket.statuses.active') },
  { value: 'hidden', label: t('admin.modelMarket.statuses.hidden') },
  { value: 'disabled', label: t('admin.modelMarket.statuses.disabled') },
])

const groupMap = computed(() => new Map(groups.value.map((group) => [group.id, group])))

const providerOptions = computed(() => [
  { value: '', label: t('admin.modelMarket.allProviders') },
  ...Array.from(new Set(items.value.map((item) => item.provider_key)))
    .filter((value) => value.length > 0)
    .sort((left, right) => left.localeCompare(right))
    .map((value) => ({ value, label: getPlatformDisplayName(value) })),
])

const capabilityOptions = computed(() => [
  { value: '', label: t('admin.modelMarket.allCapabilities') },
  ...Array.from(new Set(items.value.flatMap((item) => item.capabilities)))
    .filter((value) => value.length > 0)
    .sort((left, right) => left.localeCompare(right))
    .map((value) => ({ value, label: value })),
])

const filteredItems = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  return [...items.value]
    .filter((item) => {
      if (filters.protocol && item.protocol !== filters.protocol) return false
      if (filters.provider_key && item.provider_key !== filters.provider_key) return false
      if (filters.status && item.status !== filters.status) return false
      if (filters.capability && !item.capabilities.includes(filters.capability)) return false
      if (!keyword) return true

      const haystack = [
        item.model_id,
        item.display_name,
        item.provider_key,
        item.description || '',
        ...item.tags,
        ...item.capabilities,
      ]
        .join(' ')
        .toLowerCase()

      return haystack.includes(keyword)
    })
    .sort((left, right) => {
      if (left.sort_order !== right.sort_order) {
        return left.sort_order - right.sort_order
      }
      return (left.display_name || left.model_id).localeCompare(right.display_name || right.model_id)
    })
})

const columns = computed<Column[]>(() => [
  { key: 'display_name', label: t('admin.modelMarket.columns.model'), sortable: true },
  { key: 'provider_key', label: t('admin.modelMarket.columns.provider'), sortable: true },
  { key: 'capabilities', label: t('admin.modelMarket.columns.capabilities'), sortable: false },
  { key: 'context_window', label: t('admin.modelMarket.columns.contextWindow'), sortable: true },
  { key: 'status', label: t('admin.modelMarket.columns.status'), sortable: true },
  { key: 'sort_order', label: t('admin.modelMarket.columns.sortOrder'), sortable: true },
  { key: 'updated_at', label: t('admin.modelMarket.columns.updatedAt'), sortable: true },
  { key: 'actions', label: t('admin.modelMarket.columns.actions'), sortable: false },
])

const deleteMessage = computed(() => {
  const item = deletingItem.value
  if (!item) return ''
  return t('admin.modelMarket.deleteConfirm', {
    name: item.display_name || item.model_id,
  })
})

function statusLabel(status: string) {
  return t(`admin.modelMarket.statuses.${status}`, status)
}

function providerLabel(value?: string | null) {
  return isCompatiblePlatform(value)
    ? getProtocolDisplayName(value)
    : getPlatformDisplayName(value)
}

function protocolLabel(protocol: ModelMarketProtocol | string) {
  const key = `admin.modelMarket.protocols.${protocol}`
  const translated = t(key)
  return translated !== key ? translated : getProtocolDisplayName(protocol)
}

function protocolBadgeClass(protocol?: string | null) {
  const badgeTone = resolveProtocolBadgeKey(protocol)
  return badgeTone ? platformBadgeLightClass(badgeTone) : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200'
}

function providerBadgeTone(value?: string | null, protocol?: string | null) {
  return resolvePlatformBadgeKey(value, protocol) ?? resolveProtocolBadgeKey(protocol) ?? ''
}

function providerBadgeClass(value?: string | null, protocol?: string | null) {
  const badgeTone = providerBadgeTone(value, protocol)
  return badgeTone ? platformBadgeLightClass(badgeTone) : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200'
}

function formatVisibleGroups(groupIds: number[], platform?: string | null) {
  const names = groupIds
    .map((groupId) => groupMap.value.get(groupId))
    .filter((group) => {
      if (!group) return true
      if (!platform) return true
      return String(group.platform || '').toLowerCase() === String(platform).toLowerCase()
    })
    .map((group) => group?.name || '')
    .filter((value) => value.length > 0)

  if (names.length <= 2) {
    return names.join(', ')
  }
  return `${names.slice(0, 2).join(', ')} +${names.length - 2}`
}

function statusClass(status: string) {
  if (status === 'active') {
    return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  }
  if (status === 'hidden') {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  }
  return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
}

function formatContextWindow(value: unknown) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return '-'
  }
  return value.toLocaleString()
}

function parseStringListInput(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(/[,\n]/)
        .map((item) => item.trim())
        .filter((item) => item.length > 0)
    )
  )
}

function parseMetadataInput(value: string): Record<string, unknown> | undefined {
  if (!value.trim()) {
    return undefined
  }

  try {
    const parsed = JSON.parse(value)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('metadata must be an object')
    }
    return parsed as Record<string, unknown>
  } catch {
    appStore.showError(t('admin.modelMarket.form.metadataInvalid'))
    return undefined
  }
}

function resetForm() {
  form.model_id = ''
  form.display_name = ''
  form.provider_key = ''
  form.protocol = 'openai'
  form.capabilitiesText = ''
  form.context_window = ''
  form.description = ''
  form.tagsText = ''
  form.status = 'active'
  form.sort_order = '0'
  form.metadataText = ''
}

function syncForm(item: ModelMarketItem) {
  form.model_id = item.model_id
  form.display_name = item.display_name
  form.provider_key = item.provider_key
  form.protocol = item.protocol
  form.capabilitiesText = item.capabilities.join(', ')
  form.context_window = item.context_window ? String(item.context_window) : ''
  form.description = item.description || ''
  form.tagsText = item.tags.join(', ')
  form.status = item.status
  form.sort_order = String(item.sort_order ?? 0)
  form.metadataText = item.metadata ? JSON.stringify(item.metadata, null, 2) : ''
}

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch {
    groups.value = []
  }
}

async function loadItems() {
  loading.value = true
  try {
    items.value = await adminAPI.modelMarket.list()
  } catch {
    appStore.showError(t('admin.modelMarket.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingItem.value = null
  resetForm()
  showDialog.value = true
}

function openEditDialog(item: ModelMarketItem) {
  editingItem.value = item
  syncForm(item)
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editingItem.value = null
  resetForm()
}

function openDeleteDialog(item: ModelMarketItem) {
  deletingItem.value = item
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  try {
    await adminAPI.modelMarket.remove(deletingItem.value.id)
    appStore.showSuccess(t('admin.modelMarket.modelDeleted'))
    showDeleteDialog.value = false
    deletingItem.value = null
    await loadItems()
  } catch {
    appStore.showError(t('admin.modelMarket.deleteFailed'))
  }
}

async function handleImport() {
  importing.value = true
  try {
    const result = await adminAPI.modelMarket.importFromChannels()
    const message =
      result.message ||
      t('admin.modelMarket.importSuccess', {
        imported: result.imported_count ?? result.created_count ?? result.imported ?? 0,
        updated: result.updated_count ?? result.updated ?? 0,
      })
    appStore.showSuccess(message)
    await loadItems()
  } catch {
    appStore.showError(t('admin.modelMarket.importFailed'))
  } finally {
    importing.value = false
  }
}

async function handleSave() {
  if (!form.model_id.trim() || !form.display_name.trim() || !form.provider_key.trim()) {
    appStore.showError(t('admin.modelMarket.form.requiredFields'))
    return
  }

  const metadata = parseMetadataInput(form.metadataText)
  if (form.metadataText.trim() && !metadata) {
    return
  }

  const parsedContextWindow = form.context_window.trim()
    ? Math.trunc(Number(form.context_window))
    : null

  if (parsedContextWindow !== null && (!Number.isFinite(parsedContextWindow) || parsedContextWindow <= 0)) {
    appStore.showError(t('admin.modelMarket.form.contextWindowInvalid'))
    return
  }

  const payload = {
    model_id: form.model_id.trim(),
    display_name: form.display_name.trim(),
    provider_key: form.provider_key.trim(),
    protocol: form.protocol,
    capabilities: parseStringListInput(form.capabilitiesText),
    context_window: parsedContextWindow,
    description: form.description.trim() || null,
    tags: parseStringListInput(form.tagsText),
    status: form.status,
    sort_order: Math.trunc(Number(form.sort_order || '0')) || 0,
    ...(metadata ? { metadata } : {}),
  }

  saving.value = true
  try {
    if (editingItem.value) {
      await adminAPI.modelMarket.update(editingItem.value.id, payload)
      appStore.showSuccess(t('admin.modelMarket.modelUpdated'))
    } else {
      await adminAPI.modelMarket.create(payload)
      appStore.showSuccess(t('admin.modelMarket.modelCreated'))
    }
    closeDialog()
    await loadItems()
  } catch {
    appStore.showError(t('admin.modelMarket.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadGroups()
  void loadItems()
})
</script>
