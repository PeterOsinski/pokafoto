<template>
  <div>
    <div class="flex items-center border-b border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-xs font-semibold uppercase tracking-wide select-none">
      <div v-if="selectionEnabled" class="py-2 px-3 w-8 shrink-0" />
      <div class="py-2 px-3 shrink-0" :style="{ width: listThumbSize + 'px' }" />
      <div class="py-2 px-3 flex-1 min-w-0">Name</div>
      <div class="py-2 px-3 hidden sm:block whitespace-nowrap shrink-0">Date Taken</div>
      <div class="py-2 px-3 hidden md:block whitespace-nowrap shrink-0">Uploaded</div>
      <div class="py-2 px-3 hidden md:block shrink-0">Type</div>
      <div class="py-2 px-3 hidden lg:block whitespace-nowrap shrink-0">Size</div>
    </div>
    <RecycleScroller
      class="scroller"
      :items="files"
      :item-size="rowHeight"
      key-field="id"
      v-slot="{ item: file, index: i }"
      :buffer="300"
    >
      <div
        class="flex items-center border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors"
        :class="selectedIds.has(file.id) ? 'bg-[var(--accent)]/10' : ''"
        :style="{ height: rowHeight + 'px' }"
        @contextmenu.prevent="$emit('contextmenu', $event, file.id, file.originalName || file.filename)"
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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'

import type { FileItem } from '../types/gallery'

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
  contextmenu: [e: MouseEvent, fileId: string, fileName: string]
}>()

const listThumbSize = computed(() => {
  const t = props.thumbSizePx
  if (t <= 40) return 20
  if (t >= 400) return 80
  return Math.round(20 + (t - 40) / 6)
})

const rowHeight = computed(() => Math.max(52, listThumbSize.value + 16))

function formatDate(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const dd = d.getDate().toString().padStart(2, '0')
  const mm = (d.getMonth() + 1).toString().padStart(2, '0')
  const yy = d.getFullYear().toString().slice(-2)
  const hh = d.getHours().toString().padStart(2, '0')
  const min = d.getMinutes().toString().padStart(2, '0')
  const ss = d.getSeconds().toString().padStart(2, '0')
  return `${dd}/${mm}/${yy} ${hh}:${min}:${ss}`
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
  height: auto;
}
</style>
