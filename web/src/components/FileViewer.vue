<template>
  <div
    v-if="file"
    class="fixed inset-0 z-50 flex flex-col bg-black/95"
    @keydown.escape="$emit('close')"
    tabindex="0"
    ref="viewerEl"
  >
    <div class="flex items-center justify-between p-3 shrink-0">
      <button class="text-white text-xl hover:text-[var(--accent)]" @click="$emit('close')">✕</button>
      <span class="text-white text-sm truncate max-w-[60%]">{{ file.originalName }}</span>
      <div class="flex items-center gap-2">
        <button
          v-if="file.id"
          @click="panel = panel === 'comments' ? '' : 'comments'"
          class="text-white text-sm hover:text-[var(--accent)] px-2 py-1 rounded"
          :class="{ 'bg-white/10': panel === 'comments' }"
        >💬</button>
        <button
          v-if="file.id"
          @click="panel = panel === 'tags' ? '' : 'tags'"
          class="text-white text-sm hover:text-[var(--accent)] px-2 py-1 rounded"
          :class="{ 'bg-white/10': panel === 'tags' }"
        >🏷</button>
        <button
          class="text-white text-sm hover:text-[var(--accent)]"
          @click="downloadFile"
        >⬇ Download</button>
      </div>
    </div>

    <div class="flex-1 flex overflow-hidden min-h-0">
      <div class="flex-1 min-h-0 overflow-auto">
        <div v-if="loading" class="flex items-center justify-center h-full">
          <span class="text-[var(--text-secondary)] text-lg">Loading...</span>
        </div>

        <div v-else-if="error" class="flex flex-col items-center justify-center h-full gap-4 px-4">
          <p class="text-[var(--text-secondary)] text-lg">{{ error }}</p>
          <button
            class="px-4 py-2 rounded text-white"
            style="background: var(--accent)"
            @click="downloadFile"
          >Download raw file</button>
        </div>

        <template v-else>
          <PdfViewer v-if="viewerType === 'pdf'" :blobUrl="blobUrl" />
          <JsonViewer v-else-if="viewerType === 'json'" :content="textContent" />
          <MarkdownViewer v-else-if="viewerType === 'markdown'" :content="textContent" />
          <CsvViewer v-else-if="viewerType === 'csv'" :content="textContent" />
          <TextViewer v-else :content="textContent" />
        </template>
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
      <p class="text-xs text-white/60">{{ formatSize(file.sizeBytes) }} · {{ file.mimeType }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted, nextTick } from 'vue'
import api from '../api/client'
import PdfViewer from './viewers/PdfViewer.vue'
import JsonViewer from './viewers/JsonViewer.vue'
import MarkdownViewer from './viewers/MarkdownViewer.vue'
import CsvViewer from './viewers/CsvViewer.vue'
import TextViewer from './viewers/TextViewer.vue'
import CommentsSection from './CommentsSection.vue'
import TagInput from './TagInput.vue'

interface FileItem {
  id: string
  originalName: string
  sizeBytes: number
  mimeType: string
}

type ViewerType = 'pdf' | 'json' | 'markdown' | 'csv' | 'text'

const props = defineProps<{
  file: FileItem | null
}>()

defineEmits<{
  close: []
}>()

const viewerEl = ref<HTMLElement | null>(null)
const loading = ref(true)
const error = ref('')
const blobUrl = ref<string | null>(null)
const textContent = ref('')
const viewerType = ref<ViewerType | null>(null)
const rawBlob = ref<Blob | null>(null)
const panel = ref('')
const comments = ref<any[]>([])
const commentsLoading = ref(false)
const newComment = ref('')
const fileTags = ref<string[]>([])
const tagsLoading = ref(false)
const tagsLoaded = ref(false)

let destroyed = false
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null

watch(() => props.file, async (f) => {
  cleanup()
  panel.value = ''
  comments.value = []
  fileTags.value = []
  tagsLoaded.value = false
  if (!f?.id) {
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  viewerType.value = detectViewerType(f.originalName, f.mimeType)
  nextTick(() => viewerEl.value?.focus())

  if (!viewerType.value) {
    loading.value = false
    error.value = `No preview available for .${f.originalName.split('.').pop()} files.`
    return
  }

  try {
    const res = await api.get(`/download/${f.id}`, { responseType: 'blob' })
    if (destroyed) return
    rawBlob.value = res.data as Blob

    if (viewerType.value === 'pdf') {
      blobUrl.value = URL.createObjectURL(rawBlob.value)
    } else {
      textContent.value = await (res.data as Blob).text()
    }
  } catch {
    if (destroyed) return
    error.value = 'Could not load this file. The file may be unavailable.'
  } finally {
    if (!destroyed) loading.value = false
  }
}, { immediate: true })

onUnmounted(() => {
  destroyed = true
  cleanup()
})

function cleanup() {
  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
    blobUrl.value = null
  }
  rawBlob.value = null
  textContent.value = ''
}

function detectViewerType(name: string, mime: string): ViewerType | null {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (ext === 'pdf' || mime === 'application/pdf') return 'pdf'
  if (ext === 'json' || mime === 'application/json') return 'json'
  if (ext === 'md' || ext === 'markdown' || mime === 'text/markdown') return 'markdown'
  if (ext === 'csv' || mime === 'text/csv') return 'csv'
  if (ext === 'txt' || mime === 'text/plain') return 'text'
  return null
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

async function downloadFile() {
  if (rawBlob.value) {
    triggerDownload(rawBlob.value, props.file?.originalName || 'file')
    return
  }
  if (!props.file?.id) return
  try {
    const res = await api.get(`/download/${props.file.id}`, { responseType: 'blob' })
    triggerDownload(res.data as Blob, props.file.originalName || 'file')
  } catch {
    error.value = 'Download failed'
  }
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

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

watch(fileTags, () => {
  if (!tagsLoaded.value || !props.file?.id) return
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(async () => {
    try {
      await api.post(`/files/${props.file!.id}/tags`, { tags: fileTags.value })
    } catch {}
  }, 500)
})
</script>
