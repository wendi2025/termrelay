//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

// newRefundPreviewTestService 组装退款预览/账本读取所需的最小 PaymentService。
// 预览路径是纯读，不需要 registry/loadBalancer 等外部依赖。
func newRefundPreviewTestService(client *dbent.Client, userRepo UserRepository) *PaymentService {
	return &PaymentService{
		entClient:        client,
		balanceLedger:    NewBalanceLedgerService(client),
		refundCalculator: NewRefundCalculator(NewEntRefundLoader(client)),
		userRepo:         userRepo,
	}
}

// refundPreviewFixture 退款预览测试夹具：已完成按量充值订单 + 允许退款的渠道实例
type refundPreviewFixture struct {
	Order *dbent.PaymentOrder
	User  *User
	Inst  *dbent.PaymentProviderInstance
}

func newRefundPreviewFixture(t *testing.T, client *dbent.Client, email string, amount, payAmount, balance float64) *refundPreviewFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetUsername(email).
		SetBalance(balance).
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("refund-preview-" + email).
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	// ProviderInstanceID 必须显式写入：退款预览靠订单快照定位渠道实例，
	// 缺失时会被判为"渠道不支持退款"。
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetProviderInstanceID(strconv.FormatInt(inst.ID, 10)).
		SetAmount(amount).
		SetPayAmount(payAmount).
		SetFeeRate(0).
		SetRechargeCode("REFUND-PREVIEW-" + email).
		SetOutTradeNo("sub2_refund_preview_" + email).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-refund-preview-" + email).
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(now.Add(time.Hour)).
		SetPaidAt(now).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	return &refundPreviewFixture{
		Order: order,
		User:  &User{ID: user.ID, Balance: balance},
		Inst:  inst,
	}
}

// ageRefundPreviewOrder 把订单付款时间前移，使其脱离 24h 全额退款窗口，
// 从而走到按量充值的"本金 - 已扣本金"公式路径。
func ageRefundPreviewOrder(t *testing.T, client *dbent.Client, orderID int64, age time.Duration) {
	t.Helper()
	_, err := client.PaymentOrder.UpdateOneID(orderID).SetPaidAt(time.Now().Add(-age)).Save(context.Background())
	require.NoError(t, err)
}

// addLedgerDebit 手工补一笔该订单的本金扣费分录（模拟 FIFO 归属扣费）
func addLedgerDebit(t *testing.T, client *dbent.Client, orderID, userID int64, amount, balanceAfter float64) {
	t.Helper()
	_, err := client.UserBalanceLedger.Create().
		SetUserID(userID).
		SetOrderID(orderID).
		SetEntryType(ledgerEntryTypePrincipal).
		SetDirection(ledgerDirectionDebit).
		SetAmount(amount).
		SetBalanceAfter(balanceAfter).
		SetMemo("usage billing").
		Save(context.Background())
	require.NoError(t, err)
}

// --- 账本只读视图（方案 9.2 / 9.3 数据来源） ---

