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
const eyebrow = computed(() => locale.value === 'zh-CN' ? 'BILLING & PLANS' : 'BILLING & PLANS')
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
  width: min(100%, 1480px);
  margin: 0 auto;
  padding-bottom: 52px;
}

.billing-heading {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 32px;
  margin-bottom: 24px;
  padding: 6px 0 22px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
}

.billing-heading::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: -1px;
  width: 86px;
  height: 1px;
  background: linear-gradient(90deg, #79c4f5, rgba(121, 196, 245, 0));
}

.billing-heading-copy {
  min-width: 0;
}

.billing-eyebrow {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  margin-bottom: 9px;
  padding: 0 9px;
  border: 1px solid rgba(121, 196, 245, 0.18);
  border-radius: 999px;
  background: rgba(121, 196, 245, 0.06);
  color: rgba(139, 207, 250, 0.82);
  font-size: 0.63rem;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.billing-heading h1 {
  margin: 0;
  color: #f5f8fb;
  font-size: clamp(2rem, 2.5vw, 2.55rem);
  font-weight: 720;
  letter-spacing: -0.045em;
}

.billing-heading p {
  max-width: 650px;
  margin: 8px 0 0;
  color: rgba(212, 222, 232, 0.56);
  font-size: 0.86rem;
  line-height: 1.7;
}

.billing-assurance {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 8px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.025);
  color: rgba(224, 232, 239, 0.5);
  font-size: 0.72rem;
}

.billing-assurance-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #62d3a5;
  box-shadow: 0 0 0 4px rgba(98, 211, 165, 0.08);
}

:deep(.billing-page) {
  gap: 22px;
}

:deep(.billing-mode) {
  gap: 4px;
  padding: 5px;
  border-color: rgba(255, 255, 255, 0.075);
  border-radius: 14px;
  background: rgba(10, 13, 18, 0.72);
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.025), 0 10px 30px rgba(0, 0, 0, 0.14);
}

:deep(.billing-mode button) {
  min-width: 104px;
  min-height: 38px;
  padding: 0 18px;
  border-radius: 10px;
  transition: color 160ms ease, background 160ms ease, box-shadow 160ms ease;
}

:deep(.billing-mode button:hover:not(.active)) {
  color: rgba(255, 255, 255, 0.78);
  background: rgba(255, 255, 255, 0.035);
}

:deep(.billing-mode button.active) {
  background: linear-gradient(180deg, rgba(52, 80, 107, 0.72), rgba(34, 54, 73, 0.78));
  color: #f7fbff;
  box-shadow: 0 5px 14px rgba(0, 0, 0, 0.22), inset 0 1px rgba(255, 255, 255, 0.06);
}

:deep(.plan-list) {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(330px, 1fr));
  gap: 18px;
  align-items: stretch;
}

:deep(.plan-card) {
  position: relative;
  isolation: isolate;
  min-height: 318px;
  gap: 0;
  overflow: hidden;
  padding: 24px 24px 20px;
  border: 1px solid rgba(255, 255, 255, 0.075);
  border-radius: 18px;
  background:
    radial-gradient(circle at 100% 0%, rgba(121, 196, 245, 0.07), transparent 34%),
    linear-gradient(155deg, rgba(19, 24, 31, 0.97), rgba(10, 13, 18, 0.98));
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.025), 0 12px 36px rgba(0, 0, 0, 0.12);
  cursor: pointer;
  transition: transform 180ms ease, border-color 180ms ease, box-shadow 180ms ease, background 180ms ease;
}

:deep(.plan-card)::after {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  pointer-events: none;
  opacity: 0;
  background: linear-gradient(135deg, rgba(121, 196, 245, 0.08), transparent 48%);
  transition: opacity 180ms ease;
}

:deep(.plan-card:hover) {
  transform: translateY(-3px);
  border-color: rgba(121, 196, 245, 0.24);
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.04), 0 20px 52px rgba(0, 0, 0, 0.24);
}

:deep(.plan-card:hover)::after,
:deep(.plan-card.selected)::after {
  opacity: 1;
}

