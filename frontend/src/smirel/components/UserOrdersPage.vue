<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useSession } from '../core/session'
import { paymentApi, type PaymentOrder, type OrderStatus, type RefundPreview, type RefundLedgerEntry } from '../api/payment'
import { getErrorMessage } from '../core/api'
import PaymentStatusBadge from './payment/PaymentStatusBadge.vue'

const { t } = useI18n()
const router = useRouter()
const { isAuthenticated } = useSession()

const loading = ref(false)
const error = ref('')
const orders = ref<PaymentOrder[]>([])
const total = ref(0)
const query = ref('')
const activeStatus = ref<'all' | OrderStatus>('all')

const statusFilters = computed(() => [
  { key: 'all' as const, label: t('payment.ordersAll') },
  { key: 'PAID' as const, label: t('payment.status.PAID') },
  { key: 'PENDING' as const, label: t('payment.status.PENDING') },
  { key: 'FAILED' as const, label: t('payment.status.FAILED') },
  { key: 'REFUNDED' as const, label: t('payment.status.REFUNDED') },
])

async function load() {
  if (!isAuthenticated.value) {
    orders.value = []
    total.value = 0
    return
  }
  loading.value = true
  error.value = ''
  try {
    const r = await paymentApi.listMyOrders({
      status: activeStatus.value === 'all' ? undefined : activeStatus.value,
      q: query.value.trim() || undefined,
    })
    orders.value = r?.items || []
    total.value = r?.total || 0
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

function pickStatus(s: 'all' | OrderStatus) {
  activeStatus.value = s
  load()
}

async function cancelOrder(o: PaymentOrder) {
  if (!confirm(t('payment.ordersActionCancelConfirm'))) return
  try {
    await paymentApi.cancelOrder(o.id)
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  }
}

// ---- 退款预览（方案 9.1 / 9.2 / 9.3） ----
const refundingOrder = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const refundSubmitting = ref(false)
const refundPreview = ref<RefundPreview | null>(null)
const refundPreviewLoading = ref(false)
const refundPreviewError = ref('')
const refundShowBreakdown = ref(false)

const POLICY_KEYS: Record<string, string> = {
  full_refund_24h_window: 'payment.refund.policyFullWindow',
  balance_principal_minus_used: 'payment.refund.policyBalance',
  subscription_double_settlement: 'payment.refund.policySubscription',
  force_refund: 'payment.refund.policyForce',
  already_refunded: 'payment.refund.policyAlreadyRefunded',
}

const BLOCK_KEYS: Record<string, string> = {
  not_completed: 'payment.refund.blockNotCompleted',
  unsupported_order_type: 'payment.refund.blockUnsupportedType',
  user_refund_disabled: 'payment.refund.blockUserRefundDisabled',
  refund_disabled: 'payment.refund.blockRefundDisabled',
  already_refunded: 'payment.refund.blockAlreadyRefunded',
  nothing_to_refund: 'payment.refund.blockNothingToRefund',
  balance_not_enough: 'payment.refund.blockBalanceNotEnough',
  refund_in_progress: 'payment.refund.blockRefundInProgress',
  force_refund_not_eligible: 'payment.refund.blockForceNotEligible',
}

// 明细行顺序刻意与 9.1 / 9.2 公式的书写顺序一致，便于对着方案核对。
const BREAKDOWN_ROWS: { key: string; label: string }[] = [
  { key: 'paid_amount', label: 'payment.refund.breakdownPaid' },
  { key: 'order_principal', label: 'payment.refund.breakdownOrderPrincipal' },
  { key: 'used_principal', label: 'payment.refund.breakdownUsedPrincipal' },
  { key: 'refundable_principal', label: 'payment.refund.breakdownRefundablePrincipal' },
  { key: 'used_usd', label: 'payment.refund.breakdownUsedUsd' },
  { key: 'time_based_charge', label: 'payment.refund.breakdownTimeBased' },
  { key: 'usage_based_charge', label: 'payment.refund.breakdownUsageBased' },
  { key: 'consumed', label: 'payment.refund.breakdownConsumed' },
  { key: 'pay_as_you_go_price_per_usd', label: 'payment.refund.breakdownUnitPrice' },
]

async function openRefund(o: PaymentOrder) {
  refundingOrder.value = o
  refundReason.value = ''
  refundPreview.value = null
  refundPreviewError.value = ''
  refundShowBreakdown.value = false
  refundPreviewLoading.value = true
  try {
    refundPreview.value = await paymentApi.getRefundPreview(o.id)
  } catch (e) {
    refundPreviewError.value = getErrorMessage(e)
  } finally {
    refundPreviewLoading.value = false
  }
}

function closeRefund() {
  refundingOrder.value = null
  refundReason.value = ''
  refundPreview.value = null
  refundPreviewError.value = ''
  refundShowBreakdown.value = false
}

function fmtMoney(v: number | null | undefined, currency?: string): string {
  const cur = currency || 'CNY'
  const n = typeof v === 'number' && Number.isFinite(v) ? v : 0
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: cur, maximumFractionDigits: 2 }).format(n)
  } catch {
    return `${cur} ${n.toFixed(2)}`
  }
}

