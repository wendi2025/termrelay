<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSession } from '../core/session'
import { paymentApi, type CheckoutInfoResponse, type LedgerBalanceBreakdown, type MethodLimits, type PaymentType, type RefundLedgerEntry, type CreateOrderResponse } from '../api/payment'
import { usePaymentCheckout } from '../composables/usePaymentCheckout'
import { getErrorMessage } from '../core/api'
import PaymentMethodsPicker from './payment/PaymentMethodsPicker.vue'
import PaymentOrderSummary from './payment/PaymentOrderSummary.vue'


const props = defineProps<{ balance: number }>()
const { t } = useI18n()
const { isAuthenticated } = useSession()

const tab = ref<'recharge' | 'subscription'>('recharge')
const info = ref<CheckoutInfoResponse | null>(null)
const loadingInfo = ref(false)
const errorInfo = ref('')

const selectedAmount = ref<number | null>(50)
const customAmount = ref('')
const presetAmounts = [10, 20, 50, 100, 200, 500]

const selectedMethod = ref<PaymentType | ''>('')
const selectedPlanId = ref<number | null>(null)

const checkout = usePaymentCheckout({
  autoNavigatePublicPage: true,
  returnUrl: window.location.origin + '/payment/result',
  isMobile: typeof navigator !== 'undefined' && /mobile|android|iphone/i.test(navigator.userAgent || ''),
})

const lastOrder = ref<CreateOrderResponse | null>(null)

const effectiveAmount = computed(() => {
  const custom = Number(customAmount.value)
  if (customAmount.value.trim() && Number.isFinite(custom) && custom > 0) return custom
  return selectedAmount.value || 0
})

const effectiveLimits = computed(() => {
  return {
    global_min: info.value?.global_min ?? 0,
    global_max: info.value?.global_max ?? Number.MAX_SAFE_INTEGER,
  }
})

const selectedMethodLimits = computed<MethodLimits | null>(() => {
  if (!selectedMethod.value || !info.value) return null
  return info.value.methods[selectedMethod.value] || null
})

const selectedMethodCurrency = computed(() => selectedMethodLimits.value?.currency || 'CNY')

const methodCount = computed(() => Object.keys(info.value?.methods || {}).length)

// 渠道列表就绪后默认选中第一个可用渠道，避免「按钮可用但没选渠道」的中间态。
watch(
  () => info.value?.methods,
  (methods) => {
    if (selectedMethod.value) return
    const keys = Object.keys(methods || {})
    if (keys.length) selectedMethod.value = keys[0] as PaymentType
  },
  { immediate: true },
)



async function loadInfo() {
  if (!isAuthenticated.value) {
    info.value = null
    return
  }
  loadingInfo.value = true
  errorInfo.value = ''
  try {
    info.value = await paymentApi.getCheckoutInfo()
  } catch (e) {
    errorInfo.value = getErrorMessage(e)
  } finally {
    loadingInfo.value = false
  }
}

async function loadAll() {
  await Promise.all([loadInfo(), loadLedger()])
}

onMounted(loadAll)

watch(() => isAuthenticated.value, loadAll)

// 充值成功后 props.balance 变化，分账构成要跟着刷新
watch(() => props.balance, loadLedger)

function chooseAmount(v: number) {
  selectedAmount.value = v
  customAmount.value = ''
}

function useCustom() {
  selectedAmount.value = null
  customAmount.value = ''
}

function pickMethod(v: PaymentType | '') {
  selectedMethod.value = v
}

async function submitRecharge() {
  if (!effectiveAmount.value || !selectedMethod.value) return
  const r = await checkout.createOrder({
    amount: effectiveAmount.value,
    payment_type: selectedMethod.value,
    order_type: 'balance',
  })
  if (r) lastOrder.value = checkout.lastResponse.value
}

// 订阅卡片上的按钮不能因为「尚未选择支付方式」而变成点了没反应的死按钮。
function ensureMethod(): PaymentType | '' {
  if (selectedMethod.value) return selectedMethod.value
  const keys = Object.keys(info.value?.methods || {})
  if (keys.length) selectedMethod.value = keys[0] as PaymentType
  return selectedMethod.value
}

