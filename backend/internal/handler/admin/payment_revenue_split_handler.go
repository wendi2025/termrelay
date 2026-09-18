package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// --- 共建者分账（多受益人固定比例分账） ---
//
// 设计边界：本模块只做「记账」。
//   客户支付 → 按比例计提分录（pending）→ 生成结算单（draft）
//   → 管理员线下手工打款 → 回填流水号并标记已打款（paid）。
// 系统不会、也不能自动把钱转给受益人（那属于无牌照资金清分）。

// RevenueSplitActorID 取当前操作管理员 ID（用于结算单的 created_by / paid_by）。
func RevenueSplitActorID(c *gin.Context) int64 {
	raw, ok := c.Get(string(middleware.ContextKeyUser))
	if !ok {
		return 0
	}
	if subject, ok := raw.(middleware.AuthSubject); ok {
		return subject.UserID
	}
	return 0
}

func parseRevenueSplitTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			parsed := t
			return &parsed
		}
	}
	return nil
}

// GetRevenueSplitConfig 读取分账配置。
// GET /api/v1/admin/payment/revenue-split/config
func (h *PaymentHandler) GetRevenueSplitConfig(c *gin.Context) {
	cfg, err := h.paymentService.GetRevenueSplitConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rules, err := h.paymentService.ListRevenueSplitRules(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"config": cfg, "rules": rules})
}

// UpdateRevenueSplitConfigRequest 分账配置更新入参。
type UpdateRevenueSplitConfigRequest struct {
	Enabled           *bool    `json:"enabled"`
	BaseMode          *string  `json:"base_mode"`
	ChannelFeePercent *float64 `json:"channel_fee_percent"`
}

// UpdateRevenueSplitConfig 更新分账配置。
// PUT /api/v1/admin/payment/revenue-split/config
func (h *PaymentHandler) UpdateRevenueSplitConfig(c *gin.Context) {
	var req UpdateRevenueSplitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	cfg, err := h.paymentService.UpdateRevenueSplitConfig(c.Request.Context(), service.RevenueSplitConfigInput{
		Enabled:           req.Enabled,
		BaseMode:          req.BaseMode,
		ChannelFeePercent: req.ChannelFeePercent,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"config": cfg})
}

// ListRevenueSplitRules 列出分账规则。
// GET /api/v1/admin/payment/revenue-split/rules
func (h *PaymentHandler) ListRevenueSplitRules(c *gin.Context) {
	rules, err := h.paymentService.ListRevenueSplitRules(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"rules": rules})
}

// ReplaceRevenueSplitRulesRequest 规则全量替换入参。
type ReplaceRevenueSplitRulesRequest struct {
	Rules []service.RevenueSplitRuleInput `json:"rules"`
}

// ReplaceRevenueSplitRules 全量替换分账规则。
// PUT /api/v1/admin/payment/revenue-split/rules
func (h *PaymentHandler) ReplaceRevenueSplitRules(c *gin.Context) {
	var req ReplaceRevenueSplitRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	rules, err := h.paymentService.ReplaceRevenueSplitRules(c.Request.Context(), req.Rules)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"rules": rules})
}

// PreviewRevenueSplitRequest 试算入参。
type PreviewRevenueSplitRequest struct {
	Amount float64 `json:"amount"`
}

// PreviewRevenueSplit 按给定金额试算分账结果（不落库）。
// POST /api/v1/admin/payment/revenue-split/preview
func (h *PaymentHandler) PreviewRevenueSplit(c *gin.Context) {
	var req PreviewRevenueSplitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.Amount <= 0 {
		response.BadRequest(c, "amount must be greater than 0")
		return
	}
	result, err := h.paymentService.RevenueSplitPreview(c.Request.Context(), req.Amount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ListRevenueSplitEntries 分页查询分账明细。
// GET /api/v1/admin/payment/revenue-split/entries
func (h *PaymentHandler) ListRevenueSplitEntries(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var beneficiaryID int64
	if raw := c.Query("beneficiary_user_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			beneficiaryID = v
		}
	}
	var orderID int64
	if raw := c.Query("order_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			orderID = v
		}
	}
	entries, total, err := h.paymentService.ListRevenueSplitEntries(c.Request.Context(), service.RevenueSplitEntryFilter{
		BeneficiaryUserID: beneficiaryID,
		Status:            c.Query("status"),
		OrderID:           orderID,
		Start:             parseRevenueSplitTime(c.Query("start")),
		End:               parseRevenueSplitTime(c.Query("end")),
		Keyword:           c.Query("keyword"),
		Page:              page,
		PageSize:          pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, int64(total), page, pageSize)
}

// RevenueSplitSummary 各受益人分账汇总。
// GET /api/v1/admin/payment/revenue-split/summary
func (h *PaymentHandler) RevenueSplitSummary(c *gin.Context) {
	summaries, err := h.paymentService.RevenueSplitSummary(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"beneficiaries": summaries})
}

