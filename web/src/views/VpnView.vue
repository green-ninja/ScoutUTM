<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api/vpn'
import type { WireGuardConfig } from '@/api/types'
import { ApiError } from '@/api/client'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import Modal from '@/components/Modal.vue'
import Toast from '@/components/Toast.vue'

const config = ref<WireGuardConfig | null>(null)
const loading = ref(false)
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)

const showAddPeer = ref(false)
const showQR = ref(false)
const showRegenConfirm = ref(false)

const newPeerName = ref('')
const newPeerSplitTunnel = ref(true)

const qrData = ref({ qr_code: '', client_config: '' })

onMounted(load)

async function load() {
  try {
    config.value = await api.getWireGuard()
  } catch {
    toast.value = { message: 'Failed to load VPN config', type: 'error' }
  }
}

async function saveConfig() {
  if (!config.value) return
  loading.value = true
  try {
    config.value = await api.updateWireGuard(config.value)
    toast.value = { message: 'VPN settings saved', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  } finally {
    loading.value = false
  }
}

async function addPeer() {
  try {
    const result = await api.addPeer({
      name: newPeerName.value,
      split_tunnel: newPeerSplitTunnel.value,
    })
    showAddPeer.value = false
    newPeerName.value = ''
    newPeerSplitTunnel.value = true

    qrData.value = {
      qr_code: result.qr_code,
      client_config: result.client_config,
    }
    showQR.value = true
    await load()
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

async function removePeer(name: string) {
  try {
    await api.deletePeer(name)
    await load()
    toast.value = { message: 'Peer deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

async function regenKeys() {
  showRegenConfirm.value = false
  try {
    await api.regenerateKeys()
    await load()
    toast.value = { message: 'Keys regenerated', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

async function copyConfig() {
  try {
    await navigator.clipboard.writeText(qrData.value.client_config)
    toast.value = { message: 'Config copied to clipboard', type: 'success' }
  } catch {
    toast.value = { message: 'Failed to copy to clipboard', type: 'error' }
  }
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">VPN</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <div v-if="config" class="space-y-6">
      <!-- WireGuard settings -->
      <div class="bg-scout-800 rounded-lg p-6">
        <h3 class="text-sm font-medium text-gray-400 mb-4">WireGuard Settings</h3>
        <div class="space-y-4">
          <ToggleSwitch v-model="config.enabled" label="Enable WireGuard" />

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">Listen Port</label>
              <input
                v-model.number="config.listen_port"
                type="number"
                class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm"
              />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">Endpoint Override</label>
              <input
                v-model="config.endpoint_override"
                class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm"
                placeholder="myhost.ddns.net"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm text-gray-400 mb-1">Public Key</label>
            <div class="flex gap-2">
              <input
                :value="config.public_key || '(not generated)'"
                readonly
                class="flex-1 px-3 py-2 bg-scout-900 border border-scout-600 rounded text-gray-400 text-sm font-mono"
              />
              <button
                class="px-3 py-2 text-sm text-red-400 hover:text-red-300 border border-red-800 rounded"
                @click="showRegenConfirm = true"
              >
                Regenerate
              </button>
            </div>
          </div>

          <div class="flex justify-end">
            <button
              class="px-6 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded transition-colors disabled:opacity-50"
              :disabled="loading"
              @click="saveConfig"
            >
              {{ loading ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Peers -->
      <div class="bg-scout-800 rounded-lg p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-medium text-gray-400">Peers</h3>
          <button
            class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded transition-colors"
            @click="showAddPeer = true"
          >
            Add Peer
          </button>
        </div>

        <div class="overflow-hidden rounded">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-400 border-b border-scout-700">
                <th class="px-4 py-3">Name</th>
                <th class="px-4 py-3">Tunnel IP</th>
                <th class="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="peer in config.peers"
                :key="peer.public_key"
                class="border-b border-scout-700 text-gray-300"
              >
                <td class="px-4 py-3 text-white">{{ peer.name }}</td>
                <td class="px-4 py-3">{{ peer.tunnel_ip }}</td>
                <td class="px-4 py-3 text-right">
                  <button
                    class="text-red-400 hover:text-red-300 text-sm"
                    @click="removePeer(peer.name)"
                  >
                    Delete
                  </button>
                </td>
              </tr>
              <tr v-if="!config.peers?.length">
                <td colspan="3" class="px-4 py-8 text-center text-gray-500">No peers configured</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <p v-else class="text-gray-500">Loading...</p>

    <!-- Add Peer Modal -->
    <Modal v-if="showAddPeer" title="Add Peer" @close="showAddPeer = false">
      <form class="space-y-4" @submit.prevent="addPeer">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Name</label>
          <input
            v-model="newPeerName"
            class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm"
            placeholder="My Phone"
            required
          />
        </div>
        <ToggleSwitch v-model="newPeerSplitTunnel" label="Split tunnel (LAN access only)" />
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showAddPeer = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded">Create</button>
        </div>
      </form>
    </Modal>

    <!-- QR Code Modal -->
    <Modal v-if="showQR" title="Peer Config" @close="showQR = false">
      <div class="space-y-4">
        <div v-if="qrData.qr_code" class="flex justify-center">
          <img :src="'data:image/png;base64,' + qrData.qr_code" alt="QR Code" class="w-48 h-48" />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">Config</label>
          <pre class="w-full p-3 bg-scout-900 border border-scout-600 rounded text-gray-300 text-xs font-mono overflow-auto max-h-48">{{ qrData.client_config }}</pre>
        </div>
        <div class="flex justify-end gap-3">
          <button class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="copyConfig">Copy Config</button>
          <button class="px-4 py-2 bg-scout-accent hover:bg-blue-600 text-white text-sm rounded" @click="showQR = false">Done</button>
        </div>
      </div>
    </Modal>

    <!-- Regenerate Keys Confirmation -->
    <Modal v-if="showRegenConfirm" title="Regenerate Keys" @close="showRegenConfirm = false">
      <p class="text-gray-300 mb-4">
        This will generate new WireGuard keys. All existing peers will need to be reconfigured.
      </p>
      <div class="flex justify-end gap-3">
        <button class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showRegenConfirm = false">Cancel</button>
        <button class="px-4 py-2 bg-red-600 hover:bg-red-700 text-white text-sm rounded" @click="regenKeys">Regenerate</button>
      </div>
    </Modal>
  </div>
</template>
