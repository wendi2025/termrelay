<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import WorkspaceNavIcon from './WorkspaceNavIcon.vue'
import { adminNavigation, userNavigation, type NavItem } from '../core/navigation'
import { type SmirelLocale } from '../core/i18n'
import {
  clearNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  notifications,
  unreadNotificationCount,
} from '../core/notifications'
import {
  interfacePreferences,
  setLocale,
  setTheme,
  type ThemePreference,
} from '../core/preferences'
import { useSession } from '../core/session'
import '../styles/workspace-layout.css'

interface NavGroup {
  label: string
  items: NavItem[]
}

interface TopbarLink {
  label: string
  path: string
  icon: string
}

type UtilityPanel = 'notifications' | 'language' | 'theme'
type WorkspaceMode = 'user' | 'admin'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const mobileOpen = ref(false)
const openUtility = ref<UtilityPanel | null>(null)
const { state, isAdmin } = useSession()
const logoUrl = `${import.meta.env.BASE_URL}smirel-logo.png`
const sharedKeysNavigation = userNavigation.find((item) => item.feature === 'keys')
const MODE_STORAGE_KEY = 'smirel.workspace.mode'
const LAST_ADMIN_ROUTE_KEY = 'smirel.workspace.last-admin-route'
const LAST_USER_ROUTE_KEY = 'smirel.workspace.last-user-route'
const neutralWorkspacePaths = ['/keys', '/model-plaza', '/profile']

function readSessionValue(key: string, fallback: string) {
  if (typeof window === 'undefined') return fallback
  return window.sessionStorage.getItem(key) || fallback
}

function writeSessionValue(key: string, value: string) {
  if (typeof window === 'undefined') return
  window.sessionStorage.setItem(key, value)
}

function inferWorkspaceMode(path: string): WorkspaceMode {
  if (!isAdmin.value) return 'user'
  if (path.startsWith('/admin')) return 'admin'
  if (neutralWorkspacePaths.some((neutralPath) => path === neutralPath || path.startsWith(`${neutralPath}/`))) {
    return readSessionValue(MODE_STORAGE_KEY, 'admin') === 'user' ? 'user' : 'admin'
  }
  return 'user'
}

const workspaceMode = ref<WorkspaceMode>(inferWorkspaceMode(route.path))
const isAdminWorkspace = computed(() => isAdmin.value && workspaceMode.value === 'admin')
const workspaceModeLabels = computed(() => interfacePreferences.locale === 'zh-CN'
  ? { user: '用户端', admin: '管理端', switcher: '切换工作区' }
  : { user: 'User', admin: 'Admin', switcher: 'Switch workspace' })

const modelCatalogNavItem = computed<NavItem>(() => ({
  path: '/model-plaza',
  name: 'ModelPlaza',
  label: interfacePreferences.locale === 'zh-CN' ? '模型广场' : 'Model Marketplace',
  feature: 'model-catalog',
  short: 'MD',
}))
const navigation = computed<NavItem[]>(() => {
  if (!isAdminWorkspace.value) return [...userNavigation, modelCatalogNavItem.value]
  if (!sharedKeysNavigation) return [...adminNavigation, modelCatalogNavItem.value]
  return [adminNavigation[0], sharedKeysNavigation, ...adminNavigation.slice(1), modelCatalogNavItem.value]
})
const initials = computed(() => (state.user?.username || state.user?.email || 'S').slice(0, 1).toUpperCase())
const accountName = computed(() => state.user?.username || state.user?.email?.split('@')[0] || 'Smirel')
const accountRole = computed(() => isAdmin.value ? t('shell.roleAdmin') : t('shell.roleUser'))
const currentLocaleShort = computed(() => interfacePreferences.locale === 'zh-CN' ? '中' : 'EN')
const unreadBadge = computed(() => unreadNotificationCount.value > 99 ? '99+' : String(unreadNotificationCount.value))

const navIconByFeature: Record<string, string> = {
  dashboard: 'dashboard',
  keys: 'key',
  usage: 'chart',
  'usage-records': 'receipt',
  experience: 'activity',
  monitor: 'activity',
  wallet: 'wallet',
  subscriptions: 'credit-card',
  purchase: 'wallet',
  orders: 'receipt',
  redeem: 'receipt',
  affiliate: 'gift',
  profile: 'user-cog',
  announcements: 'bell',
  support: 'lifebuoy',
  tickets: 'message-circle',
  'model-catalog': 'layers',
  'admin-dashboard': 'dashboard',
  'admin-users': 'users',
  'admin-accounts': 'server',
  'admin-groups': 'layers',
  'admin-channels': 'network',
  'admin-usage': 'chart',
  'admin-ops': 'activity',
  'admin-payment-dashboard': 'credit-card',
  'admin-orders': 'receipt',
  'admin-settings': 'settings',
}

