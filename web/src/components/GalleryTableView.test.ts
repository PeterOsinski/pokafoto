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
  it('renders rows for all files', () => {
    const files = [makeFile('1'), makeFile('2'), makeFile('3')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: true },
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(3)
  })

  it('emits open with correct index on row click', async () => {
    const files = [makeFile('1'), makeFile('2')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: true },
    })

    const rows = wrapper.findAll('tbody tr')
    await rows[0].find('[class*="cursor-pointer"]').trigger('click')
    expect(wrapper.emitted('open')![0]).toEqual([0])
  })

  it('shows file name column', () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: true },
    })

    expect(wrapper.text()).toContain('photo-1.jpg')
  })

  it('emits download on download button click', async () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: false },
    })

    const downloadBtn = wrapper.find('button[title="Download"]')
    await downloadBtn.trigger('click')
    expect(wrapper.emitted('download')![0]).toEqual(['1'])
  })

  it('emits delete on delete button click', async () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: false },
    })

    const deleteBtn = wrapper.find('button[title="Delete"]')
    await deleteBtn.trigger('click')
    expect(wrapper.emitted('delete')![0]).toEqual(['1'])
  })

  it('shows checkboxes when selection is enabled', () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: true },
    })

    const checks = wrapper.findAll('thead th')
    expect(checks.length).toBeGreaterThanOrEqual(1)
  })

  it('hides checkboxes when selection is disabled', () => {
    const files = [makeFile('1')]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: false },
    })

    const ths = wrapper.findAll('thead th')
    const firstCell = ths[0].text()
    expect(firstCell).not.toBe('')
    expect(ths.length).toBe(5)

    const tds = wrapper.findAll('tbody tr td')
    expect(tds.length).toBe(5)
  })

  it('handles empty files array', () => {
    const wrapper = mount(GalleryTableView, {
      props: { files: [], selectedIds: new Set<string>(), selectionEnabled: true },
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(0)
  })

  it('shows formatted file size', () => {
    const files = [makeFile('1', { sizeBytes: 2500000 })]
    const wrapper = mount(GalleryTableView, {
      props: { files, selectedIds: new Set<string>(), selectionEnabled: true },
    })

    expect(wrapper.text()).toContain('2.4')
  })
})
