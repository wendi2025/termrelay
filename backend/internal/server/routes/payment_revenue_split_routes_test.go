package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 共建者分账路由的注册冒烟测试。
//
// gin 在"注册阶段"就会对同层级冲突的路径 panic（静态段与参数段混用），
// 这类错误构建期发现不了，只会在进程启动时炸掉，因此必须用测试兜住。
// /revenue-split 下同时挂了静态段（config/rules/entries/summary/settlements）
// 与参数段（settlements/:id、orders/:id/accrue），正是最容易踩坑的组合。
func TestRegisterPaymentRoutesRegistersRevenueSplit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	noop := gin.HandlerFunc(func(c *gin.Context) { c.Next() })

	engine := gin.New()
	require.NotPanics(t, func() {
		RegisterPaymentRoutes(
			engine.Group("/api/v1"),
			handler.NewPaymentHandler(nil, nil),
			handler.NewPaymentWebhookHandler(nil, nil),
			adminhandler.NewPaymentHandler(nil, nil),
			servermiddleware.JWTAuthMiddleware(noop),
			servermiddleware.AdminAuthMiddleware(noop),
			servermiddleware.AuditLogMiddleware(noop),
			nil,
			servermiddleware.NewPanelRateLimiter(nil, nil),
		)
	})

	registered := map[string]bool{}
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	for _, want := range []string{
		// 用户视角：共建者自助查看自己的分账（只读）
		"GET /api/v1/payment/revenue-split",

		// 管理端：配置与规则
		"GET /api/v1/admin/payment/revenue-split/config",
		"PUT /api/v1/admin/payment/revenue-split/config",
		"GET /api/v1/admin/payment/revenue-split/rules",
		"PUT /api/v1/admin/payment/revenue-split/rules",
		"POST /api/v1/admin/payment/revenue-split/preview",

		// 管理端：明细与汇总
		"GET /api/v1/admin/payment/revenue-split/entries",
		"GET /api/v1/admin/payment/revenue-split/summary",

		// 管理端：结算单（静态段 /settlements 必须与参数段 /settlements/:id 共存）
		"GET /api/v1/admin/payment/revenue-split/settlements",
		"POST /api/v1/admin/payment/revenue-split/settlements",
		"GET /api/v1/admin/payment/revenue-split/settlements/:id",
		"POST /api/v1/admin/payment/revenue-split/settlements/:id/pay",
		"POST /api/v1/admin/payment/revenue-split/settlements/:id/cancel",

		// 管理端：历史订单补计提
		"POST /api/v1/admin/payment/revenue-split/orders/:id/accrue",

		// 既有端点不能被挤掉
		"GET /api/v1/payment/ledger",
		"GET /api/v1/admin/payment/orders/:id",
		"POST /api/v1/admin/payment/orders/:id/simulate-paid",
	} {
		require.True(t, registered[want], "路由未注册: %s", want)
	}
}
