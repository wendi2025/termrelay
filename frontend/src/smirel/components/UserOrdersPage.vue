<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useSession } from '../core/session'
import { paymentApi, type PaymentOrder, type OrderStatus } from '../api/payment'
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

const refundingOrder = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const refundSubmitting = ref(false)

function openRefund(o: PaymentOrder) {
  refundingOrder.value = o
  refundReason.value = ''
}

function closeRefund() {
  refundingOrder.value = null
  refundReason.value = ''
}

async function submitRefund() {
  if (!refundingOrder.value) return
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
      <div class="modal">
        <h3>{{ t('payment.ordersActionRefundPrompt') }}</h3>
        <textarea
          v-model="refundReason"
          rows="4"
          :placeholder="t('payment.ordersActionRefundPrompt')"
        ></textarea>
        <div class="modal-actions">
          <button class="ghost" type="button" @click="closeRefund">{{ t('payment.cancel') }}</button>
          <button class="primary" type="button" :disabled="refundSubmitting" @click="submitRefund">
            {{ refundSubmitting ? t('payment.submitting') : t('payment.confirm') }}
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
}
</style>
