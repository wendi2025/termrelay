<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import WorkspaceNavIcon from '../components/WorkspaceNavIcon.vue'
import { api, getErrorMessage, previewMode } from '../core/api'

interface UsageRow {
  id?: number
  model?: string
  endpoint?: string
  total_tokens?: number
  actual_cost?: number
  created_at?: string
  [key: string]: unknown
}

type Period = 'today' | '7d' | '30d' | '90d' | 'custom' | 'all'
type Metric = 'tokens' | 'cost'

const loading = ref(false)
const error = ref('')
const usage = ref<UsageRow[]>([])
const search = ref('')
const modelFilter = ref('all')
const period = ref<Period>('7d')
const metric = ref<Metric>('tokens')
const customStart = ref('')
const customEnd = ref('')

const previewRows: UsageRow[] = [
  { id: 1, model: 'gpt-5.6', endpoint: '/v1/responses', total_tokens: 18420, actual_cost: 0.0820, created_at: '2026-09-07 10:18' },
  { id: 2, model: 'claude-sonnet', endpoint: '/v1/messages', total_tokens: 9820, actual_cost: 0.0510, created_at: '2026-09-07 09:42' },
  { id: 3, model: 'gpt-5.6', endpoint: '/v1/responses', total_tokens: 12680, actual_cost: 0.0564, created_at: '2026-09-06 21:54' },
  { id: 4, model: 'gemini-2.5-pro', endpoint: '/v1/chat/completions', total_tokens: 7340, actual_cost: 0.0318, created_at: '2026-09-06 16:31' },
  { id: 5, model: 'claude-sonnet', endpoint: '/v1/messages', total_tokens: 6210, actual_cost: 0.0287, created_at: '2026-09-05 18:48' },
  { id: 6, model: 'gpt-5.6', endpoint: '/v1/responses', total_tokens: 15460, actual_cost: 0.0689, created_at: '2026-09-05 11:26' },
  { id: 7, model: 'gpt-5.6-mini', endpoint: '/v1/chat/completions', total_tokens: 11320, actual_cost: 0.0241, created_at: '2026-09-04 20:08' },
  { id: 8, model: 'claude-sonnet', endpoint: '/v1/messages', total_tokens: 13880, actual_cost: 0.0706, created_at: '2026-09-03 14:22' },
  { id: 9, model: 'gemini-2.5-pro', endpoint: '/v1/chat/completions', total_tokens: 8640, actual_cost: 0.0384, created_at: '2026-09-02 17:05' },
  { id: 10, model: 'gpt-5.6', endpoint: '/v1/responses', total_tokens: 19620, actual_cost: 0.0897, created_at: '2026-09-01 09:37' },
]

function parseTime(value?: string) {
  if (!value) return 0
  const normalized = value.includes('T') ? value : value.replace(' ', 'T')
  const timestamp = new Date(normalized).getTime()
  return Number.isNaN(timestamp) ? 0 : timestamp
}

function startOfDay(value: number) {
  const date = new Date(value)
  date.setHours(0, 0, 0, 0)
  return date.getTime()
}

function endOfDay(value: number) {
  const date = new Date(value)
  date.setHours(23, 59, 59, 999)
  return date.getTime()
}

function inputDate(value: number) {
  const date = new Date(value)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const referenceTime = computed(() => {
  const timestamps = usage.value.map((item) => parseTime(item.created_at)).filter(Boolean)
  return timestamps.length ? Math.max(...timestamps) : Date.now()
})

function selectPeriod(next: Period) {
  period.value = next
  if (next !== 'custom') return

  if (!customEnd.value) customEnd.value = inputDate(referenceTime.value)
  if (!customStart.value) {
    const start = new Date(referenceTime.value)
    start.setDate(start.getDate() - 6)
    customStart.value = inputDate(start.getTime())
  }
}

const selectedBounds = computed(() => {
  const referenceEnd = endOfDay(referenceTime.value)

  if (period.value === 'all') {
    const timestamps = usage.value.map((item) => parseTime(item.created_at)).filter(Boolean)
    if (!timestamps.length) return { start: startOfDay(referenceTime.value), end: referenceEnd }
    return { start: startOfDay(Math.min(...timestamps)), end: endOfDay(Math.max(...timestamps)) }
  }

  if (period.value === 'custom') {
    const startCandidate = customStart.value ? new Date(`${customStart.value}T00:00:00`).getTime() : startOfDay(referenceTime.value)
    const endCandidate = customEnd.value ? new Date(`${customEnd.value}T23:59:59.999`).getTime() : referenceEnd
    const low = Math.min(startCandidate, endCandidate)
    const high = Math.max(startCandidate, endCandidate)
    return { start: startOfDay(low), end: endOfDay(high) }
  }

  const days = period.value === 'today' ? 1 : period.value === '7d' ? 7 : period.value === '30d' ? 30 : 90
  const start = new Date(referenceTime.value)
  start.setDate(start.getDate() - (days - 1))
  return { start: startOfDay(start.getTime()), end: referenceEnd }
})

const periodUsage = computed(() => {
  const { start, end } = selectedBounds.value
  return usage.value.filter((item) => {
    const timestamp = parseTime(item.created_at)
    return timestamp >= start && timestamp <= end
  })
})

const periodLabel = computed(() => {
  if (period.value === 'today') return '今日'
  if (period.value === '7d') return '近 7 天'
  if (period.value === '30d') return '近 30 天'
  if (period.value === '90d') return '近 90 天'
  if (period.value === 'all') return '全部'
  return `${customStart.value || '起始'} — ${customEnd.value || '结束'}`
})

const modelOptions = computed(() => Array.from(new Set(periodUsage.value.map((item) => item.model).filter(Boolean) as string[])))

const filteredUsage = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return periodUsage.value.filter((item) => {
    if (modelFilter.value !== 'all' && item.model !== modelFilter.value) return false
    if (!keyword) return true
    return `${item.model || ''} ${item.endpoint || ''}`.toLowerCase().includes(keyword)
  })
})

