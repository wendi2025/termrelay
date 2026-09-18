<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  api,
  buildOAuthStartUrl,
  getErrorMessage,
  previewMode,
  sanitizeOAuthRedirect,
  type OAuthProvider,
} from '../core/api'
import { useSession } from '../core/session'

const route = useRoute()
const router = useRouter()
const { login, register, isAdmin } = useSession()
const logoUrl = `${import.meta.env.BASE_URL}smirel-logo.png`
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const token = ref(String(route.query.token || ''))
const loading = ref(false)
const message = ref('')
const error = ref('')
const kind = computed(() => String(route.meta.authKind || 'login'))
const showOAuth = computed(() => kind.value === 'login' || kind.value === 'register')
const titles: Record<string, string> = {
  login: '登录 Smirel',
  register: '创建 Smirel 账户',
  forgot: '找回密码',
  reset: '设置新密码',
}
const subtitles: Record<string, string> = {
  login: '继续进入你的 API 工作区。',
  register: '一个账户管理密钥、用量和服务。',
  forgot: '输入邮箱，我们会发送重置链接。',
  reset: '为你的账户设置新的密码。',
}
const submitLabels: Record<string, string> = {
  login: '登录',
  register: '创建账户',
  forgot: '发送重置链接',
  reset: '更新密码',
}
const title = computed(() => titles[kind.value] || titles.login)
const subtitle = computed(() => subtitles[kind.value] || '')
const submitLabel = computed(() => submitLabels[kind.value] || submitLabels.login)

function startOAuth(provider: OAuthProvider) {
  error.value = ''
  message.value = ''

  if (previewMode) {
    void router.push('/admin/dashboard')
    return
  }

  const redirect = sanitizeOAuthRedirect(
    typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard',
  )
  sessionStorage.setItem('smirel_oauth_provider', provider)
  sessionStorage.setItem('smirel_oauth_redirect', redirect)
  window.location.assign(buildOAuthStartUrl(provider, redirect))
}

