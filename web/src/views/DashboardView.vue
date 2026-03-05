<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useStatsStore } from '@/stores/stats'
import StatCard from '@/components/StatCard.vue'
import ThroughputChart from '@/components/ThroughputChart.vue'
import * as vpnApi from '@/api/vpn'
import * as dnsApi from '@/api/dns'
import type { WireGuardConfig, DNSConfig, DNSQueryEntry } from '@/api/types'

const stats = useStatsStore()

const vpn = ref<WireGuardConfig | null>(null)
const dns = ref<DNSConfig | null>(null)
const queryLog = ref<DNSQueryEntry[]>([])
const summaryError = ref(false)

let refreshTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  try {
    const [v, d, q] = await Promise.all([vpnApi.getWireGuard(), dnsApi.getDns(), dnsApi.getQueryLog()])
    vpn.value = v
    dns.value = d
    queryLog.value = q.slice(0, 20)
  } catch {
    summaryError.value = true
  }

  refreshTimer = setInterval(async () => {
    try {
      queryLog.value = (await dnsApi.getQueryLog()).slice(0, 20)
    } catch { /* ignore */ }
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

const cpuColor = computed(() => {
  const v = stats.latest?.cpu_percent ?? 0
  if (v >= 80) return 'text-red-400'
  if (v >= 60) return 'text-yellow-400'
  return 'text-green-400'
})

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function formatBytes(bps: number): string {
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(1)} MB/s`
  if (bps >= 1_000) return `${(bps / 1_000).toFixed(1)} KB/s`
  return `${bps} B/s`
}

function formatTime(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">Dashboard</h2>

    <!-- Stat cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <StatCard
        label="CPU"
        :value="stats.latest ? `${stats.latest.cpu_percent.toFixed(0)}%` : '--'"
        :color-class="cpuColor"
      />
      <StatCard
        label="Memory"
        :value="stats.latest ? `${stats.latest.mem_used_mb} / ${stats.latest.mem_total_mb} MB` : '--'"
      />
      <StatCard
        label="Connections"
        :value="stats.latest ? String(stats.latest.conn_count) : '--'"
      />
      <StatCard
        label="Uptime"
        :value="stats.latest ? formatUptime(stats.latest.uptime_seconds) : '--'"
      />
    </div>

    <!-- Throughput chart -->
    <div class="bg-scout-800 rounded-lg p-4 mb-6">
      <h3 class="text-sm font-medium text-gray-400 mb-3">WAN Throughput</h3>
      <ThroughputChart :history="stats.history" />
    </div>

    <!-- Summary cards -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
      <div class="bg-scout-800 rounded-lg p-4">
        <h3 class="text-sm font-medium text-gray-400 mb-2">WireGuard VPN</h3>
        <template v-if="vpn">
          <p class="text-white">
            Status:
            <span :class="vpn.enabled ? 'text-green-400' : 'text-gray-500'">
              {{ vpn.enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </p>
          <p class="text-gray-400 text-sm">{{ vpn.peers.length }} peer(s)</p>
        </template>
        <p v-else-if="summaryError" class="text-gray-500 text-sm">Unable to load</p>
        <p v-else class="text-gray-500 text-sm">Loading...</p>
      </div>

      <div class="bg-scout-800 rounded-lg p-4">
        <h3 class="text-sm font-medium text-gray-400 mb-2">DNS Filtering</h3>
        <template v-if="dns">
          <div class="flex flex-wrap gap-2 text-sm">
            <span v-if="dns.block_malware" class="px-2 py-0.5 bg-green-900 text-green-300 rounded">Malware</span>
            <span v-if="dns.block_phishing" class="px-2 py-0.5 bg-green-900 text-green-300 rounded">Phishing</span>
            <span v-if="dns.block_ads" class="px-2 py-0.5 bg-green-900 text-green-300 rounded">Ads</span>
            <span v-if="dns.use_dot" class="px-2 py-0.5 bg-blue-900 text-blue-300 rounded">DoT</span>
            <span v-if="dns.use_doh" class="px-2 py-0.5 bg-blue-900 text-blue-300 rounded">DoH</span>
          </div>
          <p v-if="!dns.block_malware && !dns.block_phishing && !dns.block_ads" class="text-gray-500 text-sm">
            No filters active
          </p>
        </template>
        <p v-else-if="summaryError" class="text-gray-500 text-sm">Unable to load</p>
        <p v-else class="text-gray-500 text-sm">Loading...</p>
      </div>
    </div>

    <!-- Recent DNS Queries -->
    <div class="bg-scout-800 rounded-lg p-4">
      <h3 class="text-sm font-medium text-gray-400 mb-3">Recent DNS Queries</h3>
      <div v-if="queryLog.length" class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead>
            <tr class="text-gray-500 border-b border-scout-700">
              <th class="pb-2 font-medium">Time</th>
              <th class="pb-2 font-medium">Device</th>
              <th class="pb-2 font-medium">Domain</th>
              <th class="pb-2 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(entry, i) in queryLog" :key="i" class="border-b border-scout-700 last:border-0">
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
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="text-gray-500 text-sm">No queries logged</p>
    </div>
  </div>
</template>
