<script setup lang="ts">
import { computed, ref } from 'vue'
import HomeAccountMenu from '../components/HomeAccountMenu.vue'
import { useSession } from '../core/session'
import '../styles/home-layout.css'
import '../styles/home-gateway.css'

const { isAuthenticated, isAdmin } = useSession()
const copied = ref(false)
const logoUrl = `${import.meta.env.BASE_URL}smirel-logo.png`
const apiBase = 'https://api.smirel.com/v1'
const consolePath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')

async function copyBase() {
  await navigator.clipboard.writeText(apiBase)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1400)
}
</script>

<template>
  <div class="home-page">
    <header class="home-topbar">
      <RouterLink to="/home" class="brand-link">
        <img :src="logoUrl" alt="Smirel" />
        <span><strong>Smirel</strong><small>API SERVICE</small></span>
      </RouterLink>

      <nav class="home-nav">
        <RouterLink to="/model-plaza">模型与价格</RouterLink>
        <RouterLink to="/key-usage">用量查询</RouterLink>
        <a href="https://api.smirel.com" target="_blank" rel="noreferrer">接入文档</a>
      </nav>

      <div class="home-actions">
        <template v-if="isAuthenticated">
          <RouterLink :to="consolePath" class="top-console">{{ isAdmin ? '管理控制台' : '控制台' }}</RouterLink>
          <HomeAccountMenu />
        </template>
        <template v-else>
          <RouterLink to="/login" class="quiet-link">登录</RouterLink>
          <RouterLink to="/register" class="top-console">注册</RouterLink>
        </template>
      </div>
    </header>

    <main class="home-content">
      <section class="hero-section">
        <div class="hero-copy-block">
          <span class="hero-kicker">SMIREL / MODEL API</span>
          <h1>统一的大模型接口，<br>让您的 vibecoding 更高效。</h1>
          <p>OpenAI、Claude、Gemini、Grok 共用一个 Base URL。保留熟悉的 SDK 和调用方式，把模型切换、Key 和用量留给 Smirel。</p>

          <div class="hero-actions">
            <RouterLink :to="isAuthenticated ? consolePath : '/register'" class="primary-button">
              {{ isAuthenticated ? '进入控制台' : '开始使用' }}
            </RouterLink>
            <RouterLink to="/model-plaza" class="secondary-button">查看模型</RouterLink>
          </div>

          <div class="hero-trust">
            <span>OpenAI Compatible</span>
            <span>按实际用量计费</span>
          </div>
        </div>

        <div class="gateway-preview" aria-label="Smirel API 接入信息">
          <div class="gateway-preview-head">
            <div>
              <span>SMIREL API</span>
              <strong>生产接口</strong>
            </div>
            <RouterLink :to="isAuthenticated ? '/keys' : '/register'" class="gateway-key-link">
              {{ isAuthenticated ? 'API Keys' : '创建 API Key' }} <b>↗</b>
            </RouterLink>
          </div>

          <div class="gateway-endpoint">
            <div class="gateway-endpoint-label">
              <span>BASE URL</span>
              <button type="button" @click="copyBase">{{ copied ? '已复制' : '复制地址' }}</button>
            </div>
            <code>{{ apiBase }}</code>
          </div>

          <div class="gateway-details">
            <div class="gateway-detail">
              <span>接口格式</span>
              <strong>OpenAI-compatible API</strong>
            </div>
            <div class="gateway-detail">
              <span>认证方式</span>
              <strong>Bearer API Key</strong>
            </div>
            <div class="gateway-detail">
              <span>可用模型</span>
              <div class="gateway-models">
                <b>OpenAI</b>
                <b>Claude</b>
                <b>Gemini</b>
                <b>Grok</b>
              </div>
            </div>
          </div>

          <div class="gateway-preview-foot">
            <RouterLink to="/model-plaza">模型目录 <b>→</b></RouterLink>
            <RouterLink :to="isAuthenticated ? '/usage' : '/key-usage'">用量与日志 <b>→</b></RouterLink>
          </div>
        </div>
      </section>

      <section class="workspace-section">
        <div class="workspace-copy">
          <span>工作区</span>
          <h2>模型、Key 和用量，<br>放在一个地方。</h2>
          <p>首页只负责让您快速开始。后续的模型选择、密钥和调用记录都回到控制台处理。</p>
        </div>

        <div class="workspace-links">
          <RouterLink to="/model-plaza">
            <span>MODEL</span>
            <strong>查看模型与价格</strong>
            <p>挑选模型，查看当前价格和可用状态。</p>
            <b>→</b>
          </RouterLink>
          <RouterLink :to="isAuthenticated ? '/keys' : '/register'">
            <span>KEYS</span>
            <strong>管理 API Key</strong>
            <p>为不同项目创建独立访问凭证。</p>
            <b>→</b>
          </RouterLink>
          <RouterLink :to="isAuthenticated ? '/usage' : '/key-usage'">
            <span>USAGE</span>
            <strong>查看调用与用量</strong>
            <p>请求、Token 和费用集中查看。</p>
            <b>→</b>
          </RouterLink>
        </div>
      </section>

      <section class="closing-section closing-cta">
        <div class="closing-copy">
          <span class="closing-status"><i></i>接入准备就绪</span>
          <h2>创建一个 Key，开始调用。</h2>
          <p>保留现有 SDK，只需要换上 Smirel 的 Base URL 和 API Key，几分钟内即可完成接入。</p>
          <div class="closing-meta">
            <span>统一 Base URL</span>
            <span>OpenAI Compatible</span>
            <span>按量计费</span>
          </div>
        </div>
        <div class="closing-actions">
          <RouterLink :to="isAuthenticated ? consolePath : '/register'" class="closing-primary">
            <span>{{ isAuthenticated ? '进入控制台' : '开始使用' }}</span><b>→</b>
          </RouterLink>
          <a class="closing-secondary" href="https://api.smirel.com" target="_blank" rel="noreferrer">查看接入文档</a>
        </div>
      </section>
    </main>

    <footer class="home-footer">
      <span>© {{ new Date().getFullYear() }} Smirel</span>
      <span>api.smirel.com/v1</span>
    </footer>
  </div>