async function submitPlan(planId: number) {
  const method = ensureMethod()
  if (!method) return
  selectedPlanId.value = planId
  const plan = info.value?.plans.find((p) => p.id === planId)
  if (!plan) return
  const r = await checkout.createOrder({
    amount: plan.price,
    payment_type: method,
    order_type: 'subscription',
    plan_id: plan.id,
  })
  if (r) lastOrder.value = checkout.lastResponse.value
}

const plans = computed(() => info.value?.plans || [])
const sortedPlans = computed(() => {
  const list = [...plans.value]
  return list.sort((a, b) => a.price - b.price)
})

// 订阅 Tab 的渠道可用性按「当前选中套餐，否则最便宜套餐」判断，
// 这样在没有选中任何套餐时也不会把渠道误标成不可用。
const subscriptionPickerAmount = computed(() => {
  const picked = sortedPlans.value.find((p) => p.id === selectedPlanId.value)
  if (picked) return picked.price
  return sortedPlans.value.length ? sortedPlans.value[0].price : 0
})

const balanceCurrency = computed(() => info.value?.subscription_usd_to_cny_rate ? 'CNY' : 'CNY')

const fmtBalance = computed(() => {
  const cur = balanceCurrency.value
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: cur, maximumFractionDigits: 2 }).format(props.balance || 0)
  } catch {
    return `${cur} ${(props.balance || 0).toFixed(2)}`
  }
})

function onPlanCardClick(planId: number) {
  selectedPlanId.value = planId
  if (!selectedMethod.value) {
    // 预选第一个可用方法
    const first = (Object.values(info.value?.methods || {}) as MethodLimits[]).find(() => true)
    if (first) {
      const k = Object.entries(info.value?.methods || {}).find(([, v]) => v === first)?.[0]
      if (k) selectedMethod.value = k as PaymentType
    }
  }
}
// ---- 余额分账（方案 9.2：充值本金 / 赠送余额） ----
// 只读展示、不参与扣费；扣费顺序仍由后端账本决定（先本金后赠送）。
const ledgerBreakdown = ref<LedgerBalanceBreakdown | null>(null)
const ledgerEntries = ref<RefundLedgerEntry[]>([])
const ledgerLoading = ref(false)
const ledgerError = ref('')
const ledgerShowEntries = ref(false)

async function loadLedger() {
  if (!isAuthenticated.value) {
    ledgerBreakdown.value = null
    ledgerEntries.value = []
    return
  }
  ledgerLoading.value = true
  ledgerError.value = ''
  try {
    const r = await paymentApi.getLedger(50)
    ledgerBreakdown.value = r?.breakdown || null
    ledgerEntries.value = r?.entries || []
  } catch (e) {
    ledgerError.value = getErrorMessage(e)
    ledgerBreakdown.value = null
    ledgerEntries.value = []
  } finally {
    ledgerLoading.value = false
  }
}

function fmtMoney(v: number | null | undefined, currency = 'CNY'): string {
  const n = typeof v === 'number' && Number.isFinite(v) ? v : 0
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency, maximumFractionDigits: 2 }).format(n)
  } catch {
    return `${currency} ${n.toFixed(2)}`
  }
}

function fmtDate(s: string | undefined): string {
  if (!s) return '—'
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : d.toLocaleString()
}

function entryLabel(e: RefundLedgerEntry): string {
  const type = e.entry_type === 'bonus' ? t('payment.refund.ledgerEntryBonus') : t('payment.refund.ledgerEntryPrincipal')
  const dir = e.direction === 'debit' ? t('payment.refund.ledgerDebit') : t('payment.refund.ledgerCredit')
  return `${type} · ${dir}`
}

function ledgerEntryAmount(e: RefundLedgerEntry): string {
  return fmtMoney(e.direction === 'debit' ? -e.amount : e.amount)
}

const frozenTotal = computed(() => {
  const b = ledgerBreakdown.value
  if (!b) return 0
  return (b.frozen_principal_left || 0) + (b.frozen_bonus_left || 0)
})

// 账本上线前的历史余额没有分录，缺口显式说明，避免用户以为余额算错了
const ledgerGapHint = computed(() => {
  const gap = ledgerBreakdown.value?.ledger_gap || 0
  if (gap <= 0.005) return ''
  return t('payment.ledger.gapHint', { value: fmtMoney(gap) })
})

