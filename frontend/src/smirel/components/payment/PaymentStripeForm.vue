<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentApi, type CreateOrderResponse } from '../../api/payment'
import { getErrorMessage } from '../../core/api'

const props = defineProps<{
  order: CreateOrderResponse
  publishableKey?: string
}>()

const emit = defineEmits<{
  (e: 'paid', order: CreateOrderResponse): void
  (e: 'failed', msg: string): void
}>()

const { t } = useI18n()

const containerRef = ref<HTMLDivElement | null>(null)
const ready = ref(false)
const submitting = ref(false)
const error = ref('')
let stripe: any = null
let elements: any = null

onMounted(async () => {
  if (!containerRef.value) return
  if (!props.order.client_secret) {
    error.value = 'Missing client_secret'
    return
  }
  try {
    const { loadStripe } = await import('@stripe/stripe-js')
    const key = props.publishableKey || (window as any).__SMIREL_STRIPE_PK || ''
    stripe = await loadStripe(key)
    if (!stripe) {
      error.value = 'Stripe failed to load'
      return
    }
    elements = stripe.elements({
      clientSecret: props.order.client_secret,
      appearance: { theme: 'night', labels: 'floating' },
    })
    const paymentElement = elements.create('payment', { layout: 'tabs' })
    paymentElement.mount(containerRef.value)
    paymentElement.on('ready', () => { ready.value = true })
    paymentElement.on('change', (e: any) => {
      if (e.error) error.value = e.error.message || ''
      else error.value = ''
    })
  } catch (e: any) {
    error.value = e?.message || 'Stripe error'
  }
})

onUnmounted(() => {
  if (elements) {
    try { elements.unmount() } catch { /* ignore */ }
    elements = null
  }
})

async function submit() {
  if (!stripe || !elements) return
  submitting.value = true
  error.value = ''
  try {
    const { error: err } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: window.location.origin + '/payment/result',
      },
      redirect: 'if_required',
    })
    if (err) {
      error.value = err.message || 'Payment failed'
      emit('failed', error.value)
      submitting.value = false
      return
    }
    // success path: poll verify
    const verified = await paymentApi.verifyOrder({ out_trade_no: props.order.out_trade_no })
    emit('paid', Object.assign({}, props.order, verified || {}))
  } catch (e) {
    error.value = getErrorMessage(e)
    emit('failed', error.value)
  } finally {
    submitting.value = false
  }
}

watch(() => props.order.client_secret, () => {
  if (elements) {
    try { elements.unmount() } catch { /* ignore */ }
    elements = null
  }
  ready.value = false
})
</script>

<template>
  <section class="payment-stripe-form">
    <header>
      <span class="eyebrow">Stripe</span>
      <strong>{{ order.currency || 'USD' }} · {{ (order.pay_amount ?? order.amount).toFixed(2) }}</strong>
    </header>
    <div class="element-host" ref="containerRef"></div>
    <p v-if="error" class="error">{{ error }}</p>
    <button class="primary" type="button" :disabled="!ready || submitting" @click="submit">
      {{ submitting ? t('payment.submitting') : t('payment.submit') }}
    </button>
    <small class="secure">{{ t('payment.secure') }}</small>
  </section>
</template>

<style scoped>
.payment-stripe-form {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.payment-stripe-form header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}
.payment-stripe-form .eyebrow {
  font-size: 0.6rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.payment-stripe-form header strong {
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
.primary {
  background: #79c4f5;
  color: #071019;
  border: 1px solid #79c4f5;
  border-radius: 10px;
  min-height: 44px;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
}
.primary:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.secure {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}
</style>
