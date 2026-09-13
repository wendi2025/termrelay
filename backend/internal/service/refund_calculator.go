package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// Smirel 退款双重结算公式（详见 套餐定价方案 9.1 / 9.2）
//
// 周期套餐（OrderTypeSubscription）：
//   可退金额 = max(0, 实付金额 - max(按时间应扣金额, 已用额度×按量价折算金额))
//   其中：
//     按时间应扣金额 = 实付金额 × 已使用周期天数 / 套餐周期总天数
//     已用额度×按量价 = sum(usage_log.actual_cost USD) × group.pay_as_you_go_price_per_usd
//
// 按量充值（OrderTypeBalance）：
//   可退金额 = max(0, 订单本金 - 该订单已实际扣费金额)
//   赠送余额不提现、不折现；退款时同步取消该订单赠送余额
//
// 24 小时全额退款规则（覆盖上述公式）：
//   未产生任何调用 + paid_at 之后 24h 内 → 原路全额退款（赠送额度同步撤销）
//   平台故障/重复扣款/错误计费 → 不受 24h 时限限制，由管理员走 force_refund

// RefundQuote 退款报价（纯计算结果，不发起任何网关/数据库写入）
type RefundQuote struct {
	OrderID              int64              `json:"order_id"`
	OrderType            string             `json:"order_type"`
	RefundableAmount     float64            `json:"refundable_amount"`
	Reason               string             `json:"reason"`
	FullRefundWindow     bool               `json:"full_refund_window"`
	ForceRefundEligible  bool               `json:"force_refund_eligible"`
	CalculationBreakdown map[string]float64 `json:"calculation_breakdown"`
	GeneratedAt          time.Time          `json:"generated_at"`
}

// RefundDataLoader 数据加载器接口（由调用方实现，从 ent 拉数据）
//   本接口把"计算逻辑"和"数据访问"解耦：
//   - 计算逻辑用纯函数（无 DB 依赖），易于单元测试
//   - 数据加载在生产代码里通过 ent client 实现，新表 UserBalanceLedger 在 ent 重新生成后接入
type RefundDataLoader interface {
	// HasUsageIn24hWindow 检查订单 paid_at 后 24h 内是否有任何消费
	HasUsageIn24hWindow(ctx context.Context, order *dbent.PaymentOrder) (bool, error)
	// LoadSubscriptionPeriodUsage 加载周期套餐已使用 USD 总额（仅在订阅期内）
	LoadSubscriptionPeriodUsage(ctx context.Context, order *dbent.PaymentOrder) (float64, error)
	// LoadBalanceOrderUsedPrincipal 加载按量充值订单的"已扣本金"金额
	//   返回该订单本金 credit - 该订单所有 debit（按 FIFO 归属到该订单的扣费）
	LoadBalanceOrderUsedPrincipal(ctx context.Context, order *dbent.PaymentOrder) (float64, error)
	// LoadBonusGrantForOrder 加载订单关联的赠送余额总额（退款时需同步撤销）
	LoadBonusGrantForOrder(ctx context.Context, order *dbent.PaymentOrder) (float64, error)
	// IsAlreadyRefunded 订单是否已经退款（避免重复退款）
	IsAlreadyRefunded(ctx context.Context, order *dbent.PaymentOrder) (bool, error)
	// IsForceRefundEligible 是否满足平台故障/重复扣款等 force_refund 条件
	IsForceRefundEligible(ctx context.Context, order *dbent.PaymentOrder) (bool, error)
	// GetSubscriptionPeriod 返回套餐周期起止（按 paid_at 与 plan.validity_days 推算）
	GetSubscriptionPeriod(ctx context.Context, order *dbent.PaymentOrder) (start time.Time, end time.Time, ok bool, err error)
	// GetGroupPayAsYouGoPrice 返回 group.pay_as_you_go_price_per_usd（CNY/USD）
	GetGroupPayAsYouGoPrice(ctx context.Context, order *dbent.PaymentOrder) (float64, error)
}

// ErrNilOrder 订单为 nil
var ErrNilOrder = errors.New("refund: order is nil")

// ErrUnknownOrderType 订单类型未知
var ErrUnknownOrderType = errors.New("refund: unknown order type")

// RefundCalculator 退款计算器（纯计算，无副作用）
type RefundCalculator struct {
	loader RefundDataLoader
	// nowFn 可注入用于测试
	nowFn func() time.Time
}

// NewRefundCalculator 创建退款计算器
func NewRefundCalculator(loader RefundDataLoader) *RefundCalculator {
	return &RefundCalculator{
		loader: loader,
		nowFn:  time.Now,
	}
}

// SetNowFunc 注入 now（仅用于测试）
func (c *RefundCalculator) SetNowFunc(fn func() time.Time) {
	c.nowFn = fn
}

