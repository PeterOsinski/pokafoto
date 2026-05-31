import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import LoginView from './LoginView.vue'
import { useAuthStore } from '@/stores/auth'

vi.mock('@/api/client', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn().mockResolvedValue({ data: { allow_registration: true } }),
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

describe('LoginView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('renders login form', () => {
    const { wrapper } = mountWithRouter(LoginView)

    expect(wrapper.find('h1').text()).toBe('Drive')
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').text()).toBe('Log In')
  })

  it('has register link', async () => {
    const { wrapper } = mountWithRouter(LoginView)
    await flushPromises()

    const link = wrapper.findComponent({ name: 'RouterLink' })
    expect(link.exists()).toBe(true)
    expect(link.text()).toContain('Register')
    expect(link.props('to')).toBe('/register')
  })

  it('calls login on submit', async () => {
    const { wrapper } = mountWithRouter(LoginView)
    const store = useAuthStore()
    const loginSpy = vi.spyOn(store, 'login').mockResolvedValue({})

    await wrapper.find('input[type="text"]').setValue('testuser')
    await wrapper.find('input[type="password"]').setValue('password123')
    await wrapper.find('form').trigger('submit.prevent')

    await flushPromises()

    expect(loginSpy).toHaveBeenCalledWith('testuser', 'password123')
  })

  it('shows error message on failed login', async () => {
    const { wrapper } = mountWithRouter(LoginView)
    const store = useAuthStore()
    vi.spyOn(store, 'login').mockRejectedValue({
      response: { data: { error: { message: 'Bad credentials' } } },
    })

    await wrapper.find('input[type="text"]').setValue('user')
    await wrapper.find('input[type="password"]').setValue('bad')
    await wrapper.find('form').trigger('submit.prevent')

    await flushPromises()

    expect(wrapper.text()).toContain('Bad credentials')
  })
})
