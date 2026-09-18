## ADDED Requirements

### Requirement: 管理员必须能在 /admin/payment/providers 管理支付通道实例

系统 SHALL 提供 `AdminPaymentProvidersPage`（路由 `/admin/payment/providers`），支持查看、创建、编辑、删除支付 provider 实例。支持的 provider_key MUST 包含 `alipay` / `wxpay` / `easypay` / `stripe` / `airwallex`。

#### Scenario: 列表
- **WHEN** 管理员进入 `/admin/payment/providers`
- **THEN** 前端 MUST 调 `GET /admin/payment/providers` 并以表格展示 provider_key / 名称 / 启用 / 支持方式 / 限额

#### Scenario: 创建 ZPay(EasyPay) 实例
- **WHEN** 管理员点击"新增"并选择 provider_key=easypay
- **THEN** 前端 MUST 渲染 apiUrl / pid / key 三个必填字段
- **THEN** 前端 MUST 把表单 POST 到 `POST /admin/payment/providers`

#### Scenario: 编辑实例（保留 secret）
- **WHEN** 管理员编辑现有实例且 secret 字段留空
- **THEN** 前端 MUST 把该字段作为空字符串发送，后端 MUST NOT 修改现有密文

#### Scenario: 删除实例
- **WHEN** 管理员点击删除并二次确认
- **THEN** 前端 MUST 调 `DELETE /admin/payment/providers/:id` 并刷新列表

### Requirement: 管理员必须能在 /admin/orders/plans 管理订阅套餐

系统 SHALL 提供 `AdminPaymentPlansPage`（路由 `/admin/orders/plans`），支持查看、创建、编辑、删除订阅套餐。

#### Scenario: 列表
- **WHEN** 管理员进入 `/admin/orders/plans`
- **THEN** 前端 MUST 调 `GET /admin/payment/plans` 并展示套餐表格（id / 分组 / 名称 / 价格 / 有效期 / 在售）

#### Scenario: 创建套餐
- **WHEN** 管理员点击"新增"并填写 name / group_id / price / validity_days / features
- **THEN** 前端 MUST 调 `POST /admin/payment/plans` 并把新行插入列表

### Requirement: 管理员必须能在 /admin/payment/config 编辑支付系统设置

系统 SHALL 提供 `AdminPaymentConfigPage`（路由 `/admin/payment/config`），分 6 个区块展示并编辑支付配置。

#### Scenario: 加载配置
- **WHEN** 管理员进入 `/admin/payment/config`
- **THEN** 前端 MUST 调 `GET /admin/payment/config` 并按区块渲染表单

#### Scenario: 保存配置
- **WHEN** 管理员修改任意字段后点击"保存"
- **THEN** 前端 MUST 把全部字段 PUT 到 `PUT /admin/payment/config`
- **THEN** 前端 MUST 在成功后显示"配置已更新"提示

### Requirement: 管理员必须能在 /admin/orders/dashboard 查看真实支付数据

系统 SHALL 把 `AdminPaymentDashboardPage` 从 mock 数据切换为真实 API 调用。仪表盘 MUST 至少展示：今日实收、成功笔数、成功率、近 7 日实收趋势、支付状态分布、渠道分布、待处理事项、近期交易。

#### Scenario: 加载真实数据
- **WHEN** 管理员进入 `/admin/orders/dashboard`
- **THEN** 前端 MUST 调 `GET /admin/payment/dashboard?days=30` 并替换所有原有 mock 字段
- **THEN** 前端 MUST NOT 残留任何 `示例数据` 徽章或 mock 数组

#### Scenario: 切换时间范围
- **WHEN** 管理员选择 7d / 30d / 90d
- **THEN** 前端 MUST 用对应 days 参数重新拉 dashboard 数据

### Requirement: 管理员必须能在 /admin/orders 管理全部订单

系统 SHALL 把 `AdminOrdersPage` 从 mock 切换为真实 API，提供订单筛选、详情查看、取消、重试履约、发起退款、查询退款操作。

#### Scenario: 列表 + 筛选
- **WHEN** 管理员进入 `/admin/orders`
- **THEN** 前端 MUST 调 `GET /admin/payment/orders` 并展示分页表格
- **WHEN** 管理员选择 status=PAID 或输入 keyword
- **THEN** 前端 MUST 用筛选条件重新请求

#### Scenario: 订单详情
- **WHEN** 管理员点击订单行
- **THEN** 前端 MUST 调 `GET /admin/payment/orders/:id` 取得完整订单信息 + 审计日志
- **THEN** 前端 MUST 显示订单 + 状态时间轴 + 审计日志

#### Scenario: 取消 / 重试 / 退款
- **WHEN** 管理员点击"取消" / "重试履约" / "退款" / "查询退款"
- **THEN** 前端 MUST 调对应 `POST /admin/payment/orders/:id/{cancel|retry|refund|refund/query}` 并刷新当前详情

### Requirement: 管理员侧栏必须包含新增支付入口

`frontend/src/smirel/core/navigation.ts` 的 `adminNavigation` 数组 MUST 包含 `/admin/payment/providers` 与 `/admin/payment/config` 两个 NavItem，使它们在管理后台侧栏可见并可访问。

#### Scenario: 侧栏渲染
- **WHEN** 管理员进入任意后台页
- **THEN** 侧栏 MUST 显示"支付通道"（`/admin/payment/providers`）与"支付设置"（`/admin/payment/config`）两个导航项