:deep(.plan-card.selected) {
  border-color: rgba(121, 196, 245, 0.68);
  box-shadow: 0 0 0 1px rgba(121, 196, 245, 0.08), 0 20px 54px rgba(16, 49, 72, 0.2);
}

:deep(.plan-card.selected)::before {
  content: '';
  position: absolute;
  top: 0;
  left: 20px;
  right: 20px;
  height: 2px;
  border-radius: 0 0 999px 999px;
  background: linear-gradient(90deg, transparent, #79c4f5 22%, #8dd0fb 78%, transparent);
  box-shadow: 0 0 16px rgba(121, 196, 245, 0.44);
}

:deep(.plan-card header) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  min-height: 84px;
  padding-bottom: 17px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

:deep(.plan-card header strong) {
  max-width: 100%;
  color: #f4f7fa;
  font-size: 1.02rem;
  font-weight: 680;
  line-height: 1.42;
  letter-spacing: -0.015em;
}

:deep(.plan-card header .price) {
  display: inline-flex;
  align-items: baseline;
  color: #7ec8f7;
  font-size: 1.46rem;
  font-weight: 720;
  line-height: 1;
  letter-spacing: -0.025em;
  text-shadow: 0 0 20px rgba(121, 196, 245, 0.1);
}

:deep(.plan-card > p) {
  min-height: 42px;
  margin: 16px 0 0;
  color: rgba(204, 216, 227, 0.54);
  font-size: 0.76rem;
  line-height: 1.65;
}

:deep(.plan-card ul) {
  flex: 1;
  gap: 9px;
  margin-top: 17px;
}

:deep(.plan-card ul li) {
  position: relative;
  padding-left: 22px;
  color: rgba(221, 230, 238, 0.7);
  font-size: 0.76rem;
  line-height: 1.5;
}

:deep(.plan-card ul li)::before {
  content: '✓';
  top: 0.05em;
  left: 1px;
  display: grid;
  width: 15px;
  height: 15px;
  place-items: center;
  border: 1px solid rgba(121, 196, 245, 0.22);
  border-radius: 50%;
  background: rgba(121, 196, 245, 0.07);
  color: #82caf8;
  font-size: 0.58rem;
  font-weight: 800;
}

:deep(.plan-card .primary) {
  width: 100%;
  height: 45px;
  margin-top: 22px;
  border-color: rgba(121, 196, 245, 0.22);
  border-radius: 11px;
  background: linear-gradient(180deg, rgba(53, 91, 119, 0.74), rgba(36, 66, 89, 0.82));
  color: #eef8ff;
  font-size: 0.82rem;
  font-weight: 660;
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.07);
  transition: transform 150ms ease, background 150ms ease, border-color 150ms ease, box-shadow 150ms ease;
}

:deep(.plan-card .primary:hover:not(:disabled)) {
  transform: translateY(-1px);
  border-color: rgba(121, 196, 245, 0.48);
  background: linear-gradient(180deg, rgba(70, 125, 165, 0.92), rgba(45, 85, 115, 0.96));
  box-shadow: 0 8px 22px rgba(31, 90, 130, 0.2), inset 0 1px rgba(255, 255, 255, 0.1);
}

:deep(.plan-card.selected .primary) {
  border-color: #79c4f5;
  background: linear-gradient(180deg, #85cdf9, #68b7e9);
  color: #08121a;
}

:deep(.plan-card .primary:disabled) {
  opacity: 0.42;
}

:deep(.empty) {
  min-height: 170px;
  display: grid;
  place-items: center;
  border-radius: 18px;
  background: linear-gradient(155deg, rgba(18, 22, 28, 0.9), rgba(10, 13, 18, 0.94));
}

@media (max-width: 980px) {
  .billing-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 16px;
  }

  :deep(.plan-list) {
    grid-template-columns: repeat(auto-fit, minmax(290px, 1fr));
  }
}

@media (max-width: 640px) {
  .billing-route-page {
    padding-bottom: 32px;
  }

  .billing-heading {
    margin-bottom: 18px;
    padding-bottom: 18px;
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

  :deep(.plan-card) {
    min-height: 0;
    padding: 21px 20px 18px;
    border-radius: 16px;
  }

  :deep(.plan-card header) {
    min-height: 0;
  }
}
</style>
