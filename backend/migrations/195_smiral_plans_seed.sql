-- Migration 195: Smirel 默认产品线 + 12 个主套餐 Seed
-- 数据来源：套餐定价方案 5.2 节（更新时间 2026-09-11）
-- 事务由迁移执行器统一管理（internal/repository/migrations_runner.go）。
-- 此处不得出现 BEGIN/COMMIT：显式提交会提前结束执行器事务，
-- 使 seed 数据与 schema_migrations 记账失去原子性。

-- 5 条产品线（Group）
-- 注意：name 全局唯一，使用 Smirel 前缀避免与已有分组冲突
INSERT INTO groups (
    name, description, rate_multiplier,
    platform, subscription_type, default_validity_days,
    product_line, cost_multiplier, pay_as_you_go_price_per_usd,
    loss_coefficient, max_discount_pct,
    concurrency_limit, circuit_breaker_enabled, exclusive_quota, whitelist_only,
    is_exclusive, status, sort_order,
    created_at, updated_at
) VALUES
-- GPT Plus 线路（成本倍率 0.10，按量价 0.50）
('Smirel · GPT Plus',
 'OpenAI GPT Plus 货源线路。仅 ¥89 轻享月卡使用此货源，其他 GPT 套餐使用 Pro 货源。',
 0.10, 'openai', 'subscription', 30,
 'gpt_plus', 0.10, 0.50, 0.80, 0.15,
 0, false, false, false,
 false, 'active', 10, NOW(), NOW()),

-- GPT Pro 线路（成本倍率 0.20，按量价 0.75）
('Smirel · GPT Pro',
 'OpenAI ChatGPT Pro 货源线路。GPT 专业月卡和企业试运行档使用此货源。',
 0.20, 'openai', 'subscription', 30,
 'gpt_pro', 0.20, 0.75, 0.80, 0.15,
 0, false, false, false,
 false, 'active', 11, NOW(), NOW()),

-- ClaudeCode 无 Fable（成本倍率 0.30，按量价 0.70）
('Smirel · ClaudeCode',
 'Claude 普通线路，OpenAI 兼容 API，不含 Claude Fable 5.1。',
 0.30, 'anthropic', 'subscription', 30,
 'claude_no_fable', 0.30, 0.70, 0.80, 0.15,
 0, false, false, false,
 false, 'active', 20, NOW(), NOW()),

-- ClaudeCode 含 Fable / Max（成本倍率 1.20，按量价 2.30）
('Smirel · ClaudeCode Max',
 'Claude 高成本高权益线路，含 Claude Fable 5.1。限量开放，企业试运行档白名单销售。',
 1.20, 'anthropic', 'subscription', 30,
 'claude_fable', 1.20, 2.30, 0.80, 0.15,
 30, true, true, true,
 false, 'active', 21, NOW(), NOW()),

