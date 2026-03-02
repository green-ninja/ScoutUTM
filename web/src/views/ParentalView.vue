<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api/parental'
import * as devicesApi from '@/api/devices'
import type { ParentalProfile, ParentalConfig, Device, Schedule } from '@/api/types'
import { ApiError } from '@/api/client'
import Modal from '@/components/Modal.vue'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import Toast from '@/components/Toast.vue'

const config = ref<ParentalConfig | null>(null)
const devices = ref<Device[]>([])
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)

const showAddProfile = ref(false)
const editingProfile = ref<ParentalProfile | null>(null)

const categories = ['social_media', 'gaming', 'adult', 'streaming', 'gambling', 'drugs', 'weapons', 'malware']
const weekdays = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']

const form = ref(emptyForm())

function emptyForm() {
  return {
    name: '',
    block_categories: [] as string[],
    safe_search: false,
    youtube_restricted: false,
    schedule: { enabled: false, start_time: '21:00', end_time: '07:00', days: [] as string[] } as Schedule,
  }
}

onMounted(load)

async function load() {
  try {
    const [p, d] = await Promise.all([api.getParental(), devicesApi.getDevices()])
    config.value = p
    devices.value = d
  } catch {
    error.value = 'Failed to load parental config'
  }
}

function openAdd() {
  form.value = emptyForm()
  editingProfile.value = null
  showAddProfile.value = true
}

function openEdit(p: ParentalProfile) {
  form.value = {
    name: p.name,
    block_categories: [...p.block_categories],
    safe_search: p.safe_search,
    youtube_restricted: p.youtube_restricted,
    schedule: { ...p.schedule, days: [...p.schedule.days] },
  }
  editingProfile.value = p
  showAddProfile.value = true
}

