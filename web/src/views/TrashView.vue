<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-[var(--text-primary)]">Trash</h2>
      <div class="flex items-center gap-2">
        <span v-if="trashCount > 0" class="text-sm text-[var(--text-secondary)]">
          {{ trashCount }} {{ trashCount === 1 ? 'file' : 'files' }} ({{ formatBytes(trashSizeBytes) }})
        </span>
        <button
          v-if="trashCount > 0"
          @click="showEmptyConfirm = true"
          class="px-3 py-1 rounded text-sm text-white"
          style="background: #ef4444"
        >
          Empty Trash
        </button>
      </div>
    </div>

    <div v-if="trashCount === 0 && !loading" class="text-center py-20 text-[var(--text-secondary)]">
      <p class="text-lg">Trash is empty.</p>
      <p class="mt-1 text-sm">Deleted files will appear here for 30 days before being permanently removed.</p>
    </div>

    <ActionBar
      v-if="selectedIds.size > 0"
      :count="selectedIds.size"
      @delete="showDeleteConfirm = true"
      @restore="executeRestore"
      @deselectAll="clearSelection"
    />

    <GalleryTileView
      v-if="files.length > 0"
      :files="files"
      :thumbSizePx="180"
      :selectedIds="selectedIds"
      :selectionEnabled="true"
      @select="toggleSelect"
      @deselect="toggleSelect"
      @open="handleFileClick"
    />

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
    <div v-else-if="loadingMore" class="text-center py-4 text-[var(--text-secondary)]">Loading more...</div>
    <div ref="sentinel" class="h-4"></div>

    <div
      v-if="showDeleteConfirm"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="showDeleteConfirm = false"
    >
      <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4" style="border: 1px solid var(--border-color)">
        <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Permanently delete?</h3>
        <p class="text-sm text-[var(--text-secondary)] mb-4">
          {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'file' : 'files' }} will be permanently deleted.
          This action cannot be undone.
        </p>
        <div class="flex justify-end gap-3">
          <button @click="showDeleteConfirm = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Cancel</button>
          <button @click="executeDelete" class="px-4 py-2 rounded text-sm text-white" style="background: #ef4444">Delete Permanently</button>
        </div>
      </div>
    </div>

    <div
      v-if="showEmptyConfirm"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="showEmptyConfirm = false"
    >
      <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4" style="border: 1px solid var(--border-color)">
        <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Empty trash?</h3>
        <p class="text-sm text-[var(--text-secondary)] mb-4">
          All {{ trashCount }} {{ trashCount === 1 ? 'file' : 'files' }} ({{ formatBytes(trashSizeBytes) }}) will be permanently deleted.
          This action cannot be undone.
        </p>
        <div class="flex justify-end gap-3">
          <button @click="showEmptyConfirm = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Cancel</button>
          <button @click="executeEmpty" class="px-4 py-2 rounded text-sm text-white" style="background: #ef4444">Empty Trash</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api/client'
import GalleryTileView from '../components/GalleryTileView.vue'
import ActionBar from '../components/TrashActionBar.vue'

interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  thumbnails?: {
    sm: { url: string; width: number; height: number }
    md: { url: string; width: number; height: number }
    preview: { url: string; width: number; height: number }
  }
  deletedAt?: string
}

const files = ref<FileItem[]>([])
const nextCursor = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const sentinel = ref<HTMLElement | null>(null)
const selectedIds = ref(new Set<string>())
const showDeleteConfirm = ref(false)
const showEmptyConfirm = ref(false)
const trashCount = ref(0)
const trashSizeBytes = ref(0)

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return val.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function toggleSelect(id: string) {
  const newSet = new Set(selectedIds.value)
  if (newSet.has(id)) {
    newSet.delete(id)
  } else {
    newSet.add(id)
  }
  selectedIds.value = newSet
}

function handleFileClick(index: number) {
  const file = files.value[index]
  if (file) toggleSelect(file.id)
}

function clearSelection() {
  selectedIds.value = new Set()
}

async function loadTrashStats() {
  try {
    const res = await api.get('/trash/stats')
    trashCount.value = res.data.count
    trashSizeBytes.value = res.data.size_bytes
  } catch {}
}

async function loadFiles(reset = true) {
  if (reset) {
    files.value = []
    nextCursor.value = ''
    loading.value = true
  } else {
    loadingMore.value = true
  }
  try {
    const params: any = { limit: 500, order: 'desc' }
    if (nextCursor.value) params.cursor = nextCursor.value
    const res = await api.get('/trash', { params })
    files.value = reset ? res.data.items : [...files.value, ...res.data.items]
    nextCursor.value = res.data.nextCursor || ''
  } catch (e) {
    console.error('Failed to load trash', e)
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

async function executeRestore() {
  try {
    await api.post('/trash/batch-restore', { ids: Array.from(selectedIds.value) })
    clearSelection()
    await loadTrashStats()
    await loadFiles(true)
  } catch (e) {
    console.error('Failed to restore files', e)
  }
}

async function executeDelete() {
  try {
    await api.post('/trash/batch-permanent-delete', { ids: Array.from(selectedIds.value) })
    clearSelection()
    showDeleteConfirm.value = false
    await loadTrashStats()
    await loadFiles(true)
  } catch (e) {
    console.error('Failed to delete files', e)
  }
}

async function executeEmpty() {
  try {
    await api.post('/trash/empty')
    showEmptyConfirm.value = false
    await loadTrashStats()
    await loadFiles(true)
  } catch (e) {
    console.error('Failed to empty trash', e)
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Delete' && selectedIds.value.size > 0 && document.activeElement?.tagName !== 'INPUT') {
      showDeleteConfirm.value = true
    }
  })
}

onMounted(() => {
  loadTrashStats()
  loadFiles(true)

  if (sentinel.value) {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && nextCursor.value && !loadingMore.value) {
          loadFiles(false)
        }
      },
      { rootMargin: '200px' }
    )
    observer.observe(sentinel.value)
  }
})
</script>
