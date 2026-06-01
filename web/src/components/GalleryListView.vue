<template>
  <RecycleScroller
    class="scroller"
    :items="files"
    :item-size="52"
    key-field="id"
    v-slot="{ item: file, index: i }"
    :buffer="300"
  >
    <div
      class="flex items-center border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors"
      :class="selectedIds.has(file.id) ? 'bg-[var(--accent)]/10' : ''"
      style="height: 52px"
    >
      <div v-if="selectionEnabled" class="py-2 px-3 w-8 shrink-0">
        <button
          v-if="!selectedIds.has(file.id)"
          class="w-5 h-5 rounded border-2 border-white/30 flex items-center justify-center"
          @click.prevent.stop="$emit('select', file.id)"
        />
        <button
          v-else
          class="w-5 h-5 rounded border-2 flex items-center justify-center bg-[var(--accent)] border-[var(--accent)]"
          @click.prevent.stop="$emit('deselect', file.id)"
        >
          <span class="text-white text-xs font-bold">&#10003;</span>
        </button>
      </div>
      <div class="py-2 px-3 shrink-0 flex items-center" @click="$emit('open', i)">
        <img
          v-if="file.thumbnails?.sm?.url"
          :src="file.thumbnails.sm.url"
          :alt="file.originalName"
          class="rounded object-cover"
          :style="{ width: listThumbSize + 'px', height: listThumbSize + 'px' }"
          loading="lazy"
        />
        <div v-else-if="file.mediaType === 'file'" class="rounded flex flex-col items-center justify-center text-[var(--text-secondary)]" :style="{ width: listThumbSize + 'px', height: listThumbSize + 'px', background: 'var(--bg-elevated)' }">
          <span class="text-sm">&#128196;</span>
          <span class="text-[10px] font-mono opacity-60 leading-none">{{ fileExtension(file) }}</span>
        </div>
        <div v-else class="rounded flex items-center justify-center text-lg" :style="{ width: listThumbSize + 'px', height: listThumbSize + 'px', background: 'var(--bg-elevated)' }">
          {{ file.mediaType === 'video' ? '&#9654;' : '&#128196;' }}
        </div>
      </div>
      <div class="py-2 px-3 font-medium truncate flex-1 min-w-0 cursor-pointer text-left" @click="$emit('open', i)">{{ file.originalName }}</div>
      <div class="py-2 px-3 text-[var(--text-secondary)] hidden sm:block whitespace-nowrap shrink-0 cursor-pointer text-xs" @click="$emit('open', i)">{{ formatDate(file.takenAt) }}</div>
      <div class="py-2 px-3 text-[var(--text-secondary)] hidden md:block whitespace-nowrap shrink-0 cursor-pointer text-xs opacity-70" @click="$emit('open', i)">{{ formatDate(file.createdAt) }}</div>
      <div class="py-2 px-3 hidden md:block shrink-0 cursor-pointer" @click="$emit('open', i)">
        <span class="px-2 py-0.5 rounded text-xs" :class="typeBadgeClass(file.mediaType)">
          {{ file.mediaType }}
        </span>
      </div>
      <div class="py-2 px-3 text-[var(--text-secondary)] hidden lg:block whitespace-nowrap shrink-0 cursor-pointer" @click="$emit('open', i)">{{ formatSize(file.sizeBytes) }}</div>
    </div>
  </RecycleScroller>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'

interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  durationSec?: number
  takenAt?: string
  createdAt?: string
  folder_id?: string | null
  thumbnails?: {
    sm?: { url: string; width: number; height: number }
  }
}

const props = defineProps<{
  files: FileItem[]
  thumbSize?: string
  thumbSizePx: number
  selectedIds: Set<string>
  selectionEnabled: boolean
}>()

defineEmits<{
  select: [id: string]
  deselect: [id: string]
  open: [index: number]
}>()

const listThumbSize = computed(() => {
  const t = props.thumbSizePx
  if (t <= 40) return 20
  if (t >= 400) return 80
  return Math.round(20 + (t - 40) / 6)
})

function formatDate(takenAt?: string): string {
  if (!takenAt) return ''
  return new Date(takenAt).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function typeBadgeClass(mediaType: string): string {
  switch (mediaType) {
    case 'video': return 'bg-purple-900/40 text-purple-300'
    case 'photo': return 'bg-blue-900/40 text-blue-300'
    default: return 'bg-gray-700/40 text-gray-300'
  }
}

function fileExtension(file: FileItem): string {
  const name = file.originalName || file.filename || ''
  const ext = name.split('.').pop() || ''
  return ext ? `.${ext.toLowerCase()}` : ''
}
</script>

<style scoped>
.scroller {
  height: calc(100vh - 200px);
  min-height: 400px;
}
</style>
