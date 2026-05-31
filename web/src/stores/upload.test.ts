import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUploadStore, withConcurrency } from './upload'
import { useAuthStore } from './auth'
import api from '../api/client'

let mockWs: {
  url: string
  onmessage: ((event: MessageEvent) => void) | null
  onopen: ((event: Event) => void) | null
  onerror: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  close: () => void
} | null = null

vi.mock('../api/client', () => {
  const api = {
    post: vi.fn(),
    get: vi.fn(),
  }
  return { default: api }
})

describe('UploadStore', () => {
  beforeEach(() => {
    mockWs = null
    vi.useFakeTimers()
    vi.stubGlobal('WebSocket', class MockWebSocket {
      url: string
      onmessage: ((event: MessageEvent) => void) | null = null
      onopen: ((event: Event) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      constructor(url: string) {
        this.url = url
        mockWs = this
        setTimeout(() => {
          if (this.onopen) this.onopen(new Event('open'))
        }, 0)
      }
      close() {
        if (this.onclose) {
          this.onclose(new CloseEvent('close'))
        }
        mockWs = null
      }
    })
    setActivePinia(createPinia())
    const auth = useAuthStore()
    auth.accessToken = 'test-token'
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  describe('consumeCompletedJobs', () => {
    it('starts empty', () => {
      const upload = useUploadStore()
      expect(upload.consumeCompletedJobs()).toEqual([])
    })

    it('adds completed WS message to queue and drains on consume', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)
      expect(mockWs).not.toBeNull()

      upload.addJob({
        job_id: 'job-1',
        filename: 'test.jpg',
        status: 'queued',
      })

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'job-1',
          filename: 'test.jpg',
          status: 'completed',
          file_id: 'file-abc',
          folder_id: 'folder-xyz',
          progress: 1.0,
        }),
      }))

      const result = upload.consumeCompletedJobs()
      expect(result).toHaveLength(1)
      expect(result[0].file_id).toBe('file-abc')
      expect(result[0].filename).toBe('test.jpg')
      expect(result[0].folder_id).toBe('folder-xyz')

      expect(upload.consumeCompletedJobs()).toEqual([])
    })

    it('adds skipped job with file_id to queue', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      upload.addJob({
        job_id: 'job-2',
        filename: 'dup.jpg',
        status: 'queued',
      })

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'job-2',
          filename: 'dup.jpg',
          status: 'skipped',
          file_id: 'existing-id',
          folder_id: null,
          progress: 1.0,
        }),
      }))

      const result = upload.consumeCompletedJobs()
      expect(result).toHaveLength(1)
      expect(result[0].file_id).toBe('existing-id')
      expect(result[0].folder_id).toBeNull()
    })

    it('does not add processing job to completed queue', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      upload.addJob({
        job_id: 'job-3',
        filename: 'proc.jpg',
        status: 'queued',
      })

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'job-3',
          filename: 'proc.jpg',
          status: 'processing',
          stage: 'hashing',
          progress: 0.3,
        }),
      }))

      expect(upload.consumeCompletedJobs()).toEqual([])
    })

    it('does not add failed job without file_id to completed queue', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      upload.addJob({
        job_id: 'job-4',
        filename: 'fail.jpg',
        status: 'queued',
      })

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'job-4',
          filename: 'fail.jpg',
          status: 'failed',
          error: 'unsupported',
        }),
      }))

      expect(upload.consumeCompletedJobs()).toEqual([])
    })

    it('handles folder_id absent from WS message as undefined', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      upload.addJob({
        job_id: 'job-5',
        filename: 'no-folder.jpg',
        status: 'queued',
      })

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'job-5',
          filename: 'no-folder.jpg',
          status: 'completed',
          file_id: 'file-def',
          progress: 1.0,
        }),
      }))

      const result = upload.consumeCompletedJobs()
      expect(result).toHaveLength(1)
      expect(result[0].folder_id).toBeUndefined()
    })
  })

  describe('pendingUpdates buffer', () => {
    it('buffers WS update for unknown job_id and replays when job is added', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'server-job-id',
          filename: 'photo.jpg',
          status: 'processing',
          stage: 'hashing',
          progress: 0.1,
        }),
      }))

      upload.addJob({
        job_id: 'server-job-id',
        filename: 'photo.jpg',
        status: 'queued',
      })

      const job = upload.jobs.find(j => j.job_id === 'server-job-id')
      expect(job).toBeDefined()
      expect(job!.status).toBe('processing')
      expect(job!.stage).toBe('hashing')
      expect(job!.progress).toBe(0.1)
    })

    it('replays pending result status via WS message when job_id later matches', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'late-job',
          filename: 'late.jpg',
          status: 'completed',
          file_id: 'file-xyz',
          progress: 1.0,
        }),
      }))

      upload.addJob({
        job_id: 'late-job',
        filename: 'late.jpg',
        status: 'queued',
      })

      const job = upload.jobs.find(j => j.job_id === 'late-job')
      expect(job).toBeDefined()
      expect(job!.status).toBe('completed')
      expect(job!.file_id).toBe('file-xyz')
    })

    it('clears pending update after replay so it does not repeat', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'one-time',
          filename: 'one.jpg',
          status: 'processing',
          progress: 0.3,
        }),
      }))

      upload.addJob({
        job_id: 'one-time',
        filename: 'one.jpg',
        status: 'queued',
      })

      expect(upload.jobs.find(j => j.job_id === 'one-time')!.status).toBe('processing')

      upload.addJob({
        job_id: 'one-time',
        filename: 'one.jpg',
        status: 'queued',
      })

      expect(upload.jobs.filter(j => j.job_id === 'one-time').length).toBeGreaterThanOrEqual(1)
    })

    it('removes pending update when job is removed', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      mockWs!.onmessage!(new MessageEvent('message', {
        data: JSON.stringify({
          job_id: 'to-remove',
          filename: 'rm.jpg',
          status: 'processing',
          progress: 0.5,
        }),
      }))

      upload.removeJob('to-remove')

      upload.addJob({
        job_id: 'to-remove',
        filename: 'rm.jpg',
        status: 'queued',
      })

      const job = upload.jobs.find(j => j.job_id === 'to-remove')
      expect(job).toBeDefined()
      expect(job!.status).toBe('queued')
    })
  })

  describe('WebSocket reconnection', () => {
    it('schedules reconnect when WS closes unexpectedly', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      expect(mockWs).not.toBeNull()

      mockWs!.close()

      vi.advanceTimersByTime(1500)

      const newWs = mockWs
      expect(newWs).not.toBeNull()
      expect(newWs!.url).toContain('/upload/ws')
    })

    it('does not schedule reconnect after manual disconnect', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      upload.disconnectWS()

      const before = mockWs

      vi.advanceTimersByTime(5000)

      expect(mockWs).toBe(before)
    })

    it('resets reconnect attempts on successful connection', () => {
      const upload = useUploadStore()
      upload.connectWS()
      vi.advanceTimersByTime(1)

      expect(mockWs).not.toBeNull()

      mockWs!.close()
      vi.advanceTimersByTime(1500)

      expect(mockWs).not.toBeNull()

      mockWs!.close()
      vi.advanceTimersByTime(2500)

      expect(mockWs).not.toBeNull()
    })
  })

    describe('uploadFiles concurrency queue', () => {
    beforeEach(() => {
      vi.useRealTimers()
    })

    afterEach(() => {
      vi.useFakeTimers()
    })

    function makeFile(name: string, sizeBytes: number): File {
      return new File([new Uint8Array(sizeBytes)], name)
    }

    it('sends one POST per file', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const files = [makeFile('a.jpg', 100), makeFile('b.jpg', 200), makeFile('c.jpg', 300)]

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        if (url === '/upload') {
          return Promise.resolve({
            data: { batch_id: 'batch-1', jobs: [{ job_id: 'uuid-0', filename: 'test.jpg', status: 'queued' }] },
          })
        }
        return Promise.reject(new Error('unknown'))
      })

      await upload.uploadFiles(files, null, false)
      const uploadCalls = mockPost.mock.calls.filter((c: any) => c[0] === '/upload')
      expect(uploadCalls.length).toBe(3)
    })

    it('marks jobs as failed on network error', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const files = [makeFile('a.jpg', 100), makeFile('b.jpg', 200)]

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        if (url === '/upload') return Promise.reject(new Error('network error'))
        return Promise.reject(new Error('unknown'))
      })

      await upload.uploadFiles(files, null, false)
      for (const name of ['a.jpg', 'b.jpg']) {
        const job = upload.jobs.find(j => j.filename === name)
        expect(job!.status).toBe('failed')
        expect(job!.error).toBe('Network error')
      }
    })

    it('handles empty file list', async () => {
      const upload = useUploadStore()
      await upload.uploadFiles([], null, true)
      expect(upload.jobs).toHaveLength(0)
    })

    it('skips upload when all files are deduped', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const files = [makeFile('dup.jpg', 100)]

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') {
          return Promise.resolve({ data: { duplicates: [{ filename: 'dup.jpg', file_id: 'existing-id', size: 100 }] } })
        }
        if (url === '/upload') return Promise.resolve({ data: { batch_id: 'batch-1', jobs: [] } })
        return Promise.reject(new Error('unknown'))
      })

      await upload.uploadFiles(files, null, false)
      expect(mockPost.mock.calls.filter((c: any) => c[0] === '/upload').length).toBe(0)
      expect(upload.jobs.find(j => j.filename === 'dup.jpg')!.status).toBe('skipped')
    })
  })

})