func TestBalanceLedgerReads_BreakdownAndEntries(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := NewBalanceLedgerService(client)
	order := newBalanceLedgerTestOrder(t, client, "preview-ledger@example.com", 100, 100)

	require.NoError(t, svc.RecordRechargeCredit(ctx, order))
	require.NoError(t, svc.RecordBonusCredit(ctx, order, 20, "promo"))

	// 账本只记录构成，余额事实来源是 user.balance。把余额对齐到本金 100 + 赠送 20，
	// 用来验证 LedgerGap 会把"账本未覆盖的历史余额"显式暴露出来。
	_, err := client.User.UpdateOneID(order.UserID).SetBalance(120).Save(ctx)
	require.NoError(t, err)

	breakdown, err := svc.UserBalanceBreakdown(ctx, order.UserID)
	require.NoError(t, err)
	require.InDelta(t, 120, breakdown.AccountBalance, 0.01)
	require.InDelta(t, 100, breakdown.PrincipalCreditTotal, 0.01)
	require.InDelta(t, 20, breakdown.BonusCreditTotal, 0.01)
	require.InDelta(t, 100, breakdown.PrincipalLeft, 0.01)
	require.InDelta(t, 20, breakdown.BonusLeft, 0.01)
	require.InDelta(t, 100, breakdown.ActivePrincipalLeft, 0.01)
	require.InDelta(t, 20, breakdown.ActiveBonusLeft, 0.01)
	require.InDelta(t, 0, breakdown.FrozenPrincipalLeft, 0.01)
	require.InDelta(t, 0, breakdown.FrozenBonusLeft, 0.01)
	require.InDelta(t, 0, breakdown.LedgerGap, 0.01)
	require.Equal(t, 2, breakdown.EntryCount)

	entries, err := svc.ListOrderEntries(ctx, order.ID, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, ledgerEntryTypeBonus, entries[0].EntryType)
	require.InDelta(t, 20, entries[0].Amount, 0.01)
	require.Equal(t, ledgerEntryTypePrincipal, entries[1].EntryType)
	require.InDelta(t, 100, entries[1].Amount, 0.01)
	require.False(t, entries[0].Frozen)
	require.Equal(t, order.ID, entries[0].OrderID)
	require.NotEmpty(t, entries[0].Memo)

	// 退款完成 → 冻结订单账本：剩余额度退出可扣费构成
	require.NoError(t, svc.SettleOrderLedger(ctx, order.ID, "refund:1:preview"))
	settled, err := svc.UserBalanceBreakdown(ctx, order.UserID)
	require.NoError(t, err)
	require.InDelta(t, 100, settled.FrozenPrincipalLeft, 0.01)
	require.InDelta(t, 20, settled.FrozenBonusLeft, 0.01)
	require.InDelta(t, 0, settled.ActivePrincipalLeft, 0.01)
	require.InDelta(t, 0, settled.ActiveBonusLeft, 0.01)

	frozenEntries, err := svc.ListOrderEntries(ctx, order.ID, 10)
	require.NoError(t, err)
	require.Len(t, frozenEntries, 2)
	for _, e := range frozenEntries {
		require.True(t, e.Frozen)
		require.Equal(t, "refund:1:preview", e.RefundBatchID)
	}

	// 用户维度分录：limit 截断 + limit<=0 回落默认条数
	limited, err := svc.ListUserEntries(ctx, order.UserID, 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
	all, err := svc.ListUserEntries(ctx, order.UserID, 0)
	require.NoError(t, err)
	require.Len(t, all, 2)

	// 非法/空输入降级：不 panic，返回空
	empty, err := svc.ListOrderEntries(ctx, 0, 10)
	require.NoError(t, err)
	require.Empty(t, empty)
	nilSvc := (*BalanceLedgerService)(nil)
	if _, err := nilSvc.UserBalanceBreakdown(ctx, order.UserID); err != nil {
		t.Fatalf("nil service must degrade gracefully, got %v", err)
	}
	if got, err := nilSvc.ListUserEntries(ctx, order.UserID, 10); err != nil || got != nil {
		t.Fatalf("nil service must return nil entries, got %v err=%v", got, err)
	}
}

// --- 退款预览：按量充值余额封顶（方案 9.2） ---

func TestPreviewRefund_BalanceOrderCapsByUserBalance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-cap@example.com", 100, 100, 30)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	preview, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.True(t, preview.Requestable)
	require.Empty(t, preview.BlockedReason)
	require.InDelta(t, 30, preview.RefundablePayAmount, 0.01)
	require.InDelta(t, 30, preview.RefundableCredit, 0.01)
	require.NotNil(t, preview.BalanceCap)
	require.InDelta(t, 30, *preview.BalanceCap, 0.01)
	require.True(t, preview.BalanceCapApplied)
	// 9.3 审核记录字段：管理员视角回填账户信息
	require.Equal(t, "preview-cap@example.com", preview.UserEmail)
	require.Equal(t, "preview-cap@example.com", preview.UserName)
	require.Equal(t, f.Order.OutTradeNo, preview.OutTradeNo)
	require.Equal(t, payment.OrderTypeBalance, preview.OrderType)
	require.Equal(t, OrderStatusCompleted, preview.OrderStatus)
	require.False(t, preview.PurchasedAt.IsZero())
	require.NotNil(t, preview.ActivatedAt)
	require.False(t, preview.GeneratedAt.IsZero())
}

