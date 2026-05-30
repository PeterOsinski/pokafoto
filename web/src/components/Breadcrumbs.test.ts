import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Breadcrumbs from './Breadcrumbs.vue'

describe('Breadcrumbs', () => {
  it('renders all segments', () => {
    const chain = [
      { id: null, name: 'Root' },
      { id: '1', name: 'Vacation' },
      { id: '2', name: 'Paris' },
    ]
    const wrapper = mount(Breadcrumbs, { props: { chain } })
    const nav = wrapper.find('nav')
    expect(nav.exists()).toBe(true)
    expect(nav.text()).toContain('Root')
    expect(nav.text()).toContain('Vacation')
    expect(nav.text()).toContain('Paris')
  })

  it('emits navigate when clicking a non-last segment', async () => {
    const chain = [
      { id: null, name: 'Root' },
      { id: '1', name: 'Vacation' },
      { id: '2', name: 'Paris' },
    ]
    const wrapper = mount(Breadcrumbs, { props: { chain } })
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    expect(wrapper.emitted('navigate')?.[0]).toEqual([null])
    await buttons[1].trigger('click')
    expect(wrapper.emitted('navigate')?.[1]).toEqual(['1'])
  })

  it('last segment is not clickable', () => {
    const chain = [
      { id: null, name: 'Root' },
      { id: '1', name: 'Vacation' },
    ]
    const wrapper = mount(Breadcrumbs, { props: { chain } })
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(1)
  })

  it('renders nothing for empty chain', () => {
    const wrapper = mount(Breadcrumbs, { props: { chain: [] } })
    expect(wrapper.find('nav').exists()).toBe(false)
  })
})
