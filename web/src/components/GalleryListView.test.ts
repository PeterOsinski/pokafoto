import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import GalleryListView from './GalleryListView.vue'

function makeFile(id: string, overrides: Record<string, any> = {}) {
  return {
    id,
    originalName: `photo-${id}.jpg`,
    filename: `photo-${id}.jpg`,
    sizeBytes: 1048576,
    mimeType: 'image/jpeg',
    mediaType: 'photo',
    takenAt: '2024-03-15T10:30:00Z',
    thumbnails: {
      sm: { url: `/thumb/${id}-sm.jpg`, width: 60, height: 60 },
    },
    ...overrides,
  }
}

describe('GalleryListView', () => {
  it('renders correct number of rows', () => {
    const files = [makeFile('1'), makeFile('2'), makeFile('3')]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(3)
  })

  it('shows file names in name column', () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    expect(wrapper.text()).toContain('photo-1.jpg')
  })

  it('shows type badge with correct media type', () => {
    const files = [makeFile('1'), makeFile('2', { mediaType: 'video' })]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    const badges = wrapper.findAll('tbody tr td:nth-child(5) span')
    expect(badges[0].text()).toBe('photo')
    expect(badges[1].text()).toBe('video')
  })

  it('renders thumbnail image when sm thumbnail exists', () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('/thumb/1-sm.jpg')
  })

  it('shows fallback icon when no thumbnail for photo', () => {
    const files = [makeFile('1', { thumbnails: undefined })]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).toContain('📄')
  })

  it('shows extension icon for file type without thumbnail', () => {
    const files = [makeFile('1', {
      mediaType: 'file',
      originalName: 'report.csv',
      filename: 'report.csv',
      thumbnails: undefined,
    })]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).toContain('📄')
    expect(wrapper.text()).toContain('.csv')
  })

  it('emits open with correct index on row click', async () => {
    const files = [makeFile('1'), makeFile('2')]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    const rows = wrapper.findAll('tbody tr')
    await rows[1].find('td:not(:first-child)').trigger('click')

    expect(wrapper.emitted('open')).toBeTruthy()
    expect(wrapper.emitted('open')![0]).toEqual([1])
  })

  it('formats file size correctly', () => {
    const files = [makeFile('1', { sizeBytes: 1024 * 1024 * 5.5 })]
    const wrapper = mount(GalleryListView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    expect(wrapper.text()).toContain('5.5 MB')
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryListView, { props: { files: [], selectedIds: new Set<string>(), selectionEnabled: true } })

    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(0)
  })
})
