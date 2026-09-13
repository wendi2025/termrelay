-- Migration 194: Smirel 余额分账表
-- 实现充值本金/赠送余额分账管理，支持按订单退款（FIFO 原则）
-- 详见：套餐定价方案 9.2 节

CREATE TABLE IF NOT EXISTS user_balance_ledger (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    entry_type VARCHAR(16) NOT NULL DEFAULT 'principal',
    direction VARCHAR(8) NOT NULL DEFAULT 'credit',
    amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_after DECIMAL(20,8) NOT NULL DEFAULT 0,
    memo TEXT,
    frozen BOOLEAN NOT NULL DEFAULT FALSE,
    refund_batch_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FIFO 扣费所需：按用户 + 时间排序
CREATE INDEX IF NOT EXISTS idx_user_balance_ledger_user_created
    ON user_balance_ledger (user_id, created_at);

-- 按订单查询（退款时定位订单的所有分录）
CREATE INDEX IF NOT EXISTS idx_user_balance_ledger_order
    ON user_balance_ledger (order_id);

-- 按用户 + 类型 + 冻结状态查询（计算可用余额）
CREATE INDEX IF NOT EXISTS idx_user_balance_ledger_user_type_frozen
    ON user_balance_ledger (user_id, entry_type, frozen);

-- 退款批次查询
CREATE INDEX IF NOT EXISTS idx_user_balance_ledger_refund_batch
    ON user_balance_ledger (refund_batch_id);

COMMENT ON TABLE user_balance_ledger IS 'Smirel 余额分账表：充值本金/赠送余额分账记录，支持按订单退款（FIFO）';
COMMENT ON COLUMN user_balance_ledger.user_id IS '用户 ID';
COMMENT ON COLUMN user_balance_ledger.order_id IS '关联的充值订单 ID';
COMMENT ON COLUMN user_balance_ledger.entry_type IS '余额类型：principal=充值本金 / bonus=赠送余额';
COMMENT ON COLUMN user_balance_ledger.direction IS '变动方向：credit=入账 / debit=扣减';
COMMENT ON COLUMN user_balance_ledger.amount IS '变动金额（始终为正数）';
COMMENT ON COLUMN user_balance_ledger.balance_after IS '变动后的剩余余额（冗余，便于审计）';
COMMENT ON COLUMN user_balance_ledger.memo IS '备注/退款关联 ID/原始扣费 usage_log_id';
COMMENT ON COLUMN user_balance_ledger.frozen IS '该订单是否已冻结（退款完成后清零）';
COMMENT ON COLUMN user_balance_ledger.refund_batch_id IS '退款批次 ID';