func TestPreviewRefund_ForceBypassesBalanceCap(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-force@example.com", 100, 100, 30)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	preview, err := svc.PreviewRefund(ctx, f.Order.ID, 0, true)
	require.NoError(t, err)
	require.True(t, preview.Requestable, "blocked=%q policy=%q force=%v refundable=%v/%v", preview.BlockedReason, preview.Policy, preview.ForceRefundEligible, preview.RefundablePayAmount, preview.RefundableCredit)
	require.Equal(t, "force_refund", preview.Policy)
	require.True(t, preview.ForceRefundEligible)
	require.InDelta(t, 100, preview.RefundablePayAmount, 0.01)
	require.Nil(t, preview.BalanceCap)
	require.False(t, preview.BalanceCapApplied)
	require.Equal(t, "preview-force@example.com", preview.UserEmail)
}

func TestPreviewRefund_UserViewOmitsAccountEmail(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-user@example.com", 100, 100, 100)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	preview, err := svc.PreviewRefund(ctx, f.Order.ID, f.Order.UserID, false)
	require.NoError(t, err)
	require.True(t, preview.Requestable)
	// 用户视角不回填账户邮箱/用户名（订单已归属该用户，无需泄露）
	require.Empty(t, preview.UserEmail)
	require.Empty(t, preview.UserName)
	require.InDelta(t, 100, preview.RefundablePayAmount, 0.01)
}

func TestPreviewRefund_RejectsOtherUserOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-foreign@example.com", 100, 100, 100)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	_, err := svc.PreviewRefund(ctx, f.Order.ID, f.Order.UserID+1000, false)
	require.Error(t, err)
}

func TestPreviewRefund_OrderNotFound(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := newRefundPreviewTestService(client, nil)

	_, err := svc.PreviewRefund(ctx, 999999, 0, false)
	require.Error(t, err)
}

func TestPreviewRefund_ServiceUnavailable(t *testing.T) {
	_, err := (&PaymentService{}).PreviewRefund(context.Background(), 1, 0, false)
	require.Error(t, err)
}

// --- 退款预览：阻断原因 ---

func TestPreviewRefund_BlocksWhenUserRefundDisabled(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-disabled@example.com", 100, 100, 100)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	_, err := client.PaymentProviderInstance.UpdateOneID(f.Inst.ID).SetAllowUserRefund(false).Save(ctx)
	require.NoError(t, err)

	userView, err := svc.PreviewRefund(ctx, f.Order.ID, f.Order.UserID, false)
	require.NoError(t, err)
	require.False(t, userView.Requestable)
	require.Equal(t, refundBlockUserRefundDisable, userView.BlockedReason)

	// 渠道仍开启退款 → 管理员仍可代用户退款
	adminView, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.True(t, adminView.Requestable)

	_, err = client.PaymentProviderInstance.UpdateOneID(f.Inst.ID).SetRefundEnabled(false).Save(ctx)
	require.NoError(t, err)

	disabled, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.False(t, disabled.Requestable)
	require.Equal(t, refundBlockRefundDisabled, disabled.BlockedReason)
}

func TestPreviewRefund_BlocksWhenRefundInProgress(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-inprogress@example.com", 100, 100, 100)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	_, err := client.PaymentOrder.UpdateOneID(f.Order.ID).
		SetStatus(OrderStatusRefundRequested).
		SetRefundRequestedAt(time.Now()).
		SetRefundRequestReason("买错了").
		Save(ctx)
	require.NoError(t, err)

	preview, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.False(t, preview.Requestable)
	require.Equal(t, refundBlockRefundInProgress, preview.BlockedReason)
	require.Equal(t, OrderStatusRefundRequested, preview.OrderStatus)
	require.NotNil(t, preview.RefundRequestedAt)
	require.Equal(t, "买错了", preview.RefundRequestReason)
}

func TestPreviewRefund_BlocksUnpaidOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-pending@example.com", 100, 100, 100)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	_, err := client.PaymentOrder.UpdateOneID(f.Order.ID).SetStatus(OrderStatusPending).Save(ctx)
	require.NoError(t, err)

	preview, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.False(t, preview.Requestable)
	require.Equal(t, refundBlockNotCompleted, preview.BlockedReason)
}

