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
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const auth = useAuthStore()
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