</script>

<template>
  <div class="billing-page">
    <div class="billing-mode" role="tablist" :aria-label="t('payment.helpTitle')">
      <button type="button" :class="{ active: tab === 'recharge' }" @click="tab = 'recharge'">
        <span>{{ t('payment.tabRecharge') }}</span>
      </button>
      <button type="button" :class="{ active: tab === 'subscription' }" @click="tab = 'subscription'">
        <span>{{ t('payment.tabSubscription') }}</span>
      </button>
    </div>

    <div v-if="loadingInfo" class="loading">{{ t('payment.loading') }}</div>
    <div v-else-if="errorInfo" class="error">{{ errorInfo }}</div>

    <template v-else-if="tab === 'recharge'">
      <section class="balance-card">
        <div class="balance-label">
          <span class="eyebrow">{{ t('payment.balance') }}</span>
          <strong>{{ fmtBalance }}</strong>
          <p>{{ t('payment.balanceHint') }}</p>
        </div>
        <RouterLink class="orders-link" to="/orders">
          {{ t('payment.ordersTitle') }} →
        </RouterLink>
      </section>
      <section v-if="ledgerBreakdown" class="ledger-card">
        <header>
          <span class="eyebrow">{{ t('payment.ledger.title') }}</span>
          <button
            v-if="ledgerEntries.length"
            type="button"
            class="ledger-toggle"
            @click="ledgerShowEntries = !ledgerShowEntries"
          >
            {{ t('payment.ledger.entriesTitle') }} ({{ ledgerEntries.length }})
            <span class="ledger-toggle-mark">{{ ledgerShowEntries ? '−' : '+' }}</span>
          </button>
        </header>
        <dl class="ledger-grid">
          <div>
            <dt>{{ t('payment.ledger.principal') }}</dt>
            <dd class="mono">{{ fmtMoney(ledgerBreakdown.active_principal_left) }}</dd>
          </div>
          <div>
            <dt>{{ t('payment.ledger.bonus') }}</dt>
            <dd class="mono">{{ fmtMoney(ledgerBreakdown.active_bonus_left) }}</dd>
          </div>
          <div v-if="frozenTotal > 0">
            <dt>{{ t('payment.ledger.frozen') }}</dt>
            <dd class="mono">{{ fmtMoney(frozenTotal) }}</dd>
          </div>
        </dl>
        <p v-if="frozenTotal > 0" class="ledger-note">{{ t('payment.ledger.frozenHint') }}</p>
        <p v-if="ledgerGapHint" class="ledger-note">{{ ledgerGapHint }}</p>
        <ul v-if="ledgerShowEntries && ledgerEntries.length" class="ledger-entries">
          <li v-for="e in ledgerEntries" :key="e.id">
            <span>{{ entryLabel(e) }}</span>
            <span class="mono">{{ ledgerEntryAmount(e) }}</span>
            <span class="ledger-entry-time">{{ fmtDate(e.created_at) }}</span>
          </li>
        </ul>
        <p v-else-if="ledgerShowEntries" class="ledger-note">{{ t('payment.ledger.entriesEmpty') }}</p>
      </section>
      <p v-else-if="ledgerError" class="ledger-note ledger-error">{{ t('payment.ledger.loadFailed') }}</p>

      <section class="amount-card">
        <span class="eyebrow">{{ t('payment.chooseAmount') }}</span>
        <small>{{ t('payment.chooseAmountHint') }}</small>
        <div class="amount-grid">
          <button
            v-for="v in presetAmounts"
            :key="v"
            type="button"
            :class="{ active: selectedAmount === v && !customAmount }"
            @click="chooseAmount(v)"
          >
            {{ v }}
          </button>
          <button type="button" :class="{ active: customAmount.length > 0 }" @click="useCustom">
            {{ t('payment.customAmount') }}
          </button>
        </div>
        <input
          v-if="customAmount.length > 0 || selectedAmount === null"
          v-model="customAmount"
          type="number"
          inputmode="decimal"
          :placeholder="t('payment.customPlaceholder')"
          class="custom-input"
        />
      </section>

      <PaymentMethodsPicker
        v-model="selectedMethod"
        :methods="info?.methods || {}"
        :limits="effectiveLimits"
        :amount="effectiveAmount"
        @update:model-value="pickMethod"
      />

      <PaymentOrderSummary
        :amount="effectiveAmount"
        :fee-rate="selectedMethodLimits?.fee_rate || 0"
        :pay-amount="(selectedMethodLimits && selectedMethodLimits.fee_rate > 0) ? effectiveAmount * (1 - selectedMethodLimits.fee_rate / 100) : effectiveAmount"
        :currency="selectedMethodCurrency"
      />

      <p v-if="checkout.error.value" class="error">{{ checkout.error.value }}</p>

      <button
        class="primary"
        type="button"
        :disabled="!effectiveAmount || !selectedMethod || checkout.submitting.value"
        @click="submitRecharge"
      >
        {{ checkout.submitting.value ? t('payment.submitting') : t('payment.submit') }}
      </button>

      <small class="secure">{{ t('payment.secure') }}</small>
    </template>

    <template v-else>
      <section v-if="sortedPlans.length === 0" class="empty">
        <p>{{ t('payment.planHint') }}</p>
      </section>
      <section v-else class="plan-list">
        <article
          v-for="plan in sortedPlans"
          :key="plan.id"
          class="plan-card"
          :class="{ selected: selectedPlanId === plan.id }"
          @click="onPlanCardClick(plan.id)"
        >
          <header>
            <strong>{{ plan.name }}</strong>
            <span class="price">
              {{ (plan.currency || 'CNY') }} {{ plan.price.toFixed(2) }}
            </span>
          </header>
          <p v-if="plan.description">{{ plan.description }}</p>
          <ul v-if="plan.features && plan.features.length">
            <li v-for="(f, i) in plan.features" :key="i">{{ f }}</li>
          </ul>
          <button
            class="primary"
            type="button"
            :disabled="checkout.submitting.value || methodCount === 0"
            @click.stop="submitPlan(plan.id)"
          >
            {{ checkout.submitting.value && selectedPlanId === plan.id ? t('payment.submitting') : t('payment.planBuy') }}
          </button>
        </article>
      </section>

      <PaymentMethodsPicker
        v-if="methodCount > 0"
        v-model="selectedMethod"
        :methods="info?.methods || {}"
        :limits="effectiveLimits"
        :amount="subscriptionPickerAmount"
      />

      <PaymentOrderSummary
        v-if="selectedPlanId"
        :amount="sortedPlans.find((p) => p.id === selectedPlanId)?.price || 0"
        :fee-rate="selectedMethodLimits?.fee_rate || 0"
        :currency="sortedPlans.find((p) => p.id === selectedPlanId)?.currency || 'CNY'"
        :product-name="sortedPlans.find((p) => p.id === selectedPlanId)?.name"
      />

      <p v-if="checkout.error.value" class="error">{{ checkout.error.value }}</p>
    </template>
  </div>
