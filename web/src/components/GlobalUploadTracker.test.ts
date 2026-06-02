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
})
