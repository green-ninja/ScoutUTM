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
  <div class="min-h-screen flex items-center justify-center bg-scout-900">
    <div class="w-full max-w-sm bg-scout-800 rounded-lg shadow-xl p-8">
      <h1 class="text-2xl font-bold text-center text-white mb-8">ScoutUTM</h1>
      <form @submit.prevent="submit">
        <label class="block text-sm text-gray-400 mb-2" for="password">Password</label>
        <input
          id="password"
          v-model="password"
          type="password"
          autocomplete="current-password"
          class="w-full px-4 py-2 bg-scout-700 border border-scout-600 rounded text-white placeholder-gray-500 focus:outline-none focus:border-scout-accent"
          placeholder="Enter admin password"
          :disabled="loading"
        />
        <p v-if="error" class="mt-2 text-sm text-red-400">{{ error }}</p>
        <button
          type="submit"
          class="w-full mt-6 py-2 bg-scout-accent hover:bg-blue-600 text-white font-medium rounded transition-colors disabled:opacity-50"
          :disabled="loading || !password"
        >
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
    </div>
  </div>
</template>
