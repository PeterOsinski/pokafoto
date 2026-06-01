import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client'

export interface Album {
  id: string
  name: string
  description: string | null
  owner_id: string
  owner_name: string
  item_count: number
  is_shared: boolean
  is_owner: boolean
  share_permission: string
  shared_users: SharedUser[]
  created_at: string
  updated_at: string
}

export interface SharedUser {
  share_id: string
  user_id: string
  username: string
  permission: string
}

export const useAlbumStore = defineStore('albums', () => {
  const myAlbums = ref<Album[]>([])
  const sharedAlbums = ref<Album[]>([])
  const loading = ref(false)

  async function fetchAlbums() {
    loading.value = true
    try {
      const res = await api.get('/albums')
      myAlbums.value = res.data.myAlbums || []
      sharedAlbums.value = res.data.sharedAlbums || []
    } finally {
      loading.value = false
    }
  }

  async function createAlbum(name: string, description?: string) {
    const res = await api.post('/albums', { name, description })
    await fetchAlbums()
    return res.data
  }

  async function getAlbum(id: string): Promise<Album> {
    const res = await api.get(`/albums/${id}`)
    return res.data
  }

  async function updateAlbum(id: string, name: string, description?: string) {
    await api.put(`/albums/${id}`, { name, description })
    await fetchAlbums()
  }

  async function deleteAlbum(id: string) {
    await api.delete(`/albums/${id}`)
    await fetchAlbums()
  }

  async function shareAlbum(albumId: string, username: string, permission: string) {
    const res = await api.post(`/albums/${albumId}/shares`, { username, permission })
    return res.data
  }

  async function removeShare(albumId: string, shareId: string) {
    await api.delete(`/albums/${albumId}/shares/${shareId}`)
  }

  return {
    myAlbums,
    sharedAlbums,
    loading,
    fetchAlbums,
    createAlbum,
    getAlbum,
    updateAlbum,
    deleteAlbum,
    shareAlbum,
    removeShare,
  }
})
