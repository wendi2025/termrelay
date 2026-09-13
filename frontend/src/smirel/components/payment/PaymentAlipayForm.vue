<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CreateOrderResponse } from '../../api/payment'

const props = defineProps<{
  order: CreateOrderResponse
}>()

const emit = defineEmits<{
  (e: 'paid', order?: CreateOrderResponse): void
  (e: 'cancelled'): void
}>()

const { t } = useI18n()

// Alipay supports:
//   - qr_code 当面付 → render QR
//   - pay_url 跳转（WAP/H5/APP） → "前往支付" 按钮 + 自动跳转
//   - alipay_mobile_precreate_deep_link: if true and we have deep_link → APP 唤起
const isQr = computed(() => !!props.order.qr_code && !props.order.pay_url)
const isRedirect = computed(() => !!props.order.pay_url)
const deepLink = computed(() => {
  // 后端目前只在 alipay_mobile_precreate_deep_link=true 时启用 deep_link
  return props.order.alipay_mobile_precreate_deep_link && props.order.pay_url
    ? props.order.pay_url
    : ''
})

const launched = ref(false)

function launch() {
  if (!props.order.pay_url) return
  launched.value = true
  // APP deep link 或 WAP H5 都在新窗口打开
  window.open(props.order.pay_url, '_blank', 'noopener,noreferrer')
  emit('paid')
}
</script>

<template>
  <section class="payment-alipay-form">
    <header>
      <span class="eyebrow">{{ t('payment.method.alipay') }}</span>
      <strong>{{ order.currency || 'CNY' }} · {{ (order.pay_amount ?? order.amount).toFixed(2) }}</strong>
    </header>

    <div v-if="isQr" class="qr">
      <img :src="order.qr_code" alt="Alipay QR Code" />
      <p>{{ t('payment.qrcodeHint', { method: t('payment.method.alipay') }) }}</p>
    </div>

    <div v-else-if="isRedirect" class="redirect">
      <p>{{ t('payment.redirectHint', { method: t('payment.method.alipay') }) }}</p>
      <button class="primary" type="button" @click="launch">
        {{ deepLink ? t('payment.jsapiPay') : t('payment.redirectOpen') }}
      </button>
    </div>

    <div v-else class="empty">
      {{ t('payment.notAvailable') }}
    </div>

    <small class="secure">{{ t('payment.secure') }}</small>
  </section>
</template>

<style scoped>
.payment-alipay-form {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  align-items: stretch;
}
.payment-alipay-form header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.payment-alipay-form .eyebrow {
  font-size: 0.6rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.payment-alipay-form header strong {
  font-size: 0.96rem;
  color: rgba(255, 255, 255, 0.92);
  font-variant-numeric: tabular-nums;
}
.qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.qr img {
  width: 220px;
  height: 220px;
  background: #fff;
  border-radius: 12px;
  padding: 10px;
}
.qr p {
  margin: 0;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.6);
  text-align: center;
}
.redirect {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}
.redirect p {
  margin: 0;
  font-size: 0.82rem;
  color: rgba(255, 255, 255, 0.7);
  text-align: center;
}
.primary {
  background: #1677ff;
  color: #fff;
  border: 1px solid #1677ff;
  border-radius: 10px;
  min-width: 220px;
  min-height: 44px;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
}
.empty {
  padding: 32px 16px;
  text-align: center;
  color: rgba(255, 255, 255, 0.55);
  font-size: 0.85rem;
}
.secure {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}
</style>
