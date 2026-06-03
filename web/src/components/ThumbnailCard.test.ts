import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ThumbnailCard from './ThumbnailCard.vue'

function makeFile(overrides: Record<string, any> = {}) {
  return {
    id: 'file-1',
    originalName: 'test.jpg',
    filename: 'test.jpg',
    sizeBytes: 1024,
    mimeType: 'image/jpeg',
    mediaType: 'photo',
    thumbnails: {
      sm: { url: '/thumb/sm.jpg', width: 60, height: 60 },
      lg: { url: '/thumb/lg.jpg', width: 300, height: 300 },
      md: { url: '/thumb/md.jpg', width: 600, height: 600 },
      preview: { url: '/thumb/preview.webp', width: 720, height: 720 },
    },
    ...overrides,
  }
}

async function waitForIO() {
  await nextTick()
  await new Promise((r) => setTimeout(r, 10))
  await nextTick()
}

describe('ThumbnailCard', () => {
  describe('rendering', () => {
    it('renders preview.webp as thumbnail when no thumbSize', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/preview.webp')
    })

    it('renders sized thumbnail when thumbSize prop is provided', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'sm' } })
      await waitForIO()

      expect(wrapper.find('img').attributes('src')).toBe('/thumb/sm.jpg')
    })

    it('renders lg thumbnail when thumbSize is lg', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'lg' } })
      await waitForIO()

      expect(wrapper.find('img').attributes('src')).toBe('/thumb/lg.jpg')
    })

    it('renders ▶ icon for video without thumbnail', async () => {
      const file = makeFile({ mediaType: 'video', thumbnails: undefined })
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      expect(wrapper.text()).toContain('▶')
    })

    it('renders extension icon for file type without thumbnail', async () => {
      const file = makeFile({ mediaType: 'file', originalName: 'document.pdf', filename: 'document.pdf', thumbnails: undefined })
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      expect(wrapper.text()).toContain('📄')
      expect(wrapper.text()).toContain('.pdf')
    })

    it('shows duration badge for videos with durationSec', async () => {
      const file = makeFile({
        mediaType: 'video',
        thumbnails: { md: { url: '/thumb/md.jpg', width: 600, height: 600 } },
        durationSec: 125,
      })
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'md' } })
      await waitForIO()

      expect(wrapper.text()).toContain('2:05')
    })

    it('does not show duration badge for photos', async () => {
      const file = makeFile({ durationSec: 30 })
      const wrapper = mount(ThumbnailCard, { props: { file, selectable: false } })
      await waitForIO()

      expect(wrapper.text()).not.toContain(':')
    })

    it('shows takenAt date when available', async () => {
      const file = makeFile({ takenAt: '2024-03-15T10:30:00Z' })
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      expect(wrapper.text()).toContain('Mar 15')
    })
  })

  describe('error state', () => {
    it('shows retry button when image fails to load', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      await wrapper.find('img').trigger('error')
      await nextTick()

      expect(wrapper.text()).toContain('Retry')
    })

    it('retries loading image when retry button is clicked', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      await wrapper.find('img').trigger('error')
      await nextTick()

      expect(wrapper.find('img').exists()).toBe(false)

      await wrapper.find('button').trigger('click')
      await nextTick()

      expect(wrapper.find('img').exists()).toBe(true)
    })
  })

  describe('app-managed document indicator', () => {
    it('shows DOC badge for app-managed files', async () => {
      const file = makeFile({
        mediaType: 'file',
        isAppManaged: true,
        mimeType: 'text/markdown',
        thumbnails: undefined,
      })
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      expect(wrapper.text()).toContain('DOC')
    })

    it('does not show DOC badge for regular files', async () => {
      const file = makeFile({
        mediaType: 'file',
        isAppManaged: false,
        mimeType: 'text/plain',
        thumbnails: undefined,
      })
      const wrapper = mount(ThumbnailCard, { props: { file } })
      await waitForIO()

      expect(wrapper.text()).not.toContain('DOC')
    })
  })
})
