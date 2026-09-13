<script setup lang="ts">
// Admin Payment Dashboard —— 真实数据 (paymentAdminApi.getDashboard)
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  paymentAdminApi,
  type DashboardStats,
  type DailyStats,
  type PaymentMethodStat,
  type TopUserStat,
  type AdminPaymentOrder,
} from '../api/payment'
import { getErrorMessage } from '../core/api'
import PaymentStatusBadge from '../components/payment/PaymentStatusBadge.vue'

const { t } = useI18n()

const loading = ref(false)
const error = ref('')
const stats = ref<DashboardStats | null>(null)
const recentOrders = ref<AdminPaymentOrder[]>([])

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const [s, recent] = await Promise.all([
      paymentAdminApi.getDashboard(30),
      paymentAdminApi.listOrders({ page: 1, page_size: 8, status: 'COMPLETED' }) as Promise<{ items?: AdminPaymentOrder[] }>,
    ])
    stats.value = s
    recentOrders.value = recent?.items || []
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

function formatAmount(amounts: Record<string, number> | undefined, currencyHint?: string): string {
  if (!amounts) return '—'
  if (currencyHint) {
    const v = amounts[currencyHint] ?? amounts[currencyHint.toLowerCase()]
    if (typeof v === 'number') return currencyHint === 'CNY'
      ? `¥${v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
      : `${currencyHint === 'USD' ? '$' : ''}${v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  }
  const cny = amounts['CNY'] ?? amounts['cny']
  if (typeof cny === 'number') return `¥${cny.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  const usd = amounts['USD'] ?? amounts['usd']
  if (typeof usd === 'number') return `$${usd.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  const first = Object.values(amounts)[0]
  if (typeof first === 'number') return first.toLocaleString()
  return '—'
}

function formatCount(n: number | undefined): string {
  if (typeof n !== 'number') return '—'
  return n.toLocaleString()
}

const trendData = computed<{ days: string[]; values: number[] }>(() => {
  const series = stats.value?.daily_series || []
  const days = series.slice(-7).map((d: DailyStats) => (d.date || '').slice(5))
  const values = series.slice(-7).map((d: DailyStats) => {
    const cny = d.amount?.['CNY'] ?? d.amount?.['cny']
    return typeof cny === 'number' ? Math.round(cny) : 0
  })
  return { days, values }
})

const trendMax = computed(() => Math.max(1, ...trendData.value.values))

const channelRows = computed(() => {
  const list = stats.value?.payment_methods || []
  const total = list.reduce((sum, m) => {
    const cny = m.amount?.['CNY'] ?? m.amount?.['cny'] ?? 0
    return sum + (typeof cny === 'number' ? cny : 0)
  }, 0) || 1
  return list.map((m: PaymentMethodStat) => {
    const cny = m.amount?.['CNY'] ?? m.amount?.['cny'] ?? 0
    const v = typeof cny === 'number' ? cny : 0
    return {
      key: m.type,
      label: t('payment.method.' + m.type, m.type),
      amount: `¥${v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
      percent: Math.round((v / total) * 100),
    }
  })
})


const successRate = computed<number>(() => {
  const s = stats.value
  if (!s || !s.total_count) return 0
  return Math.round((s.today_count / s.total_count) * 1000) / 10
})

const topUsers = computed<TopUserStat[]>(() => {
  const map = stats.value?.top_users?.by_currency
  if (!map) return []
  const cny = map['CNY'] || map['cny']
  return Array.isArray(cny) ? cny.slice(0, 5) : []
})

function orderChannel(order: AdminPaymentOrder): string {
  const p = order.provider_key || order.payment_type
  return t('payment.method.' + p, p)
}

function orderAmount(order: AdminPaymentOrder): string {
  return `¥${(order.pay_amount || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

function shortTime(s: string | undefined): string {
  if (!s) return '—'
  const m = s.match(/T(\d{2}:\d{2}:\d{2})/)
  return m ? m[1] : (s.slice(11, 19) || s)
}

function shortDate(s: string | undefined): string {
  if (!s) return '—'
  return s.slice(5, 10)
}

function userName(u: TopUserStat): string {
  return u.username || u.email || `User #${u.user_id}`
}
</script>

<template>
  <section class="workspace-page payment-overview">
    <header class="payment-heading">
      <div>
        <div class="payment-eyebrow"><i></i><span>PAYMENT OPERATIONS</span></div>
        <h1>{{ t('payment.admin.title') }}</h1>
        <p>{{ t('payment.admin.description') }}</p>
      </div>
      <div class="payment-heading-actions">
        <button type="button" class="refresh-btn" :disabled="loading" @click="load">
          <i></i>{{ loading ? t('payment.loading') : t('payment.admin.refresh') }}
        </button>
        <RouterLink class="outline-link" to="/admin/orders">
          {{ t('payment.adminOrders.title') }} <b>→</b>
        </RouterLink>
      </div>
    </header>

    <p v-if="error" class="error-banner">{{ error }}</p>

    <div class="payment-summary-grid">
      <article class="payment-summary-card tone-success">
        <header><span>{{ t('payment.admin.todayAmount') }}</span><span class="summary-icon">¥</span></header>
        <strong>{{ formatAmount(stats?.today_amount) }}</strong>
        <footer><span>{{ t('payment.admin.totalAmount') }}</span><i></i></footer>
      </article>
      <article class="payment-summary-card tone-neutral">
        <header><span>{{ t('payment.admin.successOrders') }}</span><span class="summary-icon">#</span></header>
        <strong>{{ formatCount(stats?.today_count) }}</strong>
        <footer><span>{{ t('payment.admin.totalAmount') }}</span><i></i></footer>
      </article>
      <article class="payment-summary-card tone-success">
        <header><span>{{ t('payment.admin.successRate') }}</span><span class="summary-icon">%</span></header>
        <strong>{{ successRate.toFixed(1) }}%</strong>
        <footer><span>{{ t('payment.admin.recentTitle') }}</span><i></i></footer>
      </article>
      <article class="payment-summary-card tone-warning">
        <header><span>{{ t('payment.admin.pendingCount') }}</span><span class="summary-icon">⏳</span></header>
        <strong>{{ formatCount(stats?.pending_orders) }}</strong>
        <footer><span>{{ t('payment.admin.pendingAmount') }}: {{ formatAmount(stats?.today_amount) }}</span><i></i></footer>
      </article>
    </div>

    <div class="payment-main-grid">
      <section class="payment-panel trend-panel">
        <header class="panel-heading">
          <div>
            <span class="panel-kicker">REVENUE</span>
            <h2>{{ t('payment.admin.trendTitle') }}</h2>
          </div>
          <div class="trend-total">
            <span>{{ t('payment.admin.totalAmount') }}</span>
            <strong>{{ formatAmount(stats?.total_amount) }}</strong>
          </div>
        </header>
        <div v-if="!trendData.values.length" class="empty-state">{{ t('payment.admin.noData') }}</div>
        <div v-else class="trend-bars">
          <div v-for="(v, i) in trendData.values" :key="i" class="trend-bar">
            <span class="trend-bar-value">{{ v.toLocaleString() }}</span>
            <div class="trend-bar-track">
              <div class="trend-bar-fill" :style="{ height: Math.max(4, (v / trendMax) * 100) + '%' }"></div>
            </div>
            <span class="trend-bar-label">{{ trendData.days[i] }}</span>
          </div>
        </div>
      </section>

      <section class="payment-panel channel-panel">
        <header class="panel-heading">
          <div>
            <span class="panel-kicker">CHANNELS</span>
            <h2>{{ t('payment.admin.channelTitle') }}</h2>
          </div>
        </header>
        <div v-if="!channelRows.length" class="empty-state">{{ t('payment.admin.noData') }}</div>
        <ul v-else class="channel-list">
          <li v-for="row in channelRows" :key="row.key">
            <div class="channel-row-head">
              <span>{{ row.label }}</span>
              <span>{{ row.amount }}</span>
            </div>
            <div class="channel-bar"><div class="channel-bar-fill" :style="{ width: row.percent + '%' }"></div></div>
            <div class="channel-row-foot">{{ row.percent }}%</div>
          </li>
        </ul>
      </section>
    </div>

    <div class="payment-bottom-grid">
      <section class="payment-panel recent-panel">
        <header class="panel-heading">
          <div>
            <span class="panel-kicker">RECENT</span>
            <h2>{{ t('payment.admin.recentTitle') }}</h2>
          </div>
          <RouterLink class="outline-link small" to="/admin/orders">{{ t('payment.adminOrders.title') }} <b>→</b></RouterLink>
        </header>
        <div v-if="!recentOrders.length" class="empty-state">{{ t('payment.admin.noData') }}</div>
        <table v-else class="recent-table">
          <thead>
            <tr>
              <th>{{ t('payment.admin.colOrder') }}</th>
              <th>{{ t('payment.admin.colUser') }}</th>
              <th>{{ t('payment.admin.colChannel') }}</th>
              <th>{{ t('payment.admin.colAmount') }}</th>
              <th>{{ t('payment.admin.colStatus') }}</th>
              <th>{{ t('payment.admin.colTime') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in recentOrders" :key="o.id">
              <td class="mono">{{ o.out_trade_no }}</td>
              <td>{{ o.user_email || o.user_name || `User #${o.user_id}` }}</td>
              <td>{{ orderChannel(o) }}</td>
              <td>{{ orderAmount(o) }}</td>
              <td><PaymentStatusBadge :status="o.status" /></td>
              <td class="mono">{{ shortDate(o.created_at) }} {{ shortTime(o.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section class="payment-panel top-users-panel">
        <header class="panel-heading">
          <div>
            <span class="panel-kicker">USERS</span>
            <h2>{{ t('payment.admin.topUsersTitle') }}</h2>
          </div>
        </header>
        <div v-if="!topUsers.length" class="empty-state">{{ t('payment.admin.noData') }}</div>
        <ol v-else class="top-users-list">
          <li v-for="u in topUsers" :key="u.user_id">
            <span class="rank">{{ topUsers.indexOf(u) + 1 }}</span>
            <span class="user">{{ userName(u) }}</span>
            <span class="amount">{{ formatAmount(u.amount, 'CNY') }}</span>
            <span class="count">×{{ u.count }}</span>
          </li>
        </ol>
      </section>
    </div>
  </section>
</template>

<style scoped>
.payment-overview { width: 100%; max-width: 1280px; margin: 0 auto; padding: 12px 0 44px; }
.payment-heading { display: flex; align-items: center; justify-content: space-between; margin-bottom: 22px; gap: 24px; }
.payment-heading h1 { margin: 0; color: #f7f8fa; font-size: clamp(1.9rem, 2.4vw, 2.35rem); line-height: 1.08; font-weight: 680; letter-spacing: -.043em; }
.payment-heading p { max-width: 640px; margin: 10px 0 0; color: #858d97; font-size: .88rem; line-height: 1.6; }
.payment-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: #6ec0f5; font: 700 .67rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .13em; }
.payment-eyebrow i { width: 5px; height: 5px; border-radius: 50%; background: #6ec0f5; }
.payment-heading-actions { display: inline-flex; align-items: center; gap: 12px; }
.refresh-btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; font: 500 .8rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; transition: border-color .2s; }
.refresh-btn:hover:not(:disabled) { border-color: #3d4754; }
.refresh-btn:disabled { opacity: .6; cursor: not-allowed; }
.outline-link { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; border: 1px solid #2a2f37; color: #d6dbe1; text-decoration: none; font: 500 .8rem/1 ui-sans-serif, system-ui, sans-serif; transition: border-color .2s; }
.outline-link:hover { border-color: #3d4754; }
.outline-link.small { padding: 6px 10px; font-size: .72rem; }
.outline-link b { font-weight: 600; }
.error-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(239, 68, 68, .12); border: 1px solid rgba(239, 68, 68, .35); color: #fca5a5; font-size: .85rem; }
.payment-summary-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 22px; }
.payment-summary-card { padding: 18px 20px; border-radius: 14px; background: #11141a; border: 1px solid #1d2128; }
.payment-summary-card header { display: flex; align-items: center; justify-content: space-between; color: #858d97; font-size: .78rem; }
.payment-summary-card strong { display: block; margin: 8px 0 4px; color: #f7f8fa; font: 680 1.55rem/1.1 ui-sans-serif, system-ui, sans-serif; letter-spacing: -.03em; }
.payment-summary-card footer { color: #6c727b; font-size: .72rem; }
.payment-summary-card footer i { display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: #3d4754; margin-left: 4px; vertical-align: middle; }
.summary-icon { font-size: 1rem; color: #6c727b; }
.tone-success { border-color: rgba(72, 187, 153, .3); }
.tone-warning { border-color: rgba(237, 187, 56, .3); }
.tone-neutral { border-color: #1d2128; }
.payment-main-grid { display: grid; grid-template-columns: 1.6fr 1fr; gap: 14px; margin-bottom: 22px; }
.payment-bottom-grid { display: grid; grid-template-columns: 1.6fr 1fr; gap: 14px; }
.payment-panel { padding: 22px 24px; border-radius: 14px; background: #11141a; border: 1px solid #1d2128; }
.panel-heading { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; gap: 12px; }
.panel-heading h2 { margin: 4px 0 0; color: #f7f8fa; font-size: 1.1rem; font-weight: 600; letter-spacing: -.01em; }
.panel-kicker { color: #6ec0f5; font: 700 .65rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .12em; }
.trend-total { text-align: right; }
.trend-total span { display: block; color: #6c727b; font-size: .7rem; }
.trend-total strong { color: #f7f8fa; font: 600 1rem/1 ui-sans-serif, system-ui, sans-serif; }
.trend-bars { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 8px; height: 180px; align-items: end; }
.trend-bar { display: flex; flex-direction: column; align-items: center; gap: 6px; }
.trend-bar-value { color: #858d97; font-size: .68rem; }
.trend-bar-track { width: 100%; height: 140px; background: #1a1e25; border-radius: 6px; overflow: hidden; display: flex; align-items: flex-end; }
.trend-bar-fill { width: 100%; background: linear-gradient(180deg, #6ec0f5 0%, #4a93c5 100%); border-radius: 4px 4px 0 0; transition: height .6s ease; }
.trend-bar-label { color: #6c727b; font-size: .68rem; }
.empty-state { padding: 36px 0; text-align: center; color: #6c727b; font-size: .85rem; }
.channel-list { list-style: none; padding: 0; margin: 0; }
.channel-list li { margin-bottom: 16px; }
.channel-list li:last-child { margin-bottom: 0; }
.channel-row-head { display: flex; justify-content: space-between; color: #d6dbe1; font-size: .85rem; margin-bottom: 4px; }
.channel-bar { height: 6px; background: #1a1e25; border-radius: 3px; overflow: hidden; }
.channel-bar-fill { height: 100%; background: linear-gradient(90deg, #6ec0f5 0%, #4a93c5 100%); }
.channel-row-foot { margin-top: 4px; color: #6c727b; font-size: .72rem; }
.recent-table { width: 100%; border-collapse: collapse; }
.recent-table th { text-align: left; padding: 8px 10px; color: #6c727b; font: 500 .72rem/1 ui-sans-serif, system-ui, sans-serif; text-transform: uppercase; letter-spacing: .08em; border-bottom: 1px solid #1d2128; }
.recent-table td { padding: 10px; color: #d6dbe1; font-size: .82rem; border-bottom: 1px solid #161a20; }
.recent-table tr:last-child td { border-bottom: 0; }
.mono { font: 500 .78rem/1.2 ui-monospace, SFMono-Regular, Menlo, monospace; color: #b8bfc7; }
.top-users-list { list-style: none; padding: 0; margin: 0; }
.top-users-list li { display: grid; grid-template-columns: 24px 1fr auto auto; gap: 10px; align-items: center; padding: 10px 0; border-bottom: 1px solid #161a20; }
.top-users-list li:last-child { border-bottom: 0; }
.top-users-list .rank { color: #6c727b; font: 600 .85rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }
.top-users-list .user { color: #d6dbe1; font-size: .85rem; }
.top-users-list .amount { color: #6ec0f5; font: 500 .85rem/1 ui-sans-serif, system-ui, sans-serif; }
.top-users-list .count { color: #6c727b; font-size: .72rem; }
@media (max-width: 1080px) { .payment-main-grid, .payment-bottom-grid { grid-template-columns: 1fr; } }
@media (max-width: 720px) { .payment-summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .payment-heading { flex-direction: column; align-items: flex-start; } .recent-table th, .recent-table td { padding: 6px; font-size: .72rem; } }
</style>
