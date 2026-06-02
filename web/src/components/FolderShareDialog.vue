<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="$emit('close')"
    >
      <div
        class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-lg mx-4 max-h-[80vh] overflow-y-auto"
        style="border: 1px solid var(--border-color)"
      >
        <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-4">Share Folder: {{ folderName }}</h3>

        <div class="mb-6 p-4 rounded-lg" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
          <h4 class="text-sm font-medium text-[var(--text-primary)] mb-3">Create Share Link</h4>

          <div class="grid gap-3">
            <div>
              <label class="text-xs text-[var(--text-secondary)] block mb-1">Permissions</label>
              <select
                v-model="newShare.permissions"
                class="w-full px-3 py-2 rounded text-sm"
                style="background: var(--bg-primary); color: var(--text-primary); border: 1px solid var(--border-color)"
              >
                <option value="read">Read only</option>
                <option value="read_upload">Read + Upload</option>
                <option value="read_write">Read + Upload + Delete</option>
              </select>
            </div>

            <div>
              <label class="flex items-center gap-2 text-xs text-[var(--text-secondary)] cursor-pointer select-none">
                <input type="checkbox" v-model="newShare.includeSubdirs" class="accent-[var(--accent)] rounded" />
                Include subdirectories
              </label>
            </div>

            <div v-if="newShare.permissions !== 'read'">
              <label class="text-xs text-[var(--text-secondary)] block mb-1">Upload Limit (MB, optional)</label>
              <input
                v-model.number="newShare.uploadLimitMb"
                type="number"
                min="1"
                placeholder="Unlimited"
                class="w-full px-3 py-2 rounded text-sm"
                style="background: var(--bg-primary); color: var(--text-primary); border: 1px solid var(--border-color)"
              />
            </div>

            <div>
              <label class="text-xs text-[var(--text-secondary)] block mb-1">Password (optional)</label>
              <input
                v-model="newShare.password"
                type="text"
                placeholder="No password"
                class="w-full px-3 py-2 rounded text-sm"
                style="background: var(--bg-primary); color: var(--text-primary); border: 1px solid var(--border-color)"
              />
            </div>

            <div>
              <label class="text-xs text-[var(--text-secondary)] block mb-1">Expiration (optional)</label>
              <input
                v-model="newShare.expiresAt"
                type="datetime-local"
                class="w-full px-3 py-2 rounded text-sm"
                style="background: var(--bg-primary); color: var(--text-primary); border: 1px solid var(--border-color)"
              />
            </div>

            <p v-if="createError" class="text-sm text-[var(--error)]">{{ createError }}</p>

            <button
              @click="createShare"
              :disabled="creating"
              class="px-4 py-2 rounded text-sm text-white disabled:opacity-50 w-full"
              style="background: var(--accent)"
            >
              {{ creating ? 'Creating...' : 'Create Share Link' }}
            </button>
          </div>
        </div>

        <div v-if="shares.length > 0">
          <h4 class="text-sm font-medium text-[var(--text-primary)] mb-3">Active Share Links</h4>
          <div
            v-for="share in shares"
            :key="share.id"
            class="flex items-center justify-between py-3 px-3 rounded-lg mb-2"
            style="background: var(--bg-elevated); border: 1px solid var(--border-color)"
          >
            <div class="flex-1 min-w-0 mr-3">
              <div class="flex items-center gap-2 mb-1">
                <span
                  class="text-xs px-2 py-0.5 rounded font-medium"
                  :style="{ background: permissionColor(share.permissions), color: '#fff' }"
                >
                  {{ share.permissions }}
                </span>
                <span v-if="share.has_password" class="text-xs">&#x1F512;</span>
                <span v-if="share.include_subdirs" class="text-xs text-[var(--text-secondary)]">subdirs</span>
              </div>
              <span class="text-xs text-[var(--text-secondary)] block truncate">
                /share/{{ share.token.substring(0, 8) }}...
                <span v-if="share.upload_limit_bytes"> | Limit: {{ formatBytes(share.upload_limit_bytes) }}</span>
                <span v-if="share.uploaded_bytes !== undefined"> | Used: {{ formatBytes(share.uploaded_bytes) }}</span>
              </span>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <button
                @click="copyLink(share.token)"
                class="px-2 py-1 rounded text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-white/5"
                title="Copy link"
              >
                Copy
              </button>
              <button
                @click="editShare(share)"
                class="px-2 py-1 rounded text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-white/5"
                title="Edit"
              >
                Edit
              </button>
              <button
                @click="confirmDeleteShare(share.id)"
                class="px-2 py-1 rounded text-xs text-red-400 hover:text-red-300 hover:bg-red-500/10"
                title="Delete"
              >
                Delete
              </button>
            </div>
          </div>
        </div>

        <div v-if="editingShare" class="mt-4 p-4 rounded-lg" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
          <h4 class="text-sm font-medium text-[var(--text-primary)] mb-3">Edit Share</h4>
          <div class="grid gap-3">
            <div>
              <label class="text-xs text-[var(--text-secondary)] block mb-1">Permissions</label>
              <select v-model="editForm.permissions" class="w-full px-3 py-2 rounded text-sm" style="background: var(--bg-primary); color: var(--text-primary); border: 1px solid var(--border-color)">
                <option value="read">Read only</option>
                <option value="read_upload">Read + Upload</option>
                <option value="read_write">Read + Upload + Delete</option>
              </select>
            </div>
            <button @click="updateShare" :disabled="updating" class="px-4 py-2 rounded text-sm text-white disabled:opacity-50" style="background: var(--accent)">
              {{ updating ? 'Updating...' : 'Save' }}
            </button>
            <button @click="editingShare = null" class="px-4 py-2 rounded text-sm text-[var(--text-secondary)]">Cancel</button>
          </div>
        </div>

        <div class="mt-4 flex justify-end">
          <button
            @click="$emit('close')"
            class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api/client'

