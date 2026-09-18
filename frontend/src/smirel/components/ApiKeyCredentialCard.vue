<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'

interface ApiKeyGroup {
  id?: number
  name?: string
  platform?: string
}

interface ApiKeyItem {
  id: number
  name?: string
  key?: string
  status?: string
  group_id?: number | null
  group?: ApiKeyGroup | null
  quota?: number
  quota_used?: number
  current_concurrency?: number
  last_used_at?: string | null
  expires_at?: string | null
  created_at?: string
  [key: string]: unknown
}

const props = defineProps<{ item: ApiKeyItem }>()
const emit = defineEmits<{ remove: [id: number] }>()
const { locale, t } = useI18n()

const revealed = ref(false)
const copied = ref(false)
let copiedTimer: number | undefined

const labels = computed(() => locale.value === 'zh-CN'
  ? {
      show: '显示', hide: '隐藏', copy: '复制', copied: '已复制', clickCopy: '点击复制完整密钥',
      group: '分组', unbound: '未绑定分组', quota: '额度', unlimited: '不限额', concurrency: '当前并发',
      lastUsed: '最近使用', neverUsed: '尚未使用',
    }
  : {
      show: 'Show', hide: 'Hide', copy: 'Copy', copied: 'Copied', clickCopy: 'Click to copy full key',
      group: 'Group', unbound: 'No group', quota: 'Quota', unlimited: 'Unlimited', concurrency: 'Concurrency',
      lastUsed: 'Last used', neverUsed: 'Never used',
    })

const rawKey = computed(() => String(props.item.key || ''))
const serverMasked = computed(() => /[•*]{3,}/.test(rawKey.value))
const canReveal = computed(() => Boolean(rawKey.value) && !serverMasked.value)
const hasGroup = computed(() => props.item.group_id != null || Boolean(props.item.group?.name))
const groupLabel = computed(() => props.item.group?.name || (props.item.group_id != null ? `#${props.item.group_id}` : labels.value.unbound))
const quotaLabel = computed(() => {
  const limit = Number(props.item.quota || 0)
  if (limit <= 0) return labels.value.unlimited
  const used = Number(props.item.quota_used || 0)
  return `$${used.toFixed(2)} / $${limit.toFixed(2)}`
})
const concurrencyLabel = computed(() => String(Number(props.item.current_concurrency || 0)))
const lastUsedLabel = computed(() => {
  if (!props.item.last_used_at) return labels.value.neverUsed
  const date = new Date(props.item.last_used_at)
  if (Number.isNaN(date.getTime())) return String(props.item.last_used_at)
  return new Intl.DateTimeFormat(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false,
  }).format(date)
})

const maskedKey = computed(() => {
  const value = rawKey.value
  if (!value) return '••••••••'
  if (serverMasked.value) return value
  if (value.length <= 12) return '••••••••'
  const middleLength = Math.min(24, Math.max(10, value.length - 11))
  return `${value.slice(0, 7)}${'•'.repeat(middleLength)}${value.slice(-4)}`
})

const displayKey = computed(() => revealed.value && canReveal.value ? rawKey.value : maskedKey.value)

async function writeClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // Fall through to the textarea path for restricted clipboard contexts.
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

async function copyKey() {
  if (!rawKey.value) return
  await writeClipboard(rawKey.value)
  copied.value = true
  if (copiedTimer) window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => { copied.value = false }, 1600)
}

function toggleReveal() {
  if (!canReveal.value) return
  revealed.value = !revealed.value
}

onBeforeUnmount(() => {
  if (copiedTimer) window.clearTimeout(copiedTimer)
})
</script>

