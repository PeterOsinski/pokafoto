<template>
  <div :class="isSidebarOpen ? 'flex' : ''" class="min-h-screen bg-[var(--bg-primary)]">
    <div class="flex-1 min-w-0 p-6">
      <div v-if="loading" class="max-w-md mx-auto mt-20 text-center text-[var(--text-secondary)]">Loading shared folder...</div>
      <div v-else-if="error" class="max-w-md mx-auto mt-20 text-center">
        <h1 class="text-xl font-bold text-[var(--text-primary)] mb-2">Not Available</h1>
        <p class="text-[var(--text-secondary)]">{{ error }}</p>
      </div>
      <div v-else-if="needsPassword && !sessionToken" class="max-w-md mx-auto mt-20">
        <h1 class="text-xl font-bold text-[var(--text-primary)] mb-2">Password Required</h1>
        <p class="text-[var(--text-secondary)] mb-4">This shared folder is password-protected.</p>
        <input ref="passwordInput" v-model="password" type="password" placeholder="Enter password..." class="w-full px-3 py-2 rounded text-sm mb-3" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)" @keyup.enter="unlock" />
        <p v-if="unlockError" class="text-sm text-[var(--error)] mb-3">{{ unlockError }}</p>
        <button @click="unlock" :disabled="!password.trim() || unlocking" class="w-full px-4 py-2 rounded text-sm text-white disabled:opacity-50" style="background: var(--accent)">{{ unlocking ? 'Unlocking...' : 'Unlock' }}</button>
      </div>
      <div v-else-if="sessionToken">
        <header class="mb-4 flex flex-wrap gap-3 items-start justify-between">
          <div>
            <h1 class="text-xl font-bold text-[var(--text-primary)]">{{ folderName }}</h1>
            <div v-if="includeSubdirs && folderId" class="flex items-center gap-1 mt-1">
              <button @click="navigateUp" class="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]">&#8592; Back</button>
            </div>
            <span class="text-xs text-[var(--text-secondary)]">
              Shared folder
              <span v-if="permissions === 'read_upload'"> &middot; Can upload</span>
              <span v-else-if="permissions === 'read_write'"> &middot; Can upload &amp; delete</span>
              <span v-if="includeSubdirs"> &middot; Subdirs</span>
            </span>
          </div>
          <div class="text-xs text-[var(--text-secondary)]">
            <span v-if="uploadLimit !== null">{{ formatBytes(uploadedBytes) }} / {{ formatBytes(uploadLimit) }}</span>
            <span v-if="expiresAt" class="ml-3">Expires {{ formatDate(expiresAt) }}</span>
          </div>
        </header>

        <GalleryControls
          :layout="shareLayout"
          :sortBy="shareSortBy"
          :thumbLevel="shareThumbLevel"
          :previewMode="sharePreviewMode"
          :sortOptions="shareSortOptions"
          :hidePreviewToggle="isSidebarOpen"
          @update:layout="v => shareLayout = v"
          @update:sortBy="v => shareSortBy = v"
          @update:thumbLevel="(v: number) => shareThumbLevel = v"
          @togglePreviewMode="toggleSharePreview"
        />

        <div v-if="includeSubdirs && subfolders.length > 0 && shareLayout === 'tiles'" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-6">
          <button v-for="child in subfolders" :key="child.id" @click="navigateTo(child.id)" class="flex flex-col items-center gap-2 p-4 rounded-lg text-center transition-colors hover:bg-[var(--bg-elevated)]" style="border: 1px solid var(--border-color)">
            <span class="text-3xl">&#128193;</span>
            <span class="text-sm text-[var(--text-primary)] font-medium truncate w-full">{{ child.name }}</span>
            <span class="text-xs text-[var(--text-secondary)]">{{ child.file_count }} {{ child.file_count === 1 ? 'file' : 'files' }}</span>
            <button v-if="canDeleteFolder" @click.stop="deleteFolder(child.id)" class="text-xs text-red-400 hover:text-red-300 mt-1 cursor-pointer">Remove</button>
          </button>
        </div>

        <div v-if="includeSubdirs && subfolders.length > 0 && shareLayout === 'list'" class="mb-4">
          <div class="flex items-center border-b border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-xs font-semibold uppercase tracking-wide select-none px-3">
            <span class="w-10 shrink-0" />
            <span class="flex-1 min-w-0 py-2 px-3">Name</span>
            <span class="py-2 px-3 shrink-0">Files</span>
          </div>
          <button v-for="child in subfolders" :key="child.id" @click="navigateTo(child.id)" class="flex items-center w-full border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors px-3" style="height: 52px">
            <span class="text-xl w-10 shrink-0">&#128193;</span>
            <span class="flex-1 min-w-0 text-sm text-[var(--text-primary)] font-medium truncate text-left">{{ child.name }}</span>
            <span class="text-xs text-[var(--text-secondary)] shrink-0 mr-2">{{ child.file_count }} {{ child.file_count === 1 ? 'file' : 'files' }}</span>
            <button v-if="canDeleteFolder" @click.stop="deleteFolder(child.id)" class="text-xs text-red-400 hover:text-red-300 shrink-0 cursor-pointer">Remove</button>
          </button>
        </div>

        <div v-if="includeSubdirs && canWrite" class="flex items-center gap-2 mb-4">
          <button v-if="!showCreateFolder" @click="showCreateFolder = true" class="px-3 py-1 rounded text-sm text-white" style="background: var(--accent)">+ New Folder</button>
          <template v-if="showCreateFolder">
            <input ref="createFolderInput" v-model="newFolderName" type="text" placeholder="Folder name..." class="px-3 py-2 rounded text-sm flex-1" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)" @keyup.enter="createFolder" @keyup.escape="showCreateFolder = false" />
            <button @click="createFolder" :disabled="!newFolderName.trim()" class="px-3 py-2 rounded text-sm text-white" style="background: var(--accent)" :class="!newFolderName.trim() ? 'opacity-50 cursor-not-allowed' : ''">Create</button>
            <button @click="showCreateFolder = false" class="px-3 py-2 rounded text-sm text-[var(--text-secondary)]">Cancel</button>
          </template>
        </div>

        <div v-if="displayFiles.length === 0 && !filesLoaded" class="text-center py-10 text-[var(--text-secondary)]">Loading files...</div>
        <div v-else-if="displayFiles.length === 0 && subfolders.length === 0" class="text-center py-10 text-[var(--text-secondary)]">This folder is empty.</div>
        <template v-else-if="displayFiles.length > 0">
          <div v-if="shareLayout === 'tiles'" class="grid gap-3" :style="{ gridTemplateColumns: `repeat(auto-fill, minmax(${shareThumbPx}px, 1fr))` }">
            <div v-for="(file, fi) in displayFiles" :key="file.id" class="flex flex-col rounded-lg overflow-hidden cursor-pointer hover:ring-2 hover:ring-[var(--accent)] transition-all relative group" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
              <div class="aspect-square overflow-hidden bg-black/20" @click="openLightbox(fi)">
                <img v-if="file.thumbnails?.sm?.url" :src="thumbnailSrc(file)" :alt="file.original_name" class="w-full h-full object-cover" loading="lazy" />
                <div v-else class="w-full h-full flex items-center justify-center text-3xl text-[var(--text-secondary)]">{{ fileIcon(file.mime_type) }}</div>
              </div>
              <div class="p-2">
                <span class="text-xs text-[var(--text-primary)] truncate block" :title="file.original_name">{{ file.original_name }}</span>
                <span class="text-xs text-[var(--text-secondary)]">{{ formatBytes(file.size_bytes) }}</span>
              </div>
              <button v-if="canDelete" @click.stop="deleteFile(file.id)" class="absolute top-1 right-1 w-6 h-6 rounded-full flex items-center justify-center text-xs text-white opacity-0 group-hover:opacity-100 transition-opacity" style="background: #ef4444" title="Delete">&times;</button>
              <button @click.stop="downloadFile(file)" class="absolute top-1 left-1 w-6 h-6 rounded-full flex items-center justify-center text-xs text-white opacity-0 group-hover:opacity-100 transition-opacity" style="background: var(--accent)" title="Download">&#x2193;</button>
            </div>
          </div>
          <GalleryListView v-else-if="shareLayout === 'list'" :files="displayFiles" :thumbSizePx="shareThumbPx" :selectedIds="new Set()" :selectionEnabled="false" @select="() => {}" @deselect="() => {}" @open="(i: number) => openLightbox(i)" />
          <GalleryGroupedView v-else-if="shareLayout === 'grouped'" :files="displayFiles" :thumbSizePx="shareThumbPx" :selectedIds="new Set()" :selectionEnabled="false" @select="() => {}" @deselect="() => {}" @open="(i: number) => openLightbox(i)" />
        </template>

        <Lightbox v-if="!isSidebarOpen" :file="lightboxFile" :index="lightboxIndex" :total="displayFiles.length" :has-prev="lightboxIndex > 0" :has-next="lightboxIndex < displayFiles.length - 1" :hideSocial="true" :downloadUrl="downloadFileUrl" @close="closeLightbox" @next="goNext" @prev="goPrev" />

        <div v-if="canUpload" class="mt-8 p-6 border-2 border-dashed rounded-lg text-center" style="border-color: var(--border-color)">
          <p class="text-sm text-[var(--text-secondary)] mb-3">Drag files here or click to upload</p>
          <input type="file" multiple class="hidden" ref="fileInput" @change="onFilesSelected" />
          <button @click="fileInput?.click()" class="px-4 py-2 rounded text-sm text-white" style="background: var(--accent)">Select Files</button>
          <div v-if="uploadProgress" class="mt-3 text-sm text-[var(--text-secondary)]">{{ uploadProgress }}</div>
        </div>
      </div>
    </div>

    <PreviewSidebar
      v-if="isSidebarOpen"
      :file="lightboxFile"
      :index="lightboxIndex"
      :total="displayFiles.length"
      :hasPrev="lightboxIndex > 0"
      :hasNext="lightboxIndex < displayFiles.length - 1"
      :hideSocial="true"
      @close="closeLightbox"
      @prev="goPrev"
      @next="goNext"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import Lightbox from '../components/Lightbox.vue'
