<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api, getErrorMessage } from '../core/api'

type Account = { id: number; name?: string; notes?: string | null; platform?: string; concurrency?: number; priority?: number; rate_multiplier?: number; group_ids?: number[] }
type Group = { id: number; name?: string; platform?: string; status?: string }

const props = defineProps<{ show: boolean; account: Account | null }>()
const emit = defineEmits<{ close: []; updated: [] }>()
const name = ref('')
const notes = ref('')
const concurrency = ref(1)
const priority = ref(0)
const rateMultiplier = ref(1)
const groupIds = ref<number[]>([])
const groups = ref<Group[]>([])
const saving = ref(false)
const error = ref('')
const canSave = computed(() => Boolean(props.account && name.value.trim() && !saving.value))

function toggleGroup(id: number) { groupIds.value = groupIds.value.includes(id) ? groupIds.value.filter((value) => value !== id) : [...groupIds.value, id] }
function close() { if (!saving.value) emit('close') }

async function load() {
  const account = props.account
  if (!account) return
  name.value = account.name || ''
  notes.value = account.notes || ''
  concurrency.value = Math.max(1, Number(account.concurrency || 1))
  priority.value = Math.max(0, Number(account.priority || 0))
  rateMultiplier.value = Math.max(0, Number(account.rate_multiplier ?? 1))
  groupIds.value = [...(account.group_ids || [])]
  error.value = ''
  try {
    const response = await api.get<Group[]>('/admin/groups/all', { params: { platform: account.platform || undefined } })
    groups.value = Array.isArray(response.data) ? response.data : []
  } catch (caught) { groups.value = []; error.value = getErrorMessage(caught) }
}

async function submit() {
  const account = props.account
  if (!account || !name.value.trim()) return
  saving.value = true
  error.value = ''
  try {
    await api.put(`/admin/accounts/${account.id}`, {
      name: name.value.trim(), notes: notes.value.trim(),
      concurrency: Math.max(1, Math.round(Number(concurrency.value) || 1)),
      priority: Math.max(0, Math.round(Number(priority.value) || 0)),
      rate_multiplier: Math.max(0, Number(rateMultiplier.value) || 0), group_ids: groupIds.value,
    })
    emit('updated'); emit('close')
  } catch (caught) { error.value = getErrorMessage(caught) } finally { saving.value = false }
}

watch(() => props.show, (visible) => { if (visible) void load() })
</script>

<template>
  <Teleport to="body"><div v-if="show && account" class="backdrop" @click.self="close"><section class="dialog" role="dialog" aria-modal="true" aria-labelledby="edit-account-title">
    <header><div><span>UPSTREAM ACCOUNT</span><h2 id="edit-account-title">编辑上游账户</h2><p>凭据不会在此显示或被覆盖。</p></div><button type="button" aria-label="关闭" @click="close">×</button></header>
    <main>
      <label>账户名称<input v-model="name" maxlength="80" /></label>
      <label>备注<textarea v-model="notes" rows="2" maxlength="300" /></label>
      <div class="numbers"><label>最大并发<input v-model.number="concurrency" type="number" min="1" max="10000" /></label><label>优先级<input v-model.number="priority" type="number" min="0" max="9999" /></label><label>倍率<input v-model.number="rateMultiplier" type="number" min="0" step="0.01" /></label></div>
      <section><strong>调度分组</strong><div v-if="groups.length" class="groups"><button v-for="group in groups" :key="group.id" type="button" :class="{ selected: groupIds.includes(group.id) }" @click="toggleGroup(group.id)">{{ groupIds.includes(group.id) ? '✓ ' : '' }}{{ group.name || `Group #${group.id}` }}</button></div><p v-else>没有可选择的同平台分组。</p></section>
      <p v-if="error" class="error">{{ error }}</p>
    </main>
    <footer><button type="button" @click="close">取消</button><button type="button" :disabled="!canSave" @click="submit">{{ saving ? '保存中' : '保存修改' }}</button></footer>
  </section></div></Teleport>
</template>

<style scoped>
.backdrop{position:fixed;inset:0;z-index:1400;display:grid;place-items:center;padding:24px;background:#050609cc}.dialog{width:min(620px,100%);border:1px solid #30343d;border-radius:10px;background:#121419;color:#f4f6f8;box-shadow:0 24px 80px #0008}.dialog header,.dialog footer{display:flex;justify-content:space-between;align-items:center;padding:20px 24px;border-bottom:1px solid #282b32}.dialog footer{border-top:1px solid #282b32;border-bottom:0;justify-content:flex-end;gap:10px}.dialog header span{font-size:11px;color:#8d96a2}.dialog h2{margin:5px 0;font-size:20px}.dialog p{margin:0;color:#929aa5;font-size:13px}.dialog main{padding:20px 24px;display:grid;gap:16px}.dialog label{display:grid;gap:7px;color:#cbd1d8;font-size:13px}.dialog input,.dialog textarea{box-sizing:border-box;width:100%;border:1px solid #353a44;border-radius:6px;background:#0d0f13;color:#f4f6f8;padding:9px}.numbers{display:grid;grid-template-columns:repeat(3,1fr);gap:12px}.groups{display:flex;flex-wrap:wrap;gap:8px;margin-top:8px}.dialog button{border:1px solid #3a404b;border-radius:6px;background:#1a1d23;color:#e4e8ec;padding:8px 12px;cursor:pointer}.groups button.selected,.dialog footer button:last-child{border-color:#52ce9e;background:#1a493a;color:#ddfff1}.error{color:#f08b91!important}@media(max-width:560px){.numbers{grid-template-columns:1fr}.dialog{max-height:calc(100vh - 30px);overflow:auto}}
</style>
