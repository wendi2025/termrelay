<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  amount: number
  feeRate: number
  payAmount?: number
  currency?: string
  productName?: string
}>()

const { t } = useI18n()

const feeAmount = computed(() => {
  if (!props.feeRate || props.feeRate <= 0) return 0
  return Number((props.amount * props.feeRate / 100).toFixed(4))
})

const receiveAmount = computed(() => {
  if (typeof props.payAmount === 'number') return props.payAmount
  return Number((props.amount - feeAmount.value).toFixed(2))
})

const fmt = (v: number) => {
  const cur = props.currency || 'CNY'
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: cur, maximumFractionDigits: 2 }).format(v)
  } catch {
    return `${cur} ${v.toFixed(2)}`
  }
}

const feeLabel = computed(() => {
  if (!props.feeRate || props.feeRate <= 0) return t('payment.feeValue')
  return `${props.feeRate}% · ${fmt(feeAmount.value)}`
})
</script>

<template>
  <section class="payment-order-summary">
    <header>
      <span class="eyebrow">{{ t('payment.summary') }}</span>
      <strong v-if="productName">{{ productName }}</strong>
    </header>
    <ul>
      <li>
        <span>{{ t('payment.amountLabel') }}</span>
        <strong>{{ fmt(amount) }}</strong>
      </li>
      <li>
        <span>{{ t('payment.feeLabel') }}</span>
        <strong>{{ feeLabel }}</strong>
      </li>
      <li class="receive">
        <span>{{ t('payment.receiveLabel') }}</span>
        <strong>{{ fmt(receiveAmount) }}</strong>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.payment-order-summary {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(18, 22, 28, 0.85), rgba(12, 16, 22, 0.95));
  padding: 16px 18px 18px;
}
.payment-order-summary header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 12px;
}
.payment-order-summary .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  color: rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
}
.payment-order-summary header strong {
  font-size: 0.84rem;
  color: rgba(255, 255, 255, 0.85);
  font-weight: 600;
}
.payment-order-summary ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.payment-order-summary li {
  display: flex;
  justify-content: space-between;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.65);
}
.payment-order-summary li strong {
  color: rgba(255, 255, 255, 0.92);
  font-weight: 560;
  font-variant-numeric: tabular-nums;
}
.payment-order-summary li.receive {
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}
.payment-order-summary li.receive strong {
  font-size: 0.96rem;
  color: #79c4f5;
  font-weight: 620;
}
</style>
