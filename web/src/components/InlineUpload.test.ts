import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../stores/auth'
import InlineUpload from './InlineUpload.vue'

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

describe('InlineUpload', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const auth = useAuthStore()
    auth.accessToken = 'test-token'
    vi.clearAllMocks()
  })

  it('triggers chunked upload on file selection', async () => {
    const api = (await import('../api/client')).default as any
    let chunkUploadCalled = false
    api.post.mockImplementation((url: string) => {
      if (url === '/upload/chunk') {
        chunkUploadCalled = true
        return Promise.resolve({
          data: {
            upload_id: 'upload-1',
            resume_token: 'token-1',
            stored_chunks: [0],
            missing_chunks: [],
            next_chunk: 1,
          },
        })
      }
      return new Promise(() => {})
    })

    const wrapper = mount(InlineUpload)
    const file = makeFile('test.jpg', 1024)
    const input = wrapper.find('input[type="file"]')

    Object.defineProperty(input.element, 'files', {
      value: [file],
      writable: false,
    })
    await input.trigger('change')
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    expect(chunkUploadCalled).toBe(true)
  })

  it('renders with custom label', () => {
    const wrapper = mount(InlineUpload, {
      props: { label: 'Custom action' },
    })
    expect(wrapper.find('button').text()).toBe('Custom action')
  })
})
