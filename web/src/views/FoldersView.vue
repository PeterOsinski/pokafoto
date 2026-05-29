<template>
  <div>
    <ActionBar
      :count="selectedIds.size"
      @delete="showDeleteConfirm = true"
      @move="startMove"
      @copy="startCopy"
      @deselectAll="clearSelection"
    />

    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <button
          v-if="currentFolderId"
          @click="navigateUp"
          class="px-3 py-1 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-elevated)]"
        >
          &#8592; Back
        </button>
        <h3 class="text-lg font-semibold text-[var(--text-primary)]">
          {{ currentFolderName || 'Root' }}
        </h3>
      </div>
      <div class="flex items-center gap-2">
        <InlineUpload :folderId="currentFolderId" label="Upload" />
        <button
          @click="showCreate = true"
          class="px-3 py-1 rounded text-sm text-white"
          style="background: var(--accent)"
        >
          + New Folder
        </button>
      </div>
    </div>

    <div v-if="showCreate" class="flex items-center gap-2 mb-4">
      <input
        ref="createInput"
        v-model="newFolderName"
        type="text"
        placeholder="Folder name..."
        class="flex-1 px-3 py-2 rounded text-sm"
        style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
        @keyup.enter="createFolder"
        @keyup.escape="showCreate = false"
      />
      <button @click="createFolder" :disabled="!newFolderName.trim()" class="px-3 py-2 rounded text-sm text-white" style="background: var(--accent)" :class="!newFolderName.trim() ? 'opacity-50 cursor-not-allowed' : ''">Create</button>
      <button @click="showCreate = false" class="px-3 py-2 rounded text-sm text-[var(--text-secondary)]">Cancel</button>
    </div>

    <div class="flex items-center gap-2 mb-6 flex-wrap">
      <select :modelValue="sortBy" @change="sortBy = ($event.target as HTMLSelectElement).value" class="px-3 py-1 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)">
        <option value="taken_at">Date Taken</option>
        <option value="created_at">Date Uploaded</option>
        <option value="filename">File Name</option>
      </select>

      <div class="flex items-center gap-1 rounded-lg p-1" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
        <button
          v-for="opt in layoutOptions"
          :key="opt.value"
          :title="opt.label"
          class="p-1.5 rounded-md transition-colors"
          :class="opt.value === layout ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'"
          @click="layout = opt.value"
          v-html="opt.icon"
        />
      </div>

      <div class="flex items-center gap-1 rounded-lg p-1" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
        <button
          v-for="opt in sizeOptions"
          :key="opt.value"
          :title="opt.label"
          class="p-1.5 rounded-md transition-colors"
          :class="opt.value === thumbSize ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'"
          @click="thumbSize = opt.value as 'sm' | 'md' | 'lg'"
          v-html="opt.icon"
        />
      </div>
    </div>

    <div v-if="!currentFolderId && folders.children?.length" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
      <button
        v-for="child in folders.children"
        :key="child.folder.id"
        @click="navigateTo(child.folder.id)"
        class="flex flex-col items-center gap-2 p-4 rounded-lg text-center transition-colors hover:bg-[var(--bg-elevated)]"
        style="border: 1px solid var(--border-color)"
      >
        <span class="text-3xl">&#128193;</span>
        <span class="text-sm text-[var(--text-primary)] font-medium truncate w-full">{{ child.folder.name }}</span>
        <span class="text-xs text-[var(--text-secondary)]">{{ child.fileCount }} {{ child.fileCount === 1 ? 'file' : 'files' }}</span>
      </button>
    </div>

    <div v-if="currentFolderId && subfolders.length > 0" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
      <button
        v-for="child in subfolders"
        :key="child.folder.id"
        @click="navigateTo(child.folder.id)"
        class="flex flex-col items-center gap-2 p-4 rounded-lg text-center transition-colors hover:bg-[var(--bg-elevated)]"
        style="border: 1px solid var(--border-color)"
      >
        <span class="text-3xl">&#128193;</span>
        <span class="text-sm text-[var(--text-primary)] font-medium truncate w-full">{{ child.folder.name }}</span>
        <span class="text-xs text-[var(--text-secondary)]">{{ child.fileCount }} {{ child.fileCount === 1 ? 'file' : 'files' }}</span>
      </button>
    </div>

    <div v-if="!folders.children?.length && !currentFolderId && !loading" class="text-center py-10 text-[var(--text-secondary)]">
      <p class="text-lg">No folders yet.</p>
      <p class="mt-1 text-sm">Create a folder to start organizing your files.</p>
    </div>

    <div v-if="currentFolderId || (!currentFolderId && folders.children?.length === 0)">
      <div v-if="files.length === 0 && !loading" class="text-center py-8 text-[var(--text-secondary)]">
        <p>No files in this folder.</p>
      </div>

      <GalleryTileView
        v-else-if="layout === 'tiles'"
        :files="files"
        :thumbSize="thumbSize"
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
      <GalleryTableView
        v-else-if="layout === 'table'"
        :files="files"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
        @download="handleDownload"
        @delete="handleSingleDelete"
      />
    </div>

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>

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
        <p class="text-sm text-[var(--text-secondary)] mb-4">{{ deleteMessage }}</p>
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
import Lightbox from '../components/Lightbox.vue'
import FileViewer from '../components/FileViewer.vue'
import GalleryTileView from '../components/GalleryTileView.vue'
import GalleryListView from '../components/GalleryListView.vue'
import GalleryTableView from '../components/GalleryTableView.vue'
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
  thumbnails?: any
}

