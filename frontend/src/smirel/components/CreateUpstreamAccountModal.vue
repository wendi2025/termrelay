<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api, getErrorMessage, previewMode } from '../core/api'

interface GroupOption {
  id: number
  name?: string
  platform?: string
  status?: string
}

interface CreateAccountResult {
  id?: number
}

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const baseUrl = ref('')
const apiKey = ref('')
const notes = ref('')
const concurrency = ref(10)
const priority = ref(10)
const rateMultiplier = ref(1)
const groupIds = ref<number[]>([])
const groups = ref<GroupOption[]>([])
const groupsLoading = ref(false)
const showApiKey = ref(false)
const syncing = ref(false)
const saving = ref(false)
const error = ref('')
const syncedModels = ref<string[]>([])

const canSync = computed(() => Boolean(baseUrl.value.trim() && apiKey.value.trim()) && !syncing.value)
const canSave = computed(() => Boolean(name.value.trim() && baseUrl.value.trim() && apiKey.value.trim()) && !saving.value)

function cleanBaseUrl(value: string) {
  return value.trim().replace(/\/+$/, '')
}

function validateBaseUrl(value: string) {
  try {
    const url = new URL(cleanBaseUrl(value))
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function resetForm() {
  name.value = ''
  baseUrl.value = ''
  apiKey.value = ''
  notes.value = ''
  concurrency.value = 10
  priority.value = 10
  rateMultiplier.value = 1
  groupIds.value = []
  showApiKey.value = false
  syncing.value = false
  saving.value = false
  error.value = ''
  syncedModels.value = []
}

function close() {
  if (saving.value) return
  emit('close')
}

function toggleGroup(id: number) {
  groupIds.value = groupIds.value.includes(id)
    ? groupIds.value.filter((item) => item !== id)
    : [...groupIds.value, id]
}

function normalizeModels(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => {
      if (typeof item === 'string') return item
      if (item && typeof item === 'object' && 'id' in item) return String((item as { id?: unknown }).id || '')
      return ''
    })
    .map((item) => item.trim())
    .filter(Boolean)
}

async function loadGroups() {
  groupsLoading.value = true
  try {
    if (previewMode) {
      groups.value = [{ id: 1, name: 'openai-default', platform: 'openai', status: 'active' }]
    } else {
      const response = await api.get<GroupOption[]>('/admin/groups/all', { params: { platform: 'openai' } })
      groups.value = Array.isArray(response.data) ? response.data : []
    }

    if (!groupIds.value.length && groups.value.length) {
      const preferred = groups.value.find((item) => String(item.name || '').toLowerCase().includes('default')) || groups.value[0]
      groupIds.value = [preferred.id]
    }
  } catch {
    groups.value = []
  } finally {
    groupsLoading.value = false
  }
}

async function syncModels() {
  error.value = ''
  syncedModels.value = []
  if (!validateBaseUrl(baseUrl.value)) {
    error.value = '请输入有效的 Base URL，例如 https://api.example.com'
    return
  }
  if (!apiKey.value.trim()) {
    error.value = '请输入上游 API Key'
    return
  }

  syncing.value = true
  try {
    if (previewMode) {
      syncedModels.value = ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.5']
      return
    }
    const response = await api.post<{ models?: unknown[] }>('/admin/accounts/models/sync-upstream-preview', {
      platform: 'openai',
      type: 'apikey',
      base_url: cleanBaseUrl(baseUrl.value),
      api_key: apiKey.value.trim(),
    })
    syncedModels.value = normalizeModels(response.data?.models)
    if (!syncedModels.value.length) error.value = '连接成功，但上游没有返回可用模型。'
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    syncing.value = false
  }
}

async function submit() {
  error.value = ''
  if (!name.value.trim()) {
    error.value = '请输入账户名称'
    return
  }
  if (!validateBaseUrl(baseUrl.value)) {
    error.value = '请输入有效的 Base URL，例如 https://api.example.com'
    return
  }
  if (!apiKey.value.trim()) {
    error.value = '请输入上游 API Key'
    return
  }

  // Always verify the live upstream before persisting the account. Besides
  // validating the credentials, this gives us the exact model whitelist that
  // should be advertised by TermRelay for this relay account.
  if (!syncedModels.value.length) {
    await syncModels()
    if (!syncedModels.value.length) return
  }

  saving.value = true
  try {
    if (previewMode) {
      emit('created')
      emit('close')
      return
    }

    const modelMapping = Object.fromEntries(syncedModels.value.map((model) => [model, model]))
    await api.post<CreateAccountResult>('/admin/accounts', {
      name: name.value.trim(),
      notes: notes.value.trim() || undefined,
      platform: 'openai',
      type: 'apikey',
      credentials: {
        api_key: apiKey.value.trim(),
        base_url: cleanBaseUrl(baseUrl.value),
        model_mapping: modelMapping,
      },
      // OpenAI-compatible relay accounts should preserve the caller's request
      // semantics and only replace upstream authentication.
      extra: { openai_passthrough: true },
      concurrency: Math.max(1, Math.round(Number(concurrency.value) || 1)),
      priority: Math.max(0, Math.round(Number(priority.value) || 0)),
      rate_multiplier: Math.max(0, Number(rateMultiplier.value) || 0),
      group_ids: groupIds.value,
    })

    apiKey.value = ''
    emit('created')
    emit('close')
  } catch (caught) {
    error.value = getErrorMessage(caught)
  } finally {
    saving.value = false
  }
}

watch(
  () => props.show,
  (visible) => {
    if (!visible) return
    resetForm()
    void loadGroups()
  },
)
</script>

<template>
  <Teleport to="body">
    <div v-if="show" class="upstream-create-backdrop" @click.self="close">
      <section class="upstream-create-dialog" role="dialog" aria-modal="true" aria-labelledby="upstream-create-title">
        <header class="create-dialog-head">
          <div>
            <span class="create-kicker">UPSTREAM CONNECTION</span>
            <div class="create-title-row">
              <h2 id="upstream-create-title">新增上游账户</h2>
              <span class="protocol-badge"><i></i> OpenAI Compatible</span>
            </div>
            <p>接入兼容 OpenAI API 的上游服务。凭据只提交到 TermRelay 服务端。</p>
          </div>
          <button class="dialog-close" type="button" aria-label="关闭" :disabled="saving" @click="close">
            <svg viewBox="0 0 20 20" aria-hidden="true"><path d="m5 5 10 10M15 5 5 15" /></svg>
          </button>
        </header>

        <div class="create-dialog-body">
          <section class="form-section connection-section">
            <div class="section-heading">
              <span>01</span>
              <div><strong>连接信息</strong><small>上游地址与认证凭据</small></div>
            </div>

            <div class="form-grid">
              <label class="field field-wide">
                <span>账户名称</span>
                <input v-model="name" type="text" maxlength="80" placeholder="例如 SwiftAPI Primary" autocomplete="off" />
              </label>

              <label class="field field-wide">
                <span>Base URL</span>
                <div class="input-with-prefix">
                  <b>URL</b>
                  <input v-model="baseUrl" type="url" placeholder="https://api.example.com" autocomplete="off" @input="syncedModels = []" />
                </div>
                <small>填写服务根地址；TermRelay 会按 OpenAI 兼容协议转发。</small>
              </label>

              <label class="field field-wide">
                <span>API Key</span>
                <div class="secret-input">
                  <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="7" cy="10" r="3" /><path d="m9.7 8.3 6-6M13 5l2 2" /></svg>
                  <input v-model="apiKey" :type="showApiKey ? 'text' : 'password'" placeholder="sk-••••••••••••••••" autocomplete="off" @input="syncedModels = []" />
                  <button type="button" @click="showApiKey = !showApiKey">{{ showApiKey ? '隐藏' : '显示' }}</button>
                </div>
              </label>
            </div>

            <div class="model-probe" :class="{ ready: syncedModels.length }">
              <div class="probe-copy">
                <span class="probe-icon">
                  <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 5.5h12M4 10h12M4 14.5h7" /></svg>
                </span>
                <div>
                  <strong>{{ syncedModels.length ? `已发现 ${syncedModels.length} 个模型` : '检测上游模型' }}</strong>
                  <small>{{ syncedModels.length ? '模型目录已验证，创建后保持原始模型 ID 透传。' : '建议创建前先检测 /v1/models，确认凭据和地址可用。' }}</small>
                </div>
              </div>
              <button type="button" :disabled="!canSync" @click="syncModels">
                <svg :class="{ spinning: syncing }" viewBox="0 0 20 20" aria-hidden="true"><path d="M16.2 6.1A7 7 0 1 0 17 12" /><path d="M16.3 2.9v3.6h-3.6" /></svg>
                {{ syncing ? '检测中' : syncedModels.length ? '重新检测' : '同步模型' }}
              </button>
            </div>

            <div v-if="syncedModels.length" class="model-preview">
              <code v-for="model in syncedModels.slice(0, 8)" :key="model">{{ model }}</code>
              <span v-if="syncedModels.length > 8">+{{ syncedModels.length - 8 }}</span>
            </div>
          </section>

          <section class="form-section scheduling-section">
            <div class="section-heading">
              <span>02</span>
              <div><strong>调度设置</strong><small>控制并发、优先级与结算倍率</small></div>
            </div>

            <div class="schedule-grid">
              <label class="field">
                <span>最大并发</span>
                <input v-model.number="concurrency" type="number" min="1" max="10000" />
                <small>该账户允许的并发请求数。</small>
              </label>
              <label class="field">
                <span>优先级</span>
                <input v-model.number="priority" type="number" min="0" max="9999" />
                <small>数值越小，调度优先级越高。</small>
              </label>
              <label class="field">
                <span>倍率</span>
                <div class="number-suffix"><input v-model.number="rateMultiplier" type="number" min="0" step="0.01" /><b>×</b></div>
                <small>仅作为账户结算倍率，不等于销售价格。</small>
              </label>
            </div>
          </section>

          <section class="form-section group-section">
            <div class="section-heading">
              <span>03</span>
              <div><strong>调度分组</strong><small>选择允许该账户参与调度的 OpenAI 分组</small></div>
            </div>

            <div v-if="groupsLoading" class="groups-loading"><i></i><i></i><i></i></div>
            <div v-else-if="groups.length" class="group-options">
              <button
                v-for="group in groups"
                :key="group.id"
                type="button"
                :class="{ selected: groupIds.includes(group.id) }"
                @click="toggleGroup(group.id)"
              >
                <span class="group-check">
                  <svg v-if="groupIds.includes(group.id)" viewBox="0 0 16 16" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg>
                </span>
                <span><strong>{{ group.name || `Group #${group.id}` }}</strong><small>#{{ group.id }} · {{ group.platform || 'openai' }}</small></span>
              </button>
            </div>
            <p v-else class="group-empty">未找到 OpenAI 分组。账户仍可创建，之后可在分组配置中绑定。</p>
          </section>

          <label class="field notes-field">
            <span>备注 <small>可选</small></span>
            <textarea v-model="notes" rows="2" maxlength="300" placeholder="例如：第三方 OpenAI 兼容中转 · 主线路"></textarea>
          </label>

          <p v-if="error" class="create-error">
            <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="10" r="7" /><path d="M10 6.5v4M10 13.8v.2" /></svg>
            <span>{{ error }}</span>
          </p>
        </div>

        <footer class="create-dialog-foot">
          <div class="security-note">
            <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M10 2.5 16 5v4.5c0 4-2.4 6.5-6 8-3.6-1.5-6-4-6-8V5l6-2.5Z" /><path d="m7.5 10 1.6 1.6 3.5-3.6" /></svg>
            <span>API Key 不会写入前端配置或 GitHub。</span>
          </div>
          <div>
            <button class="cancel-button" type="button" :disabled="saving" @click="close">取消</button>
            <button class="save-button" type="button" :disabled="!canSave" @click="submit">
              <svg v-if="!saving" viewBox="0 0 20 20" aria-hidden="true"><path d="M10 4v12M4 10h12" /></svg>
              <svg v-else class="spinning" viewBox="0 0 20 20" aria-hidden="true"><path d="M16.2 6.1A7 7 0 1 0 17 12" /><path d="M16.3 2.9v3.6h-3.6" /></svg>
              {{ saving ? '创建中' : '创建账户' }}
            </button>
          </div>
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.upstream-create-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1400;
  padding: 24px;
  background: rgba(3, 4, 7, .78);
  backdrop-filter: blur(12px);
  display: grid;
  place-items: center;
}

