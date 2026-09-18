import { reactive } from 'vue'
import i18n, { supportedLocales, type SmirelLocale } from './i18n'

export type ThemePreference = 'dark' | 'light' | 'system'
export type ResolvedTheme = 'dark' | 'light'

const LOCALE_KEY = 'smirel:locale'
const THEME_KEY = 'smirel:theme'
const themeMedia = typeof window !== 'undefined' ? window.matchMedia('(prefers-color-scheme: light)') : null

export const interfacePreferences = reactive<{
  locale: SmirelLocale
  theme: ThemePreference
  resolvedTheme: ResolvedTheme
}>({
  locale: 'zh-CN',
  theme: 'dark',
  resolvedTheme: 'dark',
})

function isLocale(value: string | null): value is SmirelLocale {
  return supportedLocales.includes(value as SmirelLocale)
}

function isTheme(value: string | null): value is ThemePreference {
  return value === 'dark' || value === 'light' || value === 'system'
}

function resolveTheme(theme: ThemePreference): ResolvedTheme {
  if (theme !== 'system') return theme
  return themeMedia?.matches ? 'light' : 'dark'
}

function applyTheme(theme: ThemePreference) {
  const resolvedTheme = resolveTheme(theme)
  interfacePreferences.theme = theme
  interfacePreferences.resolvedTheme = resolvedTheme

  if (typeof document !== 'undefined') {
    document.documentElement.dataset.theme = resolvedTheme
    document.documentElement.dataset.themePreference = theme
    document.documentElement.style.colorScheme = resolvedTheme
  }
}

export function setLocale(locale: SmirelLocale) {
  interfacePreferences.locale = locale
  i18n.global.locale.value = locale

  if (typeof document !== 'undefined') document.documentElement.lang = locale
  if (typeof window !== 'undefined') window.localStorage.setItem(LOCALE_KEY, locale)
}

export function setTheme(theme: ThemePreference) {
  applyTheme(theme)
  if (typeof window !== 'undefined') window.localStorage.setItem(THEME_KEY, theme)
}

export function restoreInterfacePreferences() {
  if (typeof window === 'undefined') return

  const storedLocale = window.localStorage.getItem(LOCALE_KEY)
  const storedTheme = window.localStorage.getItem(THEME_KEY)

  setLocale(isLocale(storedLocale) ? storedLocale : 'zh-CN')
  applyTheme(isTheme(storedTheme) ? storedTheme : 'dark')

  themeMedia?.addEventListener('change', () => {
    if (interfacePreferences.theme === 'system') applyTheme('system')
  })
}
