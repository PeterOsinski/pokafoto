import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

vi.mock('@/api/client', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

function buildRouter(routes: RouteRecordRaw[] = []) {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      {
        path: '/login',
        name: 'login',
        component: { template: '<div>login</div>' },
      },
      {
        path: '/',
        name: 'home',
        component: { template: '<div>home</div>' },
        meta: { requiresAuth: true },
      },
      {
        path: '/admin',
        name: 'admin',
        component: { template: '<div>admin</div>' },
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      ...routes,
    ],
  })

  router.beforeEach((to, _from, next) => {
    const auth = useAuthStore()
    if (to.meta.requiresAuth && !auth.isAuthenticated) {
      next('/login')
    } else if (to.meta.requiresAdmin && auth.user?.role !== 'admin') {
      next('/')
    } else if ((to.path === '/login') && auth.isAuthenticated) {
      next('/')
    } else {
      next()
    }
  })

  return router
}

describe('Router guards', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('redirects to /login when not authenticated', async () => {
    const router = buildRouter()
    const store = useAuthStore()
    store.accessToken = null

    await router.push('/')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('allows navigation when authenticated', async () => {
    const router = buildRouter()
    const store = useAuthStore()
    store.accessToken = 'valid-token'
    store.user = { id: '1', username: 'user', role: 'member' }

    await router.push('/')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })

  it('redirects non-admin from /admin', async () => {
    const router = buildRouter()
    const store = useAuthStore()
    store.accessToken = 'valid-token'
    store.user = { id: '1', username: 'user', role: 'member' }

    await router.push('/admin')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })

  it('allows admin to access /admin', async () => {
    const router = buildRouter()
    const store = useAuthStore()
    store.accessToken = 'valid-token'
    store.user = { id: '1', username: 'admin', role: 'admin' }

    await router.push('/admin')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/admin')
  })

  it('redirects authenticated user from /login to /', async () => {
    const router = buildRouter()
    const store = useAuthStore()
    store.accessToken = 'valid-token'
    store.user = { id: '1', username: 'user', role: 'member' }

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })
})
