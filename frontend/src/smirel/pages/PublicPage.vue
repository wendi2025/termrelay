<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useSession } from '../core/session'
import PaymentPublicShell from '../components/payment/PaymentPublicShell.vue'

type ProviderKey = 'openai' | 'anthropic' | 'google'

type ModelEntry = {
  family: string
  api: string
  input: string
  output: string
  note: string
}

type ProviderCatalog = {
  key: ProviderKey
  index: string
  name: string
  protocol: string
  summary: string
  models: ModelEntry[]
}

const route = useRoute()
const { isAuthenticated, isAdmin } = useSession()
const kind = computed(() => String(route.meta.publicKind || 'public'))
const logoUrl = `${import.meta.env.BASE_URL}muxway-mark.svg?v=20260920-ribbon`
const consolePath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')

const providers: ProviderCatalog[] = [
  {
    key: 'openai',
    index: '01',
    name: 'OpenAI',
    protocol: 'Responses / Chat Completions',
    summary: '覆盖通用对话、推理与多模态调用。',
    models: [
      { family: 'GPT 通用模型', api: 'Responses / Chat Completions', input: '实时费率', output: '实时费率', note: '通用任务' },
      { family: 'GPT 轻量模型', api: 'Responses / Chat Completions', input: '实时费率', output: '实时费率', note: '高频调用' },
      { family: '推理模型', api: 'Responses API', input: '实时费率', output: '实时费率', note: '复杂推理' },
    ],
  },
  {
    key: 'anthropic',
    index: '02',
    name: 'Anthropic',
    protocol: 'Messages API',
    summary: '面向长文本、代码与复杂工作流。',
    models: [
      { family: 'Claude 高性能模型', api: 'Messages API', input: '实时费率', output: '实时费率', note: '复杂任务' },
      { family: 'Claude 通用模型', api: 'Messages API', input: '实时费率', output: '实时费率', note: '日常生产' },
      { family: 'Claude 轻量模型', api: 'Messages API', input: '实时费率', output: '实时费率', note: '低延迟' },
    ],
  },
  {
    key: 'google',
    index: '03',
    name: 'Google',
    protocol: 'Gemini API',
    summary: '覆盖 Gemini 通用与多模态模型。',
    models: [
      { family: 'Gemini Pro 系列', api: 'Gemini API', input: '实时费率', output: '实时费率', note: '高质量生成' },
      { family: 'Gemini Flash 系列', api: 'Gemini API', input: '实时费率', output: '实时费率', note: '快速响应' },
      { family: 'Gemini 多模态系列', api: 'Gemini API', input: '实时费率', output: '实时费率', note: '多模态任务' },
    ],
  },
]

const activeProvider = ref<ProviderKey>('openai')
const activeProviderData = computed(() => providers.find((provider) => provider.key === activeProvider.value) || providers[0])

function selectProvider(provider: ProviderKey) {
  activeProvider.value = provider
}

const copy = computed(() => {
  if (kind.value === 'key-usage') return { eyebrow: 'USAGE LOOKUP', title: '用量查询', text: '登录后可在工作区查看请求、Token 与实际费用。公开 Key 查询能力会在 Muxway 原生页面中提供。' }
  if (kind.value === 'legal') return { eyebrow: 'LEGAL', title: '服务条款与隐私', text: '法律文档属于 Muxway 公共页面体系，不再使用旧站点模板。' }
  if (kind.value === 'payment') return { eyebrow: 'PAYMENT', title: '支付服务', text: '支付流程保留后端能力，前端入口已经切换到 Muxway 原生页面。' }
  if (kind.value === 'callback') return { eyebrow: 'AUTH CALLBACK', title: '正在完成账户验证', text: '身份回调会返回 Muxway 登录流程。若页面长时间没有变化，请返回登录页。' }
  return { eyebrow: 'MUXWAY', title: String(route.meta.title || 'Muxway'), text: '该公共功能已经进入 Muxway 独立页面体系。' }
})
</script>

