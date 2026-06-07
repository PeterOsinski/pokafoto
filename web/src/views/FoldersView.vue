<template>
  <div class="flex h-full">
    <!-- Toggle strip -->
    <div class="shrink-0 w-10 flex flex-col items-center pt-3 border-r" style="border-color: var(--border-color); background: var(--bg-surface)">
      <button @click="treeExpanded = !treeExpanded" class="p-1 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]" :title="treeExpanded ? 'Collapse sidebar' : 'Expand sidebar'">
        <span class="text-xs" v-if="treeExpanded">&#x276E;</span>
        <span class="text-xs" v-else>&#x276F;</span>
      </button>
    </div>

    <!-- Resizable tree sidebar -->
    <div v-show="treeExpanded" class="shrink-0 border-r overflow-hidden" :style="{ width: treeWidth + 'px', borderColor: 'var(--border-color)', background: 'var(--bg-surface)' }">
      <div class="h-full overflow-y-auto p-3">
        <span class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider">Folders</span>
        <div v-if="ft.folders.value.children?.length" class="mt-2">
          <FolderNode
            v-for="child in ft.folders.value.children"
            :key="child.folder.id"
            :node="child"
            :selectedId="ft.currentFolderId.value"
            :depth="0"
            @select="(id: string) => ft.navigateTo(id)"
          />
        </div>
        <div v-else class="text-xs text-[var(--text-secondary)] py-2">No folders yet</div>
      </div>
    </div>

    <!-- Drag handle -->
    <div
      v-show="treeExpanded"
      class="w-1 shrink-0 cursor-col-resize hover:bg-[var(--accent)] transition-colors"
      :class="isResizing ? 'bg-[var(--accent)]' : ''"
      @mousedown="startResize"
    ></div>

    <!-- Main content area -->
    <div class="flex-1 min-w-0 overflow-y-auto p-4">
      <DropZone :folderId="ft.currentFolderId.value">
        <div v-if="settings.previewMode.value === 'sidebar' && lightboxFile" class="flex">
          <div class="flex-1 min-w-0">
            <GalleryContent
              :files="uf.files.value"
              :folderTiles="ft.folderTileTargets.value"
              :currentFolderId="ft.currentFolderId.value"
              :folderChain="ft.folderChain.value"
              :selectedIds="sel.selectedIds.value"
              :layout="(settings.layout.value as any)"
              :sortBy="(settings.sortBy.value as any)"
              :thumbLevel="settings.thumbLevel.value"
              :thumbSizePx="settings.thumbSizePx.value"
              :previewMode="settings.previewMode.value"
              :loading="uf.loading.value"
              :loadingMore="uf.loadingMore.value"
              :showCreateInput="showCreate"
              :passwordStatuses="ft.passwordStatuses.value"
              @navigateTo="(id: string | null) => ft.navigateTo(id)"
              @navigateUp="ft.navigateUp()"
              @showCreateInput="showCreate = true"
              @createFolder="(name: string) => handleCreateFolder(name)"
              @cancelCreateFolder="showCreate = false"
              @deleteFiles="dlg.showDeleteFilesConfirm.value = true"
              @moveFiles="dlg.moveDialogOpen.value = true"
              @copyFiles="dlg.copyDialogOpen.value = true"
              @clearSelection="sel.clear()"
              @selectAll="sel.selectAll(uf.files.value.map(f => f.id))"
              @select="(id: string) => sel.toggle(id)"
              @deselect="(id: string) => sel.toggle(id)"
              @openFile="(i: number) => handleFileClick(i)"
              @fileContextMenu="(e: MouseEvent, id: string, name: string) => dlg.openFileContextMenu(e, id, name)"
              @folderContextMenu="(e: MouseEvent, f: any) => dlg.openFolderContextMenu(e, f, ft.passwordStatuses.value[f.id])"
              @update:layout="(v: any) => settings.layout.value = v"
              @update:sortBy="(v: any) => settings.sortBy.value = v"
              @update:thumbLevel="(v: number) => settings.thumbLevel.value = v"
              @togglePreviewMode="togglePreviewMode"
            />
          </div>

          <PreviewSidebar
            :file="lightboxFile"
            :index="lightboxIndex"
            :total="uf.files.value.length"
            :hasPrev="lightboxIndex > 0"
            :hasNext="lightboxIndex < uf.files.value.length - 1"
            @close="closeLightbox"
            @prev="goPrev"
            @next="goNext"
          />
        </div>

        <template v-else>
          <GalleryContent
            :files="uf.files.value"
            :folderTiles="ft.folderTileTargets.value"
            :currentFolderId="ft.currentFolderId.value"
            :folderChain="ft.folderChain.value"
            :selectedIds="sel.selectedIds.value"
            :layout="(settings.layout.value as any)"
            :sortBy="(settings.sortBy.value as any)"
            :thumbLevel="settings.thumbLevel.value"
            :thumbSizePx="settings.thumbSizePx.value"
            :previewMode="settings.previewMode.value"
            :loading="uf.loading.value"
            :loadingMore="uf.loadingMore.value"
            :showCreateInput="showCreate"
            :showNewDocInput="showNewDocInput"
            :passwordStatuses="ft.passwordStatuses.value"
            @navigateTo="(id: string | null) => ft.navigateTo(id)"
            @navigateUp="ft.navigateUp()"
            @showCreateInput="showCreate = true"
            @createFolder="(name: string) => handleCreateFolder(name)"
            @cancelCreateFolder="showCreate = false"
            @deleteFiles="dlg.showDeleteFilesConfirm.value = true"
            @moveFiles="dlg.moveDialogOpen.value = true"
            @copyFiles="dlg.copyDialogOpen.value = true"
            @clearSelection="sel.clear()"
            @selectAll="sel.selectAll(uf.files.value.map(f => f.id))"
            @select="(id: string) => sel.toggle(id)"
            @deselect="(id: string) => sel.toggle(id)"
            @openFile="(i: number) => handleFileClick(i)"
            @fileContextMenu="(e: MouseEvent, id: string, name: string) => dlg.openFileContextMenu(e, id, name)"
            @folderContextMenu="(e: MouseEvent, f: any) => dlg.openFolderContextMenu(e, f, ft.passwordStatuses.value[f.id])"
            @update:layout="(v: any) => settings.layout.value = v"
            @update:sortBy="(v: any) => settings.sortBy.value = v"
            @update:thumbLevel="(v: number) => settings.thumbLevel.value = v"
            @togglePreviewMode="togglePreviewMode"
            @toggleNewDoc="showNewDocInput = !showNewDocInput"
          />

          <div v-if="showNewDocInput" class="fixed inset-0 z-30" @click="showNewDocInput = false">
            <div class="absolute top-[155px] left-[calc(50%+100px)] mt-1 p-2 rounded shadow-lg z-30 flex gap-2" style="background: var(--bg-surface); border: 1px solid var(--border-color)" @click.stop>
              <input
                v-model="newDocName"
                placeholder="Document name"
                class="px-2 py-1 rounded text-sm bg-black/30 text-white border border-white/10 outline-none focus:border-[var(--accent)]"
                @keydown.enter="createDocument"
                @keydown.escape="showNewDocInput = false"
              />
              <button @click="createDocument" :disabled="!newDocName.trim() || creatingDoc" class="px-3 py-1 rounded text-sm text-white disabled:opacity-50" style="background: var(--accent)">Create</button>
            </div>
          </div>

          <Lightbox
            v-if="settings.previewMode.value !== 'sidebar'"
            :file="lightboxFile"
            :index="lightboxIndex"
            :total="uf.files.value.length"
            :hasPrev="lightboxIndex > 0"
            :hasNext="lightboxIndex < uf.files.value.length - 1"
            @close="closeLightbox"
            @prev="goPrev"
            @next="goNext"
          />
        </template>

        <FileViewer :file="fileViewerFile" @close="closeFileViewer" />

        <FolderPickerDialog
          :open="dlg.moveDialogOpen.value"
          title="Move to folder"
          actionLabel="Move here"
          @close="dlg.moveDialogOpen.value = false"
          @confirm="(folderId: string | null) => handleMoveFiles(folderId)"
        />

        <FolderPickerDialog
          :open="dlg.copyDialogOpen.value"
          title="Copy to folder"
          actionLabel="Copy here"
          @close="dlg.copyDialogOpen.value = false"
          @confirm="(folderId: string | null) => handleCopyFiles(folderId)"
        />

        <div v-if="dlg.showDeleteFilesConfirm.value" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="dlg.showDeleteFilesConfirm.value = false">
          <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4" style="border: 1px solid var(--border-color)">
            <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Delete files?</h3>
            <p class="text-sm text-[var(--text-secondary)] mb-4">{{ dlg.deleteMessage }}</p>
            <div class="flex justify-end gap-3">
              <button @click="dlg.showDeleteFilesConfirm.value = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Cancel</button>
              <button @click="handleDeleteFiles" class="px-4 py-2 rounded text-sm text-white" style="background: #ef4444">Delete</button>
            </div>
          </div>
        </div>

        <FolderPasswordDialog
          :visible="dlg.passwordDialog.show"
          :folderId="dlg.passwordDialog.folderId"
          :mode="dlg.passwordDialog.mode"
          :hasPassword="!!ft.passwordStatuses.value[dlg.passwordDialog.folderId]"
          :passwordHint="dlg.passwordDialog.passwordHint"
          @close="dlg.passwordDialog.show = false"
          @unlocked="onFolderUnlocked"
          @removed="onPasswordRemoved"
        />

        <FolderShareDialog
          :visible="dlg.shareDialog.show"
          :folderId="dlg.shareDialog.folderId"
          :folderName="dlg.shareDialog.folderName"
          @close="dlg.shareDialog.show = false"
        />

        <ContextMenu
          :visible="dlg.contextMenu.visible"
          :position="{ x: dlg.contextMenu.x, y: dlg.contextMenu.y }"
          :items="dlg.contextMenu.items"
          @close="dlg.contextMenu.visible = false"
        />

        <div v-if="dlg.renameDialog.show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="dlg.renameDialog.show = false">
          <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4" style="border: 1px solid var(--border-color)">
            <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Rename {{ dlg.renameDialog.type }}</h3>
            <input
              v-model="dlg.renameDialog.currentName"
              type="text"
              class="w-full px-3 py-2 rounded text-sm mb-4"
              style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
              @keyup.enter="handleRename"
              @keyup.escape="dlg.renameDialog.show = false"
            />
            <div class="flex justify-end gap-3">
              <button @click="dlg.renameDialog.show = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Cancel</button>
              <button @click="handleRename" class="px-4 py-2 rounded text-sm text-white" style="background: var(--accent)">Rename</button>
            </div>
          </div>
        </div>

        <div v-if="dlg.deleteFolderConfirm.show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="dlg.deleteFolderConfirm.show = false">
          <div class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4" style="border: 1px solid var(--border-color)">
            <template v-if="!dlg.deleteFolderConfirm.result">
              <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Delete Folder?</h3>
              <p class="text-sm text-[var(--text-secondary)] mb-4">
                &quot;{{ dlg.deleteFolderConfirm.folderName }}&quot; and all its contents will be moved to trash.
              </p>
              <div class="flex justify-end gap-3">
                <button @click="dlg.deleteFolderConfirm.show = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Cancel</button>
                <button @click="handleFolderDelete" :disabled="dlg.deleteFolderConfirm.loading" class="px-4 py-2 rounded text-sm text-white" style="background: #ef4444">
                  {{ dlg.deleteFolderConfirm.loading ? 'Deleting...' : 'Delete' }}
                </button>
              </div>
            </template>
            <template v-else>
              <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">Folder Deleted</h3>
              <p class="text-sm text-[var(--text-secondary)] mb-4">
                {{ dlg.deleteFolderConfirm.result.deleted_files }} {{ dlg.deleteFolderConfirm.result.deleted_files === 1 ? 'file' : 'files' }} and {{ dlg.deleteFolderConfirm.result.deleted_folders }} {{ dlg.deleteFolderConfirm.result.deleted_folders === 1 ? 'subfolder' : 'subfolders' }} moved to trash.
              </p>
              <div class="flex justify-end">
                <button @click="dlg.deleteFolderConfirm.show = false" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Close</button>
              </div>
            </template>
          </div>
        </div>

        <FolderPickerDialog
          :open="dlg.moveFolderDialog.show"
          title="Move folder"
          actionLabel="Move here"
          @close="dlg.moveFolderDialog.show = false"
          @confirm="(folderId: string | null) => handleFolderMove(folderId)"
        />
      </DropZone>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client'
