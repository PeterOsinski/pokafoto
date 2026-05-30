<template>
  <div>
    <ActionBar
      :count="selectedIds.size"
      @delete="showDeleteConfirm = true"
      @move="startMove"
      @copy="startCopy"
      @deselectAll="clearSelection"
    />

    <FilterBar
      v-model:mediaType="mediaType"
      v-model:sortBy="settings.sortBy.value"
      v-model:layout="settings.layout.value"
      v-model:thumbLevel="settings.thumbLevel.value"
      v-model:includeAllFolders="includeAllFolders"
    />

    <div class="flex items-center gap-2 mb-4">
      <InlineUpload label="Upload" :skipNameSizeDedup="false" accept="image/*,video/*" />
    </div>

    <div ref="scrollRef" class="overflow-y-auto" style="flex: 1">
    <div v-if="files.length === 0 && !loading" class="text-center py-20 text-[var(--text-secondary)]">
      <p class="text-lg">No photos yet.</p>
      <p class="mt-2">Upload your first photo to get started.</p>
      <div class="mt-4 flex justify-center">
        <InlineUpload label="Upload Photos" :skipNameSizeDedup="false" accept="image/*,video/*" />
      </div>
    </div>

    <GalleryTileView
      v-else-if="layout === 'tiles'"
      :files="files"
      :thumbSizePx="settings.thumbSizePx.value"
      :selectedIds="selectedIds"
      :selectionEnabled="selectionEnabled"
      @select="toggleSelect"
      @deselect="toggleSelect"
      @open="(i: number) => handleFileClick(i)"
    />
    <GalleryListView
      v-else-if="layout === 'list'"
      :files="files"
      :selectedIds="selectedIds"
      :selectionEnabled="selectionEnabled"
      @select="toggleSelect"
      @deselect="toggleSelect"
      @open="(i: number) => handleFileClick(i)"
    />
    <GalleryGroupedView
      v-else-if="layout === 'grouped'"
      :files="files"
      :thumbSizePx="settings.thumbSizePx.value"
      :selectedIds="selectedIds"
      :selectionEnabled="selectionEnabled"
      @select="toggleSelect"
      @deselect="toggleSelect"
      @open="(i: number) => handleFileClick(i)"
    />

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
    <div v-else-if="loadingMore" class="text-center py-4 text-[var(--text-secondary)]">Loading more...</div>
    <div ref="sentinel" class="h-4"></div>
    </div>

    <Lightbox
      :file="lightboxFile"
      :index="lightboxIndex"
      :total="files.length"
      :hasPrev="lightboxIndex > 0"
      :hasNext="lightboxIndex < files.length - 1"
      @close="closeLightbox"
      @prev="goPrev"
      @next="goNext"
    />

    <FileViewer
      :file="fileViewerFile"
      @close="closeFileViewer"
    />

    <FolderPickerDialog
      :open="moveDialog.open"
      title="Move to folder"
      actionLabel="Move here"
      @close="moveDialog.open = false"
      @confirm="(folderId) => executeMove(folderId)"
    />

    <FolderPickerDialog
      :open="copyDialog.open"
      title="Copy to folder"
      actionLabel="Copy here"
      @close="copyDialog.open = false"
      @confirm="(folderId) => executeCopy(folderId)"
    />

    <div
      v-if="showDeleteConfirm"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="showDeleteConfirm = false"
    >
      <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4" style="border: 1px solid var(--border-color)">
        <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Delete files?</h3>
        <p class="text-sm text-[var(--text-secondary)] mb-4">This will move {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'file' : 'files' }} to trash. You can recover them later.</p>
        <div class="flex justify-end gap-3">
          <button @click="showDeleteConfirm = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Cancel</button>
          <button @click="executeDelete" class="px-4 py-2 rounded text-sm text-white" style="background: #ef4444">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client'
