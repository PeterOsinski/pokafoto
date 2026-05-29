import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth'
import api from '../api/client'

export interface UploadJob {
  job_id: string
  filename: string
  status: string
  file_id?: string
  progress?: number
  stage?: string
  size?: number
  uploaded?: number
  reason?: string
  error?: string
}

export interface CompletedJob {
  file_id: string
  filename: string
  folder_id: string | null | undefined
}

export const useUploadStore = defineStore('upload', () => {
  const jobs = ref<UploadJob[]>([])
  const completedJobs = ref<CompletedJob[]>([])
  let globalWs: WebSocket | null = null

  const activeJobs = computed(() => jobs.value.filter(j => j.status !== 'completed' && j.status !== 'skipped' && j.status !== 'failed'))
  const completedCount = computed(() => jobs.value.filter(j => j.status === 'completed').length)
  const hasActiveUploads = computed(() => activeJobs.value.length > 0)
  const isVisible = computed(() => jobs.value.length > 0)

  function connectWS() {
    const auth = useAuthStore()
    const token = auth.accessToken
    if (!token || globalWs) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    globalWs = new WebSocket(`${protocol}//${window.location.host}/api/v1/upload/ws?token=${token}`)

    globalWs.onmessage = (event) => {
      const update = JSON.parse(event.data) as UploadJob & { folder_id?: string | null }
      const idx = jobs.value.findIndex(j => j.job_id === update.job_id)
      if (idx >= 0) {
        jobs.value[idx] = { ...jobs.value[idx], ...update }
      }
      if ((update.status === 'completed' || (update.status === 'skipped' && update.file_id)) && update.file_id) {
        completedJobs.value.push({
          file_id: update.file_id,
          filename: update.filename,
          folder_id: update.folder_id,
        })
      }
    }

    globalWs.onclose = () => {
      globalWs = null
    }
  }

  function disconnectWS() {
    if (globalWs) {
      globalWs.close()
      globalWs = null
    }
  }

  function addJob(job: UploadJob) {
    jobs.value = [job, ...jobs.value]
  }

  function removeJob(jobId: string) {
    jobs.value = jobs.value.filter(j => j.job_id !== jobId)
  }

  function clearCompleted() {
    jobs.value = jobs.value.filter(j => j.status === 'uploading' || j.status === 'processing' || j.status === 'queued' || j.status === 'failed')
  }

  async function uploadFiles(fileList: FileList | File[], targetFolderId: string | null, skipNameSizeDedup: boolean) {
    const allFiles = Array.from(fileList)
    const fileMap = new Map<string, File>()
    const uploadJobs: UploadJob[] = []

    for (let i = 0; i < allFiles.length; i++) {
      const file = allFiles[i]
      const jobId = `upload-${Date.now()}-${i}`
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

    if (!skipNameSizeDedup) {
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
        // Continue with upload even if dedup check fails
      }
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
        if (targetFolderId) {
          form.append('folder_id', targetFolderId)
        }
        if (skipNameSizeDedup) {
          form.append('skip_name_size_dedup', 'true')
        }

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

          const realJobs: UploadJob[] = res.data.jobs.map((j: any) => ({ ...j, progress: 0 }))
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

  function consumeCompletedJobs(): CompletedJob[] {
    const drained = [...completedJobs.value]
    completedJobs.value = []
    return drained
  }

  return {
    jobs,
    completedJobs,
    activeJobs,
    completedCount,
    hasActiveUploads,
    isVisible,
    connectWS,
    disconnectWS,
    addJob,
    removeJob,
    clearCompleted,
    consumeCompletedJobs,
    uploadFiles,
  }
})
