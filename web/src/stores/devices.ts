import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Device } from '@/api/types'

export const useDevicesStore = defineStore('devices', () => {
  const devices = ref<Device[]>([])

  function push(list: Device[]) {
    devices.value = list
  }

  function clear() {
    devices.value = []
  }

  return { devices, push, clear }
})