function fmtUSD(v: number | undefined): string {
  const n = typeof v === 'number' && Number.isFinite(v) ? v : 0
  return `$${n.toFixed(4)}`
}

function policyLabel(p: string | undefined): string {
  if (!p) return t('payment.refund.policyUnknown')
  const key = POLICY_KEYS[p]
  return key ? t(key) : p
}

function blockedLabel(reason: string | undefined): string {
  if (!reason) return t('payment.refund.blockUnknown')
  const key = BLOCK_KEYS[reason]
  return key ? t(key) : reason
}

function entryLabel(e: RefundLedgerEntry): string {
  const type = e.entry_type === 'bonus' ? t('payment.refund.ledgerEntryBonus') : t('payment.refund.ledgerEntryPrincipal')
  const dir = e.direction === 'debit' ? t('payment.refund.ledgerDebit') : t('payment.refund.ledgerCredit')
  return `${type} · ${dir}`
}

const breakdownRows = computed(() => {
  const map = refundPreview.value?.calculation_breakdown
  if (!map) return [] as { label: string; value: number }[]
  return BREAKDOWN_ROWS.filter((row) => typeof map[row.key] === 'number').map((row) => ({
    label: t(row.label),
    value: map[row.key],
  }))
})

const canSubmitRefund = computed(() => !!refundPreview.value?.requestable && !refundSubmitting.value)

async function submitRefund() {
  if (!refundingOrder.value) return
  if (refundPreview.value && !refundPreview.value.requestable) return
  refundSubmitting.value = true
  try {
    await paymentApi.requestRefund(refundingOrder.value.id, { reason: refundReason.value })
    error.value = ''
    closeRefund()
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    refundSubmitting.value = false
  }
}
const fmtAmount = (o: PaymentOrder) => {
  const cur = o.currency || 'CNY'
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: cur }).format(o.pay_amount ?? o.amount)
  } catch {
    return `${cur} ${(o.pay_amount ?? o.amount).toFixed(2)}`
  }
}

const fmtDate = (s: string) => {
  try {
    const d = new Date(s)
    return d.toLocaleString()
  } catch {
    return s
  }
}

const totalsByCurrency = computed(() => {
  const map: Record<string, { count: number; total: number }> = {}
  orders.value
    .filter((o) => o.status === 'PAID' || o.status === 'COMPLETED' || o.status === 'PARTIALLY_REFUNDED' || o.status === 'RECHARGING')
    .forEach((o) => {
      const cur = o.currency || 'CNY'
      const amt = o.pay_amount ?? o.amount
      if (!map[cur]) map[cur] = { count: 0, total: 0 }
      map[cur].count += 1
      map[cur].total += amt
    })
  return map
})

const completedCount = computed(() => {
  return orders.value.filter((o) => ['PAID', 'COMPLETED', 'RECHARGING', 'PARTIALLY_REFUNDED'].includes(o.status)).length
})

function rowCanCancel(o: PaymentOrder) {
  return o.status === 'PENDING' || o.status === 'RECHARGING'
}

function rowCanRefund(o: PaymentOrder) {
  return o.status === 'PAID' || o.status === 'COMPLETED' || o.status === 'PARTIALLY_REFUNDED' || o.status === 'RECHARGING'
}

function viewOrder(o: PaymentOrder) {
  router.push({ path: '/payment/result', query: { out_trade_no: o.out_trade_no } })
}
</script>

