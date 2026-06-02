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
        <template v-if="!expanded && hasActive && activeCount > 0">
          {{ Math.round(aggregateProgress * 100) }}% &middot; {{ activeCount }}/{{ allJobs.length }} files
        </template>
        <template v-else-if="!expanded && completedCount > 0">
          {{ completedCount }} done &#x2713;
        </template>
        <template v-else>Uploads</template>
      </span>
      <span v-if="totalUploadSize > 0" class="text-xs text-[var(--text-secondary)]">
        {{ formatBytes(uploadedTotalSize) }} / {{ formatBytes(totalUploadSize) }}
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
          <div class="flex-1 min-w-0">
            <span class="truncate block text-[var(--text-primary)] text-xs">{{ job.filename }}</span>
            <span class="text-[var(--text-secondary)] block" style="font-size: 0.625rem">{{ formatBytes(job.uploadedBytes || 0) }} / {{ formatBytes(job.totalSize) }}</span>
          </div>
          <div
            v-if="(job.status === 'processing' || job.status === 'uploading' || job.status === 'assembling') && job.progress !== undefined"
            class="w-16 h-1 rounded-full overflow-hidden shrink-0"
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
            v-if="job.status === 'paused' && job.isChunked"
            @click="chunked.resumePausedJob(job.key)"
            class="text-xs px-1.5 py-0.5 rounded hover:bg-[var(--accent)] hover:text-white shrink-0"
            style="color: var(--accent); border: 1px solid var(--accent)"
          >
            Restart
          </button>
          <button
            v-if="job.status === 'failed' && job.error !== 'upload_expired'"
            @click="chunked.retryUpload(job.key)"
            class="text-xs px-1.5 py-0.5 rounded hover:bg-[var(--accent)] hover:text-white shrink-0"
            style="color: var(--accent); border: 1px solid var(--accent)"
          >
            Retry
          </button>
          <button
            v-if="job.status === 'failed' || job.status === 'completed' || job.status === 'paused' || job.status === 'uploading'"
            @click="chunked.removeJob(job.key)"
            class="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          >
            &#x2715;
          </button>
        </div>
        <div v-if="job.error" class="text-xs mt-0.5 truncate text-[var(--error)]" :title="job.error">
          {{ job.error }}
        </div>
      </div>

      <div class="flex flex-wrap gap-2 mt-3">
        <button
          v-if="pausedCount > 0"
          @click="chunked.resumeAllPausedJobs()"
          class="text-xs px-2 py-1 rounded hover:bg-[var(--accent)] hover:text-white"
          style="color: var(--accent); border: 1px solid var(--accent)"
        >
          Restart all
        </button>
        <button
          v-if="pendingCount > 0"
          @click="chunked.abortAll()"
          class="text-xs px-2 py-1 rounded hover:bg-[var(--error)] hover:text-white"
          style="color: var(--error); border: 1px solid var(--error)"
        >
          Abort all
        </button>
        <button
          v-if="finishedCount > 0 && activeCount === 0"
          @click="chunked.clearCompleted()"
          class="text-xs px-2 py-1 rounded text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          style="border: 1px solid var(--border-color)"
        >
          Clear all
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useChunkedUploadStore } from '../stores/chunkedUpload'

const chunked = useChunkedUploadStore()
const expanded = ref(false)

interface DisplayJob {
  key: string
  filename: string
  status: string
  progress?: number
  error?: string
  isChunked: boolean
  totalSize: number
  uploadedBytes: number
}

const allJobs = computed<DisplayJob[]>(() =>
  chunked.jobs.map(j => ({
    key: j.uploadId,
    filename: j.filename,
    status: j.status,
    progress: j.totalSize > 0 ? j.uploadedBytes / j.totalSize : 0,
    error: j.error,
    isChunked: true,
    totalSize: j.totalSize,
    uploadedBytes: j.uploadedBytes,
  })),
)

const activeCount = computed(() =>
  allJobs.value.filter(j =>
    j.status === 'uploading' || j.status === 'processing' ||
    j.status === 'assembling',
  ).length,
)

const completedCount = computed(() =>
  allJobs.value.filter(j => j.status === 'completed').length,
)

const pausedCount = computed(() =>
  allJobs.value.filter(j => j.status === 'paused').length,
)

const pendingCount = computed(() =>
  allJobs.value.filter(j =>
    j.status === 'uploading' || j.status === 'processing' ||
    j.status === 'assembling' || j.status === 'paused',
  ).length,
)

const finishedCount = computed(() =>
  allJobs.value.filter(j =>
    j.status === 'completed' || j.status === 'failed',
  ).length,
)

const hasActive = computed(() => activeCount.value > 0)

const aggregateProgress = computed(() => {
  const activeJobs = allJobs.value.filter(j =>
    j.status === 'uploading' || j.status === 'processing' || j.status === 'assembling',
  )
  if (activeJobs.length === 0) return 0
  const totalBytes = activeJobs.reduce((sum, j) => sum + j.totalSize, 0)
  const uploadedBytes = activeJobs.reduce((sum, j) => sum + j.uploadedBytes, 0)
  return totalBytes > 0 ? uploadedBytes / totalBytes : 0
})

const totalUploadSize = computed(() =>
  allJobs.value.reduce((sum, j) => sum + j.totalSize, 0),
)

const uploadedTotalSize = computed(() =>
  allJobs.value.reduce((sum, j) => sum + j.uploadedBytes, 0),
)

const sortedJobs = computed(() => {
  const order: Record<string, number> = {
    uploading: 0, processing: 1, assembling: 1,
    paused: 2, failed: 3, completed: 4,
  }
  return [...allJobs.value]
    .sort((a, b) => (order[a.status] ?? 5) - (order[b.status] ?? 5))
    .slice(0, 20)
})

function statusClass(status: string) {
  if (status === 'completed') return 'text-[var(--success)]'
  if (status === 'failed') return 'text-[var(--error)]'
  if (status === 'paused') return 'text-[var(--warning)]'
  return 'text-[var(--text-secondary)]'
}

function statusLabel(job: DisplayJob) {
  if (job.status === 'uploading' && job.progress !== undefined) return `${Math.round(job.progress * 100)}%`
  if (job.status === 'assembling') return 'assembling'
  return job.status
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return val.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}
</script>
