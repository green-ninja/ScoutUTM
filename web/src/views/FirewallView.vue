<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api/firewall'
import type { FirewallConfig, FirewallRule, PortForward, ForwardingRule } from '@/api/types'
import { ApiError } from '@/api/client'
import Modal from '@/components/Modal.vue'
import Toast from '@/components/Toast.vue'

const config = ref<FirewallConfig | null>(null)
const zones = ref<string[]>([])
const activeTab = ref<'rules' | 'port-forwards' | 'forwardings'>('rules')
const error = ref('')
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)

// Rule modal state
const showRuleModal = ref(false)
const editingRuleIndex = ref<number | null>(null)
const ruleForm = ref<FirewallRule>({ name: '', src: 'lan', dest: 'wan', proto: 'tcp', port: 0, action: 'accept' })

// Port forward modal state
const showPFModal = ref(false)
const editingPFIndex = ref<number | null>(null)
const pfForm = ref<PortForward>({ name: '', proto: 'tcp', src_port: 0, dest_ip: '', dest_port: 0 })

// Forwarding modal state
const showFwdModal = ref(false)
const editingFwdIndex = ref<number | null>(null)
const fwdForm = ref<ForwardingRule>({ src: 'lan', dest: 'wan' })

onMounted(load)

async function load() {
  try {
    const [fw, net] = await Promise.all([api.getFirewall(), api.getZones()])
    config.value = fw
    zones.value = net.zones.map(z => z.name)
  } catch {
    error.value = 'Failed to load firewall config'
  }
}

// --- Rules ---

function openAddRule() {
  ruleForm.value = { name: '', src: zones.value[1] || 'lan', dest: zones.value[0] || 'wan', proto: 'tcp', port: 0, action: 'accept' }
  editingRuleIndex.value = null
  showRuleModal.value = true
}

function openEditRule(index: number) {
  const r = config.value!.rules[index]
  ruleForm.value = { ...r }
  editingRuleIndex.value = index
  showRuleModal.value = true
}