async function saveProfile() {
  try {
    if (editingProfile.value) {
      await api.updateProfile(editingProfile.value.id, form.value)
      toast.value = { message: 'Profile updated', type: 'success' }
    } else {
      await api.createProfile(form.value as Omit<ParentalProfile, 'id'>)
      toast.value = { message: 'Profile created', type: 'success' }
    }
    showAddProfile.value = false
    await load()
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function deleteProfile(id: string) {
  try {
    await api.deleteProfile(id)
    toast.value = { message: 'Profile deleted', type: 'success' }
    await load()
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

function deviceProfileId(mac: string): string {
  const d = config.value?.devices.find((d) => d.mac === mac)
  return d?.profile_id ?? ''
}

async function setDeviceProfile(mac: string, profileId: string) {
  if (!config.value) return
  try {
    const assignments = devices.value.map((d) => ({
      mac: d.mac,
      profile_id: d.mac === mac ? profileId : deviceProfileId(d.mac),
    })).filter((a) => a.profile_id)
    config.value = await api.assignDevices(assignments)
    toast.value = { message: 'Device assignment updated', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

function toggleCategory(cat: string) {
  const idx = form.value.block_categories.indexOf(cat)
  if (idx >= 0) {
    form.value.block_categories.splice(idx, 1)
  } else {
    form.value.block_categories.push(cat)
  }
}

function toggleDay(day: string) {
  const idx = form.value.schedule.days.indexOf(day)
  if (idx >= 0) {
    form.value.schedule.days.splice(idx, 1)
  } else {
    form.value.schedule.days.push(day)
  }
}

function categoryLabel(cat: string): string {
  return cat.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">Parental Controls</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <p v-if="!config && !error" class="text-gray-500">Loading...</p>
    <p v-else-if="error" class="text-red-400">{{ error }}</p>

    <template v-if="config">
      <!-- Profiles -->
      <div class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-white">Profiles</h3>
          <button class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded" @click="openAdd">
            Add Profile
          </button>
        </div>

        <div v-if="!config.profiles.length" class="text-gray-500 text-sm">No profiles yet. Create one to get started.</div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div v-for="p in config.profiles" :key="p.id" class="bg-scout-800 rounded-lg p-4">
            <div class="flex items-center justify-between mb-2">
              <h4 class="text-white font-medium">{{ p.name }}</h4>
              <div class="flex gap-2">
                <button class="text-scout-accent text-sm hover:underline" @click="openEdit(p)">Edit</button>
                <button class="text-red-400 text-sm hover:underline" @click="deleteProfile(p.id)">Delete</button>
              </div>
            </div>
            <p class="text-sm text-gray-400">
              {{ p.block_categories.length }} categories blocked
              <span v-if="p.safe_search"> · Safe Search</span>
              <span v-if="p.youtube_restricted"> · YT Restricted</span>
            </p>
            <p v-if="p.schedule.enabled" class="text-xs text-gray-500 mt-1">
              Schedule: {{ p.schedule.start_time }}–{{ p.schedule.end_time }}
              ({{ p.schedule.days.length }} days)
            </p>
          </div>
        </div>
      </div>

      <!-- Device Assignments -->
      <div>
        <h3 class="text-lg font-semibold text-white mb-4">Device Assignments</h3>
        <p class="text-sm text-gray-500 mb-4">
          Assign parental profiles to devices. Note: devices may change MAC addresses if MAC randomization is enabled.
        </p>
        <div class="bg-scout-800 rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-400 border-b border-scout-700">
                <th class="px-4 py-3">Device</th>
                <th class="px-4 py-3">IP</th>
                <th class="px-4 py-3">MAC</th>
                <th class="px-4 py-3">Profile</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="d in devices" :key="d.mac" class="border-b border-scout-700 text-gray-300">
                <td class="px-4 py-3 text-white">{{ d.name || d.hostname || '—' }}</td>
                <td class="px-4 py-3">{{ d.ip }}</td>
                <td class="px-4 py-3 font-mono text-xs">{{ d.mac }}</td>
                <td class="px-4 py-3">
                  <select
                    :value="deviceProfileId(d.mac)"
                    class="px-2 py-1 bg-scout-700 border border-scout-600 rounded text-white text-sm"
                    @change="setDeviceProfile(d.mac, ($event.target as HTMLSelectElement).value)"
                  >
                    <option value="">None</option>
                    <option v-for="p in config.profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
                  </select>
                </td>
              </tr>
              <tr v-if="!devices.length">
                <td colspan="4" class="px-4 py-8 text-center text-gray-500">No devices found</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- Add/Edit Profile Modal -->
    <Modal v-if="showAddProfile" :title="editingProfile ? 'Edit Profile' : 'Add Profile'" @close="showAddProfile = false">
      <form class="space-y-4" @submit.prevent="saveProfile">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Profile Name</label>
          <input v-model="form.name" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-2">Block Categories</label>
          <div class="grid grid-cols-2 gap-2">
            <label v-for="cat in categories" :key="cat" class="flex items-center gap-2 text-sm text-gray-300">
              <input type="checkbox" :checked="form.block_categories.includes(cat)" class="rounded bg-scout-700 border-scout-600" @change="toggleCategory(cat)" />
              {{ categoryLabel(cat) }}
            </label>
          </div>
        </div>
        <ToggleSwitch v-model="form.safe_search" label="Safe Search" />
        <ToggleSwitch v-model="form.youtube_restricted" label="YouTube Restricted Mode" />
        <div>
          <ToggleSwitch v-model="form.schedule.enabled" label="Internet Schedule" />
          <template v-if="form.schedule.enabled">
            <div class="grid grid-cols-2 gap-4 mt-3">
              <div>
                <label class="block text-sm text-gray-400 mb-1">Block after</label>
                <input v-model="form.schedule.start_time" type="time" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">Unblock at</label>
                <input v-model="form.schedule.end_time" type="time" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
              </div>
            </div>
            <div class="flex flex-wrap gap-2 mt-3">
              <label v-for="day in weekdays" :key="day" class="flex items-center gap-1 text-sm text-gray-300">
                <input type="checkbox" :checked="form.schedule.days.includes(day)" class="rounded bg-scout-700 border-scout-600" @change="toggleDay(day)" />
                {{ day.slice(0, 3) }}
              </label>
            </div>
          </template>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showAddProfile = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded">
            {{ editingProfile ? 'Update' : 'Create' }}
          </button>
        </div>
      </form>
    </Modal>
  </div>
</template>
