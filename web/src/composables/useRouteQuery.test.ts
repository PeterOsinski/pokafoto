import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createRouter, createWebHistory } from 'vue-router'
import { useRouteQuery } from './useRouteQuery'
import { mount } from '@vue/test-utils'
import { h, defineComponent } from 'vue'

function createTestRouter(query: Record<string, string> = {}) {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      {
        path: '/',
        name: 'home',
        component: { template: '<div></div>' },
      },
    ],
  })
  router.replace({ query })
  return router
}

describe('useRouteQuery', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns default value when no query param is set', async () => {
    const router = createTestRouter({})
    await router.isReady()

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('folder_id', '')
          return () => h('div', { 'data-test': 'out' }, q.value || '(empty)')
        },
      }),
      { global: { plugins: [router] } },
    )

    expect(wrapper.text()).toBe('(empty)')
  })

  it('reads existing query param from the URL', async () => {
    const router = createTestRouter({ folder_id: 'abc-123' })
    await router.isReady()

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('folder_id', '')
          return () => h('div', q.value)
        },
      }),
      { global: { plugins: [router] } },
    )

    expect(wrapper.text()).toBe('abc-123')
  })

  it('writes value to URL via router.replace', async () => {
    const router = createTestRouter({})
    await router.isReady()
    const replaceSpy = vi.spyOn(router, 'replace')

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('folder_id', '')
          q.value = 'new-folder'
          return () => h('div', q.value)
        },
      }),
      { global: { plugins: [router] } },
    )

    await wrapper.vm.$nextTick()

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        query: { folder_id: 'new-folder' },
      }),
    )
  })

  it('removes the query param when set to empty string', async () => {
    const router = createTestRouter({ folder_id: 'abc-123' })
    await router.isReady()
    const replaceSpy = vi.spyOn(router, 'replace')

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('folder_id', '')
          q.value = null
          return () => h('div', q.value)
        },
      }),
      { global: { plugins: [router] } },
    )

    await wrapper.vm.$nextTick()

    const call = replaceSpy.mock.calls[0]?.[0] as { query?: Record<string, string> } | undefined
    expect(call?.query?.folder_id).toBeUndefined()
  })

  it('removes the query param when set to the default value', async () => {
    const router = createTestRouter({ layout: 'tiles' })
    await router.isReady()
    const replaceSpy = vi.spyOn(router, 'replace')

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('layout', 'grid')
          q.value = 'grid'
          return () => h('div', q.value)
        },
      }),
      { global: { plugins: [router] } },
    )

    await wrapper.vm.$nextTick()

    const call = replaceSpy.mock.calls[0]?.[0] as { query?: Record<string, string> } | undefined
    expect(call?.query?.layout).toBeUndefined()
  })

  it('removes the query param when set to null', async () => {
    const router = createTestRouter({ photo: 'file-1' })
    await router.isReady()
    const replaceSpy = vi.spyOn(router, 'replace')

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('photo', '')
          q.value = null
          return () => h('div', q.value)
        },
      }),
      { global: { plugins: [router] } },
    )

    await wrapper.vm.$nextTick()

    const call = replaceSpy.mock.calls[0]?.[0] as { query?: Record<string, string> } | undefined
    expect(call?.query?.photo).toBeUndefined()
  })

  it('returns new value synchronously after set, before route updates', async () => {
    const router = createTestRouter({})
    await router.isReady()

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('folder_id', '')
          const clicked = () => { q.value = 'immediate-folder' }
          return () => h('div', { onClick: clicked }, q.value || '(empty)')
        },
      }),
      { global: { plugins: [router] } },
    )

    await wrapper.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toBe('immediate-folder')
  })

  it('preserves other query params when updating one', async () => {
    const router = createTestRouter({ folder_id: 'abc-123', sort: 'taken_at' })
    await router.isReady()
    const replaceSpy = vi.spyOn(router, 'replace')

    const wrapper = mount(
      defineComponent({
        setup() {
          const q = useRouteQuery('layout', 'tiles')
          q.value = 'list'
          return () => h('div', q.value)
        },
      }),
      { global: { plugins: [router] } },
    )

    await wrapper.vm.$nextTick()

    const call = replaceSpy.mock.calls[0]?.[0] as { query?: Record<string, string> } | undefined
    expect(call?.query?.folder_id).toBe('abc-123')
    expect(call?.query?.sort).toBe('taken_at')
    expect(call?.query?.layout).toBe('list')
  })
})