async function saveRule() {
  try {
    if (editingRuleIndex.value !== null) {
      await api.updateRule(editingRuleIndex.value, ruleForm.value)
      toast.value = { message: 'Rule updated', type: 'success' }
    } else {
      await api.addRule(ruleForm.value)
      toast.value = { message: 'Rule added', type: 'success' }
    }
    showRuleModal.value = false
    await load()
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function deleteRule(index: number) {
  try {
    await api.deleteRule(index)
    await load()
    toast.value = { message: 'Rule deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function moveRule(from: number, to: number) {
  if (!config.value) return
  const rules = [...config.value.rules]
  const [item] = rules.splice(from, 1)
  rules.splice(to, 0, item)
  try {
    await api.updateFirewall({ ...config.value, rules })
    await load()
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

// --- Port Forwards ---

function openAddPF() {
  pfForm.value = { name: '', proto: 'tcp', src_port: 0, dest_ip: '', dest_port: 0 }
  editingPFIndex.value = null
  showPFModal.value = true
}

function openEditPF(index: number) {
  const pf = config.value!.port_forwards[index]
  pfForm.value = { ...pf }
  editingPFIndex.value = index
  showPFModal.value = true
}

async function savePF() {
  try {
    if (editingPFIndex.value !== null) {
      await api.updatePortForward(editingPFIndex.value, pfForm.value)
      toast.value = { message: 'Port forward updated', type: 'success' }
    } else {
      await api.addPortForward(pfForm.value)
      toast.value = { message: 'Port forward added', type: 'success' }
    }
    showPFModal.value = false
    await load()
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function deletePF(index: number) {
  try {
    await api.deletePortForward(index)
    await load()
    toast.value = { message: 'Port forward deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function movePF(from: number, to: number) {
  if (!config.value) return
  const pfs = [...config.value.port_forwards]
  const [item] = pfs.splice(from, 1)
  pfs.splice(to, 0, item)
  try {
    await api.updateFirewall({ ...config.value, port_forwards: pfs })
    await load()
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

// --- Forwardings ---

function openAddFwd() {
  fwdForm.value = { src: zones.value[1] || 'lan', dest: zones.value[0] || 'wan' }
  editingFwdIndex.value = null
  showFwdModal.value = true
}

function openEditFwd(index: number) {
  const f = config.value!.forwardings[index]
  fwdForm.value = { ...f }
  editingFwdIndex.value = index
  showFwdModal.value = true
}

async function saveFwd() {
  try {
    const fwds = [...(config.value!.forwardings || [])]
    if (editingFwdIndex.value !== null) {
      fwds[editingFwdIndex.value] = { ...fwdForm.value }
    } else {
      fwds.push({ ...fwdForm.value })
    }
    await api.updateFirewall({ ...config.value!, forwardings: fwds })
    showFwdModal.value = false
    await load()
    toast.value = { message: editingFwdIndex.value !== null ? 'Forwarding updated' : 'Forwarding added', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
  }
}

async function deleteFwd(index: number) {
  try {
    const fwds = [...(config.value!.forwardings || [])]
    fwds.splice(index, 1)
    await api.updateFirewall({ ...config.value!, forwardings: fwds })
    await load()
    toast.value = { message: 'Forwarding deleted', type: 'success' }
  } catch (e) {
    if (e instanceof ApiError) toast.value = { message: e.message, type: 'error' }
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
      <button
        class="pb-2 text-sm font-medium transition-colors"
        :class="activeTab === 'forwardings' ? 'text-white border-b-2 border-scout-accent' : 'text-gray-400 hover:text-white'"
        @click="activeTab = 'forwardings'"
      >
        Zone Forwardings
      </button>
    </div>

    <!-- Rules tab -->
    <div v-if="activeTab === 'rules'">
      <div class="flex justify-end mb-4">
        <button
          class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded transition-colors"
          @click="openAddRule"
        >
          Add Rule
        </button>
      </div>

      <div class="glass rounded-xl overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-400 border-b border-scout-700">
              <th class="w-16 px-3 py-3">#</th>
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
              <td class="px-3 py-3">
                <div class="flex flex-col gap-0.5">
                  <button
                    class="text-gray-500 hover:text-white text-xs leading-none disabled:opacity-25 disabled:cursor-default disabled:hover:text-gray-500"
                    :disabled="i === 0"
                    @click="moveRule(i, i - 1)"
                    title="Move up"
                  >&#9650;</button>
                  <button
                    class="text-gray-500 hover:text-white text-xs leading-none disabled:opacity-25 disabled:cursor-default disabled:hover:text-gray-500"
                    :disabled="i === (config?.rules?.length ?? 0) - 1"
                    @click="moveRule(i, i + 1)"
                    title="Move down"
                  >&#9660;</button>
                </div>
              </td>
              <td class="px-4 py-3 text-white">{{ rule.name }}</td>
              <td class="px-4 py-3">{{ rule.src }}</td>
              <td class="px-4 py-3">{{ rule.dest }}</td>
              <td class="px-4 py-3">{{ rule.proto || 'any' }}</td>
              <td class="px-4 py-3">{{ rule.port || 'any' }}</td>
              <td class="px-4 py-3">
                <span
                  class="px-2 py-0.5 rounded text-xs"
                  :class="rule.action === 'accept' ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'"
                >
                  {{ rule.action }}
                </span>
              </td>
              <td class="px-4 py-3 text-right space-x-3">
                <button class="text-scout-accent hover:text-violet-300 text-sm" @click="openEditRule(i)">Edit</button>
                <button class="text-red-400 hover:text-red-300 text-sm" @click="deleteRule(i)">Delete</button>
              </td>
            </tr>
            <tr v-if="!config?.rules?.length">
              <td colspan="8" class="px-4 py-8 text-center text-gray-500">No firewall rules</td>
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
          @click="openAddPF"
        >
          Add Port Forward
        </button>
      </div>

      <div class="glass rounded-xl overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-400 border-b border-scout-700">
              <th class="w-16 px-3 py-3">#</th>
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
              <td class="px-3 py-3">
                <div class="flex flex-col gap-0.5">
                  <button
                    class="text-gray-500 hover:text-white text-xs leading-none disabled:opacity-25 disabled:cursor-default disabled:hover:text-gray-500"
                    :disabled="i === 0"
                    @click="movePF(i, i - 1)"
                    title="Move up"
                  >&#9650;</button>
                  <button
                    class="text-gray-500 hover:text-white text-xs leading-none disabled:opacity-25 disabled:cursor-default disabled:hover:text-gray-500"
                    :disabled="i === (config?.port_forwards?.length ?? 0) - 1"
                    @click="movePF(i, i + 1)"
                    title="Move down"
                  >&#9660;</button>
                </div>
              </td>
              <td class="px-4 py-3 text-white">{{ pf.name }}</td>
              <td class="px-4 py-3">{{ pf.proto }}</td>
              <td class="px-4 py-3">{{ pf.src_port }}</td>
              <td class="px-4 py-3">{{ pf.dest_ip }}</td>
              <td class="px-4 py-3">{{ pf.dest_port }}</td>
              <td class="px-4 py-3 text-right space-x-3">
                <button class="text-scout-accent hover:text-violet-300 text-sm" @click="openEditPF(i)">Edit</button>
                <button class="text-red-400 hover:text-red-300 text-sm" @click="deletePF(i)">Delete</button>
              </td>
            </tr>
            <tr v-if="!config?.port_forwards?.length">
              <td colspan="7" class="px-4 py-8 text-center text-gray-500">No port forwards</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Forwardings tab -->
    <div v-if="activeTab === 'forwardings'">
      <div class="flex justify-end mb-4">
        <button
          class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded transition-colors"
          @click="openAddFwd"
        >
          Add Forwarding
        </button>
      </div>

      <p class="text-sm text-gray-400 mb-4">
        Zone forwardings allow traffic to flow between zones. Without a forwarding rule, inter-zone traffic is blocked.
      </p>

      <div class="glass rounded-xl overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-400 border-b border-scout-700">
              <th class="px-4 py-3">Source Zone</th>
              <th class="px-4 py-3">Destination Zone</th>
              <th class="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(fwd, i) in config?.forwardings"
              :key="i"
              class="border-b border-scout-700 text-gray-300"
            >
              <td class="px-4 py-3 text-white">{{ fwd.src }}</td>
              <td class="px-4 py-3 text-white">{{ fwd.dest }}</td>
              <td class="px-4 py-3 text-right space-x-3">
                <button class="text-scout-accent hover:text-violet-300 text-sm" @click="openEditFwd(i)">Edit</button>
                <button class="text-red-400 hover:text-red-300 text-sm" @click="deleteFwd(i)">Delete</button>
              </td>
            </tr>
            <tr v-if="!config?.forwardings?.length">
              <td colspan="3" class="px-4 py-8 text-center text-gray-500">No zone forwardings</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    </template>

    <!-- Rule Modal (Add / Edit) -->
    <Modal v-if="showRuleModal" :title="editingRuleIndex !== null ? 'Edit Rule' : 'Add Rule'" @close="showRuleModal = false">
      <form class="space-y-4" @submit.prevent="saveRule">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Name</label>
          <input v-model="ruleForm.name" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Source Zone</label>
            <select v-model="ruleForm.src" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option v-for="z in zones" :key="z" :value="z">{{ z }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Dest Zone</label>
            <select v-model="ruleForm.dest" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option v-for="z in zones" :key="z" :value="z">{{ z }}</option>
            </select>
          </div>
        </div>
        <div class="grid grid-cols-3 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Protocol</label>
            <select v-model="ruleForm.proto" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option value="">any</option>
              <option>tcp</option>
              <option>udp</option>
              <option>icmp</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Port</label>
            <input v-model.number="ruleForm.port" type="number" min="0" max="65535" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Action</label>
            <select v-model="ruleForm.action" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option>accept</option>
              <option>drop</option>
              <option>reject</option>
            </select>
          </div>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showRuleModal = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded">
            {{ editingRuleIndex !== null ? 'Update' : 'Add' }}
          </button>
        </div>
      </form>
    </Modal>

    <!-- Port Forward Modal (Add / Edit) -->
    <Modal v-if="showPFModal" :title="editingPFIndex !== null ? 'Edit Port Forward' : 'Add Port Forward'" @close="showPFModal = false">
      <form class="space-y-4" @submit.prevent="savePF">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Name</label>
          <input v-model="pfForm.name" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Protocol</label>
            <select v-model="pfForm.proto" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option>tcp</option>
              <option>udp</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Source Port</label>
            <input v-model.number="pfForm.src_port" type="number" min="1" max="65535" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Dest IP</label>
            <input v-model="pfForm.dest_ip" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" placeholder="192.168.1.100" required />
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Dest Port</label>
            <input v-model.number="pfForm.dest_port" type="number" min="1" max="65535" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" required />
          </div>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showPFModal = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded">
            {{ editingPFIndex !== null ? 'Update' : 'Add' }}
          </button>
        </div>
      </form>
    </Modal>

    <!-- Forwarding Modal (Add / Edit) -->
    <Modal v-if="showFwdModal" :title="editingFwdIndex !== null ? 'Edit Forwarding' : 'Add Forwarding'" @close="showFwdModal = false">
      <form class="space-y-4" @submit.prevent="saveFwd">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Source Zone</label>
            <select v-model="fwdForm.src" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option v-for="z in zones" :key="z" :value="z">{{ z }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Destination Zone</label>
            <select v-model="fwdForm.dest" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
              <option v-for="z in zones" :key="z" :value="z">{{ z }}</option>
            </select>
          </div>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="px-4 py-2 text-sm text-gray-400 hover:text-white" @click="showFwdModal = false">Cancel</button>
          <button type="submit" class="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded">
            {{ editingFwdIndex !== null ? 'Update' : 'Add' }}
          </button>
        </div>
      </form>
    </Modal>
  </div>
</template>
