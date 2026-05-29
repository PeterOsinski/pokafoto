<template>
  <div
    v-if="count > 0"
    class="sticky top-0 z-20 flex items-center justify-between px-4 py-2 rounded-md mb-4"
    style="background: var(--accent); color: white"
  >
    <span class="text-sm font-medium">{{ count }} selected</span>
    <div class="flex items-center gap-2">
      <button @click="$emit('delete')" class="px-3 py-1 rounded text-xs font-medium bg-white/20 hover:bg-white/30 transition-colors">
        Delete
      </button>
      <button @click="$emit('move')" class="px-3 py-1 rounded text-xs font-medium bg-white/20 hover:bg-white/30 transition-colors">
        Move
      </button>
      <button @click="$emit('copy')" class="px-3 py-1 rounded text-xs font-medium bg-white/20 hover:bg-white/30 transition-colors">
        Copy
      </button>
      <button @click="$emit('deselectAll')" class="px-3 py-1 rounded text-xs font-medium underline hover:bg-white/10 transition-colors">
        Clear
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  count: number
}>()

defineEmits<{
  delete: []
  move: []
  copy: []
  deselectAll: []
}>()
</script>
