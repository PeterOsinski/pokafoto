import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FolderTile from './FolderTile.vue'

describe('FolderTile', () => {
  it('renders name, file count, and emoji', () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'Photos', fileCount: 12, hasShares: false, hasPassword: false },
    })
    expect(wrapper.text()).toContain('Photos')
    expect(wrapper.text()).toContain('12 files')
    expect(wrapper.text()).toContain('📁')
  })

  it('shows lock icon when hasPassword is true', () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'Secrets', fileCount: 3, hasShares: false, hasPassword: true },
    })
    expect(wrapper.text()).toContain('🔒')
  })

  it('shows no lock icon when hasPassword is false', () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'Public', fileCount: 5, hasShares: false, hasPassword: false },
    })
    expect(wrapper.text()).not.toContain('🔒')
  })

  it('shows shared icon when hasShares is true', () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'Shared', fileCount: 8, hasShares: true, hasPassword: false },
    })
    expect(wrapper.text()).toContain('🔗')
  })

  it('shows both lock and shared icons when both are true', () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'ProtectedShared', fileCount: 2, hasShares: true, hasPassword: true },
    })
    expect(wrapper.text()).toContain('🔒')
    expect(wrapper.text()).toContain('🔗')
  })

  it('emits click when button is clicked', async () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'Test', fileCount: 1, hasShares: false, hasPassword: false },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('click')).toBeTruthy()
  })

  it('emits contextmenu on right click', async () => {
    const wrapper = mount(FolderTile, {
      props: { name: 'Test', fileCount: 1, hasShares: false, hasPassword: false },
    })
    await wrapper.find('button').trigger('contextmenu')
    expect(wrapper.emitted('contextmenu')).toBeTruthy()
  })
})
