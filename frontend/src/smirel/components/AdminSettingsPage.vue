<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, getErrorMessage, previewMode } from '../core/api'
import { pushNotification } from '../core/notifications'

type SettingsTab = 'auth' | 'site' | 'security'
type OAuthProvider = 'google' | 'github'

interface PlatformSettings {
  registration_enabled: boolean
  email_verify_enabled: boolean
  password_reset_enabled: boolean
  invitation_code_enabled: boolean
  site_name: string
  site_subtitle: string
  frontend_url: string
  api_base_url: string
  github_oauth_enabled: boolean
  github_oauth_client_id: string
  github_oauth_client_secret_configured: boolean
  github_oauth_redirect_url: string
  github_oauth_frontend_redirect_url: string
  google_oauth_enabled: boolean
  google_oauth_client_id: string
  google_oauth_client_secret_configured: boolean
  google_oauth_redirect_url: string
  google_oauth_frontend_redirect_url: string
  turnstile_enabled: boolean
  turnstile_site_key: string
  turnstile_secret_key_configured: boolean
  totp_enabled: boolean
  passkey_enabled: boolean
  session_binding_enabled: boolean
  step_up_enabled: boolean
}

interface SettingsForm extends PlatformSettings {
  google_oauth_client_secret: string
  github_oauth_client_secret: string
  turnstile_secret_key: string
}

const { locale } = useI18n()
const activeTab = ref<SettingsTab>('auth')
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const baseline = ref('')
const showGoogleSecret = ref(false)
const showGithubSecret = ref(false)
const showTurnstileSecret = ref(false)

const form = reactive<SettingsForm>({
  registration_enabled: true,
  email_verify_enabled: false,
  password_reset_enabled: true,
  invitation_code_enabled: false,
  site_name: 'Smirel',
  site_subtitle: 'API SERVICE',
  frontend_url: '',
  api_base_url: '',
  github_oauth_enabled: false,
  github_oauth_client_id: '',
  github_oauth_client_secret_configured: false,
  github_oauth_client_secret: '',
  github_oauth_redirect_url: '',
  github_oauth_frontend_redirect_url: '',
  google_oauth_enabled: false,
  google_oauth_client_id: '',
  google_oauth_client_secret_configured: false,
  google_oauth_client_secret: '',
  google_oauth_redirect_url: '',
  google_oauth_frontend_redirect_url: '',
  turnstile_enabled: false,
  turnstile_site_key: '',
  turnstile_secret_key_configured: false,
  turnstile_secret_key: '',
  totp_enabled: false,
  passkey_enabled: false,
  session_binding_enabled: false,
  step_up_enabled: false,
})

function text(zh: string, en: string) {
  return String(locale.value || '').toLowerCase().startsWith('zh') ? zh : en
}

const tabs = computed(() => [
  { key: 'auth' as const, label: text('登录与认证', 'Authentication'), desc: text('注册方式与 OAuth', 'Signup & OAuth') },
  { key: 'site' as const, label: text('基础信息', 'General'), desc: text('站点与域名', 'Site & domains') },
  { key: 'security' as const, label: text('安全防护', 'Security'), desc: text('验证与会话安全', 'Verification & sessions') },
])

function serializableState() {
  return JSON.stringify({
    registration_enabled: form.registration_enabled,
    email_verify_enabled: form.email_verify_enabled,
    password_reset_enabled: form.password_reset_enabled,
    invitation_code_enabled: form.invitation_code_enabled,
    site_name: form.site_name,
    site_subtitle: form.site_subtitle,
    frontend_url: form.frontend_url,
    api_base_url: form.api_base_url,
    github_oauth_enabled: form.github_oauth_enabled,
    github_oauth_client_id: form.github_oauth_client_id,
    github_oauth_client_secret: form.github_oauth_client_secret,
    github_oauth_redirect_url: form.github_oauth_redirect_url,
    github_oauth_frontend_redirect_url: form.github_oauth_frontend_redirect_url,
    google_oauth_enabled: form.google_oauth_enabled,
    google_oauth_client_id: form.google_oauth_client_id,
    google_oauth_client_secret: form.google_oauth_client_secret,
    google_oauth_redirect_url: form.google_oauth_redirect_url,
    google_oauth_frontend_redirect_url: form.google_oauth_frontend_redirect_url,
    turnstile_enabled: form.turnstile_enabled,
    turnstile_site_key: form.turnstile_site_key,
    turnstile_secret_key: form.turnstile_secret_key,
    totp_enabled: form.totp_enabled,
    passkey_enabled: form.passkey_enabled,
    session_binding_enabled: form.session_binding_enabled,
    step_up_enabled: form.step_up_enabled,
  })
}

const dirty = computed(() => !loading.value && serializableState() !== baseline.value)
const googleConfigured = computed(() => Boolean(
  form.google_oauth_client_id.trim() &&
  (form.google_oauth_client_secret_configured || form.google_oauth_client_secret.trim()) &&
  form.google_oauth_redirect_url.trim(),
))
const githubConfigured = computed(() => Boolean(
  form.github_oauth_client_id.trim() &&
  (form.github_oauth_client_secret_configured || form.github_oauth_client_secret.trim()) &&
  form.github_oauth_redirect_url.trim(),
))

