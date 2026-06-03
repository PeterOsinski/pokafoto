import { reactive, computed, ref } from 'vue'
import api from '../api/client'
import type { FolderEntry } from '../types/gallery'
import type { ContextMenuItem } from '../components/ContextMenu.vue'

export function useFolderDialogs() {
  const renameDialog = reactive({
    show: false,
    folderId: '',
    currentName: '',
    type: 'folder' as 'folder' | 'file',
  })

  const deleteFolderConfirm = reactive({
    show: false,
    folderId: '',
    folderName: '',
    loading: false,
    result: null as { deleted_files: number; deleted_folders: number } | null,
  })

  const moveFolderDialog = reactive({ show: false, folderId: '', folderName: '' })
  const passwordDialog = reactive({
    show: false,
    folderId: '',
    mode: 'set' as 'set' | 'unlock' | 'status',
    passwordHint: '',
  })
  const shareDialog = reactive({ show: false, folderId: '', folderName: '' })
  const contextMenu = reactive({ visible: false, x: 0, y: 0, items: [] as ContextMenuItem[] })

  // Use refs for these — template auto-unwraps top-level refs but not nested property access
  const showDeleteFilesConfirm = ref(false)
  const moveDialogOpen = ref(false)
  const copyDialogOpen = ref(false)

  const selectedIds = ref(new Set<string>())

  const deleteMessage = computed(() => {
    const n = selectedIds.value.size
    return `This will move ${n} ${n === 1 ? 'file' : 'files'} to trash. You can recover them later.`
  })

  function openRenameDialog(type: 'folder' | 'file', id: string, currentName: string) {
    Object.assign(renameDialog, { show: true, folderId: id, currentName, type })
  }

  async function executeRename() {
    if (!renameDialog.currentName.trim()) return
    try {
      if (renameDialog.type === 'folder') {
        await api.put(`/folders/${renameDialog.folderId}`, { name: renameDialog.currentName.trim() })
      } else {
        await api.put(`/files/${renameDialog.folderId}/rename`, { name: renameDialog.currentName.trim() })
      }
      renameDialog.show = false
      return true
    } catch (e) {
      console.error('Failed to rename', e)
      return false
    }
  }

  function openDeleteFolderConfirm(folderId: string, folderName: string) {
    Object.assign(deleteFolderConfirm, { show: true, folderId, folderName, loading: false, result: null })
  }

  async function executeFolderDelete() {
    deleteFolderConfirm.loading = true
    try {
      const res = await api.delete(`/folders/${deleteFolderConfirm.folderId}`)
      deleteFolderConfirm.result = res.data
      return true
    } catch (e) {
      console.error('Failed to delete folder', e)
      return false
    } finally {
      deleteFolderConfirm.loading = false
    }
  }

  function openMoveFolderDialog(folderId: string, folderName: string) {
    Object.assign(moveFolderDialog, { show: true, folderId, folderName })
  }

  async function executeFolderMove(targetFolderId: string | null) {
    try {
      await api.put(`/folders/${moveFolderDialog.folderId}`, { parent_id: targetFolderId || '' })
      moveFolderDialog.show = false
      return true
    } catch (e) {
      console.error('Failed to move folder', e)
      return false
    }
  }

  function openPasswordDialog(folderId: string, mode: 'set' | 'unlock' | 'status', passwordHint = '') {
    Object.assign(passwordDialog, { show: true, folderId, mode, passwordHint })
  }

  function openShareDialog(folderId: string, folderName: string) {
    Object.assign(shareDialog, { show: true, folderId, folderName })
  }

  function openFolderContextMenu(e: MouseEvent, _folder: FolderEntry, passStatus: boolean) {
    Object.assign(contextMenu, {
      visible: true,
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: passStatus ? 'Password...' : 'Set Password', icon: passStatus ? '&#x1F512;' : '&#x1F513;', action: () => {} },
        { label: 'Share', icon: '&#x1F517;', action: () => {} },
        { label: 'Rename', icon: '&#x270F;', action: () => {} },
        { label: 'Move', icon: '&#x2194;', action: () => {} },
        { label: 'Delete', icon: '&#x1F5D1;', danger: true, action: () => {} },
      ],
    })
  }

  function openFileContextMenu(e: MouseEvent, _fileId: string, _fileName: string) {
    Object.assign(contextMenu, {
      visible: true,
      x: e.clientX,
      y: e.clientY,
      items: [{ label: 'Rename', icon: '&#x270F;', action: () => {} }],
    })
  }

  async function executeDeleteFiles() {
    try {
      const ids = Array.from(selectedIds.value)
      await api.post('/files/batch-delete', { ids })
      showDeleteFilesConfirm.value = false
      return true
    } catch (e) {
      console.error('Failed to delete files', e)
      return false
    }
  }

  async function executeMoveFiles(targetFolderId: string | null) {
    try {
      await api.post('/files/batch-move', { ids: Array.from(selectedIds.value), folder_id: targetFolderId || null })
      moveDialogOpen.value = false
      return true
    } catch (e) {
      console.error('Failed to move files', e)
      return false
    }
  }

  async function executeCopyFiles(targetFolderId: string | null) {
    try {
      await api.post('/files/batch-copy', { ids: Array.from(selectedIds.value), folder_id: targetFolderId || null })
      copyDialogOpen.value = false
      return true
    } catch (e) {
      console.error('Failed to copy files', e)
      return false
    }
  }

  return {
    renameDialog, openRenameDialog, executeRename,
    deleteFolderConfirm, openDeleteFolderConfirm, executeFolderDelete,
    moveFolderDialog, openMoveFolderDialog, executeFolderMove,
    passwordDialog, openPasswordDialog,
    shareDialog, openShareDialog,
    contextMenu, openFolderContextMenu, openFileContextMenu,
    showDeleteFilesConfirm, deleteMessage, executeDeleteFiles,
    moveDialogOpen, executeMoveFiles,
    copyDialogOpen, executeCopyFiles,
    selectedIds,
  }
}