const navLabelKeyByFeature: Record<string, string> = {
  dashboard: 'nav.dashboard',
  keys: 'nav.keys',
  usage: 'nav.usage',
  'usage-records': 'nav.usageRecords',
  experience: 'nav.experience',
  monitor: 'nav.monitor',
  wallet: 'nav.wallet',
  subscriptions: 'nav.subscriptions',
  purchase: 'nav.purchase',
  orders: 'nav.orders',
  redeem: 'nav.redeem',
  affiliate: 'nav.affiliate',
  profile: 'nav.profile',
  announcements: 'nav.announcements',
  support: 'nav.support',
  tickets: 'nav.tickets',
  'admin-dashboard': 'nav.adminDashboard',
  'admin-users': 'nav.adminUsers',
  'admin-accounts': 'nav.adminAccounts',
  'admin-groups': 'nav.adminGroups',
  'admin-channels': 'nav.adminChannels',
  'admin-usage': 'nav.adminUsage',
  'admin-ops': 'nav.adminOps',
  'admin-payment-dashboard': 'nav.adminPayment',
  'admin-orders': 'nav.adminOrders',
  'admin-settings': 'nav.adminSettings',
}

function navLabel(item: NavItem) {
  const key = navLabelKeyByFeature[item.feature]
  return key ? t(key) : item.label
}

function take(items: NavItem[], features: string[]) {
  return features.flatMap((feature) => items.filter((item) => item.feature === feature))
}

const navigationGroups = computed<NavGroup[]>(() => {
  const items = navigation.value

  if (!isAdminWorkspace.value) {
    return [
      { label: t('groups.console'), items: take(items, ['dashboard', 'keys', 'usage', 'usage-records']) },
      { label: t('groups.resources'), items: take(items, ['model-catalog', 'experience', 'monitor']) },
      { label: t('groups.billing'), items: take(items, ['wallet', 'subscriptions', 'purchase', 'orders', 'redeem']) },
      { label: t('groups.account'), items: take(items, ['affiliate', 'profile']) },
      { label: t('groups.support'), items: take(items, ['announcements', 'support', 'tickets']) },
    ].filter((group) => group.items.length)
  }

  return [
    { label: t('groups.console'), items: take(items, ['admin-dashboard', 'keys', 'admin-users']) },
    { label: t('groups.resources'), items: take(items, ['admin-accounts', 'admin-groups', 'admin-channels', 'model-catalog']) },
    { label: t('groups.operations'), items: take(items, ['admin-usage', 'admin-ops']) },
    { label: t('groups.transactions'), items: take(items, ['admin-payment-dashboard', 'admin-orders']) },
    { label: t('groups.system'), items: take(items, ['admin-settings']) },
  ].filter((group) => group.items.length)
})

const activeItem = computed(() => navigation.value
  .filter((item) => route.path === item.path || route.path.startsWith(`${item.path}/`))
  .sort((a, b) => b.path.length - a.path.length)[0] || null)

const activeItemLabel = computed(() => activeItem.value
  ? navLabel(activeItem.value)
  : (isAdminWorkspace.value ? t('nav.adminDashboard') : t('nav.dashboard')))

const activeGroupLabel = computed(() => navigationGroups.value.find((group) =>
  group.items.some((item) => item.path === activeItem.value?.path),
)?.label || '')

const breadcrumbGroupLabel = computed(() => (
  activeGroupLabel.value && activeGroupLabel.value !== activeItemLabel.value
    ? activeGroupLabel.value
    : ''
))

const topbarLinks = computed<TopbarLink[]>(() => isAdminWorkspace.value
  ? [
      { label: t('quick.model'), path: '/admin/groups', icon: 'layers' },
      { label: t('quick.upstream'), path: '/admin/accounts', icon: 'server' },
      { label: t('quick.monitor'), path: '/admin/ops', icon: 'activity' },
    ]
  : [
      { label: t('quick.model'), path: '/model-plaza', icon: 'layers' },
      { label: t('quick.keys'), path: '/keys', icon: 'key' },
      { label: t('quick.status'), path: '/monitor', icon: 'activity' },
    ])

const languageOptions = computed(() => [
  { value: 'zh-CN' as SmirelLocale, label: t('utility.chinese') },
  { value: 'en-US' as SmirelLocale, label: t('utility.english') },
])

const themeOptions = computed(() => [
  { value: 'dark' as ThemePreference, label: t('utility.dark') },
  { value: 'light' as ThemePreference, label: t('utility.light') },
  { value: 'system' as ThemePreference, label: t('utility.system') },
])

