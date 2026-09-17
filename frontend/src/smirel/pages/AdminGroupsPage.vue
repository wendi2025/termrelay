<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { api, getErrorMessage } from '../core/api'

type ModelsListConfig = { enabled?: boolean; models?: string[] }
type GroupRow = {
  id: number; name: string; description?: string; platform: string; status: string
  rate_multiplier: number; subscription_type?: string; is_exclusive?: boolean
  models_list_config?: ModelsListConfig; account_count?: number; active_account_count?: number
}
type AccountRow = {
  id: number; name?: string; platform?: string; type?: string; status?: string; schedulable?: boolean
  group_ids?: number[]; credentials?: Record<string, unknown>; current_concurrency?: number; concurrency?: number
}
type ChannelRow = { id: number; name: string; description?: string; status: string; group_ids?: number[] }
type ListResponse<T> = { items?: T[]; total?: number }

const loading = ref(false)
const busy = ref('')
const error = ref('')
const notice = ref('')
const groups = ref<GroupRow[]>([])
const accounts = ref<AccountRow[]>([])
const channels = ref<ChannelRow[]>([])
const search = ref('')
const platform = ref('')
const expandedId = ref<number | null>(null)
const modelCache = reactive<Record<number, string[]>>({})
const selectedAccounts = reactive<Record<number, number[]>>({})
const showEditor = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({ name: '', description: '', platform: 'openai', rate_multiplier: 1, status: 'active' })

const visibleGroups = computed(() => {
  const q = search.value.trim().toLowerCase()
  return groups.value.filter((g) => {
    if (platform.value && g.platform !== platform.value) return false
    if (!q) return true
    return `${g.name} ${g.description || ''} ${g.platform} ${g.id}`.toLowerCase().includes(q)
  })
})
const activeCount = computed(() => groups.value.filter(g => g.status === 'active').length)
const configuredModelCount = computed(() => groups.value.reduce((sum, g) => sum + (g.models_list_config?.models?.length || 0), 0))
const publishedCount = computed(() => groups.value.filter(g => findCatalogChannel(g)).length)

function platformLabel(v: string) {
  return ({ openai: 'OpenAI Compatible', anthropic: 'Anthropic', gemini: 'Gemini', antigravity: 'Antigravity', grok: 'xAI / Grok', composite: 'Composite' } as Record<string,string>)[v] || v
}
function platformMark(v: string) {
  return ({ openai: 'O', anthropic: 'A', gemini: 'G', antigravity: 'AG', grok: 'X', composite: 'C' } as Record<string,string>)[v] || v.slice(0,1).toUpperCase()
}
function groupAccounts(g: GroupRow) { return accounts.value.filter(a => a.platform === g.platform && a.group_ids?.includes(g.id)) }
function eligibleAccounts(g: GroupRow) { return accounts.value.filter(a => a.platform === g.platform) }
function configuredModels(g: GroupRow) { return Array.isArray(g.models_list_config?.models) ? g.models_list_config!.models! : [] }
function findCatalogChannel(g: GroupRow) {
  return channels.value.find(c => c.group_ids?.includes(g.id) && (c.description || '').includes('[smirel:auto-catalog]'))
}
function accountMappingModels(a: AccountRow) {
  const raw = a.credentials?.model_mapping
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return []
  return Object.keys(raw as Record<string, unknown>).filter(Boolean)
}
function normalizeModels(value: unknown): string[] {
  const raw = Array.isArray(value) ? value : []
  const out = new Set<string>()
  for (const item of raw) {
    if (typeof item === 'string' && item.trim()) out.add(item.trim())
    else if (item && typeof item === 'object') {
      const id = String((item as Record<string, unknown>).id || (item as Record<string, unknown>).name || '').trim()
      if (id) out.add(id)
    }
  }
  return [...out].sort((a,b) => a.localeCompare(b))
}

async function loadAll() {
  loading.value = true; error.value = ''
  try {
    const [gr, ar, cr] = await Promise.all([
      api.get<ListResponse<GroupRow>>('/admin/groups', { params: { page: 1, page_size: 200, sort_by: 'sort_order', sort_order: 'asc' } }),
      api.get<ListResponse<AccountRow>>('/admin/accounts', { params: { page: 1, page_size: 500, sort_by: 'name', sort_order: 'asc' } }),
      api.get<ListResponse<ChannelRow>>('/admin/channels', { params: { page: 1, page_size: 200, sort_by: 'created_at', sort_order: 'desc' } }),
    ])
    groups.value = Array.isArray(gr.data?.items) ? gr.data.items : []
    accounts.value = Array.isArray(ar.data?.items) ? ar.data.items : []
    channels.value = Array.isArray(cr.data?.items) ? cr.data.items : []
    for (const g of groups.value) selectedAccounts[g.id] = groupAccounts(g).map(a => a.id)
  } catch (e) { error.value = getErrorMessage(e) }
  finally { loading.value = false }
}

