<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import WorkspaceNavIcon from '../components/WorkspaceNavIcon.vue'
import { api, getErrorMessage, previewMode } from '../core/api'
import { pushNotification } from '../core/notifications'
import { interfacePreferences } from '../core/preferences'

interface AdminStats {
  total_users?: number
  active_users?: number
  total_api_keys?: number
  total_accounts?: number
  active_accounts?: number
  today_requests?: number
  today_tokens?: number
  today_actual_cost?: number
  average_duration_ms?: number
  rpm?: number
}

interface Snapshot {
  generated_at?: string
  stats?: AdminStats
}

const { t } = useI18n()
const loading = ref(false)
const error = ref('')
const snapshot = ref<Snapshot | null>(null)
const stats = computed<AdminStats>(() => snapshot.value?.stats ?? {})

function compact(value: unknown) {
  const n = Number(value || 0)
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toLocaleString(interfacePreferences.locale)
}

function money(value: unknown) {
  return `$${Number(value || 0).toFixed(2)}`
}

function duration(value: unknown) {
  const n = Number(value || 0)
  return n >= 1000 ? `${(n / 1000).toFixed(2)}s` : `${Math.round(n)}ms`
}

function percent(part: unknown, total: unknown) {
  const denominator = Number(total || 0)
  if (!denominator) return 0
  return Math.max(0, Math.min(100, Math.round((Number(part || 0) / denominator) * 100)))
}

const activeUserRate = computed(() => percent(stats.value.active_users, stats.value.total_users))
const accountAvailabilityRate = computed(() => percent(stats.value.active_accounts, stats.value.total_accounts))
const generatedLabel = computed(() => {
  const value = snapshot.value?.generated_at
  if (!value) return t('admin.waitingFirstSync')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('admin.justNow')
  return t('admin.updatedAt', {
    time: date.toLocaleTimeString(interfacePreferences.locale, { hour: '2-digit', minute: '2-digit' }),
  })
})

