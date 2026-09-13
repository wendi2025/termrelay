<script setup lang="ts">
// Admin Orders —— 真实订单列表 + 取消/重试/退款
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  paymentAdminApi,
  type AdminPaymentOrder,
  type OrderStatus,
  type PaginatedResponse,
} from '../api/payment'
import { getErrorMessage } from '../core/api'
import PaymentStatusBadge from '../components/payment/PaymentStatusBadge.vue'

const { t } = useI18n()

type Filter = 'all' | 'paid' | 'pending' | 'failed' | 'refunded'

const loading = ref(false)
const error = ref('')
const orders = ref<AdminPaymentOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const query = ref('')
const activeFilter = ref<Filter>('all')

const filterDefs = computed<{ key: Filter; label: string; status?: OrderStatus }[]>(() => [
  { key: 'all', label: t('payment.adminOrders.filterAll') },
  { key: 'paid', label: t('payment.adminOrders.filterPaid'), status: 'COMPLETED' },
  { key: 'pending', label: t('payment.adminOrders.filterPending'), status: 'PENDING' },
  { key: 'failed', label: t('payment.adminOrders.filterFailed'), status: 'FAILED' },
  { key: 'refunded', label: t('payment.adminOrders.filterRefunded'), status: 'REFUNDED' },
])

const refundModalOpen = ref(false)
const refundTarget = ref<AdminPaymentOrder | null>(null)
const refundAmount = ref<number | null>(null)
const refundReason = ref('')
const refundForce = ref(false)
const refundSubmitting = ref(false)

const actionBusyId = ref<number | null>(null)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const def = filterDefs.value.find((f) => f.key === activeFilter.value)
    const r = (await paymentAdminApi.listOrders({
      page: page.value,
      page_size: pageSize.value,
      status: def?.status,
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
watch([activeFilter, page, pageSize], () => { load() })
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

function openRefund(o: AdminPaymentOrder): void {
  refundTarget.value = o
  refundAmount.value = o.pay_amount
  refundReason.value = ''
  refundForce.value = false
  refundModalOpen.value = true
}

function closeRefund(): void {
  refundModalOpen.value = false
  refundTarget.value = null
  refundAmount.value = null
  refundReason.value = ''
  refundForce.value = false
}

async function submitRefund(): Promise<void> {
  if (!refundTarget.value) return
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

    <nav class="orders-filters" role="tablist">
      <button
        v-for="f in filterDefs"
        :key="f.key"
        type="button"
        role="tab"
        :aria-selected="activeFilter === f.key"
        :class="['filter-btn', { active: activeFilter === f.key }]"
        @click="activeFilter = f.key"
      >
        {{ f.label }}
      </button>
    </nav>

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
      <div class="modal-card">
        <header><h3>{{ t('payment.adminOrders.refundPrompt') }}</h3></header>
        <p class="modal-sub">订单 {{ refundTarget?.out_trade_no }}</p>
        <label class="field">
          <span>{{ t('payment.adminOrders.refundAmount') }}</span>
          <input v-model.number="refundAmount" type="number" step="0.01" min="0" :max="refundTarget?.pay_amount || 0" />
        </label>
        <label class="field">
          <span>{{ t('payment.adminOrders.refundReason') }}</span>
          <textarea v-model="refundReason" rows="3" />
        </label>
        <label class="checkbox">
          <input v-model="refundForce" type="checkbox" />
          <span>{{ t('payment.adminOrders.refundForce') }}</span>
        </label>
        <footer class="modal-actions">
          <button type="button" class="btn ghost" :disabled="refundSubmitting" @click="closeRefund">{{ t('payment.cancel') }}</button>
          <button type="button" class="btn primary" :disabled="refundSubmitting" @click="submitRefund">
            {{ refundSubmitting ? t('payment.submitting') : t('payment.adminOrders.actionRefund') }}
          </button>
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
.orders-filters { display: flex; gap: 6px; margin-bottom: 16px; padding-bottom: 14px; border-bottom: 1px solid #1d2128; }
.filter-btn { padding: 7px 14px; border-radius: 7px; background: transparent; border: 1px solid transparent; color: #858d97; font: 500 .8rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; transition: all .15s; }
.filter-btn:hover { color: #d6dbe1; background: #16191f; }
.filter-btn.active { color: #f7f8fa; background: #1d2128; border-color: #2a2f37; }
.error-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(239, 68, 68, .12); border: 1px solid rgba(239, 68, 68, .35); color: #fca5a5; font-size: .85rem; }
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
@media (max-width: 900px) { .orders-heading { flex-direction: column; align-items: flex-start; } .orders-search { width: 100%; } .orders-table th, .orders-table td { padding: 8px; font-size: .72rem; } .action-cell { white-space: normal; } .action { margin-bottom: 4px; } }
</style>
