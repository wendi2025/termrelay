package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserBalanceLedger holds the schema definition for the UserBalanceLedger entity.
//
// 充值本金/赠送余额分账表（Smirel 退款政策要求）：
//   - 充值订单的"本金"和"赠送余额"分账记录
//   - 每次调用的扣费归属到具体订单本金/赠送
//   - 支持按订单退款（FIFO 原则）
//   - 退款时该订单的未使用赠送余额、未生效优惠和关联权益一并取消
type UserBalanceLedger struct {
	ent.Schema
}

func (UserBalanceLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_balance_ledger"},
	}
}

func (UserBalanceLedger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (UserBalanceLedger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		// 关联的充值订单
		field.Int64("order_id"),
		// 余额类型：principal（充值本金）/ bonus（赠送余额）
		field.String("entry_type").
			MaxLen(16).
			Default("principal"),
		// 变动方向：credit（入账）/ debit（扣减）
		field.String("direction").
			MaxLen(8).
			Default("credit"),
		// 变动金额（始终为正数；方向由 direction 字段决定）
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		// 变动后的剩余余额（冗余，便于审计）
		field.Float("balance_after").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		// 备注/退款关联 ID/原始扣费 usage_log_id
		field.String("memo").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		// 该订单是否已冻结（退款完成后清零）
		field.Bool("frozen").
			Default(false),
		// 退款批次 ID（同一笔退款的多个分录使用同一 batch_id）
		field.String("refund_batch_id").
			Optional().
			Nillable().
			MaxLen(64),
	}
}

func (UserBalanceLedger) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("balance_ledger").
			Field("user_id").
			Unique().
			Required(),
		// M2O：每条账本分录指向一笔订单；订单侧不设 Unique，故整体是 O2M，
		// 允许一笔订单产生多条分录（本金入账、赠送入账、多次扣费归属）。
		edge.From("order", PaymentOrder.Type).
			Ref("balance_ledger").
			Field("order_id").
			Unique().
			Required(),
	}
}

func (UserBalanceLedger) Indexes() []ent.Index {
	return []ent.Index{
		// FIFO 扣费所需：按用户 + 时间排序
		index.Fields("user_id", "created_at"),
		// 按订单查询（退款时定位订单的所有分录）
		index.Fields("order_id"),
		// 按用户 + 类型 + 冻结状态查询（计算可用余额）
		index.Fields("user_id", "entry_type", "frozen"),
		// 退款批次查询
		index.Fields("refund_batch_id"),
	}
}
