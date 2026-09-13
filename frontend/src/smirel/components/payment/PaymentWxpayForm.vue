<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CreateOrderResponse, WechatJSAPIPayload } from '../../api/payment'

const props = defineProps<{
  order: CreateOrderResponse
  inWechatBrowser?: boolean
}>()

const emit = defineEmits<{
  (e: 'paid', order?: CreateOrderResponse): void
  (e: 'cancelled'): void
}>()

const { t } = useI18n()

const error = ref('')

const mode = computed<'native' | 'h5' | 'jsapi' | 'oauth'>(() => {
  if (props.inWechatBrowser && props.order.jsapi) return 'jsapi'
  if (props.order.oauth?.authorize_url) return 'oauth'
  if (props.order.pay_url) return 'h5'
  if (props.order.qr_code) return 'native'
  return 'native'
})

async function invokeJSAPI(payload: WechatJSAPIPayload) {
  if (!payload || !payload.appId) {
    error.value = 'JSAPI payload missing'
    return
  }
  const w: any = window as any
  const WxBridge = w.WeixinJSBridge
  if (!WxBridge) {
    error.value = t('payment.jsapiFailed')
    return
  }
  WxBridge.invoke(
    'getBrandWCPayRequest',
    {
      appId: payload.appId,
      timeStamp: payload.timeStamp,
      nonceStr: payload.nonceStr,
      package: payload.package,
      signType: payload.signType || 'MD5',
      paySign: payload.paySign,
    },
    (res: any) => {
      if (res.err_msg === 'get_brand_wcpay_request:ok') {
        emit('paid')
      } else if (res.err_msg === 'get_brand_wcpay_request:cancel') {
        emit('cancelled')
      } else {
        error.value = res.err_msg || t('payment.jsapiFailed')
      }
    },
  )
}

async function startJSAPI() {
  error.value = ''
  try {
    await invokeJSAPI(props.order.jsapi!)
  } catch (e: any) {
    error.value = e?.message || t('payment.jsapiFailed')
  }
}

function startH5() {
  if (props.order.pay_url) {
    window.location.href = props.order.pay_url
  }
}

function startOAuth() {
  if (props.order.oauth?.authorize_url) {
    window.location.href = props.order.oauth.authorize_url
  }
}
</script>

<template>
  <section class="payment-wxpay-form">
    <header>
      <span class="eyebrow">{{ t('payment.method.wxpay') }}</span>
      <strong>{{ order.currency || 'CNY' }} · {{ (order.pay_amount ?? order.amount).toFixed(2) }}</strong>
    </header>

    <div v-if="mode === 'native'" class="qr">
      <img :src="order.qr_code" alt="WeChat Pay QR" />
      <p>{{ t('payment.qrcodeHint', { method: t('payment.method.wxpay') }) }}</p>
    </div>

    <div v-else-if="mode === 'jsapi'" class="jsapi">
      <p>{{ t('payment.jsapiHint') }}</p>
      <button class="primary" type="button" @click="startJSAPI">
        {{ t('payment.jsapiPay') }}
      </button>
    </div>

    <div v-else-if="mode === 'oauth'" class="oauth">
      <p>{{ t('payment.oauthHint') }}</p>
      <button class="primary" type="button" @click="startOAuth">
        {{ t('payment.oauthOpen') }}
      </button>
    </div>

    <div v-else-if="mode === 'h5'" class="h5">
      <p>{{ t('payment.redirectHint', { method: t('payment.method.wxpay') }) }}</p>
      <button class="primary" type="button" @click="startH5">
        {{ t('payment.redirectOpen') }}
      </button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <small class="secure">{{ t('payment.secure') }}</small>
  </section>
</template>

<style scoped>
.payment-wxpay-form {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  align-items: stretch;
}
.payment-wxpay-form header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.payment-wxpay-form .eyebrow {
  font-size: 0.6rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}
.payment-wxpay-form header strong {
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
.qr p,
.jsapi p,
.oauth p,
.h5 p {
  margin: 0;
  font-size: 0.82rem;
  color: rgba(255, 255, 255, 0.7);
  text-align: center;
}
.jsapi,
.oauth,
.h5 {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}
.primary {
  background: #09bb07;
  color: #fff;
  border: 1px solid #09bb07;
  border-radius: 10px;
  min-width: 220px;
  min-height: 44px;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
}
.error {
  margin: 0;
  color: #f48b8b;
  font-size: 0.78rem;
  text-align: center;
}
.secure {
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}
</style>
