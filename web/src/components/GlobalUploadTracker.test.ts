import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../stores/auth'
import { useChunkedUploadStore, type ChunkedUploadJob } from '../stores/chunkedUpload'
import GlobalUploadTracker from './GlobalUploadTracker.vue'

vi.mock('../api/client', () => {
  const api = {
    post: vi.fn(),
    get: vi.fn(),
  }
  return { default: api }
})

function makeJob(overrides: Partial<ChunkedUploadJob> = {}) {
  return {
    uploadId: 'upload-1',
    resumeToken: 'token-1',
    filename: 'test.jpg',
    totalSize: 5000000,
    totalChunks: 1,
    chunkSize: 5000000,
    storedChunks: [],
    uploadedBytes: 0,
    status: 'uploading' as const,
    targetFolderId: null,
    skipNameSizeDedup: true,
    ...overrides,
  }
}

describe('GlobalUploadTracker', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const auth = useAuthStore()
    auth.accessToken = 'test-token'
    vi.clearAllMocks()
  })

  async function mountExpanded() {
    const wrapper = mount(GlobalUploadTracker)
    await wrapper.find('button').trigger('click')
    await wrapper.vm.$nextTick()
    return wrapper
  }

  it('shows error text for failed jobs', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-1', filename: 'failed.jpg', status: 'failed', error: 'unsupported format' }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('failed.jpg')
    expect(wrapper.text()).toContain('unsupported format')
    expect(wrapper.text()).toContain('failed')
  })

  it('shows Restart button for paused jobs', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-2', filename: 'paused.jpg', status: 'paused' }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('Restart')
  })

  it('calls resumePausedJob when Restart button clicked', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-3', filename: 'click-restart.jpg', status: 'paused' }))

    const restartSpy = vi.spyOn(upload, 'resumePausedJob').mockImplementation(() => {})

    const wrapper = await mountExpanded()
    const buttons = wrapper.findAll('button')
    const restartBtn = buttons.find(b => b.text() === 'Restart')
    await restartBtn!.trigger('click')

    expect(restartSpy).toHaveBeenCalledWith('job-3')
  })

  it('does not show error text for non-failed jobs', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-4', filename: 'ok.jpg', status: 'assembling' }))

    const wrapper = await mountExpanded()

    const errorElement = wrapper.find('.text-\\[var\\(--error\\)\\]')
    expect(errorElement.exists()).toBe(false)
  })

  it('does not show Retry button for completed jobs', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-5', filename: 'done.jpg', status: 'completed', file_id: 'f-1' }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).not.toContain('Retry')
  })

  it('shows Restart all button when paused jobs exist', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-p1', filename: 'p1.jpg', status: 'paused' }))
    upload.addJob(makeJob({ uploadId: 'job-p2', filename: 'p2.jpg', status: 'paused' }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('Restart all')
  })

  it('does not show Retry button for expired jobs', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-exp', filename: 'expired.jpg', status: 'failed', error: 'upload_expired' }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).not.toContain('Retry')
  })

  it('shows Retry button for non-expired failed jobs', async () => {
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-net', filename: 'network.jpg', status: 'failed', error: 'network error' }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('Retry')
  })

  it('shows aggregate sizes in header when collapsed', async () => {
    const upload = useChunkedUploadStore()
    const MB = 1024 * 1024
    upload.addJob(makeJob({ uploadId: 'job-a', filename: 'a.bin', totalSize: 3 * MB, status: 'uploading', uploadedBytes: 1 * MB }))
    upload.addJob(makeJob({ uploadId: 'job-b', filename: 'b.bin', totalSize: 7 * MB, status: 'uploading', uploadedBytes: 4 * MB }))

    const wrapper = mount(GlobalUploadTracker)

    expect(wrapper.text()).toContain('5.0 MB / 10.0 MB')
    expect(wrapper.text()).toContain('files')
  })

  it('shows per-job sizes in expanded view', async () => {
    const MB = 1024 * 1024
    const upload = useChunkedUploadStore()
    upload.addJob(makeJob({ uploadId: 'job-c', filename: 'photo.png', totalSize: 2 * MB, status: 'uploading', uploadedBytes: 1 * MB }))

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('photo.png')
    expect(wrapper.text()).toContain('1.0 MB')
    expect(wrapper.text()).toContain('2.0 MB')
  })
})
