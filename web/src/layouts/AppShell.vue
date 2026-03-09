<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useWebSocket } from '@/composables/useWebSocket'
import NavLink from '@/components/NavLink.vue'

const router = useRouter()
const auth = useAuthStore()
const { connected, connect, disconnect } = useWebSocket()

onMounted(() => {
  connect()
})

async function handleLogout() {
  disconnect()
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-screen bg-scout-900">
    <!-- Sidebar -->
    <aside class="w-64 flex flex-col border-r border-violet-500/10 bg-scout-800/60 backdrop-blur-xl">
      <div class="px-6 py-5 border-b border-violet-500/10">
        <h1 class="text-xl font-semibold text-white tracking-tight">
          <span class="text-violet-400">Scout</span>UTM
        </h1>
      </div>

      <nav class="flex-1 py-4 px-3 space-y-0.5">
        <NavLink to="/dashboard" label="Dashboard" />
        <NavLink to="/devices" label="Devices" />
        <NavLink to="/firewall" label="Firewall" />
        <NavLink to="/dns" label="DNS" />
        <NavLink to="/vpn" label="VPN" />
        <NavLink to="/ids" label="IDS Alerts" />
        <NavLink to="/parental" label="Parental" />
        <NavLink to="/network" label="Network" />
        <NavLink to="/system" label="System" />
      </nav>

      <div class="px-4 py-4 border-t border-violet-500/10">
        <div class="flex items-center gap-2 text-sm text-gray-500 mb-3">
          <span
            class="w-2 h-2 rounded-full"
            :class="connected ? 'bg-emerald-400' : 'bg-red-400'"
          />
          {{ connected ? 'Connected' : 'Disconnected' }}
        </div>
        <button
          class="w-full py-2 text-sm text-gray-500 hover:text-white hover:bg-white/5 rounded-lg transition-colors"
          @click="handleLogout"
        >
          Logout
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 overflow-auto p-6">
      <RouterView />
    </main>
  </div>
</template>