async function toggleGroup(g: GroupRow) {
  expandedId.value = expandedId.value === g.id ? null : g.id
  if (expandedId.value === g.id && !modelCache[g.id]) await refreshCandidates(g)
}
async function refreshCandidates(g: GroupRow) {
  try {
    const r = await api.get<unknown[]>(`/admin/groups/${g.id}/models-list-candidates`, { params: { platform: g.platform } })
    modelCache[g.id] = normalizeModels(r.data)
  } catch (e) { error.value = getErrorMessage(e) }
}
function openCreate() {
  editingId.value = null
  Object.assign(form, { name: '', description: '', platform: 'openai', rate_multiplier: 1, status: 'active' })
  showEditor.value = true
}
function openEdit(g: GroupRow) {
  editingId.value = g.id
  Object.assign(form, { name: g.name, description: g.description || '', platform: g.platform, rate_multiplier: Number(g.rate_multiplier || 1), status: g.status || 'active' })
  showEditor.value = true
}
async function saveGroup() {
  if (!form.name.trim()) return
  busy.value = 'group'; error.value = ''
  try {
    const payload = { name: form.name.trim(), description: form.description.trim(), platform: form.platform, rate_multiplier: Math.max(0, Number(form.rate_multiplier) || 0), subscription_type: 'standard', status: form.status }
    if (editingId.value) await api.put(`/admin/groups/${editingId.value}`, payload)
    else await api.post('/admin/groups', payload)
    showEditor.value = false; notice.value = editingId.value ? '分组已更新' : '分组已创建'
    await loadAll()
  } catch (e) { error.value = getErrorMessage(e) }
  finally { busy.value = '' }
}

async function saveMembership(g: GroupRow) {
  busy.value = `members-${g.id}`; error.value = ''
  const selected = new Set(selectedAccounts[g.id] || [])
  try {
    for (const a of eligibleAccounts(g)) {
      const before = new Set(a.group_ids || [])
      const should = selected.has(a.id)
      if (before.has(g.id) === should) continue
      if (should) before.add(g.id); else before.delete(g.id)
      await api.put(`/admin/accounts/${a.id}`, { group_ids: [...before] })
    }
    notice.value = `${g.name} 的账号绑定已保存`
    await loadAll()
    await refreshCandidates(g)
  } catch (e) { error.value = getErrorMessage(e) }
  finally { busy.value = '' }
}

async function publishCatalog(g: GroupRow, explicitModels?: string[]) {
  const models = (explicitModels?.length ? explicitModels : configuredModels(g)).filter(Boolean)
  if (!models.length) throw new Error('没有可发布的模型，请先同步模型')
  const map = Object.fromEntries(models.map(m => [m, m]))
  const payload = {
    name: `${g.name} Catalog`,
    description: `[smirel:auto-catalog] ${g.description || g.name}`,
    group_ids: [g.id],
    model_mapping: { [g.platform]: map },
    model_pricing: [],
    billing_model_source: 'requested',
    restrict_models: true,
    features: '',
    features_config: {},
    apply_pricing_to_account_stats: false,
  }
  const existing = findCatalogChannel(g)
  if (existing) await api.put(`/admin/channels/${existing.id}`, { ...payload, status: 'active' })
  else await api.post('/admin/channels', payload)
}

