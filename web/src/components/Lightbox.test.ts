import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useLocalSettings, _resetSingleton } from '../composables/useLocalSettings'
import Lightbox from './Lightbox.vue'

const { mockApiGet, mockAccessToken } = vi.hoisted(() => ({
  mockApiGet: vi.fn(),
  mockAccessToken: vi.fn(() => ''),
}))

vi.mock('@/api/client', () => ({
  default: {
    get: mockApiGet,
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../stores/auth', () => ({
  useAuthStore: () => ({
    accessToken: mockAccessToken(),
  }),
}))

function makeFile(overrides: Record<string, any> = {}) {
  return {
    id: 'file-1',
    originalName: 'test.jpg',
    filename: 'test.jpg',
    sizeBytes: 1024,
    mimeType: 'image/jpeg',
    mediaType: 'photo',
    thumbnails: {
      sm: { url: '/thumb/sm.jpg', width: 200, height: 200 },
      md: { url: '/thumb/md.jpg', width: 800, height: 600 },
      preview: { url: '/thumb/preview.jpg', width: 1600, height: 1200 },
    },
    ...overrides,
  }
}

function mountLightbox(props: Record<string, any> = {}) {
  return mount(Lightbox, {
    props: {
      file: makeFile(),
      index: 0,
      total: 1,
      hasPrev: false,
      hasNext: false,
      ...props,
    },
  })
}

describe('Lightbox', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    _resetSingleton()
    vi.clearAllMocks()
    mockApiGet.mockResolvedValue({ data: {} })
  })

  describe('rendering', () => {
    it('renders image when file has preview thumbnail', async () => {
      const wrapper = mountLightbox()
      await flushPromises()

      const img = wrapper.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/thumb/preview.jpg')
    })

    it('falls back to md thumbnail when preview is missing', async () => {
      const file = makeFile({ thumbnails: { sm: { url: '/thumb/sm.jpg', width: 200, height: 200 }, md: { url: '/thumb/md.jpg', width: 800, height: 600 } } })
      const wrapper = mount(Lightbox, { props: { file, index: 0, total: 1, hasPrev: false, hasNext: false } })
      await flushPromises()

      const img = wrapper.find('img')
      expect(img.attributes('src')).toBe('/thumb/md.jpg')
    })

    it('shows no preview message when no thumbnails exist', async () => {
      const file = makeFile({ thumbnails: {} })
      const wrapper = mount(Lightbox, { props: { file, index: 0, total: 1, hasPrev: false, hasNext: false } })
      await flushPromises()

      expect(wrapper.text()).toContain('No preview available')
    })

    it('does not render when file is null', () => {
      const wrapper = mount(Lightbox, { props: { file: null, index: 0, total: 0, hasPrev: false, hasNext: false } })
      expect(wrapper.find('img').exists()).toBe(false)
    })
  })

  describe('keyboard navigation', () => {
    it('emits close on Escape', async () => {
      const wrapper = mountLightbox({ hasPrev: true, hasNext: true })
      await flushPromises()

      const el = wrapper.find('[tabindex="0"]')
      await el.trigger('keydown', { key: 'Escape' })

      expect(wrapper.emitted('close')).toBeTruthy()
    })

    it('emits prev on ArrowLeft when hasPrev', async () => {
      const wrapper = mountLightbox({ hasPrev: true, hasNext: false, total: 2, index: 1 })
      await flushPromises()

      const el = wrapper.find('[tabindex="0"]')
      await el.trigger('keydown', { key: 'ArrowLeft' })

      expect(wrapper.emitted('prev')).toBeTruthy()
    })

    it('does not emit prev on ArrowLeft when !hasPrev', async () => {
      const wrapper = mountLightbox({ hasPrev: false, hasNext: true })
      await flushPromises()

      const el = wrapper.find('[tabindex="0"]')
      await el.trigger('keydown', { key: 'ArrowLeft' })

      expect(wrapper.emitted('prev')).toBeFalsy()
    })

    it('emits next on ArrowRight when hasNext', async () => {
      const wrapper = mountLightbox({ hasPrev: false, hasNext: true, total: 2, index: 0 })
      await flushPromises()

      const el = wrapper.find('[tabindex="0"]')
      await el.trigger('keydown', { key: 'ArrowRight' })

      expect(wrapper.emitted('next')).toBeTruthy()
    })

    it('does not emit next on ArrowRight when !hasNext', async () => {
      const wrapper = mountLightbox({ hasPrev: true, hasNext: false })
      await flushPromises()

      const el = wrapper.find('[tabindex="0"]')
      await el.trigger('keydown', { key: 'ArrowRight' })

      expect(wrapper.emitted('next')).toBeFalsy()
    })
  })

  describe('file info display', () => {
    it('shows file name and size', async () => {
      const wrapper = mountLightbox()
      await flushPromises()

      expect(wrapper.text()).toContain('test.jpg')
      expect(wrapper.text()).toContain('1.0 KB')
    })

    it('has download button', async () => {
      const wrapper = mountLightbox()
      await flushPromises()

      expect(wrapper.text()).toContain('Download')
    })

    it('downloads file when download button is clicked', async () => {
      const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
      mockAccessToken.mockReturnValue('test-token-123')
      const wrapper = mountLightbox()
      await flushPromises()

      const downloadBtn = wrapper.findAll('button').find(b => b.text().includes('Download'))
      expect(downloadBtn).toBeTruthy()
      await downloadBtn!.trigger('click')
      await flushPromises()

      expect(openSpy).toHaveBeenCalledWith('/api/v1/download/file-1?token=test-token-123', '_blank')
      openSpy.mockRestore()
    })

    it('displays EXIF data when API returns it', async () => {
      mockApiGet.mockResolvedValueOnce({
        data: {
          exif: {
            cameraMake: 'Canon',
            cameraModel: 'EOS R5',
            focalLength: 50,
            aperture: 1.4,
            shutterSpeed: '1/1000',
            iso: 400,
            dateTaken: '2024-03-15',
            gpsLatitude: 48.8566,
            gpsLongitude: 2.3522,
          },
        },
      })

      const wrapper = mountLightbox()
      await flushPromises()

      expect(wrapper.text()).toContain('Canon')
      expect(wrapper.text()).toContain('EOS R5')
      expect(wrapper.text()).toContain('50mm')
      expect(wrapper.text()).toContain('f/1.4')
      expect(wrapper.text()).toContain('1/1000s')
      expect(wrapper.text()).toContain('ISO 400')
      expect(wrapper.text()).toContain('48.8566')
    })
  })

  describe('2000px checkbox', () => {
    function xlFile() {
      return makeFile({
        thumbnails: {
          sm: { url: '/thumb/sm.jpg', width: 200, height: 200 },
          md: { url: '/thumb/md.jpg', width: 800, height: 600 },
          xl: { url: '/thumb/xl.jpg', width: 2000, height: 1500 },
          preview: { url: '/thumb/preview.jpg', width: 1600, height: 1200 },
        },
      })
    }

    it('renders 2000px checkbox when file has xl thumbnails', async () => {
      const wrapper = mount(Lightbox, { props: { file: xlFile(), index: 0, total: 1, hasPrev: false, hasNext: false } })
      await flushPromises()

      const label = wrapper.find('label')
      expect(label.text()).toContain('2000px')
    })

    it('does not render 2000px checkbox when file has no xl thumbnails', async () => {
      const file = makeFile()
      const wrapper = mount(Lightbox, { props: { file, index: 0, total: 1, hasPrev: false, hasNext: false } })
      await flushPromises()

      expect(wrapper.text()).not.toContain('2000px')
    })

    it('sets highResDownload ref to true when checkbox is checked', async () => {
      const settings = useLocalSettings()
      const wrapper = mount(Lightbox, { props: { file: xlFile(), index: 0, total: 1, hasPrev: false, hasNext: false } })
      await flushPromises()

      const checkbox = wrapper.find<HTMLInputElement>('input[type="checkbox"]')
      await checkbox.setValue(true)

      expect(settings.highResDownload.value).toBe(true)
    })
  })
})
