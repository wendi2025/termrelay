import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import HomePage from '../smirel/pages/HomePage.vue'
import AuthPage from '../smirel/pages/AuthPage.vue'
import OAuthCallbackPage from '../smirel/pages/OAuthCallbackPage.vue'
import WorkspacePage from '../smirel/pages/WorkspacePage.vue'
import ApiKeysPage from '../smirel/pages/ApiKeysPage.vue'
import UserUsagePage from '../smirel/pages/UserUsagePage.vue'
import UserBillingRoutePage from '../smirel/pages/UserBillingRoutePage.vue'
import UserOrdersPage from '../smirel/components/UserOrdersPage.vue'
import AdminOverviewPage from '../smirel/pages/AdminOverviewPage.vue'
import AdminAccountsPage from '../smirel/pages/AdminAccountsPage.vue'
import AdminGroupsPage from '../smirel/pages/AdminGroupsPage.vue'
import AdminChannelsPage from '../smirel/pages/AdminChannelsPage.vue'
import AdminPaymentDashboardPage from '../smirel/pages/AdminPaymentDashboardPage.vue'
import AdminPaymentProvidersPage from '../smirel/pages/AdminPaymentProvidersPage.vue'
import AdminPaymentPlansPage from '../smirel/pages/AdminPaymentPlansPage.vue'
import AdminPaymentConfigPage from '../smirel/pages/AdminPaymentConfigPage.vue'
import AdminRevenueSplitPage from '../smirel/pages/AdminRevenueSplitPage.vue'
import AdminOrdersPage from '../smirel/pages/AdminOrdersPage.vue'
import AdminCompliancePage from '../smirel/pages/AdminCompliancePage.vue'
import ModelCatalogPage from '../smirel/pages/ModelCatalogPage.vue'
import PublicPage from '../smirel/pages/PublicPage.vue'
import NotFoundPage from '../smirel/pages/NotFoundPage.vue'
import { isAuthenticated, isAdmin } from '../smirel/core/session'
import { adminNavigation, userNavigation, userSecondaryRoutes } from '../smirel/core/navigation'

