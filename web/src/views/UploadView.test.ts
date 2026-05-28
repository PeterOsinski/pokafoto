import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import UploadView from './UploadView.vue'
import { useAuthStore } from '../stores/auth'

vi.mock('../api/client', () => {
  const api = {
    post: vi.fn(),
    get: vi.fn(),
  }
  return { default: api }
})

function makeFile(name: string, size: number): File {
  return new File([new ArrayBuffer(size)], name, { type: 'image/jpeg' })
}

function fakeProgressEvent(loaded: number, total: number) {
  return { loaded, total }
}

describe('UploadView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const auth = useAuthStore()
    auth.accessToken = 'test-token'
    vi.clearAllMocks()
  })

  it('shows files with uploading status immediately on selection', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation(() => new Promise(() => {}))

    const wrapper = mount(UploadView)
    const file = makeFile('test.jpg', 1024)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })
    input.trigger('change')
    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.flex.items-center')
    expect(items.length).toBe(1)
    expect(items[0].text()).toContain('test.jpg')
    expect(items[0].text()).toContain('0%')
  })

  it('updates progress via onUploadProgress', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((_url: string, _data: any, config: any) => {
      return new Promise((resolve) => {
        if (config?.onUploadProgress) {
          config.onUploadProgress(fakeProgressEvent(512, 1024))
        }
        resolve({
          data: {
            batch_id: 'batch-1',
            jobs: [{ job_id: 'job-1', filename: 'test.jpg', status: 'queued' }],
          },
        })
      })
    })

    const wrapper = mount(UploadView)
    const file = makeFile('test.jpg', 1024)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })
    await input.trigger('change')

    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.flex.items-center')
    expect(items.length).toBe(1)
    expect(items[0].text()).toContain('test.jpg')
  })

  it('replaces uploading jobs with real jobs after server response', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((_url: string, _data: any, config: any) => {
      return new Promise((resolve) => {
        if (config?.onUploadProgress) {
          config.onUploadProgress(fakeProgressEvent(1024, 1024))
        }
        resolve({
          data: {
            batch_id: 'batch-1',
            jobs: [
              { job_id: 'real-1', filename: 'test.jpg', status: 'queued' },
            ],
          },
        })
      })
    })

    const wrapper = mount(UploadView)
    const file = makeFile('test.jpg', 1024)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })

    await input.trigger('change')
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.flex.items-center')
    expect(items.length).toBe(1)
    expect(items[0].text()).toContain('queued')
  })

  it('marks jobs as failed on POST error', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockRejectedValue(new Error('Network error'))

    const wrapper = mount(UploadView)
    const file = makeFile('fail.jpg', 512)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })

    await input.trigger('change')
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.flex.items-center')
    expect(items.length).toBe(1)
    expect(items[0].text()).toContain('failed')
  })

  it('shows retry button for failed jobs', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockRejectedValue(new Error('Network error'))

    const wrapper = mount(UploadView)
    const file = makeFile('fail.jpg', 512)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })

    await input.trigger('change')
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Retry')
  })

  it('shows progress bar during upload phase', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((_url: string, _data: any, config: any) => {
      return new Promise(() => {
        if (config?.onUploadProgress) {
          config.onUploadProgress(fakeProgressEvent(256, 1024))
        }
      })
    })

    const wrapper = mount(UploadView)
    const file = makeFile('bar.jpg', 1024)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })

    await input.trigger('change')
    await wrapper.vm.$nextTick()

    const progressBars = wrapper.findAll('.h-1\\.5')
    expect(progressBars.length).toBe(1)
  })
})
