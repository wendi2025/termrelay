//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

// 回归：entRefundLoader 曾经用小写状态字面量（"completed" / "refunded"）与库中的
// UPPERCASE 枚举比较，导致两个判断恒为假：
//   - IsForceRefundEligible 恒 false → 方案 9.1 的 force_refund（平台故障/重复扣款）
//     补偿路径在预览与计算层彻底不可用；
//   - IsAlreadyRefunded 的状态分支是死代码 → 已退款订单可能被再次算出可退金额。
// 下面用真实 ent + sqlite 覆盖这两个判断。

func TestEntRefundLoader_IsForceRefundEligibleMatchesUppercaseStatuses(t *testing.T) {
	ctx := context.Background()
	loader := NewEntRefundLoader(newPaymentConfigServiceTestClient(t))

	eligible := []string{
		OrderStatusCompleted,
		OrderStatusRefundRequested,
		OrderStatusRefundPending,
		OrderStatusRefundFailed,
	}
	for _, status := range eligible {
		got, err := loader.IsForceRefundEligible(ctx, &dbent.PaymentOrder{Status: status})
		require.NoError(t, err)
		require.True(t, got, "status=%s 应允许 force_refund", status)
	}

	blocked := []string{
		OrderStatusPending,
		OrderStatusPaid,
		OrderStatusRecharging,
		OrderStatusExpired,
		OrderStatusCancelled,
		OrderStatusFailed,
		OrderStatusRefunding,
		OrderStatusRefunded,
		OrderStatusPartiallyRefunded,
	}
	for _, status := range blocked {
		got, err := loader.IsForceRefundEligible(ctx, &dbent.PaymentOrder{Status: status})
		require.NoError(t, err)
		require.False(t, got, "status=%s 不应允许 force_refund", status)
	}

	// 小写字面量（历史 bug 写法）必须不被接受，否则说明又回退到字符串比较
	got, err := loader.IsForceRefundEligible(ctx, &dbent.PaymentOrder{Status: "completed"})
	require.NoError(t, err)
	require.False(t, got)

	_, err = loader.IsForceRefundEligible(ctx, nil)
	require.ErrorIs(t, err, ErrNilOrder)
}

func TestEntRefundLoader_IsAlreadyRefundedDetectsRefundedStatuses(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	loader := NewEntRefundLoader(client)
	f := newRefundPreviewFixture(t, client, "loader-already-refunded@example.com", 100, 100, 100)

	// 正常已完成订单：未退款
	got, err := loader.IsAlreadyRefunded(ctx, f.Order)
	require.NoError(t, err)
	require.False(t, got)

	// 状态兜底：REFUNDED / PARTIALLY_REFUNDED（refund_at 可能为空）
	for _, status := range []string{OrderStatusRefunded, OrderStatusPartiallyRefunded} {
		order, err := client.PaymentOrder.UpdateOneID(f.Order.ID).SetStatus(status).Save(ctx)
		require.NoError(t, err)
		got, err := loader.IsAlreadyRefunded(ctx, order)
		require.NoError(t, err)
		require.True(t, got, "status=%s 视为已退款", status)
	}

	// refund_at 优先（生产上正是这条兜住了既有的重复退款）
	order, err := client.PaymentOrder.UpdateOneID(f.Order.ID).
		SetStatus(OrderStatusCompleted).
		SetRefundAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	got, err = loader.IsAlreadyRefunded(ctx, order)
	require.NoError(t, err)
	require.True(t, got)

	_, err = loader.IsAlreadyRefunded(ctx, nil)
	require.ErrorIs(t, err, ErrNilOrder)
}

// 端到端：真实订单 + 真实 loader，force 必须能算出全额退款
func TestCalculateRefund_ForceRefundWorksOnCompletedOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "loader-force@example.com", 100, 100, 100)
	calc := NewRefundCalculator(NewEntRefundLoader(client))

	quote, err := calc.CalculateRefund(ctx, f.Order, true)
	require.NoError(t, err)
	require.Equal(t, "force_refund", quote.Reason)
	require.True(t, quote.ForceRefundEligible)
	require.InDelta(t, 100, quote.RefundableAmount, 0.01)

	// 非可退状态（待支付）不得走 force
	pending, err := client.PaymentOrder.UpdateOneID(f.Order.ID).SetStatus(OrderStatusPending).Save(ctx)
	require.NoError(t, err)
	blocked, err := calc.CalculateRefund(ctx, pending, true)
	require.NoError(t, err)
	require.Equal(t, "force_refund_not_eligible", blocked.Reason)
	require.InDelta(t, 0, blocked.RefundableAmount, 0.01)
}

// 端到端：已退款状态即使落在 24h 窗口内也必须报 0，避免重复退款
func TestCalculateRefund_RefundedStatusReportsZero(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "loader-refunded-zero@example.com", 100, 100, 100)
	calc := NewRefundCalculator(NewEntRefundLoader(client))

	for _, status := range []string{OrderStatusRefunded, OrderStatusPartiallyRefunded} {
		order, err := client.PaymentOrder.UpdateOneID(f.Order.ID).SetStatus(status).Save(ctx)
		require.NoError(t, err)
		quote, err := calc.CalculateRefund(ctx, order, false)
		require.NoError(t, err)
		require.Equal(t, "already_refunded", quote.Reason, "status=%s", status)
		require.InDelta(t, 0, quote.RefundableAmount, 0.01)
	}
}
