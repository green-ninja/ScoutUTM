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
    class="fixed top-4 right-4 z-50 px-4 py-3 rounded-lg shadow-lg text-sm font-medium"
    :class="props.type === 'error'
      ? 'bg-red-900 text-red-200'
      : 'bg-green-900 text-green-200'"
  >
    {{ message }}
  </div>
</template>
