import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth'
import api from '../api/client'

export interface UploadJob {
  job_id: string
  batch_id?: string
  filename: string
  status: string
  file_id?: string
  progress?: number
  stage?: string
  size?: number
  uploaded?: number
  reason?: string
  error?: string
  folder_id?: string | null
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
  let reconnectAttempts = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let wsManualClose = false

  const pendingUpdates = new Map<string, Partial<UploadJob>>()

  const activeJobs = computed(() => jobs.value.filter(j => j.status !== 'completed' && j.status !== 'skipped' && j.status !== 'failed'))
  const completedCount = computed(() => jobs.value.filter(j => j.status === 'completed').length)
  const hasActiveUploads = computed(() => activeJobs.value.length > 0)
  const isVisible = computed(() => jobs.value.length > 0)

  function scheduleReconnect() {
    if (reconnectTimer) return
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000)
    reconnectAttempts++
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connectWS()
    }, delay)
  }

  function connectWS() {
    const auth = useAuthStore()
    const token = auth.accessToken
    if (!token || globalWs) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    globalWs = new WebSocket(`${protocol}//${window.location.host}/api/v1/upload/ws?token=${token}`)

    globalWs.onopen = () => {
      reconnectAttempts = 0
    }

    globalWs.onmessage = (event) => {
      const update = JSON.parse(event.data) as UploadJob & { folder_id?: string | null }
      const idx = jobs.value.findIndex(j => j.job_id === update.job_id)
      if (idx >= 0) {
        jobs.value[idx] = { ...jobs.value[idx], ...update }
      } else {
        const existing = pendingUpdates.get(update.job_id) || {}
        pendingUpdates.set(update.job_id, { ...existing, ...update })
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
      if (!wsManualClose) {
        scheduleReconnect()
      }
    }
  }

  function disconnectWS() {
    wsManualClose = true
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (globalWs) {
      globalWs.close()
      globalWs = null
    }
  }

  function addJob(job: UploadJob) {
    const pending = pendingUpdates.get(job.job_id)
    if (pending) {
      pendingUpdates.delete(job.job_id)
      job = { ...job, ...pending }
    }
    jobs.value = [job, ...jobs.value]
  }

  function removeJob(jobId: string) {
    jobs.value = jobs.value.filter(j => j.job_id !== jobId)
    pendingUpdates.delete(jobId)
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

          const batchId = res.data.batch_id as string
          const realJobs: UploadJob[] = res.data.jobs.map((j: any) => ({
            ...j,
            batch_id: batchId,
            progress: 0,
          }))
          for (const realJob of realJobs) {
            const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
            if (idx >= 0) {
              const pending = pendingUpdates.get(realJob.job_id)
              if (pending) {
                pendingUpdates.delete(realJob.job_id)
                jobs.value[idx] = { ...jobs.value[idx], ...realJob, ...pending }
              } else {
                jobs.value[idx] = { ...jobs.value[idx], ...realJob }
              }
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

  function reconcileStuckJobs() {
    const stuckJobs = jobs.value.filter(j => j.status === 'queued' || j.status === 'processing')
    if (stuckJobs.length === 0) return

    const batchIds = new Set(stuckJobs.map(j => j.batch_id).filter(Boolean) as string[])
    for (const batchId of batchIds) {
      api.get(`/upload/${batchId}/status`)
        .then(res => {
          const batchJobs: any[] = res.data.jobs || []
          for (const bj of batchJobs) {
            const idx = jobs.value.findIndex(j => j.job_id === bj.job_id)
            if (idx >= 0) {
              const current = jobs.value[idx]
              if (current.status === bj.status) continue
              jobs.value[idx] = { ...current, status: bj.status, file_id: bj.file_id, reason: bj.reason, error: bj.error }
              if ((bj.status === 'completed' || (bj.status === 'skipped' && bj.file_id)) && bj.file_id) {
                completedJobs.value.push({
                  file_id: bj.file_id,
                  filename: current.filename,
                  folder_id: current.folder_id ?? undefined,
                })
              }
            }
          }
        })
        .catch(() => {
          // REST fallback is best-effort, ignore errors
        })
    }
  }

  let reconcileInterval: ReturnType<typeof setInterval> | null = null

  function startReconcile() {
    if (reconcileInterval) return
    reconcileInterval = setInterval(reconcileStuckJobs, 5000)
  }

  startReconcile()
  setInterval(() => {
    if (pendingUpdates.size > 100) {
      const keys = [...pendingUpdates.keys()]
      for (let i = 0; i < keys.length - 100; i++) {
        pendingUpdates.delete(keys[i])
      }
    }
  }, 30000)

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
