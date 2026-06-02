import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DocumentEditor from './DocumentEditor.vue'
import api from '../api/client'

vi.mock('../api/client', () => ({
  default: {
    get: vi.fn(),
    put: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
  },
}))

describe('DocumentEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    document.createRange = () =>
      ({
        setStart: () => {},
        setEnd: () => {},
        getClientRects: () => ([] as unknown) as DOMRectList,
        getBoundingClientRect: () => ({ top: 0, left: 0, bottom: 0, right: 0, width: 0, height: 0 } as DOMRect),
      }) as any
  })

  it('shows loading state initially', () => {
    const wrapper = mount(DocumentEditor, {
      props: { fileId: 'doc-1', originalName: 'test.md' },
    })
    expect(wrapper.text()).toContain('Loading document')
  })

  it('loads and displays document content', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { id: 'doc-1', content: '# Test Content', originalName: 'test.md' },
    })

    mount(DocumentEditor, {
      props: { fileId: 'doc-1', originalName: 'test.md' },
      global: {
        stubs: {
          MdEditor: {
            template: '<textarea :value="modelValue" @input="$emit(\'onChange\', $event.target.value)" />',
            props: ['modelValue', 'language', 'theme', 'previewTheme', 'toolbars'],
            emits: ['onChange'],
          },
        },
      },
    })

    await nextTick()
    await nextTick()

    expect(api.get).toHaveBeenCalledWith('/documents/doc-1')
  })

  it('emits close when X button is clicked', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { id: 'doc-1', content: '', originalName: 'test.md' },
    })

    const wrapper = mount(DocumentEditor, {
      props: { fileId: 'doc-1', originalName: 'test.md' },
      global: {
        stubs: {
          MdEditor: {
            template: '<div class="md-editor-stub" />',
            props: ['modelValue', 'language', 'theme', 'previewTheme', 'toolbars'],
          },
        },
      },
    })

    await nextTick()
    await nextTick()

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
