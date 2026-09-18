<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSession } from '../core/session'
import { paymentApi, type CheckoutInfoResponse, type CheckoutPlan, type LedgerBalanceBreakdown, type MethodLimits, type PaymentType, type RefundLedgerEntry, type CreateOrderResponse } from '../api/payment'
import { usePaymentCheckout } from '../composables/usePaymentCheckout'
import { getErrorMessage } from '../core/api'
import PaymentMethodsPicker from './payment/PaymentMethodsPicker.vue'
import PaymentOrderSummary from './payment/PaymentOrderSummary.vue'


const props = defineProps<{ balance: number }>()
const { t, locale } = useI18n()
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

const isZh = computed(() => String(locale.value || '').toLowerCase().startsWith('zh'))

function planText(plan: CheckoutPlan): string {
  return [plan.description || '', ...(plan.features || [])].join(' ')
}

function parsePlanNumber(plan: CheckoutPlan, pattern: RegExp): number | null {
  const match = planText(plan).match(pattern)
  if (!match?.[1]) return null
  const value = Number(match[1].replace(/,/g, ''))
  return Number.isFinite(value) && value > 0 ? value : null
}

function planPaygReference(plan: CheckoutPlan): number | null {
  return parsePlanNumber(plan, /按量价\s*[≈~]?\s*[¥￥]\s*([\d,.]+)/i)
}

function planOfficialQuotaUsd(plan: CheckoutPlan): number | null {
  return parsePlanNumber(plan, /官方额度\s*[：:]\s*\$\s*([\d,.]+)/i)
}

function planOfficialReferenceCny(plan: CheckoutPlan): number | null {
  const quota = planOfficialQuotaUsd(plan)
  const rate = Number(info.value?.subscription_usd_to_cny_rate || 0)
  if (!quota || !Number.isFinite(rate) || rate <= 0) return null
  return quota * rate
}

function savingsPercent(reference: number | null, price: number): number | null {
  if (!reference || reference <= 0 || price >= reference) return null
  return Math.round((1 - price / reference) * 1000) / 10
}

function planPaygSavings(plan: CheckoutPlan): number | null {
  return savingsPercent(planPaygReference(plan), plan.price)
}

function planOfficialSavings(plan: CheckoutPlan): number | null {
  return savingsPercent(planOfficialReferenceCny(plan), plan.price)
}

function pct(value: number | null): string {
  if (value === null) return ''
  return Number.isInteger(value) ? String(value) : value.toFixed(1)
}

function money(value: number | null, digits = 0): string {
  if (value === null) return '—'
  return value.toLocaleString(undefined, { maximumFractionDigits: digits, minimumFractionDigits: digits })
}

function planValidity(plan: CheckoutPlan): string {
  const days = Number(plan.validity_days || 0)
  const unit = String(plan.validity_unit || '').toLowerCase()
  if (unit === 'day' || unit === 'days') return isZh.value ? `${days}天` : `${days} days`
  if (unit === 'month' || unit === 'months') return isZh.value ? `${days}个月` : `${days} months`
  return `${days} ${plan.validity_unit || ''}`.trim()
}

function planBadge(plan: CheckoutPlan): string {
  const group = String(plan.group_name || '').replace(/^Smirel\s*[·・-]\s*/i, '').trim()
  if (group) return group
  if (plan.group_platform) return String(plan.group_platform).toUpperCase()
  return 'SMIREL'
}

function planTier(plan: CheckoutPlan): string {
  const text = `${plan.name} ${plan.description}`
  if (/企业/.test(text)) return isZh.value ? '企业权益' : 'Enterprise'
  if (/标准|专业/.test(text)) return isZh.value ? '推荐' : 'Recommended'
  if (/轻享/.test(text)) return isZh.value ? '轻享' : 'Lite'
  return isZh.value ? '订阅' : 'Plan'
}

function displayPlanFeatures(plan: CheckoutPlan): string[] {
  return (plan.features || [])
    .map((item) => String(item || '').trim())
    .filter((item) => item && !/官方额度\s*[：:]/.test(item))
    .slice(0, 4)
}

