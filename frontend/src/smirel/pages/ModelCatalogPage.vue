<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, getErrorMessage } from '../core/api'
import { interfacePreferences } from '../core/preferences'

type Pricing = {
  billing_mode?: string
  input_price?: number | null
  output_price?: number | null
  cache_write_price?: number | null
  cache_read_price?: number | null
  per_request_price?: number | null
}

type OfficialPricing = {
  input_price?: number | null
  output_price?: number | null
  cache_write_price?: number | null
  cache_read_price?: number | null
}

type PlazaModel = {
  name: string
  platform: string
  mapped_model?: string | null
  pricing?: Pricing | null
  official_pricing?: OfficialPricing | null
}

type PlazaGroup = {
  id: number
  name: string
  description?: string
  platform: string
  subscription_type?: string
  rate_multiplier: number
  user_rate_multiplier?: number | null
  is_exclusive?: boolean
  models: PlazaModel[]
}

type PlazaResponse = { description?: string; groups?: PlazaGroup[] }
type Offer = { group: PlazaGroup; model: PlazaModel }
type PriceSource = 'channel' | 'official' | 'none'
type SortKey = 'recommended' | 'name' | 'price' | 'routes'

type ProviderInfo = {
  key: string
  name: string
  mark: string
}

type CatalogModel = {
  id: string
  provider: string
  providerKey: string
  mark: string
  offers: Offer[]
  bestRate: number
  inputPerM: number | null
  outputPerM: number | null
  cacheReadPerM: number | null
  cacheWritePerM: number | null
  priceSource: PriceSource
}

const isZh = computed(() => interfacePreferences.locale === 'zh-CN')
const loading = ref(false)
const error = ref('')
const description = ref('')
const groups = ref<PlazaGroup[]>([])
const search = ref('')
const provider = ref('all')
const groupId = ref<number | 'all'>('all')
const sortBy = ref<SortKey>('recommended')
const copied = ref('')

function rate(group: PlazaGroup) {
  return Number(group.user_rate_multiplier ?? group.rate_multiplier ?? 1)
}

function providerFromPlatform(platform: string): ProviderInfo {
  const value = String(platform || '').trim().toLowerCase()
  if (value === 'openai') return { key: 'openai', name: 'OpenAI', mark: 'O' }
  if (value === 'anthropic' || value === 'claude') return { key: 'anthropic', name: 'Anthropic', mark: 'A' }
  if (value === 'gemini' || value === 'google') return { key: 'google', name: 'Google', mark: 'G' }
  if (value === 'grok' || value === 'xai') return { key: 'xai', name: 'xAI', mark: 'X' }
  if (value === 'antigravity') return { key: 'antigravity', name: 'Antigravity', mark: 'AG' }
  if (value === 'composite') return { key: 'composite', name: 'Composite', mark: 'C' }
  return { key: value || 'other', name: value ? value.charAt(0).toUpperCase() + value.slice(1) : 'Other', mark: 'AI' }
}

