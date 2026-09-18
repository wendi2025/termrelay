<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import WorkspaceNavIcon from './WorkspaceNavIcon.vue'
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
import { type SmirelLocale } from '../core/i18n'

type UtilityPanel = 'notifications' | 'language' | 'theme'
type WorkspaceMode = 'user' | 'admin'

const router = useRouter()
const { t } = useI18n()
const { isAdmin } = useSession()
const root = ref<HTMLElement | null>(null)
const openUtility = ref<UtilityPanel | null>(null)

const MODE_STORAGE_KEY = 'smirel.workspace.mode'
const LAST_ADMIN_ROUTE_KEY = 'smirel.workspace.last-admin-route'
const LAST_USER_ROUTE_KEY = 'smirel.workspace.last-user-route'

function readSessionValue(key: string, fallback: string) {
  if (typeof window === 'undefined') return fallback
  return window.sessionStorage.getItem(key) || fallback
}

const homeMode = ref<WorkspaceMode>(
  isAdmin.value && readSessionValue(MODE_STORAGE_KEY, 'admin') !== 'user' ? 'admin' : 'user',
)

const currentLocaleShort = computed(() => interfacePreferences.locale === 'zh-CN' ? '中' : 'EN')
const unreadBadge = computed(() => unreadNotificationCount.value > 99 ? '99+' : String(unreadNotificationCount.value))

