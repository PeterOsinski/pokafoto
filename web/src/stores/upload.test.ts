import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUploadStore } from './upload'
import { useAuthStore } from './auth'

let mockWs: {
  url: string
  onmessage: ((event: MessageEvent) => void) | null
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
    vi.stubGlobal('WebSocket', class MockWebSocket {
      url: string
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      constructor(url: string) {
        this.url = url
        mockWs = this
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
  })

  describe('consumeCompletedJobs', () => {
    it('starts empty', () => {
      const upload = useUploadStore()
      expect(upload.consumeCompletedJobs()).toEqual([])
    })

    it('adds completed WS message to queue and drains on consume', () => {
      const upload = useUploadStore()
      upload.connectWS()
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
})