</template>

<style scoped>
.brand-link {
  gap: 12px;
}

.brand-link img {
  width: 84px;
  height: auto;
  object-fit: contain;
}

.brand-link strong {
  font-size: 1.12rem;
}

.brand-link small {
  margin-top: 5px;
  font-size: .75rem;
}

/*
 * The workspace used to be a split editorial/table layout. That forced the
 * headline into a narrow sticky column and made the three destinations read
 * like rows in a settings table. Keep one reading axis instead: copy first,
 * then a single three-column navigation surface underneath it.
 */
.workspace-section {
  display: block !important;
  padding: 112px 0 118px !important;
}

.workspace-copy {
  position: static !important;
  max-width: 860px;
}

.workspace-copy h2 {
  max-width: 860px;
  font-size: clamp(2.65rem, 4.25vw, 4rem) !important;
  line-height: 1.06 !important;
}

.workspace-copy h2 br {
  display: none;
}

.workspace-copy p {
  max-width: 640px !important;
  margin-top: 21px !important;
  font-size: .95rem !important;
}

.workspace-links {
  margin-top: 52px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid #272c34 !important;
  border-radius: 14px;
  background: #0c0e12;
}

.workspace-links a {
  position: relative;
  min-height: 224px !important;
  padding: 28px 30px 26px !important;
  display: flex !important;
  flex-direction: column;
  align-items: flex-start !important;
  gap: 0 !important;
  border: 0 !important;
  border-right: 1px solid #272c34 !important;
  background: transparent;
  transition: background-color .18s ease, color .18s ease !important;
}

.workspace-links a:last-child {
  border-right: 0 !important;
}

.workspace-links a::before {
  content: '';
  position: absolute;
  inset: 0 0 auto;
  height: 1px;
  background: transparent;
  transition: background-color .18s ease;
}

.workspace-links a:hover {
  background: #11141a !important;
}

.workspace-links a:hover::before {
  background: #2499e6;
}

.workspace-links span {
  padding-left: 0 !important;
  color: #5c6673 !important;
  font-size: .68rem !important;
  letter-spacing: .1em !important;
}

.workspace-links strong {
  margin-top: 30px;
  color: #f0f2f5 !important;
  font-size: 1.18rem !important;
  font-weight: 640 !important;
  letter-spacing: -.018em;
}

