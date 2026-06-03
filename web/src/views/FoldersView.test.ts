import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { nextTick } from 'vue'
import { _resetSingleton } from '../composables/useLocalSettings'

const routerReplace = vi.fn()

vi.mock('vue-router', async () => {
  const actual = await vi.importActual('vue-router')
  return {
    ...actual as any,
    useRoute: vi.fn(() => ({
      query: {},
      path: '/folders',
      hash: '',
    })),
    useRouter: vi.fn(() => ({
      replace: routerReplace,
    })),
  }
})

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

async function setupApi(foldersData: any = { children: [] }, filesData: any = { items: [], nextCursor: '', total: 0 }) {
  const api = (await import('../api/client')).default as any
  api.get.mockImplementation((url: string) => {
    if (url === '/folders') return Promise.resolve({ data: foldersData })
    if (url === '/files') return Promise.resolve({ data: filesData })
    return Promise.reject(new Error('unexpected url'))
  })
  api.post.mockResolvedValue({ data: {} })
}

async function mountView(queryOverrides: Record<string, string> = {}) {
  const { useRoute } = await import('vue-router')
  ;(useRoute as any).mockReturnValue({
    query: queryOverrides,
    path: '/folders',
    hash: '',
  })

  const FoldersView = (await import('../views/FoldersView.vue')).default
  return mount(FoldersView, {
    global: {
      stubs: {
        routerLink: { template: '<a><slot/></a>' },
        routerView: { template: '<div><slot/></div>' },
      },
    },
  })
}

describe('FoldersView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    _resetSingleton()
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('loads folders and files on mount', async () => {
    await setupApi(
      { children: [{ folder: { id: 'f-1', name: 'Work', parent_id: null }, fileCount: 5, children: [] }] },
      { items: [makeFile('file-1')], nextCursor: '', total: 1 },
    )

    await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const api = (await import('../api/client')).default as any
    expect(api.get).toHaveBeenCalledWith('/folders')
    expect(api.get).toHaveBeenCalledWith('/files', expect.anything())
  })

  it('shows folder cards at root', async () => {
    await setupApi(
      { children: [{ folder: { id: 'f-1', name: 'Work', parent_id: null }, fileCount: 5, children: [] }] },
      { items: [], nextCursor: '', total: 0 },
    )

    const wrapper = await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 10))
    await nextTick()

    expect(wrapper.text()).toContain('Work')
    expect(wrapper.text()).toContain('5 files')
  })

  it('shows empty state when no folders', async () => {
    await setupApi({ children: [] }, { items: [], nextCursor: '', total: 0 })

    const wrapper = await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 10))
    await nextTick()

    expect(wrapper.text()).toContain('No files in this folder')
  })

  it('navigates into a folder on card click', async () => {
    await setupApi(
      { children: [{ folder: { id: 'f-1', name: 'Work', parent_id: null }, fileCount: 5, children: [] }] },
      { items: [], nextCursor: '', total: 0 },
    )

    const wrapper = await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 10))
    await nextTick()

    const card = wrapper.find('.grid button')
    await card.trigger('click')
    await nextTick()

    expect(routerReplace).toHaveBeenCalled()
  })

  it('extracts folder_id from URL query', async () => {
    await setupApi(
      { children: [] },
      { items: [], nextCursor: '', total: 0 },
    )

    await mountView({ folder_id: 'f-1' })
    await nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const api = (await import('../api/client')).default as any
    const filesCall = api.get.mock.calls.find((c: any[]) => c[0] === '/files')
    expect(filesCall).toBeTruthy()
    expect(filesCall[1].params.folder_id).toBe('f-1')
  })

  it('shows new folder create button and reveals input on click', async () => {
    await setupApi(
      { children: [] },
      { items: [], nextCursor: '', total: 0 },
    )

    const wrapper = await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 10))
    await nextTick()

    const vm = wrapper.vm as any
    expect(vm.showCreate).toBe(false)

    vm.showCreate = true
    await nextTick()

    const input = wrapper.find('input[placeholder="Folder name..."]')
    expect(input.exists()).toBe(true)
  })

  it('layout renders tiles component', async () => {
    await setupApi(
      { children: [] },
      { items: [makeFile('file-1')], nextCursor: '', total: 1 },
    )

    const wrapper = await mountView({ folder_id: 'f-1' })
    await nextTick()
    await new Promise((r) => setTimeout(r, 50))
    await nextTick()

    const tileView = wrapper.findComponent({ name: 'GalleryTileView' })
    expect(tileView.exists()).toBe(true)
  })

  it('layout grouped renders GalleryGroupedView', async () => {
    localStorage.setItem('drive:layout', 'grouped')
    await setupApi(
      { children: [] },
      { items: [makeFile('file-1')], nextCursor: '', total: 1 },
    )

    const wrapper = await mountView({ folder_id: 'f-1' })
    await nextTick()
    await new Promise((r) => setTimeout(r, 50))
    await nextTick()

    const groupedView = wrapper.findComponent({ name: 'GalleryGroupedView' })
    expect(groupedView.exists()).toBe(true)
  })

  it('layout list shows folders at top when inside a folder with subfolders', async () => {
    localStorage.setItem('drive:layout', 'list')
    await setupApi(
      {
        children: [
          {
            folder: { id: 'f-1', name: 'Work', parent_id: null },
            fileCount: 5,
            children: [
              { folder: { id: 'f-2', name: 'Sub', parent_id: 'f-1' }, fileCount: 3, children: [] },
            ],
          },
        ],
      },
      { items: [makeFile('f1')], nextCursor: '', total: 1 },
    )

    const wrapper = await mountView({ folder_id: 'f-1' })
    await nextTick()
    await new Promise((r) => setTimeout(r, 50))
    await nextTick()

    const listView = wrapper.findComponent({ name: 'GalleryListView' })
    expect(listView.exists()).toBe(true)
    expect(wrapper.text()).toContain('Sub')
    expect(wrapper.text()).toContain('3 files')
  })

  it('layout list shows root folders at top', async () => {
    localStorage.setItem('drive:layout', 'list')
    await setupApi(
      { children: [{ folder: { id: 'f-1', name: 'Work', parent_id: null }, fileCount: 5, children: [] }] },
      { items: [], nextCursor: '', total: 0 },
    )

    const wrapper = await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 50))
    await nextTick()

    expect(wrapper.text()).toContain('Work')
    expect(wrapper.text()).toContain('5 files')
    const listView = wrapper.findComponent({ name: 'GalleryListView' })
    expect(listView.exists()).toBe(true)
  })

  it('layout list navigates into folder on click', async () => {
    localStorage.setItem('drive:layout', 'list')
    await setupApi(
      { children: [{ folder: { id: 'f-1', name: 'Work', parent_id: null }, fileCount: 5, children: [] }] },
      { items: [], nextCursor: '', total: 0 },
    )

    const wrapper = await mountView()
    await nextTick()
    await new Promise((r) => setTimeout(r, 50))
    await nextTick()

    const folderButton = wrapper.find('.flex.items-center.w-full.border-b')
    await folderButton.trigger('click')
    await nextTick()

    expect(routerReplace).toHaveBeenCalled()
  })
})
