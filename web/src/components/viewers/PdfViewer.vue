<template>
  <iframe
    v-if="blobUrl"
    :src="blobUrl"
    class="w-full h-full border-0"
  />
  <div v-else class="flex items-center justify-center h-full text-[var(--text-secondary)]">
    Loading PDF...
  </div>
</template>

<script setup lang="ts">
defineProps<{
  blobUrl: string | null
}>()
</script>