const totalRequests = computed(() => periodUsage.value.length)
const totalTokens = computed(() => periodUsage.value.reduce((sum, item) => sum + Number(item.total_tokens || 0), 0))
const totalCost = computed(() => periodUsage.value.reduce((sum, item) => sum + Number(item.actual_cost || 0), 0))
const averageCost = computed(() => totalRequests.value ? totalCost.value / totalRequests.value : 0)
const averageTokens = computed(() => totalRequests.value ? Math.round(totalTokens.value / totalRequests.value) : 0)

function distributionBy(key: 'model' | 'endpoint') {
  const totals = new Map<string, { tokens: number; cost: number; requests: number }>()
  periodUsage.value.forEach((item) => {
    const label = String(item[key] || 'Unknown')
    const current = totals.get(label) || { tokens: 0, cost: 0, requests: 0 }
    current.tokens += Number(item.total_tokens || 0)
    current.cost += Number(item.actual_cost || 0)
    current.requests += 1
    totals.set(label, current)
  })
  const denominator = metric.value === 'tokens' ? Math.max(1, totalTokens.value) : Math.max(0.000001, totalCost.value)
  return Array.from(totals.entries())
    .map(([label, value]) => ({
      label,
      ...value,
      amount: metric.value === 'tokens' ? value.tokens : value.cost,
      percent: Math.round(((metric.value === 'tokens' ? value.tokens : value.cost) / denominator) * 100),
    }))
    .sort((a, b) => b.amount - a.amount)
}

const modelBreakdown = computed(() => distributionBy('model').slice(0, 5))
const endpointBreakdown = computed(() => distributionBy('endpoint').slice(0, 5))

interface TrendBucket {
  label: string
  fullLabel: string
  tokens: number
  cost: number
  requests: number
  value: number
  showLabel: boolean
}