import PreviewSidebar from '../components/PreviewSidebar.vue'
import GalleryControls from '../components/GalleryControls.vue'
import GalleryListView from '../components/GalleryListView.vue'
import GalleryGroupedView from '../components/GalleryGroupedView.vue'

interface FileItem {
  id: string
  original_name: string
  originalName: string
  filename: string
  size_bytes: number
  sizeBytes: number
  mime_type: string
  mimeType: string
  media_type: string
  mediaType: string
  width?: number
  height?: number
  taken_at?: string
  TakenAt?: string
  created_at: string
  CreatedAt: string
  thumbnails?: { sm?: { url: string; width: number; height: number }; md?: { url: string; width: number; height: number }; preview?: { url: string; width: number; height: number } }
}

interface Subfolder {
  id: string
  name: string
  parent_id: string | null
  file_count: number
  created_at: string
}

const route = useRoute()
const token = ref((route.params.token as string) || '')
const folderName = ref('')
const permissions = ref('read')
const includeSubdirs = ref(false)
const uploadLimit = ref<number | null>(null)
const uploadedBytes = ref(0)
const expiresAt = ref<string | null>(null)
const needsPassword = ref(false)
const password = ref('')
const sessionToken = ref('')
const files = ref<FileItem[]>([])
const loading = ref(true)
const error = ref('')
const unlockError = ref('')
const unlocking = ref(false)
const filesLoaded = ref(false)
const canUpload = ref(false)
const canDelete = ref(false)
const canWrite = ref(false)
const canDeleteFolder = ref(false)
const uploadProgress = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const passwordInput = ref<HTMLInputElement | null>(null)
const subfolders = ref<Subfolder[]>([])
const showCreateFolder = ref(false)
const newFolderName = ref('')
const createFolderInput = ref<HTMLInputElement | null>(null)

