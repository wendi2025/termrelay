<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentApi, type CreateOrderResponse } from '../../api/payment'

const props = defineProps<{
  order: CreateOrderResponse
}>()

const emit = defineEmits<{
  (e: 'paid', order: CreateOrderResponse): void
  (e: 'failed', msg: string): void
}>()

const { t } = useI18n()

const containerRef = ref<HTMLDivElement | null>(null)
const ready = ref(false)
const error = ref('')
let dropInEl: any = null

async function mountDropIn() {
  if (!containerRef.value) return
  if (!props.order.client_secret || !props.order.intent_id) {
    error.value = 'Missing airwallex credentials'
    return
  }
  try {
    const sdk = await import('@airwallex/components-sdk')
    const env = (props.order.payment_env === 'demo' ? 'demo' : 'prod')
    await sdk.init({
      enabled: true,
      env,
      locale: 'en',
      // origin 为可选校验字段，留空即可
    })
    dropInEl = await sdk.createElement('dropIn', {
      intent_id: props.order.intent_id,
      client_secret: props.order.client_secret,
      currency: props.order.currency || 'USD',
      mode: 'payment',
      appearance: {
        mode: 'dark',
        variables: {
          colorBrand: '#79c4f5',
          colorText: '#e6edf3',
          colorBackground: '#0f1620',
        },
      },
    })
    dropInEl.mount(containerRef.value)
    dropInEl.on('success', () => {
      paymentApi.verifyOrder({ out_trade_no: props.order.out_trade_no }).then((verified) => {
        emit('paid', Object.assign({}, props.order, verified || {}))
      }).catch((e) => {
        emit('failed', String(e))
      })
    })
    dropInEl.on('error', (evt: any) => {
      error.value = evt?.message || evt?.error?.message || 'Airwallex error'
      emit('failed', error.value)
    })
    ready.value = true
  } catch (e: any) {
    error.value = e?.message || 'Failed to load Airwallex'
    emit('failed', error.value)
  }
}

onMounted(mountDropIn)

onUnmounted(() => {
  if (dropInEl) {
    try { dropInEl.unmount() } catch { /* ignore */ }
    dropInEl = null
  }
})
</script>

<template>
  <section class="payment-airwallex-form">
    <header>
      <span class="eyebrow">Airwallex</span>
      <strong>{{ order.currency || 'USD' }} · {{ (order.pay_amount ?? order.amount).toFixed(2) }}</strong>
    </header>
    <div class="element-host" ref="containerRef"></div>
    <p v-if="error" class="error">{{ error }}</p>
    <small class="secure">{{ t('payment.secure') }}</small>
  </section>
</template>

<style scoped>
.payment-airwallex-form {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.payment-airwallex-form header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.payment-airwallex-form .eyebrow {
  font-size: 0.6rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.payment-airwallex-form header strong {
  font-size: 0.96rem;
  color: rgba(255, 255, 255, 0.92);
  font-variant-numeric: tabular-nums;
}
.element-host {
  min-height: 280px;
  padding: 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.02);
}
.error {
  margin: 0;
  color: #f48b8b;
  font-size: 0.78rem;
}
.secure {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}
</style>
