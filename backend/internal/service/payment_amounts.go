package service

import (
	"math"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 1.0

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

// normalizeSubscriptionUSDToCNYRate 将非法值归一为 0（换算关闭）。
// 与余额倍率不同，0 是合法状态：表示订阅保持 price 直付的存量行为。
func normalizeSubscriptionUSDToCNYRate(rate float64) float64 {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 {
		return 0
	}
	return rate
}

func calculateCreditedBalance(paymentAmount, multiplier float64) float64 {
	return decimal.NewFromFloat(paymentAmount).
		Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).
		Round(2).
		InexactFloat64()
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(payAmount).Round(fractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(fractionDigits).
		InexactFloat64()
}

// calculateRefundCreditAmount 与 calculateGatewayRefundAmount 互为逆运算：
// 把"支付币种（实付）"口径的退款金额折算回订单记账单位金额（余额/账本口径）。
//   - 全额退款（payRefund == payAmount）→ 订单记账金额 orderAmount
//   - 比例退款 → payRefund × orderAmount ÷ payAmount
//   - 缺少记账口径（orderAmount<=0）或缺少实付口径时按 1:1 处理
func calculateRefundCreditAmount(orderAmount, payAmount, payRefund float64, currency string) float64 {
	if payRefund <= 0 {
		return 0
	}
	const creditFractionDigits = int32(2)
	if orderAmount <= 0 {
		return decimal.NewFromFloat(payRefund).Round(creditFractionDigits).InexactFloat64()
	}
	if payAmount <= 0 {
		if payRefund >= orderAmount-paymentAmountToleranceForCurrency(currency) {
			return decimal.NewFromFloat(orderAmount).Round(creditFractionDigits).InexactFloat64()
		}
		return decimal.NewFromFloat(payRefund).Round(creditFractionDigits).InexactFloat64()
	}
	if math.Abs(payRefund-payAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(orderAmount).Round(creditFractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payRefund).
		Mul(decimal.NewFromFloat(orderAmount)).
		Div(decimal.NewFromFloat(payAmount)).
		Round(creditFractionDigits).
		InexactFloat64()
}
