<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiKeyCredentialCard from '../components/ApiKeyCredentialCard.vue'
import { api, getErrorMessage, previewMode } from '../core/api'
import { pushNotification } from '../core/notifications'

interface GroupItem {
  id: number
  name: string
  description?: string
  platform?: string
  rate_multiplier?: number
  subscription_type?: string
  status?: string
}

interface ApiKeyItem {
  id: number
  name?: string
  key?: string
  status?: string
  group_id?: number | null
  group?: GroupItem | null
  quota?: number
  quota_used?: number
  expires_at?: string | null
  last_used_at?: string | null
  current_concurrency?: number
  rate_limit_5h?: number
  rate_limit_1d?: number
  rate_limit_7d?: number
  ip_whitelist?: string[]
  ip_blacklist?: string[]
  created_at?: string
  [key: string]: unknown
}

type KeyListPayload = ApiKeyItem[] | { items?: ApiKeyItem[] }

const { locale } = useI18n()
const loading = ref(false)
const creating = ref(false)
const error = ref('')
const createError = ref('')
const keys = ref<ApiKeyItem[]>([])
const groups = ref<GroupItem[]>([])
const groupRates = ref<Record<string, number>>({})
const createOpen = ref(false)
const advancedOpen = ref(false)
const createdKey = ref<ApiKeyItem | null>(null)
const copiedCreatedKey = ref(false)
let copiedTimer: number | undefined

const name = ref('')
const groupId = ref('')
const expiryDays = ref('0')
const quota = ref('')
const rateLimit5h = ref('')
const rateLimit1d = ref('')
const rateLimit7d = ref('')
const customKey = ref('')
const ipWhitelist = ref('')
const ipBlacklist = ref('')

