<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="card p-6">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div class="max-w-3xl">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('admin.traces.title') }}
            </h1>
            <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.traces.description') }}
            </p>
          </div>
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div class="rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('admin.traces.stats.records') }}
              </div>
              <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                {{ recordsPagination.total.toLocaleString() }}
              </div>
            </div>
            <div class="rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('admin.traces.stats.rules') }}
              </div>
              <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                {{ rules.length.toLocaleString() }}
              </div>
            </div>
            <div class="rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('admin.traces.stats.exports') }}
              </div>
              <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                {{ exportTasksPagination.total.toLocaleString() }}
              </div>
            </div>
            <div class="rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('admin.traces.stats.rootAdmin') }}
              </div>
              <div class="mt-2 inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-xs font-medium" :class="isRootAdmin ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'">
                <Icon :name="isRootAdmin ? 'checkCircle' : 'shield'" size="sm" />
                <span>{{ isRootAdmin ? t('admin.traces.badges.rootAdmin') : t('admin.traces.badges.adminOnly') }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="card p-2">
        <div class="grid gap-2 md:grid-cols-3">
          <button
            v-for="tab in traceTabs"
            :key="tab.key"
            type="button"
            class="flex items-center justify-between rounded-xl px-4 py-3 text-left transition-colors"
            :class="activeTab === tab.key
              ? 'bg-primary-50 text-primary-700 ring-1 ring-primary-200 dark:bg-primary-900/30 dark:text-primary-300 dark:ring-primary-800'
              : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-800 dark:hover:text-white'"
            @click="activeTab = tab.key"
          >
            <span class="flex items-center gap-3">
              <span class="inline-flex h-9 w-9 items-center justify-center rounded-lg" :class="activeTab === tab.key ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-300'">
                <Icon :name="tab.icon" size="sm" />
              </span>
              <span>
                <span class="block text-sm font-medium">{{ tab.label }}</span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">{{ tab.description }}</span>
              </span>
            </span>
            <span
              v-if="tab.key === 'exports'"
              class="rounded-full px-2 py-0.5 text-[11px] font-medium"
              :class="isRootAdmin ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'"
            >
              {{ isRootAdmin ? t('admin.traces.badges.rootAdmin') : t('admin.traces.tabs.rootOnly') }}
            </span>
          </button>
        </div>
      </section>

      <template v-if="activeTab === 'records'">
        <section class="card p-6">
          <form class="space-y-4" @submit.prevent="applyRecordFilters">
            <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.traces.records.title') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.records.description') }}
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  class="btn btn-secondary"
                  :disabled="recordsLoading"
                  @click="loadRecords"
                >
                  <Icon name="refresh" size="sm" :class="recordsLoading ? 'mr-2 animate-spin' : 'mr-2'" />
                  {{ t('common.refresh') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary"
                  @click="resetRecordFilters"
                >
                  <Icon name="x" size="sm" class="mr-2" />
                  {{ t('common.reset') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary"
                  @click="showAdvancedRecordFilters = !showAdvancedRecordFilters"
                >
                  <Icon :name="showAdvancedRecordFilters ? 'chevronUp' : 'filter'" size="sm" class="mr-2" />
                  {{ showAdvancedRecordFilters ? t('admin.traces.records.hideAdvanced') : t('admin.traces.records.showAdvanced') }}
                </button>
                <button
                  type="button"
                  class="btn btn-danger"
                  :disabled="selectedRecordCount === 0 || recordsLoading"
                  data-testid="trace-bulk-delete"
                  @click="confirmDeleteSelectedRecords"
                >
                  <Icon name="trash" size="sm" class="mr-2" />
                  {{ t('admin.traces.records.deleteSelected', { count: selectedRecordCount }) }}
                </button>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.model') }}
                </span>
                <input v-model="recordFilters.model" type="text" class="input w-full" :placeholder="t('admin.traces.filters.modelPlaceholder')" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.keyword') }}
                </span>
                <input v-model="recordFilters.keyword" type="text" class="input w-full" :placeholder="t('admin.traces.filters.keywordPlaceholder')" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.userId') }}
                </span>
                <input v-model="recordFilters.user_id" type="number" min="1" class="input w-full" placeholder="42" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.apiKeyId') }}
                </span>
                <input v-model="recordFilters.api_key_id" type="number" min="1" class="input w-full" placeholder="108" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.ruleId') }}
                </span>
                <select v-model="recordFilters.capture_rule_id" class="input w-full">
                  <option value="">{{ t('admin.traces.filters.allRules') }}</option>
                  <option v-for="rule in sortedRules" :key="rule.id" :value="String(rule.id)">
                    #{{ rule.id }} · {{ rule.name }}
                  </option>
                </select>
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.startDate') }}
                </span>
                <input v-model="recordFilters.start_date" type="date" class="input w-full" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.endDate') }}
                </span>
                <input v-model="recordFilters.end_date" type="date" class="input w-full" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.minTotalTokens') }}
                </span>
                <input v-model="recordFilters.min_total_tokens" type="number" min="0" class="input w-full" placeholder="0" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.maxTotalTokens') }}
                </span>
                <input v-model="recordFilters.max_total_tokens" type="number" min="0" class="input w-full" placeholder="50000" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.timezone') }}
                </span>
                <input v-model="recordFilters.timezone" type="text" class="input w-full" placeholder="Asia/Shanghai" />
              </label>
            </div>

            <div v-if="showAdvancedRecordFilters" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.minInputTokens') }}
                </span>
                <input v-model="recordFilters.min_input_tokens" type="number" min="0" class="input w-full" placeholder="0" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.maxInputTokens') }}
                </span>
                <input v-model="recordFilters.max_input_tokens" type="number" min="0" class="input w-full" placeholder="12000" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.minOutputTokens') }}
                </span>
                <input v-model="recordFilters.min_output_tokens" type="number" min="0" class="input w-full" placeholder="0" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.maxOutputTokens') }}
                </span>
                <input v-model="recordFilters.max_output_tokens" type="number" min="0" class="input w-full" placeholder="12000" />
              </label>
            </div>

            <div class="flex justify-end">
              <button type="submit" class="btn btn-primary" :disabled="recordsLoading">
                <Icon name="search" size="sm" class="mr-2" />
                {{ t('admin.traces.records.applyFilters') }}
              </button>
            </div>
          </form>
        </section>

        <section class="card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-6 py-4 dark:border-dark-700">
            <div class="text-sm text-gray-600 dark:text-gray-300">
              {{ t('admin.traces.records.selectedCount', { count: selectedRecordCount, total: recordsPagination.total }) }}
            </div>
            <div class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.traces.records.pageHint', { page: recordsPagination.page, pages: Math.max(recordsPagination.pages, 1) }) }}
            </div>
          </div>
          <DataTable
            :columns="recordColumns"
            :data="records"
            :loading="recordsLoading"
            row-key="id"
            default-sort-key="created_at"
            default-sort-order="desc"
            sort-storage-key="admin-traces-records-sort"
          >
            <template #header-select>
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="allVisibleSelected"
                @click.stop
                @change="toggleSelectAllVisible"
              />
            </template>
            <template #cell-select="{ row }">
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="selectedRecordIds.has(row.id)"
                @click.stop
                @change="toggleRecordSelection(row.id)"
              />
            </template>
            <template #cell-id="{ row }">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-mono text-sm font-medium text-gray-900 dark:text-white">#{{ row.id }}</span>
                  <span class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                    {{ row.protocol || '-' }}
                  </span>
                </div>
                <div class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400" :title="row.task_id">
                  {{ row.task_id || '-' }}
                </div>
              </div>
            </template>
            <template #cell-model="{ row }">
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.model || '-' }}</div>
                <div class="mt-1 flex flex-col gap-1 text-xs text-gray-500 dark:text-gray-400">
                  <span v-if="row.requested_model">{{ t('admin.traces.records.requestedModel') }}: {{ row.requested_model }}</span>
                  <span v-if="row.upstream_model">{{ t('admin.traces.records.upstreamModel') }}: {{ row.upstream_model }}</span>
                  <span>{{ row.scaffold || '-' }} · {{ row.scaffold_version || '-' }}</span>
                </div>
              </div>
            </template>
            <template #cell-actors="{ row }">
              <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                <div>{{ t('admin.traces.records.user') }}: {{ formatOptionalId(row.user_id) }}</div>
                <div>{{ t('admin.traces.records.apiKey') }}: {{ formatOptionalId(row.api_key_id) }}</div>
                <div>{{ t('admin.traces.records.account') }}: {{ formatOptionalId(row.account_id) }}</div>
                <div>{{ t('admin.traces.records.rule') }}: {{ formatOptionalId(row.capture_rule_id) }}</div>
              </div>
            </template>
            <template #cell-tokens="{ row }">
              <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                <div>{{ t('admin.traces.records.inputTokens') }}: {{ formatOptionalNumber(row.input_tokens) }}</div>
                <div>{{ t('admin.traces.records.outputTokens') }}: {{ formatOptionalNumber(row.output_tokens) }}</div>
                <div class="font-medium text-gray-900 dark:text-white">
                  {{ t('admin.traces.records.totalTokens') }}: {{ formatOptionalNumber(row.total_tokens) }}
                </div>
              </div>
            </template>
            <template #cell-status="{ row }">
              <div class="space-y-1">
                <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" :class="traceStatusClass(row.upstream_status_code)">
                  {{ formatTraceStatus(row.upstream_status_code) }}
                </span>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.request_content_type || '-' }}
                </div>
              </div>
            </template>
            <template #cell-created_at="{ value }">
              <div class="space-y-1">
                <div class="text-sm text-gray-900 dark:text-white">{{ formatDate(value) }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatRelative(value) }}</div>
              </div>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-1">
                <button
                  type="button"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="t('common.view')"
                  @click="openRecordDetail(row.id)"
                >
                  <Icon name="eye" size="sm" />
                  <span class="text-xs">{{ t('common.view') }}</span>
                </button>
                <button
                  type="button"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-300"
                  :title="t('common.delete')"
                  @click="confirmDeleteRecord(row)"
                >
                  <Icon name="trash" size="sm" />
                  <span class="text-xs">{{ t('common.delete') }}</span>
                </button>
              </div>
            </template>
            <template #empty>
              <div class="px-6 py-12 text-center">
                <div class="text-base font-medium text-gray-900 dark:text-white">
                  {{ t('admin.traces.records.emptyTitle') }}
                </div>
                <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.records.emptyDescription') }}
                </p>
              </div>
            </template>
          </DataTable>
          <Pagination
            v-if="recordsPagination.total > 0"
            :page="recordsPagination.page"
            :total="recordsPagination.total"
            :page-size="recordsPagination.page_size"
            @update:page="handleRecordPageChange"
            @update:pageSize="handleRecordPageSizeChange"
          />
        </section>
      </template>

      <template v-else-if="activeTab === 'rules'">
        <section class="card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-6 py-4 dark:border-dark-700">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('admin.traces.rules.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.traces.rules.description') }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <button type="button" class="btn btn-secondary" :disabled="rulesLoading" @click="loadRules">
                <Icon name="refresh" size="sm" :class="rulesLoading ? 'mr-2 animate-spin' : 'mr-2'" />
                {{ t('common.refresh') }}
              </button>
              <button type="button" class="btn btn-primary" @click="openCreateRuleDialog">
                <Icon name="plus" size="sm" class="mr-2" />
                {{ t('admin.traces.rules.create') }}
              </button>
            </div>
          </div>
          <DataTable
            :columns="ruleColumns"
            :data="sortedRules"
            :loading="rulesLoading"
            row-key="id"
            default-sort-key="priority"
            default-sort-order="asc"
            sort-storage-key="admin-traces-rules-sort"
          >
            <template #cell-name="{ row }">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                  <span class="rounded-full px-2 py-0.5 text-[11px] font-medium" :class="row.enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'">
                    {{ row.enabled ? t('admin.traces.badges.enabled') : t('admin.traces.badges.disabled') }}
                  </span>
                  <span class="rounded-full bg-primary-50 px-2 py-0.5 text-[11px] font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                    {{ t('admin.traces.rules.priority', { value: row.priority }) }}
                  </span>
                </div>
                <div class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">
                  #{{ row.id }}
                </div>
              </div>
            </template>
            <template #cell-targets="{ row }">
              <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                <div>{{ t('admin.traces.rules.modelPatterns') }}: {{ formatList(row.model_patterns) }}</div>
                <div>{{ t('admin.traces.rules.userIds') }}: {{ formatIdList(row.user_ids) }}</div>
                <div>{{ t('admin.traces.rules.apiKeyIds') }}: {{ formatIdList(row.api_key_ids) }}</div>
                <div>{{ t('admin.traces.rules.keywords') }}: {{ formatList(row.keywords) }}</div>
              </div>
            </template>
            <template #cell-conditions="{ row }">
              <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                <div>{{ t('admin.traces.rules.tokenRange') }}: {{ formatRuleTokenRange(row) }}</div>
                <div>{{ t('admin.traces.rules.activeWindow') }}: {{ formatRuleWindow(row) }}</div>
              </div>
            </template>
            <template #cell-sampling_ratio="{ row }">
              <div class="space-y-2">
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatPercent(row.sampling_ratio) }}
                </div>
                <div class="h-2 w-28 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full rounded-full bg-primary-500" :style="{ width: `${Math.min(Math.max(row.sampling_ratio * 100, 0), 100)}%` }"></div>
                </div>
              </div>
            </template>
            <template #cell-updated_at="{ value }">
              <div class="space-y-1">
                <div class="text-sm text-gray-900 dark:text-white">{{ formatDate(value) }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatRelative(value) }}</div>
              </div>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-1">
                <button
                  type="button"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="t('common.edit')"
                  @click="openEditRuleDialog(row)"
                >
                  <Icon name="edit" size="sm" />
                  <span class="text-xs">{{ t('common.edit') }}</span>
                </button>
                <button
                  type="button"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-300"
                  :title="t('common.delete')"
                  @click="confirmDeleteRule(row)"
                >
                  <Icon name="trash" size="sm" />
                  <span class="text-xs">{{ t('common.delete') }}</span>
                </button>
              </div>
            </template>
            <template #empty>
              <div class="px-6 py-12 text-center">
                <div class="text-base font-medium text-gray-900 dark:text-white">
                  {{ t('admin.traces.rules.emptyTitle') }}
                </div>
                <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.rules.emptyDescription') }}
                </p>
              </div>
            </template>
          </DataTable>
        </section>
      </template>

      <template v-else>
        <section class="card p-6">
          <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('admin.traces.export.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.traces.export.description') }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="exportTasksLoading || exportRestricted"
                @click="loadExportTasks"
              >
                <Icon name="refresh" size="sm" :class="exportTasksLoading ? 'mr-2 animate-spin' : 'mr-2'" />
                {{ t('common.refresh') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="recordsLoading"
                @click="copyRecordFiltersToExport"
              >
                <Icon name="copy" size="sm" class="mr-2" />
                {{ t('admin.traces.export.copyFromRecords') }}
              </button>
            </div>
          </div>

          <div
            v-if="exportRestricted"
            data-testid="trace-export-root-warning"
            class="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-4 text-amber-900 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-100"
          >
            <div class="flex items-start gap-3">
              <Icon name="shield" size="md" class="mt-0.5 text-amber-600 dark:text-amber-300" />
              <div>
                <div class="font-medium">{{ t('admin.traces.access.rootOnlyTitle') }}</div>
                <p class="mt-1 text-sm text-amber-800 dark:text-amber-200">
                  {{ exportRestrictedMessage }}
                </p>
              </div>
            </div>
          </div>

          <form class="mt-4 space-y-4" @submit.prevent="submitExportTask">
            <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.model') }}
                </span>
                <input v-model="exportFilters.model" :disabled="exportRestricted" type="text" class="input w-full" :placeholder="t('admin.traces.filters.modelPlaceholder')" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.keyword') }}
                </span>
                <input v-model="exportFilters.keyword" :disabled="exportRestricted" type="text" class="input w-full" :placeholder="t('admin.traces.filters.keywordPlaceholder')" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.userId') }}
                </span>
                <input v-model="exportFilters.user_id" :disabled="exportRestricted" type="number" min="1" class="input w-full" placeholder="42" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.apiKeyId') }}
                </span>
                <input v-model="exportFilters.api_key_id" :disabled="exportRestricted" type="number" min="1" class="input w-full" placeholder="108" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.ruleId') }}
                </span>
                <select v-model="exportFilters.capture_rule_id" :disabled="exportRestricted" class="input w-full">
                  <option value="">{{ t('admin.traces.filters.allRules') }}</option>
                  <option v-for="rule in sortedRules" :key="`export-rule-${rule.id}`" :value="String(rule.id)">
                    #{{ rule.id }} · {{ rule.name }}
                  </option>
                </select>
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.startDate') }}
                </span>
                <input v-model="exportFilters.start_date" :disabled="exportRestricted" type="date" class="input w-full" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.endDate') }}
                </span>
                <input v-model="exportFilters.end_date" :disabled="exportRestricted" type="date" class="input w-full" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.minTotalTokens') }}
                </span>
                <input v-model="exportFilters.min_total_tokens" :disabled="exportRestricted" type="number" min="0" class="input w-full" placeholder="0" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.maxTotalTokens') }}
                </span>
                <input v-model="exportFilters.max_total_tokens" :disabled="exportRestricted" type="number" min="0" class="input w-full" placeholder="50000" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.timezone') }}
                </span>
                <input v-model="exportFilters.timezone" :disabled="exportRestricted" type="text" class="input w-full" placeholder="Asia/Shanghai" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.export.targetRecords') }}
                </span>
                <input v-model="exportTargetRecords" :disabled="exportRestricted" type="number" min="1" class="input w-full" placeholder="500" />
                <span class="block text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.export.targetRecordsHint') }}
                </span>
              </label>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3">
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="exportRestricted"
                @click="showAdvancedExportFilters = !showAdvancedExportFilters"
              >
                <Icon :name="showAdvancedExportFilters ? 'chevronUp' : 'filter'" size="sm" class="mr-2" />
                {{ showAdvancedExportFilters ? t('admin.traces.records.hideAdvanced') : t('admin.traces.records.showAdvanced') }}
              </button>
              <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="includeRawOnExport" :disabled="exportRestricted" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                <span>{{ t('admin.traces.export.includeRaw') }}</span>
              </label>
            </div>

            <div v-if="showAdvancedExportFilters" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.minInputTokens') }}
                </span>
                <input v-model="exportFilters.min_input_tokens" :disabled="exportRestricted" type="number" min="0" class="input w-full" placeholder="0" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.maxInputTokens') }}
                </span>
                <input v-model="exportFilters.max_input_tokens" :disabled="exportRestricted" type="number" min="0" class="input w-full" placeholder="12000" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.minOutputTokens') }}
                </span>
                <input v-model="exportFilters.min_output_tokens" :disabled="exportRestricted" type="number" min="0" class="input w-full" placeholder="0" />
              </label>
              <label class="space-y-1 text-sm">
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.traces.filters.maxOutputTokens') }}
                </span>
                <input v-model="exportFilters.max_output_tokens" :disabled="exportRestricted" type="number" min="0" class="input w-full" placeholder="12000" />
              </label>
            </div>

            <div class="flex justify-end">
              <button
                type="submit"
                class="btn btn-primary"
                data-testid="trace-export-create"
                :disabled="exportRestricted || exportCreating"
              >
                <Icon name="download" size="sm" :class="exportCreating ? 'mr-2 animate-pulse' : 'mr-2'" />
                {{ exportCreating ? t('common.loading') : t('admin.traces.export.createTask') }}
              </button>
            </div>
          </form>
        </section>

        <section class="card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-6 py-4 dark:border-dark-700">
            <div class="text-sm text-gray-600 dark:text-gray-300">
              {{ t('admin.traces.export.taskCount', { count: exportTasksPagination.total }) }}
            </div>
            <div v-if="hasRunningExportTasks" class="inline-flex items-center gap-2 rounded-full bg-blue-100 px-3 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
              <Icon name="refresh" size="sm" class="animate-spin" />
              {{ t('admin.traces.export.polling') }}
            </div>
          </div>
          <DataTable
            :columns="exportTaskColumns"
            :data="exportTasks"
            :loading="exportTasksLoading"
            row-key="id"
            default-sort-key="created_at"
            default-sort-order="desc"
            sort-storage-key="admin-traces-export-sort"
          >
            <template #cell-id="{ row }">
              <div class="space-y-1">
                <div class="font-mono text-sm font-medium text-gray-900 dark:text-white">#{{ row.id }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.format || '-' }}
                </div>
              </div>
            </template>
            <template #cell-status="{ row }">
              <div class="space-y-1">
                <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" :class="exportTaskStatusClass(row.status)">
                  {{ exportTaskStatusLabel(row.status) }}
                </span>
                <div v-if="row.error_message" class="max-w-xs text-xs text-red-600 dark:text-red-300">
                  {{ row.error_message }}
                </div>
              </div>
            </template>
            <template #cell-filters="{ row }">
              <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                <div v-for="line in summarizeTaskFilters(row.filters)" :key="`${row.id}-${line}`">
                  {{ line }}
                </div>
              </div>
            </template>
            <template #cell-progress="{ row }">
              <div class="space-y-2">
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.traces.export.progressLabel', { processed: row.processed_records, total: exportTaskProgressTotal(row) }) }}
                </div>
                <div class="h-2 w-28 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full rounded-full bg-primary-500" :style="{ width: `${exportTaskProgress(row)}%` }"></div>
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.include_raw ? t('admin.traces.export.includeRaw') : t('admin.traces.export.metadataOnly') }}
                </div>
              </div>
            </template>
            <template #cell-file="{ row }">
              <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.download_filename || ''">
                  {{ row.download_filename || '-' }}
                </div>
                <div>{{ formatFileSize(row.file_size_bytes) }}</div>
                <div>{{ row.finished_at ? formatDate(row.finished_at) : '-' }}</div>
              </div>
            </template>
            <template #cell-created_at="{ row }">
              <div class="space-y-1">
                <div class="text-sm text-gray-900 dark:text-white">{{ formatDate(row.created_at) }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.started_at ? formatRelative(row.started_at) : formatRelative(row.created_at) }}</div>
              </div>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-1">
                <button
                  v-if="canCancelExportTask(row)"
                  type="button"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-300"
                  :title="t('admin.traces.export.cancelTask')"
                  @click="confirmCancelExportTask(row)"
                >
                  <Icon name="ban" size="sm" />
                  <span class="text-xs">{{ t('admin.traces.export.cancelTask') }}</span>
                </button>
                <button
                  v-if="row.status === 'succeeded'"
                  type="button"
                  class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="t('common.download')"
                  :disabled="exportDownloadTaskId === row.id"
                  @click="downloadExportTask(row)"
                >
                  <Icon name="download" size="sm" />
                  <span class="text-xs">
                    {{ exportDownloadTaskId === row.id ? t('common.loading') : t('common.download') }}
                  </span>
                </button>
              </div>
            </template>
            <template #empty>
              <div class="px-6 py-12 text-center">
                <div class="text-base font-medium text-gray-900 dark:text-white">
                  {{ exportRestricted ? t('admin.traces.access.rootOnlyTitle') : t('admin.traces.export.emptyTitle') }}
                </div>
                <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ exportRestricted ? exportRestrictedMessage : t('admin.traces.export.emptyDescription') }}
                </p>
              </div>
            </template>
          </DataTable>
          <Pagination
            v-if="!exportRestricted && exportTasksPagination.total > 0"
            :page="exportTasksPagination.page"
            :total="exportTasksPagination.total"
            :page-size="exportTasksPagination.page_size"
            @update:page="handleExportPageChange"
            @update:pageSize="handleExportPageSizeChange"
          />
        </section>
      </template>
    </div>

    <BaseDialog
      :show="recordDetailVisible"
      :title="recordDetailDialogTitle"
      width="extra-wide"
      @close="closeRecordDetail"
    >
      <div v-if="recordDetailLoading" class="py-16 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>
      <div v-else-if="recordDetail" class="space-y-6">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="item in recordDetailSummary"
            :key="item.label"
            class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ item.label }}</div>
            <div class="mt-2 break-all text-sm font-medium text-gray-900 dark:text-white">{{ item.value }}</div>
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="btn"
            :class="detailMode === 'structured' ? 'btn-primary' : 'btn-secondary'"
            @click="detailMode = 'structured'"
          >
            {{ t('admin.traces.detail.structured') }}
          </button>
          <button
            type="button"
            class="btn"
            :class="detailMode === 'raw' ? 'btn-primary' : 'btn-secondary'"
            @click="detailMode = 'raw'"
          >
            {{ t('admin.traces.detail.raw') }}
          </button>
        </div>

        <div class="space-y-4">
          <div
            v-for="section in visibleDetailSections"
            :key="section.key"
            class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700"
          >
            <div class="border-b border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="text-sm font-medium text-gray-900 dark:text-white">{{ section.label }}</div>
            </div>
            <pre class="max-h-[24rem] overflow-auto bg-white px-4 py-3 text-xs leading-6 text-gray-700 dark:bg-dark-800 dark:text-gray-200">{{ formatDetailSection(section.value) }}</pre>
          </div>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="ruleDialogVisible"
      :title="editingRule ? t('admin.traces.rules.edit') : t('admin.traces.rules.create')"
      width="wide"
      @close="closeRuleDialog"
    >
      <form id="trace-rule-form" class="space-y-4" @submit.prevent="saveRule">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('common.name') }}</span>
            <input v-model="ruleForm.name" type="text" class="input w-full" :placeholder="t('admin.traces.rules.namePlaceholder')" />
          </label>
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.priorityLabel') }}</span>
            <input v-model="ruleForm.priority" type="number" class="input w-full" />
          </label>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.samplingRatioLabel') }}</span>
            <input v-model="ruleForm.sampling_ratio" type="number" min="0.01" max="1" step="0.01" class="input w-full" />
          </label>
          <label class="inline-flex items-center gap-2 pt-8 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="ruleForm.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('common.enabled') }}</span>
          </label>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.modelPatterns') }}</span>
            <textarea v-model="ruleForm.model_patterns" rows="3" class="input w-full" :placeholder="t('admin.traces.rules.modelPatternsPlaceholder')"></textarea>
          </label>
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.keywords') }}</span>
            <textarea v-model="ruleForm.keywords" rows="3" class="input w-full" :placeholder="t('admin.traces.rules.keywordsPlaceholder')"></textarea>
          </label>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.userIds') }}</span>
            <textarea v-model="ruleForm.user_ids" rows="3" class="input w-full" :placeholder="t('admin.traces.rules.userIdsPlaceholder')"></textarea>
          </label>
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.apiKeyIds') }}</span>
            <textarea v-model="ruleForm.api_key_ids" rows="3" class="input w-full" :placeholder="t('admin.traces.rules.apiKeyIdsPlaceholder')"></textarea>
          </label>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.minTokensLabel') }}</span>
            <input v-model="ruleForm.min_tokens" type="number" min="0" class="input w-full" placeholder="0" />
          </label>
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.maxTokensLabel') }}</span>
            <input v-model="ruleForm.max_tokens" type="number" min="0" class="input w-full" placeholder="12000" />
          </label>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.activeFromLabel') }}</span>
            <input v-model="ruleForm.active_from" type="datetime-local" class="input w-full" />
          </label>
          <label class="space-y-1 text-sm">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.traces.rules.activeToLabel') }}</span>
            <input v-model="ruleForm.active_to" type="datetime-local" class="input w-full" />
          </label>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeRuleDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="trace-rule-form" class="btn btn-primary" :disabled="ruleSaving" data-testid="trace-rule-save">
            {{ ruleSaving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="confirmVisible"
      :title="confirmTitle"
      :message="confirmMessage"
      :confirm-text="confirmButtonText"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="runConfirmAction"
      @cancel="closeConfirmDialog"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime, formatRelativeTime } from '@/utils/format'
import type {
  CreateTraceRuleRequest,
  TraceExportTask,
  TraceRecord,
  TraceRecordFilters,
  TraceRule,
} from '@/types'
import type { Column } from '@/components/common/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'

type TraceTabKey = 'records' | 'rules' | 'exports'

interface TraceFilterFormState {
  model: string
  keyword: string
  user_id: string
  api_key_id: string
  capture_rule_id: string
  start_date: string
  end_date: string
  timezone: string
  min_input_tokens: string
  max_input_tokens: string
  min_output_tokens: string
  max_output_tokens: string
  min_total_tokens: string
  max_total_tokens: string
}

interface TraceRuleFormState {
  name: string
  enabled: boolean
  priority: string
  model_patterns: string
  user_ids: string
  api_key_ids: string
  keywords: string
  min_tokens: string
  max_tokens: string
  sampling_ratio: string
  active_from: string
  active_to: string
}

interface DetailSection {
  key: string
  label: string
  value: unknown
}

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const DEFAULT_TRACE_EXPORT_TARGET_RECORDS = 500

const activeTab = ref<TraceTabKey>('records')
const showAdvancedRecordFilters = ref(false)
const showAdvancedExportFilters = ref(false)

const records = ref<TraceRecord[]>([])
const recordsLoading = ref(false)
const recordsPagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 0,
})
const selectedRecordIds = ref<Set<number>>(new Set())
const recordDetail = ref<TraceRecord | null>(null)
const recordDetailVisible = ref(false)
const recordDetailLoading = ref(false)
const detailMode = ref<'structured' | 'raw'>('structured')

