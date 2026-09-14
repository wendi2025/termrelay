package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 退款预览 / 余额账本路由的注册冒烟测试。
//
// gin 在"注册阶段"就会对同层级冲突的路径 panic（静态段与参数段混用），
// 这类错误构建期发现不了，只会在进程启动时炸掉，因此必须用测试兜住。
// 同时断言前端依赖的三个新端点确实进了路由表，避免路径写错时静默 404。
func TestRegisterPaymentRoutesRegistersRefundPreviewAndLedger(t *testing.T) {
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
		// 用户视角：退款预览（方案 9.1/9.2/9.3 明细）
		"GET /api/v1/payment/orders/:id/refund-preview",
		// 用户视角：余额分账账本
		"GET /api/v1/payment/ledger",
		// 管理员视角：退款预览（含 force 预览）
		"GET /api/v1/admin/payment/orders/:id/refund-preview",
		// 既有端点不能被挤掉（静态段 /my 与参数段 /:id 必须共存）
		"GET /api/v1/payment/orders/my",
		"GET /api/v1/payment/orders/:id",
		"POST /api/v1/payment/orders/:id/refund-request",
		"GET /api/v1/payment/orders/refund-eligible-providers",
		"POST /api/v1/admin/payment/orders/:id/refund",
	} {
		require.True(t, registered[want], "路由未注册: %s", want)
	}
}
