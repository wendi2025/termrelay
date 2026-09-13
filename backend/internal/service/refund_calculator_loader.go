package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/userbalanceledger"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	entusage "github.com/Wei-Shaw/sub2api/ent/usagelog"
)

// entRefundLoader 退款计算器的 ent 数据加载器实现
//
// 本结构体把 RefundCalculator 所需的全部数据访问封装到 ent client 上。
// 所有方法均为只读，不发起任何写操作。
type entRefundLoader struct {
	client *dbent.Client
	nowFn  func() time.Time
}

// NewEntRefundLoader 创建 ent 数据加载器
func NewEntRefundLoader(client *dbent.Client) *entRefundLoader {
	if client == nil {
		panic("NewEntRefundLoader: client is nil")
	}
	return &entRefundLoader{client: client, nowFn: time.Now}
}

// SetNowFunc 注入 now（仅用于测试）
func (l *entRefundLoader) SetNowFunc(fn func() time.Time) { l.nowFn = fn }

// HasUsageIn24hWindow 订单 paid_at 后 24h 内是否有任何消费
func (l *entRefundLoader) HasUsageIn24hWindow(ctx context.Context, order *dbent.PaymentOrder) (bool, error) {
	if order == nil || order.PaidAt == nil {
		return false, nil
	}
	from := *order.PaidAt
	from = from.Add(-time.Minute) // 边界 race 保护
	n, err := l.client.UsageLog.Query().
		Where(entusage.UserIDEQ(order.UserID)).
		Where(entusage.CreatedAtGTE(from)).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("query usage 24h: %w", err)
	}
	return n > 0, nil
}

// LoadSubscriptionPeriodUsage 加载周期套餐已使用 USD 总额
func (l *entRefundLoader) LoadSubscriptionPeriodUsage(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil {
		return 0, ErrNilOrder
	}
	if order.SubscriptionGroupID == nil {
		return 0, nil
	}
	subs, err := l.client.UserSubscription.Query().
		Where(usersubscription.UserIDEQ(order.UserID)).
		Where(usersubscription.GroupIDEQ(*order.SubscriptionGroupID)).
		Where(usersubscription.StatusEQ("active")).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("query subscription: %w", err)
	}
	var total float64
	for _, s := range subs {
		total += s.MonthlyUsageUsd
	}
	return total, nil
}

// LoadBalanceOrderUsedPrincipal 加载按量充值订单的"已扣本金"金额
//   算法：sum(credit) - sum(debit) where order_id=该订单, entry_type=principal
//   简化版：使用 All() + 累加；性能后续可用 SQL 聚合优化
func (l *entRefundLoader) LoadBalanceOrderUsedPrincipal(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil {
		return 0, ErrNilOrder
	}
	if order.OrderType != "balance" {
		return 0, nil
	}
	entries, err := l.client.UserBalanceLedger.Query().
		Where(userbalanceledger.OrderIDEQ(order.ID)).
		Where(userbalanceledger.FrozenEQ(false)).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("query ledger: %w", err)
	}
	var credits, debits float64
	for _, e := range entries {
		switch {
		case e.EntryType == "principal" && e.Direction == "credit":
			credits += e.Amount
		case e.Direction == "debit":
			debits += e.Amount
		}
	}
	used := credits - debits
	if used < 0 {
		used = 0
	}
	return used, nil
}

// LoadBonusGrantForOrder 加载订单关联的赠送余额总额
func (l *entRefundLoader) LoadBonusGrantForOrder(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil {
		return 0, ErrNilOrder
	}
	entries, err := l.client.UserBalanceLedger.Query().
		Where(userbalanceledger.OrderIDEQ(order.ID)).
		Where(userbalanceledger.EntryTypeEQ("bonus")).
		Where(userbalanceledger.DirectionEQ("credit")).
		Where(userbalanceledger.FrozenEQ(false)).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("query bonus: %w", err)
	}
	var total float64
	for _, e := range entries {
		total += e.Amount
	}
	return total, nil
}

// IsAlreadyRefunded 订单是否已经退款
func (l *entRefundLoader) IsAlreadyRefunded(ctx context.Context, order *dbent.PaymentOrder) (bool, error) {
	if order == nil {
		return false, ErrNilOrder
	}
	if order.RefundAt != nil && !order.RefundAt.IsZero() {
		return true, nil
	}
	if order.Status == "refunded" || order.Status == "refunded_completed" {
		return true, nil
	}
	return false, nil
}

// IsForceRefundEligible 是否满足 force_refund 条件
func (l *entRefundLoader) IsForceRefundEligible(ctx context.Context, order *dbent.PaymentOrder) (bool, error) {
	if order == nil {
		return false, ErrNilOrder
	}
	ok := []string{"completed", "refund_requested", "refund_pending", "refund_failed"}
	for _, s := range ok {
		if order.Status == s {
			return true, nil
		}
	}
	return false, nil
}

// GetSubscriptionPeriod 返回套餐周期起止
func (l *entRefundLoader) GetSubscriptionPeriod(ctx context.Context, order *dbent.PaymentOrder) (time.Time, time.Time, bool, error) {
	if order == nil {
		return time.Time{}, time.Time{}, false, ErrNilOrder
	}
	if order.PaidAt == nil || order.SubscriptionDays == nil {
		return time.Time{}, time.Time{}, false, nil
	}
	start := *order.PaidAt
	end := start.Add(time.Duration(*order.SubscriptionDays) * 24 * time.Hour)
	return start, end, true, nil
}

// GetGroupPayAsYouGoPrice 返回 group.pay_as_you_go_price_per_usd（CNY/USD）
func (l *entRefundLoader) GetGroupPayAsYouGoPrice(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil {
		return 0, ErrNilOrder
	}
	if order.SubscriptionGroupID == nil {
		return 0, errors.New("order has no subscription_group_id")
	}
	g, err := l.client.Group.Get(ctx, *order.SubscriptionGroupID)
	if err != nil {
		return 0, fmt.Errorf("get group: %w", err)
	}
	return g.PayAsYouGoPricePerUsd, nil
}

// 编译期接口断言
var _ RefundDataLoader = (*entRefundLoader)(nil)

// 防止 paymentorder / entusage 未使用告警
var (
	_ = paymentorder.IDEQ
	_ = entusage.UserIDEQ
)