const rules = ref<TraceRule[]>([])
const rulesLoading = ref(false)
const ruleDialogVisible = ref(false)
const ruleSaving = ref(false)
const editingRule = ref<TraceRule | null>(null)

const exportTasks = ref<TraceExportTask[]>([])
const exportTasksLoading = ref(false)
const exportTasksLoaded = ref(false)
const exportCreating = ref(false)
const includeRawOnExport = ref(true)
const exportTargetRecords = ref<string | number | null>(String(DEFAULT_TRACE_EXPORT_TARGET_RECORDS))
const exportTasksPagination = reactive({
  page: 1,
  page_size: 10,
  total: 0,
  pages: 0,
})
const exportAccessDenied = ref(false)
const exportAccessMessage = ref('')
const exportDownloadTaskId = ref<number | null>(null)
const exportAutoDownloadTaskId = ref<number | null>(null)

const confirmVisible = ref(false)
const confirmTitle = ref('')
const confirmMessage = ref('')
const confirmButtonText = ref('')
const confirmAction = ref<null | (() => Promise<void>)>(null)

const recordFilters = reactive<TraceFilterFormState>(createTraceFilterForm())
const exportFilters = reactive<TraceFilterFormState>(createTraceFilterForm())
const ruleForm = reactive<TraceRuleFormState>(createTraceRuleForm())