function setWorkspaceMode(mode: WorkspaceMode) {
  if (!isAdmin.value || workspaceMode.value === mode) return

  const currentMode = workspaceMode.value
  const currentPath = route.fullPath
  writeSessionValue(currentMode === 'admin' ? LAST_ADMIN_ROUTE_KEY : LAST_USER_ROUTE_KEY, currentPath)
  workspaceMode.value = mode
  writeSessionValue(MODE_STORAGE_KEY, mode)
  openUtility.value = null
  mobileOpen.value = false

  const destination = mode === 'admin'
    ? readSessionValue(LAST_ADMIN_ROUTE_KEY, '/admin/dashboard')
    : readSessionValue(LAST_USER_ROUTE_KEY, '/dashboard')

  void router.push(destination)
}

function toggleUtility(panel: UtilityPanel) {
  openUtility.value = openUtility.value === panel ? null : panel
}

function chooseLocale(locale: SmirelLocale) {
  setLocale(locale)
  openUtility.value = null
}

function chooseTheme(theme: ThemePreference) {
  setTheme(theme)
  openUtility.value = null
}

function formatNotificationTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('utility.justNow')
  return date.toLocaleString(interfacePreferences.locale, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

watch(() => route.fullPath, (fullPath) => {
  if (!isAdmin.value) {
    workspaceMode.value = 'user'
    return
  }

  const path = route.path
  if (path.startsWith('/admin')) {
    workspaceMode.value = 'admin'
  } else if (!neutralWorkspacePaths.some((neutralPath) => path === neutralPath || path.startsWith(`${neutralPath}/`))) {
    workspaceMode.value = 'user'
  }

  writeSessionValue(MODE_STORAGE_KEY, workspaceMode.value)
  writeSessionValue(workspaceMode.value === 'admin' ? LAST_ADMIN_ROUTE_KEY : LAST_USER_ROUTE_KEY, fullPath)
}, { immediate: true })
</script>

<template>
  <div class="workspace-root">
    <div class="site-environment" aria-hidden="true"></div>
    <aside class="workspace-sidebar glass" :class="{ open: mobileOpen }">
      <div class="workspace-brand-row">
        <RouterLink to="/home" class="brand-link" @click="mobileOpen = false">
          <img :src="logoUrl" alt="Smirel" />
          <span class="workspace-brand-copy"><strong>Smirel</strong><small>API SERVICE</small></span>
        </RouterLink>
        <button class="mobile-close" type="button" :aria-label="t('shell.closeNav')" @click="mobileOpen = false">×</button>
      </div>

      <nav class="workspace-nav">
        <section v-for="group in navigationGroups" :key="group.label" class="workspace-nav-group">
          <div class="workspace-nav-label">{{ group.label }}</div>
          <RouterLink
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            :class="{ active: route.path === item.path || route.path.startsWith(`${item.path}/`) }"
            :aria-current="route.path === item.path || route.path.startsWith(`${item.path}/`) ? 'page' : undefined"
            @click="mobileOpen = false"
          >
            <WorkspaceNavIcon :name="navIconByFeature[item.feature] || 'circle'" />
            <span>{{ navLabel(item) }}</span>
          </RouterLink>
        </section>
      </nav>
    </aside>

    <button v-if="mobileOpen" class="workspace-scrim" type="button" :aria-label="t('shell.closeNav')" @click="mobileOpen = false"></button>
    <button v-if="openUtility" class="workspace-utility-dismiss" type="button" :aria-label="t('shell.closeNav')" @click="openUtility = null"></button>

    <section class="workspace-main">
      <header class="workspace-topbar workspace-topbar--contextual">
        <div class="workspace-topbar-left">
          <button class="mobile-menu" type="button" :aria-label="t('shell.openNav')" @click="mobileOpen = true"><span></span><span></span><span></span></button>
          <nav :key="route.path" class="workspace-breadcrumb" :aria-label="t('shell.currentLocation')">
            <span class="workspace-breadcrumb-home" aria-hidden="true">
              <WorkspaceNavIcon name="dashboard" />
            </span>
            <span class="workspace-breadcrumb-root">{{ isAdminWorkspace ? t('shell.adminConsole') : t('shell.userConsole') }}</span>
            <span class="workspace-breadcrumb-separator" aria-hidden="true">/</span>
            <template v-if="breadcrumbGroupLabel">
              <span class="workspace-breadcrumb-group">{{ breadcrumbGroupLabel }}</span>
              <span class="workspace-breadcrumb-separator" aria-hidden="true">/</span>
            </template>
            <strong class="workspace-topbar-title">{{ activeItemLabel }}</strong>
          </nav>
        </div>

        <div class="workspace-topbar-actions">
          <RouterLink
            v-for="item in topbarLinks"
            :key="item.path"
            :to="item.path"
            class="workspace-topbar-link"
            :class="{ active: route.path === item.path || route.path.startsWith(`${item.path}/`) }"
            :aria-label="item.label"
            :title="item.label"
          >
            <WorkspaceNavIcon :name="item.icon" />
            <span>{{ item.label }}</span>
          </RouterLink>

          <div v-if="isAdmin" class="workspace-mode-switch" role="group" :aria-label="workspaceModeLabels.switcher">
            <span class="workspace-mode-indicator" :class="{ 'is-admin': isAdminWorkspace }" aria-hidden="true"></span>
            <button
              type="button"
              :class="{ active: !isAdminWorkspace }"
              :aria-pressed="!isAdminWorkspace"
              :title="workspaceModeLabels.user"
              @click="setWorkspaceMode('user')"
            >
              <span class="workspace-mode-long">{{ workspaceModeLabels.user }}</span>
              <span class="workspace-mode-short">{{ interfacePreferences.locale === 'zh-CN' ? '用户' : 'User' }}</span>
            </button>
            <button
              type="button"
              :class="{ active: isAdminWorkspace }"
              :aria-pressed="isAdminWorkspace"
              :title="workspaceModeLabels.admin"
              @click="setWorkspaceMode('admin')"
            >
              <span class="workspace-mode-long">{{ workspaceModeLabels.admin }}</span>
              <span class="workspace-mode-short">{{ interfacePreferences.locale === 'zh-CN' ? '管理' : 'Admin' }}</span>
            </button>
          </div>

          <span class="workspace-topbar-divider" aria-hidden="true"></span>

          <div class="workspace-utility-control">
            <button
              class="workspace-utility-button"
              :class="{ active: openUtility === 'notifications' }"
              type="button"
              :aria-label="t('utility.notifications')"
              :title="t('utility.notifications')"
              @click="toggleUtility('notifications')"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" />
                <path d="M10 21h4" />
              </svg>
              <b v-if="unreadNotificationCount" class="workspace-notification-badge">{{ unreadBadge }}</b>
            </button>

            <section v-if="openUtility === 'notifications'" class="workspace-utility-popover workspace-notifications-panel">
              <header class="workspace-utility-head">
                <div>
                  <strong>{{ t('utility.notificationCenter') }}</strong>
                  <small v-if="unreadNotificationCount">{{ unreadNotificationCount }} {{ t('utility.notifications') }}</small>
                </div>
                <div class="workspace-notification-actions">
                  <button type="button" :disabled="!unreadNotificationCount" @click="markAllNotificationsRead">{{ t('utility.markAllRead') }}</button>
                  <button type="button" :disabled="!notifications.length" @click="clearNotifications">{{ t('utility.clearAll') }}</button>
                </div>
              </header>

              <div v-if="notifications.length" class="workspace-notification-list">
                <button
                  v-for="item in notifications"
                  :key="item.id"
                  class="workspace-notification-item"
                  :class="[{ unread: !item.read }, `tone-${item.tone}`]"
                  type="button"
                  @click="markNotificationRead(item.id)"
                >
                  <i></i>
                  <span>
                    <strong>{{ item.title }}</strong>
                    <small v-if="item.message">{{ item.message }}</small>
                    <time>{{ formatNotificationTime(item.createdAt) }}</time>
                  </span>
                </button>
              </div>
              <div v-else class="workspace-notification-empty">
                <span>
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" />
                    <path d="M10 21h4" />
                  </svg>
                </span>
                <strong>{{ t('utility.noNotifications') }}</strong>
                <small>{{ t('utility.noNotificationsHint') }}</small>
              </div>
            </section>
          </div>

          <div class="workspace-utility-control">
            <button
              class="workspace-utility-button workspace-language-button"
              :class="{ active: openUtility === 'language' }"
              type="button"
              :aria-label="t('utility.language')"
              :title="t('utility.language')"
              @click="toggleUtility('language')"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="12" cy="12" r="9" />
                <path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18" />
              </svg>
              <span>{{ currentLocaleShort }}</span>
            </button>

            <section v-if="openUtility === 'language'" class="workspace-utility-popover workspace-choice-panel">
              <header class="workspace-choice-head">
                <strong>{{ t('utility.chooseLanguage') }}</strong>
                <small>{{ t('utility.language') }}</small>
              </header>
              <button
                v-for="option in languageOptions"
                :key="option.value"
                class="workspace-choice-row"
                :class="{ selected: interfacePreferences.locale === option.value }"
                type="button"
                @click="chooseLocale(option.value)"
              >
                <span>{{ option.label }}</span>
                <svg v-if="interfacePreferences.locale === option.value" viewBox="0 0 16 16" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg>
              </button>
            </section>
          </div>

          <div class="workspace-utility-control">
            <button
              class="workspace-utility-button"
              :class="{ active: openUtility === 'theme' }"
              type="button"
              :aria-label="t('utility.theme')"
              :title="t('utility.theme')"
              @click="toggleUtility('theme')"
            >
              <svg v-if="interfacePreferences.resolvedTheme === 'dark'" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M20.2 15.3A8.5 8.5 0 0 1 8.7 3.8 8.5 8.5 0 1 0 20.2 15.3Z" />
              </svg>
              <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="12" cy="12" r="4" />
                <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
              </svg>
            </button>

            <section v-if="openUtility === 'theme'" class="workspace-utility-popover workspace-choice-panel">
              <header class="workspace-choice-head">
                <strong>{{ t('utility.chooseTheme') }}</strong>
                <small>{{ t('utility.theme') }}</small>
              </header>
              <button
                v-for="option in themeOptions"
                :key="option.value"
                class="workspace-choice-row"
                :class="{ selected: interfacePreferences.theme === option.value }"
                type="button"
                @click="chooseTheme(option.value)"
              >
                <span>{{ option.label }}</span>
                <svg v-if="interfacePreferences.theme === option.value" viewBox="0 0 16 16" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg>
              </button>
            </section>
          </div>

          <span class="workspace-topbar-divider" aria-hidden="true"></span>

          <RouterLink to="/profile" class="workspace-profile-link" :aria-label="t('shell.accountSettings')">
            <span class="mini-avatar">{{ initials }}</span>
            <span class="workspace-profile-copy">
              <strong>{{ accountName }}</strong>
              <small>{{ accountRole }}</small>
            </span>
            <svg class="workspace-profile-chevron" viewBox="0 0 16 16" aria-hidden="true">
              <path d="M6 3.5 10.5 8 6 12.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </RouterLink>
        </div>
      </header>
      <main class="workspace-canvas"><slot /></main>
    </section>
  </div>
</template>

<style scoped>
.workspace-brand-row {
  height: 66px;
  padding: 0 3px 14px;
}

.workspace-brand-row .brand-link {
  flex: 1 1 auto;
  min-width: 0;
  height: 50px;
  padding: 0 9px;
  gap: 12px;
  border: 1px solid transparent;
  border-radius: 12px;
  transition:
    background-color .18s ease,
    border-color .18s ease,
    box-shadow .18s ease;
}

.workspace-brand-row .brand-link:hover {
  border-color: var(--ws-border);
  background: var(--ws-surface-soft);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .018);
}

