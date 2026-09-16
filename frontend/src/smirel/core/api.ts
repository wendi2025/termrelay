import axios, { type AxiosError, type AxiosResponse } from 'axios'

interface ApiEnvelope<T> {
  code?: number
  message?: string
  data?: T
}

export type OAuthProvider = 'google' | 'github'

const configuredBase = String(import.meta.env.VITE_API_BASE_URL || '/api/v1').trim().replace(/\/+$/, '')

export const api = axios.create({
  baseURL: configuredBase || '/api/v1',
  withCredentials: true,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token')
  if (token && token !== 'smirel-preview-token' && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  if (config.method === 'get') {
    config.params = { ...(config.params || {}), timezone: Intl.DateTimeFormat().resolvedOptions().timeZone }
  }
  return config
})

api.interceptors.response.use(
  (response: AxiosResponse) => {
    const payload = response.data as ApiEnvelope<unknown>
    if (payload && typeof payload === 'object' && typeof payload.code === 'number') {
      if (payload.code !== 0) {
        return Promise.reject(new Error(payload.message || '请求失败'))
      }
      response.data = payload.data
    }
    return response
  },
  (error: AxiosError<Record<string, unknown>>) => {
    const body = error.response?.data
    const code = body && typeof body === 'object' && typeof body.code === 'string' ? body.code : ''
    if (
      error.response?.status === 423
      && code === 'ADMIN_COMPLIANCE_ACK_REQUIRED'
      && typeof window !== 'undefined'
      && window.location.pathname !== '/admin/compliance'
    ) {
      const redirect = `${window.location.pathname}${window.location.search}${window.location.hash}`
      const query = new URLSearchParams({ redirect })
      window.location.assign(`/admin/compliance?${query.toString()}`)
    }
    return Promise.reject(error)
  },
)

export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiEnvelope<unknown> | Record<string, unknown>>
    const body = axiosError.response?.data
    if (body && typeof body === 'object' && 'message' in body && typeof body.message === 'string') return body.message
    if (!axiosError.response) return '无法连接到服务，请稍后重试'
  }
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

export function sanitizeOAuthRedirect(value: unknown, fallback = '/dashboard'): string {
  const redirect = typeof value === 'string' ? value.trim() : ''
  if (!redirect.startsWith('/') || redirect.startsWith('//')) return fallback
  return redirect
}

function resolveOAuthApiBase(): string {
  const explicitOAuthBase = String(import.meta.env.VITE_OAUTH_API_BASE_URL || '').trim().replace(/\/+$/, '')
  if (explicitOAuthBase) return explicitOAuthBase

  if (/^https?:\/\//i.test(configuredBase)) return configuredBase

  if (typeof window !== 'undefined') {
    const hostname = window.location.hostname.toLowerCase()
    if (hostname === 'relay.smirel.com' || hostname === 'www.relay.smirel.com') {
      return 'https://api.smirel.com/api/v1'
    }
    return new URL(configuredBase || '/api/v1', window.location.origin).toString().replace(/\/+$/, '')
  }

  return configuredBase || '/api/v1'
}

export function buildOAuthStartUrl(provider: OAuthProvider, redirect?: string): string {
  const base = resolveOAuthApiBase()
  const url = new URL(`${base}/auth/oauth/${provider}/start`, typeof window !== 'undefined' ? window.location.origin : 'http://localhost')
  url.searchParams.set('redirect', sanitizeOAuthRedirect(redirect))
  return url.toString()
}

export const previewMode = import.meta.env.VITE_UI_PREVIEW === 'true'
