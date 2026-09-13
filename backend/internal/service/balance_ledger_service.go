package service

import (
	"context"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/userbalanceledger"
)

// 余额账本分录类型（user_balance_ledger.entry_type）
const (
	ledgerEntryTypePrincipal = "principal"
	ledgerEntryTypeBonus     = "bonus"
)

// 余额账本变动方向（user_balance_ledger.direction）
const (
	ledgerDirectionCredit = "credit"
	ledgerDirectionDebit  = "debit"
)

// BalanceLedgerService 余额分账账本（方案 9.2）
//
// 职责：
//   - 充值成功后按订单登记"充值本金"入账（幂等）
//   - 记录赠送余额入账（当前无赠送入口，保留完整能力）
//   - 退款完成后冻结该订单账本，保留审计链路
//   - 输出单笔订单的账本汇总，供 9.3 退款审核记录使用
//
// 单位：账本金额与 user.balance 一致，均为"记账单位"（订单 Amount 口径），
// 不是支付币种金额。换算见 refund_calculator.go 的金额单位换算小节。
type BalanceLedgerService struct {
	client *dbent.Client
}

// NewBalanceLedgerService 创建余额账本服务
func NewBalanceLedgerService(client *dbent.Client) *BalanceLedgerService {
	if client == nil {
		panic("NewBalanceLedgerService: client is nil")
	}
	return &BalanceLedgerService{client: client}
}

// LedgerOrderSummary 单笔订单的账本汇总（方案 9.3 退款审核记录字段）
type LedgerOrderSummary struct {
	OrderID         int64   `json:"order_id"`
	PrincipalCredit float64 `json:"principal_credit"`
	PrincipalDebit  float64 `json:"principal_debit"`
	PrincipalLeft   float64 `json:"principal_left"`
	BonusCredit     float64 `json:"bonus_credit"`
	BonusDebit      float64 `json:"bonus_debit"`
	BonusLeft       float64 `json:"bonus_left"`
	Frozen          bool    `json:"frozen"`
	EntryCount      int     `json:"entry_count"`
}

// RecordRechargeCredit 登记充值本金入账（幂等：同一订单只登记一笔本金 credit）
func (s *BalanceLedgerService) RecordRechargeCredit(ctx context.Context, order *dbent.PaymentOrder) error {
	if s == nil || s.client == nil || order == nil {
		return nil
	}
	if order.Amount <= 0 {
		return nil
	}
	return s.recordCredit(ctx, order, ledgerEntryTypePrincipal, order.Amount,
		fmt.Sprintf("recharge order:%d", order.ID))
}

// RecordBonusCredit 登记赠送余额入账（幂等）。赠送余额不提现、不折现，
// 退款时随订单一并取消。
func (s *BalanceLedgerService) RecordBonusCredit(ctx context.Context, order *dbent.PaymentOrder, amount float64, memo string) error {
	if s == nil || s.client == nil || order == nil || amount <= 0 {
		return nil
	}
	if strings.TrimSpace(memo) == "" {
		memo = fmt.Sprintf("bonus order:%d", order.ID)
	}
	return s.recordCredit(ctx, order, ledgerEntryTypeBonus, amount, memo)
}

func (s *BalanceLedgerService) recordCredit(ctx context.Context, order *dbent.PaymentOrder, entryType string, amount float64, memo string) error {
	exists, err := s.client.UserBalanceLedger.Query().
		Where(userbalanceledger.OrderIDEQ(order.ID)).
		Where(userbalanceledger.EntryTypeEQ(entryType)).
		Where(userbalanceledger.DirectionEQ(ledgerDirectionCredit)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check ledger credit: %w", err)
	}
	if exists {
		return nil
	}
	balanceAfter := 0.0
	if user, userErr := s.client.User.Get(ctx, order.UserID); userErr == nil && user != nil {
		balanceAfter = user.Balance
	}
	if _, err := s.client.UserBalanceLedger.Create().
		SetUserID(order.UserID).
		SetOrderID(order.ID).
		SetEntryType(entryType).
		SetDirection(ledgerDirectionCredit).
		SetAmount(amount).
		SetBalanceAfter(balanceAfter).
		SetMemo(memo).
		SetFrozen(false).
		Save(ctx); err != nil {
		return fmt.Errorf("create ledger credit: %w", err)
	}
	return nil
}

// SettleOrderLedger 退款完成后关闭该订单账本：标记 frozen 并记录退款批次 ID。
//
// 说明：冻结后该订单分录不再参与 FIFO 扣费归属与可用余额计算，
// 保证"退款后不再调用该订单余额"。余额本身的扣减由退款扣减路径
// （RefundPlan.BalanceToDeduct / 订阅天数回收）负责，本方法不重复扣款。
func (s *BalanceLedgerService) SettleOrderLedger(ctx context.Context, orderID int64, refundBatchID string) error {
	if s == nil || s.client == nil || orderID <= 0 {
		return nil
	}
	update := s.client.UserBalanceLedger.Update().
		Where(userbalanceledger.OrderIDEQ(orderID)).
		SetFrozen(true)
	if strings.TrimSpace(refundBatchID) != "" {
		update = update.SetRefundBatchID(strings.TrimSpace(refundBatchID))
	}
	if _, err := update.Save(ctx); err != nil {
		return fmt.Errorf("settle ledger order %d: %w", orderID, err)
	}
	return nil
}

// OrderSummary 汇总单笔订单的账本（本金/赠送的入账、扣减与剩余）
func (s *BalanceLedgerService) OrderSummary(ctx context.Context, orderID int64) (*LedgerOrderSummary, error) {
	if s == nil || s.client == nil || orderID <= 0 {
		return &LedgerOrderSummary{}, nil
	}
	entries, err := s.client.UserBalanceLedger.Query().
		Where(userbalanceledger.OrderIDEQ(orderID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query ledger order %d: %w", orderID, err)
	}
	summary := &LedgerOrderSummary{OrderID: orderID, EntryCount: len(entries)}
	for _, e := range entries {
		credit := e.Direction == ledgerDirectionCredit
		debit := e.Direction == ledgerDirectionDebit
		switch e.EntryType {
		case ledgerEntryTypeBonus:
			if credit {
				summary.BonusCredit += e.Amount
			} else if debit {
				summary.BonusDebit += e.Amount
			}
		default:
			if credit {
				summary.PrincipalCredit += e.Amount
			} else if debit {
				summary.PrincipalDebit += e.Amount
			}
		}
		if e.Frozen {
			summary.Frozen = true
		}
	}
	summary.PrincipalLeft = summary.PrincipalCredit - summary.PrincipalDebit
	if summary.PrincipalLeft < 0 {
		summary.PrincipalLeft = 0
	}
	summary.BonusLeft = summary.BonusCredit - summary.BonusDebit
	if summary.BonusLeft < 0 {
		summary.BonusLeft = 0
	}
	return summary, nil
}
