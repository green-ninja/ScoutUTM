<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api/system'
import type { SystemConfig, SystemAbout } from '@/api/types'
import { ApiError } from '@/api/client'
import Modal from '@/components/Modal.vue'
import Toast from '@/components/Toast.vue'

const config = ref<SystemConfig | null>(null)
const about = ref<SystemAbout | null>(null)
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)

const showPasswordModal = ref(false)
const showRebootModal = ref(false)
const showRegenCertModal = ref(false)

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')

onMounted(load)

async function load() {
  try {
    const [sys, info] = await Promise.all([api.getSystem(), api.getAbout()])
    config.value = sys
    about.value = info
  } catch {
    error.value = 'Failed to load system config'
  }
}

async function saveSettings() {
  if (!config.value) return
  try {
    config.value = await api.updateSystem({
      hostname: config.value.hostname,
      timezone: config.value.timezone,
      language: config.value.language,
      ntp_servers: config.value.ntp_servers,
    })
    toast.value = { message: 'Settings saved', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function changePassword() {
  if (newPassword.value !== confirmPassword.value) {
    toast.value = { message: 'Passwords do not match', type: 'error' }
    return
  }
  try {
    await api.changePassword(currentPassword.value, newPassword.value)
    showPasswordModal.value = false
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    toast.value = { message: 'Password changed', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function handleReboot() {
  try {
    await api.reboot()
    showRebootModal.value = false
    toast.value = { message: 'Rebooting...', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function handleBackup() {
  try {
    const blob = await api.downloadBackup()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'scout-config.json'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    toast.value = { message: 'Backup download failed', type: 'error' }
  }
}

async function handleRestore(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!confirm('This will replace your current configuration. Continue?')) {
    input.value = ''
    return
  }
  try {
    await api.uploadRestore(file)
    toast.value = { message: 'Config restored successfully', type: 'success' }
    await load()
  } catch (e) {
    if (e instanceof Error) toast.value = { message: e.message, type: 'error' }
  }
  input.value = ''
}

async function handleRegenCert() {
  try {
    await api.regenerateCert()
    showRegenCertModal.value = false
    toast.value = { message: 'Certificate regenerated. Restart may be required.', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">System</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <p v-if="!config && !error" class="text-gray-500">Loading...</p>
    <p v-else-if="error" class="text-red-400">{{ error }}</p>

    <template v-if="config">
      <!-- Settings -->
      <div class="bg-scout-800 rounded-lg p-6 mb-6">
        <h3 class="text-lg font-semibold text-white mb-4">Settings</h3>
        <div class="space-y-4 max-w-lg">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Hostname</label>
            <input v-model="config.hostname" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Timezone</label>
            <input v-model="config.timezone" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="UTC" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Language</label>
            <input v-model="config.language" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="en" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">NTP Servers (one per line)</label>
            <textarea
              :value="config.ntp_servers.join('\n')"
              @input="config!.ntp_servers = ($event.target as HTMLTextAreaElement).value.split('\n').filter(Boolean)"
              rows="3"
              class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm font-mono"
            />
          </div>
          <div class="flex justify-end">
            <button class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded" @click="saveSettings">
              Save Settings
            </button>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="bg-scout-800 rounded-lg p-6 mb-6">
        <h3 class="text-lg font-semibold text-white mb-4">Actions</h3>
        <div class="flex flex-wrap gap-3">
          <button class="px-4 py-2 bg-scout-700 hover:bg-scout-600 text-white text-sm rounded" @click="showPasswordModal = true">
            Change Password
          </button>
          <button class="px-4 py-2 bg-scout-700 hover:bg-scout-600 text-white text-sm rounded" @click="handleBackup">
            Download Backup
          </button>
          <label class="px-4 py-2 bg-scout-700 hover:bg-scout-600 text-white text-sm rounded cursor-pointer">
            Upload Restore
            <input type="file" accept=".json" class="hidden" @change="handleRestore" />
          </label>
          <button class="px-4 py-2 bg-scout-700 hover:bg-scout-600 text-white text-sm rounded" @click="showRegenCertModal = true">
            Regenerate TLS Cert
          </button>
          <button class="px-4 py-2 bg-red-900 hover:bg-red-800 text-white text-sm rounded" @click="showRebootModal = true">
            Reboot
          </button>
        </div>
      </div>

      <!-- About -->
      <div v-if="about" class="bg-scout-800 rounded-lg p-6">
        <h3 class="text-lg font-semibold text-white mb-4">About</h3>
        <div class="space-y-2 text-sm">
          <p class="text-gray-400">Version: <span class="text-white">{{ about.version }}</span></p>
          <p class="text-gray-400">Uptime: <span class="text-white">{{ formatUptime(about.uptime) }}</span></p>
          <p class="text-gray-400">Go Version: <span class="text-white">{{ about.go_version }}</span></p>
        </div>
      </div>
    </template>

    <!-- Change Password Modal -->
    <Modal v-if="showPasswordModal" title="Change Password" @close="showPasswordModal = false">
      <form class="space-y-4" @submit.prevent="changePassword">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Current Password</label>
          <input v-model="currentPassword" type="password" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">New Password (min 10 chars)</label>
          <input v-model="newPassword" type="password" minlength="10" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">Confirm New Password</label>
          <input v-model="confirmPassword" type="password" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showPasswordModal = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded">Change</button>
        </div>
      </form>
    </Modal>

    <!-- Reboot Confirmation -->
    <Modal v-if="showRebootModal" title="Confirm Reboot" @close="showRebootModal = false">
      <p class="text-gray-300 mb-4">Are you sure you want to reboot the device? This will temporarily disconnect all clients.</p>
      <div class="flex justify-end gap-3">
        <button class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showRebootModal = false">Cancel</button>
        <button class="px-4 py-2 bg-red-600 hover:bg-red-500 text-white text-sm rounded" @click="handleReboot">Reboot</button>
      </div>
    </Modal>

    <!-- Regenerate Cert Confirmation -->
    <Modal v-if="showRegenCertModal" title="Regenerate TLS Certificate" @close="showRegenCertModal = false">
      <p class="text-gray-300 mb-4">This will generate a new self-signed TLS certificate. You may need to accept the new certificate in your browser.</p>
      <div class="flex justify-end gap-3">
        <button class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showRegenCertModal = false">Cancel</button>
        <button class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded" @click="handleRegenCert">Regenerate</button>
      </div>
    </Modal>
  </div>
</template>
