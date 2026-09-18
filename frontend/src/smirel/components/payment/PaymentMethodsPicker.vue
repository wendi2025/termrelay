<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { basePaymentType, type PaymentType, type MethodLimits } from '../../api/payment'

const props = defineProps<{
  modelValue: PaymentType | ''
  methods: Record<string, MethodLimits>
  limits: { global_min: number; global_max: number }
  amount: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: PaymentType | ''): void
  (e: 'amountError', v: string | null): void
}>()

const { t } = useI18n()

interface MethodOption {
  key: string
  raw: PaymentType
  displayName: string
  feeRate: number
  currency: string
  singleMin: number
  singleMax: number
  dailyLimit: number
  group: string
  available: boolean
  reason?: string
}

const groupIcon = (g: string) => {
  switch (g) {
    case 'alipay': return 'A'
    case 'wxpay': return 'W'
    case 'easypay': return 'E'
    case 'stripe': return 'S'
    case 'airwallex': return 'X'
    default: return '?'
  }
}

const groupLabel = (g: string) => {
  const map: Record<string, string> = {
    alipay: t('payment.method.alipay'),
    wxpay: t('payment.method.wxpay'),
    easypay: t('payment.method.easypay'),
    stripe: t('payment.method.stripe'),
    airwallex: t('payment.method.airwallex'),
  }
  return map[g] || g
}

function limitRangeLabel(min: number, max: number): string {
  const hasMin = min > 0
  const hasMax = max > 0
  if (hasMin && hasMax) return `${min} - ${max}`
  if (hasMin) return `≥ ${min}`
  if (hasMax) return `≤ ${max}`
  return ''
}

const options = computed<MethodOption[]>(() => {
  const list: MethodOption[] = []
  Object.entries(props.methods || {}).forEach(([key, m]) => {
    const group = basePaymentType(key as PaymentType)
    // 后端以 0 表示「不限」（load_balancer.go 用 >0 判断），因此 0 不能被当作上下限，
    // 否则 single_max/max 为 0 的渠道会被误判为「不可用」，用户将无法选择任何支付方式。
    const minOk = m.single_min <= 0 || props.amount >= m.single_min
    const maxOk = m.single_max <= 0 || props.amount <= m.single_max
    const inRange = minOk && maxOk
    const dailyOk = m.daily_limit <= 0 || props.amount <= m.daily_limit
    const available = inRange && dailyOk
    list.push({
      key,
      raw: key as PaymentType,
      displayName: m.display_name || groupLabel(group),
      feeRate: m.fee_rate,
      currency: m.currency,
      singleMin: m.single_min,
      singleMax: m.single_max,
      dailyLimit: m.daily_limit,
      group,
      available,
      reason: !inRange
        ? limitRangeLabel(m.single_min, m.single_max)
        : !dailyOk
          ? 'daily'
          : undefined,
    })
  })
  // group: alipay/wxpay/easypay/stripe/airwallex
  return list
})

const grouped = computed(() => {
  const m = new Map<string, MethodOption[]>()
  options.value.forEach((o) => {
    if (!m.has(o.group)) m.set(o.group, [])
    m.get(o.group)!.push(o)
  })
  return Array.from(m.entries())
})

function pick(opt: MethodOption) {
  if (!opt.available) return
  emit('update:modelValue', opt.raw)
}

function isPicked(opt: MethodOption) {
  return props.modelValue === opt.raw
}
</script>

