import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FolderNode from './FolderNode.vue'
import type { FolderTreeNode, FolderEntry } from '../types/gallery'

function makeNode(overrides: Partial<FolderTreeNode> = {}): FolderTreeNode {
  return {
    folder: {
      id: 'f1',
      name: 'Test Folder',
      parent_id: null,
      created_at: '2024-01-01T00:00:00Z',
    } as FolderEntry,
    fileCount: 5,
    hasShares: false,
    hasPassword: false,
    children: [],
    ...overrides,
  }
}

describe('FolderNode', () => {
  it('renders folder name and file count', () => {
    const node = makeNode()
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    expect(wrapper.text()).toContain('Test Folder')
    expect(wrapper.text()).toContain('5')
  })

  it('shows lock icon when hasPassword is true', () => {
    const node = makeNode({ hasPassword: true })
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    expect(wrapper.text()).toContain('🔒')
  })

  it('shows no lock icon when hasPassword is false', () => {
    const node = makeNode({ hasPassword: false })
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    expect(wrapper.text()).not.toContain('🔒')
  })

  it('shows link icon when hasShares is true', () => {
    const node = makeNode({ hasShares: true })
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    expect(wrapper.text()).toContain('🔗')
  })

  it('shows no link icon when hasShares is false', () => {
    const node = makeNode({ hasShares: false })
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    expect(wrapper.text()).not.toContain('🔗')
  })

  it('shows both lock and link icons when both are true', () => {
    const node = makeNode({ hasPassword: true, hasShares: true })
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    expect(wrapper.text()).toContain('🔒')
    expect(wrapper.text()).toContain('🔗')
  })

  it('emits select when clicked', async () => {
    const node = makeNode()
    const wrapper = mount(FolderNode, {
      props: { node, selectedId: null, depth: 0 },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('select')).toBeTruthy()
    expect(wrapper.emitted('select')![0]).toEqual(['f1'])
  })
})
