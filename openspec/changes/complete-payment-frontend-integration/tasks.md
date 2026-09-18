# Tasks

实施阶段按 P0 → P1 → P2 推进；每个阶段完成后停下来等用户验收再进入下一阶段。

## Phase 0：基础与 API service 层（必须最先做）

~~- [ ] 0.1 在 `backend/internal/handler/payment_handler.go` 新增 `CreateOrderDetailed` 处理器，透传完整 `CreatePaymentResponse`（含 `result_type / pay_url / qr_code / client_secret / intent_id / currency / country_code / payment_env / jsapi / oauth`）。
~~- [ ] 0.2 在 `backend/internal/server/routes/payment.go` 注册 `POST /api/v1/payment/orders/detailed`（已认证 + 用户限流，沿用 `orders` 组的中间件）。
~~- [ ] 0.3 在 `backend/internal/service/payment_service.go` 实现 `CreateOrderDetailed`，复用 `CreateOrder` 的输入校验 + 限额检查 + provider 选取，**额外**把 `CreatePaymentResponse` 完整返回（不加 ent 字段，只在 handler 层组装新 DTO）。
~~- [ ] 0.4 写 `CreateOrderDetailed` 单元测试：mock provider，至少覆盖 4 种 ResultType 分支（qr_code 非空 / pay_url 非空 / client_secret 非空 / jsapi_ready / oauth_required）。
- [ ] 0.5 在前端 `frontend/src/smirel/api/payment.ts` 创建 API service 单文件：用户端 10 个方法 + admin 端 14 个方法，按 `payment.user.*` / `payment.admin.*` 命名空间分块。
- [ ] 0.6 在前端 `frontend/src/smirel/api/paymentProviderSchemas.ts` 硬编码 5 种 provider 的字段 schema（来自 design.md）。
- [ ] 0.7 在前端 `frontend/src/smirel/composables/usePaymentCheckout.ts` 和 `usePaymentOrderPolling.ts` 实现两个 composable（无 UI 依赖，便于单测）。
- [ ] 0.8 把 i18n 键骨架 `payment.*` 写到 `core/i18n.ts`，中英双语占位（zh-CN + en-US），后续阶段逐步填充具体文案。
- [ ] 0.9 Phase 0 验证：跑 `cd backend && go test -tags=unit ./internal/service -run Payment` + `cd backend && golangci-lint run ./internal/service/... ./internal/handler/...` 通过；前端 `pnpm typecheck` + `pnpm lint` 通过。

## Phase 1：用户端 P0（用户能充值 + 看订单）

- [ ] 1.1 改造 `frontend/src/smirel/components/UserBillingPage.vue`：
  - 移除"接入中"占位文案，提交按钮可用
  - 接入 `usePaymentCheckout()` 拉真实配置 + 限 / 套餐
  - 提交时调 `createOrderDetailed`，按 `result_type` 跳转
  - 摘要区显示真实手续费 + 到账（fee_rate 来自后端响应）
  - 增加支付方式选择（`PaymentMethodsPicker`），按 visible method 路由过滤可选项
  - 移动端：UA 检测 → 调对应 mobile 分支（AlipayMobilePrecreate 等）
- [ ] 1.2 新增 `frontend/src/smirel/components/payment/PaymentMethodsPicker.vue`：根据 `configService.GetAvailableMethodLimits()` 暴露的 alipay / wxpay 渲染两个按钮，可选扩展 Stripe / Airwallex（取决于 enabledTypes 配置）。
- [ ] 1.3 新增 `frontend/src/smirel/components/payment/PaymentOrderSummary.vue`：金额 / 手续费 / 到账。
- [ ] 1.4 新增 `frontend/src/smirel/components/payment/PaymentQrCodeCard.vue`：用 `qrcode` 包渲染 QR；显示 `expires_at` 倒计时；状态徽章。
- [ ] 1.5 改造 `frontend/src/smirel/pages/PublicPage.vue`：在 `kind === 'payment'` 内按子分支渲染：
  - `qrcode` 子分支（path 含 `/payment/qrcode`）
  - `result` 子分支（path 含 `/payment/result`）
  - `stripe` 子分支（path 含 `/payment/stripe`）
  - `stripe-popup` 子分支（path 含 `/payment/stripe-popup`）
  - `airwallex` 子分支（path 含 `/payment/airwallex`）
- [ ] 1.6 新增 `frontend/src/smirel/components/payment/PaymentStripeForm.vue` 与 `PaymentStripePopupForm.vue`：动态 import `@stripe/stripe-js`，创建 Stripe Payment Element。
- [ ] 1.7 新增 `frontend/src/smirel/components/payment/PaymentAirwallexForm.vue`：动态加载 `static.airwallex.com/components` SDK，渲染 Drop-in。
- [ ] 1.8 重写 `frontend/src/smirel/components/UserOrdersPage.vue`：拉 `listMyOrders()`，分页列表 + 状态徽章 + 操作按钮（取消、申请退款、查看详情、跳结果页）。
- [ ] 1.9 Phase 1 验证：`pnpm typecheck` + `pnpm lint` + `pnpm build`；`go test -tags=unit ./...`；本地手测（ZPay demo / Stripe test card 4242 4242 4242 4242）。

## Phase 2：admin 端 P1（Provider / Plan / Config CRUD）

- [ ] 2.1 新增 `frontend/src/smirel/pages/AdminPaymentProvidersPage.vue`：
  - 列表：`listProviders()`，表格展示 provider_key / name / enabled / supported_types / 限额
  - 创建 / 编辑：抽屉表单，按 `providerSchemas[provider_key]` 渲染字段；敏感字段 `type=password` placeholder "留空不修改"
  - 删除：二次确认
  - 创建 / 编辑后调 `refreshProviders`（后端会自动调，前端无需手动）
