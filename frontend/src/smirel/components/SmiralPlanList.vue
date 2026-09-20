<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { SMIRAL_PLANS, PLANS_BY_PRODUCT_LINE, PRODUCT_LINE_DISPLAY, type ProductLine, type SmiralPlan } from './smiralPlans'
import SmiralPlanCard from './SmiralPlanCard.vue'
import { paymentApi } from '../api/payment'
import type { AdminSubscriptionPlan } from '../api/payment'

const selectedLine = ref<ProductLine | 'all'>('all')
const selectedPlanId = ref<string | null>(null)
const loading = ref(false)
const loadError = ref<string | null>(null)
const apiPlans = ref<AdminSubscriptionPlan[]>([])

const visiblePlans = computed(() => {
  if (apiPlans.value.length > 0) {
    return apiPlans.value
      .map(toSmiralPlan)
      .filter((p: SmiralPlan) => selectedLine.value === 'all' || p.productLine === selectedLine.value)
  }
  if (selectedLine.value === 'all') return SMIRAL_PLANS
  return PLANS_BY_PRODUCT_LINE[selectedLine.value as ProductLine]
})

const productLineOptions = computed(() => {
  const keys = Object.keys(PRODUCT_LINE_DISPLAY) as ProductLine[]
  return [
    { value: 'all' as const, label: '全部' },
    ...keys.map((k) => ({ value: k, label: PRODUCT_LINE_DISPLAY[k].label })),
  ]
})

function toSmiralPlan(p: AdminSubscriptionPlan): SmiralPlan {
  const productLine = inferProductLine(p)
  const tier = inferTier(p)
  return {
    id: `api-${p.id}`,
    productLine,
    tier,
    groupName: p.group_name || '',
    name: p.name,
    priceCNY: p.price,
    monthlyUSDQuota: p.monthly_limit_usd || 0,
    weeklyUSDQuota: p.weekly_limit_usd || undefined,
    paygPricePerUSD: 0.50,
    effectivePricePerUSD: p.rate_multiplier || 0,
    costMultiplier: 0.10,
    supportModels: p.features || '',
    benefits: parseBenefits(p.features || ''),
    isEnterprise: tier === 'enterprise_trial',
    seats: tier === 'enterprise_trial' ? 20 : 1,
    concurrency: tier === 'enterprise_trial' ? 30 : 5,
    whitelistOnly: tier === 'enterprise_trial' || productLine === 'claude_fable',
    circuitBreakerEnabled: productLine === 'claude_fable',
  }
}

function inferProductLine(p: AdminSubscriptionPlan): ProductLine {
  const name = p.name.toLowerCase()
  if (name.includes('fable') || name.includes('max')) return 'claude_fable'
  if (name.includes('claude')) return 'claude_no_fable'
  if (name.includes('国产')) return 'domestic'
  if (name.includes('plus')) return 'gpt_plus'
  return 'gpt_pro'
}

function inferTier(p: AdminSubscriptionPlan): 'light' | 'pro' | 'enterprise_trial' {
  if (p.name.includes('轻享')) return 'light'
  if (p.name.includes('专业') || p.name.includes('标准')) return 'pro'
  return 'enterprise_trial'
}

function parseBenefits(features: string): string[] {
  return features
    .split(/[\n；;]/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0 && s.length <= 30)
    .slice(0, 6)
}

async function loadPlans() {
  try {
    const plans = await paymentApi.listPlans()
    apiPlans.value = Array.isArray(plans) ? plans : []
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : String(err)
    apiPlans.value = []
  }
}

async function handleSelect(planId: string) {
  selectedPlanId.value = planId
  loading.value = true
  await new Promise((resolve) => setTimeout(resolve, 1200))
  loading.value = false
  selectedPlanId.value = null
}

onMounted(loadPlans)
</script>

<template>
  <section class="smiral-plans">
    <header class="smiral-plans__header">
      <div>
        <h2>Muxway 订阅套餐</h2>
        <p>无日限额 · 周/月额度控制 · 企业试运行档含并发与独享 quota</p>
        <p v-if="loadError" class="smiral-plans__note">
          正在使用本地配置（API 暂不可用：{{ loadError }}）
        </p>
      </div>
      <nav class="smiral-plans__tabs" aria-label="产品线筛选">
        <button
          v-for="opt in productLineOptions"
          :key="opt.value"
          type="button"
          :class="{ active: selectedLine === opt.value }"
          @click="selectedLine = opt.value"
        >
          {{ opt.label }}
        </button>
      </nav>
    </header>

    <div class="smiral-plans__grid">
      <SmiralPlanCard
        v-for="plan in visiblePlans"
        :key="plan.id"
        :plan="plan"
        :loading="loading && selectedPlanId === plan.id"
        @select="handleSelect"
      />
    </div>
  </section>
</template>

<style scoped>
.smiral-plans {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.smiral-plans__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}
.smiral-plans__header h2 {
  font-size: 1.5rem;
  font-weight: 700;
  color: #111827;
  margin: 0 0 0.25rem 0;
}
.smiral-plans__header p {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}
.smiral-plans__note {
  color: #b45309 !important;
  background: #fef3c7;
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  margin-top: 0.5rem !important;
  font-size: 0.8125rem !important;
}
.smiral-plans__tabs {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.smiral-plans__tabs button {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid #e5e7eb;
  border-radius: 9999px;
  font-size: 0.875rem;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.15s ease;
}
.smiral-plans__tabs button:hover {
  border-color: #6366f1;
  color: #6366f1;
}
.smiral-plans__tabs button.active {
  background: #6366f1;
  border-color: #6366f1;
  color: white;
}
.smiral-plans__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}
</style>