const isZh = computed(() => locale.value === 'zh-CN')
const copy = computed(() => isZh.value
  ? {
      kicker: 'ACCESS CREDENTIALS',
      title: 'API 密钥',
      description: '创建和管理用于调用 Smirel API 的访问凭证，并明确绑定计费与路由分组。',
      refresh: '刷新',
      refreshing: '刷新中…',
      create: '创建 API Key',
      activeKeys: '可用密钥',
      groups: '可用分组',
      concurrency: '当前并发',
      keys: 'API Keys',
      unbound: '个密钥未绑定分组',
      unboundHint: '未绑定分组的旧密钥可能无法进入完整路由与计费链路，建议重新创建并绑定可用分组。',
      empty: '还没有 API Key。创建第一个密钥后即可开始调用 Smirel API。',
      createTitle: '创建 API Key',
      createHint: '先完成名称、分组和有效期。额度、IP 与时间窗口限制放在高级设置中。',
      name: '名称',
      namePlaceholder: '例如 Production、Cursor、Claude Code',
      group: '使用分组',
      required: '必选',
      groupPlaceholder: '请选择一个可用分组',
      groupUnavailable: '当前账户没有可绑定分组，暂时无法创建可正常路由的 API Key。',
      rate: '计费倍率',
      platform: '平台',
      expiry: '有效期',
      never: '永不过期',
      days7: '7 天',
      days30: '30 天',
      days90: '90 天',
      days180: '180 天',
      days365: '365 天',
      advanced: '高级设置',
      advancedHint: '不设置时均使用平台默认值。0 表示不限额。',
      quota: '总额度（USD）',
      quotaHint: '0 或留空表示不限额',
      rate5h: '5 小时限额（USD）',
      rate1d: '每日限额（USD）',
      rate7d: '7 天限额（USD）',
      customKey: '自定义密钥',
      customKeyHint: '可选，至少 16 个字符，仅允许字母、数字、下划线和连字符。',
      customKeyPlaceholder: '留空则自动生成',
      whitelist: 'IP 白名单',
      blacklist: 'IP 黑名单',
      ipHint: '每行或逗号分隔，支持单个 IP 与 CIDR。留空表示不限制。',
      whitelistPlaceholder: '例如 203.0.113.10\n10.0.0.0/24',
      blacklistPlaceholder: '例如 198.51.100.8',
      cancel: '取消',
      creating: '创建中…',
      successTitle: 'API Key 已创建',
      successHint: '密钥已经绑定到所选分组，可以直接用于 Smirel API。请保存在安全位置。',
      credential: '完整密钥',
      copyKey: '复制密钥',
      copied: '已复制',
      done: '完成',
      deleteConfirm: '确定删除这个 API Key 吗？删除后无法恢复。',
      deletedTitle: 'API Key 已删除',
      deletedMessage: '该密钥已从账户中移除。',
      createdTitle: 'API Key 已创建',
      createdMessage: '新密钥已经绑定分组并可以开始使用。',
      invalidCustomKey: '自定义密钥至少需要 16 个字符，且只能包含字母、数字、下划线和连字符。',
      selectGroup: '请选择一个可用分组后再创建。',
    }
  : {
      kicker: 'ACCESS CREDENTIALS',
      title: 'API Keys',
      description: 'Create and manage credentials for the Smirel API with explicit billing and routing groups.',
      refresh: 'Refresh',
      refreshing: 'Refreshing…',
      create: 'Create API Key',
      activeKeys: 'Active keys',
      groups: 'Available groups',
      concurrency: 'Current concurrency',
      keys: 'API Keys',
      unbound: 'keys are not bound to a group',
      unboundHint: 'Legacy keys without a group may not enter the full routing and billing path. Recreate them with an available group.',
      empty: 'No API keys yet. Create your first key to start using the Smirel API.',
      createTitle: 'Create API Key',
      createHint: 'Start with a name, routing group, and lifetime. Quotas, IP rules, and window limits stay under advanced settings.',
      name: 'Name',
      namePlaceholder: 'e.g. Production, Cursor, Claude Code',
      group: 'Routing group',
      required: 'Required',
      groupPlaceholder: 'Select an available group',
      groupUnavailable: 'This account has no bindable group, so a routable API key cannot be created yet.',
      rate: 'Billing rate',
      platform: 'Platform',
      expiry: 'Lifetime',
      never: 'Never expires',
      days7: '7 days',
      days30: '30 days',
      days90: '90 days',
      days180: '180 days',
      days365: '365 days',
      advanced: 'Advanced settings',
      advancedHint: 'Unset values use platform defaults. Zero means unlimited.',
      quota: 'Total quota (USD)',
      quotaHint: 'Zero or blank means unlimited',
      rate5h: '5-hour limit (USD)',
      rate1d: 'Daily limit (USD)',
      rate7d: '7-day limit (USD)',
      customKey: 'Custom key',
      customKeyHint: 'Optional. At least 16 characters using letters, numbers, underscores, or hyphens only.',
      customKeyPlaceholder: 'Leave blank to generate automatically',
      whitelist: 'IP allowlist',
      blacklist: 'IP blocklist',
      ipHint: 'One per line or comma-separated. IP and CIDR are supported. Blank means unrestricted.',
      whitelistPlaceholder: 'e.g. 203.0.113.10\n10.0.0.0/24',
      blacklistPlaceholder: 'e.g. 198.51.100.8',
      cancel: 'Cancel',
      creating: 'Creating…',
      successTitle: 'API Key created',
      successHint: 'The key is bound to the selected group and is ready for the Smirel API. Store it somewhere safe.',
      credential: 'Full credential',
      copyKey: 'Copy key',
      copied: 'Copied',
      done: 'Done',
      deleteConfirm: 'Delete this API key? This cannot be undone.',
      deletedTitle: 'API Key deleted',
      deletedMessage: 'The credential has been removed from your account.',
      createdTitle: 'API Key created',
      createdMessage: 'The new credential is group-bound and ready to use.',
      invalidCustomKey: 'A custom key must be at least 16 characters and contain only letters, numbers, underscores, and hyphens.',
      selectGroup: 'Select an available group before creating the key.',
    })

const activeKeyCount = computed(() => keys.value.filter((item) => (item.status || 'active') === 'active').length)
const unboundKeyCount = computed(() => keys.value.filter((item) => item.group_id == null).length)
const totalConcurrency = computed(() => keys.value.reduce((sum, item) => sum + Number(item.current_concurrency || 0), 0))
const selectedGroup = computed(() => groups.value.find((item) => String(item.id) === groupId.value) || null)
const canCreate = computed(() => Boolean(name.value.trim() && groupId.value && groups.value.length && !creating.value))

function effectiveRate(group: GroupItem) {
  const userRate = Number(groupRates.value[String(group.id)])
  if (Number.isFinite(userRate) && userRate > 0) return userRate
  const baseRate = Number(group.rate_multiplier)
  return Number.isFinite(baseRate) && baseRate > 0 ? baseRate : 1
}

function parseList(value: string) {
  return Array.from(new Set(value.split(/[\n,]+/).map((item) => item.trim()).filter(Boolean)))
}

function numericValue(value: string) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

function resetForm() {
  name.value = ''
  groupId.value = ''
  expiryDays.value = '0'
  quota.value = ''
  rateLimit5h.value = ''
  rateLimit1d.value = ''
  rateLimit7d.value = ''
  customKey.value = ''
  ipWhitelist.value = ''
  ipBlacklist.value = ''
  advancedOpen.value = false
  createError.value = ''
  copiedCreatedKey.value = false
}

function openCreate() {
  resetForm()
  createdKey.value = null
  createOpen.value = true
}

function closeCreate() {
  if (creating.value) return
  createOpen.value = false
  createdKey.value = null
  resetForm()
}

