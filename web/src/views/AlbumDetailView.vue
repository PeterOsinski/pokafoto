<template>
  <div class="max-w-6xl mx-auto p-4">
    <div class="flex items-center gap-3 mb-4">
      <button @click="$router.push('/albums')" class="text-gray-400 hover:text-gray-200 transition-colors">&larr; Back</button>
      <h2 class="text-xl font-semibold text-gray-200">{{ album?.name }}</h2>
      <span v-if="album?.owner_name" class="text-xs text-gray-500">by {{ album.owner_name }}</span>
    </div>

    <div v-if="loading" class="text-gray-400 text-sm">Loading...</div>

    <template v-else-if="album">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-400">{{ album?.item_count }} {{ album.item_count === 1 ? 'photo' : 'photos' }}</span>
          <button v-if="canAdd" @click="openFilePicker" class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors">
            + Add Photos
          </button>
          <button v-if="album.is_owner" @click="showShare = true" class="px-3 py-1.5 bg-green-600 hover:bg-green-700 text-white text-sm rounded-lg transition-colors">
            Share
          </button>
        </div>
        <button v-if="album.is_owner" @click="deleteAlbum" class="px-3 py-1.5 bg-red-600/20 hover:bg-red-600/40 text-red-400 text-sm rounded-lg transition-colors">
          Delete
        </button>
      </div>

      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
        <button
          v-for="item in items"
          :key="item.id"
          @click="openFile(item)"
          class="relative aspect-square bg-gray-800 rounded-lg border border-gray-700 hover:border-gray-500 overflow-hidden transition-colors"
        >
          <img v-if="item.thumbnails?.sm" :src="item.thumbnails.sm.url" class="w-full h-full object-cover" alt="" loading="lazy" />
          <div v-else class="w-full h-full flex items-center justify-center text-gray-600">
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
          </div>
        </button>
      </div>

      <div v-if="!items.length" class="text-center py-12 text-gray-500">
        <p>No photos in this album yet.</p>
      </div>
    </template>

    <div v-if="showFilePicker" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="showFilePicker = false">
      <div class="bg-gray-800 rounded-xl border border-gray-700 p-6 w-full max-w-lg max-h-[80vh] flex flex-col" @click.stop>
        <h3 class="text-lg font-medium text-gray-200 mb-4">Add Photos to Album</h3>
        <div class="flex-1 overflow-y-auto space-y-2 max-h-96">
          <label v-for="f in myFiles" :key="f.id" class="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-750 cursor-pointer">
            <input type="checkbox" :value="f.id" v-model="selectedFiles" class="accent-blue-500" />
            <img v-if="f.thumbnails?.sm" :src="f.thumbnails.sm.url" class="w-10 h-10 object-cover rounded" alt="" />
            <span class="text-sm text-gray-300 truncate flex-1">{{ f.originalName }}</span>
          </label>
          <div v-if="!myFiles.length && !filePickerLoading" class="text-sm text-gray-500 py-4 text-center">No files to add</div>
          <div v-if="filePickerLoading" class="text-sm text-gray-500 py-4 text-center">Loading...</div>
        </div>
        <div class="flex gap-2 mt-4 pt-3 border-t border-gray-700">
          <button @click="addSelected" :disabled="!selectedFiles.length" class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm rounded-lg transition-colors">
            Add {{ selectedFiles.length ? `(${selectedFiles.length})` : '' }}
          </button>
          <button @click="showFilePicker = false" class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 text-sm rounded-lg transition-colors">
            Cancel
          </button>
        </div>
      </div>
    </div>

    <Lightbox
      v-if="previewFile"
      :file="previewFile"
      :index="previewIndex"
      :total="items.length"
      :has-prev="previewIndex > 0"
      :has-next="previewIndex < items.length - 1"
      @close="previewFile = null"
      @next="onNavigate(1)"
      @prev="onNavigate(-1)"
    />

    <ShareDialog
      :visible="showShare"
      :shares="album?.shared_users || []"
      :error="shareError"
      @close="showShare = false"
      @share="doShare"
      @remove-share="doRemoveShare"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api/client'
import { useAlbumStore, type Album } from '../stores/albums'
import ShareDialog from '../components/ShareDialog.vue'
import Lightbox from '../components/Lightbox.vue'

const route = useRoute()
const router = useRouter()
const albumStore = useAlbumStore()

const album = ref<Album | null>(null)
const items = ref<any[]>([])
const loading = ref(true)
const showFilePicker = ref(false)
const selectedFiles = ref<string[]>([])
const myFiles = ref<any[]>([])
const filePickerLoading = ref(false)
const showShare = ref(false)
const shareError = ref('')
const previewFile = ref<any>(null)
const previewIndex = ref(0)

const albumId = computed(() => route.params.id as string)
const canAdd = computed(() => album.value?.is_owner || album.value?.share_permission === 'edit')

onMounted(async () => {
  await loadAlbum()
})

async function loadAlbum() {
  loading.value = true
  try {
    const [albumRes, itemsRes] = await Promise.all([
      albumStore.getAlbum(albumId.value),
      api.get(`/albums/${albumId.value}/items`),
    ])
    album.value = albumRes
    items.value = itemsRes.data.items || []
  } catch {
    router.push('/albums')
  } finally {
    loading.value = false
  }
}

async function deleteAlbum() {
  await albumStore.deleteAlbum(albumId.value)
  router.push('/albums')
}

async function openFilePicker() {
  showFilePicker.value = true
  filePickerLoading.value = true
  try {
    const res = await api.get('/files', { params: { limit: 200 } })
    myFiles.value = res.data.items || []
  } finally {
    filePickerLoading.value = false
  }
}

async function addSelected() {
  if (!selectedFiles.value.length) return
  await api.post(`/albums/${albumId.value}/items`, { file_ids: selectedFiles.value })
  showFilePicker.value = false
  selectedFiles.value = []
  await loadAlbum()
}

function openFile(item: any) {
  const idx = items.value.findIndex((i: any) => i.id === item.id)
  previewIndex.value = idx >= 0 ? idx : 0
  previewFile.value = { ...item, file: item }
}

function onNavigate(direction: number) {
  const newIdx = previewIndex.value + direction
  if (newIdx >= 0 && newIdx < items.value.length) {
    previewIndex.value = newIdx
    previewFile.value = { ...items.value[newIdx] }
  }
}

async function doShare(username: string, permission: string) {
  shareError.value = ''
  try {
    await albumStore.shareAlbum(albumId.value, username, permission)
    await loadAlbum()
  } catch (e: any) {
    shareError.value = e.response?.data?.error?.message || 'Failed to share'
  }
}

async function doRemoveShare(shareId: string) {
  await albumStore.removeShare(albumId.value, shareId)
  await loadAlbum()
}
</script>
