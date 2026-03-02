<script setup lang="ts">
import { useRoute } from 'vue-router'

const props = defineProps<{
  to: string
  label: string
}>()

const route = useRoute()
</script>

<template>
  <RouterLink
    :to="props.to"
    class="block px-4 py-2 rounded text-sm font-medium transition-colors"
    :class="route.path === props.to || route.path.startsWith(props.to + '/')
      ? 'bg-scout-accent text-white'
      : 'text-gray-400 hover:text-white hover:bg-scout-700'"
  >
    {{ props.label }}
  </RouterLink>
</template>
