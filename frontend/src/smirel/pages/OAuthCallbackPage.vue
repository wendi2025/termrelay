<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, getErrorMessage, sanitizeOAuthRedirect, type OAuthProvider } from '../core/api'
import { completeOAuthTokenLogin, isAdmin, type OAuthTokenResult } from '../core/session'

interface PendingOAuthPayload extends OAuthTokenResult {
  auth_result?: string
  provider?: string
  step?: string
  error?: string
  email?: string
  resolved_email?: string
  redirect?: string
  create_account_allowed?: boolean
  invitation_required?: boolean
}

type CallbackPhase = 'loading' | 'registration' | 'error'

const router = useRouter()
const logoUrl = `${import.meta.env.BASE_URL}smirel-logo.png`
const phase = ref<CallbackPhase>('loading')
const pending = ref<PendingOAuthPayload | null>(null)
const provider = ref<OAuthProvider | null>(null)
const password = ref('')
const confirmPassword = ref('')
const invitationCode = ref('')
const submitting = ref(false)
const errorMessage = ref('')

const providerName = computed(() => provider.value === 'github' ? 'GitHub' : 'Google')
const accountEmail = computed(() => pending.value?.resolved_email || pending.value?.email || '')

function storedProvider(): OAuthProvider | null {
  const value = String(sessionStorage.getItem('smirel_oauth_provider') || '').toLowerCase()
  return value === 'google' || value === 'github' ? value : null
}

function clearOAuthStorage() {
  sessionStorage.removeItem('smirel_oauth_provider')
  sessionStorage.removeItem('smirel_oauth_redirect')
}

