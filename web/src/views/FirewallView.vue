<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api/firewall'
import type { FirewallConfig, FirewallRule, PortForward } from '@/api/types'
import { ApiError } from '@/api/client'
import Modal from '@/components/Modal.vue'
import Toast from '@/components/Toast.vue'

const config = ref<FirewallConfig | null>(null)
const activeTab = ref<'rules' | 'port-forwards'>('rules')
const showAddRule = ref(false)
const showAddPF = ref(false)
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)

const newRule = ref<FirewallRule>({ name: '', src: 'lan', dest: 'wan', proto: 'tcp', port: 0, action: 'accept' })
const newPF = ref<PortForward>({ name: '', proto: 'tcp', src_port: 0, dest_ip: '', dest_port: 0 })

onMounted(load)

async function load() {
  try {
    config.value = await api.getFirewall()
  } catch (e) {
    error.value = 'Failed to load firewall config'
  }
}

async function addRule() {
  try {
    await api.addRule(newRule.value)
    showAddRule.value = false
    newRule.value = { name: '', src: 'lan', dest: 'wan', proto: 'tcp', port: 0, action: 'accept' }
    await load()
    toast.value = { message: 'Rule added', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

async function deleteRule(index: number) {
  try {
    await api.deleteRule(index)
    await load()
    toast.value = { message: 'Rule deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

async function addPortForward() {
  try {
    await api.addPortForward(newPF.value)
    showAddPF.value = false
    newPF.value = { name: '', proto: 'tcp', src_port: 0, dest_ip: '', dest_port: 0 }
    await load()
    toast.value = { message: 'Port forward added', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}

async function deletePortForward(index: number) {
  try {
    await api.deletePortForward(index)
    await load()
    toast.value = { message: 'Port forward deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) {
      toast.value = { message: e.message, type: 'error' }
    }
  }
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-white mb-6">Firewall</h2>

    <Toast v-if="toast" :message="toast.message" :type="toast.type" @dismiss="toast = null" />

    <p v-if="!config && !error" class="text-gray-500">Loading...</p>
    <p v-else-if="error" class="text-red-400">{{ error }}</p>

    <!-- Tabs -->
    <template v-if="config">
    <div class="flex gap-4 mb-6 border-b border-scout-700">
      <button
        class="pb-2 text-sm font-medium transition-colors"
        :class="activeTab === 'rules' ? 'text-white border-b-2 border-scout-accent' : 'text-gray-400 hover:text-white'"
        @click="activeTab = 'rules'"
      >
        Rules
      </button>
      <button
        class="pb-2 text-sm font-medium transition-colors"
        :class="activeTab === 'port-forwards' ? 'text-white border-b-2 border-scout-accent' : 'text-gray-400 hover:text-white'"
        @click="activeTab = 'port-forwards'"
      >
        Port Forwards
      </button>
    </div>

    <!-- Rules tab -->
    <div v-if="activeTab === 'rules'">
      <div class="flex justify-end mb-4">
        <button
          class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded transition-colors"
          @click="showAddRule = true"
        >
          Add Rule
        </button>
      </div>

      <div class="glass rounded-xl overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-400 border-b border-scout-700">
              <th class="px-4 py-3">Name</th>
              <th class="px-4 py-3">Src</th>
              <th class="px-4 py-3">Dest</th>
              <th class="px-4 py-3">Proto</th>
              <th class="px-4 py-3">Port</th>
              <th class="px-4 py-3">Action</th>
              <th class="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(rule, i) in config?.rules"
              :key="i"
              class="border-b border-scout-700 text-gray-300"
            >
              <td class="px-4 py-3 text-white">{{ rule.name }}</td>
              <td class="px-4 py-3">{{ rule.src }}</td>
              <td class="px-4 py-3">{{ rule.dest }}</td>
              <td class="px-4 py-3">{{ rule.proto }}</td>
              <td class="px-4 py-3">{{ rule.port || 'any' }}</td>
              <td class="px-4 py-3">
                <span
                  class="px-2 py-0.5 rounded text-xs"
                  :class="rule.action === 'accept' ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'"
                >
                  {{ rule.action }}
                </span>
              </td>
              <td class="px-4 py-3">
                <button
                  class="text-red-400 hover:text-red-300 text-sm"
                  @click="deleteRule(i)"
                >
                  Delete
                </button>
              </td>
            </tr>
            <tr v-if="!config?.rules?.length">
              <td colspan="7" class="px-4 py-8 text-center text-gray-500">No firewall rules</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Port Forwards tab -->
    <div v-if="activeTab === 'port-forwards'">
      <div class="flex justify-end mb-4">
        <button
          class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded transition-colors"
          @click="showAddPF = true"
        >
          Add Port Forward
        </button>
      </div>

      <div class="glass rounded-xl overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-400 border-b border-scout-700">
              <th class="px-4 py-3">Name</th>
              <th class="px-4 py-3">Proto</th>
              <th class="px-4 py-3">Src Port</th>
              <th class="px-4 py-3">Dest IP</th>
              <th class="px-4 py-3">Dest Port</th>
              <th class="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(pf, i) in config?.port_forwards"
              :key="i"
              class="border-b border-scout-700 text-gray-300"
            >
              <td class="px-4 py-3 text-white">{{ pf.name }}</td>
              <td class="px-4 py-3">{{ pf.proto }}</td>
              <td class="px-4 py-3">{{ pf.src_port }}</td>
              <td class="px-4 py-3">{{ pf.dest_ip }}</td>
              <td class="px-4 py-3">{{ pf.dest_port }}</td>
              <td class="px-4 py-3">
                <button
                  class="text-red-400 hover:text-red-300 text-sm"
                  @click="deletePortForward(i)"
                >
                  Delete
                </button>
              </td>
            </tr>
            <tr v-if="!config?.port_forwards?.length">
              <td colspan="6" class="px-4 py-8 text-center text-gray-500">No port forwards</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    </template>

    <!-- Add Rule Modal -->
    <Modal v-if="showAddRule" title="Add Firewall Rule" @close="showAddRule = false">
      <form class="space-y-4" @submit.prevent="addRule">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Name</label>
          <input v-model="newRule.name" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Source Zone</label>
            <input v-model="newRule.src" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Dest Zone</label>
            <input v-model="newRule.dest" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
          </div>
        </div>
        <div class="grid grid-cols-3 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Protocol</label>
            <select v-model="newRule.proto" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option>tcp</option>
              <option>udp</option>
              <option>icmp</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Port</label>
            <input v-model.number="newRule.port" type="number" min="0" max="65535" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Action</label>
            <select v-model="newRule.action" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option>accept</option>
              <option>drop</option>
            </select>
          </div>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showAddRule = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded">Add</button>
        </div>
      </form>
    </Modal>

    <!-- Add Port Forward Modal -->
    <Modal v-if="showAddPF" title="Add Port Forward" @close="showAddPF = false">
      <form class="space-y-4" @submit.prevent="addPortForward">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Name</label>
          <input v-model="newPF.name" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Protocol</label>
            <select v-model="newPF.proto" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option>tcp</option>
              <option>udp</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Source Port</label>
            <input v-model.number="newPF.src_port" type="number" min="1" max="65535" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Dest IP</label>
            <input v-model="newPF.dest_ip" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="192.168.1.100" required />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Dest Port</label>
            <input v-model.number="newPF.dest_port" type="number" min="1" max="65535" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
          </div>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showAddPF = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded">Add</button>
        </div>
      </form>
    </Modal>
  </div>
</template>
