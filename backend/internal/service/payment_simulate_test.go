//go:build unit

package service

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

// createSimulateTestBalanceOrder 造一笔「余额充值」订单：履约会走 doBalance，
// 而不是需要 groupRepo/subscriptionSvc 的订阅分支，便于在单测里跑通全链路。
func createSimulateTestBalanceOrder(
	t *testing.T,
	ctx context.Context,
	client *dbent.Client,
	status string,
) *dbent.PaymentOrder {
	t.Helper()

	order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, status, time.Now())
	order, err := client.PaymentOrder.UpdateOneID(order.ID).
		SetOrderType(payment.OrderTypeBalance).
		ClearPlanID().
		ClearSubscriptionGroupID().
		ClearSubscriptionDays().
		Save(ctx)
	require.NoError(t, err)
	return order
}

func TestSimulateOrderPaidRejectsInvalidOrderID(t *testing.T) {
	t.Parallel()

	svc := &PaymentService{}
	for _, orderID := range []int64{0, -1} {
		_, err := svc.SimulateOrderPaid(context.Background(), orderID)
		require.Error(t, err)
		require.Equal(t, "INVALID_ORDER_ID", infraerrors.Reason(err))
	}
}

func TestSimulateOrderPaidRejectsUnknownOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client}

	_, err := svc.SimulateOrderPaid(ctx, 987654321)
	require.Error(t, err)
	require.Equal(t, "ORDER_NOT_FOUND", infraerrors.Reason(err))
}

func TestSimulateOrderPaidRejectsNonPendingOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createSimulateTestBalanceOrder(t, ctx, client, OrderStatusPaid)
	svc := &PaymentService{entClient: client}

	_, err := svc.SimulateOrderPaid(ctx, order.ID)
	require.Error(t, err, "only pending orders may be simulated")
	require.Equal(t, "ORDER_NOT_PENDING", infraerrors.Reason(err))

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusPaid, reloaded.Status)

	audits, err := client.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10))).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, audits, "a rejected simulation must not write audit rows")
}

// TestSimulateOrderPaidCompletesPendingOrder 覆盖本轮模拟支付的完整链路：
// PENDING 订单 → 视为已支付（PAID）→ 履约 → COMPLETED，并且留下可追溯的审计轨迹。
func TestSimulateOrderPaidCompletesPendingOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	order := createSimulateTestBalanceOrder(t, ctx, client, OrderStatusPending)

	// 兑换码已存在且已被使用 → doBalance 会走幂等分支，只需标记完成即可。
	redeemRepo := &redeemCodeRepoStub{codesByCode: map[string]*RedeemCode{
		order.RechargeCode: {
			ID:     501,
			Code:   order.RechargeCode,
			Type:   RedeemTypeBalance,
			Value:  order.Amount,
			Status: StatusUsed,
		},
	}}
	svc := &PaymentService{
		entClient:     client,
		redeemService: &RedeemService{redeemRepo: redeemRepo},
	}

	tradeNo, err := svc.SimulateOrderPaid(ctx, order.ID)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(tradeNo, simulatedTradeNoPrefix),
		"simulated trades must be distinguishable from real provider trades, got %q", tradeNo)

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status)
	require.Equal(t, tradeNo, reloaded.PaymentTradeNo)
	require.Equal(t, order.PayAmount, reloaded.PayAmount)

	paidAudits, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("ORDER_PAID"),
		).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, paidAudits, "the shared fulfillment path must record ORDER_PAID")

	simulatedAudits, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("ORDER_PAID_SIMULATED"),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, simulatedAudits, 1, "the simulation must be recorded as an explicit admin action")
	require.Equal(t, "admin", simulatedAudits[0].Operator)

	// 已经完成的订单不能再次模拟：第二次调用必须被拒，也不能重复履约。
	_, err = svc.SimulateOrderPaid(ctx, order.ID)
	require.Error(t, err)
	require.Equal(t, "ORDER_NOT_PENDING", infraerrors.Reason(err))
	require.Empty(t, redeemRepo.useCalls, "a completed order must not be redeemed again")
}
