<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import HomeAccountMenu from '../components/HomeAccountMenu.vue'
import HomeTopbarControls from '../components/HomeTopbarControls.vue'
import { paymentApi, type PublicSubscriptionPlan } from '../api/payment'
import { interfacePreferences, setLocale, setTheme } from '../core/preferences'
import { type SmirelLocale } from '../core/i18n'
import { useSession } from '../core/session'
import claudeLogo from '../assets/providers/claude.svg'
import geminiLogo from '../assets/providers/gemini.svg'
import grokLogo from '../assets/providers/grok.svg'
import openaiLogo from '../assets/providers/openai.svg'
import kimiLogo from '../assets/providers/kimi.svg'
import glmLogo from '../assets/providers/glm.svg'
import seedanceLogo from '../assets/providers/seedance.svg'
import qwenLogo from '../assets/providers/qwen.svg'
import minimaxLogo from '../assets/providers/minimax.svg'
import '../styles/home-landing.css'

const { isAuthenticated, isAdmin } = useSession()
const plans = ref<PublicSubscriptionPlan[]>([])
const plansState = ref<'loading' | 'ready' | 'unavailable'>('loading')
const mobileMenuOpen = ref(false)
const copied = ref(false)
const apiBase = 'https://api.smirel.com/v1'
const logoUrl = `${import.meta.env.BASE_URL}muxway-mark.svg`
const consolePath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const isEnglish = computed(() => interfacePreferences.locale === 'en-US')
const copy = computed<any>(() => isEnglish.value ? en : zh)
const providers = [
  { name: 'OpenAI', src: openaiLogo },
  { name: 'Claude', src: claudeLogo },
  { name: 'Gemini', src: geminiLogo },
  { name: 'Grok', src: grokLogo },
  { name: 'Kimi', src: kimiLogo },
  { name: 'GLM', src: glmLogo },
  { name: 'Seedance', src: seedanceLogo },
  { name: 'Qwen', src: qwenLogo },
  { name: 'MiniMax', src: minimaxLogo },
]
const tools = [['CC', 'Claude Code'], ['CX', 'Codex'], ['OA', 'OpenAI SDK'], ['AN', 'Anthropic SDK'], ['GM', 'Gemini SDK'], ['IDE', 'Editors & clients']]
const groups = computed(() => {
  const result = new Map<string, PublicSubscriptionPlan[]>()
  const visiblePlans = Array.isArray(plans.value) ? plans.value : []
  visiblePlans.forEach((plan) => {
    const key = plan.group_name || plan.platform || 'Muxway'
    result.set(key, [...(result.get(key) || []), plan])
  })
  return [...result.entries()].map(([name, items]) => ({ name, items }))
})

function planTarget(plan: PublicSubscriptionPlan) {
  const target = `/subscriptions?plan=${plan.id}`
  return isAuthenticated.value ? target : `/register?redirect=${encodeURIComponent(target)}`
}
function formatPrice(plan: PublicSubscriptionPlan) {
  const value = Number(plan.price || 0).toLocaleString(interfacePreferences.locale, { maximumFractionDigits: 2 })
  return (plan.currency || 'CNY') === 'CNY' ? `¥${value}` : `${plan.currency} ${value}`
}
function validity(plan: PublicSubscriptionPlan) {
  if (!isEnglish.value && String(plan.validity_unit).toLowerCase().startsWith('day')) return `${plan.validity_days} 天`
  return `${plan.validity_days} ${plan.validity_unit}`
}
function chooseLocale(event: Event) { setLocale((event.target as HTMLSelectElement).value as SmirelLocale) }
function toggleTheme() { setTheme(interfacePreferences.resolvedTheme === 'dark' ? 'light' : 'dark') }
async function copyBase() {
  await navigator.clipboard.writeText(apiBase)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1400)
}
onMounted(async () => {
  try {
    const response: unknown = await paymentApi.listPublicPlans()
    if (!Array.isArray(response)) throw new Error('Public plans response is not an array')
    plans.value = response
    plansState.value = 'ready'
  } catch { plansState.value = 'unavailable' }
})

