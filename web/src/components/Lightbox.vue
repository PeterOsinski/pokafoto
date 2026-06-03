<template>
  <div
    v-if="file"
    class="fixed inset-0 z-50 flex flex-col bg-black/95"
    @keydown="handleKeydown"
    tabindex="0"
    ref="lightboxEl"
  >
    <div class="flex items-center justify-between p-3 shrink-0">
      <button class="text-white text-xl hover:text-[var(--accent)]" @click="handleClose">✕</button>
      <div class="flex gap-3">
        <button class="text-white text-lg hover:text-[var(--accent)]" @click="prev" :disabled="!hasPrev">◀</button>
        <span class="text-white text-sm self-center">{{ index + 1 }} / {{ total }}</span>
        <button class="text-white text-lg hover:text-[var(--accent)]" @click="next" :disabled="!hasNext">▶</button>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="file.id && !hideSocial"
          @click="panel = panel === 'comments' ? '' : 'comments'"
          class="text-white text-sm hover:text-[var(--accent)] px-2 py-1 rounded"
          :class="{ 'bg-white/10': panel === 'comments' }"
        >💬</button>
        <button
          v-if="file.id && !hideSocial"
          @click="panel = panel === 'tags' ? '' : 'tags'"
          class="text-white text-sm hover:text-[var(--accent)] px-2 py-1 rounded"
          :class="{ 'bg-white/10': panel === 'tags' }"
        >🏷</button>
        <button
          v-if="file.id"
          @click="downloadFile"
          class="text-white text-sm hover:text-[var(--accent)]"
        >⬇ Download</button>
      </div>
      <label
        v-if="file.id && file.thumbnails?.xl"
        class="text-white text-xs flex items-center gap-1 cursor-pointer select-none"
      >
        <input type="checkbox" :checked="settings.highResDownload.value" @change="settings.highResDownload.value = ($event.target as HTMLInputElement).checked" class="cursor-pointer" />
        2000px
      </label>
    </div>

    <div class="flex-1 flex overflow-hidden">
      <div
        class="flex-1 flex items-center justify-center overflow-hidden"
        @touchstart="onTouchStart"
        @touchend="onTouchEnd"
      >
        <VideoPlayer
          v-if="file.mediaType === 'video'"
          :key="file.id"
          :src="videoSrc"
          :proxySrc="proxySrc"
          :poster="file.videoStill?.url"
        />
        <img
          v-else-if="previewSrc"
          :key="settings.highResDownload.value ? 'xl' : 'preview'"
          :src="previewSrc"
          :alt="file.originalName"
          class="max-h-full max-w-full object-contain select-none"
          draggable="false"
        />
        <div v-else class="text-white/60 text-lg">No preview available</div>
      </div>

      <div v-if="panel" class="w-80 shrink-0 overflow-y-auto p-4 border-l border-white/10" style="background: rgba(0,0,0,0.9)">
        <template v-if="panel === 'comments'">
          <h3 class="text-sm font-medium text-white mb-3">Comments</h3>
          <CommentsSection
            :comments="comments"
            :file-id="file?.id || ''"
            @delete="deleteComment"
            @toggle-reaction="toggleReaction"
          />
          <div v-if="!comments.length && !commentsLoading" class="text-xs text-gray-500 text-center py-4">No comments yet</div>
          <div v-if="commentsLoading" class="text-xs text-gray-500 text-center py-4">Loading...</div>
          <form @submit.prevent="addComment" class="mt-4">
            <textarea
              v-model="newComment"
              placeholder="Add a comment..."
              rows="2"
              class="w-full px-3 py-2 bg-black/50 border border-white/10 rounded text-sm text-gray-200 placeholder-gray-500 resize-none outline-none focus:border-blue-500"
            ></textarea>
            <button type="submit" :disabled="!newComment.trim()" class="mt-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs rounded transition-colors">Post</button>
          </form>
        </template>

        <template v-else-if="panel === 'tags'">
          <h3 class="text-sm font-medium text-white mb-3">Tags</h3>
          <div v-if="tagsLoading" class="text-xs text-gray-500 text-center py-4">Loading...</div>
          <TagInput v-model="fileTags" placeholder="Add tags..." />
        </template>
      </div>
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
import { ref, computed, watch, nextTick } from 'vue'
import api from '../api/client'
import { useLocalSettings } from '../composables/useLocalSettings'
import { useAuthStore } from '../stores/auth'
import VideoPlayer from './VideoPlayer.vue'
import CommentsSection from './CommentsSection.vue'
import TagInput from './TagInput.vue'

