package service

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// refundPolicy 退款政策计算结果（方案 9.1 / 9.2）
//
// 单位：
//   - Quote.RefundableAmount：支付币种（实付口径）
//   - CreditAmount：订单记账单位（余额 / user_balance_ledger 口径）
//
// 退款执行链（余额扣减、RefundPlan.RefundAmount、payment_orders.refund_amount）
// 统一使用记账单位，避免与套餐费/充值倍率口径混淆。
type refundPolicy struct {
	Quote        *RefundQuote
	CreditAmount float64
}

// auditDetail 退款审核记录（方案 9.3：公式结果与处理口径必须可追溯）
func (p *refundPolicy) auditDetail() map[string]any {
	if p == nil || p.Quote == nil {
		return map[string]any{}
	}
	return map[string]any{
		"policy":                p.Quote.Reason,
		"refundable_pay_amount": p.Quote.RefundableAmount,
		"refundable_credit":     p.CreditAmount,
		"full_refund_window":    p.Quote.FullRefundWindow,
		"force_refund_eligible": p.Quote.ForceRefundEligible,
		"calculation_breakdown": p.Quote.CalculationBreakdown,
	}
}

// refundPolicyForOrder 依据退款计算器计算订单的可退上限。
// 计算器未注入时返回 (nil, nil)，调用方回落历史行为。
func (s *PaymentService) refundPolicyForOrder(ctx context.Context, o *dbent.PaymentOrder, force bool) (*refundPolicy, error) {
	if s == nil || s.refundCalculator == nil || o == nil {
		return nil, nil
	}
	quote, err := s.refundCalculator.CalculateRefund(ctx, o, force)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, nil
	}
	payRefund := quote.RefundableAmount
	if payRefund < 0 {
		payRefund = 0
	}
	return &refundPolicy{
		Quote:        quote,
		CreditAmount: calculateRefundCreditAmount(o.Amount, o.PayAmount, payRefund, PaymentOrderCurrency(o)),
	}, nil
}

// psMergeAuditDetail 合并审计明细，extra 覆盖同名字段
func psMergeAuditDetail(base, extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return base
	}
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range extra {
		base[k] = v
	}
	return base
}
