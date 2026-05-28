import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api/client'

export interface User {
  id: string
  username: string
  display_name?: string
  role: string
  created_at?: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const accessToken = ref<string | null>(localStorage.getItem('access_token'))
  const refreshToken = ref<string | null>(localStorage.getItem('refresh_token'))

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function setUser(u: User) { user.value = u }
  function setTokens(access: string, refresh: string) {
    accessToken.value = access
    refreshToken.value = refresh
    localStorage.setItem('access_token', access)
    localStorage.setItem('refresh_token', refresh)
  }

  async function login(username: string, password: string) {
    const res = await api.post('/auth/login', { username, password })
    setUser(res.data.user)
    setTokens(res.data.access_token, res.data.refresh_token)
    return res.data
  }

  async function register(username: string, password: string, displayName?: string) {
    const res = await api.post('/auth/register', { username, password, display_name: displayName })
    return res.data
  }

  async function fetchMe() {
    if (!accessToken.value) return
    try {
      const res = await api.get('/auth/me')
      setUser(res.data)
    } catch {
      logout()
    }
  }

  function logout() {
    user.value = null
    accessToken.value = null
    refreshToken.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  return { user, accessToken, refreshToken, isAuthenticated, isAdmin, setUser, setTokens, login, register, fetchMe, logout }
})
