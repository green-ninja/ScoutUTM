<script setup lang="ts">
defineProps<{
  label: string
  value: string
  colorClass?: string
}>()
</script>

<template>
  <div class="glass rounded-xl p-4">
    <p class="text-sm text-gray-500 mb-1">{{ label }}</p>
    <p class="text-2xl font-semibold" :class="colorClass || 'text-white'">{{ value }}</p>
  </div>
</template>
