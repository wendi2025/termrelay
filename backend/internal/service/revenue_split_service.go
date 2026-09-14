package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// ============================================================================
// Smirel 多受益人分账（共建者分成）
//
// 目的：客户每支付一笔，按固定比例把「可分配收入」记到各共建者名下。
//
// 三条不可逾越的边界：
//  1. 只记账，不出金。客户的钱整笔进平台收款账户，由管理员线下手工打款后
//     再在系统里标记「已结算」。系统自动把钱转给第三方 = 资金清分（二清），
//     无支付牌照属违法行为。
//  2. 分账基数默认扣除支付通道费（gross_after_fee）。ZPay 抽 1.0% + 0.6%，
//     不扣就会把通道费也分出去，从分账当期就开始亏。
//  3. 上游 API 成本不在本层扣。余额是预收款，客户消耗余额时才产生上游成本，
//     因此分成比例之和必须给平台留出足够空间，严禁分满 100%。
// ============================================================================

// --- 设置键（settings 表） ---
const (
	// SettingRevenueSplitEnabled 分账总开关。默认关闭，配好比例后再打开。
	SettingRevenueSplitEnabled = "revenue_split_enabled"
	// SettingRevenueSplitBaseMode 计费基数口径。
	SettingRevenueSplitBaseMode = "revenue_split_base_mode"
	// SettingRevenueSplitChannelFeePercent 支付通道费率（百分比，估算值）。
	SettingRevenueSplitChannelFeePercent = "revenue_split_channel_fee_percent"
)

// --- 计费基数口径 ---
const (
	// RevenueSplitBaseModeGross 客户实付毛额，不扣任何费用。
	RevenueSplitBaseModeGross = "gross"
	// RevenueSplitBaseModeGrossAfterFee 客户实付毛额扣除支付通道费（默认）。
	RevenueSplitBaseModeGrossAfterFee = "gross_after_fee"
)

// --- 分账分录状态 ---
const (
	RevenueSplitEntryPending  = "pending"
	RevenueSplitEntrySettled  = "settled"
	RevenueSplitEntryReversed = "reversed"
)

// --- 结算单状态 ---
const (
	RevenueSplitSettlementDraft     = "draft"
	RevenueSplitSettlementPaid      = "paid"
	RevenueSplitSettlementCancelled = "cancelled"
)

const (
	// defaultRevenueSplitChannelFeePercent ZPay 标准费率 1.0% + 0.6%（支付宝/微信）。
	defaultRevenueSplitChannelFeePercent = 1.6
	revenueSplitCurrency                 = "CNY"
	// revenueSplitFractionDigits 分账金额精度：分（0.01 元）。
	revenueSplitFractionDigits = int32(2)
	revenueSplitMaxPageSize    = 200
)

// revenueSplitQueryExecer 抽象 ent 客户端与事务客户端共有的原生 SQL 能力。
type revenueSplitQueryExecer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// RevenueSplitService 共建者分账服务。
//
// 与其他账本一样，本服务只写「记账」数据，不触碰任何出金通道。
type RevenueSplitService struct {
	client *dbent.Client
}

// NewRevenueSplitService 创建分账服务。
func NewRevenueSplitService(client *dbent.Client) *RevenueSplitService {
	if client == nil {
		panic("NewRevenueSplitService: client is nil")
	}
	return &RevenueSplitService{client: client}
}

// clientFor 在事务上下文中优先使用事务客户端，保证与调用方同一事务。
func (s *RevenueSplitService) clientFor(ctx context.Context) revenueSplitQueryExecer {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return s.client
}

// --- 配置 ---

// RevenueSplitConfig 分账全局配置。
type RevenueSplitConfig struct {
	Enabled           bool    `json:"enabled"`
	BaseMode          string  `json:"base_mode"`
	ChannelFeePercent float64 `json:"channel_fee_percent"`
}

// RevenueSplitConfigInput 分账配置更新入参（nil 表示不修改）。
type RevenueSplitConfigInput struct {
	Enabled           *bool    `json:"enabled"`
	BaseMode          *string  `json:"base_mode"`
	ChannelFeePercent *float64 `json:"channel_fee_percent"`
}

func normalizeRevenueSplitBaseMode(mode string) string {
	if strings.TrimSpace(mode) == RevenueSplitBaseModeGross {
		return RevenueSplitBaseModeGross
	}
	return RevenueSplitBaseModeGrossAfterFee
}

func normalizeRevenueSplitChannelFeePercent(v float64) float64 {
	if v < 0 || v > 100 {
		return defaultRevenueSplitChannelFeePercent
	}
	return v
}

// DefaultRevenueSplitConfig 返回默认配置（用于设置项缺失时兜底）。
func DefaultRevenueSplitConfig() RevenueSplitConfig {
	return RevenueSplitConfig{
		Enabled:           false,
		BaseMode:          RevenueSplitBaseModeGrossAfterFee,
		ChannelFeePercent: defaultRevenueSplitChannelFeePercent,
	}
}

