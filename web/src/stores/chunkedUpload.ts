import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth'
import api from '../api/client'

const CHUNK_SIZE = 5 * 1024 * 1024
const MAX_CONCURRENT_CHUNKS = 3
const MAX_CHUNK_RETRIES = 3

export interface ChunkedUploadJob {
  uploadId: string
  resumeToken: string
  filename: string
  totalSize: number
  totalChunks: number
  chunkSize: number
  storedChunks: number[]
  uploadedBytes: number
  status: 'uploading' | 'assembling' | 'processing' | 'completed' | 'failed' | 'paused'
  targetFolderId: string | null
  skipNameSizeDedup: boolean
  error?: string
  file_id?: string
}

function persistTokens(jobs: ChunkedUploadJob[]) {
  const tokens = jobs.map(j => ({
    token: j.resumeToken,
    filename: j.filename,
    totalSize: j.totalSize,
  }))
  localStorage.setItem('chunked_uploads', JSON.stringify(tokens))
}

function loadTokens(): { token: string; filename: string; totalSize: number }[] {
  try {
    const raw = localStorage.getItem('chunked_uploads')
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function clearTokens() {
  localStorage.removeItem('chunked_uploads')
}

function loadConsumedFileIds(): Set<string> {
  try {
    const raw = sessionStorage.getItem('consumed_file_ids')
    return raw ? new Set(JSON.parse(raw)) : new Set()
  } catch {
    return new Set()
  }
}

function persistConsumedFileIds(ids: Set<string>) {
  try {
    sessionStorage.setItem('consumed_file_ids', JSON.stringify([...ids]))
  } catch {
    // best-effort
  }
}

export const useChunkedUploadStore = defineStore('chunkedUpload', () => {
  const jobs = ref<ChunkedUploadJob[]>([])
  const completedJobs = ref<{ file_id: string; filename: string; folder_id: string | null }[]>([])
  const uploadError = ref('')

  const activeJobs = computed(() =>
    jobs.value.filter(j => j.status !== 'completed' && j.status !== 'failed'),
  )
  const hasActiveUploads = computed(() => activeJobs.value.length > 0)
  const completedCount = computed(() => jobs.value.filter(j => j.status === 'completed').length)
  const isVisible = computed(() => jobs.value.length > 0)

  let globalWs: WebSocket | null = null
  let reconnectAttempts = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let wsManualClose = false
  let reconcileInterval: ReturnType<typeof setInterval> | null = null

  const pendingUpdates = new Map<string, Partial<ChunkedUploadJob>>()
  const retryFiles = new Map<string, File>()

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
      const update = JSON.parse(event.data) as {
        job_id: string
        status?: string
        file_id?: string
        filename?: string
        progress?: number
        error?: string
        folder_id?: string | null
      }
      const idx = jobs.value.findIndex(j => j.uploadId === update.job_id)
      if (idx >= 0) {
        const job = jobs.value[idx]
        if (update.progress !== undefined) {
            const newBytes = Math.min(Math.round(update.progress * job.totalSize), job.totalSize)
            jobs.value[idx] = {
              ...job,
              uploadedBytes: Math.max(job.uploadedBytes, newBytes),
              error: update.error,
            }
          }
        if (update.status === 'completed' || (update.status === 'skipped' && update.file_id)) {
          if (update.file_id) {
            completedJobs.value.push({
              file_id: update.file_id,
              filename: update.filename || job.filename,
              folder_id: update.folder_id ?? null,
            })
          }
          jobs.value[idx] = { ...job, status: 'completed', file_id: update.file_id, error: undefined }
        } else if (update.status === 'failed') {
          jobs.value[idx] = { ...job, status: 'failed', error: update.error || 'Server processing failed' }
        }
      } else {
        const existing = pendingUpdates.get(update.job_id) || {}
        pendingUpdates.set(update.job_id, { ...existing, ...update } as Partial<ChunkedUploadJob>)
      }
    }

    globalWs.onclose = () => {
      globalWs = null
      if (!wsManualClose) scheduleReconnect()
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

  function scheduleReconnect() {
    if (reconnectTimer) return
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000)
    reconnectAttempts++
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connectWS()
    }, delay)
  }

  async function fetchActiveJobs() {
    try {
      const consumedIds = loadConsumedFileIds()
      const res = await api.get('/upload/active')
      const activeJobs: any[] = res.data.jobs || []
      for (const aj of activeJobs) {
        if (aj.status === 'completed' || aj.status === 'skipped') {
          if (aj.file_id) {
            if (consumedIds.has(aj.file_id)) continue
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
        if (aj.status !== 'queued' && aj.status !== 'processing') continue
        const idx = jobs.value.findIndex(j => j.uploadId === aj.job_id)
        if (idx >= 0) {
          const job = jobs.value[idx]
        if (aj.progress !== undefined) {
              const newBytes = Math.min(Math.round(aj.progress * job.totalSize), job.totalSize)
              jobs.value[idx] = { ...job, uploadedBytes: Math.max(job.uploadedBytes, newBytes) }
            }
        }
      }
    } catch {
      // best-effort
    }
  }

  function reconcileStuckJobs() {
    const stuckJobs = jobs.value.filter(j => j.status === 'processing')
    if (stuckJobs.length === 0) return

    const uploadIds = stuckJobs.map(j => j.uploadId)
    for (const uploadId of uploadIds) {
      api.get('/upload/active')
        .then(res => {
          const activeJobs: any[] = res.data.jobs || []
          const found = activeJobs.find((j: any) => j.job_id === uploadId)
          if (!found) return
          const idx = jobs.value.findIndex(j => j.uploadId === uploadId)
          if (idx < 0) return
          if (found.status === 'completed' || found.status === 'skipped') {
            if (found.file_id) {
              completedJobs.value.push({
                file_id: found.file_id,
                filename: jobs.value[idx].filename,
                folder_id: jobs.value[idx].targetFolderId,
              })
            }
            jobs.value[idx] = { ...jobs.value[idx], status: 'completed', error: undefined }
          } else if (found.status === 'failed') {
            jobs.value[idx] = {
              ...jobs.value[idx],
              status: 'failed',
              error: found.error || 'Server processing failed',
            }
          }
        })
        .catch(() => {})
    }
  }

  function startReconcile() {
    if (reconcileInterval) return
    reconcileInterval = setInterval(reconcileStuckJobs, 5000)
  }

  startReconcile()

  window.addEventListener('online', () => {
    autoResumeConnectivityJobs()
  })

  window.addEventListener('beforeunload', () => {
    persistActiveTokens()
    const activeTokens = jobs.value
      .filter(j => (j.status === 'uploading' || j.status === 'paused' || j.status === 'assembling') && j.resumeToken)
      .map(j => ({ token: j.resumeToken, uploadId: j.uploadId }))
    if (activeTokens.length > 0 && navigator.sendBeacon) {
      const payload = new Blob([JSON.stringify({ tokens: activeTokens })], { type: 'application/json' })
      navigator.sendBeacon('/api/v1/upload/progress-flush', payload)
    }
  })

  setInterval(() => {
    if (pendingUpdates.size > 100) {
      const keys = [...pendingUpdates.keys()]
      for (let i = 0; i < keys.length - 100; i++) {
        pendingUpdates.delete(keys[i])
      }
    }
  }, 30000)

  function addJob(job: ChunkedUploadJob) {
    const pending = pendingUpdates.get(job.uploadId)
    if (pending) {
      pendingUpdates.delete(job.uploadId)
      job = { ...job, ...pending }
    }
    jobs.value = [job, ...jobs.value]
    persistActiveTokens()
  }

  function removeJob(uploadId: string) {
    jobs.value = jobs.value.filter(j => j.uploadId !== uploadId)
    pendingUpdates.delete(uploadId)
    retryFiles.delete(uploadId)
    persistActiveTokens()
  }

  function updateJob(uploadId: string, updates: Partial<ChunkedUploadJob>) {
    const idx = jobs.value.findIndex(j => j.uploadId === uploadId)
    if (idx >= 0) {
      jobs.value[idx] = { ...jobs.value[idx], ...updates }
      if (updates.storedChunks) {
        const computedBytes = Math.min(updates.storedChunks.length * jobs.value[idx].chunkSize, jobs.value[idx].totalSize)
        jobs.value[idx].uploadedBytes = Math.max(jobs.value[idx].uploadedBytes, computedBytes)
      }
      if (updates.status) {
        persistActiveTokens()
      }
    }
  }

  function clearCompleted() {
    jobs.value = jobs.value.filter(j =>
      j.status === 'uploading' || j.status === 'processing' ||
      j.status === 'assembling' || j.status === 'paused' || j.status === 'failed',
    )
  }

  function persistActiveTokens() {
    persistTokens(jobs.value.filter(j =>
      j.status === 'uploading' || j.status === 'paused' || j.status === 'assembling',
    ))
  }

  async function uploadChunk(job: ChunkedUploadJob, chunkIndex: number, blob: Blob): Promise<boolean> {
    console.log('[chunkedUpload] uploading chunk', chunkIndex, 'of', job.totalChunks)

    try {
      const headers: Record<string, string> = {
        'X-Chunk-Index': String(chunkIndex),
        'X-Chunk-Size': String(blob.size),
        'X-Filename': job.filename,
        'X-Total-Size': String(job.totalSize),
        'X-Total-Chunks': String(job.totalChunks),
      }

      if (job.resumeToken) headers['X-Resume-Token'] = job.resumeToken
      if (job.targetFolderId) headers['X-Folder-ID'] = job.targetFolderId
      if (job.skipNameSizeDedup) headers['X-Skip-Name-Size-Dedup'] = 'true'

      const res = await api.post('/upload/chunk', blob, {
        headers: { ...headers, 'Content-Type': 'application/octet-stream' },
        onUploadProgress: (progressEvent: any) => {
          if (!progressEvent.total) return
          const idx = jobs.value.findIndex(j => j.uploadId === job.uploadId)
          if (idx >= 0) {
            const chunkProgress = Math.min(progressEvent.loaded / progressEvent.total, 0.99)
            const priorBytes = chunkIndex * job.chunkSize
            const newBytes = Math.min(priorBytes + Math.round(chunkProgress * blob.size), job.totalSize)
            jobs.value[idx].uploadedBytes = Math.max(jobs.value[idx].uploadedBytes, newBytes)
          }
        },
      })

      const data = res.data as {
        upload_id: string
        resume_token: string
        stored_chunks: number[]
        missing_chunks: number[]
        next_chunk: number
      }

      if (!job.uploadId && data.upload_id) job.uploadId = data.upload_id
      if (data.resume_token && !job.resumeToken) {
        job.resumeToken = data.resume_token
        persistActiveTokens()
      }

      updateJob(job.uploadId || data.upload_id, {
        storedChunks: data.stored_chunks,
      })

      return true
    } catch (e: any) {
      console.error('[chunkedUpload] uploadChunk error:', e?.message, e?.response?.status, e?.response?.data)
      if (e?.response?.status === 422) return false
      throw e
    }
  }

  async function startChunkedUpload(
    file: File,
    targetFolderId: string | null,
    skipNameSizeDedup: boolean,
  ): Promise<ChunkedUploadJob> {
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE)

    const job: ChunkedUploadJob = {
      uploadId: '',
      resumeToken: '',
      filename: file.name,
      totalSize: file.size,
      totalChunks,
      chunkSize: CHUNK_SIZE,
      storedChunks: [],
      uploadedBytes: 0,
      status: 'uploading',
      targetFolderId,
      skipNameSizeDedup,
    }

    retryFiles.set(job.uploadId || `fresh-${Date.now()}`, file)
    addJob(job)
    const jobRef = jobs.value.indexOf(job)

    try {
      const firstChunk = file.slice(0, Math.min(CHUNK_SIZE, file.size))
      await uploadChunk(job, 0, firstChunk)

      const queued: (() => Promise<boolean>)[] = []
      for (let i = 1; i < totalChunks; i++) {
        const chunkIndex = i
        queued.push(async () => {
          const start = chunkIndex * CHUNK_SIZE
          const end = Math.min(start + CHUNK_SIZE, file.size)
          const blob = file.slice(start, end)
          return uploadChunk(job, chunkIndex, blob)
        })
      }

      let concurrencyIndex = 0
      async function worker() {
        while (concurrencyIndex < queued.length) {
          const idx = concurrencyIndex++
          const current = jobs.value.find(j => j.uploadId === job.uploadId)
          if (current?.status === 'failed' || current?.status === 'paused') return
          let retries = 0
          for (;;) {
            try {
              const ok = await queued[idx]()
              if (!ok) {
                concurrencyIndex = idx
                const pi = jobs.value.findIndex(j => j.uploadId === job.uploadId)
                if (pi >= 0) jobs.value[pi].error = 'Chunk hash mismatch, retrying...'
                await new Promise(r => setTimeout(r, 1000))
              }
              break
            } catch {
              const e: any = arguments[0]
              const is409 = e?.response?.status === 409
              if (is409) {
                const pi = jobs.value.findIndex(j => j.uploadId === job.uploadId)
                if (pi >= 0) {
                  jobs.value[pi].status = 'failed'
                  jobs.value[pi].error = e?.response?.data?.error?.message || 'Upload session expired'
                }
                persistActiveTokens()
                return
              }
              retries++
              if (retries > MAX_CHUNK_RETRIES) {
                const pi = jobs.value.findIndex(j => j.uploadId === job.uploadId)
                if (pi >= 0) {
                  jobs.value[pi].status = 'paused'
                  jobs.value[pi].error = `Network error after ${MAX_CHUNK_RETRIES} retries`
                }
                persistActiveTokens()
                return
              }
              const delay = 1000 * Math.pow(2, retries - 1)
              const pi = jobs.value.findIndex(j => j.uploadId === job.uploadId)
              if (pi >= 0) jobs.value[pi].error = `Network error, retrying (${retries}/${MAX_CHUNK_RETRIES})...`
              await new Promise(r => setTimeout(r, delay))
            }
          }
        }
      }

      const workers = Math.min(MAX_CONCURRENT_CHUNKS, queued.length)
      await Promise.all(Array.from({ length: workers }, () => worker()))

      updateJob(job.uploadId, { status: 'assembling' })
      const completeRes = await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
        upload_id: job.uploadId,
      })

      const storedCount = completeRes.data.stored_chunks || 0
      const missingChunks = completeRes.data.missing_chunks || []
      if (storedCount >= totalChunks && missingChunks.length === 0) {
        updateJob(job.uploadId, { status: 'processing' })
        await pollForCompletion(job.uploadId, completeRes.data.job_id)
      }
    } catch (e: any) {
      console.error('[chunkedUpload] startChunkedUpload error:', e?.message || e, e?.response?.status, e?.response?.data)
      if (e?.response?.status === 413) {
        uploadError.value = e?.response?.data?.error?.message || 'File too large'
      }
      if (jobRef >= 0 && jobRef < jobs.value.length) {
        jobs.value[jobRef] = { ...jobs.value[jobRef], status: 'paused', error: e?.message || 'Unknown error' }
      }
      persistActiveTokens()
    }

    return job
  }

  async function checkResume(resumeToken: string): Promise<ChunkedUploadJob | null> {
    try {
      const res = await api.head(`/upload/chunk/${resumeToken}`)
      const status = res.headers['x-upload-status'] as string
      const totalChunks = parseInt(res.headers['x-total-chunks'] as string, 10)
      const storedCount = parseInt(res.headers['x-stored-count'] as string, 10)
      const totalSize = parseInt(res.headers['x-total-size'] as string, 10)
      const uploadId = res.headers['x-upload-id'] as string

      if (status === 'completed' || status === 'failed') return null

      const storedChunksRaw = res.headers['x-stored-chunks'] as string
      const storedChunks: number[] = storedChunksRaw ? JSON.parse(storedChunksRaw) : []

      return {
        uploadId,
        resumeToken,
        filename: '',
        totalSize,
        totalChunks,
        chunkSize: CHUNK_SIZE,
        storedChunks,
        uploadedBytes: storedCount * CHUNK_SIZE,
        status: 'paused',
        targetFolderId: null,
        skipNameSizeDedup: true,
      }
    } catch {
      return null
    }
  }

  async function resumeUpload(job: ChunkedUploadJob, file: File): Promise<void> {
    retryFiles.set(job.uploadId, file)
    const totalChunks = job.totalChunks
    const storedChunks = job.storedChunks || []
    const missingChunks: number[] = []
    for (let i = 0; i < totalChunks; i++) {
      if (!storedChunks.includes(i)) missingChunks.push(i)
    }

    if (missingChunks.length === 0) {
      updateJob(job.uploadId, { status: 'assembling' })
      const completeRes = await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
        upload_id: job.uploadId,
      })
      const serverStored = completeRes.data.stored_chunks || 0
      const serverMissing = completeRes.data.missing_chunks || []
      if (serverStored >= totalChunks && serverMissing.length === 0) {
        updateJob(job.uploadId, { status: 'processing' })
        await pollForCompletion(job.uploadId, completeRes.data.job_id)
      }
      return
    }

    updateJob(job.uploadId, { status: 'uploading' })

    for (const chunkIndex of missingChunks) {
      const start = chunkIndex * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      const blob = file.slice(start, end)
      await uploadChunk(job, chunkIndex, blob)
    }

    updateJob(job.uploadId, { status: 'assembling' })
    const completeRes = await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
      upload_id: job.uploadId,
    })
    const serverStored = completeRes.data.stored_chunks || 0
    const serverMissing = completeRes.data.missing_chunks || []
    if (serverStored >= totalChunks && serverMissing.length === 0) {
      updateJob(job.uploadId, { status: 'processing' })
      await pollForCompletion(job.uploadId, completeRes.data.job_id)
    }
  }

  async function checkAndResumeAll() {
    const tokens = loadTokens()
    for (const t of tokens) {
      const status = await checkResume(t.token)
      if (status) {
        status.filename = t.filename
        const exists = jobs.value.find(j => j.uploadId === status.uploadId)
        if (!exists) {
          jobs.value = [status, ...jobs.value]
        }
      } else {
        clearTokens()
      }
    }
  }

  async function autoResumeConnectivityJobs() {
    const toResume = jobs.value.filter(j =>
      (j.status === 'paused' || (j.status === 'failed' && j.resumeToken)) && j.resumeToken,
    )
    for (const job of toResume) {
      try {
        const status = await checkResume(job.resumeToken)
        if (!status) continue
        const idx = jobs.value.findIndex(j => j.uploadId === job.uploadId)
        if (idx < 0) continue
        const existing = jobs.value[idx]
        if (status.storedChunks.length >= existing.totalChunks) {
          updateJob(job.uploadId, { status: 'assembling', error: undefined })
          const completeRes = await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
            upload_id: job.uploadId,
          })
          if (completeRes.data.missing_chunks.length === 0) {
            updateJob(job.uploadId, { status: 'processing' })
            pollForCompletion(job.uploadId, completeRes.data.job_id)
          }
        } else {
          const file = retryFiles.get(job.uploadId)
          if (!file) {
            updateJob(job.uploadId, {
              status: 'paused',
              error: 'Connectivity restored but file re-selection needed to resume',
            })
            continue
          }
          resumeUpload(job, file)
        }
      } catch {
        // best-effort
      }
    }
  }

  function resumePausedJob(uploadId: string) {
    const idx = jobs.value.findIndex(j => j.uploadId === uploadId)
    if (idx < 0) return
    const job = jobs.value[idx]
    const storedChunks = job.storedChunks || []

    if (storedChunks.length >= job.totalChunks) {
      updateJob(uploadId, { status: 'assembling' })
      api.post(`/upload/chunk/${job.resumeToken}/complete`, { upload_id: job.uploadId })
        .then(completeRes => {
          const serverStored = completeRes.data.stored_chunks || 0
          const serverMissing = completeRes.data.missing_chunks || []
          if (serverStored >= job.totalChunks && serverMissing.length === 0) {
            updateJob(uploadId, { status: 'processing' })
            pollForCompletion(uploadId, completeRes.data.job_id)
          }
        })
        .catch(() => {
          updateJob(uploadId, { status: 'paused', error: 'Failed to resume' })
        })
      return
    }

    const input = document.createElement('input')
    input.type = 'file'
    input.onchange = (e: Event) => {
      const files = (e.target as HTMLInputElement).files
      if (!files || files.length === 0) return
      const file = files[0]
      if (file.name !== job.filename || file.size !== job.totalSize) {
        updateJob(uploadId, { error: 'Selected file does not match. Expected ' + job.filename })
        return
      }
      updateJob(uploadId, { error: undefined })
      resumeUpload(job, file)
    }
    input.click()
  }

  function resumeAllPausedJobs() {
    const paused = jobs.value.filter(j => j.status === 'paused')
    for (const job of paused) {
      resumePausedJob(job.uploadId)
    }
  }

  function abortAll() {
    jobs.value = jobs.value.filter(j =>
      j.status === 'completed' || j.status === 'failed',
    )
    clearTokens()
  }

  async function pollForCompletion(uploadId: string, jobId: string) {
    const maxAttempts = 60
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      await new Promise(r => setTimeout(r, 2000))
      try {
        const res = await api.get('/upload/active')
        const activeJobs: any[] = res.data.jobs || []
        const found = activeJobs.find((j: any) => j.job_id === jobId)
        if (!found || found.status === 'completed' || found.status === 'skipped') {
          updateJob(uploadId, { status: 'completed', error: undefined })
          if (found?.file_id) {
            completedJobs.value.push({
              file_id: found.file_id,
              filename: '',
              folder_id: null,
            })
          }
          return
        }
        if (found.status === 'failed') {
          updateJob(uploadId, { status: 'failed', error: found.error || 'Server processing failed' })
          return
        }
        if (found.progress !== undefined) {
          const job = jobs.value.find(j => j.uploadId === uploadId)
          if (job) {
            const newBytes = Math.min(Math.round(found.progress * job.totalSize), job.totalSize)
            updateJob(uploadId, { uploadedBytes: Math.max(job.uploadedBytes, newBytes) })
          }
        }
      } catch {
        // continue polling
      }
    }
    updateJob(uploadId, { status: 'failed', error: 'Timed out waiting for server processing' })
  }

  async function uploadFiles(
    fileList: FileList | File[],
    targetFolderId: string | null,
    skipNameSizeDedup: boolean,
  ) {
    uploadError.value = ''
    const allFiles = Array.from(fileList)
    const promises = allFiles.map(file =>
      startChunkedUpload(file, targetFolderId, skipNameSizeDedup),
    )
    await Promise.allSettled(promises)
  }

  async function retryUpload(uploadId: string) {
    const idx = jobs.value.findIndex(j => j.uploadId === uploadId)
    const file = retryFiles.get(uploadId)
    if (idx < 0 || !file) return

    const job = jobs.value[idx]
    removeJob(uploadId)
    await startChunkedUpload(file, job.targetFolderId, job.skipNameSizeDedup)
  }

  function consumeCompletedJobs(): { file_id: string; filename: string; folder_id: string | null }[] {
    const drained = [...completedJobs.value]
    const consumedIds = loadConsumedFileIds()
    for (const j of drained) {
      consumedIds.add(j.file_id)
    }
    persistConsumedFileIds(consumedIds)
    completedJobs.value = []
    return drained
  }

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
    startChunkedUpload,
    checkResume,
    resumeUpload,
    checkAndResumeAll,
    resumePausedJob,
    resumeAllPausedJobs,
    abortAll,
    addJob,
    removeJob,
    updateJob,
    clearCompleted,
    consumeCompletedJobs,
    uploadFiles,
    retryUpload,
  }
})