const trendData = computed<TrendBucket[]>(() => {
  const { start, end } = selectedBounds.value

  if (period.value === 'today') {
    const dayStart = startOfDay(end)
    return Array.from({ length: 6 }, (_, index) => {
      const bucketStart = dayStart + index * 4 * 60 * 60 * 1000
      const bucketEnd = Math.min(bucketStart + 4 * 60 * 60 * 1000 - 1, end)
      const items = periodUsage.value.filter((item) => {
        const timestamp = parseTime(item.created_at)
        return timestamp >= bucketStart && timestamp <= bucketEnd
      })
      const tokens = items.reduce((sum, item) => sum + Number(item.total_tokens || 0), 0)
      const cost = items.reduce((sum, item) => sum + Number(item.actual_cost || 0), 0)
      return {
        label: `${String(index * 4).padStart(2, '0')}:00`,
        fullLabel: `${inputDate(bucketStart)} ${String(index * 4).padStart(2, '0')}:00`,
        tokens,
        cost,
        requests: items.length,
        value: metric.value === 'tokens' ? tokens : cost,
        showLabel: true,
      }
    })
  }

  const totalDays = Math.max(1, Math.floor((end - start) / (24 * 60 * 60 * 1000)) + 1)
  const stepDays = Math.max(1, Math.ceil(totalDays / 30))
  const bucketCount = Math.ceil(totalDays / stepDays)
  const labelEvery = Math.max(1, Math.ceil(bucketCount / 7))

  return Array.from({ length: bucketCount }, (_, index) => {
    const bucketStart = new Date(start)
    bucketStart.setDate(bucketStart.getDate() + index * stepDays)
    const bucketEnd = new Date(bucketStart)
    bucketEnd.setDate(bucketEnd.getDate() + stepDays - 1)
    bucketEnd.setHours(23, 59, 59, 999)
    if (bucketEnd.getTime() > end) bucketEnd.setTime(end)

    const items = periodUsage.value.filter((item) => {
      const timestamp = parseTime(item.created_at)
      return timestamp >= bucketStart.getTime() && timestamp <= bucketEnd.getTime()
    })
    const tokens = items.reduce((sum, item) => sum + Number(item.total_tokens || 0), 0)
    const cost = items.reduce((sum, item) => sum + Number(item.actual_cost || 0), 0)
    const label = `${bucketStart.getMonth() + 1}/${bucketStart.getDate()}`

    return {
      label,
      fullLabel: stepDays === 1
        ? inputDate(bucketStart.getTime())
        : `${inputDate(bucketStart.getTime())} — ${inputDate(bucketEnd.getTime())}`,
      tokens,
      cost,
      requests: items.length,
      value: metric.value === 'tokens' ? tokens : cost,
      showLabel: index % labelEvery === 0 || index === bucketCount - 1,
    }
  })
})

const chartWidth = 1000
const chartTop = 16
const chartBottom = 166
const chartLeft = 20
const chartRight = 980

const trendPoints = computed(() => {
  const rows = trendData.value
  const max = Math.max(...rows.map((item) => item.value), 1)
  return rows.map((item, index) => {
    const x = rows.length === 1
      ? chartWidth / 2
      : chartLeft + (index / (rows.length - 1)) * (chartRight - chartLeft)
    const y = chartBottom - (item.value / max) * (chartBottom - chartTop)
    return { ...item, x, y }
  })
})