interface FolderEntry {
  id: string
  name: string
  parent_id: string | null
}

interface FolderTreeNode {
  folder: FolderEntry
  fileCount: number
  children: FolderTreeNode[]
}

interface RootNode {
  children: FolderTreeNode[]
}

const route = useRoute()

const folders = ref<RootNode>({ children: [] })
const files = ref<FileItem[]>([])
const loading = ref(false)
const showCreate = ref(false)
const newFolderName = ref('')
const createInput = ref<HTMLInputElement | null>(null)

const folderIdQuery = useRouteQuery('folder_id', '')
const layoutQuery = useRouteQuery('layout', 'tiles')
const sortQuery = useRouteQuery('sort', 'taken_at')
const thumbQuery = useRouteQuery('thumb', 'md')
const photoQuery = useRouteQuery('photo', '')

const currentFolderId = computed(() => folderIdQuery.value || null)

const currentFolderName = computed(() => {
  if (!currentFolderId.value) return ''
  const find = (nodes: FolderTreeNode[]): string | null => {
    for (const n of nodes) {
      if (n.folder.id === currentFolderId.value) return n.folder.name
      const found = find(n.children ?? [])
      if (found) return found
    }
    return null
  }
  return find(folders.value.children ?? [])
})

const subfolders = computed(() => {
  if (!currentFolderId.value) return []
  const find = (nodes: FolderTreeNode[]): FolderTreeNode[] => {
    for (const n of nodes) {
      if (n.folder.id === currentFolderId.value) return n.children ?? []
      const found = find(n.children ?? [])
      if (found.length) return found
    }
    return []
  }
  return find(folders.value.children ?? [])
})

const layout = computed({
  get: () => layoutQuery.value,
  set: (v: string) => { layoutQuery.value = v === 'tiles' ? '' : v },
})

const sortBy = computed({
  get: () => sortQuery.value,
  set: (v: string) => { sortQuery.value = v === 'taken_at' ? '' : v },
})

const thumbSize = computed({
  get: () => (thumbQuery.value || 'md') as 'sm' | 'md' | 'lg',
  set: (v: 'sm' | 'md' | 'lg') => { thumbQuery.value = v === 'md' ? '' : v },
})

const selectedIds = ref(new Set<string>())
const lastClickedIndex = ref(-1)
const selectionEnabled = ref(true)
const showDeleteConfirm = ref(false)
const pendingSingleDeleteId = ref<string | null>(null)

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

const deleteMessage = computed(() => {
  if (pendingSingleDeleteId.value) return 'Delete this file? It will be moved to trash.'
  return `This will move ${selectedIds.value.size} ${selectedIds.value.size === 1 ? 'file' : 'files'} to trash. You can recover them later.`
})

async function loadFolders() {
  try {
    const res = await api.get('/folders')
    folders.value = res.data
  } catch (e) {
    console.error('Failed to load folders', e)
  }
}

