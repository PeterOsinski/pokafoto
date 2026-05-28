import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FilterBar from './FilterBar.vue'

describe('FilterBar', () => {
  function mountFilter(props = {}) {
    return mount(FilterBar, {
      props: { mediaType: '', sortBy: 'taken_at', layout: 'tiles', thumbSize: 'md', ...props },
    })
  }

  it('renders sort select and icon button groups', () => {
    const wrapper = mountFilter()

    expect(wrapper.find('select').exists()).toBe(true)
    const buttonGroups = wrapper.findAll('.rounded-lg')
    expect(buttonGroups.length).toBe(3)
  })

  it('emits update:mediaType on media type button click', async () => {
    const wrapper = mountFilter()

    const buttonGroups = wrapper.findAll('.rounded-lg')
    const mediaButtons = buttonGroups[0].findAll('button')
    await mediaButtons[2].trigger('click')

    expect(wrapper.emitted('update:mediaType')).toBeTruthy()
    expect(wrapper.emitted('update:mediaType')![0]).toEqual(['video'])
  })

  it('emits update:sortBy on sort change', async () => {
    const wrapper = mountFilter()

    await wrapper.find('select').setValue('created_at')

    expect(wrapper.emitted('update:sortBy')).toBeTruthy()
    expect(wrapper.emitted('update:sortBy')![0]).toEqual(['created_at'])
  })

  it('emits update:layout on layout button click', async () => {
    const wrapper = mountFilter()

    const buttonGroups = wrapper.findAll('.rounded-lg')
    const layoutButtons = buttonGroups[1].findAll('button')
    await layoutButtons[1].trigger('click')

    expect(wrapper.emitted('update:layout')).toBeTruthy()
    expect(wrapper.emitted('update:layout')![0]).toEqual(['list'])
  })

  it('emits update:thumbSize on thumb size button click', async () => {
    const wrapper = mountFilter()

    const buttonGroups = wrapper.findAll('.rounded-lg')
    const sizeButtons = buttonGroups[2].findAll('button')
    await sizeButtons[2].trigger('click')

    expect(wrapper.emitted('update:thumbSize')).toBeTruthy()
    expect(wrapper.emitted('update:thumbSize')![0]).toEqual(['lg'])
  })

  it('applies active class to selected button', () => {
    const wrapper = mountFilter({ mediaType: 'photo', layout: 'list', thumbSize: 'sm' })

    const buttonGroups = wrapper.findAll('.rounded-lg')

    const mediaButtons = buttonGroups[0].findAll('button')
    expect(mediaButtons[1].classes()).toContain('bg-[var(--accent)]')

    const layoutButtons = buttonGroups[1].findAll('button')
    expect(layoutButtons[1].classes()).toContain('bg-[var(--accent)]')

    const sizeButtons = buttonGroups[2].findAll('button')
    expect(sizeButtons[0].classes()).toContain('bg-[var(--accent)]')
  })
})
