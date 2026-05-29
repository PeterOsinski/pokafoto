import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import FileViewer from './FileViewer.vue'

const { mockApiGet } = vi.hoisted(() => ({
  mockApiGet: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  default: {
    get: mockApiGet,
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

function makeFile(overrides: Record<string, any> = {}) {
  return {
    id: 'file-1',
    originalName: 'document.txt',
    sizeBytes: 1024,
    mimeType: 'text/plain',
    ...overrides,
  }
}

function mountViewer(props: Record<string, any> = {}) {
  return mount(FileViewer, {
    props: {
      file: makeFile(),
      ...props,
    },
  })
}

describe('FileViewer', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('rendering', () => {
    it('does not render when file is null', () => {
      const wrapper = mount(FileViewer, { props: { file: null } })
      expect(wrapper.find('[tabindex="0"]').exists()).toBe(false)
    })

    it('shows loading state initially', async () => {
      mockApiGet.mockReturnValue(new Promise(() => {}))
      const wrapper = mountViewer()
      expect(wrapper.text()).toContain('Loading...')
    })

    it('shows error state when API fails', async () => {
      mockApiGet.mockRejectedValue(new Error('Network error'))
      const wrapper = mountViewer()
      await flushPromises()
      expect(wrapper.text()).toContain('Could not load this file')
      expect(wrapper.text()).toContain('Download raw file')
    })

    it('displays file info in footer', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['hello world']) })
      const wrapper = mountViewer()
      await flushPromises()
      expect(wrapper.text()).toContain('1.0 KB')
      expect(wrapper.text()).toContain('text/plain')
    })

    it('has a download button in header', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['hello world']) })
      const wrapper = mountViewer()
      await flushPromises()
      expect(wrapper.text()).toContain('Download')
    })
  })

  describe('viewer type detection', () => {
    it('detects PDF by extension', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['%PDF-1.4 fake']) })
      const wrapper = mountViewer({ file: makeFile({ originalName: 'doc.pdf', mimeType: 'application/pdf' }) })
      await flushPromises()
      const iframe = wrapper.find('iframe')
      expect(iframe.exists()).toBe(true)
    })

    it('detects JSON by extension', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['{"key": "value"}']) })
      const wrapper = mountViewer({ file: makeFile({ originalName: 'data.json', mimeType: 'application/json' }) })
      await flushPromises()
      expect(wrapper.text()).toContain('"key"')
      expect(wrapper.text()).toContain('"value"')
    })

    it('detects Markdown by extension', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['# Hello']) })
      const wrapper = mountViewer({ file: makeFile({ originalName: 'README.md', mimeType: 'text/markdown' }) })
      await flushPromises()
      expect(wrapper.html()).toContain('markdown-body')
    })

    it('detects CSV by extension', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['a,b,c\n1,2,3']) })
      const wrapper = mountViewer({ file: makeFile({ originalName: 'data.csv', mimeType: 'text/csv' }) })
      await flushPromises()
      expect(wrapper.find('table').exists()).toBe(true)
    })

    it('falls back to text viewer for unknown types', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['plain text']) })
      const wrapper = mountViewer({ file: makeFile({ originalName: 'file.xyz', mimeType: 'application/octet-stream' }) })
      await flushPromises()
      expect(wrapper.text()).toContain('plain text')
    })
  })

  describe('keyboard navigation', () => {
    it('emits close on Escape', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['hello']) })
      const wrapper = mountViewer()
      await flushPromises()

      const el = wrapper.find('[tabindex="0"]')
      await el.trigger('keydown', { key: 'Escape' })
      expect(wrapper.emitted('close')).toBeTruthy()
    })
  })

  describe('download', () => {
    it('uses cached blob for download after content has loaded', async () => {
      const blob = new Blob(['hello world'])
      mockApiGet.mockResolvedValue({ data: blob })
      const wrapper = mountViewer()
      await flushPromises()

      const links = wrapper.findAll('button')
      const downloadBtn = links.find(b => b.text().includes('Download'))
      expect(downloadBtn).toBeTruthy()

      mockApiGet.mockClear()
      await downloadBtn!.trigger('click')
      await flushPromises()
      expect(mockApiGet).not.toHaveBeenCalled()
    })

    it('fetches file on download if not yet loaded', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['hello']) })
      const wrapper = mountViewer()
      await flushPromises()

      mockApiGet.mockClear()
      mockApiGet.mockResolvedValue({ data: new Blob(['fresh download']) })

      const links = wrapper.findAll('button')
      const downloadBtn = links.find(b => b.text().includes('Download'))
      await downloadBtn!.trigger('click')
    })
  })

  describe('close button', () => {
    it('emits close when the X button is clicked', async () => {
      mockApiGet.mockResolvedValue({ data: new Blob(['hello']) })
      const wrapper = mountViewer()
      await flushPromises()

      const closeBtn = wrapper.findAll('button').find(b => b.text() === '✕')
      expect(closeBtn).toBeTruthy()
      await closeBtn!.trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
    })
  })
})
