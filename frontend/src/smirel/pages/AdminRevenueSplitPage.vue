<script setup lang="ts">
// Admin Revenue Split —— 共建者分成（多受益人固定比例分账）
//
// 页面职责：配比例 → 看汇总 → 查明细 → 生成结算单 → 登记线下打款。
// 本页只做记账与核对，不触发任何出金；打款在线下完成，这里回填凭证号。
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  revenueSplitApi,
  type RevenueSplitBeneficiarySummary,
  type RevenueSplitConfig,
  type RevenueSplitConfigUpdateRequest,
  type RevenueSplitEntry,
  type RevenueSplitEntryStatus,
  type RevenueSplitPreview,
  type RevenueSplitRuleInput,
  type RevenueSplitSettlement,
  type RevenueSplitSettlementStatus,
} from '../api/payment'
import { getErrorMessage } from '../core/api'

const { t } = useI18n()

type TabKey = 'rules' | 'summary' | 'entries' | 'settlements'

const tab = ref<TabKey>('rules')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const notice = ref('')

function flash(message: string): void {
  notice.value = message
  error.value = ''
  window.setTimeout(() => {
    if (notice.value === message) notice.value = ''
  }, 4000)
}

// ---------- 配置 ----------
const config = ref<RevenueSplitConfig>({ enabled: false, base_mode: 'gross_after_fee', channel_fee_percent: 1.6 })

