import { computed, readonly, reactive } from 'vue'
import { api, previewMode } from './api'

export interface SmirelUser {
  id: number
  username: string
  email: string
  role: 'admin' | 'user'
  balance?: number
  status?: string
  avatar_url?: string | null
  created_at?: string
}

interface AuthResult {
  access_token?: string
  refresh_token?: string
  expires_in?: number
  user?: SmirelUser
  requires_2fa?: boolean
}

export interface OAuthTokenResult {
  access_token?: string
  refresh_token?: string
  expires_in?: number
  token_type?: string
}

const state = reactive({
  token: '',
  user: null as SmirelUser | null,
  ready: false,
})

export const isAuthenticated = computed(() => Boolean(state.token && state.user))
export const isAdmin = computed(() => state.user?.role === 'admin')

function previewUser(): SmirelUser {
  return {
    id: 0,
    username: 'Preview Admin',
    email: 'preview@smirel.local',
    role: 'admin',
    status: 'active',
  }
}

function persist(result: AuthResult) {
  if (!result.access_token || !result.user) throw new Error('登录响应缺少账户信息')
  state.token = result.access_token
  state.user = result.user
  localStorage.setItem('auth_token', result.access_token)
  localStorage.setItem('auth_user', JSON.stringify(result.user))
  if (result.refresh_token) localStorage.setItem('refresh_token', result.refresh_token)
  if (result.expires_in) {
    localStorage.setItem('token_expires_at', String(Date.now() + result.expires_in * 1000))
  }
}

function clear() {
  state.token = ''
  state.user = null
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('token_expires_at')
}

export async function restoreSession() {
  if (previewMode) {
    state.token = 'smirel-preview-token'
    state.user = previewUser()
    state.ready = true
    return
  }

  const token = localStorage.getItem('auth_token') || ''
  const rawUser = localStorage.getItem('auth_user')
  if (token && rawUser) {
    try {
      state.token = token
      state.user = JSON.parse(rawUser) as SmirelUser
      const response = await api.get<SmirelUser | { user: SmirelUser }>('/auth/me')
      state.user = 'user' in response.data ? response.data.user : response.data
      localStorage.setItem('auth_user', JSON.stringify(state.user))
    } catch {
      clear()
    }
  }
  state.ready = true
}

export async function login(email: string, password: string) {
  if (previewMode) return
  const { data } = await api.post<AuthResult>('/auth/login', { email, password })
  if (data.requires_2fa) {
    throw new Error('该账户启用了两步验证，请在完整账户流程中继续验证')
  }
  persist(data)
}

export async function register(email: string, password: string) {
  if (previewMode) return
  const { data } = await api.post<AuthResult>('/auth/register', { email, password })
  persist(data)
}

export async function completeOAuthTokenLogin(result: OAuthTokenResult) {
  if (previewMode) return
  if (!result.access_token) throw new Error('OAuth 登录响应缺少访问令牌')

  const { data } = await api.get<SmirelUser | { user: SmirelUser }>('/auth/me', {
    headers: { Authorization: `Bearer ${result.access_token}` },
  })
  const user = 'user' in data ? data.user : data
  persist({ ...result, user })
}

export async function logout() {
  if (!previewMode) {
    const refreshToken = localStorage.getItem('refresh_token')
    if (refreshToken) {
      try {
        await api.post('/auth/logout', { refresh_token: refreshToken })
      } catch {
        // Local logout is authoritative even if server revocation is unavailable.
      }
    }
  }

  clear()
  if (previewMode) {
    state.token = 'smirel-preview-token'
    state.user = previewUser()
  }
}

export function useSession() {
  return {
    state: readonly(state),
    isAuthenticated,
    isAdmin,
    previewMode,
    login,
    register,
    logout,
  }
}