<template>
  <div class="public-page" :class="{ 'models-public-page': kind === 'models' }">
    <div class="site-environment" aria-hidden="true"></div>

    <header class="public-topbar glass">
      <RouterLink to="/home" class="brand-link">
        <img :src="logoUrl" alt="Muxway" />
        <span><strong>Muxway</strong><small>模枢 · API SERVICE</small></span>
      </RouterLink>
      <div>
        <RouterLink to="/home">首页</RouterLink>
        <RouterLink v-if="kind === 'models'" to="/models">模型与价格</RouterLink>
        <RouterLink v-if="isAuthenticated" :to="consolePath">控制台</RouterLink>
        <RouterLink v-else to="/login">登录</RouterLink>
      </div>
    </header>

    <main v-if="kind === 'models'" class="models-shell">
      <section class="models-hero">
        <div class="models-intro">
          <span class="eyebrow">MODEL CATALOG</span>
          <h1>模型与价格</h1>
          <p>统一查看模型入口与计费方式。实际可用模型和费率，以控制台结算信息为准。</p>
          <div class="models-hero-actions">
            <RouterLink :to="isAuthenticated ? consolePath : '/register'" class="primary-button">
              {{ isAuthenticated ? '进入控制台' : '创建账户' }}
            </RouterLink>
            <RouterLink to="/home" class="secondary-button">返回首页</RouterLink>
          </div>
        </div>

        <aside class="models-overview" aria-label="服务概览">
          <div>
            <span>接入</span>
            <strong>统一 Base URL</strong>
            <small>一个入口接入多家模型服务</small>
          </div>
          <div>
            <span>计费</span>
            <strong>按实际用量</strong>
            <small>输入与输出分别记录</small>
          </div>
          <div>
            <span>账单</span>
            <strong>控制台可查</strong>
            <small>请求、Token 与费用集中查看</small>
          </div>
        </aside>
      </section>

      <section class="provider-switch" aria-label="模型服务商">
        <button
          v-for="provider in providers"
          :key="provider.key"
          type="button"
          :class="{ active: activeProvider === provider.key }"
          :aria-pressed="activeProvider === provider.key"
          @click="selectProvider(provider.key)"
        >
          <span class="provider-index">{{ provider.index }}</span>
          <span class="provider-copy">
            <strong>{{ provider.name }}</strong>
            <small>{{ provider.protocol }}</small>
          </span>
        </button>
      </section>

      <section class="catalog-panel">
        <header class="catalog-heading">
          <div>
            <span class="eyebrow">{{ activeProviderData.name.toUpperCase() }}</span>
            <h2>{{ activeProviderData.name }} 模型</h2>
            <p>{{ activeProviderData.summary }}</p>
          </div>
          <div class="catalog-status">
            <i aria-hidden="true"></i>
            <span>计费状态</span>
            <strong>实时费率</strong>
          </div>
        </header>

        <div class="catalog-table-wrap">
          <table class="catalog-table">
            <thead>
              <tr>
                <th>模型系列</th>
                <th>接口</th>
                <th>输入价格</th>
                <th>输出价格</th>
                <th>适用场景</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="model in activeProviderData.models" :key="model.family">
                <td><strong>{{ model.family }}</strong></td>
                <td><code>{{ model.api }}</code></td>
                <td><span class="rate-label">{{ model.input }}</span></td>
                <td><span class="rate-label">{{ model.output }}</span></td>
                <td><span class="model-note">{{ model.note }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="pricing-note">
        <div>
          <span class="eyebrow">PRICING NOTE</span>
          <strong>价格以请求发生时的控制台结算信息为准</strong>
          <p>公开页负责展示模型与计费结构，不写死可能变化的上游单价。</p>
        </div>
        <div class="pricing-note-meta">
          <span><b>01</b> 选择服务商</span>
          <span><b>02</b> 进入控制台</span>
          <span><b>03</b> 查看实际费率</span>
        </div>
      </section>
    </main>

    <main v-else-if="kind === 'payment'" class="payment-public-main">
      <PaymentPublicShell />
    </main>

    <main v-else class="public-surface glass">
      <span class="eyebrow">{{ copy.eyebrow }}</span>
      <h1>{{ copy.title }}</h1>
      <p>{{ copy.text }}</p>
      <div class="public-actions">
        <RouterLink :to="isAuthenticated ? consolePath : '/register'" class="primary-button">
          {{ isAuthenticated ? '进入控制台' : '创建账户' }}
        </RouterLink>
        <RouterLink to="/home" class="secondary-button">返回首页</RouterLink>
      </div>
    </main>
  </div>
</template>

<style scoped>
.models-public-page {
  min-height: 100vh;
  padding: 20px 24px 28px;
  color: #f4f7fa;
  background: #071019;
}

.models-public-page .site-environment {
  background:
    radial-gradient(circle at 50% -8%, rgba(61, 112, 154, 0.13), transparent 35%),
    linear-gradient(180deg, #071019 0%, #07111a 48%, #060d14 100%);
}

.models-public-page .site-environment::after {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.006), rgba(0, 0, 0, 0.08));
}

