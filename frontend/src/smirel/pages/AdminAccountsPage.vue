<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api, getErrorMessage, previewMode } from '../core/api'
import CreateUpstreamAccountModal from '../components/CreateUpstreamAccountModal.vue'
import EditUpstreamAccountModal from '../components/EditUpstreamAccountModal.vue'

interface UpstreamAccount {
  id: number
  name?: string
  notes?: string | null
  platform?: string
  type?: string
  status?: string
  schedulable?: boolean
  error_message?: string | null
  current_concurrency?: number
  concurrency?: number
  load_factor?: number
  priority?: number
  rate_multiplier?: number
  group_ids?: number[]
  last_used_at?: string | null
  expires_at?: number | null
  credentials_status?: string
  [key: string]: unknown
}

interface AccountListResponse {
  items?: UpstreamAccount[]
  total?: number
  page?: number
  page_size?: number
}

const loading = ref(false)
const error = ref('')
const accounts = ref<UpstreamAccount[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const search = ref('')
const platform = ref('')
const status = ref('')
const accountType = ref('')
const expandedAccountId = ref<number | null>(null)
const showCreate = ref(false)
const editingAccount = ref<UpstreamAccount | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | undefined

const previewAccounts: UpstreamAccount[] = [
  { id: 12, name: 'OpenAI Primary', platform: 'openai', type: 'oauth', status: 'active', schedulable: true, current_concurrency: 7, concurrency: 20, load_factor: 20, priority: 10, rate_multiplier: 1, group_ids: [1, 3], last_used_at: '2026-09-07T01:56:00Z', credentials_status: 'valid', notes: '主调度池 · Responses / Chat Completions' },
  { id: 18, name: 'OpenAI Reserve', platform: 'openai', type: 'oauth', status: 'active', schedulable: true, current_concurrency: 2, concurrency: 12, load_factor: 12, priority: 20, rate_multiplier: 1, group_ids: [1], last_used_at: '2026-09-07T01:48:00Z', credentials_status: 'valid', notes: '高峰期备用池' },
  { id: 21, name: 'Claude Sonnet Pool', platform: 'anthropic', type: 'setup-token', status: 'active', schedulable: true, current_concurrency: 4, concurrency: 10, load_factor: 10, priority: 15, rate_multiplier: 1, group_ids: [2], last_used_at: '2026-09-07T01:51:00Z', credentials_status: 'valid', notes: 'Claude Code / Messages' },
  { id: 25, name: 'Gemini Flash', platform: 'gemini', type: 'oauth', status: 'active', schedulable: true, current_concurrency: 1, concurrency: 8, load_factor: 8, priority: 30, rate_multiplier: 0.9, group_ids: [4], last_used_at: '2026-09-07T01:32:00Z', credentials_status: 'valid' },
  { id: 27, name: 'Grok Direct', platform: 'grok', type: 'apikey', status: 'inactive', schedulable: false, current_concurrency: 0, concurrency: 6, load_factor: 6, priority: 50, rate_multiplier: 1.05, group_ids: [], last_used_at: '2026-09-06T23:16:00Z', credentials_status: 'valid', notes: '手动暂停' },
  { id: 31, name: 'OpenAI Canary', platform: 'openai', type: 'oauth', status: 'error', schedulable: false, current_concurrency: 0, concurrency: 4, load_factor: 4, priority: 90, rate_multiplier: 1, group_ids: [3], last_used_at: '2026-09-06T20:08:00Z', credentials_status: 'expired', error_message: 'OAuth credential refresh required', notes: '仅用于小流量验证' },
]

const platformOptions = computed(() => {
  const values = new Set(accounts.value.map((item) => String(item.platform || '').trim()).filter(Boolean))
  for (const item of previewMode ? previewAccounts : []) values.add(String(item.platform || ''))
  if (platform.value) values.add(platform.value)
  return Array.from(values).sort((a, b) => platformLabel(a).localeCompare(platformLabel(b)))
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const schedulableCount = computed(() => accounts.value.filter((item) => item.status === 'active' && item.schedulable !== false).length)
const issueCount = computed(() => accounts.value.filter((item) => item.status === 'error' || item.status === 'inactive' || item.schedulable === false).length)
const currentConcurrency = computed(() => accounts.value.reduce((sum, item) => sum + Number(item.current_concurrency || 0), 0))
const maxConcurrency = computed(() => accounts.value.reduce((sum, item) => sum + Number(item.concurrency || item.load_factor || 0), 0))
const providerCount = computed(() => new Set(accounts.value.map((item) => item.platform).filter(Boolean)).size)
const hasFilters = computed(() => Boolean(search.value.trim() || platform.value || status.value || accountType.value))
const healthPercent = computed(() => accounts.value.length ? Math.round((schedulableCount.value / accounts.value.length) * 100) : 0)
const concurrencyPercent = computed(() => maxConcurrency.value > 0 ? Math.min(100, Math.round((currentConcurrency.value / maxConcurrency.value) * 100)) : 0)

function platformLabel(value?: string) {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    grok: 'Grok',
    xai: 'xAI',
    ollama: 'Ollama',
  }
  const key = String(value || '').toLowerCase()
  return labels[key] || (value ? value.charAt(0).toUpperCase() + value.slice(1) : 'Unknown')
}

function platformMark(value?: string) {
  const marks: Record<string, string> = {
    openai: 'O',
    anthropic: 'A',
    gemini: 'G',
    antigravity: 'AG',
    grok: 'X',
    xai: 'X',
    ollama: 'OL',
  }
  const key = String(value || '').toLowerCase()
  return marks[key] || platformLabel(value).slice(0, 1).toUpperCase()
}

function accountTypeLabel(value?: string) {
  const labels: Record<string, string> = {
    oauth: 'OAuth',
    'setup-token': 'Setup Token',
    apikey: 'API Key',
    upstream: 'Upstream',
    bedrock: 'Bedrock',
    service_account: 'Service Account',
  }
  return labels[String(value || '').toLowerCase()] || value || '—'
}

function healthLabel(item: UpstreamAccount) {
  if (item.status === 'error') return '异常'
  if (item.status === 'inactive') return '已停用'
  if (item.schedulable === false) return '暂停调度'
  return '可调度'
}

function healthHint(item: UpstreamAccount) {
  if (item.status === 'error') return '需要处理'
  if (item.status === 'inactive') return '手动停用'
  if (item.schedulable === false) return '调度关闭'
  return '运行正常'
}

function healthClass(item: UpstreamAccount) {
  if (item.status === 'error') return 'danger'
  if (item.status === 'inactive' || item.schedulable === false) return 'muted'
  return 'good'
}

function loadPercent(item: UpstreamAccount) {
  const max = Number(item.concurrency || item.load_factor || 0)
  if (max <= 0) return 0
  return Math.min(100, Math.round((Number(item.current_concurrency || 0) / max) * 100))
}

function formatTime(value?: string | null) {
  if (!value) return '从未使用'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function formatExpiry(value?: number | null) {
  if (!value) return '未设置'
  const date = new Date(value * 1000)
  if (Number.isNaN(date.getTime())) return '未设置'
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(date)
}

function toggleExpanded(id: number) {
  expandedAccountId.value = expandedAccountId.value === id ? null : id
}

function handleCreated() {
  showCreate.value = false
  page.value = 1
  void loadAccounts()
}

function editAccount(account: UpstreamAccount) { editingAccount.value = account }
function closeEdit() { editingAccount.value = null }
function handleUpdated() { closeEdit(); void loadAccounts() }

function resetFilters() {
  search.value = ''
  platform.value = ''
  status.value = ''
  accountType.value = ''
  page.value = 1
  void loadAccounts()
}

async function loadAccounts() {
  loading.value = true
  error.value = ''
  try {
    if (previewMode) {
      const keyword = search.value.trim().toLowerCase()
      const filtered = previewAccounts.filter((item) => {
        const matchesSearch = !keyword || `${item.name || ''} ${item.platform || ''} ${item.type || ''} ${item.id}`.toLowerCase().includes(keyword)
        const matchesPlatform = !platform.value || item.platform === platform.value
        const matchesStatus = !status.value || item.status === status.value
        const matchesType = !accountType.value || item.type === accountType.value
        return matchesSearch && matchesPlatform && matchesStatus && matchesType
      })
      total.value = filtered.length
      accounts.value = filtered.slice((page.value - 1) * pageSize, page.value * pageSize)
      return
    }

    const response = await api.get<AccountListResponse>('/admin/accounts', {
      params: {
        page: page.value,
        page_size: pageSize,
        search: search.value.trim() || undefined,
        platform: platform.value || undefined,
        status: status.value || undefined,
        type: accountType.value || undefined,
        sort_by: 'name',
        sort_order: 'asc',
      },
    })
    const data = response.data || {}
    accounts.value = Array.isArray(data.items) ? data.items : []
    total.value = Number(data.total || accounts.value.length)
  } catch (caught) {
    error.value = getErrorMessage(caught)
    accounts.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

watch([platform, status, accountType], () => {
  page.value = 1
  void loadAccounts()
})

watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    void loadAccounts()
  }, 260)
})

watch(page, () => void loadAccounts())
onMounted(() => void loadAccounts())
</script>

<template>
  <section class="admin-accounts-page">
    <header class="accounts-heading">
      <div class="accounts-heading-copy">
        <span class="accounts-kicker">账户基础设施</span>
        <div class="accounts-title-line">
          <h1>上游账户</h1>
          <span class="fleet-state" :class="{ attention: issueCount > 0 }">
            <i></i>
            {{ issueCount ? `${issueCount} 个需关注` : '运行正常' }}
          </span>
        </div>
        <p>管理接入账号、调度状态与实时负载。</p>
      </div>
      <div class="accounts-heading-actions">
        <button class="create-account-button" type="button" @click="showCreate = true">
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M10 4v12M4 10h12" /></svg>
          新增上游
        </button>
        <button class="refresh-button" type="button" :disabled="loading" @click="loadAccounts">
          <svg :class="{ spinning: loading }" viewBox="0 0 20 20" aria-hidden="true">
            <path d="M16.2 6.1A7 7 0 1 0 17 12" />
            <path d="M16.3 2.9v3.6h-3.6" />
          </svg>
          {{ loading ? '刷新中' : '刷新数据' }}
        </button>
      </div>
    </header>

    <section class="fleet-overview" aria-label="账户概览">
      <article class="fleet-primary">
        <span class="metric-label">账户总数</span>
        <div class="fleet-primary-value">
          <strong>{{ total.toLocaleString() }}</strong>
          <span>个</span>
        </div>
        <small>{{ hasFilters ? '当前筛选结果' : '已接入上游账户' }}</small>
      </article>

      <article class="fleet-metric health-metric">
        <div class="fleet-metric-head">
          <span>当前页可调度</span>
          <b>{{ healthPercent }}%</b>
        </div>
        <strong>{{ schedulableCount }}<small>/ {{ accounts.length || 0 }}</small></strong>
        <i class="fleet-progress"><b :style="{ width: `${healthPercent}%` }"></b></i>
      </article>

      <article class="fleet-metric">
        <div class="fleet-metric-head">
          <span>实时并发</span>
          <b>{{ concurrencyPercent }}%</b>
        </div>
        <strong>{{ currentConcurrency }}<small>/ {{ maxConcurrency || '—' }}</small></strong>
        <i class="fleet-progress"><b :style="{ width: `${concurrencyPercent}%` }"></b></i>
      </article>

      <article class="fleet-metric provider-metric">
        <span>上游平台</span>
        <strong>{{ providerCount }}</strong>
        <small>{{ issueCount ? `${issueCount} 个账户需关注` : '当前页状态稳定' }}</small>
      </article>
    </section>

    <section class="accounts-panel">
      <header class="accounts-toolbar">
        <div class="toolbar-title">
          <div>
            <strong>账户池</strong>
            <span>{{ total }} 个账户</span>
          </div>
          <small>按名称排序</small>
        </div>

        <div class="toolbar-controls">
          <label class="search-control">
            <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="5.5" /><path d="m13 13 4 4" /></svg>
            <input v-model="search" type="search" placeholder="搜索名称、平台或 ID" aria-label="搜索上游账户" />
          </label>

          <label class="select-control">
            <span>平台</span>
            <select v-model="platform" aria-label="筛选平台">
              <option value="">全部</option>
              <option v-for="item in platformOptions" :key="item" :value="item">{{ platformLabel(item) }}</option>
            </select>
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
          </label>

          <label class="select-control">
            <span>类型</span>
            <select v-model="accountType" aria-label="筛选账户类型">
              <option value="">全部</option>
              <option value="oauth">OAuth</option>
              <option value="setup-token">Setup Token</option>
              <option value="apikey">API Key</option>
              <option value="upstream">Upstream</option>
              <option value="bedrock">Bedrock</option>
              <option value="service_account">Service Account</option>
            </select>
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
          </label>

          <label class="select-control">
            <span>状态</span>
            <select v-model="status" aria-label="筛选状态">
              <option value="">全部</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
              <option value="error">Error</option>
            </select>
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
          </label>

          <button v-if="hasFilters" class="clear-button" type="button" @click="resetFilters">清除筛选</button>
        </div>
      </header>

      <p v-if="error" class="accounts-error">{{ error }}</p>

      <div class="upstream-table" :class="{ loading }">
        <div class="upstream-table-head">
          <span>账户</span>
          <span>状态</span>
          <span>负载</span>
          <span>调度策略</span>
          <span>分组</span>
          <span>最近使用</span>
          <span></span>
        </div>

        <template v-for="item in accounts" :key="item.id">
          <button class="upstream-row" type="button" @click="toggleExpanded(item.id)">
            <span class="upstream-identity">
              <i class="provider-mark" :data-platform="String(item.platform || '').toLowerCase()">{{ platformMark(item.platform) }}</i>
              <span>
                <strong>{{ item.name || `Account #${item.id}` }}</strong>
                <small>{{ platformLabel(item.platform) }}<b>·</b>{{ accountTypeLabel(item.type) }}<b>·</b>#{{ item.id }}</small>
              </span>
            </span>

            <span class="upstream-health">
              <span class="health-badge" :class="healthClass(item)"><i></i>{{ healthLabel(item) }}</span>
              <small>{{ healthHint(item) }}</small>
            </span>

            <span class="upstream-load">
              <span class="load-values">
                <b>{{ Number(item.current_concurrency || 0) }} / {{ Number(item.concurrency || item.load_factor || 0) || '—' }}</b>
                <small>{{ loadPercent(item) }}%</small>
              </span>
              <i class="load-track"><b :style="{ width: `${loadPercent(item)}%` }"></b></i>
            </span>

            <span class="upstream-routing">
              <b>P{{ Number(item.priority || 0) }}</b>
              <small>倍率 ×{{ Number(item.rate_multiplier ?? 1).toFixed(2) }}</small>
            </span>

            <span class="upstream-groups">
              <template v-if="item.group_ids?.length">
                <b v-for="id in item.group_ids.slice(0, 2)" :key="id">#{{ id }}</b>
                <b v-if="item.group_ids.length > 2">+{{ item.group_ids.length - 2 }}</b>
              </template>
              <small v-else>未分组</small>
            </span>

            <span class="upstream-last-used">{{ formatTime(item.last_used_at) }}</span>

            <span class="row-action" :class="{ open: expandedAccountId === item.id }" aria-hidden="true">
              <svg viewBox="0 0 16 16"><path d="m6 3 5 5-5 5" /></svg>
            </span>
          </button>

          <div v-if="expandedAccountId === item.id" class="upstream-detail">
            <div class="detail-intro">
              <span>账户说明</span>
              <strong>{{ item.notes || '暂无备注' }}</strong>
            </div>
            <div>
              <span>凭据状态</span>
              <strong>{{ item.credentials_status || '—' }}</strong>
            </div>
            <div>
              <span>到期时间</span>
              <strong>{{ formatExpiry(item.expires_at) }}</strong>
            </div>
            <div>
              <span>调度分组</span>
              <strong>{{ item.group_ids?.length ? item.group_ids.map((id) => `#${id}`).join(' · ') : '未分组' }}</strong>
            </div>
            <p v-if="item.error_message" class="account-warning">
              <span>异常原因</span>
              <strong>{{ item.error_message }}</strong>
            </p>
            <button class="edit-account-button" type="button" @click="editAccount(item)">编辑账户</button>
          </div>
        </template>

        <div v-if="!accounts.length && !loading" class="accounts-empty">
          <strong>{{ hasFilters ? '没有符合条件的账户' : '暂无上游账户' }}</strong>
          <span>{{ hasFilters ? '调整筛选条件后再试。' : '账户接入后会在这里显示运行状态和调度信息。' }}</span>
          <button v-if="hasFilters" type="button" @click="resetFilters">清除筛选</button>
        </div>

        <div v-if="loading && !accounts.length" class="accounts-loading">
          <i v-for="n in 5" :key="n"></i>
        </div>
      </div>

      <footer class="accounts-footer">
        <span>第 {{ page }} / {{ totalPages }} 页</span>
        <div>
          <button type="button" :disabled="page <= 1 || loading" @click="page -= 1">
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m10 3-5 5 5 5" /></svg>
            上一页
          </button>
          <button type="button" :disabled="page >= totalPages || loading" @click="page += 1">
            下一页
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m6 3 5 5-5 5" /></svg>
          </button>
        </div>
      </footer>
    </section>
    <CreateUpstreamAccountModal :show="showCreate" @close="showCreate = false" @created="handleCreated" />
    <EditUpstreamAccountModal :show="Boolean(editingAccount)" :account="editingAccount" @close="closeEdit" @updated="handleUpdated" />
  </section>
</template>

<style scoped>
.admin-accounts-page {
  --aa-surface: #101116;
  --aa-surface-soft: #0d0f13;
  --aa-surface-raised: #14161b;
  --aa-surface-hover: #15171d;
  --aa-border: #23262d;
  --aa-border-strong: #30343d;
  --aa-text: #f4f6f8;
  --aa-text-soft: #c8cdd4;
  --aa-muted: #858d98;
  --aa-subtle: #616a75;
  --aa-green: #43cd98;
  --aa-amber: #d7a95b;
  --aa-red: #e16c73;
  width: 100%;
  color: var(--aa-text);
  font-size: 15px;
}

.accounts-heading {
  min-height: 106px;
  padding: 4px 0 24px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 28px;
}

.accounts-heading-copy {
  min-width: 0;
}

.accounts-kicker {
  display: block;
  margin-bottom: 9px;
  color: #68717d;
  font-size: .66rem;
  line-height: 1;
  font-weight: 680;
  letter-spacing: .08em;
}

.accounts-title-line {
  display: flex;
  align-items: center;
  gap: 12px;
}

.accounts-heading h1 {
  margin: 0;
  color: #f6f7f9;
  font-size: 2.1rem;
  line-height: 1;
  font-weight: 690;
  letter-spacing: -.045em;
}

.accounts-heading p {
  margin: 11px 0 0;
  color: var(--aa-muted);
  font-size: .86rem;
  line-height: 1.5;
}

.accounts-heading-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.create-account-button {
  min-height: 40px;
  padding: 0 15px;
  border: 1px solid #d9dde2;
  border-radius: 8px;
  background: #f2f4f6;
  color: #111318;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: .76rem;
  font-weight: 670;
  cursor: pointer;
  box-shadow: 0 1px 0 rgba(255, 255, 255, .08), 0 8px 24px rgba(0, 0, 0, .15);
  transition: background .15s ease, border-color .15s ease, transform .15s ease;
}

.create-account-button:hover {
  border-color: #fff;
  background: #fff;
  transform: translateY(-1px);
}

.create-account-button svg {
  width: 14px;
  height: 14px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
}

.fleet-state {
  height: 27px;
  padding: 0 9px;
  border: 1px solid rgba(67, 205, 152, .20);
  border-radius: 999px;
  background: rgba(67, 205, 152, .06);
  color: #8bdaba;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: .68rem;
  font-weight: 620;
}

.fleet-state i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--aa-green);
}