async function syncModelsAndPublish(g: GroupRow) {
  busy.value = `sync-${g.id}`; error.value = ''; notice.value = ''
  try {
    const linked = groupAccounts(g).filter(a => a.status === 'active' && a.schedulable !== false)
    if (!linked.length) throw new Error('这个分组还没有可调度账号')
    const all = new Set<string>()
    let syncedAccounts = 0
    for (const a of linked) {
      accountMappingModels(a).forEach(m => all.add(m))
      if (a.type !== 'apikey') continue
      try {
        const r = await api.post<{ models?: unknown[] }>(`/admin/accounts/${a.id}/models/sync-upstream`)
        const models = normalizeModels(r.data?.models)
        if (!models.length) continue
        const mapping = Object.fromEntries(models.map(m => [m, m]))
        await api.put(`/admin/accounts/${a.id}`, { credentials: { model_mapping: mapping } })
        models.forEach(m => all.add(m)); syncedAccounts += 1
      } catch {
        // Other API-key account types may not implement /v1/models; keep existing mappings.
      }
    }
    if (!all.size) {
      const r = await api.get<unknown[]>(`/admin/groups/${g.id}/models-list-candidates`, { params: { platform: g.platform } })
      normalizeModels(r.data).forEach(m => all.add(m))
    }
    if (!all.size) throw new Error('没有发现可发布模型')
    const models = [...all].sort((a,b) => a.localeCompare(b))
    await api.put(`/admin/groups/${g.id}`, { models_list_config: { enabled: true, models } })
    await publishCatalog(g, models)
    modelCache[g.id] = models
    notice.value = `已保存 ${models.length} 个模型并发布到模型广场${syncedAccounts ? `，同步 ${syncedAccounts} 个上游账号` : ''}`
    await loadAll()
  } catch (e) { error.value = getErrorMessage(e) }
  finally { busy.value = '' }
}

async function setGroupStatus(g: GroupRow) {
  busy.value = `status-${g.id}`
  try {
    await api.put(`/admin/groups/${g.id}`, { status: g.status === 'active' ? 'inactive' : 'active' })
    await loadAll()
  } catch (e) { error.value = getErrorMessage(e) }
  finally { busy.value = '' }
}

onMounted(() => void loadAll())
</script>

