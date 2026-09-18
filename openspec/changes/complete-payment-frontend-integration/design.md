# Design

## 架构总览

```
[前端 Vue 3 + smirel 商业版]
        │
        │  axios via core/api.ts（自动信封解包）
        ▼
[新文件] src/smirel/api/payment.ts  ──── 所有支付 HTTP 调用的单一入口
        │
        ├── 用户端子模块：getCheckoutInfo / listPlans / getLimits / createOrder /
        │   verifyOrder / getOrder / cancelOrder / requestRefund / listMyOrders /
        │   resolveByResumeToken / verifyPublicOrder
        │
        └── admin 端子模块：listProviders / createProvider / updateProvider /
            deleteProvider / listPlans / createPlan / updatePlan / deletePlan /
            getConfig / updateConfig / getDashboard / listOrders / getOrder /
            cancelOrder / retryFulfillment / processRefund / queryRefund
```

API service 文件**不做**状态管理；调用者（页面组件 / composable）持有状态。
为减少样板代码，提供两个轻量 composable：

- `usePaymentCheckout()`：拉取 checkout-info（同时含 config / limits / plans）、暴露 `reload` 方法。
- `usePaymentOrderPolling(orderId)`：包装 `POST /payment/orders/verify` 轮询，按 order 状态切换（`PENDING` 继续轮询、`PAID`/`COMPLETED` 跳成功页、`EXPIRED`/`CANCELLED`/`FAILED` 跳失败页）。

## 路由与页面树

`frontend/src/smirel/router/index.ts` 已存在的支付相关路由：

| 路由 | 组件（现状） | 目标组件 |
|---|---|---|
| `/subscriptions` | `UserBillingRoutePage` → `UserBillingPage` | 不改路由，**改造 `UserBillingPage` 内部** |
| `/orders` | `UserOrdersPage` | **重写为真实数据** |
| `/payment/result` | `PublicPage` 占位 | `PublicPage` 内部按 `publicKind` 分支新增 `payment-result` 渲染 |
| `/payment/qrcode` | `PublicPage` 占位 | 同上新增 `payment-qrcode` 渲染 |
| `/payment/stripe` | `PublicPage` 占位 | 同上新增 `payment-stripe` 渲染（Stripe Elements） |
| `/payment/stripe-popup` | `PublicPage` 占位 | 同上新增 `payment-stripe-popup` 渲染（Stripe popup Elements） |
| `/payment/airwallex` | `PublicPage` 占位 | 同上新增 `payment-airwallex` 渲染（Airwallex Drop-in） |
| `/admin/orders/dashboard` | `AdminPaymentDashboardPage` | **改造**：mock → 真实 API |
| `/admin/orders` | `AdminOrdersPage` | **改造**：mock → 真实 API |
| `/admin/orders/plans` | `AdminPaymentPlansPage`（占位） | **新增实现** |
| `/admin/payment/providers` | （无） | **新增实现** |
| `/admin/payment/config` | （无） | **新增实现** |

`navigation.ts` 已经在 `adminNavigation` 里预留了 `admin-plans` feature，需要追加 `admin-payment-providers` 和 `admin-payment-config` 两个 NavItem。

## 组件拆分

新增组件（按目录归类）：

```
frontend/src/smirel/
├── api/
│   └── payment.ts                          ← 唯一 HTTP 入口
├── composables/
│   ├── usePaymentCheckout.ts               ← checkout-info 加载器
│   └── usePaymentOrderPolling.ts           ← 订单轮询
├── components/payment/
│   ├── PaymentQrCodeCard.vue               ← 二维码 + 倒计时 + 状态徽章
│   ├── PaymentAlipayForm.vue               ← 支付宝手机跳转/扫码选择
│   ├── PaymentWxpayForm.vue                ← 微信 H5 / JSAPI / Native 分发
│   ├── PaymentStripeForm.vue               ← Stripe Payment Element
│   ├── PaymentStripePopupForm.vue          ← Stripe popup（同上但独立路由）
│   ├── PaymentAirwallexForm.vue            ← Airwallex Drop-in
│   ├── PaymentOrderSummary.vue             ← 订单摘要（金额、币种、手续费、到账）
│   ├── PaymentMethodsPicker.vue            ← 支付方式选择（Alipay / WxPay / Stripe / Airwallex）
│   └── PaymentStatusBadge.vue              ← 订单状态徽章
├── pages/
│   ├── AdminPaymentProvidersPage.vue       ← 列表 + 编辑抽屉
│   ├── AdminPaymentPlansPage.vue           ← 列表 + 编辑抽屉
│   └── AdminPaymentConfigPage.vue          ← 配置表单（多区块）
```

