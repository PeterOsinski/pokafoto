<template>
  <div class="mb-4">
    <label class="block text-sm text-gray-400 mb-1">File size: {{ formatSize(min) }} - {{ formatSize(max) }}</label>
    <div class="flex items-center gap-3">
      <input
        type="range"
        :min="absMin"
        :max="absMax"
        :value="min"
        @input="updateMin"
        class="flex-1 h-1 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
      />
      <span class="text-xs text-gray-500 w-16 text-right">{{ formatSize(min) }}</span>
      <span class="text-xs text-gray-500">-</span>
      <input
        type="range"
        :min="absMin"
        :max="absMax"
        :value="max"
        @input="updateMax"
        class="flex-1 h-1 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
      />
      <span class="text-xs text-gray-500 w-16 text-right">{{ formatSize(max) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  min: number
  max: number
  absMin: number
  absMax: number
}>()

const emit = defineEmits<{
  'update:min': [value: number]
  'update:max': [value: number]
}>()

function updateMin(e: Event) {
  const val = parseInt((e.target as HTMLInputElement).value)
  emit('update:min', Math.min(val, props.max))
}

function updateMax(e: Event) {
  const val = parseInt((e.target as HTMLInputElement).value)
  emit('update:max', Math.max(val, props.min))
}

function formatSize(bytes: number): string {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return bytes + ' B'
}
</script>