.workspace-brand-row .brand-link:focus-visible {
  outline: none;
  border-color: #3b759f;
  box-shadow: 0 0 0 3px rgba(47, 150, 232, .10);
}

.workspace-brand-row .brand-link img {
  width: 82px;
  height: auto;
  max-height: 34px;
  flex: 0 0 82px;
  object-fit: contain;
  filter: drop-shadow(0 4px 10px rgba(47, 150, 232, .10));
}

.workspace-brand-copy {
  position: relative;
  min-width: 0;
  padding-left: 13px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}

.workspace-brand-copy::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  width: 1px;
  height: 30px;
  border-radius: 999px;
  background: var(--ws-border);
  transform: translateY(-50%);
}

.workspace-brand-copy strong {
  color: var(--ws-text);
  font-size: .98rem;
  line-height: 1;
  font-weight: 720;
  letter-spacing: -.025em;
}

.workspace-brand-copy small {
  color: var(--ws-subtle);
  font-size: .55rem;
  line-height: 1;
  font-weight: 720;
  letter-spacing: .18em;
}

.workspace-nav a {
  position: relative;
  isolation: isolate;
  gap: 11px;
  overflow: hidden;
  transition:
    color .18s ease,
    background-color .18s ease,
    border-color .18s ease,
    transform .18s cubic-bezier(.2, .75, .25, 1),
    box-shadow .18s ease;
}