.workspace-links p {
  max-width: 280px;
  margin: 12px 0 0 !important;
  color: #777f8a !important;
  font-size: .83rem !important;
  line-height: 1.65 !important;
}

.workspace-links b {
  width: 34px;
  height: 34px;
  margin-top: auto;
  align-self: flex-end;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #2c323b;
  border-radius: 50%;
  color: #8695a3 !important;
  font-size: .95rem !important;
  font-weight: 500 !important;
  transition: border-color .18s ease, color .18s ease, transform .18s ease;
}

.workspace-links a:hover b {
  border-color: #3b4652;
  color: #d7e7f2 !important;
  transform: translateX(2px);
}

.closing-cta {
  position: relative;
  min-height: 280px;
  margin: 28px 0 30px;
  padding: 46px 50px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 190px;
  align-items: center;
  gap: clamp(40px, 6vw, 84px);
  overflow: hidden;
  border: 1px solid #262b33;
  border-radius: 16px;
  background: #0f1115;
}

.closing-cta::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: #2499e6;
}

.closing-copy {
  max-width: 760px;
}

.closing-status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #8c949f !important;
  font-size: .78rem !important;
  font-weight: 650 !important;
  letter-spacing: .02em !important;
}

.closing-status i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #3bd09a;
}

.closing-cta h2 {
  max-width: 760px;
  margin: 17px 0 0;
  font-size: clamp(2.35rem, 3.7vw, 3.45rem);
  line-height: 1.08;
  letter-spacing: -.05em;
}

.closing-copy > p {
  max-width: 650px;
  margin: 18px 0 0;
  color: #858c96;
  font-size: .95rem;
  line-height: 1.75;
}

.closing-meta {
  margin-top: 24px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 0;
  color: #69717c;
  font-size: .78rem;
}

.closing-meta span {
  display: inline-flex;
  align-items: center;
}

.closing-meta span + span::before {
  content: '·';
  margin: 0 13px;
  color: #343a43;
}

.closing-actions {
  width: 190px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.closing-primary,
.closing-secondary {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  font-size: .86rem;
  font-weight: 620;
  transition: background-color .15s ease, border-color .15s ease, color .15s ease;
}

.closing-primary {
  padding: 0 16px;
  justify-content: space-between;
  border: 1px solid #2498e5;
  background: #2499e6;
  color: #fff;
}

.closing-primary:hover {
  border-color: #39aff9;
  background: #31a7f3;
}

.closing-primary b {
  font-size: 1rem;
  font-weight: 500;
}

.closing-secondary {
  border: 1px solid #30353e;
  background: #111318;
  color: #aeb4bd;
}

.closing-secondary:hover {
  border-color: #464c56;
  background: #16191f;
  color: #fff;
}

@media (max-width: 820px) {
  .workspace-section {
    padding: 84px 0 92px !important;
  }

  .workspace-links {
    margin-top: 40px;
    grid-template-columns: 1fr;
  }

  .workspace-links a {
    min-height: 168px !important;
    border-right: 0 !important;
    border-bottom: 1px solid #272c34 !important;
  }

  .workspace-links a:last-child {
    border-bottom: 0 !important;
  }

  .workspace-links strong {
    margin-top: 22px;
  }

  .workspace-links b {
    position: absolute;
    right: 26px;
    bottom: 24px;
  }

  .closing-cta {
    min-height: 0;
    padding: 38px 32px;
    grid-template-columns: 1fr;
    gap: 30px;
  }

  .closing-actions {
    width: 100%;
    flex-direction: row;
  }

  .closing-primary,
  .closing-secondary {
    flex: 1 1 0;
  }
}

@media (max-width: 560px) {
  .workspace-section {
    padding: 72px 0 80px !important;
  }

  .workspace-copy h2 {
    font-size: 2.35rem !important;
  }

  .workspace-links {
    margin-top: 34px;
    border-radius: 12px;
  }

  .workspace-links a {
    min-height: 158px !important;
    padding: 24px 22px !important;
  }

  .workspace-links b {
    right: 20px;
    bottom: 20px;
  }

  .closing-cta {
    padding: 32px 24px;
    border-radius: 12px;
  }

  .closing-cta h2 {
    font-size: 2.25rem;
  }

  .closing-actions {
    flex-direction: column;
  }
}
</style>