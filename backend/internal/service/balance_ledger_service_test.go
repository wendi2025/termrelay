//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"
)

// newBalanceLedgerTestClient 建立 sqlite 内存库的 ent client，用于验证账本服务
func newBalanceLedgerTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:balance_ledger_service?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func newBalanceLedgerTestOrder(t *testing.T, client *dbent.Client, email string, amount, payAmount float64) *dbent.PaymentOrder {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetUsername(email).
		SetBalance(amount).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(email).
		SetUserName(email).
		SetRechargeCode("RC-" + email).
		SetAmount(amount).
		SetPayAmount(payAmount).
		SetOrderType("balance").
		SetPaymentType("alipay").
		SetPaymentTradeNo("TRADE-" + email).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(now.Add(30 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("test.local").
		Save(ctx)
	require.NoError(t, err)
	return order
}

func TestBalanceLedgerService_RecordRechargeCreditIsIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newBalanceLedgerTestClient(t)
	svc := NewBalanceLedgerService(client)
	order := newBalanceLedgerTestOrder(t, client, "ledger-idem@example.com", 100, 100)

	require.NoError(t, svc.RecordRechargeCredit(ctx, order))
	require.NoError(t, svc.RecordRechargeCredit(ctx, order))

	summary, err := svc.OrderSummary(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, 1, summary.EntryCount)
	require.InDelta(t, 100, summary.PrincipalCredit, 0.000001)
	require.InDelta(t, 0, summary.PrincipalDebit, 0.000001)
	require.InDelta(t, 100, summary.PrincipalLeft, 0.000001)
	require.False(t, summary.Frozen)
}

func TestBalanceLedgerService_SettleOrderLedgerFreezesOrder(t *testing.T) {
	ctx := context.Background()
	client := newBalanceLedgerTestClient(t)
	svc := NewBalanceLedgerService(client)
	order := newBalanceLedgerTestOrder(t, client, "ledger-settle@example.com", 200, 100)

	require.NoError(t, svc.RecordRechargeCredit(ctx, order))
	require.NoError(t, svc.RecordBonusCredit(ctx, order, 20, "promo"))
	require.NoError(t, svc.SettleOrderLedger(ctx, order.ID, "refund:1:123"))

	summary, err := svc.OrderSummary(ctx, order.ID)
	require.NoError(t, err)
	require.True(t, summary.Frozen)
	require.InDelta(t, 200, summary.PrincipalCredit, 0.000001)
	require.InDelta(t, 20, summary.BonusCredit, 0.000001)
	require.InDelta(t, 20, summary.BonusLeft, 0.000001)

	entries, err := client.UserBalanceLedger.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, e := range entries {
		require.True(t, e.Frozen)
		require.NotNil(t, e.RefundBatchID)
		require.Equal(t, "refund:1:123", *e.RefundBatchID)
	}
}

func TestBalanceLedgerService_SkipsNonPositiveAmounts(t *testing.T) {
	ctx := context.Background()
	client := newBalanceLedgerTestClient(t)
	svc := NewBalanceLedgerService(client)
	order := newBalanceLedgerTestOrder(t, client, "ledger-zero@example.com", 0, 100)

	require.NoError(t, svc.RecordRechargeCredit(ctx, order))
	require.NoError(t, svc.RecordBonusCredit(ctx, order, 0, "promo"))

	summary, err := svc.OrderSummary(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, 0, summary.EntryCount)
}
