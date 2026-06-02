import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import FolderTreeView from './FolderTreeView.vue'
import { useChunkedUploadStore } from '../stores/chunkedUpload'

vi.mock('../api/client', () => {
  const api = {
    get: vi.fn(),
    post: vi.fn(),
  }
  return { default: api }
})

function makeFile(id: string, overrides: Record<string, any> = {}) {
  return {
    id,
    originalName: `photo-${id}.jpg`,
    filename: `photo-${id}.jpg`,
    sizeBytes: 1024,
    mimeType: 'image/jpeg',
    mediaType: 'photo',
    thumbnails: {
      sm: { url: `/thumb/${id}-sm.jpg`, width: 60, height: 60 },
      md: { url: `/thumb/${id}-md.jpg`, width: 600, height: 600 },
      preview: { url: `/thumb/${id}-preview.webp`, width: 960, height: 720 },
    },
    ...overrides,
  }
}

describe('FolderTreeView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  async function mountComponent(folderId: string | null = null) {
    const api = (await import('../api/client')).default as any
    api.get.mockImplementation((url: string) => {
      if (url === '/folders') {
        return Promise.resolve({
          data: {
            children: [
              {
                folder: { id: 'f-1', name: 'Work', parent_id: null },
                fileCount: 5,
                children: [],
              },
            ],
          },
        })
      }
      if (url === '/files') {
        return Promise.resolve({
          data: { items: [makeFile('file-1'), makeFile('file-2')], nextCursor: '', total: 2 },
        })
      }
      return Promise.reject(new Error('unexpected url'))
    })

    return mount(FolderTreeView, {
      props: {
        folderId: folderId,
        selectedIds: new Set<string>(),
        selectionEnabled: true,
      },
    })
  }

  it('loads files on mount', async () => {
    const wrapper = await mountComponent()
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const api = (await import('../api/client')).default as any
    expect(api.get).toHaveBeenCalledWith('/folders')
    expect(api.get).toHaveBeenCalledWith('/files', expect.anything())
  })

  it('reloads files when consumeCompletedJobs returns matching folder_id', async () => {
    const wrapper = await mountComponent('f-1')
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const api = (await import('../api/client')).default as any
    const callsBefore = api.get.mock.calls.length

    const upload = useChunkedUploadStore()
    upload.completedJobs.push({
      file_id: 'new-file',
      filename: 'new.jpg',
      folder_id: 'f-1',
    })

    await new Promise((r) => setTimeout(r, 2100))
    await wrapper.vm.$nextTick()

    expect(api.get.mock.calls.length).toBeGreaterThan(callsBefore)
  })

  it('does not reload when completed job folder_id does not match', async () => {
    const wrapper = await mountComponent('f-1')
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const api = (await import('../api/client')).default as any
    const callsBefore = api.get.mock.calls.length

    const upload = useChunkedUploadStore()
    upload.completedJobs.push({
      file_id: 'new-file',
      filename: 'new.jpg',
      folder_id: 'f-other',
    })

    await new Promise((r) => setTimeout(r, 2100))
    await wrapper.vm.$nextTick()

    expect(api.get.mock.calls.length).toBe(callsBefore)
  })

  it('does not reload when completed queue is empty', async () => {
    const wrapper = await mountComponent()
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const api = (await import('../api/client')).default as any
    const callsBefore = api.get.mock.calls.length

    await new Promise((r) => setTimeout(r, 2100))
    await wrapper.vm.$nextTick()

    expect(api.get.mock.calls.length).toBe(callsBefore)
  })
})