function applySettings(data: Partial<PlatformSettings>) {
  const boolKeys: Array<keyof PlatformSettings> = [
    'registration_enabled', 'email_verify_enabled', 'password_reset_enabled', 'invitation_code_enabled',
    'github_oauth_enabled', 'github_oauth_client_secret_configured',
    'google_oauth_enabled', 'google_oauth_client_secret_configured',
    'turnstile_enabled', 'turnstile_secret_key_configured', 'totp_enabled', 'passkey_enabled',
    'session_binding_enabled', 'step_up_enabled',
  ]
  const stringKeys: Array<keyof PlatformSettings> = [
    'site_name', 'site_subtitle', 'frontend_url', 'api_base_url',
    'github_oauth_client_id', 'github_oauth_redirect_url', 'github_oauth_frontend_redirect_url',
    'google_oauth_client_id', 'google_oauth_redirect_url', 'google_oauth_frontend_redirect_url',
    'turnstile_site_key',
  ]

  for (const key of boolKeys) {
    if (typeof data[key] === 'boolean') (form as unknown as Record<string, unknown>)[key] = data[key]
  }
  for (const key of stringKeys) {
    if (typeof data[key] === 'string') (form as unknown as Record<string, unknown>)[key] = data[key]
  }

  form.google_oauth_client_secret = ''
  form.github_oauth_client_secret = ''
  form.turnstile_secret_key = ''
}

function previewSettings(): PlatformSettings {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    password_reset_enabled: true,
    invitation_code_enabled: false,
    site_name: 'Smirel',
    site_subtitle: 'API SERVICE',
    frontend_url: 'https://relay.smirel.com',
    api_base_url: 'https://api.smirel.com',
    github_oauth_enabled: false,
    github_oauth_client_id: '',
    github_oauth_client_secret_configured: false,
    github_oauth_redirect_url: '',
    github_oauth_frontend_redirect_url: '',
    google_oauth_enabled: false,
    google_oauth_client_id: '',
    google_oauth_client_secret_configured: false,
    google_oauth_redirect_url: '',
    google_oauth_frontend_redirect_url: '',
    turnstile_enabled: false,
    turnstile_site_key: '',
    turnstile_secret_key_configured: false,
    totp_enabled: false,
    passkey_enabled: false,
    session_binding_enabled: false,
    step_up_enabled: false,
  }
}

async function loadSettings() {
  loading.value = true
  error.value = ''
  try {
    const data = previewMode ? previewSettings() : (await api.get<PlatformSettings>('/admin/settings')).data
    applySettings(data)
    baseline.value = serializableState()
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    loading.value = false
  }
}

function inferredApiOrigin() {
  const configured = form.api_base_url.trim()
  if (configured) return configured.replace(/\/+$/, '')
  if (typeof window === 'undefined') return ''
  const current = new URL(window.location.origin)
  if (current.hostname.startsWith('relay.')) current.hostname = current.hostname.replace(/^relay\./, 'api.')
  return current.origin.replace(/\/+$/, '')
}

function apiRoot() {
  const base = inferredApiOrigin()
  return base.endsWith('/api/v1') ? base : `${base}/api/v1`
}

function recommendedCallback(provider: OAuthProvider) {
  return `${apiRoot()}/auth/oauth/${provider}/callback`
}

function recommendedFrontendCallback() {
  const base = (form.frontend_url.trim() || (typeof window !== 'undefined' ? window.location.origin : '')).replace(/\/+$/, '')
  return `${base}/auth/oauth/callback`
}

async function copyValue(value: string, successMessage?: string) {
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    pushNotification({ title: successMessage || text('已复制', 'Copied'), tone: 'success' })
  } catch {
    pushNotification({ title: text('复制失败，请手动复制', 'Copy failed. Please copy manually.'), tone: 'warning' })
  }
}

async function fillCallback(provider: OAuthProvider) {
  const value = recommendedCallback(provider)
  if (provider === 'google') form.google_oauth_redirect_url = value
  else form.github_oauth_redirect_url = value
  await copyValue(value, text('后端回调地址已生成并复制', 'Backend callback URL generated and copied'))
}

async function fillFrontendCallback(provider: OAuthProvider) {
  const value = recommendedFrontendCallback()
  if (provider === 'google') form.google_oauth_frontend_redirect_url = value
  else form.github_oauth_frontend_redirect_url = value
  await copyValue(value, text('前端回调地址已生成并复制', 'Frontend callback URL generated and copied'))
}

function validateOAuth(provider: OAuthProvider) {
  const enabled = provider === 'google' ? form.google_oauth_enabled : form.github_oauth_enabled
  if (!enabled) return ''
  const clientId = provider === 'google' ? form.google_oauth_client_id : form.github_oauth_client_id
  const secretConfigured = provider === 'google'
    ? form.google_oauth_client_secret_configured || Boolean(form.google_oauth_client_secret.trim())
    : form.github_oauth_client_secret_configured || Boolean(form.github_oauth_client_secret.trim())
  const redirect = provider === 'google' ? form.google_oauth_redirect_url : form.github_oauth_redirect_url
  const frontendRedirect = provider === 'google' ? form.google_oauth_frontend_redirect_url : form.github_oauth_frontend_redirect_url
  const name = provider === 'google' ? 'Google' : 'GitHub'
  if (!clientId.trim()) return `${name} Client ID ${text('不能为空', 'is required')}`
  if (!secretConfigured) return `${name} Client Secret ${text('不能为空', 'is required')}`
  if (!redirect.trim()) return `${name} ${text('后端回调地址不能为空', 'backend callback URL is required')}`
  if (!frontendRedirect.trim()) return `${name} ${text('前端回调地址不能为空', 'frontend callback URL is required')}`
  return ''
}

