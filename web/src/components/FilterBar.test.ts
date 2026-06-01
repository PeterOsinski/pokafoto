import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FilterBar from './FilterBar.vue'

describe('FilterBar', () => {
  function mountFilter(props = {}) {
    return mount(FilterBar, {
      props: { mediaType: '', sortBy: 'taken_at', layout: 'tiles', thumbLevel: 5, includeAllFolders: false, previewMode: 'lightbox', ...props },
    })
  }

  it('renders sort select, icon button groups, and slider', () => {
    const wrapper = mountFilter()

    expect(wrapper.find('select').exists()).toBe(true)
    const buttonGroups = wrapper.findAll('.rounded-lg')
    expect(buttonGroups.length).toBe(2)
    expect(wrapper.find('input[type="range"]').exists()).toBe(true)
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

  it('emits update:thumbLevel on slider input', async () => {
    const wrapper = mountFilter()

    const slider = wrapper.find('input[type="range"]')
    await slider.setValue(3)

    expect(wrapper.emitted('update:thumbLevel')).toBeTruthy()
    expect(wrapper.emitted('update:thumbLevel')![0]).toEqual([3])
  })

  it('slider has min=0, max=9 and renders S/L labels', () => {
    const wrapper = mountFilter()

    const slider = wrapper.find('input[type="range"]')
    expect(slider.attributes('min')).toBe('0')
    expect(slider.attributes('max')).toBe('9')
    expect(wrapper.text()).toContain('S')
    expect(wrapper.text()).toContain('L')
  })

  it('applies active class to selected media and layout buttons', () => {
    const wrapper = mountFilter({ mediaType: 'photo', layout: 'list' })

    const buttonGroups = wrapper.findAll('.rounded-lg')

    const mediaButtons = buttonGroups[0].findAll('button')
    expect(mediaButtons[1].classes()).toContain('bg-[var(--accent)]')

    const layoutButtons = buttonGroups[1].findAll('button')
    expect(layoutButtons[1].classes()).toContain('bg-[var(--accent)]')
  })
})
