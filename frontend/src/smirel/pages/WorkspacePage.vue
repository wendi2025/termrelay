<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AccountSettingsPage from '../components/AccountSettingsPage.vue'
import AdminOpsPage from '../components/AdminOpsPage.vue'
import AdminSettingsPage from '../components/AdminSettingsPage.vue'
import AdminUsersPage from '../components/AdminUsersPage.vue'
import ApiKeyCredentialCard from '../components/ApiKeyCredentialCard.vue'
import UserDashboardPage from '../components/UserDashboardPage.vue'
import { api, getErrorMessage, previewMode } from '../core/api'
import { pushNotification } from '../core/notifications'
import { useSession } from '../core/session'
import '../styles/api-keys.css'
import '../styles/admin-users-commercial.css'

interface ApiKeyRow { id: number; name?: string; key?: string; status?: string; created_at?: string; [key: string]: unknown }
interface UsageRow { id?: number; model?: string; endpoint?: string; total_tokens?: number; actual_cost?: number; created_at?: string; [key: string]: unknown }
interface DashboardStats { total_api_keys?: number; active_api_keys?: number; total_requests?: number; total_tokens?: number; total_actual_cost?: number; today_requests?: number; today_tokens?: number; today_actual_cost?: number; [key: string]: unknown }

const route = useRoute()
const { t } = useI18n()
const { state } = useSession()
const feature = computed(() => String(route.meta.feature || 'module'))
const loading = ref(false)
const error = ref('')
const stats = ref<DashboardStats | null>(null)
const keys = ref<ApiKeyRow[]>([])
const usage = ref<UsageRow[]>([])
const newKeyName = ref('')

const isDashboard = computed(() => feature.value === 'dashboard')
const isKeys = computed(() => feature.value === 'keys')
const isUsage = computed(() => feature.value === 'usage')
const isProfile = computed(() => feature.value === 'profile')
const isAdminUsers = computed(() => feature.value === 'admin-users')
const isAdminOps = computed(() => feature.value === 'admin-ops')
const isAdminSettings = computed(() => feature.value === 'admin-settings')
const accountBalance = computed(() => Number(state.user?.balance || 0))
const visibleUsageTokens = computed(() => usage.value.reduce((sum, item) => sum + Number(item.total_tokens || 0), 0))
const visibleUsageCost = computed(() => usage.value.reduce((sum, item) => sum + Number(item.actual_cost || 0), 0))

const featureTitleKeys: Record<string, string> = {
  dashboard: 'nav.dashboard',
  keys: 'nav.keys',
  usage: 'nav.usage',
  subscriptions: 'nav.subscriptions',
  purchase: 'nav.purchase',
  orders: 'nav.orders',
  profile: 'nav.profile',
  'admin-dashboard': 'nav.adminDashboard',
  'admin-users': 'nav.adminUsers',
  'admin-accounts': 'nav.adminAccounts',
  'admin-groups': 'nav.adminGroups',
  'admin-channels': 'nav.adminChannels',
  'admin-usage': 'nav.adminUsage',
  'admin-ops': 'nav.adminOps',
  'admin-payment-dashboard': 'nav.adminPayment',
  'admin-orders': 'nav.adminOrders',
  'admin-settings': 'nav.adminSettings',
}

const featureDescriptionKeys: Record<string, string> = {
  dashboard: 'workspace.descriptions.dashboard',
  keys: 'workspace.descriptions.keys',
  usage: 'workspace.descriptions.usage',
  profile: 'workspace.descriptions.profile',
  'admin-users': 'workspace.descriptions.adminUsers',
  'admin-accounts': 'workspace.descriptions.adminAccounts',
  'admin-groups': 'workspace.descriptions.adminGroups',
  'admin-channels': 'workspace.descriptions.adminChannels',
  'admin-usage': 'workspace.descriptions.adminUsage',
  'admin-ops': 'workspace.descriptions.adminOps',
  'admin-payment-dashboard': 'workspace.descriptions.adminPayment',
  'admin-orders': 'workspace.descriptions.adminOrders',
  'admin-settings': 'workspace.descriptions.adminSettings',
}

const title = computed(() => {
  const key = featureTitleKeys[feature.value]
  return key ? t(key) : String(route.meta.title || 'Workspace')
})

const pageDescription = computed(() => {
  const key = featureDescriptionKeys[feature.value]
  return key ? t(key) : t('workspace.descriptions.generic')
})

async function load() {
  error.value = ''
  if (previewMode) {
    if (isDashboard.value) stats.value = { total_api_keys: 4, active_api_keys: 3, total_requests: 12480, total_tokens: 8294000, total_actual_cost: 18.72, today_requests: 842, today_tokens: 612340, today_actual_cost: 1.94 }
    if (isKeys.value) keys.value = [{ id: 1, name: 'Production', key: 'sk-••••••••9F2A', status: 'active', created_at: '2026-09-01' }, { id: 2, name: 'Development', key: 'sk-••••••••71CD', status: 'active', created_at: '2026-08-28' }]
    if (isUsage.value) usage.value = [{ id: 1, model: 'gpt-5.6', endpoint: '/v1/responses', total_tokens: 18420, actual_cost: 0.082, created_at: '2026-09-06 14:20' }, { id: 2, model: 'claude-sonnet', endpoint: '/v1/messages', total_tokens: 9820, actual_cost: 0.051, created_at: '2026-09-06 14:12' }]
    return
  }
  if (!isDashboard.value && !isKeys.value && !isUsage.value) return
  loading.value = true
  try {
    if (isDashboard.value) stats.value = (await api.get<DashboardStats>('/usage/dashboard/stats')).data
    if (isKeys.value) {
      const data = (await api.get<{ items?: ApiKeyRow[] } | ApiKeyRow[]>('/keys', { params: { page: 1, page_size: 50 } })).data
      keys.value = Array.isArray(data) ? data : (data.items || [])
    }
    if (isUsage.value) {
      const data = (await api.get<{ items?: UsageRow[] } | UsageRow[]>('/usage', { params: { page: 1, page_size: 30 } })).data
      usage.value = Array.isArray(data) ? data : (data.items || [])
    }
  } catch (caught) { error.value = getErrorMessage(caught) } finally { loading.value = false }
}

