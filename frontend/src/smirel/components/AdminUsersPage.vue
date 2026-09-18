<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, getErrorMessage, previewMode } from '../core/api'
import { pushNotification } from '../core/notifications'
import WorkspaceNavIcon from './WorkspaceNavIcon.vue'

interface AdminUserRow {
  id: number
  email: string
  username?: string
  role: string
  balance: number
  frozen_balance?: number
  concurrency: number
  current_concurrency?: number
  rpm_limit?: number
  status: string
  allowed_groups?: number[]
  last_active_at?: string | null
  last_used_at?: string | null
  created_at?: string
  updated_at?: string
  notes?: string
}

interface PaginatedUsers {
  items?: AdminUserRow[]
  total?: number
  page?: number
  page_size?: number
  pages?: number
}

const PAGE_SIZE = 20

const zh = {
  eyebrow: 'USER MANAGEMENT',
  addUser: '新增用户',
  refresh: '刷新',
  refreshing: '刷新中',
  totalUsers: '总用户',
  totalHint: '平台账户总量',
  activeUsers: '活跃账户',
  activeHint: '当前可正常使用',
  admins: '管理员',
  adminHint: '拥有后台权限',
  disabled: '已停用',
  disabledHint: '当前禁止登录',
  searchPlaceholder: '搜索邮箱、用户名…',
  allRoles: '全部角色',
  normalUser: '普通用户',
  administrator: '管理员',
  allStatuses: '全部状态',
  active: '正常',
  disabledStatus: '已停用',
  clear: '清除筛选',
  result: '用户列表',
  resultCount: '共 {count} 个账户',
  identity: '用户',
  role: '角色',
  status: '状态',
  balance: '余额',
  concurrency: '并发',
  lastActive: '最近活跃',
  action: '操作',
  allocate: '分配额度',
  allocateTitle: '调整用户额度',
  allocateText: '调整账户余额后，用户可立即使用新的额度。',
  operation: '调整方式',
  addBalance: '增加额度',
  subtractBalance: '扣减额度',
  setBalance: '设为指定额度',
  amount: '金额（USD）',
  amountPlaceholder: '例如 10.00',
  notes: '备注',
  notesPlaceholder: '可选，记录本次调整原因',
  currentBalance: '当前余额',
  afterBalance: '调整后余额',
  allocating: '提交中…',
  allocateSubmit: '确认调整',
  allocateFailed: '额度调整失败',
  allocateRequired: '请输入大于 0 的有效金额。',
  allocatedTitle: '额度已更新',
  allocatedMessage: '{email} 的余额已更新为 {balance}。',
  neverActive: '暂无记录',
  enable: '启用',
  disable: '停用',
  adminProtected: '受保护',
  emptyTitle: '没有找到用户',
  emptyText: '尝试调整搜索词或筛选条件。',
  page: '第 {page} / {pages} 页',
  previous: '上一页',
  next: '下一页',
  dialogTitle: '新增用户',
  dialogText: '创建一个可登录 Smirel 的平台账户。',
  username: '用户名',
  usernamePlaceholder: '例如 Acme Team',
  email: '邮箱',
  emailPlaceholder: 'name@example.com',
  password: '初始密码',
  passwordPlaceholder: '至少 6 位',
  accountRole: '账户角色',
  cancel: '取消',
  creating: '创建中…',
  create: '创建用户',
  createFailed: '创建失败',
  createRequired: '请填写有效邮箱和至少 6 位密码。',
  createdTitle: '用户已创建',
  createdMessage: '账户 {email} 已加入 Smirel。',
  statusTitle: '账户状态已更新',
  enabledMessage: '{email} 已恢复使用。',
  disabledMessage: '{email} 已停用。',
  loadFailed: '用户数据暂时无法加载，请稍后重试。',
  usersShown: '当前显示 {count} 条',
}

const en = {
  eyebrow: 'USER MANAGEMENT',
  addUser: 'Add user',
  refresh: 'Refresh',
  refreshing: 'Refreshing',
  totalUsers: 'Total users',
  totalHint: 'All platform accounts',
  activeUsers: 'Active accounts',
  activeHint: 'Available for use',
  admins: 'Administrators',
  adminHint: 'Console access',
  disabled: 'Disabled',
  disabledHint: 'Sign-in blocked',
  searchPlaceholder: 'Search email or username…',
  allRoles: 'All roles',
  normalUser: 'User',
  administrator: 'Administrator',
  allStatuses: 'All statuses',
  active: 'Active',
  disabledStatus: 'Disabled',
  clear: 'Clear filters',
  result: 'Users',
  resultCount: '{count} accounts',
  identity: 'User',
  role: 'Role',
  status: 'Status',
  balance: 'Balance',
  concurrency: 'Concurrency',
  lastActive: 'Last active',
  action: 'Action',
  allocate: 'Adjust balance',
  allocateTitle: 'Adjust user balance',
  allocateText: 'The new balance is available to the user immediately.',
  operation: 'Operation',
  addBalance: 'Add balance',
  subtractBalance: 'Subtract balance',
  setBalance: 'Set balance',
  amount: 'Amount (USD)',
  amountPlaceholder: 'e.g. 10.00',
  notes: 'Note',
  notesPlaceholder: 'Optional reason for this adjustment',
  currentBalance: 'Current balance',
  afterBalance: 'Balance after adjustment',
  allocating: 'Submitting…',
  allocateSubmit: 'Confirm adjustment',
  allocateFailed: 'Balance adjustment failed',
  allocateRequired: 'Enter a valid amount greater than 0.',
  allocatedTitle: 'Balance updated',
  allocatedMessage: '{email} balance is now {balance}.',
  neverActive: 'No activity',
  enable: 'Enable',
  disable: 'Disable',
  adminProtected: 'Protected',
  emptyTitle: 'No users found',
  emptyText: 'Try a different search or filter.',
  page: 'Page {page} of {pages}',
  previous: 'Previous',
  next: 'Next',
  dialogTitle: 'Add user',
  dialogText: 'Create a platform account that can sign in to Smirel.',
  username: 'Username',
  usernamePlaceholder: 'e.g. Acme Team',
  email: 'Email',
  emailPlaceholder: 'name@example.com',
  password: 'Initial password',
  passwordPlaceholder: 'At least 6 characters',
  accountRole: 'Role',
  cancel: 'Cancel',
  creating: 'Creating…',
  create: 'Create user',
  createFailed: 'Create failed',
  createRequired: 'Enter a valid email and a password with at least 6 characters.',
  createdTitle: 'User created',
  createdMessage: '{email} was added to Smirel.',
  statusTitle: 'Account status updated',
  enabledMessage: '{email} is active again.',
  disabledMessage: '{email} has been disabled.',
  loadFailed: 'User data could not be loaded. Try again shortly.',
  usersShown: '{count} rows shown',
}