function family(model: string, offers: Offer[]): ProviderInfo {
  const id = model.toLowerCase()
  if (id.startsWith('claude')) return { key: 'anthropic', name: 'Anthropic', mark: 'A' }
  if (id.startsWith('gemini')) return { key: 'google', name: 'Google', mark: 'G' }
  if (id.startsWith('grok')) return { key: 'xai', name: 'xAI', mark: 'X' }
  if (id.startsWith('deepseek')) return { key: 'deepseek', name: 'DeepSeek', mark: 'D' }
  if (id.startsWith('qwen')) return { key: 'qwen', name: 'Qwen', mark: 'Q' }
  if (id.startsWith('glm')) return { key: 'zhipu', name: 'GLM', mark: 'Z' }
  if (id.startsWith('llama')) return { key: 'meta', name: 'Meta', mark: 'M' }
  if (id.startsWith('kimi') || id.startsWith('moonshot')) return { key: 'moonshot', name: 'Kimi', mark: 'K' }
  if (id.startsWith('minimax')) return { key: 'minimax', name: 'MiniMax', mark: 'M' }
  if (id.startsWith('mimo')) return { key: 'xiaomimimo', name: 'MiMo', mark: 'MI' }
  if (id === 'hy3' || id.startsWith('hy4') || id.startsWith('hunyuan')) return { key: 'hunyuan', name: 'Hunyuan', mark: 'H' }
  if (id.startsWith('mistral') || id.startsWith('codestral')) return { key: 'mistral', name: 'Mistral', mark: 'M' }
  if (id.startsWith('gpt') || id.startsWith('o1') || id.startsWith('o3') || id.startsWith('o4')) {
    return { key: 'openai', name: 'OpenAI', mark: 'O' }
  }

  const modelPlatform = offers
    .map((offer) => String(offer.model.platform || '').toLowerCase())
    .find((value) => value && value !== 'composite')
  if (modelPlatform) return providerFromPlatform(modelPlatform)

  const groupPlatform = offers
    .map((offer) => String(offer.group.platform || '').toLowerCase())
    .find((value) => value && value !== 'composite')
  if (groupPlatform) return providerFromPlatform(groupPlatform)

  return { key: 'other', name: 'Other', mark: 'AI' }
}

function firstPrice(offers: Offer[], field: keyof Pricing): { value: number | null; source: PriceSource; rate: number } {
  const sorted = [...offers].sort((a, b) => rate(a.group) - rate(b.group))
  for (const offer of sorted) {
    const value = offer.model.pricing?.[field]
    if (typeof value === 'number') return { value, source: 'channel', rate: rate(offer.group) }
  }

  const officialField = field as keyof OfficialPricing
  for (const offer of sorted) {
    const value = offer.model.official_pricing?.[officialField]
    if (typeof value === 'number') return { value, source: 'official', rate: rate(offer.group) }
  }

  return { value: null, source: 'none', rate: sorted[0] ? rate(sorted[0].group) : 1 }
}

const groupOptions = computed(() =>
  [...groups.value].sort((a, b) => rate(a) - rate(b) || a.name.localeCompare(b.name))
)

const sourceGroups = computed(() =>
  groupId.value === 'all' ? groups.value : groups.value.filter((group) => group.id === groupId.value)
)

const models = computed<CatalogModel[]>(() => {
  const map = new Map<string, Offer[]>()
  for (const group of sourceGroups.value) {
    for (const model of group.models || []) {
      const id = String(model.name || '').trim()
      if (!id) continue
      const list = map.get(id) || []
      list.push({ group, model })
      map.set(id, list)
    }
  }

  return [...map.entries()].map(([id, offers]) => {
    const providerInfo = family(id, offers)
    const input = firstPrice(offers, 'input_price')
    const output = firstPrice(offers, 'output_price')
    const cacheRead = firstPrice(offers, 'cache_read_price')
    const cacheWrite = firstPrice(offers, 'cache_write_price')
    const bestRate = Math.min(...offers.map((offer) => rate(offer.group)))
    const source = input.source !== 'none' ? input.source : output.source
    const perMillion = (item: { value: number | null; rate: number }) =>
      item.value == null ? null : item.value * 1_000_000 * item.rate

    return {
      id,
      provider: providerInfo.name,
      providerKey: providerInfo.key,
      mark: providerInfo.mark,
      offers,
      bestRate,
      inputPerM: perMillion(input),
      outputPerM: perMillion(output),
      cacheReadPerM: perMillion(cacheRead),
      cacheWritePerM: perMillion(cacheWrite),
      priceSource: source,
    }
  })
})

const providerOrder = ['openai', 'anthropic', 'google', 'xai', 'deepseek', 'qwen', 'zhipu', 'moonshot', 'minimax', 'xiaomimimo', 'hunyuan', 'mistral', 'meta', 'antigravity']
const providers = computed(() => {
  const map = new Map<string, { key: string; name: string; count: number }>()
  for (const model of models.value) {
    const existing = map.get(model.providerKey)
    if (existing) existing.count += 1
    else map.set(model.providerKey, { key: model.providerKey, name: model.provider, count: 1 })
  }

  return [...map.values()].sort((a, b) => {
    const ai = providerOrder.indexOf(a.key)
    const bi = providerOrder.indexOf(b.key)
    if (ai !== -1 || bi !== -1) {
      if (ai === -1) return 1
      if (bi === -1) return -1
      if (ai !== bi) return ai - bi
    }
    return b.count - a.count || a.name.localeCompare(b.name)
  })
})

