<script setup lang="ts">
// Admin Payment Config —— 平台级支付参数 / 限额 / 展示策略
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  paymentAdminApi,
  type PaymentConfig,
  type PaymentConfigUpdateRequest,
  type PaymentType,
  type LoadBalanceStrategy,
  type VisibleMethodSource,
} from '../api/payment'
import { getErrorMessage } from '../core/api'

const { t } = useI18n()

const loading = ref(false)
const error = ref('')
const success = ref('')
const original = ref<PaymentConfig | null>(null)

const draft = reactive<PaymentConfigUpdateRequest>({
  enabled: true,
  min_amount: 0,
  max_amount: 0,
  daily_limit: 0,
  order_timeout_minutes: 0,
  max_pending_orders: 0,
  enabled_payment_types: [],
  balance_disabled: false,
  balance_recharge_multiplier: 1,
  subscription_usd_to_cny_rate: 0,
  recharge_fee_rate: 0,
  load_balance_strategy: undefined as LoadBalanceStrategy | undefined,
  product_name_prefix: '',
  product_name_suffix: '',
  help_image_url: '',
  help_text: '',
  cancel_rate_limit_enabled: false,
  cancel_rate_limit_max: 0,
  cancel_rate_limit_window: 0,
  cancel_rate_limit_unit: 'minute',
  cancel_rate_limit_window_mode: 'fixed',
  alipay_force_qrcode: false,
  alipay_mobile_precreate_deep_link: false,
  payment_visible_method_alipay_source: 'official_alipay',
  payment_visible_method_wxpay_source: 'official_wxpay',
  payment_visible_method_alipay_enabled: true,
  payment_visible_method_wxpay_enabled: true,
})

const validationErrors = ref<string[]>([])

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const c = (await paymentAdminApi.getConfig()) as PaymentConfig
    original.value = c
    Object.assign(draft, {
      enabled: c.enabled,
      min_amount: c.min_amount,
      max_amount: c.max_amount,
      daily_limit: c.daily_limit,
      order_timeout_minutes: c.order_timeout_minutes,
      max_pending_orders: c.max_pending_orders,
      enabled_payment_types: [...(c.enabled_payment_types || [])],
      balance_disabled: c.balance_disabled,
      balance_recharge_multiplier: c.balance_recharge_multiplier,
      subscription_usd_to_cny_rate: c.subscription_usd_to_cny_rate,
      recharge_fee_rate: c.recharge_fee_rate,
      load_balance_strategy: c.load_balance_strategy || '',
      product_name_prefix: c.product_name_prefix || '',
      product_name_suffix: c.product_name_suffix || '',
      help_image_url: c.help_image_url || '',
      help_text: c.help_text || '',
      cancel_rate_limit_enabled: c.cancel_rate_limit_enabled,
      cancel_rate_limit_max: c.cancel_rate_limit_max,
      cancel_rate_limit_window: c.cancel_rate_limit_window,
      cancel_rate_limit_unit: c.cancel_rate_limit_unit || 'minute',
      cancel_rate_limit_window_mode: c.cancel_rate_limit_window_mode || 'fixed',
      alipay_force_qrcode: c.alipay_force_qrcode,
      alipay_mobile_precreate_deep_link: c.alipay_mobile_precreate_deep_link,
      payment_visible_method_alipay_source: c.payment_visible_method_alipay_source || 'official_alipay',
      payment_visible_method_wxpay_source: c.payment_visible_method_wxpay_source || 'official_wxpay',
      payment_visible_method_alipay_enabled: c.payment_visible_method_alipay_enabled ?? true,
      payment_visible_method_wxpay_enabled: c.payment_visible_method_wxpay_enabled ?? true,
    })
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

function validate(): boolean {
  const errs: string[] = []
  if ((draft.subscription_usd_to_cny_rate || 0) < 0) errs.push(t('payment.adminConfig.validation.rateRange'))
  if ((draft.recharge_fee_rate || 0) < 0 || (draft.recharge_fee_rate || 0) > 100) errs.push(t('payment.adminConfig.validation.feeRange'))
  if ((draft.balance_recharge_multiplier || 0) <= 0) errs.push(t('payment.adminConfig.validation.multiplier'))
  validationErrors.value = errs
  return errs.length === 0
}

const saving = ref(false)

async function save(): Promise<void> {
  if (!validate()) return
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const payload: PaymentConfigUpdateRequest = { ...draft }
    if (!payload.load_balance_strategy) delete (payload as any).load_balance_strategy
    await paymentAdminApi.updateConfig(payload)
    success.value = t('payment.adminConfig.saveSuccess')
    setTimeout(() => { success.value = '' }, 2500)
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    saving.value = false
  }
}