const { t, locale } = useI18n()
const copy = computed(() => locale.value === 'en-US' ? en : zh)

const users = ref<AdminUserRow[]>([])
const total = ref(0)
const pages = ref(1)
const page = ref(1)
const loading = ref(false)
const error = ref('')
const searchInput = ref('')
const search = ref('')
const roleFilter = ref('')
const statusFilter = ref('')
const mutatingId = ref<number | null>(null)
const showCreate = ref(false)
const createError = ref('')
const creating = ref(false)
const showBalance = ref(false)
const balanceError = ref('')
const allocating = ref(false)
const balanceUser = ref<AdminUserRow | null>(null)
const balanceForm = reactive({ operation: 'add', amount: '', notes: '' })
const summary = reactive({ total: 0, active: 0, admins: 0, disabled: 0 })
const createForm = reactive({ username: '', email: '', password: '', role: 'user' })

const previewUsers: AdminUserRow[] = [
  { id: 1024, email: 'ops@smirel.com', username: 'Smirel Ops', role: 'admin', balance: 128.62, concurrency: 20, current_concurrency: 3, status: 'active', last_active_at: '2026-09-07T12:14:00Z', created_at: '2026-05-18T08:20:00Z' },
  { id: 1019, email: 'dev@northstar.ai', username: 'Northstar AI', role: 'user', balance: 42.18, concurrency: 8, current_concurrency: 2, status: 'active', last_active_at: '2026-09-07T11:52:00Z', created_at: '2026-08-29T09:10:00Z' },
  { id: 1014, email: 'api@meridian.dev', username: 'Meridian', role: 'user', balance: 18.94, concurrency: 5, current_concurrency: 1, status: 'active', last_active_at: '2026-09-07T10:41:00Z', created_at: '2026-08-21T06:35:00Z' },
  { id: 1008, email: 'team@orbitlab.io', username: 'Orbit Lab', role: 'user', balance: 0, concurrency: 3, current_concurrency: 0, status: 'disabled', last_active_at: '2026-09-02T04:20:00Z', created_at: '2026-07-17T13:05:00Z' },
  { id: 1003, email: 'hello@pixelcore.cn', username: 'PixelCore', role: 'user', balance: 9.72, concurrency: 4, current_concurrency: 0, status: 'active', last_active_at: '2026-09-06T15:28:00Z', created_at: '2026-06-30T02:15:00Z' },
]

function interpolate(template: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce((result, [key, value]) => result.replace(`{${key}}`, String(value)), template)
}

function userName(user: AdminUserRow) {
  return user.username?.trim() || user.email.split('@')[0] || `User #${user.id}`
}

function userInitial(user: AdminUserRow) {
  return userName(user).slice(0, 1).toUpperCase()
}

function percent(part: number, whole: number) {
  if (!whole) return 0
  return Math.min(100, Math.round((part / whole) * 100))
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD', minimumFractionDigits: 2 }).format(Number(value || 0))
}

