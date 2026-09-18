<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import { usePaymentOrderPolling } from '../../composables/usePaymentOrderPolling'
import { basePaymentType, type CreateOrderResponse } from '../../api/payment'
import PaymentStatusBadge from './PaymentStatusBadge.vue'

const props = defineProps<{
  order: CreateOrderResponse
}>()

const emit = defineEmits<{
  (e: 'paid', order: CreateOrderResponse): void
  (e: 'cancelled', order: CreateOrderResponse): void
  (e: 'expired', order: CreateOrderResponse): void
  (e: 'refresh'): void
}>()

const { t } = useI18n()

const qrDataUrl = ref('')
const qrError = ref('')
const copied = ref(false)
const now = ref(Date.now())

let timer: ReturnType<typeof setInterval> | null = null

const outTradeNo = computed(() => props.order.out_trade_no || '')

const { status, start: startPolling, pause, outcome } = usePaymentOrderPolling(
  () => outTradeNo.value,
  {
    intervalMs: 2500,
    maxAttempts: 360,
    onPaid: (o) => emit('paid', Object.assign({}, props.order, o)),
    onTerminal: (o) => emit('cancelled', Object.assign({}, props.order, o)),
    onError: () => {},
  },
)

const methodLabel = computed(() => {
  const g = basePaymentType(props.order.payment_type)
  return t(`payment.method.${g === 'card' || g === 'link' ? 'stripe' : g}`)
})

const expiresAt = computed(() => Date.parse(props.order.expires_at || '') || 0)
const remainingMs = computed(() => Math.max(0, expiresAt.value - now.value))
const expired = computed(() => remainingMs.value <= 0)

const fmtCountdown = computed(() => {
  const ms = remainingMs.value
  if (ms <= 0) return t('payment.qrcodeExpired')
  const total = Math.floor(ms / 1000)
  const m = Math.floor(total / 60).toString().padStart(2, '0')
  const s = (total % 60).toString().padStart(2, '0')
  return `${m}:${s}`
})

const displayAmount = computed(() => {
  const cur = props.order.currency || 'CNY'
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: cur }).format(props.order.pay_amount ?? props.order.amount)
  } catch {
    return `${cur} ${(props.order.pay_amount ?? props.order.amount).toFixed(2)}`
  }
})

async function renderQr() {
  if (!props.order.qr_code) {
    qrDataUrl.value = ''
    qrError.value = ''
    return
  }
  try {
    qrDataUrl.value = await QRCode.toDataURL(props.order.qr_code, {
      margin: 1,
      width: 220,
      color: { dark: '#0d1117', light: '#ffffff' },
    })
    qrError.value = ''
  } catch (e: any) {
    qrError.value = e?.message || 'QR error'
  }
}

watch(() => props.order.qr_code, renderQr, { immediate: true })

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now() }, 1000)
  startPolling()
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  pause()
})

watch(expired, (v) => {
  if (v) emit('expired', props.order)
})

watch(outcome, (v) => {
  if (v === 'timeout') emit('expired', props.order)
})

async function copyLink() {
  if (!props.order.qr_code) return
  try {
    await navigator.clipboard.writeText(props.order.qr_code)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    /* ignore */
  }
}

function refresh() {
  renderQr()
  emit('refresh')
}

function confirmPaid() {
  if (status.value) emit('paid', Object.assign({}, props.order, { status: status.value }))
}

function cancel() {
  pause()
  emit('cancelled', props.order)
}
</script>

<template>
  <section class="payment-qrcode-card" :class="{ expired }">
    <header class="head">
      <div class="title">
        <span class="eyebrow">{{ t('payment.qrcodeTitle') }}</span>
        <strong>{{ methodLabel }}</strong>
      </div>
      <PaymentStatusBadge v-if="status" :status="status" />
    </header>

    <div class="qr-area">
      <div v-if="qrError" class="qr-error">
        {{ qrError }}
      </div>
      <img v-else-if="qrDataUrl" :src="qrDataUrl" alt="QR Code" />
      <div v-else class="qr-placeholder">{{ t('payment.loading') }}</div>
    </div>

    <p class="hint">
      {{ t('payment.qrcodeHint', { method: methodLabel }) }}
    </p>

    <dl class="meta">
      <div>
        <dt>{{ t('payment.qrcodeAmount') }}</dt>
        <dd>{{ displayAmount }}</dd>
      </div>
      <div>
        <dt>{{ t('payment.qrcodeExpire') }}</dt>
        <dd :class="{ danger: expired }">{{ fmtCountdown }}</dd>
      </div>
    </dl>

    <div class="actions">
      <button class="ghost" type="button" :disabled="expired" @click="refresh">
        {{ t('payment.qrcodeRefresh') }}
      </button>
      <button class="ghost" type="button" @click="copyLink">
        {{ copied ? t('payment.qrcodeCopied') : t('payment.qrcodeCopyLink') }}
      </button>
      <button class="primary" type="button" :disabled="expired" @click="confirmPaid">
        {{ t('payment.qrcodePayCompleted') }}
      </button>
      <button class="danger" type="button" @click="cancel">
        {{ t('payment.qrcodeCancelOrder') }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.payment-qrcode-card {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(18, 22, 28, 0.92), rgba(12, 16, 22, 0.96));
  padding: 22px 22px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.payment-qrcode-card.expired {
  opacity: 0.78;
}
.payment-qrcode-card .head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.payment-qrcode-card .eyebrow {
  font-size: 0.6rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.payment-qrcode-card .title strong {
  display: block;
  font-size: 1.05rem;
  font-weight: 620;
  margin-top: 4px;
  color: rgba(255, 255, 255, 0.92);
}
.qr-area {
  align-self: center;
  width: 220px;
  height: 220px;
  border-radius: 12px;
  background: #fff;
  padding: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.qr-area img {
  width: 100%;
  height: 100%;
  display: block;
}
.qr-area .qr-placeholder,
.qr-area .qr-error {
  color: #0d1117;
  font-size: 0.8rem;
}
.qr-area .qr-error {
  color: #b00020;
  text-align: center;
}
.hint {
  margin: 0;
  text-align: center;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.6);
}
.meta {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin: 0;
}
.meta div {
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.02);
}
.meta dt {
  font-size: 0.62rem;
  color: rgba(255, 255, 255, 0.45);
  margin-bottom: 4px;
}
.meta d {
  display: block;
  margin: 0;
  font-size: 0.96rem;
  color: rgba(255, 255, 255, 0.92);
  font-variant-numeric: tabular-nums;
}
.meta d.danger {
  color: #f48b8b;
}
.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.actions button {
  flex: 1 1 auto;
  min-height: 38px;
  padding: 0 14px;
  border-radius: 8px;
  border: 1px solid transparent;
  font-size: 0.76rem;
  font-family: inherit;
  cursor: pointer;
}
.actions button.ghost {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.78);
}
.actions button.ghost:hover {
  background: rgba(255, 255, 255, 0.08);
}
.actions button.primary {
  background: #79c4f5;
  border-color: #79c4f5;
  color: #071019;
  font-weight: 600;
}
.actions button.primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.actions button.danger {
  background: transparent;
  border-color: rgba(244, 139, 139, 0.32);
  color: #f48b8b;
}
.actions button.danger:hover {
  background: rgba(244, 139, 139, 0.08);
}
</style>
