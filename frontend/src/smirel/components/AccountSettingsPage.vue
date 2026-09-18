<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, getErrorMessage, previewMode } from '../core/api'
import { pushNotification } from '../core/notifications'
import { logout, restoreSession, useSession } from '../core/session'

interface AccountProfile {
  id: number
  username: string
  email: string
  role: 'admin' | 'user' | string
  balance?: number
  status?: string
  avatar_url?: string | null
  created_at?: string
  last_active_at?: string | null
  balance_notify_enabled?: boolean
  balance_notify_threshold?: number | null
}

const router = useRouter()
const { state } = useSession()

const profile = ref<AccountProfile | null>(null)
const loading = ref(true)
const savingProfile = ref(false)
const savingPassword = ref(false)
const savingNotifications = ref(false)
const signingOut = ref(false)
const showDeleteDialog = ref(false)
const deletingAccount = ref(false)
const deletionConfirmation = ref('')
const deleteError = ref('')
const profileError = ref('')
const passwordError = ref('')
const notificationError = ref('')

const username = ref('')
const avatarDraft = ref('')
const avatarInput = ref<HTMLInputElement | null>(null)
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const balanceNotifyEnabled = ref(false)
const balanceNotifyThreshold = ref(5)

const displayProfile = computed<AccountProfile>(() => profile.value || {
  id: state.user?.id || 0,
  username: state.user?.username || '',
  email: state.user?.email || '',
  role: state.user?.role || 'user',
  balance: state.user?.balance || 0,
  status: state.user?.status || 'active',
  avatar_url: state.user?.avatar_url || '',
  created_at: state.user?.created_at || '',
})

const displayName = computed(() => displayProfile.value.username || displayProfile.value.email?.split('@')[0] || 'Smirel Account')
const initials = computed(() => displayName.value.slice(0, 1).toUpperCase())
const avatarSrc = computed(() => avatarDraft.value || displayProfile.value.avatar_url || '')
const roleLabel = computed(() => displayProfile.value.role === 'admin' ? '管理员' : '用户')
const statusLabel = computed(() => displayProfile.value.status === 'active' ? '正常' : (displayProfile.value.status || '未知'))
const joinedAt = computed(() => formatDate(displayProfile.value.created_at))
const lastActiveAt = computed(() => formatDateTime(displayProfile.value.last_active_at))
const balance = computed(() => Number(displayProfile.value.balance || 0))
const isAdminAccount = computed(() => displayProfile.value.role === 'admin')
const canDeleteAccount = computed(() => {
  const expected = String(displayProfile.value.email || '').trim().toLowerCase()
  return Boolean(expected) && deletionConfirmation.value.trim().toLowerCase() === expected && !deletingAccount.value
})

function formatDate(value?: string | null) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

function formatDateTime(value?: string | null) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function hydrateForm(data: AccountProfile) {
  username.value = data.username || ''
  avatarDraft.value = data.avatar_url || ''
  balanceNotifyEnabled.value = Boolean(data.balance_notify_enabled)
  balanceNotifyThreshold.value = Number(data.balance_notify_threshold ?? 5)
}

async function loadProfile() {
  loading.value = true
  profileError.value = ''
  try {
    if (previewMode) {
      const preview: AccountProfile = {
        id: state.user?.id || 0,
        username: state.user?.username || 'Preview Admin',
        email: state.user?.email || 'preview@smirel.local',
        role: state.user?.role || 'admin',
        balance: state.user?.balance || 0,
        status: state.user?.status || 'active',
        avatar_url: state.user?.avatar_url || '',
        created_at: state.user?.created_at || new Date().toISOString(),
        last_active_at: new Date().toISOString(),
        balance_notify_enabled: false,
        balance_notify_threshold: 5,
      }
      profile.value = preview
      hydrateForm(preview)
      return
    }

    const data = (await api.get<AccountProfile>('/user/profile')).data
    profile.value = data
    hydrateForm(data)
  } catch (caught) {
    profileError.value = getErrorMessage(caught)
  } finally {
    loading.value = false
  }
}

function openAvatarPicker() {
  avatarInput.value?.click()
}

function readImage(file: File) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image()
    const url = URL.createObjectURL(file)
    image.onload = () => {
      URL.revokeObjectURL(url)
      resolve(image)
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片读取失败'))
    }
    image.src = url
  })
}

