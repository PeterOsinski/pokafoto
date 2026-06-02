<template>
  <div v-if="previewMode === 'sidebar' && lightboxFile" :class="{ 'flex': true }">
    <div class="flex-1 min-w-0">
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
              <span class="text-sm" style="color: var(--text-secondary)">{{ album?.item_count }} {{ album.item_count === 1 ? 'photo' : 'photos' }}</span>
              <button v-if="canAdd" @click="openFilePicker" class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors">+ Add Photos</button>
              <button v-if="album.is_owner" @click="showShare = true" class="px-3 py-1.5 bg-green-600 hover:bg-green-700 text-white text-sm rounded-lg transition-colors">Share</button>
            </div>
            <button v-if="album.is_owner" @click="deleteAlbum" class="px-3 py-1.5 bg-red-600/20 hover:bg-red-600/40 text-red-400 text-sm rounded-lg transition-colors">Delete</button>
          </div>

          <GalleryControls
            :layout="settings.layout.value"
            :sortBy="settings.sortBy.value"
            :thumbLevel="settings.thumbLevel.value"
            :previewMode="previewMode"
            :sortOptions="albumSortOptions"
            :hidePreviewToggle="true"
            @update:layout="v => settings.layout.value = v"
            @update:sortBy="v => settings.sortBy.value = v"
            @update:thumbLevel="v => settings.thumbLevel.value = v"
          />

          <div v-if="sortedItems.length === 0" class="text-center py-12" style="color: var(--text-secondary)">No photos in this album yet.</div>

          <GalleryTileView
            v-else-if="settings.layout.value === 'tiles'"
            :files="sortedItems"
            :thumbSizePx="settings.thumbSizePx.value"
            :selectedIds="new Set()"
            :selectionEnabled="false"
            @select="() => {}"
            @deselect="() => {}"
            @open="(i: number) => openLightbox(i)"
          />
          <GalleryListView
            v-else-if="settings.layout.value === 'list'"
            :files="sortedItems"
            :thumbSizePx="settings.thumbSizePx.value"
            :selectedIds="new Set()"
            :selectionEnabled="false"
            @select="() => {}"
            @deselect="() => {}"
            @open="(i: number) => openLightbox(i)"
          />
          <GalleryGroupedView
            v-else-if="settings.layout.value === 'grouped'"
            :files="sortedItems"
            :thumbSizePx="settings.thumbSizePx.value"
            :selectedIds="new Set()"
            :selectionEnabled="false"
            @select="() => {}"
            @deselect="() => {}"
            @open="(i: number) => openLightbox(i)"
          />
        </template>
      </div>
    </div>

    <PreviewSidebar
      :file="lightboxFile"
      :index="lightboxIndex"
      :total="sortedItems.length"
      :hasPrev="lightboxIndex > 0"
      :hasNext="lightboxIndex < sortedItems.length - 1"
      @close="closeLightbox"
      @prev="goPrev"
      @next="goNext"
    />
  </div>

  <div v-else class="max-w-6xl mx-auto p-4">
    <div class="flex items-center gap-3 mb-4">
      <button @click="$router.push('/albums')" class="text-gray-400 hover:text-gray-200 transition-colors">&larr; Back</button>
      <h2 class="text-xl font-semibold text-gray-200">{{ album?.name }}</h2>
      <span v-if="album?.owner_name" class="text-xs text-gray-500">by {{ album.owner_name }}</span>
    </div>

    <div v-if="loading" class="text-gray-400 text-sm">Loading...</div>

    <template v-else-if="album">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-3">
          <span class="text-sm" style="color: var(--text-secondary)">{{ album?.item_count }} {{ album.item_count === 1 ? 'photo' : 'photos' }}</span>
          <button v-if="canAdd" @click="openFilePicker" class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors">+ Add Photos</button>
          <button v-if="album.is_owner" @click="showShare = true" class="px-3 py-1.5 bg-green-600 hover:bg-green-700 text-white text-sm rounded-lg transition-colors">Share</button>
        </div>
        <button v-if="album.is_owner" @click="deleteAlbum" class="px-3 py-1.5 bg-red-600/20 hover:bg-red-600/40 text-red-400 text-sm rounded-lg transition-colors">Delete</button>
      </div>

      <GalleryControls
        :layout="settings.layout.value"
        :sortBy="settings.sortBy.value"
        :thumbLevel="settings.thumbLevel.value"
        :previewMode="previewMode"
        :sortOptions="albumSortOptions"
        @update:layout="v => settings.layout.value = v"
        @update:sortBy="v => settings.sortBy.value = v"
        @update:thumbLevel="v => settings.thumbLevel.value = v"
        @togglePreviewMode="togglePreviewMode"
      />

      <div v-if="sortedItems.length === 0" class="text-center py-12" style="color: var(--text-secondary)">No photos in this album yet.</div>

      <GalleryTileView
        v-else-if="settings.layout.value === 'tiles'"
        :files="sortedItems"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="new Set()"
        :selectionEnabled="false"
        @select="() => {}"
        @deselect="() => {}"
        @open="(i: number) => openLightbox(i)"
      />
      <GalleryListView
        v-else-if="settings.layout.value === 'list'"
        :files="sortedItems"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="new Set()"
        :selectionEnabled="false"
        @select="() => {}"
        @deselect="() => {}"
        @open="(i: number) => openLightbox(i)"
      />
      <GalleryGroupedView
        v-else-if="settings.layout.value === 'grouped'"
        :files="sortedItems"
        :thumbSizePx="settings.thumbSizePx.value"
        :selectedIds="new Set()"
        :selectionEnabled="false"
        @select="() => {}"
        @deselect="() => {}"
        @open="(i: number) => openLightbox(i)"
      />
    </template>

    <Lightbox
      v-if="previewMode !== 'sidebar'"
      :file="lightboxFile"
      :index="lightboxIndex"
      :total="sortedItems.length"
      :has-prev="lightboxIndex > 0"
      :has-next="lightboxIndex < sortedItems.length - 1"
      @close="closeLightbox"
      @next="goNext"
      @prev="goPrev"
    />

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
          <button @click="addSelected" :disabled="!selectedFiles.length" class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm rounded-lg transition-colors">Add {{ selectedFiles.length ? `(${selectedFiles.length})` : '' }}</button>
          <button @click="showFilePicker = false" class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 text-sm rounded-lg transition-colors">Cancel</button>
        </div>
      </div>
    </div>

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
import { useLocalSettings } from '../composables/useLocalSettings'
import ShareDialog from '../components/ShareDialog.vue'
import Lightbox from '../components/Lightbox.vue'
import PreviewSidebar from '../components/PreviewSidebar.vue'
import GalleryControls from '../components/GalleryControls.vue'
import GalleryTileView from '../components/GalleryTileView.vue'
import GalleryListView from '../components/GalleryListView.vue'
import GalleryGroupedView from '../components/GalleryGroupedView.vue'

