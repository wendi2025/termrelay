package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 退款预览阻断原因（稳定枚举，前端据此本地化文案）
const (
	refundBlockNotCompleted      = "not_completed"
	refundBlockUnsupportedType   = "unsupported_order_type"
	refundBlockUserRefundDisable = "user_refund_disabled"
	refundBlockRefundDisabled    = "refund_disabled"
	refundBlockAlreadyRefunded   = "already_refunded"
	refundBlockNothingToRefund   = "nothing_to_refund"
	refundBlockBalanceNotEnough  = "balance_not_enough"
	refundBlockRefundInProgress  = "refund_in_progress"
	refundBlockForceNotEligible  = "force_refund_not_eligible"
)

// RefundPreview 退款预览与审核记录（方案 9.1 / 9.2 / 9.3）
//
// 只读：不发起网关调用，也不写任何状态。金额口径：
//   - *PayAmount 字段为支付币种（订单实付口径）
//   - *Credit 字段为记账单位（user.balance 与 user_balance_ledger 口径）
type RefundPreview struct {
	OrderID     int64  `json:"order_id"`
	OutTradeNo  string `json:"out_trade_no"`
	OrderType   string `json:"order_type"`
	OrderStatus string `json:"order_status"`
	UserID      int64  `json:"user_id"`
	UserEmail   string `json:"user_email,omitempty"`
	UserName    string `json:"user_name,omitempty"`

	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	PayAmount float64 `json:"pay_amount"`

	// 购买时间 / 激活时间 / 申请时间（方案 9.3）
	PurchasedAt         time.Time  `json:"purchased_at"`
	ActivatedAt         *time.Time `json:"activated_at,omitempty"`
	RefundRequestedAt   *time.Time `json:"refund_requested_at,omitempty"`
	RefundRequestedBy   string     `json:"refund_requested_by,omitempty"`
	RefundRequestReason string     `json:"refund_request_reason,omitempty"`

	// 已用官方额度 / 已扣金额 / 赠送余额（方案 9.3）
	UsedUSD          float64 `json:"used_usd"`
	ChargedCredit    float64 `json:"charged_credit"`
	ChargedPayAmount float64 `json:"charged_pay_amount"`
	BonusBalance     float64 `json:"bonus_balance"`

	// 公式结果（方案 9.3）
	Policy               string             `json:"policy"`
	RefundablePayAmount  float64            `json:"refundable_pay_amount"`
	RefundableCredit     float64            `json:"refundable_credit"`
	FullRefundWindow     bool               `json:"full_refund_window"`
	ForceRefundEligible  bool               `json:"force_refund_eligible"`
	CalculationBreakdown map[string]float64 `json:"calculation_breakdown,omitempty"`

	// BalanceCap 余额封顶后的实际可退上限（记账单位），仅按量充值订单有值
	BalanceCap        *float64 `json:"balance_cap,omitempty"`
	BalanceCapApplied bool     `json:"balance_cap_applied"`

	Ledger        *LedgerOrderSummary `json:"ledger,omitempty"`
	LedgerEntries []LedgerEntryView   `json:"ledger_entries,omitempty"`

	Requestable   bool   `json:"requestable"`
	BlockedReason string `json:"blocked_reason,omitempty"`

	GeneratedAt time.Time `json:"generated_at"`
}

// PreviewRefund 生成订单退款预览。
//
// userID 大于 0 时按用户视角校验归属与用户退款开关；userID 为 0 表示管理员视角。
// 不可退款的原因通过 BlockedReason 返回而不是报错，便于界面直接展示口径。
func (s *PaymentService) PreviewRefund(ctx context.Context, orderID, userID int64, force bool) (*RefundPreview, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.InternalServer("SERVICE_UNAVAILABLE", "payment service unavailable")
	}
	o, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return nil, infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if userID > 0 && o.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission")
	}

	currency := PaymentOrderCurrency(o)
	preview := &RefundPreview{
		OrderID:             o.ID,
		OutTradeNo:          o.OutTradeNo,
		OrderType:           o.OrderType,
		OrderStatus:         o.Status,
		UserID:              o.UserID,
		Currency:            currency,
		Amount:              o.Amount,
		PayAmount:           o.PayAmount,
		PurchasedAt:         o.CreatedAt,
		ActivatedAt:         o.PaidAt,
		RefundRequestedAt:   o.RefundRequestedAt,
		RefundRequestedBy:   psStringValue(o.RefundRequestedBy),
		RefundRequestReason: psStringValue(o.RefundRequestReason),
		GeneratedAt:         time.Now(),
	}
	if userID == 0 {
		if u, userErr := s.entClient.User.Get(ctx, o.UserID); userErr == nil && u != nil {
			preview.UserEmail = u.Email
			preview.UserName = u.Username
		}
	}

	s.fillRefundPreviewLedger(ctx, preview)
	if err := s.fillRefundPreviewQuote(ctx, preview, o, currency, force); err != nil {
		return nil, err
	}
	preview.Requestable, preview.BlockedReason = s.refundPreviewGate(ctx, o, userID, force, preview)
	return preview, nil
}

// fillRefundPreviewLedger 填充账本汇总与分录（账本不可用时静默跳过）
func (s *PaymentService) fillRefundPreviewLedger(ctx context.Context, preview *RefundPreview) {
	if s == nil || s.balanceLedger == nil || preview == nil {
		return
	}
	if summary, err := s.balanceLedger.OrderSummary(ctx, preview.OrderID); err == nil {
		preview.Ledger = summary
		preview.BonusBalance = summary.BonusLeft
	}
	if entries, err := s.balanceLedger.ListOrderEntries(ctx, preview.OrderID, ledgerEntryDefaultLimit); err == nil {
		preview.LedgerEntries = entries
	}
}