const paymentTypeOptions: PaymentType[] = ['alipay', 'wxpay', 'easypay', 'stripe', 'airwallex']

function toggleType(t: PaymentType): void {
  const set = new Set(draft.enabled_payment_types || [])
  if (set.has(t)) set.delete(t)
  else set.add(t)
  draft.enabled_payment_types = Array.from(set)
}

const strategyOptions: { value: LoadBalanceStrategy | ''; label: string }[] = [
  { value: '', label: 'payment.adminConfig.sourceOfficial' /* fallback */ },
  { value: 'round-robin', label: 'Round-robin' },
  { value: 'random', label: 'Random' },
  { value: 'weighted', label: 'Weighted' },
]

const sourceOptions: VisibleMethodSource[] = ['official_alipay', 'easypay_alipay', 'official_wxpay', 'easypay_wxpay']

const unitOptions = ['second', 'minute', 'hour', 'day']
const modeOptions = ['fixed', 'sliding']

function isDirty(): boolean {
  if (!original.value) return false
  return JSON.stringify(draft) !== JSON.stringify({
    enabled: original.value.enabled,
    min_amount: original.value.min_amount,
    max_amount: original.value.max_amount,
    daily_limit: original.value.daily_limit,
    order_timeout_minutes: original.value.order_timeout_minutes,
    max_pending_orders: original.value.max_pending_orders,
    enabled_payment_types: [...(original.value.enabled_payment_types || [])],
    balance_disabled: original.value.balance_disabled,
    balance_recharge_multiplier: original.value.balance_recharge_multiplier,
    subscription_usd_to_cny_rate: original.value.subscription_usd_to_cny_rate,
    recharge_fee_rate: original.value.recharge_fee_rate,
    load_balance_strategy: (original.value.load_balance_strategy || undefined) as LoadBalanceStrategy | undefined,
    product_name_prefix: original.value.product_name_prefix || '',
    product_name_suffix: original.value.product_name_suffix || '',
    help_image_url: original.value.help_image_url || '',
    help_text: original.value.help_text || '',
    cancel_rate_limit_enabled: original.value.cancel_rate_limit_enabled,
    cancel_rate_limit_max: original.value.cancel_rate_limit_max,
    cancel_rate_limit_window: original.value.cancel_rate_limit_window,
    cancel_rate_limit_unit: original.value.cancel_rate_limit_unit || 'minute',
    cancel_rate_limit_window_mode: original.value.cancel_rate_limit_window_mode || 'fixed',
    alipay_force_qrcode: original.value.alipay_force_qrcode,
    alipay_mobile_precreate_deep_link: original.value.alipay_mobile_precreate_deep_link,
    payment_visible_method_alipay_source: original.value.payment_visible_method_alipay_source || 'official_alipay',
    payment_visible_method_wxpay_source: original.value.payment_visible_method_wxpay_source || 'official_wxpay',
    payment_visible_method_alipay_enabled: original.value.payment_visible_method_alipay_enabled ?? true,
    payment_visible_method_wxpay_enabled: original.value.payment_visible_method_wxpay_enabled ?? true,
  })
}

const dirty = computed(isDirty)
</script>