async function makeAvatarDataUrl(file: File) {
  const image = await readImage(file)
  const sourceSize = Math.min(image.naturalWidth, image.naturalHeight)
  const sx = Math.max(0, (image.naturalWidth - sourceSize) / 2)
  const sy = Math.max(0, (image.naturalHeight - sourceSize) / 2)
  const canvas = document.createElement('canvas')
  const size = 320
  canvas.width = size
  canvas.height = size
  const context = canvas.getContext('2d')
  if (!context) throw new Error('浏览器无法处理该图片')
  context.drawImage(image, sx, sy, sourceSize, sourceSize, 0, 0, size, size)

  for (const quality of [0.88, 0.78, 0.68, 0.58, 0.48, 0.4]) {
    const dataUrl = canvas.toDataURL('image/jpeg', quality)
    const encoded = dataUrl.split(',')[1] || ''
    const approximateBytes = Math.ceil(encoded.length * 0.75)
    if (approximateBytes <= 95 * 1024) return dataUrl
  }

  throw new Error('图片压缩后仍然过大，请换一张图片')
}

async function handleAvatarChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    profileError.value = '请选择图片文件。'
    return
  }

  profileError.value = ''
  try {
    avatarDraft.value = await makeAvatarDataUrl(file)
  } catch (caught) {
    profileError.value = caught instanceof Error ? caught.message : '头像处理失败'
  }
}

function removeAvatar() {
  avatarDraft.value = ''
}

async function saveProfile() {
  const nextUsername = username.value.trim()
  if (!nextUsername) {
    profileError.value = '用户名不能为空。'
    return
  }

  profileError.value = ''
  savingProfile.value = true
  try {
    if (previewMode) {
      profile.value = { ...displayProfile.value, username: nextUsername, avatar_url: avatarDraft.value }
      pushNotification({ title: '个人资料已保存', message: '预览模式下已更新当前页面。', tone: 'success' })
      return
    }

    const data = (await api.put<AccountProfile>('/user', {
      username: nextUsername,
      avatar_url: avatarDraft.value,
    })).data
    profile.value = data
    hydrateForm(data)
    await restoreSession()
    pushNotification({ title: '个人资料已保存', message: '用户名和头像已更新。', tone: 'success' })
  } catch (caught) {
    profileError.value = getErrorMessage(caught)
  } finally {
    savingProfile.value = false
  }
}

async function changePassword() {
  passwordError.value = ''
  if (!oldPassword.value) {
    passwordError.value = '请输入当前密码。'
    return
  }
  if (newPassword.value.length < 6) {
    passwordError.value = '新密码至少需要 6 位。'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = '两次输入的新密码不一致。'
    return
  }

  savingPassword.value = true
  try {
    if (!previewMode) {
      await api.put('/user/password', {
        old_password: oldPassword.value,
        new_password: newPassword.value,
      })
    }
    oldPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    pushNotification({ title: '密码已更新', message: '下次登录请使用新密码。', tone: 'success' })
  } catch (caught) {
    passwordError.value = getErrorMessage(caught)
  } finally {
    savingPassword.value = false
  }
}

async function saveNotifications() {
  notificationError.value = ''
  if (balanceNotifyEnabled.value && (!Number.isFinite(balanceNotifyThreshold.value) || balanceNotifyThreshold.value < 0)) {
    notificationError.value = '请输入有效的提醒余额。'
    return
  }

  savingNotifications.value = true
  try {
    if (!previewMode) {
      const data = (await api.put<AccountProfile>('/user', {
        balance_notify_enabled: balanceNotifyEnabled.value,
        balance_notify_threshold: balanceNotifyThreshold.value,
      })).data
      profile.value = data
      hydrateForm(data)
    }
    pushNotification({
      title: '提醒设置已保存',
      message: balanceNotifyEnabled.value ? `余额低于 $${Number(balanceNotifyThreshold.value).toFixed(2)} 时将发送提醒。` : '余额提醒已关闭。',
      tone: 'success',
    })
  } catch (caught) {
    notificationError.value = getErrorMessage(caught)
  } finally {
    savingNotifications.value = false
  }
}

async function signOut() {
  if (signingOut.value) return
  signingOut.value = true
  try {
    await logout()
    if (!previewMode) await router.replace('/login')
  } finally {
    signingOut.value = false
  }
}

function openDeleteAccountDialog() {
  if (isAdminAccount.value) {
    pushNotification({
      title: '管理员账户不能直接注销',
      message: '请先转移管理员权限，再使用账户注销功能。',
      tone: 'info',
    })
    return
  }
  deletionConfirmation.value = ''
  deleteError.value = ''
  showDeleteDialog.value = true
}

function closeDeleteAccountDialog() {
  if (deletingAccount.value) return
  showDeleteDialog.value = false
  deletionConfirmation.value = ''
  deleteError.value = ''
}