.workspace-nav a::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  width: 2px;
  height: 20px;
  border-radius: 999px;
  background: #67baf1;
  opacity: 0;
  transform: translateY(-50%) scaleY(.35);
  transform-origin: center;
  transition: opacity .16s ease, transform .2s cubic-bezier(.2, .75, .25, 1);
}

.workspace-nav a::after {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  border-radius: inherit;
  background: linear-gradient(90deg, rgba(52, 151, 221, .075), rgba(52, 151, 221, .018) 64%, transparent);
  opacity: 0;
  pointer-events: none;
  transition: opacity .18s ease;
}

.workspace-nav a:hover {
  border-color: #252d36;
  background: #11151b;
  color: #eef2f6;
  transform: translateX(2px);
}

.workspace-nav a:hover::after {
  opacity: 1;
}

.workspace-nav a:active {
  transform: translateX(1px) scale(.995);
  transition-duration: .06s;
}

.workspace-nav a:focus-visible {
  outline: none;
  border-color: #365873;
  box-shadow: 0 0 0 3px rgba(57, 145, 207, .10);
  color: #f5f8fb;
}

.workspace-nav a.active {
  border-color: #2b3c4c;
  background: #121d28;
  color: #f7fafc;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .018);
}

.workspace-nav a.active::before {
  opacity: 1;
  transform: translateY(-50%) scaleY(1);
}

