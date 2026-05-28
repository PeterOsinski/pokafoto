<template>
  <video
    controls
    :poster="poster"
    class="max-h-full max-w-full object-contain"
    preload="metadata"
  >
    <source :src="src" type="video/mp4" />
  </video>
</template>

<script setup lang="ts">
defineProps<{
  src: string
  poster?: string
}>()
</script>