const filtered = computed(() => {
  const query = search.value.trim().toLowerCase()
  const items = models.value.filter((model) => {
    if (provider.value !== 'all' && model.providerKey !== provider.value) return false
    if (!query) return true

    const routeText = model.offers
      .map((offer) => `${offer.group.name} ${offer.group.platform} ${offer.model.mapped_model || ''}`)
      .join(' ')
      .toLowerCase()
    return `${model.id} ${model.provider} ${routeText}`.toLowerCase().includes(query)
  })

  return [...items].sort((a, b) => {
    if (sortBy.value === 'name') return a.id.localeCompare(b.id)
    if (sortBy.value === 'routes') return b.offers.length - a.offers.length || a.id.localeCompare(b.id)
    if (sortBy.value === 'price') {
      return (a.inputPerM ?? Number.MAX_SAFE_INTEGER) - (b.inputPerM ?? Number.MAX_SAFE_INTEGER) || a.id.localeCompare(b.id)
    }
    return b.offers.length - a.offers.length || a.bestRate - b.bestRate || a.id.localeCompare(b.id)
  })
})

const routeCount = computed(() => models.value.reduce((count, model) => count + model.offers.length, 0))
const activeGroup = computed(() =>
  groupId.value === 'all' ? null : groups.value.find((group) => group.id === groupId.value) || null
)

