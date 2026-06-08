import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

const { mockApiPost, mockApiGet, mockApiHead } = vi.hoisted(() => ({
  mockApiPost: vi.fn(),
  mockApiGet: vi.fn(),
  mockApiHead: vi.fn(),
}))

vi.mock('../api/client', () => ({
  default: {
    post: mockApiPost,
    get: mockApiGet,
    head: mockApiHead,
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('./auth', () => ({
  useAuthStore: () => ({
    accessToken: 'test-token',
    refreshToken: null,
    user: { id: 'u1', username: 'test', role: 'member' },
    isAuthenticated: true,
  }),
}))

Object.defineProperty(navigator, 'sendBeacon', {
  value: vi.fn().mockReturnValue(true),
  writable: true,
})

import { useChunkedUploadStore, type ChunkedUploadJob } from './chunkedUpload'

function make422Error(): Error {
  const err = new Error('Request failed with status code 422') as any
  err.response = { status: 422, data: { error: { code: 'CHUNK_HASH_MISMATCH' } } }
  return err
}

function makeJob(overrides: Partial<ChunkedUploadJob> = {}): ChunkedUploadJob {
  return {
    uploadId: 'job-1',
    resumeToken: 'token-1',
    filename: 'test.mp4',
    totalSize: 15 * 1024 * 1024,
    totalChunks: 3,
    chunkSize: 5 * 1024 * 1024,
    storedChunks: [],
    uploadedBytes: 0,
    status: 'uploading',
    targetFolderId: null,
    skipNameSizeDedup: false,
    ...overrides,
  }
}

describe('useChunkedUploadStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  describe('job lifecycle', () => {
    it('addJob inserts job at front of list', () => {
      const store = useChunkedUploadStore()
      store.addJob(makeJob())
      expect(store.jobs).toHaveLength(1)
      expect(store.jobs[0].uploadId).toBe('job-1')
    })

    it('removeJob removes job and cleans up', () => {
      const store = useChunkedUploadStore()
      store.addJob(makeJob())
      store.removeJob('job-1')
      expect(store.jobs).toHaveLength(0)
    })

    it('updateJob applies patches to existing job', () => {
      const store = useChunkedUploadStore()
      store.addJob(makeJob())
      store.updateJob('job-1', { status: 'assembling', storedChunks: [0, 1] })
      expect(store.jobs[0].status).toBe('assembling')
      expect(store.jobs[0].storedChunks).toEqual([0, 1])
    })

    it('clearCompleted removes completed, keeps failed and active', () => {
      const store = useChunkedUploadStore()
      store.addJob(makeJob({ uploadId: 'j1', status: 'completed' }))
      store.addJob(makeJob({ uploadId: 'j2', status: 'failed' }))
      store.addJob(makeJob({ uploadId: 'j3', status: 'uploading' }))
      store.clearCompleted()
      const ids = store.jobs.map(j => j.uploadId)
      expect(ids).toContain('j2')
      expect(ids).toContain('j3')
      expect(ids).not.toContain('j1')
    })
  })

  describe('startChunkedUpload', () => {
    it('creates a chunked upload session and processes chunks', async () => {
      const store = useChunkedUploadStore()

      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-created', resume_token: 'rt-1', stored_chunks: [0], missing_chunks: [1, 2], next_chunk: 1 },
      })
      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-created', resume_token: 'rt-1', stored_chunks: [0, 1], missing_chunks: [2], next_chunk: 2 },
      })
      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-created', resume_token: 'rt-1', stored_chunks: [0, 1, 2], missing_chunks: [], next_chunk: 3 },
      })
      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-created', batch_id: 'b1', job_id: 'job-created', status: 'assembling', stored_chunks: 3, missing_chunks: [], total_chunks: 3 },
      })
      mockApiGet.mockResolvedValueOnce({
        status: 200,
        data: { jobs: [{ job_id: 'job-created', status: 'completed', file_id: 'f1', filename: 'test.mp4' }] },
      })

      const file = new File(['a'.repeat(15 * 1024 * 1024)], 'test.mp4')
      await store.startChunkedUpload(file, null, false)

      expect(mockApiPost).toHaveBeenCalledTimes(4)
      const job = store.jobs.find(j => j.uploadId === 'job-created')
      expect(job).toBeDefined()
      expect(job!.status).toBe('completed')
    })

    it('retries chunk on hash mismatch', async () => {
      const store = useChunkedUploadStore()

      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-1', resume_token: 'rt-1', stored_chunks: [0], missing_chunks: [1], next_chunk: 1 },
      })
      mockApiPost.mockRejectedValueOnce(make422Error())
      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-1', resume_token: 'rt-1', stored_chunks: [0, 1], missing_chunks: [], next_chunk: 2 },
      })
      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'job-1', batch_id: 'b1', job_id: 'job-1', status: 'assembling', stored_chunks: 2, missing_chunks: [], total_chunks: 2 },
      })
      mockApiGet.mockResolvedValueOnce({
        status: 200,
        data: { jobs: [{ job_id: 'job-1', status: 'completed', file_id: 'f1', filename: 'test.mp4' }] },
      })

      const file = new File(['a'.repeat(10 * 1024 * 1024)], 'test.mp4')
      await store.startChunkedUpload(file, null, false)

      expect(mockApiPost).toHaveBeenCalledTimes(4)
      expect(store.jobs[0].status).toBe('completed')
    })
  })

  describe('autoResumeConnectivityJobs', () => {
    it('auto-completes when all chunks already stored on server', async () => {
      const store = useChunkedUploadStore()
      const job = makeJob({ uploadId: 'paused-job', status: 'paused', totalChunks: 3, storedChunks: [0, 1, 2], uploadedBytes: 15 * 1024 * 1024 })
      store.addJob(job)

      mockApiHead.mockResolvedValueOnce({
        status: 200,
        headers: {
          'x-upload-status': 'queued',
          'x-total-chunks': '3',
          'x-stored-count': '3',
          'x-total-size': '15728640',
          'x-stored-chunks': '[0,1,2]',
          'x-upload-id': 'paused-job',
        },
      })
      mockApiPost.mockResolvedValueOnce({
        status: 202,
        data: { upload_id: 'paused-job', stored_chunks: 3, missing_chunks: [], total_chunks: 3, job_id: 'paused-job' },
      })

      window.dispatchEvent(new Event('online'))

      await vi.waitFor(() => {
        expect(mockApiHead).toHaveBeenCalled()
      }, { timeout: 1000 })

      await vi.waitFor(() => {
        expect(mockApiPost).toHaveBeenCalled()
      }, { timeout: 1000 })
    })
  })

  describe('beforeunload', () => {
    it('persists active tokens on beforeunload', () => {
      const store = useChunkedUploadStore()
      store.addJob(makeJob({ uploadId: 'j1', status: 'uploading', resumeToken: 'rt1' }))
      store.addJob(makeJob({ uploadId: 'j2', status: 'paused', resumeToken: 'rt2' }))
      store.addJob(makeJob({ uploadId: 'j3', status: 'completed', resumeToken: 'rt3' }))

      window.dispatchEvent(new Event('beforeunload'))

      const stored = localStorage.getItem('chunked_uploads')
      expect(stored).not.toBeNull()
      const parsed = JSON.parse(stored!)
      expect(parsed).toHaveLength(2)
      const tokens = parsed.map((p: any) => p.token)
      expect(tokens).toContain('rt1')
      expect(tokens).toContain('rt2')
      expect(navigator.sendBeacon).toHaveBeenCalled()
    })
  })

  describe('consumeCompletedJobs', () => {
    it('drains completed jobs list', () => {
      const store = useChunkedUploadStore()
      store.completedJobs.push(
        { file_id: 'f1', filename: 'a.mp4', folder_id: null },
        { file_id: 'f2', filename: 'b.mp4', folder_id: 'folder-1' },
      )
      const drained = store.consumeCompletedJobs()
      expect(drained).toHaveLength(2)
      expect(store.completedJobs).toHaveLength(0)
    })

    it('returns empty array when no completed jobs', () => {
      const store = useChunkedUploadStore()
      const drained = store.consumeCompletedJobs()
      expect(drained).toHaveLength(0)
    })
  })
})
