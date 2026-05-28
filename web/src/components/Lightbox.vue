<template>
  <div
    v-if="file"
    class="fixed inset-0 z-50 flex flex-col bg-black/95"
    @keydown="handleKeydown"
    tabindex="0"
    ref="lightboxEl"
  >
    <div class="flex items-center justify-between p-3 shrink-0">
      <button class="text-white text-xl hover:text-[var(--accent)]" @click="$emit('close')">✕</button>
      <div class="flex gap-3">
        <button class="text-white text-lg hover:text-[var(--accent)]" @click="prev" :disabled="!hasPrev">◀</button>
        <span class="text-white text-sm self-center">{{ index + 1 }} / {{ total }}</span>
        <button class="text-white text-lg hover:text-[var(--accent)]" @click="next" :disabled="!hasNext">▶</button>
      </div>
      <button
        v-if="file.id"
        @click="downloadFile"
        class="text-white text-sm hover:text-[var(--accent)]"
      >⬇ Download</button>
    </div>

    <div
      class="flex-1 flex items-center justify-center overflow-hidden"
      @touchstart="onTouchStart"
      @touchend="onTouchEnd"
    >
      <VideoPlayer
        v-if="file.mediaType === 'video' && videoSrc"
        :src="videoSrc"
        :poster="file.videoStill?.url"
      />
      <div v-else-if="file.mediaType === 'video' && videoLoading" class="text-[var(--text-secondary)] text-lg">Loading video...</div>
      <img
        v-else-if="file.thumbnails?.preview || file.thumbnails?.md"
        :src="(file.thumbnails?.preview || file.thumbnails?.md)?.url"
        :alt="file.originalName"
        class="max-h-full max-w-full object-contain select-none"
        draggable="false"
      />
      <div v-else class="text-[var(--text-secondary)] text-lg">No preview available</div>
    </div>

    <div class="p-3 shrink-0" style="background: rgba(0,0,0,0.7)">
      <p class="text-sm text-white">{{ file.originalName }}</p>
      <p class="text-xs text-white/60">{{ formatSize(file.sizeBytes) }}</p>
      <div v-if="exif" class="mt-2 text-xs text-white/70 space-y-1">
        <p v-if="exif.cameraMake || exif.cameraModel">
          📷 {{ [exif.cameraMake, exif.cameraModel].filter(Boolean).join(' ') }}
        </p>
        <p v-if="exif.focalLength || exif.aperture || exif.shutterSpeed || exif.iso">
          {{ exif.focalLength ? exif.focalLength + 'mm' : '' }}
          {{ exif.aperture ? 'f/' + exif.aperture : '' }}
          {{ exif.shutterSpeed ? exif.shutterSpeed + 's' : '' }}
          {{ exif.iso ? 'ISO ' + exif.iso : '' }}
        </p>
        <p v-if="exif.dateTaken">🗓 {{ exif.dateTaken }}</p>
        <p v-if="exif.gpsLatitude && exif.gpsLongitude">
          📍 {{ exif.gpsLatitude.toFixed(4) }}, {{ exif.gpsLongitude.toFixed(4) }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import api from '../api/client'
import VideoPlayer from './VideoPlayer.vue'

interface FileItem {
  id: string
  originalName: string
  mediaType?: string
  sizeBytes: number
  videoStill?: { url: string }
  thumbnails?: {
    sm: { url: string; width: number; height: number }
    md: { url: string; width: number; height: number }
    preview: { url: string; width: number; height: number }
  }
}

interface ExifData {
  cameraMake?: string
  cameraModel?: string
  focalLength?: number
  aperture?: number
  shutterSpeed?: string
  iso?: number
  dateTaken?: string
  gpsLatitude?: number
  gpsLongitude?: number
}

const props = defineProps<{
  file: FileItem | null
  index: number
  total: number
  hasPrev: boolean
  hasNext: boolean
}>()

const emit = defineEmits<{
  close: []
  prev: []
  next: []
}>()

const lightboxEl = ref<HTMLElement | null>(null)
const exif = ref<ExifData | null>(null)
const videoSrc = ref('')
const videoLoading = ref(false)

let oldVideoURL = ''

let touchStartX = 0
let touchStartY = 0

watch(() => props.file, async (file) => {
  if (oldVideoURL) {
    URL.revokeObjectURL(oldVideoURL)
    oldVideoURL = ''
  }
  videoSrc.value = ''
  videoLoading.value = false

  if (!file?.id) {
    exif.value = null
    return
  }
  nextTick(() => lightboxEl.value?.focus())

  if (file.mediaType === 'video') {
    videoLoading.value = true
    try {
      const res = await api.get(`/download/${file.id}`, { responseType: 'blob' })
      oldVideoURL = URL.createObjectURL(res.data)
      videoSrc.value = oldVideoURL
    } catch (e) {
      console.error('Failed to load video', e)
    } finally {
      videoLoading.value = false
    }
  }

  try {
    const res = await api.get(`/files/${file.id}`)
    exif.value = res.data.exif || null
  } catch {}
}, { immediate: true })

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowLeft' && props.hasPrev) emit('prev')
  if (e.key === 'ArrowRight' && props.hasNext) emit('next')
}

function onTouchStart(e: TouchEvent) {
  touchStartX = e.touches[0].clientX
  touchStartY = e.touches[0].clientY
}

function onTouchEnd(e: TouchEvent) {
  const dx = e.changedTouches[0].clientX - touchStartX
  const dy = e.changedTouches[0].clientY - touchStartY
  if (Math.abs(dx) > Math.abs(dy) && Math.abs(dx) > 50) {
    if (dx < 0 && props.hasNext) emit('next')
    if (dx > 0 && props.hasPrev) emit('prev')
  }
  if (Math.abs(dy) > 100 && !exif.value) emit('close')
}

function prev() { if (props.hasPrev) emit('prev') }
function next() { if (props.hasNext) emit('next') }

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

async function downloadFile() {
  if (!props.file?.id) return
  try {
    const res = await api.get(`/download/${props.file.id}`, { responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = props.file.originalName || ''
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e) {
    console.error('Download failed', e)
  }
}
</script>
