<template>
  <DropZone :folderId="currentFolderId">
    <div v-if="settings.previewMode.value === 'sidebar' && lightboxFile" :class="{ 'flex': true }">
      <div class="flex-1 min-w-0">

    <ActionBar
      :count="selectedIds.size"
      :totalFiles="files.length"
      @delete="showDeleteConfirm = true"
      @move="startMove"
      @copy="startCopy"
      @deselectAll="clearSelection"
      @selectAll="selectAllFiles"
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
        <Breadcrumbs :chain="folderChain" @navigate="navigateTo" />
      </div>
      <div class="flex items-center gap-2">
        <InlineUpload v-if="currentFolderId" :folderId="currentFolderId" label="Upload" />
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

    <GalleryControls
      :layout="settings.layout.value"
      :sortBy="settings.sortBy.value"
      :thumbLevel="settings.thumbLevel.value"
      :previewMode="settings.previewMode.value"
      @update:layout="v => settings.layout.value = v"
      @update:sortBy="v => settings.sortBy.value = v"
      @update:thumbLevel="v => settings.thumbLevel.value = v"
      @togglePreviewMode="togglePreviewMode"
    />

    <div v-if="!currentFolderId && folders.children?.length && settings.layout.value === 'tiles'" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
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
          <button
            :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'"
            class="text-xs hover:scale-110 transition-transform"
            @click="openPasswordDialog(child.folder.id)"
          >
            {{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}
          </button>
          <button
            title="Share"
            class="text-xs hover:scale-110 transition-transform"
            @click="openShareDialog(child.folder.id, child.folder.name)"
          >
            &#x1F517;
          </button>
        </div>
      </button>
    </div>

    <div v-if="currentFolderId && subfolders.length > 0 && settings.layout.value === 'tiles'" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
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
          <button
            :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'"
            class="text-xs hover:scale-110 transition-transform"
            @click="openPasswordDialog(child.folder.id)"
          >
            {{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}
          </button>
          <button
            title="Share"
            class="text-xs hover:scale-110 transition-transform"
            @click="openShareDialog(child.folder.id, child.folder.name)"
          >
            &#x1F517;
          </button>
        </div>
      </button>
    </div>

    <div v-if="!folders.children?.length && !currentFolderId && !loading && settings.layout.value === 'tiles'" class="text-center py-10 text-[var(--text-secondary)]">
      <p class="text-lg">No folders yet.</p>
      <p class="mt-1 text-sm">Create a folder to start organizing your files.</p>
    </div>

    <div v-if="currentFolderId || settings.layout.value !== 'tiles' || (!currentFolderId && folders.children?.length === 0)">
      <div v-if="files.length === 0 && !loading" class="text-center py-8 text-[var(--text-secondary)]">
        <p v-if="currentFolderId">No files in this folder.</p>
        <p v-else-if="!folders.children?.length">No files in this folder.</p>
        <p v-else>No files to show.</p>
      </div>

      <div v-if="settings.layout.value === 'list'" class="mb-4">
        <div class="flex items-center border-b border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-xs font-semibold uppercase tracking-wide select-none px-3">
          <span class="w-10 shrink-0" />
          <span class="flex-1 min-w-0 py-2 px-3">Name</span>
          <span class="py-2 px-3 hidden sm:block whitespace-nowrap shrink-0 mr-4">Created</span>
          <span class="py-2 px-3 shrink-0">Files</span>
        </div>
        <button
          v-for="child in listFolders"
          :key="child.folder.id"
          @click="navigateTo(child.folder.id)"
          class="flex items-center w-full border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors px-3"
          style="height: 52px"
        >
          <span class="text-xl w-10 shrink-0">&#128193;</span>
          <span class="flex-1 min-w-0 text-sm text-[var(--text-primary)] font-medium truncate text-left">{{ child.folder.name }}</span>
          <span class="text-xs text-[var(--text-secondary)] shrink-0 hidden sm:block mr-4">{{ formatFolderDate(child.folder.created_at) }}</span>
          <span class="text-xs text-[var(--text-secondary)] shrink-0 mr-2">{{ child.fileCount }} {{ child.fileCount === 1 ? 'file' : 'files' }}</span>
          <span class="text-xs shrink-0 flex items-center gap-1" @click.stop>
            <span class="cursor-pointer hover:scale-110" :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'" @click="openPasswordDialog(child.folder.id)">{{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}</span>
            <span class="cursor-pointer hover:scale-110" title="Share" @click="openShareDialog(child.folder.id, child.folder.name)">&#x1F517;</span>
          </span>
        </button>
      </div>

      <GalleryTileView
        v-if="settings.layout.value === 'tiles'"
        :files="files"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
      />
      <GalleryListView
        v-else-if="settings.layout.value === 'list'"
        :files="files"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
      />
      <GalleryGroupedView
        v-else-if="settings.layout.value === 'grouped'"
        :files="files"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
      />
    </div>

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
    <div v-else-if="loadingMore" class="text-center py-4 text-[var(--text-secondary)]">Loading more...</div>
    <div ref="sentinel" class="h-4"></div>

      </div>

      <PreviewSidebar
        :file="lightboxFile"
        :index="lightboxIndex"
        :total="files.length"
        :hasPrev="lightboxIndex > 0"
        :hasNext="lightboxIndex < files.length - 1"
        @close="closeLightbox"
        @prev="goPrev"
        @next="goNext"
      />
    </div>

    <template v-else>
      <ActionBar
        :count="selectedIds.size"
        :totalFiles="files.length"
        @delete="showDeleteConfirm = true"
        @move="startMove"
        @copy="startCopy"
        @deselectAll="clearSelection"
        @selectAll="selectAllFiles"
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
        <Breadcrumbs :chain="folderChain" @navigate="navigateTo" />
      </div>
      <div class="flex items-center gap-2">
        <InlineUpload v-if="currentFolderId" :folderId="currentFolderId" label="Upload" />
        <div class="relative">
          <button
            @click="showNewDocInput = !showNewDocInput"
            class="px-3 py-1 rounded text-sm text-white"
            style="background: var(--accent-secondary, #8b5cf6)"
          >
            + New Document
          </button>
          <div
            v-if="showNewDocInput"
            class="absolute top-full left-0 mt-1 p-2 rounded shadow-lg z-30 flex gap-2"
            style="background: var(--bg-surface); border: 1px solid var(--border-color)"
          >
            <input
              v-model="newDocName"
              placeholder="Document name"
              class="px-2 py-1 rounded text-sm bg-black/30 text-white border border-white/10 outline-none focus:border-[var(--accent)]"
              @keydown.enter="createDocument"
              @keydown.escape="showNewDocInput = false"
            />
            <button
              @click="createDocument"
              :disabled="!newDocName.trim() || creatingDoc"
              class="px-3 py-1 rounded text-sm text-white disabled:opacity-50"
              style="background: var(--accent)"
            >
              Create
            </button>
          </div>
        </div>
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

    <GalleryControls
      :layout="settings.layout.value"
      :sortBy="settings.sortBy.value"
      :thumbLevel="settings.thumbLevel.value"
      :previewMode="settings.previewMode.value"
      @update:layout="v => settings.layout.value = v"
      @update:sortBy="v => settings.sortBy.value = v"
      @update:thumbLevel="v => settings.thumbLevel.value = v"
      @togglePreviewMode="togglePreviewMode"
    />

    <div v-if="!currentFolderId && folders.children?.length && settings.layout.value === 'tiles'" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
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
          <button
            :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'"
            class="text-xs hover:scale-110 transition-transform"
            @click="openPasswordDialog(child.folder.id)"
          >
            {{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}
          </button>
          <button
            title="Share"
            class="text-xs hover:scale-110 transition-transform"
            @click="openShareDialog(child.folder.id, child.folder.name)"
          >
            &#x1F517;
          </button>
        </div>
      </button>
    </div>

    <div v-if="currentFolderId && subfolders.length > 0 && settings.layout.value === 'tiles'" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
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
          <button
            :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'"
            class="text-xs hover:scale-110 transition-transform"
            @click="openPasswordDialog(child.folder.id)"
          >
            {{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}
          </button>
          <button
            title="Share"
            class="text-xs hover:scale-110 transition-transform"
            @click="openShareDialog(child.folder.id, child.folder.name)"
          >
            &#x1F517;
          </button>
        </div>
      </button>
    </div>

    <div v-if="!folders.children?.length && !currentFolderId && !loading && settings.layout.value === 'tiles'" class="text-center py-10 text-[var(--text-secondary)]">
      <p class="text-lg">No folders yet.</p>
      <p class="mt-1 text-sm">Create a folder to start organizing your files.</p>
    </div>

    <div v-if="currentFolderId || settings.layout.value !== 'tiles' || (!currentFolderId && folders.children?.length === 0)">
      <div v-if="files.length === 0 && !loading" class="text-center py-8 text-[var(--text-secondary)]">
        <p v-if="currentFolderId">No files in this folder.</p>
        <p v-else-if="!folders.children?.length">No files in this folder.</p>
        <p v-else>No files to show.</p>
      </div>

      <div v-if="settings.layout.value === 'list'" class="mb-4">
        <div class="flex items-center border-b border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-xs font-semibold uppercase tracking-wide select-none px-3">
          <span class="w-10 shrink-0" />
          <span class="flex-1 min-w-0 py-2 px-3">Name</span>
          <span class="py-2 px-3 hidden sm:block whitespace-nowrap shrink-0 mr-4">Created</span>
          <span class="py-2 px-3 shrink-0">Files</span>
        </div>
        <button
          v-for="child in listFolders"
          :key="child.folder.id"
          @click="navigateTo(child.folder.id)"
          class="flex items-center w-full border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors px-3"
          style="height: 52px"
        >
          <span class="text-xl w-10 shrink-0">&#128193;</span>
          <span class="flex-1 min-w-0 text-sm text-[var(--text-primary)] font-medium truncate text-left">{{ child.folder.name }}</span>
          <span class="text-xs text-[var(--text-secondary)] shrink-0 hidden sm:block mr-4">{{ formatFolderDate(child.folder.created_at) }}</span>
          <span class="text-xs text-[var(--text-secondary)] shrink-0 mr-2">{{ child.fileCount }} {{ child.fileCount === 1 ? 'file' : 'files' }}</span>
          <span class="text-xs shrink-0 flex items-center gap-1" @click.stop>
            <span class="cursor-pointer hover:scale-110" :title="folderPasswordStatus[child.folder.id] ? 'Password protected (click to manage)' : 'Set password'" @click="openPasswordDialog(child.folder.id)">{{ folderPasswordStatus[child.folder.id] ? '&#x1F512;' : '&#x1F513;' }}</span>
            <span class="cursor-pointer hover:scale-110" title="Share" @click="openShareDialog(child.folder.id, child.folder.name)">&#x1F517;</span>
          </span>
        </button>
      </div>

      <GalleryTileView
        v-if="settings.layout.value === 'tiles'"
        :files="files"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
      />
      <GalleryListView
        v-else-if="settings.layout.value === 'list'"
        :files="files"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
      />
      <GalleryGroupedView
        v-else-if="settings.layout.value === 'grouped'"
        :files="files"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="selectedIds"
        :selectionEnabled="selectionEnabled"
        @select="toggleSelect"
        @deselect="toggleSelect"
        @open="(i: number) => handleFileClick(i)"
      />
    </div>

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
    <div v-else-if="loadingMore" class="text-center py-4 text-[var(--text-secondary)]">Loading more...</div>
    <div ref="sentinel" class="h-4"></div>

    <Lightbox
      v-if="settings.previewMode.value !== 'sidebar'"
      :file="lightboxFile"
      :index="lightboxIndex"
      :total="files.length"
      :hasPrev="lightboxIndex > 0"
      :hasNext="lightboxIndex < files.length - 1"
      @close="closeLightbox"
      @prev="goPrev"
      @next="goNext"
    />

    </template>

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

    <FolderPasswordDialog
      :visible="passwordDialog.show"
      :folderId="passwordDialog.folderId"
      :mode="passwordDialog.mode"
      :hasPassword="passwordDialog.hasPassword"
      :expiresAt="passwordDialog.expiresAt"
      @close="passwordDialog.show = false"
      @unlocked="onFolderUnlocked"
      @removed="onPasswordRemoved"
    />

    <FolderShareDialog
      :visible="shareDialog.show"
      :folderId="shareDialog.folderId"
      :folderName="shareDialog.folderName"
      @close="shareDialog.show = false"
    />
  </DropZone>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client'
import { useRouteQuery } from '../composables/useRouteQuery'
import { useLocalSettings } from '../composables/useLocalSettings'
import Lightbox from '../components/Lightbox.vue'
import FileViewer from '../components/FileViewer.vue'
import PreviewSidebar from '../components/PreviewSidebar.vue'
import GalleryTileView from '../components/GalleryTileView.vue'
import GalleryControls from '../components/GalleryControls.vue'
import GalleryListView from '../components/GalleryListView.vue'
import GalleryGroupedView from '../components/GalleryGroupedView.vue'
import ActionBar from '../components/ActionBar.vue'
import FolderPickerDialog from '../components/FolderPickerDialog.vue'
import InlineUpload from '../components/InlineUpload.vue'
import Breadcrumbs from '../components/Breadcrumbs.vue'
import DropZone from '../components/DropZone.vue'
import FolderPasswordDialog from '../components/FolderPasswordDialog.vue'
import FolderShareDialog from '../components/FolderShareDialog.vue'
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
  createdAt?: string
  folder_id?: string | null
  isAppManaged?: boolean
  thumbnails?: any
}

interface FolderEntry {
  id: string
  name: string
  parent_id: string | null
  created_at: string
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
const nextCursor = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const showCreate = ref(false)
const newFolderName = ref('')
const showNewDocInput = ref(false)
const newDocName = ref('')
const creatingDoc = ref(false)
const createInput = ref<HTMLInputElement | null>(null)
const sentinel = ref<HTMLElement | null>(null)

const folderIdQuery = useRouteQuery('folder_id', '')
const photoQuery = useRouteQuery('photo', '')

const settings = useLocalSettings()

const currentFolderId = computed(() => folderIdQuery.value || null)

const folderChain = computed(() => {
  const chain: { id: string | null; name: string }[] = [{ id: null, name: 'Root' }]
  if (!currentFolderId.value) return chain

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

  buildPath(folders.value.children ?? [], currentFolderId.value, chain)
  return chain
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

const listFolders = computed(() => {
  if (currentFolderId.value) return subfolders.value
  return folders.value.children ?? []
})

const selectedIds = ref(new Set<string>())
const lastClickedIndex = ref(-1)
const selectionEnabled = ref(true)
const showDeleteConfirm = ref(false)

const upload = useChunkedUploadStore()
let refreshInterval: ReturnType<typeof setInterval> | null = null

const moveDialog = ref({ open: false })
const copyDialog = ref({ open: false })

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
  return `This will move ${selectedIds.value.size} ${selectedIds.value.size === 1 ? 'file' : 'files'} to trash. You can recover them later.`
})

async function loadFolders() {
  try {
    const res = await api.get('/folders')
    folders.value = res.data
    await loadPasswordStatuses()
  } catch (e) {
    console.error('Failed to load folders', e)
  }
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
    const params: any = { sort: settings.sortBy.value, order: 'desc', limit: 500 }
    if (currentFolderId.value) {
      params.folder_id = currentFolderId.value
    }
    if (nextCursor.value) params.cursor = nextCursor.value
    const res = await api.get('/files', { params })
    files.value = reset ? res.data.items : [...files.value, ...res.data.items]
    nextCursor.value = res.data.nextCursor || ''
  } catch (e) {
    console.error('Failed to load files', e)
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function handleFileClick(index: number) {
  if (isShiftHeld() && lastClickedIndex.value >= 0) {
    selectRange(index)
  } else {
    lastClickedIndex.value = index
    if (isShiftHeld()) {
      toggleSelect(files.value[index].id)
    } else {
      openLightbox(index)
    }
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

function selectAllFiles() {
  selectedIds.value = new Set(files.value.map(f => f.id))
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
    loadFiles(true)
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
    loadFiles(true)
  } catch (e) {
    console.error('Failed to copy files', e)
  }
}

async function executeDelete() {
  try {
    const ids = Array.from(selectedIds.value)
    await api.post('/files/batch-delete', { ids })
    clearSelection()
    showDeleteConfirm.value = false
    loadFolders()
    loadFiles(true)
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
  const file = files.value[index]
  if (file) {
    if (file.mediaType === 'file') {
      fileViewerFile.value = file
    } else {
      photoQuery.value = file.id
    }
  }
}

function closeFileViewer() {
  fileViewerFile.value = null
}

async function createDocument() {
  const name = newDocName.value.trim()
  if (!name || creatingDoc.value) return
  creatingDoc.value = true
  try {
    const res = await api.post('/documents', { name, folder_id: currentFolderId.value || null })
    const newFile: FileItem = {
      id: res.data.id,
      originalName: name + '.md',
      filename: '_app_documents/' + name + '.md',
      sizeBytes: 0,
      mimeType: 'text/markdown',
      mediaType: 'file',
      isAppManaged: true,
    }
    files.value.unshift(newFile)
    fileViewerFile.value = newFile
    showNewDocInput.value = false
    newDocName.value = ''
  } catch {
  } finally {
    creatingDoc.value = false
  }
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

function navigateTo(id: string | null) {
  folderIdQuery.value = id ?? null
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

function formatFolderDate(dateStr: string): string {
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

function togglePreviewMode() {
  settings.previewMode.value = settings.previewMode.value === 'sidebar' ? 'lightbox' : 'sidebar'
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

function onFolderUnlocked(_folderId: string) {
  passwordDialog.value.show = false
  loadPasswordStatuses()
  loadFiles(true)
}

function onPasswordRemoved() {
  passwordDialog.value.show = false
  loadPasswordStatuses()
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

if (typeof window !== 'undefined') {
  window.addEventListener('keydown', (e) => { (window as any).__shiftHeld = e.shiftKey })
  window.addEventListener('keyup', (e) => { (window as any).__shiftHeld = e.shiftKey })
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Delete' && selectedIds.value.size > 0 && document.activeElement?.tagName !== 'INPUT') {
      showDeleteConfirm.value = true
    }
  })
}

watch([() => route.query.folder_id, () => route.query.media, () => route.query.all_folders], () => {
  loadFolders()
  loadFiles(true)
}, { immediate: false })

watch(showCreate, (v) => {
  if (v) nextTick(() => createInput.value?.focus())
})

onMounted(() => {
  loadFolders()
  loadPasswordStatuses()
  loadFiles(true)

  window.addEventListener('folder-password-required', ((e: CustomEvent) => {
    const folderId = e.detail?.folderId
    if (folderId) {
      openPasswordDialog(folderId)
      passwordDialog.value.mode = 'unlock'
    }
  }) as EventListener)

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

  refreshInterval = setInterval(async () => {
    const completed = upload.consumeCompletedJobs()
    if (completed.length === 0) return
    const folderKey = currentFolderId.value ?? null
    const relevant = completed.filter(j => (j.folder_id ?? null) === folderKey)
    for (const job of relevant) {
      try {
        const res = await api.get(`/files/${job.file_id}`)
        const newFile = res.data as FileItem
        if (files.value.some(f => f.id === newFile.id)) continue
        if (settings.sortBy.value === 'taken_at' || settings.sortBy.value === 'created_at') {
          files.value.unshift(newFile)
        } else {
          files.value.push(newFile)
        }
      } catch (e) {
        console.error('Failed to fetch new file', e)
      }
    }
    if (relevant.length > 0) {
      loadFolders()
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>