<template>
  <section class="payment-methods-picker">
    <header>
      <span class="eyebrow">{{ t('payment.methodLabel') }}</span>
      <small>{{ t('payment.methodDescription') }}</small>
    </header>
    <div v-if="grouped.length === 0" class="methods-empty">
      <span class="methods-empty-icon">—</span>
      <div>
        <strong>{{ t('payment.methodUnavailable') }}</strong>
        <small>{{ t('payment.methodDescription') }}</small>
      </div>
    </div>
    <div v-for="[group, items] in grouped" :key="group" class="group">
      <div class="group-head">
        <span class="group-icon">{{ groupIcon(group) }}</span>
        <span>{{ groupLabel(group) }}</span>
      </div>
      <div class="options">
        <button
          v-for="opt in items"
          :key="opt.key"
          type="button"
          :disabled="!opt.available"
          :class="{ active: isPicked(opt), unavailable: !opt.available }"
          @click="pick(opt)"
        >
          <strong>{{ opt.displayName }}</strong>
          <small>
            <template v-if="opt.feeRate > 0">{{ opt.feeRate }}%</template>
            <template v-else>{{ t('payment.feeValue') }}</template>
            <span> · {{ opt.currency }}</span>
          </small>
          <span v-if="!opt.available" class="reason">
            <template v-if="opt.reason === 'daily'">{{ t('payment.methodUnavailable') }} (daily)</template>
            <template v-else>{{ opt.reason }}</template>
          </span>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.payment-methods-picker {
  padding: 18px;
  border: 1px solid var(--billing-border, rgba(255, 255, 255, 0.08));
  border-radius: 14px;
  background: var(--billing-surface, rgba(15, 18, 24, 0.78));
}

.payment-methods-picker header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 14px;
}

.payment-methods-picker .eyebrow {
  color: var(--billing-text, rgba(255,255,255,.9));
  font-size: .76rem;
  font-weight: 700;
}

.payment-methods-picker header small {
  color: var(--billing-muted, rgba(255,255,255,.42));
  font-size: .68rem;
}

.methods-empty {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 82px;
  padding: 15px;
  border: 1px dashed var(--billing-border, rgba(255,255,255,.1));
  border-radius: 12px;
  background: var(--billing-surface-soft, rgba(255,255,255,.025));
}

.methods-empty-icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 10px;
  background: var(--billing-accent-soft, rgba(121,196,245,.08));
  color: var(--billing-accent-strong, #79c4f5);
  font-size: 1rem;
}

.methods-empty strong {
  display: block;
  color: var(--billing-text-soft, rgba(255,255,255,.75));
  font-size: .76rem;
  font-weight: 680;
}

.methods-empty small {
  display: block;
  margin-top: 4px;
  color: var(--billing-muted, rgba(255,255,255,.42));
  font-size: .66rem;
  line-height: 1.45;
}

.group {
  margin-bottom: 14px;
}

.group:last-child {
  margin-bottom: 0;
}

.group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  color: var(--billing-muted, rgba(255,255,255,.55));
  font-size: .71rem;
  font-weight: 620;
}

.group-icon {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--billing-accent, #79c4f5) 16%, transparent);
  border-radius: 7px;
  background: var(--billing-accent-soft, rgba(120,175,230,.12));
  color: var(--billing-accent-strong, #79c4f5);
  font-size: .66rem;
  font-weight: 760;
}

.options {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(175px, 1fr));
  gap: 8px;
}

.options button {
  position: relative;
  display: flex;
  min-height: 66px;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  padding: 11px 13px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 11px;
  background: var(--billing-surface-soft, rgba(255,255,255,.02));
  color: var(--billing-text-soft, rgba(255,255,255,.78));
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: transform .16s ease, border-color .16s ease, background .16s ease, box-shadow .16s ease;
}

.options button:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: var(--billing-border-strong, rgba(120,175,230,.4));
  background: var(--billing-accent-soft, rgba(120,175,230,.05));
}

.options button.active {
  border-color: var(--billing-accent, #79c4f5);
  background: var(--billing-accent-soft, rgba(120,175,230,.12));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--billing-accent, #79c4f5) 8%, transparent);
}

.options button.active::after {
  content: '✓';
  position: absolute;
  top: 9px;
  right: 10px;
  display: grid;
  width: 18px;
  height: 18px;
  place-items: center;
  border-radius: 50%;
  background: var(--billing-accent, #79c4f5);
  color: #07131c;
  font-size: .58rem;
  font-weight: 900;
}

.options button.unavailable {
  opacity: .48;
  cursor: not-allowed;
}

.options button strong {
  padding-right: 24px;
  font-size: .8rem;
  font-weight: 680;
}

.options button small {
  color: var(--billing-muted, rgba(255,255,255,.45));
  font-size: .64rem;
}

.options button .reason {
  margin-top: 2px;
  color: var(--billing-danger, #f48b8b);
  font-size: .6rem;
}

@media (max-width: 560px) {
  .payment-methods-picker {
    padding: 15px;
  }

  .options {
    grid-template-columns: 1fr;
  }
}
</style>