async function loadFiles() {
  loading.value = true
  try {
    const params: any = { sort: sortBy.value, order: 'desc', limit: 100 }
    if (currentFolderId.value) {
      params.folder_id = currentFolderId.value
    }
    const res = await api.get('/files', { params })
    files.value = res.data.items
  } catch (e) {
    console.error('Failed to load files', e)
  } finally {
    loading.value = false
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

async function executeMove(targetFolderId: string | null) {
  try {
    await api.post('/files/batch-move', {
      ids: Array.from(selectedIds.value),
      folder_id: targetFolderId || null,
    })
    clearSelection()
    moveDialog.value.open = false
    loadFolders()
    loadFiles()
  } catch (e) {
    console.error('Failed to move files', e)
  }
}

async function executeCopy(targetFolderId: string | null) {
  try {
    await api.post('/files/batch-copy', {
      ids: Array.from(selectedIds.value),
      folder_id: targetFolderId || null,
    })
    clearSelection()
    copyDialog.value.open = false
    loadFolders()
    loadFiles()
  } catch (e) {
    console.error('Failed to copy files', e)
  }
}

async function executeDelete() {
  try {
    const ids = pendingSingleDeleteId.value
      ? [pendingSingleDeleteId.value]
      : Array.from(selectedIds.value)
    await api.post('/files/batch-delete', { ids })
    clearSelection()
    pendingSingleDeleteId.value = null
    showDeleteConfirm.value = false
    loadFolders()
    loadFiles()
  } catch (e) {
    console.error('Failed to delete files', e)
  }
}

function handleDownload(fileId: string) {
  window.open(`/api/v1/download/${fileId}`, '_blank')
}

function handleSingleDelete(fileId: string) {
  pendingSingleDeleteId.value = fileId
  showDeleteConfirm.value = true
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

function navigateTo(id: string) {
  folderIdQuery.value = id
  selectedIds.value = new Set()
}

function navigateUp() {
  if (!currentFolderId.value) return
  const parentId = findParent(folders.value.children ?? [], currentFolderId.value)
  folderIdQuery.value = parentId ?? null
  selectedIds.value = new Set()
}

function findParent(nodes: FolderTreeNode[], targetId: string): string | null {
  for (const n of nodes) {
    if (n.children?.some(c => c.folder.id === targetId)) return n.folder.id
    for (const c of n.children ?? []) {
      const found = findParent([c], targetId)
      if (found !== null) return found
    }
  }
  return null
}

async function createFolder() {
  if (!newFolderName.value.trim()) return
  try {
    await api.post('/folders', {
      name: newFolderName.value.trim(),
      parent_id: currentFolderId.value,
    })
    newFolderName.value = ''
    showCreate.value = false
    await loadFolders()
  } catch (e) {
    console.error('Failed to create folder', e)
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

watch(() => route.query, () => {
  loadFolders()
  loadFiles()
}, { immediate: false })

watch(showCreate, (v) => {
  if (v) nextTick(() => createInput.value?.focus())
})

onMounted(() => {
  loadFolders()
  loadFiles()

  refreshInterval = setInterval(() => {
    const completed = upload.consumeCompletedJobs()
    if (completed.length === 0) return
    const folderKey = currentFolderId.value ?? null
    const relevant = completed.filter(j => (j.folder_id ?? null) === folderKey)
    if (relevant.length > 0) {
      loadFiles()
      loadFolders()
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

const layoutOptions = [
  { value: 'tiles', label: 'Tiles', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>' },
  { value: 'list', label: 'List', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>' },
  { value: 'table', label: 'Table', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="1"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="3" x2="9" y2="21"/></svg>' },
]

const sizeOptions = [
  { value: 'sm', label: 'Small', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="5" height="5" rx=".5"/><rect x="10" y="3" width="5" height="5" rx=".5"/><rect x="17" y="3" width="5" height="5" rx=".5"/><rect x="3" y="10" width="5" height="5" rx=".5"/><rect x="10" y="10" width="5" height="5" rx=".5"/><rect x="17" y="10" width="5" height="5" rx=".5"/></svg>' },
  { value: 'md', label: 'Medium', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="8" height="8" rx="1"/><rect x="13" y="3" width="8" height="8" rx="1"/><rect x="3" y="13" width="8" height="8" rx="1"/><rect x="13" y="13" width="8" height="8" rx="1"/></svg>' },
  { value: 'lg', label: 'Large', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="8" rx="1"/><rect x="3" y="13" width="18" height="8" rx="1"/></svg>' },
]
</script>