</template>

<style scoped>
.billing-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 0 0 36px;
}
.billing-mode {
  display: inline-flex;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  padding: 4px;
  width: fit-content;
  background: rgba(15, 18, 24, 0.7);
}
.billing-mode button {
  padding: 8px 18px;
  border-radius: 9px;
  border: 0;
  background: transparent;
  color: rgba(255, 255, 255, 0.55);
  font-family: inherit;
  font-size: 0.84rem;
  cursor: pointer;
}
.billing-mode button.active {
  background: rgba(120, 175, 230, 0.18);
  color: #fff;
  font-weight: 600;
}
.loading,
.error {
  text-align: center;
  padding: 24px;
  border-radius: 12px;
  background: rgba(15, 18, 24, 0.78);
  border: 1px solid rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.65);
  font-size: 0.86rem;
}
.error {
  background: rgba(244, 139, 139, 0.06);
  border-color: rgba(244, 139, 139, 0.32);
  color: #f48b8b;
}
.balance-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  background: linear-gradient(180deg, rgba(18, 22, 28, 0.88), rgba(12, 16, 22, 0.94));
  padding: 18px 22px;
}
.balance-label .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.balance-label strong {
  display: block;
  font-size: 1.7rem;
  color: rgba(255, 255, 255, 0.95);
  font-variant-numeric: tabular-nums;
  margin-top: 4px;
  font-weight: 640;
}
.balance-label p {
  margin: 4px 0 0;
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.45);
}
.orders-link {
  font-size: 0.78rem;
  color: #79c4f5;
  text-decoration: none;
}
.amount-card {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  background: rgba(15, 18, 24, 0.78);
  padding: 18px 22px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.amount-card .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.amount-card small {
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.42);
}
.amount-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 8px;
}
.amount-grid button {
  min-height: 42px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.02);
  color: rgba(255, 255, 255, 0.78);
  font-family: inherit;
  font-size: 0.86rem;
  cursor: pointer;
}
.amount-grid button.active {
  border-color: #79c4f5;
  background: rgba(120, 175, 230, 0.12);
  color: #fff;
}
.custom-input {
  width: 100%;
  height: 44px;
  padding: 0 14px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(15, 18, 24, 0.95);
  color: #fff;
  font-size: 0.95rem;
  font-family: inherit;
}
.primary {
  background: #79c4f5;
  border: 1px solid #79c4f5;
  border-radius: 10px;
  height: 46px;
  font-size: 0.92rem;
  font-weight: 600;
  color: #071019;
  cursor: pointer;
  font-family: inherit;
}
.primary:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.secure {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}
.empty {
  padding: 28px;
  border-radius: 12px;
  background: rgba(15, 18, 24, 0.78);
  border: 1px solid rgba(255, 255, 255, 0.06);
  text-align: center;
  color: rgba(255, 255, 255, 0.55);
  font-size: 0.84rem;
}
.plan-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.plan-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(18, 22, 28, 0.88), rgba(12, 16, 22, 0.94));
  padding: 18px 18px 16px;
  cursor: pointer;
}
.plan-card.selected {
  border-color: #79c4f5;
}
.plan-card header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.plan-card header strong {
  font-size: 0.98rem;
  color: rgba(255, 255, 255, 0.92);
}
.plan-card header .price {
  font-size: 1rem;
  color: #79c4f5;
  font-variant-numeric: tabular-nums;
  font-weight: 620;
}
.plan-card p {
  margin: 0;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.55);
  line-height: 1.6;
}
.plan-card ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.plan-card ul li {
  font-size: 0.74rem;
  color: rgba(255, 255, 255, 0.6);
  padding-left: 16px;
  position: relative;
}
.plan-card ul li::before {
  content: '·';
  position: absolute;
  left: 4px;
  color: #79c4f5;
}
.plan-card .primary {
  margin-top: auto;
}
.mono {
  font-family: ui-monospace, monospace;
  font-variant-numeric: tabular-nums;
}
.ledger-card {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  background: linear-gradient(180deg, rgba(18, 22, 28, 0.88), rgba(12, 16, 22, 0.94));
  padding: 16px 22px;
}
.ledger-card header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.ledger-card .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.ledger-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 10px 18px;
  margin: 12px 0 0;
}
.ledger-grid dt {
  font-size: 0.68rem;
  color: rgba(255, 255, 255, 0.45);
}
.ledger-grid dd {
  margin: 3px 0 0;
  font-size: 1rem;
  color: rgba(255, 255, 255, 0.92);
  font-variant-numeric: tabular-nums;
}
.ledger-note {
  margin: 10px 0 0;
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.45);
}
.ledger-error {
  color: #f48b8b;
}
.ledger-toggle {
  height: 26px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.72);
  font-family: inherit;
  font-size: 0.72rem;
  cursor: pointer;
}
.ledger-toggle:hover {
  background: rgba(255, 255, 255, 0.08);
}
.ledger-toggle-mark {
  margin-left: 6px;
  color: #79c4f5;
}
.ledger-entries {
  list-style: none;
  margin: 12px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ledger-entries li {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
  align-items: center;
  padding: 7px 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  font-size: 0.74rem;
  color: rgba(255, 255, 255, 0.78);
}
.ledger-entry-time {
  font-size: 0.68rem;
  color: rgba(255, 255, 255, 0.4);
}
</style>
