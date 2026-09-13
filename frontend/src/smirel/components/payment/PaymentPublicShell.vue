<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { paymentApi, type CreateOrderResponse } from '../../api/payment'
import { useSession } from '../../core/session'
import PaymentQrCodeCard from './PaymentQrCodeCard.vue'
import PaymentStripeForm from './PaymentStripeForm.vue'
import PaymentStripePopupForm from './PaymentStripePopupForm.vue'
import PaymentAirwallexForm from './PaymentAirwallexForm.vue'
import PaymentRedirect from './PaymentRedirect.vue'
import PaymentWechatOAuth from './PaymentWechatOAuth.vue'
import PaymentAlipayForm from './PaymentAlipayForm.vue'
import PaymentWxpayForm from './PaymentWxpayForm.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { isAuthenticated } = useSession()

const order = ref<CreateOrderResponse | null>(null)
const loading = ref(true)
const error = ref('')

const outTrade = computed(() => String(route.query.out_trade_no || ''))
const resumeToken = computed(() => String(route.query.resume_token || ''))
const isWechatBrowser = computed(() => {
  const ua = (navigator.userAgent || '').toLowerCase()
  return ua.indexOf('micromessenger') !== -1
})

async function fetchOrder() {
  if (!outTrade.value) {
    error.value = 'Missing out_trade_no'
    loading.value = false
    return
  }
  try {
    if (isAuthenticated.value && resumeToken.value) {
      const r = await paymentApi.verifyOrder({ out_trade_no: outTrade.value })
      order.value = r as CreateOrderResponse
    } else if (resumeToken.value) {
      const r = await paymentApi.resolvePublicOrder({
        out_trade_no: outTrade.value,
        resume_token: resumeToken.value,
      })
      order.value = r.order as unknown as CreateOrderResponse
    } else {
      const r = await paymentApi.verifyOrder({ out_trade_no: outTrade.value })
      order.value = r as CreateOrderResponse
    }
  } catch (e: any) {
    error.value = e?.message || t('payment.errorVerifyOrder')
  } finally {
    loading.value = false
  }
}

onMounted(fetchOrder)

function onPaid(o?: CreateOrderResponse) {
  if (o && o.out_trade_no) {
    router.push({ path: '/payment/result', query: { out_trade_no: o.out_trade_no, status: 'PAID' } })
  } else {
    router.push('/payment/result')
  }
}

function onCancelled() {
  router.push('/subscriptions')
}

function onExpired() {
  router.push({ path: '/payment/result', query: { out_trade_no: outTrade.value, status: 'EXPIRED' } })
}

function onFailed(msg: string) {
  error.value = msg
}

function retry() {
  loading.value = true
  error.value = ''
  fetchOrder()
}
</script>

<template>
  <div class="payment-public-shell">
    <header class="shell-head">
      <RouterLink to="/subscriptions" class="back">{{ t('payment.back') }}</RouterLink>
      <span class="title">{{ t('payment.summary') }}</span>
      <span class="amount" v-if="order">{{ order.currency || 'CNY' }} · {{ (order.pay_amount ?? order.amount).toFixed(2) }}</span>
    </header>

    <div v-if="loading" class="loading">{{ t('payment.loading') }}</div>

    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <button class="ghost" type="button" @click="retry">{{ t('payment.retry') }}</button>
      <RouterLink class="ghost" to="/subscriptions">{{ t('payment.resultBack') }}</RouterLink>
    </div>

    <template v-else-if="order">
      <PaymentQrCodeCard
        v-if="$route.path === '/payment/qrcode'"
        :order="order"
        @paid="onPaid"
        @cancelled="onCancelled"
        @expired="onExpired"
        @refresh="retry"
      />
      <PaymentStripeForm
        v-else-if="$route.path === '/payment/stripe'"
        :order="order"
        @paid="onPaid"
        @failed="onFailed"
      />
      <PaymentStripePopupForm
        v-else-if="$route.path === '/payment/stripe-popup'"
        :order="order"
        @paid="onPaid"
        @failed="onFailed"
        @cancelled="onCancelled"
      />
      <PaymentAirwallexForm
        v-else-if="$route.path === '/payment/airwallex'"
        :order="order"
        @paid="onPaid"
        @failed="onFailed"
      />
      <PaymentRedirect v-else-if="$route.path === '/payment/redirect'" />
      <PaymentWechatOAuth v-else-if="$route.path === '/payment/wechat-oauth'" />
      <PaymentAlipayForm
        v-else-if="order.payment_type && order.payment_type.startsWith('alipay')"
        :order="order"
        @paid="onPaid"
        @cancelled="onCancelled"
      />
      <PaymentWxpayForm
        v-else-if="order.payment_type && order.payment_type.startsWith('wxpay')"
        :order="order"
        :in-wechat-browser="isWechatBrowser"
        @paid="onPaid"
        @cancelled="onCancelled"
      />
      <div v-else class="unknown">
        <p>Unknown payment type: {{ order.payment_type }}</p>
        <RouterLink class="ghost" to="/subscriptions">{{ t('payment.resultBack') }}</RouterLink>
      </div>
    </template>
  </div>
</template>

<style scoped>
.payment-public-shell {
  width: min(560px, calc(100vw - 32px));
  margin: 24px auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.shell-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px;
}
.shell-head .back {
  font-size: 0.74rem;
  color: rgba(255, 255, 255, 0.55);
  text-decoration: none;
}
.shell-head .title {
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.shell-head .amount {
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.85);
  font-variant-numeric: tabular-nums;
}
.loading {
  text-align: center;
  padding: 48px 16px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.86rem;
}
.error {
  padding: 24px 18px;
  border-radius: 12px;
  border: 1px solid rgba(244, 139, 139, 0.32);
  background: rgba(244, 139, 139, 0.08);
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}
.error p {
  margin: 0;
  color: #f48b8b;
  font-size: 0.86rem;
}
.error .ghost {
  padding: 0 14px;
  height: 36px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.78);
  font-size: 0.78rem;
  text-decoration: none;
  line-height: 34px;
}
.unknown {
  text-align: center;
  padding: 36px 16px;
  color: rgba(255, 255, 255, 0.6);
}
.unknown .ghost {
  display: inline-block;
  margin-top: 12px;
  padding: 0 14px;
  height: 36px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.78);
  font-size: 0.78rem;
  text-decoration: none;
  line-height: 34px;
}
</style>
