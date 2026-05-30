<template>
  <RecycleScroller
    class="scroller"
    :items="files"
    :item-size="44"
    key-field="id"
    v-slot="{ item: file, index: i }"
    :buffer="300"
  >
    <div
      class="flex items-center border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors"
      :class="selectedIds.has(file.id) ? 'bg-[var(--accent)]/10' : ''"
      style="height: 44px"
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
      <div class="py-2 px-3 font-medium flex-1 min-w-0 cursor-pointer" @click="$emit('open', i)">
        <div class="flex items-center gap-2">
          <div v-if="file.mediaType === 'photo'" class="text-lg shrink-0">&#128247;</div>
          <div v-else-if="file.mediaType === 'video'" class="text-lg shrink-0">&#127916;</div>
          <div v-else class="text-lg shrink-0">&#128196;</div>
          <span class="truncate">{{ file.originalName }}</span>
        </div>
      </div>
      <div class="py-2 px-3 hidden sm:block shrink-0 cursor-pointer" @click="$emit('open', i)">
        <span class="px-2 py-0.5 rounded text-xs" :class="typeBadgeClass(file.mediaType)">
          {{ file.mediaType }}
        </span>
      </div>
      <div class="py-2 px-3 text-[var(--text-secondary)] hidden md:block whitespace-nowrap shrink-0 cursor-pointer" @click="$emit('open', i)">
        {{ formatSize(file.sizeBytes) }}
      </div>
      <div class="py-2 px-3 text-[var(--text-secondary)] hidden lg:block whitespace-nowrap shrink-0 cursor-pointer" @click="$emit('open', i)">
        {{ formatDate(file.takenAt) }}
      </div>
      <div class="py-2 px-3 shrink-0">
        <div class="flex items-center gap-1">
          <button
            @click.prevent.stop="$emit('download', file.id)"
            class="p-1 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
            title="Download"
          >
            &#11015;
          </button>
          <button
            @click.prevent.stop="$emit('delete', file.id)"
            class="p-1 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-red-400"
            title="Delete"
          >
            &#128465;
          </button>
        </div>
      </div>
    </div>
  </RecycleScroller>
</template>

<script setup lang="ts">
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
  folder_id?: string | null
  thumbnails?: any
}

defineProps<{
  files: FileItem[]
  selectedIds: Set<string>
  selectionEnabled: boolean
}>()

defineEmits<{
  select: [id: string]
  deselect: [id: string]
  open: [index: number]
  download: [id: string]
  delete: [id: string]
}>()

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
</script>

<style scoped>
.scroller {
  height: calc(100vh - 200px);
  min-height: 400px;
}
</style>