function normalizeKeys(data: KeyListPayload) {
  return Array.isArray(data) ? data : (data.items || [])
}

async function loadPage() {
  loading.value = true
  error.value = ''

  if (previewMode) {
    groups.value = [
      { id: 1, name: 'OpenAI Standard', description: 'Responses / Chat Completions', platform: 'openai', rate_multiplier: 1, status: 'active' },
      { id: 2, name: 'Claude Standard', description: 'Messages / Claude Code', platform: 'anthropic', rate_multiplier: 1, status: 'active' },
    ]
    keys.value = [
      { id: 1, name: 'Production', key: 'sk-preview-production-9f2a7c8d', status: 'active', group_id: 1, group: groups.value[0], quota: 100, quota_used: 18.42, current_concurrency: 2, last_used_at: new Date().toISOString(), created_at: '2026-09-01' },
      { id: 2, name: 'Legacy', key: 'sk-preview-legacy-71cdef89', status: 'active', group_id: null, quota: 0, quota_used: 2.15, current_concurrency: 0, created_at: '2026-08-28' },
    ]
    loading.value = false
    return
  }

  try {
    const [keyResponse, groupResponse] = await Promise.all([
      api.get<KeyListPayload>('/keys', { params: { page: 1, page_size: 50 } }),
      api.get<GroupItem[]>('/groups/available'),
    ])
    keys.value = normalizeKeys(keyResponse.data)
    groups.value = Array.isArray(groupResponse.data) ? groupResponse.data : []

    try {
      const ratesResponse = await api.get<Record<string, number>>('/groups/rates')
      groupRates.value = ratesResponse.data || {}
    } catch {
      groupRates.value = {}
    }
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    loading.value = false
  }
}

async function createKey() {
  createError.value = ''
  const trimmedName = name.value.trim()
  if (!trimmedName) return
  if (!groupId.value) {
    createError.value = copy.value.selectGroup
    return
  }

  const trimmedCustomKey = customKey.value.trim()
  if (trimmedCustomKey && (trimmedCustomKey.length < 16 || !/^[A-Za-z0-9_-]+$/.test(trimmedCustomKey))) {
    createError.value = copy.value.invalidCustomKey
    return
  }

  const selected = selectedGroup.value
  if (!selected) {
    createError.value = copy.value.selectGroup
    return
  }

  const expires = Number(expiryDays.value)
  const payload: Record<string, unknown> = {
    name: trimmedName,
    group_id: Number(groupId.value),
    quota: numericValue(quota.value),
    rate_limit_5h: numericValue(rateLimit5h.value),
    rate_limit_1d: numericValue(rateLimit1d.value),
    rate_limit_7d: numericValue(rateLimit7d.value),
    ip_whitelist: parseList(ipWhitelist.value),
    ip_blacklist: parseList(ipBlacklist.value),
  }
  if (expires > 0) payload.expires_in_days = expires
  if (trimmedCustomKey) payload.custom_key = trimmedCustomKey

  creating.value = true
  try {
    let created: ApiKeyItem
    if (previewMode) {
      created = {
        id: Date.now(),
        name: trimmedName,
        key: trimmedCustomKey || `sk-preview-${Math.random().toString(36).slice(2)}${Math.random().toString(36).slice(2)}`,
        status: 'active',
        group_id: selected.id,
        group: selected,
        quota: numericValue(quota.value),
        quota_used: 0,
        current_concurrency: 0,
        created_at: new Date().toISOString(),
      }
    } else {
      created = (await api.post<ApiKeyItem>('/keys', payload)).data
    }

    if (!created.group) created.group = selected
    keys.value = [created, ...keys.value.filter((item) => item.id !== created.id)]
    createdKey.value = created
    pushNotification({ title: copy.value.createdTitle, message: copy.value.createdMessage, tone: 'success' })
  } catch (caught) {
    createError.value = getErrorMessage(caught)
  } finally {
    creating.value = false
  }
}

async function removeKey(id: number) {
  if (!window.confirm(copy.value.deleteConfirm)) return
  try {
    if (!previewMode) await api.delete(`/keys/${id}`)
    keys.value = keys.value.filter((item) => item.id !== id)
    pushNotification({ title: copy.value.deletedTitle, message: copy.value.deletedMessage, tone: 'info' })
  } catch (caught) {
    error.value = getErrorMessage(caught)
  }
}

async function copyCreatedCredential() {
  const credential = String(createdKey.value?.key || '')
  if (!credential) return
  try {
    await navigator.clipboard.writeText(credential)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = credential
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
  copiedCreatedKey.value = true
  if (copiedTimer) window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => { copiedCreatedKey.value = false }, 1600)
}

