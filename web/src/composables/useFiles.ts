import { ref, type Ref, type ComputedRef, onMounted, onUnmounted } from 'vue'
import api from '../api/client'
import type { FileItem, SortBy } from '../types/gallery'

export function useFiles(folderId: ComputedRef<string | null>, sortBy: ComputedRef<SortBy>) {
  const files = ref<FileItem[]>([])
  const nextCursor = ref('')
  const loading = ref(false)
  const loadingMore = ref(false)

  async function loadFiles(reset = true) {
    if (reset) {
      files.value = []
      nextCursor.value = ''
      loading.value = true
    } else {
      loadingMore.value = true
    }
    try {
      const params: any = { sort: sortBy.value, order: 'desc', limit: 500 }
      if (folderId.value) {
        params.folder_id = folderId.value
      }
      if (nextCursor.value) params.cursor = nextCursor.value
      const res = await api.get('/files', { params })
      files.value = reset ? res.data.items : [...files.value, ...res.data.items]
      nextCursor.value = res.data.nextCursor || ''
    } catch (e) {
      console.error('Failed to load files', e)
    } finally {
      loading.value = false
      loadingMore.value = false
    }
  }

  function observeInfiniteScroll(sentinelRef: Ref<HTMLElement | null>) {
    let observer: IntersectionObserver | null = null
    onMounted(() => {
      if (sentinelRef.value) {
        observer = new IntersectionObserver(
          (entries) => {
            if (entries[0]?.isIntersecting && nextCursor.value && !loadingMore.value) {
              loadFiles(false)
            }
          },
          { rootMargin: '200px' },
        )
        observer.observe(sentinelRef.value)
      }
    })
    onUnmounted(() => {
      observer?.disconnect()
    })
  }

  return { files, nextCursor, loading, loadingMore, loadFiles, observeInfiniteScroll }
}
