<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { StatsSnapshot } from '@/api/types'

const props = defineProps<{
  history: StatsSnapshot[]
}>()

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
    legend: { show: true },
    scales: {
      x: { time: true },
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
  const len = props.history.length
  const ts = new Float64Array(len)
  const rx = new Float64Array(len)
  const tx = new Float64Array(len)

  for (let i = 0; i < len; i++) {
    ts[i] = props.history[i].timestamp
    rx[i] = props.history[i].wan_rx_bps
    tx[i] = props.history[i].wan_tx_bps
  }

  return [ts, rx, tx]
}

watch(
  () => props.history.length,
  () => {
    if (chart) {
      chart.setData(buildData())
    }
  },
)

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
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
  <div ref="container" class="w-full" />
</template>
