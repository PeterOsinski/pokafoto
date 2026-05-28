<template>
  <div
    class="relative rounded-md overflow-hidden cursor-pointer group"
    style="background: var(--bg-elevated); aspect-ratio: 1"
  >
    <img
      v-if="file.thumbnails?.md && !loadError"
      :src="imgSrc"
      :alt="file.originalName"
      class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
      loading="lazy"
      @error="onImageError"
    />
    <div v-if="loadError" class="w-full h-full flex flex-col items-center justify-center gap-2 text-sm text-[var(--text-secondary)]">
      <span>{{ file.mediaType === 'video' ? '▶' : '📄' }}</span>
      <button @click.prevent.stop="retryLoad" class="px-2 py-1 rounded text-xs bg-[var(--accent)] text-white">
        Retry
      </button>
    </div>
    <div v-else-if="!file.thumbnails?.md" class="w-full h-full flex items-center justify-center text-3xl text-[var(--text-secondary)]">
      {{ file.mediaType === 'video' ? '▶' : '📄' }}
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
  mediaType: string
  durationSec?: number
  takenAt?: string
  thumbnails?: {
    sm?: { url: string; width: number; height: number }
    md?: { url: string; width: number; height: number }
  }
}

const props = defineProps<{ file: FileItem }>()

const loadError = ref(false)
const retryCounter = ref(0)

const imgSrc = computed(() => {
  const base = props.file.thumbnails?.md?.url || ''
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
