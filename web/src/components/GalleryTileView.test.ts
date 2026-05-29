import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import GalleryTileView from './GalleryTileView.vue'

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
      lg: { url: `/thumb/${id}-lg.jpg`, width: 300, height: 300 },
      md: { url: `/thumb/${id}-md.jpg`, width: 600, height: 600 },
    },
    ...overrides,
  }
}

describe('GalleryTileView', () => {
  it('renders correct number of thumbnail cards', () => {
    const files = [makeFile('1'), makeFile('2'), makeFile('3')]
    const wrapper = mount(GalleryTileView, { props: { files, selectedIds: new Set<string>(), selectionEnabled: true } })

    const cards = wrapper.findAll('.grid > div')
    expect(cards.length).toBe(3)
  })

  it('passes thumbSize prop to thumbnail cards', () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryTileView, { props: { files, thumbSize: 'sm', selectedIds: new Set<string>(), selectionEnabled: true } })

    const img = wrapper.find('img')
    expect(img.attributes('src')).toBe('/thumb/1-lg.jpg')
  })

  it('emits open with correct index on click', async () => {
    const files = [makeFile('1'), makeFile('2')]
    const wrapper = mount(GalleryTileView, { props: { files, selectedIds: new Set(["fake-id"]), selectionEnabled: true } })

    const cards = wrapper.findAll('.grid > div')
    await cards[1].find('.cursor-pointer').trigger('click')

    expect(wrapper.emitted('open')).toBeTruthy()
    expect(wrapper.emitted('open')![0]).toEqual([1])
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryTileView, { props: { files: [], selectedIds: new Set<string>(), selectionEnabled: true } })

    const cards = wrapper.findAll('.grid > div')
    expect(cards.length).toBe(0)
  })
})