.models-public-page .public-topbar {
  width: min(1180px, calc(100vw - 48px));
  height: 58px;
  padding: 0 18px;
  border: 1px solid rgba(255, 255, 255, 0.085);
  border-radius: 11px;
  background: #09141e;
  box-shadow: none;
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
}

.models-public-page .public-topbar > div {
  gap: 20px;
  font-size: 0.66rem;
  color: rgba(255, 255, 255, 0.55);
}

.models-public-page .public-topbar a:hover,
.models-public-page .public-topbar a.router-link-active {
  color: rgba(255, 255, 255, 0.94);
}

.models-shell {
  width: min(1180px, calc(100vw - 48px));
  margin: 22px auto 0;
  display: grid;
  gap: 10px;
}

.models-hero {
  min-height: 176px;
  display: grid;
  grid-template-columns: minmax(0, 1.28fr) minmax(390px, 0.72fr);
  gap: 34px;
  align-items: center;
  padding: 22px 4px 24px;
}

.models-intro h1 {
  margin: 8px 0 0;
  font-size: clamp(2.2rem, 4vw, 3.35rem);
  line-height: 1;
  letter-spacing: -0.055em;
  font-weight: 610;
}

.models-intro > p {
  max-width: 650px;
  margin: 14px 0 0;
  font-size: 0.78rem;
  line-height: 1.65;
  color: rgba(255, 255, 255, 0.52);
}

.models-hero-actions {
  display: flex;
  gap: 8px;
  margin-top: 19px;
  font-size: 0.69rem;
}

.models-public-page .models-hero-actions .primary-button {
  min-width: 104px;
  background: #eef3f7;
  border-color: #eef3f7;
  color: #071019;
  font-weight: 650;
}

.models-public-page .models-hero-actions .primary-button:hover {
  background: #ffffff;
  transform: none;
}

.models-public-page .models-hero-actions .secondary-button {
  background: transparent;
  border-color: rgba(255, 255, 255, 0.1);
}

.models-overview {
  min-height: 134px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  border: 1px solid rgba(255, 255, 255, 0.075);
  border-radius: 10px;
  background: #09141e;
  overflow: hidden;
}

.models-overview > div {
  min-width: 0;
  padding: 18px 16px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.models-overview > div + div {
  border-left: 1px solid rgba(255, 255, 255, 0.065);
}

.models-overview span {
  font-size: 0.55rem;
  color: rgba(255, 255, 255, 0.36);
}

.models-overview strong {
  margin-top: 7px;
  font-size: 0.79rem;
  font-weight: 610;
  color: rgba(255, 255, 255, 0.9);
}

.models-overview small {
  margin-top: 5px;
  font-size: 0.58rem;
  line-height: 1.45;
  color: rgba(255, 255, 255, 0.4);
}

.provider-switch {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 7px;
  padding: 6px;
  border: 1px solid rgba(255, 255, 255, 0.075);
  border-radius: 10px;
  background: #08131d;
}

.provider-switch button {
  min-height: 58px;
  padding: 0 15px;
  display: grid;
  grid-template-columns: 30px 1fr;
  align-items: center;
  gap: 10px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: rgba(255, 255, 255, 0.68);
  text-align: left;
  cursor: pointer;
  transition: background 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.provider-switch button:hover {
  background: rgba(255, 255, 255, 0.025);
  color: rgba(255, 255, 255, 0.9);
}

.provider-switch button.active {
  border-color: rgba(91, 151, 205, 0.38);
  background: rgba(59, 111, 156, 0.105);
  color: #ffffff;
}

.provider-index {
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.075);
  border-radius: 6px;
  font: 0.54rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  color: rgba(255, 255, 255, 0.36);
}

.provider-switch button.active .provider-index {
  border-color: rgba(113, 173, 226, 0.26);
  color: #9dc9ec;
}

.provider-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.provider-copy strong {
  font-size: 0.75rem;
  font-weight: 620;
}

.provider-copy small {
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.56rem;
  color: rgba(255, 255, 255, 0.36);
}

.catalog-panel {
  border: 1px solid rgba(255, 255, 255, 0.075);
  border-radius: 10px;
  background: #09141e;
  overflow: hidden;
}

.catalog-heading {
  min-height: 94px;
  padding: 17px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 22px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.065);
}

.catalog-heading h2 {
  margin: 5px 0 0;
  font-size: 1.12rem;
  letter-spacing: -0.03em;
  font-weight: 610;
}

.catalog-heading p {
  margin: 5px 0 0;
  font-size: 0.62rem;
  color: rgba(255, 255, 255, 0.4);
}

.catalog-status {
  min-width: 138px;
  display: grid;
  grid-template-columns: 7px auto;
  grid-template-rows: auto auto;
  column-gap: 8px;
  align-items: center;
}

