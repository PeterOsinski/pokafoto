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
        <Breadcrumbs :chain="folderChain" @navigate="(id) => emit('navigate', id)" />
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
        <div class="flex items-center gap-1 mt-1" @click.stop>
          <span class="text-xs cursor-pointer hover:scale-110" :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'" @click="openPasswordDialog(child.folder.id)">{{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}</span>
          <span title="Share" class="text-xs cursor-pointer hover:scale-110" @click="openShareDialog(child.folder.id, child.folder.name)">&#x1F517;</span>
        </div>
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
        <div class="flex items-center gap-1 mt-1" @click.stop>
          <span class="text-xs cursor-pointer hover:scale-110" :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'" @click="openPasswordDialog(child.folder.id)">{{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}</span>
          <span title="Share" class="text-xs cursor-pointer hover:scale-110" @click="openShareDialog(child.folder.id, child.folder.name)">&#x1F517;</span>
        </div>
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

    <FolderPasswordDialog
      :visible="passwordDialog.show"
      :folderId="passwordDialog.folderId"
      :mode="passwordDialog.mode"
      :hasPassword="passwordDialog.hasPassword"
      :expiresAt="passwordDialog.expiresAt"
      @close="passwordDialog.show = false"
      @unlocked="passwordDialog.show = false; loadPasswordStatuses()"
      @removed="passwordDialog.show = false; loadPasswordStatuses()"
    />

    <FolderShareDialog
      :visible="shareDialog.show"
      :folderId="shareDialog.folderId"
      :folderName="shareDialog.folderName"
      @close="shareDialog.show = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import api from '../api/client'
import ThumbnailCard from './ThumbnailCard.vue'
import Breadcrumbs from './Breadcrumbs.vue'
import InlineUpload from './InlineUpload.vue'
import FolderPasswordDialog from './FolderPasswordDialog.vue'
import FolderShareDialog from './FolderShareDialog.vue'
import { useChunkedUploadStore } from '../stores/chunkedUpload'
import { useFolderUnlockStore } from '../stores/folderUnlock'

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
  hasShares: boolean
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

const upload = useChunkedUploadStore()
let refreshInterval: ReturnType<typeof setInterval> | null = null

const unlockStore = useFolderUnlockStore()
const folderPasswordStatus = ref<Record<string, boolean>>({})

const passwordDialog = ref({
  show: false,
  folderId: '',
  mode: 'set' as 'set' | 'unlock' | 'status',
  hasPassword: false,
  expiresAt: '',
})

const shareDialog = ref({
  show: false,
  folderId: '',
  folderName: '',
})

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
  loadPasswordStatuses()
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

const currentFolderId = computed(() => props.folderId)

const folderChain = computed(() => {
  const chain: { id: string | null; name: string }[] = [{ id: null, name: 'Root' }]
  if (!props.folderId) return chain

  const buildPath = (nodes: FolderTreeNode[], target: string, path: { id: string | null; name: string }[]): boolean => {
    for (const n of nodes) {
      if (n.folder.id === target) {
        path.push({ id: n.folder.id, name: n.folder.name })
        return true
      }
      if (n.children?.length) {
        path.push({ id: n.folder.id, name: n.folder.name })
        if (buildPath(n.children, target, path)) return true
        path.pop()
      }
    }
    return false
  }

  buildPath(folders.value.children ?? [], props.folderId, chain)
  return chain
})

const subfolders = computed(() => {
  if (!props.folderId) return []
  const find = (nodes: FolderTreeNode[]): FolderTreeNode[] => {
    for (const n of nodes) {
      if (n.folder.id === props.folderId) return n.children ?? []
      const found = find(n.children ?? [])
      if (found.length) return found
    }
    return []
  }
  return find(folders.value.children ?? [])
})

watch(() => props.folderId, () => {
  loadFolders()
  loadFiles()
}, { immediate: true })

async function loadFolders() {
  try {
    const res = await api.get('/folders')
    folders.value = res.data
    await loadPasswordStatuses()
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
      if (n.children?.some(c => c.folder.id === targetId)) return n.folder.id
      for (const c of n.children ?? []) {
        const found = findParent([c], targetId)
        if (found !== undefined) return found
      }
    }
    return null
  }
  if (!props.folderId) return
  const parent = findParent(folders.value.children ?? [], props.folderId)
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
    await loadPasswordStatuses()
  } catch (e) {
    console.error('Failed to create folder', e)
  }
}

function openPasswordDialog(folderId: string) {
  const hasPass = folderPasswordStatus.value[folderId]
  const unlocked = unlockStore.isUnlocked(folderId)
  passwordDialog.value = {
    show: true,
    folderId,
    mode: hasPass ? (unlocked ? 'status' : 'unlock') : 'set',
    hasPassword: hasPass,
    expiresAt: '',
  }
}

function openShareDialog(folderId: string, folderName: string) {
  shareDialog.value = { show: true, folderId, folderName }
}

async function loadPasswordStatuses() {
  try {
    const allIds: string[] = []
    const collect = (nodes: FolderTreeNode[]) => {
      for (const n of nodes) {
        allIds.push(n.folder.id)
        collect(n.children ?? [])
      }
    }
    collect(folders.value.children ?? [])

    for (const id of allIds) {
      try {
        const res = await api.get(`/folders/${id}/password`)
        folderPasswordStatus.value[id] = res.data.has_password || false
      } catch {
        folderPasswordStatus.value[id] = false
      }
    }
  } catch {}
}

watch(showCreate, (v) => {
  if (v) nextTick(() => createInput.value?.focus())
})
</script>