let exportTaskPoller: ReturnType<typeof setInterval> | null = null

const isRootAdmin = computed(() => authStore.user?.is_root_admin === true)
const selectedRecordCount = computed(() => selectedRecordIds.value.size)
const allVisibleSelected = computed(
  () => records.value.length > 0 && records.value.every((record) => selectedRecordIds.value.has(record.id))
)
const sortedRules = computed(() =>
  [...rules.value].sort((left, right) => {
    if (left.priority !== right.priority) {
      return left.priority - right.priority
    }
    return right.id - left.id
  })
)
const hasRunningExportTasks = computed(() =>
  exportTasks.value.some((task) => task.status === 'pending' || task.status === 'running')
)
const exportRestricted = computed(() => !isRootAdmin.value || exportAccessDenied.value)
const exportRestrictedMessage = computed(() =>
  exportAccessMessage.value || t('admin.traces.access.rootOnlyDescription')
)
const recordDetailDialogTitle = computed(() =>
  recordDetail.value
    ? t('admin.traces.detail.title', { id: recordDetail.value.id })
    : t('admin.traces.detail.title', { id: '-' })
)

const traceTabs = computed(() => [
  {
    key: 'records' as const,
    icon: 'database' as const,
    label: t('admin.traces.tabs.records'),
    description: t('admin.traces.tabs.recordsDescription'),
  },
  {
    key: 'rules' as const,
    icon: 'shield' as const,
    label: t('admin.traces.tabs.rules'),
    description: t('admin.traces.tabs.rulesDescription'),
  },
  {
    key: 'exports' as const,
    icon: 'download' as const,
    label: t('admin.traces.tabs.exports'),
    description: t('admin.traces.tabs.exportsDescription'),
  },
])

