import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import GalleryGroupedView from './GalleryGroupedView.vue'

function makeFile(id: string, overrides: Record<string, any> = {}) {
  return {
    id,
    originalName: `photo-${id}.jpg`,
    filename: `photo-${id}.jpg`,
    sizeBytes: 1024,
    mimeType: 'image/jpeg',
    mediaType: 'photo',
    thumbnails: {
      md: { url: `/thumb/${id}-md.jpg`, width: 600, height: 600 },
    },
    ...overrides,
  }
}

describe('GalleryGroupedView', () => {
  it('is a valid component', () => {
    const files = [
      makeFile('1', { takenAt: '2024-03-15T10:00:00Z' }),
      makeFile('2', { takenAt: '2024-03-16T09:00:00Z' }),
    ]
    const wrapper = mount(GalleryGroupedView, {
      props: { files, thumbSizePx: 200, selectedIds: new Set<string>(), selectionEnabled: true },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryGroupedView, {
      props: { files: [], thumbSizePx: 200, selectedIds: new Set<string>(), selectionEnabled: true },
    })
    expect(wrapper.exists()).toBe(true)
  })
})