const quickLinks = computed(() => homeMode.value === 'admin' && isAdmin.value
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

function enterWorkspace(mode: WorkspaceMode) {
  homeMode.value = mode
  if (typeof window !== 'undefined') window.sessionStorage.setItem(MODE_STORAGE_KEY, mode)

  const fallback = mode === 'admin' ? '/admin/dashboard' : '/dashboard'
  const target = readSessionValue(mode === 'admin' ? LAST_ADMIN_ROUTE_KEY : LAST_USER_ROUTE_KEY, fallback)
  void router.push(target)
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

function handlePointerDown(event: PointerEvent) {
  if (!openUtility.value) return
  const target = event.target as Node | null
  if (target && !root.value?.contains(target)) openUtility.value = null
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') openUtility.value = null
}

onMounted(() => {
  document.addEventListener('pointerdown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div ref="root" class="home-workspace-controls">
    <RouterLink
      v-for="item in quickLinks"
      :key="item.path"
      :to="item.path"
      class="home-workspace-link"
      :title="item.label"
    >
      <WorkspaceNavIcon :name="item.icon" />
      <span>{{ item.label }}</span>
    </RouterLink>

    <div v-if="isAdmin" class="home-mode-switch" role="group" :aria-label="interfacePreferences.locale === 'zh-CN' ? '切换工作区' : 'Switch workspace'">
      <span class="home-mode-indicator" :class="{ 'is-admin': homeMode === 'admin' }" aria-hidden="true"></span>
      <button
        type="button"
        :class="{ active: homeMode === 'user' }"
        :aria-pressed="homeMode === 'user'"
        @click="enterWorkspace('user')"
      >{{ interfacePreferences.locale === 'zh-CN' ? '用户端' : 'User' }}</button>
      <button
        type="button"
        :class="{ active: homeMode === 'admin' }"
        :aria-pressed="homeMode === 'admin'"
        @click="enterWorkspace('admin')"
      >{{ interfacePreferences.locale === 'zh-CN' ? '管理端' : 'Admin' }}</button>
    </div>

    <span class="home-control-divider" aria-hidden="true"></span>

    <div class="home-utility-control">
      <button
        class="home-utility-button"
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
        <b v-if="unreadNotificationCount" class="home-notification-badge">{{ unreadBadge }}</b>
      </button>

      <section v-if="openUtility === 'notifications'" class="home-utility-popover home-notifications-panel">
        <header>
          <div>
            <strong>{{ t('utility.notificationCenter') }}</strong>
            <small v-if="unreadNotificationCount">{{ unreadNotificationCount }} {{ t('utility.notifications') }}</small>
          </div>
          <div class="home-notification-actions">
            <button type="button" :disabled="!unreadNotificationCount" @click="markAllNotificationsRead">{{ t('utility.markAllRead') }}</button>
            <button type="button" :disabled="!notifications.length" @click="clearNotifications">{{ t('utility.clearAll') }}</button>
          </div>
        </header>

        <div v-if="notifications.length" class="home-notification-list">
          <button
            v-for="item in notifications"
            :key="item.id"
            class="home-notification-item"
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
        <div v-else class="home-notification-empty">
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

    <div class="home-utility-control">
      <button
        class="home-utility-button home-language-button"
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
        <svg class="home-chevron" viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
      </button>

      <section v-if="openUtility === 'language'" class="home-utility-popover home-choice-panel">
        <header><strong>{{ t('utility.chooseLanguage') }}</strong></header>
        <button
          v-for="option in languageOptions"
          :key="option.value"
          class="home-choice-row"
          :class="{ selected: interfacePreferences.locale === option.value }"
          type="button"
          @click="chooseLocale(option.value)"
        >
          <span>{{ option.label }}</span>
          <svg v-if="interfacePreferences.locale === option.value" viewBox="0 0 16 16" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg>
        </button>
      </section>
    </div>

    <div class="home-utility-control">
      <button
        class="home-utility-button"
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

      <section v-if="openUtility === 'theme'" class="home-utility-popover home-choice-panel home-theme-panel">
        <header><strong>{{ t('utility.chooseTheme') }}</strong></header>
        <button
          v-for="option in themeOptions"
          :key="option.value"
          class="home-choice-row home-theme-choice"
          :class="{ selected: interfacePreferences.theme === option.value }"
          type="button"
          @click="chooseTheme(option.value)"
        >
          <i :class="`theme-swatch theme-${option.value}`"></i>
          <span>{{ option.label }}</span>
          <svg v-if="interfacePreferences.theme === option.value" viewBox="0 0 16 16" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg>
        </button>
      </section>
    </div>

    <span class="home-control-divider" aria-hidden="true"></span>
  </div>
</template>

<style scoped>
.home-workspace-controls {
  --home-toolbar-surface: var(--surface, #0f1115);
  --home-toolbar-soft: var(--surface-hover, #14171c);
  --home-toolbar-border: var(--border, #2a2f37);
  --home-toolbar-border-strong: var(--border-strong, #39414b);
  --home-toolbar-text: var(--text, #dce1e7);
  --home-toolbar-muted: var(--muted, #89939e);
  --home-toolbar-subtle: var(--subtle, #69737e);
  --home-toolbar-accent: var(--accent, #2f96e8);
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  white-space: nowrap;
}

.home-workspace-link,
.home-utility-button {
  height: 38px;
  border: 1px solid transparent;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--home-toolbar-muted);
  background: transparent;
  transition:
    color .16s ease,
    border-color .16s ease,
    background-color .16s ease,
    transform .16s cubic-bezier(.2, .75, .25, 1),
    box-shadow .16s ease;
}

.home-workspace-link {
  min-width: 38px;
  flex: 0 0 auto;
  padding: 0 10px;
  gap: 7px;
  font-size: .76rem;
  font-weight: 610;
  white-space: nowrap;
}

.home-workspace-link :deep(.workspace-nav-icon) {
  width: 16px;
  height: 16px;
  color: var(--home-toolbar-subtle);
}

.home-workspace-link:hover,
.home-utility-button:hover {
  border-color: var(--home-toolbar-border);
  color: var(--home-toolbar-text);
  background: var(--home-toolbar-soft);
  transform: translateY(-1px);
}

.home-workspace-link:hover :deep(.workspace-nav-icon) {
  color: var(--home-toolbar-text);
}

.home-mode-switch {
  position: relative;
  width: 164px;
  height: 42px;
  margin-left: 5px;
  padding: 4px;
  border: 1px solid var(--home-toolbar-border);
  border-radius: 11px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  background: var(--home-toolbar-surface);
  box-shadow: inset 0 1px color-mix(in srgb, var(--home-toolbar-text) 4%, transparent);
  flex: 0 0 auto;
}

.home-mode-indicator {
  position: absolute;
  top: 4px;
  left: 4px;
  width: calc(50% - 4px);
  height: 32px;
  border: 1px solid color-mix(in srgb, var(--home-toolbar-accent) 25%, var(--home-toolbar-border));
  border-radius: 8px;
  background: color-mix(in srgb, var(--home-toolbar-accent) 8%, var(--home-toolbar-surface));
  box-shadow: 0 3px 10px color-mix(in srgb, var(--home-toolbar-text) 8%, transparent);
  transition: transform .28s cubic-bezier(.22,1,.36,1);
}

.home-mode-indicator.is-admin {
  transform: translateX(100%);
}

.home-mode-switch button {
  position: relative;
  z-index: 1;
  border: 0;
  color: var(--home-toolbar-muted);
  background: transparent;
  font: inherit;
  font-size: .71rem;
  font-weight: 650;
  cursor: pointer;
}

.home-mode-switch button.active {
  color: color-mix(in srgb, var(--home-toolbar-accent) 78%, var(--home-toolbar-text));
}

.home-control-divider {
  width: 1px;
  height: 22px;
  margin: 0 7px;
  background: var(--home-toolbar-border);
}

.home-utility-control {
  position: relative;
}

.home-utility-button {
  position: relative;
  min-width: 40px;
  padding: 0 10px;
  border-color: var(--home-toolbar-border);
  background: var(--home-toolbar-surface);
  cursor: pointer;
}

.home-utility-button.active {
  border-color: color-mix(in srgb, var(--home-toolbar-accent) 34%, var(--home-toolbar-border-strong));
  color: var(--home-toolbar-text);
  background: color-mix(in srgb, var(--home-toolbar-accent) 6%, var(--home-toolbar-surface));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--home-toolbar-accent) 7%, transparent);
}

.home-utility-button svg {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.65;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.home-language-button {
  min-width: 74px;
  gap: 7px;
}

.home-language-button > span {
  font-size: .7rem;
  font-weight: 720;
}

.home-language-button .home-chevron {
  width: 12px;
  height: 12px;
  opacity: .55;
}

.home-notification-badge {
  position: absolute;
  top: -5px;
  right: -5px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border: 2px solid var(--bg, #090a0c);
  border-radius: 999px;
  color: #fff;
  background: var(--home-toolbar-accent);
  font-size: .49rem;
  line-height: 12px;
}

.home-utility-popover {
  position: absolute;
  z-index: 85;
  top: calc(100% + 10px);
  right: 0;
  border: 1px solid var(--home-toolbar-border-strong);
  border-radius: 13px;
  color: var(--home-toolbar-text);
  background: var(--home-toolbar-surface);
  box-shadow: 0 24px 60px rgba(0,0,0,.36);
  transform-origin: top right;
  animation: homeUtilityIn .16s cubic-bezier(.16,1,.3,1) both;
}

.home-notifications-panel {
  width: min(390px, calc(100vw - 28px));
  overflow: hidden;
}

.home-notifications-panel > header,
.home-choice-panel > header {
  min-height: 56px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--home-toolbar-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.home-notifications-panel > header > div:first-child {
  display: flex;
  align-items: center;
  gap: 8px;
}

.home-notifications-panel header strong,
.home-choice-panel header strong {
  font-size: .83rem;
  font-weight: 680;
}

.home-notifications-panel header small {
  padding: 3px 6px;
  border: 1px solid #29455b;
  border-radius: 999px;
  color: #78b9e7;
  font-size: .55rem;
}

.home-notification-actions {
  display: flex;
  gap: 3px;
}

.home-notification-actions button {
  min-height: 28px;
  padding: 0 7px;
  border: 1px solid transparent;
  border-radius: 7px;
  color: var(--home-toolbar-muted);
  background: transparent;
  font: inherit;
  font-size: .59rem;
  cursor: pointer;
}

.home-notification-actions button:hover:not(:disabled) {
  border-color: var(--home-toolbar-border);
  color: var(--home-toolbar-text);
  background: var(--home-toolbar-soft);
}

.home-notification-actions button:disabled {
  opacity: .34;
}

.home-notification-list {
  max-height: min(390px, calc(100vh - 132px));
  padding: 6px;
  overflow-y: auto;
}

.home-notification-item {
  width: 100%;
  min-height: 66px;
  padding: 10px 11px;
  border: 1px solid transparent;
  border-radius: 9px;
  display: grid;
  grid-template-columns: 7px minmax(0, 1fr);
  gap: 10px;
  color: var(--home-toolbar-muted);
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.home-notification-item:hover,
.home-notification-item.unread {
  border-color: var(--home-toolbar-border);
  background: var(--home-toolbar-soft);
}

.home-notification-item > i {
  width: 6px;
  height: 6px;
  margin-top: 5px;
  border-radius: 50%;
  background: #6f7882;
}

.home-notification-item.tone-info > i { background: #4f9fda; }
.home-notification-item.tone-success > i { background: #45b887; }
.home-notification-item.tone-warning > i { background: #d1a04e; }
.home-notification-item.tone-error > i { background: #d36b6b; }

.home-notification-item > span {
  position: relative;
  min-width: 0;
  padding-right: 62px;
  display: flex;
  flex-direction: column;
}

.home-notification-item strong {
  overflow: hidden;
  color: var(--home-toolbar-text);
  font-size: .73rem;
  font-weight: 640;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-notification-item small {
  margin-top: 4px;
  color: var(--home-toolbar-muted);
  font-size: .62rem;
  line-height: 1.4;
}

.home-notification-item time {
  position: absolute;
  top: 0;
  right: 0;
  color: var(--home-toolbar-subtle);
  font-size: .55rem;
}

.home-notification-empty {
  min-height: 190px;
  padding: 28px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--home-toolbar-muted);
}

.home-notification-empty > span {
  width: 40px;
  height: 40px;
  border: 1px solid var(--home-toolbar-border);
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: var(--home-toolbar-soft);
}

.home-notification-empty svg {
  width: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
}

.home-notification-empty strong {
  margin-top: 12px;
  color: var(--home-toolbar-text);
  font-size: .77rem;
}

.home-notification-empty small {
  margin-top: 5px;
  font-size: .62rem;
}

.home-choice-panel {
  width: 238px;
  padding: 6px;
}

.home-choice-panel > header {
  min-height: 48px;
  margin: 0 2px 4px;
  padding: 9px 10px;
}

.home-choice-row {
  width: 100%;
  min-height: 46px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 9px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 16px;
  align-items: center;
  gap: 9px;
  color: var(--home-toolbar-muted);
  background: transparent;
  font: inherit;
  font-size: .74rem;
  text-align: left;
  cursor: pointer;
}

.home-choice-row:hover,
.home-choice-row.selected {
  border-color: var(--home-toolbar-border);
  color: var(--home-toolbar-text);
  background: var(--home-toolbar-soft);
}

.home-choice-row > svg {
  width: 15px;
  height: 15px;
  fill: none;
  stroke: #62b2eb;
  stroke-width: 2;
}

.home-theme-panel {
  width: 270px;
}

.home-theme-choice {
  grid-template-columns: 18px minmax(0, 1fr) 16px;
}

.theme-swatch {
  width: 16px;
  height: 16px;
  border: 1px solid #48505a;
  border-radius: 50%;
}

.theme-dark { background: #0a0c0f; }
.theme-light { background: #f2f4f6; }
.theme-system { background: linear-gradient(135deg, #f2f4f6 0 49%, #171a1f 50% 100%); }

:global(html.smirel-app[data-theme='light'] .home-page) .home-utility-popover {
  box-shadow: 0 20px 50px rgba(35,47,59,.14);
}

@keyframes homeUtilityIn {
  from { opacity: 0; transform: translateY(-5px) scale(.982); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

@media (max-width: 1750px) {
  .home-workspace-link {
    width: 38px;
    padding: 0;
  }

  .home-workspace-link > span {
    display: none;
  }

  .home-mode-switch {
    width: 148px;
  }
}

@media (max-width: 1100px) {
  .home-mode-switch {
    width: 116px;
  }

  .home-mode-switch button {
    font-size: .66rem;
  }
}

@media (max-width: 1040px) {
  .home-workspace-link,
  .home-mode-switch,
  .home-control-divider:first-of-type {
    display: none;
  }
}

@media (max-width: 720px) {
  .home-language-button {
    min-width: 40px;
  }

  .home-language-button > span,
  .home-language-button .home-chevron {
    display: none;
  }

  .home-control-divider {
    margin-inline: 3px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-workspace-link,
  .home-utility-button,
  .home-mode-indicator,
  .home-utility-popover {
    animation: none;
    transition: none;
  }
}
</style>
