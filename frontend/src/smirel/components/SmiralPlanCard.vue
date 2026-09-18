<script setup lang="ts">
import { computed } from 'vue'
import type { SmiralPlan } from './smiralPlans'

const props = defineProps<{
  plan: SmiralPlan
  highlighted?: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'select', planId: string): void
}>()

const priceDisplay = computed(() => `¥${props.plan.priceCNY.toLocaleString('zh-CN')}`)
const quotaDisplay = computed(() => `$${props.plan.monthlyUSDQuota.toLocaleString('zh-CN')}`)
const paygDisplay = computed(() => `¥${props.plan.paygPricePerUSD.toFixed(2)} / 官方$`)
const effectivePriceDisplay = computed(() => `¥${props.plan.effectivePricePerUSD.toFixed(2)} / 官方$`)

const savingsBadge = computed(() => {
  const paygTotal = props.plan.monthlyUSDQuota * props.plan.paygPricePerUSD
  const savings = paygTotal - props.plan.priceCNY
  if (savings > 0) {
    return `较按量省 ¥${Math.round(savings).toLocaleString('zh-CN')}`
  }
  return '企业权益溢价'
})
</script>

<template>
  <article
    class="smiral-plan-card"
    :class="{ 'is-highlighted': plan.isEnterprise, 'is-loading': loading }"
  >
    <header class="smiral-plan-card__header">
      <div class="smiral-plan-card__tier">
        <span v-if="plan.tier === 'light'">轻享</span>
        <span v-else-if="plan.tier === 'pro'">专业</span>
        <span v-else>企业试运行</span>
      </div>
      <span v-if="plan.isEnterprise" class="smiral-plan-card__badge">20 席 · 独享 quota</span>
      <span v-else-if="plan.whitelistOnly" class="smiral-plan-card__badge">限量 / 白名单</span>
    </header>

    <h3 class="smiral-plan-card__name">{{ plan.name }}</h3>
    <p class="smiral-plan-card__group">{{ plan.groupName }}</p>

    <div class="smiral-plan-card__price">
      <strong>{{ priceDisplay }}</strong>
      <span>/ 月</span>
    </div>

    <div class="smiral-plan-card__savings">{{ savingsBadge }}</div>

    <ul class="smiral-plan-card__benefits">
      <li v-for="benefit in plan.benefits" :key="benefit">{{ benefit }}</li>
    </ul>

    <dl class="smiral-plan-card__metrics">
      <div>
        <dt>月度额度</dt>
        <dd>{{ quotaDisplay }}</dd>
      </div>
      <div v-if="plan.weeklyUSDQuota">
        <dt>周控制额度</dt>
        <dd>${{ plan.weeklyUSDQuota.toLocaleString('zh-CN') }}</dd>
      </div>
      <div>
        <dt>按量报价</dt>
        <dd>{{ paygDisplay }}</dd>
      </div>
      <div>
        <dt>有效售价</dt>
        <dd>{{ effectivePriceDisplay }}</dd>
      </div>
      <div>
        <dt>支持模型</dt>
        <dd class="smiral-plan-card__models">{{ plan.supportModels }}</dd>
      </div>
    </dl>

    <footer class="smiral-plan-card__footer">
      <button
        type="button"
        class="smiral-plan-card__cta"
        :disabled="loading"
        @click="emit('select', plan.id)"
      >
        <span v-if="loading">处理中…</span>
        <span v-else-if="plan.isEnterprise">申请试用</span>
        <span v-else-if="plan.whitelistOnly">申请白名单</span>
        <span v-else>立即订阅</span>
      </button>
    </footer>
  </article>
</template>

<style scoped>
.smiral-plan-card {
  position: relative;
  display: flex;
  flex-direction: column;
  background: var(--card-bg, #fff);
  border: 1px solid var(--card-border, #e5e7eb);
  border-radius: 16px;
  padding: 1.5rem;
  transition: all 0.2s ease;
}

.smiral-plan-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
}

.smiral-plan-card.is-highlighted {
  border-color: #6366f1;
  background: linear-gradient(180deg, #f5f3ff 0%, #ffffff 30%);
}

.smiral-plan-card.is-loading {
  opacity: 0.6;
  pointer-events: none;
}

.smiral-plan-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.smiral-plan-card__tier {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: #6366f1;
  background: #eef2ff;
  padding: 0.25rem 0.625rem;
  border-radius: 9999px;
}

.smiral-plan-card__badge {
  font-size: 0.7rem;
  color: #b45309;
  background: #fef3c7;
  padding: 0.2rem 0.5rem;
  border-radius: 9999px;
}

.smiral-plan-card__name {
  font-size: 1.25rem;
  font-weight: 700;
  color: #111827;
  margin: 0 0 0.25rem 0;
}

.smiral-plan-card__group {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0 0 1rem 0;
}

.smiral-plan-card__price {
  display: flex;
  align-items: baseline;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
}

.smiral-plan-card__price strong {
  font-size: 2rem;
  font-weight: 700;
  color: #111827;
}

.smiral-plan-card__price span {
  font-size: 0.875rem;
  color: #6b7280;
}

.smiral-plan-card__savings {
  font-size: 0.75rem;
  color: #059669;
  background: #ecfdf5;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  display: inline-block;
  margin-bottom: 1rem;
  align-self: flex-start;
}

.smiral-plan-card__benefits {
  list-style: none;
  padding: 0;
  margin: 0 0 1.25rem 0;
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}

.smiral-plan-card__benefits li {
  font-size: 0.75rem;
  color: #374151;
  background: #f3f4f6;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.smiral-plan-card__metrics {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.5rem;
  margin: 0 0 1.5rem 0;
  padding: 1rem;
  background: #f9fafb;
  border-radius: 8px;
}

.smiral-plan-card__metrics > div {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  font-size: 0.8125rem;
}

.smiral-plan-card__metrics dt {
  color: #6b7280;
  margin: 0;
}

.smiral-plan-card__metrics dd {
  margin: 0;
  font-weight: 500;
  color: #111827;
  text-align: right;
}

.smiral-plan-card__models {
  font-size: 0.7rem;
  line-height: 1.4;
  max-width: 65%;
}

.smiral-plan-card__footer {
  margin-top: auto;
}

.smiral-plan-card__cta {
  width: 100%;
  padding: 0.75rem 1rem;
  background: #6366f1;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease;
}

.smiral-plan-card__cta:hover:not(:disabled) {
  background: #4f46e5;
}

.smiral-plan-card__cta:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}
</style>