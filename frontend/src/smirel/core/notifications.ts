import { computed, reactive } from 'vue'

export type NotificationTone = 'info' | 'success' | 'warning' | 'error'

export interface SmirelNotification {
  id: string
  title: string
  message?: string
  createdAt: string
  read: boolean
  tone: NotificationTone
}

const STORAGE_KEY = 'smirel:notifications'
const MAX_NOTIFICATIONS = 40

const state = reactive<{ items: SmirelNotification[] }>({ items: [] })

function persist() {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state.items.slice(0, MAX_NOTIFICATIONS)))
}

export function restoreNotifications() {
  if (typeof window === 'undefined') return
  try {
    const stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) || '[]')
    if (!Array.isArray(stored)) return
    state.items = stored
      .filter((item): item is SmirelNotification => Boolean(item && typeof item.id === 'string' && typeof item.title === 'string'))
      .slice(0, MAX_NOTIFICATIONS)
  } catch {
    state.items = []
  }
}

export function pushNotification(input: {
  title: string
  message?: string
  tone?: NotificationTone
}) {
  const item: SmirelNotification = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    title: input.title,
    message: input.message,
    createdAt: new Date().toISOString(),
    read: false,
    tone: input.tone || 'info',
  }
  state.items.unshift(item)
  if (state.items.length > MAX_NOTIFICATIONS) state.items.length = MAX_NOTIFICATIONS
  persist()
  return item
}

export function markNotificationRead(id: string) {
  const item = state.items.find((notification) => notification.id === id)
  if (!item || item.read) return
  item.read = true
  persist()
}

export function markAllNotificationsRead() {
  state.items.forEach((item) => { item.read = true })
  persist()
}

export function clearNotifications() {
  state.items = []
  persist()
}

export const notifications = computed(() => state.items)
export const unreadNotificationCount = computed(() => state.items.filter((item) => !item.read).length)