<template>
  <section class="workspace-page payment-config">
    <header class="config-heading">
      <div>
        <div class="config-eyebrow"><i></i><span>CONFIG</span></div>
        <h1>{{ t('payment.adminConfig.title') }}</h1>
        <p>{{ t('payment.adminConfig.description') }}</p>
      </div>
      <div class="config-heading-actions">
        <button type="button" class="primary-btn" :disabled="saving || !dirty" @click="save">
          {{ saving ? t('payment.saving') : t('payment.save') }}
        </button>
      </div>
    </header>

    <p v-if="error" class="error-banner">{{ error }}</p>
    <p v-if="success" class="success-banner">{{ success }}</p>
    <ul v-if="validationErrors.length" class="error-banner">
      <li v-for="(e, i) in validationErrors" :key="i">{{ e }}</li>
    </ul>

    <div v-if="loading" class="empty-state">{{ t('payment.loading') }}</div>

    <div v-else class="config-sections">
      <section class="config-card">
        <h2>{{ t('payment.adminConfig.sectionGeneral') }}</h2>
        <label class="checkbox"><input v-model="draft.enabled" type="checkbox" /><span>{{ t('payment.adminConfig.fieldEnabled') }}</span></label>
        <label class="field"><span>{{ t('payment.adminConfig.fieldEnabledTypes') }}</span>
          <div class="type-chips">
            <button
              v-for="pt in paymentTypeOptions"
              :key="pt"
              type="button"
              :class="['chip', { active: (draft.enabled_payment_types || []).includes(pt) }]"
              @click="toggleType(pt)"
            >
              {{ t('payment.method.' + pt, pt) }}
            </button>
          </div>
        </label>
        <div class="form-grid">
          <label class="field"><span>{{ t('payment.adminConfig.fieldOrderTimeout') }}</span><input v-model.number="draft.order_timeout_minutes" type="number" min="1" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldMaxPending') }}</span><input v-model.number="draft.max_pending_orders" type="number" min="1" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldBalanceMultiplier') }}</span><input v-model.number="draft.balance_recharge_multiplier" type="number" step="0.01" min="0.01" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldRechargeFeeRate') }}</span><input v-model.number="draft.recharge_fee_rate" type="number" step="0.1" min="0" max="100" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldSubscriptionRate') }}</span><input v-model.number="draft.subscription_usd_to_cny_rate" type="number" step="0.01" min="0" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldLoadBalanceStrategy') }}</span>
            <select v-model="draft.load_balance_strategy">
              <option v-for="o in strategyOptions" :key="o.value || 'none'" :value="o.value">{{ o.label }}</option>
            </select>
          </label>
        </div>
        <label class="checkbox"><input v-model="draft.balance_disabled" type="checkbox" /><span>{{ t('payment.adminConfig.fieldBalanceDisabled') }}</span></label>
      </section>

      <section class="config-card">
        <h2>{{ t('payment.adminConfig.sectionLimits') }}</h2>
        <div class="form-grid">
          <label class="field"><span>{{ t('payment.adminConfig.fieldMinAmount') }}</span><input v-model.number="draft.min_amount" type="number" step="0.01" min="0" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldMaxAmount') }}</span><input v-model.number="draft.max_amount" type="number" step="0.01" min="0" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldDailyLimit') }}</span><input v-model.number="draft.daily_limit" type="number" step="0.01" min="0" /></label>
        </div>
        <div class="form-grid">
          <label class="field"><span>{{ t('payment.adminConfig.fieldProductNamePrefix') }}</span><input v-model="draft.product_name_prefix" type="text" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldProductNameSuffix') }}</span><input v-model="draft.product_name_suffix" type="text" /></label>
        </div>
        <label class="checkbox"><input v-model="draft.alipay_force_qrcode" type="checkbox" /><span>{{ t('payment.adminConfig.fieldAlipayForceQrcode') }}</span></label>
        <label class="checkbox"><input v-model="draft.alipay_mobile_precreate_deep_link" type="checkbox" /><span>{{ t('payment.adminConfig.fieldAlipayMobileDeepLink') }}</span></label>
      </section>

      <section class="config-card">
        <h2>{{ t('payment.adminConfig.sectionCancel') }}</h2>
        <label class="checkbox"><input v-model="draft.cancel_rate_limit_enabled" type="checkbox" /><span>{{ t('payment.adminConfig.fieldCancelEnabled') }}</span></label>
        <div class="form-grid">
          <label class="field"><span>{{ t('payment.adminConfig.fieldCancelMax') }}</span><input v-model.number="draft.cancel_rate_limit_max" type="number" min="1" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldCancelWindow') }}</span><input v-model.number="draft.cancel_rate_limit_window" type="number" min="1" /></label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldCancelUnit') }}</span>
            <select v-model="draft.cancel_rate_limit_unit">
              <option v-for="u in unitOptions" :key="u" :value="u">{{ u }}</option>
            </select>
          </label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldCancelMode') }}</span>
            <select v-model="draft.cancel_rate_limit_window_mode">
              <option v-for="m in modeOptions" :key="m" :value="m">{{ m }}</option>
            </select>
          </label>
        </div>
      </section>

      <section class="config-card">
        <h2>{{ t('payment.adminConfig.sectionVisibleMethod') }}</h2>
        <div class="form-grid">
          <label class="checkbox"><input v-model="draft.payment_visible_method_alipay_enabled" type="checkbox" /><span>{{ t('payment.adminConfig.fieldAlipayVisible') }}</span></label>
          <label class="checkbox"><input v-model="draft.payment_visible_method_wxpay_enabled" type="checkbox" /><span>{{ t('payment.adminConfig.fieldWxpayVisible') }}</span></label>
        </div>
        <div class="form-grid">
          <label class="field"><span>{{ t('payment.adminConfig.fieldAlipaySource') }}</span>
            <select v-model="draft.payment_visible_method_alipay_source">
              <option v-for="s in sourceOptions.filter((x) => x.includes('alipay'))" :key="s" :value="s">
                {{ s === 'official_alipay' ? t('payment.adminConfig.sourceOfficial') : t('payment.adminConfig.sourceEasypay') }}
              </option>
            </select>
          </label>
          <label class="field"><span>{{ t('payment.adminConfig.fieldWxpaySource') }}</span>
            <select v-model="draft.payment_visible_method_wxpay_source">
              <option v-for="s in sourceOptions.filter((x) => x.includes('wxpay'))" :key="s" :value="s">
                {{ s === 'official_wxpay' ? t('payment.adminConfig.sourceOfficial') : t('payment.adminConfig.sourceEasypay') }}
              </option>
            </select>
          </label>
        </div>
      </section>

      <section class="config-card">
        <h2>{{ t('payment.adminConfig.sectionHelp') }}</h2>
        <label class="field"><span>{{ t('payment.adminConfig.fieldHelpImage') }}</span><input v-model="draft.help_image_url" type="url" /></label>
        <label class="field full"><span>{{ t('payment.adminConfig.fieldHelpText') }}</span><textarea v-model="draft.help_text" rows="4" /></label>
      </section>
    </div>
  </section>