onMounted(() => void loadPage())
onBeforeUnmount(() => {
  if (copiedTimer) window.clearTimeout(copiedTimer)
})
</script>

<template>
  <section class="workspace-page api-keys-page">
    <header class="keys-page-heading">
      <div class="keys-page-heading-copy">
        <span>{{ copy.kicker }}</span>
        <h1>{{ copy.title }}</h1>
        <p>{{ copy.description }}</p>
      </div>
      <div class="keys-page-actions">
        <button class="keys-secondary-button" type="button" :disabled="loading" @click="loadPage">
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M16.7 7.2A7 7 0 1 0 17 11" /><path d="M16.7 3.8v3.8h-3.8" /></svg>
          {{ loading ? copy.refreshing : copy.refresh }}
        </button>
        <button class="keys-primary-button" type="button" @click="openCreate">
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M10 4v12M4 10h12" /></svg>
          {{ copy.create }}
        </button>
      </div>
    </header>

    <p v-if="error" class="keys-inline-error">{{ error }}</p>

    <section class="keys-overview" aria-label="API key overview">
      <article>
        <span>{{ copy.activeKeys }}</span>
        <strong>{{ activeKeyCount }}</strong>
        <small>/ {{ keys.length }}</small>
      </article>
      <article>
        <span>{{ copy.groups }}</span>
        <strong>{{ groups.length }}</strong>
        <small>{{ groups.length ? `${effectiveRate(groups[0]).toFixed(2)}×+` : '—' }}</small>
      </article>
      <article>
        <span>{{ copy.concurrency }}</span>
        <strong>{{ totalConcurrency }}</strong>
        <small>LIVE</small>
      </article>
    </section>

    <section v-if="unboundKeyCount" class="keys-warning">
      <span class="keys-warning-icon">!</span>
      <div>
        <strong>{{ unboundKeyCount }} {{ copy.unbound }}</strong>
        <p>{{ copy.unboundHint }}</p>
      </div>
    </section>

    <section class="keys-library">
      <header class="keys-library-heading">
        <div>
          <strong>{{ copy.keys }}</strong>
          <span>{{ keys.length }}</span>
        </div>
      </header>

      <div v-if="loading && !keys.length" class="keys-loading-state">
        <i v-for="index in 2" :key="index" />
      </div>
      <div v-else-if="keys.length" class="api-key-grid">
        <ApiKeyCredentialCard v-for="item in keys" :key="item.id" :item="item" @remove="removeKey" />
      </div>
      <div v-else class="keys-empty-state">
        <span class="keys-empty-mark">
          <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="8" cy="15" r="3.5" /><path d="m10.7 12.3 7.6-7.6M15.7 7.3l2 2M13.6 9.4l2 2" /></svg>
        </span>
        <strong>{{ copy.empty }}</strong>
        <button class="keys-primary-button" type="button" @click="openCreate">{{ copy.create }}</button>
      </div>
    </section>

    <div v-if="createOpen" class="keys-modal-backdrop" @mousedown.self="closeCreate">
      <section class="keys-create-modal" role="dialog" aria-modal="true" :aria-label="copy.createTitle">
        <template v-if="createdKey">
          <header class="keys-modal-header keys-success-header">
            <span class="keys-success-mark">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4 4L19 6.5" /></svg>
            </span>
            <div>
              <h2>{{ copy.successTitle }}</h2>
              <p>{{ copy.successHint }}</p>
            </div>
          </header>
          <div class="keys-success-body">
            <div class="keys-success-meta">
              <span>{{ createdKey.name }}</span>
              <b>{{ createdKey.group?.name || selectedGroup?.name }}</b>
            </div>
            <div class="keys-created-credential">
              <label>{{ copy.credential }}</label>
              <code>{{ createdKey.key || '—' }}</code>
              <button type="button" :disabled="!createdKey.key" @click="copyCreatedCredential">
                <svg v-if="!copiedCreatedKey" viewBox="0 0 20 20" aria-hidden="true"><rect x="7.5" y="7.5" width="8.5" height="8.5" rx="1.6" /><path d="M12.5 7.5V5.7A1.7 1.7 0 0 0 10.8 4H5.7A1.7 1.7 0 0 0 4 5.7v5.1a1.7 1.7 0 0 0 1.7 1.7h1.8" /></svg>
                <svg v-else viewBox="0 0 20 20" aria-hidden="true"><path d="m4.5 10 3.2 3.2 7.8-7.8" /></svg>
                {{ copiedCreatedKey ? copy.copied : copy.copyKey }}
              </button>
            </div>
          </div>
          <footer class="keys-modal-footer success-footer">
            <button class="keys-primary-button" type="button" @click="closeCreate">{{ copy.done }}</button>
          </footer>
        </template>

        <template v-else>
          <header class="keys-modal-header">
            <div>
              <span>{{ copy.kicker }}</span>
              <h2>{{ copy.createTitle }}</h2>
              <p>{{ copy.createHint }}</p>
            </div>
            <button class="keys-modal-close" type="button" :aria-label="copy.cancel" @click="closeCreate">
              <svg viewBox="0 0 20 20" aria-hidden="true"><path d="m5 5 10 10M15 5 5 15" /></svg>
            </button>
          </header>

          <div class="keys-modal-body">
            <div class="keys-core-fields">
              <label class="keys-form-field">
                <span>{{ copy.name }} <b>{{ copy.required }}</b></span>
                <input v-model="name" type="text" autocomplete="off" :placeholder="copy.namePlaceholder" />
              </label>

              <label class="keys-form-field">
                <span>{{ copy.group }} <b>{{ copy.required }}</b></span>
                <select v-model="groupId" :disabled="!groups.length">
                  <option value="" disabled>{{ copy.groupPlaceholder }}</option>
                  <option v-for="group in groups" :key="group.id" :value="String(group.id)">
                    {{ group.name }} · {{ group.platform || 'API' }} · {{ effectiveRate(group).toFixed(2) }}×
                  </option>
                </select>
              </label>

              <div v-if="selectedGroup" class="keys-selected-group">
                <div>
                  <span>{{ selectedGroup.platform || 'API' }}</span>
                  <strong>{{ selectedGroup.name }}</strong>
                  <p>{{ selectedGroup.description || 'Smirel API routing group' }}</p>
                </div>
                <dl>
                  <div><dt>{{ copy.rate }}</dt><dd>{{ effectiveRate(selectedGroup).toFixed(2) }}×</dd></div>
                  <div><dt>{{ copy.platform }}</dt><dd>{{ selectedGroup.platform || 'API' }}</dd></div>
                </dl>
              </div>
              <p v-else-if="!groups.length && !loading" class="keys-no-groups">{{ copy.groupUnavailable }}</p>

              <label class="keys-form-field">
                <span>{{ copy.expiry }}</span>
                <select v-model="expiryDays">
                  <option value="0">{{ copy.never }}</option>
                  <option value="7">{{ copy.days7 }}</option>
                  <option value="30">{{ copy.days30 }}</option>
                  <option value="90">{{ copy.days90 }}</option>
                  <option value="180">{{ copy.days180 }}</option>
                  <option value="365">{{ copy.days365 }}</option>
                </select>
              </label>
            </div>

            <button class="keys-advanced-toggle" type="button" :aria-expanded="advancedOpen" @click="advancedOpen = !advancedOpen">
              <div><strong>{{ copy.advanced }}</strong><span>{{ copy.advancedHint }}</span></div>
              <svg viewBox="0 0 20 20" aria-hidden="true" :class="{ open: advancedOpen }"><path d="m6 8 4 4 4-4" /></svg>
            </button>

            <section v-if="advancedOpen" class="keys-advanced-panel">
              <div class="keys-number-grid">
                <label class="keys-form-field">
                  <span>{{ copy.quota }}</span>
                  <input v-model="quota" type="number" min="0" step="0.01" placeholder="0" />
                  <small>{{ copy.quotaHint }}</small>
                </label>
                <label class="keys-form-field">
                  <span>{{ copy.rate5h }}</span>
                  <input v-model="rateLimit5h" type="number" min="0" step="0.01" placeholder="0" />
                </label>
                <label class="keys-form-field">
                  <span>{{ copy.rate1d }}</span>
                  <input v-model="rateLimit1d" type="number" min="0" step="0.01" placeholder="0" />
                </label>
                <label class="keys-form-field">
                  <span>{{ copy.rate7d }}</span>
                  <input v-model="rateLimit7d" type="number" min="0" step="0.01" placeholder="0" />
                </label>
              </div>

              <label class="keys-form-field">
                <span>{{ copy.customKey }}</span>
                <input v-model="customKey" type="text" autocomplete="off" :placeholder="copy.customKeyPlaceholder" />
                <small>{{ copy.customKeyHint }}</small>
              </label>

              <div class="keys-ip-grid">
                <label class="keys-form-field">
                  <span>{{ copy.whitelist }}</span>
                  <textarea v-model="ipWhitelist" rows="4" :placeholder="copy.whitelistPlaceholder" />
                </label>
                <label class="keys-form-field">
                  <span>{{ copy.blacklist }}</span>
                  <textarea v-model="ipBlacklist" rows="4" :placeholder="copy.blacklistPlaceholder" />
                </label>
              </div>
              <p class="keys-ip-hint">{{ copy.ipHint }}</p>
            </section>

            <p v-if="createError" class="keys-create-error">{{ createError }}</p>
          </div>

          <footer class="keys-modal-footer">
            <button class="keys-secondary-button" type="button" :disabled="creating" @click="closeCreate">{{ copy.cancel }}</button>
            <button class="keys-primary-button" type="button" :disabled="!canCreate" @click="createKey">
              {{ creating ? copy.creating : copy.create }}
            </button>
          </footer>
        </template>
      </section>
    </div>
  </section>