async function saveSettings() {
  if (saving.value || loading.value) return
  const validationError = validateOAuth('google') || validateOAuth('github')
  if (validationError) {
    error.value = validationError
    activeTab.value = 'auth'
    return
  }

  saving.value = true
  error.value = ''
  try {
    const payload: Record<string, unknown> = {
      registration_enabled: form.registration_enabled,
      email_verify_enabled: form.email_verify_enabled,
      password_reset_enabled: form.password_reset_enabled,
      invitation_code_enabled: form.invitation_code_enabled,
      site_name: form.site_name.trim(),
      site_subtitle: form.site_subtitle.trim(),
      frontend_url: form.frontend_url.trim(),
      api_base_url: form.api_base_url.trim(),
      google_oauth_enabled: form.google_oauth_enabled,
      google_oauth_client_id: form.google_oauth_client_id.trim(),
      google_oauth_redirect_url: form.google_oauth_redirect_url.trim(),
      google_oauth_frontend_redirect_url: form.google_oauth_frontend_redirect_url.trim(),
      github_oauth_enabled: form.github_oauth_enabled,
      github_oauth_client_id: form.github_oauth_client_id.trim(),
      github_oauth_redirect_url: form.github_oauth_redirect_url.trim(),
      github_oauth_frontend_redirect_url: form.github_oauth_frontend_redirect_url.trim(),
      turnstile_enabled: form.turnstile_enabled,
      turnstile_site_key: form.turnstile_site_key.trim(),
      totp_enabled: form.totp_enabled,
      passkey_enabled: form.passkey_enabled,
      session_binding_enabled: form.session_binding_enabled,
      step_up_enabled: form.step_up_enabled,
    }
    if (form.google_oauth_client_secret.trim()) payload.google_oauth_client_secret = form.google_oauth_client_secret.trim()
    if (form.github_oauth_client_secret.trim()) payload.github_oauth_client_secret = form.github_oauth_client_secret.trim()
    if (form.turnstile_secret_key.trim()) payload.turnstile_secret_key = form.turnstile_secret_key.trim()

    const updated = previewMode ? ({ ...previewSettings(), ...payload } as PlatformSettings) : (await api.put<PlatformSettings>('/admin/settings', payload)).data
    applySettings(updated)
    baseline.value = serializableState()
    pushNotification({
      title: text('平台设置已保存', 'Platform settings saved'),
      message: text('新的登录与安全配置已经生效。', 'Authentication and security settings are now active.'),
      tone: 'success',
    })
  } catch (caught) {
    error.value = getErrorMessage(caught)
    pushNotification({ title: text('保存失败', 'Save failed'), message: error.value, tone: 'error' })
  } finally {
    saving.value = false
  }
}

function goToLogin() {
  if (typeof window !== 'undefined') window.open('/login', '_blank', 'noopener,noreferrer')
}

onMounted(() => void loadSettings())
</script>