async function submit() {
  error.value = ''
  message.value = ''
  if (kind.value === 'register' && password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }

  loading.value = true
  try {
    if (previewMode) {
      await router.push('/admin/dashboard')
      return
    }

    if (kind.value === 'login') {
      await login(email.value.trim(), password.value)
      const redirect = typeof route.query.redirect === 'string'
        ? route.query.redirect
        : (isAdmin.value ? '/admin/dashboard' : '/dashboard')
      await router.push(redirect)
    } else if (kind.value === 'register') {
      await register(email.value.trim(), password.value)
      await router.push('/dashboard')
    } else if (kind.value === 'forgot') {
      await api.post('/auth/forgot-password', { email: email.value.trim() })
      message.value = '重置链接已发送，请检查邮箱。'
    } else {
      await api.post('/auth/reset-password', {
        email: email.value.trim(),
        token: token.value.trim(),
        new_password: password.value,
      })
      message.value = '密码已更新，现在可以登录。'
    }
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <RouterLink to="/home" class="auth-brand brand-link">
      <img :src="logoUrl" alt="Smirel" />
      <span>
        <strong>Smirel</strong>
        <small>API SERVICE</small>
      </span>
    </RouterLink>

    <main class="auth-layout">
      <section class="auth-intro" aria-label="Smirel Console">
        <span class="auth-kicker">SMIREL CONSOLE</span>
        <h2>统一管理你的<br />API 工作区。</h2>
        <p>密钥、模型、用量与账单，集中在一个清晰、稳定的控制台。</p>

        <div class="auth-capabilities">
          <div>
            <b>01</b>
            <span>
              <strong>统一入口</strong>
              <small>一个账户管理 API Keys 与服务</small>
            </span>
          </div>
          <div>
            <b>02</b>
            <span>
              <strong>清晰用量</strong>
              <small>调用、余额与订单状态集中查看</small>
            </span>
          </div>
          <div>
            <b>03</b>
            <span>
              <strong>账户管理</strong>
              <small>工作区与账户信息统一维护</small>
            </span>
          </div>
        </div>
      </section>

      <section class="auth-card">
        <div class="auth-card-meta">
          <span>SMIREL ACCOUNT</span>
          <i><b></b>SECURE ACCESS</i>
        </div>

        <header>
          <h1>{{ title }}</h1>
          <p>{{ subtitle }}</p>
        </header>

        <div v-if="showOAuth" class="oauth-login">
          <div class="oauth-actions">
            <button type="button" class="oauth-button" @click="startOAuth('google')">
              <span class="oauth-provider-mark" aria-hidden="true">
                <svg viewBox="0 0 18 18" role="presentation">
                  <path fill="#4285F4" d="M17.64 9.205c0-.639-.057-1.252-.164-1.841H9v3.481h4.844a4.14 4.14 0 0 1-1.797 2.716v2.26h2.909c1.702-1.567 2.684-3.874 2.684-6.616Z" />
                  <path fill="#34A853" d="M9 18c2.43 0 4.467-.806 5.956-2.179l-2.909-2.26c-.806.54-1.835.859-3.047.859-2.344 0-4.328-1.585-5.037-3.714H.957v2.332A9 9 0 0 0 9 18Z" />
                  <path fill="#FBBC05" d="M3.963 10.706A5.41 5.41 0 0 1 3.682 9c0-.592.102-1.168.281-1.706V4.962H.957A9 9 0 0 0 0 9c0 1.452.347 2.827.957 4.038l3.006-2.332Z" />
                  <path fill="#EA4335" d="M9 3.58c1.322 0 2.508.455 3.441 1.346l2.582-2.582C13.463.892 11.426 0 9 0A9 9 0 0 0 .957 4.962l3.006 2.332C4.672 5.165 6.656 3.58 9 3.58Z" />
                </svg>
              </span>
              <span>使用 Google 继续</span>
            </button>
            <button type="button" class="oauth-button" @click="startOAuth('github')">
              <span class="oauth-provider-mark github" aria-hidden="true">
                <svg viewBox="0 0 24 24" role="presentation">
                  <path fill="currentColor" d="M12 .5C5.648.5.5 5.648.5 12c0 5.08 3.292 9.387 7.86 10.907.575.105.785-.25.785-.555 0-.274-.01-1-.016-1.962-3.197.695-3.872-1.54-3.872-1.54-.523-1.33-1.278-1.684-1.278-1.684-1.045-.714.079-.7.079-.7 1.155.081 1.762 1.186 1.762 1.186 1.027 1.76 2.695 1.252 3.352.957.104-.744.402-1.252.732-1.54-2.552-.291-5.236-1.276-5.236-5.68 0-1.255.449-2.281 1.184-3.085-.118-.291-.513-1.462.113-3.048 0 0 .966-.309 3.165 1.178A10.98 10.98 0 0 1 12 6.096c.977.004 1.96.132 2.88.387 2.198-1.487 3.162-1.178 3.162-1.178.627 1.586.232 2.757.114 3.048.737.804 1.183 1.83 1.183 3.085 0 4.415-2.688 5.386-5.248 5.67.413.355.781 1.057.781 2.13 0 1.538-.014 2.779-.014 3.157 0 .308.207.666.79.553C20.21 21.383 23.5 17.078 23.5 12 23.5 5.648 18.352.5 12 .5Z" />
                </svg>
              </span>
              <span>使用 GitHub 继续</span>
            </button>
          </div>
          <div class="oauth-divider"><span>或使用邮箱</span></div>
        </div>

        <form :class="{ 'with-oauth': showOAuth }" @submit.prevent="submit">
          <label>
            <span>邮箱</span>
            <input
              v-model="email"
              type="email"
              autocomplete="email"
              required
              placeholder="name@example.com"
            />
          </label>

          <label v-if="kind !== 'forgot'">
            <span>密码</span>
            <input
              v-model="password"
              type="password"
              :autocomplete="kind === 'login' ? 'current-password' : 'new-password'"
              required
              placeholder="••••••••"
            />
          </label>

          <label v-if="kind === 'register'">
            <span>确认密码</span>
            <input
              v-model="confirmPassword"
              type="password"
              autocomplete="new-password"
              required
              placeholder="••••••••"
            />
          </label>

          <label v-if="kind === 'reset'">
            <span>重置令牌</span>
            <input v-model="token" type="text" required placeholder="Reset token" />
          </label>

          <p v-if="error" class="form-error">{{ error }}</p>
          <p v-if="message" class="form-success">{{ message }}</p>

          <button class="auth-submit" type="submit" :disabled="loading">
            <span>{{ loading ? '处理中…' : (previewMode ? '进入预览控制台' : submitLabel) }}</span>
            <b aria-hidden="true">→</b>
          </button>
        </form>

        <footer v-if="kind === 'login'">
          <RouterLink to="/forgot-password">忘记密码？</RouterLink>
          <span>没有账户？ <RouterLink to="/register">注册</RouterLink></span>
        </footer>
        <footer v-else>
          <RouterLink to="/login">← 返回登录</RouterLink>
        </footer>
      </section>
    </main>

    <footer class="auth-page-footer">
      <span>Smirel · API Service</span>
      <span>Console Access</span>
    </footer>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  padding: 0;
  color: #f4f7fa;
  background:
    radial-gradient(circle at 74% 38%, rgba(47, 113, 164, .10), transparent 30%),
    radial-gradient(circle at 18% 82%, rgba(42, 76, 106, .06), transparent 34%),
    #07090d;
}

.auth-page::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(90deg, transparent 0 49.96%, rgba(255,255,255,.018) 50%, transparent 50.04%),
    linear-gradient(180deg, rgba(255,255,255,.014), transparent 18%);
  opacity: .65;
}