.catalog-status i {
  grid-row: 1 / 3;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #7edbb9;
}

.catalog-status span {
  font-size: 0.52rem;
  color: rgba(255, 255, 255, 0.34);
}

.catalog-status strong {
  margin-top: 2px;
  font-size: 0.66rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.78);
}

.catalog-table-wrap {
  width: 100%;
  overflow-x: auto;
}

.catalog-table {
  width: 100%;
  min-width: 760px;
  border-collapse: collapse;
  table-layout: fixed;
}

.catalog-table th,
.catalog-table td {
  padding: 0 18px;
  text-align: left;
  vertical-align: middle;
}

.catalog-table th {
  height: 34px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.055);
  font-size: 0.5rem;
  letter-spacing: 0.08em;
  font-weight: 560;
  color: rgba(255, 255, 255, 0.3);
}

.catalog-table td {
  height: 50px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 0.62rem;
  color: rgba(255, 255, 255, 0.55);
}

.catalog-table tbody tr:last-child td {
  border-bottom: 0;
}

.catalog-table tbody tr:hover td {
  background: rgba(255, 255, 255, 0.017);
}

.catalog-table th:first-child,
.catalog-table td:first-child {
  width: 23%;
}

.catalog-table th:nth-child(2),
.catalog-table td:nth-child(2) {
  width: 28%;
}

.catalog-table th:nth-child(3),
.catalog-table td:nth-child(3),
.catalog-table th:nth-child(4),
.catalog-table td:nth-child(4) {
  width: 16%;
}

.catalog-table th:last-child,
.catalog-table td:last-child {
  width: 17%;
}

.catalog-table td strong {
  font-size: 0.68rem;
  font-weight: 610;
  color: rgba(255, 255, 255, 0.88);
}

.catalog-table code {
  font: 0.57rem/1.45 ui-monospace, SFMono-Regular, Menlo, monospace;
  color: rgba(255, 255, 255, 0.5);
}

.rate-label {
  display: inline-flex;
  min-height: 24px;
  align-items: center;
  padding: 0 8px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.025);
  font-size: 0.56rem;
  color: rgba(255, 255, 255, 0.66);
}

.model-note {
  font-size: 0.59rem;
  color: rgba(255, 255, 255, 0.42);
}

.pricing-note {
  min-height: 82px;
  padding: 14px 18px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 30px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 10px;
  background: #08131d;
}

.pricing-note > div:first-child {
  min-width: 0;
}

.pricing-note > div:first-child > strong {
  display: block;
  margin-top: 4px;
  font-size: 0.72rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.84);
}

.pricing-note p {
  margin: 4px 0 0;
  font-size: 0.58rem;
  color: rgba(255, 255, 255, 0.36);
}

.pricing-note-meta {
  display: flex;
  gap: 18px;
  align-items: center;
  font-size: 0.57rem;
  color: rgba(255, 255, 255, 0.44);
  white-space: nowrap;
}

.pricing-note-meta span {
  display: flex;
  align-items: center;
  gap: 6px;
}

.pricing-note-meta b {
  font: 0.5rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  color: rgba(255, 255, 255, 0.28);
}

@media (max-width: 960px) {
  .models-hero {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .models-overview {
    min-height: 112px;
  }

  .pricing-note {
    grid-template-columns: 1fr;
    gap: 12px;
  }
}

@media (max-width: 680px) {
  .models-public-page {
    padding: 14px;
  }

  .models-public-page .public-topbar,
  .models-shell {
    width: 100%;
  }

  .models-public-page .public-topbar > div {
    gap: 12px;
  }

  .models-public-page .public-topbar > div a:nth-child(2) {
    display: none;
  }

  .models-shell {
    margin-top: 12px;
    gap: 8px;
  }

  .models-hero {
    min-height: auto;
    padding: 24px 2px 18px;
  }

  .models-intro h1 {
    font-size: 2.35rem;
  }

  .models-overview {
    grid-template-columns: 1fr;
  }

  .models-overview > div {
    min-height: 70px;
    padding: 13px 15px;
  }

  .models-overview > div + div {
    border-left: 0;
    border-top: 1px solid rgba(255, 255, 255, 0.065);
  }

  .provider-switch {
    grid-template-columns: 1fr;
  }

  .provider-switch button {
    min-height: 52px;
  }

  .catalog-heading {
    align-items: flex-start;
  }

  .catalog-status {
    min-width: auto;
  }

  .pricing-note-meta {
    gap: 10px;
    overflow-x: auto;
  }
}
</style>
