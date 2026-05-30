import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import GalleryTableView from './GalleryTableView.vue'

function makeFile(id: string, overrides: Record<string, any> = {}) {
  return {
    id,
    originalName: `photo-${id}.jpg`,
    filename: `photo-${id}.jpg`,
    sizeBytes: 2457600,
    mimeType: 'image/jpeg',
    mediaType: 'photo',
    takenAt: '2025-07-15T12:00:00Z',
    ...overrides,
  }
}

describe('GalleryTableView', () => {
  it('is a valid component', () => {
    const files = [makeFile('1'), makeFile('2'), makeFile('3')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: true },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryTableView, {
      props: { files: [], selectedIds: new Set<string>(), selectionEnabled: true },
    })
    expect(wrapper.exists()).toBe(true)
  })
})
