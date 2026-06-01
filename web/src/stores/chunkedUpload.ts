import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api/client'

const CHUNK_SIZE = 5 * 1024 * 1024
const CHUNK_THRESHOLD = 50 * 1024 * 1024
const MAX_CONCURRENT_CHUNKS = 3

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

async function computeChunkSHA256(blob: Blob): Promise<string> {
  const buffer = await blob.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  return Array.from(new Uint8Array(hashBuffer))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
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

export const useChunkedUploadStore = defineStore('chunkedUpload', () => {
  const jobs = ref<ChunkedUploadJob[]>([])
  const completedJobs = ref<{ file_id: string; filename: string; folder_id: string | null }[]>([])

  const activeJobs = computed(() => jobs.value.filter(j => j.status !== 'completed' && j.status !== 'failed'))
  const hasActiveUploads = computed(() => activeJobs.value.length > 0)

  function shouldUseChunked(file: File): boolean {
    return file.size >= CHUNK_THRESHOLD
  }

  function addJob(job: ChunkedUploadJob) {
    jobs.value = [job, ...jobs.value]
    persistTokens(jobs.value.filter(j => j.status === 'uploading' || j.status === 'paused'))
  }

  function removeJob(uploadId: string) {
    jobs.value = jobs.value.filter(j => j.uploadId !== uploadId)
    persistTokens(jobs.value.filter(j => j.status === 'uploading' || j.status === 'paused'))
  }

  function updateJob(uploadId: string, updates: Partial<ChunkedUploadJob>) {
    const idx = jobs.value.findIndex(j => j.uploadId === uploadId)
    if (idx >= 0) {
      jobs.value[idx] = { ...jobs.value[idx], ...updates }
      if (updates.storedChunks) {
        jobs.value[idx].uploadedBytes = updates.storedChunks.length * jobs.value[idx].chunkSize
      }
    }
  }

  async function uploadChunk(job: ChunkedUploadJob, chunkIndex: number, blob: Blob): Promise<boolean> {
    const sha256hex = await computeChunkSHA256(blob)

    try {
      const headers: Record<string, string> = {
        'X-Chunk-Index': String(chunkIndex),
        'X-Chunk-Size': String(blob.size),
        'X-Chunk-SHA256': sha256hex,
        'X-Filename': job.filename,
        'X-Total-Size': String(job.totalSize),
        'X-Total-Chunks': String(job.totalChunks),
      }

      if (job.resumeToken) {
        headers['X-Resume-Token'] = job.resumeToken
      }
      if (job.targetFolderId) {
        headers['X-Folder-ID'] = job.targetFolderId
      }
      if (job.skipNameSizeDedup) {
        headers['X-Skip-Name-Size-Dedup'] = 'true'
      }

      const res = await api.post('/upload/chunk', blob, {
        headers: {
          ...headers,
          'Content-Type': 'application/octet-stream',
        },
        onUploadProgress: (progressEvent: any) => {
          if (!progressEvent.total) return
          const idx = jobs.value.findIndex(j => j.uploadId === job.uploadId)
          if (idx >= 0) {
            const chunkProgress = Math.min(progressEvent.loaded / progressEvent.total, 0.99)
            const priorBytes = chunkIndex * job.chunkSize
            jobs.value[idx].uploadedBytes = priorBytes + Math.round(chunkProgress * blob.size)
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

      if (!job.uploadId && data.upload_id) {
        job.uploadId = data.upload_id
      }
      if (data.resume_token && !job.resumeToken) {
        job.resumeToken = data.resume_token
        persistTokens(jobs.value.filter(j => j.status === 'uploading' || j.status === 'paused'))
      }

      updateJob(job.uploadId || data.upload_id, {
        storedChunks: data.stored_chunks,
        uploadedBytes: data.stored_chunks.length * job.chunkSize,
      })

      return true
    } catch (e: any) {
      if (e?.response?.status === 422) {
        return false
      }
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

    addJob(job)

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
          if (current?.status === 'failed') return
          try {
            const ok = await queued[idx]()
            if (!ok) {
              concurrencyIndex = idx
              const pi = jobs.value.findIndex(j => j.uploadId === job.uploadId)
              if (pi >= 0) {
                jobs.value[pi].error = 'Chunk hash mismatch, retrying...'
              }
              await new Promise(r => setTimeout(r, 1000))
            }
          } catch {
            const pi = jobs.value.findIndex(j => j.uploadId === job.uploadId)
            if (pi >= 0) {
              jobs.value[pi].status = 'failed'
              jobs.value[pi].error = 'Network error during chunk upload'
            }
            return
          }
        }
      }

      const workers = Math.min(MAX_CONCURRENT_CHUNKS, queued.length)
      await Promise.all(Array.from({ length: workers }, () => worker()))

      if (job.storedChunks.length >= totalChunks) {
        updateJob(job.uploadId, { status: 'assembling' })

        const completeRes = await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
          upload_id: job.uploadId,
        })

        if (completeRes.data.status === 'assembling' && completeRes.data.missing_chunks.length === 0) {
          updateJob(job.uploadId, { status: 'processing' })
        }
      }
    } catch {
      updateJob(job.uploadId, { status: 'paused' })
      persistTokens(jobs.value.filter(j => j.status === 'uploading' || j.status === 'paused'))
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

      if (status === 'completed' || status === 'failed') {
        return null
      }

      const storedChunks: number[] = []
      for (let i = 0; i < storedCount; i++) {
        storedChunks.push(i)
      }

      return {
        uploadId,
        resumeToken,
        filename: '',
        totalSize,
        totalChunks,
        chunkSize: CHUNK_SIZE,
        storedChunks,
        uploadedBytes: storedCount * CHUNK_SIZE,
        status: 'uploading',
        targetFolderId: null,
        skipNameSizeDedup: true,
      }
    } catch {
      return null
    }
  }

  async function resumeUpload(job: ChunkedUploadJob, file: File): Promise<void> {
    const totalChunks = job.totalChunks
    const missingChunks: number[] = []
    for (let i = 0; i < totalChunks; i++) {
      if (!job.storedChunks.includes(i)) {
        missingChunks.push(i)
      }
    }

    if (missingChunks.length === 0) {
      updateJob(job.uploadId, { status: 'assembling' })
      await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
        upload_id: job.uploadId,
      })
      updateJob(job.uploadId, { status: 'processing' })
      return
    }

    job.status = 'uploading'

    for (const chunkIndex of missingChunks) {
      const start = chunkIndex * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      const blob = file.slice(start, end)
      await uploadChunk(job, chunkIndex, blob)
    }

    updateJob(job.uploadId, { status: 'assembling' })
    await api.post(`/upload/chunk/${job.resumeToken}/complete`, {
      upload_id: job.uploadId,
    })
    updateJob(job.uploadId, { status: 'processing' })
  }

  async function checkAndResumeAll() {
    const tokens = loadTokens()
    for (const t of tokens) {
      const status = await checkResume(t.token)
      if (status) {
        status.filename = t.filename
        jobs.value = [status, ...jobs.value]
      } else {
        clearTokens()
      }
    }
  }

  function consumeCompletedJobs(): { file_id: string; filename: string; folder_id: string | null }[] {
    const drained = [...completedJobs.value]
    completedJobs.value = []
    return drained
  }

  return {
    jobs,
    completedJobs,
    activeJobs,
    hasActiveUploads,
    shouldUseChunked,
    startChunkedUpload,
    checkResume,
    resumeUpload,
    checkAndResumeAll,
    addJob,
    removeJob,
    updateJob,
    consumeCompletedJobs,
  }
})
