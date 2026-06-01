<template>
  <div
    v-if="file"
    class="fixed top-0 right-0 h-full z-50 flex flex-col shadow-2xl"
    :style="{ width: sidebarWidth + 'px', background: 'var(--bg-surface)' }"
    @keydown="handleKeydown"
    tabindex="0"
    ref="sidebarEl"
    style="border-left: 1px solid var(--border-color)"
  >
    <div
      class="absolute top-0 left-0 h-full cursor-col-resize z-10 hover:bg-[var(--accent)]/30 transition-colors"
      :style="{ width: '4px', marginLeft: '-2px' }"
      @mousedown="startResize"
    />

    <div class="flex items-center justify-between p-3 shrink-0" style="border-bottom: 1px solid var(--border-color)">
      <div class="flex items-center gap-3">
        <button
          class="text-[var(--text-secondary)] text-lg hover:text-[var(--text-primary)]"
          @click="$emit('close')"
        >&#10005;</button>
        <span class="text-xs text-[var(--text-secondary)]">{{ index + 1 }} / {{ total }}</span>
      </div>
      <div class="flex items-center gap-2">
        <button class="text-[var(--text-secondary)] hover:text-[var(--text-primary)]" @click="prev" :disabled="!hasPrev">&#9664;</button>
        <button class="text-[var(--text-secondary)] hover:text-[var(--text-primary)]" @click="next" :disabled="!hasNext">&#9654;</button>
      </div>
    </div>

    <div class="flex-1 overflow-y-auto">
      <div class="relative" style="background: var(--bg-color); min-height: 200px">
        <VideoPlayer
          v-if="file.mediaType === 'video' && videoSrc"
          :src="videoSrc"
          :poster="file.thumbnails?.videoStill?.url"
        />
        <img
          v-else-if="previewSrc"
          :src="previewSrc"
          :alt="file.originalName"
          class="w-full object-contain"
          draggable="false"
          style="max-height: 60vh"
        />
        <div v-else class="flex items-center justify-center py-12 text-[var(--text-secondary)]">
          No preview available
        </div>
      </div>

      <div class="p-3 space-y-4">
        <div>
          <p class="text-sm font-medium text-[var(--text-primary)]">{{ file.originalName }}</p>
          <p class="text-xs text-[var(--text-secondary)]">{{ formatSize(file.sizeBytes) }}</p>
        </div>

        <div v-if="exif" class="space-y-4">
          <div v-if="exif.cameraMake || exif.cameraModel">
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">Camera & Lens</h3>
            <p class="text-sm text-[var(--text-primary)]" v-if="exif.cameraMake || exif.cameraModel">
              {{ [exif.cameraMake, exif.cameraModel].filter(Boolean).join(' ') }}
            </p>
            <p class="text-xs text-[var(--text-secondary)]" v-if="exif.lensMake || exif.lensModel">
              Lens: {{ [exif.lensMake, exif.lensModel].filter(Boolean).join(' ') }}
            </p>
          </div>

          <div v-if="hasSettings">
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">Settings</h3>
            <p class="text-sm text-[var(--text-primary)]">
              <template v-if="exif.focalLength">{{ exif.focalLength }}mm </template>
              <template v-if="exif.aperture">f/{{ exif.aperture }} </template>
              <template v-if="exif.shutterSpeed">{{ exif.shutterSpeed }}s </template>
              <template v-if="exif.iso">ISO {{ exif.iso }} </template>
            </p>
            <p class="text-xs text-[var(--text-secondary)]" v-if="exif.dateTaken">{{ exif.dateTaken }}</p>
          </div>

          <div v-if="exif.gpsLatitude && exif.gpsLongitude">
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">Location</h3>
            <p class="text-sm text-[var(--text-primary)]">
              {{ exif.gpsLatitude.toFixed(4) }}, {{ exif.gpsLongitude.toFixed(4) }}
            </p>
            <p class="text-xs text-[var(--text-secondary)]" v-if="exif.gpsAltitude">
              Altitude: {{ exif.gpsAltitude }}m
            </p>
            <a
              :href="`https://www.openstreetmap.org/?mlat=${exif.gpsLatitude}&mlon=${exif.gpsLongitude}&zoom=15`"
              target="_blank"
              class="text-xs text-[var(--accent)] hover:underline"
            >View on Map</a>
          </div>

          <div>
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">File Info</h3>
            <p class="text-sm text-[var(--text-primary)]">{{ formatSize(file.sizeBytes) }} · {{ file.mimeType || 'unknown' }}</p>
            <p class="text-xs text-[var(--text-secondary)]" v-if="file.width && file.height">{{ file.width }}&#215;{{ file.height }}</p>
          </div>

          <div v-if="hasTechnical">
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">Technical</h3>
            <div class="text-xs text-[var(--text-secondary)] space-y-0.5">
              <p v-if="exif.orientation">Orientation: {{ exif.orientation }}</p>
              <p v-if="exif.colorSpace">Color Space: {{ exif.colorSpace }}</p>
              <p v-if="exif.flash !== undefined">Flash: {{ exif.flash ? 'Fired' : 'No' }}</p>
              <p v-if="exif.software">Software: {{ exif.software }}</p>
            </div>
          </div>

          <div v-if="exif.rawJson">
            <button
              @click="showRawJson = !showRawJson"
              class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide hover:text-[var(--text-primary)] transition-colors flex items-center gap-1"
            >
              Raw EXIF JSON
              <span class="text-[10px]">{{ showRawJson ? '&#9660;' : '&#9654;' }}</span>
            </button>
            <pre v-if="showRawJson" class="mt-1 p-2 rounded text-[11px] text-[var(--text-secondary)] overflow-x-auto" style="background: var(--bg-color); max-height: 300px;">{{ formatJson(exif.rawJson) }}</pre>
          </div>
        </div>
        <div v-else class="text-xs text-[var(--text-secondary)]">Loading metadata...</div>
      </div>
    </div>

    <div class="p-3 shrink-0 flex gap-2" style="border-top: 1px solid var(--border-color)">
      <button
        v-if="file.id"
        @click="downloadFile"
        class="flex-1 px-3 py-2 rounded text-sm text-white text-center"
        style="background: var(--accent)"
      >&#8595; Download</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import api from '../api/client'
