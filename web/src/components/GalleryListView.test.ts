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
  it('is a valid component with files', () => {
    const files = [makeFile('1'), makeFile('2'), makeFile('3')]
    const wrapper = mount(GalleryListView, { props: { files, thumbSizePx: 100, selectedIds: new Set<string>(), selectionEnabled: true } })
    expect(wrapper.exists()).toBe(true)
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryListView, { props: { files: [], thumbSizePx: 100, selectedIds: new Set<string>(), selectionEnabled: true } })
    expect(wrapper.exists()).toBe(true)
  })
})