.fleet-state.attention {
  border-color: rgba(215, 169, 91, .22);
  background: rgba(215, 169, 91, .06);
  color: #d8b578;
}

.fleet-state.attention i {
  background: var(--aa-amber);
}

.refresh-button,
.clear-button,
.accounts-footer button,
.accounts-empty button {
  border: 1px solid var(--aa-border-strong);
  background: #13151a;
  color: var(--aa-text-soft);
  cursor: pointer;
  transition: border-color .15s ease, background .15s ease, color .15s ease;
}

.edit-account-button { border: 1px solid #3b414c; border-radius: 6px; background: #181b21; color: #d8dde3; padding: 7px 11px; cursor: pointer; font-size: .72rem; }
.edit-account-button:hover { border-color: #55cd9f; color: #a8efd0; }

.refresh-button {
  min-height: 40px;
  padding: 0 14px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: .76rem;
  font-weight: 620;
}

.refresh-button svg {
  width: 15px;
  height: 15px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.55;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.refresh-button svg.spinning {
  animation: account-spin .8s linear infinite;
}

.refresh-button:hover:not(:disabled),
.clear-button:hover,
.accounts-footer button:hover:not(:disabled),
.accounts-empty button:hover {
  border-color: #454a54;
  background: #181a20;
  color: #fff;
}

.refresh-button:disabled,
.accounts-footer button:disabled {
  opacity: .42;
  cursor: default;
}

.fleet-overview {
  min-height: 126px;
  display: grid;
  grid-template-columns: 1.25fr repeat(3, minmax(0, .82fr));
  border: 1px solid var(--aa-border);
  border-radius: 12px;
  background: linear-gradient(180deg, #111217, #0f1014);
  overflow: hidden;
}

.fleet-overview article {
  position: relative;
  min-width: 0;
  padding: 21px 22px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.fleet-overview article + article::before {
  content: '';
  position: absolute;
  inset: 20px auto 20px 0;
  width: 1px;
  background: #24272e;
}

.metric-label,
.fleet-metric > span,
.fleet-metric-head > span {
  color: #7d8590;
  font-size: .70rem;
  font-weight: 620;
}

.fleet-primary-value {
  margin-top: 8px;
  display: flex;
  align-items: baseline;
  gap: 5px;
}

.fleet-primary-value strong {
  color: #f7f8fa;
  font-size: 2rem;
  line-height: .96;
  font-weight: 700;
  letter-spacing: -.04em;
}

.fleet-primary-value span {
  color: #6d7580;
  font-size: .72rem;
}

.fleet-primary > small,
.provider-metric > small {
  margin-top: 9px;
  color: #656d78;
  font-size: .66rem;
}

.fleet-metric-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.fleet-metric-head > b {
  color: #6d7580;
  font-size: .65rem;
  font-weight: 620;
}

.fleet-metric > strong {
  margin-top: 9px;
  color: #eff2f5;
  font-size: 1.35rem;
  line-height: 1;
  font-weight: 680;
  letter-spacing: -.025em;
}

.fleet-metric > strong small {
  margin-left: 4px;
  color: #747c87;
  font-size: .74rem;
  font-weight: 560;
}

.fleet-progress {
  width: 100%;
  height: 4px;
  margin-top: 13px;
  overflow: hidden;
  border-radius: 99px;
  background: #282b32;
}

.fleet-progress > b {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #7d8792;
}

.health-metric .fleet-progress > b {
  background: var(--aa-green);
}

.provider-metric > strong {
  margin-top: 9px;
  font-size: 1.55rem;
}

.accounts-panel {
  margin-top: 14px;
  border: 1px solid var(--aa-border);
  border-radius: 12px;
  background: #0f1014;
  overflow: hidden;
}

.accounts-toolbar {
  min-height: 72px;
  padding: 12px 14px 12px 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  border-bottom: 1px solid var(--aa-border);
  background: #101116;
}

.toolbar-title {
  min-width: 145px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.toolbar-title > div {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.toolbar-title strong {
  color: #edf0f3;
  font-size: .92rem;
  font-weight: 660;
}

.toolbar-title span,
.toolbar-title small {
  color: #68717c;
  font-size: .66rem;
}

.toolbar-title small {
  padding-left: 12px;
  border-left: 1px solid #2b2e35;
  white-space: nowrap;
}

.toolbar-controls {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 7px;
}

.search-control {
  width: min(330px, 27vw);
  height: 40px;
  padding: 0 12px;
  border: 1px solid #292d34;
  border-radius: 8px;
  background: #0b0d11;
  color: #666f7a;
  display: flex;
  align-items: center;
  gap: 9px;
  transition: border-color .15s ease, box-shadow .15s ease;
}

.search-control:focus-within {
  border-color: #3b4653;
  box-shadow: 0 0 0 3px rgba(120, 143, 166, .06);
}

.search-control svg {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.45;
  stroke-linecap: round;
}

.search-control input {
  width: 100%;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--aa-text);
  font-size: .76rem;
}

.search-control input::placeholder {
  color: #59616c;
}

.search-control input::-webkit-search-cancel-button {
  opacity: .55;
  filter: invert(1);
}

.select-control {
  position: relative;
  min-width: 116px;
  height: 40px;
  padding: 0 27px 0 11px;
  border: 1px solid #292d34;
  border-radius: 8px;
  background: #0b0d11;
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 7px;
}

.select-control > span {
  color: #626b76;
  font-size: .66rem;
  pointer-events: none;
}

.select-control select {
  min-width: 0;
  height: 38px;
  padding: 0;
  border: 0;
  outline: 0;
  appearance: none;
  background: transparent;
  color: #c5cad1;
  font-size: .72rem;
  cursor: pointer;
}

.select-control > svg {
  position: absolute;
  right: 9px;
  width: 13px;
  height: 13px;
  fill: none;
  stroke: #727a85;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
}

.clear-button {
  height: 40px;
  padding: 0 11px;
  border-radius: 8px;
  font-size: .70rem;
  white-space: nowrap;
}

.accounts-error {
  margin: 12px 14px 0;
  padding: 10px 12px;
  border: 1px solid rgba(225, 108, 115, .28);
  border-radius: 8px;
  background: rgba(225, 108, 115, .07);
  color: #e6a0a5;
  font-size: .74rem;
}

.upstream-table {
  min-height: 312px;
}

.upstream-table.loading {
  opacity: .74;
}

.upstream-table-head,
.upstream-row {
  display: grid;
  grid-template-columns: minmax(260px, 1.55fr) minmax(126px, .72fr) minmax(142px, .78fr) minmax(116px, .62fr) minmax(118px, .66fr) minmax(112px, .66fr) 30px;
  gap: 16px;
  align-items: center;
}

.upstream-table-head {
  min-height: 42px;
  padding: 0 18px;
  border-bottom: 1px solid #23262d;
  background: #0c0e12;
  color: #656d78;
  font-size: .64rem;
  font-weight: 650;
  letter-spacing: .02em;
}

.upstream-row {
  width: 100%;
  min-height: 68px;
  padding: 0 18px;
  border: 0;
  border-bottom: 1px solid #1e2127;
  background: transparent;
  color: var(--aa-text-soft);
  text-align: left;
  cursor: pointer;
  transition: background .14s ease, box-shadow .14s ease;
}

.upstream-row:hover {
  background: var(--aa-surface-hover);
  box-shadow: inset 2px 0 #3a414a;
}

.upstream-identity {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 11px;
}

.provider-mark {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  border: 1px solid #30343c;
  border-radius: 10px;
  background: #17191e;
  color: #b2b8c0;
  display: grid;
  place-items: center;
  font: 680 .67rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  font-style: normal;
}

.provider-mark[data-platform="openai"] { color: #dde1e5; background: #171a1d; }
.provider-mark[data-platform="anthropic"] { color: #d7b894; background: #1a1714; }
.provider-mark[data-platform="gemini"] { color: #9bc3ef; background: #14191f; }
.provider-mark[data-platform="grok"],
.provider-mark[data-platform="xai"] { color: #d6d8dc; background: #16171a; }

.upstream-identity > span {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.upstream-identity strong {
  overflow: hidden;
  color: #f0f2f4;
  font-size: .81rem;
  line-height: 1.2;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upstream-identity small {
  margin-top: 5px;
  overflow: hidden;
  color: #68717c;
  font-size: .64rem;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upstream-identity small b {
  margin: 0 5px;
  color: #3f454e;
  font-weight: 400;
}

.upstream-health,
.upstream-load,
.upstream-routing {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.upstream-health {
  align-items: flex-start;
  gap: 5px;
}

.health-badge {
  height: 24px;
  padding: 0 8px;
  border: 1px solid #30343b;
  border-radius: 999px;
  background: #14161a;
  color: #a9b0b8;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: .67rem;
  font-weight: 620;
}

.health-badge i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #7d8590;
}

.health-badge.good {
  border-color: rgba(67, 205, 152, .17);
  background: rgba(67, 205, 152, .055);
  color: #90d7bb;
}

.health-badge.good i { background: var(--aa-green); }

.health-badge.muted {
  border-color: rgba(215, 169, 91, .18);
  background: rgba(215, 169, 91, .055);
  color: #cbb07e;
}

.health-badge.muted i { background: var(--aa-amber); }

.health-badge.danger {
  border-color: rgba(225, 108, 115, .2);
  background: rgba(225, 108, 115, .06);
  color: #dfa0a5;
}

.health-badge.danger i { background: var(--aa-red); }

.upstream-health > small,
.upstream-routing > small {
  color: #626b76;
  font-size: .62rem;
}

.upstream-load {
  gap: 8px;
}

.load-values {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.load-values > b {
  color: #cdd2d8;
  font-size: .70rem;
  font-weight: 650;
}

.load-values > small {
  color: #626a75;
  font-size: .62rem;
}

.load-track {
  width: 100%;
  max-width: 108px;
  height: 4px;
  overflow: hidden;
  border-radius: 99px;
  background: #292c33;
}

.load-track > b {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #818b97;
}

.upstream-routing {
  align-items: flex-start;
  gap: 5px;
}

.upstream-routing > b {
  min-width: 38px;
  height: 23px;
  padding: 0 7px;
  border: 1px solid #31353d;
  border-radius: 6px;
  background: #15171b;
  color: #d0d4da;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font: 650 .66rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.upstream-groups {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
}

.upstream-groups > b {
  height: 23px;
  padding: 0 7px;
  border: 1px solid #2d3138;
  border-radius: 6px;
  background: #121419;
  color: #89919b;
  display: inline-flex;
  align-items: center;
  font: 560 .62rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.upstream-groups > small,
.upstream-last-used {
  color: #737b86;
  font-size: .68rem;
}

.upstream-last-used {
  white-space: nowrap;
}

.row-action {
  width: 28px;
  height: 28px;
  justify-self: end;
  border: 1px solid transparent;
  border-radius: 7px;
  color: #68717c;
  display: grid;
  place-items: center;
  transition: border-color .15s ease, background .15s ease, color .15s ease;
}

.upstream-row:hover .row-action {
  border-color: #30343c;
  background: #181a20;
  color: #c3c8ce;
}

.row-action svg {
  width: 14px;
  height: 14px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  transition: transform .15s ease;
}

.row-action.open svg {
  transform: rotate(90deg);
}

.upstream-detail {
  margin: -1px 0 0;
  padding: 17px 18px 18px 63px;
  display: grid;
  grid-template-columns: minmax(220px, 1.35fr) .72fr .78fr .95fr;
  gap: 13px 24px;
  border-bottom: 1px solid var(--aa-border);
  background: #0c0e12;
}

.upstream-detail > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.upstream-detail span {
  color: #626b76;
  font-size: .62rem;
}

.upstream-detail strong {
  overflow: hidden;
  color: #bdc3ca;
  font-size: .71rem;
  font-weight: 580;
  line-height: 1.5;
  text-overflow: ellipsis;
}

.account-warning {
  grid-column: 1 / -1;
  margin: 2px 0 0;
  padding: 10px 12px;
  border: 1px solid rgba(225, 108, 115, .20);
  border-radius: 8px;
  background: rgba(225, 108, 115, .055);
  display: flex;
  gap: 10px;
}

.account-warning strong {
  color: #dda0a4;
}

.accounts-empty {
  min-height: 280px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--aa-muted);
  text-align: center;
}

.accounts-empty strong {
  color: var(--aa-text-soft);
  font-size: .88rem;
}

.accounts-empty span {
  margin-top: 7px;
  font-size: .72rem;
}

.accounts-empty button {
  margin-top: 14px;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 7px;
  font-size: .7rem;
}

.accounts-loading {
  padding: 12px 18px;
  display: grid;
  gap: 9px;
}

.accounts-loading i {
  display: block;
  height: 52px;
  border-radius: 8px;
  background: linear-gradient(90deg, #111318 20%, #17191f 50%, #111318 80%);
  background-size: 200% 100%;
  animation: skeleton-shift 1.15s linear infinite;
}

.accounts-footer {
  min-height: 56px;
  padding: 0 14px 0 18px;
  border-top: 1px solid var(--aa-border);
  background: #0c0e12;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.accounts-footer > span {
  color: #626b76;
  font-size: .67rem;
}

.accounts-footer > div {
  display: flex;
  gap: 7px;
}

.accounts-footer button {
  min-height: 34px;
  padding: 0 10px;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: .68rem;
}

.accounts-footer button svg {
  width: 13px;
  height: 13px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

@keyframes account-spin { to { transform: rotate(360deg); } }
@keyframes skeleton-shift { to { background-position: -200% 0; } }

@media (max-width: 1180px) {
  .fleet-overview {
    grid-template-columns: 1.15fr repeat(3, minmax(0, .85fr));
  }

  .upstream-table-head,
  .upstream-row {
    grid-template-columns: minmax(230px, 1.42fr) minmax(122px, .74fr) minmax(136px, .76fr) minmax(105px, .62fr) minmax(105px, .62fr) 30px;
  }

  .upstream-table-head > span:nth-child(5),
  .upstream-row > span:nth-child(5) {
    display: none;
  }

  .toolbar-title small,
  .select-control:nth-of-type(2) {
    display: none;
  }
}

@media (max-width: 960px) {
  .fleet-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .fleet-overview article:nth-child(3)::before {
    display: none;
  }

  .fleet-overview article:nth-child(n + 3) {
    border-top: 1px solid #24272e;
  }

  .accounts-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .toolbar-controls {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .search-control {
    width: 100%;
  }

  .upstream-table-head,
  .upstream-row {
    grid-template-columns: minmax(220px, 1.35fr) minmax(120px, .8fr) minmax(125px, .75fr) 30px;
  }

  .upstream-table-head > span:nth-child(4),
  .upstream-table-head > span:nth-child(5),
  .upstream-table-head > span:nth-child(6),
  .upstream-row > span:nth-child(4),
  .upstream-row > span:nth-child(5),
  .upstream-row > span:nth-child(6) {
    display: none;
  }

  .upstream-detail {
    padding-left: 18px;
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 640px) {
  .accounts-heading {
    min-height: auto;
    padding-bottom: 20px;
  }

  .accounts-title-line {
    align-items: flex-start;
    flex-direction: column;
    gap: 9px;
  }

  .accounts-heading h1 {
    font-size: 1.75rem;
  }

  .accounts-heading {
    flex-direction: column;
    gap: 14px;
  }

  .accounts-heading-actions {
    width: 100%;
  }

  .create-account-button {
    flex: 1;
    justify-content: center;
  }

  .refresh-button {
    padding: 0 11px;
  }

  .fleet-overview {
    grid-template-columns: 1fr;
  }

  .fleet-overview article + article::before {
    display: none;
  }

  .fleet-overview article + article {
    border-top: 1px solid #24272e;
  }

  .select-control {
    flex: 1 1 130px;
  }

  .upstream-table-head {
    display: none;
  }

  .upstream-row {
    grid-template-columns: minmax(0, 1fr) 30px;
    min-height: 76px;
  }

  .upstream-row > span:not(.upstream-identity):not(.row-action) {
    display: none;
  }

  .upstream-detail {
    grid-template-columns: 1fr;
  }
}
</style>
