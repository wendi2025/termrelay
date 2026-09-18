<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import UserBillingPage from '../components/UserBillingPage.vue'
import { useSession } from '../core/session'

const { locale } = useI18n()
const { state } = useSession()

const title = computed(() => locale.value === 'zh-CN' ? '充值/订阅' : 'Recharge / Subscription')
const description = computed(() => locale.value === 'zh-CN'
  ? '为账户充值余额，或选择适合你的订阅方案。'
  : 'Add account balance or choose a subscription plan that fits your needs.')
const eyebrow = computed(() => 'BILLING & PLANS')
const assurance = computed(() => locale.value === 'zh-CN' ? '安全支付 · 即时开通' : 'Secure payment · Instant activation')
const accountBalance = computed(() => Number(state.user?.balance || 0))
</script>

<template>
  <section class="workspace-page billing-route-page">
    <header class="page-heading billing-heading">
      <div class="billing-heading-copy">
        <span class="billing-eyebrow">{{ eyebrow }}</span>
        <h1>{{ title }}</h1>
        <p>{{ description }}</p>
      </div>
      <div class="billing-assurance" aria-hidden="true">
        <span class="billing-assurance-dot"></span>
        {{ assurance }}
      </div>
    </header>
    <UserBillingPage :balance="accountBalance" />
  </section>
</template>

<style scoped>
.billing-route-page {
  --billing-surface: rgba(15, 19, 25, 0.94);
  --billing-surface-raised: rgba(20, 25, 32, 0.96);
  --billing-surface-soft: rgba(255, 255, 255, 0.035);
  --billing-border: rgba(255, 255, 255, 0.08);
  --billing-border-strong: rgba(121, 196, 245, 0.38);
  --billing-text: #f4f7fa;
  --billing-text-soft: rgba(226, 234, 241, 0.72);
  --billing-muted: rgba(207, 218, 228, 0.5);
  --billing-subtle: rgba(199, 212, 224, 0.36);
  --billing-accent: #79c4f5;
  --billing-accent-strong: #4ca7e0;
  --billing-accent-soft: rgba(121, 196, 245, 0.1);
  --billing-success: #61d9b1;
  --billing-danger: #f48b8b;
  --billing-shadow: 0 18px 48px rgba(0, 0, 0, 0.18);

  width: min(100%, 1240px);
  margin: 0 auto;
  padding: 4px 2px 52px;
}

:global(html.smirel-app[data-theme='light']) .billing-route-page {
  --billing-surface: #ffffff;
  --billing-surface-raised: #ffffff;
  --billing-surface-soft: #f7f9fb;
  --billing-border: #e1e7ed;
  --billing-border-strong: #9cc9e8;
  --billing-text: #17212b;
  --billing-text-soft: #4d5a67;
  --billing-muted: #72808d;
  --billing-subtle: #98a3ad;
  --billing-accent: #1978bd;
  --billing-accent-strong: #11649f;
  --billing-accent-soft: #edf6fc;
  --billing-success: #168c66;
  --billing-danger: #b84e4e;
  --billing-shadow: 0 16px 42px rgba(37, 54, 70, 0.07);
}

.billing-heading {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 32px;
  min-height: 112px;
  margin-bottom: 22px;
  padding: 10px 0 24px;
  border-bottom: 1px solid var(--billing-border);
}

.billing-heading::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: -1px;
  width: 96px;
  height: 2px;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--billing-accent), transparent);
}

.billing-heading-copy {
  min-width: 0;
}

.billing-eyebrow {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  margin-bottom: 10px;
  padding: 0 9px;
  border: 1px solid var(--billing-border);
  border-radius: 999px;
  background: var(--billing-accent-soft);
  color: var(--billing-accent);
  font-size: 0.62rem;
  font-weight: 760;
  letter-spacing: 0.13em;
}

.billing-heading h1 {
  margin: 0;
  color: var(--billing-text);
  font-size: clamp(2rem, 2.5vw, 2.45rem);
  font-weight: 740;
  letter-spacing: -0.045em;
}

.billing-heading p {
  max-width: 650px;
  margin: 8px 0 0;
  color: var(--billing-muted);
  font-size: 0.82rem;
  line-height: 1.7;
}

.billing-assurance {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 8px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid var(--billing-border);
  border-radius: 999px;
  background: var(--billing-surface);
  color: var(--billing-muted);
  font-size: 0.72rem;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.04);
}

.billing-assurance-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--billing-success);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--billing-success) 12%, transparent);
}

:deep(.billing-page) {
  gap: 18px;
}

:deep(.billing-mode) {
  gap: 4px;
  padding: 4px;
  border-color: var(--billing-border);
  border-radius: 13px;
  background: var(--billing-surface);
  box-shadow: 0 6px 20px rgba(26, 42, 58, 0.04);
}

:deep(.billing-mode button) {
  min-width: 108px;
  min-height: 38px;
  padding: 0 18px;
  border-radius: 9px;
  color: var(--billing-muted);
  transition: color 160ms ease, background 160ms ease, box-shadow 160ms ease;
}

:deep(.billing-mode button:hover:not(.active)) {
  color: var(--billing-text-soft);
  background: var(--billing-surface-soft);
}

:deep(.billing-mode button.active) {
  background: var(--billing-accent-soft);
  color: var(--billing-accent-strong);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--billing-accent) 18%, transparent);
}

:deep(.plan-list) {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
  align-items: stretch;
}

:deep(.pricing-card) {
  border-color: var(--billing-border);
  background:
    radial-gradient(circle at 100% 0%, color-mix(in srgb, var(--billing-accent) 8%, transparent), transparent 34%),
    var(--billing-surface);
  box-shadow: var(--billing-shadow);
}

:deep(.pricing-card:hover) {
  border-color: var(--billing-border-strong);
}

:deep(.pricing-card.selected) {
  border-color: var(--billing-accent);
}

:deep(.pricing-title h3),
:deep(.pricing-price) {
  color: var(--billing-text);
}

:deep(.pricing-title p),
:deep(.pricing-reference),
:deep(.pricing-facts dt),
:deep(.pricing-benefits li),
:deep(.pricing-footnote) {
  color: var(--billing-muted);
}

:deep(.pricing-facts),
:deep(.pricing-facts > div) {
  border-color: var(--billing-border);
}

:deep(.pricing-facts dd) {
  color: var(--billing-text-soft);
}

:deep(.empty) {
  border-color: var(--billing-border);
  background: var(--billing-surface);
  color: var(--billing-muted);
}

@media (max-width: 980px) {
  .billing-route-page {
    width: min(100%, 860px);
  }

  .billing-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  :deep(.plan-list) {
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  }
}

@media (max-width: 640px) {
  .billing-route-page {
    padding-bottom: 32px;
  }

  .billing-heading {
    min-height: 0;
    margin-bottom: 18px;
    padding: 4px 0 18px;
  }

  .billing-heading h1 {
    font-size: 1.9rem;
  }

  .billing-assurance {
    display: none;
  }

  :deep(.billing-mode) {
    display: grid;
    grid-template-columns: 1fr 1fr;
    width: 100%;
  }

  :deep(.billing-mode button) {
    min-width: 0;
    width: 100%;
  }

  :deep(.plan-list) {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  :deep(.pricing-card) {
    min-height: 0;
    border-radius: 16px;
  }
}
</style>