const route = useRoute()
const router = useRouter()
const albumStore = useAlbumStore()
const settings = useLocalSettings()

const album = ref<Album | null>(null)
const items = ref<any[]>([])
const loading = ref(true)
const showFilePicker = ref(false)
const selectedFiles = ref<string[]>([])
const myFiles = ref<any[]>([])
const filePickerLoading = ref(false)
const showShare = ref(false)
const shareError = ref('')

const albumId = computed(() => route.params.id as string)
const canAdd = computed(() => album.value?.is_owner || album.value?.share_permission === 'edit')

const previewMode = computed({
  get: () => settings.previewMode.value,
  set: (v) => settings.previewMode.value = v,
})

const lightboxFile = computed(() => {
  if (previewIndex.value < 0 || previewIndex.value >= sortedItems.value.length) return null
  const item = sortedItems.value[previewIndex.value]
  return { ...item, file: item }
})

const lightboxIndex = ref(0)

const previewIndex = computed({
  get: () => lightboxIndex.value,
  set: (v) => lightboxIndex.value = v,
})

const albumSortOptions = [
  { value: 'created_at', label: 'Date Added' },
  { value: 'taken_at', label: 'Date Taken' },
  { value: 'filename', label: 'File Name' },
]

const sortedItems = computed(() => {
  const list = [...items.value]
  const by = settings.sortBy.value
  list.sort((a, b) => {
    let va: any, vb: any
    if (by === 'filename') {
      va = (a.originalName || a.OriginalName || '')
      vb = (b.originalName || b.OriginalName || '')
      return va.localeCompare(vb)
    }
    if (by === 'taken_at') {
      va = a.TakenAt || a.takenAt || a.taken_at || ''
      vb = b.TakenAt || b.takenAt || b.taken_at || ''
    } else {
      va = a.CreatedAt || a.createdAt || a.created_at || ''
      vb = b.CreatedAt || b.createdAt || b.created_at || ''
    }
    if (!va && !vb) return 0
    if (!va) return 1
    if (!vb) return -1
    return vb.localeCompare(va)
  })
  return list
})

function openLightbox(index: number) {
  lightboxIndex.value = index
}

function closeLightbox() {
  lightboxIndex.value = -1
}

function goPrev() {
  if (lightboxIndex.value > 0) {
    lightboxIndex.value--
  }
}

function goNext() {
  if (lightboxIndex.value < sortedItems.value.length - 1) {
    lightboxIndex.value++
  }
}

function togglePreviewMode() {
  settings.previewMode.value = settings.previewMode.value === 'sidebar' ? 'lightbox' : 'sidebar'
}

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