const workspaceRoutes: RouteRecordRaw[] = [
  ...[...userNavigation, ...userSecondaryRoutes].map((item) => ({
    path: item.path,
    name: item.name,
    component: item.path === '/keys'
      ? ApiKeysPage
      : item.path === '/usage'
        ? UserUsagePage
        : item.path === '/subscriptions'
          ? UserBillingRoutePage
          : item.path === '/orders'
            ? UserOrdersPage
            : WorkspacePage,
    meta: { shell: 'workspace', requiresAuth: true, title: item.label, feature: item.feature },
  })),
  ...adminNavigation.map((item) => ({
    path: item.path,
    name: item.name,
    component: item.path === '/admin/dashboard'
      ? AdminOverviewPage
      : item.path === '/admin/accounts'
        ? AdminAccountsPage
        : item.path === '/admin/groups'
          ? AdminGroupsPage
          : item.path === '/admin/channels/pricing'
          ? AdminChannelsPage
          : item.path === '/admin/orders/dashboard'
            ? AdminPaymentDashboardPage
            : item.path === '/admin/orders'
              ? AdminOrdersPage
              : item.path === '/admin/orders/plans'
                ? AdminPaymentPlansPage
                : item.path === '/admin/payment/providers'
                  ? AdminPaymentProvidersPage
                  : item.path === '/admin/payment/config'
                    ? AdminPaymentConfigPage
                    : item.path === '/admin/payment/revenue-split'
                      ? AdminRevenueSplitPage
                      : WorkspacePage,
    meta: { shell: 'workspace', requiresAuth: true, requiresAdmin: true, title: item.label, feature: item.feature },
  })),
]

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/home' },
  { path: '/home', name: 'Home', component: HomePage, meta: { title: '首页' } },
  { path: '/login', name: 'Login', component: AuthPage, meta: { title: '登录', authKind: 'login' } },
  { path: '/register', name: 'Register', component: AuthPage, meta: { title: '注册', authKind: 'register' } },
  { path: '/forgot-password', name: 'ForgotPassword', component: AuthPage, meta: { title: '找回密码', authKind: 'forgot' } },
  { path: '/reset-password', name: 'ResetPassword', component: AuthPage, meta: { title: '重置密码', authKind: 'reset' } },
  { path: '/email-verify', name: 'EmailVerify', component: PublicPage, meta: { title: '邮箱验证', publicKind: 'callback' } },
  { path: '/model-plaza', name: 'ModelPlaza', component: ModelCatalogPage, meta: { shell: 'workspace', requiresAuth: true, title: '模型广场', feature: 'model-catalog' } },
  { path: '/key-usage', name: 'KeyUsage', component: PublicPage, meta: { title: '用量查询', publicKind: 'key-usage' } },
  { path: '/legal/:documentId', name: 'LegalDocument', component: PublicPage, meta: { title: '法律文档', publicKind: 'legal' } },
  { path: '/setup', name: 'Setup', component: PublicPage, meta: { title: '初始化', publicKind: 'setup' } },
  { path: '/payment/result', name: 'PaymentResult', component: PublicPage, meta: { title: '支付结果', publicKind: 'payment' } },
  { path: '/payment/redirect', name: 'PaymentRedirect', component: PublicPage, meta: { title: '支付跳转', publicKind: 'payment' } },
  { path: '/payment/wechat-oauth', name: 'PaymentWechatOAuth', component: PublicPage, meta: { title: '微信授权', publicKind: 'payment' } },
  { path: '/payment/qrcode', name: 'PaymentQRCode', component: PublicPage, meta: { title: '支付', publicKind: 'payment', requiresAuth: true } },
  { path: '/payment/stripe', name: 'StripePayment', component: PublicPage, meta: { title: '支付', publicKind: 'payment' } },
  { path: '/payment/stripe-popup', name: 'StripePayment', component: PublicPage, meta: { title: '支付', publicKind: 'payment' } },
  { path: '/payment/airwallex', name: 'AirwallexPayment', component: PublicPage, meta: { title: '支付', publicKind: 'payment' } },
  { path: '/admin/compliance', name: 'AdminCompliance', component: AdminCompliancePage, meta: { requiresAuth: true, requiresAdmin: true, title: '管理员合规确认' } },
  { path: '/auth/wechat/payment/callback', name: 'WeChatPaymentCallback', component: PublicPage, meta: { title: '支付回调', publicKind: 'callback' } },
  { path: '/auth/oauth/callback', name: 'SmirelOAuthCallback', component: OAuthCallbackPage, meta: { title: '登录回调' } },
  { path: '/auth/:provider/callback', name: 'OAuthProviderCallback', component: PublicPage, meta: { title: '登录回调', publicKind: 'callback' } },
  { path: '/auth/callback', redirect: '/auth/oauth/callback' },
  { path: '/auth/dingtalk/email-completion', name: 'DingTalkEmailCompletion', component: PublicPage, meta: { title: '完成登录', publicKind: 'callback' } },
  { path: '/docs/batch-image', redirect: '/batch-image' },
  { path: '/custom/:id', name: 'CustomPage', component: WorkspacePage, meta: { shell: 'workspace', requiresAuth: true, title: '自定义页面', feature: 'custom' } },
  ...workspaceRoutes,
  { path: '/admin', redirect: '/admin/dashboard' },
  { path: '/admin/channels', redirect: '/admin/channels/pricing' },
  { path: '/admin/affiliates', redirect: '/admin/affiliates/invites' },
  { path: '/:pathMatch(.*)*', name: 'NotFound', component: NotFoundPage, meta: { title: '页面不存在' } },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach((to) => {
  document.title = `${String(to.meta.title || 'Muxway')} · Muxway 模枢`

  if (to.meta.requiresAuth && !isAuthenticated.value) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAdmin && !isAdmin.value) {
    return '/dashboard'
  }
  if ((to.path === '/login' || to.path === '/register') && isAuthenticated.value) {
    return isAdmin.value ? '/admin/dashboard' : '/dashboard'
  }
  return true
})

export default router