function clearSensitiveFragment() {
  if (!window.location.hash) return
  window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}`)
}

function tokenResultFromFragment(fragment: URLSearchParams): OAuthTokenResult | null {
  const accessToken = fragment.get('access_token') || ''
  if (!accessToken) return null

  const expiresInRaw = Number(fragment.get('expires_in') || '')
  return {
    access_token: accessToken,
    refresh_token: fragment.get('refresh_token') || undefined,
    expires_in: Number.isFinite(expiresInRaw) && expiresInRaw > 0 ? expiresInRaw : undefined,
    token_type: fragment.get('token_type') || 'Bearer',
  }
}

async function finishLogin(tokens: OAuthTokenResult, requestedRedirect?: string) {
  await completeOAuthTokenLogin(tokens)
  let redirect = sanitizeOAuthRedirect(
    requestedRedirect || sessionStorage.getItem('smirel_oauth_redirect'),
    '/dashboard',
  )
  if (redirect === '/dashboard' && isAdmin.value) redirect = '/admin/dashboard'
  clearOAuthStorage()
  await router.replace(redirect)
}

async function initialize() {
  errorMessage.value = ''
  phase.value = 'loading'

  try {
    const fragment = new URLSearchParams(window.location.hash.replace(/^#/, ''))
    const fragmentTokens = tokenResultFromFragment(fragment)
    const fragmentRedirect = fragment.get('redirect') || undefined
    clearSensitiveFragment()

    if (fragmentTokens) {
      await finishLogin(fragmentTokens, fragmentRedirect)
      return
    }

    const { data } = await api.post<PendingOAuthPayload>('/auth/oauth/pending/exchange', {})
    if (data?.access_token) {
      await finishLogin(data, data.redirect)
      return
    }

    const providerValue = String(data?.provider || storedProvider() || '').toLowerCase()
    if ((providerValue === 'google' || providerValue === 'github') && data?.create_account_allowed !== false) {
      provider.value = providerValue
      pending.value = data
      phase.value = 'registration'
      return
    }

    throw new Error(data?.error || 'OAuth 登录未完成，请返回登录页重试')
  } catch (caught) {
    errorMessage.value = getErrorMessage(caught)
    phase.value = 'error'
  }
}

async function submitRegistration() {
  errorMessage.value = ''
  if (!provider.value) {
    errorMessage.value = '登录来源无效，请重新发起登录'
    return
  }
  if (password.value.length < 6) {
    errorMessage.value = '密码至少需要 6 位'
    return
  }
  if (password.value !== confirmPassword.value) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }

  submitting.value = true
  try {
    const body: Record<string, string> = { password: password.value }
    if (invitationCode.value.trim()) body.invitation_code = invitationCode.value.trim()

    const { data } = await api.post<OAuthTokenResult>(
      `/auth/oauth/${provider.value}/complete-registration`,
      body,
    )
    await finishLogin(data, pending.value?.redirect)
  } catch (caught) {
    errorMessage.value = getErrorMessage(caught)
  } finally {
    submitting.value = false
  }
}

function backToLogin() {
  clearSensitiveFragment()
  clearOAuthStorage()
  void router.replace('/login')
}

onMounted(initialize)
</script>

<template>
  <div class="oauth-page">
    <RouterLink to="/home" class="oauth-brand">
      <img :src="logoUrl" alt="Smirel" />
      <span>
        <strong>Smirel</strong>
        <small>API SERVICE</small>
      </span>
    </RouterLink>

    <main class="oauth-card">
      <div class="oauth-card-meta">
        <span>SMIREL ACCOUNT</span>
        <i><b></b>SECURE ACCESS</i>
      </div>

      <section v-if="phase === 'loading'" class="oauth-state" aria-live="polite">
        <span class="oauth-spinner" aria-hidden="true"></span>
        <h1>正在完成登录</h1>
        <p>正在验证第三方账户并建立 Smirel 会话。</p>
      </section>

      <section v-else-if="phase === 'registration'" class="oauth-registration">
        <header>
          <span class="provider-badge">{{ providerName }}</span>
          <h1>完成账户创建</h1>
          <p>第三方身份已验证。设置一个 Smirel 密码，用于后续账户管理与备用登录。</p>
        </header>

        <div v-if="accountEmail" class="verified-email">
          <span>已验证邮箱</span>
          <strong>{{ accountEmail }}</strong>
        </div>

        <form @submit.prevent="submitRegistration">
          <label>
            <span>设置密码</span>
            <input
              v-model="password"
              type="password"
              autocomplete="new-password"
              minlength="6"
              required
              placeholder="至少 6 位"
            />
          </label>

          <label>
            <span>确认密码</span>
            <input
              v-model="confirmPassword"
              type="password"
              autocomplete="new-password"
              minlength="6"
              required
              placeholder="再次输入密码"
            />
          </label>

          <label v-if="pending?.invitation_required">
            <span>邀请码</span>
            <input
              v-model="invitationCode"
              type="text"
              autocomplete="off"
              required
              placeholder="输入邀请码"
            />
          </label>

          <p v-if="errorMessage" class="oauth-error">{{ errorMessage }}</p>

          <button class="oauth-submit" type="submit" :disabled="submitting">
            {{ submitting ? '正在创建账户…' : '创建账户并继续' }}
            <b aria-hidden="true">→</b>
          </button>
        </form>
      </section>

      <section v-else class="oauth-state oauth-state-error">
        <span class="oauth-error-mark" aria-hidden="true">!</span>
        <h1>登录未完成</h1>
        <p>{{ errorMessage || '第三方登录失败，请重新尝试。' }}</p>
        <div class="oauth-error-actions">
          <button type="button" @click="initialize">重试</button>
          <button type="button" class="secondary" @click="backToLogin">返回登录</button>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.oauth-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 96px 24px 48px;
  position: relative;
  overflow: hidden;
  color: #f4f7fa;
  background:
    radial-gradient(circle at 58% 38%, rgba(47, 113, 164, .10), transparent 32%),
    #07090d;
}

.oauth-page::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(180deg, rgba(255,255,255,.014), transparent 20%);
}

.oauth-brand {
  position: absolute;
  z-index: 2;
  top: 34px;
  left: 40px;
  display: inline-flex;
  align-items: center;
  gap: 11px;
  text-decoration: none;
}

.oauth-brand img {
  width: 39px;
  height: 39px;
}

.oauth-brand > span {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.oauth-brand strong {
  color: #f4f6f8;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -.025em;
}

.oauth-brand small {
  color: #6f7a86;
  font-size: .60rem;
  font-weight: 650;
  letter-spacing: .16em;
}

.oauth-card {
  position: relative;
  z-index: 1;
  width: min(440px, 100%);
  padding: 34px;
  border: 1px solid #252c34;
  border-radius: 16px;
  background: #0d1015;
  box-shadow: 0 26px 80px rgba(0,0,0,.30), inset 0 1px rgba(255,255,255,.018);
}

.oauth-card-meta {
  min-height: 22px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  color: #65717d;
  font-size: .66rem;
  font-weight: 700;
  letter-spacing: .13em;
}

.oauth-card-meta i {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #657b70;
  font-style: normal;
  font-size: .60rem;
  letter-spacing: .08em;
}

.oauth-card-meta i b {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #53bf8e;
  box-shadow: 0 0 10px rgba(83,191,142,.22);
}

.oauth-state,
.oauth-registration {
  margin-top: 28px;
}

.oauth-state {
  min-height: 240px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.oauth-state h1,
.oauth-registration h1 {
  margin: 17px 0 0;
  color: #f5f7f9;
  font-size: 1.8rem;
  font-weight: 670;
  line-height: 1.15;
  letter-spacing: -.045em;
}

.oauth-state p,
.oauth-registration header p {
  margin: 10px 0 0;
  color: #89939d;
  font-size: .82rem;
  line-height: 1.65;
}

.oauth-spinner {
  width: 28px;
  height: 28px;
  border: 2px solid #26313b;
  border-top-color: #9fc9e8;
  border-radius: 50%;
  animation: oauth-spin .8s linear infinite;
}

@keyframes oauth-spin {
  to { transform: rotate(360deg); }
}

.provider-badge {
  display: inline-flex;
  align-items: center;
  min-height: 25px;
  padding: 0 9px;
  border: 1px solid #29333d;
  border-radius: 999px;
  color: #9da8b2;
  background: #090c10;
  font-size: .66rem;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}

.verified-email {
  margin-top: 22px;
  padding: 12px 13px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  border: 1px solid #202a33;
  border-radius: 9px;
  background: #090c10;
}

.verified-email span {
  color: #65717c;
  font-size: .66rem;
}

.verified-email strong {
  color: #cbd3da;
  font-size: .80rem;
  font-weight: 600;
  word-break: break-all;
}

.oauth-registration form {
  margin-top: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.oauth-registration label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: #a6afb8;
  font-size: .77rem;
  font-weight: 620;
}

.oauth-registration input {
  width: 100%;
  height: 48px;
  padding: 0 14px;
  border: 1px solid #29313a;
  border-radius: 10px;
  outline: none;
  background: #090c10;
  color: #edf1f4;
  font: inherit;
  font-size: .88rem;
}

.oauth-registration input:focus {
  border-color: #3a6f9b;
  box-shadow: 0 0 0 3px rgba(60,126,178,.10);
}

.oauth-error {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid rgba(223, 101, 101, .22);
  border-radius: 8px;
  color: #e7a1a1;
  background: rgba(139, 47, 47, .10);
  font-size: .74rem;
  line-height: 1.5;
}

.oauth-submit,
.oauth-error-actions button {
  height: 48px;
  border: 1px solid #dce2e7;
  border-radius: 10px;
  background: #eef2f5;
  color: #101419;
  font: inherit;
  font-size: .82rem;
  font-weight: 700;
  cursor: pointer;
}

.oauth-submit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.oauth-submit:disabled {
  cursor: wait;
  opacity: .58;
}

.oauth-error-mark {
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(223, 101, 101, .35);
  border-radius: 50%;
  color: #e39a9a;
  font-weight: 800;
}

.oauth-error-actions {
  width: 100%;
  margin-top: 24px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.oauth-error-actions .secondary {
  border-color: #29313a;
  background: #090c10;
  color: #b7c0c8;
}

@media (max-width: 560px) {
  .oauth-page {
    place-items: start center;
    padding: 96px 14px 30px;
    overflow: auto;
  }

  .oauth-brand {
    top: 22px;
    left: 22px;
  }

  .oauth-brand img {
    width: 36px;
    height: 36px;
  }

  .oauth-card {
    padding: 27px 22px 24px;
    border-radius: 14px;
  }
}
</style>