const baseURL = '/api/v1'

const shareLayout = ref(readShareSetting('layout', 'tiles'))
const shareSortBy = ref(readShareSetting('sort', 'taken_at'))
const shareThumbLevel = ref(Number(readShareSetting('thumbLevel', '5')))
const sharePreviewMode = ref(readShareSetting('previewMode', 'lightbox'))

const shareSortOptions = [
  { value: 'taken_at', label: 'Date Taken' },
  { value: 'filename', label: 'File Name' },
]

const shareThumbPx = computed(() => {
  const t = Number(shareThumbLevel.value) || 5
  if (t <= 0) return 40
  if (t >= 9) return 400
  return Math.round(40 + (t / 9) * 360)
})

const isSidebarOpen = computed(() => sharePreviewMode.value === 'sidebar' && lightboxIndex.value >= 0)

const lightboxIndex = ref(-1)
const folderId = ref('')

const sortedFiles = computed(() => {
  const list = [...files.value]
  const by = shareSortBy.value
  list.sort((a, b) => {
    let va: any, vb: any
    if (by === 'filename') {
      va = a.original_name || ''
      vb = b.original_name || ''
      return va.localeCompare(vb)
    }
    va = a.taken_at || ''
    vb = b.taken_at || ''
    if (!va && !vb) return 0
    if (!va) return 1
    if (!vb) return -1
    return vb.localeCompare(va)
  })
  return list
})