## ResultType 分发策略

`POST /payment/orders` 的响应在 `PaymentOrderResult` 之外携带 `result_type`（来自 `CreatePaymentResponse.ResultType`）：

| `result_type` | 前端路由 |
|---|---|
| `order_created` + `qr_code` 非空 | 跳 `/payment/qrcode?out_trade_no=...&resume_token=...` |
| `order_created` + `qr_code` 空 + `pay_url` 非空（Alipay WAP / WxPay H5 / EasyPay H5） | 跳 `/payment/redirect?url=...&out_trade_no=...&resume_token=...` |
| `order_created` + `client_secret` 非空（Stripe / Airwallex） | 跳对应 SDK 渲染页：`/payment/stripe?out_trade_no=...` 或 `/payment/airwallex?out_trade_no=...` |
| `jsapi_ready`（微信内 JSAPI） | 在 `UserBillingPage` 内直接渲染 JSAPI 调用按钮（不跳路由） |
| `oauth_required`（微信外部需先 OAuth） | 渲染"前往微信授权"按钮，点击跳 `authorize_url` |

实际数据流：`PaymentService.CreateOrder` 当前没有把 `ResultType` 透传到 `PaymentOrderResult`。需要在 service 层加一个并列方法 `CreateOrderDetailed()` 或者扩展现有响应，把 `CreatePaymentResponse` 中的 `TradeNo/PayURL/QRCode/ClientSecret/IntentID/ResultType/JSAPI/OAuth` 透传给前端。前端靠这些字段决策渲染。

为了让改动最小化且不破坏现有契约，**新增**一个内部端点 `POST /payment/orders/detailed`（在已有 `handler/payment_handler.go` 新增一个方法 `CreateOrderDetailed`），专门返回完整 `CreatePaymentResponse`。其他端点保持向后兼容。

## Stripe / Airwallex 集成

- Stripe：在 `PaymentStripeForm.vue` 加载 `@stripe/stripe-js`，用后端返回的 `client_secret` 创建 `StripePaymentElement`，挂到容器，submit 调用 `stripe.confirmPayment({ confirmParams: { return_url } })`。成功后 Stripe 跳回 `return_url`，前端从 URL 解析 `payment_intent` + `redirect_status` 再调 `POST /payment/orders/verify` 完成最终确认。
- Airwallex：在 `PaymentAirwallexForm.vue` 加载 Airwallex Web SDK（动态插入 `https://static.airwallex.com/components-0.0.0.min.js`），初始化 `Airwallex.createElement({ intent_id, client_secret, country_code, currency })`，挂到容器，submit 调用 `element.confirm()`。回调成功则调 `POST /payment/orders/verify`。
- 两个 SDK 都使用动态 import；只在用户进入对应支付页时才加载，不在首屏阻塞。
- 当后端 security header CSP 已放行 Airwallex 域名（`static.airwallex.com` / `checkout.airwallex.com` + demo 变体），无需额外配置。

## Provider 字段表单

5 种 provider 的字段白名单直接从 `backend/internal/service/payment_config_providers.go` 的 3 个表派生（不能凭空写）：