<template>
  <article class="api-key-card" :class="{ 'api-key-card--unbound': !hasGroup }">
    <header class="api-key-card-head">
      <div class="api-key-identity">
        <span class="api-key-mark" aria-hidden="true">
          <svg viewBox="0 0 24 24">
            <circle cx="8" cy="15" r="3.5" />
            <path d="m10.7 12.3 7.6-7.6M15.7 7.3l2 2M13.6 9.4l2 2" />
          </svg>
        </span>
        <div>
          <strong>{{ item.name || `Key #${item.id}` }}</strong>
          <small>API KEY</small>
        </div>
      </div>
      <span class="api-key-state" :class="{ 'is-inactive': item.status && item.status !== 'active' }"><i></i>{{ item.status || 'active' }}</span>
    </header>

    <div class="api-key-secret">
      <div class="api-key-secret-head">
        <span>{{ t('workspace.key') }}</span>
        <div class="api-key-secret-actions">
          <button
            v-if="canReveal"
            type="button"
            :aria-label="revealed ? labels.hide : labels.show"
            :title="revealed ? labels.hide : labels.show"
            @click="toggleReveal"
          >
            <svg v-if="!revealed" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z" />
              <circle cx="12" cy="12" r="2.8" />
            </svg>
            <svg v-else viewBox="0 0 24 24" aria-hidden="true">
              <path d="m3 3 18 18" />
              <path d="M10.6 6.2A11 11 0 0 1 12 6c6.5 0 10 6 10 6a17 17 0 0 1-2.1 2.8" />
              <path d="M6.6 6.6C3.6 8.3 2 12 2 12s3.5 6 10 6a10.8 10.8 0 0 0 4.3-.9" />
              <path d="M9.9 9.9A3 3 0 0 0 14.1 14.1" />
            </svg>
            <span>{{ revealed ? labels.hide : labels.show }}</span>
          </button>
          <button
            class="api-key-copy-button"
            :class="{ copied }"
            type="button"
            :aria-label="copied ? labels.copied : labels.copy"
            :title="copied ? labels.copied : labels.copy"
            @click="copyKey"
          >
            <svg v-if="!copied" viewBox="0 0 24 24" aria-hidden="true">
              <rect x="9" y="9" width="11" height="11" rx="2" />
              <path d="M15 9V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h3" />
            </svg>
            <svg v-else viewBox="0 0 24 24" aria-hidden="true">
              <path d="m5 12 4 4L19 6" />
            </svg>
            <span>{{ copied ? labels.copied : labels.copy }}</span>
          </button>
        </div>
      </div>

      <button class="api-key-value" type="button" :title="labels.clickCopy" @click="copyKey">
        <code>{{ displayKey }}</code>
      </button>
    </div>

    <dl class="api-key-meta">
      <div :class="{ warning: !hasGroup }">
        <dt>{{ labels.group }}</dt>
        <dd>{{ groupLabel }}</dd>
      </div>
      <div>
        <dt>{{ labels.quota }}</dt>
        <dd>{{ quotaLabel }}</dd>
      </div>
      <div>
        <dt>{{ labels.concurrency }}</dt>
        <dd>{{ concurrencyLabel }}</dd>
      </div>
      <div>
        <dt>{{ labels.lastUsed }}</dt>
        <dd>{{ lastUsedLabel }}</dd>
      </div>
    </dl>

    <footer class="api-key-card-foot">
      <span class="api-key-created">
        <b>{{ t('workspace.createdAt') }}</b>
        {{ item.created_at || '—' }}
      </span>
      <button class="api-key-delete" type="button" @click="emit('remove', item.id)">{{ t('workspace.delete') }}</button>
    </footer>
  </article>
</template>

<style scoped>
.api-key-secret-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.api-key-secret-head > span {
  color: #69727e;
  font-size: .66rem;
  font-weight: 720;
  letter-spacing: .1em;
}

.api-key-secret-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.api-key-secret-actions button {
  min-height: 30px;
  padding: 0 9px;
  border: 1px solid #272d35;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: #89929d;
  background: #101318;
  cursor: pointer;
  font: inherit;
  font-size: .69rem;
  font-weight: 600;
  transition: border-color .15s ease, color .15s ease, background-color .15s ease;
}

.api-key-secret-actions button:hover {
  border-color: #3a424c;
  color: #dfe5ea;
  background: #151920;
}

.api-key-secret-actions button:focus-visible,
.api-key-value:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(47, 150, 232, .10);
}

.api-key-secret-actions button svg {
  width: 14px;
  height: 14px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.api-key-secret-actions .api-key-copy-button.copied {
  border-color: #275044;
  color: #74d7b0;
  background: #101b17;
}

.api-key-value {
  width: 100%;
  min-width: 0;
  margin-top: 8px;
  padding: 3px 0 0;
  border: 0;
  border-radius: 5px;
  display: block;
  overflow: hidden;
  color: inherit;
  background: transparent;
  cursor: copy;
  text-align: left;
}

.api-key-value code {
  margin-top: 0 !important;
  color: #b6c4d2 !important;
  font-size: .85rem !important;
  transition: color .15s ease;
}

.api-key-value:hover code {
  color: #d9e8f4 !important;
}

.api-key-meta {
  margin: 14px 0 0;
  padding: 12px 0 0;
  border-top: 1px solid #20252c;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 18px;
}

.api-key-meta div {
  min-width: 0;
}

.api-key-meta dt {
  color: #59636e;
  font-size: .59rem;
  font-weight: 680;
  letter-spacing: .06em;
}

.api-key-meta dd {
  margin: 3px 0 0;
  overflow: hidden;
  color: #9aa5b0;
  font-size: .69rem;
  font-weight: 560;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-meta .warning dd {
  color: #c9a76a;
}

.api-key-card--unbound {
  border-color: #3b3529 !important;
}

.api-key-state.is-inactive {
  border-color: #3d3032 !important;
  color: #aa8589 !important;
  background: #171113 !important;
}

.api-key-state.is-inactive i {
  background: #b96f78 !important;
}

@media (max-width: 560px) {
  .api-key-secret-head {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .api-key-secret-actions {
    width: 100%;
  }

  .api-key-secret-actions button {
    flex: 1 1 0;
  }

  .api-key-meta {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .api-key-secret-actions button,
  .api-key-value code {
    transition: none;
  }
}
</style>