const zh = {
  nav: ['平台能力', '工具兼容', '套餐', '常见问题'],
  hero: ['MUXWAY · 模枢', '一个接口，通达百模', 'One API. Every model.', '沿用熟悉的 SDK 和调用方式，在一个控制台中管理模型、API Key、用量与套餐。', '开始使用', '查看模型与价格', '已接入主流模型'],
  pain: ['01 · 痛点', '使用 AI 时，最容易卡在这四件事。', '模型越多、平台越分散，配置、费用和账号管理就越容易失控。', [['配置太多', '每个平台都要申请 Key、填写地址，来回切换很麻烦。'], ['模型不好换', '想试新模型时，常常还要改 SDK、环境变量和业务代码。'], ['费用看不清', '请求量、Token 和花费散落在不同后台，月底才发现超预算。'], ['账号怕受限', '单一账号或单一路线出问题，项目调用就可能被打断。']]],
  cap: ['02 · 能力', '我们帮你解决。', '简单四步，轻松接入并稳定使用 AI。', [['一键接入', '获取一个 API 密钥，就能调用已接入的主流 AI 模型，无需分别申请。'], ['稳定可靠', '智能调度多个上游账号，自动切换和负载均衡，减少频繁报错。'], ['用多少付多少', '按实际使用量计费，也支持设置配额上限，团队用量一目了然。'], ['不怕封号', '持续优选线路与接入方式，做好风险隔离，让调用更安心。']]],
  tools: ['03 · 工具', '融入已有的开发工作流。', '从终端、编辑器到 SDK，统一通过 Muxway 模枢文档完成接入。', '查看接入文档'],
  compare: ['04 · 对比', '一个入口，保留选择权。', ['对比项', '分别接入服务商', 'Muxway'], [['模型入口', '按服务商分别配置', '一个 Base URL 统一接入'], ['认证管理', '多套账号与密钥', '集中管理 API Key'], ['用量查看', '分散在不同后台', '请求、Token 和费用集中查看'], ['接入方式', '逐项适配调用差异', '保留熟悉 SDK 与工作流']]],
  steps: ['05 · 接入', '三步开始调用。', '从创建账号到第一次请求，路径保持简单清晰。', [['01', '创建账户', '注册后进入控制台，创建独立的 API Key。'], ['02', '选择模型与套餐', '查看可用模型，按量充值或选择适合的订阅。'], ['03', '替换 Base URL', '在现有工具中填入 Muxway 地址，开始调用。']], '复制地址', '已复制'],
  pricing: ['06 · 套餐', '选择适合当前工作流的方案。', '套餐、价格和在售状态直接来自控制台配置。', '正在加载当前套餐…', '套餐暂时不可用，请登录控制台查看最新信息。', '查看并开通', '查看申请方式', '有效期', '月额度', '席', '并发'],
  faq: ['07 · 答疑', '常见问题', '先把常见的接入与计费问题说清楚。', [['支持哪些模型和协议？', '以模型广场和当前账户可见模型为准。Muxway 模枢提供统一 API 入口，并保留常见 SDK 的接入方式。'], ['现有项目需要重写吗？', '通常只需替换 Base URL 和 API Key。具体参数和模型名称请以接入文档为准。'], ['按量充值与订阅套餐如何选择？', '按量充值适合弹性调用；订阅套餐适合有稳定模型和额度需求的工作流。'], ['如何管理团队或项目用量？', '可为不同项目创建独立 Key，在控制台查看请求、Token 和费用记录。']]],
  closing: ['准备好把模型接入，变成一件简单的事了吗？', '创建账户，生成 API Key，然后继续使用你熟悉的工具。', '进入控制台', '阅读接入文档'],
}
const en = {
  nav: ['Capabilities', 'Tools', 'Plans', 'FAQ'],
  hero: ['MUXWAY', 'One API. Every model.', '一个接口，通达百模', 'Keep familiar SDKs while managing models, API keys, usage, and plans from one console.', 'Get started', 'Models and pricing', 'Leading models available'],
  pain: ['01 · Problems', 'Four things that make using AI harder than it should be.', 'As models and providers multiply, setup, cost, and account management can quickly get out of hand.', [['Too much setup', 'Every platform has its own key, endpoint, and setup steps.'], ['Hard to switch', 'Trying a new model can mean changing SDKs, environment variables, and code.'], ['Costs are unclear', 'Requests, tokens, and spend are scattered across dashboards.'], ['Accounts get limited', 'A problem with one route or account can interrupt your project.']]],
  cap: ['02 · Capabilities', 'We solve that for you.', 'Four simple ways to start using AI with confidence.', [['One-click access', 'Use one API key to call available leading AI models without separate applications.'], ['Reliable by design', 'Smart routing, automatic switching, and load balancing reduce avoidable errors.'], ['Pay for what you use', 'Pay by usage and set quota limits so team spend stays visible.'], ['Lower account risk', 'Diversified routes and risk isolation help keep requests running.']]],
  tools: ['03 · Tools', 'Fits into your existing developer workflow.', 'From terminal tools and editors to SDKs, start from the Muxway integration docs.', 'Read integration docs'],
  compare: ['04 · Compare', 'One entry point, with room to choose.', ['Area', 'Separate providers', 'Muxway'], [['Model access', 'Configure each provider separately', 'One Base URL'], ['Credentials', 'Multiple accounts and keys', 'Centralized API keys'], ['Usage', 'Distributed dashboards', 'Requests, tokens, and costs together'], ['Integration', 'Adapt to each API', 'Keep familiar SDK workflows']]],
  steps: ['05 · Connect', 'Start in three steps.', 'From account creation to the first request, the path stays direct.', [['01', 'Create an account', 'Enter the console and create a dedicated API key.'], ['02', 'Choose models and plans', 'Review available models, then add balance or choose a plan.'], ['03', 'Replace the Base URL', 'Use the Muxway endpoint in your existing tools.']], 'Copy URL', 'Copied'],
  pricing: ['06 · Plans', 'Choose a plan for your workflow.', 'Plan availability and pricing come directly from the console configuration.', 'Loading current plans…', 'Plans are temporarily unavailable. Sign in to view the latest catalog.', 'View and activate', 'View application process', 'Validity', 'Monthly quota', 'seats', 'concurrency'],
  faq: ['07 · FAQ', 'Frequently asked questions', 'The common integration and billing questions, answered upfront.', [['Which models and protocols are supported?', 'Use the model catalog and the models visible to your account as the source of truth.'], ['Do existing projects need a rewrite?', 'Usually you only need to replace the Base URL and API key.'], ['How do balance and subscriptions differ?', 'Balance works for variable usage. Subscriptions suit stable model and quota needs.'], ['How can teams manage usage?', 'Create separate keys for projects and review requests, tokens, and costs in the console.']]],
  closing: ['Ready to make model integration simpler?', 'Create an account, generate an API key, and keep using the tools you know.', 'Open console', 'Read integration docs'],
}
</script>