const props = defineProps<{
  visible: boolean
  folderId: string
  folderName: string
}>()

const emit = defineEmits<{
  close: []
}>()

interface Share {
  id: string
  token: string
  permissions: string
  include_subdirs?: boolean
  upload_limit_bytes?: number | null
  uploaded_bytes?: number
  has_password: boolean
  expires_at?: string | null
}

const shares = ref<Share[]>([])
const creating = ref(false)
const updating = ref(false)
const createError = ref('')
const editingShare = ref<Share | null>(null)

const newShare = ref({
  permissions: 'read',
  includeSubdirs: false,
  uploadLimitMb: null as number | null,
  password: '',
  expiresAt: '',
})

const editForm = ref({ permissions: 'read' })

onMounted(async () => {
  await fetchShares()
})

async function fetchShares() {
  try {
    const res = await api.get(`/folders/${props.folderId}/shares`)
    shares.value = res.data.shares || []
  } catch {}
}

async function createShare() {
  creating.value = true
  createError.value = ''
  try {
    const body: any = { permissions: newShare.value.permissions, include_subdirs: newShare.value.includeSubdirs }
    if (newShare.value.uploadLimitMb) {
      body.upload_limit_bytes = newShare.value.uploadLimitMb * 1024 * 1024
    }
    if (newShare.value.password) body.password = newShare.value.password
    if (newShare.value.expiresAt) body.expires_at = newShare.value.expiresAt + ':00Z'

    await api.post(`/folders/${props.folderId}/shares`, body)
    newShare.value = { permissions: 'read', includeSubdirs: false, uploadLimitMb: null, password: '', expiresAt: '' }
    await fetchShares()
  } catch (err: any) {
    createError.value = err?.response?.data?.error?.message || 'Failed to create share'
  } finally {
    creating.value = false
  }
}

async function updateShare() {
  if (!editingShare.value) return
  updating.value = true
  try {
    await api.put(`/folders/${props.folderId}/shares/${editingShare.value.id}`, {
      permissions: editForm.value.permissions,
    })
    editingShare.value = null
    await fetchShares()
  } catch {} finally {
    updating.value = false
  }
}

async function confirmDeleteShare(shareId: string) {
  try {
    await api.delete(`/folders/${props.folderId}/shares/${shareId}`)
    await fetchShares()
  } catch {}
}

function copyLink(token: string) {
  const url = `${window.location.origin}/share/${token}`
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(url).catch(() => fallbackCopy(url))
  } else {
    fallbackCopy(url)
  }
}

function fallbackCopy(text: string) {
  const area = document.createElement('textarea')
  area.value = text
  area.style.position = 'fixed'
  area.style.left = '-9999px'
  document.body.appendChild(area)
  area.select()
  try { document.execCommand('copy') } catch {}
  document.body.removeChild(area)
}

function editShare(share: Share) {
  editingShare.value = share
  editForm.value = { permissions: share.permissions }
}

function permissionColor(perm: string): string {
  switch (perm) {
    case 'read': return '#3b82f6'
    case 'read_upload': return '#f59e0b'
    case 'read_write': return '#ef4444'
    default: return '#6b7280'
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
</script>
