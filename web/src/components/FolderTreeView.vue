<template>
  <div>
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

    <div v-if="!folders.children?.length && !currentFolderId" class="text-center py-10 text-[var(--text-secondary)]">
      <p class="text-lg">No folders yet.</p>
      <p class="mt-1 text-sm">Create a folder to start organizing your files.</p>
    </div>

    <div v-if="currentFolderId || (!currentFolderId && folders.children?.length === 0)" class="mt-4">
      <div v-if="files.length === 0 && !loading" class="text-center py-8 text-[var(--text-secondary)]">
        <p>No files in this folder.</p>
      </div>

      <ThumbnailCard
        v-for="(file, i) in files"
        :key="file.id"
        :file="file"
        :thumbSize="thumbSize"
        :selected="selectedIds.has(file.id)"
        :selectable="selectionEnabled"
        :anySelected="selectedIds.size > 0"
        @select="$emit('select', file.id)"
        @deselect="$emit('deselect', file.id)"
        @open="$emit('open', i)"
      />
    </div>

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import api from '../api/client'
import ThumbnailCard from './ThumbnailCard.vue'
import InlineUpload from './InlineUpload.vue'
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

const props = defineProps<{
  folderId: string | null
  selectedIds: Set<string>
  selectionEnabled: boolean
  thumbSize?: 'sm' | 'md' | 'lg'
}>()

const emit = defineEmits<{
  navigate: [folderId: string | null]
  select: [id: string]
  deselect: [id: string]
  open: [index: number]
}>()

const folders = ref<RootNode>({ children: [] })
const files = ref<FileItem[]>([])
const loading = ref(false)
const showCreate = ref(false)
const newFolderName = ref('')
const createInput = ref<HTMLInputElement | null>(null)

const upload = useUploadStore()
let refreshInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  refreshInterval = setInterval(() => {
    const completed = upload.consumeCompletedJobs()
    if (completed.length === 0) return
    const folderKey = props.folderId ?? null
    const relevant = completed.filter(j => (j.folder_id ?? null) === folderKey)
    if (relevant.length > 0) {
      loadFiles()
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

const currentFolderId = computed(() => props.folderId)
const currentFolderName = computed(() => {
  if (!props.folderId) return ''
  const find = (nodes: FolderTreeNode[]): string | null => {
    for (const n of nodes) {
      if (n.folder.id === props.folderId) return n.folder.name
      const found = find(n.children)
      if (found) return found
    }
    return null
  }
  return find(folders.value.children)
})

const subfolders = computed(() => {
  if (!props.folderId) return []
  const find = (nodes: FolderTreeNode[]): FolderTreeNode[] => {
    for (const n of nodes) {
      if (n.folder.id === props.folderId) return n.children
      const found = find(n.children)
      if (found) return found
    }
    return []
  }
  return find(folders.value.children)
})

watch(() => props.folderId, () => {
  loadFolders()
  loadFiles()
}, { immediate: true })

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
    const params: any = { sort: 'taken_at', order: 'desc', limit: 100 }
    if (props.folderId) {
      params.folder_id = props.folderId
    }
    const res = await api.get('/files', { params })
    files.value = res.data.items
  } catch (e) {
    console.error('Failed to load files', e)
  } finally {
    loading.value = false
  }
}

function navigateTo(id: string) {
  emit('navigate', id)
}

function navigateUp() {
  const findParent = (nodes: FolderTreeNode[], targetId: string): string | null => {
    for (const n of nodes) {
      if (n.children.some(c => c.folder.id === targetId)) return n.folder.id
      for (const c of n.children) {
        const found = findParent([c], targetId)
        if (found !== undefined) return found
      }
    }
    return null
  }
  if (!props.folderId) return
  const parent = findParent(folders.value.children, props.folderId)
  emit('navigate', parent)
}

async function createFolder() {
  if (!newFolderName.value.trim()) return
  try {
    await api.post('/folders', {
      name: newFolderName.value.trim(),
      parent_id: props.folderId,
    })
    newFolderName.value = ''
    showCreate.value = false
    await loadFolders()
  } catch (e) {
    console.error('Failed to create folder', e)
  }
}

watch(showCreate, (v) => {
  if (v) nextTick(() => createInput.value?.focus())
})
</script>
