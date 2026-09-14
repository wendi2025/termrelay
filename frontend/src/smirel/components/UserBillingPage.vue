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
          <button v-if="ledgerEntries.length" type="button" class="ledger-toggle" @click="ledgerShowEntries = !ledgerShowEntries">
            {{ t('payment.ledger.entriesTitle') }} ({{ ledgerEntries.length }})
            <span class="ledger-toggle-mark">{{ ledgerShowEntries ? '−' : '+' }}</span>
          </button>
        </header>
        <dl class="ledger-grid">
          <div><dt>{{ t('payment.ledger.principal') }}</dt><dd class="mono">{{ fmtMoney(ledgerBreakdown.active_principal_left) }}</dd></div>
          <div><dt>{{ t('payment.ledger.bonus') }}</dt><dd class="mono">{{ fmtMoney(ledgerBreakdown.active_bonus_left) }}</dd></div>
          <div v-if="frozenTotal > 0"><dt>{{ t('payment.ledger.frozen') }}</dt><dd class="mono">{{ fmtMoney(frozenTotal) }}</dd></div>
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
          <button v-for="v in presetAmounts" :key="v" type="button" :class="{ active: selectedAmount === v && !customAmount }" @click="chooseAmount(v)">{{ v }}</button>
          <button type="button" :class="{ active: customAmount.length > 0 }" @click="useCustom">{{ t('payment.customAmount') }}</button>
        </div>
        <input v-if="customAmount.length > 0 || selectedAmount === null" v-model="customAmount" type="number" inputmode="decimal" :placeholder="t('payment.customPlaceholder')" class="custom-input" />
      </section>

      <PaymentMethodsPicker v-model="selectedMethod" :methods="info?.methods || {}" :limits="effectiveLimits" :amount="effectiveAmount" @update:model-value="pickMethod" />
      <PaymentOrderSummary :amount="effectiveAmount" :fee-rate="selectedMethodLimits?.fee_rate || 0" :pay-amount="(selectedMethodLimits && selectedMethodLimits.fee_rate > 0) ? effectiveAmount * (1 - selectedMethodLimits.fee_rate / 100) : effectiveAmount" :currency="selectedMethodCurrency" />
      <p v-if="checkout.error.value" class="error">{{ checkout.error.value }}</p>
      <button class="primary" type="button" :disabled="!effectiveAmount || !selectedMethod || checkout.submitting.value" @click="submitRecharge">{{ checkout.submitting.value ? t('payment.submitting') : t('payment.submit') }}</button>
      <small class="secure">{{ t('payment.secure') }}</small>
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
.billing-page{display:flex;flex-direction:column;gap:18px;padding:0 0 36px}.billing-mode{display:inline-flex;border:1px solid rgba(255,255,255,.08);border-radius:12px;padding:4px;width:fit-content;background:rgba(15,18,24,.7)}.billing-mode button{padding:8px 18px;border-radius:9px;border:0;background:transparent;color:rgba(255,255,255,.55);font-family:inherit;font-size:.84rem;cursor:pointer}.billing-mode button.active{background:rgba(120,175,230,.18);color:#fff;font-weight:600}.loading,.error{text-align:center;padding:24px;border-radius:12px;background:rgba(15,18,24,.78);border:1px solid rgba(255,255,255,.06);color:rgba(255,255,255,.65);font-size:.86rem}.error{background:rgba(244,139,139,.06);border-color:rgba(244,139,139,.32);color:#f48b8b}.balance-card{display:flex;justify-content:space-between;align-items:center;border:1px solid rgba(255,255,255,.08);border-radius:14px;background:linear-gradient(180deg,rgba(18,22,28,.88),rgba(12,16,22,.94));padding:18px 22px}.balance-label .eyebrow{font-size:.62rem;letter-spacing:.1em;text-transform:uppercase;color:rgba(255,255,255,.45)}.balance-label strong{display:block;font-size:1.7rem;color:rgba(255,255,255,.95);font-variant-numeric:tabular-nums;margin-top:4px;font-weight:640}.balance-label p{margin:4px 0 0;font-size:.7rem;color:rgba(255,255,255,.45)}.orders-link{font-size:.78rem;color:#79c4f5;text-decoration:none}.amount-card{border:1px solid rgba(255,255,255,.08);border-radius:14px;background:rgba(15,18,24,.78);padding:18px 22px;display:flex;flex-direction:column;gap:12px}.amount-card .eyebrow{font-size:.62rem;letter-spacing:.1em;text-transform:uppercase;color:rgba(255,255,255,.45)}.amount-card small{font-size:.7rem;color:rgba(255,255,255,.42)}.amount-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(110px,1fr));gap:8px}.amount-grid button{min-height:42px;border-radius:10px;border:1px solid rgba(255,255,255,.08);background:rgba(255,255,255,.02);color:rgba(255,255,255,.78);font-family:inherit;font-size:.86rem;cursor:pointer}.amount-grid button.active{border-color:#79c4f5;background:rgba(120,175,230,.12);color:#fff}.custom-input{width:100%;height:44px;padding:0 14px;border-radius:10px;border:1px solid rgba(255,255,255,.1);background:rgba(15,18,24,.95);color:#fff;font-size:.95rem;font-family:inherit}.primary{background:#79c4f5;border:1px solid #79c4f5;border-radius:10px;height:46px;font-size:.92rem;font-weight:600;color:#071019;cursor:pointer;font-family:inherit}.primary:disabled{opacity:.55;cursor:not-allowed}.secure{font-size:.66rem;color:rgba(255,255,255,.42);text-align:center}.empty{padding:28px;border-radius:12px;background:rgba(15,18,24,.78);border:1px solid rgba(255,255,255,.06);text-align:center;color:rgba(255,255,255,.55);font-size:.84rem}.plan-list.pricing-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(300px,1fr));gap:16px;align-items:stretch}.pricing-card{position:relative;display:flex;flex-direction:column;min-height:500px;overflow:hidden;border:1px solid rgba(255,255,255,.09);border-radius:18px;background:radial-gradient(circle at 100% 0%,rgba(91,186,246,.07),transparent 34%),linear-gradient(158deg,rgba(18,22,29,.98),rgba(8,11,15,.99));padding:22px 22px 18px;cursor:pointer;box-shadow:inset 0 1px rgba(255,255,255,.025),0 16px 42px rgba(0,0,0,.15);transition:transform .18s ease,border-color .18s ease,box-shadow .18s ease}.pricing-card::before{content:'';position:absolute;inset:0;pointer-events:none;opacity:0;background:linear-gradient(135deg,rgba(121,196,245,.075),transparent 42%);transition:opacity .18s ease}.pricing-card:hover{transform:translateY(-3px);border-color:rgba(121,196,245,.27);box-shadow:inset 0 1px rgba(255,255,255,.035),0 24px 60px rgba(0,0,0,.26)}.pricing-card:hover::before,.pricing-card.selected::before{opacity:1}.pricing-card.selected{border-color:rgba(121,196,245,.7);box-shadow:0 0 0 1px rgba(121,196,245,.08),0 22px 58px rgba(23,81,119,.19)}.pricing-card.selected::after{content:'';position:absolute;top:0;left:20px;right:20px;height:2px;background:linear-gradient(90deg,transparent,#6bc5fa 20%,#6ee7c5 80%,transparent);box-shadow:0 0 18px rgba(107,197,250,.45)}.pricing-topline{position:relative;z-index:1;display:flex;align-items:center;justify-content:space-between;gap:10px;min-height:25px}.pricing-brand,.pricing-tier{display:inline-flex;align-items:center;min-height:25px;padding:0 9px;border-radius:999px;font-size:.66rem;font-weight:700;letter-spacing:.02em}.pricing-brand{border:1px solid rgba(255,255,255,.075);background:rgba(255,255,255,.035);color:rgba(228,235,242,.68)}.pricing-tier{border:1px solid rgba(121,196,245,.16);background:rgba(121,196,245,.07);color:#82caf8}.pricing-tier.hot{border-color:rgba(106,226,190,.2);background:rgba(74,194,160,.09);color:#6ee2bf}.pricing-title{position:relative;z-index:1;min-height:80px;margin-top:15px}.pricing-title h3{margin:0;color:#f5f8fb;font-size:1.08rem;font-weight:700;line-height:1.42;letter-spacing:-.018em}.pricing-title p{margin:8px 0 0;color:rgba(210,220,230,.53);font-size:.75rem;line-height:1.55}.pricing-price{position:relative;z-index:1;display:flex;align-items:flex-end;gap:6px;margin-top:10px;color:#f6f9fb}.pricing-currency{margin-bottom:6px;color:rgba(206,219,230,.55);font-size:.69rem;font-weight:700}.pricing-price>strong{font-size:clamp(2rem,2.4vw,2.5rem);font-weight:760;line-height:.96;letter-spacing:-.055em;font-variant-numeric:tabular-nums}.pricing-period{margin-bottom:4px;color:rgba(206,219,230,.48);font-size:.75rem}.pricing-reference{position:relative;z-index:1;display:flex;align-items:center;flex-wrap:wrap;gap:6px;min-height:24px;margin-top:9px;color:rgba(203,214,224,.43);font-size:.7rem}.pricing-reference s{color:rgba(203,214,224,.38)}.pricing-reference b{color:rgba(111,226,192,.86);font-weight:650}.pricing-savings{position:relative;z-index:1;display:grid;gap:7px;margin-top:14px;padding:13px 14px;border:1px solid rgba(87,224,188,.2);border-radius:12px;background:linear-gradient(135deg,rgba(53,181,151,.09),rgba(40,116,103,.035))}.pricing-savings.neutral{border-color:rgba(121,196,245,.14);background:rgba(121,196,245,.045)}.pricing-savings>div{display:flex;align-items:center;gap:8px}.pricing-savings i{width:6px;height:6px;flex:0 0 auto;border-radius:50%;background:#61dfba;box-shadow:0 0 0 4px rgba(97,223,186,.08)}.pricing-savings.neutral i{background:#79c4f5;box-shadow:0 0 0 4px rgba(121,196,245,.07)}.pricing-savings strong{color:#77e5c4;font-size:.75rem;font-weight:700;line-height:1.35}.pricing-savings.neutral strong{color:rgba(150,207,244,.8)}.pricing-facts{position:relative;z-index:1;display:grid;margin:16px -22px 0;border-top:1px solid rgba(255,255,255,.065);border-bottom:1px solid rgba(255,255,255,.065)}.pricing-facts>div{display:flex;align-items:center;justify-content:space-between;min-height:39px;padding:0 22px;border-bottom:1px solid rgba(255,255,255,.045)}.pricing-facts>div:last-child{border-bottom:0}.pricing-facts dt{color:rgba(206,217,227,.5);font-size:.71rem}.pricing-facts dd{margin:0;color:rgba(241,246,250,.88);font-size:.74rem;font-weight:680;font-variant-numeric:tabular-nums}.pricing-benefits{position:relative;z-index:1;display:grid;gap:8px;flex:1;list-style:none;margin:15px 0 0;padding:0}.pricing-benefits li{display:flex;align-items:flex-start;gap:8px;color:rgba(218,226,234,.65);font-size:.71rem;line-height:1.48}.pricing-benefits li b{display:grid;place-items:center;width:16px;height:16px;flex:0 0 auto;margin-top:1px;border:1px solid rgba(121,196,245,.18);border-radius:50%;background:rgba(121,196,245,.055);color:#79c4f5;font-size:.59rem}.pricing-cta{position:relative;z-index:1;width:100%;height:44px;margin-top:18px;border-color:rgba(121,196,245,.38);border-radius:11px;background:linear-gradient(180deg,rgba(48,88,118,.92),rgba(31,62,84,.96));color:#eff9ff;box-shadow:inset 0 1px rgba(255,255,255,.08);transition:transform .15s ease,border-color .15s ease,background .15s ease,box-shadow .15s ease}.pricing-cta:hover:not(:disabled){transform:translateY(-1px);border-color:rgba(121,196,245,.72);background:linear-gradient(180deg,rgba(72,134,177,.98),rgba(43,91,124,.98));box-shadow:0 9px 26px rgba(32,98,140,.2),inset 0 1px rgba(255,255,255,.1)}.pricing-card.selected .pricing-cta{border-color:#80cdfa;background:linear-gradient(180deg,#82cdf9,#67b8ea);color:#07131c}.pricing-footnote{position:relative;z-index:1;margin-top:9px;color:rgba(205,216,226,.35);font-size:.63rem;text-align:center}@media(max-width:720px){.plan-list.pricing-grid{grid-template-columns:1fr}.pricing-card{min-height:0;padding:20px 19px 17px}.pricing-facts{margin-left:-19px;margin-right:-19px}.pricing-facts>div{padding:0 19px}}.mono{font-family:ui-monospace,monospace;font-variant-numeric:tabular-nums}.ledger-card{border:1px solid rgba(255,255,255,.08);border-radius:14px;background:linear-gradient(180deg,rgba(18,22,28,.88),rgba(12,16,22,.94));padding:16px 22px}.ledger-card header{display:flex;align-items:center;justify-content:space-between;gap:12px}.ledger-card .eyebrow{font-size:.62rem;letter-spacing:.1em;text-transform:uppercase;color:rgba(255,255,255,.45)}.ledger-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px 18px;margin:12px 0 0}.ledger-grid dt{font-size:.68rem;color:rgba(255,255,255,.45)}.ledger-grid dd{margin:3px 0 0;font-size:1rem;color:rgba(255,255,255,.92);font-variant-numeric:tabular-nums}.ledger-note{margin:10px 0 0;font-size:.7rem;color:rgba(255,255,255,.45)}.ledger-error{color:#f48b8b}.ledger-toggle{height:26px;padding:0 10px;border-radius:6px;border:1px solid rgba(255,255,255,.1);background:rgba(255,255,255,.04);color:rgba(255,255,255,.72);font-family:inherit;font-size:.72rem;cursor:pointer}.ledger-toggle:hover{background:rgba(255,255,255,.08)}.ledger-toggle-mark{margin-left:6px;color:#79c4f5}.ledger-entries{list-style:none;margin:12px 0 0;padding:0;display:flex;flex-direction:column;gap:6px}.ledger-entries li{display:grid;grid-template-columns:1fr auto auto;gap:10px;align-items:center;padding:7px 12px;border-radius:8px;background:rgba(255,255,255,.03);font-size:.74rem;color:rgba(255,255,255,.78)}.ledger-entry-time{font-size:.68rem;color:rgba(255,255,255,.4)}
</style>