.workspace-nav a.active::after {
  opacity: 1;
  background: linear-gradient(90deg, rgba(47, 150, 232, .12), rgba(47, 150, 232, .025) 72%, transparent);
}

.workspace-nav a.active:hover {
  border-color: #33495c;
  background: #142230;
  transform: translateX(1px);
}

.workspace-nav-icon {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  color: #68717d;
  transition: color .18s ease, transform .18s cubic-bezier(.2, .75, .25, 1);
}

.workspace-nav a:hover .workspace-nav-icon {
  color: #aeb8c3;
  transform: translateX(1px);
}

.workspace-nav a.active .workspace-nav-icon {
  color: #78c2f3;
  transform: translateX(1px);
}

.workspace-nav a > span {
  min-width: 0;
  transition: color .18s ease, transform .18s cubic-bezier(.2, .75, .25, 1);
}

.workspace-nav a:hover > span {
  transform: translateX(1px);
}

.workspace-nav a.active > span {
  color: #f3f7fa;
}

.workspace-topbar.workspace-topbar--contextual {
  position: sticky;
  top: 0;
  z-index: 35;
  height: 64px;
  padding: 0 28px 0 34px;
  border-bottom: 1px solid #20232a;
  background: rgba(10, 11, 15, .988);
  align-items: center;
}

.workspace-topbar--contextual .workspace-topbar-left {
  min-width: 0;
  gap: 14px;
}