.auth-brand {
  position: absolute;
  z-index: 3;
  top: 34px;
  left: 40px;
  gap: 11px;
}

.auth-brand img {
  width: 39px;
  height: 39px;
}

.auth-brand > span {
  gap: 2px;
}

.auth-brand strong {
  color: #f4f6f8;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -.025em;
}

.auth-brand small {
  color: #6f7a86;
  font-size: .60rem;
  font-weight: 650;
  letter-spacing: .16em;
}

.auth-layout {
  position: relative;
  z-index: 2;
  width: min(1120px, calc(100vw - 96px));
  min-height: 100vh;
  margin: 0 auto;
  padding: 118px 0 92px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 440px;
  gap: 92px;
  align-items: center;
}

.auth-intro {
  max-width: 530px;
  padding: 14px 0 18px;
}

.auth-kicker {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  color: #708090;
  font-size: .69rem;
  font-weight: 700;
  letter-spacing: .16em;
}

.auth-kicker::before {
  content: '';
  width: 22px;
  height: 1px;
  background: #4385b9;
}

.auth-intro h2 {
  margin: 22px 0 0;
  color: #f1f4f7;
  font-size: clamp(2.7rem, 4.2vw, 3.7rem);
  font-weight: 620;
  line-height: 1.05;
  letter-spacing: -.055em;
}

.auth-intro > p {
  max-width: 470px;
  margin: 20px 0 0;
  color: #8a949f;
  font-size: .95rem;
  line-height: 1.7;
}

.auth-capabilities {
  width: min(440px, 100%);
  margin-top: 42px;
  border-top: 1px solid #20262d;
}

.auth-capabilities > div {
  min-height: 74px;
  display: grid;
  grid-template-columns: 42px 1fr;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid #20262d;
}