import { useRouteQuery } from '../composables/useRouteQuery'
import { useLocalSettings } from '../composables/useLocalSettings'
import { useFolderTree } from '../composables/useFolderTree'
import { useFiles } from '../composables/useFiles'
import { useSelection } from '../composables/useSelection'
import { useFolderDialogs } from '../composables/useFolderDialogs'
import GalleryContent from '../components/GalleryContent.vue'
import Lightbox from '../components/Lightbox.vue'
import FileViewer from '../components/FileViewer.vue'
import PreviewSidebar from '../components/PreviewSidebar.vue'
import FolderPickerDialog from '../components/FolderPickerDialog.vue'
import FolderPasswordDialog from '../components/FolderPasswordDialog.vue'
import FolderShareDialog from '../components/FolderShareDialog.vue'
import FolderNode from '../components/FolderNode.vue'
import DropZone from '../components/DropZone.vue'
import ContextMenu from '../components/ContextMenu.vue'
import { useChunkedUploadStore } from '../stores/chunkedUpload'
import type { FileItem } from '../types/gallery'

const route = useRoute()
const settings = useLocalSettings()
const ft = useFolderTree()
const uf = useFiles(ft.currentFolderId, computed(() => settings.sortBy.value as any))
const sel = useSelection()
const dlg = useFolderDialogs()

