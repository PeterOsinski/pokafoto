import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.accessToken) {
    config.headers.Authorization = `Bearer ${auth.accessToken}`
  }

  const folderId = extractFolderId(config)
  if (folderId) {
    try {
      const stored = localStorage.getItem('folderUnlocks')
      if (stored) {
        const entries = JSON.parse(stored)
        const entry = entries.find((e: any) => e.folderId === folderId)
        if (entry && new Date(entry.expiresAt) > new Date()) {
          config.headers['X-Folder-Unlock-Token'] = entry.token
        }
      }
    } catch {}
  }

  return config
})

function extractFolderId(config: any): string | null {
  if (config.folderId) return config.folderId
  const url: string = config.url || ''
  const folderMatch = url.match(/\/folders\/([^/]+)/)
  if (folderMatch) return folderMatch[1]
  if (config.params?.folder_id) return config.params.folder_id
  if (config.data?.folder_id) return config.data.folder_id
  if (config.method === 'get' || config.method === 'post') {
    const paramMatch = url.match(/[?&]folder_id=([^&]+)/)
    if (paramMatch) return paramMatch[1]
  }
  return null
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const auth = useAuthStore()
    if (error.response?.status === 403 &&
        error.response?.data?.error?.code === 'FOLDER_PASSWORD_REQUIRED') {
      const folderId = extractFolderId(error.config)
      window.dispatchEvent(new CustomEvent('folder-password-required', {
        detail: { folderId: folderId || '' },
      }))
      return Promise.reject(error)
    }
    if (error.response?.status === 401 && auth.refreshToken) {
      try {
        const res = await axios.post('/api/v1/auth/refresh', {
          refresh_token: auth.refreshToken,
        })
        auth.setTokens(res.data.access_token, res.data.refresh_token)
        error.config.headers.Authorization = `Bearer ${res.data.access_token}`
        return api(error.config)
      } catch {
        auth.logout()
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)

export default api