// LoadConfig 读取分账配置。设置项缺失时回落到默认值（默认关闭）。
func (s *RevenueSplitService) LoadConfig(ctx context.Context) (RevenueSplitConfig, error) {
	cfg := DefaultRevenueSplitConfig()
	if s == nil || s.client == nil {
		return cfg, nil
	}
	rows, err := s.clientFor(ctx).QueryContext(ctx,
		`SELECT key, value FROM settings WHERE key IN ($1, $2, $3)`,
		SettingRevenueSplitEnabled, SettingRevenueSplitBaseMode, SettingRevenueSplitChannelFeePercent)
	if err != nil {
		return cfg, fmt.Errorf("load revenue split settings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return cfg, fmt.Errorf("scan revenue split setting: %w", err)
		}
		value = strings.TrimSpace(value)
		switch key {
		case SettingRevenueSplitEnabled:
			cfg.Enabled = value == "true" || value == "1" || strings.EqualFold(value, "yes")
		case SettingRevenueSplitBaseMode:
			cfg.BaseMode = normalizeRevenueSplitBaseMode(value)
		case SettingRevenueSplitChannelFeePercent:
			if v, perr := parseFloatLoose(value); perr == nil {
				cfg.ChannelFeePercent = normalizeRevenueSplitChannelFeePercent(v)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return cfg, fmt.Errorf("iterate revenue split settings: %w", err)
	}
	return cfg, nil
}

// SaveConfig 保存分账配置（仅更新传入字段）。
func (s *RevenueSplitService) SaveConfig(ctx context.Context, in RevenueSplitConfigInput) (RevenueSplitConfig, error) {
	if s == nil || s.client == nil {
		return RevenueSplitConfig{}, infraerrors.BadRequest("REVENUE_SPLIT_UNAVAILABLE", "revenue split service is unavailable")
	}
	current, err := s.LoadConfig(ctx)
	if err != nil {
		return RevenueSplitConfig{}, err
	}
	if in.BaseMode != nil {
		mode := strings.TrimSpace(*in.BaseMode)
		if mode != RevenueSplitBaseModeGross && mode != RevenueSplitBaseModeGrossAfterFee {
			return RevenueSplitConfig{}, infraerrors.BadRequest("INVALID_BASE_MODE", "base_mode must be gross or gross_after_fee")
		}
		current.BaseMode = mode
	}
	if in.ChannelFeePercent != nil {
		v := *in.ChannelFeePercent
		if v < 0 || v > 100 {
			return RevenueSplitConfig{}, infraerrors.BadRequest("INVALID_CHANNEL_FEE_PERCENT", "channel_fee_percent must be between 0 and 100")
		}
		current.ChannelFeePercent = v
	}
	if in.Enabled != nil {
		current.Enabled = *in.Enabled
	}
	// 打开开关时必须已有启用规则，否则会静默不计提。
	if current.Enabled {
		rules, rerr := s.ListRules(ctx)
		if rerr != nil {
			return RevenueSplitConfig{}, rerr
		}
		hasEnabled := false
		for _, r := range rules {
			if r.Enabled && r.RatioPercent > 0 {
				hasEnabled = true
				break
			}
		}
		if !hasEnabled {
			return RevenueSplitConfig{}, infraerrors.BadRequest("REVENUE_SPLIT_NO_RULE", "cannot enable revenue split without at least one enabled beneficiary rule")
		}
	}
	upserts := []struct {
		key   string
		value string
	}{
		{SettingRevenueSplitEnabled, boolToSettingValue(current.Enabled)},
		{SettingRevenueSplitBaseMode, current.BaseMode},
		{SettingRevenueSplitChannelFeePercent, formatFloatLoose(current.ChannelFeePercent)},
	}
	for _, u := range upserts {
		if _, err := s.clientFor(ctx).ExecContext(ctx,
			`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW())
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			u.key, u.value); err != nil {
			return RevenueSplitConfig{}, fmt.Errorf("save revenue split setting %s: %w", u.key, err)
		}
	}
	return s.LoadConfig(ctx)
}

// --- 规则 ---

// RevenueSplitRule 一条分账规则（受益人 + 固定比例）。
type RevenueSplitRule struct {
	ID                int64   `json:"id"`
	BeneficiaryUserID int64   `json:"beneficiary_user_id"`
	BeneficiaryName   string  `json:"beneficiary_name"`
	BeneficiaryEmail  string  `json:"beneficiary_email"`
	BeneficiaryStatus string  `json:"beneficiary_status"`
	RatioPercent      float64 `json:"ratio_percent"`
	Enabled           bool    `json:"enabled"`
	Note              string  `json:"note"`
	SortOrder         int     `json:"sort_order"`
}

// RevenueSplitRuleInput 规则写入入参。
type RevenueSplitRuleInput struct {
	BeneficiaryUserID int64   `json:"beneficiary_user_id"`
	BeneficiaryName   string  `json:"beneficiary_name"`
	RatioPercent      float64 `json:"ratio_percent"`
	Enabled           *bool   `json:"enabled"`
	Note              string  `json:"note"`
	SortOrder         int     `json:"sort_order"`
}

// ValidateRevenueSplitRules 校验规则集：受益人唯一、比例合法、启用规则之和 <= 100。
func ValidateRevenueSplitRules(in []RevenueSplitRuleInput) error {
	if len(in) == 0 {
		return infraerrors.BadRequest("INVALID_REVENUE_SPLIT_RULES", "at least one beneficiary rule is required")
	}
	seen := make(map[int64]struct{}, len(in))
	total := decimal.Zero
	for _, r := range in {
		if r.BeneficiaryUserID <= 0 {
			return infraerrors.BadRequest("INVALID_BENEFICIARY", "beneficiary_user_id is required")
		}
		if _, dup := seen[r.BeneficiaryUserID]; dup {
			return infraerrors.BadRequest("DUPLICATE_BENEFICIARY", fmt.Sprintf("beneficiary %d appears more than once", r.BeneficiaryUserID))
		}
		seen[r.BeneficiaryUserID] = struct{}{}
		if r.RatioPercent <= 0 || r.RatioPercent > 100 {
			return infraerrors.BadRequest("INVALID_RATIO", fmt.Sprintf("beneficiary %d ratio_percent must be between 0 and 100", r.BeneficiaryUserID))
		}
		enabled := r.Enabled == nil || *r.Enabled
		if enabled {
			total = total.Add(decimal.NewFromFloat(r.RatioPercent))
		}
	}
	if total.GreaterThan(decimal.NewFromInt(100)) {
		return infraerrors.BadRequest("REVENUE_SPLIT_OVERFLOW",
			fmt.Sprintf("enabled beneficiary ratios sum to %s%%, which exceeds 100%%", total.String()))
	}
	return nil
}

const revenueSplitRulesSelectSQL = `
SELECT r.id,
       r.beneficiary_user_id,
       COALESCE(NULLIF(r.beneficiary_name, ''), COALESCE(u.username, ''), ''),
       COALESCE(u.email, ''),
       COALESCE(u.status, ''),
       r.ratio_percent::double precision,
       r.enabled,
       COALESCE(r.note, ''),
       r.sort_order
FROM revenue_split_rules r
LEFT JOIN users u ON u.id = r.beneficiary_user_id
ORDER BY r.sort_order ASC, r.id ASC`

// ListRules 列出全部分账规则（含停用）。
func (s *RevenueSplitService) ListRules(ctx context.Context) ([]RevenueSplitRule, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.clientFor(ctx).QueryContext(ctx, revenueSplitRulesSelectSQL)
	if err != nil {
		return nil, fmt.Errorf("list revenue split rules: %w", err)
	}
	defer func() { _ = rows.Close() }()
	rules := make([]RevenueSplitRule, 0, 4)
	for rows.Next() {
		var r RevenueSplitRule
		if err := rows.Scan(&r.ID, &r.BeneficiaryUserID, &r.BeneficiaryName, &r.BeneficiaryEmail,
			&r.BeneficiaryStatus, &r.RatioPercent, &r.Enabled, &r.Note, &r.SortOrder); err != nil {
			return nil, fmt.Errorf("scan revenue split rule: %w", err)
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revenue split rules: %w", err)
	}
	return rules, nil
}

// ReplaceRules 全量替换分账规则（先清空再写入），返回替换后的规则集。
func (s *RevenueSplitService) ReplaceRules(ctx context.Context, in []RevenueSplitRuleInput) ([]RevenueSplitRule, error) {
	if s == nil || s.client == nil {
		return nil, infraerrors.BadRequest("REVENUE_SPLIT_UNAVAILABLE", "revenue split service is unavailable")
	}
	if err := ValidateRevenueSplitRules(in); err != nil {
		return nil, err
	}
	if err := s.assertBeneficiariesExist(ctx, in); err != nil {
		return nil, err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin revenue split rules tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()

	if _, err := txClient.ExecContext(ctx, `DELETE FROM revenue_split_rules`); err != nil {
		return nil, fmt.Errorf("clear revenue split rules: %w", err)
	}
	for i, r := range in {
		enabled := r.Enabled == nil || *r.Enabled
		sortOrder := r.SortOrder
		if sortOrder == 0 {
			sortOrder = i
		}
		if _, err := txClient.ExecContext(ctx,
			`INSERT INTO revenue_split_rules
			   (beneficiary_user_id, beneficiary_name, ratio_percent, enabled, note, sort_order, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
			r.BeneficiaryUserID, strings.TrimSpace(r.BeneficiaryName), r.RatioPercent,
			enabled, strings.TrimSpace(r.Note), sortOrder); err != nil {
			return nil, fmt.Errorf("insert revenue split rule: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit revenue split rules tx: %w", err)
	}
	return s.ListRules(ctx)
}

func (s *RevenueSplitService) assertBeneficiariesExist(ctx context.Context, in []RevenueSplitRuleInput) error {
	client := s.clientFor(ctx)
	for _, r := range in {
		var exists bool
		rows, err := client.QueryContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, r.BeneficiaryUserID)
		if err != nil {
			return fmt.Errorf("check beneficiary exists: %w", err)
		}
		if !rows.Next() {
			_ = rows.Close()
			return infraerrors.BadRequest("BENEFICIARY_NOT_FOUND", fmt.Sprintf("beneficiary user %d does not exist", r.BeneficiaryUserID))
		}
		if err := rows.Scan(&exists); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan beneficiary exists: %w", err)
		}
		_ = rows.Close()
		if !exists {
			return infraerrors.BadRequest("BENEFICIARY_NOT_FOUND", fmt.Sprintf("beneficiary user %d does not exist", r.BeneficiaryUserID))
		}
	}
	return nil
}

// --- 计提 ---

// AccrueForOrder 为支付成功的订单计提分账（幂等）。
//
// 同一订单重复调用只会补齐缺失的分录，不会重复计提，因此可以安全地在履约
// 重试、管理员补计提等场景下反复调用。
func (s *RevenueSplitService) AccrueForOrder(ctx context.Context, order *dbent.PaymentOrder) (int, error) {
	if s == nil || s.client == nil || order == nil {
		return 0, nil
	}
	cfg, err := s.LoadConfig(ctx)
	if err != nil {
		return 0, err
	}
	if !cfg.Enabled {
		return 0, nil
	}
	cash := revenueSplitCashAmount(order)
	if cash <= 0 {
		return 0, nil
	}
	base, feeAmount := revenueSplitBaseAmount(cash, cfg)
	if base <= 0 {
		return 0, nil
	}
	rules, err := s.ListRules(ctx)
	if err != nil {
		return 0, err
	}
	client := s.clientFor(ctx)
	created := 0
	for _, r := range rules {
		if !r.Enabled || r.RatioPercent <= 0 {
			continue
		}
		split := revenueSplitSplitAmount(base, r.RatioPercent)
		if split <= 0 {
			continue
		}
		res, err := client.ExecContext(ctx,
			`INSERT INTO revenue_split_entries
			   (order_id, beneficiary_user_id, beneficiary_name, order_amount, pay_amount,
			    channel_fee_percent, channel_fee_amount, base_mode, base_amount, ratio_percent,
			    split_amount, currency, status, memo, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
			 ON CONFLICT ON CONSTRAINT revenue_split_entries_order_beneficiary_key DO NOTHING`,
			order.ID, r.BeneficiaryUserID, r.BeneficiaryName, order.Amount, cash,
			cfg.ChannelFeePercent, feeAmount, cfg.BaseMode, base, r.RatioPercent,
			split, revenueSplitCurrency, RevenueSplitEntryPending,
			fmt.Sprintf("order:%d %s", order.ID, order.OrderType))
		if err != nil {
			return created, fmt.Errorf("insert revenue split entry: %w", err)
		}
		if n, aerr := res.RowsAffected(); aerr == nil && n > 0 {
			created++
		}
	}
	return created, nil
}

// ReverseForOrder 订单退款后冲回该订单的分账（幂等）。
//
// 注意：已进入结算单（甚至已打款）的分录同样会被冲回 —— 退款是事实，
// 记账必须跟随事实。已打款的部分需要管理员在下一期结算中扣减，系统在
// 明细与汇总（reversed 金额）里都能看到。
func (s *RevenueSplitService) ReverseForOrder(ctx context.Context, orderID int64, reason string) (int, error) {
	if s == nil || s.client == nil || orderID <= 0 {
		return 0, nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "order refunded"
	}
	res, err := s.clientFor(ctx).ExecContext(ctx,
		`UPDATE revenue_split_entries
		    SET status = $2, reversed_at = NOW(), reverse_reason = $3, updated_at = NOW()
		  WHERE order_id = $1 AND status <> $2`,
		orderID, RevenueSplitEntryReversed, strings.TrimSpace(reason))
	if err != nil {
		return 0, fmt.Errorf("reverse revenue split entries: %w", err)
	}
	n, aerr := res.RowsAffected()
	if aerr != nil {
		return 0, nil
	}
	return int(n), nil
}

// --- 分录查询 ---

// RevenueSplitEntry 一条分账明细。
type RevenueSplitEntry struct {
	ID                int64      `json:"id"`
	OrderID           int64      `json:"order_id"`
	BeneficiaryUserID int64      `json:"beneficiary_user_id"`
	BeneficiaryName   string     `json:"beneficiary_name"`
	OrderAmount       float64    `json:"order_amount"`
	PayAmount         float64    `json:"pay_amount"`
	ChannelFeePercent float64    `json:"channel_fee_percent"`
	ChannelFeeAmount  float64    `json:"channel_fee_amount"`
	BaseMode          string     `json:"base_mode"`
	BaseAmount        float64    `json:"base_amount"`
	RatioPercent      float64    `json:"ratio_percent"`
	SplitAmount       float64    `json:"split_amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	SettlementID      *int64     `json:"settlement_id"`
	ReversedAt        *time.Time `json:"reversed_at"`
	ReverseReason     string     `json:"reverse_reason"`
	Memo              string     `json:"memo"`
	CreatedAt         time.Time  `json:"created_at"`
	OrderOutTradeNo   string     `json:"order_out_trade_no"`
	OrderUserID       int64      `json:"order_user_id"`
	OrderType         string     `json:"order_type"`
	PaymentType       string     `json:"payment_type"`
}

// RevenueSplitEntryFilter 分账明细查询条件。
type RevenueSplitEntryFilter struct {
	BeneficiaryUserID int64
	Status            string
	OrderID           int64
	// SettlementID 按结算单过滤（>0 生效），对账单需要它来精确取全明细。
	SettlementID int64
	Start        *time.Time
	End          *time.Time
	Keyword      string
	Page         int
	PageSize     int
}

func normalizeRevenueSplitPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > revenueSplitMaxPageSize {
		pageSize = revenueSplitMaxPageSize
	}
	return page, pageSize
}

func revenueSplitEntryWhere(f RevenueSplitEntryFilter) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 6)
	add := func(cond string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(cond, len(args)))
	}
	if f.BeneficiaryUserID > 0 {
		add("e.beneficiary_user_id = $%d", f.BeneficiaryUserID)
	}
	if st := strings.TrimSpace(f.Status); st != "" {
		add("e.status = $%d", st)
	}
	if f.OrderID > 0 {
		add("e.order_id = $%d", f.OrderID)
	}
	if f.SettlementID > 0 {
		add("e.settlement_id = $%d", f.SettlementID)
	}
	if f.Start != nil {
		add("e.created_at >= $%d", *f.Start)
	}
	if f.End != nil {
		add("e.created_at <= $%d", *f.End)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		args = append(args, "%"+kw+"%")
		conditions = append(conditions, fmt.Sprintf(
			"(o.out_trade_no ILIKE $%d OR o.user_email ILIKE $%d OR e.beneficiary_name ILIKE $%d)",
			len(args), len(args), len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

const revenueSplitEntriesSelectSQL = `
SELECT e.id, e.order_id, e.beneficiary_user_id, e.beneficiary_name,
       e.order_amount::double precision, e.pay_amount::double precision,
       e.channel_fee_percent::double precision, e.channel_fee_amount::double precision,
       e.base_mode, e.base_amount::double precision, e.ratio_percent::double precision,
       e.split_amount::double precision, e.currency, e.status, e.settlement_id,
       e.reversed_at, COALESCE(e.reverse_reason, ''), COALESCE(e.memo, ''), e.created_at,
       COALESCE(o.out_trade_no, ''), COALESCE(o.user_id, 0), COALESCE(o.order_type, ''), COALESCE(o.payment_type, '')
FROM revenue_split_entries e
LEFT JOIN payment_orders o ON o.id = e.order_id`

// ListEntries 分页查询分账明细。
func (s *RevenueSplitService) ListEntries(ctx context.Context, f RevenueSplitEntryFilter) ([]RevenueSplitEntry, int64, error) {
	if s == nil || s.client == nil {
		return nil, 0, nil
	}
	page, pageSize := normalizeRevenueSplitPage(f.Page, f.PageSize)
	where, args := revenueSplitEntryWhere(f)
	client := s.clientFor(ctx)

	var total int64
	if err := revenueSplitQueryScalar(ctx, client,
		`SELECT COUNT(*) FROM revenue_split_entries e LEFT JOIN payment_orders o ON o.id = e.order_id `+where,
		args, &total); err != nil {
		return nil, 0, err
	}

	pageArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := revenueSplitEntriesSelectSQL + " " + where +
		fmt.Sprintf(" ORDER BY e.id DESC LIMIT $%d OFFSET $%d", len(pageArgs)-1, len(pageArgs))
	rows, err := client.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list revenue split entries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	entries := make([]RevenueSplitEntry, 0, pageSize)
	for rows.Next() {
		var e RevenueSplitEntry
		if err := rows.Scan(&e.ID, &e.OrderID, &e.BeneficiaryUserID, &e.BeneficiaryName,
			&e.OrderAmount, &e.PayAmount, &e.ChannelFeePercent, &e.ChannelFeeAmount,
			&e.BaseMode, &e.BaseAmount, &e.RatioPercent, &e.SplitAmount, &e.Currency,
			&e.Status, &e.SettlementID, &e.ReversedAt, &e.ReverseReason, &e.Memo, &e.CreatedAt,
			&e.OrderOutTradeNo, &e.OrderUserID, &e.OrderType, &e.PaymentType); err != nil {
			return nil, 0, fmt.Errorf("scan revenue split entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate revenue split entries: %w", err)
	}
	return entries, total, nil
}

// --- 受益人汇总 ---

// RevenueSplitBeneficiarySummary 单个受益人的分账汇总。
type RevenueSplitBeneficiarySummary struct {
	BeneficiaryUserID int64      `json:"beneficiary_user_id"`
	BeneficiaryName   string     `json:"beneficiary_name"`
	BeneficiaryEmail  string     `json:"beneficiary_email"`
	RatioPercent      float64    `json:"ratio_percent"`
	Enabled           bool       `json:"enabled"`
	PendingAmount     float64    `json:"pending_amount"`
	LockedAmount      float64    `json:"locked_amount"`
	SettledAmount     float64    `json:"settled_amount"`
	ReversedAmount    float64    `json:"reversed_amount"`
	EntryCount        int        `json:"entry_count"`
	PendingCount      int        `json:"pending_count"`
	LastEntryAt       *time.Time `json:"last_entry_at"`
}

const revenueSplitSummarySQL = `
SELECT e.beneficiary_user_id,
       COALESCE(NULLIF(MAX(e.beneficiary_name), ''), COALESCE(u.username, ''), ''),
       COALESCE(u.email, ''),
       COALESCE(SUM(CASE WHEN e.status = 'pending' AND e.settlement_id IS NULL THEN e.split_amount ELSE 0 END), 0)::double precision,
       COALESCE(SUM(CASE WHEN e.status = 'pending' AND e.settlement_id IS NOT NULL THEN e.split_amount ELSE 0 END), 0)::double precision,
       COALESCE(SUM(CASE WHEN e.status = 'settled' THEN e.split_amount ELSE 0 END), 0)::double precision,
       COALESCE(SUM(CASE WHEN e.status = 'reversed' THEN e.split_amount ELSE 0 END), 0)::double precision,
       COUNT(*)::int,
       COALESCE(SUM(CASE WHEN e.status = 'pending' AND e.settlement_id IS NULL THEN 1 ELSE 0 END), 0)::int,
       MAX(e.created_at)
FROM revenue_split_entries e
LEFT JOIN users u ON u.id = e.beneficiary_user_id
GROUP BY e.beneficiary_user_id, u.email, u.username
ORDER BY 4 DESC, e.beneficiary_user_id ASC`

// Summarize 汇总全部受益人的分账金额，并合并规则里的比例与启用状态。
func (s *RevenueSplitService) Summarize(ctx context.Context) ([]RevenueSplitBeneficiarySummary, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.clientFor(ctx).QueryContext(ctx, revenueSplitSummarySQL)
	if err != nil {
		return nil, fmt.Errorf("summarize revenue split: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]RevenueSplitBeneficiarySummary, 0, 4)
	byUser := make(map[int64]int, 4)
	for rows.Next() {
		var item RevenueSplitBeneficiarySummary
		if err := rows.Scan(&item.BeneficiaryUserID, &item.BeneficiaryName, &item.BeneficiaryEmail,
			&item.PendingAmount, &item.LockedAmount, &item.SettledAmount, &item.ReversedAmount,
			&item.EntryCount, &item.PendingCount, &item.LastEntryAt); err != nil {
			return nil, fmt.Errorf("scan revenue split summary: %w", err)
		}
		byUser[item.BeneficiaryUserID] = len(out)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revenue split summary: %w", err)
	}

	// 规则里配置了但还没有任何分录的受益人，也要出现在汇总里（金额为 0）。
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		idx, ok := byUser[r.BeneficiaryUserID]
		if !ok {
			out = append(out, RevenueSplitBeneficiarySummary{
				BeneficiaryUserID: r.BeneficiaryUserID,
				BeneficiaryName:   r.BeneficiaryName,
				BeneficiaryEmail:  r.BeneficiaryEmail,
			})
			idx = len(out) - 1
			byUser[r.BeneficiaryUserID] = idx
		}
		out[idx].RatioPercent = r.RatioPercent
		out[idx].Enabled = r.Enabled
	}
	return out, nil
}

// BeneficiarySummary 单个受益人的汇总（共建者自助查看用）。
func (s *RevenueSplitService) BeneficiarySummary(ctx context.Context, userID int64) (*RevenueSplitBeneficiarySummary, error) {
	if s == nil || s.client == nil || userID <= 0 {
		return &RevenueSplitBeneficiarySummary{}, nil
	}
	summaries, err := s.Summarize(ctx)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		if summaries[i].BeneficiaryUserID == userID {
			return &summaries[i], nil
		}
	}
	return &RevenueSplitBeneficiarySummary{BeneficiaryUserID: userID}, nil
}

// --- 结算单 ---

// RevenueSplitSettlement 一张结算单（一次线下手工打款）。
type RevenueSplitSettlement struct {
	ID                int64      `json:"id"`
	BeneficiaryUserID int64      `json:"beneficiary_user_id"`
	BeneficiaryName   string     `json:"beneficiary_name"`
	PeriodStart       *time.Time `json:"period_start"`
	PeriodEnd         *time.Time `json:"period_end"`
	EntryCount        int        `json:"entry_count"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	Method            string     `json:"method"`
	Reference         string     `json:"reference"`
	Note              string     `json:"note"`
	PaidAt            *time.Time `json:"paid_at"`
	PaidBy            *int64     `json:"paid_by"`
	CreatedBy         *int64     `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

const revenueSplitSettlementSelectSQL = `
SELECT s.id, s.beneficiary_user_id, COALESCE(NULLIF(s.beneficiary_name, ''), COALESCE(u.username, ''), ''),
       s.period_start, s.period_end, s.entry_count, s.amount::double precision, s.currency,
       s.status, COALESCE(s.method, ''), COALESCE(s.reference, ''), COALESCE(s.note, ''),
       s.paid_at, s.paid_by, s.created_by, s.created_at, s.updated_at
FROM revenue_split_settlements s
LEFT JOIN users u ON u.id = s.beneficiary_user_id`

func scanRevenueSplitSettlement(scan func(dest ...any) error) (RevenueSplitSettlement, error) {
	var item RevenueSplitSettlement
	err := scan(&item.ID, &item.BeneficiaryUserID, &item.BeneficiaryName, &item.PeriodStart, &item.PeriodEnd,
		&item.EntryCount, &item.Amount, &item.Currency, &item.Status, &item.Method, &item.Reference,
		&item.Note, &item.PaidAt, &item.PaidBy, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

// ListSettlements 分页查询结算单。
func (s *RevenueSplitService) ListSettlements(ctx context.Context, beneficiaryUserID int64, status string, page, pageSize int) ([]RevenueSplitSettlement, int64, error) {
	if s == nil || s.client == nil {
		return nil, 0, nil
	}
	page, pageSize = normalizeRevenueSplitPage(page, pageSize)
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if beneficiaryUserID > 0 {
		args = append(args, beneficiaryUserID)
		conditions = append(conditions, fmt.Sprintf("s.beneficiary_user_id = $%d", len(args)))
	}
	if st := strings.TrimSpace(status); st != "" {
		args = append(args, st)
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", len(args)))
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	client := s.clientFor(ctx)
	var total int64
	if err := revenueSplitQueryScalar(ctx, client,
		`SELECT COUNT(*) FROM revenue_split_settlements s `+where, args, &total); err != nil {
		return nil, 0, err
	}
	pageArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := client.QueryContext(ctx,
		revenueSplitSettlementSelectSQL+" "+where+
			fmt.Sprintf(" ORDER BY s.id DESC LIMIT $%d OFFSET $%d", len(pageArgs)-1, len(pageArgs)), pageArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list revenue split settlements: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]RevenueSplitSettlement, 0, pageSize)
	for rows.Next() {
		item, err := scanRevenueSplitSettlement(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scan revenue split settlement: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate revenue split settlements: %w", err)
	}
	return out, total, nil
}

// RevenueSplitSettlementInput 生成结算单入参。
type RevenueSplitSettlementInput struct {
	BeneficiaryUserID int64      `json:"beneficiary_user_id"`
	PeriodStart       *time.Time `json:"period_start"`
	PeriodEnd         *time.Time `json:"period_end"`
	Method            string     `json:"method"`
	Note              string     `json:"note"`
}

// RevenueSplitSettlementPaidInput 标记已打款入参。
type RevenueSplitSettlementPaidInput struct {
	Method    string `json:"method"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

// GetSettlement 读取一张结算单。
func (s *RevenueSplitService) GetSettlement(ctx context.Context, id int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.client == nil {
		return nil, infraerrors.NotFound("SETTLEMENT_NOT_FOUND", "settlement not found")
	}
	rows, err := s.clientFor(ctx).QueryContext(ctx, revenueSplitSettlementSelectSQL+" WHERE s.id = $1 LIMIT 1", id)
	if err != nil {
		return nil, fmt.Errorf("get revenue split settlement: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("get revenue split settlement: %w", err)
		}
		return nil, infraerrors.NotFound("SETTLEMENT_NOT_FOUND", "settlement not found")
	}
	item, err := scanRevenueSplitSettlement(rows.Scan)
	if err != nil {
		return nil, fmt.Errorf("scan revenue split settlement: %w", err)
	}
	return &item, nil
}

// CreateSettlement 把某受益人当前「可结算」的分录汇总成一张 draft 结算单，
// 并锁定这些分录（写入 settlement_id），防止重复结算。
func (s *RevenueSplitService) CreateSettlement(ctx context.Context, in RevenueSplitSettlementInput, actorID int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.client == nil {
		return nil, infraerrors.BadRequest("REVENUE_SPLIT_UNAVAILABLE", "revenue split service is unavailable")
	}
	if in.BeneficiaryUserID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_BENEFICIARY", "beneficiary_user_id is required")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin settlement tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()

	conditions := []string{"beneficiary_user_id = $1", "status = $2", "settlement_id IS NULL"}
	args := []any{in.BeneficiaryUserID, RevenueSplitEntryPending}
	if in.PeriodStart != nil {
		args = append(args, *in.PeriodStart)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if in.PeriodEnd != nil {
		args = append(args, *in.PeriodEnd)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	where := strings.Join(conditions, " AND ")

	var count int
	var amount float64
	if err := revenueSplitQueryScalar(ctx, txClient,
		`SELECT COUNT(*)::int, COALESCE(SUM(split_amount), 0)::double precision FROM revenue_split_entries WHERE `+where,
		args, &count, &amount); err != nil {
		return nil, err
	}
	if count == 0 || amount <= 0 {
		return nil, infraerrors.BadRequest("NOTHING_TO_SETTLE", "no pending revenue split entries for this beneficiary")
	}

	var name string
	var username string
	if err := revenueSplitQueryScalar(ctx, txClient,
		`SELECT COALESCE(username, '') FROM users WHERE id = $1`, []any{in.BeneficiaryUserID}, &username); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err := revenueSplitQueryScalar(ctx, txClient,
		`SELECT COALESCE(NULLIF(COALESCE(MAX(beneficiary_name), ''), ''), '')
		   FROM revenue_split_entries WHERE beneficiary_user_id = $1`,
		[]any{in.BeneficiaryUserID}, &name); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = username
	}

	insertArgs := []any{in.BeneficiaryUserID, name, in.PeriodStart, in.PeriodEnd, count,
		amount, revenueSplitCurrency, RevenueSplitSettlementDraft,
		strings.TrimSpace(in.Method), strings.TrimSpace(in.Note), nullableActorID(actorID)}
	var settlementID int64
	if err := revenueSplitQueryScalar(ctx, txClient,
		`INSERT INTO revenue_split_settlements
		   (beneficiary_user_id, beneficiary_name, period_start, period_end, entry_count, amount,
		    currency, status, method, note, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		 RETURNING id`,
		insertArgs, &settlementID); err != nil {
		return nil, fmt.Errorf("insert revenue split settlement: %w", err)
	}

	lockArgs := append(append([]any{}, args...), settlementID)
	if _, err := txClient.ExecContext(ctx,
		fmt.Sprintf(`UPDATE revenue_split_entries SET settlement_id = $%d, updated_at = NOW() WHERE %s`,
			len(lockArgs), where), lockArgs...); err != nil {
		return nil, fmt.Errorf("lock revenue split entries: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit settlement tx: %w", err)
	}
	return s.GetSettlement(ctx, settlementID)
}

// MarkSettlementPaid 标记结算单已线下打款，并把其下分录置为 settled。
func (s *RevenueSplitService) MarkSettlementPaid(ctx context.Context, id int64, in RevenueSplitSettlementPaidInput, actorID int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.client == nil {
		return nil, infraerrors.BadRequest("REVENUE_SPLIT_UNAVAILABLE", "revenue split service is unavailable")
	}
	if strings.TrimSpace(in.Reference) == "" {
		return nil, infraerrors.BadRequest("REFERENCE_REQUIRED", "payment reference is required to mark a settlement as paid")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin settlement paid tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()

	res, err := txClient.ExecContext(ctx,
		`UPDATE revenue_split_settlements
		    SET status = $2, paid_at = NOW(), paid_by = $3,
		        method = COALESCE(NULLIF($4, ''), method),
		        reference = $5,
		        note = COALESCE(NULLIF($6, ''), note),
		        updated_at = NOW()
		  WHERE id = $1 AND status = $7`,
		id, RevenueSplitSettlementPaid, nullableActorID(actorID),
		strings.TrimSpace(in.Method), strings.TrimSpace(in.Reference), strings.TrimSpace(in.Note),
		RevenueSplitSettlementDraft)
	if err != nil {
		return nil, fmt.Errorf("mark settlement paid: %w", err)
	}
	if n, aerr := res.RowsAffected(); aerr == nil && n == 0 {
		return nil, infraerrors.Conflict("SETTLEMENT_NOT_DRAFT", "settlement is not in draft status")
	}
	// 只结算仍是 pending 的分录：draft 期间若发生退款，分录已被 ReverseForOrder
	// 置为 reversed，这里绝不能把它翻回 settled，否则退款冲回会被静默抹掉。
	if _, err := txClient.ExecContext(ctx,
		`UPDATE revenue_split_entries SET status = $2, updated_at = NOW() WHERE settlement_id = $1 AND status = $3`,
		id, RevenueSplitEntrySettled, RevenueSplitEntryPending); err != nil {
		return nil, fmt.Errorf("settle revenue split entries: %w", err)
	}

	// draft 期间可能发生退款冲回，因此以真正落到 settled 的分录重新核对结算单
	// 金额与笔数，保证结算单上的数字与明细永远对得上（线下打款前必须能对上账）。
	var settledCount int
	var settledAmount float64
	if err := revenueSplitQueryScalar(ctx, txClient,
		`SELECT COUNT(*)::int, COALESCE(SUM(split_amount), 0)::double precision
		   FROM revenue_split_entries WHERE settlement_id = $1 AND status = $2`,
		[]any{id, RevenueSplitEntrySettled}, &settledCount, &settledAmount); err != nil {
		return nil, err
	}
	if _, err := txClient.ExecContext(ctx,
		`UPDATE revenue_split_settlements SET entry_count = $2, amount = $3, updated_at = NOW() WHERE id = $1`,
		id, settledCount, settledAmount); err != nil {
		return nil, fmt.Errorf("recompute settlement amount: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit settlement paid tx: %w", err)
	}
	return s.GetSettlement(ctx, id)
}

// CancelSettlement 取消一张 draft 结算单并释放其锁定的分录。
func (s *RevenueSplitService) CancelSettlement(ctx context.Context, id int64) (*RevenueSplitSettlement, error) {
	if s == nil || s.client == nil {
		return nil, infraerrors.BadRequest("REVENUE_SPLIT_UNAVAILABLE", "revenue split service is unavailable")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin settlement cancel tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()

	res, err := txClient.ExecContext(ctx,
		`UPDATE revenue_split_settlements SET status = $2, updated_at = NOW() WHERE id = $1 AND status = $3`,
		id, RevenueSplitSettlementCancelled, RevenueSplitSettlementDraft)
	if err != nil {
		return nil, fmt.Errorf("cancel settlement: %w", err)
	}
	if n, aerr := res.RowsAffected(); aerr == nil && n == 0 {
		return nil, infraerrors.Conflict("SETTLEMENT_NOT_DRAFT", "settlement is not in draft status")
	}
	if _, err := txClient.ExecContext(ctx,
		`UPDATE revenue_split_entries SET settlement_id = NULL, updated_at = NOW() WHERE settlement_id = $1`,
		id); err != nil {
		return nil, fmt.Errorf("release revenue split entries: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit settlement cancel tx: %w", err)
	}
	return s.GetSettlement(ctx, id)
}

// --- 工具函数 ---

// revenueSplitQueryScalar 执行只返回一行（或零行）的原生查询并扫描结果。
func revenueSplitQueryScalar(ctx context.Context, client revenueSplitQueryExecer, query string, args []any, dest ...any) error {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return rows.Scan(dest...)
}

// revenueSplitCashAmount 取订单的现金口径金额（支付币种）。
func revenueSplitCashAmount(order *dbent.PaymentOrder) float64 {
	if order == nil {
		return 0
	}
	if order.PayAmount > 0 {
		return order.PayAmount
	}
	if order.Amount > 0 {
		return order.Amount
	}
	return 0
}

// revenueSplitBaseAmount 按配置口径计算分账基数与通道费金额。
//
// 推导：平台实收现金 ≈ 客户实付 × (1 - 通道费率)。
// 由于余额入账额与客户实付额同比例，把实收现金换算回「记账单位」后，
// 恰好等于 pay_amount × (1 - f)，与订单是否把手续费转嫁给客户无关。
func revenueSplitBaseAmount(cash float64, cfg RevenueSplitConfig) (float64, float64) {
	if cash <= 0 {
		return 0, 0
	}
	d := decimal.NewFromFloat(cash).Round(revenueSplitFractionDigits)
	if cfg.BaseMode != RevenueSplitBaseModeGrossAfterFee || cfg.ChannelFeePercent <= 0 {
		return d.InexactFloat64(), 0
	}
	fee := d.Mul(decimal.NewFromFloat(cfg.ChannelFeePercent)).
		Div(decimal.NewFromInt(100)).
		Round(revenueSplitFractionDigits)
	base := d.Sub(fee)
	if base.IsNegative() {
		base = decimal.Zero
	}
	return base.InexactFloat64(), fee.InexactFloat64()
}

// RevenueSplitPreviewBaseAmount 暴露基数计算供管理端「试算」使用。
func RevenueSplitPreviewBaseAmount(cash float64, cfg RevenueSplitConfig) (float64, float64) {
	return revenueSplitBaseAmount(cash, cfg)
}

// RevenueSplitPreviewSplitAmount 暴露单受益人金额计算供管理端「试算」使用。
func RevenueSplitPreviewSplitAmount(base, ratioPercent float64) float64 {
	return revenueSplitSplitAmount(base, ratioPercent)
}

// revenueSplitSplitAmount 计算单个受益人的应分金额。
func revenueSplitSplitAmount(base, ratioPercent float64) float64 {
	if base <= 0 || ratioPercent <= 0 {
		return 0
	}
	return decimal.NewFromFloat(base).
		Mul(decimal.NewFromFloat(ratioPercent)).
		Div(decimal.NewFromInt(100)).
		Round(revenueSplitFractionDigits).
		InexactFloat64()
}

func nullableActorID(actorID int64) any {
	if actorID <= 0 {
		return nil
	}
	return actorID
}

func boolToSettingValue(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func formatFloatLoose(v float64) string {
	return decimal.NewFromFloat(v).String()
}

func parseFloatLoose(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty number")
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return 0, err
	}
	return d.InexactFloat64(), nil
}