```ts
// 由 backend 透出的元数据；前端硬编码到 src/smirel/api/paymentProviderSchemas.ts
export const providerSchemas = {
  alipay: {
    required: ['appId', 'appPrivateKey', 'alipayPublicKey'],
    optional: ['signType', 'gateway', 'useCert', 'appCert', 'alipayCert', 'alipayRootCert'],
    sensitive: ['appPrivateKey', 'appCert', 'alipayCert', 'alipayRootCert'],
  },
  wxpay: {
    required: ['mchId', 'apiV3Key', 'appId', 'serialNo'],
    optional: ['privateKey', 'merchantCertPath', 'apiClientCert'],
    sensitive: ['apiV3Key', 'privateKey'],
  },
  easypay: {
    required: ['apiUrl', 'pid', 'key'],
    optional: ['signType', 'returnUrl'],
    sensitive: ['key'],
  },
  stripe: {
    required: ['secretKey'],
    optional: ['publishableKey', 'webhookSecret', 'apiBase'],
    sensitive: ['secretKey', 'webhookSecret'],
  },
  airwallex: {
    required: ['clientId', 'apiKey', 'webhookSecret'],
    optional: ['apiBase', 'accountId', 'currency', 'countryCode'],
    sensitive: ['apiKey', 'webhookSecret'],
  },
}
```

这些字段在 admin form 中以**只写**模式呈现（`type=password` + placeholder 留空 = 不修改）。读取时只返回 `has_<field>` 标记，前端永远不持有真实 secret。

## i18n 策略

- 在现有 `core/i18n.ts` 内追加 namespace `payment`，结构：

```ts
payment: {
  user: { /* UserBillingPage / UserOrdersPage / public pages */ },
  admin: {
    dashboard: { /* AdminPaymentDashboardPage */ },
    orders: { /* AdminOrdersPage */ },
    plans: { /* AdminPaymentPlansPage */ },
    providers: { /* AdminPaymentProvidersPage */ },
    config: { /* AdminPaymentConfigPage */ },
  },
  status: { PENDING: '...', PAID: '...', COMPLETED: '...', EXPIRED: '...', ... },
  method: { alipay: '...', wxpay: '...', stripe: '...', airwallex: '...', easypay: '...' },
}
```

- 不拆 i18n 文件，避免模块加载顺序坑；直接在 `messages['zh-CN']` 和 `messages['en-US']` 两个对象里同步添加。
- 字段命名沿用现有驼峰风格（参见 `workspace.refresh` 等键）。

## 校验与错误处理

- 表单提交使用现有 `<form>` + 简单 required 校验；保留 smirel 商业版的输入框视觉。
- API 错误通过 `core/api.ts` 的 `getErrorMessage` 统一提取；用户提示放在 inline alert（不弹 toast，与现有页面一致）。
- 订单超时（默认 30 分钟）：在创建订单时记录 `expires_at`，UI 显示倒计时；到 0 自动调一次 `verify` 后跳失败页。

## 测试策略

- 单元测试：API service 调用使用 `axios-mock-adapter` 验证路径与 envelope；composable 用 vitest + `@vue/test-utils`。
- 组件快照：`PaymentStatusBadge` `PaymentOrderSummary` `PaymentMethodsPicker` 三个纯展示组件做快照测试。
- 端到端（手测）：开发环境用 ZPay demo / Stripe test card / Airwallex sandbox 验证完整流程；文档化在 `docs/PAYMENT_FRONTEND_TESTING.md`。
- 不引入新的 CI step；沿用现有 `pnpm test` + `golangci-lint`。

## 兼容性 / 风险

- 第三方 SDK 体积：Stripe.js ~80KB gz；Airwallex ~120KB gz。通过动态 import 隔离，不影响首屏。
- CSP：Stripe + Airwallex 域名已在后端 `security_headers.go` 放行，无需改动。
- 大表单（Payment Config 30+ 字段）：用分组折叠面板，避免一屏塞太满。
- 已存在的 mock 数据 (`AdminPaymentDashboardPage` / `AdminOrdersPage`) 在接入真实 API 时必须清空，不能保留 mock 兜底（避免误导）。
