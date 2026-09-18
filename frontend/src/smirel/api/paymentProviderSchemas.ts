// 支付 Provider 配置字段元数据。
//
// 与后端 service/payment_config_providers.go 的
//   providerSensitiveConfigFields
//   providerPendingOrderProtectedConfigFields
// 保持一一对应。如果后端新增/移除某个 config 字段，必须同步更新本文件，
// 否则 admin UI 会缺字段或暴露应该脱敏的字段。

import type { PaymentType, PaymentMode } from './payment'

export type ProviderFieldType = 'string' | 'secret' | 'pem' | 'url' | 'boolean' | 'number'

export interface ProviderFieldDef {
  key: string
  label: string
  type: ProviderFieldType
  required: boolean
  sensitive: boolean
  // 受保护字段（敏感 + 在用户有 in-progress 订单时禁止修改）
  protectedWhenPending?: boolean
  description?: string
  placeholder?: string
}

export interface PaymentProviderSchema {
  provider_key: PaymentType
  display_name: string
  // 支持的支付方式（多选）
  supported_payment_types: PaymentType[]
  // 默认 payment_mode
  default_payment_mode: PaymentMode
  // 默认 supported_types
  default_supported_types: PaymentType[]
  fields: ProviderFieldDef[]
  // 文档链接（来自 docs/PAYMENT.md）
  docs_url?: string
  // 配置回填提示
  hint?: string
}

// ----- 5 个 provider 的字段定义 -----

const easypayFields: ProviderFieldDef[] = [
  { key: 'pid', label: '商户 ID (PID)', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: 'EasyPay/ZPay 商户 PID', placeholder: '1000xx' },
  { key: 'pkey', label: '商户密钥 (PKey)', type: 'secret', required: true, sensitive: true, protectedWhenPending: true, description: 'EasyPay/ZPay 商户密钥' },
  { key: 'apiBase', label: 'API 基础地址', type: 'url', required: true, sensitive: false, description: 'EasyPay/ZPay 接口地址（不需要包含 /submit.php 等子路径）', placeholder: 'https://pay.example.com' },
  { key: 'notifyUrl', label: '异步通知地址', type: 'url', required: true, sensitive: false, description: '订单回调 URL', placeholder: 'https://your-domain.com/api/v1/payment/webhook/easypay' },
  { key: 'returnUrl', label: '同步返回地址', type: 'url', required: true, sensitive: false, description: '支付完成后跳转 URL' },
  { key: 'cid', label: '聚合支付渠道 ID (CID)', type: 'string', required: false, sensitive: false, description: '聚合场景下子渠道 ID（可留空）' },
  { key: 'cidAlipay', label: '支付宝 CID', type: 'string', required: false, sensitive: false, description: '单独指定支付宝子渠道 ID' },
  { key: 'cidWxpay', label: '微信 CID', type: 'string', required: false, sensitive: false, description: '单独指定微信子渠道 ID' },
]

const alipayFields: ProviderFieldDef[] = [
  { key: 'appId', label: 'AppID', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '支付宝开放平台 AppID', placeholder: '2021000000000000' },
  { key: 'privateKey', label: '应用私钥', type: 'pem', required: true, sensitive: true, protectedWhenPending: true, description: 'RSA2 私钥（PKCS#8 或 PKCS#1 均可，后端自动转换）' },
  { key: 'publicKey', label: '支付宝公钥', type: 'pem', required: true, sensitive: true, protectedWhenPending: true, description: '支付宝开放平台→应用公钥 对应的支付宝公钥' },
  { key: 'alipayPublicKey', label: '支付宝公钥（备选）', type: 'pem', required: false, sensitive: true, protectedWhenPending: true, description: '与 publicKey 二选一；公钥值变化时可填这里覆盖' },
  { key: 'notifyUrl', label: '异步通知地址', type: 'url', required: false, sensitive: false, description: '订单回调 URL（不填时使用系统默认）' },
  { key: 'returnUrl', label: '同步返回地址', type: 'url', required: false, sensitive: false, description: '支付完成后跳转 URL' },
]

const wxpayFields: ProviderFieldDef[] = [
  { key: 'appId', label: 'AppID（公众号/小程序）', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '微信公众号/小程序 AppID' },
  { key: 'mpAppId', label: 'JSAPI 公众号 AppID（可选）', type: 'string', required: false, sensitive: false, protectedWhenPending: true, description: 'JSAPI 内调起支付时使用的公众号 AppID，不填时使用 appId' },
  { key: 'mchId', label: '商户号 (MchID)', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '微信支付商户号', placeholder: '1230000109' },
  { key: 'privateKey', label: '商户 API 私钥', type: 'pem', required: true, sensitive: true, protectedWhenPending: true, description: '商户 API 证书私钥' },
  { key: 'apiV3Key', label: 'APIv3 密钥 (32 字符)', type: 'secret', required: true, sensitive: true, protectedWhenPending: true, description: '商户平台 APIv3 密钥，必须 32 字符' },
  { key: 'certSerial', label: '商户证书序列号', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '商户 API 证书序列号', placeholder: 'xxx' },
  { key: 'publicKey', label: '微信支付公钥', type: 'pem', required: true, sensitive: true, protectedWhenPending: true, description: '微信支付公钥（pubkey 验证模式，2024-10 后新商户必须）' },
  { key: 'publicKeyId', label: '公钥 ID', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '微信支付公钥 ID' },
  { key: 'notifyUrl', label: '异步通知地址', type: 'url', required: false, sensitive: false, description: '订单回调 URL' },
  { key: 'h5AppName', label: 'H5 应用名', type: 'string', required: false, sensitive: false, description: 'H5 调起时显示的应用名' },
  { key: 'h5AppUrl', label: 'H5 应用 URL', type: 'url', required: false, sensitive: false, description: 'H5 调起时的应用 URL' },
]