<template>
  <div class="home-page">
    <header class="home-topbar">
      <RouterLink to="/home" class="home-brand"><img :src="logoUrl" alt="Muxway"><span><strong>Muxway</strong><small>模枢 · API SERVICE</small></span></RouterLink>
      <nav class="home-nav" :class="{ 'is-open': mobileMenuOpen }"><a href="#capabilities">{{ copy.nav[0] }}</a><a href="#tools">{{ copy.nav[1] }}</a><a href="#pricing">{{ copy.nav[2] }}</a><a href="#faq">{{ copy.nav[3] }}</a></nav>
      <div class="home-actions"><template v-if="isAuthenticated"><HomeTopbarControls /><HomeAccountMenu variant="toolbar" /></template><template v-else><select :value="interfacePreferences.locale" aria-label="Language" @change="chooseLocale"><option value="zh-CN">中文</option><option value="en-US">EN</option></select><button class="home-theme-toggle" type="button" :aria-label="isEnglish ? 'Toggle dark mode' : '切换暗夜模式'" :title="isEnglish ? 'Toggle dark mode' : '切换暗夜模式'" @click="toggleTheme"><svg v-if="interfacePreferences.resolvedTheme === 'dark'" viewBox="0 0 24 24" aria-hidden="true"><path d="M20.2 15.3A8.5 8.5 0 0 1 8.7 3.8 8.5 8.5 0 1 0 20.2 15.3Z" /></svg><svg v-else viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></svg></button><RouterLink to="/login">{{ isEnglish ? 'Log in' : '登录' }}</RouterLink><RouterLink class="home-register" to="/register">{{ isEnglish ? 'Sign up' : '注册' }}</RouterLink></template><button class="home-menu" type="button" @click="mobileMenuOpen = !mobileMenuOpen">{{ isEnglish ? 'Menu' : '菜单' }}</button></div>
    </header>
    <main>
      <section class="home-hero"><p class="home-eyebrow">{{ copy.hero[0] }}</p><h1>{{ copy.hero[1] }}</h1><p class="home-hero-lead">{{ copy.hero[2] }}</p><p class="home-hero-copy">{{ copy.hero[3] }}</p><div class="home-hero-actions"><RouterLink class="home-primary" :to="isAuthenticated ? consolePath : '/register'">{{ copy.hero[4] }}</RouterLink><RouterLink class="home-secondary" to="/model-plaza">{{ copy.hero[5] }}</RouterLink></div><div class="home-model-rail"><span>{{ copy.hero[6] }}</span><div class="home-model-list"><span v-for="provider in providers" :key="provider.name" class="home-model-chip"><img :src="provider.src" :alt="provider.name"><small>{{ provider.name }}</small></span></div></div></section>
      <section class="home-section home-pain"><div class="home-heading"><span>{{ copy.pain[0] }}</span><h2>{{ copy.pain[1] }}</h2><p>{{ copy.pain[2] }}</p></div><div class="home-card-grid"><article v-for="([title, text], index) in copy.pain[3]" :key="title" :class="{ 'is-emphasis': index === 3 }"><span>0{{ index + 1 }}</span><h3>{{ title }}</h3><p>{{ text }}</p></article></div></section>
      <section id="capabilities" class="home-section home-capabilities"><div class="home-heading"><span>{{ copy.cap[0] }}</span><h2>{{ copy.cap[1] }}</h2><p>{{ copy.cap[2] }}</p></div><div class="home-card-grid"><article v-for="([title, text], index) in copy.cap[3]" :key="title" :class="{ 'is-emphasis': index === 0 || index === 3 }"><span>0{{ index + 1 }}</span><h3>{{ title }}</h3><p>{{ text }}</p></article></div></section>
      <section id="tools" class="home-section"><div class="home-heading"><span>{{ copy.tools[0] }}</span><h2>{{ copy.tools[1] }}</h2><p>{{ copy.tools[2] }}</p></div><div class="home-tool-grid"><a v-for="tool in tools" :key="tool[1]" href="https://api.smirel.com" target="_blank" rel="noreferrer"><b>{{ tool[0] }}</b><span>{{ tool[1] }}</span><i>↗</i></a></div><a class="home-inline-link" href="https://api.smirel.com" target="_blank" rel="noreferrer">{{ copy.tools[3] }} →</a></section>
      <section class="home-section"><div class="home-heading"><span>{{ copy.compare[0] }}</span><h2>{{ copy.compare[1] }}</h2></div><div class="home-table"><div><b v-for="cell in copy.compare[2]" :key="cell">{{ cell }}</b></div><div v-for="row in copy.compare[3]" :key="row[0]"><span v-for="cell in row" :key="cell">{{ cell }}</span></div></div></section>
      <section class="home-section"><div class="home-heading"><span>{{ copy.steps[0] }}</span><h2>{{ copy.steps[1] }}</h2><p>{{ copy.steps[2] }}</p></div><div class="home-steps"><article v-for="([number, title, text], index) in copy.steps[3]" :key="number"><span>{{ number }}</span><h3>{{ title }}</h3><p>{{ text }}</p><div v-if="index === 2" class="home-code"><code>{{ apiBase }}</code><button type="button" @click="copyBase">{{ copied ? copy.steps[5] : copy.steps[4] }}</button></div></article></div></section>
      <section id="pricing" class="home-section"><div class="home-heading"><span>{{ copy.pricing[0] }}</span><h2>{{ copy.pricing[1] }}</h2><p>{{ copy.pricing[2] }}</p></div><p v-if="plansState === 'loading'" class="home-state">{{ copy.pricing[3] }}</p><div v-else-if="plansState === 'ready' && groups.length" class="home-plan-groups"><section v-for="group in groups" :key="group.name"><header><h3>{{ group.name }}</h3><span>{{ group.items.length }}</span></header><div class="home-plan-grid"><article v-for="plan in group.items" :key="plan.id" class="home-plan" :class="{ 'is-featured': plan.card_featured }"><div><span>{{ plan.card_badge || plan.card_tier || plan.platform || 'MUXWAY' }}</span><small v-if="plan.card_featured">FEATURED</small></div><h4>{{ plan.name }}</h4><p>{{ plan.description }}</p><strong>{{ formatPrice(plan) }}</strong><em>/ {{ validity(plan) }}</em><dl><div><dt>{{ copy.pricing[7] }}</dt><dd>{{ validity(plan) }}</dd></div><div v-if="plan.monthly_limit_usd"><dt>{{ copy.pricing[8] }}</dt><dd>${{ Number(plan.monthly_limit_usd).toLocaleString() }}</dd></div><div v-if="plan.seat_limit"><dt>{{ copy.pricing[9] }}</dt><dd>{{ plan.seat_limit }}</dd></div><div v-if="plan.concurrency_limit"><dt>{{ copy.pricing[10] }}</dt><dd>{{ plan.concurrency_limit }}</dd></div></dl><ul><li v-for="feature in plan.features.slice(0, 4)" :key="feature">{{ feature }}</li></ul><RouterLink :to="planTarget(plan)">{{ plan.purchase_policy === 'approval' ? copy.pricing[6] : copy.pricing[5] }} →</RouterLink></article></div></section></div><div v-else class="home-state"><p>{{ copy.pricing[4] }}</p><RouterLink class="home-secondary" :to="isAuthenticated ? '/subscriptions' : '/login'">{{ isEnglish ? 'View plans' : '查看套餐' }}</RouterLink></div></section>
      <section id="faq" class="home-section"><div class="home-heading"><span>{{ copy.faq[0] }}</span><h2>{{ copy.faq[1] }}</h2><p>{{ copy.faq[2] }}</p></div><div class="home-faq"><details v-for="([question, answer], index) in copy.faq[3]" :key="question" :open="index === 0"><summary>{{ question }}<b>+</b></summary><p>{{ answer }}</p></details></div></section>
      <section class="home-closing"><div><span>MUXWAY · 模枢</span><h2>{{ copy.closing[0] }}</h2><p>{{ copy.closing[1] }}</p></div><div><RouterLink class="home-primary" :to="isAuthenticated ? consolePath : '/register'">{{ copy.closing[2] }}</RouterLink><a class="home-secondary" href="https://api.smirel.com" target="_blank" rel="noreferrer">{{ copy.closing[3] }}</a></div></section>
    </main>
    <footer class="home-footer"><div><RouterLink to="/home" class="home-brand"><img :src="logoUrl" alt="Muxway"><span><strong>Muxway</strong><small>模枢 · API SERVICE</small></span></RouterLink><p>{{ isEnglish ? 'Muxway unified AI API gateway.' : 'Muxway 模枢一站式 AI API 网关平台。' }}</p></div><div><RouterLink to="/model-plaza">{{ isEnglish ? 'Models and pricing' : '模型与价格' }}</RouterLink><RouterLink to="/key-usage">{{ isEnglish ? 'Usage lookup' : '用量查询' }}</RouterLink><a href="https://api.smirel.com" target="_blank" rel="noreferrer">{{ isEnglish ? 'Integration docs' : '接入文档' }}</a></div><small>© {{ new Date().getFullYear() }} Muxway 模枢</small></footer>
  </div>
</template>
