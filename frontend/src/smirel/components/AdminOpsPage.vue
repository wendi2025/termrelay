<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import WorkspaceNavIcon from './WorkspaceNavIcon.vue'
import { api, getErrorMessage, previewMode } from '../core/api'
import { interfacePreferences } from '../core/preferences'
import '../styles/admin-ops.css'

interface AdminDashboardStats {
  today_requests?: number
  today_tokens?: number
  today_actual_cost?: number
  average_duration_ms?: number
  rpm?: number
  tpm?: number
  stats_updated_at?: string
  stats_stale?: boolean
}

interface UsageTrendPoint {
  date: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  total_tokens: number
  cost: number
  actual_cost: number
}

interface UsageTrendResponse {
  trend: UsageTrendPoint[]
  start_date: string
  end_date: string
  granularity: string
}

type PeriodDays = 7 | 30 | 90

const { t } = useI18n()
const loading = ref(false)
const error = ref('')
const stats = ref<AdminDashboardStats>({})
const trend = ref<UsageTrendPoint[]>([])
const periodDays = ref<PeriodDays>(7)
const refreshedAt = ref<Date | null>(null)

function compact(value: unknown) {
  const n = Number(value || 0)
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(1)}B`
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toLocaleString(interfacePreferences.locale)
}

function money(value: unknown, digits = 2) {
  return `$${Number(value || 0).toFixed(digits)}`
}

function duration(value: unknown) {
  const n = Number(value || 0)
  return n >= 1000 ? `${(n / 1000).toFixed(2)}s` : `${Math.round(n)}ms`
}

function toDateParam(date: Date) {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function currentRange() {
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - periodDays.value + 1)
  return { start_date: toDateParam(start), end_date: toDateParam(end), granularity: 'day' }
}

function sum(selector: (point: UsageTrendPoint) => number) {
  return trend.value.reduce((total, point) => total + Number(selector(point) || 0), 0)
}

function sparkline(selector: (point: UsageTrendPoint) => number) {
  if (!trend.value.length) return ''
  const values = trend.value.map((point) => Math.max(0, Number(selector(point) || 0)))
  const max = Math.max(...values, 1)
  const width = 320
  const height = 104
  const top = 10
  const bottom = 10
  const drawableHeight = height - top - bottom
  return values.map((value, index) => {
    const x = values.length === 1 ? width / 2 : (index / (values.length - 1)) * width
    const y = top + drawableHeight - (value / max) * drawableHeight
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

const periodRequests = computed(() => sum((point) => point.requests))
const periodTokens = computed(() => sum((point) => point.total_tokens))
const periodCost = computed(() => sum((point) => point.actual_cost))
const averageRequests = computed(() => trend.value.length ? periodRequests.value / trend.value.length : 0)
const averageCostPerRequest = computed(() => periodRequests.value ? periodCost.value / periodRequests.value : 0)
const peakRequests = computed(() => trend.value.length ? Math.max(...trend.value.map((point) => Number(point.requests || 0))) : 0)

const generatedLabel = computed(() => {
  if (!refreshedAt.value) return t('admin.waitingFirstSync')
  return t('admin.updatedAt', {
    time: refreshedAt.value.toLocaleTimeString(interfacePreferences.locale, { hour: '2-digit', minute: '2-digit' }),
  })
})

const rangeLabel = computed(() => {
  if (!trend.value.length) return `最近 ${periodDays.value} 天`
  const first = trend.value[0]?.date?.slice(0, 10)
  const last = trend.value[trend.value.length - 1]?.date?.slice(0, 10)
  return first && last ? `${first} — ${last}` : `最近 ${periodDays.value} 天`
})

const trendCharts = computed(() => [
  {
    key: 'requests',
    eyebrow: 'REQUESTS',
    label: '请求量趋势',
    value: compact(periodRequests.value),
    unit: 'REQUESTS',
    points: sparkline((point) => point.requests),
    peak: `峰值 ${compact(peakRequests.value)} / 日`,
  },
  {
    key: 'tokens',
    eyebrow: 'TOKENS',
    label: 'Token 趋势',
    value: compact(periodTokens.value),
    unit: 'TOKENS',
    points: sparkline((point) => point.total_tokens),
    peak: `日均 ${compact(trend.value.length ? periodTokens.value / trend.value.length : 0)}`,
  },
  {
    key: 'cost',
    eyebrow: 'ACTUAL COST',
    label: '实际成本趋势',
    value: money(periodCost.value),
    unit: 'USD',
    points: sparkline((point) => point.actual_cost),
    peak: `单次均值 ${money(averageCostPerRequest.value, 4)}`,
  },
])

async function load() {
  error.value = ''
  loading.value = true

  try {
    if (previewMode) {
      stats.value = {}
      trend.value = []
      refreshedAt.value = new Date()
      return
    }

    const range = currentRange()
    const [statsResponse, trendResponse] = await Promise.all([
      api.get<AdminDashboardStats>('/admin/dashboard/stats'),
      api.get<UsageTrendResponse>('/admin/dashboard/trend', { params: range }),
    ])

    stats.value = statsResponse.data || {}
    trend.value = Array.isArray(trendResponse.data?.trend) ? trendResponse.data.trend : []
    refreshedAt.value = new Date()
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    loading.value = false
  }
}

function setPeriod(days: PeriodDays) {
  if (periodDays.value === days || loading.value) return
  periodDays.value = days
  void load()
}

onMounted(() => void load())
</script>

<template>
  <div class="admin-ops-workspace">
    <div class="ops-snapshot-bar">
      <div class="ops-snapshot-state" :class="{ ready: refreshedAt && !error }">
        <span class="ops-state-icon"><WorkspaceNavIcon name="activity" /></span>
        <span>
          <strong>{{ stats.stats_stale ? '统计数据正在追赶' : (refreshedAt ? t('admin.dataSynced') : t('admin.waitingData')) }}</strong>
          <small>{{ generatedLabel }}</small>
        </span>
      </div>
      <div class="ops-toolbar">
        <div class="ops-period-switch" aria-label="Trend period">
          <button v-for="days in ([7, 30, 90] as PeriodDays[])" :key="days" type="button" :class="{ active: periodDays === days }" :disabled="loading" @click="setPeriod(days)">
            {{ days }}D
          </button>
        </div>
        <button class="ops-refresh" type="button" :disabled="loading" @click="load">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M20 6v5h-5M4 18v-5h5" />
            <path d="M6.1 9a7 7 0 0 1 11.5-2.4L20 11M4 13l2.4 4.4A7 7 0 0 0 18 15" />
          </svg>
          <span>{{ loading ? t('workspace.refreshing') : t('admin.refreshData') }}</span>
        </button>
      </div>
    </div>

    <p v-if="error" class="inline-error ops-error">{{ error }}</p>

    <section class="ops-metric-strip" aria-label="Operations snapshot">
      <article>
        <span class="ops-metric-label">TODAY REQUESTS</span>
        <div><strong>{{ compact(stats.today_requests) }}</strong><small>REQ</small></div>
        <p>今日累计请求量</p>
      </article>
      <article>
        <span class="ops-metric-label">TODAY TOKENS</span>
        <div><strong>{{ compact(stats.today_tokens) }}</strong><small>TOKENS</small></div>
        <p>今日累计 Token 用量</p>
      </article>
      <article>
        <span class="ops-metric-label">TODAY COST</span>
        <div><strong>{{ money(stats.today_actual_cost) }}</strong><small>USD</small></div>
        <p>今日实际扣除成本</p>
      </article>
      <article>
        <span class="ops-metric-label">REQUEST RATE</span>
        <div><strong>{{ compact(stats.rpm) }}</strong><small>RPM</small></div>
        <p>近 5 分钟平均请求率 · TPM {{ compact(stats.tpm) }}</p>
      </article>
    </section>

    <div class="ops-main-grid">
      <section class="ops-trend-panel">
        <header class="ops-panel-head">
          <div>
            <span>REAL USAGE TREND</span>
            <strong>真实用量趋势</strong>
          </div>
          <small class="ops-range-label">{{ rangeLabel }}</small>
        </header>

        <div v-if="trend.length" class="ops-trend-stack">
          <article v-for="chart in trendCharts" :key="chart.key" class="ops-trend-card">
            <div class="ops-trend-copy">
              <span>{{ chart.eyebrow }}</span>
              <strong>{{ chart.label }}</strong>
              <div><b>{{ chart.value }}</b><small>{{ chart.unit }}</small></div>
              <p>{{ chart.peak }}</p>
            </div>
            <div class="ops-trend-plot" aria-hidden="true">
              <svg viewBox="0 0 320 104" preserveAspectRatio="none">
                <line x1="0" y1="10" x2="320" y2="10" />
                <line x1="0" y1="52" x2="320" y2="52" />
                <line x1="0" y1="94" x2="320" y2="94" />
                <polyline :points="chart.points" />
              </svg>
              <div><span>{{ trend[0]?.date?.slice(5, 10) }}</span><span>{{ trend[trend.length - 1]?.date?.slice(5, 10) }}</span></div>
            </div>
          </article>
        </div>
        <div v-else class="ops-trend-empty">
          <WorkspaceNavIcon name="chart" />
          <strong>当前周期暂无真实用量记录</strong>
          <span>趋势区只展示后端 usage trend 接口返回的数据，不使用模拟曲线。</span>
        </div>
      </section>

      <section class="ops-period-panel">
        <header class="ops-panel-head">
          <div>
            <span>PERIOD DETAIL</span>
            <strong>周期详细指标</strong>
          </div>
          <RouterLink to="/admin/usage">{{ t('admin.usageRecords') }} →</RouterLink>
        </header>

        <div class="ops-period-total">
          <span>最近 {{ periodDays }} 天实际成本</span>
          <strong>{{ money(periodCost) }}</strong>
          <small>{{ rangeLabel }}</small>
        </div>

        <dl class="ops-period-list">
          <div>
            <dt>周期请求量</dt>
            <dd>{{ compact(periodRequests) }}</dd>
          </div>
          <div>
            <dt>周期 Token</dt>
            <dd>{{ compact(periodTokens) }}</dd>
          </div>
          <div>
            <dt>日均请求</dt>
            <dd>{{ compact(averageRequests) }}</dd>
          </div>
          <div>
            <dt>单次平均成本</dt>
            <dd>{{ money(averageCostPerRequest, 4) }}</dd>
          </div>
          <div>
            <dt>请求峰值</dt>
            <dd>{{ compact(peakRequests) }} / 日</dd>
          </div>
          <div>
            <dt>当前平均响应</dt>
            <dd>{{ duration(stats.average_duration_ms) }}</dd>
          </div>
        </dl>
      </section>
    </div>

    <nav class="ops-shortcuts" aria-label="Operations links">
      <RouterLink to="/admin/accounts">
        <span class="ops-shortcut-icon"><WorkspaceNavIcon name="server" /></span>
        <span><small>UPSTREAM</small><strong>{{ t('admin.upstreamScheduling') }}</strong></span>
        <b>→</b>
      </RouterLink>
      <RouterLink to="/admin/channels/pricing">
        <span class="ops-shortcut-icon"><WorkspaceNavIcon name="network" /></span>
        <span><small>CHANNELS</small><strong>{{ t('nav.adminChannels') }}</strong></span>
        <b>→</b>
      </RouterLink>
      <RouterLink to="/admin/usage">
        <span class="ops-shortcut-icon"><WorkspaceNavIcon name="chart" /></span>
        <span><small>USAGE</small><strong>{{ t('admin.usageRecords') }}</strong></span>
        <b>→</b>
      </RouterLink>
    </nav>
  </div>
</template>

<style scoped>
.ops-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ops-period-switch {
  padding: 3px;
  border: 1px solid #252a31;
  border-radius: 8px;
  display: flex;
  gap: 2px;
  background: #101318;
}

.ops-period-switch button {
  min-width: 38px;
  height: 30px;
  padding: 0 9px;
  border: 0;
  border-radius: 6px;
  color: #67717d;
  background: transparent;
  cursor: pointer;
  font: 700 .61rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .06em;
}

.ops-period-switch button:hover:not(:disabled) {
  color: #c7cdd3;
  background: #171b20;
}

.ops-period-switch button.active {
  color: #e7ebee;
  background: #242a31;
}

.ops-period-switch button:disabled {
  cursor: wait;
}

.ops-trend-panel,
.ops-period-panel {
  min-width: 0;
  border: 1px solid var(--ws-border);
  border-radius: 10px;
  overflow: hidden;
  background: #0f1115;
}

.ops-range-label {
  color: #69737e;
  font: 600 .65rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .02em;
}

.ops-trend-stack {
  padding: 0 20px;
}

.ops-trend-card {
  min-height: 145px;
  padding: 18px 0;
  border-bottom: 1px solid #22262d;
  display: grid;
  grid-template-columns: 170px minmax(0, 1fr);
  align-items: center;
  gap: 22px;
}

.ops-trend-card:last-child {
  border-bottom: 0;
}

.ops-trend-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.ops-trend-copy > span {
  color: #59636f;
  font: 700 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .09em;
}

.ops-trend-copy > strong {
  margin-top: 6px;
  color: #cfd4d9;
  font-size: .78rem;
  font-weight: 620;
}

.ops-trend-copy > div {
  margin-top: 13px;
  display: flex;
  align-items: baseline;
  gap: 7px;
}

.ops-trend-copy b {
  color: #f0f2f4;
  font-size: 1.35rem;
  font-weight: 680;
  line-height: 1;
  letter-spacing: -.035em;
}

.ops-trend-copy div small {
  color: #68727d;
  font: 700 .55rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .07em;
}

.ops-trend-copy p {
  margin: 8px 0 0;
  color: #68717c;
  font-size: .65rem;
}

.ops-trend-plot {
  min-width: 0;
}

.ops-trend-plot svg {
  width: 100%;
  height: 92px;
  overflow: visible;
}

.ops-trend-plot line {
  stroke: #22272e;
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}

.ops-trend-plot polyline {
  fill: none;
  stroke: #74b9e5;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}

.ops-trend-card:nth-child(2) .ops-trend-plot polyline {
  stroke: #6fc6a2;
}

.ops-trend-card:nth-child(3) .ops-trend-plot polyline {
  stroke: #c6a878;
}

.ops-trend-plot > div {
  margin-top: 2px;
  display: flex;
  justify-content: space-between;
  color: #555f6b;
  font: 600 .57rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.ops-trend-empty {
  min-height: 435px;
  padding: 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.ops-trend-empty .workspace-nav-icon {
  width: 22px;
  height: 22px;
  color: #5e6873;
}

.ops-trend-empty strong {
  margin-top: 12px;
  color: #c6ccd2;
  font-size: .8rem;
  font-weight: 620;
}

.ops-trend-empty span {
  max-width: 360px;
  margin-top: 7px;
  color: #656f79;
  font-size: .68rem;
  line-height: 1.6;
}

.ops-period-panel {
  display: flex;
  flex-direction: column;
}

.ops-period-total {
  padding: 25px 22px 23px;
  border-bottom: 1px solid #242830;
  display: flex;
  flex-direction: column;
}

.ops-period-total span {
  color: #737c87;
  font-size: .7rem;
}

.ops-period-total strong {
  margin-top: 10px;
  color: #f0f2f4;
  font-size: 2rem;
  font-weight: 690;
  line-height: 1;
  letter-spacing: -.04em;
}

.ops-period-total small {
  margin-top: 9px;
  color: #58626d;
  font: 600 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.ops-period-list {
  margin: 0;
  padding: 0 22px;
}

.ops-period-list > div {
  min-height: 55px;
  border-bottom: 1px solid #22262d;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.ops-period-list > div:last-child {
  border-bottom: 0;
}

.ops-period-list dt {
  color: #6f7883;
  font-size: .69rem;
}

.ops-period-list dd {
  margin: 0;
  color: #d7dce1;
  font-size: .75rem;
  font-weight: 640;
  text-align: right;
}

@media (max-width: 760px) {
  .ops-snapshot-bar {
    align-items: flex-start;
    flex-direction: column;
  }

  .ops-toolbar {
    width: 100%;
    justify-content: space-between;
  }

  .ops-trend-card {
    grid-template-columns: 1fr;
    gap: 14px;
  }
}
</style>
