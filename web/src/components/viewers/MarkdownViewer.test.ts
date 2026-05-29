import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MarkdownViewer from './MarkdownViewer.vue'

describe('MarkdownViewer', () => {
  it('renders headings', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '# Main Title\n## Subtitle\n### Section' },
    })

    const html = wrapper.html()
    expect(html).toContain('<h1')
    expect(html).toContain('Main Title')
    expect(html).toContain('<h2')
    expect(html).toContain('Subtitle')
    expect(html).toContain('<h3')
    expect(html).toContain('Section')
  })

  it('renders paragraphs', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: 'Hello world\n\nAnother paragraph' },
    })

    const html = wrapper.html()
    expect(html).toContain('<p>')
    expect(html).toContain('Hello world')
    expect(html).toContain('Another paragraph')
  })

  it('renders code blocks', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '```js\nconst x = 1;\n```' },
    })

    const html = wrapper.html()
    expect(html).toContain('<pre>')
    expect(html).toContain('<code')
    expect(html).toContain('const x = 1;')
  })

  it('renders inline code', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: 'Use `const` keyword' },
    })

    const html = wrapper.html()
    expect(html).toContain('<code>')
    expect(html).toContain('const')
  })

  it('renders lists', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '- Item 1\n- Item 2\n- Item 3' },
    })

    const html = wrapper.html()
    expect(html).toContain('<ul>')
    expect(html).toContain('<li>')
    expect(html).toContain('Item 1')
    expect(html).toContain('Item 2')
    expect(html).toContain('Item 3')
  })

  it('renders ordered lists', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '1. First\n2. Second\n3. Third' },
    })

    const html = wrapper.html()
    expect(html).toContain('<ol>')
  })

  it('renders links', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '[Click here](https://example.com)' },
    })

    const html = wrapper.html()
    expect(html).toContain('<a')
    expect(html).toContain('href="https://example.com"')
    expect(html).toContain('Click here')
  })

  it('renders bold and italic', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '**bold** and *italic*' },
    })

    const html = wrapper.html()
    expect(html).toContain('<strong>')
    expect(html).toContain('bold')
    expect(html).toContain('<em>')
  })

  it('renders blockquotes', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '> This is a quote' },
    })

    const html = wrapper.html()
    expect(html).toContain('<blockquote>')
    expect(html).toContain('This is a quote')
  })

  it('renders tables', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '| A | B |\n|---|---|\n| 1 | 2 |' },
    })

    const html = wrapper.html()
    expect(html).toContain('<table>')
    expect(html).toContain('<th>')
    expect(html).toContain('A')
    expect(html).toContain('B')
    expect(html).toContain('1')
    expect(html).toContain('2')
  })

  it('handles empty content', () => {
    const wrapper = mount(MarkdownViewer, {
      props: { content: '' },
    })

    const html = wrapper.html()
    expect(html).toContain('markdown-body')
  })
})
