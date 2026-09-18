//go:build unit

package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

// --- 规则校验 ---------------------------------------------------------------

func TestValidateRevenueSplitRules(t *testing.T) {
	t.Parallel()

	disabled := false

	tests := []struct {
		name    string
		rules   []RevenueSplitRuleInput
		wantErr string // infraerrors.Reason；空表示期望通过
	}{
		{
			name: "30/70 是合法的（用户最初想要的分法）",
			rules: []RevenueSplitRuleInput{
				{BeneficiaryUserID: 1, RatioPercent: 30},
				{BeneficiaryUserID: 2, RatioPercent: 70},
			},
		},
		{
			name: "启用比例之和恰好 100 允许通过",
			rules: []RevenueSplitRuleInput{
				{BeneficiaryUserID: 1, RatioPercent: 99.5},
				{BeneficiaryUserID: 2, RatioPercent: 0.5},
			},
		},
		{
			name: "启用比例之和超过 100 报 REVENUE_SPLIT_OVERFLOW",
			rules: []RevenueSplitRuleInput{
				{BeneficiaryUserID: 1, RatioPercent: 60},
				{BeneficiaryUserID: 2, RatioPercent: 50},
			},
			wantErr: "REVENUE_SPLIT_OVERFLOW",
		},
		{
			name: "停用的规则不计入总和",
			rules: []RevenueSplitRuleInput{
				{BeneficiaryUserID: 1, RatioPercent: 80},
				{BeneficiaryUserID: 2, RatioPercent: 80, Enabled: &disabled},
			},
		},
		{
			name: "同一受益人出现两次报 DUPLICATE_BENEFICIARY",
			rules: []RevenueSplitRuleInput{
				{BeneficiaryUserID: 7, RatioPercent: 10},
				{BeneficiaryUserID: 7, RatioPercent: 20},
			},
			wantErr: "DUPLICATE_BENEFICIARY",
		},
		{
			name:    "空规则集被拒绝",
			rules:   nil,
			wantErr: "INVALID_REVENUE_SPLIT_RULES",
		},
		{
			name:    "缺少受益人报 INVALID_BENEFICIARY",
			rules:   []RevenueSplitRuleInput{{BeneficiaryUserID: 0, RatioPercent: 10}},
			wantErr: "INVALID_BENEFICIARY",
		},
		{
			name:    "比例为 0 报 INVALID_RATIO",
			rules:   []RevenueSplitRuleInput{{BeneficiaryUserID: 1, RatioPercent: 0}},
			wantErr: "INVALID_RATIO",
		},
		{
			name:    "比例超过 100 报 INVALID_RATIO",
			rules:   []RevenueSplitRuleInput{{BeneficiaryUserID: 1, RatioPercent: 100.5}},
			wantErr: "INVALID_RATIO",
		},
		{
			name:    "负数比例报 INVALID_RATIO",
			rules:   []RevenueSplitRuleInput{{BeneficiaryUserID: 1, RatioPercent: -1}},
			wantErr: "INVALID_RATIO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRevenueSplitRules(tt.rules)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tt.wantErr, infraerrors.Reason(err))
		})
	}
}

// --- 分账基数与金额 ---------------------------------------------------------

func TestRevenueSplitBaseAmount(t *testing.T) {
	t.Parallel()

	cfgAfterFee := RevenueSplitConfig{Enabled: true, BaseMode: RevenueSplitBaseModeGrossAfterFee, ChannelFeePercent: 1.6}
	cfgGross := RevenueSplitConfig{Enabled: true, BaseMode: RevenueSplitBaseModeGross, ChannelFeePercent: 1.6}
	cfgZeroFee := RevenueSplitConfig{Enabled: true, BaseMode: RevenueSplitBaseModeGrossAfterFee, ChannelFeePercent: 0}

	tests := []struct {
		name     string
		cash     float64
		cfg      RevenueSplitConfig
		wantBase float64
		wantFee  float64
	}{
		{"100 元扣 1.6% 通道费后基数为 98.40", 100, cfgAfterFee, 98.40, 1.60},
		{"gross 口径不扣费", 100, cfgGross, 100, 0},
		{"费率为 0 时不扣费", 100, cfgZeroFee, 100, 0},
		{"1 元小额：0.016 进位到 0.02，基数 0.98", 1, cfgAfterFee, 0.98, 0.02},
		{"金额为 0 返回 0", 0, cfgAfterFee, 0, 0},
		{"负数金额返回 0", -5, cfgAfterFee, 0, 0},
		{"小数金额仍按分取整", 33.33, cfgAfterFee, 32.80, 0.53},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, fee := revenueSplitBaseAmount(tt.cash, tt.cfg)
			require.InDelta(t, tt.wantBase, base, 1e-9, "base")
			require.InDelta(t, tt.wantFee, fee, 1e-9, "fee")
		})
	}
}

