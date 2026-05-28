import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FilterBar from './FilterBar.vue'

describe('FilterBar', () => {
  it('renders media type and sort selects', () => {
    const wrapper = mount(FilterBar, {
      props: { mediaType: '', sortBy: 'taken_at' },
    })

    const selects = wrapper.findAll('select')
    expect(selects.length).toBe(2)
  })

  it('emits update:mediaType on media type change', async () => {
    const wrapper = mount(FilterBar, {
      props: { mediaType: '', sortBy: 'taken_at' },
    })

    const mediaSelect = wrapper.findAll('select')[0]
    await mediaSelect.setValue('video')

    expect(wrapper.emitted('update:mediaType')).toBeTruthy()
    expect(wrapper.emitted('update:mediaType')![0]).toEqual(['video'])
  })

  it('emits update:sortBy on sort change', async () => {
    const wrapper = mount(FilterBar, {
      props: { mediaType: '', sortBy: 'taken_at' },
    })

    const sortSelect = wrapper.findAll('select')[1]
    await sortSelect.setValue('created_at')

    expect(wrapper.emitted('update:sortBy')).toBeTruthy()
    expect(wrapper.emitted('update:sortBy')![0]).toEqual(['created_at'])
  })

  it('renders with initial values', () => {
    const wrapper = mount(FilterBar, {
      props: { mediaType: 'photo', sortBy: 'filename' },
    })

    expect(wrapper.props('mediaType')).toBe('photo')
    expect(wrapper.props('sortBy')).toBe('filename')
  })
})
