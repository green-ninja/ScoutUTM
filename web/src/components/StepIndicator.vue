<script setup lang="ts">
defineProps<{
  steps: number
  current: number
}>()
</script>

<template>
  <div class="flex items-center justify-center gap-2">
    <template v-for="i in steps" :key="i">
      <div
        class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors"
        :class="
          i < current
            ? 'bg-green-600 text-white'
            : i === current
              ? 'bg-scout-accent text-white'
              : 'bg-scout-700 text-gray-400'
        "
      >
        <span v-if="i < current">&#10003;</span>
        <span v-else>{{ i }}</span>
      </div>
      <div
        v-if="i < steps"
        class="w-8 h-0.5 transition-colors"
        :class="i < current ? 'bg-green-600' : 'bg-scout-700'"
      />
    </template>
  </div>
</template>
