<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    @click.self="$emit('close')"
  >
    <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-md mx-4" style="border: 1px solid var(--border-color)">
      <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-4">{{ title }}</h3>

      <div class="mb-4 max-h-60 overflow-y-auto">
        <button
          @click="selectFolder(null)"
          class="flex items-center w-full text-left text-sm py-2 px-3 rounded hover:bg-[var(--bg-elevated)]"
          :class="selectedFolderId === null ? 'text-[var(--accent)] bg-[var(--accent)]/10' : 'text-[var(--text-primary)]'"
        >
          <span class="mr-2">&#128194;</span>
          <span>Root (no folder)</span>
        </button>
        <FolderNode
          v-for="child in root.children"
          :key="child.folder.id"
          :node="child"
          :selectedId="selectedFolderId"
          :depth="0"
          @select="selectFolder"
        />
        <p v-if="!root.children?.length" class="text-sm text-[var(--text-secondary)] py-4 text-center">No folders yet. Create one below.</p>
      </div>

      <div class="flex items-center gap-2 mb-4">
        <input
          v-model="newFolderName"
          type="text"
          placeholder="New folder name..."
          class="flex-1 px-3 py-2 rounded text-sm"
          style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
          @keyup.enter="createFolder"
        />
        <button
          @click="createFolder"
          :disabled="!newFolderName.trim()"
          class="px-3 py-2 rounded text-sm text-white"
          style="background: var(--accent)"
          :class="!newFolderName.trim() ? 'opacity-50 cursor-not-allowed' : ''"
        >
          Create
        </button>
      </div>

      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">
          Cancel
        </button>
        <button
          @click="confirm"
          class="px-4 py-2 rounded text-sm text-white"
          style="background: var(--accent)"
        >
          {{ actionLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api/client'
import FolderNode from './FolderNode.vue'

const props = defineProps<{
  open: boolean
  title: string
  actionLabel: string
}>()

const emit = defineEmits<{
  close: []
  confirm: [folderId: string | null]
}>()

interface FolderEntry {
  id: string
  name: string
  parent_id: string | null
  user_id: string
  created_at: string
  updated_at: string
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

const root = ref<RootNode>({ children: [] })
const selectedFolderId = ref<string | null>(null)
const newFolderName = ref('')

onMounted(() => loadFolders())

async function loadFolders() {
  try {
    const res = await api.get('/folders')
    root.value = res.data
  } catch (e) {
    console.error('Failed to load folders', e)
  }
}

function selectFolder(id: string | null) {
  selectedFolderId.value = id
}

async function createFolder() {
  if (!newFolderName.value.trim()) return
  try {
    await api.post('/folders', { name: newFolderName.value.trim(), parent_id: selectedFolderId.value })
    newFolderName.value = ''
    await loadFolders()
  } catch (e) {
    console.error('Failed to create folder', e)
  }
}

function confirm() {
  emit('confirm', selectedFolderId.value)
}
</script>
