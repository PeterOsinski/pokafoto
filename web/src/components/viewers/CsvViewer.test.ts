import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CsvViewer from './CsvViewer.vue'

describe('CsvViewer', () => {
  it('renders CSV as a table with headers and rows', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'name,age,city\nAlice,30,NYC\nBob,25,SF' },
    })

    expect(wrapper.find('table').exists()).toBe(true)
    const headers = wrapper.findAll('th')
    expect(headers).toHaveLength(3)
    expect(headers[0].text()).toBe('name')
    expect(headers[1].text()).toBe('age')
    expect(headers[2].text()).toBe('city')

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
  })

  it('handles empty CSV', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: '' },
    })

    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.text()).toContain('No data in CSV')
  })

  it('handles header-only CSV', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'a,b,c' },
    })

    const headers = wrapper.findAll('th')
    expect(headers).toHaveLength(3)
    expect(headers[0].text()).toBe('a')
    expect(headers[1].text()).toBe('b')
    expect(headers[2].text()).toBe('c')
  })

  it('handles quoted fields with commas', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'name,description\n"Acme, Inc.","sells widgets, gadgets"' },
    })

    const rows = wrapper.findAll('tbody tr')
    const cells = rows[0].findAll('td')
    expect(cells[0].text()).toBe('Acme, Inc.')
    expect(cells[1].text()).toBe('sells widgets, gadgets')
  })

  it('handles quoted fields with escaped quotes', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'name,quote\nAlice,"She said ""hello"""' },
    })

    const cells = wrapper.findAll('tbody td')
    expect(cells[1].text()).toBe('She said "hello"')
  })

  it('handles multiline quoted fields', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'name,bio\nAlice,"Line 1\nLine 2"' },
    })

    const cells = wrapper.findAll('tbody td')
    expect(cells[1].html()).toContain('Line 1')
    expect(cells[1].html()).toContain('Line 2')
  })

  it('renders with proper row structure', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'col1,col2\nv1,v2\nv3,v4' },
    })

    expect(wrapper.findAll('thead tr')).toHaveLength(1)
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
  })

  it('handles single column CSV', () => {
    const wrapper = mount(CsvViewer, {
      props: { content: 'value\n1\n2\n3' },
    })

    const headers = wrapper.findAll('th')
    expect(headers).toHaveLength(1)
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
  })
})
