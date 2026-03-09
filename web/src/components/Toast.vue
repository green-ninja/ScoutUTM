<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  message: string
  type?: 'success' | 'error'
}>()

const emit = defineEmits<{
  dismiss: []
}>()

let timer: ReturnType<typeof setTimeout>
onMounted(() => {
  timer = setTimeout(() => emit('dismiss'), 3000)
})
onUnmounted(() => {
  clearTimeout(timer)
})
</script>

<template>
  <div
    class="fixed top-4 right-4 z-50 px-4 py-3 rounded-xl shadow-lg text-sm font-medium border"
    :class="props.type === 'error'
      ? 'bg-red-950/80 text-red-200 border-red-500/20'
      : 'bg-emerald-950/80 text-emerald-200 border-emerald-500/20'"
  >
    {{ message }}
  </div>
</template>