async function deleteAccount() {
  if (!canDeleteAccount.value) return
  deleteError.value = ''
  deletingAccount.value = true
  try {
    if (previewMode) {
      showDeleteDialog.value = false
      pushNotification({ title: '预览模式', message: '预览模式不会真正注销账户。', tone: 'info' })
      return
    }

    await api.delete('/user/account', {
      data: { confirmation: deletionConfirmation.value.trim() },
    })
    await logout()
    await router.replace('/login')
  } catch (caught) {
    deleteError.value = getErrorMessage(caught)
  } finally {
    deletingAccount.value = false
  }
}

onMounted(() => void loadProfile())
</script>

<template>
  <div class="account-settings">
    <section class="account-hero">
      <div class="account-identity">
        <div class="account-avatar-wrap">
          <div class="account-avatar">
            <img v-if="avatarSrc" :src="avatarSrc" alt="" />
            <span v-else>{{ initials }}</span>
          </div>
          <span class="account-online-dot" aria-hidden="true"></span>
        </div>
        <div class="account-identity-copy">
          <div class="account-title-row">
            <h2>{{ displayName }}</h2>
            <span class="account-status-badge"><i></i>{{ statusLabel }}</span>
          </div>
          <p>{{ displayProfile.email || '—' }}</p>
          <div class="account-meta-row">
            <span class="account-role-badge">{{ roleLabel }}</span>
            <span class="account-user-id">用户 #{{ displayProfile.id || '—' }}</span>
          </div>
        </div>
      </div>

      <dl class="account-metrics">
        <div>
          <dt>可用余额</dt>
          <dd>${{ balance.toFixed(2) }}</dd>
        </div>
        <div>
          <dt>注册时间</dt>
          <dd>{{ joinedAt }}</dd>
        </div>
        <div>
          <dt>最近活动</dt>
          <dd>{{ lastActiveAt }}</dd>
        </div>
      </dl>
    </section>

    <div v-if="loading" class="account-loading">正在加载账户信息…</div>

    <div v-else class="account-grid">
      <section class="account-card account-profile-card">
        <header class="account-card-head">
          <div>
            <span class="account-card-kicker">PROFILE</span>
            <h3>个人资料</h3>
            <p>管理你在 Smirel 中显示的基础账户信息。</p>
          </div>
        </header>

        <div class="avatar-editor">
          <div class="avatar-editor-preview">
            <img v-if="avatarSrc" :src="avatarSrc" alt="头像预览" />
            <span v-else>{{ initials }}</span>
          </div>
          <div class="avatar-editor-copy">
            <strong>头像</strong>
            <span>自动裁剪为正方形，建议使用清晰图片。</span>
            <div class="avatar-editor-actions">
              <button class="account-button secondary" type="button" @click="openAvatarPicker">更换头像</button>
              <button v-if="avatarSrc" class="account-text-button" type="button" @click="removeAvatar">移除</button>
            </div>
          </div>
          <input ref="avatarInput" class="account-file-input" type="file" accept="image/*" @change="handleAvatarChange" />
        </div>

        <div class="account-form-grid">
          <label class="account-field">
            <span>用户名</span>
            <input v-model="username" autocomplete="nickname" maxlength="64" placeholder="输入用户名" />
          </label>
          <label class="account-field">
            <span>邮箱地址</span>
            <input :value="displayProfile.email" type="email" readonly />
            <small>当前登录邮箱暂不支持在这里修改。</small>
          </label>
        </div>

        <p v-if="profileError" class="account-error">{{ profileError }}</p>
        <footer class="account-card-footer">
          <span>修改后会同步到右上角账户菜单。</span>
          <button class="account-button primary" type="button" :disabled="savingProfile" @click="saveProfile">
            {{ savingProfile ? '保存中…' : '保存资料' }}
          </button>
        </footer>
      </section>

      <section class="account-card account-security-card">
        <header class="account-card-head">
          <div>
            <span class="account-card-kicker">SECURITY</span>
            <h3>账户安全</h3>
            <p>更新登录密码。OAuth 登录账户可继续使用原登录方式。</p>
          </div>
          <div class="security-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24"><path d="M12 3.2 5.2 6v5.1c0 4.5 2.6 7.8 6.8 9.7 4.2-1.9 6.8-5.2 6.8-9.7V6L12 3.2Z"/><path d="m9.3 12.1 1.8 1.8 3.8-4"/></svg>
          </div>
        </header>

        <div class="password-fields">
          <label class="account-field">
            <span>当前密码</span>
            <input v-model="oldPassword" type="password" autocomplete="current-password" placeholder="输入当前密码" />
          </label>
          <label class="account-field">
            <span>新密码</span>
            <input v-model="newPassword" type="password" autocomplete="new-password" placeholder="至少 6 位" />
          </label>
          <label class="account-field">
            <span>确认新密码</span>
            <input v-model="confirmPassword" type="password" autocomplete="new-password" placeholder="再次输入新密码" @keydown.enter="changePassword" />
          </label>
        </div>

        <p v-if="passwordError" class="account-error">{{ passwordError }}</p>
        <footer class="account-card-footer">
          <RouterLink class="account-link" to="/forgot-password">忘记当前密码？</RouterLink>
          <button class="account-button secondary" type="button" :disabled="savingPassword" @click="changePassword">
            {{ savingPassword ? '更新中…' : '修改密码' }}
          </button>
        </footer>
      </section>

      <section class="account-card account-notification-card">
        <header class="account-card-head compact">
          <div>
            <span class="account-card-kicker">NOTIFICATION</span>
            <h3>余额提醒</h3>
            <p>余额较低时通过账户通知渠道提醒你及时充值。</p>
          </div>
        </header>

        <div class="setting-row">
          <div>
            <strong>低余额提醒</strong>
            <span>避免余额不足导致 API 请求中断。</span>
          </div>
          <button
            class="account-switch"
            :class="{ 'is-on': balanceNotifyEnabled }"
            type="button"
            role="switch"
            :aria-checked="balanceNotifyEnabled"
            @click="balanceNotifyEnabled = !balanceNotifyEnabled"
          >
            <span></span>
          </button>
        </div>

        <label class="account-field notification-threshold" :class="{ 'is-disabled': !balanceNotifyEnabled }">
          <span>提醒阈值</span>
          <div class="money-input"><span>$</span><input v-model.number="balanceNotifyThreshold" type="number" min="0" step="0.01" :disabled="!balanceNotifyEnabled" /></div>
        </label>

        <p v-if="notificationError" class="account-error">{{ notificationError }}</p>
        <footer class="account-card-footer">
          <span>{{ balanceNotifyEnabled ? '余额低于该数值时触发提醒。' : '当前不会发送低余额提醒。' }}</span>
          <button class="account-button secondary" type="button" :disabled="savingNotifications" @click="saveNotifications">
            {{ savingNotifications ? '保存中…' : '保存设置' }}
          </button>
        </footer>
      </section>

      <section class="account-card account-session-card">
        <header class="account-card-head compact">
          <div>
            <span class="account-card-kicker">SESSION</span>
            <h3>当前会话</h3>
            <p>管理当前浏览器的登录状态。</p>
          </div>
        </header>

        <div class="session-row">
          <div class="session-device-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24"><rect x="3.5" y="5" width="17" height="12" rx="2"/><path d="M8.5 20h7M12 17v3"/></svg>
          </div>
          <div class="session-copy">
            <strong>当前浏览器</strong>
            <span>Web Console · 已认证</span>
          </div>
          <span class="session-current">当前</span>
        </div>

        <div class="signout-zone">
          <div>
            <strong>退出登录</strong>
            <span>清除本机登录状态并返回登录页面，不会删除账户数据。</span>
          </div>
          <button class="account-button danger" type="button" :disabled="signingOut" @click="signOut">
            <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M8.4 4H5.8A1.8 1.8 0 0 0 4 5.8v8.4A1.8 1.8 0 0 0 5.8 16h2.6M12.4 6.5 16 10l-3.6 3.5M8 10h8"/></svg>
            {{ signingOut ? '正在退出…' : '退出登录' }}
          </button>
        </div>
      </section>

      <section class="account-card account-danger-card">
        <header class="account-card-head compact">
          <div>
            <span class="account-card-kicker danger-kicker">DANGER ZONE</span>
            <h3>注销账户</h3>
            <p>永久停止使用当前 Smirel 账户。此操作与“退出登录”不同。</p>
          </div>
        </header>

        <div class="delete-account-row">
          <div>
            <strong>注销当前账户</strong>
            <span v-if="!isAdminAccount">注销后将无法再次登录，当前 API 密钥会立即失效；必要的计费与审计历史会按平台规则保留。</span>
            <span v-else>管理员账户为避免误删不能直接注销，请先转移管理员权限。</span>
          </div>
          <button class="account-button delete-account-button" type="button" :disabled="isAdminAccount" @click="openDeleteAccountDialog">
            注销账户
          </button>
        </div>
      </section>
    </div>

    <div v-if="showDeleteDialog" class="delete-dialog-backdrop" role="presentation" @click.self="closeDeleteAccountDialog">
      <section class="delete-dialog" role="dialog" aria-modal="true" aria-labelledby="delete-account-title">
        <div class="delete-dialog-icon" aria-hidden="true">!</div>
        <h3 id="delete-account-title">确认注销账户</h3>
        <p>注销后当前账户将立即失效，所有设备会话和 API 访问都会失去权限。此操作不能在前台撤销。</p>

        <div class="delete-confirm-copy">
          <span>请输入当前账户邮箱以确认：</span>
          <strong>{{ displayProfile.email }}</strong>
        </div>

        <label class="account-field delete-confirm-field">
          <span>确认邮箱</span>
          <input
            v-model="deletionConfirmation"
            type="email"
            autocomplete="off"
            :placeholder="displayProfile.email"
            @keydown.enter="deleteAccount"
          />
        </label>

        <p v-if="deleteError" class="account-error">{{ deleteError }}</p>

        <div class="delete-dialog-actions">
          <button class="account-button secondary" type="button" :disabled="deletingAccount" @click="closeDeleteAccountDialog">取消</button>
          <button class="account-button confirm-delete" type="button" :disabled="!canDeleteAccount" @click="deleteAccount">
            {{ deletingAccount ? '正在注销…' : '确认注销账户' }}
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.account-settings {
  --account-border: #242a31;
  --account-border-soft: #1d2229;
  --account-input: #0c1015;
  --account-text: #eef2f6;
  --account-muted: #858e99;
  --account-subtle: #67717d;
  --account-danger: #ef858d;
  display: flex;
  flex-direction: column;
  gap: 18px;
  width: 100%;
}

