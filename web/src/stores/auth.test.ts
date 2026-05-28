import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'
import type { AxiosResponse } from 'axios'

function mockResponse<T>(data: T): AxiosResponse<T> {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config: {} as any,
  }
}

vi.mock('../api/client', () => {
  const api = {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  }
  return { default: api }
})

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  describe('login', () => {
    it('sets user and tokens on success', async () => {
      const store = useAuthStore()
      const mockData = {
        access_token: 'access-123',
        refresh_token: 'refresh-456',
        expires_in: 259200,
        user: { id: 'user-1', username: 'testuser', role: 'member' },
      }

      const api = (await import('../api/client')).default as any
      api.post.mockResolvedValue(mockResponse(mockData))

      await store.login('testuser', 'password123')

      expect(api.post).toHaveBeenCalledWith('/auth/login', {
        username: 'testuser',
        password: 'password123',
      })
      expect(store.accessToken).toBe('access-123')
      expect(store.refreshToken).toBe('refresh-456')
      expect(store.user).toEqual(mockData.user)
      expect(localStorage.getItem('access_token')).toBe('access-123')
      expect(localStorage.getItem('refresh_token')).toBe('refresh-456')
    })

    it('throws on login failure without clearing existing state', async () => {
      const store = useAuthStore()
      store.accessToken = 'old-token'
      store.user = { id: 'x', username: 'x', role: 'member' }

      const api = (await import('../api/client')).default as any
      api.post.mockRejectedValue(new Error('Network error'))

      await expect(store.login('user', 'pass')).rejects.toThrow()
      expect(store.user).not.toBeNull()
      expect(store.accessToken).toBe('old-token')
    })
  })

  describe('register', () => {
    it('calls POST /auth/register', async () => {
      const store = useAuthStore()

      const api = (await import('../api/client')).default as any
      api.post.mockResolvedValue(mockResponse({
        user: { id: 'u2', username: 'new', role: 'member' },
      }))

      const result = await store.register('new', 'password123')

      expect(api.post).toHaveBeenCalledWith('/auth/register', {
        username: 'new',
        password: 'password123',
      })
      expect(result.user.username).toBe('new')
    })
  })

  describe('fetchMe', () => {
    it('populates user on success', async () => {
      localStorage.setItem('access_token', 'valid-token')
      setActivePinia(createPinia())

      const store = useAuthStore()
      const user = { id: 'me-1', username: 'me', role: 'member' }

      const api = (await import('../api/client')).default as any
      api.get.mockResolvedValue(mockResponse(user))

      await store.fetchMe()

      expect(store.user).toEqual(user)
    })
  })

  describe('logout', () => {
    it('clears user and tokens', () => {
      const store = useAuthStore()
      store.accessToken = 'token'
      store.refreshToken = 'refresh'
      store.user = { id: 'x', username: 'x', role: 'member' }
      localStorage.setItem('access_token', 'token')

      store.logout()

      expect(store.accessToken).toBeNull()
      expect(store.refreshToken).toBeNull()
      expect(store.user).toBeNull()
      expect(localStorage.getItem('access_token')).toBeNull()
    })
  })

  describe('isAuthenticated', () => {
    it('returns true when accessToken exists', () => {
      const store = useAuthStore()
      store.accessToken = 'token'
      expect(store.isAuthenticated).toBe(true)
    })

    it('returns false when accessToken is null', () => {
      const store = useAuthStore()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('isAdmin', () => {
    it('returns true when user role is admin', () => {
      const store = useAuthStore()
      store.user = { id: 'a', username: 'admin', role: 'admin' }
      expect(store.isAdmin).toBe(true)
    })

    it('returns false for member role', () => {
      const store = useAuthStore()
      store.user = { id: 'm', username: 'member', role: 'member' }
      expect(store.isAdmin).toBe(false)
    })

    it('returns false when no user', () => {
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })
  })
})
