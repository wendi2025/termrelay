<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OrderStatus } from '../../api/payment'

const props = defineProps<{
  status: OrderStatus | ''
  size?: 'sm' | 'md'
}>()

const { t } = useI18n()

const tone = computed(() => {
  switch (props.status) {
    case 'PENDING':
    case 'RECHARGING':
    case 'REFUNDING':
    case 'REFUND_PENDING':
    case 'REFUND_REQUESTED':
      return 'pending'
    case 'PAID':
    case 'COMPLETED':
    case 'PARTIALLY_REFUNDED':
      return 'success'
    case 'CANCELLED':
    case 'EXPIRED':
      return 'muted'
    case 'FAILED':
    case 'REFUND_FAILED':
      return 'failed'
    case 'REFUNDED':
      return 'refunded'
    default:
      return 'muted'
  }
})

const label = computed(() => {
  if (!props.status) return ''
  return t(`payment.status.${props.status}`)
})
</script>

<template>
  <span class="payment-status-badge" :class="[`tone-${tone}`, `size-${size || 'md'}`]">
    <i class="dot" aria-hidden="true"></i>
    <span>{{ label }}</span>
  </span>
</template>

<style scoped>
.payment-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  height: 22px;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 560;
  line-height: 1;
  border: 1px solid transparent;
}
.payment-status-badge.size-sm {
  height: 18px;
  font-size: 0.62rem;
  padding: 0 8px;
}
.payment-status-badge .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.04);
}
.payment-status-badge.tone-pending {
  background: rgba(120, 175, 230, 0.1);
  border-color: rgba(120, 175, 230, 0.32);
  color: #79c4f5;
}
.payment-status-badge.tone-success {
  background: rgba(67, 196, 135, 0.1);
  border-color: rgba(67, 196, 135, 0.32);
  color: #78dca6;
}
.payment-status-badge.tone-muted {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.55);
}
.payment-status-badge.tone-failed {
  background: rgba(232, 89, 89, 0.1);
  border-color: rgba(232, 89, 89, 0.32);
  color: #f48b8b;
}
.payment-status-badge.tone-refunded {
  background: rgba(206, 165, 92, 0.1);
  border-color: rgba(206, 165, 92, 0.32);
  color: #e8c684;
}
</style>
