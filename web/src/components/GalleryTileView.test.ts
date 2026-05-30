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
      preview: { url: `/thumb/${id}-preview.webp`, width: 720, height: 720 },
    },
    ...overrides,
  }
}

describe('GalleryTileView', () => {
  it('is a valid component', () => {
    const files = [makeFile('1'), makeFile('2'), makeFile('3')]
    const wrapper = mount(GalleryTileView, { props: { files, thumbSizePx: 200, selectedIds: new Set<string>(), selectionEnabled: true } })
    expect(wrapper.exists()).toBe(true)
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryTileView, { props: { files: [], thumbSizePx: 200, selectedIds: new Set<string>(), selectionEnabled: true } })
    expect(wrapper.exists()).toBe(true)
  })
})