- [ ] 2.2 新增 `frontend/src/smirel/pages/AdminPaymentPlansPage.vue`：
  - 列表：`listPlans()`，表格展示 name / group / price / validity
  - 创建 / 编辑：抽屉表单（id / group_id / name / description / price / original_price / currency / validity_days / validity_unit / features / product_name / for_sale / sort_order）
- [ ] 2.3 新增 `frontend/src/smirel/pages/AdminPaymentConfigPage.vue`：
  - 表单分 6 个分组：基础（enabled / min_amount / max_amount / daily_limit）、订单（timeout / max_pending / cancel_rate_limit）、商品（prefix / suffix / help_*）、渠道（alipay_force_qrcode / mobile_precreate_deep_link / enabled_types / load_balance_strategy / visible_method_*_source / *_enabled）、货币（recharge_multiplier / subscription_usd_to_cny_rate / fee_rate）、实验性（balance_disabled）
  - 保存：调 `updateConfig`
- [ ] 2.4 修改 `frontend/src/smirel/router/index.ts`：注册 `/admin/payment/providers` `plans` `config` 三个路由；workspaceRoutes 复用现有映射模式
- [ ] 2.5 修改 `frontend/src/smirel/core/navigation.ts`：在 `adminNavigation` 数组插入两个 NavItem：
  - `{ path: '/admin/payment/providers', feature: 'admin-payment-providers', short: 'PV' }`
  - `{ path: '/admin/payment/config', feature: 'admin-payment-config', short: 'PC' }`
- [ ] 2.6 Phase 2 验证：`pnpm typecheck` + `pnpm lint` + `pnpm build`；admin 三页面在本地 dev 跑通创建 / 编辑 / 删除流程（用 ZPay demo + Stripe test）。

## Phase 3：admin 端 P2（Dashboard / Orders 接真实数据）

- [ ] 3.1 改造 `frontend/src/smirel/pages/AdminPaymentDashboardPage.vue`：
  - 删除所有 mock 数组
  - summary：`getDashboard(days=30)` 返回 `today_revenue / success_count / success_rate / pending_settlement`
  - trend：调真实数据（如果后端有 7 日序列）；否则保留图表但显示"数据接入中"
  - 渠道分布：按 `payment_type` 聚合
  - 状态分布：按 `status` 聚合
  - 待处理事项：Fulfilled 失败订单数、退款中数
  - 交易流水：跳到 `AdminOrdersPage`
- [ ] 3.2 改造 `frontend/src/smirel/pages/AdminOrdersPage.vue`：
  - 删除 mock 数据
  - 列表：`listOrders({ page, pageSize, status, order_type, payment_type, keyword, user_id })`
  - 筛选条：状态 / 类型 / 支付方式 / 关键字 / 用户 ID
  - 行操作：详情（跳详情抽屉）/ 取消 / 重试履约 / 退款 / 查询退款
  - 详情抽屉：调用 `getOrderDetail(id)` 拿到 `order + auditLogs`
- [ ] 3.3 修改 `AdminOrdersPage.vue` 的"详情"逻辑：通过 `adminPaymentHandler.GetOrderDetail` 返回 `{ order, auditLogs }` 渲染；显示 provider_key / provider_instance_id / amount / pay_amount / fee_rate / qr_code_img / pay_url / refund_amount / 全状态时间轴。
- [ ] 3.4 Phase 3 验证：本地手测三种 provider 各创建一个测试订单，确认 admin 能看到完整链路。

## Phase 4：清理与文档

- [ ] 4.1 跑 `golangci-lint` + `pnpm lint` 全部 0 issue。
- [ ] 4.2 跑 `cd backend && go test -tags=unit ./...` + `cd backend && go test -tags=integration ./...`（若集成测试可用）。
- [ ] 4.3 跑 `cd frontend && pnpm test` + `pnpm build`。
- [ ] 4.4 写 `docs/PAYMENT_FRONTEND_TESTING.md`：5 种 provider 的端到端手测步骤（demo / sandbox 凭据、callback URL 配置、Stripe test card、Airwallex test mode）。
- [ ] 4.5 更新 `docs/PAYMENT.md`：补充"前端已接入商业版控制台"一节，列出 5 个路由说明。
- [ ] 4.6 更新 `README.md` 与 `README_CN.md`：Feature list 取消 "即将开放" 标记，加链接到 `PAYMENT_FRONTEND_TESTING.md`。
- [ ] 4.7 在 `openspec/changes/complete-payment-frontend-integration/implementation-evidence.md` 写完成报告：每阶段产物、验证输出、截图位置。

## 依赖与顺序

```
Phase 0 ────→ Phase 1 ────→ Phase 2 ────→ Phase 3 ────→ Phase 4
   │              │              │              │
   ▼              ▼              ▼              ▼
 API service    UserBilling    Admin CRUD    Dashboard
 + CreateOrderDetailed + 4 public + Provider/   + Orders
 + 5 schemas       pages         Plan/Config    真实数据
```

- Phase 0 是后续所有阶段的依赖，必须最先完成并验证。
- Phase 1 与 Phase 2 在前端代码层面相互独立（不同文件），理论上可并行；建议先做 Phase 1 让用户能充值，再做 Phase 2 让管理员能配置。
- Phase 3 依赖 Phase 2 的导航 + 路由结构，但只动 admin dashboard / orders 两个页面，可以等 Phase 2 完成后独立做。
