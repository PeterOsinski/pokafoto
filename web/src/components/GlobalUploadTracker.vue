<template>
  <div
    v-if="allJobs.length > 0"
    class="fixed bottom-4 right-4 z-50 w-80 shadow-xl rounded-lg overflow-hidden"
    style="background: var(--bg-surface); border: 1px solid var(--border-color)"
  >
    <button
      @click="expanded = !expanded"
      class="flex items-center justify-between w-full px-4 py-2 text-sm cursor-pointer hover:bg-[var(--bg-elevated)]"
    >
      <span class="text-[var(--text-primary)] font-medium">
        Uploads
        <span v-if="activeCount > 0" class="text-[var(--accent)]">
          ({{ activeCount }} active)
        </span>
        <span v-if="completedCount > 0 && activeCount === 0" class="text-[var(--success)]">
          ({{ completedCount }} complete)
        </span>
      </span>
      <span class="text-xs text-[var(--text-secondary)]">
        {{ expanded ? '&#9660;' : '&#9650;' }}
      </span>
    </button>

    <div v-if="expanded" class="px-4 pb-3 max-h-80 overflow-y-auto">
      <div
        v-for="job in sortedJobs"
        :key="job.key"
        class="py-1.5 text-sm"
      >
        <div class="flex items-center gap-3">
          <span class="truncate flex-1 text-[var(--text-primary)] text-xs">{{ job.filename }}</span>
          <div
            v-if="(job.status === 'processing' || job.status === 'uploading' || job.status === 'assembling') && job.progress !== undefined"
            class="w-16 h-1 rounded-full overflow-hidden"
            style="background: var(--bg-elevated)"
          >
            <div
              class="h-full rounded-full transition-all"
              style="background: var(--accent)"
              :style="{ width: Math.min(job.progress * 100, 100) + '%' }"
            ></div>
          </div>
          <span class="text-xs w-14 text-right" :class="statusClass(job.status)">
            {{ statusLabel(job) }}
          </span>
          <button
            v-if="job.status === 'failed' && !job.isChunked"
            @click="upload.retryUpload(job.key)"
            class="text-xs px-1.5 py-0.5 rounded hover:bg-[var(--accent)] hover:text-white shrink-0"
            style="color: var(--accent); border: 1px solid var(--accent)"
          >
            Retry
          </button>
          <button
            v-if="job.status === 'failed' || job.status === 'skipped' || job.status === 'completed'"
            @click="upload.removeJob(job.key); chunked.removeJob(job.key)"
            class="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          >
            &#x2715;
          </button>
        </div>
        <div v-if="job.status === 'failed' && (job.error || job.reason)" class="text-xs mt-0.5 truncate" :class="'text-[var(--error)]'" :title="job.error || job.reason">
          {{ job.error || job.reason }}
        </div>
      </div>

      <button
        v-if="activeCount === 0 && jobsWithDone.length > 0"
        @click="upload.clearCompleted()"
        class="mt-2 text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
      >
        Clear all
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useUploadStore } from '../stores/upload'
import { useChunkedUploadStore } from '../stores/chunkedUpload'

const upload = useUploadStore()
const chunked = useChunkedUploadStore()
const expanded = ref(false)

interface DisplayJob {
  key: string
  filename: string
  status: string
  progress?: number
  stage?: string
  error?: string
  reason?: string
  isChunked: boolean
}

const allJobs = computed<DisplayJob[]>(() => {
  const regular = upload.jobs.map(j => ({
    key: j.job_id,
    filename: j.filename,
    status: j.status,
    progress: j.progress,
    stage: j.stage,
    error: j.error,
    reason: j.reason,
    isChunked: false,
  }))
  const chunkedJobs = chunked.jobs.map(j => ({
    key: j.uploadId,
    filename: j.filename,
    status: j.status,
    progress: j.uploadedBytes / Math.max(j.totalSize, 1),
    stage: j.status === 'assembling' ? 'assembling' : j.status,
    error: j.error,
    reason: undefined,
    isChunked: true,
  }))
  return [...regular, ...chunkedJobs]
})

const jobsWithDone = computed(() =>
  allJobs.value.filter(j => j.status === 'completed' || j.status === 'skipped')
)

const activeCount = computed(() =>
  allJobs.value.filter(j => j.status === 'uploading' || j.status === 'processing' || j.status === 'assembling' || j.status === 'queued').length
)

const completedCount = computed(() =>
  allJobs.value.filter(j => j.status === 'completed' || j.status === 'skipped').length
)

const sortedJobs = computed(() => {
  const order: Record<string, number> = {
    uploading: 0, processing: 1, assembling: 1, queued: 2,
    paused: 3, failed: 4, completed: 5, skipped: 6,
  }
  return [...allJobs.value]
    .sort((a, b) => (order[a.status] ?? 5) - (order[b.status] ?? 5))
    .slice(0, 20)
})

function statusClass(status: string) {
  if (status === 'completed') return 'text-[var(--success)]'
  if (status === 'failed') return 'text-[var(--error)]'
  if (status === 'skipped') return 'text-[var(--warning)]'
  if (status === 'paused') return 'text-[var(--warning)]'
  return 'text-[var(--text-secondary)]'
}

function statusLabel(job: DisplayJob) {
  if (job.status === 'uploading' && job.progress !== undefined) return `${Math.round(job.progress * 100)}%`
  if (job.status === 'processing' && job.stage) return job.stage
  if (job.status === 'assembling') return 'assembling'
  return job.status
}
</script>
