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
  padding: 18px;
  border: 1px solid var(--billing-border, rgba(255,255,255,.08));
  border-radius: 14px;
  background: var(--billing-surface, rgba(15,18,24,.9));
  box-shadow: 0 10px 30px rgba(26, 42, 58, .035);
}

.payment-order-summary header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 12px;
}

.payment-order-summary .eyebrow {
  color: var(--billing-text, rgba(255,255,255,.9));
  font-size: .76rem;
  font-weight: 700;
}

.payment-order-summary header strong {
  color: var(--billing-text-soft, rgba(255,255,255,.85));
  font-size: .78rem;
  font-weight: 650;
}

.payment-order-summary ul {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.payment-order-summary li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  color: var(--billing-muted, rgba(255,255,255,.65));
  font-size: .75rem;
}

.payment-order-summary li strong {
  color: var(--billing-text-soft, rgba(255,255,255,.92));
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.payment-order-summary li.receive {
  margin-top: 2px;
  padding-top: 12px;
  border-top: 1px solid var(--billing-border, rgba(255,255,255,.06));
}

.payment-order-summary li.receive span {
  color: var(--billing-text-soft, rgba(255,255,255,.72));
  font-weight: 620;
}

.payment-order-summary li.receive strong {
  color: var(--billing-accent-strong, #79c4f5);
  font-size: 1rem;
  font-weight: 760;
}
</style>
