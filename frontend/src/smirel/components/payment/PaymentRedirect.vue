<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { paymentApi, type PaymentOrder } from '../../api/payment'

const { t } = useI18n()
const route = useRoute()
const payUrl = ref('')
const order = ref<PaymentOrder | null>(null)
const opened = ref(false)
const error = ref('')

const outTrade = computed(() => String(route.query.out_trade_no || ''))

onMounted(async () => {
  payUrl.value = sessionStorage.getItem('smirel.payment.pay_url') || ''
  sessionStorage.removeItem('smirel.payment.pay_url')
  if (!payUrl.value) {
    error.value = 'Missing pay_url'
    return
  }
  if (outTrade.value) {
    try {
      order.value = await paymentApi.verifyOrder({ out_trade_no: outTrade.value })
    } catch (e: any) {
      // 忽略，下方轮询
    }
  }
  openExternal()
})

function openExternal() {
  if (!payUrl.value) return
  try {
    const w = window.open(payUrl.value, '_blank', 'noopener,noreferrer')
    if (w) {
      opened.value = true
    } else {
      // 弹窗被拦截，提示用户手动打开
      opened.value = false
    }
  } catch (e: any) {
    error.value = e?.message || t('payment.errorCreateOrder')
  }
}

function manualOpen() {
  if (!payUrl.value) return
  window.location.href = payUrl.value
}
</script>

<template>
  <section class="payment-redirect">
    <header>
      <span class="eyebrow">{{ t('payment.redirectTitle') }}</span>
    </header>
    <p class="hint">{{ t('payment.redirectHint', { method: '' }) }}</p>
    <p v-if="error" class="error">{{ error }}</p>
    <div class="actions">
      <button v-if="!opened" class="primary" type="button" @click="manualOpen">
        {{ t('payment.redirectOpen') }}
      </button>
      <a v-else class="primary" :href="payUrl" target="_blank" rel="noopener">
        {{ t('payment.redirectOpen') }}
      </a>
      <RouterLink class="ghost" to="/subscriptions">{{ t('payment.resultBack') }}</RouterLink>
    </div>
  </section>
</template>

<style scoped>
.payment-redirect {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.payment-redirect .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.payment-redirect .hint {
  margin: 0;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.7);
}
.payment-redirect .error {
  margin: 0;
  color: #f48b8b;
  font-size: 0.78rem;
}
.actions {
  display: flex;
  gap: 8px;
}
.actions button,
.actions a {
  flex: 1;
  min-height: 38px;
  padding: 0 14px;
  border-radius: 8px;
  border: 1px solid transparent;
  font-size: 0.78rem;
  font-family: inherit;
  cursor: pointer;
  text-align: center;
  line-height: 36px;
  text-decoration: none;
}
.actions .primary {
  background: #79c4f5;
  border-color: #79c4f5;
  color: #071019;
  font-weight: 600;
}
.actions .ghost {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.78);
}
</style>
