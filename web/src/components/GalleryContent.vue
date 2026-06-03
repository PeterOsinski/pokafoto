<template>
  <div>
    <ActionBar
      :count="selectedIds.size"
      :totalFiles="files.length"
      @delete="$emit('deleteFiles')"
      @move="$emit('moveFiles')"
      @copy="$emit('copyFiles')"
      @deselectAll="$emit('clearSelection')"
      @selectAll="$emit('selectAll')"
    />

    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <button
          v-if="currentFolderId"
          @click="$emit('navigateUp')"
          class="px-3 py-1 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-elevated)]"
        >
          &#8592; Back
        </button>
        <Breadcrumbs :chain="folderChain" @navigate="(id: string | null) => $emit('navigateTo', id)" />
      </div>
      <div class="flex items-center gap-2">
        <InlineUpload v-if="currentFolderId" :folderId="currentFolderId" label="Upload" />
        <div v-if="showNewDocInput !== undefined" class="relative">
          <button
            @click="$emit('toggleNewDoc')"
            class="px-3 py-1 rounded text-sm text-white"
            style="background: var(--accent-secondary, #8b5cf6)"
          >
            + New Document
          </button>
        </div>
        <button
          @click="$emit('showCreateInput')"
          class="px-3 py-1 rounded text-sm text-white"
          style="background: var(--accent)"
        >
          + New Folder
        </button>
      </div>
    </div>

    <div v-if="showCreateInput" class="flex items-center gap-2 mb-4">
      <input
        v-model="newFolderName"
        type="text"
        placeholder="Folder name..."
        class="flex-1 px-3 py-2 rounded text-sm"
        style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
        @keyup.enter="createAndReset"
        @keyup.escape="$emit('cancelCreateFolder')"
      />
      <button @click="createAndReset" :disabled="!newFolderName.trim()" class="px-3 py-2 rounded text-sm text-white" style="background: var(--accent)" :class="!newFolderName.trim() ? 'opacity-50 cursor-not-allowed' : ''">Create</button>
      <button @click="$emit('cancelCreateFolder')" class="px-3 py-2 rounded text-sm text-[var(--text-secondary)]">Cancel</button>
    </div>

    <GalleryControls
      :layout="layout"
      :sortBy="sortBy"
      :thumbLevel="thumbLevel"
      :previewMode="previewMode"
      @update:layout="(v: string) => $emit('update:layout', v)"
      @update:sortBy="(v: string) => $emit('update:sortBy', v)"
      @update:thumbLevel="(v: number) => $emit('update:thumbLevel', v)"
      @togglePreviewMode="$emit('togglePreviewMode')"
    />

    <div ref="tileContainer">
      <!-- TILE LAYOUT: unified CSS grid, folder tiles first, then file ThumbnailCards -->
      <div v-if="layout === 'tiles'" class="grid gap-2" :style="{ gridTemplateColumns: `repeat(${tileColumns}, 1fr)` }">
        <FolderTile
          v-for="f in folderTiles"
          :key="f.folder.id"
          :name="f.folder.name"
          :fileCount="f.fileCount"
          :hasShares="f.hasShares"
          :hasPassword="!!pmap[f.folder.id]"
          @click="$emit('navigateTo', f.folder.id)"
          @contextmenu="(e: MouseEvent) => $emit('folderContextMenu', e, f.folder)"
        />
        <ThumbnailCard
          v-for="(file, index) in files"
          :key="file.id"
          :file="file"
          :selected="selectedIds.has(file.id)"
          :selectable="true"
          :anySelected="selectedIds.size > 0"
          :thumbSize="effectiveThumbSize"
          @select="$emit('select', file.id)"
          @deselect="$emit('deselect', file.id)"
          @open="$emit('openFile', index)"
          @contextmenu="(e: MouseEvent) => $emit('fileContextMenu', e, file.id, file.originalName || file.filename)"
        />
      </div>

      <!-- LIST LAYOUT: folder rows then GalleryListView -->
      <div v-if="layout === 'list'">
        <div class="flex items-center border-b border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-xs font-semibold uppercase tracking-wide select-none px-3">
          <span class="w-10 shrink-0" />
          <span class="flex-1 min-w-0 py-2 px-3">Name</span>
          <span class="py-2 px-3 hidden sm:block whitespace-nowrap shrink-0 mr-4">Created</span>
          <span class="py-2 px-3 shrink-0">Files</span>
        </div>
        <button
          v-for="f in folderTiles"
          :key="f.folder.id"
          @click="$emit('navigateTo', f.folder.id)"
          @contextmenu.prevent="(e: MouseEvent) => $emit('folderContextMenu', e, f.folder)"
          class="flex items-center w-full border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors px-3"
          style="height: 52px"
        >
          <span class="text-xl w-10 shrink-0">&#128193;</span>
          <span class="flex-1 min-w-0 text-sm text-[var(--text-primary)] font-medium truncate text-left">
            <span v-if="!!pmap[f.folder.id]" class="mr-1" title="Password protected">&#x1F512;</span>
            <span v-if="f.hasShares" class="mr-1" title="Shared">&#x1F517;</span>{{ f.folder.name }}
          </span>
          <span class="text-xs text-[var(--text-secondary)] shrink-0 hidden sm:block mr-4">{{ formatDate(f.folder.created_at) }}</span>
          <span class="text-xs text-[var(--text-secondary)] shrink-0 mr-2">{{ f.fileCount }} {{ f.fileCount === 1 ? 'file' : 'files' }}</span>
        </button>
        <GalleryListView
          :files="files"
          :thumbSizePx="thumbSizePx"
          :selectedIds="selectedIds"
          :selectionEnabled="true"
          @select="(id: string) => $emit('select', id)"
          @deselect="(id: string) => $emit('deselect', id)"
          @open="(i: number) => $emit('openFile', i)"
          @contextmenu="(e: MouseEvent, fileId: string, fileName: string) => $emit('fileContextMenu', e, fileId, fileName)"
        />
      </div>

      <!-- GROUPED LAYOUT: GalleryGroupedView -->
      <GalleryGroupedView
        v-if="layout === 'grouped'"
        :files="files"
        :thumbSizePx="thumbSizePx"
        :selectedIds="selectedIds"
        :selectionEnabled="true"
        @select="(id: string) => $emit('select', id)"
        @deselect="(id: string) => $emit('deselect', id)"
        @open="(i: number) => $emit('openFile', i)"
        @contextmenu="(e: MouseEvent, fileId: string, fileName: string) => $emit('fileContextMenu', e, fileId, fileName)"
      />
    </div>

    <div v-if="files.length === 0 && !loading && folderTiles.length === 0" class="text-center py-8 text-[var(--text-secondary)]">
      <p v-if="currentFolderId">No files in this folder.</p>
      <p v-else>No contents to show.</p>
    </div>

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
    <div v-else-if="loadingMore" class="text-center py-4 text-[var(--text-secondary)]">Loading more...</div>
    <div ref="sentinel" class="h-4"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import ActionBar from './ActionBar.vue'
