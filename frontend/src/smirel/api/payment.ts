// 支付域类型定义 + API service
//
// 后端响应包络：{ code: 0, message: 'success', data: T }
// core/api.ts 的 response 拦截器已自动解包 code/data，
// 因此本文件所有方法返回的都是 data 字段（Promise<T>）。
//
// 类型字段严格对齐后端 service struct。

import { api } from '../core/api'

// ---------- 枚举 / 联合类型 ----------

export type PaymentType =
  | 'alipay'
  | 'wxpay'
  | 'alipay_direct'
  | 'wxpay_direct'
  | 'stripe'
  | 'card'
  | 'link'
  | 'easypay'
  | 'airwallex'

export function basePaymentType(t: PaymentType): string {
  if (t === 'easypay') return 'easypay'
  if (t === 'airwallex') return 'airwallex'
  if (t === 'stripe' || t === 'card' || t === 'link') return 'stripe'
  if (typeof t === 'string' && t.startsWith('alipay')) return 'alipay'
  if (typeof t === 'string' && t.startsWith('wxpay')) return 'wxpay'
  return t
}

export type OrderType = 'balance' | 'subscription'

export type OrderStatus =
  | 'PENDING'
  | 'PAID'
  | 'RECHARGING'
  | 'COMPLETED'
  | 'EXPIRED'
  | 'CANCELLED'
  | 'FAILED'
  | 'REFUND_REQUESTED'
  | 'REFUNDING'
  | 'REFUND_PENDING'
  | 'PARTIALLY_REFUNDED'
  | 'REFUNDED'
  | 'REFUND_FAILED'

export type CreatePaymentResultType =
  | 'order_created'
  | 'oauth_required'
  | 'jsapi_ready'

export type PaymentEnv = 'prod' | 'demo'

export type PaymentMode = 'qrcode' | 'redirect' | 'popup' | 'jsapi'

export type LoadBalanceStrategy = 'round-robin' | 'random' | 'weighted'

export type VisibleMethodSource =
  | 'official_alipay'
  | 'easypay_alipay'
  | 'official_wxpay'
  | 'easypay_wxpay'

export interface ProviderInstance {
  id: number
  provider_key: PaymentType
  name: string
  config: Record<string, string>
  supported_types: PaymentType[]
  limits: string
  enabled: boolean
  refund_enabled: boolean
  allow_user_refund: boolean
  sort_order: number
  payment_mode: PaymentMode
}

export interface WechatOAuthInfo {
  authorize_url?: string
  appid?: string
  openid?: string
  scope?: string
  state?: string
  redirect_url?: string
}

export interface WechatJSAPIPayload {
  appId?: string
  timeStamp?: string
  nonceStr?: string
  package?: string
  signType?: string
  paySign?: string
}

export interface CreateOrderResponse {
  order_id: number
  amount: number
  pay_amount: number
  fee_rate: number
  status: OrderStatus
  result_type: CreatePaymentResultType
  payment_type: PaymentType
  out_trade_no?: string
  pay_url?: string
  qr_code?: string
  client_secret?: string
  intent_id?: string
  currency?: string
  country_code?: string
  payment_env?: PaymentEnv
  oauth?: WechatOAuthInfo
  jsapi?: WechatJSAPIPayload
  expires_at: string
  payment_mode?: PaymentMode
  resume_token?: string
  alipay_mobile_precreate_deep_link?: boolean
}

export interface MethodLimits {
  payment_type: PaymentType
  display_name?: string
  currency: string
  fee_rate: number
  daily_limit: number
  single_min: number
  single_max: number
}

export interface MethodLimitsResponse {
  methods: Record<string, MethodLimits>
  global_min: number
  global_max: number
}

export interface CheckoutPlan {
  id: number
  group_id: number
  group_platform?: string
  group_name?: string
  rate_multiplier?: number
  peak_rate_enabled?: boolean
  peak_start?: string
  peak_end?: string
  peak_rate_multiplier?: number
  daily_limit_usd?: number | null
  weekly_limit_usd?: number | null
  monthly_limit_usd?: number | null
  supported_model_scopes?: string[]
  name: string
  description: string
  price: number
  original_price?: number | null
  currency?: string
  validity_days: number
  validity_unit: string
  features: string[]
  product_name: string
}

export interface CheckoutInfoResponse {
  methods: Record<string, MethodLimits>
  global_min: number
  global_max: number
  plans: CheckoutPlan[]
  balance_disabled: boolean
  balance_recharge_multiplier: number
  subscription_usd_to_cny_rate: number
  recharge_fee_rate: number
  help_text: string
  help_image_url: string
  stripe_publishable_key: string
  alipay_force_qrcode: boolean
  alipay_mobile_precreate_deep_link: boolean
}

