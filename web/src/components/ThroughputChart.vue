<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { StatsSnapshot, ThroughputSample } from '@/api/types'
import { getStatsHistory } from '@/api/system'

const props = defineProps<{
  latest: StatsSnapshot | null
}>()

function formatBpsLabel(val: number): string {
  if (val >= 1_000_000) return `${(val / 1_000_000).toFixed(1)} MB/s`
  if (val >= 1_000) return `${(val / 1_000).toFixed(1)} KB/s`
  return `${val} B/s`
}

const ranges = [
  { key: '5m', label: '5 min' },
  { key: '30m', label: '30 min' },
  { key: '1h', label: '1 hour' },
  { key: '12h', label: '12 hours' },
  { key: '24h', label: '24 hours' },
] as const

type RangeKey = (typeof ranges)[number]['key']

const rangeSeconds: Record<RangeKey, number> = {
  '5m': 300,
  '30m': 1800,
  '1h': 3600,
  '12h': 43200,
  '24h': 86400,
}

const selectedRange = ref<RangeKey>('5m')
const chartData = ref<ThroughputSample[]>([])

const container = ref<HTMLDivElement>()
let chart: uPlot | null = null

function formatBps(self: uPlot, val: number): string {
  if (val >= 1_000_000) return `${(val / 1_000_000).toFixed(1)} MB/s`
  if (val >= 1_000) return `${(val / 1_000).toFixed(0)} KB/s`
  return `${val} B/s`
}

function createChart() {
  if (!container.value) return

  const opts: uPlot.Options = {
    width: container.value.clientWidth,
    height: 200,
    cursor: { show: false },
    legend: { show: false },
    scales: {
      x: {
        time: true,
        range: (_u: uPlot, _dataMin: number, dataMax: number) => {
          const window = rangeSeconds[selectedRange.value]
          const max = dataMax || Math.floor(Date.now() / 1000)
          return [max - window, max]
        },
      },
      y: { auto: true },
    },
    axes: [
      {
        stroke: '#4b5563',
        grid: { stroke: '#161620', width: 1 },
      },
      {
        stroke: '#4b5563',
        grid: { stroke: '#161620', width: 1 },
        values: (self: uPlot, vals: number[]) => vals.map((v) => formatBps(self, v)),
        size: 70,
      },
    ],
    series: [
      {},
      {
        label: 'RX',
        stroke: '#a78bfa',
        width: 2,
        fill: 'rgba(167,139,250,0.08)',
      },
      {
        label: 'TX',
        stroke: '#34d399',
        width: 2,
        fill: 'rgba(52,211,153,0.1)',
      },
    ],
  }

  const data = buildData()
  chart = new uPlot(opts, data, container.value)
}

function buildData(): uPlot.AlignedData {
  const d = chartData.value
  const len = d.length
  const ts = new Float64Array(len)
  const rx = new Float64Array(len)
  const tx = new Float64Array(len)

  for (let i = 0; i < len; i++) {
    ts[i] = d[i].timestamp
    rx[i] = d[i].rx_bps
    tx[i] = d[i].tx_bps
  }

  return [ts, rx, tx]
}

function updateChart() {
  if (chart) {
    chart.setData(buildData())
  }
}

async function fetchHistory() {
  try {
    chartData.value = await getStatsHistory(selectedRange.value)
    updateChart()
  } catch {
    /* ignore — chart stays with current data */
  }
}

// Append live WS updates and trim old points.
watch(
  () => props.latest,
  (snap) => {
    if (!snap) return
    const sample: ThroughputSample = {
      timestamp: snap.timestamp,
      rx_bps: snap.wan_rx_bps,
      tx_bps: snap.wan_tx_bps,
    }

    // Avoid duplicate timestamps.
    const d = chartData.value
    if (d.length > 0 && d[d.length - 1].timestamp >= sample.timestamp) return

    const cutoff = sample.timestamp - rangeSeconds[selectedRange.value]
    // Find first index within range.
    let start = 0
    while (start < d.length && d[start].timestamp < cutoff) start++

    chartData.value = [...d.slice(start), sample]
    updateChart()
  },
)

// On range change: immediately rescale, then fetch fresh data.
watch(selectedRange, () => {
  updateChart() // rescale x-axis immediately with current data
  fetchHistory()
})

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  fetchHistory()
  createChart()
  if (container.value) {
    resizeObserver = new ResizeObserver(() => {
      if (chart && container.value) {
        chart.setSize({ width: container.value.clientWidth, height: 200 })
      }
    })
    resizeObserver.observe(container.value)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  chart?.destroy()
  chart = null
})
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-3">
      <div class="flex gap-1">
        <button
          v-for="r in ranges"
          :key="r.key"
          class="px-2.5 py-1 text-xs rounded-md font-medium transition-colors"
          :class="
            selectedRange === r.key
              ? 'bg-violet-600 text-white'
              : 'bg-scout-700 text-gray-400 hover:text-white hover:bg-scout-600'
          "
          @click="selectedRange = r.key"
        >
          {{ r.label }}
        </button>
      </div>
      <div class="flex gap-4 text-xs">
        <span class="flex items-center gap-1.5">
          <span class="inline-block w-2.5 h-2.5 rounded-full bg-[#a78bfa]" />
          <span class="text-gray-400">Download</span>
          <span class="text-white font-medium">{{ latest ? formatBpsLabel(latest.wan_rx_bps) : '--' }}</span>
        </span>
        <span class="flex items-center gap-1.5">
          <span class="inline-block w-2.5 h-2.5 rounded-full bg-[#34d399]" />
          <span class="text-gray-400">Upload</span>
          <span class="text-white font-medium">{{ latest ? formatBpsLabel(latest.wan_tx_bps) : '--' }}</span>
        </span>
      </div>
    </div>
    <div ref="container" class="w-full" />
  </div>
</template>
