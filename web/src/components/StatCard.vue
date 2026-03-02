<script setup lang="ts">
defineProps<{
  label: string
  value: string
  colorClass?: string
}>()
</script>

<template>
  <div class="bg-scout-800 rounded-lg p-4">
    <p class="text-sm text-gray-400 mb-1">{{ label }}</p>
    <p class="text-2xl font-bold" :class="colorClass || 'text-white'">{{ value }}</p>
  </div>
</template>
