<template>
  <div class="min-h-screen bg-[var(--bg-primary)] p-6">
    <!-- Loading -->
    <div v-if="loading" class="max-w-md mx-auto mt-20 text-center text-[var(--text-secondary)]">
      Loading shared folder...
    </div>

    <!-- Error -->
    <div v-else-if="error" class="max-w-md mx-auto mt-20 text-center">
      <h1 class="text-xl font-bold text-[var(--text-primary)] mb-2">Not Available</h1>
      <p class="text-[var(--text-secondary)]">{{ error }}</p>
    </div>

    <!-- Password unlock step -->
    <div v-else-if="needsPassword && !sessionToken" class="max-w-md mx-auto mt-20">
      <h1 class="text-xl font-bold text-[var(--text-primary)] mb-2">Password Required</h1>
      <p class="text-[var(--text-secondary)] mb-4">This shared folder is password-protected.</p>
      <input
        ref="passwordInput"
        v-model="password"
        type="password"
        placeholder="Enter password..."
        class="w-full px-3 py-2 rounded text-sm mb-3"
        style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
        @keyup.enter="unlock"
      />
      <p v-if="unlockError" class="text-sm text-[var(--error)] mb-3">{{ unlockError }}</p>
      <button
        @click="unlock"
        :disabled="!password.trim() || unlocking"
        class="w-full px-4 py-2 rounded text-sm text-white disabled:opacity-50"
        style="background: var(--accent)"
      >
        {{ unlocking ? 'Unlocking...' : 'Unlock' }}
      </button>
    </div>

    <!-- Browse shared folder -->
    <div v-else-if="sessionToken">
      <header class="flex items-center justify-between mb-6 flex-wrap gap-3">
        <div>
          <h1 class="text-xl font-bold text-[var(--text-primary)]">{{ folderName }}</h1>
          <span class="text-xs text-[var(--text-secondary)]">
            Shared folder
            <span v-if="permissions === 'read_upload'"> &middot; Can upload</span>
            <span v-else-if="permissions === 'read_write'"> &middot; Can upload &amp; delete</span>
          </span>
        </div>
        <div class="text-xs text-[var(--text-secondary)]">
          <span v-if="uploadLimit !== null">{{ formatBytes(uploadedBytes) }} / {{ formatBytes(uploadLimit) }}</span>
          <span v-if="expiresAt" class="ml-3">Expires {{ formatDate(expiresAt) }}</span>
        </div>
      </header>

      <!-- File grid -->
      <div v-if="files.length === 0 && !filesLoaded" class="text-center py-10 text-[var(--text-secondary)]">
        Loading files...
      </div>
      <div v-else-if="files.length === 0" class="text-center py-10 text-[var(--text-secondary)]">
        This folder is empty.
      </div>
      <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
        <div
          v-for="file in files"
          :key="file.id"
          class="flex flex-col rounded-lg overflow-hidden cursor-pointer hover:ring-2 hover:ring-[var(--accent)] transition-all relative group"
          style="background: var(--bg-elevated); border: 1px solid var(--border-color)"
        >
          <div class="aspect-square overflow-hidden bg-black/20">
            <img
              v-if="file.thumbnails?.sm?.url"
              :src="thumbnailSrc(file)"
              :alt="file.original_name"
              class="w-full h-full object-cover"
              loading="lazy"
            />
            <div v-else class="w-full h-full flex items-center justify-center text-3xl text-[var(--text-secondary)]">
              {{ fileIcon(file.mime_type) }}
            </div>
          </div>
          <div class="p-2">
            <span class="text-xs text-[var(--text-primary)] truncate block" :title="file.original_name">
              {{ file.original_name }}
            </span>
            <span class="text-xs text-[var(--text-secondary)]">{{ formatBytes(file.size_bytes) }}</span>
          </div>
          <button
            v-if="canDelete"
            @click.stop="deleteFile(file.id)"
            class="absolute top-1 right-1 w-6 h-6 rounded-full flex items-center justify-center text-xs text-white opacity-0 group-hover:opacity-100 transition-opacity"
            style="background: #ef4444"
            title="Delete"
          >
            &times;
          </button>
          <button
            @click.stop="downloadFile(file)"
            class="absolute top-1 left-1 w-6 h-6 rounded-full flex items-center justify-center text-xs text-white opacity-0 group-hover:opacity-100 transition-opacity"
            style="background: var(--accent)"
            title="Download"
          >
            &#x2193;
          </button>
        </div>
      </div>

      <!-- Upload zone -->
      <div v-if="canUpload" class="mt-8 p-6 border-2 border-dashed rounded-lg text-center" style="border-color: var(--border-color)">
        <p class="text-sm text-[var(--text-secondary)] mb-3">Drag files here or click to upload</p>
        <input
          type="file"
          multiple
          class="hidden"
          ref="fileInput"
          @change="onFilesSelected"
        />
        <button
          @click="fileInput?.click()"
          class="px-4 py-2 rounded text-sm text-white"
          style="background: var(--accent)"
        >
          Select Files
        </button>
        <div v-if="uploadProgress" class="mt-3 text-sm text-[var(--text-secondary)]">{{ uploadProgress }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'

interface FileItem {
  id: string
  original_name: string
  size_bytes: number
  mime_type: string
  media_type: string
  width?: number
  height?: number
  created_at: string
  thumbnails?: { sm?: { url: string }; md?: { url: string } }
}

const route = useRoute()
const token = ref((route.params.token as string) || '')
const folderName = ref('')
const permissions = ref('read')
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
const uploadProgress = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const passwordInput = ref<HTMLInputElement | null>(null)

const baseURL = '/api/v1'

async function fetchInfo() {
  loading.value = true
  error.value = ''
  try {
    const res = await axios.get(`${baseURL}/share/${token.value}`)
    const d = res.data
    folderName.value = d.folder_name || 'Shared Folder'
    permissions.value = d.permissions || 'read'
    uploadLimit.value = d.upload_limit_bytes || null
    uploadedBytes.value = d.uploaded_bytes || 0
    needsPassword.value = d.needs_password || false
    expiresAt.value = d.expires_at || null

    canUpload.value = permissions.value === 'read_upload' || permissions.value === 'read_write'
    canDelete.value = permissions.value === 'read_write'

    if (!d.needs_password) {
      await unlockWithoutPassword()
    }
  } catch (err: any) {
    if (err.response?.status === 410) error.value = 'This share link has expired.'
    else if (err.response?.status === 404) error.value = 'Share link not found.'
    else error.value = 'Failed to load shared folder.'
  } finally {
    loading.value = false
  }
}

async function unlockWithoutPassword() {
  try {
    const res = await axios.post(`${baseURL}/share/${token.value}/unlock`, {})
    sessionToken.value = res.data.share_session_token
    await fetchFiles()
  } catch {}
}

async function unlock() {
  if (!password.value.trim()) return
  unlocking.value = true
  unlockError.value = ''
  try {
    const res = await axios.post(`${baseURL}/share/${token.value}/unlock`, { password: password.value })
    sessionToken.value = res.data.share_session_token
    await fetchFiles()
  } catch (err: any) {
    unlockError.value = err?.response?.data?.error?.message || 'Invalid password'
  } finally {
    unlocking.value = false
  }
}

async function fetchFiles() {
  try {
    const res = await axios.get(`${baseURL}/share/${token.value}/files`, {
      headers: { 'X-Share-Session-Token': sessionToken.value },
    })
    files.value = res.data.items || []
    filesLoaded.value = true
  } catch {
    files.value = []
  }
}

function thumbnailSrc(file: FileItem): string {
  const size = file.thumbnails?.sm?.url || file.thumbnails?.md?.url
  if (size) return size
  const thumbUrl = `${baseURL}/share/${token.value}/thumb/${file.id}/sm.jpg`
  return `${thumbUrl}${sessionToken.value ? '?share_session_token=' + sessionToken.value : ''}`
}

function fileIcon(mime: string): string {
  if (!mime) return '\uD83D\uDCC4'
  if (mime.startsWith('image/')) return '\uD83D\uDDBC'
  if (mime.startsWith('video/')) return '\uD83C\uDFAC'
  if (mime.startsWith('audio/')) return '\uD83C\uDFB5'
  if (mime.includes('pdf')) return '\uD83D\uDCC4'
  return '\uD83D\uDCC4'
}

async function downloadFile(file: FileItem) {
  const url = `${baseURL}/share/${token.value}/download/${file.id}?share_session_token=${sessionToken.value}`
  window.open(url, '_blank')
}

async function deleteFile(fileId: string) {
  try {
    await axios.delete(`${baseURL}/share/${token.value}/files/${fileId}`, {
      headers: { 'X-Share-Session-Token': sessionToken.value },
    })
    files.value = files.value.filter(f => f.id !== fileId)
  } catch {}
}

async function onFilesSelected(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return

  const formData = new FormData()
  for (const f of input.files) {
    formData.append('files', f)
  }

  uploadProgress.value = 'Uploading...'
  try {
    await axios.post(`${baseURL}/share/${token.value}/upload`, formData, {
      headers: {
        'X-Share-Session-Token': sessionToken.value,
        'Content-Type': 'multipart/form-data',
      },
    })
    uploadProgress.value = 'Upload complete! Refreshing...'
    setTimeout(async () => {
      await fetchInfo()
      await fetchFiles()
      uploadProgress.value = ''
    }, 1500)
  } catch (err: any) {
    uploadProgress.value = err?.response?.data?.error?.message || 'Upload failed'
  } finally {
    input.value = ''
  }
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) { val /= 1024; i++ }
  return val.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString()
}

onMounted(async () => {
  await fetchInfo()
  nextTick(() => passwordInput.value?.focus())
})
</script>
