import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import UploadView from './UploadView.vue'
import { useAuthStore } from '../stores/auth'

let mockWs: {
  url: string
  onmessage: ((event: MessageEvent) => void) | null
  onerror: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  close: () => void
} | null = null

function fakeWSMessage(data: object) {
  if (mockWs?.onmessage) {
    mockWs.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }))
  }
}

vi.mock('../api/client', () => {
  const api = {
    post: vi.fn(),
    get: vi.fn(),
  }
  return { default: api }
})

function makeFile(name: string, size: number, type?: string): File {
  return new File([new ArrayBuffer(size)], name, { type: type || 'image/jpeg' })
}

function fakeProgressEvent(loaded: number, total: number) {
  return { loaded, total }
}

describe('UploadView', () => {
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

  it('shows files with uploading status immediately on selection', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((url: string) => {
      if (url === '/upload/check') {
        return Promise.resolve({ data: { duplicates: [] } })
      }
      return new Promise(() => {})
    })

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

  it('marks duplicate files as skipped after pre-upload check', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((url: string) => {
      if (url === '/upload/check') {
        return Promise.resolve({
          data: {
            duplicates: [{ filename: 'existing.jpg', file_id: 'file-1', size: 2048 }],
          },
        })
      }
      return new Promise(() => {})
    })

    const wrapper = mount(UploadView)
    const existingFile = makeFile('existing.jpg', 2048)
    const newFile = makeFile('new.jpg', 1024)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [existingFile, newFile],
      writable: false,
    })
    await input.trigger('change')
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.flex.items-center')
    expect(items.length).toBe(2)

    const skippedItem = items.find((el) => el.text().includes('existing.jpg'))
    expect(skippedItem).toBeTruthy()
    expect(skippedItem!.text()).toContain('skipped')
  })

  it('uploads each file independently with separate progress', async () => {
    const api = (await import('../api/client')).default as any
    const progressCallbacks: Array<(pe: { loaded: number; total: number }) => void> = []

    api.post.mockImplementation((url: string, _data: any, config: any) => {
      if (url === '/upload/check') {
        return Promise.resolve({ data: { duplicates: [] } })
      }
      if (config?.onUploadProgress) {
        progressCallbacks.push(config.onUploadProgress)
      }
      return new Promise((resolve) => {
        setTimeout(() => {
          resolve({
            data: {
              batch_id: 'batch-1',
              jobs: [{ job_id: 'real-1', filename: 'a.jpg', status: 'queued' }],
            },
          })
        }, 5)
      })
    })

    const wrapper = mount(UploadView)
    const file1 = makeFile('a.jpg', 1000)
    const file2 = makeFile('b.jpg', 2000)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file1, file2],
      writable: false,
    })
    await input.trigger('change')
    await wrapper.vm.$nextTick()

    expect(progressCallbacks.length).toBeGreaterThanOrEqual(1)
  })

  it('replaces uploading jobs with real jobs after server response', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((url: string, _data: any, config: any) => {
      if (url === '/upload/check') {
        return Promise.resolve({ data: { duplicates: [] } })
      }
      if (config?.onUploadProgress) {
        config.onUploadProgress(fakeProgressEvent(1024, 1024))
      }
      return Promise.resolve({
        data: {
          batch_id: 'batch-1',
          jobs: [
            { job_id: 'real-1', filename: 'test.jpg', status: 'queued' },
          ],
        },
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
    api.post.mockImplementation((url: string) => {
      if (url === '/upload/check') {
        return Promise.resolve({ data: { duplicates: [] } })
      }
      return Promise.reject(new Error('Network error'))
    })

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

  it('shows progress bar during upload phase', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((url: string, _data: any, config: any) => {
      if (url === '/upload/check') {
        return Promise.resolve({ data: { duplicates: [] } })
      }
      if (config?.onUploadProgress) {
        config.onUploadProgress(fakeProgressEvent(256, 1024))
      }
      return new Promise(() => {})
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

  it('receives processing updates via global WebSocket', async () => {
    const api = (await import('../api/client')).default as any
    api.post.mockImplementation((url: string, _data: any, _config: any) => {
      if (url === '/upload/check') {
        return Promise.resolve({ data: { duplicates: [] } })
      }
      return Promise.resolve({
        data: {
          batch_id: 'ws-batch',
          jobs: [{ job_id: 'ws-job', filename: 'ws-file.jpg', status: 'queued' }],
        },
      })
    })

    const wrapper = mount(UploadView)
    expect(mockWs).not.toBeNull()

    const file = makeFile('ws-file.jpg', 512)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })

    await input.trigger('change')
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()

    expect(mockWs).not.toBeNull()

    fakeWSMessage({ job_id: 'ws-job', filename: 'ws-file.jpg', status: 'processing', stage: 'hashing', progress: 0.3 })

    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.flex.items-center')
    expect(items[0].text()).toContain('hashing')
  })
})
