import { ref, onUnmounted } from 'vue'
import { useStatsStore } from '@/stores/stats'
import type { StatsSnapshot } from '@/api/types'

const RECONNECT_DELAY = 3000
const PING_INTERVAL = 30000

export function useWebSocket() {
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let pingTimer: ReturnType<typeof setInterval> | null = null

  function connect() {
    if (ws) return

    const token = sessionStorage.getItem('scout_token')
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}/api/v1/ws${token ? `?token=${token}` : ''}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      pingTimer = setInterval(() => {
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send('ping')
        }
      }, PING_INTERVAL)
    }

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'stats') {
          const stats = useStatsStore()
          stats.push(msg.data as StatsSnapshot)
        }
      } catch {
        // Ignore non-JSON messages.
      }
    }

    ws.onclose = () => {
      cleanup()
      // Only reconnect if we still have a valid session token.
      if (!sessionStorage.getItem('scout_token')) return
      reconnectTimer = setTimeout(connect, RECONNECT_DELAY)
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      ws.onclose = null // Prevent reconnect.
      ws.close()
      cleanup()
    }
  }

  function cleanup() {
    connected.value = false
    if (pingTimer) {
      clearInterval(pingTimer)
      pingTimer = null
    }
    ws = null
  }

  onUnmounted(disconnect)

  return { connected, connect, disconnect }
}
