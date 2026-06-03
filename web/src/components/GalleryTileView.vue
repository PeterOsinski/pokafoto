<template>
  <div ref="containerRef" style="width: 100%; min-height: 400px">
    <RecycleScroller
      v-if="containerWidth > 0"
      class="scroller"
      :items="rows"
      :item-size="itemSize"
      key-field="key"
      v-slot="{ item: rowItems, index: rowIndex }"
      :buffer="300"
    >
      <div class="grid gap-2" :style="{ gridTemplateColumns: `repeat(${columns}, 1fr)` }">
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
          @contextmenu="($event) => $emit('contextmenu', $event, file.id, file.originalName || file.filename)"
        />
      </div>
    </RecycleScroller>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
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
  contextmenu: [e: MouseEvent, fileId: string, fileName: string]
}>()

const containerRef = ref<HTMLElement | null>(null)
const containerWidth = ref(0)
let resizeObserver: ResizeObserver | null = null

const gap = 8

const columns = computed(() => {
  if (containerWidth.value <= 0) return 1
  return Math.max(1, Math.floor(containerWidth.value / (props.thumbSizePx + gap)))
})

const itemSize = computed(() => {
  if (columns.value <= 0) return props.thumbSizePx + gap
  return Math.round(containerWidth.value / columns.value) + gap
})

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

onMounted(() => {
  if (containerRef.value) {
    containerWidth.value = containerRef.value.clientWidth
    resizeObserver = new ResizeObserver((entries) => {
      if (entries[0]) {
        containerWidth.value = entries[0].contentRect.width
      }
    })
    resizeObserver.observe(containerRef.value)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
})
</script>

<style scoped>
.scroller {
  height: calc(100vh - 200px);
}
</style>
