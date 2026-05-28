<template>
  <div>
    <h2 class="text-xl font-bold mb-6 text-[var(--text-primary)]">Upload</h2>
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

    <div v-if="jobs.length > 0" class="mt-6" style="background: var(--bg-surface); border-radius: 0.5rem; padding: 1rem">
      <h3 class="text-sm font-semibold mb-3 text-[var(--text-primary)]">Upload Queue</h3>
      <div v-for="job in sortedJobs" :key="job.job_id" class="flex items-center gap-3 py-2 text-sm">
        <span class="truncate flex-1 text-[var(--text-primary)]">{{ job.filename }}</span>
        <div v-if="job.status === 'processing' && job.progress !== undefined" class="w-24 h-1.5 rounded-full overflow-hidden" style="background: var(--bg-elevated)">
          <div class="h-full rounded-full transition-all" style="background: var(--accent)" :style="{ width: (job.progress * 100) + '%' }"></div>
        </div>
        <span class="text-xs w-16 text-right" :class="statusClass(job.status)">{{ statusLabel(job) }}</span>
        <div v-if="job.status === 'failed'" class="flex gap-1">
          <button @click="retryJob(job)" class="px-1.5 py-0.5 rounded text-xs bg-[var(--accent)] text-white">Retry</button>
          <button @click="skipJob(job)" class="px-1.5 py-0.5 rounded text-xs border" style="border-color: var(--border-color); color: var(--text-secondary)">Skip</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

interface Job {
  job_id: string
  filename: string
  status: string
  file_id?: string
  progress?: number
  stage?: string
}

const auth = useAuthStore()
const fileInput = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const jobs = ref<Job[]>([])
const currentBatchID = ref('')

const sortedJobs = computed(() => {
  const order: Record<string, number> = { processing: 0, queued: 1, completed: 2, skipped: 3, failed: 4 }
  return [...jobs.value].sort((a, b) => (order[a.status] ?? 5) - (order[b.status] ?? 5))
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

function statusLabel(job: Job) {
  if (job.status === 'processing' && job.stage) return job.stage
  return job.status
}

function connectWS(batchID: string) {
  const token = auth.accessToken
  if (!token) return

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/upload/ws?token=${token}&batch=${batchID}`)

  ws.onmessage = (event) => {
    const update = JSON.parse(event.data) as Job
    const idx = jobs.value.findIndex(j => j.job_id === update.job_id)
    if (idx >= 0) {
      jobs.value[idx] = { ...jobs.value[idx], ...update }
    }
  }

  ws.onerror = () => {
    console.error('WebSocket error, falling back to polling')
    pollStatus(batchID)
  }
}

async function pollStatus(batchID: string) {
  const interval = setInterval(async () => {
    try {
      const res = await api.get(`/upload/${batchID}/status`)
      for (const j of res.data.jobs) {
        const idx = jobs.value.findIndex(existing => existing.job_id === j.job_id)
        if (idx >= 0) {
          jobs.value[idx] = { ...jobs.value[idx], ...j }
        }
      }
      if (res.data.completed + res.data.failed >= res.data.total) {
        clearInterval(interval)
      }
    } catch {
      clearInterval(interval)
    }
  }, 2000)
}

async function uploadFiles(fileList: FileList | File[]) {
  const form = new FormData()
  for (let i = 0; i < fileList.length; i++) {
    form.append('files', fileList[i])
  }
  try {
    const res = await api.post('/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    const newJobs: Job[] = res.data.jobs.map((j: any) => ({ ...j, progress: 0 }))
    jobs.value = [...newJobs, ...jobs.value]

    if (res.data.batch_id) {
      currentBatchID.value = res.data.batch_id
      connectWS(res.data.batch_id)
    }
  } catch (e: any) {
    console.error('Upload failed', e)
  }
}

function handleDrop(e: DragEvent) {
  dragOver.value = false
  if (e.dataTransfer?.files) uploadFiles(e.dataTransfer.files)
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) uploadFiles(input.files)
}

function skipJob(job: Job) {
  jobs.value = jobs.value.filter(j => j.job_id !== job.job_id)
}

function retryJob(job: Job) {
  job.status = 'queued'
  job.progress = 0
  skipJob(job)
}
</script>
