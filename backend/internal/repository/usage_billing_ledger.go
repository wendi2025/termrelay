package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// usageBillingLedgerDebitSQL 把一次余额扣费按 FIFO 归属到用户的充值订单账本。
//
// 规则（方案 9.2）：
//   - 先扣充值本金（principal），后扣赠送余额（bonus）
//   - 同一类型内按充值时间先进先出
//   - 已冻结（退款完成）的订单不参与归属
//
// 参数：$1=user_id，$2=本次扣费金额，$3=memo（含 request_id，便于对账与重放）
// balance_after 在 debit 分录上表示"该订单该分录后的剩余本金/赠送"。
const usageBillingLedgerDebitSQL = `
WITH credits AS (
    SELECT l.order_id,
           l.entry_type,
           MIN(l.created_at) AS first_credited_at,
           SUM(l.amount) AS credited
    FROM user_balance_ledger l
    WHERE l.user_id = $1 AND l.direction = 'credit' AND l.frozen = FALSE
    GROUP BY l.order_id, l.entry_type
),
consumed AS (
    SELECT l.order_id, l.entry_type, SUM(l.amount) AS used
    FROM user_balance_ledger l
    WHERE l.user_id = $1 AND l.direction = 'debit'
    GROUP BY l.order_id, l.entry_type
),
open_order AS (
    SELECT c.order_id,
           c.entry_type,
           c.first_credited_at,
           c.credited - COALESCE(x.used, 0) AS remaining
    FROM credits c
    LEFT JOIN consumed x ON x.order_id = c.order_id AND x.entry_type = c.entry_type
    WHERE c.credited - COALESCE(x.used, 0) > 0
),
alloc AS (
    SELECT order_id,
           entry_type,
           remaining,
           LEAST(remaining, GREATEST($2 - COALESCE(SUM(remaining) OVER (
               ORDER BY (entry_type = 'bonus'), first_credited_at, order_id
               ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
           ), 0), 0)) AS take
    FROM open_order
)
INSERT INTO user_balance_ledger
    (user_id, order_id, entry_type, direction, amount, balance_after, memo, frozen, created_at, updated_at)
SELECT $1, order_id, entry_type, 'debit', take, remaining - take, $3, FALSE, NOW(), NOW()
FROM alloc
WHERE take > 0
  AND NOT EXISTS (
      SELECT 1 FROM user_balance_ledger d
      WHERE d.user_id = $1 AND d.direction = 'debit' AND d.memo = $3
  )
`

// recordUsageBillingLedgerDebit 记录一次余额扣费的订单归属。
//
// 该写入不参与计费主事务：账本异常不得导致计费失败（失败会连 usage_log 一起丢），
// 因此失败时只记录 ALERT 日志，由运维按 request_id 对账补录。
func (r *usageBillingRepository) recordUsageBillingLedgerDebit(ctx context.Context, userID int64, amount float64, memo string) {
	if r == nil || r.db == nil || userID <= 0 || amount <= 0 {
		return
	}
	memo = strings.TrimSpace(memo)
	if memo == "" {
		return
	}
	if _, err := r.db.ExecContext(ctx, usageBillingLedgerDebitSQL, userID, amount, memo); err != nil {
		logger.LegacyPrintf("repository.usage_billing",
			"ALERT: balance ledger debit failed user=%d amount=%f memo=%s: %v", userID, amount, memo, err)
	}
}

// ledgerDebitMemo 构造账本扣费备注（同一 request_id 幂等，可安全重放）
func ledgerDebitMemo(prefix, requestID string) string {
	id := strings.TrimSpace(requestID)
	if id == "" {
		return ""
	}
	if prefix == "" {
		prefix = "usage"
	}
	return fmt.Sprintf("%s:%s", prefix, id)
}
