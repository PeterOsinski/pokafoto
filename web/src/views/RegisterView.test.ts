import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import RegisterView from './RegisterView.vue'
import { useAuthStore } from '@/stores/auth'

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

function mountWithRouter(component: any) {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/login', name: 'login', component: { template: '<div>login</div>' } },
      { path: '/', name: 'home', component: { template: '<div>home</div>' } },
      { path: '/register', name: 'register', component: { template: '<div>register</div>' } },
    ],
  })
  return { router, wrapper: mount(component, { global: { plugins: [router] } }) }
}

describe('RegisterView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('renders registration form', () => {
    const { wrapper } = mountWithRouter(RegisterView)

    expect(wrapper.find('h1').text()).toBe('Create Account')
    expect(wrapper.findAll('input[type="text"]').length).toBe(2)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').text()).toBe('Create Account')
  })

  it('calls register then login on submit', async () => {
    const { wrapper } = mountWithRouter(RegisterView)
    const store = useAuthStore()
    const registerSpy = vi.spyOn(store, 'register').mockResolvedValue({})
    const loginSpy = vi.spyOn(store, 'login').mockResolvedValue({})

    await wrapper.find('input[placeholder="Username (3-32 chars)"]').setValue('newuser')
    await wrapper.find('input[type="password"]').setValue('password123')
    await wrapper.find('input[placeholder="Display Name (optional)"]').setValue('Test User')
    await wrapper.find('form').trigger('submit.prevent')

    await flushPromises()

    expect(registerSpy).toHaveBeenCalledWith('newuser', 'password123', 'Test User')
    expect(loginSpy).toHaveBeenCalledWith('newuser', 'password123')
  })

  it('handles registration failure', async () => {
    const { wrapper } = mountWithRouter(RegisterView)
    const store = useAuthStore()
    vi.spyOn(store, 'register').mockRejectedValue({
      response: { data: { error: { message: 'Username taken' } } },
    })

    await wrapper.find('input[placeholder="Username (3-32 chars)"]').setValue('existing')
    await wrapper.find('input[type="password"]').setValue('password123')
    await wrapper.find('form').trigger('submit.prevent')

    await flushPromises()

    expect(wrapper.text()).toContain('Username taken')
  })
})
