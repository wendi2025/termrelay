-- Migration 196: Smirel 多受益人分账（共建者分成）
-- 背景：中转站由多人共建，客户每支付一笔，需要按固定比例把「可分配收入」
--       记到各受益人名下。
--
-- 重要边界（法律红线）：
--   本表只做「记账」，不做任何对外出金。客户支付的钱整笔进入平台收款账户，
--   由管理员线下手工打款给共建者，再在系统里标记「已结算」。
--   系统自动把钱转给第三方 = 资金清分（二清），无牌照属违法行为。
--
-- 计费基数口径（revenue_split_base_mode）：
--   gross           = 客户实付毛额
--   gross_after_fee = 客户实付毛额 - 支付通道费（默认，避免把通道费也分掉）
--   上游 API 成本不在其中：余额是预收款，客户消耗时才产生上游成本，
--   因此分成比例必须留出足够空间，不要分满 100%。

CREATE TABLE IF NOT EXISTS revenue_split_rules (
    id                  BIGSERIAL PRIMARY KEY,
    beneficiary_user_id BIGINT NOT NULL,
    beneficiary_name    TEXT NOT NULL DEFAULT '',
    ratio_percent       DECIMAL(10,6) NOT NULL DEFAULT 0,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    note                TEXT,
    sort_order          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_revenue_split_rules_beneficiary
    ON revenue_split_rules (beneficiary_user_id);

CREATE INDEX IF NOT EXISTS idx_revenue_split_rules_enabled
    ON revenue_split_rules (enabled, sort_order);

CREATE TABLE IF NOT EXISTS revenue_split_entries (
    id                  BIGSERIAL PRIMARY KEY,
    order_id            BIGINT NOT NULL,
    beneficiary_user_id BIGINT NOT NULL,
    beneficiary_name    TEXT NOT NULL DEFAULT '',
    order_amount        DECIMAL(20,8) NOT NULL DEFAULT 0,
    pay_amount          DECIMAL(20,8) NOT NULL DEFAULT 0,
    channel_fee_percent DECIMAL(10,6) NOT NULL DEFAULT 0,
    channel_fee_amount  DECIMAL(20,8) NOT NULL DEFAULT 0,
    base_mode           VARCHAR(32) NOT NULL DEFAULT 'gross_after_fee',
    base_amount         DECIMAL(20,8) NOT NULL DEFAULT 0,
    ratio_percent       DECIMAL(10,6) NOT NULL DEFAULT 0,
    split_amount        DECIMAL(20,8) NOT NULL DEFAULT 0,
    currency            VARCHAR(8) NOT NULL DEFAULT 'CNY',
    status              VARCHAR(16) NOT NULL DEFAULT 'pending',
    settlement_id       BIGINT,
    reversed_at         TIMESTAMPTZ,
    reverse_reason      TEXT,
    memo                TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT revenue_split_entries_order_beneficiary_key UNIQUE (order_id, beneficiary_user_id)
);

CREATE INDEX IF NOT EXISTS idx_revenue_split_entries_beneficiary_status
    ON revenue_split_entries (beneficiary_user_id, status);

CREATE INDEX IF NOT EXISTS idx_revenue_split_entries_order
    ON revenue_split_entries (order_id);

CREATE INDEX IF NOT EXISTS idx_revenue_split_entries_pending
    ON revenue_split_entries (status, settlement_id);

CREATE INDEX IF NOT EXISTS idx_revenue_split_entries_created
    ON revenue_split_entries (created_at);

CREATE TABLE IF NOT EXISTS revenue_split_settlements (
    id                  BIGSERIAL PRIMARY KEY,
    beneficiary_user_id BIGINT NOT NULL,
    beneficiary_name    TEXT NOT NULL DEFAULT '',
    period_start        TIMESTAMPTZ,
    period_end          TIMESTAMPTZ,
    entry_count         INT NOT NULL DEFAULT 0,
    amount              DECIMAL(20,8) NOT NULL DEFAULT 0,
    currency            VARCHAR(8) NOT NULL DEFAULT 'CNY',
    status              VARCHAR(16) NOT NULL DEFAULT 'draft',
    method              VARCHAR(32),
    reference           TEXT,
    note                TEXT,
    paid_at             TIMESTAMPTZ,
    paid_by             BIGINT,
    created_by          BIGINT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_revenue_split_settlements_beneficiary
    ON revenue_split_settlements (beneficiary_user_id, status);

CREATE INDEX IF NOT EXISTS idx_revenue_split_settlements_status
    ON revenue_split_settlements (status, created_at DESC);

-- 默认设置：先关闭。管理员在后台配置好比例后再打开，避免误计提。
INSERT INTO settings (key, value, updated_at) VALUES
    ('revenue_split_enabled', 'false', NOW()),
    ('revenue_split_base_mode', 'gross_after_fee', NOW()),
    ('revenue_split_channel_fee_percent', '1.6', NOW())
ON CONFLICT (key) DO NOTHING;

COMMENT ON TABLE revenue_split_rules IS 'Smirel 分账规则：各共建者的固定分成比例（记账用，不出金）';
COMMENT ON COLUMN revenue_split_rules.ratio_percent IS '分成比例（百分比，0-100）；所有启用规则之和必须 <= 100';
COMMENT ON TABLE revenue_split_entries IS 'Smirel 分账明细：一笔订单 × 一个受益人 = 一条分录（幂等）';
COMMENT ON COLUMN revenue_split_entries.base_amount IS '计费基数（按 base_mode 口径计算后的金额）';
COMMENT ON COLUMN revenue_split_entries.split_amount IS '应分金额 = base_amount × ratio_percent / 100';
COMMENT ON COLUMN revenue_split_entries.status IS 'pending=待结算 / settled=已结算 / reversed=已冲回（退款）';
COMMENT ON COLUMN revenue_split_entries.settlement_id IS '所属结算单；非空表示已被结算单锁定，不可重复结算';
COMMENT ON TABLE revenue_split_settlements IS 'Smirel 结算单：一次线下手工打款的凭证记录（draft/paid/cancelled）';
COMMENT ON COLUMN revenue_split_settlements.reference IS '线下打款流水号/凭证号';