// ListRevenueSplitSettlements 分页查询结算单。
// GET /api/v1/admin/payment/revenue-split/settlements
func (h *PaymentHandler) ListRevenueSplitSettlements(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var beneficiaryID int64
	if raw := c.Query("beneficiary_user_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			beneficiaryID = v
		}
	}
	settlements, total, err := h.paymentService.ListRevenueSplitSettlements(
		c.Request.Context(), beneficiaryID, c.Query("status"), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, settlements, int64(total), page, pageSize)
}

// GetRevenueSplitSettlement 结算单详情（含锁定的分账明细）。
// GET /api/v1/admin/payment/revenue-split/settlements/:id
func (h *PaymentHandler) GetRevenueSplitSettlement(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	settlement, entries, err := h.paymentService.RevenueSplitSettlementDetail(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"settlement": settlement, "entries": entries})
}

// CreateRevenueSplitSettlementRequest 生成结算单入参。
type CreateRevenueSplitSettlementRequest struct {
	BeneficiaryUserID int64  `json:"beneficiary_user_id"`
	PeriodStart       string `json:"period_start"`
	PeriodEnd         string `json:"period_end"`
	Method            string `json:"method"`
	Note              string `json:"note"`
}

// CreateRevenueSplitSettlement 为某受益人生成结算单（锁定待结算分录）。
// POST /api/v1/admin/payment/revenue-split/settlements
func (h *PaymentHandler) CreateRevenueSplitSettlement(c *gin.Context) {
	var req CreateRevenueSplitSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settlement, err := h.paymentService.CreateRevenueSplitSettlement(c.Request.Context(), service.RevenueSplitSettlementInput{
		BeneficiaryUserID: req.BeneficiaryUserID,
		PeriodStart:       parseRevenueSplitTime(req.PeriodStart),
		PeriodEnd:         parseRevenueSplitTime(req.PeriodEnd),
		Method:            req.Method,
		Note:              req.Note,
	}, RevenueSplitActorID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settlement)
}

// MarkRevenueSplitSettlementPaidRequest 标记已打款入参。
type MarkRevenueSplitSettlementPaidRequest struct {
	Method    string `json:"method"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

// MarkRevenueSplitSettlementPaid 标记结算单已线下打款。
// POST /api/v1/admin/payment/revenue-split/settlements/:id/pay
func (h *PaymentHandler) MarkRevenueSplitSettlementPaid(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req MarkRevenueSplitSettlementPaidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settlement, err := h.paymentService.MarkRevenueSplitSettlementPaid(c.Request.Context(), id,
		service.RevenueSplitSettlementPaidInput{
			Method:    req.Method,
			Reference: req.Reference,
			Note:      req.Note,
		}, RevenueSplitActorID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settlement)
}

// CancelRevenueSplitSettlement 取消结算单并释放分录。
// POST /api/v1/admin/payment/revenue-split/settlements/:id/cancel
func (h *PaymentHandler) CancelRevenueSplitSettlement(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	settlement, err := h.paymentService.CancelRevenueSplitSettlement(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settlement)
}

// AccrueRevenueSplitForOrder 为指定已完成订单补计提分账（幂等）。
// POST /api/v1/admin/payment/revenue-split/orders/:id/accrue
func (h *PaymentHandler) AccrueRevenueSplitForOrder(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	created, err := h.paymentService.RepairRevenueSplitForOrder(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"entries": created})
}