const recordColumns = computed<Column[]>(() => [
  { key: 'select', label: '', sortable: false },
  { key: 'id', label: t('admin.traces.records.columns.id'), sortable: true },
  { key: 'model', label: t('admin.traces.records.columns.model'), sortable: true },
  { key: 'actors', label: t('admin.traces.records.columns.actors'), sortable: false },
  { key: 'tokens', label: t('admin.traces.records.columns.tokens'), sortable: false },
  { key: 'status', label: t('admin.traces.records.columns.status'), sortable: false },
  { key: 'created_at', label: t('admin.traces.records.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false },
])

const ruleColumns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.traces.rules.columns.name'), sortable: true },
  { key: 'targets', label: t('admin.traces.rules.columns.targets'), sortable: false },
  { key: 'conditions', label: t('admin.traces.rules.columns.conditions'), sortable: false },
  { key: 'sampling_ratio', label: t('admin.traces.rules.columns.sampling'), sortable: true },
  { key: 'updated_at', label: t('admin.traces.rules.columns.updatedAt'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false },
])

const exportTaskColumns = computed<Column[]>(() => [
  { key: 'id', label: t('admin.traces.export.columns.id'), sortable: true },
  { key: 'status', label: t('admin.traces.export.columns.status'), sortable: true },
  { key: 'filters', label: t('admin.traces.export.columns.filters'), sortable: false },
  { key: 'progress', label: t('admin.traces.export.columns.progress'), sortable: false },
  { key: 'file', label: t('admin.traces.export.columns.file'), sortable: false },
  { key: 'created_at', label: t('admin.traces.export.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false },
])

const recordDetailSummary = computed(() => {
  if (!recordDetail.value) {
    return []
  }

  const record = recordDetail.value
  return [
    { label: t('admin.traces.detail.fields.recordId'), value: `#${record.id}` },
    { label: t('admin.traces.detail.fields.taskId'), value: record.task_id || '-' },
    { label: t('admin.traces.detail.fields.requestId'), value: record.request_id || '-' },
    { label: t('admin.traces.detail.fields.responseId'), value: record.response_id || '-' },
    { label: t('admin.traces.detail.fields.model'), value: record.model || '-' },
    { label: t('admin.traces.detail.fields.protocol'), value: record.protocol || '-' },
    { label: t('admin.traces.detail.fields.userId'), value: formatOptionalId(record.user_id) },
    { label: t('admin.traces.detail.fields.apiKeyId'), value: formatOptionalId(record.api_key_id) },
    { label: t('admin.traces.detail.fields.accountId'), value: formatOptionalId(record.account_id) },
    { label: t('admin.traces.detail.fields.ruleId'), value: formatOptionalId(record.capture_rule_id) },
    { label: t('admin.traces.detail.fields.totalTokens'), value: formatOptionalNumber(record.total_tokens) },
    { label: t('admin.traces.detail.fields.upstreamStatus'), value: formatTraceStatus(record.upstream_status_code) },
    { label: t('admin.traces.detail.fields.createdAt'), value: formatDate(record.created_at) },
    { label: t('admin.traces.detail.fields.requestContentType'), value: record.request_content_type || '-' },
    { label: t('admin.traces.detail.fields.responseContentType'), value: record.response_content_type || '-' },
    { label: t('admin.traces.detail.fields.dedupeHash'), value: record.dedupe_hash || '-' },
  ]
})

const detailStructuredSections = computed<DetailSection[]>(() => [
  { key: 'prompt', label: t('admin.traces.detail.sections.prompt'), value: recordDetail.value?.prompt },
  { key: 'candidates', label: t('admin.traces.detail.sections.candidates'), value: recordDetail.value?.candidates },
  { key: 'tools', label: t('admin.traces.detail.sections.tools'), value: recordDetail.value?.tools },
  { key: 'signature', label: t('admin.traces.detail.sections.signature'), value: recordDetail.value?.signature },
  { key: 'meta', label: t('admin.traces.detail.sections.meta'), value: recordDetail.value?.meta },
])

const detailRawSections = computed<DetailSection[]>(() => [
  { key: 'raw_request', label: t('admin.traces.detail.sections.rawRequest'), value: recordDetail.value?.raw_request },
  { key: 'raw_response', label: t('admin.traces.detail.sections.rawResponse'), value: recordDetail.value?.raw_response },
  { key: 'raw_request_text', label: t('admin.traces.detail.sections.rawRequestText'), value: recordDetail.value?.raw_request_text },
  { key: 'raw_response_text', label: t('admin.traces.detail.sections.rawResponseText'), value: recordDetail.value?.raw_response_text },
])

const visibleDetailSections = computed(() =>
  (detailMode.value === 'structured' ? detailStructuredSections.value : detailRawSections.value)
    .filter((section) => hasDetailContent(section.value))
)

function getUserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

function createTraceFilterForm(): TraceFilterFormState {
  return {
    model: '',
    keyword: '',
    user_id: '',
    api_key_id: '',
    capture_rule_id: '',
    start_date: '',
    end_date: '',
    timezone: getUserTimezone(),
    min_input_tokens: '',
    max_input_tokens: '',
    min_output_tokens: '',
    max_output_tokens: '',
    min_total_tokens: '',
    max_total_tokens: '',
  }
}

function createTraceRuleForm(): TraceRuleFormState {
  return {
    name: '',
    enabled: true,
    priority: '100',
    model_patterns: '',
    user_ids: '',
    api_key_ids: '',
    keywords: '',
    min_tokens: '',
    max_tokens: '',
    sampling_ratio: '1',
    active_from: '',
    active_to: '',
  }
}

function showError(error: unknown, fallbackKey: string) {
  appStore.showError(extractApiErrorMessage(error, t(fallbackKey)))
}

function trimFormValue(value: unknown): string {
  return String(value ?? '').trim()
}

function parsePositiveIntegerField(value: unknown, labelKey: string): number | undefined {
  const trimmed = trimFormValue(value)
  if (!trimmed) {
    return undefined
  }
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    throw new Error(t('admin.traces.validation.invalidPositiveInteger', { field: t(labelKey) }))
  }
  return parsed
}

function parseNonNegativeIntegerField(value: unknown, labelKey: string): number | undefined {
  const trimmed = trimFormValue(value)
  if (!trimmed) {
    return undefined
  }
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed < 0) {
    throw new Error(t('admin.traces.validation.invalidNonNegativeInteger', { field: t(labelKey) }))
  }
  return parsed
}

function parseCommaList(value: unknown): string[] {
  const trimmed = trimFormValue(value)
  return Array.from(
    new Set(
      trimmed
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter((item) => item.length > 0)
    )
  )
}

function parseIdList(value: unknown, labelKey: string): number[] {
  const trimmed = trimFormValue(value)
  if (!trimmed) {
    return []
  }
  return Array.from(
    new Set(
      trimmed
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter((item) => item.length > 0)
        .map((item) => {
          const parsed = Number(item)
          if (!Number.isInteger(parsed) || parsed <= 0) {
            throw new Error(t('admin.traces.validation.invalidIdList', { field: t(labelKey) }))
          }
          return parsed
        })
    )
  )
}

function toDateTimeLocal(value: string | null | undefined): string {
  if (!value) {
    return ''
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

function fromDateTimeLocal(value: unknown): string | null {
  const trimmed = trimFormValue(value)
  if (!trimmed) {
    return null
  }
  const date = new Date(trimmed)
  if (Number.isNaN(date.getTime())) {
    return null
  }
  return date.toISOString()
}

function buildTraceFilters(form: TraceFilterFormState): TraceRecordFilters {
  const filters: TraceRecordFilters = {}
  const model = trimFormValue(form.model)
  const keyword = trimFormValue(form.keyword)
  const startDate = trimFormValue(form.start_date)
  const endDate = trimFormValue(form.end_date)
  const timezone = trimFormValue(form.timezone) || getUserTimezone()

  if (model) filters.model = model
  if (keyword) filters.keyword = keyword
  if (startDate) filters.start_date = startDate
  if (endDate) filters.end_date = endDate
  if (startDate || endDate) filters.timezone = timezone

  filters.user_id = parsePositiveIntegerField(form.user_id, 'admin.traces.filters.userId') ?? null
  filters.api_key_id = parsePositiveIntegerField(form.api_key_id, 'admin.traces.filters.apiKeyId') ?? null
  filters.capture_rule_id = parsePositiveIntegerField(form.capture_rule_id, 'admin.traces.filters.ruleId') ?? null
  filters.min_input_tokens = parseNonNegativeIntegerField(form.min_input_tokens, 'admin.traces.filters.minInputTokens') ?? null
  filters.max_input_tokens = parseNonNegativeIntegerField(form.max_input_tokens, 'admin.traces.filters.maxInputTokens') ?? null
  filters.min_output_tokens = parseNonNegativeIntegerField(form.min_output_tokens, 'admin.traces.filters.minOutputTokens') ?? null
  filters.max_output_tokens = parseNonNegativeIntegerField(form.max_output_tokens, 'admin.traces.filters.maxOutputTokens') ?? null
  filters.min_total_tokens = parseNonNegativeIntegerField(form.min_total_tokens, 'admin.traces.filters.minTotalTokens') ?? null
  filters.max_total_tokens = parseNonNegativeIntegerField(form.max_total_tokens, 'admin.traces.filters.maxTotalTokens') ?? null

  if (
    filters.min_total_tokens != null
    && filters.max_total_tokens != null
    && filters.min_total_tokens > filters.max_total_tokens
  ) {
    throw new Error(t('admin.traces.validation.invalidRange', { field: t('admin.traces.filters.totalTokens') }))
  }
  if (
    filters.min_input_tokens != null
    && filters.max_input_tokens != null
    && filters.min_input_tokens > filters.max_input_tokens
  ) {
    throw new Error(t('admin.traces.validation.invalidRange', { field: t('admin.traces.filters.inputTokens') }))
  }
  if (
    filters.min_output_tokens != null
    && filters.max_output_tokens != null
    && filters.min_output_tokens > filters.max_output_tokens
  ) {
    throw new Error(t('admin.traces.validation.invalidRange', { field: t('admin.traces.filters.outputTokens') }))
  }

  return filters
}

function buildRulePayload(): CreateTraceRuleRequest {
  const name = trimFormValue(ruleForm.name)
  if (!name) {
    throw new Error(t('admin.traces.validation.requiredField', { field: t('common.name') }))
  }

  const priority = Number(trimFormValue(ruleForm.priority) || '0')
  if (!Number.isInteger(priority)) {
    throw new Error(t('admin.traces.validation.invalidInteger', { field: t('admin.traces.rules.priorityLabel') }))
  }

  const samplingRatio = Number(trimFormValue(ruleForm.sampling_ratio) || '1')
  if (!Number.isFinite(samplingRatio) || samplingRatio <= 0 || samplingRatio > 1) {
    throw new Error(t('admin.traces.validation.invalidSamplingRatio'))
  }

  const minTokens = parseNonNegativeIntegerField(ruleForm.min_tokens, 'admin.traces.rules.minTokensLabel')
  const maxTokens = parseNonNegativeIntegerField(ruleForm.max_tokens, 'admin.traces.rules.maxTokensLabel')
  if (minTokens != null && maxTokens != null && minTokens > maxTokens) {
    throw new Error(t('admin.traces.validation.invalidRange', { field: t('admin.traces.rules.tokenRange') }))
  }

  const activeFrom = fromDateTimeLocal(ruleForm.active_from)
  const activeTo = fromDateTimeLocal(ruleForm.active_to)
  if (activeFrom && activeTo && new Date(activeFrom).getTime() > new Date(activeTo).getTime()) {
    throw new Error(t('admin.traces.validation.invalidWindowRange'))
  }

  return {
    name,
    enabled: ruleForm.enabled,
    priority,
    model_patterns: parseCommaList(ruleForm.model_patterns),
    user_ids: parseIdList(ruleForm.user_ids, 'admin.traces.rules.userIds'),
    api_key_ids: parseIdList(ruleForm.api_key_ids, 'admin.traces.rules.apiKeyIds'),
    keywords: parseCommaList(ruleForm.keywords),
    min_tokens: minTokens ?? null,
    max_tokens: maxTokens ?? null,
    sampling_ratio: samplingRatio,
    active_from: activeFrom,
    active_to: activeTo,
  }
}

async function loadRecords() {
  recordsLoading.value = true
  try {
    const filters = buildTraceFilters(recordFilters)
    const response = await adminAPI.traces.listRecords({
      page: recordsPagination.page,
      page_size: recordsPagination.page_size,
      ...filters,
    })
    records.value = response.items
    recordsPagination.total = response.total
    recordsPagination.page = response.page
    recordsPagination.page_size = response.page_size
    recordsPagination.pages = response.pages
    syncSelectedRecords()
  } catch (error) {
    showError(error, 'admin.traces.records.loadFailed')
  } finally {
    recordsLoading.value = false
  }
}

async function loadRules() {
  rulesLoading.value = true
  try {
    rules.value = await adminAPI.traces.listRules()
  } catch (error) {
    showError(error, 'admin.traces.rules.loadFailed')
  } finally {
    rulesLoading.value = false
  }
}

async function loadExportTasks() {
  if (exportRestricted.value) {
    exportTasks.value = []
    exportTasksPagination.total = 0
    exportTasksPagination.pages = 0
    stopExportTaskPolling()
    return
  }

  exportTasksLoading.value = true
  try {
    const response = await adminAPI.traces.listExportTasks({
      page: exportTasksPagination.page,
      page_size: exportTasksPagination.page_size,
    })
    exportTasks.value = response.items
    exportTasksPagination.total = response.total
    exportTasksPagination.page = response.page
    exportTasksPagination.page_size = response.page_size
    exportTasksPagination.pages = response.pages
    exportTasksLoaded.value = true
    exportAccessDenied.value = false
    exportAccessMessage.value = ''
    syncExportTaskPolling()
    await tryAutoDownloadExportTask()
  } catch (error: any) {
    if (isRootAdminPermissionError(error)) {
      exportAccessDenied.value = true
      exportAccessMessage.value = extractApiErrorMessage(error, t('admin.traces.access.rootOnlyDescription'))
      appStore.showError(t('admin.traces.access.rootOnlyToast'))
      exportTasks.value = []
      exportTasksPagination.total = 0
      exportTasksPagination.pages = 0
      stopExportTaskPolling()
      return
    }
    showError(error, 'admin.traces.export.loadFailed')
  } finally {
    exportTasksLoading.value = false
  }
}

function syncSelectedRecords() {
  const currentIds = new Set(records.value.map((record) => record.id))
  selectedRecordIds.value = new Set(
    [...selectedRecordIds.value].filter((id) => currentIds.has(id))
  )
}

function toggleRecordSelection(id: number) {
  const next = new Set(selectedRecordIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selectedRecordIds.value = next
}

function toggleSelectAllVisible(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  const next = new Set(selectedRecordIds.value)
  for (const record of records.value) {
    if (checked) {
      next.add(record.id)
    } else {
      next.delete(record.id)
    }
  }
  selectedRecordIds.value = next
}

function clearSelectedRecords() {
  selectedRecordIds.value = new Set()
}

function applyRecordFilters() {
  recordsPagination.page = 1
  void loadRecords()
}

function resetRecordFilters() {
  Object.assign(recordFilters, createTraceFilterForm())
  recordsPagination.page = 1
  showAdvancedRecordFilters.value = false
  void loadRecords()
}

function copyRecordFiltersToExport() {
  Object.assign(exportFilters, { ...recordFilters })
  exportFilters.timezone = trimFormValue(recordFilters.timezone) || getUserTimezone()
  includeRawOnExport.value = true
  activeTab.value = 'exports'
}

function handleRecordPageChange(page: number) {
  recordsPagination.page = page
  void loadRecords()
}

function handleRecordPageSizeChange(pageSize: number) {
  recordsPagination.page = 1
  recordsPagination.page_size = pageSize
  void loadRecords()
}

function handleExportPageChange(page: number) {
  exportTasksPagination.page = page
  void loadExportTasks()
}

function handleExportPageSizeChange(pageSize: number) {
  exportTasksPagination.page = 1
  exportTasksPagination.page_size = pageSize
  void loadExportTasks()
}

async function openRecordDetail(recordId: number) {
  recordDetailVisible.value = true
  detailMode.value = 'structured'
  recordDetailLoading.value = true
  try {
    recordDetail.value = await adminAPI.traces.getRecord(recordId)
  } catch (error) {
    recordDetail.value = null
    showError(error, 'admin.traces.records.detailLoadFailed')
  } finally {
    recordDetailLoading.value = false
  }
}

function closeRecordDetail() {
  recordDetailVisible.value = false
  recordDetail.value = null
}

function openCreateRuleDialog() {
  editingRule.value = null
  Object.assign(ruleForm, createTraceRuleForm())
  ruleDialogVisible.value = true
}

function openEditRuleDialog(rule: TraceRule) {
  editingRule.value = rule
  Object.assign(ruleForm, {
    name: rule.name,
    enabled: rule.enabled,
    priority: String(rule.priority),
    model_patterns: rule.model_patterns.join(', '),
    user_ids: rule.user_ids.join(', '),
    api_key_ids: rule.api_key_ids.join(', '),
    keywords: rule.keywords.join(', '),
    min_tokens: rule.min_tokens != null ? String(rule.min_tokens) : '',
    max_tokens: rule.max_tokens != null ? String(rule.max_tokens) : '',
    sampling_ratio: String(rule.sampling_ratio ?? 1),
    active_from: toDateTimeLocal(rule.active_from),
    active_to: toDateTimeLocal(rule.active_to),
  })
  ruleDialogVisible.value = true
}

function closeRuleDialog() {
  ruleDialogVisible.value = false
  editingRule.value = null
  Object.assign(ruleForm, createTraceRuleForm())
}

async function saveRule() {
  ruleSaving.value = true
  try {
    const payload = buildRulePayload()
    if (editingRule.value) {
      await adminAPI.traces.updateRule(editingRule.value.id, payload)
      appStore.showSuccess(t('admin.traces.rules.updated'))
    } else {
      await adminAPI.traces.createRule(payload)
      appStore.showSuccess(t('admin.traces.rules.created'))
    }
    closeRuleDialog()
    await loadRules()
  } catch (error) {
    showError(error, 'admin.traces.rules.saveFailed')
  } finally {
    ruleSaving.value = false
  }
}

function openConfirmDialog(
  title: string,
  message: string,
  confirmText: string,
  action: () => Promise<void>
) {
  confirmTitle.value = title
  confirmMessage.value = message
  confirmButtonText.value = confirmText
  confirmAction.value = action
  confirmVisible.value = true
}

function closeConfirmDialog() {
  confirmVisible.value = false
  confirmAction.value = null
}

async function runConfirmAction() {
  const action = confirmAction.value
  closeConfirmDialog()
  if (!action) {
    return
  }
  await action()
}

function confirmDeleteRecord(record: TraceRecord) {
  openConfirmDialog(
    t('admin.traces.records.deleteTitle'),
    t('admin.traces.records.deleteConfirm', { id: record.id, model: record.model || '-' }),
    t('common.delete'),
    async () => {
      try {
        await adminAPI.traces.deleteRecord(record.id)
        appStore.showSuccess(t('admin.traces.records.deleted'))
        if (records.value.length === 1 && recordsPagination.page > 1) {
          recordsPagination.page -= 1
        }
        clearSelectedRecords()
        await loadRecords()
      } catch (error) {
        showError(error, 'admin.traces.records.deleteFailed')
      }
    }
  )
}

function confirmDeleteSelectedRecords() {
  if (selectedRecordCount.value === 0) {
    appStore.showWarning(t('admin.traces.records.noSelection'))
    return
  }
  openConfirmDialog(
    t('admin.traces.records.deleteSelectedTitle'),
    t('admin.traces.records.deleteSelectedConfirm', { count: selectedRecordCount.value }),
    t('common.delete'),
    async () => {
      try {
        const ids = [...selectedRecordIds.value]
        await adminAPI.traces.batchDeleteRecords(ids)
        appStore.showSuccess(t('admin.traces.records.batchDeleted', { count: ids.length }))
        if (records.value.length <= ids.length && recordsPagination.page > 1) {
          recordsPagination.page -= 1
        }
        clearSelectedRecords()
        await loadRecords()
      } catch (error) {
        showError(error, 'admin.traces.records.batchDeleteFailed')
      }
    }
  )
}

function confirmDeleteRule(rule: TraceRule) {
  openConfirmDialog(
    t('admin.traces.rules.deleteTitle'),
    t('admin.traces.rules.deleteConfirm', { name: rule.name }),
    t('common.delete'),
    async () => {
      try {
        await adminAPI.traces.deleteRule(rule.id)
        appStore.showSuccess(t('admin.traces.rules.deleted'))
        await loadRules()
      } catch (error) {
        showError(error, 'admin.traces.rules.deleteFailed')
      }
    }
  )
}

async function submitExportTask() {
  if (exportRestricted.value) {
    appStore.showWarning(t('admin.traces.access.rootOnlyToast'))
    return
  }

  exportCreating.value = true
  try {
    const filters = buildTraceFilters(exportFilters)
    const timezone = trimFormValue(exportFilters.timezone) || getUserTimezone()
    const targetRecords = parsePositiveIntegerField(
      exportTargetRecords.value,
      'admin.traces.export.targetRecords'
    ) ?? DEFAULT_TRACE_EXPORT_TARGET_RECORDS
    const task = await adminAPI.traces.createExportTask({
      filters: {
        ...filters,
        timezone,
      },
      include_raw: includeRawOnExport.value,
      target_records: targetRecords,
    })
    exportAutoDownloadTaskId.value = task.id
    appStore.showSuccess(t('admin.traces.export.created'))
    exportTasksPagination.page = 1
    await loadExportTasks()
  } catch (error: any) {
    if (isRootAdminPermissionError(error)) {
      exportAccessDenied.value = true
      exportAccessMessage.value = extractApiErrorMessage(error, t('admin.traces.access.rootOnlyDescription'))
      appStore.showError(t('admin.traces.access.rootOnlyToast'))
      stopExportTaskPolling()
      return
    }
    showError(error, 'admin.traces.export.createFailed')
  } finally {
    exportCreating.value = false
  }
}

async function tryAutoDownloadExportTask() {
  const taskId = exportAutoDownloadTaskId.value
  if (!taskId) {
    return
  }
  const task = exportTasks.value.find((item) => item.id === taskId)
  if (!task) {
    return
  }
  if (task.status === 'succeeded') {
    exportAutoDownloadTaskId.value = null
    await downloadExportTask(task)
    return
  }
  if (task.status === 'failed' || task.status === 'canceled') {
    exportAutoDownloadTaskId.value = null
    const message = task.error_message || t(`admin.traces.export.status.${task.status}`)
    appStore.showError(message)
  }
}

function confirmCancelExportTask(task: TraceExportTask) {
  openConfirmDialog(
    t('admin.traces.export.cancelTitle'),
    t('admin.traces.export.cancelConfirm', { id: task.id }),
    t('admin.traces.export.cancelTask'),
    async () => {
      try {
        await adminAPI.traces.cancelExportTask(task.id)
        appStore.showSuccess(t('admin.traces.export.canceled'))
        await loadExportTasks()
      } catch (error) {
        showError(error, 'admin.traces.export.cancelFailed')
      }
    }
  )
}

async function downloadExportTask(task: TraceExportTask) {
  exportDownloadTaskId.value = task.id
  try {
    const result = await adminAPI.traces.downloadExportTask(task.id)
    const filename = result.filename || task.download_filename || `trace-export-${task.id}.json`
    const url = window.URL.createObjectURL(result.blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    showError(error, 'admin.traces.export.downloadFailed')
  } finally {
    exportDownloadTaskId.value = null
  }
}

function canCancelExportTask(task: TraceExportTask): boolean {
  return task.status === 'pending' || task.status === 'running'
}

function isRootAdminPermissionError(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false
  }
  const status = (error as { status?: number }).status
  const message = String((error as { message?: string }).message || '').toLowerCase()
  return status === 403 && message.includes('root admin')
}

function stopExportTaskPolling() {
  if (exportTaskPoller) {
    clearInterval(exportTaskPoller)
    exportTaskPoller = null
  }
}

function syncExportTaskPolling() {
  if (activeTab.value !== 'exports' || exportRestricted.value || !hasRunningExportTasks.value) {
    stopExportTaskPolling()
    return
  }
  if (exportTaskPoller) {
    return
  }
  exportTaskPoller = setInterval(() => {
    void loadExportTasks()
  }, 4000)
}

function formatDate(value: string | null | undefined): string {
  if (!value) {
    return '-'
  }
  return formatDateTime(value) || value
}

function formatRelative(value: string | null | undefined): string {
  return value ? formatRelativeTime(value) : '-'
}

function formatOptionalNumber(value: number | null | undefined): string {
  return typeof value === 'number' && Number.isFinite(value) ? value.toLocaleString() : '-'
}

function formatOptionalId(value: number | null | undefined): string {
  return typeof value === 'number' && Number.isFinite(value) ? String(value) : '-'
}

function formatList(values: string[]): string {
  return values.length > 0 ? values.join(', ') : '-'
}

function formatIdList(values: number[]): string {
  return values.length > 0 ? values.join(', ') : '-'
}

function formatRuleTokenRange(rule: TraceRule): string {
  const min = rule.min_tokens != null ? rule.min_tokens.toLocaleString() : '-'
  const max = rule.max_tokens != null ? rule.max_tokens.toLocaleString() : '-'
  if (rule.min_tokens == null && rule.max_tokens == null) {
    return t('admin.traces.rules.unbounded')
  }
  return `${min} ~ ${max}`
}

function formatRuleWindow(rule: TraceRule): string {
  if (!rule.active_from && !rule.active_to) {
    return t('admin.traces.rules.alwaysOn')
  }
  return `${formatDate(rule.active_from)} → ${formatDate(rule.active_to)}`
}

function formatPercent(value: number): string {
  return `${Math.round(Math.min(Math.max(value * 100, 0), 100))}%`
}

function formatTraceStatus(statusCode: number | null | undefined): string {
  if (typeof statusCode !== 'number' || !Number.isFinite(statusCode)) {
    return t('admin.traces.records.statusUnknown')
  }
  return `HTTP ${statusCode}`
}

function traceStatusClass(statusCode: number | null | undefined): string {
  if (typeof statusCode !== 'number' || !Number.isFinite(statusCode)) {
    return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
  if (statusCode >= 500) {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
  if (statusCode >= 400) {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  }
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
}

function exportTaskStatusLabel(status: string): string {
  const key = `admin.traces.export.status.${status}`
  const translated = t(key)
  return translated !== key ? translated : status
}

function exportTaskStatusClass(status: string): string {
  switch (status) {
    case 'succeeded':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 'running':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case 'canceled':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

function exportTaskProgress(task: TraceExportTask): number {
  const total = exportTaskProgressTotal(task)
  if (!total || total <= 0) {
    return task.status === 'succeeded' ? 100 : 0
  }
  return Math.min(Math.max((task.processed_records / total) * 100, 0), 100)
}

function exportTaskProgressTotal(task: TraceExportTask): number {
  return task.target_records || task.total_records || 0
}

function summarizeTaskFilters(filters: TraceRecordFilters): string[] {
  const lines: string[] = []
  if (filters.model) {
    lines.push(`${t('admin.traces.filters.model')}: ${filters.model}`)
  }
  if (filters.keyword) {
    lines.push(`${t('admin.traces.filters.keyword')}: ${filters.keyword}`)
  }
  if (filters.user_id) {
    lines.push(`${t('admin.traces.filters.userId')}: ${filters.user_id}`)
  }
  if (filters.api_key_id) {
    lines.push(`${t('admin.traces.filters.apiKeyId')}: ${filters.api_key_id}`)
  }
  if (filters.capture_rule_id) {
    lines.push(`${t('admin.traces.filters.ruleId')}: ${filters.capture_rule_id}`)
  }
  if (filters.start_time || filters.end_time) {
    lines.push(`${t('admin.traces.rules.activeWindow')}: ${filters.start_time || '-'} → ${filters.end_time || '-'}`)
  } else if (filters.start_date || filters.end_date) {
    lines.push(`${t('admin.traces.rules.activeWindow')}: ${filters.start_date || '-'} → ${filters.end_date || '-'}`)
  }
  if (filters.min_total_tokens != null || filters.max_total_tokens != null) {
    lines.push(
      `${t('admin.traces.filters.totalTokens')}: ${filters.min_total_tokens ?? '-'} ~ ${filters.max_total_tokens ?? '-'}`
    )
  }
  if (lines.length === 0) {
    lines.push(t('admin.traces.export.noFilters'))
  }
  return lines
}

function formatFileSize(bytes: number): string {
  if (!bytes || bytes <= 0) {
    return '-'
  }
  return formatBytes(bytes)
}

function hasDetailContent(value: unknown): boolean {
  if (value == null) {
    return false
  }
  if (typeof value === 'string') {
    return value.trim().length > 0
  }
  if (Array.isArray(value)) {
    return value.length > 0
  }
  if (typeof value === 'object') {
    return Object.keys(value as Record<string, unknown>).length > 0
  }
  return true
}

function formatDetailSection(value: unknown): string {
  if (!hasDetailContent(value)) {
    return t('admin.traces.detail.noData')
  }
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return t('admin.traces.detail.noData')
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2)
    } catch {
      return value
    }
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

watch(
  activeTab,
  (tab) => {
    if (tab === 'exports' && !exportTasksLoaded.value && !exportRestricted.value) {
      void loadExportTasks()
      return
    }
    if (tab !== 'exports') {
      stopExportTaskPolling()
      return
    }
    syncExportTaskPolling()
  }
)

watch(
  () => hasRunningExportTasks.value,
  () => {
    syncExportTaskPolling()
  }
)

watch(
  isRootAdmin,
  (value) => {
    if (!value) {
      stopExportTaskPolling()
      exportTasks.value = []
      exportTasksPagination.total = 0
      exportTasksPagination.pages = 0
      exportAccessDenied.value = false
      exportAccessMessage.value = ''
      return
    }
    if (activeTab.value === 'exports') {
      void loadExportTasks()
    }
  }
)

onMounted(async () => {
  await Promise.all([loadRecords(), loadRules()])
  if (activeTab.value === 'exports' && !exportRestricted.value) {
    await loadExportTasks()
  }
})

onBeforeUnmount(() => {
  stopExportTaskPolling()
})
</script>