.auth-capabilities b {
  color: #56616c;
  font: 650 .69rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.auth-capabilities span {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.auth-capabilities strong {
  color: #cbd1d7;
  font-size: .84rem;
  font-weight: 650;
}

.auth-capabilities small {
  color: #68737e;
  font-size: .73rem;
  line-height: 1.45;
}

.auth-card {
  width: 100%;
  margin: 0;
  padding: 34px 34px 28px;
  border: 1px solid #252c34;
  border-radius: 16px;
  background: #0d1015;
  box-shadow: 0 26px 80px rgba(0,0,0,.30), inset 0 1px rgba(255,255,255,.018);
}

.auth-card-meta {
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

.auth-card-meta i {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #657b70;
  font-style: normal;
  font-size: .60rem;
  letter-spacing: .08em;
}

.auth-card-meta i b {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #53bf8e;
  box-shadow: 0 0 10px rgba(83,191,142,.22);
}

.auth-card header {
  margin-top: 24px;
}

.auth-card header h1 {
  margin: 0;
  color: #f5f7f9;
  font-size: 2rem;
  font-weight: 670;
  line-height: 1.12;
  letter-spacing: -.045em;
}

.auth-card header p {
  margin: 10px 0 0;
  color: #8a949f;
  font-size: .86rem;
  line-height: 1.6;
}

.oauth-login {
  margin-top: 24px;
}

.oauth-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.oauth-button {
  min-width: 0;
  height: 46px;
  padding: 0 11px;
  border: 1px solid #29313a;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: #090c10;
  color: #dbe1e6;
  font: inherit;
  font-size: .76rem;
  font-weight: 650;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color .16s ease, background-color .16s ease, transform .16s ease;
}

.oauth-button:hover {
  border-color: #414e5a;
  background: #11161c;
  transform: translateY(-1px);
}

.oauth-provider-mark {
  width: 22px;
  height: 22px;
  flex: 0 0 22px;
  display: grid;
  place-items: center;
  border: 1px solid #303b45;
  border-radius: 50%;
  color: #eef2f5;
}

.oauth-provider-mark svg {
  width: 15px;
  height: 15px;
  display: block;
}

.oauth-provider-mark.github svg {
  width: 16px;
  height: 16px;
}

.oauth-divider {
  margin-top: 17px;
  display: flex;
  align-items: center;
  gap: 11px;
  color: #5e6974;
  font-size: .66rem;
  white-space: nowrap;
}

.oauth-divider::before,
.oauth-divider::after {
  content: '';
  height: 1px;
  flex: 1;
  background: #20262d;
}

.auth-card form {
  display: flex;
  flex-direction: column;
  gap: 17px;
  margin-top: 29px;
}

.auth-card form.with-oauth {
  margin-top: 17px;
}

.auth-card label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: #a6afb8;
  font-size: .77rem;
  font-weight: 620;
}

.auth-card label > span {
  padding-left: 1px;
}

.auth-card input {
  width: 100%;
  height: 50px;
  padding: 0 14px;
  border: 1px solid #29313a;
  border-radius: 10px;
  outline: none;
  background: #090c10;
  color: #edf1f4;
  font-size: .90rem;
  transition: border-color .16s ease, background-color .16s ease, box-shadow .16s ease;
}

.auth-card input::placeholder {
  color: #4d5660;
}

.auth-card input:hover {
  border-color: #35404b;
}

.auth-card input:focus {
  border-color: #3a6f9b;
  background: #0b0f14;
  box-shadow: 0 0 0 3px rgba(60,126,178,.10);
}

.auth-submit {
  width: 100%;
  height: 50px;
  margin-top: 3px;
  padding: 0 16px;
  border: 1px solid #dce2e7;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: #eef2f5;
  color: #101419;
  font: inherit;
  font-size: .86rem;
  font-weight: 700;
  cursor: pointer;
  transition: background-color .16s ease, border-color .16s ease, transform .16s ease;
}

.auth-submit:hover:not(:disabled) {
  border-color: #fff;
  background: #fff;
  transform: translateY(-1px);
}

.auth-submit:disabled {
  cursor: wait;
  opacity: .58;
}

.auth-submit b {
  font-size: .95rem;
  font-weight: 500;
  transition: transform .16s ease;
}

.auth-submit:hover:not(:disabled) b {
  transform: translateX(2px);
}

.form-error,
.form-success {
  margin: -2px 0 0;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: .76rem;
  line-height: 1.5;
}

.auth-card footer {
  margin-top: 21px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  color: #737d87;
  font-size: .74rem;
}

.auth-card footer a {
  color: #aab4be;
  transition: color .15s ease;
}

.auth-card footer a:hover {
  color: #f1f4f6;
}

.auth-card footer span a {
  margin-left: 3px;
  color: #7fb8e1;
}

.auth-page-footer {
  position: absolute;
  z-index: 2;
  left: 40px;
  right: 40px;
  bottom: 27px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #444e58;
  font-size: .64rem;
  letter-spacing: .04em;
}

@media (max-width: 980px) {
  .auth-layout {
    width: min(460px, calc(100vw - 48px));
    grid-template-columns: 1fr;
    gap: 0;
    padding-top: 118px;
  }

  .auth-intro {
    display: none;
  }
}

@media (max-width: 560px) {
  .auth-page {
    overflow: auto;
  }

  .auth-brand {
    top: 22px;
    left: 22px;
  }

  .auth-brand img {
    width: 36px;
    height: 36px;
  }

  .auth-layout {
    width: calc(100vw - 28px);
    min-height: 100vh;
    padding: 96px 0 78px;
    align-items: start;
  }

  .auth-card {
    padding: 27px 22px 24px;
    border-radius: 14px;
  }

  .auth-card-meta {
    font-size: .61rem;
  }

  .auth-card-meta i {
    font-size: .56rem;
  }

  .auth-card header {
    margin-top: 21px;
  }

  .auth-card header h1 {
    font-size: 1.78rem;
  }

  .oauth-actions {
    grid-template-columns: 1fr;
  }

  .auth-card form {
    margin-top: 25px;
  }

  .auth-card form.with-oauth {
    margin-top: 17px;
  }

  .auth-card footer {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .auth-page-footer {
    left: 22px;
    right: 22px;
    bottom: 20px;
    font-size: .59rem;
  }
}
</style>