func TestRevenueSplitSplitAmount(t *testing.T) {
	t.Parallel()

	// 100 元订单 → 基数 98.40 → 30/70 分成
	base, fee := revenueSplitBaseAmount(100, RevenueSplitConfig{
		Enabled: true, BaseMode: RevenueSplitBaseModeGrossAfterFee, ChannelFeePercent: 1.6,
	})
	require.InDelta(t, 1.60, fee, 1e-9)

	you := revenueSplitSplitAmount(base, 30)
	other := revenueSplitSplitAmount(base, 70)
	require.InDelta(t, 29.52, you, 1e-9, "30% 应得 29.52")
	require.InDelta(t, 68.88, other, 1e-9, "70% 应得 68.88")

	// 关键不变量：分出去的总额不得超过基数，平台不能倒贴。
	require.LessOrEqual(t, you+other, base+1e-9, "分账总额不得超过基数")
	require.InDelta(t, 98.40, you+other, 1e-9, "30/70 刚好分完基数，平台余额为 0")

	require.Zero(t, revenueSplitSplitAmount(0, 30), "基数为 0 时分账为 0")
	require.Zero(t, revenueSplitSplitAmount(100, 0), "比例为 0 时分账为 0")
	require.Zero(t, revenueSplitSplitAmount(-1, 30), "负基数分账为 0")
	require.Zero(t, revenueSplitSplitAmount(100, -5), "负比例分账为 0")
}

func TestRevenueSplitFullSplitLeavesNoPlatformMargin(t *testing.T) {
	t.Parallel()

	// 这是给管理端试算页用的不变量：比例之和 = 100 时平台余额必然为 0，
	// 而上游 API 成本还没扣，所以真实配置必须留出空间。
	cfg := RevenueSplitConfig{Enabled: true, BaseMode: RevenueSplitBaseModeGrossAfterFee, ChannelFeePercent: 1.6}
	base, _ := revenueSplitBaseAmount(100, cfg)

	allocated := revenueSplitSplitAmount(base, 30) + revenueSplitSplitAmount(base, 70)
	require.InDelta(t, 0, base-allocated, 1e-9, "分满 100% 时平台留存为 0")

	allocated80 := revenueSplitSplitAmount(base, 30) + revenueSplitSplitAmount(base, 50)
	require.InDelta(t, 19.68, base-allocated80, 1e-9, "只分 80% 时平台留存 19.68")
}

func TestRevenueSplitCashAmountPrefersPayAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		order *dbent.PaymentOrder
		want  float64
	}{
		{"nil 订单返回 0", nil, 0},
		{"有实付金额时优先用实付", &dbent.PaymentOrder{Amount: 100, PayAmount: 99}, 99},
		{"没有实付金额时回落到订单金额", &dbent.PaymentOrder{Amount: 100}, 100},
		{"两者都为 0 返回 0", &dbent.PaymentOrder{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.InDelta(t, tt.want, revenueSplitCashAmount(tt.order), 1e-12)
		})
	}
}

// --- 配置规范化 -------------------------------------------------------------

func TestNormalizeRevenueSplitBaseMode(t *testing.T) {
	t.Parallel()

	require.Equal(t, RevenueSplitBaseModeGross, normalizeRevenueSplitBaseMode("gross"))
	require.Equal(t, RevenueSplitBaseModeGross, normalizeRevenueSplitBaseMode("  gross  "))
	require.Equal(t, RevenueSplitBaseModeGrossAfterFee, normalizeRevenueSplitBaseMode("gross_after_fee"))
	require.Equal(t, RevenueSplitBaseModeGrossAfterFee, normalizeRevenueSplitBaseMode(""))
	require.Equal(t, RevenueSplitBaseModeGrossAfterFee, normalizeRevenueSplitBaseMode("whatever"))
}

func TestNormalizeRevenueSplitChannelFeePercent(t *testing.T) {
	t.Parallel()

	require.InDelta(t, 1.6, normalizeRevenueSplitChannelFeePercent(1.6), 1e-12)
	require.InDelta(t, 0, normalizeRevenueSplitChannelFeePercent(0), 1e-12)
	require.InDelta(t, 100, normalizeRevenueSplitChannelFeePercent(100), 1e-12)
	require.InDelta(t, defaultRevenueSplitChannelFeePercent, normalizeRevenueSplitChannelFeePercent(-1), 1e-12)
	require.InDelta(t, defaultRevenueSplitChannelFeePercent, normalizeRevenueSplitChannelFeePercent(101), 1e-12)
}

func TestDefaultRevenueSplitConfigIsDisabled(t *testing.T) {
	t.Parallel()

	cfg := DefaultRevenueSplitConfig()
	require.False(t, cfg.Enabled, "默认必须关闭：没配好比例前不能静默计提")
	require.Equal(t, RevenueSplitBaseModeGrossAfterFee, cfg.BaseMode)
	require.InDelta(t, 1.6, cfg.ChannelFeePercent, 1e-12)
}

// --- 无依赖路径 -------------------------------------------------------------

