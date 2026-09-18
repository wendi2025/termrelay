package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// simulatedTradeNoPrefix 标记由管理端手工模拟出来的交易号，便于在订单与审计日志里一眼区分。
const simulatedTradeNoPrefix = "SIMULATED-"

// SimulateOrderPaid 把一笔仍处于 PENDING 的订单直接置为已支付，全程不调用任何上游渠道。
//
// 用途：
//   - 真实商户凭据到位之前，验证「下单 → 支付 → 履约 → 余额/订阅到账」整条链路；
//   - 上游渠道已在场外确认收款、需要我们补记账。
//
// 安全约束：
//   - 仅供管理端调用（路由挂在 /admin/payment 分组下，带 admin 鉴权与审计中间件）。
//   - 订单必须仍为 PENDING；已支付、已取消、已过期的订单会被拒绝。
//   - 复用与真实回调完全相同的 HandlePaymentNotification 路径，因此渠道匹配、金额
//     校验、幂等（履约 lease）保护全部照常生效——这里只伪造了「上游说已付款」这一步。
func (s *PaymentService) SimulateOrderPaid(ctx context.Context, orderID int64) (string, error) {
	if orderID <= 0 {
		return "", infraerrors.BadRequest("INVALID_ORDER_ID", "invalid order id")
	}

	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return "", infraerrors.NotFound("ORDER_NOT_FOUND", "order not found")
		}
		return "", fmt.Errorf("load order: %w", err)
	}
	if order.Status != OrderStatusPending {
		return "", infraerrors.Conflict("ORDER_NOT_PENDING",
			"only pending orders can be simulated, current status: "+order.Status)
	}
	if strings.TrimSpace(order.OutTradeNo) == "" {
		return "", infraerrors.Conflict("ORDER_WITHOUT_TRADE_NO", "order has no out_trade_no")
	}

	// 与 confirmPayment 保持一致：优先取订单绑定的渠道实例，其次取订单快照/字段，
	// 最后才回退到注册表按支付方式解析。只有 providerKey 与订单自身一致时，
	// HandlePaymentNotification 的渠道匹配校验才会通过。
	instanceProviderKey := ""
	if inst, instErr := s.getOrderProviderInstance(ctx, order); instErr == nil && inst != nil {
		instanceProviderKey = inst.ProviderKey
	}
	providerKey := expectedNotificationProviderKeyForOrder(s.registry, order, instanceProviderKey)
	if strings.TrimSpace(providerKey) == "" {
		return "", infraerrors.Conflict("ORDER_PROVIDER_UNKNOWN", "order has no resolvable payment provider")
	}

	tradeNo := simulatedTradeNoPrefix + time.Now().UTC().Format("20060102T150405Z")
	notification := &payment.PaymentNotification{
		TradeNo:  tradeNo,
		OrderID:  order.OutTradeNo,
		Amount:   order.PayAmount,
		Status:   payment.NotificationStatusSuccess,
		RawData:  fmt.Sprintf("{\"simulated\":true,\"orderId\":%d}", order.ID),
		Metadata: map[string]string{},
	}
	if err := s.HandlePaymentNotification(ctx, notification, providerKey); err != nil {
		return "", err
	}

	// HandlePaymentNotification 自身会写 ORDER_PAID；这里再补一条来源审计，
	// 让「这笔是被人工模拟的」在时间轴上有据可查。
	s.writeAuditLog(ctx, order.ID, "ORDER_PAID_SIMULATED", "admin", map[string]any{
		"tradeNo":     tradeNo,
		"paidAmount":  order.PayAmount,
		"providerKey": providerKey,
		"note":        "marked paid by admin without upstream provider call",
	})

	return tradeNo, nil
}