async function saveConfig(): Promise<void> {
  saving.value = true
  error.value = ''
  try {
    const body: RevenueSplitConfigUpdateRequest = {
      enabled: config.value.enabled,
      base_mode: config.value.base_mode,
      channel_fee_percent: config.value.channel_fee_percent,
    }
    const r = await revenueSplitApi.updateConfig(body)
    config.value = r.config
    flash(t('payment.adminRevenueSplit.configSaved'))
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(): Promise<void> {
  config.value.enabled = !config.value.enabled
  await saveConfig()
}

// ---------- 比例规则 ----------
type RuleDraft = {
  beneficiary_user_id: number | null
  beneficiary_name: string
  ratio_percent: number
  enabled: boolean
  note: string
}

const ruleDrafts = ref<RuleDraft[]>([])
const rulesLoaded = ref(false)

function draftFromRule(r: {
  beneficiary_user_id: number
  beneficiary_name: string
  ratio_percent: number
  enabled: boolean
  note: string
}): RuleDraft {
  return {
    beneficiary_user_id: r.beneficiary_user_id,
    beneficiary_name: r.beneficiary_name || '',
    ratio_percent: Number(r.ratio_percent) || 0,
    enabled: r.enabled,
    note: r.note || '',
  }
}

const ratioTotal = computed(() =>
  ruleDrafts.value
    .filter((r) => r.enabled)
    .reduce((sum, r) => sum + (Number(r.ratio_percent) || 0), 0),
)

const ratioOverflow = computed(() => ratioTotal.value > 100 + 1e-9)

function addRule(): void {
  ruleDrafts.value.push({ beneficiary_user_id: null, beneficiary_name: '', ratio_percent: 0, enabled: true, note: '' })
}

function removeRule(index: number): void {
  ruleDrafts.value.splice(index, 1)
}

async function saveRules(): Promise<void> {
  error.value = ''
  if (ratioOverflow.value) {
    error.value = t('payment.adminRevenueSplit.ratioTotalOverflow')
    return
  }
  const payload: RevenueSplitRuleInput[] = []
  const seen = new Set<number>()
  for (const r of ruleDrafts.value) {
    const id = Number(r.beneficiary_user_id) || 0
    if (id <= 0) {
      error.value = t('payment.adminRevenueSplit.beneficiaryIdHint')
      return
    }
    if (seen.has(id)) {
      error.value = `受益人 #${id} 重复`
      return
    }
    seen.add(id)
    payload.push({
      beneficiary_user_id: id,
      beneficiary_name: r.beneficiary_name,
      ratio_percent: Number(r.ratio_percent) || 0,
      enabled: r.enabled,
      note: r.note,
    })
  }
  if (payload.length === 0) {
    error.value = t('payment.adminRevenueSplit.needOneRule')
    return
  }
  saving.value = true
  try {
    const r = await revenueSplitApi.replaceRules(payload)
    ruleDrafts.value = r.rules.map(draftFromRule)
    flash(t('payment.adminRevenueSplit.rulesSaved'))
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    saving.value = false
  }
}

// ---------- 试算 ----------
const previewAmount = ref(100)
const preview = ref<RevenueSplitPreview | null>(null)
const previewing = ref(false)

async function runPreview(): Promise<void> {
  if (!(previewAmount.value > 0)) return
  previewing.value = true
  error.value = ''
  try {
    preview.value = await revenueSplitApi.preview(Number(previewAmount.value))
  } catch (e) {
    preview.value = null
    error.value = getErrorMessage(e)
  } finally {
    previewing.value = false
  }
}

// ---------- 汇总 ----------
const summary = ref<RevenueSplitBeneficiarySummary[]>([])

async function loadSummary(): Promise<void> {
  const r = await revenueSplitApi.summary()
  summary.value = Array.isArray(r.beneficiaries) ? r.beneficiaries : []
}

// ---------- 明细 ----------
const entryFilters = reactive({
  beneficiary_user_id: '' as string,
  status: '' as RevenueSplitEntryStatus | '',
  keyword: '',
  start: '',
  end: '',
})
const entries = ref<RevenueSplitEntry[]>([])
const entryPage = ref(1)
const entryPages = ref(1)
const entryTotal = ref(0)
const entryPageSize = 20

async function loadEntries(resetPage = false): Promise<void> {
  if (resetPage) entryPage.value = 1
  const params: Record<string, unknown> = { page: entryPage.value, page_size: entryPageSize }
  const bid = Number(entryFilters.beneficiary_user_id)
  if (bid > 0) params.beneficiary_user_id = bid
  if (entryFilters.status) params.status = entryFilters.status
  if (entryFilters.keyword.trim()) params.keyword = entryFilters.keyword.trim()
  if (entryFilters.start) params.start = entryFilters.start
  if (entryFilters.end) params.end = entryFilters.end
  const r = await revenueSplitApi.listEntries(params)
  entries.value = r.items || []
  entryTotal.value = r.total || 0
  entryPages.value = r.pages || 1
  entryPage.value = r.page || 1
}

function resetEntryFilters(): void {
  entryFilters.beneficiary_user_id = ''
  entryFilters.status = ''
  entryFilters.keyword = ''
  entryFilters.start = ''
  entryFilters.end = ''
  void loadEntries(true)
}

// ---------- 结算单 ----------
const settlements = ref<RevenueSplitSettlement[]>([])
const settlementStatus = ref<RevenueSplitSettlementStatus | ''>('')
const settlementPage = ref(1)
const settlementPages = ref(1)
const settlementTotal = ref(0)
const settlementPageSize = 20

async function loadSettlements(resetPage = false): Promise<void> {
  if (resetPage) settlementPage.value = 1
  const params: Record<string, unknown> = { page: settlementPage.value, page_size: settlementPageSize }
  if (settlementStatus.value) params.status = settlementStatus.value
  const r = await revenueSplitApi.listSettlements(params)
  settlements.value = r.items || []
  settlementTotal.value = r.total || 0
  settlementPages.value = r.pages || 1
  settlementPage.value = r.page || 1
}

// 生成结算单
const createOpen = ref(false)
const createForm = reactive({ beneficiary_user_id: 0, beneficiary_label: '', period_start: '', period_end: '', method: '', note: '' })

function openCreate(item: RevenueSplitBeneficiarySummary): void {
  createForm.beneficiary_user_id = item.beneficiary_user_id
  createForm.beneficiary_label = item.beneficiary_name || `#${item.beneficiary_user_id}`
  createForm.period_start = ''
  createForm.period_end = ''
  createForm.method = ''
  createForm.note = ''
  createOpen.value = true
}

async function submitCreate(): Promise<void> {
  saving.value = true
  error.value = ''
  try {
    await revenueSplitApi.createSettlement({
      beneficiary_user_id: createForm.beneficiary_user_id,
      period_start: createForm.period_start || undefined,
      period_end: createForm.period_end || undefined,
      method: createForm.method || undefined,
      note: createForm.note || undefined,
    })
    createOpen.value = false
    flash(t('payment.adminRevenueSplit.settlementCreated'))
    await Promise.all([loadSummary(), loadSettlements(true)])
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    saving.value = false
  }
}

// 结算单详情
const detailOpen = ref(false)
const detailSettlement = ref<RevenueSplitSettlement | null>(null)
const detailEntries = ref<RevenueSplitEntry[]>([])
const detailLoading = ref(false)

async function openDetail(row: RevenueSplitSettlement): Promise<void> {
  detailOpen.value = true
  detailLoading.value = true
  detailSettlement.value = row
  detailEntries.value = []
  error.value = ''
  try {
    const r = await revenueSplitApi.getSettlement(row.id)
    detailSettlement.value = r.settlement
    detailEntries.value = r.entries || []
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    detailLoading.value = false
  }
}

function closeDetail(): void {
  detailOpen.value = false
}

// 标记已打款
const payOpen = ref(false)
const payForm = reactive({ id: 0, method: '', reference: '', note: '' })

function openPay(row: RevenueSplitSettlement): void {
  payForm.id = row.id
  payForm.method = row.method || ''
  payForm.reference = ''
  payForm.note = ''
  payOpen.value = true
}

async function submitPay(): Promise<void> {
  if (!payForm.reference.trim()) {
    error.value = t('payment.adminRevenueSplit.referenceRequired')
    return
  }
  saving.value = true
  error.value = ''
  try {
    await revenueSplitApi.markSettlementPaid(payForm.id, {
      method: payForm.method || undefined,
      reference: payForm.reference.trim(),
      note: payForm.note || undefined,
    })
    payOpen.value = false
    flash(t('payment.adminRevenueSplit.settlementPaid'))
    await Promise.all([loadSummary(), loadSettlements(), loadEntries()])
    if (detailOpen.value) await openDetail(detailSettlement.value as RevenueSplitSettlement)
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    saving.value = false
  }
}

async function cancelSettlement(row: RevenueSplitSettlement): Promise<void> {
  if (!window.confirm(t('payment.adminRevenueSplit.cancelConfirm'))) return
  try {
    await revenueSplitApi.cancelSettlement(row.id)
    flash(t('payment.adminRevenueSplit.settlementCancelled'))
    if (detailOpen.value) closeDetail()
    await Promise.all([loadSummary(), loadSettlements(), loadEntries()])
  } catch (e) {
    error.value = getErrorMessage(e)
  }
}

// ---------- 历史订单补计提 ----------
const repairOrderId = ref<number | null>(null)
const repairing = ref(false)

async function runRepair(): Promise<void> {
  const id = Number(repairOrderId.value)
  if (!(id > 0)) return
  repairing.value = true
  error.value = ''
  try {
    const r = await revenueSplitApi.accrueOrder(id)
    flash(r.entries > 0 ? t('payment.adminRevenueSplit.repairDone', { n: r.entries }) : t('payment.adminRevenueSplit.repairNone'))
    await Promise.all([loadSummary(), loadEntries(true)])
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    repairing.value = false
  }
}

// ---------- 格式化 ----------
function fmtMoney(value: number | null | undefined, currency = 'CNY'): string {
  const n = Number(value) || 0
  const symbol = currency === 'CNY' ? '¥' : `${currency} `
  return symbol + n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function fmtPercent(value: number | null | undefined): string {
  const n = Number(value) || 0
  return `${Number(n.toFixed(2))}%`
}

function fmtTime(raw: string | null | undefined): string {
  if (!raw) return '—'
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return String(raw)
  return d.toLocaleString('zh-CN', { hour12: false })
}

function periodLabel(row: RevenueSplitSettlement): string {
  if (!row.period_start && !row.period_end) return '—'
  return `${fmtTime(row.period_start)} ~ ${fmtTime(row.period_end)}`
}

function entryStatusLabel(status: RevenueSplitEntryStatus | string): string {
  if (status === 'settled') return t('payment.adminRevenueSplit.statusSettled')
  if (status === 'reversed') return t('payment.adminRevenueSplit.statusReversed')
  return t('payment.adminRevenueSplit.statusPending')
}

function settlementStatusLabel(status: RevenueSplitSettlementStatus | string): string {
  if (status === 'paid') return t('payment.adminRevenueSplit.statusPaid')
  if (status === 'cancelled') return t('payment.adminRevenueSplit.statusCancelled')
  return t('payment.adminRevenueSplit.statusDraft')
}

// ---------- 装载 ----------
async function loadAll(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const cfg = await revenueSplitApi.getConfig()
    config.value = cfg.config
    ruleDrafts.value = (cfg.rules || []).map(draftFromRule)
    rulesLoaded.value = true
    await Promise.all([loadSummary(), loadEntries(true), loadSettlements(true)])
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

async function switchTab(next: TabKey): Promise<void> {
  tab.value = next
  error.value = ''
  try {
    if (next === 'summary') await loadSummary()
    if (next === 'entries') await loadEntries()
    if (next === 'settlements') await loadSettlements()
  } catch (e) {
    error.value = getErrorMessage(e)
  }
}

onMounted(loadAll)
</script>

<template>
  <section class="revenue-split">
    <header class="rs-heading">
      <div>
        <h1>{{ t('payment.adminRevenueSplit.title') }}</h1>
        <p>{{ t('payment.adminRevenueSplit.description') }}</p>
      </div>
      <div class="rs-heading-actions">
        <span class="status-chip" :class="{ on: config.enabled }">
          {{ config.enabled ? t('payment.adminRevenueSplit.enabledOn') : t('payment.adminRevenueSplit.enabledOff') }}
        </span>
        <button type="button" class="primary-btn" :disabled="saving" @click="toggleEnabled">
          {{ config.enabled ? t('payment.adminRevenueSplit.enabledOff') : t('payment.adminRevenueSplit.enabledOn') }}
        </button>
      </div>
    </header>

    <p class="legal-banner">{{ t('payment.adminRevenueSplit.legalWarning') }}</p>
    <p v-if="error" class="error-banner">{{ error }}</p>
    <p v-if="notice" class="notice-banner">{{ notice }}</p>

    <nav class="rs-tabs">
      <button type="button" :class="{ active: tab === 'rules' }" @click="switchTab('rules')">
        {{ t('payment.adminRevenueSplit.tabRules') }}
      </button>
      <button type="button" :class="{ active: tab === 'summary' }" @click="switchTab('summary')">
        {{ t('payment.adminRevenueSplit.tabSummary') }}
      </button>
      <button type="button" :class="{ active: tab === 'entries' }" @click="switchTab('entries')">
        {{ t('payment.adminRevenueSplit.tabEntries') }}
      </button>
      <button type="button" :class="{ active: tab === 'settlements' }" @click="switchTab('settlements')">
        {{ t('payment.adminRevenueSplit.tabSettlements') }}
      </button>
    </nav>

    <div v-if="loading" class="empty-state">…</div>

    <!-- ---------- 规则与设置 ---------- -->
    <template v-else-if="tab === 'rules'">
      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.baseModeLabel') }}</h2>
        </div>
        <div class="form-grid">
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.enableLabel') }}</span>
            <select v-model="config.enabled">
              <option :value="true">{{ t('payment.adminRevenueSplit.enabledOn') }}</option>
              <option :value="false">{{ t('payment.adminRevenueSplit.enabledOff') }}</option>
            </select>
            <em class="hint">{{ t('payment.adminRevenueSplit.enableHint') }}</em>
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.baseModeLabel') }}</span>
            <select v-model="config.base_mode">
              <option value="gross_after_fee">{{ t('payment.adminRevenueSplit.baseModeGrossAfterFee') }}</option>
              <option value="gross">{{ t('payment.adminRevenueSplit.baseModeGross') }}</option>
            </select>
            <em class="hint">{{ t('payment.adminRevenueSplit.baseModeHint') }}</em>
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.channelFeeLabel') }}</span>
            <input v-model.number="config.channel_fee_percent" type="number" min="0" max="100" step="0.1" />
            <em class="hint">{{ t('payment.adminRevenueSplit.channelFeeHint') }}</em>
          </label>
        </div>
        <div class="card-actions">
          <button type="button" class="btn primary" :disabled="saving" @click="saveConfig">
            {{ saving ? t('payment.saving') : t('payment.adminRevenueSplit.saveConfig') }}
          </button>
        </div>
      </section>

      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.rulesTitle') }}</h2>
          <span class="ratio-total" :class="{ bad: ratioOverflow }">
            {{ t('payment.adminRevenueSplit.ratioTotal') }}：{{ fmtPercent(ratioTotal) }}
            · {{ ratioOverflow ? t('payment.adminRevenueSplit.ratioTotalOverflow') : t('payment.adminRevenueSplit.ratioTotalOk') }}
          </span>
        </div>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.rulesHint') }}</p>

        <table v-if="ruleDrafts.length" class="rs-table">
          <thead>
            <tr>
              <th>{{ t('payment.adminRevenueSplit.beneficiaryId') }}</th>
              <th>{{ t('payment.adminRevenueSplit.beneficiaryName') }}</th>
              <th>{{ t('payment.adminRevenueSplit.ratio') }}</th>
              <th>{{ t('payment.adminRevenueSplit.enabled') }}</th>
              <th>{{ t('payment.adminRevenueSplit.note') }}</th>
              <th class="action-cell">{{ t('payment.adminRevenueSplit.colAction') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in ruleDrafts" :key="index">
              <td><input v-model.number="row.beneficiary_user_id" type="number" min="1" class="cell-input narrow" /></td>
              <td><input v-model="row.beneficiary_name" type="text" class="cell-input" /></td>
              <td><input v-model.number="row.ratio_percent" type="number" min="0" max="100" step="0.01" class="cell-input narrow" /></td>
              <td><input v-model="row.enabled" type="checkbox" /></td>
              <td><input v-model="row.note" type="text" class="cell-input" /></td>
              <td class="action-cell">
                <button type="button" class="action danger" @click="removeRule(index)">
                  {{ t('payment.adminRevenueSplit.removeRule') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-state">{{ t('payment.adminRevenueSplit.emptyRules') }}</p>

        <div class="card-actions">
          <button type="button" class="btn ghost" @click="addRule">
            {{ t('payment.adminRevenueSplit.addRule') }}
          </button>
          <button type="button" class="btn primary" :disabled="saving || ratioOverflow" @click="saveRules">
            {{ saving ? t('payment.saving') : t('payment.adminRevenueSplit.saveRules') }}
          </button>
        </div>
      </section>

      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.previewTitle') }}</h2>
        </div>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.previewHint') }}</p>
        <div class="preview-row">
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.previewAmount') }}</span>
            <input v-model.number="previewAmount" type="number" min="0.01" step="0.01" />
          </label>
          <button type="button" class="btn primary" :disabled="previewing" @click="runPreview">
            {{ t('payment.adminRevenueSplit.previewRun') }}
          </button>
        </div>

        <div v-if="preview" class="preview-result">
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.previewGross') }}</span><strong>{{ fmtMoney(preview.gross_amount) }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.previewFee') }}</span><strong>-{{ fmtMoney(preview.channel_fee_amount) }}</strong></div>
          <div class="metric base"><span>{{ t('payment.adminRevenueSplit.previewBase') }}</span><strong>{{ fmtMoney(preview.base_amount) }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.previewAllocated') }}</span><strong>-{{ fmtMoney(preview.allocated_amount) }}</strong></div>
          <div class="metric" :class="{ bad: preview.platform_remainder <= 0 }">
            <span>{{ t('payment.adminRevenueSplit.previewRemainder') }}</span>
            <strong>{{ fmtMoney(preview.platform_remainder) }}</strong>
          </div>

          <table v-if="preview.items.length" class="rs-table compact">
            <thead>
              <tr>
                <th>{{ t('payment.adminRevenueSplit.previewBeneficiary') }}</th>
                <th>{{ t('payment.adminRevenueSplit.colRatio') }}</th>
                <th>{{ t('payment.adminRevenueSplit.previewSplit') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in preview.items" :key="item.beneficiary_user_id">
                <td>
                  <strong>{{ item.beneficiary_name || ('#' + item.beneficiary_user_id) }}</strong>
                  <div class="sub">#{{ item.beneficiary_user_id }}</div>
                </td>
                <td>{{ fmtPercent(item.ratio_percent) }}</td>
                <td>{{ fmtMoney(item.split_amount) }}</td>
              </tr>
            </tbody>
          </table>
          <p v-else class="empty-state">{{ t('payment.adminRevenueSplit.previewEmpty') }}</p>

          <p class="legal-banner soft">{{ t('payment.adminRevenueSplit.marginWarning') }}</p>
        </div>
      </section>
    </template>

    <!-- ---------- 汇总 ---------- -->
    <template v-else-if="tab === 'summary'">
      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.summaryTitle') }}</h2>
          <button type="button" class="btn ghost" @click="loadSummary">{{ t('workspace.refresh') }}</button>
        </div>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.summaryHint') }}</p>

        <table v-if="summary.length" class="rs-table">
          <thead>
            <tr>
              <th>{{ t('payment.adminRevenueSplit.colBeneficiary') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colRatio') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colPending') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colLocked') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colSettled') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colReversed') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colPendingCount') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colLastEntry') }}</th>
              <th class="action-cell">{{ t('payment.adminRevenueSplit.colAction') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in summary" :key="item.beneficiary_user_id">
              <td>
                <strong>{{ item.beneficiary_name || ('#' + item.beneficiary_user_id) }}</strong>
                <div class="sub">#{{ item.beneficiary_user_id }}<template v-if="item.beneficiary_email"> · {{ item.beneficiary_email }}</template></div>
              </td>
              <td>
                {{ fmtPercent(item.ratio_percent) }}
                <span v-if="!item.enabled" class="badge off">{{ t('payment.adminRevenueSplit.enabledOff') }}</span>
              </td>
              <td class="num strong">{{ fmtMoney(item.pending_amount) }}</td>
              <td class="num">{{ fmtMoney(item.locked_amount) }}</td>
              <td class="num">{{ fmtMoney(item.settled_amount) }}</td>
              <td class="num muted">{{ fmtMoney(item.reversed_amount) }}</td>
              <td class="num">{{ item.pending_count }}</td>
              <td class="muted">{{ fmtTime(item.last_entry_at) }}</td>
              <td class="action-cell">
                <button type="button" class="action" :disabled="item.pending_amount <= 0" @click="openCreate(item)">
                  {{ t('payment.adminRevenueSplit.createSettlement') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-state">{{ t('payment.adminRevenueSplit.noSummary') }}</p>
      </section>

      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.repairTitle') }}</h2>
        </div>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.repairHint') }}</p>
        <div class="preview-row">
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.repairOrderId') }}</span>
            <input v-model.number="repairOrderId" type="number" min="1" />
          </label>
          <button type="button" class="btn primary" :disabled="repairing" @click="runRepair">
            {{ t('payment.adminRevenueSplit.repairRun') }}
          </button>
        </div>
      </section>
    </template>

    <!-- ---------- 明细 ---------- -->
    <template v-else-if="tab === 'entries'">
      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.entriesTitle') }}</h2>
        </div>
        <div class="filter-row">
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.filterBeneficiary') }}</span>
            <input v-model="entryFilters.beneficiary_user_id" type="number" min="1" />
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.filterStatus') }}</span>
            <select v-model="entryFilters.status">
              <option value="">{{ t('payment.adminRevenueSplit.colStatus') }}</option>
              <option value="pending">{{ t('payment.adminRevenueSplit.statusPending') }}</option>
              <option value="settled">{{ t('payment.adminRevenueSplit.statusSettled') }}</option>
              <option value="reversed">{{ t('payment.adminRevenueSplit.statusReversed') }}</option>
            </select>
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.filterStart') }}</span>
            <input v-model="entryFilters.start" type="date" />
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.filterEnd') }}</span>
            <input v-model="entryFilters.end" type="date" />
          </label>
          <label class="field wide">
            <span>{{ t('payment.adminRevenueSplit.filterKeyword') }}</span>
            <input v-model="entryFilters.keyword" type="text" />
          </label>
          <div class="filter-actions">
            <button type="button" class="btn primary" @click="loadEntries(true)">{{ t('payment.adminRevenueSplit.filterApply') }}</button>
            <button type="button" class="btn ghost" @click="resetEntryFilters">{{ t('payment.adminRevenueSplit.filterReset') }}</button>
          </div>
        </div>

        <table v-if="entries.length" class="rs-table">
          <thead>
            <tr>
              <th>{{ t('payment.adminRevenueSplit.colTime') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colOrder') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colBeneficiary') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colPayAmount') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colFee') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colBase') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colRatio') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colSplit') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colStatus') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colSettlement') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in entries" :key="row.id">
              <td class="muted">{{ fmtTime(row.created_at) }}</td>
              <td>
                <strong>#{{ row.order_id }}</strong>
                <div class="sub">{{ row.order_out_trade_no || '—' }}</div>
              </td>
              <td>
                {{ row.beneficiary_name || ('#' + row.beneficiary_user_id) }}
                <div class="sub">#{{ row.beneficiary_user_id }}</div>
              </td>
              <td class="num">{{ fmtMoney(row.pay_amount, row.currency) }}</td>
              <td class="num muted">-{{ fmtMoney(row.channel_fee_amount, row.currency) }}</td>
              <td class="num">{{ fmtMoney(row.base_amount, row.currency) }}</td>
              <td>{{ fmtPercent(row.ratio_percent) }}</td>
              <td class="num strong">{{ fmtMoney(row.split_amount, row.currency) }}</td>
              <td>
                <span class="badge" :class="row.status">{{ entryStatusLabel(row.status) }}</span>
              </td>
              <td>
                <span v-if="row.settlement_id">#{{ row.settlement_id }}</span>
                <span v-else class="muted">{{ t('payment.adminRevenueSplit.colUnsettled') }}</span>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-state">{{ t('payment.adminRevenueSplit.emptyEntries') }}</p>

        <div class="pager">
          <button type="button" class="btn ghost" :disabled="entryPage <= 1" @click="entryPage -= 1; loadEntries()">
            {{ t('payment.adminRevenueSplit.pagePrev') }}
          </button>
          <span>{{ t('payment.adminRevenueSplit.pageInfo', { page: entryPage, pages: entryPages, total: entryTotal }) }}</span>
          <button type="button" class="btn ghost" :disabled="entryPage >= entryPages" @click="entryPage += 1; loadEntries()">
            {{ t('payment.adminRevenueSplit.pageNext') }}
          </button>
        </div>
      </section>
    </template>

    <!-- ---------- 结算单 ---------- -->
    <template v-else>
      <section class="card">
        <div class="card-head">
          <h2>{{ t('payment.adminRevenueSplit.settlementsTitle') }}</h2>
          <label class="field inline">
            <select v-model="settlementStatus" @change="loadSettlements(true)">
              <option value="">{{ t('payment.adminRevenueSplit.colStatus') }}</option>
              <option value="draft">{{ t('payment.adminRevenueSplit.statusDraft') }}</option>
              <option value="paid">{{ t('payment.adminRevenueSplit.statusPaid') }}</option>
              <option value="cancelled">{{ t('payment.adminRevenueSplit.statusCancelled') }}</option>
            </select>
          </label>
        </div>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.settlementsHint') }}</p>

        <table v-if="settlements.length" class="rs-table">
          <thead>
            <tr>
              <th>#</th>
              <th>{{ t('payment.adminRevenueSplit.colBeneficiary') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colPeriod') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colEntryCount') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colAmount') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colSettlementStatus') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colReference') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colCreatedAt') }}</th>
              <th class="action-cell">{{ t('payment.adminRevenueSplit.colAction') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in settlements" :key="row.id">
              <td class="muted">{{ row.id }}</td>
              <td>
                {{ row.beneficiary_name || ('#' + row.beneficiary_user_id) }}
                <div class="sub">#{{ row.beneficiary_user_id }}</div>
              </td>
              <td class="muted">{{ periodLabel(row) }}</td>
              <td class="num">{{ row.entry_count }}</td>
              <td class="num strong">{{ fmtMoney(row.amount, row.currency) }}</td>
              <td><span class="badge" :class="row.status">{{ settlementStatusLabel(row.status) }}</span></td>
              <td class="muted">{{ row.reference || '—' }}</td>
              <td class="muted">{{ fmtTime(row.created_at) }}</td>
              <td class="action-cell">
                <button type="button" class="action" @click="openDetail(row)">{{ t('payment.adminRevenueSplit.viewDetail') }}</button>
                <button v-if="row.status === 'draft'" type="button" class="action" @click="openPay(row)">
                  {{ t('payment.adminRevenueSplit.markPaid') }}
                </button>
                <button v-if="row.status === 'draft'" type="button" class="action danger" @click="cancelSettlement(row)">
                  {{ t('payment.adminRevenueSplit.cancelSettlement') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-state">{{ t('payment.adminRevenueSplit.emptySettlements') }}</p>

        <div class="pager">
          <button type="button" class="btn ghost" :disabled="settlementPage <= 1" @click="settlementPage -= 1; loadSettlements()">
            {{ t('payment.adminRevenueSplit.pagePrev') }}
          </button>
          <span>{{ t('payment.adminRevenueSplit.pageInfo', { page: settlementPage, pages: settlementPages, total: settlementTotal }) }}</span>
          <button type="button" class="btn ghost" :disabled="settlementPage >= settlementPages" @click="settlementPage += 1; loadSettlements()">
            {{ t('payment.adminRevenueSplit.pageNext') }}
          </button>
        </div>
      </section>
    </template>

    <!-- ---------- 弹窗：生成结算单 ---------- -->
    <div v-if="createOpen" class="modal-mask" @click.self="createOpen = false">
      <div class="modal-card">
        <h3>{{ t('payment.adminRevenueSplit.createSettlementTitle') }}</h3>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.createSettlementHint') }}</p>
        <div class="form-grid">
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.colBeneficiary') }}</span>
            <input :value="createForm.beneficiary_label" type="text" readonly />
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.periodStart') }}</span>
            <input v-model="createForm.period_start" type="date" />
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.periodEnd') }}</span>
            <input v-model="createForm.period_end" type="date" />
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.settlementMethod') }}</span>
            <input v-model="createForm.method" type="text" :placeholder="t('payment.adminRevenueSplit.methodPlaceholder')" />
          </label>
          <label class="field full">
            <span>{{ t('payment.adminRevenueSplit.settlementNote') }}</span>
            <input v-model="createForm.note" type="text" />
          </label>
        </div>
        <footer class="modal-actions">
          <button type="button" class="btn ghost" @click="createOpen = false">{{ t('payment.cancel') }}</button>
          <button type="button" class="btn primary" :disabled="saving" @click="submitCreate">
            {{ saving ? t('payment.saving') : t('payment.adminRevenueSplit.createSettlement') }}
          </button>
        </footer>
      </div>
    </div>

    <!-- ---------- 弹窗：标记已打款 ---------- -->
    <div v-if="payOpen" class="modal-mask" @click.self="payOpen = false">
      <div class="modal-card">
        <h3>{{ t('payment.adminRevenueSplit.markPaidTitle') }}</h3>
        <p class="card-hint">{{ t('payment.adminRevenueSplit.confirmMarkPaid') }}</p>
        <div class="form-grid">
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.referenceLabel') }}</span>
            <input v-model="payForm.reference" type="text" :placeholder="t('payment.adminRevenueSplit.referenceHint')" />
          </label>
          <label class="field">
            <span>{{ t('payment.adminRevenueSplit.methodLabel') }}</span>
            <input v-model="payForm.method" type="text" :placeholder="t('payment.adminRevenueSplit.methodPlaceholder')" />
          </label>
          <label class="field full">
            <span>{{ t('payment.adminRevenueSplit.noteLabel') }}</span>
            <input v-model="payForm.note" type="text" />
          </label>
        </div>
        <footer class="modal-actions">
          <button type="button" class="btn ghost" @click="payOpen = false">{{ t('payment.cancel') }}</button>
          <button type="button" class="btn primary" :disabled="saving" @click="submitPay">
            {{ saving ? t('payment.saving') : t('payment.adminRevenueSplit.markPaid') }}
          </button>
        </footer>
      </div>
    </div>

    <!-- ---------- 弹窗：结算单详情 ---------- -->
    <div v-if="detailOpen" class="modal-mask" @click.self="closeDetail">
      <div class="modal-card wide">
        <h3>
          {{ t('payment.adminRevenueSplit.settlementDetail') }} #{{ detailSettlement?.id }}
          <span v-if="detailSettlement" class="badge" :class="detailSettlement.status">
            {{ settlementStatusLabel(detailSettlement.status) }}
          </span>
        </h3>
        <div v-if="detailSettlement" class="detail-grid">
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.colBeneficiary') }}</span><strong>{{ detailSettlement.beneficiary_name || ('#' + detailSettlement.beneficiary_user_id) }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.colPeriod') }}</span><strong>{{ periodLabel(detailSettlement) }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.colEntryCount') }}</span><strong>{{ detailSettlement.entry_count }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.colAmount') }}</span><strong>{{ fmtMoney(detailSettlement.amount, detailSettlement.currency) }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.colReference') }}</span><strong>{{ detailSettlement.reference || '—' }}</strong></div>
          <div class="metric"><span>{{ t('payment.adminRevenueSplit.colCreatedAt') }}</span><strong>{{ fmtTime(detailSettlement.created_at) }}</strong></div>
        </div>

        <h4>{{ t('payment.adminRevenueSplit.lockedEntries') }}</h4>
        <p v-if="detailLoading" class="empty-state">…</p>
        <table v-else-if="detailEntries.length" class="rs-table compact">
          <thead>
            <tr>
              <th>{{ t('payment.adminRevenueSplit.colTime') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colOrder') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colPayAmount') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colBase') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colRatio') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colSplit') }}</th>
              <th>{{ t('payment.adminRevenueSplit.colStatus') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in detailEntries" :key="row.id">
              <td class="muted">{{ fmtTime(row.created_at) }}</td>
              <td>#{{ row.order_id }}</td>
              <td class="num">{{ fmtMoney(row.pay_amount, row.currency) }}</td>
              <td class="num">{{ fmtMoney(row.base_amount, row.currency) }}</td>
              <td>{{ fmtPercent(row.ratio_percent) }}</td>
              <td class="num strong">{{ fmtMoney(row.split_amount, row.currency) }}</td>
              <td><span class="badge" :class="row.status">{{ entryStatusLabel(row.status) }}</span></td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty-state">{{ t('payment.adminRevenueSplit.emptyEntries') }}</p>

        <footer class="modal-actions">
          <button type="button" class="btn ghost" @click="closeDetail">{{ t('payment.adminRevenueSplit.backToList') }}</button>
          <button
            v-if="detailSettlement && detailSettlement.status === 'draft'"
            type="button"
            class="btn primary"
            @click="openPay(detailSettlement)"
          >
            {{ t('payment.adminRevenueSplit.markPaid') }}
          </button>
        </footer>
      </div>
    </div>
  </section>
</template>

<style scoped>
.revenue-split { width: 100%; max-width: 1360px; margin: 0 auto; padding: 12px 0 44px; }
.rs-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; margin-bottom: 18px; }
.rs-heading h1 { margin: 0; color: #f7f8fa; font-size: clamp(1.7rem, 2.2vw, 2.15rem); line-height: 1.1; font-weight: 680; letter-spacing: -.04em; }
.rs-heading p { max-width: 760px; margin: 10px 0 0; color: #858d97; font-size: .88rem; line-height: 1.6; }
.rs-heading-actions { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.status-chip { padding: 5px 12px; border-radius: 999px; font: 600 .72rem/1 ui-sans-serif, system-ui, sans-serif; background: rgba(140, 145, 152, .14); color: #8c9198; }
.status-chip.on { background: rgba(72, 187, 153, .14); color: #48bb99; }
.legal-banner { margin: 0 0 14px; padding: 11px 14px; border-radius: 8px; background: rgba(74, 147, 197, .09); border: 1px solid rgba(74, 147, 197, .3); color: #a9c9e0; font-size: .8rem; line-height: 1.62; }
.legal-banner.soft { background: rgba(214, 158, 46, .09); border-color: rgba(214, 158, 46, .3); color: #d9bd85; margin-top: 14px; }
.error-banner { margin: 0 0 14px; padding: 10px 14px; border-radius: 8px; background: rgba(239, 68, 68, .12); border: 1px solid rgba(239, 68, 68, .35); color: #fca5a5; font-size: .84rem; }
.notice-banner { margin: 0 0 14px; padding: 10px 14px; border-radius: 8px; background: rgba(72, 187, 153, .12); border: 1px solid rgba(72, 187, 153, .32); color: #86e0be; font-size: .84rem; }
.rs-tabs { display: flex; gap: 6px; margin-bottom: 18px; border-bottom: 1px solid #1d2128; }
.rs-tabs button { padding: 10px 16px; border: 0; border-bottom: 2px solid transparent; background: transparent; color: #858d97; font: 500 .85rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.rs-tabs button:hover { color: #d6dbe1; }
.rs-tabs button.active { color: #f7f8fa; border-bottom-color: #4a93c5; }
.card { margin-bottom: 18px; padding: 20px 22px; border-radius: 14px; background: #11141a; border: 1px solid #1d2128; }
.card-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 10px; flex-wrap: wrap; }
.card-head h2 { margin: 0; color: #f7f8fa; font-size: 1rem; font-weight: 620; }
.card-hint { margin: 0 0 14px; color: #7d858e; font-size: .79rem; line-height: 1.6; }
.card-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 16px; }
.ratio-total { font: 500 .78rem/1 ui-sans-serif, system-ui, sans-serif; color: #8c9198; }
.ratio-total.bad { color: #fca5a5; }
.form-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px; }
.field { display: block; }
.field.inline { margin: 0; }
.field.full { grid-column: 1 / -1; }
.field.wide { min-width: 220px; }
.field > span { display: block; margin-bottom: 5px; color: #b8bfc7; font-size: .78rem; }
.field .hint { display: block; margin-top: 5px; color: #6c727b; font-size: .72rem; font-style: normal; line-height: 1.5; }
.field input, .field select { width: 100%; padding: 8px 12px; border-radius: 7px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .82rem/1.4 ui-sans-serif, system-ui, sans-serif; box-sizing: border-box; }
.field input:focus, .field select:focus { outline: none; border-color: #4a93c5; }
.field input[readonly] { color: #8c9198; }
.preview-row { display: flex; align-items: flex-end; gap: 12px; max-width: 560px; }
.preview-row .field { flex: 1; }
.preview-result { margin-top: 18px; }
.metric { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; padding: 8px 0; border-bottom: 1px solid #161a20; font-size: .84rem; }
.metric span { color: #858d97; }
.metric strong { color: #f7f8fa; font-weight: 620; }
.metric.base strong { color: #6ec0f5; }
.metric.bad strong { color: #fca5a5; }
.detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0 20px; margin-bottom: 18px; }
.rs-table { width: 100%; margin-top: 8px; border-collapse: collapse; border-radius: 12px; background: #0f1217; border: 1px solid #1d2128; overflow: hidden; }
.rs-table th { text-align: left; padding: 11px 12px; color: #6c727b; font: 500 .7rem/1 ui-sans-serif, system-ui, sans-serif; text-transform: uppercase; letter-spacing: .07em; border-bottom: 1px solid #1d2128; background: #0f1217; white-space: nowrap; }
.rs-table td { padding: 12px; color: #d6dbe1; font-size: .82rem; border-bottom: 1px solid #161a20; vertical-align: middle; }
.rs-table tr:last-child td { border-bottom: 0; }
.rs-table.compact td, .rs-table.compact th { padding: 8px 10px; font-size: .78rem; }
.rs-table .sub { color: #6c727b; font-size: .7rem; margin-top: 2px; }
.rs-table .num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
.rs-table .num.strong, .rs-table .strong { color: #f7f8fa; font-weight: 600; }
.rs-table .muted, .muted { color: #7d858e; }
.cell-input { width: 100%; padding: 6px 9px; border-radius: 6px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .8rem/1.3 ui-sans-serif, system-ui, sans-serif; box-sizing: border-box; }
.cell-input.narrow { max-width: 120px; }
.cell-input:focus { outline: none; border-color: #4a93c5; }
.action-cell { white-space: nowrap; }
.action { padding: 5px 10px; margin-right: 6px; border-radius: 6px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; font: 500 .72rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.action:hover:not(:disabled) { border-color: #3d4754; }
.action:disabled { opacity: .45; cursor: not-allowed; }
.action.danger { color: #fca5a5; border-color: rgba(239, 68, 68, .35); }
.action.danger:hover { background: rgba(239, 68, 68, .08); }
.badge { display: inline-block; padding: 3px 9px; border-radius: 999px; font: 500 .72rem/1.4 ui-sans-serif, system-ui, sans-serif; background: rgba(140, 145, 152, .14); color: #8c9198; }
.badge.pending, .badge.draft { background: rgba(214, 158, 46, .14); color: #d9bd85; }
.badge.settled, .badge.paid { background: rgba(72, 187, 153, .14); color: #48bb99; }
.badge.reversed, .badge.cancelled, .badge.off { background: rgba(140, 145, 152, .14); color: #8c9198; }
.filter-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; margin-bottom: 12px; }
.filter-row .field { min-width: 150px; }
.filter-actions { display: flex; gap: 8px; margin-left: auto; }
.pager { display: flex; align-items: center; justify-content: center; gap: 14px; margin-top: 16px; color: #7d858e; font-size: .78rem; }
.empty-state { padding: 46px 0; text-align: center; color: #6c727b; font-size: .84rem; }
.primary-btn { padding: 8px 16px; border-radius: 8px; background: #4a93c5; border: 1px solid #4a93c5; color: #0d0f12; font: 600 .82rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.primary-btn:hover:not(:disabled) { background: #5fa3d5; border-color: #5fa3d5; }
.primary-btn:disabled { opacity: .6; cursor: not-allowed; }
.btn { padding: 9px 20px; border-radius: 7px; border: 1px solid transparent; font: 500 .85rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.btn.ghost { background: #16191f; border-color: #2a2f37; color: #d6dbe1; }
.btn.ghost:hover:not(:disabled) { border-color: #3d4754; }
.btn.primary { background: #4a93c5; color: #0d0f12; }
.btn.primary:hover:not(:disabled) { background: #5fa3d5; }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.modal-mask { position: fixed; inset: 0; background: rgba(8, 10, 14, .7); display: flex; align-items: flex-start; justify-content: center; z-index: 1000; overflow-y: auto; padding: 40px 16px; }
.modal-card { width: min(720px, 100%); padding: 24px 26px; background: #11141a; border: 1px solid #1d2128; border-radius: 14px; }
.modal-card.wide { width: min(1080px, 100%); }
.modal-card h3 { display: flex; align-items: center; gap: 10px; margin: 0 0 12px; color: #f7f8fa; font-size: 1.05rem; font-weight: 620; }
.modal-card h4 { margin: 18px 0 6px; color: #b8bfc7; font-size: .85rem; font-weight: 600; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
@media (max-width: 900px) {
  .rs-heading { flex-direction: column; }
  .rs-table { display: block; overflow-x: auto; }
  .detail-grid { grid-template-columns: 1fr; }
}
</style>