export interface PaymentConfig {
  enabled: boolean
  min_amount: number
  max_amount: number
  daily_limit: number
  order_timeout_minutes: number
  max_pending_orders: number
  enabled_payment_types: PaymentType[]
  balance_disabled: boolean
  balance_recharge_multiplier: number
  subscription_usd_to_cny_rate: number
  recharge_fee_rate: number
  load_balance_strategy: LoadBalanceStrategy | ''
  product_name_prefix: string
  product_name_suffix: string
  help_image_url: string
  help_text: string
  stripe_publishable_key?: string
  cancel_rate_limit_enabled: boolean
  cancel_rate_limit_max: number
  cancel_rate_limit_window: number
  cancel_rate_limit_unit: string
  cancel_rate_limit_window_mode: string
  alipay_force_qrcode: boolean
  alipay_mobile_precreate_deep_link: boolean
  payment_visible_method_alipay_source?: VisibleMethodSource
  payment_visible_method_wxpay_source?: VisibleMethodSource
  payment_visible_method_alipay_enabled?: boolean
  payment_visible_method_wxpay_enabled?: boolean
}

export type PaymentConfigUpdateRequest = {
  enabled?: boolean
  min_amount?: number
  max_amount?: number
  daily_limit?: number
  order_timeout_minutes?: number
  max_pending_orders?: number
  enabled_payment_types?: PaymentType[]
  balance_disabled?: boolean
  balance_recharge_multiplier?: number
  subscription_usd_to_cny_rate?: number
  recharge_fee_rate?: number
  load_balance_strategy?: LoadBalanceStrategy
  product_name_prefix?: string
  product_name_suffix?: string
  help_image_url?: string
  help_text?: string
  cancel_rate_limit_enabled?: boolean
  cancel_rate_limit_max?: number
  cancel_rate_limit_window?: number
  cancel_rate_limit_unit?: string
  cancel_rate_limit_window_mode?: string
  alipay_force_qrcode?: boolean
  alipay_mobile_precreate_deep_link?: boolean
  payment_visible_method_alipay_source?: VisibleMethodSource
  payment_visible_method_wxpay_source?: VisibleMethodSource
  payment_visible_method_alipay_enabled?: boolean
  payment_visible_method_wxpay_enabled?: boolean
}

export interface PaymentOrder {
  id: number
  user_id: number
  amount: number
  pay_amount: number
  fee_rate: number
  currency?: string
  payment_type: PaymentType
  out_trade_no: string
  status: OrderStatus
  order_type: OrderType
  created_at: string
  expires_at: string
  paid_at?: string | null
  completed_at?: string | null
  refund_amount: number
  refund_reason?: string | null
  refund_requested_at?: string | null
  refund_requested_by?: string | null
  refund_request_reason?: string | null
  plan_id?: number | null
  provider_instance_id?: string | null
}

export interface AdminPaymentOrder extends PaymentOrder {
  user_email?: string
  user_name?: string
  user_notes?: string | null
  recharge_code?: string
  payment_trade_no?: string
  pay_url?: string | null
  qr_code?: string | null
  qr_code_img?: string | null
  subscription_group_id?: number | null
  subscription_days?: number | null
  provider_key?: string | null
  refund_at?: string | null
  force_refund?: boolean
  failed_at?: string | null
  failed_reason?: string | null
  client_ip?: string
  src_host?: string
  src_url?: string | null
  updated_at: string
}

