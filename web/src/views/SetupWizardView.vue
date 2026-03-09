<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { wizardStep } from '@/api/wizard'
import { getSystem } from '@/api/system'
import StepIndicator from '@/components/StepIndicator.vue'

const router = useRouter()
const step = ref(1)
const loading = ref(false)
const error = ref('')

// Step 2: Password
const password = ref('')
const confirmPass = ref('')

// Step 3: Interfaces (returned by API)
const interfaces = ref<{ name: string; zone: string; type: string }[]>([])

// Step 4: LAN config
const lanAddress = ref('192.168.1.1/24')

// Step 5: Security
const idsEnabled = ref(true)
const idsMode = ref('ids')
const blockThreats = ref(true)

// Step 6: VLANs (optional, simplified)
const skipVLAN = ref(true)

onMounted(async () => {
  try {
    const sys = await getSystem()
    if (sys.setup_done) {
      router.push('/dashboard')
    }
  } catch {
    // Not yet authenticated or server not ready — that's fine for wizard.
  }
})

function passwordStrength(pw: string): { label: string; color: string } {
  if (pw.length < 10) return { label: 'Too short', color: 'bg-red-500' }
  if (pw.length < 14) return { label: 'Fair', color: 'bg-yellow-500' }
  if (pw.length >= 14 && /[A-Z]/.test(pw) && /[0-9]/.test(pw)) return { label: 'Strong', color: 'bg-green-500' }
  return { label: 'Good', color: 'bg-violet-500' }
}

async function next() {
  error.value = ''
  loading.value = true

  try {
    switch (step.value) {
      case 1: {
        await wizardStep(1)
        break
      }
      case 2: {
        if (password.value.length < 10) {
          error.value = 'Password must be at least 10 characters'
          loading.value = false
          return
        }
        if (password.value !== confirmPass.value) {
          error.value = 'Passwords do not match'
          loading.value = false
          return
        }
        await wizardStep(2, { password: password.value })
        break
      }
      case 3: {
        const resp = await wizardStep(3)
        if (resp.interfaces) interfaces.value = resp.interfaces
        break
      }
      case 4: {
        await wizardStep(4, { lan_address: lanAddress.value })
        break
      }
      case 5: {
        await wizardStep(5, {
          ids_enabled: idsEnabled.value,
          ids_mode: idsMode.value,
          block_threats: blockThreats.value,
        })
        break
      }
      case 6: {
        await wizardStep(6, { vlans: [], zones: [] })
        break
      }
      case 7: {
        await wizardStep(7)
        router.push('/login')
        return
      }
    }
    step.value++
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'An error occurred'
  } finally {
    loading.value = false
  }
}

function back() {
  if (step.value > 1) step.value--
}
</script>

