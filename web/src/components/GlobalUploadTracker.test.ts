import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../stores/auth'
import { useUploadStore } from '../stores/upload'
import GlobalUploadTracker from './GlobalUploadTracker.vue'

vi.mock('../api/client', () => {
  const api = {
    post: vi.fn(),
    get: vi.fn(),
  }
  return { default: api }
})

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
    const upload = useUploadStore()
    upload.addJob({
      job_id: 'job-1',
      filename: 'failed.jpg',
      status: 'failed',
      error: 'unsupported format',
    })

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('failed.jpg')
    expect(wrapper.text()).toContain('unsupported format')
    expect(wrapper.text()).toContain('failed')
  })

  it('shows Retry button for failed jobs', async () => {
    const upload = useUploadStore()
    upload.addJob({
      job_id: 'job-2',
      filename: 'retry-me.jpg',
      status: 'failed',
      error: 'Network error',
    })

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('Retry')
    const retryBtn = wrapper.find('button[style*="color: var(--accent)"]')
    expect(retryBtn.exists()).toBe(true)
  })

  it('calls retryUpload when Retry button clicked', async () => {
    const upload = useUploadStore()
    upload.addJob({
      job_id: 'job-3',
      filename: 'click-retry.jpg',
      status: 'failed',
      error: 'Network error',
    })

    const retrySpy = vi.spyOn(upload, 'retryUpload').mockResolvedValue()

    const wrapper = await mountExpanded()

    const retryBtn = wrapper.find('button[style*="color: var(--accent)"]')
    await retryBtn.trigger('click')

    expect(retrySpy).toHaveBeenCalledWith('job-3')
  })

  it('does not show error text for non-failed jobs', async () => {
    const upload = useUploadStore()
    upload.addJob({
      job_id: 'job-4',
      filename: 'ok.jpg',
      status: 'queued',
    })

    const wrapper = await mountExpanded()

    const errorElement = wrapper.find('.text-\\[var\\(--error\\)\\]')
    expect(errorElement.exists()).toBe(false)
  })

  it('does not show Retry button for completed jobs', async () => {
    const upload = useUploadStore()
    upload.addJob({
      job_id: 'job-5',
      filename: 'done.jpg',
      status: 'completed',
      file_id: 'f-1',
    })

    const wrapper = await mountExpanded()

    expect(wrapper.text()).not.toContain('Retry')
  })

  it('shows dismiss button for failed, skipped, and completed jobs', async () => {
    const upload = useUploadStore()
    upload.addJob({ job_id: 'job-f', filename: 'f.jpg', status: 'failed', error: 'err' })
    upload.addJob({ job_id: 'job-s', filename: 's.jpg', status: 'skipped', reason: 'dup' })
    upload.addJob({ job_id: 'job-c', filename: 'c.jpg', status: 'completed', file_id: 'f-2' })

    const wrapper = await mountExpanded()

    const closeButtons = wrapper.findAll('[class*="--text-secondary"]')
    expect(closeButtons.length).toBeGreaterThanOrEqual(3)
  })

  it('shows reason text instead of error when reason is set', async () => {
    const upload = useUploadStore()
    upload.addJob({
      job_id: 'job-6',
      filename: 'reasoned.jpg',
      status: 'failed',
      reason: 'unsupported',
    })

    const wrapper = await mountExpanded()

    expect(wrapper.text()).toContain('unsupported')
  })
})
