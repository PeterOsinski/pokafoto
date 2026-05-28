import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ThumbnailCard from './ThumbnailCard.vue'

function makeFile(overrides: Record<string, any> = {}) {
  return {
    id: 'file-1',
    originalName: 'test.jpg',
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

describe('ThumbnailCard', () => {
  describe('rendering', () => {
    it('renders image with correct src when thumbnail exists', () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file } })

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/lg.jpg')
    })

    it('renders lg thumbnail when thumbSize is sm', () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'sm' } })

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/lg.jpg')
    })

    it('renders md thumbnail when thumbSize is lg', () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'lg' } })

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/md.jpg')
    })

    it('renders lg thumbnail when thumbSize is md', () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'md' } })

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/lg.jpg')
    })

    it('falls back to md when requested size is missing', () => {
      const file = makeFile({ thumbnails: { md: { url: '/thumb/md.jpg', width: 600, height: 600 } } })
      const wrapper = mount(ThumbnailCard, { props: { file, thumbSize: 'sm' } })

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/md.jpg')
    })

    it('renders ▶ icon for video without thumbnail', () => {
      const file = makeFile({ mediaType: 'video', thumbnails: undefined })
      const wrapper = mount(ThumbnailCard, { props: { file } })

      expect(wrapper.text()).toContain('▶')
    })

    it('renders 📄 icon for file without thumbnail', () => {
      const file = makeFile({ mediaType: 'file', thumbnails: undefined })
      const wrapper = mount(ThumbnailCard, { props: { file } })

      expect(wrapper.text()).toContain('📄')
    })

    it('shows duration badge for videos with durationSec', () => {
      const file = makeFile({
        mediaType: 'video',
        thumbnails: { md: { url: '/thumb/md.jpg', width: 600, height: 600 } },
        durationSec: 125,
      })
      const wrapper = mount(ThumbnailCard, { props: { file } })

      expect(wrapper.text()).toContain('2:05')
    })

    it('does not show duration badge for photos', () => {
      const file = makeFile({ durationSec: 30 })
      const wrapper = mount(ThumbnailCard, { props: { file } })

      expect(wrapper.text()).not.toContain(':')
    })

    it('shows takenAt date when available', () => {
      const file = makeFile({ takenAt: '2024-03-15T10:30:00Z' })
      const wrapper = mount(ThumbnailCard, { props: { file } })

      expect(wrapper.text()).toContain('Mar 15')
    })
  })

  describe('error state', () => {
    it('shows retry button when image fails to load', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file } })

      await wrapper.find('img').trigger('error')
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('Retry')
    })

    it('retries loading image when retry button is clicked', async () => {
      const file = makeFile()
      const wrapper = mount(ThumbnailCard, { props: { file } })

      await wrapper.find('img').trigger('error')
      await wrapper.vm.$nextTick()

      expect(wrapper.find('img').exists()).toBe(false)

      await wrapper.find('button').trigger('click')
      await wrapper.vm.$nextTick()

      expect(wrapper.find('img').exists()).toBe(true)
    })
  })
})