function formatDate(value?: string | null) {
  if (!value) return copy.value.neverActive
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return copy.value.neverActive
  return new Intl.DateTimeFormat(locale.value, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

function roleLabel(role: string) {
  return role === 'admin' ? copy.value.administrator : copy.value.normalUser
}

function statusLabel(status: string) {
  return status === 'disabled' ? copy.value.disabledStatus : copy.value.active
}

function resetCreateForm() {
  createForm.username = ''
  createForm.email = ''
  createForm.password = ''
  createForm.role = 'user'
  createError.value = ''
}

function closeCreateDialog() {
  if (creating.value) return
  showCreate.value = false
  resetCreateForm()
}

function openCreateDialog() {
  resetCreateForm()
  showCreate.value = true
}

function resetBalanceForm() {
  balanceForm.operation = 'add'
  balanceForm.amount = ''
  balanceForm.notes = ''
  balanceError.value = ''
}

function openBalanceDialog(user: AdminUserRow) {
  resetBalanceForm()
  balanceUser.value = user
  showBalance.value = true
}

function closeBalanceDialog() {
  if (allocating.value) return
  showBalance.value = false
  balanceUser.value = null
  resetBalanceForm()
}

function balanceAfterAdjustment() {
  const user = balanceUser.value
  const amount = Number(balanceForm.amount)
  if (!user || !Number.isFinite(amount)) return Number(user?.balance || 0)
  if (balanceForm.operation === 'set') return amount
  if (balanceForm.operation === 'subtract') return Number(user.balance || 0) - amount
  return Number(user.balance || 0) + amount
}

async function updateUserBalance() {
  const user = balanceUser.value
  const amount = Number(balanceForm.amount)
  if (!user || !Number.isFinite(amount) || amount <= 0) {
    balanceError.value = copy.value.allocateRequired
    return
  }

  allocating.value = true
  balanceError.value = ''
  try {
    const nextBalance = balanceAfterAdjustment()
    if (previewMode) {
      const source = previewUsers.find((item) => item.id === user.id)
      if (source) source.balance = nextBalance
    } else {
      await api.post<AdminUserRow>(`/admin/users/${user.id}/balance`, {
        balance: amount,
        operation: balanceForm.operation,
        notes: balanceForm.notes.trim(),
      })
    }
    user.balance = nextBalance
    showBalance.value = false
    balanceUser.value = null
    resetBalanceForm()
    pushNotification({
      title: copy.value.allocatedTitle,
      message: interpolate(copy.value.allocatedMessage, { email: user.email, balance: formatCurrency(nextBalance) }),
      tone: 'success',
    })
    await loadUsers()
  } catch (caught) {
    balanceError.value = getErrorMessage(caught) || copy.value.allocateFailed
  } finally {
    allocating.value = false
  }
}

function filteredPreviewUsers() {
  const needle = search.value.trim().toLowerCase()
  return previewUsers.filter((user) => {
    const matchesSearch = !needle || `${user.email} ${user.username || ''}`.toLowerCase().includes(needle)
    const matchesRole = !roleFilter.value || user.role === roleFilter.value
    const matchesStatus = !statusFilter.value || user.status === statusFilter.value
    return matchesSearch && matchesRole && matchesStatus
  })
}

function updatePreviewSummary() {
  summary.total = previewUsers.length
  summary.active = previewUsers.filter((user) => user.status === 'active').length
  summary.admins = previewUsers.filter((user) => user.role === 'admin').length
  summary.disabled = previewUsers.filter((user) => user.status === 'disabled').length
}

async function loadSummary() {
  if (previewMode) {
    updatePreviewSummary()
    return
  }

  const params = { page: 1, page_size: 1, include_subscriptions: false }
  const [allResult, activeResult, adminResult, disabledResult] = await Promise.all([
    api.get<PaginatedUsers>('/admin/users', { params }),
    api.get<PaginatedUsers>('/admin/users', { params: { ...params, status: 'active' } }),
    api.get<PaginatedUsers>('/admin/users', { params: { ...params, role: 'admin' } }),
    api.get<PaginatedUsers>('/admin/users', { params: { ...params, status: 'disabled' } }),
  ])

  summary.total = Number(allResult.data.total || 0)
  summary.active = Number(activeResult.data.total || 0)
  summary.admins = Number(adminResult.data.total || 0)
  summary.disabled = Number(disabledResult.data.total || 0)
}

async function loadUsers(options: { refreshSummary?: boolean } = {}) {
  error.value = ''
  loading.value = true
  try {
    if (previewMode) {
      updatePreviewSummary()
      const filtered = filteredPreviewUsers()
      total.value = filtered.length
      pages.value = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
      page.value = Math.min(page.value, pages.value)
      const start = (page.value - 1) * PAGE_SIZE
      users.value = filtered.slice(start, start + PAGE_SIZE)
      return
    }

    const data = (await api.get<PaginatedUsers>('/admin/users', {
      params: {
        page: page.value,
        page_size: PAGE_SIZE,
        search: search.value || undefined,
        role: roleFilter.value || undefined,
        status: statusFilter.value || undefined,
        include_subscriptions: false,
      },
    })).data

    users.value = data.items || []
    total.value = Number(data.total || 0)
    pages.value = Math.max(1, Number(data.pages || 1))
    if (options.refreshSummary) await loadSummary()
  } catch (caught) {
    error.value = getErrorMessage(caught) || copy.value.loadFailed
  } finally {
    loading.value = false
  }
}

async function refreshAll() {
  await loadUsers({ refreshSummary: true })
}

function applyFilters() {
  search.value = searchInput.value.trim()
  page.value = 1
  void loadUsers()
}

function updateFilters() {
  page.value = 1
  void loadUsers()
}

function clearFilters() {
  searchInput.value = ''
  search.value = ''
  roleFilter.value = ''
  statusFilter.value = ''
  page.value = 1
  void loadUsers()
}

function changePage(nextPage: number) {
  if (nextPage < 1 || nextPage > pages.value || nextPage === page.value) return
  page.value = nextPage
  void loadUsers()
}

async function createUser() {
  const email = createForm.email.trim()
  const password = createForm.password
  if (!email.includes('@') || password.length < 6) {
    createError.value = copy.value.createRequired
    return
  }

  creating.value = true
  createError.value = ''
  try {
    if (previewMode) {
      previewUsers.unshift({
        id: Date.now(),
        email,
        username: createForm.username.trim(),
        role: createForm.role,
        balance: 0,
        concurrency: 0,
        current_concurrency: 0,
        status: 'active',
        last_active_at: null,
        created_at: new Date().toISOString(),
      })
    } else {
      await api.post<AdminUserRow>('/admin/users', {
        email,
        password,
        username: createForm.username.trim(),
        role: createForm.role,
      })
    }

    showCreate.value = false
    resetCreateForm()
    page.value = 1
    await refreshAll()
    pushNotification({
      title: copy.value.createdTitle,
      message: interpolate(copy.value.createdMessage, { email }),
      tone: 'success',
    })
  } catch (caught) {
    createError.value = getErrorMessage(caught) || copy.value.createFailed
  } finally {
    creating.value = false
  }
}

async function toggleUserStatus(user: AdminUserRow) {
  if (user.role === 'admin') return
  const nextStatus = user.status === 'disabled' ? 'active' : 'disabled'
  mutatingId.value = user.id
  error.value = ''
  try {
    if (previewMode) {
      const source = previewUsers.find((item) => item.id === user.id)
      if (source) source.status = nextStatus
    } else {
      await api.put(`/admin/users/${user.id}`, { status: nextStatus })
    }
    user.status = nextStatus
    await loadSummary()
    pushNotification({
      title: copy.value.statusTitle,
      message: interpolate(nextStatus === 'active' ? copy.value.enabledMessage : copy.value.disabledMessage, { email: user.email }),
      tone: nextStatus === 'active' ? 'success' : 'info',
    })
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    mutatingId.value = null
  }
}

onMounted(() => void refreshAll())
</script>

<template>
  <div class="admin-users-page">
    <header class="admin-users-heading">
      <div class="admin-users-heading-copy">
        <div class="admin-users-eyebrow"><span></span>{{ copy.eyebrow }}</div>
        <h1>{{ t('nav.adminUsers') }}</h1>
        <p>{{ t('workspace.descriptions.adminUsers') }}</p>
      </div>
      <div class="admin-users-heading-actions">
        <button class="admin-users-secondary-button" type="button" :disabled="loading" @click="refreshAll">
          <WorkspaceNavIcon name="refresh" />
          <span>{{ loading ? copy.refreshing : copy.refresh }}</span>
        </button>
        <button class="admin-users-primary-button" type="button" @click="openCreateDialog">
          <WorkspaceNavIcon name="user-plus" />
          <span>{{ copy.addUser }}</span>
        </button>
      </div>
    </header>

    <p v-if="error" class="admin-users-error">{{ error }}</p>

    <section class="admin-users-summary" aria-label="User summary">
      <article class="admin-users-summary-card">
        <span class="admin-users-summary-icon"><WorkspaceNavIcon name="users" /></span>
        <div><span>{{ copy.totalUsers }}</span><strong>{{ summary.total.toLocaleString() }}</strong><small>{{ copy.totalHint }}</small></div>
      </article>
      <article class="admin-users-summary-card">
        <span class="admin-users-summary-icon is-success"><WorkspaceNavIcon name="user-check" /></span>
        <div><span>{{ copy.activeUsers }}</span><strong>{{ summary.active.toLocaleString() }}</strong><small>{{ copy.activeHint }} · {{ percent(summary.active, summary.total) }}%</small></div>
      </article>
      <article class="admin-users-summary-card">
        <span class="admin-users-summary-icon"><WorkspaceNavIcon name="shield" /></span>
        <div><span>{{ copy.admins }}</span><strong>{{ summary.admins.toLocaleString() }}</strong><small>{{ copy.adminHint }}</small></div>
      </article>
      <article class="admin-users-summary-card">
        <span class="admin-users-summary-icon is-muted"><WorkspaceNavIcon name="user-x" /></span>
        <div><span>{{ copy.disabled }}</span><strong>{{ summary.disabled.toLocaleString() }}</strong><small>{{ copy.disabledHint }}</small></div>
      </article>
    </section>

    <section class="admin-users-panel">
      <div class="admin-users-toolbar">
        <form class="admin-users-search" @submit.prevent="applyFilters">
          <WorkspaceNavIcon name="search" />
          <input v-model="searchInput" :placeholder="copy.searchPlaceholder" autocomplete="off" />
        </form>
        <select v-model="roleFilter" :aria-label="copy.role" @change="updateFilters">
          <option value="">{{ copy.allRoles }}</option>
          <option value="user">{{ copy.normalUser }}</option>
          <option value="admin">{{ copy.administrator }}</option>
        </select>
        <select v-model="statusFilter" :aria-label="copy.status" @change="updateFilters">
          <option value="">{{ copy.allStatuses }}</option>
          <option value="active">{{ copy.active }}</option>
          <option value="disabled">{{ copy.disabledStatus }}</option>
        </select>
        <button v-if="search || roleFilter || statusFilter" class="admin-users-clear" type="button" @click="clearFilters">{{ copy.clear }}</button>
      </div>

      <div class="admin-users-table-heading">
        <div><strong>{{ copy.result }}</strong><span>{{ interpolate(copy.resultCount, { count: total.toLocaleString() }) }}</span></div>
        <small>{{ interpolate(copy.usersShown, { count: users.length }) }}</small>
      </div>

      <div class="admin-users-table" :class="{ 'is-loading': loading }">
        <div class="admin-users-table-head">
          <span>{{ copy.identity }}</span>
          <span>{{ copy.role }}</span>
          <span>{{ copy.status }}</span>
          <span>{{ copy.balance }}</span>
          <span>{{ copy.concurrency }}</span>
          <span>{{ copy.lastActive }}</span>
          <span>{{ copy.action }}</span>
        </div>

        <div v-for="user in users" :key="user.id" class="admin-users-row">
          <div class="admin-users-identity">
            <span class="admin-users-avatar">{{ userInitial(user) }}</span>
            <div><strong>{{ userName(user) }}</strong><span>{{ user.email }}</span><small>#{{ user.id }}</small></div>
          </div>
          <div><span class="admin-users-role" :class="{ 'is-admin': user.role === 'admin' }"><WorkspaceNavIcon v-if="user.role === 'admin'" name="shield" />{{ roleLabel(user.role) }}</span></div>
          <div><span class="admin-users-status" :class="`is-${user.status || 'active'}`"><i></i>{{ statusLabel(user.status) }}</span></div>
          <div class="admin-users-balance"><strong>{{ formatCurrency(user.balance) }}</strong><small v-if="Number(user.frozen_balance || 0) > 0">+ {{ formatCurrency(Number(user.frozen_balance || 0)) }} frozen</small></div>
          <div class="admin-users-concurrency"><strong>{{ Number(user.current_concurrency || 0) }}</strong><span>/ {{ user.concurrency || '∞' }}</span></div>
          <div class="admin-users-activity"><strong>{{ formatDate(user.last_active_at || user.last_used_at) }}</strong><span v-if="user.rpm_limit">RPM {{ user.rpm_limit }}</span></div>
          <div class="admin-users-action-cell">
            <button class="admin-users-balance-button" type="button" @click="openBalanceDialog(user)"><WorkspaceNavIcon name="wallet" />{{ copy.allocate }}</button>
            <span v-if="user.role === 'admin'" class="admin-users-protected"><WorkspaceNavIcon name="shield" />{{ copy.adminProtected }}</span>
            <button v-else type="button" :class="{ 'is-enable': user.status === 'disabled' }" :disabled="mutatingId === user.id" @click="toggleUserStatus(user)">{{ user.status === 'disabled' ? copy.enable : copy.disable }}</button>
          </div>
        </div>

        <div v-if="!users.length && !loading" class="admin-users-empty">
          <span><WorkspaceNavIcon name="users" /></span>
          <strong>{{ copy.emptyTitle }}</strong>
          <p>{{ copy.emptyText }}</p>
        </div>
      </div>

      <footer v-if="total > 0" class="admin-users-pagination">
        <span>{{ interpolate(copy.page, { page, pages }) }}</span>
        <div>
          <button type="button" :disabled="page <= 1 || loading" @click="changePage(page - 1)"><WorkspaceNavIcon name="chevron-left" />{{ copy.previous }}</button>
          <button type="button" :disabled="page >= pages || loading" @click="changePage(page + 1)">{{ copy.next }}<WorkspaceNavIcon name="chevron-right" /></button>
        </div>
      </footer>
    </section>

    <Teleport to="body">
      <div v-if="showCreate" class="admin-user-dialog-backdrop" @mousedown.self="closeCreateDialog">
        <form class="admin-user-dialog" @submit.prevent="createUser">
          <div class="admin-user-dialog-header">
            <div class="admin-user-dialog-icon"><WorkspaceNavIcon name="user-plus" /></div>
            <div><h2>{{ copy.dialogTitle }}</h2><p>{{ copy.dialogText }}</p></div>
          </div>

          <p v-if="createError" class="admin-user-dialog-error">{{ createError }}</p>

          <label>
            <span>{{ copy.username }}</span>
            <input v-model="createForm.username" :placeholder="copy.usernamePlaceholder" autocomplete="off" />
          </label>
          <label>
            <span>{{ copy.email }}</span>
            <input v-model="createForm.email" type="email" :placeholder="copy.emailPlaceholder" autocomplete="email" required />
          </label>
          <label>
            <span>{{ copy.password }}</span>
            <input v-model="createForm.password" type="password" :placeholder="copy.passwordPlaceholder" autocomplete="new-password" minlength="6" required />
          </label>
          <label>
            <span>{{ copy.accountRole }}</span>
            <select v-model="createForm.role">
              <option value="user">{{ copy.normalUser }}</option>
              <option value="admin">{{ copy.administrator }}</option>
            </select>
          </label>

          <div class="admin-user-dialog-actions">
            <button class="admin-users-secondary-button" type="button" :disabled="creating" @click="closeCreateDialog">{{ copy.cancel }}</button>
            <button class="admin-users-primary-button" type="submit" :disabled="creating">{{ creating ? copy.creating : copy.create }}</button>
          </div>
        </form>
      </div>
      <div v-if="showBalance && balanceUser" class="admin-user-dialog-backdrop" @mousedown.self="closeBalanceDialog">
        <form class="admin-user-dialog" @submit.prevent="updateUserBalance">
          <div class="admin-user-dialog-header">
            <div class="admin-user-dialog-icon"><WorkspaceNavIcon name="wallet" /></div>
            <div><h2>{{ copy.allocateTitle }}</h2><p>{{ copy.allocateText }}</p></div>
          </div>
          <div class="admin-user-balance-target"><strong>{{ userName(balanceUser) }}</strong><span>{{ balanceUser.email }}</span></div>
          <p v-if="balanceError" class="admin-user-dialog-error">{{ balanceError }}</p>
          <div class="admin-user-balance-preview"><span>{{ copy.currentBalance }} <strong>{{ formatCurrency(balanceUser.balance) }}</strong></span><span>{{ copy.afterBalance }} <strong>{{ formatCurrency(balanceAfterAdjustment()) }}</strong></span></div>
          <label><span>{{ copy.operation }}</span><select v-model="balanceForm.operation"><option value="add">{{ copy.addBalance }}</option><option value="subtract">{{ copy.subtractBalance }}</option><option value="set">{{ copy.setBalance }}</option></select></label>
          <label><span>{{ copy.amount }}</span><input v-model="balanceForm.amount" type="number" min="0.01" step="0.01" :placeholder="copy.amountPlaceholder" required /></label>
          <label><span>{{ copy.notes }}</span><input v-model="balanceForm.notes" :placeholder="copy.notesPlaceholder" maxlength="500" /></label>
          <div class="admin-user-dialog-actions"><button class="admin-users-secondary-button" type="button" :disabled="allocating" @click="closeBalanceDialog">{{ copy.cancel }}</button><button class="admin-users-primary-button" type="submit" :disabled="allocating">{{ allocating ? copy.allocating : copy.allocateSubmit }}</button></div>
        </form>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.admin-users-page { width: 100%; }

.admin-users-heading {
  min-height: 92px;
  margin-bottom: 24px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.admin-users-heading-copy { min-width: 0; }
.admin-users-eyebrow {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #66707c;
  font: 700 .66rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .12em;
}
.admin-users-eyebrow > span { width: 18px; height: 1px; background: #3199ea; }
.admin-users-heading h1 { margin: 0; color: #f5f6f8; font-size: 2rem; line-height: 1.12; font-weight: 680; letter-spacing: -.035em; }
.admin-users-heading p { margin: 9px 0 0; color: #858c96; font-size: .88rem; line-height: 1.55; }
.admin-users-heading-actions { display: flex; gap: 9px; align-items: center; }

.admin-users-primary-button,
.admin-users-secondary-button {
  min-height: 40px;
  padding: 0 14px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: .80rem;
  font-weight: 620;
  cursor: pointer;
  transition: border-color .15s ease, background-color .15s ease, color .15s ease, transform .15s ease;
}
.admin-users-primary-button { border: 1px solid #359ce8; background: #2f96e8; color: #fff; }
.admin-users-primary-button:hover:not(:disabled) { background: #3a9feb; border-color: #55adf0; transform: translateY(-1px); }
.admin-users-secondary-button { border: 1px solid #2a2f37; background: #111318; color: #b6bcc4; }
.admin-users-secondary-button:hover:not(:disabled) { border-color: #3a4049; background: #16191f; color: #eef1f4; }
.admin-users-primary-button:disabled,
.admin-users-secondary-button:disabled { opacity: .55; cursor: not-allowed; }
.admin-users-primary-button :deep(.workspace-nav-icon),
.admin-users-secondary-button :deep(.workspace-nav-icon) { width: 16px; height: 16px; }

.admin-users-error {
  margin: -8px 0 16px;
  padding: 10px 12px;
  border: 1px solid rgba(239, 112, 112, .22);
  border-radius: 8px;
  background: rgba(116, 35, 35, .12);
  color: #d9a0a0;
  font-size: .78rem;
}

.admin-users-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.admin-users-summary-card {
  position: relative;
  min-height: 132px;
  padding: 20px;
  border: 1px solid #23262d;
  border-radius: 11px;
  background: #101116;
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
  overflow: hidden;
}
.admin-users-summary-card::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 72px;
  height: 72px;
  background: radial-gradient(circle at top right, rgba(47, 150, 232, .06), transparent 68%);
  pointer-events: none;
}
.admin-users-summary-icon {
  width: 38px;
  height: 38px;
  border: 1px solid #2a3541;
  border-radius: 9px;
  background: #111923;
  color: #62afe9;
  display: grid;
  place-items: center;
}
.admin-users-summary-icon.is-success { border-color: #273b35; background: #101a17; color: #62cda5; }
.admin-users-summary-icon.is-muted { border-color: #303239; background: #15161a; color: #9298a1; }
.admin-users-summary-icon :deep(.workspace-nav-icon) { width: 17px; height: 17px; }
.admin-users-summary-card > div { min-width: 0; display: flex; flex-direction: column; }
.admin-users-summary-card > div > span { color: #888f99; font-size: .77rem; font-weight: 590; }
.admin-users-summary-card strong { margin-top: 8px; color: #f2f4f6; font-size: 1.72rem; line-height: 1; font-weight: 680; letter-spacing: -.035em; }
.admin-users-summary-card small { margin-top: 9px; color: #69717c; font-size: .70rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.admin-users-panel {
  margin-top: 14px;
  border: 1px solid #23262d;
  border-radius: 11px;
  background: #101116;
  overflow: hidden;
}
.admin-users-toolbar {
  min-height: 66px;
  padding: 11px 13px;
  border-bottom: 1px solid #22262c;
  display: flex;
  align-items: center;
  gap: 8px;
}
.admin-users-search {
  width: min(410px, 42vw);
  height: 40px;
  padding: 0 12px;
  border: 1px solid #2a2f37;
  border-radius: 8px;
  background: #0c0e12;
  display: flex;
  align-items: center;
  gap: 9px;
  color: #656e79;
}
.admin-users-search:focus-within { border-color: #3b6b91; box-shadow: 0 0 0 3px rgba(47, 150, 232, .07); }
.admin-users-search :deep(.workspace-nav-icon) { width: 16px; height: 16px; flex: 0 0 16px; }
.admin-users-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: #e3e6ea; font: inherit; font-size: .80rem; }
.admin-users-search input::placeholder { color: #626a74; }
.admin-users-toolbar select {
  height: 40px;
  min-width: 126px;
  padding: 0 32px 0 11px;
  border: 1px solid #2a2f37;
  border-radius: 8px;
  background: #0c0e12;
  color: #aeb4bc;
  outline: none;
  font-size: .78rem;
}
.admin-users-toolbar select:focus { border-color: #3b6b91; }
.admin-users-clear { margin-left: auto; padding: 8px 10px; border: 0; background: transparent; color: #777f8a; font-size: .73rem; cursor: pointer; }
.admin-users-clear:hover { color: #b8bec6; }

.admin-users-table-heading {
  min-height: 54px;
  padding: 0 17px;
  border-bottom: 1px solid #22262c;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.admin-users-table-heading > div { display: flex; align-items: center; gap: 9px; }
.admin-users-table-heading strong { color: #dce0e5; font-size: .83rem; font-weight: 630; }
.admin-users-table-heading span { padding: 3px 8px; border-radius: 999px; background: #181b20; color: #777f89; font-size: .67rem; }
.admin-users-table-heading small { color: #656d78; font-size: .68rem; }

.admin-users-table { width: 100%; overflow-x: auto; transition: opacity .15s ease; }
.admin-users-table.is-loading { opacity: .58; }
.admin-users-table-head,
.admin-users-row {
  min-width: 1060px;
  display: grid;
  grid-template-columns: minmax(240px, 1.8fr) minmax(105px, .7fr) minmax(95px, .65fr) minmax(110px, .75fr) minmax(95px, .6fr) minmax(150px, 1fr) minmax(86px, .55fr);
  gap: 16px;
  align-items: center;
}
.admin-users-table-head {
  height: 43px;
  padding: 0 17px;
  border-bottom: 1px solid #252930;
  background: #0d0f13;
  color: #707985;
  font-size: .68rem;
  font-weight: 620;
  letter-spacing: .015em;
}
.admin-users-row {
  min-height: 74px;
  padding: 10px 17px;
  border-bottom: 1px solid #20242a;
  color: #aab0b8;
  font-size: .76rem;
  transition: background-color .14s ease;
}
.admin-users-row:last-child { border-bottom: 0; }
.admin-users-row:hover { background: #121419; }
.admin-users-identity { min-width: 0; display: grid; grid-template-columns: 36px minmax(0, 1fr); gap: 11px; align-items: center; }
.admin-users-avatar {
  width: 36px;
  height: 36px;
  border: 1px solid #293744;
  border-radius: 9px;
  background: #111923;
  color: #83bde8;
  display: grid;
  place-items: center;
  font-size: .76rem;
  font-weight: 700;
}
.admin-users-identity > div { min-width: 0; display: grid; grid-template-columns: minmax(0, auto) auto; column-gap: 7px; align-items: baseline; }
.admin-users-identity strong { min-width: 0; color: #e2e5e9; font-size: .78rem; font-weight: 620; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.admin-users-identity span { grid-column: 1 / -1; margin-top: 3px; color: #777f89; font-size: .69rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.admin-users-identity small { color: #535b65; font-size: .62rem; }
.admin-users-role,
.admin-users-status {
  width: fit-content;
  min-height: 27px;
  padding: 0 8px;
  border: 1px solid #2a2e35;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #14161b;
  color: #989fa8;
  font-size: .69rem;
  font-weight: 580;
}
.admin-users-role.is-admin { border-color: #293949; background: #111b25; color: #89bde5; }
.admin-users-role :deep(.workspace-nav-icon) { width: 12px; height: 12px; }
.admin-users-status { border-color: transparent; background: transparent; padding-left: 0; }
.admin-users-status i { width: 6px; height: 6px; border-radius: 50%; background: #42ce99; box-shadow: 0 0 0 3px rgba(66, 206, 153, .07); }
.admin-users-status.is-disabled i { background: #707782; box-shadow: none; }
.admin-users-status.is-disabled { color: #777f89; }
.admin-users-balance,
.admin-users-activity { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.admin-users-balance strong,
.admin-users-activity strong { color: #c7ccd2; font-size: .74rem; font-weight: 580; white-space: nowrap; }
.admin-users-balance small,
.admin-users-activity span { color: #5f6873; font-size: .63rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.admin-users-concurrency { display: flex; align-items: baseline; gap: 4px; }
.admin-users-concurrency strong { color: #d4d8dd; font-size: .82rem; font-weight: 640; }
.admin-users-concurrency span { color: #656d78; font-size: .68rem; }
.admin-users-action-cell { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; justify-content: flex-start; }
.admin-users-action-cell button {
  min-width: 58px;
  height: 30px;
  padding: 0 9px;
  border: 1px solid #30343b;
  border-radius: 7px;
  background: #121419;
  color: #9299a2;
  font-size: .68rem;
  font-weight: 590;
  cursor: pointer;
}
.admin-users-action-cell button:hover:not(:disabled) { border-color: #53383b; background: #1a1315; color: #d59b9f; }
.admin-users-action-cell button.is-enable:hover:not(:disabled) { border-color: #315044; background: #111a17; color: #79c7aa; }
.admin-users-action-cell button:disabled { opacity: .45; cursor: not-allowed; }
.admin-users-action-cell .admin-users-balance-button { min-width: 0; border-color: #29445a; background: #111b25; color: #83bde8; display: inline-flex; align-items: center; gap: 5px; }
.admin-users-action-cell .admin-users-balance-button:hover:not(:disabled) { border-color: #3e78a5; background: #152535; color: #b4ddf7; }
.admin-users-action-cell .admin-users-balance-button :deep(.workspace-nav-icon) { width: 12px; height: 12px; }
.admin-users-protected { display: inline-flex; align-items: center; gap: 5px; color: #626a75; font-size: .65rem; }
.admin-users-protected :deep(.workspace-nav-icon) { width: 12px; height: 12px; }

.admin-users-empty { min-width: 700px; min-height: 230px; display: flex; flex-direction: column; align-items: center; justify-content: center; color: #676f79; }
.admin-users-empty > span { width: 40px; height: 40px; margin-bottom: 12px; border: 1px solid #292d34; border-radius: 10px; background: #121419; display: grid; place-items: center; color: #747d88; }
.admin-users-empty > span :deep(.workspace-nav-icon) { width: 18px; height: 18px; }
.admin-users-empty strong { color: #a7adb5; font-size: .80rem; font-weight: 620; }
.admin-users-empty p { margin: 6px 0 0; font-size: .70rem; }

.admin-users-pagination {
  min-height: 58px;
  padding: 0 14px 0 17px;
  border-top: 1px solid #22262c;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.admin-users-pagination > span { color: #6e7681; font-size: .69rem; }
.admin-users-pagination > div { display: flex; gap: 7px; }
.admin-users-pagination button {
  height: 32px;
  padding: 0 10px;
  border: 1px solid #2b3037;
  border-radius: 7px;
  background: #111318;
  color: #9299a2;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: .68rem;
  cursor: pointer;
}
.admin-users-pagination button:hover:not(:disabled) { border-color: #3b414a; color: #d7dbe0; }
.admin-users-pagination button:disabled { opacity: .38; cursor: not-allowed; }
.admin-users-pagination :deep(.workspace-nav-icon) { width: 13px; height: 13px; }

.admin-user-dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: 200;
  padding: 24px;
  background: rgba(3, 4, 6, .72);
  backdrop-filter: blur(5px);
  display: grid;
  place-items: center;
}
.admin-user-dialog {
  width: min(460px, 100%);
  padding: 22px;
  border: 1px solid #2b3038;
  border-radius: 13px;
  background: #101216;
  box-shadow: 0 24px 70px rgba(0, 0, 0, .42);
  display: grid;
  gap: 15px;
}
.admin-user-dialog-header { margin-bottom: 2px; display: grid; grid-template-columns: 40px minmax(0, 1fr); gap: 13px; align-items: start; }
.admin-user-dialog-icon { width: 40px; height: 40px; border: 1px solid #2c3d4c; border-radius: 10px; background: #111c27; color: #67afe6; display: grid; place-items: center; }
.admin-user-dialog-icon :deep(.workspace-nav-icon) { width: 18px; height: 18px; }
.admin-user-dialog h2 { margin: 1px 0 0; color: #f0f2f4; font-size: 1.04rem; font-weight: 650; }
.admin-user-dialog p { margin: 5px 0 0; color: #737b86; font-size: .72rem; line-height: 1.5; }
.admin-user-dialog-error { padding: 9px 10px; border: 1px solid rgba(239, 112, 112, .22); border-radius: 7px; background: rgba(116, 35, 35, .12); color: #d9a0a0 !important; }
.admin-user-balance-target { display: flex; flex-direction: column; gap: 3px; padding: 10px 11px; border: 1px solid #242d36; border-radius: 8px; background: #0d141b; }
.admin-user-balance-target strong { color: #e2e7eb; font-size: .79rem; font-weight: 620; }
.admin-user-balance-target span { color: #778391; font-size: .69rem; }
.admin-user-balance-preview { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.admin-user-balance-preview span { min-width: 0; padding: 9px 10px; border: 1px solid #242a32; border-radius: 7px; color: #777f89; font-size: .66rem; }
.admin-user-balance-preview strong { display: block; margin-top: 4px; color: #cbd4dc; font-size: .78rem; font-weight: 620; }
.admin-user-dialog label { display: grid; gap: 7px; color: #8c939d; font-size: .71rem; font-weight: 580; }
.admin-user-dialog input,
.admin-user-dialog select {
  width: 100%;
  height: 41px;
  padding: 0 11px;
  border: 1px solid #2a2f37;
  border-radius: 8px;
  background: #0b0d11;
  color: #e4e7ea;
  outline: none;
  font: inherit;
  font-size: .78rem;
}
.admin-user-dialog input:focus,
.admin-user-dialog select:focus { border-color: #3a6d95; box-shadow: 0 0 0 3px rgba(47, 150, 232, .07); }
.admin-user-dialog input::placeholder { color: #59616b; }
.admin-user-dialog-actions { margin-top: 3px; padding-top: 15px; border-top: 1px solid #23272e; display: flex; justify-content: flex-end; gap: 8px; }

@media (max-width: 1080px) {
  .admin-users-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .admin-users-toolbar { flex-wrap: wrap; }
  .admin-users-search { width: 100%; flex: 1 0 100%; }
  .admin-users-clear { margin-left: 0; }
}

@media (max-width: 720px) {
  .admin-users-heading { min-height: 0; flex-direction: column; }
  .admin-users-heading-actions { width: 100%; }
  .admin-users-heading-actions > button { flex: 1; }
  .admin-users-summary { grid-template-columns: 1fr; }
  .admin-users-summary-card { min-height: 112px; }
  .admin-users-toolbar select { flex: 1; min-width: 0; }
  .admin-users-clear { width: 100%; text-align: left; }
  .admin-users-table-heading small { display: none; }
  .admin-users-pagination { align-items: flex-start; flex-direction: column; padding-top: 13px; padding-bottom: 13px; }
  .admin-users-pagination > div { width: 100%; }
  .admin-users-pagination button { flex: 1; justify-content: center; }
}
</style>
