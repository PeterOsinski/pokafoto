<template>
  <div
    v-if="file"
    class="flex flex-col shadow-2xl shrink-0"
    :style="{ width: sidebarWidth + 'px', background: 'var(--bg-surface)', height: 'calc(100vh - 4rem)' }"
    @keydown="handleKeydown"
    tabindex="0"
    ref="sidebarEl"
    style="border-left: 1px solid var(--border-color); position: relative"
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
          v-if="file.mediaType === 'video'"
          :src="videoSrc"
          :proxySrc="proxySrc"
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

        <div v-if="!hideSocial">
          <div class="flex items-center justify-between mb-1">
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide">Albums</h3>
          </div>
          <div v-if="fileAlbumsLoading" class="text-xs text-[var(--text-secondary)]">Loading...</div>
          <div v-else>
            <div v-if="fileAlbums.length" class="flex flex-wrap gap-1 mb-2">
              <span
                v-for="a in fileAlbums"
                :key="a.id"
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs"
                :class="a.is_shared ? 'bg-green-500/10 text-green-400' : 'bg-[var(--bg-elevated)] text-[var(--text-secondary)]'"
              >
                {{ a.name }}
                <span v-if="a.is_shared" title="Shared">🔗</span>
              </span>
            </div>
            <button @click="toggleAddToAlbum" class="text-xs text-[var(--accent)] hover:underline cursor-pointer">
              {{ showAddToAlbum ? 'Cancel' : '+ Add to album' }}
            </button>
            <div v-if="showAddToAlbum" class="mt-2">
              <div v-if="availableAlbumsLoading" class="text-xs text-[var(--text-secondary)]">Loading...</div>
              <div v-else-if="!availableAlbums.length" class="text-xs text-[var(--text-secondary)]">No albums available. Create one first.</div>
              <div v-else class="max-h-32 overflow-y-auto border border-[var(--border-color)] rounded">
                <button
                  v-for="a in availableAlbums"
                  :key="a.id"
                  @click="addToAlbum(a.id)"
                  class="w-full text-left px-3 py-1.5 text-xs text-[var(--text-primary)] hover:bg-[var(--bg-elevated)] transition-colors"
                >
                  {{ a.name }} <span class="text-[var(--text-secondary)]">({{ a.item_count }})</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="!hideSocial">
          <div class="flex items-center justify-between mb-1">
            <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide">Comments</h3>
            <span class="text-xs text-[var(--text-secondary)]">{{ sidebarComments.length }}</span>
          </div>
          <div v-if="sidebarCommentsLoading" class="text-xs text-[var(--text-secondary)]">Loading...</div>
          <div v-else>
            <div v-if="sidebarComments.length" class="mb-2 max-h-40 overflow-y-auto">
              <div v-for="c in sidebarComments.slice(0, 5)" :key="c.id" class="flex gap-2 mb-2">
                <div class="flex-1">
                  <div class="flex items-center gap-1">
                    <span class="text-xs font-medium text-[var(--text-primary)]">{{ c.username }}</span>
                    <span class="text-[10px] text-[var(--text-secondary)]">{{ formatDate(c.created_at) }}</span>
                  </div>
                  <p class="text-xs text-[var(--text-secondary)]">{{ c.content }}</p>
                </div>
              </div>
              <p v-if="sidebarComments.length > 5" class="text-[10px] text-[var(--text-secondary)]">
                +{{ sidebarComments.length - 5 }} more
              </p>
            </div>
            <textarea
              v-model="newSidebarComment"
              placeholder="Add a comment..."
              rows="2"
              class="w-full px-2 py-1.5 text-xs bg-[var(--bg-color)] border border-[var(--border-color)] rounded text-[var(--text-primary)] placeholder-[var(--text-secondary)]/50 resize-none outline-none focus:border-[var(--accent)]"
            ></textarea>
            <button
              @click="addSidebarComment"
              :disabled="!newSidebarComment.trim()"
              class="mt-1 px-2 py-1 text-xs rounded bg-[var(--accent)] text-white disabled:opacity-50 hover:opacity-90 transition-opacity"
            >Post</button>
          </div>
        </div>

        <div v-if="!hideSocial">
          <h3 class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">Tags</h3>
          <div v-if="sidebarTagsLoading" class="text-xs text-[var(--text-secondary)]">Loading...</div>
          <div v-else>
            <TagInput v-model="sidebarTags" placeholder="Add tags..." />
            <button
              @click="saveSidebarTags"
              :disabled="sidebarTagsSaving"
              class="mt-2 px-2 py-1 text-xs rounded bg-[var(--accent)] text-white disabled:opacity-50 hover:opacity-90 transition-opacity"
            >{{ sidebarTagsSaving ? 'Saving...' : 'Save' }}</button>
          </div>
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
import { useAuthStore } from '../stores/auth'
import VideoPlayer from './VideoPlayer.vue'
import TagInput from './TagInput.vue'
import { useAlbumStore } from '../stores/albums'

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
    videoProxy?: { url: string; width: number; height: number }
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
  hideSocial?: boolean
}>()