.account-hero,
.account-card {
  border: 1px solid var(--account-border);
  background: linear-gradient(180deg, rgba(18, 22, 28, .96), rgba(14, 18, 23, .98));
  box-shadow: 0 16px 36px rgba(0, 0, 0, .13);
}

.account-hero {
  min-height: 136px;
  padding: 20px 24px;
  border-radius: 12px;
  display: grid;
  grid-template-columns: minmax(320px, .92fr) minmax(540px, 1.08fr);
  gap: clamp(28px, 4vw, 64px);
  align-items: center;
}

.account-identity {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 16px;
}

.account-avatar-wrap {
  position: relative;
  flex: 0 0 auto;
}

.account-avatar,
.avatar-editor-preview {
  overflow: hidden;
  border: 1px solid #30495f;
  background: linear-gradient(145deg, #16283a, #11202d);
  color: #79c4f5;
  display: grid;
  place-items: center;
  font-weight: 700;
}

.account-avatar {
  width: 68px;
  height: 68px;
  border-radius: 15px;
  font-size: 1.22rem;
}

.account-avatar img,
.avatar-editor-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.account-online-dot {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 14px;
  height: 14px;
  border: 3px solid #11161c;
  border-radius: 50%;
  background: #43c487;
}

.account-identity-copy {
  min-width: 0;
}

.account-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 9px;
}