import { useRouteQuery } from '../composables/useRouteQuery'
import { useLocalSettings } from '../composables/useLocalSettings'
import Lightbox from '../components/Lightbox.vue'
import FileViewer from '../components/FileViewer.vue'
import GalleryTileView from '../components/GalleryTileView.vue'
import GalleryListView from '../components/GalleryListView.vue'
import GalleryGroupedView from '../components/GalleryGroupedView.vue'
import FilterBar from '../components/FilterBar.vue'
import ActionBar from '../components/ActionBar.vue'
import FolderPickerDialog from '../components/FolderPickerDialog.vue'
import InlineUpload from '../components/InlineUpload.vue'
import { useUploadStore } from '../stores/upload'

interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  durationSec?: number
  takenAt?: string
  folder_id?: string | null
  thumbnails?: {
    sm: { url: string; width: number; height: number }
    lg: { url: string; width: number; height: number }
    md: { url: string; width: number; height: number }
    preview: { url: string; width: number; height: number }
  }
}

const route = useRoute()

const files = ref<FileItem[]>([])
const total = ref(0)
const nextCursor = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const scrollRef = ref<HTMLElement | null>(null)
const sentinel = ref<HTMLElement | null>(null)

const pathQuery = useRouteQuery('path', '')
const mediaQuery = useRouteQuery('media', '')
const allFoldersQuery = useRouteQuery('all_folders', '')
const photoQuery = useRouteQuery('photo', '')

const settings = useLocalSettings()

const currentPath = computed(() => pathQuery.value || null)
const layout = computed(() => settings.layout.value)
const sortBy = computed(() => settings.sortBy.value)
const mediaType = computed({
  get: () => mediaQuery.value,
  set: (v: string) => { mediaQuery.value = v || null },
})

const includeAllFolders = computed({
  get: () => allFoldersQuery.value === 'true',
  set: (v: boolean) => { allFoldersQuery.value = v ? 'true' : '' },
})

const selectedIds = ref(new Set<string>())
const lastClickedIndex = ref(-1)
const selectionEnabled = ref(true)
const showDeleteConfirm = ref(false)

const upload = useUploadStore()
let refreshInterval: ReturnType<typeof setInterval> | null = null

const moveDialog = ref({ open: false })
const copyDialog = ref({ open: false })

const lightboxFile = computed(() => {
  if (!photoQuery.value) return null
  return files.value.find(f => f.id === photoQuery.value) ?? null
})

const lightboxIndex = computed(() => {
  if (!lightboxFile.value) return -1
  return files.value.indexOf(lightboxFile.value)
})

const fileViewerFile = ref<FileItem | null>(null)

