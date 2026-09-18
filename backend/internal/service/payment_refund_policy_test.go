package service

import (
	"context"
	"math"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func TestCalculateRefundCreditAmount(t *testing.T) {
	cases := []struct {
		name        string
		orderAmount float64
		payAmount   float64
		payRefund   float64
		want        float64
	}{
		{name: "全额退款直接取记账金额", orderAmount: 200, payAmount: 100, payRefund: 100, want: 200},
		{name: "比例退款按记账倍率折算", orderAmount: 200, payAmount: 100, payRefund: 50, want: 100},
		{name: "无记账口径按 1:1", orderAmount: 0, payAmount: 100, payRefund: 50, want: 50},
		{name: "无实付口径按记账金额封顶", orderAmount: 100, payAmount: 0, payRefund: 100, want: 100},
		{name: "零退款", orderAmount: 200, payAmount: 100, payRefund: 0, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateRefundCreditAmount(tc.orderAmount, tc.payAmount, tc.payRefund, "CNY")
			if math.Abs(got-tc.want) > 0.01 {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestRefundPolicyForOrder_BalanceKeepsCreditUnit(t *testing.T) {
	paidAt := time.Now().Add(-48 * time.Hour)
	loader := &mockRefundLoader{hasUsage: true, usedPrincipal: 60}
	svc := &PaymentService{refundCalculator: NewRefundCalculator(loader)}
	o := &dbent.PaymentOrder{ID: 11, OrderType: "balance", Amount: 200, PayAmount: 100, PaidAt: &paidAt}

	policy, err := svc.refundPolicyForOrder(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if policy == nil {
		t.Fatal("want policy, got nil")
	}
	// 本金 200 - 已扣 60 = 140 记账单位，按 100/200 折算 = 70 支付币种
	if math.Abs(policy.Quote.RefundableAmount-70) > 0.01 {
		t.Fatalf("want pay amount 70, got %v", policy.Quote.RefundableAmount)
	}
	if math.Abs(policy.CreditAmount-140) > 0.01 {
		t.Fatalf("want credit amount 140, got %v", policy.CreditAmount)
	}
	if policy.auditDetail()["policy"] != "balance_principal_minus_used" {
		t.Fatalf("unexpected audit policy: %v", policy.auditDetail()["policy"])
	}
}

func TestRefundPolicyForOrder_NoCalculatorFallsBack(t *testing.T) {
	svc := &PaymentService{}
	policy, err := svc.refundPolicyForOrder(context.Background(), &dbent.PaymentOrder{ID: 12}, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if policy != nil {
		t.Fatalf("want nil policy when calculator is absent, got %+v", policy)
	}
}

func TestRefundPolicyForOrder_SubscriptionExhaustedReturnsZero(t *testing.T) {
	paidAt := time.Now().Add(-40 * 24 * time.Hour)
	loader := &mockRefundLoader{
		hasUsage:  true,
		usedUSD:   100,
		unitPrice: 1.0,
		start:     paidAt,
		end:       paidAt.Add(30 * 24 * time.Hour),
		hasPeriod: true,
	}
	svc := &PaymentService{refundCalculator: NewRefundCalculator(loader)}
	o := &dbent.PaymentOrder{ID: 13, OrderType: "subscription", Amount: 100, PayAmount: 100, PaidAt: &paidAt}

	policy, err := svc.refundPolicyForOrder(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if policy.CreditAmount != 0 || policy.Quote.RefundableAmount != 0 {
		t.Fatalf("want zero refund, got credit=%v pay=%v", policy.CreditAmount, policy.Quote.RefundableAmount)
	}
}

// 存量订单（只写了 Amount、没有 PayAmount）在 24h 全额窗口与按量充值公式下
// 必须按 1:1 退化，不能报价为 0。
func TestRefundPolicyForOrder_LegacyOrderFallsBackToOneToOne(t *testing.T) {
	paidAt := time.Now().Add(-1 * time.Hour)
	svc := &PaymentService{refundCalculator: NewRefundCalculator(&mockRefundLoader{})}
	o := &dbent.PaymentOrder{ID: 14, OrderType: "balance", Amount: 100, PayAmount: 0, PaidAt: &paidAt}

	policy, err := svc.refundPolicyForOrder(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if policy == nil {
		t.Fatal("want policy, got nil")
	}
	if math.Abs(policy.Quote.RefundableAmount-100) > 0.01 {
		t.Fatalf("want pay amount 100, got %v", policy.Quote.RefundableAmount)
	}
	if math.Abs(policy.CreditAmount-100) > 0.01 {
		t.Fatalf("want credit amount 100, got %v", policy.CreditAmount)
	}
}

// 存量充值订单走按量公式（24h 窗口外）时同样按 1:1 退化。
func TestRefundPolicyForOrder_LegacyBalanceFormulaFallsBack(t *testing.T) {
	paidAt := time.Now().Add(-72 * time.Hour)
	loader := &mockRefundLoader{hasUsage: true, usedPrincipal: 30}
	svc := &PaymentService{refundCalculator: NewRefundCalculator(loader)}
	o := &dbent.PaymentOrder{ID: 15, OrderType: "balance", Amount: 100, PayAmount: 0, PaidAt: &paidAt}

	policy, err := svc.refundPolicyForOrder(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if math.Abs(policy.Quote.RefundableAmount-70) > 0.01 {
		t.Fatalf("want pay amount 70, got %v", policy.Quote.RefundableAmount)
	}
	if math.Abs(policy.CreditAmount-70) > 0.01 {
		t.Fatalf("want credit amount 70, got %v", policy.CreditAmount)
	}
}

func TestPsMergeAuditDetail(t *testing.T) {
	merged := psMergeAuditDetail(map[string]any{"amount": 1.0}, map[string]any{"policy": "full_refund_24h_window"})
	if merged["amount"] != 1.0 || merged["policy"] != "full_refund_24h_window" {
		t.Fatalf("unexpected merge result: %+v", merged)
	}
	if got := psMergeAuditDetail(map[string]any{"amount": 1.0}, nil); got["policy"] != nil {
		t.Fatalf("nil extra must not add keys: %+v", got)
	}
}