const displayFiles = computed(() => sortedFiles.value)

const lightboxFile = computed(() => {
  if (lightboxIndex.value < 0 || lightboxIndex.value >= displayFiles.value.length) return null
  const f = displayFiles.value[lightboxIndex.value]
  const t = f.thumbnails
  return {
    id: f.id,
    originalName: f.originalName || f.original_name,
    mediaType: f.mediaType || f.media_type,
    sizeBytes: f.sizeBytes || f.size_bytes || 0,
    thumbnails: t ? {
      sm: t.sm || { url: '', width: 60, height: 60 },
      md: t.md || { url: '', width: 600, height: 600 },
      preview: t.preview || { url: '', width: 720, height: 720 },
    } : undefined,
  }
})

const downloadFileUrl = computed(() => {
  if (!lightboxFile.value) return ''
  return `${baseURL}/share/${token.value}/download/${lightboxFile.value.id}?share_session_token=${sessionToken.value}`
})

function readShareSetting(key: string, fallback: string): string {
  try { const raw = localStorage.getItem(`share:${key}`); return raw || fallback } catch { return fallback }
}

function writeShareSetting(key: string, value: string) {
  try { localStorage.setItem(`share:${key}`, value) } catch {}
}

watch(shareLayout, v => writeShareSetting('layout', v))
watch(shareSortBy, v => writeShareSetting('sort', v))
watch(shareThumbLevel, v => writeShareSetting('thumbLevel', String(v)))
watch(sharePreviewMode, v => writeShareSetting('previewMode', v))
watch(showCreateFolder, (v) => { if (v) nextTick(() => createFolderInput.value?.focus()) })

function toggleSharePreview() {
  sharePreviewMode.value = sharePreviewMode.value === 'sidebar' ? 'lightbox' : 'sidebar'
}

function openLightbox(index: number) { lightboxIndex.value = index }
function closeLightbox() { lightboxIndex.value = -1 }
function goPrev() { if (lightboxIndex.value > 0) lightboxIndex.value-- }
function goNext() { if (lightboxIndex.value < displayFiles.value.length - 1) lightboxIndex.value++ }

async function fetchInfo() {
  loading.value = true
  error.value = ''
  try {
    const res = await axios.get(`${baseURL}/share/${token.value}`)
    const d = res.data
    folderName.value = d.folder_name || 'Shared Folder'
    permissions.value = d.permissions || 'read'
    includeSubdirs.value = d.include_subdirs || false
    uploadLimit.value = d.upload_limit_bytes || null
    uploadedBytes.value = d.uploaded_bytes || 0
    needsPassword.value = d.needs_password || false
    expiresAt.value = d.expires_at || null
    canUpload.value = permissions.value === 'read_upload' || permissions.value === 'read_write'
    canDelete.value = permissions.value === 'read_write'
    canWrite.value = permissions.value === 'read_write' && includeSubdirs.value
    canDeleteFolder.value = permissions.value === 'read_write' && includeSubdirs.value
    if (!d.needs_password) await unlockWithoutPassword()
  } catch (err: any) {
    if (err.response?.status === 410) error.value = 'This share link has expired.'
    else if (err.response?.status === 404) error.value = 'Share link not found.'
    else error.value = 'Failed to load shared folder.'
  } finally { loading.value = false }
}

async function unlockWithoutPassword() {
  try {
    const res = await axios.post(`${baseURL}/share/${token.value}/unlock`, {})
    sessionToken.value = res.data.share_session_token
    await loadAll()
  } catch {}
}

async function unlock() {
  if (!password.value.trim()) return
  unlocking.value = true; unlockError.value = ''
  try {
    const res = await axios.post(`${baseURL}/share/${token.value}/unlock`, { password: password.value })
    sessionToken.value = res.data.share_session_token
    await loadAll()
  } catch (err: any) {
    unlockError.value = err?.response?.data?.error?.message || 'Invalid password'
  } finally { unlocking.value = false }
}

