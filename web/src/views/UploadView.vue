<template>
  <div>
    <h2 class="text-xl font-bold mb-6 text-[var(--text-primary)]">Upload</h2>

    <div class="mb-4">
      <label class="block text-sm text-[var(--text-secondary)] mb-2">Target folder</label>
      <div class="flex items-center gap-2 flex-wrap">
        <button
          @click="targetFolderId = null"
          class="px-3 py-1.5 rounded text-sm transition-colors"
          :class="targetFolderId === null ? 'text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'"
          :style="targetFolderId === null ? { background: 'var(--accent)' } : { background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }"
        >
          Auto-organize by date
        </button>
        <button
          v-for="f in recentFolders"
          :key="f.id"
          @click="targetFolderId = f.id"
          class="px-3 py-1.5 rounded text-sm transition-colors"
          :class="targetFolderId === f.id ? 'text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'"
          :style="targetFolderId === f.id ? { background: 'var(--accent)' } : { background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }"
        >
          {{ f.name }}
        </button>
        <button
          @click="showFolderPicker = true"
          class="px-3 py-1.5 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          style="background: var(--bg-elevated); border: 1px solid var(--border-color)"
        >
          Browse...
        </button>
      </div>
    </div>

    <div
      class="border-2 border-dashed rounded-xl p-12 text-center transition-colors"
      :class="dragOver ? 'border-[var(--accent)]' : 'border-[var(--border-color)]'"
      style="background: var(--bg-surface)"
      @dragover.prevent="dragOver = true"
      @dragleave="dragOver = false"
      @drop.prevent="handleDrop"
    >
      <p class="text-lg text-[var(--text-primary)]">Drop files here</p>
      <p class="text-sm text-[var(--text-secondary)] mt-2">or click to browse</p>
      <input
        ref="fileInput"
        type="file"
        multiple
        class="hidden"
        @change="handleFileSelect"
      />
      <button @click="triggerFileInput" class="mt-4 px-6 py-2 rounded-md text-white" style="background: var(--accent)">
        Select Files
      </button>
    </div>

    <div v-if="upload.jobs.length > 0" class="mt-6" style="background: var(--bg-surface); border-radius: 0.5rem; padding: 1rem">
      <h3 class="text-sm font-semibold mb-3 text-[var(--text-primary)]">Upload Queue</h3>
      <div v-for="job in sortedJobs" :key="job.job_id" class="flex items-center gap-3 py-2 text-sm" data-testid="upload-queue-item">
        <span class="truncate flex-1 text-[var(--text-primary)]">{{ job.filename }}</span>
        <div v-if="(job.status === 'processing' || job.status === 'uploading') && job.progress !== undefined" class="w-24 h-1.5 rounded-full overflow-hidden" style="background: var(--bg-elevated)">
          <div class="h-full rounded-full transition-all" style="background: var(--accent)" :style="{ width: (job.progress * 100) + '%' }"></div>
        </div>
        <span class="text-xs w-16 text-right" :class="statusClass(job.status)">{{ statusLabel(job) }}</span>
        <div v-if="job.status === 'failed'" class="flex gap-1">
          <button @click="upload.removeJob(job.job_id)" class="px-1.5 py-0.5 rounded text-xs border" style="border-color: var(--border-color); color: var(--text-secondary)">Remove</button>
        </div>
      </div>
    </div>

    <FolderPickerDialog
      :open="showFolderPicker"
      title="Select target folder"
      actionLabel="Select"
      @close="showFolderPicker = false"
      @confirm="(folderId) => { targetFolderId = folderId; showFolderPicker = false }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUploadStore } from '../stores/upload'
import api from '../api/client'
import FolderPickerDialog from '../components/FolderPickerDialog.vue'

interface FolderInfo {
  id: string
  name: string
  parent_id: string | null
}

const upload = useUploadStore()
const fileInput = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const targetFolderId = ref<string | null>(null)
const recentFolders = ref<FolderInfo[]>([])
const showFolderPicker = ref(false)

const sortedJobs = computed(() => {
  const order: Record<string, number> = { uploading: 0, processing: 1, queued: 2, completed: 3, skipped: 4, failed: 5 }
  return [...upload.jobs].sort((a, b) => (order[a.status] ?? 5) - (order[b.status] ?? 5))
})

function triggerFileInput() {
  fileInput.value?.click()
}

function statusClass(status: string) {
  if (status === 'completed') return 'text-[var(--success)]'
  if (status === 'failed') return 'text-[var(--error)]'
  if (status === 'skipped') return 'text-[var(--warning)]'
  return 'text-[var(--text-secondary)]'
}

function statusLabel(job: { status: string; progress?: number; stage?: string }) {
  if (job.status === 'uploading' && job.progress !== undefined) return `${Math.round(job.progress * 100)}%`
  if (job.status === 'processing' && job.stage) return job.stage
  return job.status
}

async function loadFolders() {
  try {
    const res = await api.get('/folders')
    const flatten = (nodes: any[]): FolderInfo[] => {
      let result: FolderInfo[] = []
      for (const n of nodes) {
        result.push({ id: n.folder.id, name: n.folder.name, parent_id: n.folder.parent_id })
        result = result.concat(flatten(n.children || []))
      }
      return result
    }
    recentFolders.value = flatten(res.data.children || []).slice(0, 6)
  } catch {}
}

onMounted(() => {
  loadFolders()
})

function handleDrop(e: DragEvent) {
  dragOver.value = false
  if (e.dataTransfer?.files) {
    upload.uploadFiles(e.dataTransfer.files, targetFolderId.value, !!targetFolderId.value)
  }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) {
    upload.uploadFiles(input.files, targetFolderId.value, !!targetFolderId.value)
  }
}
</script>
