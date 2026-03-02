import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(sessionStorage.getItem('scout_token'))
  const isAuthenticated = computed(() => token.value !== null)

  async function login(password: string) {
    const t = await authApi.login(password)
    sessionStorage.setItem('scout_token', t)
    token.value = t
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      clearSession()
    }
  }

  function clearSession() {
    sessionStorage.removeItem('scout_token')
    token.value = null
  }

  return { isAuthenticated, login, logout, clearSession }
})
