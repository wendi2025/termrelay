<script setup lang="ts">
// Admin Orders —— 真实订单列表 + 取消/重试/退款
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  paymentAdminApi,
  type AdminOrderAuditLog,
  type AdminPaymentOrder,
  type OrderStatus,
  type PaginatedResponse,
  type RefundLedgerEntry,
  type RefundPreview,
} from '../api/payment'
import { getErrorMessage } from '../core/api'
import PaymentStatusBadge from '../components/payment/PaymentStatusBadge.vue'

const { t } = useI18n()

type Filter = 'all' | OrderStatus

// 后端 payment.OrderStatus 全集，保证任何终态都能被筛出来
const ALL_STATUSES: OrderStatus[] = [
  'PENDING',
  'PAID',
  'RECHARGING',
  'COMPLETED',
  'EXPIRED',
  'CANCELLED',
  'FAILED',
  'REFUND_REQUESTED',
  'REFUNDING',
  'REFUND_PENDING',
  'PARTIALLY_REFUNDED',
  'REFUNDED',
  'REFUND_FAILED',
]

const loading = ref(false)
const error = ref('')
const success = ref('')
const orders = ref<AdminPaymentOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const query = ref('')
const activeFilter = ref<Filter>('all')

const filterDefs = computed<{ key: Filter; label: string }[]>(() => [
  { key: 'all', label: t('payment.adminOrders.filterAll') },
  ...ALL_STATUSES.map((s) => ({ key: s as Filter, label: t('payment.status.' + s, s) })),
])

function onFilterChange(e: Event): void {
  activeFilter.value = (e.target as HTMLSelectElement).value as Filter
}

const refundModalOpen = ref(false)
const refundTarget = ref<AdminPaymentOrder | null>(null)
const refundAmount = ref<number | null>(null)
const refundReason = ref('')
const refundForce = ref(false)
const refundSubmitting = ref(false)
const refundPreview = ref<RefundPreview | null>(null)
const refundPreviewLoading = ref(false)
const refundPreviewError = ref('')
const refundAmountTouched = ref(false)
const refundShowBreakdown = ref(false)

const actionBusyId = ref<number | null>(null)

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailOrder = ref<AdminPaymentOrder | null>(null)
const detailLogs = ref<AdminOrderAuditLog[]>([])

async function openDetail(o: AdminPaymentOrder): Promise<void> {
  detailOpen.value = true
  detailLoading.value = true
  detailOrder.value = o
  detailLogs.value = []
  try {
    const r = await paymentAdminApi.getOrderDetail(o.id)
    detailOrder.value = r?.order || o
    detailLogs.value = r?.auditLogs || []
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    detailLoading.value = false
  }
}

function closeDetail(): void {
  detailOpen.value = false
  detailOrder.value = null
  detailLogs.value = []
}