// CalculateRefund 计算订单可退金额（不发起任何网关/DB 写入）
//
// 参数：
//   - order：必须是 paid 状态；其他状态返回 ErrRefundNotEligible
//   - force：管理员强制退款（绕过 24h 窗口与双重结算公式，按实付金额退款）
func (c *RefundCalculator) CalculateRefund(ctx context.Context, order *dbent.PaymentOrder, force bool) (*RefundQuote, error) {
	if order == nil {
		return nil, ErrNilOrder
	}
	if c.loader == nil {
		return nil, errors.New("refund: loader is nil")
	}

	quote := &RefundQuote{
		OrderID:              order.ID,
		CalculationBreakdown: map[string]float64{},
		GeneratedAt:          c.nowFn(),
	}

	// 1. 查订单类型
	quote.OrderType = order.OrderType

	// 2. 已退款订单：直接返回 0
	refunded, err := c.loader.IsAlreadyRefunded(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("check already refunded: %w", err)
	}
	if refunded {
		quote.Reason = "already_refunded"
		quote.RefundableAmount = 0
		return quote, nil
	}

	paidAmount := order.PayAmount

	// 3. force_refund 路径：跳过公式，按实付金额退
	if force {
		eligible, err := c.loader.IsForceRefundEligible(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("check force refund eligibility: %w", err)
		}
		if !eligible {
			quote.Reason = "force_refund_not_eligible"
			quote.RefundableAmount = 0
			return quote, nil
		}
		quote.ForceRefundEligible = true
		quote.RefundableAmount = paidAmount
		quote.CalculationBreakdown["paid_amount"] = paidAmount
		quote.Reason = "force_refund"
		return quote, nil
	}

	// 4. 24h 全额窗口：未消费 + 24h 内
	hasUsage, err := c.loader.HasUsageIn24hWindow(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("check 24h usage: %w", err)
	}
	if !hasUsage && c.within24h(safePaidAt(order.PaidAt)) {
		quote.FullRefundWindow = true
		quote.RefundableAmount = paidAmount
		quote.CalculationBreakdown["paid_amount"] = paidAmount
		quote.CalculationBreakdown["window_hours"] = 24
		quote.Reason = "full_refund_24h_window"
		return quote, nil
	}

	// 5. 按订单类型套用公式
	switch order.OrderType {
	case payment.OrderTypeSubscription:
		return c.calculateSubscriptionRefund(ctx, order, quote)
	case payment.OrderTypeBalance:
		return c.calculateBalanceRefund(ctx, order, quote)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownOrderType, order.OrderType)
	}
}

// safePaidAt 防止 *time.Time 为 nil
func safePaidAt(t *time.Time) time.Time { if t == nil { return time.Time{} }; return *t }

// within24h 订单 paid_at 距今是否在 24 小时内
func (c *RefundCalculator) within24h(paidAt time.Time) bool {
	if paidAt.IsZero() {
		return false
	}
	return c.nowFn().Sub(paidAt) <= 24*time.Hour
}

// calculateSubscriptionRefund 周期套餐双重结算
//   可退金额 = max(0, 实付 - max(按时间应扣, 已用 USD × 按量价))
func (c *RefundCalculator) calculateSubscriptionRefund(ctx context.Context, order *dbent.PaymentOrder, quote *RefundQuote) (*RefundQuote, error) {
	paidAmount := order.PayAmount
	quote.CalculationBreakdown["paid_amount"] = paidAmount

	// 已使用 USD（按订单 paid_at 之后至 now 的 usage_log.actual_cost 求和）
	usedUSD, err := c.loader.LoadSubscriptionPeriodUsage(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("load subscription usage: %w", err)
	}
	quote.CalculationBreakdown["used_usd"] = usedUSD

	// 按时间应扣金额
	timeBasedCharge := paidAmount
	start, end, ok, err := c.loader.GetSubscriptionPeriod(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("get subscription period: %w", err)
	}
	if ok && !end.IsZero() && end.After(start) {
		elapsed := c.nowFn().Sub(start)
		if elapsed < 0 {
			elapsed = 0
		}
		total := end.Sub(start)
		if total > 0 {
			timeBasedCharge = paidAmount * float64(elapsed) / float64(total)
		}
	}
	quote.CalculationBreakdown["time_based_charge"] = timeBasedCharge

	// 已用额度 × 按量价
	unitPrice, err := c.loader.GetGroupPayAsYouGoPrice(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("get pay-as-you-go price: %w", err)
	}
	usageBasedCharge := usedUSD * unitPrice
	quote.CalculationBreakdown["pay_as_you_go_price_per_usd"] = unitPrice
	quote.CalculationBreakdown["usage_based_charge"] = usageBasedCharge

	// 取较大者作为已消耗金额
	consumed := timeBasedCharge
	if usageBasedCharge > consumed {
		consumed = usageBasedCharge
	}
	quote.CalculationBreakdown["consumed"] = consumed

	refundable := paidAmount - consumed
	if refundable < 0 {
		refundable = 0
	}
	quote.RefundableAmount = refundable
	quote.Reason = "subscription_double_settlement"
	return quote, nil
}

// calculateBalanceRefund 按量充值退款
//   可退金额 = max(0, 订单本金 - 已扣本金)
//   赠送余额退款时由调用方同步撤销（不在本计算结果中体现金额）
func (c *RefundCalculator) calculateBalanceRefund(ctx context.Context, order *dbent.PaymentOrder, quote *RefundQuote) (*RefundQuote, error) {
	paidAmount := order.PayAmount
	quote.CalculationBreakdown["paid_amount"] = paidAmount

	usedPrincipal, err := c.loader.LoadBalanceOrderUsedPrincipal(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("load balance used principal: %w", err)
	}
	quote.CalculationBreakdown["used_principal"] = usedPrincipal

	refundable := paidAmount - usedPrincipal
	if refundable < 0 {
		refundable = 0
	}
	quote.RefundableAmount = refundable
	quote.Reason = "balance_principal_minus_used"
	return quote, nil
}