describe('withConcurrency', () => {
  beforeEach(() => {
    vi.useRealTimers()
    ;(api.get as ReturnType<typeof vi.fn>).mockReturnValue(new Promise(() => {}))
  })

  it('limits concurrent execution to given limit', async () => {
    let inFlight = 0
    let maxConcurrent = 0

    const tasks = Array.from({ length: 10 }, () => async () => {
      inFlight++
      maxConcurrent = Math.max(maxConcurrent, inFlight)
      await Promise.resolve()
      inFlight--
    })

    await withConcurrency(tasks, 3)
    expect(maxConcurrent).toBe(3)
  })

  it('returns results in correct order', async () => {
    const tasks = Array.from({ length: 5 }, (_, i) => async () => i)
    const results = await withConcurrency(tasks, 2)
    expect(results).toEqual([0, 1, 2, 3, 4])
  })

  it('handles empty task array', async () => {
    const results = await withConcurrency([], 3)
    expect(results).toEqual([])
  })

  it('handles limit greater than task count', async () => {
    let maxConcurrent = 0
    let inFlight = 0
    const tasks = Array.from({ length: 2 }, () => async () => {
      inFlight++
      maxConcurrent = Math.max(maxConcurrent, inFlight)
      await Promise.resolve()
      inFlight--
    })
    await withConcurrency(tasks, 10)
    expect(maxConcurrent).toBe(2)
  })

  describe('retryUpload', () => {
    beforeEach(() => {
      vi.useRealTimers()
    })

    afterEach(() => {
      vi.useFakeTimers()
    })

    function makeFile(name: string, sizeBytes: number): File {
      return new File([new Uint8Array(sizeBytes)], name)
    }

    it('resets failed job and re-submits via upload API', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const file = makeFile('retry.jpg', 100)

      let uploadCalls = 0
      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        if (url === '/upload') {
          uploadCalls++
          if (uploadCalls === 1) return Promise.reject(new Error('network error'))
          return Promise.resolve({ data: { batch_id: 'batch-retry', jobs: [{ job_id: 'uuid-retry', filename: 'retry.jpg', status: 'queued' }] } })
        }
        return Promise.reject(new Error('unknown'))
      })

      await upload.uploadFiles([file], null, true)

      const failedJob = upload.jobs.find(j => j.filename === 'retry.jpg')
      expect(failedJob!.status).toBe('failed')
      expect(failedJob!.error).toBe('Network error')
      const failedJobId = failedJob!.job_id

      await upload.retryUpload(failedJobId)

      const job = upload.jobs.find(j => j.job_id === 'uuid-retry')
      expect(job).toBeDefined()
      expect(job!.status).toBe('queued')
      expect(job!.error).toBeUndefined()
      expect(job!.reason).toBeUndefined()
      expect(job!.batch_id).toBe('batch-retry')
    })

    it('does nothing for unknown jobId', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      mockPost.mockClear()
      await upload.retryUpload('nonexistent')
      expect(mockPost).not.toHaveBeenCalled()
    })

    it('marks as failed on network error during retry', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const file = makeFile('netfail.jpg', 100)

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        return Promise.reject(new Error('network error'))
      })

      await upload.uploadFiles([file], null, true)

      const failedJob = upload.jobs.find(j => j.filename === 'netfail.jpg')
      expect(failedJob!.status).toBe('failed')
      const jobId = failedJob!.job_id

      await upload.retryUpload(jobId)

      const job = upload.jobs.find(j => j.job_id === jobId)
      expect(job!.status).toBe('failed')
      expect(job!.error).toBe('Network error')
    })

    it('sets uploadError when server returns 413 QUOTA_EXCEEDED', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const file = makeFile('bigfile.jpg', 100)

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        return Promise.reject({
          response: {
            status: 413,
            data: { error: { message: 'Upload would exceed space quota (500 used + 200 incoming > 600 limit)' } }
          }
        })
      })

      await upload.uploadFiles([file], null, true)

      expect(upload.uploadError).toContain('exceed space quota')
    })
  })
})
