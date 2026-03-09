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
    class="block px-4 py-2.5 rounded-lg text-sm font-medium transition-all"
    :class="route.path === props.to || route.path.startsWith(props.to + '/')
      ? 'bg-violet-500/10 text-white'
      : 'text-gray-500 hover:text-gray-200 hover:bg-white/[0.03]'"
  >
    {{ props.label }}
  </RouterLink>
</template>