.account-title-row h2 {
  margin: 0;
  color: var(--account-text);
  font-size: 1.3rem;
  line-height: 1.2;
  font-weight: 700;
  letter-spacing: -.018em;
}

.account-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 24px;
  padding: 0 9px;
  border: 1px solid rgba(67, 196, 135, .24);
  border-radius: 999px;
  background: rgba(67, 196, 135, .08);
  color: #78dca6;
  font-size: .7rem;
  font-weight: 650;
}

.account-status-badge i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #50cd90;
}

.account-identity-copy > p {
  margin: 5px 0 0;
  overflow: hidden;
  color: var(--account-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: .8rem;
  line-height: 1.35;
}

.account-meta-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 9px;
  margin-top: 10px;
}

.account-role-badge {
  min-height: 24px;
  padding: 0 8px;
  border: 1px solid var(--account-border);
  border-radius: 6px;
  background: rgba(255, 255, 255, .018);
  color: #8f99a5;
  display: inline-flex;
  align-items: center;
  font-size: .69rem;
  font-weight: 650;
  letter-spacing: .01em;
}

.account-user-id {
  color: var(--account-subtle);
  font-size: .7rem;
  font-weight: 560;
}

.account-metrics {
  width: min(100%, 760px);
  margin: 0;
  justify-self: end;
  display: grid;
  grid-template-columns: repeat(3, minmax(150px, 1fr));
}

.account-metrics > div {
  min-width: 0;
  padding: 2px 20px;
  border-left: 1px solid var(--account-border);
}

.account-metrics dt {
  color: var(--account-subtle);
  font-size: .72rem;
  font-weight: 560;
  line-height: 1.3;
}

.account-metrics dd {
  margin: 7px 0 0;
  overflow: hidden;
  color: #dce3ea;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: .92rem;
  line-height: 1.35;
  font-weight: 640;
  letter-spacing: -.01em;
}

.account-metrics > div:first-child dd {
  font-size: 1rem;
  font-weight: 680;
}

