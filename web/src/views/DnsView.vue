<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import * as api from '@/api/dns'
import type { DNSConfig, DNSQueryEntry } from '@/api/types'
import { ApiError } from '@/api/client'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import Toast from '@/components/Toast.vue'

const config = ref<DNSConfig | null>(null)
const queryLog = ref<DNSQueryEntry[]>([])
const loading = ref(false)
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)

const upstreamText = ref('')
const customBlockText = ref('')
const customAllowText = ref('')

const pageSize = 20
const currentPage = ref(1)
const totalPages = computed(() => Math.max(1, Math.ceil(queryLog.value.length / pageSize)))
const pagedQueryLog = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return queryLog.value.slice(start, start + pageSize)
})

onMounted(load)

async function load() {
  try {
    const [c, q] = await Promise.all([api.getDns(), api.getQueryLog()])
    config.value = c
    queryLog.value = q
    upstreamText.value = c.upstream.join('\n')
    customBlockText.value = c.custom_block.join('\n')
    customAllowText.value = c.custom_allow.join('\n')
  } catch {
    toast.value = { message: 'Failed to load DNS config', type: 'error' }
  }
}

async function save() {
  if (!config.value) return
  loading.value = true
  try {
    const updated: DNSConfig = {
      ...config.value,
      upstream: upstreamText.value.split('\n').map((s) => s.trim()).filter(Boolean),
      custom_block: customBlockText.value.split('\n').map((s) => s.trim()).filter(Boolean),
      custom_allow: customAllowText.value.split('\n').map((s) => s.trim()).filter(Boolean),
    }
    config.value = await api.updateDns(updated)
    upstreamText.value = config.value.upstream.join('\n')
    customBlockText.value = config.value.custom_block.join('\n')
    customAllowText.value = config.value.custom_allow.join('\n')
    toast.value = { message: 'DNS settings saved', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  } finally {
    loading.value = false
  }
}

function formatTime(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function isBlocked(domain: string): boolean {
  if (!config.value) return false
  return config.value.custom_block.includes(domain)
}

function isAllowed(domain: string): boolean {
  if (!config.value) return false
  return config.value.custom_allow.includes(domain)
}

async function quickBlock(domain: string) {
  if (!config.value) return
  const block = config.value.custom_block.filter((d) => d !== domain)
  block.push(domain)
  const allow = config.value.custom_allow.filter((d) => d !== domain)
  try {
    config.value = await api.updateDns({ ...config.value, custom_block: block, custom_allow: allow })
    customBlockText.value = config.value.custom_block.join('\n')
    customAllowText.value = config.value.custom_allow.join('\n')
    toast.value = { message: `Blocked ${domain}`, type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function quickUnblock(domain: string) {
  if (!config.value) return
  const block = config.value.custom_block.filter((d) => d !== domain)
  const allow = config.value.custom_allow.filter((d) => d !== domain)
  allow.push(domain)
  try {
    config.value = await api.updateDns({ ...config.value, custom_block: block, custom_allow: allow })
    customBlockText.value = config.value.custom_block.join('\n')
    customAllowText.value = config.value.custom_allow.join('\n')
    toast.value = { message: `Unblocked ${domain}`, type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">DNS</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <div v-if="config" class="space-y-6">
      <!-- Filtering toggles -->
      <div class="glass rounded-xl p-6">
        <h3 class="text-sm font-medium text-gray-400 mb-4">Filtering</h3>
        <div class="space-y-3">
          <ToggleSwitch v-model="config.block_threats" label="Block potentially dangerous and unwanted content" />
          <p class="text-xs text-gray-500 mt-2">Blocks ads, tracking, affiliate links, telemetry, phishing, malware, scams and more. The blocklist is refreshed on every reboot.</p>
        </div>
      </div>

      <!-- Encrypted DNS -->
      <div class="glass rounded-xl p-6">
        <h3 class="text-sm font-medium text-gray-400 mb-4">Encrypted DNS</h3>
        <ToggleSwitch v-model="config.use_dot" label="DNS-over-TLS (DoT)" />
      </div>

      <!-- Query logging -->
      <div class="glass rounded-xl p-6">
        <h3 class="text-sm font-medium text-gray-400 mb-4">Logging</h3>
        <ToggleSwitch v-model="config.query_logging" label="Enable query logging" />
      </div>

      <!-- Upstream DNS -->
      <div class="glass rounded-xl p-6">
        <h3 class="text-sm font-medium text-gray-400 mb-2">Upstream DNS Servers</h3>
        <p class="text-xs text-gray-500 mb-2">One IP per line</p>
        <textarea
          v-model="upstreamText"
          rows="3"
          class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm font-mono"
        />
      </div>

      <!-- Custom block/allow -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div class="glass rounded-xl p-6">
          <h3 class="text-sm font-medium text-gray-400 mb-2">Custom Block List</h3>
          <p class="text-xs text-gray-500 mb-2">One domain per line</p>
          <textarea
            v-model="customBlockText"
            rows="4"
            class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm font-mono"
          />
        </div>
        <div class="glass rounded-xl p-6">
          <h3 class="text-sm font-medium text-gray-400 mb-2">Custom Allow List</h3>
          <p class="text-xs text-gray-500 mb-2">One domain per line</p>
          <textarea
            v-model="customAllowText"
            rows="4"
            class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm font-mono"
          />
        </div>
      </div>

      <!-- Save button -->
      <div class="flex justify-end">
        <button
          class="px-6 py-2 bg-violet-600 hover:bg-violet-500 text-white font-medium rounded transition-colors disabled:opacity-50"
          :disabled="loading"
          @click="save"
        >
          {{ loading ? 'Saving...' : 'Save' }}
        </button>
      </div>

      <!-- Query Log -->
      <div class="glass rounded-xl p-6">
        <h3 class="text-sm font-medium text-gray-400 mb-3">Query Log</h3>
        <div v-if="queryLog.length" class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead>
              <tr class="text-gray-500 border-b border-scout-700">
                <th class="pb-2 font-medium">Time</th>
                <th class="pb-2 font-medium">Device</th>
                <th class="pb-2 font-medium">Domain</th>
                <th class="pb-2 font-medium">Status</th>
                <th class="pb-2 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(entry, i) in pagedQueryLog" :key="i" class="border-b border-scout-700 last:border-0">
                <td class="py-2 text-gray-400 whitespace-nowrap">{{ formatTime(entry.timestamp) }}</td>
                <td class="py-2 text-white">{{ entry.device_name || entry.device_ip }}</td>
                <td class="py-2 text-white font-mono text-xs">{{ entry.domain }}</td>
                <td class="py-2">
                  <span
                    class="px-2 py-0.5 rounded text-xs font-medium"
                    :class="entry.action === 'block' ? 'bg-red-900 text-red-300' : 'bg-green-900 text-green-300'"
                  >
                    {{ entry.action === 'block' ? 'Blocked' : 'Allowed' }}
                  </span>
                </td>
                <td class="py-2 text-right">
                  <button
                    v-if="!isBlocked(entry.domain)"
                    class="px-2 py-0.5 text-xs text-red-400 hover:text-red-300 hover:bg-red-900/30 rounded transition-colors"
                    @click.stop="quickBlock(entry.domain)"
                  >
                    Block
                  </button>
                  <button
                    v-if="isBlocked(entry.domain) || (entry.action === 'block' && !isAllowed(entry.domain))"
                    class="px-2 py-0.5 text-xs text-green-400 hover:text-green-300 hover:bg-green-900/30 rounded transition-colors"
                    @click.stop="quickUnblock(entry.domain)"
                  >
                    Unblock
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="totalPages > 1" class="flex items-center justify-between mt-4 pt-3 border-t border-scout-700">
            <span class="text-xs text-gray-500">Page {{ currentPage }} of {{ totalPages }}</span>
            <div class="flex gap-2">
              <button
                class="px-3 py-1 text-sm rounded bg-scout-700 text-gray-300 hover:bg-scout-600 disabled:opacity-40 disabled:cursor-not-allowed"
                :disabled="currentPage <= 1"
                @click="currentPage--"
              >
                Previous
              </button>
              <button
                class="px-3 py-1 text-sm rounded bg-scout-700 text-gray-300 hover:bg-scout-600 disabled:opacity-40 disabled:cursor-not-allowed"
                :disabled="currentPage >= totalPages"
                @click="currentPage++"
              >
                Next
              </button>
            </div>
          </div>
        </div>
        <p v-else class="text-gray-500 text-sm">No queries logged</p>
      </div>
    </div>

    <p v-else class="text-gray-500">Loading...</p>
  </div>
</template>