import { useLocalSettings } from '../composables/useLocalSettings'
import VideoPlayer from './VideoPlayer.vue'

interface FileItem {
  id: string
  originalName: string
  filename?: string
  mediaType?: string
  sizeBytes: number
  width?: number
  height?: number
  mimeType?: string
  thumbnails?: {
    sm?: { url: string; width: number; height: number }
    md?: { url: string; width: number; height: number }
    preview?: { url: string; width: number; height: number }
    videoStill?: { url: string; width: number; height: number }
  }
}

interface ExifData {
  cameraMake?: string
  cameraModel?: string
  lensMake?: string
  lensModel?: string
  focalLength?: number
  aperture?: number
  shutterSpeed?: string
  iso?: number
  dateTaken?: string
  gpsLatitude?: number
  gpsLongitude?: number
  gpsAltitude?: number
  orientation?: number
  colorSpace?: string
  flash?: number
  software?: string
  rawJson?: string
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

const settings = useLocalSettings()
const sidebarEl = ref<HTMLElement | null>(null)
const exif = ref<ExifData | null>(null)
const videoSrc = ref('')
const showRawJson = ref(false)
const sidebarWidth = ref(settings.sidebarWidth.value)

let oldVideoURL = ''
let isResizing = false
let resizeStartX = 0
let resizeStartWidth = 0

const previewSrc = computed(() => {
  const t = props.file?.thumbnails
  if (!t) return ''
  return t.preview?.url || t.md?.url || ''
})

const hasSettings = computed(() => {
  return !!(exif.value?.focalLength || exif.value?.aperture || exif.value?.shutterSpeed || exif.value?.iso || exif.value?.dateTaken)
})

const hasTechnical = computed(() => {
  return !!(exif.value?.orientation || exif.value?.colorSpace || exif.value?.flash !== undefined || exif.value?.software)
})

watch(() => props.file, async (file) => {
  if (oldVideoURL) {
    URL.revokeObjectURL(oldVideoURL)
    oldVideoURL = ''
  }
  videoSrc.value = ''
  showRawJson.value = false

  if (!file?.id) {
    exif.value = null
    return
  }
  nextTick(() => sidebarEl.value?.focus())

  if (file.mediaType === 'video') {
    try {
      const res = await api.get(`/download/${file.id}`, { responseType: 'blob' })
      oldVideoURL = URL.createObjectURL(res.data as Blob)
      videoSrc.value = oldVideoURL
    } catch {
      // ignore
    }
  }

  try {
    const res = await api.get(`/files/${file.id}`)
    exif.value = res.data.exif || null
  } catch {
    exif.value = null
  }
}, { immediate: true })

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowLeft' && props.hasPrev) emit('prev')
  if (e.key === 'ArrowRight' && props.hasNext) emit('next')
}

function prev() { if (props.hasPrev) emit('prev') }
function next() { if (props.hasNext) emit('next') }

function startResize(e: MouseEvent) {
  isResizing = true
  resizeStartX = e.clientX
  resizeStartWidth = sidebarWidth.value
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

function onResize(e: MouseEvent) {
  if (!isResizing) return
  const delta = resizeStartX - e.clientX
  const newWidth = Math.max(300, Math.min(window.innerWidth * 0.7, resizeStartWidth + delta))
  sidebarWidth.value = newWidth
}

function stopResize() {
  isResizing = false
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  settings.sidebarWidth.value = sidebarWidth.value
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

function formatJson(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

async function downloadFile() {
  if (!props.file?.id) return
  try {
    const res = await api.get(`/download/${props.file.id}`, { responseType: 'blob' })
    const url = URL.createObjectURL(res.data as Blob)
    const a = document.createElement('a')
    a.href = url
    a.download = props.file.originalName || ''
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch {
    // ignore
  }
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
})
</script>