async function load() {
  error.value = ''
  loading.value = true
  try {
    if (previewMode) {
      snapshot.value = {
        generated_at: new Date().toISOString(),
        stats: {
          total_users: 1284,
          active_users: 438,
          total_api_keys: 2168,
          total_accounts: 42,
          active_accounts: 39,
          today_requests: 68420,
          today_tokens: 18400000,
          today_actual_cost: 428.36,
          average_duration_ms: 846,
          rpm: 176,
        },
      }
    } else {
      snapshot.value = (await api.get<Snapshot>('/admin/dashboard/snapshot-v2', {
        params: { include_stats: true },
      })).data
    }
  } catch (caught) {
    error.value = getErrorMessage(caught)
    pushNotification({
      title: t('admin.refreshFailedTitle'),
      message: t('admin.refreshFailedMessage'),
      tone: 'error',
    })
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <section class="workspace-page admin-overview">
    <header class="page-heading admin-page-heading">
      <div class="admin-heading-copy">
        <div class="admin-heading-eyebrow"><i></i><span>{{ t('admin.eyebrow') }}</span></div>
        <h1>{{ t('admin.title') }}</h1>
        <p>{{ t('admin.description') }}</p>
      </div>

      <div class="admin-utility-cluster">
        <div class="admin-snapshot-meta" :class="{ ready: snapshot }">
          <span class="admin-snapshot-icon"><WorkspaceNavIcon name="activity" /></span>
          <span class="admin-snapshot-copy">
            <strong>{{ snapshot ? t('admin.dataSynced') : t('admin.waitingData') }}</strong>
            <small>{{ generatedLabel }}</small>
          </span>
        </div>
        <span class="admin-utility-divider" aria-hidden="true"></span>
        <button class="admin-refresh" type="button" :disabled="loading" @click="load">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M20 6v5h-5M4 18v-5h5" />
            <path d="M6.1 9a7 7 0 0 1 11.5-2.4L20 11M4 13l2.4 4.4A7 7 0 0 0 18 15" />
          </svg>
          <span>{{ loading ? t('workspace.refreshing') : t('admin.refreshData') }}</span>
        </button>
      </div>
    </header>

    <p v-if="error" class="inline-error">{{ error }}</p>

    <div class="admin-summary-grid">
      <RouterLink to="/admin/users" class="glass metric-card admin-metric-card admin-metric-users">
        <div class="admin-metric-head">
          <span>{{ t('admin.activeUsers') }}</span>
          <span class="admin-metric-actions">
            <span class="admin-metric-icon"><WorkspaceNavIcon name="users" /></span>
            <span class="admin-metric-arrow" aria-hidden="true"><WorkspaceNavIcon name="arrow-up-right" /></span>
          </span>
        </div>
        <strong>{{ compact(stats.active_users) }}</strong>
        <footer>
          <span>{{ t('admin.totalUsers', { value: compact(stats.total_users) }) }}</span>
          <b>{{ t('admin.activeRate', { value: activeUserRate }) }}</b>
        </footer>
        <div class="admin-metric-progress" aria-hidden="true"><i :style="{ width: `${activeUserRate}%` }"></i></div>
      </RouterLink>

      <RouterLink to="/admin/accounts" class="glass metric-card admin-metric-card admin-metric-upstream">
        <div class="admin-metric-head">
          <span>{{ t('admin.upstreamAccounts') }}</span>
          <span class="admin-metric-actions">
            <span class="admin-metric-icon"><WorkspaceNavIcon name="server" /></span>
            <span class="admin-metric-arrow" aria-hidden="true"><WorkspaceNavIcon name="arrow-up-right" /></span>
          </span>
        </div>
        <strong>{{ stats.active_accounts || 0 }} / {{ stats.total_accounts || 0 }}</strong>
        <footer>
          <span>{{ t('admin.accountsAvailable', { value: stats.active_accounts || 0 }) }}</span>
          <b>{{ t('admin.availability', { value: accountAvailabilityRate }) }}</b>
        </footer>
        <div class="admin-metric-progress" aria-hidden="true"><i :style="{ width: `${accountAvailabilityRate}%` }"></i></div>
      </RouterLink>

      <RouterLink to="/admin/usage" class="glass metric-card admin-metric-card admin-metric-requests">
        <div class="admin-metric-head">
          <span class="admin-live-label"><i></i>{{ t('admin.todayRequests') }}</span>
          <span class="admin-metric-actions">
            <span class="admin-metric-icon"><WorkspaceNavIcon name="activity" /></span>
            <span class="admin-metric-arrow" aria-hidden="true"><WorkspaceNavIcon name="arrow-up-right" /></span>
          </span>
        </div>
        <strong>{{ compact(stats.today_requests) }}</strong>
        <footer>
          <span>{{ t('admin.realtimeLoad') }}</span>
          <b>RPM {{ compact(stats.rpm) }}</b>
        </footer>
        <div class="admin-request-context">
          <span><small>{{ t('admin.todayToken') }}</small><b>{{ compact(stats.today_tokens) }}</b></span>
          <span><small>{{ t('admin.averageResponse') }}</small><b>{{ duration(stats.average_duration_ms) }}</b></span>
        </div>
      </RouterLink>
    </div>

    <section class="glass admin-ops-panel">
      <header class="admin-ops-head">
        <div>
          <span class="admin-ops-kicker">{{ t('admin.operationsOverview') }}</span>
          <strong>{{ t('admin.resourcePerformance') }}</strong>
        </div>
        <div class="admin-ops-state" :class="{ ready: snapshot }"><i></i>{{ snapshot ? t('admin.snapshotSynced') : t('admin.waitingSync') }}</div>
      </header>

      <div class="admin-health-grid">
        <div class="admin-health-stat">
          <span class="admin-health-icon"><WorkspaceNavIcon name="chart" /></span>
          <div><span>{{ t('admin.todayToken') }}</span><strong>{{ compact(stats.today_tokens) }}</strong><small>TODAY</small></div>
        </div>
        <div class="admin-health-stat">
          <span class="admin-health-icon"><WorkspaceNavIcon name="wallet" /></span>
          <div><span>{{ t('admin.todayCost') }}</span><strong>{{ money(stats.today_actual_cost) }}</strong><small>USD</small></div>
        </div>
        <div class="admin-health-stat">
          <span class="admin-health-icon"><WorkspaceNavIcon name="activity" /></span>
          <div><span>{{ t('admin.averageResponse') }}</span><strong>{{ duration(stats.average_duration_ms) }}</strong><small>LATENCY</small></div>
        </div>
        <div class="admin-health-stat">
          <span class="admin-health-icon"><WorkspaceNavIcon name="key" /></span>
          <div><span>API Keys</span><strong>{{ compact(stats.total_api_keys) }}</strong><small>TOTAL</small></div>
        </div>
      </div>
    </section>

    <section class="admin-management-panel">
      <header class="admin-management-head">
        <div><span>{{ t('admin.managementEntry') }}</span><strong>{{ t('admin.managementHint') }}</strong></div>
        <RouterLink to="/admin/ops">{{ t('admin.viewOperations') }} <b>→</b></RouterLink>
      </header>
      <div class="admin-management-links">
        <RouterLink to="/admin/accounts">
          <span class="admin-management-icon"><WorkspaceNavIcon name="server" /></span>
          <span class="admin-management-copy"><small>UPSTREAM</small><strong>{{ t('admin.upstreamScheduling') }}</strong><b>{{ t('admin.availableInline', { active: stats.active_accounts || 0, total: stats.total_accounts || 0 }) }}</b></span>
          <span class="admin-management-meta">{{ accountAvailabilityRate }}%</span>
          <i>→</i>
        </RouterLink>
        <RouterLink to="/admin/ops">
          <span class="admin-management-icon"><WorkspaceNavIcon name="activity" /></span>
          <span class="admin-management-copy"><small>OPERATIONS</small><strong>{{ t('admin.operationsMonitor') }}</strong><b>{{ t('admin.averageResponseInline', { value: duration(stats.average_duration_ms) }) }}</b></span>
          <span class="admin-management-meta">RPM {{ compact(stats.rpm) }}</span>
          <i>→</i>
        </RouterLink>
        <RouterLink to="/admin/usage">
          <span class="admin-management-icon"><WorkspaceNavIcon name="chart" /></span>
          <span class="admin-management-copy"><small>USAGE</small><strong>{{ t('admin.usageRecords') }}</strong><b>{{ compact(stats.today_tokens) }} Token</b></span>
          <span class="admin-management-meta">{{ money(stats.today_actual_cost) }}</span>
          <i>→</i>
        </RouterLink>
      </div>
    </section>
  </section>
</template>

<style scoped>
.admin-overview {
  width: 100%;
  max-width: 1280px;
  margin: 0 auto;
  padding-bottom: 34px;
}

.admin-page-heading {
  min-height: auto;
  margin-bottom: 30px;
  align-items: flex-end;
}

.admin-heading-copy { min-width: 0; }

.admin-heading-eyebrow {
  margin-bottom: 13px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #66717e;
  font: 700 .66rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .11em;
}

.admin-heading-eyebrow i,
.admin-live-label i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #42ce99;
  box-shadow: 0 0 0 4px rgba(66, 206, 153, .07);
}

.admin-page-heading h1 {
  margin-top: 0;
  color: #f6f7f9;
  font-size: 2.3rem;
  font-weight: 700;
  line-height: 1.04;
  letter-spacing: -.045em;
}

.admin-page-heading p {
  margin-top: 10px;
  color: #78818d;
  font-size: .88rem;
}

.admin-utility-cluster {
  min-height: 46px;
  padding: 4px;
  border: 1px solid #242932;
  border-radius: 11px;
  display: flex;
  align-items: center;
  gap: 4px;
  background: #0d0f13;
  box-shadow: inset 0 1px rgba(255,255,255,.018);
}

.admin-snapshot-meta {
  min-height: 36px;
  padding: 0 9px 0 5px;
  display: flex;
  align-items: center;
  gap: 9px;
}

.admin-snapshot-icon {
  width: 28px;
  height: 28px;
  border: 1px solid #28313a;
  border-radius: 7px;
  display: grid;
  place-items: center;
  color: #74808c;
  background: #12151a;
}

.admin-snapshot-meta.ready .admin-snapshot-icon {
  border-color: #234236;
  color: #57c99c;
  background: #0e1915;
}

.admin-snapshot-icon :deep(.workspace-nav-icon) { width: 14px; height: 14px; }

.admin-snapshot-copy { display: flex; flex-direction: column; gap: 2px; }
.admin-snapshot-copy strong { color: #b8bec6; font-size: .7rem; font-weight: 620; }
.admin-snapshot-copy small { color: #5f6873; font-size: .62rem; }

.admin-utility-divider { width: 1px; height: 26px; background: #252a31; }

.admin-refresh {
  min-height: 36px;
  padding: 0 10px;
  border: 0;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  background: transparent;
  color: #a4abb5;
  cursor: pointer;
  font: inherit;
  font-size: .73rem;
  transition: color .15s ease, background-color .15s ease;
}

.admin-refresh:hover:not(:disabled) { color: #eef1f4; background: #15181d; }
.admin-refresh:disabled { color: #66707a; cursor: wait; }
.admin-refresh svg { width: 15px; height: 15px; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
.admin-refresh:disabled svg { animation: admin-refresh-spin .9s linear infinite; }
@keyframes admin-refresh-spin { to { transform: rotate(360deg); } }

.admin-summary-grid {
  display: grid;
  grid-template-columns: minmax(0, .92fr) minmax(0, .92fr) minmax(0, 1.16fr);
  gap: 14px;
}

.admin-metric-card {
  --metric-accent: #4da9eb;
  position: relative;
  isolation: isolate;
  min-height: 174px;
  padding: 22px 23px 18px;
  overflow: hidden;
  border-color: #222730;
  border-radius: 13px;
  display: flex;
  flex-direction: column;
  color: inherit;
  background: #0f1115;
  transition: border-color .16s ease, background-color .16s ease;
}

.admin-metric-card:hover { border-color: #313842; background: #11141a; }
.admin-metric-card::before { content: ''; position: absolute; inset: 0 auto auto 0; z-index: -1; width: 46%; height: 1px; background: linear-gradient(90deg, var(--metric-accent), transparent); opacity: .72; }
.admin-metric-card::after { content: ''; position: absolute; z-index: -1; top: -52px; right: -42px; width: 150px; height: 150px; border-radius: 50%; background: radial-gradient(circle, color-mix(in srgb, var(--metric-accent) 10%, transparent), transparent 68%); pointer-events: none; }
.admin-metric-upstream { --metric-accent: #49c89a; }
.admin-metric-requests { --metric-accent: #789df4; min-height: 188px; background: #0f1218; }

.admin-metric-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; }
.admin-metric-head > span:first-child { color: #8b939e; font-size: .8rem; font-weight: 590; }
.admin-live-label { display: inline-flex; align-items: center; gap: 8px; }
.admin-live-label i { width: 5px; height: 5px; box-shadow: none; }
.admin-metric-actions { display: inline-flex; align-items: center; gap: 8px; }
.admin-metric-icon { width: 34px; height: 34px; flex: 0 0 34px; border: 1px solid color-mix(in srgb, var(--metric-accent) 28%, #252a31); border-radius: 9px; display: grid; place-items: center; color: var(--metric-accent); background: color-mix(in srgb, var(--metric-accent) 7%, #111318); }
.admin-metric-icon :deep(.workspace-nav-icon) { width: 17px; height: 17px; }
.admin-metric-arrow {
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  border: 1px solid transparent;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #67727e;
  background: transparent;
  transition: color .15s ease, background-color .15s ease, border-color .15s ease;
}
.admin-metric-arrow :deep(.workspace-nav-icon) {
  width: 14px;
  height: 14px;
  transition: transform .15s ease;
}
.admin-metric-card:hover .admin-metric-arrow {
  border-color: #2c3540;
  color: #c2d4e2;
  background: #171c23;
}
.admin-metric-card:hover .admin-metric-arrow :deep(.workspace-nav-icon) { transform: translate(1px, -1px); }

.admin-metric-card > strong { margin-top: 17px; color: #f6f7f9; font-size: 2.02rem; font-weight: 690; line-height: 1; letter-spacing: -.04em; }
.admin-metric-requests > strong { font-size: 2.28rem; }

.admin-metric-card footer { margin-top: auto; padding-top: 17px; display: flex; align-items: center; justify-content: space-between; gap: 12px; color: #66707b; font-size: .71rem; }
.admin-metric-card footer b { padding: 4px 7px; border: 1px solid #262c34; border-radius: 999px; color: #88939f; background: #11141a; font-size: .64rem; font-weight: 600; white-space: nowrap; }

.admin-metric-progress { height: 3px; margin-top: 13px; overflow: hidden; border-radius: 999px; background: #1b2027; }
.admin-metric-progress i { display: block; height: 100%; border-radius: inherit; background: var(--metric-accent); opacity: .72; transition: width .35s ease; }

.admin-request-context { margin-top: 13px; padding-top: 12px; border-top: 1px solid #232932; display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.admin-request-context span { min-width: 0; display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
.admin-request-context small { color: #5d6672; font-size: .62rem; }
.admin-request-context b { color: #aeb8c4; font-size: .69rem; font-weight: 620; white-space: nowrap; }

.admin-ops-panel { position: relative; margin-top: 14px; padding: 0; overflow: hidden; border-color: #222730; border-radius: 13px; background: #0d0f13; }
.admin-ops-panel::after { content: ''; position: absolute; inset: 0; pointer-events: none; background-image: linear-gradient(rgba(255,255,255,.014) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.014) 1px, transparent 1px); background-size: 32px 32px; mask-image: linear-gradient(to bottom, rgba(0,0,0,.55), transparent 72%); }
.admin-ops-head { position: relative; z-index: 1; min-height: 72px; padding: 0 22px; border-bottom: 1px solid #22272e; display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.admin-ops-head > div:first-child { display: flex; flex-direction: column; gap: 5px; }
.admin-ops-kicker { color: #68727e; font-size: .67rem; font-weight: 650; }
.admin-ops-head strong { color: #dce0e5; font-size: .84rem; font-weight: 620; }
.admin-ops-state { min-height: 28px; padding: 0 9px; border: 1px solid #2a2f37; border-radius: 999px; display: inline-flex; align-items: center; gap: 7px; color: #7b848f; background: #111318; font-size: .66rem; font-weight: 570; }
.admin-ops-state i { width: 6px; height: 6px; border-radius: 50%; background: #69727d; }
.admin-ops-state.ready { border-color: #244337; color: #79c8a9; background: #0f1815; }
.admin-ops-state.ready i { background: #42ce99; }

.admin-health-grid { position: relative; z-index: 1; min-height: 142px; display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); }
.admin-health-stat { min-width: 0; padding: 24px 22px; display: grid; grid-template-columns: 38px minmax(0, 1fr); align-items: center; gap: 14px; }
.admin-health-stat + .admin-health-stat { border-left: 1px solid #232830; }
.admin-health-icon { width: 38px; height: 38px; border: 1px solid #28303a; border-radius: 9px; display: grid; place-items: center; color: #72869a; background: #11151a; }
.admin-health-icon :deep(.workspace-nav-icon) { width: 17px; height: 17px; }
.admin-health-stat > div { min-width: 0; display: flex; flex-direction: column; }
.admin-health-stat > div > span { color: #77818c; font-size: .71rem; font-weight: 560; }
.admin-health-stat strong { margin-top: 6px; color: #edf0f3; font-size: 1.22rem; font-weight: 650; line-height: 1.05; letter-spacing: -.025em; }
.admin-health-stat small { margin-top: 7px; color: #4f5863; font: 650 .56rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .1em; }

.admin-management-panel { margin-top: 14px; overflow: hidden; border: 1px solid #222730; border-radius: 13px; background: #0d0f13; }
.admin-management-head { min-height: 66px; padding: 0 20px 0 22px; border-bottom: 1px solid #22272e; display: flex; align-items: center; justify-content: space-between; gap: 20px; }
.admin-management-head > div { display: flex; flex-direction: column; gap: 4px; }
.admin-management-head > div span { color: #68727e; font-size: .66rem; font-weight: 650; }
.admin-management-head > div strong { color: #cfd4da; font-size: .79rem; font-weight: 600; }
.admin-management-head > a { display: inline-flex; align-items: center; gap: 7px; color: #6f7d8b; font-size: .68rem; transition: color .15s ease; }
.admin-management-head > a:hover { color: #cbd9e5; }
.admin-management-head > a b { font-size: .8rem; font-weight: 500; }

.admin-management-links { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); }
.admin-management-links > a { min-width: 0; min-height: 112px; padding: 19px 20px; display: grid; grid-template-columns: 38px minmax(0, 1fr) auto 18px; align-items: center; gap: 12px; color: inherit; background: transparent; transition: background-color .15s ease; }
.admin-management-links > a + a { border-left: 1px solid #232830; }
.admin-management-links > a:hover { background: #11141a; }
.admin-management-icon { width: 38px; height: 38px; border: 1px solid #29313a; border-radius: 9px; display: grid; place-items: center; color: #74899c; background: #11151a; }
.admin-management-icon :deep(.workspace-nav-icon) { width: 17px; height: 17px; }
.admin-management-copy { min-width: 0; display: flex; flex-direction: column; }
.admin-management-copy small { color: #4f5964; font: 650 .54rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .09em; }
.admin-management-copy strong { margin-top: 6px; color: #dce0e5; font-size: .79rem; font-weight: 620; }
.admin-management-copy b { margin-top: 4px; overflow: hidden; color: #69737e; font-size: .65rem; font-weight: 520; text-overflow: ellipsis; white-space: nowrap; }
.admin-management-meta { color: #8c99a6; font-size: .67rem; font-weight: 600; white-space: nowrap; }
.admin-management-links > a > i { color: #4d5864; font-style: normal; font-size: .9rem; transition: color .15s ease, transform .15s ease; }
.admin-management-links > a:hover > i { color: #aabfce; transform: translateX(2px); }

@media (max-width: 1120px) {
  .admin-page-heading { align-items: flex-start; }
  .admin-summary-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .admin-metric-requests > strong { font-size: 2.02rem; }
  .admin-management-links > a { grid-template-columns: 38px minmax(0, 1fr) 18px; }
  .admin-management-meta { display: none; }
}

@media (max-width: 920px) {
  .admin-page-heading { flex-direction: column; gap: 18px; }
  .admin-utility-cluster { width: 100%; }
  .admin-snapshot-meta { flex: 1; }
  .admin-summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .admin-metric-requests { grid-column: 1 / -1; }
  .admin-health-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .admin-health-stat:nth-child(odd) { border-left: 0; }
  .admin-health-stat:nth-child(n + 3) { border-top: 1px solid #232830; }
  .admin-management-links { grid-template-columns: 1fr; }
  .admin-management-links > a + a { border-left: 0; border-top: 1px solid #232830; }
  .admin-management-links > a { min-height: 92px; grid-template-columns: 38px minmax(0, 1fr) auto 18px; }
  .admin-management-meta { display: inline; }
}

@media (max-width: 620px) {
  .admin-page-heading h1 { font-size: 1.9rem; }
  .admin-utility-cluster { align-items: stretch; flex-direction: column; padding: 6px; }
  .admin-utility-divider { width: 100%; height: 1px; }
  .admin-refresh { justify-content: center; }
  .admin-summary-grid { grid-template-columns: 1fr; }
  .admin-metric-requests { grid-column: auto; }
  .admin-request-context { grid-template-columns: 1fr; gap: 7px; }
  .admin-ops-head { min-height: 86px; align-items: flex-start; flex-direction: column; justify-content: center; gap: 10px; }
  .admin-health-grid { grid-template-columns: 1fr; }
  .admin-health-stat, .admin-health-stat:nth-child(odd) { border-left: 0; }
  .admin-health-stat + .admin-health-stat, .admin-health-stat:nth-child(n + 3) { border-top: 1px solid #232830; }
  .admin-management-head { min-height: 80px; align-items: flex-start; flex-direction: column; justify-content: center; gap: 8px; }
  .admin-management-links > a { grid-template-columns: 38px minmax(0, 1fr) 18px; }
  .admin-management-meta { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  .admin-refresh:disabled svg { animation: none; }
  .admin-metric-progress i, .admin-metric-arrow, .admin-metric-arrow :deep(.workspace-nav-icon), .admin-management-links > a > i { transition: none; }
}
</style>
