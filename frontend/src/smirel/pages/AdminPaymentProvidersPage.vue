<script setup lang="ts">
// Admin Payment Providers —— CRUD + 动态字段表单
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  paymentAdminApi,
  type ProviderInstance,
  type PaymentType,
  type PaymentMode,
} from '../api/payment'
import { PROVIDER_SCHEMAS, getProviderSchema } from '../api/paymentProviderSchemas'
import { getErrorMessage } from '../core/api'

const { t } = useI18n()

const loading = ref(false)
const error = ref('')
const providers = ref<ProviderInstance[]>([])

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    providers.value = (await paymentAdminApi.listProviders()) || []
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const draft = ref<{
  id?: number
  provider_key: PaymentType
  name: string
  config: Record<string, string>
  supported_types: PaymentType[]
  enabled: boolean
  payment_mode: PaymentMode
  sort_order: number
  limits: string
  refund_enabled: boolean
  allow_user_refund: boolean
}>(emptyDraft())

function emptyDraft() {
  return {
    provider_key: 'easypay' as PaymentType,
    name: '',
    config: {} as Record<string, string>,
    supported_types: ['easypay'] as PaymentType[],
    enabled: true,
    payment_mode: 'redirect' as PaymentMode,
    sort_order: 0,
    limits: '',
    refund_enabled: true,
    allow_user_refund: false,
  }
}

const schema = computed(() => getProviderSchema(draft.value.provider_key))
const isEdit = computed(() => editorMode.value === 'edit')
const saving = ref(false)
const togglingId = ref<number | null>(null)

function openCreate(): void {
  editorMode.value = 'create'
  draft.value = emptyDraft()
  applySchemaDefaults()
  editorOpen.value = true
}

function openEdit(p: ProviderInstance): void {
  editorMode.value = 'edit'
  draft.value = {
    id: p.id,
    provider_key: p.provider_key,
    name: p.name,
    config: { ...p.config },
    supported_types: [...p.supported_types],
    enabled: p.enabled,
    payment_mode: p.payment_mode,
    sort_order: p.sort_order,
    limits: p.limits || '',
    refund_enabled: p.refund_enabled,
    allow_user_refund: p.allow_user_refund,
  }
  // 敏感字段全部置空，让用户决定是否覆盖
  const sch = getProviderSchema(p.provider_key)
  if (sch) {
    for (const f of sch.fields) {
      if (f.sensitive) draft.value.config[f.key] = ''
    }
  }
  editorOpen.value = true
}

function applySchemaDefaults(): void {
  const sch = getProviderSchema(draft.value.provider_key)
  if (!sch) return
  draft.value.payment_mode = sch.default_payment_mode
  draft.value.supported_types = [...sch.default_supported_types]
  if (!draft.value.name) draft.value.name = sch.display_name
}

function pickProvider(key: PaymentType): void {
  draft.value.provider_key = key
  draft.value.config = {}
  applySchemaDefaults()
}

function closeEditor(): void {
  editorOpen.value = false
}

function fieldValue(key: string): string {
  return draft.value.config[key] ?? ''
}

function setField(key: string, value: string): void {
  draft.value.config[key] = value
}

function toggleType(t: PaymentType): void {
  const set = new Set(draft.value.supported_types)
  if (set.has(t)) set.delete(t)
  else set.add(t)
  draft.value.supported_types = Array.from(set)
}

