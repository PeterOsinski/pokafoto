<template>
  <div class="space-y-8">
    <div v-for="group in groups" :key="group.label">
      <h3 class="sticky top-0 z-10 text-sm font-semibold text-[var(--text-secondary)] py-2 mb-3 border-b border-[var(--border-color)]" style="background: var(--bg-color)">
        {{ group.label }}
        <span class="font-normal text-xs">· {{ group.files.length }} {{ group.files.length === 1 ? 'photo' : 'photos' }}</span>
      </h3>
      <div class="grid gap-2" :style="gridStyle">
        <div v-for="item in group.files" :key="item.file.id">
          <ThumbnailCard
            :file="item.file"
            :selected="selectedIds.has(item.file.id)"
            :selectable="selectionEnabled"
            :anySelected="selectedIds.size > 0"
            @select="$emit('select', item.file.id)"
            @deselect="$emit('deselect', item.file.id)"
            @open="$emit('open', item.index)"
          />
        </div>
      </div>
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

interface GroupedItem {
  file: FileItem
  index: number
}

interface DayGroup {
  label: string
  dateKey: string
  files: GroupedItem[]
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

const groups = computed<DayGroup[]>(() => {
  const map = new Map<string, GroupedItem[]>()
  const unknown: GroupedItem[] = []

  props.files.forEach((file, index) => {
    if (!file.takenAt) {
      unknown.push({ file, index })
      return
    }
    const key = new Date(file.takenAt).toISOString().slice(0, 10)
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push({ file, index })
  })

  const sorted = Array.from(map.entries())
    .sort(([a], [b]) => b.localeCompare(a))
    .map(([dateKey, entries]) => {
      const firstDate = new Date(entries[0].file.takenAt!)
      const label = firstDate.toLocaleDateString(undefined, {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      })
      return { label, dateKey, files: entries }
    })

  if (unknown.length > 0) {
    sorted.push({ label: 'Unknown date', dateKey: '__unknown__', files: unknown })
  }

  return sorted
})
</script>