<template>
  <section class="groups-page">
    <header class="page-head">
      <div><span class="kicker">ROUTING GROUPS</span><h1>分组与模型</h1><p>调度分组是长期逻辑路由：套餐绑定分组，真实上游账号可随时增删替换；模型白名单与模型广场跟随分组配置。</p></div>
      <div class="head-actions"><button class="ghost" :disabled="loading" @click="loadAll">{{ loading ? '刷新中' : '刷新' }}</button><button class="primary" @click="openCreate">＋ 新建分组</button></div>
    </header>

    <section class="overview">
      <article><span>分组总数</span><strong>{{ groups.length }}</strong><small>{{ activeCount }} 个启用</small></article>
      <article><span>上游账号</span><strong>{{ accounts.length }}</strong><small>当前账户池</small></article>
      <article><span>已配置模型</span><strong>{{ configuredModelCount }}</strong><small>/v1/models 白名单</small></article>
      <article><span>已发布目录</span><strong>{{ publishedCount }}</strong><small>模型广场渠道</small></article>
    </section>

    <div v-if="error" class="alert danger">{{ error }}</div>
    <div v-if="notice" class="alert success">{{ notice }}</div>

    <section class="panel">
      <header class="toolbar">
        <div><strong>逻辑调度分组</strong><span>{{ visibleGroups.length }} / {{ groups.length }}</span></div>
        <div class="filters"><input v-model="search" placeholder="搜索分组、平台或 ID" /><select v-model="platform"><option value="">全部平台</option><option value="openai">OpenAI Compatible</option><option value="anthropic">Anthropic</option><option value="gemini">Gemini</option><option value="antigravity">Antigravity</option><option value="grok">xAI / Grok</option><option value="composite">Composite</option></select></div>
      </header>

      <div class="group-list">
        <article v-for="g in visibleGroups" :key="g.id" class="group-card" :class="{ open: expandedId === g.id }">
          <button class="group-main" @click="toggleGroup(g)">
            <span class="mark">{{ platformMark(g.platform) }}</span>
            <span class="identity"><strong>{{ g.name }}</strong><small>{{ platformLabel(g.platform) }} · #{{ g.id }}</small></span>
            <span class="state" :class="g.status"><i></i>{{ g.status === 'active' ? '启用' : '停用' }}</span>
            <span class="metric"><small>上游账号</small><b>{{ groupAccounts(g).length }}</b></span>
            <span class="metric"><small>模型</small><b>{{ configuredModels(g).length || modelCache[g.id]?.length || 0 }}</b></span>
            <span class="metric"><small>倍率</small><b>{{ Number(g.rate_multiplier || 1).toFixed(2) }}×</b></span>
            <span class="publish" :class="{ on: !!findCatalogChannel(g) }"><i></i>{{ findCatalogChannel(g) ? '已发布' : '未发布' }}</span>
            <span class="chev">›</span>
          </button>

          <div v-if="expandedId === g.id" class="detail">
            <div class="detail-top">
              <div><span>分组说明</span><strong>{{ g.description || '暂无说明' }}</strong></div>
              <div class="detail-actions"><button @click="openEdit(g)">编辑分组</button><button @click="setGroupStatus(g)">{{ g.status === 'active' ? '停用' : '启用' }}</button><button class="accent" :disabled="busy === `sync-${g.id}`" @click="syncModelsAndPublish(g)">{{ busy === `sync-${g.id}` ? '同步中…' : '同步模型并发布' }}</button></div>
            </div>

            <div class="detail-grid">
              <section class="subpanel">
                <header><div><strong>上游账号绑定</strong><small>真实货源，可随时增删替换；不会改变套餐定义</small></div><button :disabled="busy === `members-${g.id}`" @click="saveMembership(g)">保存绑定</button></header>
                <label v-for="a in eligibleAccounts(g)" :key="a.id" class="account-option">
                  <input v-model="selectedAccounts[g.id]" type="checkbox" :value="a.id" />
                  <span><b>{{ a.name || `Account #${a.id}` }}</b><small>{{ a.type }} · #{{ a.id }}</small></span>
                  <em :class="{ ok: a.status === 'active' && a.schedulable !== false }">{{ a.status === 'active' && a.schedulable !== false ? '可调度' : '不可调度' }}</em>
                </label>
                <p v-if="!eligibleAccounts(g).length" class="empty">当前没有同平台上游账号。套餐和调度分组仍会保留，接入新上游后再绑定即可。</p>
              </section>

              <section class="subpanel models-panel">
                <header><div><strong>模型白名单</strong><small>真实用于 /v1/models 与目录发布</small></div><button @click="refreshCandidates(g)">刷新候选</button></header>
                <div class="model-summary"><b>{{ configuredModels(g).length || modelCache[g.id]?.length || 0 }}</b><span>个已配置模型</span><em>{{ findCatalogChannel(g) ? '目录已同步' : '尚未发布' }}</em></div>
                <div class="chips"><span v-for="m in (configuredModels(g).length ? configuredModels(g) : modelCache[g.id] || []).slice(0, 18)" :key="m">{{ m }}</span><span v-if="(configuredModels(g).length ? configuredModels(g) : modelCache[g.id] || []).length > 18">+{{ (configuredModels(g).length ? configuredModels(g) : modelCache[g.id] || []).length - 18 }}</span></div>
                <p class="model-hint">“同步模型并发布”会从已绑定 API Key 上游读取真实模型，安全合并保存 model_mapping，再更新分组模型列表与模型广场渠道。</p>
              </section>
            </div>
          </div>
        </article>
        <div v-if="!visibleGroups.length && !loading" class="empty big">没有符合条件的分组。</div>
      </div>
    </section>

    <div v-if="showEditor" class="overlay" @click.self="showEditor = false">
      <form class="dialog" @submit.prevent="saveGroup">
        <header><div><span>GROUP CONFIGURATION</span><h2>{{ editingId ? '编辑分组' : '新建分组' }}</h2></div><button type="button" @click="showEditor = false">×</button></header>
        <div class="form-grid"><label class="wide"><span>分组名称</span><input v-model="form.name" required placeholder="例如 swiftapi-default" /></label><label><span>平台协议</span><select v-model="form.platform"><option value="openai">OpenAI Compatible</option><option value="anthropic">Anthropic</option><option value="gemini">Gemini</option><option value="antigravity">Antigravity</option><option value="grok">xAI / Grok</option><option value="composite">Composite</option></select></label><label><span>结算倍率</span><input v-model.number="form.rate_multiplier" type="number" min="0" step="0.01" /></label><label v-if="editingId"><span>状态</span><select v-model="form.status"><option value="active">启用</option><option value="inactive">停用</option></select></label><label class="wide"><span>说明</span><textarea v-model="form.description" rows="3" placeholder="说明此分组对应的上游与用途"></textarea></label></div>
        <footer><button type="button" class="ghost" @click="showEditor = false">取消</button><button class="primary" :disabled="busy === 'group'">{{ busy === 'group' ? '保存中…' : '保存分组' }}</button></footer>
      </form>
    </div>
  </section>
</template>

