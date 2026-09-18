## Why

Smirel / Sub2API 后端的支付子系统（schema、provider、service、handler、webhook、admin API）已经完整实现，但商业版前端 (`frontend/src/smirel/`) 仍停留在"已完成商业版设计，支付接口尚未开放"的状态：

- `UserBillingPage.vue` 的提交按钮 `cursor: not-allowed`，文案明确写明"支付通道接入完成后即可在线充值"。
- `PublicPage.vue` 的 `/payment/result` `/payment/qrcode` `/payment/stripe` `/payment/airwallex` 路由全部只渲染占位文案。
- `AdminPaymentDashboardPage.vue` 与 `AdminOrdersPage.vue` 使用硬编码 mock 数组，顶部展示"示例数据"徽章。
- Admin 侧没有 `providers` `plans` `config` 三个管理页面的实现。

`navigation.ts` 已经预留 `admin-payment-dashboard` `admin-orders` `admin-plans` 等 feature 入口，但都没有接上 `AdminPaymentDashboardPage` / `AdminOrdersPage` 之外的页面。

因此用户无法充值，管理员也无法配置支付通道。需要在前端侧补齐所有支付相关的 UI 与 API 调用。

## What Changes

- 新增前端支付 API service 模块（`frontend/src/smirel/api/payment.ts`），封装用户端与 admin 端所有支付相关接口。
- 改造 `UserBillingPage.vue` 与 `UserBillingRoutePage.vue`，接通 `POST /payment/orders`、`GET /payment/checkout-info`、`GET /payment/plans`、`GET /payment/limits`、`POST /payment/orders/:id/cancel`、`POST /payment/orders/:id/refund-request`。
- 新增 4 个公开支付页组件并接入到现有 `PublicPage.vue` 占位：
  - `/payment/qrcode`：渲染二维码（EasyPay / Alipay / WxPay Native）并轮询订单状态。
  - `/payment/result`：根据 `resume_token` 解析订单并展示支付结果。
  - `/payment/stripe` 与 `/payment/stripe-popup`：通过 Stripe Elements 渲染 Payment Element 完成卡支付。
  - `/payment/airwallex`：通过 Airwallex Drop-in / Checkout Element 完成卡或本地支付。
- 新增 `UserOrdersPage.vue` 的真实实现：拉取 `GET /payment/orders/my`，支持取消、申请退款、查看明细、跳到公开结果页。
- 新增 admin 端 3 个页面：
  - `AdminPaymentProvidersPage.vue`：5 种 provider 实例的列表 / 创建 / 编辑 / 删除；表单字段直接从 `backend/internal/service/payment_config_providers.go` 的白名单派生。
  - `AdminPaymentPlansPage.vue`：套餐 CRUD（名称、描述、价格、有效期、分组、特性、商品名）。
  - `AdminPaymentConfigPage.vue`：30+ 配置字段（启用、最小/最大金额、日限额、超时、pending 上限、可见方法路由、币种、商品名前后缀、Help 链接、退款限流、强制二维码、移动端 precreate 等）。
- 把 `AdminPaymentDashboardPage.vue` 与 `AdminOrdersPage.vue` 的 mock 数据切换为真实 API：
  - Dashboard：`GET /admin/payment/dashboard?days=`。
  - Orders：`GET /admin/payment/orders`（分页 + 筛选）、`GET /admin/payment/orders/:id`、`POST /admin/payment/orders/:id/cancel|retry|refund|refund/query`。
- 调整 `navigation.ts` 与 `router/index.ts`，把新增 admin 页面挂到 `/admin/payment/providers` `plans` `config`。
- 补全 i18n 键（中英文）：支付概览、订单管理、Provider / Plan / Config 表单、公开支付页文案、错误提示。
- 在 `core/i18n.ts` 已有 zh-CN / en-US 基础键的基础上扩展，不引入新的语言。
- 不修改后端任何 Go 代码、不修改 schema、不修改现有路由。

## Capabilities

### New Capabilities

- `payment-user-frontend`：定义用户端充值、订单管理、公开支付页（qrcode / result / stripe / airwallex）与真实后端 API 的集成契约。
- `payment-admin-frontend`：定义 admin 端 Provider / Plan / Config CRUD、Payment Dashboard / Orders 接真实数据的能力。

### Modified Capabilities

无。本仓库当前没有已发布的 OpenSpec capability。

## Impact

- **新增文件**：约 12 个 Vue 组件 / SFC + 1 个 API service 文件 + i18n 键扩展。覆盖现有 mock 页面，新增 admin 3 个页面，新增 4 个公开支付分支渲染。
- **依赖的现有资源**：复用 `core/api.ts` 的 axios 客户端与信封解析、复用 i18n 框架、复用 smirel 设计系统（`smirel/styles`）、复用 `WorkspaceNavIcon` 等公共组件。
- **外部依赖**：
  - Stripe.js（`@stripe/stripe-js` + `@stripe/react-stripe-js`，用于 Stripe Elements）。
  - Airwallex Web SDK（`https://static.airwallex.com/components-0.0.0.min.js` 或 npm 包，用于 Drop-in）。
  - 二维码渲染库（`qrcode` npm 包，用于把 `qr_code` 字段渲染成可见二维码）。
  - 后端安全头中间件已经预放 Airwallex `static.airwallex.com` / `checkout.airwallex.com` 与 `static-demo` / `checkout-demo` 域名 CSP，无需后端改动。
- **兼容性**：纯前端改造，不影响现有 API 契约；老用户访问 `/subscriptions` `/orders` 等路由得到的功能升级而非破坏。
- **i18n**：必须中英双语同步上线。
- **风险**：第三方 JS SDK（Stripe / Airwallex）需要在受限网络环境加载；如客户使用受限网络，前端需提示并提供 fallback 复制支付链接。

## Non-goals

- 不修改后端 Go 代码、Ent schema、迁移、路由、provider 实现。
- 不增加新的支付通道（仅消费后端已经支持的 5 种：alipay / wxpay / easypay(ZPay) / stripe / airwallex）。
- 不实现订阅管理的高级功能（自动续费、升降级、变更订阅）；订阅计划仍为"购买型"，到期后用户重新购买。
- 不引入新的前端框架或设计系统；继续使用 smirel 商业版暗色主题 + glass 视觉。
- 不实现支付运营深度看板（如漏斗、地区分布、退款原因聚合）；本变更只把现有 mock 替换为真实数据展示。
- 不实现告警/通知系统对接；订单状态变更继续依赖现有 webhook + 邮件通知。

## Execution References

- `design.md`：前端架构、API service 拆分、组件树、ResultType 分发、Provider 字段表单、i18n 策略。
- `specs/payment-user-frontend/spec.md`：用户端所有页面与流程的需求 + Scenario。
- `specs/payment-admin-frontend/spec.md`：admin 端所有页面与流程的需求 + Scenario。
- `tasks.md`：按 P0/P1/P2 阶段拆分的实现任务与验证证据。
