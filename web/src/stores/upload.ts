import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth'
import { useChunkedUploadStore } from './chunkedUpload'
import api from '../api/client'

const MAX_CONCURRENT_UPLOADS = 5

async function withConcurrency<T>(tasks: (() => Promise<T>)[], limit: number): Promise<T[]> {
  const results: T[] = new Array(tasks.length)
  let index = 0

  async function worker() {
    while (index < tasks.length) {
      const currentIndex = index++
      results[currentIndex] = await tasks[currentIndex]()
    }
  }

  await Promise.all(
    Array.from({ length: Math.min(limit, tasks.length) }, () => worker()),
  )
  return results
}


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
  targetFolderId?: string | null
  skipNameSizeDedup?: boolean
}

export interface CompletedJob {
  file_id: string
  filename: string
  folder_id: string | null | undefined
}

export const useUploadStore = defineStore('upload', () => {
  const jobs = ref<UploadJob[]>([])
  const uploadError = ref('')
  const completedJobs = ref<CompletedJob[]>([])

  let globalWs: WebSocket | null = null
  let reconnectAttempts = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let wsManualClose = false

  const pendingUpdates = new Map<string, Partial<UploadJob>>()
  const retryFiles = new Map<string, File>()

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
      fetchActiveJobs()
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

  async function fetchActiveJobs() {
    try {
      const res = await api.get('/upload/active')
      const activeJobs: any[] = res.data.jobs || []
      for (const aj of activeJobs) {
        if (aj.status === 'completed' || aj.status === 'skipped') {
          if (aj.file_id) {
            const alreadyCompleted = completedJobs.value.some(cj => cj.file_id === aj.file_id)
            if (!alreadyCompleted) {
              completedJobs.value.push({
                file_id: aj.file_id,
                filename: aj.filename,
                folder_id: aj.folder_id,
              })
            }
          }
          continue
        }
        if (aj.status !== 'queued' && aj.status !== 'processing') {
          continue
        }
        const idx = jobs.value.findIndex(j => j.job_id === aj.job_id)
        if (idx >= 0) {
          jobs.value[idx] = { ...jobs.value[idx], ...aj }
        } else {
          jobs.value = [aj as UploadJob, ...jobs.value]
        }
      }
    } catch {
      // best-effort: jobs will still appear via WS reconciliation
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
    retryFiles.delete(jobId)
  }

  function clearCompleted() {
    jobs.value = jobs.value.filter(j => j.status === 'uploading' || j.status === 'processing' || j.status === 'queued' || j.status === 'failed')
  }

  async function uploadFiles(fileList: FileList | File[], targetFolderId: string | null, skipNameSizeDedup: boolean) {
    uploadError.value = ''
    const allFiles = Array.from(fileList)

    const chunkedStore = useChunkedUploadStore()
    const smallFiles: File[] = []
    const largeFilePromises: Promise<any>[] = []

    for (const file of allFiles) {
      if (chunkedStore.shouldUseChunked(file)) {
        largeFilePromises.push(
          chunkedStore.startChunkedUpload(file, targetFolderId, skipNameSizeDedup),
        )
      } else {
        smallFiles.push(file)
      }
    }

    if (smallFiles.length === 0) {
      if (largeFilePromises.length > 0) {
        await Promise.allSettled(largeFilePromises)
      }
      return
    }

    const fileMap = new Map<string, File>()
    const uploadJobs: UploadJob[] = []

    for (let i = 0; i < smallFiles.length; i++) {
      const file = smallFiles[i]
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
        targetFolderId: targetFolderId,
        skipNameSizeDedup: skipNameSizeDedup,
      })
      retryFiles.set(jobId, file)
    }
    jobs.value = [...uploadJobs, ...jobs.value]

    if (!skipNameSizeDedup) {
      const checkPayload = smallFiles.map(f => ({ filename: f.name, size: f.size }))
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

    const uploadTasks = uploadJobs
      .filter(job => {
        const current = jobs.value.find(j => j.job_id === job.job_id)
        return current?.status === 'uploading'
      })
      .map(job => {
        const file = fileMap.get(job.job_id)
        if (!file) return null

        return async () => {
          const currentIdx = jobs.value.findIndex(j => j.job_id === job.job_id)
          if (currentIdx < 0) return

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
              onUploadProgress: (progressEvent: any) => {
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
            const realJobs: any[] = res.data.jobs || []
            if (realJobs.length > 0) {
              const realJob = realJobs[0]
              const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
              if (idx >= 0) {
                const oldJobId = jobs.value[idx].job_id
                const pending = pendingUpdates.get(realJob.job_id)
                if (pending) {
                  pendingUpdates.delete(realJob.job_id)
                  jobs.value[idx] = { ...jobs.value[idx], ...realJob, batch_id: batchId, progress: 0, ...pending }
                } else {
                  jobs.value[idx] = { ...jobs.value[idx], ...realJob, batch_id: batchId, progress: 0 }
                }
                if (oldJobId !== realJob.job_id) {
                  const f = retryFiles.get(oldJobId)
                  if (f) {
                    retryFiles.delete(oldJobId)
                    retryFiles.set(realJob.job_id, f)
                  }
                }
              }
            }
          } catch (e: any) {
            const idx = jobs.value.findIndex(j => j.job_id === job.job_id)
            const errMsg = e?.response?.data?.error?.message || e?.response?.data?.message || 'Network error'
            if (e?.response?.status === 413) {
              uploadError.value = errMsg
            }
            if (idx >= 0) {
              jobs.value[idx] = {
                ...jobs.value[idx],
                status: 'failed',
                error: errMsg,
                stage: undefined,
              }
            }
          }
        }
      })
      .filter(Boolean) as (() => Promise<void>)[]

    await withConcurrency(uploadTasks, MAX_CONCURRENT_UPLOADS)
  }

  async function retryUpload(jobId: string) {
    const idx = jobs.value.findIndex(j => j.job_id === jobId)
    const file = retryFiles.get(jobId)
    if (idx < 0 || !file) return

    const job = jobs.value[idx]
    const targetFolderId = job.targetFolderId ?? null
    const skipNameSizeDedup = job.skipNameSizeDedup ?? true

    jobs.value[idx] = {
      ...job,
      status: 'uploading',
      progress: 0,
      stage: 'uploading',
      uploaded: 0,
      error: undefined,
      reason: undefined,
      file_id: undefined,
      batch_id: undefined,
    }

    if (!skipNameSizeDedup) {
      try {
        const checkRes = await api.post('/upload/check', [{ filename: file.name, size: file.size }])
        const duplicate = (checkRes.data.duplicates || []).find((d: { filename: string; file_id: string }) => d.filename === file.name)
        if (duplicate) {
          jobs.value[idx] = {
            ...jobs.value[idx],
            status: 'skipped',
            progress: 1,
            file_id: duplicate.file_id,
            stage: undefined,
          }
          return
        }
      } catch {
        // continue with upload even if dedup check fails
      }
    }

    const form = new FormData()
    form.append('files', file)
    if (targetFolderId) form.append('folder_id', targetFolderId)
    if (skipNameSizeDedup) form.append('skip_name_size_dedup', 'true')

    try {
      const res = await api.post('/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (progressEvent: any) => {
          if (!progressEvent.total) return
          const progress = Math.min(progressEvent.loaded / progressEvent.total, 0.99)
          jobs.value[idx] = {
            ...jobs.value[idx],
            progress: Math.round(progress * 100) / 100,
            uploaded: Math.round(progress * (job.size || 1)),
          }
        },
      })

      const batchId = res.data.batch_id as string
      const realJobs: any[] = res.data.jobs || []

      if (realJobs.length > 0) {
        const realJob = realJobs[0]
        const oldRetryJobId = jobs.value[idx].job_id
        const pending = pendingUpdates.get(realJob.job_id)
        if (pending) {
          pendingUpdates.delete(realJob.job_id)
          jobs.value[idx] = { ...jobs.value[idx], ...realJob, batch_id: batchId, progress: 0, uploaded: 0, ...pending }
        } else {
          jobs.value[idx] = { ...jobs.value[idx], ...realJob, batch_id: batchId, progress: 0, uploaded: 0 }
        }
        if (oldRetryJobId !== realJob.job_id) {
          const rf = retryFiles.get(oldRetryJobId)
          if (rf) {
            retryFiles.delete(oldRetryJobId)
            retryFiles.set(realJob.job_id, rf)
          }
        }
      }

      for (let i = 1; i < realJobs.length; i++) {
        addJob({
          job_id: realJobs[i].job_id,
          batch_id: batchId,
          filename: realJobs[i].filename,
          status: realJobs[i].status,
          file_id: realJobs[i].file_id,
          progress: 0,
        })
      }
    } catch {
      jobs.value[idx] = {
        ...jobs.value[idx],
        status: 'failed',
        error: 'Network error',
        stage: undefined,
      }
    }
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
    uploadError,
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
    retryUpload,
  }
})

export { withConcurrency, MAX_CONCURRENT_UPLOADS }

