<template>
  <div class="relative inline-block">
    <button @click="open = !open" class="px-2 py-1 text-lg hover:bg-gray-700 rounded transition-colors" title="Add reaction">
      <span v-if="!open">😀</span>
      <span v-else>✕</span>
    </button>
    <div v-if="open" class="absolute bottom-full left-0 mb-1 bg-gray-800 border border-gray-700 rounded-lg shadow-lg p-1 flex gap-1 z-50">
      <button
        v-for="e in emojis"
        :key="e"
        @click="select(e)"
        class="px-2 py-1 text-lg hover:bg-gray-700 rounded transition-colors"
        :title="e"
      >
        {{ e }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emojis = ['👍', '❤️', '😂', '😮', '😢', '🙏']
const emit = defineEmits<{
  select: [emoji: string]
}>()

const open = ref(false)

function select(emoji: string) {
  emit('select', emoji)
  open.value = false
}
</script>
