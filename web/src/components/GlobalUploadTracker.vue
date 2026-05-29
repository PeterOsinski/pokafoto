<template>
  <div
    v-if="upload.isVisible"
    class="fixed bottom-4 right-4 z-50 w-80 shadow-xl rounded-lg overflow-hidden"
    style="background: var(--bg-surface); border: 1px solid var(--border-color)"
  >
    <button
      @click="expanded = !expanded"
      class="flex items-center justify-between w-full px-4 py-2 text-sm cursor-pointer hover:bg-[var(--bg-elevated)]"
    >
      <span class="text-[var(--text-primary)] font-medium">
        Uploads
        <span v-if="upload.hasActiveUploads" class="text-[var(--accent)]">
          ({{ upload.activeJobs.length }} active)
        </span>
        <span v-if="upload.completedCount > 0 && !upload.hasActiveUploads" class="text-[var(--success)]">
          ({{ upload.completedCount }} complete)
        </span>
      </span>
      <span class="text-xs text-[var(--text-secondary)]">
        {{ expanded ? '&#9660;' : '&#9650;' }}
      </span>
    </button>

    <div v-if="expanded" class="px-4 pb-3 max-h-80 overflow-y-auto">
      <div
        v-for="job in sortedJobs"
        :key="job.job_id"
        class="flex items-center gap-3 py-1.5 text-sm"
      >
        <span class="truncate flex-1 text-[var(--text-primary)] text-xs">{{ job.filename }}</span>
        <div
          v-if="(job.status === 'processing' || job.status === 'uploading') && job.progress !== undefined"
          class="w-16 h-1 rounded-full overflow-hidden"
          style="background: var(--bg-elevated)"
        >
          <div
            class="h-full rounded-full transition-all"
            style="background: var(--accent)"
            :style="{ width: (job.progress * 100) + '%' }"
          ></div>
        </div>
        <span class="text-xs w-14 text-right" :class="statusClass(job.status)">
          {{ statusLabel(job) }}
        </span>
        <button
          v-if="job.status === 'failed' || job.status === 'skipped' || job.status === 'completed'"
          @click="upload.removeJob(job.job_id)"
          class="text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
        >
          &#x2715;
        </button>
      </div>

      <button
        v-if="!upload.hasActiveUploads && jobsWithDone.length > 0"
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

const upload = useUploadStore()
const expanded = ref(false)

const jobsWithDone = computed(() =>
  upload.jobs.filter(j => j.status === 'completed' || j.status === 'skipped')
)

const sortedJobs = computed(() => {
  const order: Record<string, number> = {
    uploading: 0, processing: 1, queued: 2,
    failed: 3, completed: 4, skipped: 5,
  }
  return [...upload.jobs]
    .sort((a, b) => (order[a.status] ?? 5) - (order[b.status] ?? 5))
    .slice(0, 20)
})

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
</script>