<style scoped>
.groups-page{--bg:#0b0d11;--panel:#101217;--panel2:#0d0f13;--border:#252931;--border2:#343943;--text:#f4f6f8;--soft:#c6ccd3;--muted:#77818c;--green:#43cd98;--blue:#78aee8;width:100%;color:var(--text);font-size:14px}.page-head{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;padding:2px 0 24px}.kicker,.dialog header span{color:#66717d;font-size:.66rem;font-weight:700;letter-spacing:.12em}.page-head h1{margin:8px 0 0;font-size:2.05rem;line-height:1;font-weight:700;letter-spacing:-.045em}.page-head p{margin:11px 0 0;color:#858e99;font-size:.84rem}.head-actions,.detail-actions{display:flex;gap:8px}button{font:inherit}.ghost,.primary,.detail-actions button,.subpanel header button{height:40px;padding:0 14px;border:1px solid var(--border2);border-radius:8px;background:#13161b;color:#cbd1d7;cursor:pointer}.primary,.detail-actions .accent{border-color:#e2e6ea;background:#f2f4f6;color:#111318;font-weight:680}.primary:hover,.detail-actions .accent:hover{background:#fff}.ghost:hover,.detail-actions button:hover,.subpanel header button:hover{border-color:#4a515c;color:#fff}.overview{display:grid;grid-template-columns:repeat(4,1fr);border:1px solid var(--border);border-radius:12px;background:linear-gradient(180deg,#111318,#0e1014);overflow:hidden}.overview article{min-height:112px;padding:20px 22px;display:flex;flex-direction:column;justify-content:center;position:relative}.overview article+article:before{content:"";position:absolute;left:0;top:20px;bottom:20px;width:1px;background:#272b32}.overview span{color:#7a838e;font-size:.7rem}.overview strong{margin-top:8px;font-size:1.8rem;line-height:1;font-weight:700}.overview small{margin-top:8px;color:#626b76;font-size:.65rem}.alert{margin-top:12px;padding:10px 13px;border-radius:8px;font-size:.74rem}.alert.danger{border:1px solid rgba(225,108,115,.28);background:rgba(225,108,115,.07);color:#e7a2a7}.alert.success{border:1px solid rgba(67,205,152,.22);background:rgba(67,205,152,.06);color:#91dabd}.panel{margin-top:14px;border:1px solid var(--border);border-radius:12px;background:#0f1115;overflow:hidden}.toolbar{min-height:68px;padding:12px 14px 12px 18px;display:flex;align-items:center;justify-content:space-between;gap:16px;border-bottom:1px solid var(--border);background:#101217}.toolbar>div:first-child{display:flex;align-items:baseline;gap:9px}.toolbar strong{font-size:.9rem}.toolbar span{color:#66717c;font-size:.67rem}.filters{display:flex;gap:7px}.filters input,.filters select,.form-grid input,.form-grid select,.form-grid textarea{border:1px solid #2b3038;border-radius:8px;background:#0b0d11;color:#e8ebee;outline:none}.filters input{width:280px;height:40px;padding:0 12px}.filters select{height:40px;padding:0 30px 0 11px}.group-list{min-height:300px}.group-card+.group-card{border-top:1px solid #20242a}.group-card.open{background:#0d0f13}.group-main{width:100%;min-height:72px;padding:0 18px;border:0;background:transparent;color:inherit;display:grid;grid-template-columns:38px minmax(230px,1.4fr) 105px repeat(3,85px) 92px 24px;align-items:center;gap:13px;text-align:left;cursor:pointer}.group-main:hover{background:#14171c}.mark{width:34px;height:34px;border:1px solid #343943;border-radius:10px;background:#171a1f;display:grid;place-items:center;font:700 .66rem ui-monospace,SFMono-Regular,monospace;color:#cfd5db}.identity{min-width:0;display:flex;flex-direction:column}.identity strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.identity small{margin-top:5px;color:#69737e;font-size:.65rem}.state,.publish{display:inline-flex;align-items:center;gap:6px;width:max-content;color:#b8bfc7;font-size:.67rem}.state i,.publish i{width:6px;height:6px;border-radius:50%;background:#727b86}.state.active{color:#8fd9bb}.state.active i,.publish.on i{background:var(--green)}.publish.on{color:#8fd9bb}.metric{display:flex;flex-direction:column;gap:5px}.metric small{color:#68727d;font-size:.62rem}.metric b{font-size:.75rem}.chev{justify-self:end;color:#6b7580;font-size:1.3rem;transform:rotate(0);transition:.15s}.open .chev{transform:rotate(90deg)}.detail{padding:0 18px 18px 69px}.detail-top{padding:15px 0;display:flex;align-items:center;justify-content:space-between;gap:16px;border-top:1px solid #20242a}.detail-top>div:first-child{display:flex;flex-direction:column;gap:5px}.detail-top span{color:#68727d;font-size:.63rem}.detail-top strong{color:#cbd1d7;font-size:.74rem;font-weight:560}.detail-actions button{height:34px;padding:0 11px;font-size:.67rem}.detail-grid{display:grid;grid-template-columns:1fr 1.25fr;gap:12px}.subpanel{border:1px solid #242830;border-radius:10px;background:#0a0c10;overflow:hidden}.subpanel header{min-height:54px;padding:0 13px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #22262c}.subpanel header>div{display:flex;flex-direction:column;gap:3px}.subpanel header strong{font-size:.76rem}.subpanel header small{color:#68727d;font-size:.61rem}.subpanel header button{height:30px;padding:0 9px;font-size:.63rem}.account-option{min-height:48px;padding:0 13px;display:grid;grid-template-columns:18px 1fr auto;align-items:center;gap:9px;border-bottom:1px solid #1c2026;cursor:pointer}.account-option:last-child{border-bottom:0}.account-option input{accent-color:#7e8791}.account-option span{display:flex;flex-direction:column;gap:3px}.account-option b{font-size:.7rem}.account-option small{color:#68727d;font-size:.6rem}.account-option em{font-style:normal;color:#9c7e5a;font-size:.61rem}.account-option em.ok{color:#83cdb0}.model-summary{padding:15px 14px;display:flex;align-items:baseline;gap:7px}.model-summary b{font-size:1.5rem}.model-summary span{color:#707a85;font-size:.67rem}.model-summary em{margin-left:auto;font-style:normal;color:#6f7984;font-size:.61rem}.chips{padding:0 14px 8px;display:flex;flex-wrap:wrap;gap:6px}.chips span{padding:5px 7px;border:1px solid #293039;border-radius:6px;background:#11151a;color:#aab3bd;font:560 .59rem ui-monospace,SFMono-Regular,monospace}.model-hint{margin:5px 14px 14px;color:#65707b;font-size:.62rem;line-height:1.55}.empty{padding:18px;color:#68727d;font-size:.68rem}.empty.big{min-height:240px;display:grid;place-items:center}.overlay{position:fixed;inset:0;z-index:80;padding:24px;background:rgba(0,0,0,.68);backdrop-filter:blur(8px);display:grid;place-items:center}.dialog{width:min(620px,100%);border:1px solid #333842;border-radius:14px;background:#111318;box-shadow:0 28px 90px rgba(0,0,0,.55);overflow:hidden}.dialog header{padding:20px 22px;display:flex;justify-content:space-between;border-bottom:1px solid #262a31}.dialog h2{margin:6px 0 0;font-size:1.25rem}.dialog header button{width:34px;height:34px;border:1px solid #30353d;border-radius:8px;background:#171a1f;color:#98a1ac;cursor:pointer}.form-grid{padding:20px 22px;display:grid;grid-template-columns:1fr 1fr;gap:14px}.form-grid label{display:flex;flex-direction:column;gap:7px}.form-grid label.wide{grid-column:1/-1}.form-grid label>span{color:#858e98;font-size:.68rem}.form-grid input,.form-grid select{height:42px;padding:0 11px}.form-grid textarea{padding:10px 11px;resize:vertical}.dialog footer{padding:14px 22px;border-top:1px solid #262a31;display:flex;justify-content:flex-end;gap:8px}@media(max-width:1050px){.overview{grid-template-columns:1fr 1fr}.overview article:nth-child(3):before{display:none}.overview article:nth-child(n+3){border-top:1px solid #272b32}.group-main{grid-template-columns:38px 1fr 90px 70px 70px 24px}.group-main>.metric:nth-of-type(3),.publish{display:none}.detail{padding-left:18px}.detail-grid{grid-template-columns:1fr}}@media(max-width:680px){.page-head{flex-direction:column}.head-actions{width:100%}.head-actions button{flex:1}.overview{grid-template-columns:1fr 1fr}.toolbar{align-items:flex-start;flex-direction:column}.filters{width:100%;flex-direction:column}.filters input,.filters select{width:100%}.group-main{grid-template-columns:38px 1fr 24px}.group-main>.state,.group-main>.metric,.group-main>.publish{display:none}.detail-top{align-items:flex-start;flex-direction:column}.detail-actions{width:100%;flex-wrap:wrap}.detail-actions button{flex:1}.form-grid{grid-template-columns:1fr}.form-grid label.wide{grid-column:auto}}
</style>