<template>
  <div class="orders-page">
    <header class="page-head">
      <div>
        <h1>{{ t('payment.ordersTitle') }}</h1>
        <p class="hint">{{ t('payment.ordersSubtitle') }}</p>
      </div>
      <RouterLink class="primary" to="/subscriptions">{{ t('payment.tabRecharge') }}</RouterLink>
    </header>

    <section class="stats">
      <div class="stat">
        <span class="eyebrow">{{ t('payment.ordersTotal') }}</span>
        <strong>{{ total }}</strong>
      </div>
      <div class="stat">
        <span class="eyebrow">{{ t('payment.ordersCompleted') }}</span>
        <strong>{{ completedCount }}</strong>
      </div>
      <div class="stat">
        <span class="eyebrow">{{ t('payment.ordersPaidAmount') }}</span>
        <strong>
          <span v-for="(v, k) in totalsByCurrency" :key="k">
            {{ k }} {{ v.total.toFixed(2) }}
          </span>
          <span v-if="Object.keys(totalsByCurrency).length === 0">—</span>
        </strong>
      </div>
    </section>

    <section class="filter-bar">
      <div class="tabs">
        <button
          v-for="f in statusFilters"
          :key="f.key"
          type="button"
          :class="{ active: activeStatus === f.key }"
          @click="pickStatus(f.key)"
        >
          {{ f.label }}
        </button>
      </div>
      <input
        v-model="query"
        type="search"
        :placeholder="t('payment.ordersSearch')"
        class="search"
        @input="load"
      />
    </section>

    <p v-if="error" class="error">{{ error }}</p>
    <div v-if="loading" class="loading">{{ t('payment.loading') }}</div>

    <section v-else-if="orders.length === 0" class="empty">
      <p>{{ t('payment.ordersEmpty') }}</p>
      <small>{{ t('payment.ordersEmptyHint') }}</small>
      <RouterLink class="primary" to="/subscriptions">{{ t('payment.ordersGoRecharge') }}</RouterLink>
    </section>

    <section v-else class="orders-table">
      <table>
        <thead>
          <tr>
            <th>{{ t('payment.ordersTableOrder') }}</th>
            <th>{{ t('payment.ordersTableCreatedAt') }}</th>
            <th>{{ t('payment.ordersTableAmount') }}</th>
            <th>{{ t('payment.ordersTableChannel') }}</th>
            <th>{{ t('payment.ordersTableStatus') }}</th>
            <th>{{ t('payment.ordersTableAction') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.id">
            <td><code>{{ o.out_trade_no }}</code></td>
            <td>{{ fmtDate(o.created_at) }}</td>
            <td class="amount">{{ fmtAmount(o) }}</td>
            <td>{{ o.payment_type }}</td>
            <td><PaymentStatusBadge :status="o.status" /></td>
            <td class="actions">
              <button v-if="rowCanCancel(o)" class="ghost danger" type="button" @click="cancelOrder(o)">
                {{ t('payment.ordersActionCancel') }}
              </button>
              <button v-if="rowCanRefund(o)" class="ghost" type="button" @click="openRefund(o)">
                {{ t('payment.ordersActionRefundRequest') }}
              </button>
              <button class="ghost" type="button" @click="viewOrder(o)">
                {{ t('payment.ordersTableAction') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="refundingOrder" class="modal-mask" @click.self="closeRefund">
      <div class="modal refund-modal">
        <h3>{{ t('payment.refund.previewTitle') }}</h3>
        <p class="modal-sub">
          {{ t('payment.refund.orderNo') }} <code>{{ refundingOrder.out_trade_no }}</code>
        </p>

        <div v-if="refundPreviewLoading" class="loading">{{ t('payment.refund.loadingPreview') }}</div>
        <p v-else-if="refundPreviewError" class="error">{{ refundPreviewError }}</p>

        <template v-else-if="refundPreview">
          <dl class="refund-grid">
            <div>
              <dt>{{ t('payment.refund.purchasedAt') }}</dt>
              <dd>{{ fmtDate(refundPreview.purchased_at) }}</dd>
            </div>
            <div>
              <dt>{{ t('payment.refund.activatedAt') }}</dt>
              <dd>{{ refundPreview.activated_at ? fmtDate(refundPreview.activated_at) : '—' }}</dd>
            </div>
            <div v-if="refundPreview.refund_requested_at">
              <dt>{{ t('payment.refund.requestedAt') }}</dt>
              <dd>{{ fmtDate(refundPreview.refund_requested_at) }}</dd>
            </div>
            <div>
              <dt>{{ t('payment.refund.usedUsd') }}</dt>
              <dd class="mono">{{ fmtUSD(refundPreview.used_usd) }}</dd>
            </div>
            <div>
              <dt>{{ t('payment.refund.charged') }}</dt>
              <dd class="mono">{{ fmtMoney(refundPreview.charged_pay_amount, refundPreview.currency) }}</dd>
            </div>
            <div>
              <dt>{{ t('payment.refund.bonusBalance') }}</dt>
              <dd class="mono">{{ fmtMoney(refundPreview.bonus_balance, refundPreview.currency) }}</dd>
            </div>
            <div class="span-2">
              <dt>{{ t('payment.refund.policy') }}</dt>
              <dd>{{ policyLabel(refundPreview.policy) }}</dd>
            </div>
          </dl>

          <div class="refund-result">
            <span class="eyebrow">{{ t('payment.refund.formulaResult') }}</span>
            <strong>{{ fmtMoney(refundPreview.refundable_pay_amount, refundPreview.currency) }}</strong>
            <small v-if="refundPreview.balance_cap_applied">{{ t('payment.refund.balanceCapApplied') }}</small>
            <small v-else>{{ t('payment.refund.formulaPay') }}</small>
          </div>

          <p v-if="!refundPreview.requestable" class="refund-blocked">
            {{ blockedLabel(refundPreview.blocked_reason) }}
          </p>

          <button
            v-if="breakdownRows.length"
            type="button"
            class="refund-toggle"
            @click="refundShowBreakdown = !refundShowBreakdown"
          >
            {{ refundShowBreakdown ? t('payment.refund.collapse') : t('payment.refund.expand') }}
          </button>

          <dl v-if="refundShowBreakdown && breakdownRows.length" class="refund-breakdown">
            <div v-for="row in breakdownRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd class="mono">{{ fmtMoney(row.value, refundPreview.currency) }}</dd>
            </div>
          </dl>

          <section v-if="refundPreview.ledger" class="refund-ledger">
            <span class="eyebrow">{{ t('payment.refund.ledgerTitle') }}</span>
            <dl class="refund-breakdown">
              <div>
                <dt>{{ t('payment.refund.ledgerPrincipalCredit') }}</dt>
                <dd class="mono">{{ fmtMoney(refundPreview.ledger.principal_credit, refundPreview.currency) }}</dd>
              </div>
              <div>
                <dt>{{ t('payment.refund.ledgerPrincipalDebit') }}</dt>
                <dd class="mono">{{ fmtMoney(refundPreview.ledger.principal_debit, refundPreview.currency) }}</dd>
              </div>
              <div>
                <dt>{{ t('payment.refund.ledgerPrincipalLeft') }}</dt>
                <dd class="mono">{{ fmtMoney(refundPreview.ledger.principal_left, refundPreview.currency) }}</dd>
              </div>
              <div v-if="refundPreview.ledger.bonus_credit > 0">
                <dt>{{ t('payment.refund.ledgerBonusCredit') }}</dt>
                <dd class="mono">{{ fmtMoney(refundPreview.ledger.bonus_credit, refundPreview.currency) }}</dd>
              </div>
              <div v-if="refundPreview.ledger.bonus_left > 0">
                <dt>{{ t('payment.refund.ledgerBonusLeft') }}</dt>
                <dd class="mono">{{ fmtMoney(refundPreview.ledger.bonus_left, refundPreview.currency) }}</dd>
              </div>
            </dl>
            <p v-if="refundPreview.ledger.frozen" class="refund-note">
              {{ t('payment.refund.ledgerFrozen') }}
            </p>
            <small class="refund-note">{{ t('payment.refund.bonusKeepHint') }}</small>
          </section>

          <section v-if="refundPreview.ledger_entries?.length" class="refund-ledger">
            <span class="eyebrow">{{ t('payment.refund.ledgerEntriesTitle') }}</span>
            <ul class="refund-entries">
              <li v-for="e in refundPreview.ledger_entries" :key="e.id">
                <span>{{ entryLabel(e) }}</span>
                <span class="mono">{{ fmtMoney(e.amount, refundPreview.currency) }}</span>
                <span class="refund-entry-time">{{ fmtDate(e.created_at) }}</span>
              </li>
            </ul>
          </section>
        </template>

        <textarea
          v-model="refundReason"
          rows="3"
          :placeholder="t('payment.ordersActionRefundPrompt')"
        ></textarea>

        <div class="modal-actions">
          <button class="ghost" type="button" @click="closeRefund">{{ t('payment.cancel') }}</button>
          <button class="primary" type="button" :disabled="!canSubmitRefund" @click="submitRefund">
            {{ refundSubmitting ? t('payment.submitting') : t('payment.refund.submit') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.orders-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 0 0 36px;
}
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.page-head h1 {
  margin: 0;
  font-size: 1.4rem;
  font-weight: 640;
  color: rgba(255, 255, 255, 0.92);
}
.page-head .hint {
  margin: 4px 0 0;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.5);
}
.page-head .primary,
.empty .primary {
  background: #79c4f5;
  color: #071019;
  border: 1px solid #79c4f5;
  border-radius: 10px;
  padding: 0 16px;
  height: 38px;
  font-size: 0.84rem;
  font-weight: 600;
  text-decoration: none;
  line-height: 36px;
}
.stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
.stat {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(18, 22, 28, 0.88), rgba(12, 16, 22, 0.94));
  padding: 16px 18px;
}
.stat .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.stat strong {
  display: block;
  font-size: 1.4rem;
  font-weight: 640;
  color: rgba(255, 255, 255, 0.95);
  font-variant-numeric: tabular-nums;
  margin-top: 6px;
}
.filter-bar {
  display: flex;
  gap: 12px;
  align-items: center;
}
.tabs {
  display: flex;
  gap: 6px;
  flex: 1;
  flex-wrap: wrap;
}
.tabs button {
  height: 32px;
  padding: 0 14px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(15, 18, 24, 0.78);
  color: rgba(255, 255, 255, 0.62);
  font-family: inherit;
  font-size: 0.78rem;
  cursor: pointer;
}
.tabs button.active {
  border-color: #79c4f5;
  color: #fff;
  background: rgba(120, 175, 230, 0.16);
}
.search {
  height: 32px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(15, 18, 24, 0.78);
  color: #fff;
  font-family: inherit;
  font-size: 0.78rem;
  min-width: 220px;
}
.error {
  margin: 0;
  padding: 12px 14px;
  border-radius: 10px;
  background: rgba(244, 139, 139, 0.08);
  border: 1px solid rgba(244, 139, 139, 0.32);
  color: #f48b8b;
  font-size: 0.84rem;
}
.loading,
.empty {
  padding: 36px 18px;
  border-radius: 12px;
  background: rgba(15, 18, 24, 0.78);
  border: 1px solid rgba(255, 255, 255, 0.06);
  text-align: center;
  color: rgba(255, 255, 255, 0.55);
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: center;
}
.empty p {
  margin: 0;
  font-size: 0.96rem;
  color: rgba(255, 255, 255, 0.75);
}
.empty small {
  font-size: 0.74rem;
}
.orders-table {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: rgba(15, 18, 24, 0.78);
  overflow: hidden;
}
.orders-table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.78rem;
}
.orders-table th {
  text-align: left;
  padding: 12px 14px;
  font-size: 0.62rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.orders-table td {
  padding: 12px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.8);
}
.orders-table tr:last-child td {
  border-bottom: 0;
}
.orders-table .amount {
  font-variant-numeric: tabular-nums;
  color: rgba(255, 255, 255, 0.95);
}
.orders-table code {
  font: 0.74rem ui-monospace, monospace;
  color: rgba(255, 255, 255, 0.7);
}
.orders-table .actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.orders-table .actions button {
  height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.78);
  font-family: inherit;
  font-size: 0.72rem;
  cursor: pointer;
}
.orders-table .actions button:hover {
  background: rgba(255, 255, 255, 0.08);
}
.orders-table .actions button.danger {
  border-color: rgba(244, 139, 139, 0.32);
  color: #f48b8b;
}
.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(8, 12, 18, 0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal {
  background: #0e141b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 14px;
  width: min(420px, calc(100vw - 32px));
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.modal h3 {
  margin: 0;
  font-size: 0.95rem;
  color: rgba(255, 255, 255, 0.92);
}
.modal textarea {
  width: 100%;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(15, 18, 24, 0.95);
  color: #fff;
  font-family: inherit;
  font-size: 0.84rem;
  resize: vertical;
}
.modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.modal-actions button {
  height: 36px;
  padding: 0 16px;
  border-radius: 8px;
  font-family: inherit;
  font-size: 0.82rem;
  cursor: pointer;
}
.modal-actions .ghost {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.78);
}
.modal-actions .primary {
  background: #79c4f5;
  border: 1px solid #79c4f5;
  color: #071019;
  font-weight: 600;
}
.refund-modal {
  width: min(620px, calc(100vw - 32px));
  max-height: calc(100vh - 48px);
  overflow-y: auto;
}
.modal-sub {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 0.76rem;
  color: rgba(255, 255, 255, 0.5);
}
.modal-sub code,
.refund-grid dd code {
  font: 0.76rem ui-monospace, monospace;
  color: rgba(255, 255, 255, 0.72);
}
.mono {
  font-family: ui-monospace, monospace;
  font-variant-numeric: tabular-nums;
}
.refund-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 16px;
  margin: 0;
}
.refund-grid > div {
  min-width: 0;
}
.refund-grid .span-2 {
  grid-column: span 2;
}
.refund-grid dt,
.refund-ledger .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.42);
}
.refund-grid dd {
  margin: 3px 0 0;
  font-size: 0.84rem;
  color: rgba(255, 255, 255, 0.9);
  overflow-wrap: anywhere;
}
.refund-result {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid rgba(121, 196, 245, 0.28);
  background: rgba(121, 196, 245, 0.1);
}
.refund-result .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.refund-result strong {
  font-size: 1.3rem;
  font-weight: 640;
  color: #d8ecfb;
  font-variant-numeric: tabular-nums;
}
.refund-result small,
.refund-note {
  margin: 0;
  font-size: 0.72rem;
  color: rgba(255, 255, 255, 0.5);
}
.refund-blocked {
  margin: 0;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(244, 197, 139, 0.1);
  border: 1px solid rgba(244, 197, 139, 0.3);
  color: #f4c58b;
  font-size: 0.8rem;
}
.refund-toggle {
  align-self: flex-start;
  height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.72);
  font-family: inherit;
  font-size: 0.74rem;
  cursor: pointer;
}
.refund-toggle:hover {
  background: rgba(255, 255, 255, 0.08);
}
.refund-breakdown {
  display: flex;
  flex-direction: column;
  margin: 0;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 10px;
  overflow: hidden;
}
.refund-breakdown > div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.refund-breakdown > div:last-child {
  border-bottom: 0;
}
.refund-breakdown dt {
  font-size: 0.76rem;
  color: rgba(255, 255, 255, 0.55);
}
.refund-breakdown dd {
  margin: 0;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.92);
}
.refund-ledger {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.refund-entries {
  display: flex;
  flex-direction: column;
  gap: 6px;
  list-style: none;
  margin: 0;
  padding: 0;
}
.refund-entries li {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
  align-items: center;
  padding: 7px 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  font-size: 0.76rem;
  color: rgba(255, 255, 255, 0.78);
}
.refund-entry-time {
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.4);
}

@media (max-width: 720px) {
  .stats {
    grid-template-columns: 1fr;
  }
  .filter-bar {
    flex-direction: column;
    align-items: stretch;
  }
  .orders-table table,
  .orders-table thead,
  .orders-table tbody,
  .orders-table tr,
  .orders-table td,
  .orders-table th {
    display: block;
  }
  .orders-table thead {
    display: none;
  }
  .orders-table tr {
    padding: 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }
  .orders-table td {
    padding: 4px 0;
    border: 0;
  }
  .orders-table .actions {
    margin-top: 8px;
  }
  .refund-grid {
    grid-template-columns: 1fr;
  }
  .refund-grid .span-2 {
    grid-column: auto;
  }
}
</style>