interface FileItem {
  id: string
  originalName: string
  mediaType?: string
  sizeBytes: number
  videoStill?: { url: string }
  thumbnails?: {
    sm: { url: string; width: number; height: number }
    md: { url: string; width: number; height: number }
    xl?: { url: string; width: number; height: number }
    preview: { url: string; width: number; height: number }
    videoProxy?: { url: string; width: number; height: number }
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
  hideSocial?: boolean
  downloadUrl?: string
}>()

const emit = defineEmits<{
  close: []
  prev: []
  next: []
}>()

const lightboxEl = ref<HTMLElement | null>(null)
const exif = ref<ExifData | null>(null)
const videoSrc = ref('')
const proxySrc = ref('')
const settings = useLocalSettings()
const authStore = useAuthStore()
const panel = ref('')
const comments = ref<any[]>([])
const commentsLoading = ref(false)
const newComment = ref('')
const fileTags = ref<string[]>([])
const tagsLoading = ref(false)
const tagsLoaded = ref(false)

const previewSrc = computed(() => {
  const t = props.file?.thumbnails
  if (!t) return ''
  if (settings.highResDownload.value && t.xl) return t.xl.url
  return t.preview?.url || t.md?.url || ''
})

let touchStartX = 0
let touchStartY = 0

watch(() => props.file, async (file) => {
  panel.value = ''
  comments.value = []
  fileTags.value = []
  tagsLoaded.value = false

  videoSrc.value = ''
  proxySrc.value = ''

  if (!file?.id) {
    exif.value = null
    return
  }
  nextTick(() => lightboxEl.value?.focus())

  if (file.mediaType === 'video') {
    const token = authStore.accessToken ? `&token=${authStore.accessToken}` : ''
    videoSrc.value = `/api/v1/video/${file.id}?quality=original${token}`
    proxySrc.value = file.thumbnails?.videoProxy?.url ? `${file.thumbnails.videoProxy.url}${token}` : ''
  }

  try {
    const res = await api.get(`/files/${file.id}`)
    exif.value = res.data.exif || null
  } catch {}
}, { immediate: true })

async function fetchComments() {
  if (!props.file?.id) return
  commentsLoading.value = true
  try {
    const res = await api.get(`/files/${props.file.id}/comments`)
    comments.value = res.data.comments || []
  } catch {
    comments.value = []
  } finally {
    commentsLoading.value = false
  }
}

async function fetchTags() {
  if (!props.file?.id) return
  tagsLoading.value = true
  tagsLoaded.value = false
  try {
    const res = await api.get(`/files/${props.file.id}/tags`)
    fileTags.value = (res.data.tags || []).map((t: any) => t.name)
  } catch {
    fileTags.value = []
  } finally {
    tagsLoading.value = false
    nextTick(() => { tagsLoaded.value = true })
  }
}

watch(() => panel.value, (val) => {
  if (val === 'comments') fetchComments()
  if (val === 'tags') fetchTags()
})

async function addComment() {
  if (!newComment.value.trim() || !props.file?.id) return
  try {
    await api.post(`/files/${props.file.id}/comments`, { content: newComment.value.trim() })
    newComment.value = ''
    await fetchComments()
  } catch {}
}

async function deleteComment(commentId: string) {
  if (!props.file?.id) return
  try {
    await api.delete(`/files/${props.file.id}/comments/${commentId}`)
    await fetchComments()
  } catch {}
}

async function toggleReaction(commentId: string, emoji: string) {
  if (!props.file?.id) return
  try {
    await api.post(`/files/${props.file.id}/comments/${commentId}/reactions`, { emoji })
    await fetchComments()
  } catch {}
}

let autoSaveTimer: ReturnType<typeof setTimeout> | null = null

watch(fileTags, () => {
  if (!tagsLoaded.value || !props.file?.id) return
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(async () => {
    try {
      await api.post(`/files/${props.file!.id}/tags`, { tags: fileTags.value })
    } catch {}
  }, 500)
})

function handleClose() {
  panel.value = ''
  emit('close')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') handleClose()
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
  if (Math.abs(dy) > 100 && !exif.value) handleClose()
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
  if (props.downloadUrl) {
    window.open(props.downloadUrl, '_blank')
    return
  }
  const token = authStore.accessToken ? `?token=${authStore.accessToken}` : ''
  window.open(`/api/v1/download/${props.file.id}${token}`, '_blank')
}
</script>