<template>
  <div class="commercial-settings">
    <section class="settings-command glass">
      <div class="command-status">
        <span class="status-orb" :class="{ dirty }"></span>
        <div>
          <strong>{{ dirty ? text('有未保存的更改', 'Unsaved changes') : text('配置已同步', 'Configuration synced') }}</strong>
          <small>{{ text('修改会作用于登录、注册和平台安全策略', 'Changes affect sign-in, registration and platform security') }}</small>
        </div>
      </div>
      <div class="command-actions">
        <button class="secondary-button" type="button" :disabled="loading || saving" @click="loadSettings">
          {{ text('重新加载', 'Reload') }}
        </button>
        <button class="primary-button save-settings-button" type="button" :disabled="loading || saving || !dirty" @click="saveSettings">
          <span v-if="saving" class="button-spinner"></span>
          {{ saving ? text('保存中', 'Saving') : text('保存更改', 'Save changes') }}
        </button>
      </div>
    </section>

    <p v-if="error" class="settings-error">{{ error }}</p>

    <div v-if="loading" class="settings-loading glass">
      <span class="loading-ring"></span>
      <div><strong>{{ text('正在载入平台配置', 'Loading platform settings') }}</strong><small>{{ text('正在读取当前配置状态…', 'Reading the current configuration…') }}</small></div>
    </div>

    <div v-else class="settings-layout">
      <aside class="settings-side">
        <nav class="settings-nav glass" :aria-label="text('平台设置分类', 'Platform settings sections')">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="settings-nav-item"
            :class="{ active: activeTab === tab.key }"
            @click="activeTab = tab.key"
          >
            <span class="nav-icon" aria-hidden="true">
              <svg v-if="tab.key === 'auth'" viewBox="0 0 24 24"><path d="M12 3 5 6v5c0 4.4 2.8 8.4 7 10 4.2-1.6 7-5.6 7-10V6l-7-3Z"/><path d="M9.5 12.2 11 13.7l3.8-4"/></svg>
              <svg v-else-if="tab.key === 'site'" viewBox="0 0 24 24"><path d="M4 5.5h16v13H4z"/><path d="M4 9h16M8 5.5V9"/></svg>
              <svg v-else viewBox="0 0 24 24"><path d="M12 3 5 6v5c0 4.5 2.7 8.5 7 10 4.3-1.5 7-5.5 7-10V6l-7-3Z"/><path d="M9.5 11.5V10a2.5 2.5 0 0 1 5 0v1.5M9 11.5h6v5H9z"/></svg>
            </span>
            <span><strong>{{ tab.label }}</strong><small>{{ tab.desc }}</small></span>
            <i></i>
          </button>
        </nav>

        <section class="settings-context glass">
          <span>{{ text('当前环境', 'Environment') }}</span>
          <strong>Production</strong>
          <div><i></i>{{ text('服务运行中', 'Service online') }}</div>
          <code>{{ inferredApiOrigin() || '—' }}</code>
        </section>
      </aside>

      <main class="settings-content">
        <template v-if="activeTab === 'auth'">
          <section class="settings-card glass">
            <header class="settings-card-head">
              <div>
                <span class="section-kicker">OAUTH 2.0</span>
                <h2>{{ text('第三方登录', 'Social sign-in') }}</h2>
                <p>{{ text('配置 Google 与 GitHub 登录。启用后，登录页会自动显示对应入口。', 'Configure Google and GitHub sign-in. Enabled providers appear automatically on the login page.') }}</p>
              </div>
              <button class="provider-test-button" type="button" @click="goToLogin">{{ text('打开登录页', 'Open sign-in page') }} ↗</button>
            </header>

            <div class="provider-stack">
              <article class="provider-panel" :class="{ enabled: form.google_oauth_enabled }">
                <header class="provider-head">
                  <div class="provider-identity">
                    <span class="provider-logo google-logo" aria-hidden="true">
                      <svg viewBox="0 0 24 24"><path fill="#4285F4" d="M21.6 12.23c0-.71-.06-1.4-.18-2.07H12v3.92h5.39a4.61 4.61 0 0 1-2 3.02v2.54h3.23c1.89-1.74 2.98-4.3 2.98-7.41Z"/><path fill="#34A853" d="M12 22c2.7 0 4.96-.9 6.62-2.36l-3.23-2.54c-.9.6-2.04.95-3.39.95-2.61 0-4.82-1.76-5.61-4.13H3.05v2.62A10 10 0 0 0 12 22Z"/><path fill="#FBBC05" d="M6.39 13.92A6 6 0 0 1 6.08 12c0-.67.12-1.32.31-1.92V7.46H3.05A10 10 0 0 0 2 12c0 1.61.39 3.14 1.05 4.54l3.34-2.62Z"/><path fill="#EA4335" d="M12 5.95c1.47 0 2.79.51 3.83 1.5l2.87-2.87A9.65 9.65 0 0 0 12 2a10 10 0 0 0-8.95 5.46l3.34 2.62C7.18 7.71 9.39 5.95 12 5.95Z"/></svg>
                    </span>
                    <div><strong>Google</strong><small>OAuth 2.0 · Web Application</small></div>
                  </div>
                  <div class="provider-state-wrap">
                    <span class="provider-state" :class="{ ready: googleConfigured }"><i></i>{{ googleConfigured ? text('已配置', 'Configured') : text('未配置', 'Not configured') }}</span>
                    <button class="switch-control" :class="{ on: form.google_oauth_enabled }" type="button" role="switch" :aria-checked="form.google_oauth_enabled" @click="form.google_oauth_enabled = !form.google_oauth_enabled"><span></span></button>
                  </div>
                </header>

                <div class="provider-fields">
                  <label class="settings-field">
                    <span>Client ID</span>
                    <input v-model="form.google_oauth_client_id" type="text" autocomplete="off" placeholder="xxxx.apps.googleusercontent.com" />
                  </label>
                  <label class="settings-field">
                    <span>Client Secret <em v-if="form.google_oauth_client_secret_configured">{{ text('已保存', 'Saved') }}</em></span>
                    <div class="secret-input">
                      <input v-model="form.google_oauth_client_secret" :type="showGoogleSecret ? 'text' : 'password'" autocomplete="new-password" :placeholder="form.google_oauth_client_secret_configured ? text('已配置，留空则保持不变', 'Configured — leave blank to keep it') : text('请输入 Client Secret', 'Enter Client Secret')" />
                      <button type="button" @click="showGoogleSecret = !showGoogleSecret">{{ showGoogleSecret ? text('隐藏', 'Hide') : text('显示', 'Show') }}</button>
                    </div>
                  </label>
                  <label class="settings-field field-wide">
                    <span>{{ text('后端授权回调地址', 'Backend authorization callback') }}</span>
                    <div class="url-input-row">
                      <input v-model="form.google_oauth_redirect_url" type="url" placeholder="https://api.example.com/api/v1/auth/oauth/google/callback" />
                      <button type="button" @click="fillCallback('google')">{{ text('生成并复制', 'Generate & copy') }}</button>
                    </div>
                    <small>{{ text('这一个地址填写到 Google Cloud 的 Authorized redirect URIs。', 'Use this exact URL in Google Cloud → Authorized redirect URIs.') }}</small>
                  </label>
                  <label class="settings-field field-wide">
                    <span>{{ text('登录完成返回地址', 'Frontend return URL') }}</span>
                    <div class="url-input-row">
                      <input v-model="form.google_oauth_frontend_redirect_url" type="url" placeholder="https://relay.example.com/auth/oauth/callback" />
                      <button type="button" @click="fillFrontendCallback('google')">{{ text('生成并复制', 'Generate & copy') }}</button>
                    </div>
                  </label>
                </div>
              </article>

              <article class="provider-panel" :class="{ enabled: form.github_oauth_enabled }">
                <header class="provider-head">
                  <div class="provider-identity">
                    <span class="provider-logo github-logo" aria-hidden="true">
                      <svg viewBox="0 0 24 24"><path d="M12 2.2a10 10 0 0 0-3.16 19.49c.5.09.68-.22.68-.48v-1.88c-2.78.6-3.36-1.18-3.36-1.18-.45-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.61.07-.61 1 .07 1.53 1.03 1.53 1.03.9 1.53 2.35 1.09 2.92.83.09-.65.35-1.09.64-1.34-2.22-.25-4.55-1.11-4.55-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.64 0 0 .84-.27 2.75 1.02A9.54 9.54 0 0 1 12 7.01a9.5 9.5 0 0 1 2.5.34c1.91-1.29 2.75-1.02 2.75-1.02.55 1.37.2 2.39.1 2.64.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.68-4.57 4.93.36.31.68.92.68 1.86v2.77c0 .27.18.58.69.48A10 10 0 0 0 12 2.2Z"/></svg>
                    </span>
                    <div><strong>GitHub</strong><small>OAuth App · Web</small></div>
                  </div>
                  <div class="provider-state-wrap">
                    <span class="provider-state" :class="{ ready: githubConfigured }"><i></i>{{ githubConfigured ? text('已配置', 'Configured') : text('未配置', 'Not configured') }}</span>
                    <button class="switch-control" :class="{ on: form.github_oauth_enabled }" type="button" role="switch" :aria-checked="form.github_oauth_enabled" @click="form.github_oauth_enabled = !form.github_oauth_enabled"><span></span></button>
                  </div>
                </header>

                <div class="provider-fields">
                  <label class="settings-field">
                    <span>Client ID</span>
                    <input v-model="form.github_oauth_client_id" type="text" autocomplete="off" placeholder="GitHub OAuth Client ID" />
                  </label>
                  <label class="settings-field">
                    <span>Client Secret <em v-if="form.github_oauth_client_secret_configured">{{ text('已保存', 'Saved') }}</em></span>
                    <div class="secret-input">
                      <input v-model="form.github_oauth_client_secret" :type="showGithubSecret ? 'text' : 'password'" autocomplete="new-password" :placeholder="form.github_oauth_client_secret_configured ? text('已配置，留空则保持不变', 'Configured — leave blank to keep it') : text('请输入 Client Secret', 'Enter Client Secret')" />
                      <button type="button" @click="showGithubSecret = !showGithubSecret">{{ showGithubSecret ? text('隐藏', 'Hide') : text('显示', 'Show') }}</button>
                    </div>
                  </label>
                  <label class="settings-field field-wide">
                    <span>{{ text('后端授权回调地址', 'Backend authorization callback') }}</span>
                    <div class="url-input-row">
                      <input v-model="form.github_oauth_redirect_url" type="url" placeholder="https://api.example.com/api/v1/auth/oauth/github/callback" />
                      <button type="button" @click="fillCallback('github')">{{ text('生成并复制', 'Generate & copy') }}</button>
                    </div>
                  </label>
                  <label class="settings-field field-wide">
                    <span>{{ text('登录完成返回地址', 'Frontend return URL') }}</span>
                    <div class="url-input-row">
                      <input v-model="form.github_oauth_frontend_redirect_url" type="url" placeholder="https://relay.example.com/auth/oauth/callback" />
                      <button type="button" @click="fillFrontendCallback('github')">{{ text('生成并复制', 'Generate & copy') }}</button>
                    </div>
                  </label>
                </div>
              </article>
            </div>
          </section>

          <section class="settings-card glass">
            <header class="settings-card-head compact-head">
              <div><span class="section-kicker">SIGN UP</span><h2>{{ text('注册策略', 'Registration policy') }}</h2><p>{{ text('控制用户如何创建和恢复账号。', 'Control how users create and recover accounts.') }}</p></div>
            </header>
            <div class="toggle-list">
              <div class="toggle-row"><div><strong>{{ text('开放注册', 'Public registration') }}</strong><small>{{ text('允许新用户自行创建账户', 'Allow new users to create accounts') }}</small></div><button class="switch-control" :class="{ on: form.registration_enabled }" type="button" @click="form.registration_enabled = !form.registration_enabled"><span></span></button></div>
              <div class="toggle-row"><div><strong>{{ text('邮箱验证', 'Email verification') }}</strong><small>{{ text('注册时要求验证邮箱地址', 'Require email verification during signup') }}</small></div><button class="switch-control" :class="{ on: form.email_verify_enabled }" type="button" @click="form.email_verify_enabled = !form.email_verify_enabled"><span></span></button></div>
              <div class="toggle-row"><div><strong>{{ text('找回密码', 'Password recovery') }}</strong><small>{{ text('允许用户通过邮箱重置密码', 'Allow password reset by email') }}</small></div><button class="switch-control" :class="{ on: form.password_reset_enabled }" type="button" @click="form.password_reset_enabled = !form.password_reset_enabled"><span></span></button></div>
              <div class="toggle-row"><div><strong>{{ text('邀请码注册', 'Invitation-only signup') }}</strong><small>{{ text('开启后，新账号需要邀请码', 'Require an invitation code for new accounts') }}</small></div><button class="switch-control" :class="{ on: form.invitation_code_enabled }" type="button" @click="form.invitation_code_enabled = !form.invitation_code_enabled"><span></span></button></div>
            </div>
          </section>
        </template>

        <template v-else-if="activeTab === 'site'">
          <section class="settings-card glass">
            <header class="settings-card-head"><div><span class="section-kicker">BRANDING</span><h2>{{ text('站点信息', 'Site identity') }}</h2><p>{{ text('这些信息用于平台标题、基础品牌展示和链接生成。', 'Used for platform identity, branding and generated links.') }}</p></div></header>
            <div class="general-form">
              <label class="settings-field"><span>{{ text('站点名称', 'Site name') }}</span><input v-model="form.site_name" type="text" placeholder="Smirel" /></label>
              <label class="settings-field"><span>{{ text('站点副标题', 'Site subtitle') }}</span><input v-model="form.site_subtitle" type="text" placeholder="API SERVICE" /></label>
              <label class="settings-field field-wide"><span>{{ text('前端访问地址', 'Frontend URL') }}</span><input v-model="form.frontend_url" type="url" placeholder="https://relay.smirel.com" /><small>{{ text('用于 OAuth 登录完成后返回用户控制台。', 'Used as the return origin after OAuth sign-in.') }}</small></label>
              <label class="settings-field field-wide"><span>{{ text('API 公网地址', 'Public API URL') }}</span><input v-model="form.api_base_url" type="url" placeholder="https://api.smirel.com" /><small>{{ text('用于生成 OAuth 后端回调地址；可填写域名或完整 /api/v1 地址。', 'Used to generate backend OAuth callbacks; domain or full /api/v1 URL is accepted.') }}</small></label>
            </div>
          </section>

          <section class="settings-card glass domain-preview-card">
            <header class="settings-card-head compact-head"><div><span class="section-kicker">ROUTING</span><h2>{{ text('地址预览', 'URL preview') }}</h2><p>{{ text('根据当前填写内容实时生成，不会额外保存密钥。', 'Generated live from the values above; no secrets are exposed.') }}</p></div></header>
            <div class="route-preview-list">
              <div><span>{{ text('Google 回调', 'Google callback') }}</span><code>{{ recommendedCallback('google') }}</code><button type="button" @click="copyValue(recommendedCallback('google'))">{{ text('复制', 'Copy') }}</button></div>
              <div><span>{{ text('GitHub 回调', 'GitHub callback') }}</span><code>{{ recommendedCallback('github') }}</code><button type="button" @click="copyValue(recommendedCallback('github'))">{{ text('复制', 'Copy') }}</button></div>
              <div><span>{{ text('前端返回', 'Frontend return') }}</span><code>{{ recommendedFrontendCallback() }}</code><button type="button" @click="copyValue(recommendedFrontendCallback())">{{ text('复制', 'Copy') }}</button></div>
            </div>
          </section>
        </template>

        <template v-else>
          <section class="settings-card glass">
            <header class="settings-card-head"><div><span class="section-kicker">SECURITY</span><h2>{{ text('账号与会话安全', 'Account & session security') }}</h2><p>{{ text('为管理操作和用户登录增加额外保护。', 'Add additional protection to sign-in and sensitive operations.') }}</p></div></header>
            <div class="toggle-list security-toggle-list">
              <div class="toggle-row"><div><strong>TOTP 2FA</strong><small>{{ text('允许用户配置动态验证码', 'Allow time-based one-time password authentication') }}</small></div><button class="switch-control" :class="{ on: form.totp_enabled }" type="button" @click="form.totp_enabled = !form.totp_enabled"><span></span></button></div>
              <div class="toggle-row"><div><strong>Passkey</strong><small>{{ text('启用 WebAuthn / 无密码登录能力', 'Enable WebAuthn and passwordless sign-in') }}</small></div><button class="switch-control" :class="{ on: form.passkey_enabled }" type="button" @click="form.passkey_enabled = !form.passkey_enabled"><span></span></button></div>
              <div class="toggle-row"><div><strong>{{ text('会话绑定', 'Session binding') }}</strong><small>{{ text('将登录会话与 IP / 浏览器环境绑定', 'Bind sessions to IP and browser context') }}</small></div><button class="switch-control" :class="{ on: form.session_binding_enabled }" type="button" @click="form.session_binding_enabled = !form.session_binding_enabled"><span></span></button></div>
              <div class="toggle-row"><div><strong>Step-up 2FA</strong><small>{{ text('敏感操作要求再次进行二次验证', 'Require re-authentication for sensitive actions') }}</small></div><button class="switch-control" :class="{ on: form.step_up_enabled }" type="button" @click="form.step_up_enabled = !form.step_up_enabled"><span></span></button></div>
            </div>
          </section>

          <section class="settings-card glass">
            <header class="settings-card-head"><div><span class="section-kicker">TURNSTILE</span><h2>Cloudflare Turnstile</h2><p>{{ text('在登录与注册入口增加人机验证，减少机器人请求。', 'Add bot protection to sign-in and registration flows.') }}</p></div><button class="switch-control" :class="{ on: form.turnstile_enabled }" type="button" @click="form.turnstile_enabled = !form.turnstile_enabled"><span></span></button></header>
            <div class="general-form turnstile-form">
              <label class="settings-field"><span>Site Key</span><input v-model="form.turnstile_site_key" type="text" autocomplete="off" placeholder="0x4AAAA..." /></label>
              <label class="settings-field"><span>Secret Key <em v-if="form.turnstile_secret_key_configured">{{ text('已保存', 'Saved') }}</em></span><div class="secret-input"><input v-model="form.turnstile_secret_key" :type="showTurnstileSecret ? 'text' : 'password'" autocomplete="new-password" :placeholder="form.turnstile_secret_key_configured ? text('已配置，留空则保持不变', 'Configured — leave blank to keep it') : text('请输入 Secret Key', 'Enter Secret Key')" /><button type="button" @click="showTurnstileSecret = !showTurnstileSecret">{{ showTurnstileSecret ? text('隐藏', 'Hide') : text('显示', 'Show') }}</button></div></label>
            </div>
            <div v-if="form.turnstile_enabled && (!form.turnstile_site_key.trim() || (!form.turnstile_secret_key_configured && !form.turnstile_secret_key.trim()))" class="security-note"><strong>{{ text('需要补充配置', 'Configuration required') }}</strong><span>{{ text('Turnstile 已开启，但 Site Key 或 Secret Key 尚未完整配置。', 'Turnstile is enabled but the Site Key or Secret Key is still missing.') }}</span></div>
          </section>
        </template>
      </main>
    </div>
  </div>
