import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import JsonViewer from './JsonViewer.vue'

describe('JsonViewer', () => {
  it('renders formatted JSON with syntax highlighting', () => {
    const wrapper = mount(JsonViewer, {
      props: { content: JSON.stringify({ name: 'Alice', age: 30, active: true, score: null }) },
    })

    const html = wrapper.html()
    expect(html).toContain('text-yellow-300')
    expect(html).toContain('"name"')
    expect(html).toContain('text-green-300')
    expect(html).toContain('"Alice"')
    expect(html).toContain('text-blue-300')
    expect(html).toContain('30')
    expect(html).toContain('text-purple-300')
  })

  it('handles arrays', () => {
    const wrapper = mount(JsonViewer, {
      props: { content: JSON.stringify([1, 2, 3]) },
    })

    const text = wrapper.text()
    expect(text).toContain('1')
    expect(text).toContain('2')
    expect(text).toContain('3')
    expect(wrapper.html()).toContain('text-blue-300')
  })

  it('handles nested objects', () => {
    const wrapper = mount(JsonViewer, {
      props: { content: JSON.stringify({ user: { name: 'Bob', items: [1, 2] } }) },
    })

    const html = wrapper.html()
    expect(html).toContain('"user"')
    expect(html).toContain('"Bob"')
    expect(html).toContain('"items"')
  })

  it('handles malformed JSON gracefully', () => {
    const wrapper = mount(JsonViewer, {
      props: { content: '{invalid json' },
    })

    const html = wrapper.html()
    expect(html).toContain('text-red-400')
    expect(html).toContain('{invalid json')
  })

  it('handles empty object', () => {
    const wrapper = mount(JsonViewer, {
      props: { content: '{}' },
    })

    const html = wrapper.html()
    expect(html).toContain('{}')
  })

  it('handles large numbers', () => {
    const wrapper = mount(JsonViewer, {
      props: { content: JSON.stringify({ big: 1.5e10, negative: -42 }) },
    })

    const html = wrapper.html()
    expect(html).toContain('15000000000')
    expect(html).toContain('-42')
  })
})