const stripeFields: ProviderFieldDef[] = [
  { key: 'secretKey', label: 'Secret Key', type: 'secret', required: true, sensitive: true, protectedWhenPending: true, description: 'Stripe 平台 Secret Key（建议 sk_live_/sk_test_）', placeholder: 'sk_live_xxx 或 sk_test_xxx' },
  { key: 'publishableKey', label: 'Publishable Key', type: 'string', required: false, sensitive: false, description: '前端加载 Stripe.js 时需要；不填时 Stripe 通道不可在前端发起', placeholder: 'pk_live_xxx 或 pk_test_xxx' },
  { key: 'webhookSecret', label: 'Webhook 签名密钥', type: 'secret', required: true, sensitive: true, protectedWhenPending: true, description: 'Stripe Dashboard → Webhooks → Signing secret' },
  { key: 'currency', label: '结算币种', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '3 位 ISO 4217 小写币种代码', placeholder: 'usd' },
]

const airwallexFields: ProviderFieldDef[] = [
  { key: 'clientId', label: 'Client ID', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: 'Airwallex 平台 Client ID' },
  { key: 'apiKey', label: 'API Key', type: 'secret', required: true, sensitive: true, protectedWhenPending: true, description: 'Airwallex 平台 API Key' },
  { key: 'webhookSecret', label: 'Webhook 签名密钥', type: 'secret', required: true, sensitive: true, protectedWhenPending: true, description: 'Airwallex Webhook 签名密钥' },
  { key: 'apiBase', label: 'API 基础地址', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '生产/沙盒环境 API Base', placeholder: 'https://api.airwallex.com 或 https://api-demo.airwallex.com' },
  { key: 'accountId', label: 'Account ID', type: 'string', required: false, sensitive: false, protectedWhenPending: true, description: '子账户 ID（多账户场景）' },
  { key: 'currency', label: '结算币种', type: 'string', required: true, sensitive: false, protectedWhenPending: true, description: '3 位 ISO 4217 小写币种代码', placeholder: 'usd' },
  { key: 'countryCode', label: '国家/地区代码', type: 'string', required: false, sensitive: false, description: 'Airwallex 收银台国家/地区代码', placeholder: 'US' },
  { key: 'descriptor', label: '账单描述', type: 'string', required: false, sensitive: false, description: '账单对账单上的描述' },
]

// ----- 5 个 provider schema -----

export const PROVIDER_SCHEMAS: Record<string, PaymentProviderSchema> = {
  easypay: {
    provider_key: 'easypay',
    display_name: 'EasyPay / ZPay',
    supported_payment_types: ['easypay', 'alipay', 'wxpay'],
    default_payment_mode: 'redirect',
    default_supported_types: ['easypay', 'alipay', 'wxpay'],
    fields: easypayFields,
    docs_url: '/docs/PAYMENT_CN.md#easypay',
    hint: 'EasyPay / ZPay 聚合支付，单一商户接入支付宝、微信、QQ 等。',
  },
  alipay: {
    provider_key: 'alipay',
    display_name: '支付宝官方',
    supported_payment_types: ['alipay'],
    default_payment_mode: 'qrcode',
    default_supported_types: ['alipay'],
    fields: alipayFields,
    docs_url: '/docs/PAYMENT_CN.md#alipay',
    hint: '支付宝开放平台官方通道，支持当面付、WAP。',
  },
  wxpay: {
    provider_key: 'wxpay',
    display_name: '微信支付官方',
    supported_payment_types: ['wxpay'],
    default_payment_mode: 'qrcode',
    default_supported_types: ['wxpay'],
    fields: wxpayFields,
    docs_url: '/docs/PAYMENT_CN.md#wxpay',
    hint: '微信支付 V3 通道，使用公钥验证模式（pubkey verifier），不支持平台证书。',
  },
  stripe: {
    provider_key: 'stripe',
    display_name: 'Stripe',
    supported_payment_types: ['stripe', 'card', 'link'],
    default_payment_mode: 'popup',
    default_supported_types: ['stripe'],
    fields: stripeFields,
    docs_url: '/docs/PAYMENT_CN.md#stripe',
    hint: 'Stripe 卡支付 + Link 一键支付，前端通过 Stripe.js 渲染 Elements。',
  },
  airwallex: {
    provider_key: 'airwallex',
    display_name: 'Airwallex',
    supported_payment_types: ['airwallex'],
    default_payment_mode: 'popup',
    default_supported_types: ['airwallex'],
    fields: airwallexFields,
    docs_url: '/docs/PAYMENT_CN.md#airwallex',
    hint: 'Airwallex 全球收单，前端通过 Airwallex Drop-in 渲染支付组件。',
  },
}

export function getProviderSchema(key: string): PaymentProviderSchema | undefined {
  return PROVIDER_SCHEMAS[key]
}

export function listProviderSchemas(): PaymentProviderSchema[] {
  return Object.values(PROVIDER_SCHEMAS)
}