</template>

<style scoped>
.commercial-settings { width: 100%; }
.settings-command { min-height: 72px; padding: 14px 16px; margin-bottom: 16px; border-radius: 11px; display: flex; align-items: center; justify-content: space-between; gap: 18px; }
.command-status { display: flex; align-items: center; gap: 12px; min-width: 0; }
.status-orb { width: 10px; height: 10px; border-radius: 50%; flex: 0 0 auto; background: #42ce99; box-shadow: 0 0 0 4px rgba(66,206,153,.08); }
.status-orb.dirty { background: #f0aa4f; box-shadow: 0 0 0 4px rgba(240,170,79,.08); }
.command-status > div { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.command-status strong { color: #e8ebef; font-size: .83rem; font-weight: 640; }
.command-status small { color: #737b86; font-size: .72rem; }
.command-actions { display: flex; align-items: center; gap: 8px; }
.command-actions button:disabled { opacity: .48; cursor: not-allowed; }
.save-settings-button { min-width: 104px; display: inline-flex; align-items: center; justify-content: center; gap: 8px; }
.button-spinner, .loading-ring { border-radius: 50%; border: 2px solid rgba(255,255,255,.18); border-top-color: currentColor; animation: settings-spin .75s linear infinite; }
.button-spinner { width: 14px; height: 14px; }
.settings-error { margin: 0 0 16px; padding: 11px 13px; border: 1px solid #4b252c; border-radius: 8px; background: #1a1013; color: #ee9299; font-size: .78rem; }
.settings-loading { min-height: 260px; border-radius: 11px; display: flex; align-items: center; justify-content: center; gap: 14px; }
.loading-ring { width: 24px; height: 24px; color: #55a8e9; border-color: #202f3c; border-top-color: #55a8e9; }
.settings-loading > div { display: flex; flex-direction: column; gap: 4px; }
.settings-loading strong { color: #dfe3e8; font-size: .86rem; }
.settings-loading small { color: #707985; font-size: .72rem; }
.settings-layout { display: grid; grid-template-columns: 224px minmax(0, 1fr); gap: 16px; align-items: start; }
.settings-side { position: sticky; top: 82px; display: flex; flex-direction: column; gap: 12px; }
.settings-nav { padding: 7px; border-radius: 11px; }
.settings-nav-item { width: 100%; min-height: 58px; padding: 8px 9px; border: 1px solid transparent; border-radius: 8px; background: transparent; color: #858d97; display: grid; grid-template-columns: 34px minmax(0,1fr) 4px; align-items: center; gap: 9px; text-align: left; cursor: pointer; transition: .15s ease; }
.settings-nav-item:hover { border-color: #242a31; background: #121419; color: #d7dce2; }
.settings-nav-item.active { border-color: #2a3b4b; background: #12202d; color: #f1f5f9; }
.settings-nav-item > span:nth-child(2) { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.settings-nav-item strong { font-size: .79rem; font-weight: 620; }
.settings-nav-item small { color: #69727d; font-size: .66rem; }
.settings-nav-item.active small { color: #8295a7; }
.settings-nav-item > i { width: 3px; height: 22px; border-radius: 2px; background: transparent; }
.settings-nav-item.active > i { background: #3aa1ee; }
.nav-icon { width: 34px; height: 34px; border: 1px solid #282d34; border-radius: 8px; background: #0c0e12; display: grid; place-items: center; }
.nav-icon svg { width: 17px; height: 17px; fill: none; stroke: currentColor; stroke-width: 1.6; stroke-linecap: round; stroke-linejoin: round; }
.settings-nav-item.active .nav-icon { border-color: #2b465d; background: #10263a; color: #57afea; }
.settings-context { padding: 14px; border-radius: 11px; display: flex; flex-direction: column; gap: 7px; }
.settings-context > span { color: #666f7a; font-size: .67rem; font-weight: 650; letter-spacing: .04em; }
.settings-context > strong { color: #dde1e6; font-size: .8rem; }
.settings-context > div { color: #81908e; font-size: .69rem; display: flex; align-items: center; gap: 6px; }
.settings-context > div i { width: 6px; height: 6px; border-radius: 50%; background: #42ce99; }
.settings-context code { margin-top: 2px; color: #677381; font: .64rem/1.45 ui-monospace, SFMono-Regular, Menlo, monospace; word-break: break-all; }
.settings-content { min-width: 0; }
.settings-card { margin-bottom: 16px; border-radius: 11px; overflow: hidden; }
.settings-card-head { min-height: 92px; padding: 18px 20px; border-bottom: 1px solid #252930; display: flex; align-items: center; justify-content: space-between; gap: 18px; background: #0f1115; }
.settings-card-head.compact-head { min-height: 82px; }
.settings-card-head > div { min-width: 0; }
.section-kicker { display: block; margin-bottom: 5px; color: #557995; font-size: .62rem; font-weight: 760; letter-spacing: .12em; }
.settings-card-head h2 { margin: 0; color: #eef1f4; font-size: 1rem; line-height: 1.35; font-weight: 650; }
.settings-card-head p { margin: 5px 0 0; max-width: 690px; color: #777f89; font-size: .73rem; line-height: 1.55; }
.provider-test-button { min-height: 34px; padding: 0 11px; flex: 0 0 auto; border: 1px solid #2b3037; border-radius: 7px; background: #121419; color: #9ba3ad; font-size: .7rem; cursor: pointer; }
.provider-test-button:hover { border-color: #3a4652; color: #e3e7eb; }
.provider-stack { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.provider-panel { border: 1px solid #272b32; border-radius: 10px; background: #0c0e12; overflow: hidden; transition: border-color .15s ease; }
.provider-panel.enabled { border-color: #2d4253; }
.provider-head { min-height: 68px; padding: 12px 14px; border-bottom: 1px solid #22262c; display: flex; align-items: center; justify-content: space-between; gap: 16px; background: #0e1014; }
.provider-identity, .provider-state-wrap { display: flex; align-items: center; }
.provider-identity { gap: 10px; min-width: 0; }
.provider-identity > div { display: flex; flex-direction: column; gap: 3px; }
.provider-identity strong { color: #e6e9ed; font-size: .84rem; font-weight: 650; }
.provider-identity small { color: #666f7b; font-size: .66rem; }
.provider-logo { width: 38px; height: 38px; border: 1px solid #292e35; border-radius: 9px; background: #13161b; display: grid; place-items: center; }
.provider-logo svg { width: 21px; height: 21px; }
.google-logo { background: #f7f8fa; border-color: #353940; }
.github-logo svg { fill: #e9edf2; }
.provider-state-wrap { gap: 12px; }
.provider-state { height: 26px; padding: 0 9px; border: 1px solid #30343b; border-radius: 13px; color: #777f8a; font-size: .65rem; display: inline-flex; align-items: center; gap: 6px; }
.provider-state i { width: 6px; height: 6px; border-radius: 50%; background: #666e78; }
.provider-state.ready { border-color: #28453b; background: #0f1916; color: #72b99e; }
.provider-state.ready i { background: #42ce99; }
.switch-control { width: 38px; height: 22px; padding: 2px; border: 1px solid #343941; border-radius: 12px; background: #191c21; cursor: pointer; transition: .16s ease; }
.switch-control span { display: block; width: 16px; height: 16px; border-radius: 50%; background: #7b838d; transform: translateX(0); transition: .16s ease; }
.switch-control.on { border-color: #2c729f; background: #174568; }
.switch-control.on span { background: #eef7fd; transform: translateX(16px); }
.provider-fields, .general-form { padding: 15px; display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: 13px; }
.settings-field { min-width: 0; display: flex; flex-direction: column; gap: 7px; }
.settings-field > span { color: #9ba2ab; font-size: .7rem; font-weight: 590; }
.settings-field > span em { margin-left: 6px; padding: 2px 6px; border-radius: 9px; background: #11261f; color: #63b293; font-size: .59rem; font-style: normal; font-weight: 650; }
.settings-field input { width: 100%; height: 40px; padding: 0 11px; border: 1px solid #2c3037; border-radius: 7px; background: #090b0e; color: #e0e4e8; outline: none; font-size: .75rem; transition: .15s ease; }
.settings-field input::placeholder { color: #4f5761; }
.settings-field input:focus { border-color: #3a6e96; box-shadow: 0 0 0 3px rgba(47,150,232,.07); }
.settings-field > small { margin-top: -2px; color: #626b76; font-size: .64rem; line-height: 1.45; }
.field-wide { grid-column: 1 / -1; }
.secret-input, .url-input-row { display: grid; grid-template-columns: minmax(0,1fr) auto; }
.secret-input input, .url-input-row input { border-radius: 7px 0 0 7px; }
.secret-input button, .url-input-row button { min-width: 64px; padding: 0 10px; border: 1px solid #2c3037; border-left: 0; border-radius: 0 7px 7px 0; background: #12151a; color: #838c96; font-size: .66rem; cursor: pointer; white-space: nowrap; }
.secret-input button:hover, .url-input-row button:hover { background: #171b20; color: #d7dce2; }
.url-input-row button { min-width: 88px; }
.toggle-list { padding: 2px 18px; }
.toggle-row { min-height: 66px; padding: 10px 2px; border-bottom: 1px solid #22262c; display: flex; align-items: center; justify-content: space-between; gap: 20px; }
.toggle-row:last-child { border-bottom: 0; }
.toggle-row > div { display: flex; flex-direction: column; gap: 4px; }
.toggle-row strong { color: #d8dce1; font-size: .77rem; font-weight: 620; }
.toggle-row small { color: #69727d; font-size: .68rem; line-height: 1.45; }
.general-form { padding: 18px; gap: 15px; }
.domain-preview-card .settings-card-head { border-bottom-color: #22262c; }
.route-preview-list { padding: 4px 18px; }
.route-preview-list > div { min-height: 58px; border-bottom: 1px solid #22262c; display: grid; grid-template-columns: 112px minmax(0,1fr) auto; gap: 12px; align-items: center; }
.route-preview-list > div:last-child { border-bottom: 0; }
.route-preview-list span { color: #7d8690; font-size: .69rem; }
.route-preview-list code { min-width: 0; overflow: hidden; text-overflow: ellipsis; color: #a7b3bf; font: .68rem/1.4 ui-monospace, SFMono-Regular, Menlo, monospace; white-space: nowrap; }
.route-preview-list button { min-height: 30px; padding: 0 9px; border: 1px solid #2b3037; border-radius: 6px; background: #121419; color: #818a95; font-size: .64rem; cursor: pointer; }
.security-toggle-list { padding-top: 2px; }
.turnstile-form { border-bottom: 0; }
.security-note { margin: 0 18px 18px; padding: 11px 12px; border: 1px solid #4d3b22; border-radius: 8px; background: #17130c; display: flex; flex-direction: column; gap: 4px; }
.security-note strong { color: #d5ad70; font-size: .7rem; }
.security-note span { color: #8f7757; font-size: .66rem; line-height: 1.45; }
@keyframes settings-spin { to { transform: rotate(360deg); } }
@media (max-width: 980px) {
  .settings-layout { grid-template-columns: 1fr; }
  .settings-side { position: static; }
  .settings-nav { display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); gap: 6px; }
  .settings-context { display: none; }
  .settings-nav-item { grid-template-columns: 30px minmax(0,1fr); }
  .settings-nav-item > i { display: none; }
  .nav-icon { width: 30px; height: 30px; }
}
@media (max-width: 720px) {
  .settings-command, .settings-card-head, .provider-head { align-items: flex-start; flex-direction: column; }
  .command-actions { width: 100%; }
  .command-actions button { flex: 1; }
  .settings-nav { grid-template-columns: 1fr; }
  .settings-nav-item { min-height: 52px; }
  .provider-state-wrap { width: 100%; justify-content: space-between; }
  .provider-fields, .general-form { grid-template-columns: 1fr; }
  .field-wide { grid-column: auto; }
  .route-preview-list > div { padding: 10px 0; grid-template-columns: 1fr auto; }
  .route-preview-list span { grid-column: 1 / -1; }
  .provider-test-button { width: 100%; }
}
</style>
