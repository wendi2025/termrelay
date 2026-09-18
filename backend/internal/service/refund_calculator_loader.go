package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	entusage "github.com/Wei-Shaw/sub2api/ent/usagelog"
	"github.com/Wei-Shaw/sub2api/ent/userbalanceledger"
	"github.com/Wei-Shaw/sub2api/internal/payment"
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

// LoadSubscriptionPeriodUsage 加载周期套餐已使用官方 $ 额度
//
// 口径（方案 9.1）：以平台最终结算记录为准 —— 取该订单周期内
// （paid_at → paid_at + subscription_days）该分组下 usage_log.actual_cost 之和。
// 与 user_subscriptions.monthly_usage_usd 相比，按订单周期取数可避免
// 续费/跨周期时把新周期的用量算进旧订单。
func (l *entRefundLoader) LoadSubscriptionPeriodUsage(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil {
		return 0, ErrNilOrder
	}
	if order.SubscriptionGroupID == nil {
		return 0, nil
	}
	from := safePaidAt(order.PaidAt)
	if from.IsZero() {
		return 0, nil
	}
	query := l.client.UsageLog.Query().
		Where(entusage.UserIDEQ(order.UserID)).
		Where(entusage.GroupIDEQ(*order.SubscriptionGroupID)).
		Where(entusage.CreatedAtGTE(from))
	if order.SubscriptionDays != nil && *order.SubscriptionDays > 0 {
		query = query.Where(entusage.CreatedAtLT(from.Add(time.Duration(*order.SubscriptionDays) * 24 * time.Hour)))
	}
	logs, err := query.Select(entusage.FieldActualCost).All(ctx)
	if err != nil {
		return 0, fmt.Errorf("query subscription usage: %w", err)
	}
	var total float64
	for _, entry := range logs {
		total += entry.ActualCost
	}
	return total, nil
}

// LoadBalanceOrderUsedPrincipal 加载按量充值订单的"已扣本金"金额（记账单位）
//
//	算法：sum(principal credit) - sum(principal debit)，按 order_id 归属。
//	只统计本金分录：赠送余额不参与本金退款计算。
//	不过滤 frozen —— frozen 表示该订单账本已在退款完成时关闭，
//	而本方法用于"退款前计算"，必须看到全部历史分录。
func (l *entRefundLoader) LoadBalanceOrderUsedPrincipal(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil {
		return 0, ErrNilOrder
	}
	if order.OrderType != payment.OrderTypeBalance {
		return 0, nil
	}
	entries, err := l.client.UserBalanceLedger.Query().
		Where(userbalanceledger.OrderIDEQ(order.ID)).
		Where(userbalanceledger.EntryTypeEQ(ledgerEntryTypePrincipal)).
		Where(userbalanceledger.DirectionEQ(ledgerDirectionDebit)).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("query ledger: %w", err)
	}
	// debit 分录的 amount 即本次扣费按 FIFO 归属到该订单的消耗金额，直接求和即为"已扣本金"。
	var used float64
	for _, e := range entries {
		used += e.Amount
	}
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
	// 订单状态在库里是 UPPERCASE 枚举（payment.OrderStatus*），
	// 这里必须用常量比较：曾经写成小写字面量导致该分支永远不生效。
	switch order.Status {
	case OrderStatusRefunded, OrderStatusPartiallyRefunded:
		return true, nil
	}
	return false, nil
}

// IsForceRefundEligible 是否满足 force_refund 条件
func (l *entRefundLoader) IsForceRefundEligible(ctx context.Context, order *dbent.PaymentOrder) (bool, error) {
	if order == nil {
		return false, ErrNilOrder
	}
	// 同上：必须用 UPPERCASE 状态常量。小写字面量会让 force_refund
	// 恒返回 not_eligible，使"平台故障/重复扣款"补偿路径彻底不可用。
	switch order.Status {
	case OrderStatusCompleted, OrderStatusRefundRequested, OrderStatusRefundPending, OrderStatusRefundFailed:
		return true, nil
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
