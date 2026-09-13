package service

import (
	"context"
	"errors"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// mockRefundLoader 测试用的 mock loader
type mockRefundLoader struct {
	hasUsage         bool
	hasUsageErr      error
	usedUSD          float64
	usedUSDErr       error
	usedPrincipal    float64
	usedPrincipalErr error
	bonusGrant       float64
	bonusErr         error
	alreadyRefunded  bool
	refundedErr      error
	forceEligible    bool
	forceErr         error
	start            time.Time
	end              time.Time
	hasPeriod        bool
	periodErr        error
	unitPrice        float64
	unitPriceErr     error
}

func (m *mockRefundLoader) HasUsageIn24hWindow(_ context.Context, _ *dbent.PaymentOrder) (bool, error) {
	return m.hasUsage, m.hasUsageErr
}
func (m *mockRefundLoader) LoadSubscriptionPeriodUsage(_ context.Context, _ *dbent.PaymentOrder) (float64, error) {
	return m.usedUSD, m.usedUSDErr
}
func (m *mockRefundLoader) LoadBalanceOrderUsedPrincipal(_ context.Context, _ *dbent.PaymentOrder) (float64, error) {
	return m.usedPrincipal, m.usedPrincipalErr
}
func (m *mockRefundLoader) LoadBonusGrantForOrder(_ context.Context, _ *dbent.PaymentOrder) (float64, error) {
	return m.bonusGrant, m.bonusErr
}
func (m *mockRefundLoader) IsAlreadyRefunded(_ context.Context, _ *dbent.PaymentOrder) (bool, error) {
	return m.alreadyRefunded, m.refundedErr
}
func (m *mockRefundLoader) IsForceRefundEligible(_ context.Context, _ *dbent.PaymentOrder) (bool, error) {
	return m.forceEligible, m.forceErr
}
func (m *mockRefundLoader) GetSubscriptionPeriod(_ context.Context, _ *dbent.PaymentOrder) (time.Time, time.Time, bool, error) {
	return m.start, m.end, m.hasPeriod, m.periodErr
}
func (m *mockRefundLoader) GetGroupPayAsYouGoPrice(_ context.Context, _ *dbent.PaymentOrder) (float64, error) {
	return m.unitPrice, m.unitPriceErr
}

func refundPtr(t time.Time) *time.Time { return &t }
func refundPtrI(i int) *int            { return &i }

// --- 测试 ---

func TestRefundCalculator_NilOrder(t *testing.T) {
	c := NewRefundCalculator(&mockRefundLoader{})
	_, err := c.CalculateRefund(context.Background(), nil, false)
	if !errors.Is(err, ErrNilOrder) {
		t.Fatalf("want ErrNilOrder, got %v", err)
	}
}

func TestRefundCalculator_AlreadyRefunded(t *testing.T) {
	loader := &mockRefundLoader{alreadyRefunded: true}
	c := NewRefundCalculator(loader)
	paidAt := time.Now().Add(-48 * time.Hour)
	o := &dbent.PaymentOrder{ID: 1, OrderType: "balance", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if q.RefundableAmount != 0 || q.Reason != "already_refunded" {
		t.Fatalf("want 0/already_refunded, got %v/%s", q.RefundableAmount, q.Reason)
	}
}

func TestRefundCalculator_FullRefund24hWindow(t *testing.T) {
	paidAt := time.Now().Add(-2 * time.Hour) // 2h 内
	loader := &mockRefundLoader{
		hasUsage:  false,
		unitPrice: 0.5,
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 2, OrderType: "subscription", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !q.FullRefundWindow {
		t.Fatalf("want FullRefundWindow=true")
	}
	if q.RefundableAmount != 100 {
		t.Fatalf("want 100, got %v", q.RefundableAmount)
	}
}

func TestRefundCalculator_SubscriptionDoubleSettlement(t *testing.T) {
	paidAt := time.Now().Add(-15 * 24 * time.Hour) // 15 天前
	periodStart := paidAt
	periodEnd := paidAt.Add(30 * 24 * time.Hour) // 30 天套餐
	loader := &mockRefundLoader{
		hasUsage:  true, // 已使用 → 跳过 24h 窗口
		usedUSD:   60,   // 已用 60 USD
		unitPrice: 1.0,  // 1 CNY/USD
		start:     periodStart,
		end:       periodEnd,
		hasPeriod: true,
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 3, OrderType: "subscription", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// 公式: time_based = 100 * 15/30 = 50
	//       usage_based = 60 * 1.0 = 60
	//       consumed = max(50, 60) = 60
	//       refund = 100 - 60 = 40
	if q.RefundableAmount != 40 {
		t.Fatalf("want 40, got %v (breakdown=%v)", q.RefundableAmount, q.CalculationBreakdown)
	}
}

func TestRefundCalculator_SubscriptionTimeBasedWins(t *testing.T) {
	paidAt := time.Now().Add(-25 * 24 * time.Hour) // 25 天前
	periodStart := paidAt
	periodEnd := paidAt.Add(30 * 24 * time.Hour) // 30 天套餐
	loader := &mockRefundLoader{
		hasUsage:  true,
		usedUSD:   10, // 使用很少
		unitPrice: 1.0,
		start:     periodStart,
		end:       periodEnd,
		hasPeriod: true,
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 4, OrderType: "subscription", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// time_based = 100 * 25/30 ≈ 83.33
	// usage_based = 10 * 1.0 = 10
	// consumed = max(83.33, 10) ≈ 83.33
	// refund = 100 - 83.33 ≈ 16.67
	if q.RefundableAmount < 16 || q.RefundableAmount > 17 {
		t.Fatalf("want ~16.67, got %v", q.RefundableAmount)
	}
}

func TestRefundCalculator_BalanceRefund(t *testing.T) {
	paidAt := time.Now().Add(-48 * time.Hour)
	loader := &mockRefundLoader{
		hasUsage:      true,
		usedPrincipal: 30, // 已扣 30 本金
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 5, OrderType: "balance", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if q.RefundableAmount != 70 {
		t.Fatalf("want 70, got %v", q.RefundableAmount)
	}
	if q.Reason != "balance_principal_minus_used" {
		t.Fatalf("want balance_principal_minus_used, got %s", q.Reason)
	}
}

func TestRefundCalculator_BalanceNoUsageFullRefund(t *testing.T) {
	paidAt := time.Now().Add(-48 * time.Hour)
	loader := &mockRefundLoader{
		hasUsage:      false, // 没消费
		usedPrincipal: 0,
		forceEligible: false,
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 6, OrderType: "balance", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// paid_at > 24h → 跳过 24h 窗口
	// 按量公式: 100 - 0 = 100
	if q.RefundableAmount != 100 {
		t.Fatalf("want 100, got %v", q.RefundableAmount)
	}
}

func TestRefundCalculator_ForceRefund(t *testing.T) {
	paidAt := time.Now().Add(-48 * time.Hour)
	loader := &mockRefundLoader{
		forceEligible: true,
		hasUsage:      true,
		usedPrincipal: 30,
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 7, OrderType: "balance", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, true) // force=true
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !q.ForceRefundEligible {
		t.Fatal("want ForceRefundEligible=true")
	}
	if q.RefundableAmount != 100 {
		t.Fatalf("force refund should be full pay amount, got %v", q.RefundableAmount)
	}
}

func TestRefundCalculator_ForceRefundNotEligible(t *testing.T) {
	paidAt := time.Now().Add(-48 * time.Hour)
	loader := &mockRefundLoader{
		forceEligible: false,
	}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 8, OrderType: "balance", PayAmount: 100, PaidAt: &paidAt}
	q, err := c.CalculateRefund(context.Background(), o, true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if q.RefundableAmount != 0 || q.Reason != "force_refund_not_eligible" {
		t.Fatalf("want 0/force_refund_not_eligible, got %v/%s", q.RefundableAmount, q.Reason)
	}
}

func TestRefundCalculator_UnknownOrderType(t *testing.T) {
	paidAt := time.Now().Add(-48 * time.Hour)
	loader := &mockRefundLoader{hasUsage: true}
	c := NewRefundCalculator(loader)
	o := &dbent.PaymentOrder{ID: 9, OrderType: "weird_type", PayAmount: 100, PaidAt: &paidAt}
	_, err := c.CalculateRefund(context.Background(), o, false)
	if !errors.Is(err, ErrUnknownOrderType) {
		t.Fatalf("want ErrUnknownOrderType, got %v", err)
	}
}

func TestRefundCalculator_NilLoader(t *testing.T) {
	c := &RefundCalculator{loader: nil, nowFn: time.Now}
	_, err := c.CalculateRefund(context.Background(), &dbent.PaymentOrder{}, false)
	if err == nil {
		t.Fatal("want error for nil loader")
	}
}
