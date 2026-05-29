<template>
  <pre class="whitespace-pre-wrap font-mono text-sm p-4 text-[var(--text-primary)] overflow-auto h-full">{{ content }}</pre>
</template>

<script setup lang="ts">
defineProps<{
  content: string
}>()
</script>