.upstream-create-dialog {
  --uc-surface: #101217;
  --uc-raised: #14171d;
  --uc-border: #292d35;
  --uc-border-strong: #373c46;
  --uc-text: #f4f6f8;
  --uc-soft: #c7cdd4;
  --uc-muted: #7d8692;
  --uc-green: #43cd98;
  width: min(760px, 100%);
  max-height: min(860px, calc(100vh - 48px));
  border: 1px solid #30343d;
  border-radius: 16px;
  background: linear-gradient(180deg, #121419 0%, #0e1014 100%);
  color: var(--uc-text);
  box-shadow: 0 28px 90px rgba(0, 0, 0, .56), 0 0 0 1px rgba(255, 255, 255, .015) inset;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.create-dialog-head {
  padding: 25px 27px 21px;
  border-bottom: 1px solid #24272e;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.create-kicker {
  display: block;
  margin-bottom: 9px;
  color: #68727e;
  font: 650 .64rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .11em;
}

.create-title-row { display: flex; align-items: center; gap: 11px; }
.create-dialog-head h2 { margin: 0; font-size: 1.42rem; line-height: 1.15; font-weight: 690; letter-spacing: -.035em; }
.create-dialog-head p { margin: 9px 0 0; color: #79828e; font-size: .72rem; line-height: 1.55; }

.protocol-badge {
  height: 25px;
  padding: 0 9px;
  border: 1px solid rgba(67, 205, 152, .20);
  border-radius: 999px;
  background: rgba(67, 205, 152, .055);
  color: #8cd9bb;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: .62rem;
  font-weight: 620;
}
.protocol-badge i { width: 5px; height: 5px; border-radius: 50%; background: var(--uc-green); }

.dialog-close {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  border: 1px solid #2b2f37;
  border-radius: 8px;
  background: #15171c;
  color: #818a95;
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: .15s ease;
}
.dialog-close:hover:not(:disabled) { border-color: #454b56; background: #1a1d23; color: #fff; }
.dialog-close svg { width: 15px; height: 15px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; }

.create-dialog-body { padding: 22px 27px 26px; overflow-y: auto; }
.form-section + .form-section { margin-top: 23px; padding-top: 22px; border-top: 1px solid #202329; }

.section-heading { margin-bottom: 14px; display: flex; align-items: center; gap: 10px; }
.section-heading > span {
  width: 27px; height: 27px; border: 1px solid #30343c; border-radius: 7px; background: #17191f; color: #858e99;
  display: grid; place-items: center; font: 650 .62rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
}
.section-heading > div { display: flex; flex-direction: column; gap: 3px; }
.section-heading strong { color: #e8ebee; font-size: .80rem; font-weight: 650; }
.section-heading small { color: #69727d; font-size: .62rem; }

.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.field { min-width: 0; display: flex; flex-direction: column; gap: 7px; }
.field-wide { grid-column: 1 / -1; }
.field > span { color: #aeb5bd; font-size: .68rem; font-weight: 610; }
.field > span small { margin-left: 5px; color: #626b76; font-size: .6rem; font-weight: 500; }
.field > small { color: #626b76; font-size: .60rem; line-height: 1.45; }

.field input,
.field textarea,
.input-with-prefix,
.secret-input,
.number-suffix {
  border: 1px solid #2a2e36;
  border-radius: 9px;
  background: #0b0d11;
  color: #edf0f3;
  transition: border-color .15s ease, box-shadow .15s ease;
}
.field input { height: 42px; padding: 0 12px; outline: 0; font-size: .74rem; }
.field textarea { width: 100%; min-height: 68px; padding: 11px 12px; outline: 0; resize: vertical; font: inherit; font-size: .72rem; line-height: 1.5; box-sizing: border-box; }
.field input:focus, .field textarea:focus, .input-with-prefix:focus-within, .secret-input:focus-within, .number-suffix:focus-within {
  border-color: #46505d; box-shadow: 0 0 0 3px rgba(111, 132, 154, .07);
}
.field input::placeholder, .field textarea::placeholder { color: #525b66; }

.input-with-prefix, .secret-input, .number-suffix { height: 42px; display: flex; align-items: center; overflow: hidden; }
.input-with-prefix b { height: 100%; padding: 0 11px; border-right: 1px solid #262a31; color: #65707c; display: grid; place-items: center; font: 650 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }
.input-with-prefix input, .secret-input input, .number-suffix input { min-width: 0; flex: 1; height: 40px; border: 0; border-radius: 0; box-shadow: none !important; background: transparent; }
.secret-input > svg { width: 15px; height: 15px; margin-left: 12px; flex: 0 0 auto; fill: none; stroke: #626c78; stroke-width: 1.4; stroke-linecap: round; }
.secret-input button { height: 28px; margin-right: 7px; padding: 0 8px; border: 0; border-radius: 6px; background: #181b21; color: #858e99; font-size: .62rem; cursor: pointer; }
.number-suffix b { padding-right: 12px; color: #747e89; font-size: .68rem; }

.model-probe {
  margin-top: 14px; padding: 12px 13px; border: 1px solid #292d35; border-radius: 10px; background: #0d0f13;
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
}
.model-probe.ready { border-color: rgba(67, 205, 152, .19); background: rgba(67, 205, 152, .025); }
.probe-copy { min-width: 0; display: flex; align-items: center; gap: 10px; }
.probe-icon { width: 32px; height: 32px; flex: 0 0 32px; border: 1px solid #2e333b; border-radius: 8px; background: #15181d; color: #8d96a1; display: grid; place-items: center; }
.probe-icon svg { width: 15px; height: 15px; fill: none; stroke: currentColor; stroke-width: 1.4; stroke-linecap: round; }
.probe-copy > div { min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.probe-copy strong { color: #d6dbe0; font-size: .70rem; font-weight: 630; }
.probe-copy small { color: #646d78; font-size: .60rem; line-height: 1.4; }
.model-probe > button {
  min-height: 34px; padding: 0 10px; flex: 0 0 auto; border: 1px solid #343941; border-radius: 7px; background: #171a20; color: #bfc5cc;
  display: inline-flex; align-items: center; gap: 6px; font-size: .64rem; font-weight: 610; cursor: pointer;
}
.model-probe > button:hover:not(:disabled) { border-color: #4a515d; color: #fff; }
.model-probe > button:disabled { opacity: .42; cursor: default; }
.model-probe > button svg { width: 13px; height: 13px; fill: none; stroke: currentColor; stroke-width: 1.45; stroke-linecap: round; }

.model-preview { margin-top: 9px; display: flex; flex-wrap: wrap; gap: 5px; }
.model-preview code, .model-preview span { min-height: 23px; padding: 0 7px; border: 1px solid #292e35; border-radius: 6px; background: #121419; color: #858e99; display: inline-flex; align-items: center; font: 560 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }
.model-preview span { color: #6e7782; }

.schedule-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.group-options { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.group-options > button {
  min-height: 54px; padding: 9px 11px; border: 1px solid #292d35; border-radius: 9px; background: #0c0e12; color: #b9c0c7;
  display: flex; align-items: center; gap: 9px; text-align: left; cursor: pointer; transition: .15s ease;
}
.group-options > button:hover { border-color: #3b414b; background: #12151a; }
.group-options > button.selected { border-color: rgba(67, 205, 152, .31); background: rgba(67, 205, 152, .045); }
.group-check { width: 20px; height: 20px; flex: 0 0 20px; border: 1px solid #343943; border-radius: 6px; background: #15171c; display: grid; place-items: center; }
.selected .group-check { border-color: rgba(67, 205, 152, .45); background: rgba(67, 205, 152, .10); color: #6bd7ad; }
.group-check svg { width: 12px; height: 12px; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
.group-options > button > span:last-child { min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.group-options strong { overflow: hidden; color: #cdd2d8; font-size: .68rem; font-weight: 620; text-overflow: ellipsis; white-space: nowrap; }
.group-options small { color: #66707b; font: 500 .58rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }
.group-empty { margin: 0; padding: 12px; border: 1px dashed #2c3038; border-radius: 9px; color: #69727e; font-size: .66rem; line-height: 1.5; }
.groups-loading { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
.groups-loading i { height: 52px; border-radius: 9px; background: linear-gradient(90deg, #111318 20%, #181a20 50%, #111318 80%); background-size: 200% 100%; animation: upstream-shimmer 1.1s linear infinite; }

.notes-field { margin-top: 23px; padding-top: 22px; border-top: 1px solid #202329; }
.create-error { margin: 14px 0 0; padding: 10px 12px; border: 1px solid rgba(225, 108, 115, .25); border-radius: 9px; background: rgba(225, 108, 115, .06); color: #dfa0a5; display: flex; align-items: flex-start; gap: 8px; font-size: .66rem; line-height: 1.45; }
.create-error svg { width: 15px; height: 15px; flex: 0 0 auto; fill: none; stroke: currentColor; stroke-width: 1.45; stroke-linecap: round; }

.create-dialog-foot {
  min-height: 68px; padding: 12px 27px; border-top: 1px solid #24272e; background: #0d0f13;
  display: flex; align-items: center; justify-content: space-between; gap: 18px;
}
.security-note { min-width: 0; color: #626c77; display: flex; align-items: center; gap: 7px; font-size: .60rem; }
.security-note svg { width: 15px; height: 15px; flex: 0 0 auto; fill: none; stroke: #657d72; stroke-width: 1.35; stroke-linecap: round; stroke-linejoin: round; }
.create-dialog-foot > div:last-child { display: flex; gap: 8px; }
.cancel-button, .save-button { min-height: 38px; padding: 0 13px; border-radius: 8px; font-size: .68rem; font-weight: 620; cursor: pointer; }
.cancel-button { border: 1px solid #30343c; background: #14171c; color: #aeb5bd; }
.cancel-button:hover:not(:disabled) { border-color: #464c57; color: #fff; }
.save-button { border: 1px solid #d8dce1; background: #f2f4f6; color: #111318; display: inline-flex; align-items: center; gap: 7px; }
.save-button:hover:not(:disabled) { background: #fff; }
.save-button:disabled, .cancel-button:disabled, .dialog-close:disabled { opacity: .42; cursor: default; }
.save-button svg { width: 14px; height: 14px; fill: none; stroke: currentColor; stroke-width: 1.6; stroke-linecap: round; stroke-linejoin: round; }
.spinning { animation: upstream-spin .8s linear infinite; }

@keyframes upstream-spin { to { transform: rotate(360deg); } }
@keyframes upstream-shimmer { to { background-position: -200% 0; } }

@media (max-width: 720px) {
  .upstream-create-backdrop { padding: 10px; align-items: end; }
  .upstream-create-dialog { width: 100%; max-height: calc(100vh - 20px); border-radius: 15px 15px 10px 10px; }
  .create-dialog-head { padding: 20px 18px 17px; }
  .create-dialog-head p { max-width: 520px; }
  .create-title-row { align-items: flex-start; flex-direction: column; gap: 8px; }
  .create-dialog-body { padding: 18px; }
  .form-grid, .schedule-grid, .group-options { grid-template-columns: 1fr; }
  .model-probe { align-items: flex-start; flex-direction: column; }
  .model-probe > button { width: 100%; justify-content: center; }
  .create-dialog-foot { padding: 12px 18px; align-items: stretch; flex-direction: column; }
  .security-note { order: 2; }
  .create-dialog-foot > div:last-child { width: 100%; }
  .cancel-button, .save-button { flex: 1; justify-content: center; }
}
</style>