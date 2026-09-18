<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const authorizeUrl = ref('')
const error = ref('')

onMounted(() => {
  authorizeUrl.value = sessionStorage.getItem('smirel.payment.authorize_url') || ''
  sessionStorage.removeItem('smirel.payment.authorize_url')
  if (!authorizeUrl.value) {
    error.value = t('payment.errorCreateOrder')
  }
})

function go() {
  if (!authorizeUrl.value) return
  window.location.href = authorizeUrl.value
}
</script>

<template>
  <section class="payment-wechat-oauth">
    <header>
      <span class="eyebrow">{{ t('payment.oauthTitle') }}</span>
    </header>
    <p class="hint">{{ t('payment.oauthHint') }}</p>
    <p v-if="error" class="error">{{ error }}</p>
    <div class="actions">
      <button class="primary" type="button" :disabled="!authorizeUrl" @click="go">
        {{ t('payment.oauthOpen') }}
      </button>
      <RouterLink class="ghost" to="/subscriptions">{{ t('payment.resultBack') }}</RouterLink>
    </div>
  </section>
</template>

<style scoped>
.payment-wechat-oauth {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(12, 16, 22, 0.95);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.payment-wechat-oauth .eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.45);
}
.payment-wechat-oauth .hint {
  margin: 0;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.7);
  line-height: 1.6;
}
.payment-wechat-oauth .error {
  margin: 0;
  color: #f48b8b;
  font-size: 0.78rem;
}
.actions {
  display: flex;
  gap: 8px;
}
.actions button,
.actions a {
  flex: 1;
  min-height: 38px;
  padding: 0 14px;
  border-radius: 8px;
  border: 1px solid transparent;
  font-size: 0.78rem;
  font-family: inherit;
  cursor: pointer;
  text-align: center;
  line-height: 36px;
  text-decoration: none;
}
.actions .primary {
  background: #09bb07;
  border-color: #09bb07;
  color: #fff;
  font-weight: 600;
}
.actions .primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.actions .ghost {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.78);
}
</style>