async function createKey() {
  const keyName = newKeyName.value.trim()
  if (!keyName) return

  if (previewMode) {
    keys.value.unshift({ id: Date.now(), name: keyName, key: 'sk-preview-new', status: 'active', created_at: new Date().toISOString().slice(0, 10) })
    newKeyName.value = ''
    pushNotification({
      title: t('workspace.keyCreatedTitle'),
      message: t('workspace.keyCreatedMessage', { name: keyName }),
      tone: 'success',
    })
    return
  }

  loading.value = true
  try {
    const created = (await api.post<ApiKeyRow>('/keys', { name: keyName })).data
    keys.value.unshift(created)
    newKeyName.value = ''
    pushNotification({
      title: t('workspace.keyCreatedTitle'),
      message: t('workspace.keyCreatedMessage', { name: keyName }),
      tone: 'success',
    })
  } catch (caught) { error.value = getErrorMessage(caught) } finally { loading.value = false }
}

async function removeKey(id: number) {
  if (!window.confirm(t('workspace.confirmDelete'))) return
  if (!previewMode) await api.delete(`/keys/${id}`)
  keys.value = keys.value.filter((item) => item.id !== id)
  pushNotification({
    title: t('workspace.keyDeletedTitle'),
    message: t('workspace.keyDeletedMessage'),
    tone: 'info',
  })
}

watch(() => route.fullPath, () => void load())
onMounted(() => void load())
</script>

<template>
  <section class="workspace-page">
    <AdminUsersPage v-if="isAdminUsers" />

    <template v-else>
      <header v-if="!isDashboard" class="page-heading">
        <div>
          <h1>{{ title }}</h1>
          <p>{{ pageDescription }}</p>
        </div>
        <button v-if="isKeys || isUsage" class="ghost-button" type="button" :disabled="loading" @click="load">{{ loading ? t('workspace.refreshing') : t('workspace.refresh') }}</button>
      </header>

      <p v-if="error" class="inline-error">{{ error }}</p>

      <AdminOpsPage v-if="isAdminOps" />

      <AdminSettingsPage v-else-if="isAdminSettings" />

      <UserDashboardPage
        v-else-if="isDashboard"
        :stats="stats"
        :balance="accountBalance"
        :loading="loading"
        @refresh="load"
      />

      <template v-else-if="isKeys">
        <section class="keys-create-panel">
          <div class="keys-create-copy">
            <span>NEW KEY</span>
            <strong>{{ t('workspace.createKey') }}</strong>
          </div>
          <div class="keys-create-form">
            <input v-model="newKeyName" :aria-label="`${t('workspace.key')} ${t('workspace.name')}`" :placeholder="t('workspace.keyNamePlaceholder')" @keydown.enter="createKey" />
            <button class="primary-button" type="button" :disabled="loading" @click="createKey">{{ t('workspace.createKey') }}</button>
          </div>
        </section>

        <section class="keys-library">
          <header class="keys-library-head">
            <div>
              <strong>API Keys</strong>
              <span class="keys-library-count">{{ keys.length }}</span>
            </div>
          </header>

          <div v-if="keys.length" class="api-key-grid">
            <ApiKeyCredentialCard
              v-for="item in keys"
              :key="item.id"
              :item="item"
              @remove="removeKey"
            />
          </div>

          <p v-else-if="!loading" class="keys-empty-state">{{ t('workspace.noKeys') }}</p>
        </section>
      </template>

      <template v-else-if="isUsage">
        <section class="glass table-toolbar standalone-toolbar">
          <div><strong>{{ t('workspace.recentRequests') }}</strong><span class="table-count">{{ usage.length }}</span></div>
          <div class="usage-total"><span>{{ visibleUsageTokens.toLocaleString() }} Tokens</span><span>${{ visibleUsageCost.toFixed(4) }}</span></div>
        </section>
        <div class="glass data-table usage-table">
          <div class="table-head"><span>{{ t('workspace.time') }}</span><span>{{ t('workspace.model') }}</span><span>{{ t('workspace.endpoint') }}</span><span>{{ t('workspace.token') }}</span><span>{{ t('workspace.cost') }}</span></div>
          <div v-for="(item, index) in usage" :key="item.id || index" class="table-row">
            <span>{{ item.created_at || '—' }}</span>
            <strong>{{ item.model || '—' }}</strong>
            <code>{{ item.endpoint || '—' }}</code>
            <span>{{ Number(item.total_tokens || 0).toLocaleString() }}</span>
            <span>${{ Number(item.actual_cost || 0).toFixed(4) }}</span>
          </div>
          <p v-if="!usage.length && !loading" class="empty-state">{{ t('workspace.noUsage') }}</p>
        </div>
      </template>

      <AccountSettingsPage v-else-if="isProfile" />

      <template v-else>
        <section class="glass module-panel">
          <h2>{{ title }}</h2>
          <p>{{ t('workspace.modulePending') }}</p>
        </section>
      </template>
    </template>
  </section>
</template>