const trendLinePath = computed(() => {
  const points = trendPoints.value
  if (!points.length) return ''
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`

  let path = `M ${points[0].x} ${points[0].y}`
  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1]
    const current = points[index]
    const middleX = (previous.x + current.x) / 2
    path += ` C ${middleX} ${previous.y}, ${middleX} ${current.y}, ${current.x} ${current.y}`
  }
  return path
})

const trendAreaPath = computed(() => {
  const points = trendPoints.value
  if (points.length < 2) return ''
  return `${trendLinePath.value} L ${points[points.length - 1].x} ${chartBottom} L ${points[0].x} ${chartBottom} Z`
})

const topModel = computed(() => modelBreakdown.value[0]?.label || '—')
const topEndpoint = computed(() => endpointBreakdown.value[0]?.label || '—')

function compact(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return value.toLocaleString()
}

function money(value: number) {
  return `$${value.toFixed(4)}`
}

function formatTime(value?: string) {
  if (!value) return '—'
  const normalized = value.includes('T') ? value : value.replace(' ', 'T')
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}

function trendValue(item: TrendBucket) {
  return metric.value === 'tokens' ? `${compact(item.tokens)} Tokens` : money(item.cost)
}

async function load() {
  error.value = ''
  loading.value = true
  try {
    if (previewMode) {
      usage.value = previewRows
      return
    }
    const data = (await api.get<{ items?: UsageRow[] } | UsageRow[]>('/usage', { params: { page: 1, page_size: 100 } })).data
    usage.value = Array.isArray(data) ? data : (data.items || [])
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <section class="workspace-page user-usage-page">
    <header class="usage-heading">
      <div class="usage-heading-copy">
        <span class="usage-eyebrow"><i></i>USAGE ANALYTICS</span>
        <h1>用量与日志</h1>
        <p>查看 API 请求、Token 与费用趋势，了解主要模型和 Endpoint 的消耗情况。</p>
      </div>
      <div class="usage-heading-actions">
        <div class="period-switch" aria-label="统计范围">
          <button :class="{ active: period === 'today' }" type="button" @click="selectPeriod('today')">今日</button>
          <button :class="{ active: period === '7d' }" type="button" @click="selectPeriod('7d')">7 天</button>
          <button :class="{ active: period === '30d' }" type="button" @click="selectPeriod('30d')">30 天</button>
          <button :class="{ active: period === '90d' }" type="button" @click="selectPeriod('90d')">90 天</button>
          <button :class="{ active: period === 'custom' }" type="button" @click="selectPeriod('custom')">自定义</button>
          <button :class="{ active: period === 'all' }" type="button" @click="selectPeriod('all')">全部</button>
        </div>
        <button class="usage-refresh" type="button" :disabled="loading" aria-label="刷新数据" @click="load">
          <WorkspaceNavIcon name="refresh" />
        </button>
      </div>
    </header>

    <div v-if="period === 'custom'" class="custom-range-row">
      <span>自定义时间</span>
      <label>
        <small>开始</small>
        <input v-model="customStart" type="date" :max="customEnd || undefined" />
      </label>
      <i>—</i>
      <label>
        <small>结束</small>
        <input v-model="customEnd" type="date" :min="customStart || undefined" />
      </label>
    </div>

    <p v-if="error" class="inline-error">{{ error }}</p>

    <div class="usage-summary-grid">
      <article class="usage-summary-card">
        <div class="summary-icon"><WorkspaceNavIcon name="activity" /></div>
        <div class="summary-copy">
          <span>总请求数</span>
          <strong>{{ totalRequests.toLocaleString() }}</strong>
          <small>所选范围内 API 调用</small>
        </div>
      </article>
      <article class="usage-summary-card">
        <div class="summary-icon"><WorkspaceNavIcon name="chart" /></div>
        <div class="summary-copy">
          <span>总 Token</span>
          <strong>{{ compact(totalTokens) }}</strong>
          <small>平均 {{ compact(averageTokens) }} / 请求</small>
        </div>
      </article>
      <article class="usage-summary-card">
        <div class="summary-icon"><WorkspaceNavIcon name="wallet" /></div>
        <div class="summary-copy">
          <span>总费用</span>
          <strong>{{ money(totalCost) }}</strong>
          <small>按实际记录累计</small>
        </div>
      </article>
      <article class="usage-summary-card">
        <div class="summary-icon"><WorkspaceNavIcon name="receipt" /></div>
        <div class="summary-copy">
          <span>平均请求成本</span>
          <strong>{{ money(averageCost) }}</strong>
          <small>每次 API 调用</small>
        </div>
      </article>
    </div>

    <section class="analytics-toolbar">
      <div class="toolbar-context">
        <WorkspaceNavIcon name="chart" />
        <span>用量分析</span>
        <b>{{ totalRequests }} 条记录</b>
      </div>
      <div class="metric-switch" aria-label="统计口径">
        <button :class="{ active: metric === 'tokens' }" type="button" @click="metric = 'tokens'">按 Token</button>
        <button :class="{ active: metric === 'cost' }" type="button" @click="metric = 'cost'">按费用</button>
      </div>
    </section>

    <div class="analytics-grid">
      <article class="analytics-panel trend-panel">
        <header class="panel-heading trend-heading">
          <div>
            <small>TREND</small>
            <h2>{{ metric === 'tokens' ? 'Token 使用趋势' : '费用趋势' }}</h2>
          </div>
          <div class="panel-total">
            <span>{{ periodLabel }}</span>
            <strong>{{ metric === 'tokens' ? compact(totalTokens) : money(totalCost) }}</strong>
          </div>
        </header>

        <div class="trend-chart">
          <div class="trend-grid-lines"><i></i><i></i><i></i><i></i></div>
          <svg
            class="trend-line-svg"
            :viewBox="`0 0 ${chartWidth} 190`"
            preserveAspectRatio="none"
            role="img"
            :aria-label="metric === 'tokens' ? 'Token 使用趋势曲线图' : '费用趋势曲线图'"
          >
            <defs>
              <linearGradient id="usageTrendArea" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="currentColor" stop-opacity=".16" />
                <stop offset="100%" stop-color="currentColor" stop-opacity="0" />
              </linearGradient>
            </defs>
            <path v-if="trendAreaPath" class="trend-area" :d="trendAreaPath" />
            <path v-if="trendLinePath" class="trend-line" :d="trendLinePath" />
            <g v-for="point in trendPoints" :key="point.fullLabel" class="trend-point">
              <circle :cx="point.x" :cy="point.y" r="6">
                <title>{{ point.fullLabel }} · {{ trendValue(point) }} · {{ point.requests }} 次请求</title>
              </circle>
            </g>
          </svg>
          <div class="trend-axis" :style="{ gridTemplateColumns: `repeat(${Math.max(trendData.length, 1)}, minmax(0, 1fr))` }">
            <small
              v-for="item in trendData"
              :key="`axis-${item.fullLabel}`"
              :class="{ muted: !item.showLabel }"
            >{{ item.label }}</small>
          </div>
        </div>
      </article>

      <article class="analytics-panel distribution-panel">
        <header class="panel-heading">
          <div>
            <small>MODEL MIX</small>
            <h2>模型分布</h2>
          </div>
          <span class="panel-caption">{{ metric === 'tokens' ? '按 Token' : '按费用' }}</span>
        </header>
        <div v-if="modelBreakdown.length" class="distribution-list">
          <div v-for="(item, index) in modelBreakdown" :key="item.label" class="distribution-row">
            <div class="distribution-head">
              <span><i>{{ String(index + 1).padStart(2, '0') }}</i><strong>{{ item.label }}</strong></span>
              <b>{{ item.percent }}%</b>
            </div>
            <div class="distribution-track"><i :style="{ width: `${item.percent}%` }"></i></div>
            <small>{{ metric === 'tokens' ? `${compact(item.tokens)} Tokens` : money(item.cost) }} · {{ item.requests }} 次</small>
          </div>
        </div>
        <div v-else class="panel-empty compact"><strong>暂无模型数据</strong></div>
      </article>

      <article class="analytics-panel distribution-panel endpoint-panel">
        <header class="panel-heading">
          <div>
            <small>ENDPOINT</small>
            <h2>Endpoint 分布</h2>
          </div>
          <span class="panel-caption">{{ endpointBreakdown.length }} 个接口</span>
        </header>
        <div v-if="endpointBreakdown.length" class="distribution-list endpoint-list">
          <div v-for="item in endpointBreakdown" :key="item.label" class="distribution-row">
            <div class="distribution-head">
              <code>{{ item.label }}</code>
              <b>{{ item.percent }}%</b>
            </div>
            <div class="distribution-track"><i :style="{ width: `${item.percent}%` }"></i></div>
            <small>{{ item.requests }} 次请求 · {{ metric === 'tokens' ? `${compact(item.tokens)} Tokens` : money(item.cost) }}</small>
          </div>
        </div>
        <div v-else class="panel-empty compact"><strong>暂无 Endpoint 数据</strong></div>
      </article>

      <article class="analytics-panel overview-panel">
        <header class="panel-heading">
          <div>
            <small>PROFILE</small>
            <h2>使用概况</h2>
          </div>
          <span class="panel-caption">当前范围</span>
        </header>
        <dl class="usage-profile-list">
          <div><dt>主要模型</dt><dd>{{ topModel }}</dd></div>
          <div><dt>主要 Endpoint</dt><dd><code>{{ topEndpoint }}</code></dd></div>
          <div><dt>平均 Token / 请求</dt><dd>{{ compact(averageTokens) }}</dd></div>
          <div><dt>平均费用 / 请求</dt><dd>{{ money(averageCost) }}</dd></div>
        </dl>
      </article>
    </div>

    <section class="usage-log-panel">
      <header class="usage-log-heading">
        <div>
          <small>REQUEST LOG</small>
          <h2>明细记录</h2>
          <p>查看每次 API 调用对应的模型、Endpoint、Token 和费用。</p>
        </div>
        <div class="log-filters">
          <label class="usage-search">
            <WorkspaceNavIcon name="search" />
            <input v-model="search" type="search" placeholder="搜索模型或 Endpoint" />
          </label>
          <select v-model="modelFilter" aria-label="筛选模型">
            <option value="all">全部模型</option>
            <option v-for="model in modelOptions" :key="model" :value="model">{{ model }}</option>
          </select>
        </div>
      </header>

      <div class="usage-table-scroll">
        <div class="usage-table">
          <div class="usage-row usage-table-head">
            <span>时间</span><span>模型</span><span>Endpoint</span><span>Token</span><span>费用</span>
          </div>
          <div v-for="(item, index) in filteredUsage" :key="item.id || index" class="usage-row">
            <time>{{ formatTime(item.created_at) }}</time>
            <div class="usage-model-cell"><i></i><strong>{{ item.model || '—' }}</strong></div>
            <code>{{ item.endpoint || '—' }}</code>
            <span class="usage-number">{{ Number(item.total_tokens || 0).toLocaleString() }}</span>
            <strong class="usage-cost">{{ money(Number(item.actual_cost || 0)) }}</strong>
          </div>
          <div v-if="!filteredUsage.length && !loading" class="usage-empty">没有匹配的用量记录</div>
        </div>
      </div>
      <footer class="log-footer">
        <span>当前显示 {{ filteredUsage.length }} / {{ periodUsage.length }} 条记录</span>
        <span>费用以最终账单结算为准</span>
      </footer>
    </section>
  </section>
</template>

<style scoped>
.user-usage-page {
  width: 100%;
  max-width: 1320px;
  margin: 0 auto;
  padding-bottom: 42px;
}

.usage-heading {
  min-height: 116px;
  margin-bottom: 28px;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 28px;
}

.usage-heading-copy { min-width: 0; }
.usage-eyebrow {
  margin-bottom: 12px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #69727c;
  font: 700 .64rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .12em;
}
.usage-eyebrow i { width: 6px; height: 6px; border-radius: 50%; background: #54c99a; box-shadow: 0 0 0 4px rgba(84,201,154,.06); }
.usage-heading h1 { margin: 0; color: #f5f6f7; font-size: 2.25rem; line-height: 1.03; font-weight: 720; letter-spacing: -.045em; }
.usage-heading p { margin: 10px 0 0; color: #7b838d; font-size: .88rem; line-height: 1.65; }
.usage-heading-actions { display: flex; align-items: center; gap: 9px; }

.period-switch,
.metric-switch {
  padding: 3px;
  border: 1px solid #272b31;
  border-radius: 9px;
  display: flex;
  align-items: center;
  background: #0f1114;
}
.period-switch { flex-wrap: wrap; justify-content: flex-end; }
.period-switch button,
.metric-switch button {
  min-height: 33px;
  padding: 0 13px;
  border: 0;
  border-radius: 6px;
  color: #777f89;
  background: transparent;
  font: inherit;
  font-size: .72rem;
  cursor: pointer;
}
.period-switch button.active,
.metric-switch button.active { color: #eef0f2; background: #202329; box-shadow: inset 0 1px rgba(255,255,255,.04); }

.custom-range-row {
  min-height: 54px;
  margin: -14px 0 18px;
  padding: 9px 12px 9px 16px;
  border: 1px solid #23272d;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  background: #0d0f12;
}
.custom-range-row > span { margin-right: auto; color: #7b838d; font-size: .7rem; font-weight: 600; }
.custom-range-row > i { color: #4f5862; font-style: normal; }
.custom-range-row label { display: flex; align-items: center; gap: 8px; }
.custom-range-row label small { color: #626b75; font-size: .61rem; }
.custom-range-row input {
  height: 34px;
  padding: 0 9px;
  border: 1px solid #292e35;
  border-radius: 7px;
  color: #b9c0c8;
  color-scheme: dark;
  background: #111317;
  font: inherit;
  font-size: .68rem;
}

.usage-refresh { width: 41px; height: 41px; flex: 0 0 41px; border: 1px solid #272b31; border-radius: 9px; display: grid; place-items: center; color: #8c949e; background: #0f1114; cursor: pointer; }
.usage-refresh:hover:not(:disabled) { color: #f0f2f4; border-color: #363b43; }
.usage-refresh:disabled { opacity: .5; cursor: wait; }
.usage-refresh :deep(.workspace-nav-icon) { width: 15px; height: 15px; }

.usage-summary-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.usage-summary-card {
  min-height: 118px;
  padding: 20px;
  border: 1px solid #23272e;
  border-radius: 12px;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  background: #0f1115;
  box-shadow: inset 0 1px rgba(255,255,255,.012);
}
.summary-icon { width: 38px; height: 38px; flex: 0 0 38px; border: 1px solid #2b3038; border-radius: 9px; display: grid; place-items: center; color: #87919c; background: #14171b; }
.summary-icon :deep(.workspace-nav-icon) { width: 17px; height: 17px; }
.summary-copy { min-width: 0; display: flex; flex-direction: column; }
.summary-copy > span { color: #818995; font-size: .76rem; font-weight: 600; }
.summary-copy > strong { margin-top: 8px; color: #f1f3f5; font-size: 1.52rem; line-height: 1; font-weight: 680; letter-spacing: -.03em; }
.summary-copy > small { margin-top: 10px; color: #626b75; font-size: .65rem; }

.analytics-toolbar { margin-top: 13px; min-height: 56px; padding: 0 10px 0 17px; border: 1px solid #23272d; border-radius: 11px; display: flex; align-items: center; justify-content: space-between; gap: 18px; background: #0d0f12; }
.toolbar-context { display: flex; align-items: center; gap: 9px; color: #9aa1aa; font-size: .75rem; font-weight: 600; }
.toolbar-context :deep(.workspace-nav-icon) { width: 15px; height: 15px; color: #68727d; }
.toolbar-context b { min-height: 23px; padding: 0 8px; border: 1px solid #282d34; border-radius: 999px; display: inline-flex; align-items: center; color: #626b75; font-size: .61rem; font-weight: 600; }
.metric-switch button { min-height: 29px; padding: 0 11px; font-size: .68rem; }

.analytics-grid { margin-top: 13px; display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(330px, .8fr); gap: 13px; }
.analytics-panel { min-width: 0; border: 1px solid #23272e; border-radius: 12px; background: #0e1013; overflow: hidden; }
.panel-heading { min-height: 74px; padding: 17px 20px; border-bottom: 1px solid #20242a; display: flex; align-items: center; justify-content: space-between; gap: 18px; }
.panel-heading small { display: block; margin-bottom: 6px; color: #606974; font: 700 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .11em; }
.panel-heading h2 { margin: 0; color: #e9ebee; font-size: .95rem; font-weight: 650; letter-spacing: -.01em; }
.panel-caption { color: #666f79; font-size: .66rem; }
.panel-total { text-align: right; }
.panel-total span { display: block; margin-bottom: 4px; color: #626b75; font-size: .63rem; }
.panel-total strong { color: #e5e8eb; font-size: .94rem; }

.trend-panel { min-height: 330px; }
.trend-chart { position: relative; height: 255px; padding: 24px 22px 18px; color: #8e98a3; }
.trend-grid-lines { position: absolute; inset: 24px 22px 53px; display: flex; flex-direction: column; justify-content: space-between; pointer-events: none; }
.trend-grid-lines i { height: 1px; border-top: 1px dashed #242931; opacity: .92; }
.trend-line-svg { position: absolute; inset: 18px 22px 39px; width: calc(100% - 44px); height: calc(100% - 57px); overflow: visible; }
.trend-area { fill: url(#usageTrendArea); color: #818b96; }
.trend-line {
  fill: none;
  stroke: #949eaa;
  stroke-width: 2.4;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
  filter: drop-shadow(0 0 5px rgba(148,158,170,.1));
}
.trend-point circle {
  fill: #111419;
  stroke: #a2acb7;
  stroke-width: 2;
  vector-effect: non-scaling-stroke;
  opacity: .92;
  cursor: crosshair;
  transition: r .12s ease, opacity .12s ease;
}
.trend-point:hover circle { r: 8; opacity: 1; }
.trend-axis { position: absolute; left: 22px; right: 22px; bottom: 15px; display: grid; align-items: center; }
.trend-axis small { min-width: 0; color: #68717b; font-size: .61rem; text-align: center; white-space: nowrap; }
.trend-axis small.muted { visibility: hidden; }

.distribution-panel { min-height: 330px; }
.distribution-list { padding: 8px 20px 16px; }
.distribution-row { padding: 13px 0 11px; border-bottom: 1px solid #1d2025; }
.distribution-row:last-child { border-bottom: 0; }
.distribution-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; }
.distribution-head > span { min-width: 0; display: flex; align-items: center; gap: 9px; }
.distribution-head > span i { color: #4f5862; font: 600 .56rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; font-style: normal; }
.distribution-head strong { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #bdc3ca; font-size: .72rem; font-weight: 600; }
.distribution-head b { color: #d9dde1; font-size: .7rem; font-weight: 650; }
.distribution-head code { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #a8b0b8; font: 500 .66rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }
.distribution-track { margin-top: 8px; height: 4px; border-radius: 99px; overflow: hidden; background: #1c2025; }
.distribution-track i { display: block; height: 100%; border-radius: inherit; background: #626c76; }
.distribution-row > small { margin-top: 6px; display: block; color: #5f6872; font-size: .6rem; }
.endpoint-panel { min-height: 285px; }
.endpoint-list { padding-top: 6px; }

.overview-panel { min-height: 285px; }
.usage-profile-list { margin: 0; padding: 7px 20px 14px; }
.usage-profile-list > div { min-height: 48px; display: grid; grid-template-columns: 150px minmax(0, 1fr); align-items: center; gap: 16px; border-bottom: 1px solid #1d2025; }
.usage-profile-list > div:last-child { border-bottom: 0; }
.usage-profile-list dt { color: #68717b; font-size: .68rem; }
.usage-profile-list dd { margin: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #d7dbe0; font-size: .72rem; font-weight: 600; text-align: right; }
.usage-profile-list code { color: #aeb6be; font: 500 .65rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }

.panel-empty { min-height: 250px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; color: #555e68; }
.panel-empty :deep(.workspace-nav-icon) { width: 27px; height: 27px; color: #3f464e; }
.panel-empty strong { color: #717a84; font-size: .78rem; }
.panel-empty span { color: #505862; font-size: .65rem; }
.panel-empty.compact { min-height: 190px; }

.usage-log-panel { margin-top: 13px; border: 1px solid #23272e; border-radius: 12px; overflow: hidden; background: #0e1013; }
.usage-log-heading { min-height: 86px; padding: 17px 20px; border-bottom: 1px solid #20242a; display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.usage-log-heading small { display: block; margin-bottom: 5px; color: #606974; font: 700 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .11em; }
.usage-log-heading h2 { margin: 0; color: #e9ebee; font-size: .95rem; font-weight: 650; }
.usage-log-heading p { margin: 6px 0 0; color: #626b75; font-size: .65rem; }
.log-filters { display: flex; align-items: center; gap: 8px; }
.usage-search { width: 220px; height: 36px; padding: 0 11px; border: 1px solid #292e35; border-radius: 8px; display: flex; align-items: center; gap: 8px; background: #111317; }
.usage-search :deep(.workspace-nav-icon) { width: 14px; height: 14px; color: #616a74; }
.usage-search input { width: 100%; border: 0; outline: none; color: #d7dbe0; background: transparent; font: inherit; font-size: .68rem; }
.usage-search input::placeholder { color: #525a64; }
.log-filters select { min-width: 126px; height: 36px; padding: 0 28px 0 10px; border: 1px solid #292e35; border-radius: 8px; color: #a4abb4; background-color: #111317; font: inherit; font-size: .68rem; }

.usage-table-scroll { overflow-x: auto; }
.usage-table { min-width: 820px; }
.usage-row { min-height: 56px; padding: 0 20px; display: grid; grid-template-columns: 1fr 1.25fr 1.4fr .75fr .72fr; align-items: center; gap: 18px; border-bottom: 1px solid #1d2025; color: #8b939d; font-size: .71rem; }
.usage-row:last-child { border-bottom: 0; }
.usage-table-head { min-height: 42px; color: #59626c; font-size: .61rem; font-weight: 650; letter-spacing: .035em; }
.usage-row time { color: #737c86; }
.usage-model-cell { min-width: 0; display: flex; align-items: center; gap: 8px; }
.usage-model-cell i { width: 5px; height: 5px; flex: 0 0 5px; border-radius: 50%; background: #69747e; }
.usage-model-cell strong { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #dce0e4; font-size: .72rem; font-weight: 620; }
.usage-row code { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #9ba4ad; font: 500 .66rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }
.usage-number { color: #aeb5bd; font-variant-numeric: tabular-nums; }
.usage-cost { color: #d8dce0; font-size: .71rem; font-weight: 620; font-variant-numeric: tabular-nums; }
.usage-empty { min-height: 140px; display: grid; place-items: center; color: #606974; font-size: .72rem; }
.log-footer { min-height: 45px; padding: 0 20px; border-top: 1px solid #20242a; display: flex; align-items: center; justify-content: space-between; gap: 16px; color: #59626c; font-size: .61rem; }

@media (max-width: 1120px) {
  .usage-summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .analytics-grid { grid-template-columns: 1fr; }
  .trend-panel, .distribution-panel, .endpoint-panel, .overview-panel { min-height: auto; }
}

@media (max-width: 860px) {
  .usage-heading { align-items: flex-start; flex-direction: column; }
  .usage-heading-actions { width: 100%; }
  .period-switch { flex: 1; justify-content: flex-start; }
}

@media (max-width: 760px) {
  .usage-heading { min-height: auto; padding-top: 16px; }
  .usage-heading-actions { align-items: stretch; }
  .period-switch { overflow-x: auto; flex-wrap: nowrap; }
  .period-switch button { flex: 0 0 auto; padding: 0 10px; }
  .usage-heading h1 { font-size: 1.9rem; }
  .custom-range-row { margin-top: -8px; align-items: stretch; flex-wrap: wrap; justify-content: flex-start; }
  .custom-range-row > span { width: 100%; margin: 0; }
  .custom-range-row label { flex: 1; min-width: 130px; }
  .custom-range-row input { width: 100%; }
  .usage-summary-grid { grid-template-columns: 1fr; }
  .analytics-toolbar { align-items: flex-start; flex-direction: column; padding: 12px; }
  .metric-switch { width: 100%; }
  .metric-switch button { flex: 1; }
  .trend-chart { height: 230px; padding-inline: 14px; }
  .trend-grid-lines { left: 14px; right: 14px; }
  .trend-line-svg { left: 14px; right: 14px; width: calc(100% - 28px); }
  .trend-axis { left: 14px; right: 14px; }
  .usage-log-heading { align-items: flex-start; flex-direction: column; }
  .log-filters { width: 100%; }
  .usage-search { width: auto; flex: 1; }
  .usage-profile-list > div { grid-template-columns: 1fr; gap: 5px; padding: 10px 0; }
  .usage-profile-list dd { text-align: left; }
  .log-footer { align-items: flex-start; flex-direction: column; justify-content: center; padding: 10px 20px; }
}
</style>
