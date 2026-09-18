package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/userbalanceledger"
)

// 余额账本只读视图（方案 9.2 / 9.3）
//
// 写路径见 balance_ledger_service.go 与 repository/usage_billing_ledger.go；
// 本文件只负责把账本读出来给退款审核与用户账单界面使用，不做任何写入。

// ledgerEntryDefaultLimit 分录默认返回条数
const ledgerEntryDefaultLimit = 50

// ledgerEntryMaxLimit 分录最大返回条数
const ledgerEntryMaxLimit = 500

// LedgerEntryView 单条余额账本分录
type LedgerEntryView struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	EntryType     string    `json:"entry_type"`
	Direction     string    `json:"direction"`
	Amount        float64   `json:"amount"`
	BalanceAfter  float64   `json:"balance_after"`
	Memo          string    `json:"memo,omitempty"`
	Frozen        bool      `json:"frozen"`
	RefundBatchID string    `json:"refund_batch_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LedgerBalanceBreakdown 用户余额的本金/赠送构成
//
// AccountBalance 是 user.balance（余额事实来源）；ActivePrincipalLeft 与
// ActiveBonusLeft 是账本口径的"当前可扣费余额构成"。历史余额可能早于账本上线，
// 因此二者之差以 LedgerGap 显式暴露，供退款审核判断账本覆盖度。
type LedgerBalanceBreakdown struct {
	UserID               int64   `json:"user_id"`
	AccountBalance       float64 `json:"account_balance"`
	PrincipalCreditTotal float64 `json:"principal_credit_total"`
	PrincipalDebitTotal  float64 `json:"principal_debit_total"`
	PrincipalLeft        float64 `json:"principal_left"`
	BonusCreditTotal     float64 `json:"bonus_credit_total"`
	BonusDebitTotal      float64 `json:"bonus_debit_total"`
	BonusLeft            float64 `json:"bonus_left"`
	ActivePrincipalLeft  float64 `json:"active_principal_left"`
	ActiveBonusLeft      float64 `json:"active_bonus_left"`
	FrozenPrincipalLeft  float64 `json:"frozen_principal_left"`
	FrozenBonusLeft      float64 `json:"frozen_bonus_left"`
	LedgerGap            float64 `json:"ledger_gap"`
	EntryCount           int     `json:"entry_count"`
}

// ListOrderEntries 返回单笔订单的账本分录（新到旧）
func (s *BalanceLedgerService) ListOrderEntries(ctx context.Context, orderID int64, limit int) ([]LedgerEntryView, error) {
	if s == nil || s.client == nil || orderID <= 0 {
		return nil, nil
	}
	entries, err := s.client.UserBalanceLedger.Query().
		Where(userbalanceledger.OrderIDEQ(orderID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ledger order %d: %w", orderID, err)
	}
	return ledgerEntryViews(entries, limit), nil
}

// ListUserEntries 返回用户维度的账本分录（新到旧）
func (s *BalanceLedgerService) ListUserEntries(ctx context.Context, userID int64, limit int) ([]LedgerEntryView, error) {
	if s == nil || s.client == nil || userID <= 0 {
		return nil, nil
	}
	entries, err := s.client.UserBalanceLedger.Query().
		Where(userbalanceledger.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ledger user %d: %w", userID, err)
	}
	return ledgerEntryViews(entries, limit), nil
}

// ledgerEntryViews 排序（新到旧）并按 limit 截断。
// 排序在内存完成：账本单用户量级小，且避免额外索引依赖。
func ledgerEntryViews(entries []*dbent.UserBalanceLedger, limit int) []LedgerEntryView {
	if limit <= 0 {
		limit = ledgerEntryDefaultLimit
	}
	if limit > ledgerEntryMaxLimit {
		limit = ledgerEntryMaxLimit
	}
	views := make([]LedgerEntryView, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		views = append(views, LedgerEntryView{
			ID:            e.ID,
			OrderID:       e.OrderID,
			EntryType:     e.EntryType,
			Direction:     e.Direction,
			Amount:        e.Amount,
			BalanceAfter:  e.BalanceAfter,
			Memo:          psStringValue(e.Memo),
			Frozen:        e.Frozen,
			RefundBatchID: psStringValue(e.RefundBatchID),
			CreatedAt:     e.CreatedAt,
		})
	}
	sortLedgerEntryViewsDesc(views)
	if len(views) > limit {
		views = views[:limit]
	}
	return views
}

// sortLedgerEntryViewsDesc 按创建时间倒序，时间相同按 ID 倒序，保证输出稳定
func sortLedgerEntryViewsDesc(views []LedgerEntryView) {
	for i := 1; i < len(views); i++ {
		cur := views[i]
		j := i - 1
		for j >= 0 && ledgerEntryViewLess(cur, views[j]) {
			views[j+1] = views[j]
			j--
		}
		views[j+1] = cur
	}
}

func ledgerEntryViewLess(a, b LedgerEntryView) bool {
	if a.CreatedAt.Equal(b.CreatedAt) {
		return a.ID > b.ID
	}
	return a.CreatedAt.After(b.CreatedAt)
}

// UserBalanceBreakdown 汇总用户余额的本金/赠送构成（方案 9.2 分账口径）
func (s *BalanceLedgerService) UserBalanceBreakdown(ctx context.Context, userID int64) (*LedgerBalanceBreakdown, error) {
	if s == nil || s.client == nil || userID <= 0 {
		return &LedgerBalanceBreakdown{}, nil
	}
	entries, err := s.client.UserBalanceLedger.Query().
		Where(userbalanceledger.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("breakdown ledger user %d: %w", userID, err)
	}
	out := &LedgerBalanceBreakdown{UserID: userID, EntryCount: len(entries)}
	for _, e := range entries {
		if e == nil {
			continue
		}
		credit := e.Direction == ledgerDirectionCredit
		debit := e.Direction == ledgerDirectionDebit
		isBonus := e.EntryType == ledgerEntryTypeBonus
		switch {
		case isBonus && credit:
			out.BonusCreditTotal += e.Amount
		case isBonus && debit:
			out.BonusDebitTotal += e.Amount
		case credit:
			out.PrincipalCreditTotal += e.Amount
		case debit:
			out.PrincipalDebitTotal += e.Amount
		}
		if e.Frozen {
			if isBonus {
				out.FrozenBonusLeft += ledgerEntryNet(e)
			} else {
				out.FrozenPrincipalLeft += ledgerEntryNet(e)
			}
		}
	}
	out.PrincipalLeft = ledgerFloorZero(out.PrincipalCreditTotal - out.PrincipalDebitTotal)
	out.BonusLeft = ledgerFloorZero(out.BonusCreditTotal - out.BonusDebitTotal)
	out.FrozenPrincipalLeft = ledgerFloorZero(out.FrozenPrincipalLeft)
	out.FrozenBonusLeft = ledgerFloorZero(out.FrozenBonusLeft)
	// 冻结订单的剩余额度不再参与扣费，从可扣费构成中剔除
	out.ActivePrincipalLeft = ledgerFloorZero(out.PrincipalLeft - out.FrozenPrincipalLeft)
	out.ActiveBonusLeft = ledgerFloorZero(out.BonusLeft - out.FrozenBonusLeft)
	if user, userErr := s.client.User.Get(ctx, userID); userErr == nil && user != nil {
		out.AccountBalance = user.Balance
	}
	out.LedgerGap = out.AccountBalance - (out.ActivePrincipalLeft + out.ActiveBonusLeft)
	return out, nil
}

// ledgerEntryNet 单条分录对所属订单剩余额度的净贡献（credit 为正、debit 取负）
func ledgerEntryNet(e *dbent.UserBalanceLedger) float64 {
	if e.Direction == ledgerDirectionDebit {
		return -e.Amount
	}
	return e.Amount
}

func ledgerFloorZero(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
