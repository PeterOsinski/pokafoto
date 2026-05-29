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
      <button
        class="text-white text-sm hover:text-[var(--accent)]"
        @click="downloadFile"
      >⬇ Download</button>
    </div>

    <div class="flex-1 min-h-0">
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
const viewerType = ref<ViewerType>('text')
const rawBlob = ref<Blob | null>(null)

let destroyed = false

watch(() => props.file, async (f) => {
  cleanup()
  if (!f?.id) {
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  viewerType.value = detectViewerType(f.originalName, f.mimeType)
  nextTick(() => viewerEl.value?.focus())

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

function detectViewerType(name: string, mime: string): ViewerType {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (ext === 'pdf' || mime === 'application/pdf') return 'pdf'
  if (ext === 'json' || mime === 'application/json') return 'json'
  if (ext === 'md' || ext === 'markdown' || mime === 'text/markdown') return 'markdown'
  if (ext === 'csv' || mime === 'text/csv') return 'csv'
  return 'text'
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
</script>
