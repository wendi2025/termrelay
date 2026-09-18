## ADDED Requirements

### Requirement: 系统必须暴露完整订单创建响应供前端渲染分流

系统 SHALL 在 `POST /api/v1/payment/orders` 之外提供 `POST /api/v1/payment/orders/detailed` 端点，返回完整 `CreatePaymentResponse` 字段，使前端能够根据 `result_type` 与渠道能力把用户重定向到二维码页、H5 跳转页、Stripe 收银台或 Airwallex Drop-in。响应 MUST 至少包含 `result_type` / `trade_no` / `pay_url` / `qr_code` / `client_secret` / `intent_id` / `currency` / `country_code` / `payment_env`，以及可选的 `jsapi` 与 `oauth` 子对象。

#### Scenario: 用户选择 EasyPay(ZPay) Alipay 创建订单
- **WHEN** 用户在 `UserBillingPage` 选择 Alipay 提交 100 元充值
- **THEN** 前端调 `POST /payment/orders/detailed`，系统 MUST 返回 `result_type=order_created` 且 `qr_code` 非空
- **THEN** 前端 MUST 跳到 `/payment/qrcode` 并展示二维码

#### Scenario: 用户在手机端选择 Alipay WAP
- **WHEN** User-Agent 检测为 mobile 且 `alipay_mobile_precreate_deep_link` 未开启
- **THEN** 系统 MUST 返回 `result_type=order_created` 且 `pay_url` 非空
- **THEN** 前端 MUST 跳到 `/payment/redirect` 或在 iframe 内打开 `pay_url`

#### Scenario: 用户选择 Stripe 卡支付
- **WHEN** User-Agent 检测为桌面端且 provider_key=`stripe`
- **THEN** 系统 MUST 返回 `result_type=order_created` 且 `client_secret` 非空
- **THEN** 前端 MUST 跳到 `/payment/stripe` 渲染 Stripe Payment Element

#### Scenario: 用户在微信内选择 WxPay
- **WHEN** User-Agent 包含 `micromessenger` 且 provider_key=`wxpay` 且已有 openid
- **THEN** 系统 MUST 返回 `result_type=jsapi_ready` 且 `jsapi.appId` 等字段非空
- **THEN** 前端 MUST 在当前页内直接调用 `WeixinJSBridge.invoke('getBrandWCPayRequest', ...)`

#### Scenario: 用户在微信外首次选择 WxPay
- **WHEN** User-Agent 不含 `micromessenger` 且 provider_key=`wxpay` 且未绑定 openid
- **THEN** 系统 MUST 返回 `result_type=oauth_required` 且 `oauth.authorize_url` 非空
- **THEN** 前端 MUST 渲染"前往微信授权"按钮

### Requirement: 用户必须能在 UserBillingPage 完成余额充值

系统 SHALL 提供 `UserBillingPage`（绑定到路由 `/subscriptions`），允许已登录用户选择金额（预设或自定义）、选择支付方式，并完成订单创建。提交按钮 MUST 在 payment_enabled=true 且金额在合法范围内时可用。

#### Scenario: 默认金额与自定义金额
- **WHEN** 用户进入 `/subscriptions`
- **THEN** 系统 MUST 显示当前余额、最近交易入口、金额预设（10 / 20 / 50 / 100 / 200 / 500）、自定义金额输入框
- **WHEN** 用户点击自定义金额并输入合法正数
- **THEN** 系统 MUST 切换到自定义模式并把 `effective_amount` 更新为该值

#### Scenario: 金额超出范围
- **WHEN** 用户输入金额 < `min_amount` 或 > `max_amount`
- **THEN** 提交按钮 MUST 被禁用且前端 MUST 显示"金额不在允许范围内"

#### Scenario: 提交订单
- **WHEN** 用户点击"立即支付"按钮
- **THEN** 前端 MUST 调 `POST /payment/orders/detailed` 并根据返回 `result_type` 跳到对应支付页或渲染 JSAPI

### Requirement: 用户必须能在公开支付页完成支付并查看结果

系统 SHALL 通过 `PublicPage` 的支付子分支渲染 5 种公开支付场景：QR 码等待、Stripe Elements、Stripe Popup、Airwallex Drop-in、订单结果。所有公开页 MUST 支持 `resume_token` 持久化，以便用户刷新或稍后回来查看。

#### Scenario: 二维码页
- **WHEN** 用户进入 `/payment/qrcode?out_trade_no=&resume_token=`
- **THEN** 前端 MUST 渲染 QR 码并显示订单金额、过期倒计时
- **THEN** 前端 MUST 每 2-3 秒调 `POST /payment/orders/verify` 一次订单状态
- **WHEN** 状态变为 PAID / COMPLETED
- **THEN** 前端 MUST 自动跳到 `/payment/result?out_trade_no=&resume_token=`

#### Scenario: Stripe Elements 页
- **WHEN** 用户进入 `/payment/stripe` 且 `client_secret` 存在
- **THEN** 前端 MUST 动态加载 `@stripe/stripe-js` 并创建 `StripePaymentElement`
- **WHEN** 用户在 Stripe Element 完成卡输入并提交
- **THEN** 前端 MUST 调 `stripe.confirmPayment({ confirmParams: { return_url } })` 并在 Stripe 跳回时调 `verify`

#### Scenario: Airwallex Drop-in 页
- **WHEN** 用户进入 `/payment/airwallex` 且 `intent_id` 与 `client_secret` 存在
- **THEN** 前端 MUST 动态加载 Airwallex Web SDK 并渲染 Drop-in
- **WHEN** 用户完成支付回调成功
- **THEN** 前端 MUST 调 `verify` 完成最终确认

#### Scenario: 结果页
- **WHEN** 用户进入 `/payment/result?out_trade_no=&resume_token=`
- **THEN** 前端 MUST 调 `POST /payment/public/orders/resolve` 解析订单并按状态展示成功 / 失败 / 等待 / 取消
- **WHEN** 状态为 COMPLETED
- **THEN** 前端 MUST 显示"支付成功"并提供"返回工作台"按钮

### Requirement: 用户必须能在"我的订单"页查看全部订单并执行操作

系统 SHALL 提供 `UserOrdersPage`（路由 `/orders`），拉取 `GET /payment/orders/my` 的分页列表，并提供以下操作：查看详情、取消未支付订单、申请退款。

#### Scenario: 默认分页
- **WHEN** 用户进入 `/orders`
- **THEN** 前端 MUST 拉取 `GET /payment/orders/my?page=1&page_size=20` 并按 `created_at` 倒序展示

#### Scenario: 取消订单
- **WHEN** 用户点击未支付订单的"取消"按钮
- **THEN** 前端 MUST 调 `POST /payment/orders/:id/cancel` 并刷新当前列表

#### Scenario: 申请退款
- **WHEN** 用户点击已完成订单的"申请退款"按钮并填写退款原因
- **THEN** 前端 MUST 调 `POST /payment/orders/:id/refund-request` 并展示退款进度

### Requirement: 系统必须提供支付相关中英文 i18n

所有用户可见的支付文案 MUST 同时提供 zh-CN 与 en-US 版本，键结构按 `payment.{user,admin}.*` / `payment.status.*` / `payment.method.*` 命名空间组织。

#### Scenario: 切换语言
- **WHEN** 用户在右上角切换 zh-CN → en-US
- **THEN** 所有支付相关文案 MUST 立即更新为英文
- **THEN** MUST NOT 出现"中英混合"的硬编码字符串
