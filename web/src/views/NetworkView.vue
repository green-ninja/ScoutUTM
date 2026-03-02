<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiFetch } from '@/api/client'
import type { NetworkConfig, ZoneConfig, VLANConfig } from '@/api/types'
import { ApiError } from '@/api/client'
import Modal from '@/components/Modal.vue'
import Toast from '@/components/Toast.vue'

const config = ref<NetworkConfig | null>(null)
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)
const activeTab = ref<'interfaces' | 'zones' | 'vlans'>('interfaces')

const showAddVLAN = ref(false)
const newVLAN = ref<VLANConfig>({ id: 100, parent: 'eth1', zone: '', subnet: '', dhcp_range: { start: '', end: '' } })

onMounted(load)

async function load() {
  try {
    config.value = await apiFetch<NetworkConfig>('/network')
  } catch {
    error.value = 'Failed to load network config'
  }
}

async function addVLAN() {
  try {
    await apiFetch('/network/vlans', {
      method: 'POST',
      body: JSON.stringify(newVLAN.value),
    })
    showAddVLAN.value = false
    newVLAN.value = { id: 100, parent: 'eth1', zone: '', subnet: '', dhcp_range: { start: '', end: '' } }
    await load()
    toast.value = { message: 'VLAN added', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function deleteVLAN(id: number) {
  try {
    await apiFetch(`/network/vlans/${id}`, { method: 'DELETE' })
    await load()
    toast.value = { message: 'VLAN deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">Network</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <p v-if="!config && !error" class="text-gray-500">Loading...</p>
    <p v-else-if="error" class="text-red-400">{{ error }}</p>

    <template v-if="config">
      <!-- Tabs -->
      <div class="flex gap-4 mb-6 border-b border-scout-700">
        <button
          v-for="tab in (['interfaces', 'zones', 'vlans'] as const)"
          :key="tab"
          class="pb-2 text-sm font-medium transition-colors capitalize"
          :class="activeTab === tab ? 'text-white border-b-2 border-scout-accent' : 'text-gray-400 hover:text-white'"
          @click="activeTab = tab"
        >
          {{ tab }}
        </button>
      </div>

      <!-- Interfaces -->
      <div v-if="activeTab === 'interfaces'">
        <div class="bg-scout-800 rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-400 border-b border-scout-700">
                <th class="px-4 py-3">Name</th>
                <th class="px-4 py-3">Zone</th>
                <th class="px-4 py-3">Type</th>
                <th class="px-4 py-3">IPv4 Mode</th>
                <th class="px-4 py-3">Address</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="iface in config.interfaces" :key="iface.name" class="border-b border-scout-700 text-gray-300">
                <td class="px-4 py-3 text-white font-mono">{{ iface.name }}</td>
                <td class="px-4 py-3">{{ iface.zone }}</td>
                <td class="px-4 py-3">{{ iface.type }}</td>
                <td class="px-4 py-3">{{ iface.ipv4_mode }}</td>
                <td class="px-4 py-3 font-mono">{{ iface.ipv4_address || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Zones -->
      <div v-if="activeTab === 'zones'">
        <div class="bg-scout-800 rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-400 border-b border-scout-700">
                <th class="px-4 py-3">Name</th>
                <th class="px-4 py-3">Input</th>
                <th class="px-4 py-3">Forward</th>
                <th class="px-4 py-3">Output</th>
                <th class="px-4 py-3">Masquerade</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="zone in config.zones" :key="zone.name" class="border-b border-scout-700 text-gray-300">
                <td class="px-4 py-3 text-white">{{ zone.name }}</td>
                <td class="px-4 py-3">
                  <span class="px-2 py-0.5 rounded text-xs" :class="zone.input_policy === 'accept' ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'">
                    {{ zone.input_policy }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <span class="px-2 py-0.5 rounded text-xs" :class="zone.forward_policy === 'accept' ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'">
                    {{ zone.forward_policy }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <span class="px-2 py-0.5 rounded text-xs" :class="zone.output_policy === 'accept' ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'">
                    {{ zone.output_policy }}
                  </span>
                </td>
                <td class="px-4 py-3">{{ zone.masquerade ? 'Yes' : 'No' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- VLANs -->
      <div v-if="activeTab === 'vlans'">
        <div class="flex justify-end mb-4">
          <button class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded" @click="showAddVLAN = true">
            Add VLAN
          </button>
        </div>
        <div class="bg-scout-800 rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-400 border-b border-scout-700">
                <th class="px-4 py-3">ID</th>
                <th class="px-4 py-3">Parent</th>
                <th class="px-4 py-3">Zone</th>
                <th class="px-4 py-3">Subnet</th>
                <th class="px-4 py-3">DHCP Range</th>
                <th class="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="vlan in config.vlans" :key="vlan.id" class="border-b border-scout-700 text-gray-300">
                <td class="px-4 py-3 text-white">{{ vlan.id }}</td>
                <td class="px-4 py-3 font-mono">{{ vlan.parent }}</td>
                <td class="px-4 py-3">{{ vlan.zone }}</td>
                <td class="px-4 py-3 font-mono">{{ vlan.subnet }}</td>
                <td class="px-4 py-3 font-mono text-xs">{{ vlan.dhcp_range.start }} – {{ vlan.dhcp_range.end }}</td>
                <td class="px-4 py-3">
                  <button class="text-red-400 hover:text-red-300 text-sm" @click="deleteVLAN(vlan.id)">Delete</button>
                </td>
              </tr>
              <tr v-if="!config.vlans.length">
                <td colspan="6" class="px-4 py-8 text-center text-gray-500">No VLANs configured</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- Add VLAN Modal -->
    <Modal v-if="showAddVLAN" title="Add VLAN" @close="showAddVLAN = false">
      <form class="space-y-4" @submit.prevent="addVLAN">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">VLAN ID</label>
            <input v-model.number="newVLAN.id" type="number" min="1" max="4094" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Parent Interface</label>
            <input v-model="newVLAN.parent" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Zone</label>
            <input v-model="newVLAN.zone" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="e.g. iot" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Subnet</label>
            <input v-model="newVLAN.subnet" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="192.168.10.0/24" required />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">DHCP Start</label>
            <input v-model="newVLAN.dhcp_range.start" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="192.168.10.100" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">DHCP End</label>
            <input v-model="newVLAN.dhcp_range.end" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="192.168.10.200" />
          </div>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showAddVLAN = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded">Add</button>
        </div>
      </form>
    </Modal>
  </div>
</template>
