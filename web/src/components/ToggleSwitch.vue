<script setup lang="ts">
defineProps<{
  modelValue: boolean
  label?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()
</script>

<template>
  <label class="flex items-center gap-3 cursor-pointer">
    <button
      type="button"
      role="switch"
      :aria-checked="modelValue"
      :aria-label="label"
      class="relative inline-flex h-6 w-11 shrink-0 rounded-full transition-colors"
      :class="modelValue ? 'bg-scout-accent' : 'bg-scout-600'"
      @click="emit('update:modelValue', !modelValue)"
    >
      <span
        class="inline-block h-4 w-4 rounded-full bg-white transition-transform mt-1"
        :class="modelValue ? 'translate-x-6 ml-0' : 'translate-x-1'"
      />
    </button>
    <span v-if="label" class="text-sm text-gray-300">{{ label }}</span>
  </label>
</template>