async function loadCatalog() {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get<PlazaResponse>('/model-plaza')
    description.value = String(response.data?.description || '')
    groups.value = Array.isArray(response.data?.groups) ? response.data.groups : []
    if (groupId.value !== 'all' && !groups.value.some((group) => group.id === groupId.value)) groupId.value = 'all'
  } catch (e) {
    groups.value = []
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function money(value: number | null) {
  if (value == null) return '—'
  if (value < 0.01) return `$${value.toFixed(4)}`
  if (value < 1) return `$${value.toFixed(3)}`
  return `$${value.toFixed(2)}`
}

function protocol(platform: string) {
  return (
    {
      openai: 'OpenAI Compatible',
      anthropic: 'Messages API',
      gemini: 'Gemini API',
      google: 'Gemini API',
      antigravity: 'Antigravity',
      grok: 'OpenAI Compatible',
      xai: 'OpenAI Compatible',
      composite: 'Composite',
    } as Record<string, string>
  )[String(platform || '').toLowerCase()] || platform || 'Compatible API'
}

function providerInfoForGroup(group: PlazaGroup): ProviderInfo {
  if (String(group.name || '').toLowerCase().startsWith('smirel')) {
    return { key: 'composite', name: 'Smirel', mark: 'S' }
  }
  const firstModel = group.models?.[0]
  if (firstModel) return family(firstModel.name, [{ group, model: firstModel }])
  return providerFromPlatform(group.platform)
}

function providerKeyForGroup(group: PlazaGroup) {
  return providerInfoForGroup(group).key
}

function providerMarkForGroup(group: PlazaGroup) {
  return providerInfoForGroup(group).mark
}

function protocols(model: CatalogModel) {
  return [...new Set(model.offers.map((offer) => protocol(offer.group.platform)).filter(Boolean))]
}

function mappedModels(model: CatalogModel) {
  return [
    ...new Set(
      model.offers
        .map((offer) => String(offer.model.mapped_model || '').trim())
        .filter((mapped) => mapped && mapped !== model.id)
    ),
  ]
}

function priceSourceLabel(source: PriceSource) {
  if (source === 'channel') return isZh.value ? '渠道配置' : 'Channel pricing'
  if (source === 'official') return isZh.value ? '官方参考价 × 分组倍率' : 'Official reference × group rate'
  return isZh.value ? '未配置' : 'Not configured'
}

function reset() {
  search.value = ''
  provider.value = 'all'
  groupId.value = 'all'
  sortBy.value = 'recommended'
}

function selectGroup(id: number | 'all') {
  groupId.value = id
  provider.value = 'all'
}

async function copyId(id: string) {
  if (!navigator.clipboard) return
  await navigator.clipboard.writeText(id)
  copied.value = id
  window.setTimeout(() => {
    if (copied.value === id) copied.value = ''
  }, 1200)
}

onMounted(() => void loadCatalog())
</script>

<template>
  <section class="workspace-page model-market-page">
    <header class="model-market-heading">
      <div class="market-heading-copy">
        <span class="market-kicker">MODEL MARKETPLACE</span>
        <div class="market-title-line">
          <h1>{{ isZh ? '模型广场' : 'Model Marketplace' }}</h1>
          <span class="market-count">{{ models.length }} {{ isZh ? '个模型' : 'models' }}</span>
        </div>
        <p>{{ description || (isZh ? '实时浏览模型、分组、价格与可用路由。' : 'Browse live models, groups, pricing and available routes.') }}</p>
      </div>

      <div class="market-heading-actions">
        <label class="market-search">
          <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="5.5" /><path d="m13 13 4 4" /></svg>
          <input v-model="search" type="search" :placeholder="isZh ? '搜索模型、品牌、分组或映射 ID' : 'Search model, provider, group or mapped ID'" />
        </label>
        <label class="market-sort">
          <span>{{ isZh ? '排序' : 'Sort' }}</span>
          <select v-model="sortBy">
            <option value="recommended">{{ isZh ? '推荐' : 'Recommended' }}</option>
            <option value="name">{{ isZh ? '名称' : 'Name' }}</option>
            <option value="price">{{ isZh ? '输入价格' : 'Input price' }}</option>
            <option value="routes">{{ isZh ? '路由数量' : 'Routes' }}</option>
          </select>
        </label>
      </div>
    </header>

    <div v-if="error" class="market-error">
      <strong>{{ isZh ? '模型目录暂不可用' : 'Catalog unavailable' }}</strong>
      <span>{{ error }}</span>
      <button type="button" @click="loadCatalog">{{ isZh ? '重新加载' : 'Retry' }}</button>
    </div>

    <template v-else>
      <section class="market-filter-shell">
        <div class="provider-filter" role="tablist" :aria-label="isZh ? '模型服务商' : 'Providers'">
          <button type="button" data-provider="all" :class="{ active: provider === 'all' }" @click="provider = 'all'">
            <span>{{ isZh ? '全部模型' : 'All models' }}</span>
            <b>{{ models.length }}</b>
          </button>
          <button
            v-for="item in providers"
            :key="item.key"
            type="button"
            :data-provider="item.key"
            :class="{ active: provider === item.key }"
            @click="provider = item.key"
          >
            <span>{{ item.name }}</span>
            <b>{{ item.count }}</b>
          </button>
        </div>

        <div class="group-filter-row">
          <div class="group-filter-label">
            <span>{{ isZh ? '分组' : 'Groups' }}</span>
            <small>{{ isZh ? '按真实路由分组筛选' : 'Filter by live route group' }}</small>
          </div>
          <div class="group-filter-list">
            <button type="button" :class="{ active: groupId === 'all' }" @click="selectGroup('all')">
              <span class="group-all-icon" aria-hidden="true"></span>
              <span>{{ isZh ? '全部分组' : 'All groups' }}</span>
              <b>{{ groups.length }}</b>
            </button>
            <button
              v-for="group in groupOptions"
              :key="group.id"
              type="button"
              :class="{ active: groupId === group.id }"
              @click="selectGroup(group.id)"
            >
              <i class="provider-mini-mark" :data-provider="providerKeyForGroup(group)">{{ providerMarkForGroup(group) }}</i>
              <span>{{ group.name }}</span>
              <em>{{ rate(group).toFixed(2) }}×</em>
            </button>
          </div>
        </div>

        <div class="market-filter-meta">
          <div class="group-selection-summary">
            <span>{{ activeGroup ? (isZh ? '当前分组' : 'Active group') : (isZh ? '分组视图' : 'Group view') }}</span>
            <strong>{{ activeGroup?.name || (isZh ? '全部已发布分组' : 'All published groups') }}</strong>
            <small v-if="activeGroup">#{{ activeGroup.id }} · {{ protocol(activeGroup.platform) }} · {{ rate(activeGroup).toFixed(2) }}×</small>
          </div>
          <div class="catalog-state">
            <span><i></i>{{ loading ? (isZh ? '同步中' : 'Syncing') : (isZh ? '服务端实时数据' : 'Live server data') }}</span>
            <small>{{ isZh ? '模型、价格与路由均来自当前 TermRelay 配置。' : 'Models, prices and routes come from the current TermRelay configuration.' }}</small>
          </div>
        </div>
      </section>

      <section class="market-result-head">
        <div>
          <strong>{{ filtered.length }} {{ isZh ? '个模型' : 'models' }}</strong>
          <span>·</span>
          <span>{{ sourceGroups.length }} {{ isZh ? '个分组' : 'groups' }}</span>
          <span>·</span>
          <span>{{ routeCount }} {{ isZh ? '条模型路由' : 'model routes' }}</span>
        </div>
        <div>
          <span>Base URL</span>
          <code>https://api.smirel.com/v1</code>
          <button class="catalog-refresh" type="button" :disabled="loading" @click="loadCatalog">{{ loading ? '…' : '↻' }}</button>
        </div>
      </section>

      <div v-if="filtered.length" class="model-market-grid">
        <article v-for="(model, index) in filtered" :key="model.id" class="model-market-card">
          <header class="model-card-head">
            <div class="model-identity">
              <span class="provider-mark" :data-provider="model.providerKey">{{ model.mark }}</span>
              <div>
                <h2>{{ model.id }}</h2>
                <div class="model-tags">
                  <span>{{ model.provider }}</span>
                  <span>{{ model.bestRate.toFixed(2) }}×</span>
                  <span>{{ model.offers.length }} {{ isZh ? '条路由' : 'routes' }}</span>
                </div>
              </div>
            </div>
            <span class="model-rank">#{{ String(index + 1).padStart(2, '0') }}</span>
          </header>

          <section class="model-price-grid">
            <div>
              <span>{{ isZh ? '输入' : 'Input' }}</span>
              <strong>{{ money(model.inputPerM) }}<small v-if="model.inputPerM != null"> / M tokens</small></strong>
            </div>
            <div>
              <span>{{ isZh ? '输出' : 'Output' }}</span>
              <strong>{{ money(model.outputPerM) }}<small v-if="model.outputPerM != null"> / M tokens</small></strong>
            </div>
            <div>
              <span>{{ isZh ? '缓存写入' : 'Cache write' }}</span>
              <strong>{{ money(model.cacheWritePerM) }}<small v-if="model.cacheWritePerM != null"> / M tokens</small></strong>
            </div>
            <div>
              <span>{{ isZh ? '缓存读取' : 'Cache read' }}</span>
              <strong>{{ money(model.cacheReadPerM) }}<small v-if="model.cacheReadPerM != null"> / M tokens</small></strong>
            </div>
          </section>

          <section class="model-specs">
            <div><span>{{ isZh ? '可用分组' : 'Groups' }}</span><strong>{{ model.offers.length }}</strong></div>
            <div><span>{{ isZh ? '最低倍率' : 'Best rate' }}</span><strong>{{ model.bestRate.toFixed(2) }}×</strong></div>
            <div><span>{{ isZh ? '价格来源' : 'Price source' }}</span><strong>{{ priceSourceLabel(model.priceSource) }}</strong></div>
          </section>

          <section class="model-access-block">
            <div class="model-id-row">
              <span>{{ isZh ? '模型 ID' : 'Model ID' }}</span>
              <code>{{ model.id }}</code>
              <button type="button" @click="copyId(model.id)">{{ copied === model.id ? (isZh ? '已复制' : 'Copied') : (isZh ? '复制' : 'Copy') }}</button>
            </div>
            <div class="protocol-list">
              <span>{{ isZh ? '兼容接口' : 'APIs' }}</span>
              <div><b v-for="item in protocols(model)" :key="item">{{ item }}</b></div>
            </div>
            <div v-if="mappedModels(model).length" class="mapped-model-list">
              <span>{{ isZh ? '上游映射' : 'Mapped IDs' }}</span>
              <div><code v-for="mapped in mappedModels(model)" :key="mapped">{{ mapped }}</code></div>
            </div>
          </section>

          <footer class="model-group-section">
            <div class="model-group-title">
              <span>{{ isZh ? '可用分组 / 路由' : 'Available groups / routes' }}</span>
              <strong>{{ model.offers.length }}</strong>
            </div>
            <div class="model-group-list">
              <div v-for="offer in model.offers" :key="`${offer.group.id}-${offer.model.platform}-${offer.model.mapped_model || offer.model.name}`" class="model-group-item">
                <i class="provider-mini-mark" :data-provider="providerKeyForGroup(offer.group)">{{ providerMarkForGroup(offer.group) }}</i>
                <span class="model-group-copy">
                  <b>{{ offer.group.name }}</b>
                  <small>#{{ offer.group.id }} · {{ protocol(offer.group.platform) }}<template v-if="offer.model.mapped_model && offer.model.mapped_model !== model.id"> · {{ offer.model.mapped_model }}</template></small>
                </span>
                <em>{{ rate(offer.group).toFixed(2) }}×</em>
              </div>
            </div>
          </footer>
        </article>
      </div>

      <section v-else-if="!loading" class="market-empty">
        <strong>{{ models.length ? (isZh ? '没有符合筛选条件的模型' : 'No matching models') : (isZh ? '模型目录还没有发布任何模型' : 'No models have been published yet') }}</strong>
        <span v-if="!models.length">{{ isZh ? '管理员可在「分组与模型」中同步真实上游模型并发布。' : 'An admin can sync and publish models from Groups & Models.' }}</span>
        <button v-if="models.length" type="button" @click="reset">{{ isZh ? '清除筛选' : 'Reset filters' }}</button>
      </section>

      <div v-if="loading && !models.length" class="market-loading" aria-label="loading">
        <i v-for="n in 6" :key="n"></i>
      </div>
    </template>
  </section>
</template>

<style scoped>
.market-error {
  min-height: 260px;
  border: 1px solid rgba(225, 108, 115, .25);
  border-radius: 10px;
  background: rgba(225, 108, 115, .05);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 9px;
  text-align: center;
}

.market-error strong { color: #f0c0c4; font-size: .86rem; }
.market-error span { max-width: 620px; color: #a8757a; font-size: .69rem; }
.market-error button,
.catalog-refresh {
  border: 1px solid #2d3239;
  border-radius: 6px;
  color: #b7bdc5;
  background: #16191d;
  cursor: pointer;
}
.market-error button { height: 32px; padding: 0 11px; font-size: .65rem; }
.catalog-refresh { width: 28px; height: 28px; margin-left: 2px; }
.catalog-refresh:disabled { opacity: .45; cursor: wait; }

.group-filter-row {
  min-height: 64px;
  padding: 9px 12px;
  border-top: 1px solid var(--ws-border);
  display: grid;
  grid-template-columns: 118px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}

.group-filter-label { display: flex; flex-direction: column; gap: 4px; }
.group-filter-label > span { color: #aeb5bd; font-size: .69rem; font-weight: 680; }
.group-filter-label > small { color: #5f6771; font-size: .57rem; }

.group-filter-list {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.group-filter-list button {
  min-width: max-content;
  height: 38px;
  padding: 0 10px;
  border: 1px solid #282d34;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #8f97a1;
  background: #0d0f13;
  cursor: pointer;
  transition: .15s ease;
}

.group-filter-list button:hover { border-color: #3a4048; color: #c7ccd2; background: #14171b; }
.group-filter-list button.active { border-color: #48505a; color: #eef1f4; background: #191c21; }
.group-filter-list button > span:not(.group-all-icon) { font-size: .67rem; font-weight: 620; }
.group-filter-list button > b,
.group-filter-list button > em {
  min-width: 24px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  display: inline-grid;
  place-items: center;
  color: #78818b;
  background: #090b0e;
  font-size: .57rem;
  font-style: normal;
  font-weight: 660;
}
.group-filter-list button.active > b,
.group-filter-list button.active > em { color: #b8c0c9; }
.group-all-icon {
  width: 15px;
  height: 15px;
  border-radius: 4px;
  background: linear-gradient(90deg, #728aa5 0 42%, transparent 42% 58%, #728aa5 58%), linear-gradient(#728aa5 0 42%, transparent 42% 58%, #728aa5 58%);
  opacity: .9;
}

.group-selection-summary {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.group-selection-summary > span { color: #656d77; font-size: .61rem; }
.group-selection-summary > strong { color: #aeb5bd; font-size: .67rem; font-weight: 650; }
.group-selection-summary > small { color: #5e6670; font-size: .59rem; }

.provider-mini-mark {
  width: 22px;
  height: 22px;
  flex: 0 0 22px;
  border: 1px solid #2b3037;
  border-radius: 6px;
  display: inline-grid;
  place-items: center;
  color: #aeb5bd;
  background: #15181d;
  font: 700 .47rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  font-style: normal;
}

.mapped-model-list {
  min-height: 48px;
  padding: 8px 11px;
  border-top: 1px solid #1f2329;
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  align-items: center;
  gap: 9px;
}
.mapped-model-list > span { color: #626a74; font-size: .61rem; }
.mapped-model-list > div { min-width: 0; display: flex; flex-wrap: wrap; gap: 5px; }
.mapped-model-list code {
  max-width: 100%;
  padding: 4px 7px;
  border-radius: 5px;
  overflow: hidden;
  color: #8eaed2;
  background: #15181d;
  font: .57rem/1.25 ui-monospace, SFMono-Regular, Menlo, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-group-section { padding: 12px 14px 14px; border-top: 1px solid #22262c; }
.model-group-title { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.model-group-title > span { color: #68717b; font-size: .62rem; }
.model-group-title > strong {
  min-width: 24px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  display: grid;
  place-items: center;
  color: #9ba4ae;
  background: #0b0d10;
  font-size: .58rem;
}
.model-group-list { margin-top: 8px; display: grid; gap: 6px; }
.model-group-item {
  min-height: 45px;
  padding: 6px 8px;
  border: 1px solid #20242a;
  border-radius: 7px;
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  background: #0c0e12;
}
.model-group-copy { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.model-group-copy > b { overflow: hidden; color: #b7bec6; font-size: .65rem; font-weight: 640; text-overflow: ellipsis; white-space: nowrap; }
.model-group-copy > small { overflow: hidden; color: #59626c; font-size: .56rem; text-overflow: ellipsis; white-space: nowrap; }
.model-group-item > em { color: #91b6df; font-size: .61rem; font-style: normal; font-weight: 680; }

.market-loading { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.market-loading > i {
  min-height: 320px;
  border: 1px solid #23272e;
  border-radius: 12px;
  background: linear-gradient(100deg, #0f1115 20%, #15181d 40%, #0f1115 60%);
  background-size: 220% 100%;
  animation: market-shimmer 1.4s linear infinite;
}
@keyframes market-shimmer { to { background-position: -220% 0; } }

@media (max-width: 820px) {
  .group-filter-row { grid-template-columns: 1fr; }
  .group-filter-label { flex-direction: row; align-items: baseline; }
  .group-selection-summary { width: 100%; flex-wrap: wrap; }
}

@media (max-width: 640px) {
  .group-filter-list { width: 100%; }
  .market-loading { grid-template-columns: 1fr; }
  .model-group-item { grid-template-columns: 22px minmax(0, 1fr) auto; }
}
</style>
