import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface UnlockEntry {
  folderId: string
  token: string
  expiresAt: string
}

export const useFolderUnlockStore = defineStore('folderUnlock', () => {
  const unlockedFolders = ref<UnlockEntry[]>(
    JSON.parse(localStorage.getItem('folderUnlocks') || '[]')
  )

  const isUnlocked = computed(() => (folderId: string) => {
    const entry = unlockedFolders.value.find(e => e.folderId === folderId)
    if (!entry) return false
    return new Date(entry.expiresAt) > new Date()
  })

  const getToken = (folderId: string): string | null => {
    const entry = unlockedFolders.value.find(e => e.folderId === folderId)
    if (!entry || new Date(entry.expiresAt) <= new Date()) return null
    return entry.token
  }

  const getTimeLeft = (folderId: string): number => {
    const entry = unlockedFolders.value.find(e => e.folderId === folderId)
    if (!entry) return 0
    return Math.max(0, new Date(entry.expiresAt).getTime() - Date.now())
  }

  const setToken = (folderId: string, token: string, expiresAt: string) => {
    unlockedFolders.value = [
      ...unlockedFolders.value.filter(e => e.folderId !== folderId),
      { folderId, token, expiresAt },
    ]
    localStorage.setItem('folderUnlocks', JSON.stringify(unlockedFolders.value))
  }

  const removeToken = (folderId: string) => {
    unlockedFolders.value = unlockedFolders.value.filter(e => e.folderId !== folderId)
    localStorage.setItem('folderUnlocks', JSON.stringify(unlockedFolders.value))
  }

  return { unlockedFolders, isUnlocked, getToken, getTimeLeft, setToken, removeToken }
})