.account-loading {
  padding: 30px;
  border: 1px solid var(--account-border);
  border-radius: 12px;
  color: var(--account-muted);
  text-align: center;
  font-size: .82rem;
}

.account-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(0, .92fr);
  gap: 18px;
  align-items: start;
}

.account-card {
  min-width: 0;
  border-radius: 12px;
  padding: 22px;
}

.account-card-head {
  min-height: 56px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
}

.account-card-head.compact {
  min-height: 50px;
}

.account-card-kicker {
  display: block;
  margin-bottom: 7px;
  color: #557184;
  font-size: .62rem;
  font-weight: 760;
  letter-spacing: .14em;
}

.account-card-head h3 {
  margin: 0;
  color: var(--account-text);
  font-size: 1rem;
  font-weight: 670;
}

.account-card-head p {
  margin: 6px 0 0;
  color: var(--account-muted);
  font-size: .76rem;
  line-height: 1.55;
}

.security-icon {
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  border: 1px solid #283949;
  border-radius: 9px;
  background: rgba(70, 149, 209, .07);
  color: #6eb6e8;
  display: grid;
  place-items: center;
}

.security-icon svg,
.session-device-icon svg,
.account-button.danger svg {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.65;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.security-icon svg {
  width: 19px;
  height: 19px;
}

.avatar-editor {
  position: relative;
  margin-top: 18px;
  padding: 16px;
  border: 1px solid var(--account-border-soft);
  border-radius: 10px;
  background: rgba(255, 255, 255, .014);
  display: flex;
  align-items: center;
  gap: 14px;
}

.avatar-editor-preview {
  width: 54px;
  height: 54px;
  flex: 0 0 auto;
  border-radius: 11px;
  font-size: 1rem;
}

.avatar-editor-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.avatar-editor-copy strong,
.setting-row strong,
.session-copy strong,
.signout-zone strong {
  color: #dfe5eb;
  font-size: .79rem;
  font-weight: 640;
}

.avatar-editor-copy > span,
.setting-row span,
.session-copy span,
.signout-zone span {
  color: var(--account-subtle);
  font-size: .71rem;
  line-height: 1.45;
}

.avatar-editor-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
}

.account-file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.account-form-grid,
.password-fields {
  display: grid;
  gap: 14px;
  margin-top: 16px;
}

.account-form-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.password-fields {
  grid-template-columns: 1fr;
}

