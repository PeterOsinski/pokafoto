<template>
  <button
    class="rounded-md overflow-hidden hover:opacity-80 transition-opacity cursor-pointer"
    style="background: var(--bg-elevated); aspect-ratio: 1"
    @click="$emit('click')"
    @contextmenu.prevent="$emit('contextmenu', $event)"
  >
    <span class="flex flex-col items-center justify-center gap-1 h-full">
      <span class="text-3xl relative">
        &#128193;
        <span v-if="hasPassword" class="absolute -bottom-1 -right-1 text-xs" title="Password protected">&#x1F512;</span>
      </span>
      <span class="text-xs text-[var(--text-primary)] font-medium truncate w-full px-2 text-center">
        <span v-if="hasPassword" class="mr-1" title="Password protected">&#x1F512;</span>
        <span v-if="hasShares" class="mr-1" title="Shared">&#x1F517;</span>{{ name }}
      </span>
      <span class="text-[10px] text-[var(--text-secondary)]">
        {{ fileCount }} {{ fileCount === 1 ? 'file' : 'files' }}
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  name: string
  fileCount: number
  hasShares: boolean
  hasPassword: boolean
}>()

defineEmits<{
  click: []
  contextmenu: [e: MouseEvent]
}>()
</script>
