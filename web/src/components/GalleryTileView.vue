<template>
  <RecycleScroller
    class="scroller"
    :items="rows"
    :item-size="props.thumbSizePx + 8"
    key-field="key"
    v-slot="{ item: rowItems, index: rowIndex }"
    :buffer="300"
  >
    <div class="flex gap-2" :style="{ gridTemplateColumns: `repeat(${columns}, 1fr)`, display: 'grid' }">
      <ThumbnailCard
        v-for="(file, colIndex) in rowItems"
        :key="file.id"
        :file="file"
        :selected="selectedIds.has(file.id)"
        :selectable="selectionEnabled"
        :anySelected="selectedIds.size > 0"
        :thumbSize="effectiveThumbSize"
        @select="$emit('select', file.id)"
        @deselect="$emit('deselect', file.id)"
        @open="$emit('open', rowIndex * columns + colIndex)"
      />
    </div>
  </RecycleScroller>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
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

interface RowItem extends Array<FileItem> {
  key: string
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

const columns = computed(() => Math.max(1, Math.floor(1200 / (props.thumbSizePx + 8))))

const effectiveThumbSize = computed<'sm' | 'md' | 'lg'>(() => {
  if (props.thumbSizePx <= 100) return 'sm'
  if (props.thumbSizePx <= 250) return 'lg'
  return 'md'
})

const rows = computed(() => {
  const cols = columns.value
  const result: RowItem[] = []
  for (let i = 0; i < props.files.length; i += cols) {
    const chunk = props.files.slice(i, i + cols) as RowItem
    chunk.key = `row-${i}`
    result.push(chunk)
  }
  return result
})
</script>

<style scoped>
.scroller {
  height: calc(100vh - 200px);
  min-height: 400px;
}
</style>