import Breadcrumbs from './Breadcrumbs.vue'
import GalleryControls from './GalleryControls.vue'
import GalleryGroupedView from './GalleryGroupedView.vue'
import GalleryListView from './GalleryListView.vue'
import InlineUpload from './InlineUpload.vue'
import ThumbnailCard from './ThumbnailCard.vue'
import FolderTile from './FolderTile.vue'
import { useGalleryColumns } from '../composables/useGalleryColumns'
import type { FileItem, FolderTreeNode, LayoutMode, SortBy } from '../types/gallery'

const props = defineProps<{
  files: FileItem[]
  folderTiles: FolderTreeNode[]
  currentFolderId: string | null
  folderChain: { id: string | null; name: string }[]
  selectedIds: Set<string>
  layout: LayoutMode
  sortBy: SortBy
  thumbLevel: number
  thumbSizePx: number
  previewMode: string
  loading: boolean
  loadingMore: boolean
  showCreateInput: boolean
  showNewDocInput?: boolean
  passwordStatuses?: Record<string, boolean>
}>()

const pmap = computed(() => props.passwordStatuses || ({} as Record<string, boolean>))

const emit = defineEmits<{
  navigateTo: [id: string | null]
  navigateUp: []
  createFolder: [name: string]
  cancelCreateFolder: []
  showCreateInput: []
  deleteFiles: []
  moveFiles: []
  copyFiles: []
  clearSelection: []
  selectAll: []
  select: [id: string]
  deselect: [id: string]
  openFile: [index: number]
  fileContextMenu: [e: MouseEvent, fileId: string, fileName: string]
  folderContextMenu: [e: MouseEvent, folder: FolderTreeNode['folder']]
  'update:layout': [v: string]
  'update:sortBy': [v: string]
  'update:thumbLevel': [v: number]
  togglePreviewMode: []
  toggleNewDoc: []
}>()

const tileContainer = ref<HTMLElement | null>(null)
const sentinel = ref<HTMLElement | null>(null)
defineExpose({ sentinel })

const newFolderName = ref('')

const { columns: tileColumns } = useGalleryColumns(tileContainer, computed(() => props.thumbSizePx))

const effectiveThumbSize = computed<'sm' | 'md' | 'lg'>(() => {
  if (props.thumbSizePx <= 100) return 'sm'
  if (props.thumbSizePx <= 250) return 'lg'
  return 'md'
})

function createAndReset() {
  if (!newFolderName.value.trim()) return
  emit('createFolder', newFolderName.value.trim())
  newFolderName.value = ''
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const dd = d.getDate().toString().padStart(2, '0')
  const mm = (d.getMonth() + 1).toString().padStart(2, '0')
  const yy = d.getFullYear().toString().slice(-2)
  const hh = d.getHours().toString().padStart(2, '0')
  const min = d.getMinutes().toString().padStart(2, '0')
  const ss = d.getSeconds().toString().padStart(2, '0')
  return `${dd}/${mm}/${yy} ${hh}:${min}:${ss}`
}
</script>
