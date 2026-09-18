package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPaymentRoutes registers all payment-related routes:
// user-facing endpoints, webhook endpoints, and admin endpoints.
func RegisterPaymentRoutes(
	v1 *gin.RouterGroup,
	paymentHandler *handler.PaymentHandler,
	webhookHandler *handler.PaymentWebhookHandler,
	adminPaymentHandler *admin.PaymentHandler,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	// --- User-facing payment endpoints (authenticated) ---
	authenticated := v1.Group("/payment")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	// 面板全局按用户限流
	authenticated.Use(panelRateLimiter.Global())
	{
		authenticated.GET("/config", paymentHandler.GetPaymentConfig)
		authenticated.GET("/checkout-info", paymentHandler.GetCheckoutInfo)
		authenticated.GET("/plans", paymentHandler.GetPlans)
		authenticated.GET("/limits", paymentHandler.GetLimits)

		orders := authenticated.Group("/orders")
		{
			orders.POST("", paymentHandler.CreateOrder)
			orders.POST("/verify", paymentHandler.VerifyOrder)
			orders.GET("/my", paymentHandler.GetMyOrders)
			orders.GET("/:id", paymentHandler.GetOrder)
			orders.POST("/:id/cancel", paymentHandler.CancelOrder)
			orders.POST("/:id/refund-request", paymentHandler.RequestRefund)
			orders.GET("/refund-eligible-providers", paymentHandler.GetRefundEligibleProviders)
			orders.GET("/:id/refund-preview", paymentHandler.GetRefundPreview)
		}

		// 余额分账账本（方案 9.2：充值本金 / 赠送余额 / 扣费归属）
		authenticated.GET("/ledger", paymentHandler.GetBalanceLedger)

		// 共建者分账（受益人自助查看自己的待结算 / 已结算金额，只读）
		authenticated.GET("/revenue-split", paymentHandler.GetMyRevenueSplit)
	}

	// --- Public payment endpoints (no auth) ---
	// Signed resume-token recovery is the preferred public lookup path.
	// The legacy anonymous out_trade_no verify endpoint remains available as a
	// persisted-state compatibility path for staggered upgrades.
	public := v1.Group("/payment/public")
	{
		public.POST("/orders/verify", paymentHandler.VerifyOrderPublic)
		public.POST("/orders/resolve", paymentHandler.ResolveOrderPublicByResumeToken)
	}

	// --- Webhook endpoints (no auth) ---
	webhook := v1.Group("/payment/webhook")
	{
		// EasyPay sends GET callbacks with query params
		webhook.GET("/easypay", webhookHandler.EasyPayNotify)
		webhook.POST("/easypay", webhookHandler.EasyPayNotify)
		webhook.POST("/alipay", webhookHandler.AlipayNotify)
		webhook.POST("/wxpay", webhookHandler.WxpayNotify)
		webhook.POST("/stripe", webhookHandler.StripeWebhook)
		webhook.POST("/airwallex", webhookHandler.AirwallexWebhook)
	}

	// --- Admin payment endpoints (admin auth) ---
	adminGroup := v1.Group("/admin/payment")
	adminGroup.Use(gin.HandlerFunc(adminAuth))
	adminGroup.Use(gin.HandlerFunc(auditLog))
	adminGroup.Use(middleware.AdminComplianceGuard(settingService))
	{
		// Dashboard
		adminGroup.GET("/dashboard", adminPaymentHandler.GetDashboard)

		// Config
		adminGroup.GET("/config", adminPaymentHandler.GetConfig)
		adminGroup.PUT("/config", adminPaymentHandler.UpdateConfig)

		// Orders
		adminOrders := adminGroup.Group("/orders")
		{
			adminOrders.GET("", adminPaymentHandler.ListOrders)
			adminOrders.GET("/:id", adminPaymentHandler.GetOrderDetail)
			adminOrders.POST("/:id/cancel", adminPaymentHandler.CancelOrder)
			adminOrders.POST("/:id/retry", adminPaymentHandler.RetryFulfillment)
			adminOrders.POST("/:id/simulate-paid", adminPaymentHandler.SimulatePaid)
			adminOrders.POST("/:id/refund", adminPaymentHandler.ProcessRefund)
			adminOrders.POST("/:id/refund/query", adminPaymentHandler.QueryAndFinalizeRefund)
			adminOrders.GET("/:id/refund-preview", adminPaymentHandler.GetRefundPreview)
		}

		// Subscription Plans
		plans := adminGroup.Group("/plans")
		{
			plans.GET("", adminPaymentHandler.ListPlans)
			plans.POST("", adminPaymentHandler.CreatePlan)
			plans.PUT("/:id", adminPaymentHandler.UpdatePlan)
			plans.DELETE("/:id", adminPaymentHandler.DeletePlan)
		}

		// Provider Instances
		providers := adminGroup.Group("/providers")
		{
			providers.GET("", adminPaymentHandler.ListProviders)
			providers.POST("", adminPaymentHandler.CreateProvider)
			providers.PUT("/:id", adminPaymentHandler.UpdateProvider)
			providers.DELETE("/:id", adminPaymentHandler.DeleteProvider)
		}

		// Revenue Split（共建者分账）
		//
		// 只做记账：客户付款后按比例给每个共建者计提一笔「应分金额」，
		// 钱仍整笔留在平台收款账户，由管理员线下打款后再回填流水号核销。
		// 任何"系统自动把钱转给第三方"的做法都属无牌照资金清分，禁止实现。
		split := adminGroup.Group("/revenue-split")
		{
			split.GET("/config", adminPaymentHandler.GetRevenueSplitConfig)
			split.PUT("/config", adminPaymentHandler.UpdateRevenueSplitConfig)
			split.GET("/rules", adminPaymentHandler.ListRevenueSplitRules)
			split.PUT("/rules", adminPaymentHandler.ReplaceRevenueSplitRules)
			split.POST("/preview", adminPaymentHandler.PreviewRevenueSplit)

			split.GET("/entries", adminPaymentHandler.ListRevenueSplitEntries)
			split.GET("/summary", adminPaymentHandler.RevenueSplitSummary)

			split.GET("/settlements", adminPaymentHandler.ListRevenueSplitSettlements)
			split.POST("/settlements", adminPaymentHandler.CreateRevenueSplitSettlement)
			split.GET("/settlements/:id", adminPaymentHandler.GetRevenueSplitSettlement)
			split.POST("/settlements/:id/pay", adminPaymentHandler.MarkRevenueSplitSettlementPaid)
			split.POST("/settlements/:id/cancel", adminPaymentHandler.CancelRevenueSplitSettlement)

			// 历史订单补计提（幂等）
			split.POST("/orders/:id/accrue", adminPaymentHandler.AccrueRevenueSplitForOrder)
		}
	}
}