function enterpriseValueCopy(plan: CheckoutPlan): string {
  const text = planText(plan)
  if (/企业|20\s*席|独享|30\+/.test(text)) {
    return isZh.value ? '多席位 · 高并发 · 企业权益' : 'Multi-seat · High concurrency · Enterprise benefits'
  }
  return isZh.value ? '固定周期额度 · 即时开通' : 'Fixed-period quota · Instant activation'
}

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
    const first = (Object.values(info.value?.methods || {}) as MethodLimits[]).find(() => true)
    if (first) {
      const k = Object.entries(info.value?.methods || {}).find(([, v]) => v === first)?.[0]
      if (k) selectedMethod.value = k as PaymentType
    }
  }
}

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
      <section class="wallet-card">
        <div class="wallet-balance">
          <span class="wallet-kicker"><i></i>{{ t('payment.balance') }}</span>
          <strong>{{ fmtBalance }}</strong>
          <p>{{ t('payment.balanceHint') }}</p>
        </div>

        <dl v-if="ledgerBreakdown" class="wallet-stats">
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

        <div class="wallet-actions">
          <RouterLink class="orders-link" to="/orders">
            {{ t('payment.ordersTitle') }} <span>→</span>
          </RouterLink>
          <button v-if="ledgerEntries.length" type="button" class="ledger-toggle" @click="ledgerShowEntries = !ledgerShowEntries">
            {{ t('payment.ledger.entriesTitle') }}
            <span class="ledger-toggle-mark">{{ ledgerShowEntries ? '−' : '+' }}</span>
          </button>
        </div>

        <p v-if="frozenTotal > 0" class="ledger-note wallet-note">{{ t('payment.ledger.frozenHint') }}</p>
        <p v-if="ledgerGapHint" class="ledger-note wallet-note">{{ ledgerGapHint }}</p>
        <ul v-if="ledgerShowEntries && ledgerEntries.length" class="ledger-entries wallet-entries">
          <li v-for="e in ledgerEntries" :key="e.id">
            <span>{{ entryLabel(e) }}</span>
            <span class="mono">{{ ledgerEntryAmount(e) }}</span>
            <span class="ledger-entry-time">{{ fmtDate(e.created_at) }}</span>
          </li>
        </ul>
      </section>
      <p v-if="ledgerError" class="ledger-note ledger-error">{{ t('payment.ledger.loadFailed') }}</p>

      <div class="checkout-layout">
        <div class="checkout-main">
          <section class="amount-card">
            <header class="checkout-section-head">
              <div class="section-heading-copy">
                <span class="section-index">01</span>
                <div>
                  <strong>{{ t('payment.chooseAmount') }}</strong>
                  <small>{{ t('payment.chooseAmountHint') }}</small>
                </div>
              </div>
              <span class="amount-preview">{{ fmtMoney(effectiveAmount) }}</span>
            </header>

            <div class="amount-grid">
              <button
                v-for="v in presetAmounts"
                :key="v"
                type="button"
                :class="{ active: selectedAmount === v && !customAmount }"
                @click="chooseAmount(v)"
              >
                <span>¥</span>{{ v }}
              </button>
              <button type="button" :class="{ active: selectedAmount === null }" @click="useCustom">
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

          <section class="method-shell">
            <header class="checkout-section-head">
              <div class="section-heading-copy">
                <span class="section-index">02</span>
                <div>
                  <strong>{{ t('payment.methodLabel') }}</strong>
                  <small>{{ t('payment.methodDescription') }}</small>
                </div>
              </div>
            </header>
            <PaymentMethodsPicker
              v-model="selectedMethod"
              :methods="info?.methods || {}"
              :limits="effectiveLimits"
              :amount="effectiveAmount"
              @update:model-value="pickMethod"
            />
          </section>
        </div>

        <aside class="checkout-sidebar">
          <header class="checkout-section-head summary-heading">
            <div class="section-heading-copy">
              <span class="section-index">03</span>
              <div>
                <strong>{{ t('payment.summary') }}</strong>
                <small>{{ t('payment.secure') }}</small>
              </div>
            </div>
          </header>

          <PaymentOrderSummary
            :amount="effectiveAmount"
            :fee-rate="selectedMethodLimits?.fee_rate || 0"
            :pay-amount="(selectedMethodLimits && selectedMethodLimits.fee_rate > 0) ? effectiveAmount * (1 - selectedMethodLimits.fee_rate / 100) : effectiveAmount"
            :currency="selectedMethodCurrency"
          />

          <p v-if="checkout.error.value" class="error checkout-error">{{ checkout.error.value }}</p>

          <button
            class="primary checkout-submit"
            type="button"
            :disabled="!effectiveAmount || !selectedMethod || checkout.submitting.value"
            @click="submitRecharge"
          >
            <span>{{ checkout.submitting.value ? t('payment.submitting') : t('payment.submit') }}</span>
            <b v-if="effectiveAmount">{{ fmtMoney(effectiveAmount, selectedMethodCurrency) }}</b>
          </button>
          <small class="secure">{{ t('payment.secure') }}</small>
        </aside>
      </div>
    </template>

    <template v-else>
      <section v-if="sortedPlans.length === 0" class="empty"><p>{{ t('payment.planHint') }}</p></section>
      <section v-else class="plan-list pricing-grid">
        <article v-for="plan in sortedPlans" :key="plan.id" class="pricing-card" :class="{ selected: selectedPlanId === plan.id }" @click="onPlanCardClick(plan.id)">
          <div class="pricing-topline">
            <span class="pricing-brand">{{ planBadge(plan) }}</span>
            <span class="pricing-tier" :class="{ hot: /推荐|Recommended/.test(planTier(plan)) }">{{ planTier(plan) }}</span>
          </div>
          <div class="pricing-title"><h3>{{ plan.name }}</h3><p v-if="plan.description">{{ plan.description }}</p></div>
          <div class="pricing-price">
            <span class="pricing-currency">{{ plan.currency || 'CNY' }}</span>
            <strong>{{ money(plan.price, 0) }}</strong>
            <span class="pricing-period">/ {{ planValidity(plan) }}</span>
          </div>
          <div v-if="planPaygReference(plan)" class="pricing-reference">
            <span>{{ isZh ? '按量约' : 'PAYG approx.' }}</span>
            <s>¥{{ money(planPaygReference(plan), 0) }}</s>
            <b v-if="planPaygSavings(plan)">{{ isZh ? `立省 ¥${money((planPaygReference(plan) || 0) - plan.price, 0)}` : `Save ¥${money((planPaygReference(plan) || 0) - plan.price, 0)}` }}</b>
          </div>
          <div class="pricing-savings" :class="{ neutral: !planPaygSavings(plan) && !planOfficialSavings(plan) }">
            <div v-if="planPaygSavings(plan)"><i></i><strong>{{ isZh ? `比按量计费省 ${pct(planPaygSavings(plan))}%` : `${pct(planPaygSavings(plan))}% less than PAYG` }}</strong></div>
            <div v-if="planOfficialSavings(plan)"><i></i><strong>{{ isZh ? `比官方 API 折算省 ${pct(planOfficialSavings(plan))}%` : `${pct(planOfficialSavings(plan))}% less than official API equivalent` }}</strong></div>
            <div v-if="!planPaygSavings(plan) && !planOfficialSavings(plan)"><i></i><strong>{{ enterpriseValueCopy(plan) }}</strong></div>
          </div>
          <dl class="pricing-facts">
            <div v-if="plan.rate_multiplier"><dt>{{ isZh ? '计费倍率' : 'Rate multiplier' }}</dt><dd>×{{ Number(plan.rate_multiplier).toFixed(2) }}</dd></div>
            <div v-if="plan.monthly_limit_usd"><dt>{{ isZh ? '月额度' : 'Monthly quota' }}</dt><dd>${{ money(Number(plan.monthly_limit_usd), 0) }}</dd></div>
            <div v-else-if="planOfficialQuotaUsd(plan)"><dt>{{ isZh ? '官方额度' : 'Official quota' }}</dt><dd>${{ money(planOfficialQuotaUsd(plan), 0) }}</dd></div>
            <div><dt>{{ isZh ? '有效期' : 'Validity' }}</dt><dd>{{ planValidity(plan) }}</dd></div>
          </dl>
          <ul v-if="displayPlanFeatures(plan).length" class="pricing-benefits">
            <li v-for="(f, i) in displayPlanFeatures(plan)" :key="i"><b>✓</b><span>{{ f }}</span></li>
          </ul>
          <button class="primary pricing-cta" type="button" :disabled="checkout.submitting.value || methodCount === 0" @click.stop="submitPlan(plan.id)">
            {{ checkout.submitting.value && selectedPlanId === plan.id ? t('payment.submitting') : (isZh ? '立即开通 →' : 'Subscribe now →') }}
          </button>
          <small class="pricing-footnote">{{ isZh ? '即时开通 · 订单可追踪' : 'Instant activation · Trackable order' }}</small>
        </article>
      </section>

      <PaymentMethodsPicker v-if="methodCount > 0" v-model="selectedMethod" :methods="info?.methods || {}" :limits="effectiveLimits" :amount="subscriptionPickerAmount" />
      <PaymentOrderSummary v-if="selectedPlanId" :amount="sortedPlans.find((p) => p.id === selectedPlanId)?.price || 0" :fee-rate="selectedMethodLimits?.fee_rate || 0" :currency="sortedPlans.find((p) => p.id === selectedPlanId)?.currency || 'CNY'" :product-name="sortedPlans.find((p) => p.id === selectedPlanId)?.name" />
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
  color: var(--billing-text, #f4f7fa);
}

