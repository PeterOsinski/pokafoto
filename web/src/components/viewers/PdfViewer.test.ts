import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PdfViewer from './PdfViewer.vue'

describe('PdfViewer', () => {
  it('renders iframe when blobUrl is provided', () => {
    const wrapper = mount(PdfViewer, {
      props: { blobUrl: 'blob:http://localhost/abc-123' },
    })

    const iframe = wrapper.find('iframe')
    expect(iframe.exists()).toBe(true)
    expect(iframe.attributes('src')).toBe('blob:http://localhost/abc-123')
  })

  it('shows loading message when blobUrl is null', () => {
    const wrapper = mount(PdfViewer, {
      props: { blobUrl: null },
    })

    expect(wrapper.text()).toContain('Loading PDF')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })

  it('sets iframe to full width and height', () => {
    const wrapper = mount(PdfViewer, {
      props: { blobUrl: 'blob:test' },
    })

    const iframe = wrapper.find('iframe')
    expect(iframe.classes()).toContain('w-full')
    expect(iframe.classes()).toContain('h-full')
    expect(iframe.classes()).toContain('border-0')
  })
})
