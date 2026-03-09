<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import * as api from '@/api/devices'
import * as parentalApi from '@/api/parental'
import { useDevicesStore } from '@/stores/devices'
import type { Device, ParentalProfile } from '@/api/types'
import { ApiError } from '@/api/client'
import Modal from '@/components/Modal.vue'
import Toast from '@/components/Toast.vue'

const devicesStore = useDevicesStore()
const devices = ref<Device[]>([])
const profiles = ref<ParentalProfile[]>([])
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)
const sortBy = ref<'name' | 'ip' | 'status'>('ip')

const editDevice = ref<Device | null>(null)
const editName = ref('')
const editProfileId = ref('')

onMounted(load)

async function load() {
  try {
    const [d, p] = await Promise.all([api.getDevices(), parentalApi.getProfiles()])
    devices.value = d
    profiles.value = p
    devicesStore.push(d)
  } catch {
    error.value = 'Failed to load devices'
  }
}

const sorted = computed(() => {
  const list = [...devices.value]
  switch (sortBy.value) {
    case 'name':
      return list.sort((a, b) => (a.name || a.hostname).localeCompare(b.name || b.hostname))
    case 'status':
      return list.sort((a, b) => (b.online ? 1 : 0) - (a.online ? 1 : 0))
    default:
      return list.sort((a, b) => a.ip.localeCompare(b.ip, undefined, { numeric: true }))
  }
})

function openEdit(d: Device) {
  editDevice.value = d
  editName.value = d.name
  editProfileId.value = d.profile_id
}

async function saveEdit() {
  if (!editDevice.value) return
  try {
    await api.updateDevice(editDevice.value.mac, {
      name: editName.value,
      profile_id: editProfileId.value,
    })
    editDevice.value = null
    await load()
    toast.value = { message: 'Device updated', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

function profileName(id: string): string {
  const p = profiles.value.find((p) => p.id === id)
  return p ? p.name : ''
}

// Listen for WebSocket device updates.
devicesStore.$subscribe(() => {
  if (devicesStore.devices.length > 0) {
    devices.value = devicesStore.devices
  }
})
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">Devices</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <p v-if="!devices.length && !error" class="text-gray-500">Loading...</p>
    <p v-else-if="error" class="text-red-400">{{ error }}</p>

    <template v-if="devices.length">
      <div class="flex gap-4 mb-4">
        <label class="text-sm text-gray-400">
          Sort by:
          <select v-model="sortBy" class="ml-2 px-2 py-1 bg-scout-700 border border-scout-600 rounded text-white text-sm">
            <option value="ip">IP</option>
            <option value="name">Name</option>
            <option value="status">Status</option>
          </select>
        </label>
      </div>

      <div class="glass rounded-xl overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-400 border-b border-scout-700">
              <th class="px-4 py-3">Status</th>
              <th class="px-4 py-3">Name</th>
              <th class="px-4 py-3">IP</th>
              <th class="px-4 py-3">MAC</th>
              <th class="px-4 py-3">Manufacturer</th>
              <th class="px-4 py-3">Profile</th>
              <th class="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="d in sorted"
              :key="d.mac"
              class="border-b border-scout-700 text-gray-300 hover:bg-scout-700/50 cursor-pointer"
              @click="openEdit(d)"
            >
              <td class="px-4 py-3">
                <span
                  class="inline-block w-2.5 h-2.5 rounded-full"
                  :class="d.online ? 'bg-green-400' : 'bg-gray-500'"
                />
              </td>
              <td class="px-4 py-3 text-white">{{ d.name || d.hostname || '—' }}</td>
              <td class="px-4 py-3">{{ d.ip }}</td>
              <td class="px-4 py-3 font-mono text-xs">{{ d.mac }}</td>
              <td class="px-4 py-3 text-gray-400 text-xs">{{ d.manufacturer || '—' }}</td>
              <td class="px-4 py-3">
                <span v-if="d.profile_id" class="px-2 py-0.5 rounded text-xs bg-purple-900 text-purple-300">
                  {{ profileName(d.profile_id) }}
                </span>
              </td>
              <td class="px-4 py-3 text-scout-accent text-sm">Edit</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <!-- Edit Modal -->
    <Modal v-if="editDevice" title="Edit Device" @close="editDevice = null">
      <form class="space-y-4" @submit.prevent="saveEdit">
        <div>
          <label class="block text-sm text-gray-400 mb-1">MAC Address</label>
          <p class="text-white font-mono text-sm">{{ editDevice.mac }}</p>
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">Custom Name</label>
          <input v-model="editName" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="e.g. My Laptop" />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">Parental Profile</label>
          <select v-model="editProfileId" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
            <option value="">None</option>
            <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="editDevice = null">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded">Save</button>
        </div>
      </form>
    </Modal>
  </div>
</template>
