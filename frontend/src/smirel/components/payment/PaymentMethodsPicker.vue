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
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: rgba(15, 18, 24, 0.78);
  padding: 16px 18px 18px;
}
.payment-methods-picker header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 14px;
}
.payment-methods-picker .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  color: rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
}
.payment-methods-picker header small {
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.42);
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
  font-size: 0.72rem;
  color: rgba(255, 255, 255, 0.55);
  font-weight: 560;
}
.group-icon {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: rgba(120, 175, 230, 0.12);
  color: #79c4f5;
  font-size: 0.7rem;
  font-weight: 700;
}
.options {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
}
.options button {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.02);
  color: rgba(255, 255, 255, 0.78);
  text-align: left;
  cursor: pointer;
  transition: all 0.18s ease;
  font-family: inherit;
}
.options button:hover:not(:disabled) {
  border-color: rgba(120, 175, 230, 0.4);
  background: rgba(120, 175, 230, 0.05);
}
.options button.active {
  border-color: #79c4f5;
  background: rgba(120, 175, 230, 0.12);
  color: #fff;
}
.options button.unavailable {
  opacity: 0.5;
  cursor: not-allowed;
}
.options button strong {
  font-size: 0.84rem;
  font-weight: 600;
}
.options button small {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.45);
}
.options button .reason {
  font-size: 0.6rem;
  color: #f48b8b;
  margin-top: 2px;
}
</style>
