import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api/client'

const DOWNLOAD_CHUNK_SIZE = 2 * 1024 * 1024
const MAX_CONCURRENT_DOWNLOAD_CHUNKS = 4

interface DownloadChunk {
  start: number
  end: number
  status: 'pending' | 'downloading' | 'complete'
  data?: Uint8Array
}

export interface DownloadJob {
  jobId: string
  fileId: string
  filename: string
  mimeType: string
  totalSize: number
  downloadedBytes: number
  status: 'downloading' | 'completed' | 'paused' | 'failed'
  chunks: DownloadChunk[]
  error?: string
}

export const useDownloadStore = defineStore('download', () => {
  const jobs = ref<DownloadJob[]>([])

  const activeDownloads = computed(() => jobs.value.filter(j => j.status === 'downloading'))

  function addJob(job: DownloadJob) {
    jobs.value = [job, ...jobs.value]
  }

  function removeJob(jobId: string) {
    jobs.value = jobs.value.filter(j => j.jobId !== jobId)
  }

  function updateJob(jobId: string, updates: Partial<DownloadJob>) {
    const idx = jobs.value.findIndex(j => j.jobId === jobId)
    if (idx >= 0) {
      jobs.value[idx] = { ...jobs.value[idx], ...updates }
    }
  }

  async function fetchChunk(
    fileId: string,
    start: number,
    end: number,
    signal?: AbortSignal,
  ): Promise<Uint8Array> {
    const res = await api.get(`/download/${fileId}`, {
      responseType: 'arraybuffer',
      headers: {
        Range: `bytes=${start}-${end}`,
      },
      signal,
    })
    return new Uint8Array(res.data)
  }

  async function startDownload(fileId: string, filename: string, mimeType: string, totalSize: number): Promise<DownloadJob> {
    const totalChunks = Math.ceil(totalSize / DOWNLOAD_CHUNK_SIZE)
    const chunks: DownloadChunk[] = []

    for (let i = 0; i < totalChunks; i++) {
      const start = i * DOWNLOAD_CHUNK_SIZE
      const end = Math.min(start + DOWNLOAD_CHUNK_SIZE - 1, totalSize - 1)
      chunks.push({ start, end, status: 'pending' })
    }

    const job: DownloadJob = {
      jobId: `dl-${Date.now()}-${fileId.substring(0, 8)}`,
      fileId,
      filename,
      mimeType,
      totalSize,
      downloadedBytes: 0,
      status: 'downloading',
      chunks,
    }

    addJob(job)

    try {
      await downloadAllChunks(job)
    } catch {
      updateJob(job.jobId, { status: 'paused' })
    }

    return job
  }

  async function downloadAllChunks(job: DownloadJob): Promise<void> {
    const pending = job.chunks.filter(c => c.status === 'pending')
    const abortController = new AbortController()

    let concurrencyIndex = 0
    const chunkArray = [...pending]

    async function worker() {
      while (concurrencyIndex < chunkArray.length) {
        const idx = concurrencyIndex++
        const chunk = chunkArray[idx]
        const chunkIdx = job.chunks.indexOf(chunk)
        if (chunkIdx < 0) continue

        job.chunks[chunkIdx].status = 'downloading'

        try {
          const data = await fetchChunk(job.fileId, chunk.start, chunk.end, abortController.signal)
          job.chunks[chunkIdx].data = data
          job.chunks[chunkIdx].status = 'complete'
          job.downloadedBytes += data.length
        } catch (e: any) {
          if (e?.name !== 'AbortError') {
            throw e
          }
          return
        }
      }
    }

    await Promise.all(
      Array.from({ length: Math.min(MAX_CONCURRENT_DOWNLOAD_CHUNKS, pending.length) }, () => worker()),
    )

    if (job.chunks.every(c => c.status === 'complete')) {
      await assembleAndSave(job)
    }
  }

  async function assembleAndSave(job: DownloadJob): Promise<void> {
    const totalBytes = job.chunks.reduce((sum, c) => sum + (c.data?.length || 0), 0)
    const buffer = new Uint8Array(totalBytes)
    let offset = 0

    for (const chunk of job.chunks) {
      if (chunk.data) {
        buffer.set(chunk.data, offset)
        offset += chunk.data.length
      }
    }

    const blob = new Blob([buffer], { type: job.mimeType || 'application/octet-stream' })
    const url = URL.createObjectURL(blob)

    const a = document.createElement('a')
    a.href = url
    a.download = job.filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    updateJob(job.jobId, { status: 'completed' })
  }

  async function resumeDownload(jobId: string): Promise<void> {
    const job = jobs.value.find(j => j.jobId === jobId)
    if (!job) return

    updateJob(jobId, { status: 'downloading' })

    try {
      await downloadAllChunks(job)
    } catch {
      updateJob(jobId, { status: 'paused' })
    }
  }

  function pauseDownload(jobId: string) {
    updateJob(jobId, { status: 'paused' })
  }

  return {
    jobs,
    activeDownloads,
    startDownload,
    resumeDownload,
    pauseDownload,
    addJob,
    removeJob,
    updateJob,
  }
})