func TestNewRevenueSplitServicePanicsOnNilClient(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { NewRevenueSplitService(nil) })
}

func TestRevenueSplitAccrueToleratesNilInputs(t *testing.T) {
	t.Parallel()

	svc := &RevenueSplitService{}
	created, err := svc.AccrueForOrder(context.Background(), nil)
	require.NoError(t, err)
	require.Zero(t, created)

	created, err = svc.AccrueForOrder(context.Background(), &dbent.PaymentOrder{ID: 1, Amount: 100})
	require.NoError(t, err)
	require.Zero(t, created)
}

func TestRevenueSplitReverseForOrderIgnoresInvalidOrderID(t *testing.T) {
	t.Parallel()

	svc := &RevenueSplitService{}
	for _, orderID := range []int64{0, -1} {
		n, err := svc.ReverseForOrder(context.Background(), orderID, "test")
		require.NoError(t, err)
		require.Zero(t, n)
	}
}

func TestPaymentServiceRepairRevenueSplitRejectsInvalidOrderID(t *testing.T) {
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{
		entClient:    client,
		revenueSplit: NewRevenueSplitService(client),
	}
	for _, orderID := range []int64{0, -1} {
		_, err := svc.RepairRevenueSplitForOrder(context.Background(), orderID)
		require.Error(t, err)
		require.Equal(t, "INVALID_ORDER_ID", infraerrors.Reason(err))
	}
}

func TestPaymentServiceRevenueSplitWrappersDegradeWithoutService(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := &PaymentService{}

	cfg, err := svc.GetRevenueSplitConfig(ctx)
	require.NoError(t, err)
	require.False(t, cfg.Enabled)

	rules, err := svc.ListRevenueSplitRules(ctx)
	require.NoError(t, err)
	require.Empty(t, rules)

	entries, total, err := svc.ListRevenueSplitEntries(ctx, RevenueSplitEntryFilter{})
	require.NoError(t, err)
	require.Empty(t, entries)
	require.Zero(t, total)

	summary, err := svc.RevenueSplitSummary(ctx)
	require.NoError(t, err)
	require.Empty(t, summary)

	mine, err := svc.MyRevenueSplit(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, mine)
	require.Zero(t, mine.PendingAmount)

	_, err = svc.RevenueSplitPreview(ctx, 100)
	require.Error(t, err, "没有分账服务时试算必须报错，而不是给出误导性数字")

	_, err = svc.UpdateRevenueSplitConfig(ctx, RevenueSplitConfigInput{})
	require.Error(t, err)
}

// --- 小工具 -----------------------------------------------------------------

func TestRevenueSplitSettingHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, "true", boolToSettingValue(true))
	require.Equal(t, "false", boolToSettingValue(false))
	require.Nil(t, nullableActorID(0))
	require.Equal(t, int64(9), nullableActorID(9))

	v, err := parseFloatLoose(" 1.6 ")
	require.NoError(t, err)
	require.InDelta(t, 1.6, v, 1e-12)

	_, err = parseFloatLoose("   ")
	require.Error(t, err)
	_, err = parseFloatLoose("abc")
	require.Error(t, err)

	require.Equal(t, "1.6", formatFloatLoose(1.6))
}

// --- 明细查询条件 -----------------------------------------------------------

// 结算单对账单必须能精确取到该结算单下的全部分录。
// 曾经的实现是"按受益人取前 200 条再内存过滤"，明细一多就会漏掉旧结算单，
// 这里锁住 settlement_id 过滤条件的生成方式，防止回归。
func TestRevenueSplitEntryWhereSettlementFilter(t *testing.T) {
	t.Parallel()

	t.Run("未指定结算单时不产生 settlement_id 条件", func(t *testing.T) {
		t.Parallel()
		where, args := revenueSplitEntryWhere(RevenueSplitEntryFilter{BeneficiaryUserID: 9})
		require.NotContains(t, where, "e.settlement_id")
		require.Equal(t, []any{int64(9)}, args)
	})

	t.Run("指定结算单时按 settlement_id 精确过滤", func(t *testing.T) {
		t.Parallel()
		where, args := revenueSplitEntryWhere(RevenueSplitEntryFilter{SettlementID: 42})
		require.Contains(t, where, "e.settlement_id = $1")
		require.Equal(t, []any{int64(42)}, args)
	})

	t.Run("多条件叠加时占位符序号正确", func(t *testing.T) {
		t.Parallel()
		where, args := revenueSplitEntryWhere(RevenueSplitEntryFilter{
			BeneficiaryUserID: 3,
			Status:            RevenueSplitEntryPending,
			SettlementID:      7,
		})
		require.Contains(t, where, "e.beneficiary_user_id = $1")
		require.Contains(t, where, "e.status = $2")
		require.Contains(t, where, "e.settlement_id = $3")
		require.Equal(t, []any{int64(3), RevenueSplitEntryPending, int64(7)}, args)
	})
}
