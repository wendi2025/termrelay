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
  id?: number
  order_id: string
  action: string
  detail: string
  operator: string
  created_at: string
}

// ---- Smirel 退款/分账（套餐定价方案 9.1 / 9.2 / 9.3） ----

/** 单笔订单的余额账本汇总（退款审核记录口径） */
export interface RefundLedgerSummary {
  order_id: number
  principal_credit: number
  principal_debit: number
  principal_left: number
  bonus_credit: number
  bonus_debit: number
  bonus_left: number
  frozen: boolean
  entry_count: number
}

/** 单条余额账本分录 */
export interface RefundLedgerEntry {
  id: number
  order_id: number
  entry_type: 'principal' | 'bonus'
  direction: 'credit' | 'debit'
  amount: number
  balance_after: number
  memo?: string
  frozen: boolean
  refund_batch_id?: string
  created_at: string
}

/** 余额本金/赠送构成 */
export interface LedgerBalanceBreakdown {
  user_id: number
  account_balance: number
  principal_credit_total: number
  principal_debit_total: number
  principal_left: number
  bonus_credit_total: number
  bonus_debit_total: number
  bonus_left: number
  active_principal_left: number
  active_bonus_left: number
  frozen_principal_left: number
  frozen_bonus_left: number
  /** account_balance 与账本构成之差：>0 表示存在早于账本上线期的历史余额 */
  ledger_gap: number
  entry_count: number
}

/**
 * 退款预览（只读报价）。
 * `*_pay_amount` 为支付币种口径，`*_credit` 为记账单位（user.balance）口径。
 */
