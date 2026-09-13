-- Migration 193: Smirel 订阅/定价扩展字段
-- Adds product_line + cost/pricing configuration fields to the groups table so
-- each product line (GPT Plus/Pro, ClaudeCode 无/含 Fable, 国产模型) can carry
-- its own 成本倍率 / 按量报价 / 损耗系数 / 折扣上限 / 并发 / 熔断 / 独享 quota
-- / 白名单配置。

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS product_line VARCHAR(32) NOT NULL DEFAULT 'custom';

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS cost_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0.10;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS pay_as_you_go_price_per_usd DECIMAL(10,4) NOT NULL DEFAULT 0.50;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS loss_coefficient DECIMAL(10,4) NOT NULL DEFAULT 0.80;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS max_discount_pct DECIMAL(5,4) NOT NULL DEFAULT 0.15;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS concurrency_limit INTEGER NOT NULL DEFAULT 0;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS circuit_breaker_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS exclusive_quota BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS whitelist_only BOOLEAN NOT NULL DEFAULT FALSE;

-- product_line 过滤索引（后台按产品线筛选分组）
CREATE INDEX IF NOT EXISTS idx_groups_product_line_active
    ON groups (product_line)
    WHERE deleted_at IS NULL;

-- whitelist_only 复合索引（白名单销售筛选）
CREATE INDEX IF NOT EXISTS idx_groups_whitelist_only
    ON groups (whitelist_only)
    WHERE deleted_at IS NULL AND whitelist_only = TRUE;

COMMENT ON COLUMN groups.product_line IS '产品线标识：gpt_plus/gpt_pro/claude_no_fable/claude_fable/domestic/custom';
COMMENT ON COLUMN groups.cost_multiplier IS '上游成本倍率（¥/官方$）';
COMMENT ON COLUMN groups.pay_as_you_go_price_per_usd IS '按量计费对外报价（¥/官方$）';
COMMENT ON COLUMN groups.loss_coefficient IS '综合损耗系数，用于财务压力测试';
COMMENT ON COLUMN groups.max_discount_pct IS '最高折扣比例，返利和大客折扣不叠加';
COMMENT ON COLUMN groups.concurrency_limit IS '该分组最大并发数，0 = 不限制';
COMMENT ON COLUMN groups.circuit_breaker_enabled IS '是否启用异常熔断';
COMMENT ON COLUMN groups.exclusive_quota IS '是否独享 quota（企业试运行档）';
COMMENT ON COLUMN groups.whitelist_only IS '白名单销售：仅白名单内用户可订阅';