.billing-mode {
  display: inline-flex;
  width: fit-content;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 12px;
  padding: 4px;
  background: var(--billing-surface, rgba(15,18,24,.78));
}

.billing-mode button {
  padding: 8px 18px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--billing-muted, rgba(255,255,255,.55));
  font-family: inherit;
  font-size: .84rem;
  cursor: pointer;
}

.billing-mode button.active {
  background: var(--billing-accent-soft, rgba(120,175,230,.12));
  color: var(--billing-accent-strong, #79c4f5);
  font-weight: 680;
}

.loading,
.error {
  padding: 20px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 14px;
  background: var(--billing-surface, rgba(15,18,24,.78));
  color: var(--billing-muted, rgba(255,255,255,.65));
  font-size: .82rem;
  text-align: center;
}

.error {
  border-color: color-mix(in srgb, var(--billing-danger, #f48b8b) 32%, transparent);
  background: color-mix(in srgb, var(--billing-danger, #f48b8b) 7%, transparent);
  color: var(--billing-danger, #f48b8b);
}

.wallet-card {
  position: relative;
  display: grid;
  grid-template-columns: minmax(240px, 1.2fr) minmax(280px, 1fr) auto;
  align-items: center;
  gap: 22px;
  overflow: hidden;
  min-height: 132px;
  padding: 22px 24px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 18px;
  background:
    radial-gradient(circle at 0 0, color-mix(in srgb, var(--billing-accent, #79c4f5) 10%, transparent), transparent 34%),
    var(--billing-surface, rgba(15,18,24,.78));
  box-shadow: var(--billing-shadow, 0 18px 48px rgba(0,0,0,.15));
}

.wallet-card::after {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: linear-gradient(180deg, var(--billing-accent, #79c4f5), color-mix(in srgb, var(--billing-accent, #79c4f5) 12%, transparent));
}

.wallet-balance {
  min-width: 0;
}

.wallet-kicker {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--billing-muted, rgba(255,255,255,.5));
  font-size: .68rem;
  font-weight: 680;
}

.wallet-kicker i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--billing-success, #61d9b1);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--billing-success, #61d9b1) 12%, transparent);
}

.wallet-balance > strong {
  display: block;
  margin-top: 8px;
  color: var(--billing-text, #f4f7fa);
  font-size: clamp(1.85rem, 3vw, 2.35rem);
  font-weight: 760;
  letter-spacing: -.045em;
  font-variant-numeric: tabular-nums;
}

.wallet-balance p {
  margin: 5px 0 0;
  color: var(--billing-muted, rgba(255,255,255,.45));
  font-size: .72rem;
}

.wallet-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(90px, 1fr));
  gap: 10px;
  margin: 0;
}

.wallet-stats > div {
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.07));
  border-radius: 12px;
  background: var(--billing-surface-soft, rgba(255,255,255,.025));
}

.wallet-stats dt {
  color: var(--billing-muted, rgba(255,255,255,.45));
  font-size: .66rem;
}

.wallet-stats dd {
  margin: 5px 0 0;
  color: var(--billing-text-soft, rgba(255,255,255,.88));
  font-size: .9rem;
  font-weight: 650;
}

.wallet-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.orders-link,
.ledger-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  padding: 0 11px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 9px;
  background: var(--billing-surface-soft, rgba(255,255,255,.03));
  color: var(--billing-text-soft, rgba(255,255,255,.72));
  font-family: inherit;
  font-size: .72rem;
  text-decoration: none;
  cursor: pointer;
  transition: border-color .16s ease, background .16s ease, color .16s ease, transform .16s ease;
}

.orders-link:hover,
.ledger-toggle:hover {
  transform: translateY(-1px);
  border-color: var(--billing-border-strong, rgba(121,196,245,.35));
  background: var(--billing-accent-soft, rgba(121,196,245,.08));
  color: var(--billing-accent-strong, #79c4f5);
}

.orders-link span,
.ledger-toggle-mark {
  margin-left: 6px;
  color: var(--billing-accent, #79c4f5);
}

.wallet-note,
.wallet-entries {
  grid-column: 1 / -1;
}

.checkout-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 330px;
  gap: 18px;
  align-items: start;
}

.checkout-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 18px;
}

.amount-card,
.method-shell,
.checkout-sidebar {
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 16px;
  background: var(--billing-surface, rgba(15,18,24,.78));
  box-shadow: 0 10px 30px rgba(26, 42, 58, 0.035);
}

.amount-card,
.method-shell {
  padding: 20px;
}

.checkout-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.section-heading-copy {
  display: flex;
  align-items: center;
  gap: 11px;
  min-width: 0;
}

.section-heading-copy > div {
  min-width: 0;
}

.section-heading-copy strong {
  display: block;
  color: var(--billing-text, #f4f7fa);
  font-size: .82rem;
  font-weight: 700;
}

.section-heading-copy small {
  display: block;
  margin-top: 3px;
  color: var(--billing-muted, rgba(255,255,255,.45));
  font-size: .68rem;
  line-height: 1.45;
}

.section-index {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid var(--billing-border-strong, rgba(121,196,245,.26));
  border-radius: 9px;
  background: var(--billing-accent-soft, rgba(121,196,245,.08));
  color: var(--billing-accent-strong, #79c4f5);
  font-size: .64rem;
  font-weight: 760;
  letter-spacing: .04em;
}

.amount-preview {
  flex: 0 0 auto;
  color: var(--billing-accent-strong, #79c4f5);
  font-size: .82rem;
  font-weight: 720;
  font-variant-numeric: tabular-nums;
}

.amount-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(76px, 1fr));
  gap: 9px;
}

.amount-grid button {
  min-height: 46px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 11px;
  background: var(--billing-surface-soft, rgba(255,255,255,.02));
  color: var(--billing-text-soft, rgba(255,255,255,.78));
  font-family: inherit;
  font-size: .82rem;
  font-weight: 620;
  cursor: pointer;
  transition: transform .14s ease, border-color .14s ease, background .14s ease, color .14s ease, box-shadow .14s ease;
}

.amount-grid button span {
  margin-right: 2px;
  color: var(--billing-subtle, rgba(255,255,255,.36));
  font-size: .66rem;
}

.amount-grid button:hover {
  transform: translateY(-1px);
  border-color: var(--billing-border-strong, rgba(121,196,245,.34));
  background: var(--billing-accent-soft, rgba(121,196,245,.06));
}

.amount-grid button.active {
  border-color: var(--billing-accent, #79c4f5);
  background: var(--billing-accent-soft, rgba(120,175,230,.12));
  color: var(--billing-accent-strong, #79c4f5);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--billing-accent, #79c4f5) 8%, transparent);
}

.custom-input {
  width: 100%;
  height: 46px;
  margin-top: 12px;
  padding: 0 14px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.1));
  border-radius: 11px;
  outline: none;
  background: var(--billing-surface-soft, rgba(15,18,24,.95));
  color: var(--billing-text, #fff);
  font-family: inherit;
  font-size: .9rem;
}

.custom-input:focus {
  border-color: var(--billing-accent, #79c4f5);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--billing-accent, #79c4f5) 10%, transparent);
}

.method-shell :deep(.payment-methods-picker) {
  padding: 0;
  border: 0;
  background: transparent;
  box-shadow: none;
}

.method-shell :deep(.payment-methods-picker > header) {
  display: none;
}

.checkout-sidebar {
  position: sticky;
  top: 74px;
  padding: 20px;
}

.summary-heading {
  margin-bottom: 14px;
}

.checkout-sidebar :deep(.payment-order-summary) {
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.checkout-sidebar :deep(.payment-order-summary > header) {
  display: none;
}

.checkout-sidebar :deep(.payment-order-summary ul) {
  gap: 0;
}

.checkout-sidebar :deep(.payment-order-summary li) {
  min-height: 40px;
  align-items: center;
  border-bottom: 1px solid var(--billing-border, rgba(255,255,255,.06));
}

.checkout-sidebar :deep(.payment-order-summary li.receive) {
  min-height: 58px;
  margin-top: 4px;
  padding-top: 4px;
  border-top: 0;
  border-bottom: 0;
}

.checkout-sidebar :deep(.payment-order-summary li.receive strong) {
  font-size: 1.15rem;
}

.checkout-submit {
  width: 100%;
  min-height: 50px;
  margin-top: 16px;
  padding: 0 16px;
  border: 1px solid var(--billing-accent, #79c4f5);
  border-radius: 12px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--billing-accent, #79c4f5) 92%, white 8%), var(--billing-accent-strong, #4ca7e0));
  color: #07131c;
  font-family: inherit;
  font-size: .86rem;
  font-weight: 760;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  transition: transform .15s ease, box-shadow .15s ease, opacity .15s ease;
}

.checkout-submit:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 10px 26px color-mix(in srgb, var(--billing-accent, #79c4f5) 22%, transparent);
}

.checkout-submit b {
  font-size: .78rem;
  font-weight: 760;
  font-variant-numeric: tabular-nums;
}

.checkout-submit:disabled {
  border-color: var(--billing-border, rgba(255,255,255,.08));
  background: var(--billing-surface-soft, rgba(255,255,255,.04));
  color: var(--billing-subtle, rgba(255,255,255,.38));
  cursor: not-allowed;
  box-shadow: none;
}

.secure {
  display: block;
  margin-top: 10px;
  color: var(--billing-subtle, rgba(255,255,255,.42));
  font-size: .64rem;
  text-align: center;
}

.checkout-error {
  margin-top: 12px;
  padding: 10px 12px;
  font-size: .72rem;
  text-align: left;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-variant-numeric: tabular-nums;
}

.ledger-note {
  margin: 8px 0 0;
  color: var(--billing-muted, rgba(255,255,255,.45));
  font-size: .69rem;
}

.ledger-error {
  color: var(--billing-danger, #f48b8b);
}

.ledger-entries {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 4px 0 0;
  padding: 12px 0 0;
  border-top: 1px solid var(--billing-border, rgba(255,255,255,.06));
  list-style: none;
}

.ledger-entries li {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 12px;
  align-items: center;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--billing-surface-soft, rgba(255,255,255,.03));
  color: var(--billing-text-soft, rgba(255,255,255,.78));
  font-size: .72rem;
}

.ledger-entry-time {
  color: var(--billing-subtle, rgba(255,255,255,.4));
  font-size: .66rem;
}

.empty {
  padding: 28px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.06));
  border-radius: 14px;
  background: var(--billing-surface, rgba(15,18,24,.78));
  color: var(--billing-muted, rgba(255,255,255,.55));
  font-size: .82rem;
  text-align: center;
}

.plan-list.pricing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(290px, 1fr));
  gap: 16px;
  align-items: stretch;
}

.pricing-card {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 500px;
  overflow: hidden;
  padding: 22px 22px 18px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.09));
  border-radius: 18px;
  background:
    radial-gradient(circle at 100% 0%, color-mix(in srgb, var(--billing-accent, #79c4f5) 8%, transparent), transparent 34%),
    var(--billing-surface, rgba(15,18,24,.98));
  box-shadow: var(--billing-shadow, 0 16px 42px rgba(0,0,0,.15));
  cursor: pointer;
  transition: transform .18s ease, border-color .18s ease, box-shadow .18s ease;
}

.pricing-card::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  opacity: 0;
  background: linear-gradient(135deg, color-mix(in srgb, var(--billing-accent, #79c4f5) 8%, transparent), transparent 42%);
  transition: opacity .18s ease;
}

.pricing-card:hover {
  transform: translateY(-3px);
  border-color: var(--billing-border-strong, rgba(121,196,245,.27));
}

.pricing-card:hover::before,
.pricing-card.selected::before {
  opacity: 1;
}

.pricing-card.selected {
  border-color: var(--billing-accent, #79c4f5);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--billing-accent, #79c4f5) 8%, transparent), var(--billing-shadow, 0 22px 58px rgba(23,81,119,.19));
}

.pricing-card.selected::after {
  content: '';
  position: absolute;
  top: 0;
  left: 20px;
  right: 20px;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--billing-accent, #6bc5fa) 20%, var(--billing-success, #6ee7c5) 80%, transparent);
}

.pricing-topline,
.pricing-title,
.pricing-price,
.pricing-reference,
.pricing-savings,
.pricing-facts,
.pricing-benefits,
.pricing-cta,
.pricing-footnote {
  position: relative;
  z-index: 1;
}

.pricing-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 25px;
}

.pricing-brand,
.pricing-tier {
  display: inline-flex;
  align-items: center;
  min-height: 25px;
  padding: 0 9px;
  border-radius: 999px;
  font-size: .66rem;
  font-weight: 700;
}

.pricing-brand {
  border: 1px solid var(--billing-border, rgba(255,255,255,.075));
  background: var(--billing-surface-soft, rgba(255,255,255,.035));
  color: var(--billing-muted, rgba(228,235,242,.68));
}

.pricing-tier {
  border: 1px solid color-mix(in srgb, var(--billing-accent, #79c4f5) 20%, transparent);
  background: var(--billing-accent-soft, rgba(121,196,245,.07));
  color: var(--billing-accent-strong, #82caf8);
}

.pricing-tier.hot {
  border-color: color-mix(in srgb, var(--billing-success, #6ee2bf) 24%, transparent);
  background: color-mix(in srgb, var(--billing-success, #6ee2bf) 9%, transparent);
  color: var(--billing-success, #6ee2bf);
}

.pricing-title {
  min-height: 80px;
  margin-top: 15px;
}

.pricing-title h3 {
  margin: 0;
  color: var(--billing-text, #f5f8fb);
  font-size: 1.08rem;
  font-weight: 700;
  line-height: 1.42;
}

.pricing-title p {
  margin: 8px 0 0;
  color: var(--billing-muted, rgba(210,220,230,.53));
  font-size: .75rem;
  line-height: 1.55;
}

.pricing-price {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  margin-top: 10px;
  color: var(--billing-text, #f6f9fb);
}

.pricing-currency {
  margin-bottom: 6px;
  color: var(--billing-muted, rgba(206,219,230,.55));
  font-size: .69rem;
  font-weight: 700;
}

.pricing-price > strong {
  font-size: clamp(2rem,2.4vw,2.5rem);
  font-weight: 760;
  line-height: .96;
  letter-spacing: -.055em;
}

.pricing-period {
  margin-bottom: 4px;
  color: var(--billing-muted, rgba(206,219,230,.48));
  font-size: .75rem;
}

.pricing-reference {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 24px;
  margin-top: 9px;
  color: var(--billing-muted, rgba(203,214,224,.43));
  font-size: .7rem;
}

.pricing-reference b {
  color: var(--billing-success, #6fe2c0);
  font-weight: 650;
}

.pricing-savings {
  display: grid;
  gap: 7px;
  margin-top: 14px;
  padding: 13px 14px;
  border: 1px solid color-mix(in srgb, var(--billing-success, #61dfba) 22%, transparent);
  border-radius: 12px;
  background: color-mix(in srgb, var(--billing-success, #61dfba) 7%, transparent);
}

.pricing-savings.neutral {
  border-color: color-mix(in srgb, var(--billing-accent, #79c4f5) 18%, transparent);
  background: var(--billing-accent-soft, rgba(121,196,245,.045));
}

.pricing-savings > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pricing-savings i {
  width: 6px;
  height: 6px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--billing-success, #61dfba);
}

.pricing-savings strong {
  color: var(--billing-success, #77e5c4);
  font-size: .75rem;
  font-weight: 700;
}

.pricing-savings.neutral strong {
  color: var(--billing-accent-strong, #79c4f5);
}

.pricing-facts {
  display: grid;
  margin: 16px -22px 0;
  border-top: 1px solid var(--billing-border, rgba(255,255,255,.065));
  border-bottom: 1px solid var(--billing-border, rgba(255,255,255,.065));
}

.pricing-facts > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 39px;
  padding: 0 22px;
  border-bottom: 1px solid var(--billing-border, rgba(255,255,255,.045));
}

.pricing-facts > div:last-child {
  border-bottom: 0;
}

.pricing-facts dt {
  color: var(--billing-muted, rgba(206,217,227,.5));
  font-size: .71rem;
}

.pricing-facts dd {
  margin: 0;
  color: var(--billing-text-soft, rgba(241,246,250,.88));
  font-size: .74rem;
  font-weight: 680;
}

.pricing-benefits {
  display: grid;
  gap: 8px;
  flex: 1;
  margin: 15px 0 0;
  padding: 0;
  list-style: none;
}

.pricing-benefits li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: var(--billing-muted, rgba(218,226,234,.65));
  font-size: .71rem;
  line-height: 1.48;
}

.pricing-benefits li b {
  display: grid;
  width: 16px;
  height: 16px;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--billing-accent, #79c4f5) 20%, transparent);
  border-radius: 50%;
  background: var(--billing-accent-soft, rgba(121,196,245,.055));
  color: var(--billing-accent, #79c4f5);
  font-size: .59rem;
}

.primary {
  border: 1px solid var(--billing-accent, #79c4f5);
  border-radius: 10px;
  background: var(--billing-accent, #79c4f5);
  color: #071019;
  font-family: inherit;
  font-weight: 650;
  cursor: pointer;
}

.primary:disabled {
  opacity: .55;
  cursor: not-allowed;
}

.pricing-cta {
  width: 100%;
  height: 44px;
  margin-top: 18px;
  border-radius: 11px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--billing-accent, #79c4f5) 86%, #1a3142 14%), var(--billing-accent-strong, #4ca7e0));
  color: #eff9ff;
}

.pricing-card.selected .pricing-cta {
  color: #07131c;
}

.pricing-footnote {
  margin-top: 9px;
  color: var(--billing-subtle, rgba(205,216,226,.35));
  font-size: .63rem;
  text-align: center;
}

@media (max-width: 1080px) {
  .wallet-card {
    grid-template-columns: minmax(230px, 1fr) minmax(260px, 1fr);
  }

  .wallet-actions {
    grid-column: 1 / -1;
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
  }

  .checkout-layout {
    grid-template-columns: minmax(0, 1fr) 300px;
  }

  .amount-grid {
    grid-template-columns: repeat(4, minmax(82px, 1fr));
  }
}

@media (max-width: 820px) {
  .wallet-card {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .wallet-stats {
    grid-template-columns: repeat(3, 1fr);
  }

  .wallet-actions {
    grid-column: auto;
  }

  .checkout-layout {
    grid-template-columns: 1fr;
  }

  .checkout-sidebar {
    position: static;
  }
}

@media (max-width: 560px) {
  .billing-page {
    gap: 14px;
  }

  .wallet-card,
  .amount-card,
  .method-shell,
  .checkout-sidebar {
    border-radius: 14px;
  }

  .wallet-card {
    padding: 18px;
  }

  .wallet-stats {
    grid-template-columns: 1fr 1fr;
  }

  .wallet-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .orders-link,
  .ledger-toggle {
    width: 100%;
  }

  .amount-card,
  .method-shell,
  .checkout-sidebar {
    padding: 17px;
  }

  .checkout-section-head {
    align-items: flex-start;
  }

  .amount-preview {
    margin-top: 7px;
  }

  .amount-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .ledger-entries li {
    grid-template-columns: 1fr auto;
  }

  .ledger-entry-time {
    grid-column: 1 / -1;
  }

  .plan-list.pricing-grid {
    grid-template-columns: 1fr;
  }

  .pricing-card {
    min-height: 0;
    padding: 20px 19px 17px;
  }

  .pricing-facts {
    margin-left: -19px;
    margin-right: -19px;
  }

  .pricing-facts > div {
    padding: 0 19px;
  }
}
</style>