.workspace-breadcrumb {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.workspace-breadcrumb-root,
.workspace-breadcrumb-group {
  color: #737b86;
  font-size: .73rem;
  font-weight: 560;
}

.workspace-breadcrumb-group {
  color: #9299a3;
}

.workspace-breadcrumb-separator {
  color: #41464f;
  font-size: .70rem;
  user-select: none;
}

.workspace-topbar--contextual .workspace-topbar-title {
  min-width: 0;
  overflow: hidden;
  color: #eceff3;
  font-size: .82rem;
  font-weight: 640;
  text-overflow: ellipsis;
}

.workspace-topbar-actions {
  min-width: 0;
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.workspace-topbar-link {
  min-height: 36px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: 9px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #858d98;
  font-size: .72rem;
  font-weight: 570;
  white-space: nowrap;
  transition: color .15s ease, background-color .15s ease, border-color .15s ease;
}

.workspace-topbar-link .workspace-nav-icon {
  width: 17px;
  height: 17px;
  flex-basis: 17px;
  color: #68717d;
}

.workspace-topbar-link:hover {
  border-color: #292e36;
  background: #121419;
  color: #e2e6eb;
}

.workspace-topbar-link:hover .workspace-nav-icon {
  color: #b6bec8;
}

.workspace-topbar-link.active {
  border-color: #273644;
  background: #111c27;
  color: #dcecf8;
}

.workspace-topbar-link.active .workspace-nav-icon {
  color: #73bdf2;
}

.workspace-mode-switch {
  position: relative;
  width: 122px;
  height: 36px;
  padding: 3px;
  border: 1px solid #252a31;
  border-radius: 9px;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  align-items: stretch;
  background: #0d0f13;
  box-shadow: inset 0 1px rgba(255, 255, 255, .018);
}

.workspace-mode-indicator {
  position: absolute;
  left: 3px;
  top: 3px;
  width: calc(50% - 3px);
  height: 28px;
  border: 1px solid #30353d;
  border-radius: 7px;
  background: #181b20;
  box-shadow: 0 1px 2px rgba(0, 0, 0, .24), inset 0 1px rgba(255, 255, 255, .02);
  transform: translateX(0);
  transition: transform .2s cubic-bezier(.22, .72, .25, 1), border-color .18s ease, background-color .18s ease;
}

.workspace-mode-indicator.is-admin {
  border-color: #294358;
  background: #121d27;
  transform: translateX(100%);
}

.workspace-mode-switch button {
  position: relative;
  z-index: 1;
  min-width: 0;
  border: 0;
  border-radius: 7px;
  color: #69727d;
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: .67rem;
  font-weight: 610;
  white-space: nowrap;
  transition: color .16s ease;
}

.workspace-mode-switch button:hover {
  color: #b7bec7;
}

.workspace-mode-switch button.active {
  color: #e9edf1;
}

.workspace-mode-switch button:focus-visible {
  outline: 1px solid #4d8fbd;
  outline-offset: -2px;
}

.workspace-mode-short {
  display: none;
}

.workspace-topbar-divider {
  width: 1px;
  height: 28px;
  margin: 0 3px;
  background: #252930;
}

.workspace-utility-dismiss {
  position: fixed;
  inset: 0;
  z-index: 36;
  border: 0;
  background: transparent;
  cursor: default;
}

.workspace-utility-control {
  position: relative;
  z-index: 37;
}

.workspace-utility-button {
  position: relative;
  min-width: 36px;
  height: 36px;
  padding: 0 9px;
  border: 1px solid var(--ws-border);
  border-radius: 9px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--ws-muted);
  background: var(--ws-surface);
  cursor: pointer;
  transition: border-color .15s ease, background-color .15s ease, color .15s ease;
}

.workspace-utility-button:hover,
.workspace-utility-button.active {
  border-color: var(--ws-border-strong);
  color: var(--ws-text-soft);
  background: var(--ws-surface-soft);
}

.workspace-utility-button svg {
  width: 17px;
  height: 17px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.workspace-language-button span {
  color: inherit;
  font-size: .66rem;
  font-weight: 700;
}

.workspace-notification-badge {
  position: absolute;
  top: -5px;
  right: -5px;
  min-width: 17px;
  height: 17px;
  padding: 0 4px;
  border: 2px solid var(--ws-bg);
  border-radius: 999px;
  display: grid;
  place-items: center;
  color: #fff;
  background: #e05252;
  font-size: .52rem;
  line-height: 1;
  font-weight: 750;
}

.workspace-utility-popover {
  position: absolute;
  top: 44px;
  right: 0;
  z-index: 45;
  border: 1px solid var(--ws-border);
  border-radius: 12px;
  color: var(--ws-text);
  background: var(--ws-surface);
  box-shadow: 0 18px 46px rgba(0, 0, 0, .28);
  overflow: hidden;
}

.workspace-notifications-panel {
  width: min(360px, calc(100vw - 28px));
}

.workspace-utility-head {
  min-height: 64px;
  padding: 13px 14px;
  border-bottom: 1px solid var(--ws-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.workspace-utility-head > div:first-child {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.workspace-utility-head strong,
.workspace-choice-head strong {
  color: var(--ws-text);
  font-size: .79rem;
  font-weight: 660;
}

.workspace-utility-head small,
.workspace-choice-head small {
  color: var(--ws-subtle);
  font-size: .63rem;
}

.workspace-notification-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.workspace-notification-actions button {
  min-height: 28px;
  padding: 0 7px;
  border: 0;
  border-radius: 6px;
  color: var(--ws-muted);
  background: transparent;
  font-size: .62rem;
  cursor: pointer;
}

.workspace-notification-actions button:hover:not(:disabled) {
  color: var(--ws-text-soft);
  background: var(--ws-surface-soft);
}

.workspace-notification-actions button:disabled {
  opacity: .42;
  cursor: default;
}

.workspace-notification-list {
  max-height: min(420px, calc(100vh - 120px));
  overflow-y: auto;
}

.workspace-notification-item {
  width: 100%;
  min-height: 74px;
  padding: 12px 14px;
  border: 0;
  border-bottom: 1px solid var(--ws-border);
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr);
  gap: 10px;
  text-align: left;
  color: inherit;
  background: transparent;
  cursor: pointer;
}

.workspace-notification-item:last-child {
  border-bottom: 0;
}

.workspace-notification-item:hover,
.workspace-notification-item.unread {
  background: var(--ws-surface-soft);
}

.workspace-notification-item > i {
  width: 6px;
  height: 6px;
  margin-top: 5px;
  border-radius: 50%;
  background: #66717d;
}

.workspace-notification-item.tone-success > i { background: #42ce99; }
.workspace-notification-item.tone-warning > i { background: #d9a94e; }
.workspace-notification-item.tone-error > i { background: #df6565; }
.workspace-notification-item.tone-info > i { background: #5fa9e5; }

.workspace-notification-item > span {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.workspace-notification-item strong {
  color: var(--ws-text-soft);
  font-size: .72rem;
  font-weight: 630;
}

.workspace-notification-item small {
  margin-top: 4px;
  color: var(--ws-muted);
  font-size: .65rem;
  line-height: 1.45;
}

.workspace-notification-item time {
  margin-top: 6px;
  color: var(--ws-subtle);
  font-size: .58rem;
}

.workspace-notification-empty {
  min-height: 218px;
  padding: 26px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.workspace-notification-empty > span {
  width: 40px;
  height: 40px;
  border: 1px solid var(--ws-border);
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: var(--ws-subtle);
  background: var(--ws-surface-soft);
}

.workspace-notification-empty svg {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.workspace-notification-empty strong {
  margin-top: 12px;
  color: var(--ws-text-soft);
  font-size: .76rem;
}

.workspace-notification-empty small {
  max-width: 230px;
  margin-top: 5px;
  color: var(--ws-subtle);
  font-size: .64rem;
  line-height: 1.5;
}

.workspace-choice-panel {
  width: 226px;
  padding: 7px;
}

.workspace-choice-head {
  padding: 9px 9px 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.workspace-choice-row {
  width: 100%;
  min-height: 38px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 7px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--ws-muted);
  background: transparent;
  font-size: .72rem;
  text-align: left;
  cursor: pointer;
}

.workspace-choice-row:hover {
  color: var(--ws-text-soft);
  background: var(--ws-surface-soft);
}

.workspace-choice-row.selected {
  border-color: var(--ws-border);
  color: var(--ws-text);
  background: var(--ws-surface-soft);
}

.workspace-choice-row svg {
  width: 15px;
  height: 15px;
  fill: none;
  stroke: #4da9eb;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.workspace-profile-link {
  min-width: 0;
  min-height: 42px;
  padding: 4px 9px 4px 5px;
  border: 1px solid #252930;
  border-radius: 10px;
  display: flex;
  align-items: center;
  gap: 9px;
  color: inherit;
  background: #101217;
  transition: background-color .15s ease, border-color .15s ease;
}

.workspace-profile-link:hover {
  border-color: #373c45;
  background: #15181e;
}

.workspace-profile-link .mini-avatar {
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  border-color: #343943;
  background: #181b21;
}

.workspace-profile-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.workspace-profile-copy strong {
  max-width: 148px;
  overflow: hidden;
  color: #d9dde2;
  font-size: .74rem;
  line-height: 1.15;
  font-weight: 620;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-profile-copy small {
  color: #6f7781;
  font-size: .64rem;
  line-height: 1.1;
  font-weight: 540;
}

.workspace-profile-chevron {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  color: #555d68;
}

.workspace-canvas {
  padding-top: 28px;
}

@media (prefers-reduced-motion: reduce) {
  .workspace-nav a,
  .workspace-nav a::before,
  .workspace-nav a::after,
  .workspace-nav-icon,
  .workspace-nav a > span,
  .workspace-mode-indicator {
    transition: none;
  }

  .workspace-nav a:hover,
  .workspace-nav a:active,
  .workspace-nav a.active:hover,
  .workspace-nav a:hover .workspace-nav-icon,
  .workspace-nav a.active .workspace-nav-icon,
  .workspace-nav a:hover > span {
    transform: none;
  }
}

@media (max-width: 1260px) {
  .workspace-topbar-link {
    width: 36px;
    padding: 0;
    justify-content: center;
  }

  .workspace-topbar-link > span {
    display: none;
  }

  .workspace-mode-switch {
    width: 108px;
  }
}

@media (max-width: 1120px) {
  .workspace-topbar.workspace-topbar--contextual {
    padding: 0 24px;
  }

  .workspace-profile-copy,
  .workspace-profile-chevron {
    display: none;
  }

  .workspace-profile-link {
    min-width: 38px;
    padding: 4px 3px;
    justify-content: center;
  }
}

@media (max-width: 980px) {
  .workspace-topbar.workspace-topbar--contextual {
    height: 56px;
    padding: 0 14px;
  }

  .workspace-topbar--contextual .workspace-topbar-left {
    display: flex;
    align-items: center;
  }

  .workspace-breadcrumb-root,
  .workspace-breadcrumb-group,
  .workspace-breadcrumb-separator,
  .workspace-topbar-link,
  .workspace-topbar-divider {
    display: none;
  }

  .workspace-mode-switch {
    width: 92px;
    height: 34px;
  }

  .workspace-mode-indicator {
    height: 26px;
  }

  .workspace-mode-long {
    display: none;
  }

  .workspace-mode-short {
    display: inline;
  }

  .workspace-profile-link {
    min-height: 36px;
    border: 0;
    background: transparent;
  }

  .workspace-canvas {
    padding-top: 24px;
  }
}

@media (max-width: 620px) {
  .workspace-topbar-actions {
    gap: 5px;
  }

  .workspace-mode-switch {
    width: 82px;
  }

  .workspace-mode-switch button {
    font-size: .61rem;
  }

  .workspace-utility-button {
    min-width: 34px;
    height: 34px;
    padding: 0 7px;
  }

  .workspace-language-button span {
    display: none;
  }
}
</style>
