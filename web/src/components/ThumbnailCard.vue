<template>
  <div
    class="relative rounded-md overflow-hidden group"
    :class="selected ? 'ring-2 ring-[var(--accent)]' : ''"
    style="background: var(--bg-elevated); aspect-ratio: 1"
    >
    <button
      v-if="selectable && !selected"
      aria-label="Select"
      class="absolute top-1.5 left-1.5 z-10 w-5 h-5 rounded border-2 flex items-center justify-center transition-opacity bg-white/20 border-white/60"
      :class="anySelected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
      @click.prevent.stop="$emit('select')"
    />
    <button
      v-if="selectable && selected"
      aria-label="Deselect"
      class="absolute top-1.5 left-1.5 z-10 w-5 h-5 rounded border-2 flex items-center justify-center bg-[var(--accent)] border-[var(--accent)] opacity-100"
      @click.prevent.stop="$emit('deselect')"
    >
      <span class="text-white text-xs font-bold">&#10003;</span>
    </button>

    <div @click="$emit('open')" class="cursor-pointer w-full h-full">
      <img
        v-if="imgSrc && !loadError"
        :src="imgSrc"
        :alt="file.originalName"
        class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
        loading="lazy"
        @error="onImageError"
      />
      <div v-if="loadError" class="w-full h-full flex flex-col items-center justify-center gap-2 text-sm text-[var(--text-secondary)]">
        <span v-if="file.mediaType === 'video'">&#9654;</span>
        <span v-else-if="file.mediaType === 'file'" class="flex flex-col items-center gap-1">
          <span class="text-2xl">&#128196;</span>
          <span class="text-xs font-mono opacity-60">{{ fileExtension }}</span>
        </span>
        <span v-else>&#128196;</span>
        <button @click.prevent.stop="retryLoad" class="px-2 py-1 rounded text-xs bg-[var(--accent)] text-white">
          Retry
        </button>
      </div>
      <div v-else-if="!imgSrc" class="w-full h-full flex flex-col items-center justify-center gap-1 text-[var(--text-secondary)]">
        <template v-if="file.mediaType === 'video'">
          <span class="text-3xl">&#9654;</span>
        </template>
        <template v-else-if="file.mediaType === 'file'">
          <span class="text-2xl">&#128196;</span>
          <span class="text-xs font-mono opacity-60">{{ fileExtension }}</span>
        </template>
        <template v-else>
          <span class="text-3xl">&#128196;</span>
        </template>
      </div>
    </div>
    <div v-if="file.mediaType === 'video' && file.durationSec && !loadError" class="absolute bottom-1 right-1 px-1.5 py-0.5 rounded text-xs text-white bg-black/60">
      {{ formatDuration(file.durationSec) }}
    </div>
    <div v-if="file.takenAt && !loadError" class="absolute bottom-1 left-1 px-1.5 py-0.5 rounded text-xs text-white bg-black/60">
      {{ formatDate(file.takenAt) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface FileItem {
  id: string
  originalName: string
  filename: string
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
  file: FileItem
  thumbSize?: 'sm' | 'md' | 'lg'
  selected?: boolean
  selectable?: boolean
  anySelected?: boolean
}>()

defineEmits<{
  select: []
  deselect: []
  open: []
}>()

const loadError = ref(false)
const retryCounter = ref(0)

const fileExtension = computed(() => {
  const name = props.file.originalName || props.file.filename || ''
  const ext = name.split('.').pop() || ''
  return ext ? `.${ext.toLowerCase()}` : ''
})

const sizeKey = computed<'sm' | 'lg' | 'md'>(() => {
  if (props.thumbSize === 'lg') return 'md'
  if (!props.thumbSize || props.thumbSize === 'md' || props.thumbSize === 'sm') return 'lg'
  return 'sm'
})

const imgSrc = computed(() => {
  const t = props.file.thumbnails
  if (!t) return ''
  const primary = t[sizeKey.value]
  const fallback = t.md || t.sm || t.videoStill || t.preview
  const base = primary?.url || fallback?.url || ''
  if (!base) return ''
  return retryCounter.value > 0 ? `${base}#t=${retryCounter.value}` : base
})

function onImageError() {
  loadError.value = true
}

function retryLoad() {
  retryCounter.value++
  loadError.value = false
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

function formatDate(takenAt: string): string {
  const d = new Date(takenAt)
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}
</script>