function prettyDetail(raw: string | undefined): string {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const r = (await paymentAdminApi.listOrders({
      page: page.value,
      page_size: pageSize.value,
      status: activeFilter.value === 'all' ? undefined : activeFilter.value,
      q: query.value.trim() || undefined,
    })) as PaginatedResponse<AdminPaymentOrder>
    orders.value = r?.items || []
    total.value = r?.total || 0
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(activeFilter, () => { page.value = 1; load() })
watch([page, pageSize], () => { load() })
watch(query, () => { page.value = 1; load() })

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

function channelOf(o: AdminPaymentOrder): string {
  const p = o.provider_key || o.payment_type
  return t('payment.method.' + p, p)
}

function fmtAmount(v: number | undefined): string {
  const n = typeof v === 'number' ? v : 0
  return `¥${n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

function fmtDate(s: string | undefined): string {
  if (!s) return '—'
  return s.replace('T', ' ').slice(0, 19)
}

function userLabel(o: AdminPaymentOrder): string {
  return o.user_email || o.user_name || `User #${o.user_id}`
}

async function doCancel(o: AdminPaymentOrder): Promise<void> {
  if (!confirm(t('payment.adminOrders.cancelConfirm'))) return
  actionBusyId.value = o.id
  try {
    await paymentAdminApi.cancelOrder(o.id)
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    actionBusyId.value = null
  }
}

async function doRetry(o: AdminPaymentOrder): Promise<void> {
  actionBusyId.value = o.id
  try {
    await paymentAdminApi.retryFulfillment(o.id)
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    actionBusyId.value = null
  }
}

// 模拟支付（仅管理员）：不经过上游渠道，直接把 PENDING 订单置为已支付并跑真实履约。
// 只用于真实商户凭据到位前验证链路，或为场外已确认收款的订单补记账。
async function doSimulatePaid(o: AdminPaymentOrder): Promise<void> {
  if (!confirm(t('payment.adminOrders.simulateConfirm'))) return
  actionBusyId.value = o.id
  try {
    await paymentAdminApi.simulatePaid(o.id)
    await load()
    // 必须放在 load() 之后：load() 会清掉上一条成功提示
    success.value = t('payment.adminOrders.simulateSuccess')
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    actionBusyId.value = null
  }
}

// ---- 退款预览（方案 9.1 / 9.2 / 9.3，管理员视角） ----
// 预览是审核记录的唯一来源：可退金额、策略、账本都取自
// /admin/payment/orders/:id/refund-preview，不再直接用订单上的 pay_amount 开票。
let refundPreviewSeq = 0

async function loadRefundPreview(): Promise<void> {
  const target = refundTarget.value
  if (!target) return
  const force = refundForce.value
  const seq = ++refundPreviewSeq
  refundPreviewLoading.value = true
  refundPreviewError.value = ''
  try {
    const preview = await paymentAdminApi.getRefundPreview(target.id, force)
    if (seq !== refundPreviewSeq) return
    refundPreview.value = preview
    // 未手工改过金额时始终跟随公式结果，避免展示一个实际退不出去的报价
    if (!refundAmountTouched.value) refundAmount.value = preview?.refundable_pay_amount ?? null
  } catch (e) {
    if (seq !== refundPreviewSeq) return
    refundPreview.value = null
    refundPreviewError.value = getErrorMessage(e)
  } finally {
    if (seq === refundPreviewSeq) refundPreviewLoading.value = false
  }
}

function openRefund(o: AdminPaymentOrder): void {
  // 先复位 modal 与 force，避免上一次的 force 余值触发多余的预览请求
  refundModalOpen.value = false
  refundForce.value = false
  refundTarget.value = o
  refundAmount.value = null
  refundAmountTouched.value = false
  refundReason.value = ''
  refundPreview.value = null
  refundPreviewError.value = ''
  refundShowBreakdown.value = false
  refundModalOpen.value = true
  void loadRefundPreview()
}

function closeRefund(): void {
  refundPreviewSeq += 1
  refundModalOpen.value = false
  refundTarget.value = null
  refundAmount.value = null
  refundAmountTouched.value = false
  refundReason.value = ''
  refundForce.value = false
  refundPreview.value = null
  refundPreviewError.value = ''
  refundPreviewLoading.value = false
  refundShowBreakdown.value = false
}

// 强制退款会放宽 24h / 消耗限制，可退口径随之变化，必须重新取预览而不是复用旧报价
watch(refundForce, () => {
  if (refundModalOpen.value && refundTarget.value) void loadRefundPreview()
})

async function submitRefund(): Promise<void> {
  if (!refundTarget.value) return
  if (refundPreview.value && !refundPreview.value.requestable) return
  refundSubmitting.value = true
  try {
    await paymentAdminApi.refund(refundTarget.value.id, {
      amount: refundAmount.value,
      reason: refundReason.value,
      force: refundForce.value,
    })
    closeRefund()
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    refundSubmitting.value = false
  }
}

function fmtMoney(v: number | null | undefined, currency?: string): string {
  const cur = currency || 'CNY'
  const n = typeof v === 'number' && Number.isFinite(v) ? v : 0
  try {
    return new Intl.NumberFormat('zh-CN', { style: 'currency', currency: cur, maximumFractionDigits: 2 }).format(n)
  } catch {
    return `${cur} ${n.toFixed(2)}`
  }
}

function fmtUSD(v: number | undefined): string {
  const n = typeof v === 'number' && Number.isFinite(v) ? v : 0
  return `$${n.toFixed(4)}`
}

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

// 明细行顺序刻意与 9.1 / 9.2 公式的书写顺序一致，便于对着方案核对
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

/** 9.3 记录字段：订单号 / 账户 / 购买·激活·申请时间 / 已用官方 $ / 已扣金额 / 赠送余额 / 计算方式 */
const refundMetaRows = computed<{ label: string; value: string }[]>(() => {
  const p = refundPreview.value
  if (!p) return []
  const rows: { label: string; value: string }[] = [
    { label: t('payment.refund.account'), value: p.user_email || p.user_name || `User #${p.user_id}` },
    {
      label: t('payment.adminOrders.fieldOrderType'),
      value:
        p.order_type === 'subscription'
          ? t('payment.adminOrders.orderTypeSubscription')
          : t('payment.adminOrders.orderTypeRecharge'),
    },
    { label: t('payment.refund.purchasedAt'), value: fmtDate(p.purchased_at) },
    { label: t('payment.refund.activatedAt'), value: p.activated_at ? fmtDate(p.activated_at) : '—' },
  ]
  if (p.refund_requested_at) rows.push({ label: t('payment.refund.requestedAt'), value: fmtDate(p.refund_requested_at) })
  if (p.refund_requested_by) rows.push({ label: t('payment.refund.requestedBy'), value: p.refund_requested_by })
  if (p.refund_request_reason) rows.push({ label: t('payment.refund.requestReason'), value: p.refund_request_reason })
  rows.push(
    { label: t('payment.refund.usedUsd'), value: fmtUSD(p.used_usd) },
    { label: t('payment.refund.charged'), value: fmtMoney(p.charged_pay_amount, p.currency) },
    { label: t('payment.refund.bonusBalance'), value: fmtMoney(p.bonus_balance, p.currency) },
    { label: t('payment.refund.policy'), value: policyLabel(p.policy) },
  )
  return rows
})

const refundBreakdownRows = computed(() => {
  const map = refundPreview.value?.calculation_breakdown
  if (!map) return [] as { label: string; value: number }[]
  return BREAKDOWN_ROWS.filter((row) => typeof map[row.key] === 'number').map((row) => ({
    label: t(row.label),
    value: map[row.key],
  }))
})

const refundLedgerRows = computed(() => {
  const l = refundPreview.value?.ledger
  if (!l) return [] as { label: string; value: number }[]
  const rows: { label: string; value: number }[] = [
    { label: t('payment.refund.ledgerPrincipalCredit'), value: l.principal_credit },
    { label: t('payment.refund.ledgerPrincipalDebit'), value: l.principal_debit },
    { label: t('payment.refund.ledgerPrincipalLeft'), value: l.principal_left },
  ]
  if (l.bonus_credit > 0) rows.push({ label: t('payment.refund.ledgerBonusCredit'), value: l.bonus_credit })
  if (l.bonus_left > 0) rows.push({ label: t('payment.refund.ledgerBonusLeft'), value: l.bonus_left })
  return rows
})

const canSubmitRefund = computed(() => !!refundPreview.value?.requestable && !refundSubmitting.value)
</script>

<template>
  <section class="workspace-page payment-orders">
    <header class="orders-heading">
      <div>
        <div class="orders-eyebrow"><i></i><span>ORDERS</span></div>
        <h1>{{ t('payment.adminOrders.title') }}</h1>
        <p>{{ t('payment.adminOrders.description') }}</p>
      </div>
      <div class="orders-heading-actions">
        <button type="button" class="refresh-btn" :disabled="loading" @click="load">
          {{ loading ? t('payment.loading') : t('payment.refresh') }}
        </button>
        <input v-model="query" class="orders-search" type="search" :placeholder="t('payment.ordersSearch')" />
      </div>
    </header>

    <p v-if="error" class="error-banner">{{ error }}</p>
    <p v-if="success" class="success-banner">{{ success }}</p>

    <div class="orders-filters">
      <label class="filter-select">
        <span>{{ t('payment.adminOrders.statusFilter') }}</span>
        <select :value="activeFilter" @change="onFilterChange">
          <option v-for="f in filterDefs" :key="f.key" :value="f.key">{{ f.label }}</option>
        </select>
      </label>
    </div>

    <section class="orders-panel">
      <div v-if="!orders.length && !loading" class="empty-state">{{ t('payment.adminOrders.empty') }}</div>
      <table v-else class="orders-table">
        <thead>
          <tr>
            <th>{{ t('payment.adminOrders.colOrder') }}</th>
            <th>{{ t('payment.adminOrders.colUser') }}</th>
            <th>{{ t('payment.adminOrders.colProduct') }}</th>
            <th>{{ t('payment.adminOrders.colAmount') }}</th>
            <th>{{ t('payment.adminOrders.colChannel') }}</th>
            <th>{{ t('payment.adminOrders.colStatus') }}</th>
            <th>{{ t('payment.adminOrders.colCreatedAt') }}</th>
            <th>{{ t('payment.adminOrders.colAction') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.id">
            <td class="mono">{{ o.out_trade_no }}</td>
            <td>{{ userLabel(o) }}</td>
            <td>{{ o.plan_id ? `Plan #${o.plan_id}` : t('payment.tabRecharge') }}</td>
            <td>{{ fmtAmount(o.pay_amount) }}</td>
            <td>{{ channelOf(o) }}</td>
            <td><PaymentStatusBadge :status="o.status" /></td>
            <td class="mono">{{ fmtDate(o.created_at) }}</td>
            <td class="action-cell">
              <button type="button" class="action" @click="openDetail(o)">
                {{ t('payment.adminOrders.actionView') }}
              </button>
              <button
                v-if="o.status === 'PENDING'"
                type="button"
                class="action"
                :disabled="actionBusyId === o.id"
                :title="t('payment.adminOrders.simulateConfirm')"
                @click="doSimulatePaid(o)"
              >
                {{ t('payment.adminOrders.actionSimulate') }}
              </button>
              <button v-if="o.status === 'PENDING' || o.status === 'FAILED'" type="button" class="action danger" :disabled="actionBusyId === o.id" @click="doCancel(o)">
                {{ t('payment.adminOrders.actionCancel') }}
              </button>
              <button v-if="o.status === 'FAILED' || o.status === 'RECHARGING' || o.status === 'REFUND_FAILED'" type="button" class="action" :disabled="actionBusyId === o.id" @click="doRetry(o)">
                {{ t('payment.adminOrders.actionRetry') }}
              </button>
              <button v-if="o.status === 'COMPLETED' || o.status === 'PAID' || o.status === 'PARTIALLY_REFUNDED'" type="button" class="action primary" @click="openRefund(o)">
                {{ t('payment.adminOrders.actionRefund') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <footer v-if="totalPages > 1" class="orders-pager">
        <span>{{ total }} {{ t('payment.ordersTotal') }}</span>
        <div class="pager-buttons">
          <button type="button" class="pager-btn" :disabled="page <= 1" @click="page = Math.max(1, page - 1)">‹</button>
          <span>{{ page }} / {{ totalPages }}</span>
          <button type="button" class="pager-btn" :disabled="page >= totalPages" @click="page = Math.min(totalPages, page + 1)">›</button>
        </div>
      </footer>
    </section>

    <div v-if="refundModalOpen" class="modal-mask" @click.self="closeRefund">
      <div class="modal-card wide">
        <header><h3>{{ t('payment.adminOrders.refundPrompt') }}</h3></header>
        <p class="modal-sub">{{ t('payment.refund.orderNo') }} {{ refundTarget?.out_trade_no }}</p>

        <div v-if="refundPreviewLoading" class="empty-state">{{ t('payment.refund.loadingPreview') }}</div>
        <p v-else-if="refundPreviewError" class="error-banner">{{ refundPreviewError }}</p>

        <template v-else-if="refundPreview">
          <h4 class="detail-heading">{{ t('payment.refund.previewTitle') }}</h4>
          <dl class="detail-grid">
            <div v-for="row in refundMetaRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </div>
          </dl>

          <div class="refund-result">
            <span class="refund-eyebrow">{{ t('payment.refund.formulaResult') }}</span>
            <strong>{{ fmtMoney(refundPreview.refundable_pay_amount, refundPreview.currency) }}</strong>
            <small v-if="refundPreview.balance_cap_applied">{{ t('payment.refund.balanceCapApplied') }}</small>
            <small v-else>{{ t('payment.refund.formulaPay') }}</small>
          </div>

          <p v-if="!refundPreview.requestable" class="refund-blocked">
            {{ blockedLabel(refundPreview.blocked_reason) }}
          </p>

          <button
            v-if="refundBreakdownRows.length"
            type="button"
            class="refund-toggle"
            @click="refundShowBreakdown = !refundShowBreakdown"
          >
            {{ refundShowBreakdown ? t('payment.refund.collapse') : t('payment.refund.expand') }}
          </button>

          <dl v-if="refundShowBreakdown && refundBreakdownRows.length" class="refund-breakdown">
            <div v-for="row in refundBreakdownRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd class="mono">{{ fmtMoney(row.value, refundPreview.currency) }}</dd>
            </div>
          </dl>

          <section v-if="refundPreview.ledger" class="refund-ledger">
            <h4 class="detail-heading">{{ t('payment.refund.ledgerTitle') }}</h4>
            <dl class="refund-breakdown">
              <div v-for="row in refundLedgerRows" :key="row.label">
                <dt>{{ row.label }}</dt>
                <dd class="mono">{{ fmtMoney(row.value, refundPreview.currency) }}</dd>
              </div>
            </dl>
            <p v-if="refundPreview.ledger.frozen" class="refund-note">{{ t('payment.refund.ledgerFrozen') }}</p>
            <small class="refund-note">{{ t('payment.refund.bonusKeepHint') }}</small>
          </section>

          <section v-if="refundPreview.ledger_entries?.length" class="refund-ledger">
            <h4 class="detail-heading">{{ t('payment.refund.ledgerEntriesTitle') }}</h4>
            <ul class="refund-entries">
              <li v-for="e in refundPreview.ledger_entries" :key="e.id">
                <span>{{ entryLabel(e) }}</span>
                <span class="mono">{{ fmtMoney(e.amount, refundPreview.currency) }}</span>
                <span class="refund-entry-time">{{ fmtDate(e.created_at) }}</span>
              </li>
            </ul>
          </section>
        </template>

        <label class="field">
          <span>{{ t('payment.adminOrders.refundAmount') }}</span>
          <input
            v-model.number="refundAmount"
            type="number"
            step="0.01"
            min="0"
            :max="refundPreview?.refundable_pay_amount ?? refundTarget?.pay_amount ?? 0"
            @input="refundAmountTouched = true"
          />
        </label>
        <label class="field">
          <span>{{ t('payment.adminOrders.refundReason') }}</span>
          <textarea v-model="refundReason" rows="3" />
        </label>
        <label class="checkbox">
          <input v-model="refundForce" type="checkbox" />
          <span>{{ t('payment.adminOrders.refundForce') }}</span>
        </label>
        <p class="refund-hint">{{ t('payment.adminOrders.refundForceHint') }}</p>
        <footer class="modal-actions">
          <button type="button" class="btn ghost" :disabled="refundSubmitting" @click="closeRefund">{{ t('payment.cancel') }}</button>
          <button type="button" class="btn primary" :disabled="refundSubmitting || !canSubmitRefund" @click="submitRefund">
            {{ refundSubmitting ? t('payment.submitting') : t('payment.adminOrders.actionRefund') }}
          </button>
        </footer>
      </div>
    </div>

    <div v-if="detailOpen" class="modal-mask" @click.self="closeDetail">
      <div class="modal-card wide">
        <header><h3>{{ t('payment.adminOrders.detailTitle') }}</h3></header>
        <p class="modal-sub">{{ detailOrder?.out_trade_no }}</p>

        <div v-if="detailLoading" class="empty-state">{{ t('payment.loading') }}</div>

        <template v-else-if="detailOrder">
          <h4 class="detail-heading">{{ t('payment.adminOrders.detailBasic') }}</h4>
          <dl class="detail-grid">
            <div><dt>{{ t('payment.adminOrders.colUser') }}</dt><dd>{{ userLabel(detailOrder) }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.colStatus') }}</dt><dd><PaymentStatusBadge :status="detailOrder.status" /></dd></div>
            <div><dt>{{ t('payment.adminOrders.colAmount') }}</dt><dd>{{ fmtAmount(detailOrder.pay_amount) }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.colChannel') }}</dt><dd>{{ channelOf(detailOrder) }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.fieldTradeNo') }}</dt><dd class="mono">{{ detailOrder.payment_trade_no || '—' }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.fieldOrderType') }}</dt><dd>{{ detailOrder.order_type === 'subscription' ? t('payment.adminOrders.orderTypeSubscription') : t('payment.adminOrders.orderTypeRecharge') }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.fieldProduct') }}</dt><dd>{{ detailOrder.plan_id ? `Plan #${detailOrder.plan_id}` : t('payment.tabRecharge') }}</dd></div>
            <div v-if="detailOrder.subscription_group_id"><dt>{{ t('payment.adminOrders.fieldGroup') }}</dt><dd>#{{ detailOrder.subscription_group_id }} · {{ detailOrder.subscription_days || 0 }} {{ t('payment.adminOrders.fieldDays') }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.fieldPaidAt') }}</dt><dd class="mono">{{ fmtDate(detailOrder.paid_at || undefined) }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.fieldCompletedAt') }}</dt><dd class="mono">{{ fmtDate(detailOrder.completed_at || undefined) }}</dd></div>
            <div><dt>{{ t('payment.adminOrders.fieldRefundAmount') }}</dt><dd>{{ fmtAmount(detailOrder.refund_amount) }}</dd></div>
            <div v-if="detailOrder.failed_reason"><dt>{{ t('payment.adminOrders.fieldFailedReason') }}</dt><dd>{{ detailOrder.failed_reason }}</dd></div>
          </dl>

          <h4 class="detail-heading">{{ t('payment.adminOrders.detailTimeline') }}</h4>
          <div v-if="!detailLogs.length" class="empty-state">{{ t('payment.adminOrders.noAuditLog') }}</div>
          <ol v-else class="audit-timeline">
            <li v-for="(log, i) in detailLogs" :key="log.id ?? i">
              <span class="audit-dot"></span>
              <div class="audit-head">
                <strong>{{ log.action }}</strong>
                <span class="mono">{{ fmtDate(log.created_at) }}</span>
              </div>
              <div class="audit-meta">{{ log.operator }}</div>
              <pre v-if="log.detail" class="audit-detail">{{ prettyDetail(log.detail) }}</pre>
            </li>
          </ol>
        </template>

        <footer class="modal-actions">
          <button type="button" class="btn primary" @click="closeDetail">{{ t('payment.close') }}</button>
        </footer>
      </div>
    </div>
  </section>
</template>

<style scoped>
.payment-orders { width: 100%; max-width: 1280px; margin: 0 auto; padding: 12px 0 44px; }
.orders-heading { display: flex; align-items: center; justify-content: space-between; min-height: 92px; margin-bottom: 22px; gap: 24px; }
.orders-heading h1 { margin: 0; color: #f7f8fa; font-size: clamp(1.9rem, 2.4vw, 2.35rem); line-height: 1.08; font-weight: 680; letter-spacing: -.043em; }
.orders-heading p { max-width: 640px; margin: 10px 0 0; color: #858d97; font-size: .88rem; line-height: 1.6; }
.orders-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: #6ec0f5; font: 700 .67rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .13em; }
.orders-eyebrow i { width: 5px; height: 5px; border-radius: 50%; background: #6ec0f5; }
.orders-heading-actions { display: inline-flex; align-items: center; gap: 10px; }
.refresh-btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; font: 500 .8rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; transition: border-color .2s; }
.refresh-btn:hover:not(:disabled) { border-color: #3d4754; }
.refresh-btn:disabled { opacity: .6; cursor: not-allowed; }
.orders-search { width: 240px; padding: 8px 12px; border-radius: 8px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .82rem/1 ui-sans-serif, system-ui, sans-serif; }
.orders-search:focus { outline: none; border-color: #4a93c5; }
.orders-filters { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; padding-bottom: 14px; border-bottom: 1px solid #1d2128; }
.filter-select { display: inline-flex; align-items: center; gap: 8px; color: #858d97; font-size: .78rem; }
.filter-select select { padding: 8px 12px; border-radius: 8px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .82rem/1 ui-sans-serif, system-ui, sans-serif; }
.filter-select select:focus { outline: none; border-color: #4a93c5; }
.error-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(239, 68, 68, .12); border: 1px solid rgba(239, 68, 68, .35); color: #fca5a5; font-size: .85rem; }
.success-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(72, 187, 153, .12); border: 1px solid rgba(72, 187, 153, .35); color: #48bb99; font-size: .85rem; }
.orders-panel { padding: 0; border-radius: 14px; background: #11141a; border: 1px solid #1d2128; overflow: hidden; }
.orders-table { width: 100%; border-collapse: collapse; }
.orders-table th { text-align: left; padding: 12px 14px; color: #6c727b; font: 500 .72rem/1 ui-sans-serif, system-ui, sans-serif; text-transform: uppercase; letter-spacing: .08em; border-bottom: 1px solid #1d2128; background: #0f1217; }
.orders-table td { padding: 12px 14px; color: #d6dbe1; font-size: .82rem; border-bottom: 1px solid #161a20; }
.orders-table tr:last-child td { border-bottom: 0; }
.orders-table tr:hover td { background: #131820; }
.mono { font: 500 .78rem/1.2 ui-monospace, SFMono-Regular, Menlo, monospace; color: #b8bfc7; }
.action-cell { white-space: nowrap; }
.action { display: inline-block; padding: 5px 10px; margin-right: 6px; border-radius: 6px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; font: 500 .72rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; transition: all .15s; }
.action:hover:not(:disabled) { background: #1d2128; border-color: #3d4754; }
.action:disabled { opacity: .5; cursor: not-allowed; }
.action.danger { color: #fca5a5; border-color: rgba(239, 68, 68, .35); }
.action.danger:hover:not(:disabled) { background: rgba(239, 68, 68, .1); }
.action.primary { color: #6ec0f5; border-color: rgba(110, 192, 245, .35); }
.action.primary:hover:not(:disabled) { background: rgba(110, 192, 245, .1); }
.orders-pager { display: flex; align-items: center; justify-content: space-between; padding: 14px 18px; border-top: 1px solid #1d2128; color: #6c727b; font-size: .78rem; }
.pager-buttons { display: inline-flex; align-items: center; gap: 12px; }
.pager-btn { width: 30px; height: 30px; border-radius: 6px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; cursor: pointer; }
.pager-btn:hover:not(:disabled) { border-color: #3d4754; }
.pager-btn:disabled { opacity: .4; cursor: not-allowed; }
.empty-state { padding: 56px 0; text-align: center; color: #6c727b; font-size: .85rem; }
.modal-mask { position: fixed; inset: 0; background: rgba(8, 10, 14, .7); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-card { width: min(440px, 90vw); padding: 24px; background: #11141a; border: 1px solid #1d2128; border-radius: 14px; }
.modal-card h3 { margin: 0 0 4px; color: #f7f8fa; font-size: 1.05rem; font-weight: 600; }
.modal-sub { margin: 0 0 16px; color: #6c727b; font-size: .78rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.field { display: block; margin-bottom: 14px; }
.field span { display: block; margin-bottom: 6px; color: #b8bfc7; font-size: .78rem; }
.field input, .field textarea { width: 100%; padding: 8px 12px; border-radius: 7px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .85rem/1.4 ui-sans-serif, system-ui, sans-serif; box-sizing: border-box; }
.field input:focus, .field textarea:focus { outline: none; border-color: #4a93c5; }
.checkbox { display: inline-flex; align-items: center; gap: 8px; margin-bottom: 16px; color: #b8bfc7; font-size: .82rem; cursor: pointer; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; }
.btn { padding: 8px 18px; border-radius: 7px; border: 1px solid transparent; font: 500 .82rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.btn.ghost { background: #16191f; border-color: #2a2f37; color: #d6dbe1; }
.btn.ghost:hover:not(:disabled) { border-color: #3d4754; }
.btn.primary { background: #4a93c5; color: #0d0f12; }
.btn.primary:hover:not(:disabled) { background: #5fa3d5; }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.modal-card.wide { width: min(720px, 94vw); max-height: 88vh; overflow-y: auto; }
.detail-heading { margin: 18px 0 10px; color: #b8bfc7; font: 600 .74rem/1 ui-sans-serif, system-ui, sans-serif; text-transform: uppercase; letter-spacing: .09em; }
.detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 10px 18px; margin: 0; }
.detail-grid > div { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.detail-grid dt { color: #6c727b; font-size: .72rem; }
.detail-grid dd { margin: 0; color: #d6dbe1; font-size: .82rem; word-break: break-all; }
.audit-timeline { list-style: none; margin: 0; padding: 0 0 0 6px; }
.audit-timeline li { position: relative; padding: 0 0 14px 18px; border-left: 1px solid #1d2128; }
.audit-timeline li:last-child { border-left-color: transparent; padding-bottom: 0; }
.audit-dot { position: absolute; left: -4px; top: 4px; width: 7px; height: 7px; border-radius: 50%; background: #6ec0f5; }
.audit-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.audit-head strong { color: #f7f8fa; font-size: .8rem; }
.audit-head span { color: #6c727b; font-size: .72rem; }
.audit-meta { margin-top: 2px; color: #858d97; font-size: .74rem; }
.audit-detail { margin: 6px 0 0; padding: 8px 10px; border-radius: 6px; background: #0d0f12; border: 1px solid #1d2128; color: #9aa4ae; font: 400 .72rem/1.5 ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre-wrap; word-break: break-all; }
.refund-eyebrow { display: inline-flex; align-items: center; color: #6ec0f5; font: 700 .67rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .13em; text-transform: uppercase; }
.refund-result { display: flex; flex-direction: column; gap: 4px; margin-top: 16px; padding: 14px 16px; border-radius: 10px; background: rgba(110, 192, 245, .1); border: 1px solid rgba(110, 192, 245, .3); }
.refund-result strong { color: #e6f3fd; font-size: 1.35rem; font-weight: 660; font-variant-numeric: tabular-nums; }
.refund-result small { color: #858d97; font-size: .74rem; }
.refund-blocked { margin: 12px 0 0; padding: 10px 14px; border-radius: 8px; background: rgba(234, 179, 8, .12); border: 1px solid rgba(234, 179, 8, .35); color: #fcd34d; font-size: .82rem; }
.refund-toggle { margin-top: 12px; padding: 6px 12px; border-radius: 6px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; font: 500 .74rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.refund-toggle:hover { border-color: #3d4754; }
.refund-breakdown { margin: 10px 0 0; border: 1px solid #1d2128; border-radius: 10px; overflow: hidden; }
.refund-breakdown > div { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; padding: 8px 14px; border-bottom: 1px solid #161a20; }
.refund-breakdown > div:last-child { border-bottom: 0; }
.refund-breakdown dt { color: #858d97; font-size: .76rem; }
.refund-breakdown dd { margin: 0; color: #d6dbe1; font-size: .78rem; }
.refund-ledger { margin-top: 16px; }
.refund-note { display: block; margin: 8px 0 0; color: #6c727b; font-size: .74rem; }
.refund-hint { margin: -8px 0 16px; color: #6c727b; font-size: .74rem; }
.refund-entries { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.refund-entries li { display: grid; grid-template-columns: 1fr auto auto; gap: 10px; align-items: center; padding: 7px 12px; border-radius: 7px; background: #0f1217; border: 1px solid #1d2128; color: #b8bfc7; font-size: .76rem; }
.refund-entry-time { color: #6c727b; font-size: .72rem; }
@media (max-width: 900px) { .orders-heading { flex-direction: column; align-items: flex-start; } .orders-search { width: 100%; } .orders-table th, .orders-table td { padding: 8px; font-size: .72rem; } .action-cell { white-space: normal; } .action { margin-bottom: 4px; } }
</style>
