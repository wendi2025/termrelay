// usePaymentCheckout
//
// 单一职责：调用 paymentApi.createOrder 并按 result_type 把结果"分发"到正确的 UI 流。
// 不在这里直接弹二维码 / 调 Stripe / 跳支付页 —— 由调用方在拿到 dispatch 后决定怎么渲染。
// 这样做的好处：同一份逻辑既可以驱动 /payment/qrcode 公开页，又可以驱动 /subscriptions 用户页。

import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { paymentApi, type CreateOrderRequest, type CreateOrderResponse } from '../api/payment'
import { getErrorMessage } from '../core/api'
import { basePaymentType } from '../api/payment'

export type PaymentDispatch =
  | { kind: 'qrcode'; order: CreateOrderResponse }
  | { kind: 'redirect'; order: CreateOrderResponse; pay_url: string }
  | { kind: 'stripe-embedded'; order: CreateOrderResponse; client_secret: string }
  | { kind: 'stripe-popup'; order: CreateOrderResponse; client_secret: string }
  | { kind: 'airwallex'; order: CreateOrderResponse; client_secret: string; intent_id: string }
  | { kind: 'wechat-jsapi'; order: CreateOrderResponse }
  | { kind: 'wechat-oauth'; order: CreateOrderResponse; authorize_url: string }

export interface UsePaymentCheckoutOptions {
  // 创建订单后，是否自动跳到公开支付页（默认 true）
  autoNavigatePublicPage?: boolean
  // return_url 用于 WAP / H5 调起支付成功后的回跳
  returnUrl?: string
  // 前端自行声明移动端
  isMobile?: boolean
}

export function usePaymentCheckout(options: UsePaymentCheckoutOptions = {}) {
  const router = useRouter()
  const submitting = ref(false)
  const error = ref<string>('')
  const lastResponse = ref<CreateOrderResponse | null>(null)
  const lastDispatch = ref<PaymentDispatch | null>(null)

  function classify(order: CreateOrderResponse): PaymentDispatch | null {
    // 1. WxPay 在微信内：result_type=jsapi_ready，jsapi 字段非空 → 立即在当前页 invoke WeixinJSBridge
    if (order.result_type === 'jsapi_ready' && order.jsapi) {
      return { kind: 'wechat-jsapi', order }
    }
    // 2. WxPay 在微信外：result_type=oauth_required，oauth.authorize_url 非空 → 跳微信授权
    if (order.result_type === 'oauth_required' && order.oauth?.authorize_url) {
      return { kind: 'wechat-oauth', order, authorize_url: order.oauth.authorize_url }
    }
    // 3. Airwallex Drop-in：client_secret + intent_id
    if (order.client_secret && order.intent_id && basePaymentType(order.payment_type) === 'airwallex') {
      return { kind: 'airwallex', order, client_secret: order.client_secret, intent_id: order.intent_id }
    }
    // 4. Stripe 弹窗模式：payment_mode=popup
    if (order.client_secret && order.payment_mode === 'popup' && basePaymentType(order.payment_type) === 'stripe') {
      return { kind: 'stripe-popup', order, client_secret: order.client_secret }
    }
    // 5. Stripe 内嵌模式：client_secret + Elements
    if (order.client_secret && basePaymentType(order.payment_type) === 'stripe') {
      return { kind: 'stripe-embedded', order, client_secret: order.client_secret }
    }
    // 6. pay_url（非空且非二维码时优先）
    if (order.pay_url && !order.qr_code) {
      return { kind: 'redirect', order, pay_url: order.pay_url }
    }
    // 7. qr_code（EasyPay 聚合 / Alipay 当面 / WxPay Native）—— result_type 通常为 order_created
    if (order.qr_code) {
      return { kind: 'qrcode', order }
    }
    // 8. 兜底：pay_url 优先
    if (order.pay_url) {
      return { kind: 'redirect', order, pay_url: order.pay_url }
    }
    return null
  }

  async function createOrder(body: CreateOrderRequest): Promise<PaymentDispatch | null> {
    submitting.value = true
    error.value = ''
    lastResponse.value = null
    lastDispatch.value = null
    try {
      const order = await paymentApi.createOrder({
        ...body,
        return_url: body.return_url ?? options.returnUrl,
        is_mobile: body.is_mobile ?? options.isMobile,
      })
      lastResponse.value = order
      const dispatch = classify(order)
      lastDispatch.value = dispatch
      // 自动跳转：把 dispatch 的状态写到 /payment/qrcode 公开页（由该页内的组件按 kind 渲染）
      if (options.autoNavigatePublicPage !== false) {
        await navigateForDispatch(dispatch)
      }
      return dispatch
    } catch (e) {
      error.value = getErrorMessage(e)
      return null
    } finally {
      submitting.value = false
    }
  }

  async function navigateForDispatch(dispatch: PaymentDispatch | null) {
    if (!dispatch) return
    const order = dispatch.order
    const token = order.resume_token
    const outTrade = order.out_trade_no
    const query = { out_trade_no: outTrade, resume_token: token }
    switch (dispatch.kind) {
      case 'qrcode':
        await router.push({ path: '/payment/qrcode', query })
        break
      case 'stripe-embedded':
        await router.push({ path: '/payment/stripe', query })
        break
      case 'stripe-popup':
        await router.push({ path: '/payment/stripe-popup', query })
        break
      case 'airwallex':
        await router.push({ path: '/payment/airwallex', query })
        break
      case 'redirect':
        // 公开支付页有专门的"调起第三方支付"按钮，URL 写进 sessionStorage
        sessionStorage.setItem('smirel.payment.pay_url', dispatch.pay_url)
        await router.push({ path: '/payment/redirect', query })
        break
      case 'wechat-jsapi':
        // 当前页内 invoke WeixinJSBridge；不跳路由
        break
      case 'wechat-oauth':
        // 公开支付页有"前往微信授权"按钮
        sessionStorage.setItem('smirel.payment.authorize_url', dispatch.authorize_url)
        await router.push({ path: '/payment/wechat-oauth', query })
        break
    }
  }

  const resultType = computed(() => lastResponse.value?.result_type ?? '')
  const outTradeNo = computed(() => lastResponse.value?.out_trade_no ?? '')

  return {
    submitting,
    error,
    lastResponse,
    lastDispatch,
    resultType,
    outTradeNo,
    createOrder,
    classify,
  }
}