async function loadFiles(reset = true) {
  const savedScroll = scrollRef.value?.scrollTop ?? 0
  if (reset) {
    files.value = []
    nextCursor.value = ''
    loading.value = true
  } else {
    loadingMore.value = true
  }
  try {
    const params: any = { sort: sortBy.value, order: 'desc', limit: 500 }
    if (mediaType.value) params.media_type = mediaType.value
    if (nextCursor.value) params.cursor = nextCursor.value
    if (route.query.date_from) params.date_from = route.query.date_from
    if (route.query.date_to) params.date_to = route.query.date_to
    if (currentPath.value) params.path = currentPath.value
    if (includeAllFolders.value) params.all_folders = 'true'
    const res = await api.get('/files', { params })
    files.value = reset ? res.data.items : [...files.value, ...res.data.items]
    total.value = res.data.total
    nextCursor.value = res.data.nextCursor || ''
    if (!reset) {
      await nextTick()
      if (scrollRef.value) scrollRef.value.scrollTop = savedScroll
    }
  } catch (e) {
    console.error('Failed to load files', e)
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function handleFileClick(index: number) {
  if (isShiftHeld()) {
    selectRange(index)
  } else {
    openLightbox(index)
  }
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

function clearSelection() {
  selectedIds.value = new Set()
}

function startMove() {
  moveDialog.value.open = true
}

function startCopy() {
  copyDialog.value.open = true
}

async function executeMove(folderId: string | null) {
  try {
    await api.post('/files/batch-move', {
      ids: Array.from(selectedIds.value),
      folder_id: folderId || null,
    })
    clearSelection()
    moveDialog.value.open = false
    loadFiles()
  } catch (e) {
    console.error('Failed to move files', e)
  }
}

async function executeCopy(folderId: string | null) {
  try {
    await api.post('/files/batch-copy', {
      ids: Array.from(selectedIds.value),
      folder_id: folderId || null,
    })
    clearSelection()
    copyDialog.value.open = false
    loadFiles()
  } catch (e) {
    console.error('Failed to copy files', e)
  }
}

async function executeDelete() {
  try {
    await api.post('/files/batch-delete', {
      ids: Array.from(selectedIds.value),
    })
    clearSelection()
    showDeleteConfirm.value = false
    loadFiles()
  } catch (e) {
    console.error('Failed to delete files', e)
  }
}

function isShiftHeld(): boolean {
  return typeof window !== 'undefined' && !!(window as any).__shiftHeld
}

function selectRange(index: number) {
  if (lastClickedIndex.value < 0) {
    toggleSelect(files.value[index].id)
    lastClickedIndex.value = index
    return
  }
  const start = Math.min(lastClickedIndex.value, index)
  const end = Math.max(lastClickedIndex.value, index)
  const newSet = new Set(selectedIds.value)
  for (let i = start; i <= end; i++) {
    newSet.add(files.value[i].id)
  }
  selectedIds.value = newSet
  lastClickedIndex.value = index
}

function openLightbox(index: number) {
  if (isShiftHeld()) {
    selectRange(index)
  } else {
    const file = files.value[index]
    if (file) {
      if (file.mediaType === 'file') {
        fileViewerFile.value = file
      } else {
        photoQuery.value = file.id
      }
    }
  }
}

function closeFileViewer() {
  fileViewerFile.value = null
}

function closeLightbox() {
  photoQuery.value = null
}

function goPrev() {
  if (lightboxIndex.value > 0) {
    photoQuery.value = files.value[lightboxIndex.value - 1].id
  }
}

function goNext() {
  if (lightboxIndex.value < files.value.length - 1) {
    photoQuery.value = files.value[lightboxIndex.value + 1].id
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('keydown', (e) => { (window as any).__shiftHeld = e.shiftKey })
  window.addEventListener('keyup', (e) => { (window as any).__shiftHeld = e.shiftKey })
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Delete' && selectedIds.value.size > 0 && document.activeElement?.tagName !== 'INPUT') {
      showDeleteConfirm.value = true
    }
  })
}

watch([() => route.query.path, () => route.query.media, () => route.query.all_folders], () => loadFiles(true), { immediate: false })
watch([mediaType, sortBy, includeAllFolders], () => loadFiles(true))

onMounted(() => {
  let observer: IntersectionObserver | null = null
  if (sentinel.value) {
    observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && nextCursor.value && !loadingMore.value) {
          loadFiles(false)
        }
      },
      { rootMargin: '200px' }
    )
    observer.observe(sentinel.value)
  }
  refreshInterval = setInterval(async () => {
    const completed = upload.consumeCompletedJobs()
    if (completed.length === 0) return
    const pathKey = currentPath.value ?? null
    const relevant = completed.filter(j => (j.folder_id ?? null) === pathKey)
    for (const job of relevant) {
      try {
        const res = await api.get(`/files/${job.file_id}`)
        const newFile = res.data as FileItem
        if (sortBy.value === 'taken_at' || sortBy.value === 'created_at') {
          files.value.unshift(newFile)
        } else {
          files.value.push(newFile)
        }
      } catch (e) {
        console.error('Failed to fetch new file', e)
      }
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

loadFiles()
</script>
