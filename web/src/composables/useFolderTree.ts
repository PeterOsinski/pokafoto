import { ref, computed } from 'vue'
import api from '../api/client'
import { useRouteQuery } from './useRouteQuery'
import { useFolderUnlockStore } from '../stores/folderUnlock'
import type { RootNode, FolderTreeNode } from '../types/gallery'

export function useFolderTree() {
  const folderIdQuery = useRouteQuery('folder_id', '')
  const currentFolderId = computed(() => folderIdQuery.value || null)

  const folders = ref<RootNode>({ children: [] })
  const loading = ref(false)
  const passwordStatuses = ref<Record<string, boolean>>({})
  const passwordHints = ref<Record<string, string>>({})
  const unlockStore = useFolderUnlockStore()

  const subfolders = computed(() => {
    if (!currentFolderId.value) return []
    const find = (nodes: FolderTreeNode[]): FolderTreeNode[] => {
      for (const n of nodes) {
        if (n.folder.id === currentFolderId.value) return n.children ?? []
        const found = find(n.children ?? [])
        if (found.length) return found
      }
      return []
    }
    return find(folders.value.children ?? [])
  })

  const listFolders = computed(() => {
    if (currentFolderId.value) return subfolders.value
    return folders.value.children ?? []
  })

  const folderTileTargets = computed(() => {
    if (currentFolderId.value) return subfolders.value
    return folders.value.children ?? []
  })

  const folderHasShares = computed(() => {
    const map: Record<string, boolean> = {}
    const collect = (nodes: FolderTreeNode[]) => {
      for (const n of nodes) {
        map[n.folder.id] = n.hasShares
        collect(n.children ?? [])
      }
    }
    collect(folders.value.children ?? [])
    return map
  })

  const folderChain = computed(() => {
    const chain: { id: string | null; name: string }[] = [{ id: null, name: 'Root' }]
    if (!currentFolderId.value) return chain

    const buildPath = (nodes: FolderTreeNode[], target: string, path: { id: string | null; name: string }[]): boolean => {
      for (const n of nodes) {
        if (n.folder.id === target) {
          path.push({ id: n.folder.id, name: n.folder.name })
          return true
        }
        if (n.children?.length) {
          path.push({ id: n.folder.id, name: n.folder.name })
          if (buildPath(n.children, target, path)) return true
          path.pop()
        }
      }
      return false
    }
    buildPath(folders.value.children ?? [], currentFolderId.value, chain)
    return chain
  })

  function findParentId(targetId: string, nodes: FolderTreeNode[]): string | null {
    for (const n of nodes) {
      if (n.children?.some(c => c.folder.id === targetId)) return n.folder.id
      for (const c of n.children ?? []) {
        const found = findParentId(targetId, [c])
        if (found !== null) return found
      }
    }
    return null
  }

  function navigateTo(id: string | null) {
    folderIdQuery.value = id ?? null
  }

  function navigateUp() {
    if (!currentFolderId.value) return
    const parentId = findParentId(currentFolderId.value, folders.value.children ?? [])
    folderIdQuery.value = parentId ?? null
  }

  async function loadFolders() {
    loading.value = true
    try {
      const res = await api.get('/folders')
      folders.value = res.data
    } catch (e) {
      console.error('Failed to load folders', e)
    } finally {
      loading.value = false
    }
  }

  async function createFolder(name: string) {
    try {
      await api.post('/folders', {
        name,
        parent_id: currentFolderId.value,
      })
    } catch (e) {
      console.error('Failed to create folder', e)
    }
  }

  async function loadPasswordStatuses() {
    try {
      const allIds: string[] = []
      const collect = (nodes: FolderTreeNode[]) => {
        for (const n of nodes) {
          allIds.push(n.folder.id)
          collect(n.children ?? [])
        }
      }
      collect(folders.value.children ?? [])

      for (const id of allIds) {
        try {
          const res = await api.get(`/folders/${id}/password`)
          passwordStatuses.value[id] = res.data.has_password || false
          passwordHints.value[id] = res.data.password_hint || ''
        } catch {
          passwordStatuses.value[id] = false
        }
      }
    } catch {}
  }

  function openPasswordDialog(folderId: string): { show: true; folderId: string; mode: 'set' | 'unlock' | 'status'; passwordHint: string } {
    const hasPass = passwordStatuses.value[folderId]
    const unlocked = unlockStore.isUnlocked(folderId)
    return {
      show: true,
      folderId,
      mode: hasPass ? (unlocked ? 'status' : 'unlock') : 'set',
      passwordHint: passwordHints.value[folderId] || '',
    }
  }

  return {
    folders, loading, currentFolderId,
    subfolders, listFolders, folderTileTargets, folderHasShares, folderChain,
    passwordStatuses, passwordHints,
    loadFolders, createFolder, navigateTo, navigateUp,
    loadPasswordStatuses, openPasswordDialog,
  }
}