const photoQuery = useRouteQuery('photo', '')
const showCreate = ref(false)
const showNewDocInput = ref(false)
const newDocName = ref('')
const creatingDoc = ref(false)
const fileViewerFile = ref<FileItem | null>(null)

const treeExpanded = ref(true)
const treeWidth = ref(settings.sidebarWidth.value)
const isResizing = ref(false)

function startResize(e: MouseEvent) {
  isResizing.value = true
  const startX = e.clientX
  const startWidth = treeWidth.value

  function onMove(e: MouseEvent) {
    const newWidth = Math.max(160, Math.min(600, startWidth + e.clientX - startX))
    treeWidth.value = newWidth
  }

  function onUp() {
    isResizing.value = false
    settings.sidebarWidth.value = treeWidth.value
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.userSelect = ''
    document.body.style.cursor = ''
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'col-resize'
}

const upload = useChunkedUploadStore()
let refreshInterval: ReturnType<typeof setInterval> | null = null

const lightboxFile = computed(() => {
  if (!photoQuery.value) return null
  return uf.files.value.find(f => f.id === photoQuery.value) ?? null
})

const lightboxIndex = computed(() => {
  if (!lightboxFile.value) return -1
  return uf.files.value.indexOf(lightboxFile.value)
})

function togglePreviewMode() {
  settings.previewMode.value = settings.previewMode.value === 'sidebar' ? 'lightbox' : 'sidebar'
}

function closeLightbox() { photoQuery.value = null }
function goPrev() {
  if (lightboxIndex.value > 0) photoQuery.value = uf.files.value[lightboxIndex.value - 1].id
}
function goNext() {
  if (lightboxIndex.value < uf.files.value.length - 1) photoQuery.value = uf.files.value[lightboxIndex.value + 1].id
}
function closeFileViewer() { fileViewerFile.value = null }

function handleFileClick(index: number) {
  if (sel.isShiftHeld() && sel.lastClickedIndex.value >= 0) {
    const start = Math.min(sel.lastClickedIndex.value, index)
    const end = Math.max(sel.lastClickedIndex.value, index)
    for (let i = start; i <= end; i++) sel.toggle(uf.files.value[i].id)
  } else {
    sel.lastClickedIndex.value = index
    if (sel.isShiftHeld()) {
      sel.toggle(uf.files.value[index].id)
    } else {
      const file = uf.files.value[index]
      if (file) {
        if (file.mediaType === 'file') {
          fileViewerFile.value = file
        } else {
          photoQuery.value = file.id
        }
      }
    }
  }
}

async function handleCreateFolder(name: string) {
  await ft.createFolder(name)
  showCreate.value = false
  await ft.loadFolders()
  await ft.loadPasswordStatuses()
}

async function handleRename() {
  const ok = await dlg.executeRename()
  if (ok) {
    await ft.loadFolders()
    await ft.loadPasswordStatuses()
    await uf.loadFiles(true)
  }
}

async function handleFolderDelete() {
  const ok = await dlg.executeFolderDelete()
  if (ok) {
    await ft.loadFolders()
    await ft.loadPasswordStatuses()
    await uf.loadFiles(true)
  }
}

async function handleFolderMove(targetId: string | null) {
  const ok = await dlg.executeFolderMove(targetId)
  if (ok) {
    await ft.loadFolders()
    await ft.loadPasswordStatuses()
    await uf.loadFiles(true)
  }
}

async function handleDeleteFiles() {
  const ok = await dlg.executeDeleteFiles()
  if (ok) {
    sel.clear()
    await ft.loadFolders()
    await uf.loadFiles(true)
  }
}

async function handleMoveFiles(targetId: string | null) {
  const ok = await dlg.executeMoveFiles(targetId)
  if (ok) {
    sel.clear()
    await ft.loadFolders()
    await uf.loadFiles(true)
  }
}

async function handleCopyFiles(targetId: string | null) {
  const ok = await dlg.executeCopyFiles(targetId)
  if (ok) {
    sel.clear()
    await ft.loadFolders()
    await uf.loadFiles(true)
  }
}

async function createDocument() {
  const name = newDocName.value.trim()
  if (!name || creatingDoc.value) return
  creatingDoc.value = true
  try {
    const res = await api.post('/documents', { name, folder_id: ft.currentFolderId.value || null })
    const newFile: FileItem = {
      id: res.data.id,
      originalName: name + '.md',
      filename: '_app_documents/' + name + '.md',
      sizeBytes: 0,
      mimeType: 'text/markdown',
      mediaType: 'file',
      isAppManaged: true,
    }
    uf.files.value.unshift(newFile)
    fileViewerFile.value = newFile
    showNewDocInput.value = false
    newDocName.value = ''
  } catch {} finally {
    creatingDoc.value = false
  }
}

function onFolderUnlocked(_folderId: string) {
  dlg.passwordDialog.show = false
  ft.loadPasswordStatuses()
  uf.loadFiles(true)
}

function onPasswordRemoved() {
  dlg.passwordDialog.show = false
  ft.loadPasswordStatuses()
}

// Wire up context menu actions
const origOpenFolderCtx = dlg.openFolderContextMenu
const origOpenFileCtx = dlg.openFileContextMenu

dlg.openFolderContextMenu = function(e: MouseEvent, folder: any, passStatus: boolean) {
  origOpenFolderCtx(e, folder, passStatus)
  const items = dlg.contextMenu.items
  items[0].action = () => {
    dlg.contextMenu.visible = false
    const info = ft.openPasswordDialog(folder.id)
    dlg.openPasswordDialog(folder.id, info.mode, info.passwordHint)
  }
  items[1].action = () => {
    dlg.contextMenu.visible = false
    dlg.openShareDialog(folder.id, folder.name)
  }
  items[2].action = () => {
    dlg.contextMenu.visible = false
    dlg.openRenameDialog('folder', folder.id, folder.name)
  }
  items[3].action = () => {
    dlg.contextMenu.visible = false
    dlg.openMoveFolderDialog(folder.id, folder.name)
  }
  items[4].action = () => {
    dlg.contextMenu.visible = false
    dlg.openDeleteFolderConfirm(folder.id, folder.name)
  }
}

dlg.openFileContextMenu = function(e: MouseEvent, fileId: string, fileName: string) {
  origOpenFileCtx(e, fileId, fileName)
  dlg.contextMenu.items[0].action = () => {
    dlg.contextMenu.visible = false
    dlg.openRenameDialog('file', fileId, fileName)
  }
}

// HACK: wire selectedIds from dlg to sel (they share the same reference for file operations)
Object.defineProperty(dlg, 'selectedIds', { get: () => sel.selectedIds })

watch([() => route.query.folder_id, () => route.query.media], () => {
  ft.loadFolders()
  ft.loadPasswordStatuses()
  uf.loadFiles(true)
}, { immediate: false })

watch(showCreate, (v) => {
  if (v) nextTick(() => {
    const input = document.querySelector('input[placeholder="Folder name..."]') as HTMLInputElement
    input?.focus()
  })
})

sel.setupKeyboard(() => { dlg.showDeleteFilesConfirm.value = true })

onMounted(async () => {
  await ft.loadFolders()
  await ft.loadPasswordStatuses()
  await uf.loadFiles(true)

  window.addEventListener('folder-password-required', ((e: CustomEvent) => {
    const folderId = e.detail?.folderId
    if (folderId) {
      const info = ft.openPasswordDialog(folderId)
      dlg.openPasswordDialog(folderId, 'unlock', info.passwordHint)
    }
  }) as EventListener)

  refreshInterval = setInterval(async () => {
    const completed = upload.consumeCompletedJobs()
    if (completed.length === 0) return
    const folderKey = ft.currentFolderId.value ?? null
    const relevant = completed.filter(j => (j.folder_id ?? null) === folderKey)
    for (const job of relevant) {
      try {
        const res = await api.get(`/files/${job.file_id}`)
        const newFile = res.data as FileItem
        if (uf.files.value.some(f => f.id === newFile.id)) continue
        if (settings.sortBy.value === 'taken_at' || settings.sortBy.value === 'created_at') {
          uf.files.value.unshift(newFile)
        } else {
          uf.files.value.push(newFile)
        }
      } catch (e) {
        console.error('Failed to fetch new file', e)
      }
    }
    if (relevant.length > 0) {
      ft.loadFolders()
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
  sel.teardownKeyboard()
})
</script>
