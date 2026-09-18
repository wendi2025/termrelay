<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import '../styles/user-dashboard.css'
import '../styles/user-dashboard-refresh.css'
import '../styles/user-dashboard-lower.css'

interface DashboardStats {
  total_api_keys?: number
  active_api_keys?: number
  total_requests?: number
  total_tokens?: number
  total_actual_cost?: number
  today_requests?: number
  today_tokens?: number
  today_actual_cost?: number
  [key: string]: unknown
}

const props = defineProps<{
  stats: DashboardStats | null
  balance: number
  loading: boolean
}>()

const emit = defineEmits<{
  refresh: []
}>()

const { t } = useI18n()
const copied = ref(false)
const apiBase = 'https://api.smirel.com/v1'

const activeKeys = computed(() => Number(props.stats?.active_api_keys || 0))
const totalKeys = computed(() => Number(props.stats?.total_api_keys || 0))
const keyRatio = computed(() => totalKeys.value > 0 ? Math.round((activeKeys.value / totalKeys.value) * 100) : 0)

async function copyEndpoint() {
  await navigator.clipboard.writeText(apiBase)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1400)
}
</script>

<template>
  <div class="user-console-page">
    <header class="user-console-heading">
      <div>
        <span class="user-console-eyebrow">ACCOUNT / CONSOLE</span>
        <h1>{{ t('nav.dashboard') }}</h1>
        <p>{{ t('workspace.descriptions.dashboard') }}</p>
      </div>
      <button class="user-console-refresh" type="button" :disabled="loading" @click="emit('refresh')">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M20 11a8 8 0 1 0 2 5.3" />
          <path d="M20 4v7h-7" />
        </svg>
        <span>{{ loading ? t('workspace.refreshing') : t('workspace.refresh') }}</span>
      </button>
    </header>

    <section class="user-console-overview">
      <div class="user-balance-panel">
        <div class="user-panel-label">{{ t('workspace.availableBalance') }}</div>
        <strong class="user-balance-value">${{ balance.toFixed(2) }}</strong>
        <div class="user-key-status">
          <span><i></i>{{ activeKeys }} / {{ totalKeys }} {{ t('nav.keys') }}</span>
          <small>{{ keyRatio }}%</small>
        </div>
        <div class="user-key-progress" aria-hidden="true"><span :style="{ width: `${keyRatio}%` }"></span></div>
        <div class="user-balance-actions">
          <RouterLink to="/purchase" class="user-primary-action">{{ t('workspace.purchase') }}</RouterLink>
          <RouterLink to="/keys" class="user-secondary-action">{{ t('nav.keys') }}</RouterLink>
        </div>
      </div>

      <div class="user-endpoint-panel">
        <div class="user-endpoint-head">
          <div>
            <span>API ENDPOINT</span>
            <strong>{{ t('userDashboard.endpointTitle') }}</strong>
          </div>
          <button type="button" @click="copyEndpoint">{{ copied ? t('userDashboard.copied') : t('userDashboard.copy') }}</button>
        </div>
        <code>{{ apiBase }}</code>
        <div class="user-endpoint-meta">
          <span><i></i>OpenAI Compatible</span>
          <span>Bearer API Key</span>
          <RouterLink to="/model-plaza">{{ t('userDashboard.models') }} →</RouterLink>
        </div>
      </div>
    </section>

    <section class="user-usage-surface">
      <header>
        <div>
          <span>{{ t('userDashboard.today') }}</span>
          <strong>{{ t('userDashboard.usageTitle') }}</strong>
        </div>
        <RouterLink to="/usage">{{ t('userDashboard.viewUsage') }} →</RouterLink>
      </header>

      <div class="user-usage-grid">
        <article>
          <span>{{ t('workspace.todayRequests') }}</span>
          <strong>{{ Number(stats?.today_requests || 0).toLocaleString() }}</strong>
          <small>{{ t('userDashboard.requestsHint') }}</small>
        </article>
        <article>
          <span>{{ t('workspace.todayTokens') }}</span>
          <strong>{{ Number(stats?.today_tokens || 0).toLocaleString() }}</strong>
          <small>{{ t('userDashboard.tokensHint') }}</small>
        </article>
        <article>
          <span>{{ t('workspace.todayCost') }}</span>
          <strong>${{ Number(stats?.today_actual_cost || 0).toFixed(3) }}</strong>
          <small>{{ t('userDashboard.costHint') }}</small>
        </article>
      </div>
    </section>

    <section class="user-console-lower">
      <div class="user-lifetime-panel">
        <header>
          <span>{{ t('userDashboard.lifetime') }}</span>
          <strong>{{ t('userDashboard.lifetimeTitle') }}</strong>
        </header>
        <dl>
          <div><dt>{{ t('userDashboard.totalRequests') }}</dt><dd>{{ Number(stats?.total_requests || 0).toLocaleString() }}</dd></div>
          <div><dt>{{ t('userDashboard.totalTokens') }}</dt><dd>{{ Number(stats?.total_tokens || 0).toLocaleString() }}</dd></div>
          <div><dt>{{ t('userDashboard.totalCost') }}</dt><dd>${{ Number(stats?.total_actual_cost || 0).toFixed(3) }}</dd></div>
        </dl>
      </div>

      <div class="user-quick-panel">
        <header>
          <span>{{ t('userDashboard.quick') }}</span>
          <strong>{{ t('userDashboard.quickTitle') }}</strong>
        </header>
        <nav>
          <RouterLink to="/keys"><span>01</span><strong>{{ t('nav.keys') }}</strong><b>→</b></RouterLink>
          <RouterLink to="/model-plaza"><span>02</span><strong>{{ t('userDashboard.models') }}</strong><b>→</b></RouterLink>
          <RouterLink to="/usage"><span>03</span><strong>{{ t('nav.usage') }}</strong><b>→</b></RouterLink>
        </nav>
      </div>
    </section>
  </div>
</template>
