<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import HomeAccountMenu from './smirel/components/HomeAccountMenu.vue'
import WorkspaceShell from './smirel/components/WorkspaceShell.vue'

const route = useRoute()
const useWorkspace = computed(() => route.meta.shell === 'workspace')
</script>

<template>
  <RouterView v-slot="{ Component }">
    <template v-if="useWorkspace">
      <WorkspaceShell>
        <Transition name="workspace-route" mode="out-in">
          <div :key="route.path" class="workspace-route-stage">
            <component :is="Component" />
          </div>
        </Transition>
      </WorkspaceShell>
      <Teleport to=".workspace-topbar-actions">
        <HomeAccountMenu variant="workspace" />
      </Teleport>
    </template>
    <Transition v-else name="app-route" mode="out-in">
      <component :is="Component" :key="route.path" />
    </Transition>
  </RouterView>
</template>