async function save(): Promise<void> {
  saving.value = true
  error.value = ''
  try {
    if (editorMode.value === 'create') {
      const payload: any = { ...draft.value }
      // 清理空字符串
      const cfg: Record<string, string> = {}
      for (const [k, v] of Object.entries(payload.config)) {
        if (typeof v === 'string' && v.length) cfg[k] = v
      }
      payload.config = cfg
      await paymentAdminApi.createProvider(payload)
    } else {
      const payload: any = {
        name: draft.value.name,
        supported_types: draft.value.supported_types,
        enabled: draft.value.enabled,
        payment_mode: draft.value.payment_mode,
        sort_order: draft.value.sort_order,
        limits: draft.value.limits,
        refund_enabled: draft.value.refund_enabled,
        allow_user_refund: draft.value.allow_user_refund,
      }
      // 只提交非空的 config（敏感字段空字符串 = 不修改）
      const cfg: Record<string, string> = {}
      for (const [k, v] of Object.entries(draft.value.config)) {
        if (typeof v === 'string' && v.length) cfg[k] = v
      }
      if (Object.keys(cfg).length) payload.config = cfg
      await paymentAdminApi.updateProvider(draft.value.id!, payload)
    }
    closeEditor()
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(p: ProviderInstance): Promise<void> {
  togglingId.value = p.id
  try {
    await paymentAdminApi.updateProvider(p.id, { enabled: !p.enabled })
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    togglingId.value = null
  }
}

async function remove(p: ProviderInstance): Promise<void> {
  if (!confirm(t('payment.adminProviders.deleteConfirm'))) return
  try {
    await paymentAdminApi.deleteProvider(p.id)
    await load()
  } catch (e) {
    error.value = getErrorMessage(e)
  }
}

const providerKeyOptions = computed(() => Object.values(PROVIDER_SCHEMAS).map((s) => ({
  key: s.provider_key,
  label: s.display_name,
})))

const modeOptions: { key: PaymentMode; label: string }[] = [
  { key: 'qrcode', label: 'modeQrcode' },
  { key: 'redirect', label: 'modeRedirect' },
  { key: 'popup', label: 'modePopup' },
  { key: 'jsapi', label: 'modeJsapi' },
]

</script>

<template>
  <section class="workspace-page payment-providers">
    <header class="providers-heading">
      <div>
        <div class="providers-eyebrow"><i></i><span>PROVIDERS</span></div>
        <h1>{{ t('payment.adminProviders.title') }}</h1>
        <p>{{ t('payment.adminProviders.description') }}</p>
      </div>
      <div class="providers-heading-actions">
        <button type="button" class="primary-btn" @click="openCreate">
          + {{ t('payment.adminProviders.addProvider') }}
        </button>
      </div>
    </header>

    <p v-if="error" class="error-banner">{{ error }}</p>

    <div v-if="!providers.length && !loading" class="empty-state">
      <strong>{{ t('payment.adminProviders.empty') }}</strong>
    </div>

    <div v-else class="providers-grid">
      <article v-for="p in providers" :key="p.id" class="provider-card" :class="{ disabled: !p.enabled }">
        <header class="provider-card-head">
          <div>
            <span class="provider-badge">{{ getProviderSchema(p.provider_key)?.display_name || p.provider_key }}</span>
            <h3>{{ p.name }}</h3>
          </div>
          <span :class="['provider-toggle', { on: p.enabled, busy: togglingId === p.id }]" @click="toggleEnabled(p)">
            <i></i><span>{{ p.enabled ? t('payment.adminProviders.enabledOn') : t('payment.adminProviders.enabledOff') }}</span>
          </span>
        </header>
        <ul class="provider-meta">
          <li><span>{{ t('payment.adminProviders.formMode') }}</span><b>{{ t('payment.adminProviders.mode' + (p.payment_mode.charAt(0).toUpperCase() + p.payment_mode.slice(1)), p.payment_mode) }}</b></li>
          <li><span>{{ t('payment.adminProviders.formTypes') }}</span><b>{{ p.supported_types.map((x) => t('payment.method.' + x, x)).join(' / ') }}</b></li>
          <li><span>{{ t('payment.adminProviders.colRefund') }}</span><b>{{ p.refund_enabled ? t('payment.adminProviders.refundYes') : t('payment.adminProviders.refundNo') }}</b></li>
          <li><span>{{ t('payment.adminProviders.formUserRefund') }}</span><b>{{ p.allow_user_refund ? t('payment.adminProviders.userRefundYes') : t('payment.adminProviders.userRefundNo') }}</b></li>
          <li><span>{{ t('payment.adminProviders.colSort') }}</span><b>{{ p.sort_order }}</b></li>
        </ul>
        <footer class="provider-card-foot">
          <button type="button" class="action" @click="openEdit(p)">{{ t('payment.edit') }}</button>
          <button type="button" class="action danger" @click="remove(p)">{{ t('payment.delete') }}</button>
        </footer>
      </article>
    </div>

    <div v-if="editorOpen" class="modal-mask" @click.self="closeEditor">
      <div class="modal-card">
        <header><h3>{{ isEdit ? t('payment.adminProviders.editTitle') : t('payment.adminProviders.createTitle') }}</h3>
          <a v-if="schema?.docs_url" class="docs-link" :href="schema.docs_url" target="_blank" rel="noopener">📖 docs</a>
        </header>
        <p v-if="schema?.hint" class="modal-sub">{{ schema.hint }}</p>

        <div class="form-grid">
          <label class="field">
            <span>{{ t('payment.adminProviders.formProvider') }}</span>
            <select :value="draft.provider_key" :disabled="isEdit" @change="pickProvider(($event.target as HTMLSelectElement).value as PaymentType)">
              <option v-for="o in providerKeyOptions" :key="o.key" :value="o.key">{{ o.label }}</option>
            </select>
          </label>
          <label class="field">
            <span>{{ t('payment.adminProviders.formName') }}</span>
            <input v-model="draft.name" type="text" />
          </label>
          <label class="field">
            <span>{{ t('payment.adminProviders.formMode') }}</span>
            <select v-model="draft.payment_mode">
              <option v-for="m in modeOptions" :key="m.key" :value="m.key">{{ t('payment.adminProviders.' + m.label) }}</option>
            </select>
          </label>
          <label class="field">
            <span>{{ t('payment.adminProviders.formSort') }}</span>
            <input v-model.number="draft.sort_order" type="number" />
          </label>
          <label class="field full">
            <span>{{ t('payment.adminProviders.formTypes') }}</span>
            <div class="type-chips">
              <button
                v-for="t in schema?.supported_payment_types || []"
                :key="t"
                type="button"
                :class="['chip', { active: draft.supported_types.includes(t) }]"
                @click="toggleType(t)"
              >
                {{ $t('payment.method.' + t, t) }}
              </button>
            </div>
          </label>
          <label class="field full">
            <span>{{ t('payment.adminProviders.formLimits') }}</span>
            <input v-model="draft.limits" type="text" :placeholder="t('payment.adminProviders.formLimitsHint')" />
          </label>
        </div>

        <h4 class="config-heading">{{ t('payment.adminProviders.formConfig') }}</h4>
        <p class="form-hint">{{ t('payment.adminProviders.formConfigHint') }}</p>
        <div class="form-grid">
          <template v-for="f in schema?.fields || []" :key="f.key">
            <label class="field full">
              <span>
                {{ f.label }}
                <em v-if="f.required" class="req">*</em>
                <em v-if="f.protectedWhenPending && isEdit" class="protected">🔒</em>
              </span>
              <textarea
                v-if="f.type === 'pem'"
                :value="fieldValue(f.key)"
                :placeholder="isEdit && f.sensitive ? t('payment.adminProviders.secretPlaceholder') : (f.placeholder || '')"
                rows="4"
                @input="setField(f.key, ($event.target as HTMLTextAreaElement).value)"
              ></textarea>
              <input
                v-else
                :type="f.type === 'number' ? 'number' : (f.type === 'secret' ? 'password' : 'text')"
                :value="fieldValue(f.key)"
                :placeholder="isEdit && f.sensitive ? t('payment.adminProviders.secretPlaceholder') : (f.placeholder || '')"
                @input="setField(f.key, ($event.target as HTMLInputElement).value)"
              />
              <small v-if="f.description">{{ f.description }}</small>
              <small v-if="isEdit && f.sensitive" class="secret-hint">{{ t('payment.adminProviders.secretReplaceHint') }}</small>
            </label>
          </template>
        </div>

        <div class="form-grid">
          <label class="checkbox"><input v-model="draft.enabled" type="checkbox" /><span>{{ t('payment.adminProviders.formEnabled') }}</span></label>
          <label class="checkbox"><input v-model="draft.refund_enabled" type="checkbox" /><span>{{ t('payment.adminProviders.formRefund') }}</span></label>
          <label class="checkbox"><input v-model="draft.allow_user_refund" type="checkbox" /><span>{{ t('payment.adminProviders.formUserRefund') }}</span></label>
        </div>

        <footer class="modal-actions">
          <button type="button" class="btn ghost" :disabled="saving" @click="closeEditor">{{ t('payment.cancel') }}</button>
          <button type="button" class="btn primary" :disabled="saving" @click="save">
            {{ saving ? t('payment.saving') : t('payment.save') }}
          </button>
        </footer>
      </div>
    </div>
  </section>
</template>

<style scoped>
.payment-providers { width: 100%; max-width: 1280px; margin: 0 auto; padding: 12px 0 44px; }
.providers-heading { display: flex; align-items: center; justify-content: space-between; min-height: 92px; margin-bottom: 22px; gap: 24px; }
.providers-heading h1 { margin: 0; color: #f7f8fa; font-size: clamp(1.9rem, 2.4vw, 2.35rem); line-height: 1.08; font-weight: 680; letter-spacing: -.043em; }
.providers-heading p { max-width: 640px; margin: 10px 0 0; color: #858d97; font-size: .88rem; line-height: 1.6; }
.providers-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: #6ec0f5; font: 700 .67rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .13em; }
.providers-eyebrow i { width: 5px; height: 5px; border-radius: 50%; background: #6ec0f5; }
.primary-btn { padding: 8px 16px; border-radius: 8px; background: #4a93c5; border: 1px solid #4a93c5; color: #0d0f12; font: 600 .82rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.primary-btn:hover { background: #5fa3d5; border-color: #5fa3d5; }
.error-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(239, 68, 68, .12); border: 1px solid rgba(239, 68, 68, .35); color: #fca5a5; font-size: .85rem; }
.empty-state { padding: 56px 0; text-align: center; color: #6c727b; font-size: .85rem; }
.providers-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 14px; }
.provider-card { padding: 20px 22px; background: #11141a; border: 1px solid #1d2128; border-radius: 14px; }
.provider-card.disabled { opacity: .65; }
.provider-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 14px; }
.provider-card-head h3 { margin: 6px 0 0; color: #f7f8fa; font-size: 1.05rem; font-weight: 600; }
.provider-badge { display: inline-block; padding: 3px 8px; border-radius: 4px; background: rgba(110, 192, 245, .12); color: #6ec0f5; font: 600 .68rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .04em; }
.provider-toggle { display: inline-flex; align-items: center; gap: 6px; padding: 5px 10px; border-radius: 14px; background: #16191f; border: 1px solid #2a2f37; color: #858d97; font: 500 .72rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.provider-toggle.on { color: #48bb99; border-color: rgba(72, 187, 153, .35); background: rgba(72, 187, 153, .08); }
.provider-toggle.busy { opacity: .5; }
.provider-toggle i { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.provider-meta { list-style: none; padding: 0; margin: 0 0 14px; display: grid; grid-template-columns: repeat(2, 1fr); gap: 6px 12px; }
.provider-meta li { display: flex; justify-content: space-between; gap: 8px; padding: 4px 0; font-size: .78rem; }
.provider-meta li span { color: #6c727b; }
.provider-meta li b { color: #d6dbe1; font-weight: 500; }
.provider-card-foot { display: flex; gap: 8px; padding-top: 12px; border-top: 1px solid #1d2128; }
.action { padding: 5px 12px; border-radius: 6px; background: #16191f; border: 1px solid #2a2f37; color: #d6dbe1; font: 500 .76rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.action:hover { border-color: #3d4754; }
.action.danger { color: #fca5a5; border-color: rgba(239, 68, 68, .35); }
.action.danger:hover { background: rgba(239, 68, 68, .08); }
.modal-mask { position: fixed; inset: 0; background: rgba(8, 10, 14, .75); display: flex; align-items: flex-start; justify-content: center; z-index: 1000; overflow-y: auto; padding: 40px 16px; }
.modal-card { width: min(720px, 100%); padding: 26px 28px; background: #11141a; border: 1px solid #1d2128; border-radius: 14px; }
.modal-card header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.modal-card h3 { margin: 0; color: #f7f8fa; font-size: 1.1rem; font-weight: 600; }
.docs-link { color: #6ec0f5; font-size: .78rem; text-decoration: none; }
.modal-sub { margin: 0 0 18px; padding: 10px 12px; background: rgba(110, 192, 245, .08); border: 1px solid rgba(110, 192, 245, .25); border-radius: 8px; color: #b8bfc7; font-size: .78rem; line-height: 1.5; }
.config-heading { margin: 18px 0 6px; color: #f7f8fa; font-size: .9rem; font-weight: 600; }
.form-hint { margin: 0 0 12px; color: #6c727b; font-size: .72rem; }
.form-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; margin-bottom: 6px; }
.field { display: block; }
.field.full { grid-column: 1 / -1; }
.field > span { display: block; margin-bottom: 5px; color: #b8bfc7; font-size: .78rem; }
.req { color: #fca5a5; font-style: normal; margin-left: 2px; }
.protected { font-style: normal; margin-left: 4px; opacity: .7; }
.field input, .field textarea, .field select { width: 100%; padding: 8px 12px; border-radius: 7px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .82rem/1.4 ui-sans-serif, system-ui, sans-serif; box-sizing: border-box; }
.field input:focus, .field textarea:focus, .field select:focus { outline: none; border-color: #4a93c5; }
.field small { display: block; margin-top: 4px; color: #6c727b; font-size: .68rem; line-height: 1.4; }
.field small.secret-hint { color: #6ec0f5; }
.type-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.chip { padding: 5px 12px; border-radius: 14px; background: #16191f; border: 1px solid #2a2f37; color: #858d97; font: 500 .76rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.chip.active { color: #6ec0f5; background: rgba(110, 192, 245, .1); border-color: rgba(110, 192, 245, .35); }
.checkbox { display: inline-flex; align-items: center; gap: 8px; margin: 0 16px 0 0; color: #b8bfc7; font-size: .82rem; cursor: pointer; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
.btn { padding: 9px 20px; border-radius: 7px; border: 1px solid transparent; font: 500 .85rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.btn.ghost { background: #16191f; border-color: #2a2f37; color: #d6dbe1; }
.btn.ghost:hover:not(:disabled) { border-color: #3d4754; }
.btn.primary { background: #4a93c5; color: #0d0f12; }
.btn.primary:hover:not(:disabled) { background: #5fa3d5; }
.btn:disabled { opacity: .6; cursor: not-allowed; }
@media (max-width: 720px) { .providers-heading { flex-direction: column; align-items: flex-start; } .form-grid { grid-template-columns: 1fr; } }
</style>


function fieldDef(key: string): ProviderFieldDef | undefined {
  return schema.value?.fields.find((f) => f.key === key)
}