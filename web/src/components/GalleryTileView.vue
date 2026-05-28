<template>
  <div class="grid gap-2" :class="gridClass">
    <div v-for="(file, i) in files" :key="file.id" @click="$emit('open', i)">
      <ThumbnailCard :file="file" :thumbSize="thumbSize" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ThumbnailCard from './ThumbnailCard.vue'

interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  durationSec?: number
  takenAt?: string
  thumbnails?: {
    sm?: { url: string; width: number; height: number }
    lg?: { url: string; width: number; height: number }
    md?: { url: string; width: number; height: number }
    preview?: { url: string; width: number; height: number }
    videoStill?: { url: string; width: number; height: number }
  }
}

const props = defineProps<{
  files: FileItem[]
  thumbSize?: 'sm' | 'md' | 'lg'
}>()

const gridClass = computed(() => {
  if (props.thumbSize === 'sm') return 'grid-cols-5 sm:grid-cols-6 lg:grid-cols-8 xl:grid-cols-10 2xl:grid-cols-12'
  if (props.thumbSize === 'lg') return 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6'
  return 'grid-cols-3 sm:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8'
})
</script>