</template>

<style scoped>
.api-keys-page {
  width: 100%;
}

.keys-page-heading {
  margin-bottom: 22px;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 28px;
}

.keys-page-heading-copy > span,
.keys-modal-header > div > span {
  color: #596472;
  font-size: .64rem;
  font-weight: 760;
  letter-spacing: .14em;
}

.keys-page-heading h1 {
  margin: 7px 0 6px;
  color: #f2f4f6;
  font-size: clamp(1.65rem, 2vw, 2rem);
  font-weight: 690;
  letter-spacing: -.035em;
}

.keys-page-heading p {
  max-width: 720px;
  margin: 0;
  color: #7a838f;
  font-size: .82rem;
  line-height: 1.65;
}

.keys-page-actions,
.keys-modal-footer {
  display: flex;
  align-items: center;
  gap: 9px;
}

.keys-primary-button,
.keys-secondary-button {
  min-height: 40px;
  padding: 0 14px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  cursor: pointer;
  font: inherit;
  font-size: .76rem;
  font-weight: 650;
  transition: border-color .15s ease, background-color .15s ease, color .15s ease, transform .12s ease;
}

.keys-primary-button {
  border: 1px solid #2f96e8;
  color: #f8fbfe;
  background: #2f96e8;
}

.keys-primary-button:hover:not(:disabled) {
  border-color: #51a8ed;
  background: #3ca0eb;
  transform: translateY(-1px);
}