</template>

<style scoped>
.payment-config { width: 100%; max-width: 1080px; margin: 0 auto; padding: 12px 0 44px; }
.config-heading { display: flex; align-items: center; justify-content: space-between; min-height: 92px; margin-bottom: 22px; gap: 24px; }
.config-heading h1 { margin: 0; color: #f7f8fa; font-size: clamp(1.9rem, 2.4vw, 2.35rem); line-height: 1.08; font-weight: 680; letter-spacing: -.043em; }
.config-heading p { max-width: 640px; margin: 10px 0 0; color: #858d97; font-size: .88rem; line-height: 1.6; }
.config-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: #6ec0f5; font: 700 .67rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .13em; }
.config-eyebrow i { width: 5px; height: 5px; border-radius: 50%; background: #6ec0f5; }
.primary-btn { padding: 8px 18px; border-radius: 8px; background: #4a93c5; border: 1px solid #4a93c5; color: #0d0f12; font: 600 .82rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.primary-btn:hover:not(:disabled) { background: #5fa3d5; border-color: #5fa3d5; }
.primary-btn:disabled { opacity: .4; cursor: not-allowed; }
.error-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(239, 68, 68, .12); border: 1px solid rgba(239, 68, 68, .35); color: #fca5a5; font-size: .85rem; }
.success-banner { margin: 0 0 16px; padding: 10px 14px; border-radius: 8px; background: rgba(72, 187, 153, .12); border: 1px solid rgba(72, 187, 153, .35); color: #48bb99; font-size: .85rem; }
.config-sections { display: flex; flex-direction: column; gap: 14px; }
.config-card { padding: 22px 24px; background: #11141a; border: 1px solid #1d2128; border-radius: 14px; }
.config-card h2 { margin: 0 0 16px; color: #f7f8fa; font-size: 1rem; font-weight: 600; letter-spacing: -.01em; }
.form-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 10px; }
.field { display: block; margin-bottom: 6px; }
.field.full { grid-column: 1 / -1; }
.field > span { display: block; margin-bottom: 5px; color: #b8bfc7; font-size: .78rem; }
.field input, .field textarea, .field select { width: 100%; padding: 8px 12px; border-radius: 7px; background: #0d0f12; border: 1px solid #2a2f37; color: #f7f8fa; font: 400 .82rem/1.4 ui-sans-serif, system-ui, sans-serif; box-sizing: border-box; }
.field input:focus, .field textarea:focus, .field select:focus { outline: none; border-color: #4a93c5; }
.checkbox { display: inline-flex; align-items: center; gap: 8px; margin: 4px 24px 12px 0; color: #b8bfc7; font-size: .85rem; cursor: pointer; }
.type-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.chip { padding: 6px 14px; border-radius: 14px; background: #16191f; border: 1px solid #2a2f37; color: #858d97; font: 500 .78rem/1 ui-sans-serif, system-ui, sans-serif; cursor: pointer; }
.chip.active { color: #6ec0f5; background: rgba(110, 192, 245, .1); border-color: rgba(110, 192, 245, .35); }
.empty-state { padding: 56px 0; text-align: center; color: #6c727b; font-size: .85rem; }
@media (max-width: 900px) { .config-heading { flex-direction: column; align-items: flex-start; } .form-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 560px) { .form-grid { grid-template-columns: 1fr; } }
</style>
