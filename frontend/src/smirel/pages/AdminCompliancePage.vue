<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, getErrorMessage } from '../core/api'

interface ComplianceStatus {
  required: boolean
  version: string
  document_url_zh: string
  document_url_en: string
  ack_phrase_zh: string
  ack_phrase_en: string
}

const route = useRoute()
const router = useRouter()
const status = ref<ComplianceStatus | null>(null)
const loading = ref(true)
const submitting = ref(false)
const errorMessage = ref('')
const phrase = ref('')
const reviewed = ref(false)

function safeRedirect(): string {
  const raw = typeof route.query.redirect === 'string' ? route.query.redirect.trim() : ''
  if (!raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/admin/compliance')) return '/admin/dashboard'
  return raw
}

const canAccept = computed(() => {
  return Boolean(
    reviewed.value
    && status.value
    && phrase.value.trim() === status.value.ack_phrase_zh,
  )
})

async function loadStatus() {
  loading.value = true
  errorMessage.value = ''
  try {
    const { data } = await api.get<ComplianceStatus>('/admin/compliance')
    status.value = data
    if (!data.required) await router.replace(safeRedirect())
  } catch (error) {
    errorMessage.value = getErrorMessage(error)
  } finally {
    loading.value = false
  }
}

async function accept() {
  if (!canAccept.value) return
  submitting.value = true
  errorMessage.value = ''
  try {
    await api.post('/admin/compliance/accept', {
      phrase: phrase.value.trim(),
      language: 'zh',
    })
    await router.replace(safeRedirect())
  } catch (error) {
    errorMessage.value = getErrorMessage(error)
  } finally {
    submitting.value = false
  }
}

onMounted(loadStatus)
</script>

<template>
  <main class="compliance-page">
    <section class="compliance-card">
      <div class="eyebrow">ADMINISTRATOR COMPLIANCE</div>
      <h1>管理员合规确认</h1>

      <p v-if="loading" class="muted">正在读取当前合规状态…</p>

      <template v-else-if="status">
        <p class="lead">
          后端要求管理员在继续使用管理功能前阅读并确认部署与运营合规承诺。
          这是一次显式确认，不会由系统代替你完成。
        </p>

        <div class="meta-row">
          <span>版本</span>
          <strong>{{ status.version }}</strong>
        </div>

        <div class="doc-links">
          <a :href="status.document_url_zh" target="_blank" rel="noopener noreferrer">阅读中文承诺</a>
          <a :href="status.document_url_en" target="_blank" rel="noopener noreferrer">Read English version</a>
        </div>

        <label class="review-check">
          <input v-model="reviewed" type="checkbox" />
          <span>我已阅读上述合规承诺，并准备进行确认。</span>
        </label>

        <div class="phrase-box">
          <div class="phrase-label">请输入下面的完整确认语句</div>
          <code>{{ status.ack_phrase_zh }}</code>
          <input
            v-model="phrase"
            class="phrase-input"
            type="text"
            autocomplete="off"
            spellcheck="false"
            placeholder="输入完整确认语句"
            @keyup.enter="accept"
          />
        </div>

        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

        <button class="accept-button" :disabled="!canAccept || submitting" @click="accept">
          {{ submitting ? '正在确认…' : '确认并进入管理后台' }}
        </button>
      </template>

      <template v-else>
        <p class="error">{{ errorMessage || '无法读取合规状态，请稍后重试。' }}</p>
        <button class="secondary-button" @click="loadStatus">重新加载</button>
      </template>
    </section>
  </main>
</template>

<style scoped>
.compliance-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 32px 20px;
  background: #070b10;
  color: #f2f5f8;
}

.compliance-card {
  width: min(720px, 100%);
  padding: 36px;
  border: 1px solid #25303b;
  border-radius: 18px;
  background: #0b1118;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.28);
}

.eyebrow {
  margin-bottom: 12px;
  color: #7f94aa;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.16em;
}

h1 {
  margin: 0 0 16px;
  font-size: 32px;
}

.lead,
.muted {
  color: #9eacba;
  line-height: 1.75;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  margin: 24px 0 16px;
  padding: 14px 16px;
  border: 1px solid #202b36;
  border-radius: 12px;
  background: #0d151d;
}

.doc-links {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 22px;
}

.doc-links a {
  color: #9fc8ff;
  text-decoration: none;
}

.review-check {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  margin-bottom: 20px;
  color: #c7d0d9;
  line-height: 1.6;
}

.review-check input {
  margin-top: 5px;
}

.phrase-box {
  padding: 18px;
  border: 1px solid #27323d;
  border-radius: 14px;
  background: #080d12;
}

.phrase-label {
  margin-bottom: 10px;
  color: #8594a3;
  font-size: 13px;
}

.phrase-box code {
  display: block;
  margin-bottom: 14px;
  color: #e1e8ef;
  line-height: 1.7;
  white-space: normal;
}

.phrase-input {
  width: 100%;
  box-sizing: border-box;
  padding: 12px 14px;
  border: 1px solid #354352;
  border-radius: 10px;
  outline: none;
  background: #0c141c;
  color: #f4f7fa;
}

.phrase-input:focus {
  border-color: #6e8eae;
}

.error {
  margin-top: 16px;
  color: #ff9b9b;
}

.accept-button,
.secondary-button {
  width: 100%;
  margin-top: 20px;
  padding: 13px 16px;
  border: 0;
  border-radius: 11px;
  font-weight: 700;
  cursor: pointer;
}

.accept-button {
  background: #f0f4f8;
  color: #0b1118;
}

.accept-button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.secondary-button {
  background: #17212b;
  color: #dce5ee;
}

@media (max-width: 640px) {
  .compliance-card {
    padding: 24px;
  }
}
</style>