const emit = defineEmits<{
  close: []
  prev: []
  next: []
}>()

const settings = useLocalSettings()
const albumStore = useAlbumStore()
const authStore = useAuthStore()
const sidebarEl = ref<HTMLElement | null>(null)
const exif = ref<ExifData | null>(null)
const videoSrc = ref('')
const proxySrc = ref('')
const showRawJson = ref(false)
const sidebarWidth = ref(settings.sidebarWidth.value)

const fileAlbums = ref<any[]>([])
const fileAlbumsLoading = ref(false)
const showAddToAlbum = ref(false)
const availableAlbums = ref<any[]>([])
const availableAlbumsLoading = ref(false)

const sidebarComments = ref<any[]>([])
const sidebarCommentsLoading = ref(false)
const newSidebarComment = ref('')

const sidebarTags = ref<string[]>([])
const sidebarTagsLoading = ref(false)
const sidebarTagsSaving = ref(false)

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
  showAddToAlbum.value = false
  fileAlbums.value = []
  sidebarComments.value = []
  sidebarTags.value = []

  videoSrc.value = ''
  proxySrc.value = ''
  showRawJson.value = false

  if (!file?.id) {
    exif.value = null
    return
  }
  nextTick(() => sidebarEl.value?.focus())

  if (file.mediaType === 'video') {
    const token = authStore.accessToken ? `&token=${authStore.accessToken}` : ''
    videoSrc.value = `/api/v1/video/${file.id}?quality=original${token}`
    proxySrc.value = file.thumbnails?.videoProxy?.url ? `${file.thumbnails.videoProxy.url}${token}` : ''
  }

  try {
    const res = await api.get(`/files/${file.id}`)
    exif.value = res.data.exif || null
  } catch {
    exif.value = null
  }

  fetchFileAlbums()
  fetchSidebarComments()
  fetchSidebarTags()
}, { immediate: true })

async function fetchFileAlbums() {
  if (!props.file?.id) return
  fileAlbumsLoading.value = true
  try {
    const res = await api.get(`/files/${props.file.id}/albums`)
    fileAlbums.value = res.data.albums || []
  } catch {
    fileAlbums.value = []
  } finally {
    fileAlbumsLoading.value = false
  }
}

async function toggleAddToAlbum() {
  showAddToAlbum.value = !showAddToAlbum.value
  if (showAddToAlbum.value) {
    availableAlbumsLoading.value = true
    try {
      await albumStore.fetchAlbums()
      availableAlbums.value = albumStore.myAlbums.filter(
        a => !fileAlbums.value.some(fa => fa.id === a.id)
      )
    } catch {} finally {
      availableAlbumsLoading.value = false
    }
  }
}

async function addToAlbum(albumId: string) {
  if (!props.file?.id) return
  try {
    await api.post(`/albums/${albumId}/items`, { file_ids: [props.file.id] })
    showAddToAlbum.value = false
    await fetchFileAlbums()
  } catch {}
}

async function fetchSidebarComments() {
  if (!props.file?.id) return
  sidebarCommentsLoading.value = true
  try {
    const res = await api.get(`/files/${props.file.id}/comments`)
    sidebarComments.value = res.data.comments || []
  } catch {
    sidebarComments.value = []
  } finally {
    sidebarCommentsLoading.value = false
  }
}

async function addSidebarComment() {
  if (!newSidebarComment.value.trim() || !props.file?.id) return
  try {
    await api.post(`/files/${props.file.id}/comments`, { content: newSidebarComment.value.trim() })
    newSidebarComment.value = ''
    await fetchSidebarComments()
  } catch {}
}

async function fetchSidebarTags() {
  if (!props.file?.id) return
  sidebarTagsLoading.value = true
  try {
    const res = await api.get(`/files/${props.file.id}/tags`)
    sidebarTags.value = (res.data.tags || []).map((t: any) => t.name)
  } catch {
    sidebarTags.value = []
  } finally {
    sidebarTagsLoading.value = false
  }
}

async function saveSidebarTags() {
  if (!props.file?.id) return
  sidebarTagsSaving.value = true
  try {
    await api.post(`/files/${props.file.id}/tags`, { tags: sidebarTags.value })
  } catch {} finally {
    sidebarTagsSaving.value = false
  }
}

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

function formatDate(d: string): string {
  return new Date(d).toLocaleDateString()
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
  const token = authStore.accessToken ? `?token=${authStore.accessToken}` : ''
  window.open(`/api/v1/download/${props.file.id}${token}`, '_blank')
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
})
</script>