export interface RefundPreview {
  order_id: number
  out_trade_no: string
  order_type: OrderType
  order_status: OrderStatus
  user_id: number
  user_email?: string
  user_name?: string
  currency: string
  amount: number
  pay_amount: number
  purchased_at: string
  activated_at?: string | null
  refund_requested_at?: string | null
  refund_requested_by?: string
  refund_request_reason?: string
  /** 已用官方 $ 额度（周期套餐口径；按量充值为 0） */
  used_usd: number
  /** 已扣金额（记账单位） */
  charged_credit: number
  /** 已扣金额（支付币种） */
  charged_pay_amount: number
  /** 该订单关联的赠送余额剩余 */
  bonus_balance: number
  /** 命中的退款策略，例如 balance_principal_minus_used */
  policy: string
  /** 公式结果：可退金额（支付币种） */
  refundable_pay_amount: number
  /** 公式结果：可退金额（记账单位） */
  refundable_credit: number
  full_refund_window: boolean
  force_refund_eligible: boolean
  calculation_breakdown?: Record<string, number>
  /** 按用户当前余额封顶后的上限（记账单位）；仅按量充值订单有值 */
  balance_cap?: number | null
  balance_cap_applied: boolean
  ledger?: RefundLedgerSummary | null
  ledger_entries?: RefundLedgerEntry[] | null
  requestable: boolean
  /** 不可退款原因枚举，前端用 payment.refundBlock.* 本地化 */
  blocked_reason?: string
  /** 该报价是否按「线下退款」口径给出（不调用渠道退款接口，仅记账） */
  offline?: boolean
  generated_at: string
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

/** 模拟支付接口的返回体：后端同时回传人类可读消息与伪造的交易号。 */
export interface AdminPaymentSimulationResult {
  message: string
  trade_no: string
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
  /** 退款预览（方案 9.1/9.2/9.3）：只读，不发起任何扣款或退款 */
  getRefundPreview: (id: string | number) => ok<RefundPreview>(api.get('/payment/orders/' + id + '/refund-preview')),
  /** 余额分账账本：本金/赠送构成 + 近期分录 */
  getLedger: (limit = 50) =>
    ok<{ breakdown: LedgerBalanceBreakdown; entries: RefundLedgerEntry[] }>(api.get('/payment/ledger', { params: { limit } })),
  resolvePublicOrder: (body: any) => ok<any>(api.post('/payment/public/orders/resolve', body)),
  /**
   * 共建分成（受益人自助查看）：只读，返回自己在分账账本里的待结算/已锁定/
   * 已结算/已冲回金额与近期明细。分账是独立账本，不进入可消费余额。
   */
  getMyRevenueSplit: (limit = 50) =>
    ok<{ summary: RevenueSplitBeneficiarySummary; entries: RevenueSplitEntry[] }>(
      api.get('/payment/revenue-split', { params: { limit } }),
    ),
}

export const paymentAdminApi = {
  getDashboard: (days = 30) => ok(api.get('/admin/payment/dashboard', { params: { days } })),
  getConfig: () => ok(api.get('/admin/payment/config')),
  updateConfig: (body: PaymentConfigUpdateRequest) => ok<PaymentConfig>(api.put('/admin/payment/config', body)),
  listOrders: (params = {}) => ok(api.get('/admin/payment/orders', { params })),
  getOrderDetail: (id: string | number) =>
    ok<{ order: AdminPaymentOrder; auditLogs: AdminOrderAuditLog[] }>(api.get('/admin/payment/orders/' + id)),
  cancelOrder: (id: string | number) => ok<PaymentOrder>(api.post('/admin/payment/orders/' + id + '/cancel', {})),
  retryFulfillment: (id: string | number) => ok<AdminPaymentOrder>(api.post('/admin/payment/orders/' + id + '/retry', {})),
  /**
   * 模拟支付（仅管理员）：不调用任何上游渠道，直接把 PENDING 订单标记为已支付并履约。
   * 用于在真实商户凭据到位前跑通「下单 → 支付 → 履约 → 余额/订阅到账」全链路，
   * 或为场外已确认收款的订单补记账。后端复用真实回调的渠道/金额/幂等校验。
   */
  simulatePaid: (id: string | number) =>
    ok<AdminPaymentSimulationResult>(api.post('/admin/payment/orders/' + id + '/simulate-paid', {})),
  refund: (id: string | number, body: any) => ok<AdminPaymentOrder>(api.post('/admin/payment/orders/' + id + '/refund', body)),
  /** 管理员退款预览；force=true 预览强制退款（平台故障/重复扣款）路径 */
  /**
   * 管理员退款预览。
   * offline=true 按线下退款口径预览：渠道没有退款 API 或客户已在渠道外收到退款时，
   * 渠道退款开关关闭也不再判定为不可退（该口径不调用任何上游接口）。
   */
  getRefundPreview: (id: string | number, force = false, offline = false) =>
    ok<RefundPreview>(
      api.get('/admin/payment/orders/' + id + '/refund-preview', {
        params: { ...(force ? { force: 'true' } : {}), ...(offline ? { offline: 'true' } : {}) },
      }),
    ),
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

// ---------- 共建者分成（多受益人固定比例分账） ----------
//
// 设计边界（务必保留这段注释，避免后来者误加「自动打款」逻辑）：
//   客户支付 → 按比例计提分录（pending）→ 生成结算单（draft）
//   → 管理员线下手工打款 → 回填流水号并标记已打款（paid）。
// 系统只记账，不做出金；自动把钱转给第三方属于无牌照资金清分。

export type RevenueSplitBaseMode = 'gross' | 'gross_after_fee'

export type RevenueSplitEntryStatus = 'pending' | 'settled' | 'reversed'

export type RevenueSplitSettlementStatus = 'draft' | 'paid' | 'cancelled'

export interface RevenueSplitConfig {
  enabled: boolean
  base_mode: RevenueSplitBaseMode
  channel_fee_percent: number
}

export interface RevenueSplitConfigUpdateRequest {
  enabled?: boolean
  base_mode?: RevenueSplitBaseMode
  channel_fee_percent?: number
}

export interface RevenueSplitRule {
  id: number
  beneficiary_user_id: number
  beneficiary_name: string
  beneficiary_email: string
  beneficiary_status: string
  ratio_percent: number
  enabled: boolean
  note: string
  sort_order: number
}

/** 规则写入入参（全量替换） */
export interface RevenueSplitRuleInput {
  beneficiary_user_id: number
  beneficiary_name?: string
  ratio_percent: number
  enabled?: boolean
  note?: string
  sort_order?: number
}

export interface RevenueSplitEntry {
  id: number
  order_id: number
  beneficiary_user_id: number
  beneficiary_name: string
  order_amount: number
  pay_amount: number
  channel_fee_percent: number
  channel_fee_amount: number
  base_mode: RevenueSplitBaseMode
  base_amount: number
  ratio_percent: number
  split_amount: number
  currency: string
  status: RevenueSplitEntryStatus
  settlement_id: number | null
  reversed_at: string | null
  reverse_reason: string
  memo: string
  created_at: string
  order_out_trade_no: string
  order_user_id: number
  order_type: string
  payment_type: string
}

export interface RevenueSplitBeneficiarySummary {
  beneficiary_user_id: number
  beneficiary_name: string
  beneficiary_email: string
  ratio_percent: number
  enabled: boolean
  /** 已计提、尚未被结算单锁定 */
  pending_amount: number
  /** 已被 draft 结算单锁定，等待线下打款 */
  locked_amount: number
  settled_amount: number
  reversed_amount: number
  entry_count: number
  pending_count: number
  last_entry_at: string | null
}

export interface RevenueSplitSettlement {
  id: number
  beneficiary_user_id: number
  beneficiary_name: string
  period_start: string | null
  period_end: string | null
  entry_count: number
  amount: number
  currency: string
  status: RevenueSplitSettlementStatus
  method: string
  reference: string
  note: string
  paid_at: string | null
  paid_by: number | null
  created_by: number | null
  created_at: string
  updated_at: string
}

export interface RevenueSplitPreviewItem {
  beneficiary_user_id: number
  beneficiary_name: string
  ratio_percent: number
  split_amount: number
}

export interface RevenueSplitPreview {
  enabled: boolean
  base_mode: RevenueSplitBaseMode
  channel_fee_percent: number
  gross_amount: number
  channel_fee_amount: number
  /** 真正的分账基数（gross_after_fee 口径下 = 毛额 - 通道费） */
  base_amount: number
  allocated_amount: number
  /** 分完后仍留在平台的部分；上游 API 成本要从这里出 */
  platform_remainder: number
  items: RevenueSplitPreviewItem[]
}

export interface RevenueSplitEntryQuery {
  page?: number
  page_size?: number
  beneficiary_user_id?: number
  status?: RevenueSplitEntryStatus | ''
  order_id?: number
  start?: string
  end?: string
  keyword?: string
}

export interface RevenueSplitSettlementQuery {
  page?: number
  page_size?: number
  beneficiary_user_id?: number
  status?: RevenueSplitSettlementStatus | ''
}

export interface PaginatedRevenueSplit<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

export const revenueSplitApi = {
  getConfig: () =>
    ok<{ config: RevenueSplitConfig; rules: RevenueSplitRule[] }>(api.get('/admin/payment/revenue-split/config')),
  updateConfig: (body: RevenueSplitConfigUpdateRequest) =>
    ok<{ config: RevenueSplitConfig }>(api.put('/admin/payment/revenue-split/config', body)),
  listRules: () => ok<{ rules: RevenueSplitRule[] }>(api.get('/admin/payment/revenue-split/rules')),
  /** 全量替换规则：传空数组会清空，开启开关时后端要求至少一条启用规则 */
  replaceRules: (rules: RevenueSplitRuleInput[]) =>
    ok<{ rules: RevenueSplitRule[] }>(api.put('/admin/payment/revenue-split/rules', { rules })),
  /** 试算：不落库，用来在保存前看清「客户付 100 谁拿多少」 */
  preview: (amount: number) => ok<RevenueSplitPreview>(api.post('/admin/payment/revenue-split/preview', { amount })),
  listEntries: (params: RevenueSplitEntryQuery = {}) =>
    ok<PaginatedRevenueSplit<RevenueSplitEntry>>(api.get('/admin/payment/revenue-split/entries', { params })),
  summary: () =>
    ok<{ beneficiaries: RevenueSplitBeneficiarySummary[] }>(api.get('/admin/payment/revenue-split/summary')),
  listSettlements: (params: RevenueSplitSettlementQuery = {}) =>
    ok<PaginatedRevenueSplit<RevenueSplitSettlement>>(api.get('/admin/payment/revenue-split/settlements', { params })),
  getSettlement: (id: string | number) =>
    ok<{ settlement: RevenueSplitSettlement; entries: RevenueSplitEntry[] }>(
      api.get('/admin/payment/revenue-split/settlements/' + id),
    ),
  /** 生成结算单：把该受益人当前「可结算」的分录汇总并锁定 */
  createSettlement: (body: {
    beneficiary_user_id: number
    period_start?: string
    period_end?: string
    method?: string
    note?: string
  }) => ok<RevenueSplitSettlement>(api.post('/admin/payment/revenue-split/settlements', body)),
  /** 标记已线下打款：reference 必填，作为打款凭证号 */
  markSettlementPaid: (id: string | number, body: { method?: string; reference: string; note?: string }) =>
    ok<RevenueSplitSettlement>(api.post('/admin/payment/revenue-split/settlements/' + id + '/pay', body)),
  cancelSettlement: (id: string | number) =>
    ok<RevenueSplitSettlement>(api.post('/admin/payment/revenue-split/settlements/' + id + '/cancel', {})),
  /** 为历史已完成订单补计提（幂等，可重复点） */
  accrueOrder: (orderId: string | number) =>
    ok<{ entries: number }>(api.post('/admin/payment/revenue-split/orders/' + orderId + '/accrue', {})),
}