async function loadAll() { await Promise.all([fetchFiles(), fetchSubfolders()]) }

async function fetchFiles() {
  try {
    const params: any = { limit: 500 }
    if (folderId.value) params.folder_id = folderId.value
    const res = await axios.get(`${baseURL}/share/${token.value}/files`, { headers: { 'X-Share-Session-Token': sessionToken.value }, params })
    files.value = (res.data.items || []).map((f: any) => ({ ...f, originalName: f.original_name, filename: f.original_name, sizeBytes: f.size_bytes || 0, mimeType: f.mime_type || '', mediaType: f.media_type || 'file', CreatedAt: f.created_at || '' }))
    filesLoaded.value = true
  } catch { files.value = [] }
}

async function fetchSubfolders() {
  if (!includeSubdirs.value) return
  try {
    const params: any = {}
    if (folderId.value) params.parent_id = folderId.value
    const res = await axios.get(`${baseURL}/share/${token.value}/folders`, { headers: { 'X-Share-Session-Token': sessionToken.value }, params })
    subfolders.value = res.data.folders || []
  } catch { subfolders.value = [] }
}

function navigateTo(id: string) { folderId.value = id; lightboxIndex.value = -1; loadAll() }
function navigateUp() { if (!folderId.value) return; folderId.value = ''; lightboxIndex.value = -1; loadAll() }

async function createFolder() {
  if (!newFolderName.value.trim()) return
  try {
    await axios.post(`${baseURL}/share/${token.value}/folders`, { name: newFolderName.value.trim(), parent_id: folderId.value || undefined }, { headers: { 'X-Share-Session-Token': sessionToken.value, 'Content-Type': 'application/json' } })
    newFolderName.value = ''; showCreateFolder.value = false
    await fetchSubfolders()
  } catch {}
}

async function deleteFolder(id: string) {
  try {
    await axios.delete(`${baseURL}/share/${token.value}/folders/${id}`, { headers: { 'X-Share-Session-Token': sessionToken.value } })
    subfolders.value = subfolders.value.filter(s => s.id !== id)
  } catch {}
}

async function onFilesSelected(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  const formData = new FormData()
  for (const f of input.files) formData.append('files', f)
  if (folderId.value) formData.append('folder_id', folderId.value)
  uploadProgress.value = 'Uploading...'
  try {
    await axios.post(`${baseURL}/share/${token.value}/upload`, formData, { headers: { 'X-Share-Session-Token': sessionToken.value, 'Content-Type': 'multipart/form-data' } })
    uploadProgress.value = 'Upload complete! Refreshing...'
    setTimeout(async () => { await fetchInfo(); await loadAll(); uploadProgress.value = '' }, 1500)
  } catch (err: any) { uploadProgress.value = err?.response?.data?.error?.message || 'Upload failed' }
  finally { input.value = '' }
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']; let i = 0; let val = bytes
  while (val >= 1024 && i < units.length - 1) { val /= 1024; i++ }
  return val.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function formatDate(dateStr: string): string { return dateStr ? new Date(dateStr).toLocaleDateString() : '' }

function thumbnailSrc(file: FileItem): string {
  const size = file.thumbnails?.sm?.url || file.thumbnails?.md?.url
  if (size) return size
  return `${baseURL}/share/${token.value}/thumb/${file.id}/sm.jpg${sessionToken.value ? '?share_session_token=' + sessionToken.value : ''}`
}

function fileIcon(mime: string): string {
  if (!mime) return '\uD83D\uDCC4'
  if (mime.startsWith('image/')) return '\uD83D\uDDBC'
  if (mime.startsWith('video/')) return '\uD83C\uDFAC'
  if (mime.startsWith('audio/')) return '\uD83C\uDFB5'
  if (mime.includes('pdf')) return '\uD83D\uDCC4'
  return '\uD83D\uDCC4'
}

function downloadFile(file: FileItem) { window.open(`${baseURL}/share/${token.value}/download/${file.id}?share_session_token=${sessionToken.value}`, '_blank') }

async function deleteFile(fileId: string) {
  try {
    await axios.delete(`${baseURL}/share/${token.value}/files/${fileId}`, { headers: { 'X-Share-Session-Token': sessionToken.value } })
    files.value = files.value.filter(f => f.id !== fileId)
  } catch {}
}

onMounted(async () => { await fetchInfo(); nextTick(() => passwordInput.value?.focus()) })
</script>
