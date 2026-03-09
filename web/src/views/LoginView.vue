<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'

const router = useRouter()
const auth = useAuthStore()

const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(password.value)
    router.push('/dashboard')
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 429) {
        error.value = 'Too many attempts. Please wait and try again.'
      } else {
        error.value = 'Invalid password'
      }
    } else {
      error.value = 'Connection failed'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-scout-900 relative overflow-hidden">
    <div class="absolute inset-0 bg-[radial-gradient(ellipse_at_50%_30%,rgba(139,92,246,0.06),transparent_60%)]" />

    <div class="relative w-full max-w-sm glass rounded-xl p-8">
      <h1 class="text-2xl font-semibold text-center text-white mb-1">
        <span class="text-violet-400">Scout</span>UTM
      </h1>
      <p class="text-center text-gray-500 text-sm mb-8">Sign in to your dashboard</p>
      <form @submit.prevent="submit">
        <label class="block text-sm text-gray-400 mb-2" for="password">Password</label>
        <input
          id="password"
          v-model="password"
          type="password"
          autocomplete="current-password"
          class="w-full px-4 py-2.5 bg-scout-700/50 border border-violet-500/10 rounded-lg text-white placeholder-gray-600 focus:outline-none focus:border-violet-500/30 transition-colors"
          placeholder="Enter admin password"
          :disabled="loading"
        />
        <p v-if="error" class="mt-2 text-sm text-red-400">{{ error }}</p>
        <button
          type="submit"
          class="w-full mt-6 py-2.5 bg-violet-600 hover:bg-violet-500 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
          :disabled="loading || !password"
        >
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
    </div>
  </div>
</template>