export interface AdminOrderAuditLog {
  order_id: string
  action: string
  detail: string
  operator: string
  created_at: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface PublicOrderResolveResult {
  order: PaymentOrder
  audit_logs?: AdminOrderAuditLog[]
}

export interface AdminSubscriptionPlan {
  id: number
  group_id: number
  group_platform?: string
  group_name?: string
  rate_multiplier?: number
  daily_limit_usd?: number | null
  weekly_limit_usd?: number | null
  monthly_limit_usd?: number | null
  supported_model_scopes?: string[]
  name: string
  description: string
  price: number
  original_price?: number | null
  currency?: string
  validity_days: number
  validity_unit: string
  features: string
  product_name: string
  for_sale: boolean
  sort_order: number
  created_at?: string
  updated_at?: string
}

export interface CreatePlanRequest {
  group_id: number
  name: string
  description: string
  price: number
  original_price?: number | null
  currency?: string
  validity_days: number
  validity_unit: string
  features: string
  product_name: string
  for_sale: boolean
  sort_order: number
}

export type UpdatePlanRequest = Partial<CreatePlanRequest>

export interface CreateProviderRequest {
  provider_key: PaymentType
  name: string
  config: Record<string, string>
  supported_types: PaymentType[]
  enabled: boolean
  payment_mode: PaymentMode
  sort_order: number
  limits: string
  refund_enabled: boolean
  allow_user_refund: boolean
}

export interface UpdateProviderRequest {
  name?: string
  config?: Record<string, string>
  supported_types?: PaymentType[]
  enabled?: boolean
  payment_mode?: PaymentMode
  sort_order?: number
  limits?: string
  refund_enabled?: boolean
  allow_user_refund?: boolean
}

export type CurrencyAmounts = Record<string, number>

export interface DailyStats {
  date: string
  amount: CurrencyAmounts
  count: number
}

export interface PaymentMethodStat {
  type: string
  amount: CurrencyAmounts
  count: number
}

export interface TopUserStat {
  user_id: number
  email: string
  username: string
  amount: CurrencyAmounts
  count: number
}

export interface TopUsersByCurrency {
  by_currency: Record<string, TopUserStat[]>
}

export interface DashboardStats {
  today_amount: CurrencyAmounts
  total_amount: CurrencyAmounts
  today_count: number
  total_count: number
  avg_amount: CurrencyAmounts
  pending_orders: number
  daily_series: DailyStats[]
  payment_methods: PaymentMethodStat[]
  top_users: TopUsersByCurrency
}

export interface CreateOrderRequest {
  amount: number
  payment_type: PaymentType
  order_type?: OrderType
  plan_id?: number
  openid?: string
  wechat_resume_token?: string
  return_url?: string
  payment_source?: string
  is_mobile?: boolean
}

async function ok<T = any>(p: Promise<any>): Promise<T> {
  const r = await p
  return r.data
}

export const paymentApi = {
  getConfig: () => ok(api.get('/payment/config')),
  getCheckoutInfo: () => ok(api.get('/payment/checkout-info')),
  listPlans: () => ok(api.get('/payment/plans')),
  getLimits: () => ok(api.get('/payment/limits')),
  createOrder: (body: CreateOrderRequest) => ok<CreateOrderResponse>(api.post('/payment/orders', body)),
  verifyOrder: (body: any) => ok<any>(api.post('/payment/orders/verify', body)),
  listMyOrders: (params = {}) => ok(api.get('/payment/orders/my', { params })),
  getOrder: (id: string | number) => ok<PaymentOrder>(api.get('/payment/orders/' + id)),
  cancelOrder: (id: string | number) => ok<PaymentOrder>(api.post('/payment/orders/' + id + '/cancel', {})),
  requestRefund: (id: string | number, body: any) => ok<PaymentOrder>(api.post('/payment/orders/' + id + '/refund-request', body)),
  getRefundEligibleProviders: () => ok(api.get('/payment/orders/refund-eligible-providers')),
  resolvePublicOrder: (body: any) => ok<any>(api.post('/payment/public/orders/resolve', body)),
}

export const paymentAdminApi = {
  getDashboard: (days = 30) => ok(api.get('/admin/payment/dashboard', { params: { days } })),
  getConfig: () => ok(api.get('/admin/payment/config')),
  updateConfig: (body: PaymentConfigUpdateRequest) => ok<PaymentConfig>(api.put('/admin/payment/config', body)),
  listOrders: (params = {}) => ok(api.get('/admin/payment/orders', { params })),
  getOrder: (id: string | number) => ok<PaymentOrder>(api.get('/admin/payment/orders/' + id)),
  cancelOrder: (id: string | number) => ok<PaymentOrder>(api.post('/admin/payment/orders/' + id + '/cancel', {})),
  retryFulfillment: (id: string | number) => ok<AdminPaymentOrder>(api.post('/admin/payment/orders/' + id + '/retry', {})),
  refund: (id: string | number, body: any) => ok<AdminPaymentOrder>(api.post('/admin/payment/orders/' + id + '/refund', body)),
  queryRefund: (id: string | number) => ok<AdminPaymentOrder>(api.post('/admin/payment/orders/' + id + '/refund/query', {})),
  listPlans: () => ok(api.get('/admin/payment/plans')),
  createPlan: (body: CreatePlanRequest) => ok<AdminSubscriptionPlan>(api.post('/admin/payment/plans', body)),
  updatePlan: (id: string | number, body: UpdatePlanRequest) => ok<AdminSubscriptionPlan>(api.put('/admin/payment/plans/' + id, body)),
  deletePlan: (id: string | number) => ok<{ success: boolean }>(api.delete('/admin/payment/plans/' + id)),
  listProviders: () => ok(api.get('/admin/payment/providers')),
  createProvider: (body: CreateProviderRequest) => ok<ProviderInstance>(api.post('/admin/payment/providers', body)),
  updateProvider: (id: string | number, body: UpdateProviderRequest) => ok<ProviderInstance>(api.put('/admin/payment/providers/' + id, body)),
  deleteProvider: (id: string | number) => ok<{ success: boolean }>(api.delete('/admin/payment/providers/' + id)),
}