// fillRefundPreviewQuote 填充退款公式报价与 9.3 口径金额
func (s *PaymentService) fillRefundPreviewQuote(ctx context.Context, preview *RefundPreview, o *dbent.PaymentOrder, currency string, force bool) error {
	if s == nil || s.refundCalculator == nil || preview == nil || o == nil {
		return nil
	}
	quote, err := s.refundCalculator.CalculateRefund(ctx, o, force)
	if err != nil {
		return err
	}
	if quote == nil {
		return nil
	}
	preview.Policy = quote.Reason
	preview.RefundablePayAmount = quote.RefundableAmount
	preview.RefundableCredit = calculateRefundCreditAmount(o.Amount, o.PayAmount, quote.RefundableAmount, currency)
	preview.FullRefundWindow = quote.FullRefundWindow
	preview.ForceRefundEligible = quote.ForceRefundEligible
	preview.CalculationBreakdown = quote.CalculationBreakdown
	preview.UsedUSD = quote.CalculationBreakdown["used_usd"]
	preview.ChargedPayAmount, preview.ChargedCredit = refundPreviewChargedAmounts(o, currency, quote)
	if o.OrderType == payment.OrderTypeBalance && !force {
		s.applyRefundPreviewBalanceCap(ctx, preview, o, currency)
	}
	return nil
}

// refundPreviewChargedAmounts 已扣金额（支付币种 + 记账单位）
func refundPreviewChargedAmounts(o *dbent.PaymentOrder, currency string, quote *RefundQuote) (payAmount, credit float64) {
	if o == nil || quote == nil {
		return 0, 0
	}
	switch o.OrderType {
	case payment.OrderTypeBalance:
		credit = quote.CalculationBreakdown["used_principal"]
		return refundPrincipalToPaidAmount(o, credit), credit
	case payment.OrderTypeSubscription:
		payAmount = quote.CalculationBreakdown["consumed"]
		return payAmount, calculateRefundCreditAmount(o.Amount, o.PayAmount, payAmount, currency)
	default:
		return 0, 0
	}
}

// applyRefundPreviewBalanceCap 按量充值订单：以用户当前余额封顶可退金额
func (s *PaymentService) applyRefundPreviewBalanceCap(ctx context.Context, preview *RefundPreview, o *dbent.PaymentOrder, currency string) {
	if s == nil || preview == nil || o == nil {
		return
	}
	capped, known, err := s.balanceRefundCapForOrder(ctx, o, preview.RefundableCredit, currency)
	if err != nil || !known {
		return
	}
	preview.BalanceCap = &capped
	if capped < preview.RefundableCredit {
		preview.BalanceCapApplied = true
		preview.RefundableCredit = capped
		// 与 PrepareRefund 保持一致：余额封顶后原路退款金额同步下调，
		// 否则界面会展示一个实际退不出去的报价。
		preview.RefundablePayAmount = calculateGatewayRefundAmount(o.Amount, o.PayAmount, capped, currency)
	}
}

// refundPreviewGate 判定是否可发起退款，并返回不可退款原因
func (s *PaymentService) refundPreviewGate(ctx context.Context, o *dbent.PaymentOrder, userID int64, force bool, preview *RefundPreview) (bool, string) {
	if o == nil {
		return false, refundBlockNotCompleted
	}
	if o.OrderType != payment.OrderTypeBalance && o.OrderType != payment.OrderTypeSubscription {
		return false, refundBlockUnsupportedType
	}
	switch o.Status {
	case OrderStatusRefunded, OrderStatusPartiallyRefunded:
		return false, refundBlockAlreadyRefunded
	case OrderStatusRefundRequested, OrderStatusRefunding, OrderStatusRefundPending:
		return false, refundBlockRefundInProgress
	case OrderStatusCompleted, OrderStatusRefundFailed:
	default:
		return false, refundBlockNotCompleted
	}
	if !s.refundPreviewProviderAllows(ctx, o, userID) {
		if userID > 0 {
			return false, refundBlockUserRefundDisable
		}
		return false, refundBlockRefundDisabled
	}
	if preview == nil {
		return true, ""
	}
	if preview.Policy == "already_refunded" {
		return false, refundBlockAlreadyRefunded
	}
	if force && !preview.ForceRefundEligible {
		return false, refundBlockForceNotEligible
	}
	if preview.RefundablePayAmount <= 0 || preview.RefundableCredit <= 0 {
		if preview.BalanceCapApplied {
			return false, refundBlockBalanceNotEnough
		}
		return false, refundBlockNothingToRefund
	}
	return true, ""
}

// refundPreviewProviderAllows 用户路径要求 provider 允许用户退款；管理员路径要求开启退款
func (s *PaymentService) refundPreviewProviderAllows(ctx context.Context, o *dbent.PaymentOrder, userID int64) bool {
	if s == nil || s.entClient == nil || o == nil {
		return false
	}
	inst, err := s.getRefundOrderProviderInstance(ctx, o)
	if err != nil || inst == nil {
		return false
	}
	if userID > 0 {
		return inst.AllowUserRefund
	}
	return inst.RefundEnabled
}

// BalanceLedgerBreakdown 返回用户余额的分账构成与近期账本分录（方案 9.2）
func (s *PaymentService) BalanceLedgerBreakdown(ctx context.Context, userID int64, limit int) (*LedgerBalanceBreakdown, []LedgerEntryView, error) {
	if s == nil || s.balanceLedger == nil {
		return &LedgerBalanceBreakdown{UserID: userID}, nil, nil
	}
	breakdown, err := s.balanceLedger.UserBalanceBreakdown(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.balanceLedger.ListUserEntries(ctx, userID, limit)
	if err != nil {
		return breakdown, nil, err
	}
	return breakdown, entries, nil
}
