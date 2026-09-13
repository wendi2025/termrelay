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
  (e: 'cancelled'): void
}>()

const { t } = useI18n()

const popupRef = ref<Window | null>(null)
const status = ref<'opening' | 'opened' | 'closed' | 'failed'>('opening')
const error = ref('')
const verifying = ref(false)
let pollHandle: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  openPopup()
})

onUnmounted(() => {
  if (pollHandle) clearInterval(pollHandle)
  if (popupRef.value && !popupRef.value.closed) {
    try { popupRef.value.close() } catch { /* ignore */ }
  }
})

watch(status, (v) => {
  if (v === 'opened') startVerifying()
  if (v === 'closed') handleClosed()
})

function openPopup() {
  // popup 模式：使用 Stripe 托管页面 session URL 或 redirect URL
  const url = props.order.pay_url
  if (!url) {
    error.value = 'Missing pay_url for popup mode'
    status.value = 'failed'
    return
  }
  const w = window.open(url, 'smirel-stripe-popup', 'width=520,height=720,resizable=yes,scrollbars=yes')
  if (!w) {
    error.value = 'Browser blocked popup. Please allow popups.'
    status.value = 'failed'
    emit('failed', error.value)
    return
  }
  popupRef.value = w
  status.value = 'opened'
}

function startVerifying() {
  verifying.value = true
  pollHandle = setInterval(async () => {
    if (!props.order.out_trade_no) return
    try {
      const verified = await paymentApi.verifyOrder({ out_trade_no: props.order.out_trade_no })
      if (verified && verified.status === 'PAID') {
        if (pollHandle) clearInterval(pollHandle)
        emit('paid', Object.assign({}, props.order, verified))
      }
    } catch (e) {
      // 静默继续轮询
    }
    if (popupRef.value && popupRef.value.closed) {
      status.value = 'closed'
    }
  }, 2500)
}

async function handleClosed() {
  if (pollHandle) clearInterval(pollHandle)
  if (!props.order.out_trade_no) return
  try {
    const verified = await paymentApi.verifyOrder({ out_trade_no: props.order.out_trade_no })
    if (verified && verified.status === 'PAID') {
      emit('paid', Object.assign({}, props.order, verified))
      return
    }
  } catch (e) {
    error.value = getErrorMessage(e)
  }
  emit('cancelled')
}

function reopen() {
  status.value = 'opening'
  error.value = ''
  openPopup()
}
</script>

<template>
  <section class="payment-stripe-popup">
    <header>
      <span class="eyebrow">Stripe Popup</span>
      <strong>{{ order.currency || 'USD' }} · {{ (order.pay_amount ?? order.amount).toFixed(2) }}</strong>
    </header>

    <div class="status-card">
      <div v-if="status === 'opening'" class="state">{{ t('payment.redirectTitle') }}</div>
      <div v-else-if="status === 'opened'" class="state">
        {{ t('payment.redirectHint', { method: 'Stripe' }) }}
      </div>
      <div v-else-if="status === 'closed'" class="state warn">
        {{ t('payment.errorOrderCancelled') }}
      </div>
      <div v-else class="state failed">
        {{ error || t('payment.errorCreateOrder') }}
      </div>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div class="actions">
      <button v-if="status === 'opened'" class="ghost" type="button" @click="popupRef?.focus?.()">
        {{ t('payment.redirectOpen') }}
      </button>
      <button v-if="status === 'closed' || status === 'failed'" class="primary" type="button" @click="reopen">
        {{ t('payment.resultRetry') }}
      </button>
    </div>

    <small class="secure">{{ t('payment.secure') }}</small>
  </section>
</template>

<style scoped>
.payment-stripe-popup {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.payment-stripe-popup header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.payment-stripe-popup .eyebrow {
  font-size: 0.6rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.payment-stripe-popup header strong {
  font-size: 0.96rem;
  color: rgba(255, 255, 255, 0.92);
  font-variant-numeric: tabular-nums;
}
.status-card {
  padding: 16px;
  border-radius: 10px;
  background: rgba(120, 175, 230, 0.08);
  border: 1px solid rgba(120, 175, 230, 0.18);
  text-align: center;
}
.status-card .state.warn {
  background: rgba(255, 196, 102, 0.08);
  border-color: rgba(255, 196, 102, 0.32);
  color: #e8c684;
}
.status-card .state.failed {
  background: rgba(244, 139, 139, 0.08);
  border-color: rgba(244, 139, 139, 0.32);
  color: #f48b8b;
}
.status-card .state {
  font-size: 0.88rem;
  color: rgba(255, 255, 255, 0.85);
}
.error {
  margin: 0;
  color: #f48b8b;
  font-size: 0.78rem;
}
.actions {
  display: flex;
  gap: 8px;
}
.actions button {
  flex: 1;
  min-height: 38px;
  padding: 0 14px;
  border-radius: 8px;
  border: 1px solid transparent;
  font-size: 0.78rem;
  font-family: inherit;
  cursor: pointer;
}
.actions button.ghost {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.78);
}
.actions button.primary {
  background: #79c4f5;
  border-color: #79c4f5;
  color: #071019;
  font-weight: 600;
}
.secure {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}
</style>
