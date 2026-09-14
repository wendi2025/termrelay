export interface NavItem {
  path: string
  name: string
  label: string
  feature: string
  short: string
}

// Primary user navigation: only pages that deserve a persistent sidebar entry.
export const userNavigation: NavItem[] = [
  { path: '/dashboard', name: 'Dashboard', label: '控制台', feature: 'dashboard', short: 'DB' },
  { path: '/keys', name: 'Keys', label: 'API 密钥', feature: 'keys', short: 'AK' },
  { path: '/usage', name: 'Usage', label: '用量统计', feature: 'usage', short: 'US' },
  { path: '/subscriptions', name: 'Subscriptions', label: '充值/订阅', feature: 'subscriptions', short: 'SB' },
  { path: '/orders', name: 'Orders', label: '我的订单', feature: 'orders', short: 'OR' },
  { path: '/affiliate', name: 'Affiliate', label: '邀请返利', feature: 'affiliate', short: 'AF' },
  { path: '/profile', name: 'Profile', label: '账户设置', feature: 'profile', short: 'AC' },
  { path: '/support', name: 'Support', label: '帮助与支持', feature: 'support', short: 'HP' },
]

// Secondary user routes stay addressable for existing links, top-bar shortcuts and future
// detail pages, but do not compete with core tasks in the primary sidebar.
export const userSecondaryRoutes: NavItem[] = [
  { path: '/usage-records', name: 'UsageRecords', label: '使用记录', feature: 'usage-records', short: 'UR' },
  { path: '/experience', name: 'Experience', label: '体验中心', feature: 'experience', short: 'EX' },
  { path: '/monitor', name: 'Monitor', label: '服务状态', feature: 'monitor', short: 'ST' },
  { path: '/wallet', name: 'Wallet', label: '我的钱包', feature: 'wallet', short: 'WL' },
  { path: '/purchase', name: 'Purchase', label: '购买服务', feature: 'purchase', short: 'BY' },
  { path: '/redeem', name: 'Redeem', label: '兑换码', feature: 'redeem', short: 'RD' },
  { path: '/announcements', name: 'Announcements', label: '系统公告', feature: 'announcements', short: 'AN' },
  { path: '/tickets', name: 'Tickets', label: '工单中心', feature: 'tickets', short: 'TK' },
  { path: '/available-channels', name: 'AvailableChannels', label: '可用渠道', feature: 'channels', short: 'CH' },
  { path: '/batch-image', name: 'BatchImage', label: '批量图片', feature: 'batch-image', short: 'BI' },
]

export const adminNavigation: NavItem[] = [
  { path: '/admin/dashboard', name: 'AdminDashboard', label: '控制台', feature: 'admin-dashboard', short: 'OV' },
  { path: '/admin/users', name: 'AdminUsers', label: '用户', feature: 'admin-users', short: 'UR' },
  { path: '/admin/accounts', name: 'AdminAccounts', label: '上游账户', feature: 'admin-accounts', short: 'UP' },
  { path: '/admin/groups', name: 'AdminGroups', label: '分组与模型', feature: 'admin-groups', short: 'GR' },
  { path: '/admin/channels/pricing', name: 'AdminChannels', label: '渠道与价格', feature: 'admin-channels', short: 'PR' },
  { path: '/admin/channels/monitor', name: 'AdminChannelMonitor', label: '渠道监控', feature: 'admin-monitor', short: 'CM' },
  { path: '/admin/usage', name: 'AdminUsage', label: '用量记录', feature: 'admin-usage', short: 'US' },
  { path: '/admin/ops', name: 'AdminOps', label: '运行监控', feature: 'admin-ops', short: 'OP' },
  { path: '/admin/audit-logs', name: 'AdminAudit', label: '审计日志', feature: 'admin-audit', short: 'AL' },
  { path: '/admin/prompt-audit', name: 'AdminPromptAudit', label: 'Prompt 审计', feature: 'admin-prompt-audit', short: 'PA' },
  { path: '/admin/subscriptions', name: 'AdminSubscriptions', label: '订阅管理', feature: 'admin-subscriptions', short: 'SB' },
  { path: '/admin/orders/dashboard', name: 'AdminPaymentDashboard', label: '支付概览', feature: 'admin-payment-dashboard', short: 'PD' },
  { path: '/admin/orders', name: 'AdminOrders', label: '订单管理', feature: 'admin-orders', short: 'OR' },
  { path: '/admin/orders/plans', name: 'AdminPaymentPlans', label: '订阅计划', feature: 'admin-plans', short: 'PL' },
  { path: '/admin/payment/providers', name: 'AdminPaymentProviders', label: '支付渠道', feature: 'admin-payment-providers', short: 'PV' },
  { path: '/admin/payment/config', name: 'AdminPaymentConfig', label: '支付配置', feature: 'admin-payment-config', short: 'PC' },
  { path: '/admin/payment/revenue-split', name: 'AdminRevenueSplit', label: '共建分成', feature: 'admin-payment-revenue-split', short: 'RS' },
  { path: '/admin/redeem', name: 'AdminRedeem', label: '兑换码', feature: 'admin-redeem', short: 'RD' },
  { path: '/admin/promo-codes', name: 'AdminPromo', label: '优惠码', feature: 'admin-promo', short: 'PC' },
  { path: '/admin/affiliates/invites', name: 'AdminAffiliateInvites', label: '邀请记录', feature: 'admin-affiliate-invites', short: 'AI' },
  { path: '/admin/affiliates/rebates', name: 'AdminAffiliateRebates', label: '返利记录', feature: 'admin-affiliate-rebates', short: 'AR' },
  { path: '/admin/affiliates/transfers', name: 'AdminAffiliateTransfers', label: '转账记录', feature: 'admin-affiliate-transfers', short: 'AT' },
  { path: '/admin/proxies', name: 'AdminProxies', label: '代理', feature: 'admin-proxies', short: 'PX' },
  { path: '/admin/announcements', name: 'AdminAnnouncements', label: '公告', feature: 'admin-announcements', short: 'AN' },
  { path: '/admin/risk-control', name: 'AdminRiskControl', label: '风控', feature: 'admin-risk', short: 'RC' },
  { path: '/admin/backup', name: 'AdminBackup', label: '备份', feature: 'admin-backup', short: 'BK' },
  { path: '/admin/settings', name: 'AdminSettings', label: '平台设置', feature: 'admin-settings', short: 'ST' },
]
