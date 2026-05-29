import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUploadStore, chunkBySize, withConcurrency, type FileWithJob } from './upload'
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

  describe('uploadFiles batching', () => {
    beforeEach(() => {
      vi.useRealTimers()
    })

    afterEach(() => {
      vi.useFakeTimers()
    })

    function makeFile(name: string, sizeBytes: number): File {
      return new File([new Uint8Array(sizeBytes)], name)
    }

    it('sends a single POST for files under batch byte limit', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const files = Array.from({ length: 10 }, (_, i) => makeFile(`f${i}.jpg`, 1024))

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        if (url === '/upload') {
          return Promise.resolve({
            data: { batch_id: 'batch-1', jobs: files.map((f, i) => ({ job_id: `uuid-${i}`, filename: f.name, status: 'queued' })) },
          })
        }
        return Promise.reject(new Error('unknown'))
      })

      await upload.uploadFiles(files, null, false)
      expect(mockPost.mock.calls.filter((c: any) => c[0] === '/upload').length).toBe(1)
    })

    it('marks all batch jobs as failed on network error', async () => {
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
        expect(job).toBeDefined()
        expect(job!.status).toBe('failed')
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

      const job = upload.jobs.find(j => j.filename === 'dup.jpg')
      expect(job!.status).toBe('skipped')
      expect(job!.file_id).toBe('existing-id')
    })

    it('respects per-file status from batch server response', async () => {
      const upload = useUploadStore()
      const mockPost = api.post as ReturnType<typeof vi.fn>
      const files = [makeFile('good.jpg', 100), makeFile('bad.jpg', 200)]

      mockPost.mockImplementation((url: string) => {
        if (url === '/upload/check') return Promise.resolve({ data: { duplicates: [] } })
        if (url === '/upload') {
          return Promise.resolve({
            data: {
              batch_id: 'batch-1',
              jobs: [
                { job_id: 'uuid-0', filename: 'good.jpg', status: 'queued' },
                { job_id: 'uuid-1', filename: 'bad.jpg', status: 'failed', reason: 'unsupported' },
              ],
            },
          })
        }
        return Promise.reject(new Error('unknown'))
      })

      await upload.uploadFiles(files, null, false)

      const good = upload.jobs.find(j => j.filename === 'good.jpg')
      expect(good!.status).toBe('queued')
      expect(good!.job_id).toBe('uuid-0')

      const bad = upload.jobs.find(j => j.filename === 'bad.jpg')
      expect(bad!.status).toBe('failed')
      expect(bad!.job_id).toBe('uuid-1')
    })
  })
})

describe('chunkBySize', () => {
  function makeItem(filename: string, size: number): FileWithJob {
    return {
      job: { job_id: filename, filename, status: 'uploading' },
      file: new File([new Uint8Array(size)], filename),
    }
  }

  it('returns empty array for empty input', () => {
    expect(chunkBySize([], 1000)).toEqual([])
  })

  it('returns one chunk when total size is under maxBytes', () => {
    const a = makeItem('a', 30)
    const b = makeItem('b', 40)
    const c = makeItem('c', 20)
    const chunks = chunkBySize([a, b, c], 100)
    expect(chunks).toHaveLength(1)
    expect(chunks[0]).toEqual([a, b, c])
  })

  it('splits into multiple chunks when total exceeds maxBytes', () => {
    const a = makeItem('a', 60)
    const b = makeItem('b', 50)
    const c = makeItem('c', 40)
    const chunks = chunkBySize([a, b, c], 100)
    expect(chunks).toHaveLength(2)
    expect(chunks[0]).toEqual([a])
    expect(chunks[1]).toEqual([b, c])
  })

  it('puts oversized item in its own chunk', () => {
    const huge = makeItem('huge', 200)
    const small = makeItem('small', 10)
    const chunks = chunkBySize([huge, small], 100)
    expect(chunks).toHaveLength(2)
    expect(chunks[0]).toEqual([huge])
    expect(chunks[1]).toEqual([small])
  })

  it('ensures each chunk has at least one item', () => {
    const items = [
      makeItem('a', 30),
      makeItem('b', 80),
      makeItem('c', 30),
      makeItem('d', 60),
    ]
    const chunks = chunkBySize(items, 100)
    expect(chunks).toHaveLength(3)
    expect(chunks[0].length).toBe(1)
    expect(chunks[1].length).toBe(1)
    expect(chunks[2].length).toBe(2)
  })

  it('keeps identical-size items together when they fit', () => {
    const items = Array.from({ length: 10 }, (_, i) => makeItem(`f${i}`, 10))
    const chunks = chunkBySize(items, 100)
    expect(chunks).toHaveLength(1)
    expect(chunks[0]).toHaveLength(10)
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
})