// 回归：按量充值必须退"剩余本金"（订单本金 - 已扣本金）。
// 曾经的实现把"剩余本金"当成"已扣本金"返回，导致未消费退 0、全额消费反而退全款。
func TestPreviewRefund_BalanceOrderRefundsRemainingPrincipal(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-principal@example.com", 100, 100, 100)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})
	ageRefundPreviewOrder(t, client, f.Order.ID, 72*time.Hour)

	untouched, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.Equal(t, "balance_principal_minus_used", untouched.Policy)
	require.True(t, untouched.Requestable)
	require.InDelta(t, 0, untouched.ChargedCredit, 0.01)
	require.InDelta(t, 100, untouched.RefundableCredit, 0.01)
	require.InDelta(t, 100, untouched.RefundablePayAmount, 0.01)
	require.InDelta(t, 100, untouched.CalculationBreakdown["order_principal"], 0.01)
	require.InDelta(t, 0, untouched.CalculationBreakdown["used_principal"], 0.01)

	// 已扣本金 40 → 只剩 60 可退
	addLedgerDebit(t, client, f.Order.ID, f.Order.UserID, 40, 60)
	partial, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.True(t, partial.Requestable)
	require.InDelta(t, 40, partial.ChargedCredit, 0.01)
	require.InDelta(t, 40, partial.ChargedPayAmount, 0.01)
	require.InDelta(t, 60, partial.RefundableCredit, 0.01)
	require.InDelta(t, 60, partial.RefundablePayAmount, 0.01)
	require.False(t, partial.BalanceCapApplied)
	require.InDelta(t, 40, partial.CalculationBreakdown["used_principal"], 0.01)

	// 本金全部消耗 → 无可退金额
	addLedgerDebit(t, client, f.Order.ID, f.Order.UserID, 60, 0)
	consumed, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.False(t, consumed.Requestable)
	require.Equal(t, refundBlockNothingToRefund, consumed.BlockedReason)
	require.InDelta(t, 100, consumed.ChargedCredit, 0.01)
	require.InDelta(t, 0, consumed.RefundableCredit, 0.01)
	require.InDelta(t, 0, consumed.RefundablePayAmount, 0.01)
}

// 余额为 0（额度已全部消费并扣光）→ 走"余额不足"而不是"无可退金额"
func TestPreviewRefund_BlocksWhenBalanceDrained(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	f := newRefundPreviewFixture(t, client, "preview-drained@example.com", 100, 100, 0)
	svc := newRefundPreviewTestService(client, &userRepoStub{user: f.User})

	preview, err := svc.PreviewRefund(ctx, f.Order.ID, 0, false)
	require.NoError(t, err)
	require.False(t, preview.Requestable)
	require.Equal(t, refundBlockBalanceNotEnough, preview.BlockedReason)
	require.True(t, preview.BalanceCapApplied)
	require.InDelta(t, 0, preview.RefundableCredit, 0.01)
	require.InDelta(t, 0, preview.RefundablePayAmount, 0.01)
}

// --- PaymentService 账本读数封装 ---

func TestPaymentServiceBalanceLedgerBreakdown_WithoutLedgerService(t *testing.T) {
	breakdown, entries, err := (&PaymentService{}).BalanceLedgerBreakdown(context.Background(), 7, 10)
	require.NoError(t, err)
	require.NotNil(t, breakdown)
	require.Equal(t, int64(7), breakdown.UserID)
	require.Nil(t, entries)
}

func TestPaymentServiceBalanceLedgerBreakdown_ReturnsLedgerData(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := newBalanceLedgerTestOrder(t, client, "preview-svc-ledger@example.com", 50, 50)
	svc := newRefundPreviewTestService(client, nil)
	require.NoError(t, svc.balanceLedger.RecordRechargeCredit(ctx, order))

	breakdown, entries, err := svc.BalanceLedgerBreakdown(ctx, order.UserID, 10)
	require.NoError(t, err)
	require.InDelta(t, 50, breakdown.PrincipalLeft, 0.01)
	require.Equal(t, 1, breakdown.EntryCount)
	require.Len(t, entries, 1)
	require.Equal(t, order.ID, entries[0].OrderID)
	require.InDelta(t, 50, entries[0].Amount, 0.01)
}
