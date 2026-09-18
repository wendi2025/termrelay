package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ============================================================================
// PaymentService 的共建者分账入口
//
// 履约成功后的记账动作拆成两类：
//   - 邀请返利（applyAffiliateRebateForOrder）：失败阻断履约，保持存量语义；
//   - 共建者分账（applyRevenueSplitForOrder）：失败只告警，不阻断客户充值到账。
//
// 分账失败可以安全重跑（AccrueForOrder 幂等），管理端「补计提」即调用它。
// ============================================================================

// applyPostPaymentAccounting 履约成功后的记账动作集合。
func (s *PaymentService) applyPostPaymentAccounting(ctx context.Context, o *dbent.PaymentOrder) error {
	if err := s.applyAffiliateRebateForOrder(ctx, o); err != nil {
		return err
	}
	s.applyRevenueSplitForOrder(ctx, o)
	return nil
}

// applyRevenueSplitForOrder 客户支付成功后的共建者分账计提。
//
// 分账只影响记账，不影响客户余额与订单状态，因此失败不阻断履约：
// 记一条审计日志 + 告警，管理员可重跑（幂等）。
func (s *PaymentService) applyRevenueSplitForOrder(ctx context.Context, o *dbent.PaymentOrder) {
	if s == nil || s.revenueSplit == nil || o == nil {
		return
	}
	created, err := s.revenueSplit.AccrueForOrder(ctx, o)
	if err != nil {
		slog.Warn("payment: revenue split accrue failed", "orderID", o.ID, "error", err)
		s.writeAuditLog(ctx, o.ID, "REVENUE_SPLIT_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return
	}
	if created > 0 {
		s.writeAuditLog(ctx, o.ID, "REVENUE_SPLIT_APPLIED", "system", map[string]any{
			"entries": created,
		})
	}
}

// RepairRevenueSplitForOrder 为某笔已完成的订单补计提分账（幂等）。
//
// 用于分账功能上线前已完成的历史订单，或分账写入曾失败需要补齐的情况。
func (s *PaymentService) RepairRevenueSplitForOrder(ctx context.Context, orderID int64) (int, error) {
	if s == nil || s.entClient == nil {
		return 0, errors.New("payment service is unavailable")
	}
	if s.revenueSplit == nil {
		return 0, errors.New("revenue split service is unavailable")
	}
	if orderID <= 0 {
		return 0, infraerrors.BadRequest("INVALID_ORDER_ID", "order id is required")
	}
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return 0, fmt.Errorf("load payment order %d: %w", orderID, err)
	}
	if order.Status != OrderStatusCompleted {
		return 0, infraerrors.BadRequest("ORDER_NOT_COMPLETED",
			"only completed orders can accrue revenue split entries")
	}
	created, err := s.revenueSplit.AccrueForOrder(ctx, order)
	if err != nil {
		return created, err
	}
	s.writeAuditLog(ctx, order.ID, "REVENUE_SPLIT_REPAIRED", "admin", map[string]any{
		"entries": created,
	})
	return created, nil
}

// --- 管理端 / 用户端查询入口（薄封装，便于 handler 只依赖 PaymentService） ---

// GetRevenueSplitConfig 读取分账配置。
func (s *PaymentService) GetRevenueSplitConfig(ctx context.Context) (RevenueSplitConfig, error) {
	if s == nil || s.revenueSplit == nil {
		return DefaultRevenueSplitConfig(), nil
	}
	return s.revenueSplit.LoadConfig(ctx)
}

// UpdateRevenueSplitConfig 更新分账配置。
func (s *PaymentService) UpdateRevenueSplitConfig(ctx context.Context, in RevenueSplitConfigInput) (RevenueSplitConfig, error) {
	if s == nil || s.revenueSplit == nil {
		return RevenueSplitConfig{}, errors.New("revenue split service is unavailable")
	}
	return s.revenueSplit.SaveConfig(ctx, in)
}

// ListRevenueSplitRules 列出分账规则。
func (s *PaymentService) ListRevenueSplitRules(ctx context.Context) ([]RevenueSplitRule, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, nil
	}
	return s.revenueSplit.ListRules(ctx)
}

// ReplaceRevenueSplitRules 全量替换分账规则。
func (s *PaymentService) ReplaceRevenueSplitRules(ctx context.Context, in []RevenueSplitRuleInput) ([]RevenueSplitRule, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, errors.New("revenue split service is unavailable")
	}
	return s.revenueSplit.ReplaceRules(ctx, in)
}

// ListRevenueSplitEntries 分页查询分账明细。
func (s *PaymentService) ListRevenueSplitEntries(ctx context.Context, f RevenueSplitEntryFilter) ([]RevenueSplitEntry, int64, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, 0, nil
	}
	return s.revenueSplit.ListEntries(ctx, f)
}