.account-field {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.account-field > span {
  color: #a8b0ba;
  font-size: .73rem;
  font-weight: 590;
}

.account-field input {
  width: 100%;
  height: 40px;
  border: 1px solid #282e36;
  border-radius: 8px;
  outline: none;
  background: var(--account-input);
  color: #e8edf2;
  padding: 0 12px;
  font: inherit;
  font-size: .78rem;
  transition: border-color .18s ease, box-shadow .18s ease, background .18s ease;
}

.account-field input:hover:not(:disabled):not([readonly]) {
  border-color: #34404c;
}

.account-field input:focus:not(:disabled):not([readonly]) {
  border-color: #3d7eab;
  box-shadow: 0 0 0 3px rgba(69, 151, 209, .09);
}

.account-field input[readonly],
.account-field input:disabled {
  color: #7b8490;
  background: #0f1318;
  cursor: default;
}

.account-field small {
  color: #626b76;
  font-size: .67rem;
}

.account-error {
  margin: 14px 0 0;
  padding: 9px 11px;
  border: 1px solid #543038;
  border-radius: 8px;
  background: rgba(98, 35, 43, .14);
  color: #ef9aa1;
  font-size: .72rem;
}

.account-card-footer {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--account-border-soft);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.account-card-footer > span {
  color: #66707a;
  font-size: .69rem;
  line-height: 1.45;
}

.account-button {
  min-height: 36px;
  padding: 0 13px;
  border-radius: 7px;
  font: inherit;
  font-size: .73rem;
  font-weight: 620;
  cursor: pointer;
  transition: border-color .16s ease, background .16s ease, color .16s ease, transform .16s ease;
}

.account-button:hover:not(:disabled) {
  transform: translateY(-1px);
}

.account-button:disabled {
  opacity: .55;
  cursor: not-allowed;
}

.account-button.primary {
  border: 1px solid #3e83b3;
  background: #1f6f9f;
  color: #f5fbff;
}

.account-button.primary:hover:not(:disabled) {
  background: #247aae;
}

.account-button.secondary {
  border: 1px solid #303740;
  background: #171c22;
  color: #cbd2da;
}

.account-button.secondary:hover:not(:disabled) {
  border-color: #3b4651;
  background: #1b2129;
}

.account-text-button,
.account-link {
  border: 0;
  background: transparent;
  color: #719fc0;
  font-size: .7rem;
  text-decoration: none;
}

.account-text-button {
  padding: 4px 2px;
  cursor: pointer;
}

.account-text-button:hover,
.account-link:hover {
  color: #89c7f1;
}

.setting-row {
  min-height: 64px;
  margin-top: 18px;
  padding: 12px 0;
  border-top: 1px solid var(--account-border-soft);
  border-bottom: 1px solid var(--account-border-soft);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.setting-row > div {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.account-switch {
  position: relative;
  width: 38px;
  height: 21px;
  flex: 0 0 auto;
  padding: 0;
  border: 1px solid #333b44;
  border-radius: 999px;
  background: #171c22;
  cursor: pointer;
  transition: .18s ease;
}

.account-switch span {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 13px;
  height: 13px;
  border-radius: 50%;
  background: #727c87;
  transition: .18s ease;
}

.account-switch.is-on {
  border-color: #337cac;
  background: #1d6694;
}

.account-switch.is-on span {
  left: 20px;
  background: #eef9ff;
}

.notification-threshold {
  margin-top: 16px;
}

.notification-threshold.is-disabled {
  opacity: .58;
}

.money-input {
  position: relative;
}

.money-input > span {
  position: absolute;
  left: 12px;
  top: 50%;
  z-index: 1;
  color: #68737f;
  transform: translateY(-50%);
  font-size: .76rem;
}

.money-input input {
  padding-left: 27px;
}

.session-row {
  min-height: 72px;
  margin-top: 18px;
  padding: 13px 14px;
  border: 1px solid var(--account-border-soft);
  border-radius: 9px;
  background: rgba(255, 255, 255, .013);
  display: grid;
  grid-template-columns: 38px 1fr auto;
  align-items: center;
  gap: 12px;
}

.session-device-icon {
  width: 36px;
  height: 36px;
  border: 1px solid #2d3945;
  border-radius: 8px;
  background: rgba(73, 132, 175, .06);
  color: #7da9c8;
  display: grid;
  place-items: center;
}

.session-device-icon svg {
  width: 18px;
  height: 18px;
}

.session-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.session-current {
  padding: 4px 8px;
  border: 1px solid rgba(72, 187, 129, .21);
  border-radius: 999px;
  background: rgba(72, 187, 129, .07);
  color: #72ce9e;
  font-size: .64rem;
  font-weight: 650;
}

.signout-zone {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--account-border-soft);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.signout-zone > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.account-button.danger {
  flex: 0 0 auto;
  min-width: 104px;
  border: 1px solid #583039;
  background: rgba(92, 37, 45, .13);
  color: var(--account-danger);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
}

.account-button.danger:hover:not(:disabled) {
  border-color: #76404a;
  background: rgba(110, 43, 54, .2);
  color: #ff9da5;
}

.account-button.danger svg {
  width: 15px;
  height: 15px;
}

.account-danger-card {
  grid-column: 1 / -1;
  border-color: #3d282e;
  background: linear-gradient(180deg, rgba(28, 18, 21, .9), rgba(18, 15, 18, .96));
}

.danger-kicker {
  color: #9d5f67;
}

.delete-account-row {
  min-height: 78px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(108, 55, 64, .32);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.delete-account-row > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.delete-account-row strong {
  color: #e9d9dc;
  font-size: .79rem;
  font-weight: 650;
}

.delete-account-row span {
  max-width: 760px;
  color: #8e7378;
  font-size: .71rem;
  line-height: 1.55;
}

.account-button.delete-account-button,
.account-button.confirm-delete {
  flex: 0 0 auto;
  border: 1px solid #7b3540;
  background: #7d2935;
  color: #fff4f5;
}

.account-button.delete-account-button:hover:not(:disabled),
.account-button.confirm-delete:hover:not(:disabled) {
  border-color: #a34654;
  background: #933341;
}

.delete-dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1300;
  padding: 20px;
  background: rgba(3, 5, 8, .72);
  backdrop-filter: blur(6px);
  display: grid;
  place-items: center;
}

.delete-dialog {
  width: min(460px, 100%);
  padding: 24px;
  border: 1px solid #4e2b32;
  border-radius: 13px;
  background: #11151a;
  box-shadow: 0 28px 80px rgba(0, 0, 0, .42);
}

.delete-dialog-icon {
  width: 38px;
  height: 38px;
  border: 1px solid #6d343e;
  border-radius: 10px;
  background: rgba(135, 45, 58, .15);
  color: #f19aa2;
  display: grid;
  place-items: center;
  font-size: 1rem;
  font-weight: 760;
}

.delete-dialog h3 {
  margin: 16px 0 0;
  color: #f1e7e9;
  font-size: 1.06rem;
}

.delete-dialog > p {
  margin: 8px 0 0;
  color: #987f84;
  font-size: .75rem;
  line-height: 1.65;
}

.delete-confirm-copy {
  margin-top: 18px;
  padding: 12px 13px;
  border: 1px solid #31282b;
  border-radius: 8px;
  background: #0d1014;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.delete-confirm-copy span {
  color: #786a6e;
  font-size: .69rem;
}

.delete-confirm-copy strong {
  color: #d9c7ca;
  font-size: .76rem;
  font-weight: 620;
  overflow-wrap: anywhere;
}

.delete-confirm-field {
  margin-top: 14px;
}

.delete-dialog-actions {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

:global(html.smirel-app[data-theme='light']) .account-settings {
  --account-border: #dce2e8;
  --account-border-soft: #e5e9ee;
  --account-input: #ffffff;
  --account-text: #17202a;
  --account-muted: #66717d;
  --account-subtle: #818b96;
}

:global(html.smirel-app[data-theme='light']) .account-hero,
:global(html.smirel-app[data-theme='light']) .account-card {
  background: #ffffff;
  box-shadow: 0 12px 28px rgba(38, 50, 62, .05);
}

:global(html.smirel-app[data-theme='light']) .account-title-row h2,
:global(html.smirel-app[data-theme='light']) .account-card-head h3 {
  color: #17202a;
}

:global(html.smirel-app[data-theme='light']) .account-metrics dd,
:global(html.smirel-app[data-theme='light']) .avatar-editor-copy strong,
:global(html.smirel-app[data-theme='light']) .setting-row strong,
:global(html.smirel-app[data-theme='light']) .session-copy strong,
:global(html.smirel-app[data-theme='light']) .signout-zone strong {
  color: #28323d;
}

:global(html.smirel-app[data-theme='light']) .account-field input {
  border-color: #d9e0e6;
  background: #fbfcfd;
  color: #27313b;
}

:global(html.smirel-app[data-theme='light']) .account-field input[readonly],
:global(html.smirel-app[data-theme='light']) .account-field input:disabled {
  background: #f4f6f8;
  color: #7b8490;
}

:global(html.smirel-app[data-theme='light']) .account-button.secondary {
  border-color: #d7dde3;
  background: #f7f9fb;
  color: #37424d;
}

:global(html.smirel-app[data-theme='light']) .account-danger-card {
  border-color: #edd4d8;
  background: #fffafa;
}

:global(html.smirel-app[data-theme='light']) .delete-account-row strong,
:global(html.smirel-app[data-theme='light']) .delete-dialog h3 {
  color: #49282e;
}

:global(html.smirel-app[data-theme='light']) .delete-dialog {
  border-color: #e4c9ce;
  background: #ffffff;
}

:global(html.smirel-app[data-theme='light']) .delete-confirm-copy {
  border-color: #eadde0;
  background: #fffafa;
}

:global(html.smirel-app[data-theme='light']) .delete-confirm-copy strong {
  color: #5b3c42;
}

@media (max-width: 1050px) {
  .account-hero {
    grid-template-columns: 1fr;
    gap: 20px;
  }

  .account-metrics {
    width: 100%;
    justify-self: stretch;
  }

  .account-metrics > div:first-child {
    border-left: 0;
    padding-left: 0;
  }

  .account-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 680px) {
  .account-hero,
  .account-card {
    padding: 18px;
  }

  .account-hero {
    min-height: auto;
  }

  .account-metrics {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .account-metrics > div,
  .account-metrics > div:first-child {
    padding: 9px 0;
    border-left: 0;
    border-top: 1px solid var(--account-border-soft);
  }

  .account-form-grid {
    grid-template-columns: 1fr;
  }

  .account-card-footer,
  .signout-zone,
  .delete-account-row {
    align-items: stretch;
    flex-direction: column;
  }

  .account-card-footer .account-button,
  .signout-zone .account-button,
  .delete-account-row .account-button {
    width: 100%;
  }

  .delete-dialog-actions {
    flex-direction: column-reverse;
  }

  .delete-dialog-actions .account-button {
    width: 100%;
  }
}

@media (max-width: 460px) {
  .account-identity {
    align-items: flex-start;
  }

  .account-avatar {
    width: 60px;
    height: 60px;
    border-radius: 13px;
  }

  .avatar-editor {
    align-items: flex-start;
  }

  .avatar-editor-actions {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }

  .session-row {
    grid-template-columns: 36px 1fr;
  }

  .session-current {
    grid-column: 2;
    justify-self: start;
  }
}
</style>
