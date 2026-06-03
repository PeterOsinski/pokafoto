<template>
  <DynamicScroller
    class="scroller"
    :items="groupItems"
    :min-item-size="80"
    key-field="key"
    v-slot="{ item, active }"
    :buffer="300"
  >
    <DynamicScrollerItem :item="item" :active="active">
      <div v-if="item.type === 'header'" class="sticky top-0 z-10 text-sm font-semibold text-[var(--text-secondary)] py-2 border-b border-[var(--border-color)]" style="background: var(--bg-color)">
        {{ item.label }}
        <span class="font-normal text-xs">· {{ item.fileCount }} {{ item.fileCount === 1 ? 'photo' : 'photos' }}</span>
      </div>
      <div v-else-if="item.type === 'row'" class="grid gap-2 mb-2" :style="{ gridTemplateColumns: `repeat(${columns}, 1fr)` }">
        <ThumbnailCard
          v-for="entry in item.files"
          :key="entry.file.id"
          :file="entry.file"
          :selected="selectedIds.has(entry.file.id)"
          :selectable="selectionEnabled"
          :anySelected="selectedIds.size > 0"
          :thumbSize="effectiveThumbSize"
          @select="$emit('select', entry.file.id)"
          @deselect="$emit('deselect', entry.file.id)"
          @open="$emit('open', entry.index)"
          @contextmenu="($event) => $emit('contextmenu', $event, entry.file.id, entry.file.originalName || entry.file.filename)"
        />
      </div>
    </DynamicScrollerItem>
  </DynamicScroller>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
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

interface GroupedEntry {
  file: FileItem
  index: number
}

interface GroupItem {
  type: 'header' | 'row'
  key: string
  label?: string
  fileCount?: number
  files?: GroupedEntry[]
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

const columns = computed(() => Math.max(1, Math.floor(1200 / (props.thumbSizePx + 8))))

const effectiveThumbSize = computed<'sm' | 'md' | 'lg'>(() => {
  if (props.thumbSizePx <= 100) return 'sm'
  if (props.thumbSizePx <= 250) return 'lg'
  return 'md'
})

const groupItems = computed<GroupItem[]>(() => {
  const map = new Map<string, GroupedEntry[]>()
  const unknown: GroupedEntry[] = []

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

  const items: GroupItem[] = []
  sorted.forEach(([dateKey, entries]) => {
    const firstDate = new Date(entries[0].file.takenAt!)
    const label = firstDate.toLocaleDateString(undefined, {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
    items.push({ type: 'header', key: `h-${dateKey}`, label, fileCount: entries.length })

    const cols = columns.value
    for (let i = 0; i < entries.length; i += cols) {
      items.push({ type: 'row', key: `${dateKey}-r${i}`, files: entries.slice(i, i + cols) })
    }
  })

  if (unknown.length > 0) {
    items.push({ type: 'header', key: 'h-unknown', label: 'Unknown date', fileCount: unknown.length })
    const cols = columns.value
    for (let i = 0; i < unknown.length; i += cols) {
      items.push({ type: 'row', key: `unknown-r${i}`, files: unknown.slice(i, i + cols) })
    }
  }

  return items
})
</script>

<style scoped>
.scroller {
  height: calc(100vh - 200px);
  min-height: 400px;
}
</style>