// RevenueSplitSummary 各受益人的分账汇总。
func (s *PaymentService) RevenueSplitSummary(ctx context.Context) ([]RevenueSplitBeneficiarySummary, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, nil
	}
	return s.revenueSplit.Summarize(ctx)
}

// RevenueSplitPreview 按给定金额试算分账结果（不落库）。
func (s *PaymentService) RevenueSplitPreview(ctx context.Context, amount float64) (map[string]any, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, errors.New("revenue split service is unavailable")
	}
	cfg, err := s.revenueSplit.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.revenueSplit.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	base, fee := RevenueSplitPreviewBaseAmount(amount, cfg)
	items := make([]map[string]any, 0, len(rules))
	allocated := 0.0
	for _, r := range rules {
		if !r.Enabled || r.RatioPercent <= 0 {
			continue
		}
		split := RevenueSplitPreviewSplitAmount(base, r.RatioPercent)
		allocated += split
		items = append(items, map[string]any{
			"beneficiary_user_id": r.BeneficiaryUserID,
			"beneficiary_name":    r.BeneficiaryName,
			"ratio_percent":       r.RatioPercent,
			"split_amount":        split,
		})
	}
	return map[string]any{
		"enabled":             cfg.Enabled,
		"base_mode":           cfg.BaseMode,
		"channel_fee_percent": cfg.ChannelFeePercent,
		"gross_amount":        amount,
		"channel_fee_amount":  fee,
		"base_amount":         base,
		"allocated_amount":    allocated,
		"platform_remainder":  base - allocated,
		"items":               items,
	}, nil
}

// MyRevenueSplit 共建者查看自己的分账汇总。
func (s *PaymentService) MyRevenueSplit(ctx context.Context, userID int64) (*RevenueSplitBeneficiarySummary, error) {
	if s == nil || s.revenueSplit == nil {
		return &RevenueSplitBeneficiarySummary{}, nil
	}
	return s.revenueSplit.BeneficiarySummary(ctx, userID)
}

// ListRevenueSplitSettlements 分页查询结算单。
func (s *PaymentService) ListRevenueSplitSettlements(ctx context.Context, beneficiaryUserID int64, status string, page, pageSize int) ([]RevenueSplitSettlement, int64, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, 0, nil
	}
	return s.revenueSplit.ListSettlements(ctx, beneficiaryUserID, status, page, pageSize)
}

// CreateRevenueSplitSettlement 生成结算单（锁定待结算分录）。
func (s *PaymentService) CreateRevenueSplitSettlement(ctx context.Context, in RevenueSplitSettlementInput, actorID int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, errors.New("revenue split service is unavailable")
	}
	return s.revenueSplit.CreateSettlement(ctx, in, actorID)
}

// MarkRevenueSplitSettlementPaid 标记结算单已线下打款。
func (s *PaymentService) MarkRevenueSplitSettlementPaid(ctx context.Context, id int64, in RevenueSplitSettlementPaidInput, actorID int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, errors.New("revenue split service is unavailable")
	}
	return s.revenueSplit.MarkSettlementPaid(ctx, id, in, actorID)
}

// CancelRevenueSplitSettlement 取消结算单并释放分录。
func (s *PaymentService) CancelRevenueSplitSettlement(ctx context.Context, id int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, errors.New("revenue split service is unavailable")
	}
	return s.revenueSplit.CancelSettlement(ctx, id)
}

// revenueSplitSettlementDetail 返回结算单及其分录（对账单用）。
func (s *PaymentService) RevenueSplitSettlementDetail(ctx context.Context, id int64) (*RevenueSplitSettlement, []RevenueSplitEntry, error) {
	if s == nil || s.revenueSplit == nil {
		return nil, nil, errors.New("revenue split service is unavailable")
	}
	settlement, err := s.revenueSplit.GetSettlement(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	// 按结算单直接过滤并翻页取全。旧写法是"取该受益人前 N 条再过滤"，
	// 分录一多（明细按 id 倒序）就会把较早结算单的明细整段漏掉，
	// 对账单必须完整，因此这里改成按 settlement_id 精确查询。
	entries := make([]RevenueSplitEntry, 0, 32)
	for page := 1; ; page++ {
		batch, total, err := s.revenueSplit.ListEntries(ctx, RevenueSplitEntryFilter{
			SettlementID: id,
			Page:         page,
			PageSize:     revenueSplitMaxPageSize,
		})
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, batch...)
		if len(batch) == 0 || int64(len(entries)) >= total {
			break
		}
	}
	return settlement, entries, nil
}