<template>
  <div class="min-h-screen bg-scout-900 flex flex-col">
    <!-- Header -->
    <div class="py-6 text-center">
      <h1 class="text-2xl font-bold text-white">ScoutUTM Setup</h1>
      <div class="mt-4">
        <StepIndicator :steps="7" :current="step" />
      </div>
    </div>

    <!-- Content -->
    <div class="flex-1 flex items-center justify-center px-4 pb-12">
      <div class="w-full max-w-lg">
        <!-- Error -->
        <p v-if="error" class="mb-4 px-4 py-2 bg-red-900/50 text-red-300 rounded text-sm">{{ error }}</p>

        <!-- Step 1: Welcome -->
        <div v-if="step === 1" class="text-center space-y-6">
          <div class="text-6xl">&#x1F6E1;</div>
          <h2 class="text-2xl font-bold text-white">Welcome to ScoutUTM</h2>
          <p class="text-gray-400">Let's set up your home network security appliance. This wizard will guide you through the initial configuration.</p>
        </div>

        <!-- Step 2: Password -->
        <div v-if="step === 2" class="space-y-4">
          <h2 class="text-xl font-bold text-white">Set Admin Password</h2>
          <p class="text-gray-400 text-sm">Choose a strong password to secure your ScoutUTM admin panel.</p>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Password (min 10 characters)</label>
            <input v-model="password" type="password" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
            <div v-if="password" class="mt-2 flex items-center gap-2">
              <div class="h-1.5 flex-1 rounded bg-scout-700 overflow-hidden">
                <div class="h-full rounded transition-all" :class="passwordStrength(password).color" :style="{ width: Math.min(100, password.length * 7) + '%' }" />
              </div>
              <span class="text-xs text-gray-400">{{ passwordStrength(password).label }}</span>
            </div>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Confirm Password</label>
            <input v-model="confirmPass" type="password" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm" />
          </div>
        </div>

        <!-- Step 3: Network Detection -->
        <div v-if="step === 3" class="space-y-4">
          <h2 class="text-xl font-bold text-white">Network Interfaces</h2>
          <p class="text-gray-400 text-sm">These network interfaces were detected on your device.</p>
          <div v-if="interfaces.length" class="glass rounded-xl overflow-hidden">
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-gray-400 border-b border-scout-700">
                  <th class="px-4 py-3">Interface</th>
                  <th class="px-4 py-3">Zone</th>
                  <th class="px-4 py-3">Type</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="i in interfaces" :key="i.name" class="border-b border-scout-700 text-gray-300">
                  <td class="px-4 py-3 text-white font-mono">{{ i.name }}</td>
                  <td class="px-4 py-3">
                    <span class="px-2 py-0.5 rounded text-xs" :class="i.zone === 'wan' ? 'bg-red-900 text-red-300' : 'bg-green-900 text-green-300'">
                      {{ i.zone }}
                    </span>
                  </td>
                  <td class="px-4 py-3">{{ i.type }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Step 4: LAN Config -->
        <div v-if="step === 4" class="space-y-4">
          <h2 class="text-xl font-bold text-white">LAN Configuration</h2>
          <p class="text-gray-400 text-sm">Configure your local network settings.</p>
          <div>
            <label class="block text-sm text-gray-400 mb-1">LAN IP Address (CIDR)</label>
            <input v-model="lanAddress" class="w-full px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm font-mono" placeholder="192.168.1.1/24" />
          </div>
        </div>

        <!-- Step 5: Security Defaults -->
        <div v-if="step === 5" class="space-y-4">
          <h2 class="text-xl font-bold text-white">Security Defaults</h2>
          <p class="text-gray-400 text-sm">Configure your initial security settings. You can change these later.</p>
          <div class="space-y-3">
            <label class="flex items-center gap-3 text-sm text-gray-300">
              <input v-model="idsEnabled" type="checkbox" class="rounded bg-scout-700 border-scout-600" />
              Enable Intrusion Detection
            </label>
            <div v-if="idsEnabled" class="ml-7">
              <select v-model="idsMode" class="px-3 py-2 bg-scout-700 border border-scout-600 rounded text-white text-sm">
                <option value="ids">IDS (detect only)</option>
                <option value="ips">IPS (detect + block)</option>
              </select>
            </div>
            <label class="flex items-center gap-3 text-sm text-gray-300">
              <input v-model="blockThreats" type="checkbox" class="rounded bg-scout-700 border-scout-600" />
              Block potentially dangerous and unwanted content
            </label>
          </div>
        </div>

        <!-- Step 6: VLANs -->
        <div v-if="step === 6" class="space-y-4">
          <h2 class="text-xl font-bold text-white">VLAN Setup (Optional)</h2>
          <p class="text-gray-400 text-sm">VLANs allow you to segment your network (e.g., IoT, Guest, Kids). You can set this up later.</p>
          <label class="flex items-center gap-3 text-sm text-gray-300">
            <input v-model="skipVLAN" type="checkbox" class="rounded bg-scout-700 border-scout-600" />
            Skip VLAN setup for now
          </label>
        </div>

        <!-- Step 7: Review + Apply -->
        <div v-if="step === 7" class="space-y-4">
          <h2 class="text-xl font-bold text-white">Review &amp; Apply</h2>
          <p class="text-gray-400 text-sm">Your ScoutUTM is ready to be configured with the settings above.</p>
          <div class="glass rounded-xl p-4 space-y-2 text-sm">
            <p class="text-gray-400">Admin password: <span class="text-green-400">Set</span></p>
            <p class="text-gray-400">LAN: <span class="text-white">{{ lanAddress }}</span></p>
            <p class="text-gray-400">IDS: <span class="text-white">{{ idsEnabled ? idsMode.toUpperCase() : 'Disabled' }}</span></p>
            <p class="text-gray-400">DNS Filtering: <span class="text-white">{{ blockThreats ? 'Enabled' : 'Disabled' }}</span></p>
          </div>
          <p class="text-gray-500 text-xs">Click "Apply" to finalize setup and start protecting your network.</p>
        </div>

        <!-- Navigation -->
        <div class="flex justify-between mt-8">
          <button
            v-if="step > 1"
            class="px-6 py-2 text-sm text-gray-400 hover:text-white transition-colors"
            :disabled="loading"
            @click="back"
          >
            Back
          </button>
          <div v-else />
          <button
            class="px-6 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded transition-colors disabled:opacity-50"
            :disabled="loading"
            @click="next"
          >
            {{ loading ? 'Please wait...' : step === 7 ? 'Apply' : step === 1 ? 'Get Started' : 'Next' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