.keys-secondary-button {
  border: 1px solid #292f37;
  color: #a3abb5;
  background: #111419;
}

.keys-secondary-button:hover:not(:disabled) {
  border-color: #39414b;
  color: #e1e6eb;
  background: #151920;
}

.keys-primary-button:disabled,
.keys-secondary-button:disabled {
  opacity: .5;
  cursor: not-allowed;
  transform: none;
}

.keys-primary-button svg,
.keys-secondary-button svg {
  width: 15px;
  height: 15px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.keys-inline-error,
.keys-create-error {
  margin: 0 0 16px;
  padding: 11px 13px;
  border: 1px solid #4b2a31;
  border-radius: 8px;
  color: #e08b93;
  background: #170f12;
  font-size: .76rem;
}

.keys-overview {
  margin-bottom: 18px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border: 1px solid #242a31;
  border-radius: 10px;
  overflow: hidden;
  background: #0e1014;
}

.keys-overview article {
  min-height: 78px;
  padding: 16px 18px;
  position: relative;
  display: grid;
  grid-template-columns: 1fr auto;
  grid-template-rows: auto auto;
  align-items: end;
}

.keys-overview article + article {
  border-left: 1px solid #22272e;
}

.keys-overview span {
  grid-column: 1 / -1;
  color: #69737f;
  font-size: .69rem;
  font-weight: 620;
}

.keys-overview strong {
  color: #edf1f4;
  font-size: 1.35rem;
  font-weight: 670;
  letter-spacing: -.03em;
}

.keys-overview small {
  color: #56616d;
  font-size: .63rem;
  font-weight: 700;
  letter-spacing: .08em;
}

.keys-warning {
  margin-bottom: 18px;
  padding: 13px 15px;
  border: 1px solid #443923;
  border-radius: 9px;
  display: flex;
  align-items: flex-start;
  gap: 11px;
  background: #15130e;
}

.keys-warning-icon {
  width: 22px;
  height: 22px;
  flex: 0 0 22px;
  border: 1px solid #66542d;
  border-radius: 50%;
  display: grid;
  place-items: center;
  color: #d4af63;
  font-size: .72rem;
  font-weight: 760;
}

.keys-warning strong {
  color: #d8c18d;
  font-size: .76rem;
}

.keys-warning p {
  margin: 3px 0 0;
  color: #8f846b;
  font-size: .7rem;
  line-height: 1.55;
}

.keys-library-heading {
  min-height: 46px;
  padding: 0 3px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.keys-library-heading > div {
  display: flex;
  align-items: center;
  gap: 9px;
}

.keys-library-heading strong {
  color: #dce1e6;
  font-size: .82rem;
  font-weight: 650;
}

.keys-library-heading span {
  min-width: 26px;
  height: 21px;
  padding: 0 7px;
  border: 1px solid #292f36;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #747e89;
  background: #101318;
  font-size: .65rem;
}

.api-key-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.keys-loading-state {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.keys-loading-state i {
  min-height: 190px;
  border: 1px solid #22272e;
  border-radius: 11px;
  background: linear-gradient(105deg, #0e1014 25%, #141820 43%, #0e1014 62%);
  background-size: 260% 100%;
  animation: keys-shimmer 1.2s linear infinite;
}

.keys-empty-state {
  min-height: 230px;
  padding: 30px;
  border: 1px dashed #2a3037;
  border-radius: 11px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 13px;
  color: #717b86;
  background: #0d0f13;
  text-align: center;
}

.keys-empty-state strong {
  max-width: 470px;
  font-size: .79rem;
  font-weight: 540;
  line-height: 1.6;
}

.keys-empty-mark {
  width: 42px;
  height: 42px;
  border: 1px solid #29333d;
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: #70b5e8;
  background: #101721;
}

.keys-empty-mark svg {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.65;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.keys-modal-backdrop {
  position: fixed;
  z-index: 1200;
  inset: 0;
  padding: 28px;
  display: grid;
  place-items: center;
  background: rgba(2, 4, 7, .76);
  backdrop-filter: blur(5px);
}

.keys-create-modal {
  width: min(760px, 100%);
  max-height: min(860px, calc(100vh - 56px));
  border: 1px solid #303640;
  border-radius: 14px;
  overflow: hidden;
  color: #e8ebee;
  background: #101216;
  box-shadow: 0 24px 80px rgba(0, 0, 0, .42);
}

.keys-modal-header {
  min-height: 112px;
  padding: 22px 24px;
  border-bottom: 1px solid #272c33;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  background: #111318;
}

.keys-modal-header h2 {
  margin: 6px 0 5px;
  color: #f1f3f5;
  font-size: 1.22rem;
  font-weight: 680;
  letter-spacing: -.025em;
}

.keys-modal-header p {
  max-width: 580px;
  margin: 0;
  color: #737d88;
  font-size: .74rem;
  line-height: 1.55;
}

.keys-modal-close {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  border: 1px solid transparent;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #68727d;
  background: transparent;
  cursor: pointer;
}

.keys-modal-close:hover {
  border-color: #2c323a;
  color: #d7dce1;
  background: #171a20;
}

.keys-modal-close svg {
  width: 17px;
  height: 17px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
  stroke-linecap: round;
}

.keys-modal-body,
.keys-success-body {
  max-height: calc(100vh - 258px);
  padding: 22px 24px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #424852 transparent;
}

.keys-core-fields {
  display: grid;
  gap: 16px;
}

.keys-form-field {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.keys-form-field > span {
  color: #cbd1d7;
  font-size: .75rem;
  font-weight: 620;
}

.keys-form-field > span b {
  margin-left: 5px;
  color: #5f9bc8;
  font-size: .61rem;
  font-weight: 680;
}

.keys-form-field input,
.keys-form-field select,
.keys-form-field textarea {
  width: 100%;
  border: 1px solid #2b3038;
  border-radius: 8px;
  color: #e9edf1;
  background: #0b0d11;
  outline: none;
  font: inherit;
  font-size: .78rem;
  transition: border-color .15s ease, box-shadow .15s ease, background-color .15s ease;
}

.keys-form-field input,
.keys-form-field select {
  height: 43px;
  padding: 0 12px;
}

.keys-form-field textarea {
  min-height: 98px;
  padding: 11px 12px;
  resize: vertical;
  line-height: 1.5;
}

.keys-form-field input:focus,
.keys-form-field select:focus,
.keys-form-field textarea:focus {
  border-color: #3b6d95;
  background: #0d1015;
  box-shadow: 0 0 0 3px rgba(47, 150, 232, .08);
}

.keys-form-field input::placeholder,
.keys-form-field textarea::placeholder {
  color: #525b66;
}

.keys-form-field select:disabled {
  opacity: .48;
  cursor: not-allowed;
}

.keys-form-field small,
.keys-ip-hint {
  color: #646e79;
  font-size: .66rem;
  line-height: 1.5;
}

.keys-selected-group {
  margin-top: -5px;
  padding: 13px 14px;
  border: 1px solid #27333e;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  background: #0d1218;
}

.keys-selected-group > div {
  min-width: 0;
}

.keys-selected-group > div > span {
  color: #5f9ac5;
  font-size: .6rem;
  font-weight: 760;
  letter-spacing: .1em;
  text-transform: uppercase;
}

.keys-selected-group strong {
  margin-top: 3px;
  display: block;
  color: #dce5ed;
  font-size: .8rem;
}

.keys-selected-group p {
  margin: 3px 0 0;
  overflow: hidden;
  color: #687582;
  font-size: .68rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.keys-selected-group dl {
  margin: 0;
  display: flex;
  gap: 20px;
}

.keys-selected-group dl div {
  min-width: 72px;
}

.keys-selected-group dt {
  color: #596470;
  font-size: .59rem;
}

.keys-selected-group dd {
  margin: 3px 0 0;
  color: #bac6d1;
  font-size: .72rem;
  font-weight: 620;
  text-transform: capitalize;
}

.keys-no-groups {
  margin: -5px 0 0;
  padding: 10px 12px;
  border: 1px solid #46342c;
  border-radius: 8px;
  color: #b49382;
  background: #15110f;
  font-size: .7rem;
  line-height: 1.5;
}

.keys-advanced-toggle {
  width: 100%;
  margin-top: 20px;
  padding: 14px 0 0;
  border: 0;
  border-top: 1px solid #242930;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  color: inherit;
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.keys-advanced-toggle div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.keys-advanced-toggle strong {
  color: #cad1d8;
  font-size: .75rem;
}

.keys-advanced-toggle span {
  color: #606a75;
  font-size: .65rem;
}

.keys-advanced-toggle svg {
  width: 17px;
  height: 17px;
  fill: none;
  stroke: #717b86;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
  transition: transform .16s ease;
}

.keys-advanced-toggle svg.open {
  transform: rotate(180deg);
}

.keys-advanced-panel {
  margin-top: 15px;
  padding: 16px;
  border: 1px solid #252b32;
  border-radius: 9px;
  display: grid;
  gap: 16px;
  background: #0d0f13;
}

.keys-number-grid,
.keys-ip-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 13px;
}

.keys-ip-hint {
  margin: -7px 0 0;
}

.keys-create-error {
  margin: 16px 0 0;
}

.keys-modal-footer {
  min-height: 66px;
  padding: 12px 24px;
  border-top: 1px solid #272c33;
  justify-content: flex-end;
  background: #111318;
}

.keys-success-header {
  justify-content: flex-start;
}

.keys-success-mark {
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  border: 1px solid #285041;
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: #68d5a8;
  background: #101b17;
}

.keys-success-mark svg {
  width: 19px;
  height: 19px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.keys-success-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #77828d;
  font-size: .71rem;
}

.keys-success-meta span {
  color: #c8d0d8;
  font-weight: 650;
}

.keys-success-meta b {
  padding: 3px 7px;
  border: 1px solid #2c343d;
  border-radius: 999px;
  color: #8496a6;
  font-size: .62rem;
  font-weight: 600;
}

.keys-created-credential {
  margin-top: 16px;
  padding: 15px;
  border: 1px solid #28313a;
  border-radius: 9px;
  background: #0a0d11;
}

.keys-created-credential label {
  display: block;
  color: #626d78;
  font-size: .61rem;
  font-weight: 720;
  letter-spacing: .09em;
}

.keys-created-credential code {
  margin: 9px 0 12px;
  display: block;
  overflow-wrap: anywhere;
  color: #c5d5e3;
  font: .81rem/1.55 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.keys-created-credential button {
  min-height: 34px;
  padding: 0 10px;
  border: 1px solid #2b3540;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #9fb4c5;
  background: #111720;
  cursor: pointer;
  font: inherit;
  font-size: .69rem;
  font-weight: 620;
}

.keys-created-credential button:hover:not(:disabled) {
  border-color: #3a4856;
  color: #dbe8f2;
}

.keys-created-credential button svg {
  width: 14px;
  height: 14px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.success-footer .keys-primary-button {
  min-width: 96px;
}

@keyframes keys-shimmer {
  to { background-position: -160% 0; }
}

@media (max-width: 820px) {
  .keys-page-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .keys-page-actions {
    width: 100%;
  }

  .keys-page-actions > button {
    flex: 1 1 0;
  }

  .api-key-grid,
  .keys-loading-state {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .keys-overview {
    grid-template-columns: 1fr;
  }

  .keys-overview article + article {
    border-left: 0;
    border-top: 1px solid #22272e;
  }

  .keys-modal-backdrop {
    padding: 10px;
  }

  .keys-create-modal {
    max-height: calc(100vh - 20px);
    border-radius: 11px;
  }

  .keys-modal-header,
  .keys-modal-body,
  .keys-success-body,
  .keys-modal-footer {
    padding-left: 17px;
    padding-right: 17px;
  }

  .keys-modal-body,
  .keys-success-body {
    max-height: calc(100vh - 228px);
  }

  .keys-number-grid,
  .keys-ip-grid {
    grid-template-columns: 1fr;
  }

  .keys-selected-group {
    align-items: flex-start;
    flex-direction: column;
  }

  .keys-selected-group dl {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .keys-primary-button,
  .keys-secondary-button,
  .keys-advanced-toggle svg,
  .keys-form-field input,
  .keys-form-field select,
  .keys-form-field textarea {
    transition: none;
  }

  .keys-loading-state i {
    animation: none;
  }
}
</style>
