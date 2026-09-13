package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Group holds the schema definition for the Group entity.
type Group struct {
	ent.Schema
}

func (Group) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "groups"},
	}
}

func (Group) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Group) Fields() []ent.Field {
	return []ent.Field{
		// 唯一约束通过部分索引实现（WHERE deleted_at IS NULL），支持软删除后重用
		// 见迁移文件 016_soft_delete_partial_unique_indexes.sql
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Float("rate_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(1.0),
		// 高峰时段倍率（added by migration 158）
		field.Bool("peak_rate_enabled").
			Default(false).
			Comment("是否启用高峰时段倍率"),
		field.String("peak_start").
			MaxLen(5).
			Default("").
			Comment("高峰开始时间 HH:MM（含），如 14:00；空表示未配置；不支持跨天"),
		field.String("peak_end").
			MaxLen(5).
			Default("").
			Comment("高峰结束时间 HH:MM（不含），必须大于 peak_start；不支持跨天，如 22:00-02:00"),
		field.Float("peak_rate_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(1.0).
			Comment("高峰时段叠加倍率，仅在 peak_rate_enabled 且处于 [peak_start, peak_end) 时乘入文本倍率"),
		field.Bool("is_exclusive").
			Default(false),
		field.String("status").
			MaxLen(20).
			Default(domain.StatusActive),
		field.String("duplicate_operation_id").
			MaxLen(64).
			Optional().
			Nillable().
			Immutable().
			Comment("内部幂等恢复标识，不对 API 暴露"),

		// Subscription-related fields (added by migration 003)
		field.String("platform").
			MaxLen(50).
			Default(domain.PlatformAnthropic),
		field.String("subscription_type").
			MaxLen(20).
			Default(domain.SubscriptionTypeStandard),
		field.Float("daily_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("weekly_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("monthly_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int("default_validity_days").
			Default(30),

		// 图片生成计费配置（antigravity 和 gemini 平台使用）
		field.Bool("allow_image_generation").
			Default(false).
			Comment("是否允许该分组使用图片生成能力"),
		field.Bool("allow_batch_image_generation").
			Default(false).
			Comment("是否允许该分组使用批量图片生成能力"),
		field.Bool("image_rate_independent").
			Default(false).
			Comment("图片生成是否使用独立倍率；false 表示共享分组有效倍率"),
		field.Float("image_rate_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(1.0).
			Comment("图片生成独立倍率，仅 image_rate_independent=true 时生效"),
		field.Float("image_price_1k").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("image_price_2k").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("image_price_4k").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("batch_image_discount_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(0.5).
			Comment("批量图片生成折扣倍率，最终单价会乘以该值；0 表示免费"),
		field.Float("batch_image_hold_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(0.6).
			Comment("批量图片生成冻结价格比例，按普通生图原价乘以该比例冻结，结算后释放差额"),
		field.Bool("video_rate_independent").
			Default(false).
			Comment("视频生成是否使用独立倍率；false 表示共享分组有效倍率"),
		field.Float("video_rate_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(1.0).
			Comment("视频生成独立倍率，仅 video_rate_independent=true 时生效"),
		field.Float("video_price_480p").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("video_price_720p").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("video_price_1080p").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("web_search_price_per_call").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Comment("Codex alpha/search 网页搜索单次价格（USD/次）；nil 表示使用默认价 0.01（官方 $10/1000 次）"),

		// Claude Code 客户端限制 (added by migration 029)
		field.Bool("claude_code_only").
			Default(false).
			Comment("是否仅允许 Claude Code 客户端"),
		field.Int64("fallback_group_id").
			Optional().
			Nillable().
			Comment("非 Claude Code 请求降级使用的分组 ID"),
		field.Int64("fallback_group_id_on_invalid_request").
			Optional().
			Nillable().
			Comment("无效请求兜底使用的分组 ID"),

		// 模型路由配置 (added by migration 040)
		field.JSON("model_routing", map[string][]int64{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("模型路由配置：模型模式 -> 优先账号ID列表"),

		// 模型路由开关 (added by migration 041)
		field.Bool("model_routing_enabled").
			Default(false).
			Comment("是否启用模型路由配置"),

		// MCP XML 协议注入开关 (added by migration 042)
		field.Bool("mcp_xml_inject").
			Default(true).
			Comment("是否注入 MCP XML 调用协议提示词（仅 antigravity 平台）"),

		// 支持的模型系列 (added by migration 046)
		field.JSON("supported_model_scopes", []string{}).
			Default([]string{"claude", "gemini_text", "gemini_image"}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("支持的模型系列：claude, gemini_text, gemini_image"),

		// 分组排序 (added by migration 052)
		field.Int("sort_order").
			Default(0).
			Comment("分组显示排序，数值越小越靠前"),

		// OpenAI Messages 调度配置 (added by migration 069)
		field.Bool("allow_messages_dispatch").
			Default(false).
			Comment("是否允许 /v1/messages 调度到此 OpenAI 分组"),
		field.Bool("allow_live").
			Default(false).
			Comment("是否允许此 OpenAI 分组访问 Live 接口"),
		field.Bool("require_oauth_only").
			Default(false).
			Comment("仅允许非 apikey 类型账号关联到此分组"),
		field.Bool("require_privacy_set").
			Default(false).
			Comment("调度时仅允许 privacy 已成功设置的账号"),
		field.String("default_mapped_model").
			MaxLen(100).
			Default("").
			Comment("默认映射模型 ID，当账号级映射找不到时使用此值"),
		field.JSON("messages_dispatch_model_config", domain.OpenAIMessagesDispatchModelConfig{}).
			Default(domain.OpenAIMessagesDispatchModelConfig{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("OpenAI Messages 调度模型配置：按 Claude 系列/精确模型映射到目标 GPT 模型"),
		field.JSON("models_list_config", domain.GroupModelsListConfig{}).
			Default(domain.GroupModelsListConfig{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("自定义 /v1/models 展示列表配置；仅影响模型列表响应，不影响调度"),

		// 分组级每分钟请求数上限（0 = 不限制）。设置后优先于用户级兜底生效。
		field.Int("rpm_limit").
			Default(0).
			Comment("分组 RPM 上限，0 表示不限制；设置后接管该分组用户的限流"),

		// OpenAI/Codex 请求的推理强度上限（空字符串表示不限制）。
		field.String("max_reasoning_effort").
			MaxLen(20).
			Default("").
			Comment("OpenAI reasoning effort 上限；可选 minimal/low/medium/high/xhigh/max"),
		field.JSON("reasoning_effort_mappings", []domain.ReasoningEffortMapping{}).
			Default([]domain.ReasoningEffortMapping{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("OpenAI reasoning effort 自定义精确映射；先映射再应用上限"),

		// ===== Smirel 订阅/定价扩展字段（migration 193） =====
		// 产品线标识（与 platform 字段不同：platform 表示上游账号类型，product_line 表示面向用户的产品线）
		// 取值：gpt_plus / gpt_pro / claude_no_fable / claude_fable / domestic / custom
		field.String("product_line").
			MaxLen(32).
			Default("custom").
			Comment("产品线标识：gpt_plus/gpt_pro/claude_no_fable/claude_fable/domestic/custom"),

		// 上游成本倍率（¥/官方$）：用于财务压力测试与盈亏平衡计算
		field.Float("cost_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(0.10).
			Comment("上游成本倍率（¥/官方$），用于毛利率与盈亏平衡测算"),

		// 按量计费对外报价（¥/官方$）：用户充值后按官方 × 此值扣费
		field.Float("pay_as_you_go_price_per_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(0.50).
			Comment("按量计费对外报价（¥/官方$），充值余额扣费时使用"),

		// 综合损耗系数（用于财务测算）：0.80 表示 80% 的额度能成功兑付
		field.Float("loss_coefficient").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(0.80).
			Comment("综合损耗系数，用于财务压力测试与盈亏平衡测算"),

		// 最高折扣比例（0~1）：大客折扣、返利等不与该字段叠加
		field.Float("max_discount_pct").
			SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}).
			Default(0.15).
			Comment("最高折扣比例，返利和大客折扣不叠加"),

		// 并发数限制（0 表示不限制）
		field.Int("concurrency_limit").
			Default(0).
			Comment("该分组用户最大并发数，0 表示不限制；与 user_allowed_groups 配合做白名单"),

		// 是否启用异常熔断（基于错误率/RPM 抖动自动降级）
		field.Bool("circuit_breaker_enabled").
			Default(false).
			Comment("是否启用异常熔断；启用后上游错误率过高时自动暂停调度"),

		// 是否独享 quota（不与其他用户/分组共享上游账号池）
		field.Bool("exclusive_quota").
			Default(false).
			Comment("是否独享 quota（企业试运行档用，不与其他用户共享上游账号）"),

		// 白名单销售：仅允许 user_allowed_groups 中存在的用户购买
		field.Bool("whitelist_only").
			Default(false).
			Comment("白名单销售：仅 user_allowed_groups 白名单内用户可订阅"),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("api_keys", APIKey.Type),
		edge.To("redeem_codes", RedeemCode.Type),
		edge.To("subscriptions", UserSubscription.Type),
		edge.To("usage_logs", UsageLog.Type),
		edge.From("accounts", Account.Type).
			Ref("groups").
			Through("account_groups", AccountGroup.Type),
		edge.From("allowed_users", User.Type).
			Ref("allowed_groups").
			Through("user_allowed_groups", UserAllowedGroup.Type),
		// 注意：fallback_group_id 直接作为字段使用，不定义 edge
		// 这样允许多个分组指向同一个降级分组（M2O 关系）
	}
}

func (Group) Indexes() []ent.Index {
	return []ent.Index{
		// name 字段已在 Fields() 中声明 Unique()，无需重复索引
		index.Fields("status"),
		index.Fields("platform"),
		index.Fields("subscription_type"),
		index.Fields("is_exclusive"),
		index.Fields("deleted_at"),
		index.Fields("sort_order"),
		index.Fields("duplicate_operation_id").
			Unique().
			StorageKey("idx_groups_duplicate_operation_id_active").
			Annotations(entsql.IndexWhere("duplicate_operation_id IS NOT NULL AND deleted_at IS NULL")),
	}
}
