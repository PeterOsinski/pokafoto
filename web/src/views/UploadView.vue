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
        <div v-if="(job.status === 'processing' || job.status === 'uploading') && job.progress !== undefined" class="w-24 h-1.5 rounded-full overflow-hidden" style="background: var(--bg-elevated)">
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

interface Job {
  job_id: string
  filename: string
  status: string
  file_id?: string
  progress?: number
  stage?: string
  size?: number
  uploaded?: number
}

const auth = useAuthStore()
const fileInput = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const jobs = ref<Job[]>([])

let globalWs: WebSocket | null = null

const sortedJobs = computed(() => {
  const order: Record<string, number> = { uploading: 0, processing: 1, queued: 2, completed: 3, skipped: 4, failed: 5 }
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
  if (job.status === 'uploading' && job.progress !== undefined) return `${Math.round(job.progress * 100)}%`
  if (job.status === 'processing' && job.stage) return job.stage
  return job.status
}

function connectGlobalWS() {
  const token = auth.accessToken
  if (!token) return

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  globalWs = new WebSocket(`${protocol}//${window.location.host}/api/v1/upload/ws?token=${token}`)

  globalWs.onmessage = (event) => {
    const update = JSON.parse(event.data) as Job
    const idx = jobs.value.findIndex(j => j.job_id === update.job_id)
    if (idx >= 0) {
      jobs.value[idx] = { ...jobs.value[idx], ...update }
    }
  }

  globalWs.onerror = () => {
    console.error('WebSocket error')
  }

  globalWs.onclose = () => {
    globalWs = null
  }
}

onMounted(() => {
  connectGlobalWS()
})

onUnmounted(() => {
  if (globalWs) {
    globalWs.close()
    globalWs = null
  }
})

async function uploadFiles(fileList: FileList | File[]) {
  const allFiles = Array.from(fileList)
  const fileMap = new Map<string, File>()
  const uploadJobs: Job[] = []

  for (let i = 0; i < allFiles.length; i++) {
    const file = allFiles[i]
    const jobId = `upload-${i}`
    fileMap.set(jobId, file)
    uploadJobs.push({
      job_id: jobId,
      filename: file.name,
      status: 'uploading',
      progress: 0,
      stage: 'uploading',
      size: file.size,
      uploaded: 0,
    })
  }
  jobs.value = [...uploadJobs, ...jobs.value]

  const checkPayload = allFiles.map(f => ({ filename: f.name, size: f.size }))
  try {
    const checkRes = await api.post('/upload/check', checkPayload)
    const duplicates: Set<string> = new Set(
      (checkRes.data.duplicates || []).map((d: { filename: string; file_id: string }) => d.filename)
    )

    for (const job of uploadJobs) {
      if (duplicates.has(job.filename)) {
        const dupInfo = checkRes.data.duplicates.find((d: { filename: string; file_id: string }) => d.filename === job.filename)
        const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
        if (idx >= 0) {
          jobs.value[idx] = {
            ...jobs.value[idx],
            status: 'skipped',
            progress: 1,
            file_id: dupInfo?.file_id,
            stage: undefined,
          }
        }
      }
    }
  } catch {
  }

  const uploadPromises = uploadJobs
    .filter(job => {
      const current = jobs.value.find(j => j.job_id === job.job_id)
      return current?.status === 'uploading'
    })
    .map(async (job) => {
      const file = fileMap.get(job.job_id)
      if (!file) return

      const form = new FormData()
      form.append('files', file)

      try {
        const res = await api.post('/upload', form, {
          headers: { 'Content-Type': 'multipart/form-data' },
          onUploadProgress: (progressEvent) => {
            if (!progressEvent.total) return
            const progress = Math.min(progressEvent.loaded / progressEvent.total, 0.99)
            const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
            if (idx >= 0) {
              jobs.value[idx] = {
                ...jobs.value[idx],
                progress: Math.round(progress * 100) / 100,
                uploaded: Math.round(progress * (job.size || 1)),
              }
            }
          },
        })

        const realJobs: Job[] = res.data.jobs.map((j: any) => ({ ...j, progress: 0 }))
        for (const realJob of realJobs) {
          const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
          if (idx >= 0) {
            jobs.value[idx] = { ...jobs.value[idx], ...realJob, progress: 0 }
          }
        }
      } catch {
        const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
        if (idx >= 0) {
          jobs.value[idx] = {
            ...jobs.value[idx],
            status: 'failed',
            stage: undefined,
          }
        }
      }
    })

  await Promise.allSettled(uploadPromises)
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
