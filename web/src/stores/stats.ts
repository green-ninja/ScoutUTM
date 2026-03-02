import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { StatsSnapshot } from '@/api/types'

const MAX_HISTORY = 60 // 60 samples × 2s = 2min window

export const useStatsStore = defineStore('stats', () => {
  const latest = ref<StatsSnapshot | null>(null)
  const history = ref<StatsSnapshot[]>([])

  function push(snap: StatsSnapshot) {
    latest.value = snap
    // Replace array reference (not in-place mutate) so watchers on
    // history.length fire even when the ring buffer is full.
    history.value = [...history.value.slice(-(MAX_HISTORY - 1)), snap]
  }

  function clear() {
    latest.value = null
    history.value = []
  }

  return { latest, history, push, clear }
})
