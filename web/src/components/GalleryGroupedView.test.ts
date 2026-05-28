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
  it('groups files by day correctly', () => {
    const files = [
      makeFile('1', { takenAt: '2024-03-15T10:00:00Z' }),
      makeFile('2', { takenAt: '2024-03-15T14:00:00Z' }),
      makeFile('3', { takenAt: '2024-03-16T09:00:00Z' }),
    ]
    const wrapper = mount(GalleryGroupedView, { props: { files } })

    const headers = wrapper.findAll('h3')
    expect(headers.length).toBe(2)
    expect(headers[0].text()).toContain('1 photo')
    expect(headers[1].text()).toContain('2 photos')
  })

  it('sorts groups newest first', () => {
    const files = [
      makeFile('1', { takenAt: '2024-01-15T10:00:00Z' }),
      makeFile('2', { takenAt: '2024-03-20T10:00:00Z' }),
    ]
    const wrapper = mount(GalleryGroupedView, { props: { files } })

    const headers = wrapper.findAll('h3')
    expect(headers.length).toBe(2)
    expect(headers[0].text()).toContain('Mar 20')
    expect(headers[1].text()).toContain('Jan 15')
  })

  it('groups files without takenAt into Unknown date group', () => {
    const files = [
      makeFile('1', { takenAt: '2024-03-15T10:00:00Z' }),
      makeFile('2', { takenAt: undefined }),
      makeFile('3', { takenAt: undefined }),
    ]
    const wrapper = mount(GalleryGroupedView, { props: { files } })

    const headers = wrapper.findAll('h3')
    expect(headers.length).toBe(2)

    const unknownHeader = headers.find(h => h.text().includes('Unknown date'))
    expect(unknownHeader).toBeTruthy()
    expect(unknownHeader!.text()).toContain('2 photos')
  })

  it('emits open with correct original index', async () => {
    const files = [
      makeFile('1', { takenAt: '2024-03-15T10:00:00Z' }),
      makeFile('2', { takenAt: '2024-03-16T10:00:00Z' }),
    ]
    const wrapper = mount(GalleryGroupedView, { props: { files } })

    const grids = wrapper.findAll('.grid')
    const firstGroupCards = grids[0].findAll('.grid > div')
    await firstGroupCards[0].trigger('click')

    expect(wrapper.emitted('open')).toBeTruthy()
    expect(wrapper.emitted('open')![0]).toEqual([1])
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryGroupedView, { props: { files: [] } })

    const headers = wrapper.findAll('h3')
    expect(headers.length).toBe(0)
  })

  it('passes thumbSize prop to thumbnail cards', () => {
    const files = [
      makeFile('1', {
        takenAt: '2024-03-15T10:00:00Z',
        thumbnails: {
          sm: { url: '/thumb/1-sm.jpg', width: 60, height: 60 },
          lg: { url: '/thumb/1-lg.jpg', width: 300, height: 300 },
          md: { url: '/thumb/1-md.jpg', width: 600, height: 600 },
        },
      }),
    ]
    const wrapper = mount(GalleryGroupedView, { props: { files, thumbSize: 'sm' } })

    const img = wrapper.find('img')
    expect(img.attributes('src')).toBe('/thumb/1-lg.jpg')
  })
})
