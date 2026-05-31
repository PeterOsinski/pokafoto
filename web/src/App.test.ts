import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'

vi.mock('@/api/client', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))
vi.mock('axios', () => ({
  default: {
    create: () => ({
      interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
    }),
  },
}))

let mockWs: { url: string; onopen: (() => void) | null; close: () => void } | null = null

describe('App', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
    mockWs = { url: '', onopen: null, close: vi.fn() }
    vi.stubGlobal('WebSocket', class MockWS {
      url: string
      onopen: (() => void) | null = null
      onmessage: ((e: MessageEvent) => void) | null = null
      onerror: ((e: Event) => void) | null = null
      onclose: ((e: CloseEvent) => void) | null = null
      constructor(url: string) {
        this.url = url
        mockWs!.url = url
      }
      close() { mockWs!.close() }
      send() {}
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  function mountApp() {
    const router = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/', component: { template: '<div>Gallery</div>' } },
        { path: '/login', component: { template: '<div>Login</div>' } },
        { path: '/folders', component: { template: '<div>Folders</div>' } },
        { path: '/timeline', component: { template: '<div>Timeline</div>' } },
        { path: '/map', component: { template: '<div>Map</div>' } },
        { path: '/admin', component: { template: '<div>Admin</div>' } },
      ],
    })
    return mount(App, { global: { plugins: [router] } })
  }

  it('renders quota progress bar when space_quota is set', async () => {
    localStorage.setItem('access_token', 'test-token')
    localStorage.setItem('refresh_token', 'test-refresh')
    const { default: api } = await import('@/api/client')
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/auth/me') return Promise.resolve({ data: { id: 'u1', username: 'test', role: 'member', space_quota: 10737418240 } })
      if (url === '/stats') return Promise.resolve({ data: { total_size_bytes: 1610612736 } })
      if (url === '/health') return Promise.resolve({ data: {} })
      return Promise.resolve({ data: {} })
    })

    const wrapper = mountApp()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('1.5 GB / 10.0 GB')
  })

  it('does not render quota progress bar when space_quota is null', async () => {
    localStorage.setItem('access_token', 'test-token')
    localStorage.setItem('refresh_token', 'test-refresh')
    const { default: api } = await import('@/api/client')
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/auth/me') return Promise.resolve({ data: { id: 'u1', username: 'test', role: 'member', space_quota: null } })
      if (url === '/stats') return Promise.resolve({ data: { total_size_bytes: 1610612736 } })
      if (url === '/health') return Promise.resolve({ data: {} })
      return Promise.resolve({ data: {} })
    })

    const wrapper = mountApp()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).not.toContain('GB /')
  })

  it('displays formatted usage vs quota text', async () => {
    localStorage.setItem('access_token', 'test-token')
    localStorage.setItem('refresh_token', 'test-refresh')
    const { default: api } = await import('@/api/client')
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/auth/me') return Promise.resolve({ data: { id: 'u1', username: 'test', role: 'member', space_quota: 1024 } })
      if (url === '/stats') return Promise.resolve({ data: { total_size_bytes: 512 } })
      if (url === '/health') return Promise.resolve({ data: {} })
      return Promise.resolve({ data: {} })
    })

    const wrapper = mountApp()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('512 B / 1.0 KB')
  })
})
