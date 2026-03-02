import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { IDSAlert } from '@/api/types'

const MAX_ALERTS = 100

export const useAlertsStore = defineStore('alerts', () => {
  const alerts = ref<IDSAlert[]>([])

  function push(alert: IDSAlert) {
    alerts.value = [...alerts.value.slice(-(MAX_ALERTS - 1)), alert]
  }

  function setAll(list: IDSAlert[]) {
    alerts.value = list.slice(-MAX_ALERTS)
  }

  function clear() {
    alerts.value = []
  }

  return { alerts, push, setAll, clear }
})
