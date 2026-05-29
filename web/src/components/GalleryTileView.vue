<template>
  <div class="grid gap-2" :style="gridStyle">
    <div v-for="(file, i) in files" :key="file.id">
      <ThumbnailCard
        :file="file"
        :selected="selectedIds.has(file.id)"
        :selectable="selectionEnabled"
        :anySelected="selectedIds.size > 0"
        @select="$emit('select', file.id)"
        @deselect="$emit('deselect', file.id)"
        @open="$emit('open', i)"
      />
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
  folder_id?: string | null
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
  thumbSizePx: number
  selectedIds: Set<string>
  selectionEnabled: boolean
}>()

defineEmits<{
  select: [id: string]
  deselect: [id: string]
  open: [index: number]
}>()

const gridStyle = computed(() => ({
  gridTemplateColumns: `repeat(auto-fill, minmax(${props.thumbSizePx}px, 1fr))`,
}))
</script>