-- 国产模型（成本倍率 0.10，按量价 0.50）
('Smirel · 国产模型',
 'DeepSeek、Kimi、GLM、Qwen 等国产模型线路。无日限额。',
 0.10, 'composite', 'subscription', 30,
 'domestic', 0.10, 0.50, 0.80, 0.15,
 0, false, false, false,
 false, 'active', 30, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- 12 个主套餐（SubscriptionPlan）
-- 注：validity_days 统一为 30（一个月）；周控制额度在 group 上不设置，由用户订阅时按 group 限额执行
-- sort_order 越小越靠前
INSERT INTO subscription_plans (
    group_id, name, description, price, original_price, currency,
    validity_days, validity_unit, features, product_name,
    for_sale, sort_order,
    created_at, updated_at
)
SELECT g.id, plan.name, plan.description, plan.price, plan.original_price, 'CNY',
       30, 'day', plan.features, plan.product_name,
       true, plan.sort_order, NOW(), NOW()
FROM (VALUES
  -- GPT 线路 3 档（来源：套餐定价方案 5.2）
  ('Smirel · GPT Plus', 'GPT Plus 轻享月卡', 'GPT Plus 货源，1 席轻量使用，无日限额',
    89.00, 190.00, 'GPT Plus 轻享月卡',
    'Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5；不含 Claude Fable 5.1。
模型：GPT-6 Astra / GPT-5.6 Sol / GPT-5.6 Terra / GPT-5.6 Luna
官方额度：$190/月（按 ¥0.50/官方$ 的按量价 ≈ ¥95，本卡 ¥89 ≈ 95 折）
1 席，5 并发，无日限额',
    1),
  ('Smirel · GPT Pro', 'GPT Pro 专业月卡', 'GPT Pro 货源，1 席标准使用',
    379.00, 600.00, 'GPT Pro 专业月卡',
    '模型：GPT-6 Astra / GPT-5.6 Sol / GPT-5.6 Terra / GPT-5.6 Luna
官方额度：$600/月（按量价 ≈ ¥450，本卡 ¥379 ≈ 85 折）
1 席，5 并发，无日限额',
    2),
  ('Smirel · GPT Pro', 'GPT Pro 企业试运行', 'GPT Pro 货源，20 席企业试运行',
    2469.00, 3000.00, 'GPT Pro 企业试运行',
    '模型：GPT-6 Astra / GPT-5.6 Sol / GPT-5.6 Terra / GPT-5.6 Luna
官方额度：$3000/月（按量价 ≈ ¥2250，本卡 ¥2469 ≈ 1.10 倍权益溢价）
20 席，30+ 并发，独享 quota',
    3),

  -- Claude 无 Fable 3 档
  ('Smirel · ClaudeCode', 'ClaudeCode Plus 轻享月卡', 'Claude 普通线路，1 席轻量使用',
    559.00, 850.00, 'ClaudeCode Plus 轻享月卡',
    '支持：Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5；不含 Claude Fable 5.1
官方额度：$850/月（按量价 ≈ ¥595，本卡 ¥559 ≈ 95 折）
1 席，8 并发，无日限额',
    4),
  ('Smirel · ClaudeCode', 'ClaudeCode Plus 标准月卡', 'Claude 普通线路，1 席标准使用',
    889.00, 1500.00, 'ClaudeCode Plus 标准月卡',
    '支持：Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5；不含 Claude Fable 5.1
官方额度：$1500/月（按量价 ≈ ¥1050，本卡 ¥889 ≈ 85 折）
1 席，12 并发，无日限额',
    5),
  ('Smirel · ClaudeCode', 'ClaudeCode Plus 企业试运行', 'Claude 普通线路，20 席企业试运行',
    2309.00, 3000.00, 'ClaudeCode Plus 企业试运行',
    '支持：Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5；不含 Claude Fable 5.1
官方额度：$3000/月（按量价 ≈ ¥2100，本卡 ¥2309 ≈ 1.10 倍权益溢价）
20 席，30+ 并发，独享 quota',
    6),

  -- Claude 含 Fable / Max 3 档
  ('Smirel · ClaudeCode Max', 'ClaudeCode Max 轻享月卡', 'Claude 高权益线路，限量开放',
    1179.00, 540.00, 'ClaudeCode Max 轻享月卡',
    '支持：Claude Fable 5.1、Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5
官方额度：$540/月（按量价 ≈ ¥1242，本卡 ¥1179 略低于按量价）
1 席，5 并发，限量开放（白名单优先）',
    7),
  ('Smirel · ClaudeCode Max', 'ClaudeCode Max 标准月卡', 'Claude 高权益线路，限量开放',
    2049.00, 1050.00, 'ClaudeCode Max 标准月卡',
    '支持：Claude Fable 5.1、Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5
官方额度：$1050/月（按量价 ≈ ¥2415，本卡 ¥2049 低于按量价）
1 席，8 并发，限量开放',
    8),
  ('Smirel · ClaudeCode Max', 'ClaudeCode Max 企业试运行', 'Claude 高权益线路，企业试运行',
    9099.00, 3600.00, 'ClaudeCode Max 企业试运行',
    '支持：Claude Fable 5.1、Claude Opus 5、Claude Sonnet 5、Claude Haiku 4.5
官方额度：$3600/月（按量价 ≈ ¥8280，本卡 ¥9099 ≈ 1.10 倍权益溢价 + Fable 溢价）
20 席，30+ 并发，白名单开放',
    9),

  -- 国产模型 3 档
  ('Smirel · 国产模型', '国产模型轻享月卡', '国产模型线路，1 席轻量使用',
    69.00, 160.00, '国产模型轻享月卡',
    '支持：DeepSeek、Kimi、GLM、Qwen 等系统实时上架的国产模型
官方额度：$160/月（按量价 ≈ ¥80，本卡 ¥69 ≈ 95 折）
1 席，5 并发，无日限额',
    10),
  ('Smirel · 国产模型', '国产模型专业月卡', '国产模型线路，1 席专业使用',
    209.00, 500.00, '国产模型专业月卡',
    '支持：DeepSeek、Kimi、GLM、Qwen 等系统实时上架的国产模型
官方额度：$500/月（按量价 ≈ ¥250，本卡 ¥209 ≈ 85 折）
1 席，10 并发，无日限额',
    11),
  ('Smirel · 国产模型', '国产模型企业试运行', '国产模型线路，20 席企业试运行',
    739.00, 1350.00, '国产模型企业试运行',
    '支持：DeepSeek、Kimi、GLM、Qwen 等系统实时上架的国产模型
官方额度：$1350/月（按量价 ≈ ¥675，本卡 ¥739 ≈ 1.10 倍权益溢价）
20 席，30+ 并发，企业风控',
    12)
) AS plan(group_name, name, description, price, original_price, product_name, features, sort_order)
JOIN groups g ON g.name = plan.group_name
ON CONFLICT DO NOTHING;
