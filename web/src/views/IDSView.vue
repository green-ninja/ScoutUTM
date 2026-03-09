<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api/ids'
import { useAlertsStore } from '@/stores/alerts'
import type { IDSConfig, IDSAlert } from '@/api/types'
import { ApiError } from '@/api/client'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import Toast from '@/components/Toast.vue'

const alertsStore = useAlertsStore()
const config = ref<IDSConfig | null>(null)
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)
const expandedAlert = ref<number | null>(null)

onMounted(load)

async function load() {
  try {
    const [cfg, alerts] = await Promise.all([api.getIDSConfig(), api.getAlerts()])
    config.value = cfg
    alertsStore.setAll(alerts)
  } catch {
    error.value = 'Failed to load IDS config'
  }
}

async function saveConfig() {
  if (!config.value) return
  try {
    config.value = await api.updateIDSConfig(config.value)
    toast.value = { message: 'IDS config updated', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

function severityClass(severity: number): string {
  switch (severity) {
    case 1: return 'bg-red-900 text-red-300'
    case 2: return 'bg-orange-900 text-orange-300'
    case 3: return 'bg-yellow-900 text-yellow-300'
    default: return 'bg-gray-700 text-gray-300'
  }
}

function severityLabel(severity: number): string {
  switch (severity) {
    case 1: return 'High'
    case 2: return 'Medium'
    case 3: return 'Low'
    default: return 'Info'
  }
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString()
  } catch {
    return ts
  }
}

function toggleAlert(i: number) {
  expandedAlert.value = expandedAlert.value === i ? null : i
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">IDS / IPS</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <p v-if="!config && !error" class="text-gray-500">Loading...</p>
    <p v-else-if="error" class="text-red-400">{{ error }}</p>

    <!-- Settings -->
    <template v-if="config">
      <div class="glass rounded-xl p-6 mb-6">
        <h3 class="text-lg font-semibold text-white mb-4">Settings</h3>
        <div class="space-y-4">
          <ToggleSwitch v-model="config.enabled" label="IDS Enabled" />
          <div>
            <label class="block text-sm text-gray-400 mb-1">Mode</label>
            <select v-model="config.mode" class="px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option value="ids">IDS (detect only)</option>
              <option value="ips">IPS (detect + block)</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Rulesets</label>
            <div class="space-y-1">
              <label v-for="rs in ['et-open', 'abuse-ch', 'tgreen']" :key="rs" class="flex items-center gap-2 text-sm text-gray-300">
                <input
                  type="checkbox"
                  :checked="config.rulesets.includes(rs)"
                  class="rounded bg-scout-700 border-scout-600"
                  @change="config!.rulesets = ($event.target as HTMLInputElement).checked
                    ? [...config!.rulesets, rs]
                    : config!.rulesets.filter(r => r !== rs)"
                />
                {{ rs }}
              </label>
            </div>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Home Network (CIDR)</label>
            <textarea
              :value="config.home_net.join('\n')"
              @input="config!.home_net = ($event.target as HTMLTextAreaElement).value.split('\n').filter(Boolean)"
              rows="3"
              class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm font-mono"
              placeholder="192.168.1.0/24"
            />
          </div>
          <div class="flex justify-end">
            <button class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded" @click="saveConfig">
              Save Settings
            </button>
          </div>
        </div>
      </div>
    </template>

    <!-- Alerts -->
    <div class="glass rounded-xl overflow-hidden">
      <div class="px-6 py-4 border-b border-scout-700">
        <h3 class="text-lg font-semibold text-white">Alerts</h3>
      </div>
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-gray-400 border-b border-scout-700">
            <th class="px-4 py-3">Time</th>
            <th class="px-4 py-3">Severity</th>
            <th class="px-4 py-3">Signature</th>
            <th class="px-4 py-3">Source</th>
            <th class="px-4 py-3">Destination</th>
            <th class="px-4 py-3">Action</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="(a, i) in [...alertsStore.alerts].reverse()" :key="i">
            <tr
              class="border-b border-scout-700 text-gray-300 hover:bg-scout-700/50 cursor-pointer"
              @click="toggleAlert(i)"
            >
              <td class="px-4 py-3">{{ formatTime(a.timestamp) }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-xs" :class="severityClass(a.alert.severity)">
                  {{ severityLabel(a.alert.severity) }}
                </span>
              </td>
              <td class="px-4 py-3 text-white max-w-xs truncate">{{ a.alert.signature }}</td>
              <td class="px-4 py-3 font-mono text-xs">{{ a.src_ip }}:{{ a.src_port }}</td>
              <td class="px-4 py-3 font-mono text-xs">{{ a.dest_ip }}:{{ a.dest_port }}</td>
              <td class="px-4 py-3">
                <span
                  class="px-2 py-0.5 rounded text-xs"
                  :class="a.alert.action === 'blocked' ? 'bg-red-900 text-red-300' : 'bg-yellow-900 text-yellow-300'"
                >
                  {{ a.alert.action }}
                </span>
              </td>
            </tr>
            <tr v-if="expandedAlert === i" class="border-b border-scout-700">
              <td colspan="6" class="px-4 py-3 bg-scout-900">
                <div class="text-sm space-y-1">
                  <p class="text-gray-400">SID: <span class="text-white">{{ a.alert.signature_id }}</span></p>
                  <p class="text-gray-400">Category: <span class="text-white">{{ a.alert.category }}</span></p>
                  <p class="text-gray-400">Protocol: <span class="text-white">{{ a.proto }}</span></p>
                  <p class="text-gray-400">Full signature: <span class="text-white">{{ a.alert.signature }}</span></p>
                </div>
              </td>
            </tr>
          </template>
          <tr v-if="!alertsStore.alerts.length">
            <td colspan="6" class="px-4 py-8 text-center text-gray-500">No alerts</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
