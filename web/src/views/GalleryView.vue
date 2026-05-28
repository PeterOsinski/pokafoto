<template>
  <div>
    <FilterBar
      v-model:mediaType="mediaType"
      v-model:sortBy="sortBy"
      v-model:layout="layout"
      v-model:thumbSize="thumbSize"
      @update:mediaType="loadFiles()"
      @update:sortBy="loadFiles()"
    />

    <div v-if="files.length === 0 && !loading" class="text-center py-20 text-[var(--text-secondary)]">
      <p class="text-lg">No photos yet.</p>
      <p class="mt-2">Upload your first photo to get started.</p>
      <router-link to="/upload" class="mt-4 inline-block px-6 py-2 rounded-md text-white" style="background: var(--accent)">Upload</router-link>
    </div>

    <GalleryTileView v-if="layout === 'tiles'" :files="files" :thumbSize="thumbSize" @open="openLightbox" />
    <GalleryListView v-else-if="layout === 'list'" :files="files" @open="openLightbox" />
    <GalleryGroupedView v-else-if="layout === 'grouped'" :files="files" :thumbSize="thumbSize" @open="openLightbox" />

    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">Loading...</div>
    <div ref="sentinel" class="h-4"></div>

    <Lightbox
      :file="lightboxFile"
      :index="lightboxIndex"
      :total="files.length"
      :hasPrev="lightboxIndex > 0"
      :hasNext="lightboxIndex < files.length - 1"
      @close="lightboxIndex = -1"
      @prev="lightboxIndex--"
      @next="lightboxIndex++"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client'
import Lightbox from '../components/Lightbox.vue'
import GalleryTileView from '../components/GalleryTileView.vue'
import GalleryListView from '../components/GalleryListView.vue'
import GalleryGroupedView from '../components/GalleryGroupedView.vue'
import FilterBar from '../components/FilterBar.vue'

interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  durationSec?: number
  takenAt?: string
  videoStill?: { url: string }
  thumbnails?: {
    sm: { url: string; width: number; height: number }
    lg: { url: string; width: number; height: number }
    md: { url: string; width: number; height: number }
    preview: { url: string; width: number; height: number }
  }
}

const route = useRoute()

const files = ref<FileItem[]>([])
const total = ref(0)
const nextCursor = ref('')
const loading = ref(false)
const mediaType = ref('')
const sortBy = ref('taken_at')
const layout = ref('tiles')
const thumbSize = ref<'sm' | 'md' | 'lg'>('md')
const lightboxIndex = ref(-1)

const lightboxFile = computed(() => {
  if (lightboxIndex.value < 0 || lightboxIndex.value >= files.value.length) return null
  return files.value[lightboxIndex.value]
})

async function loadFiles(reset = true) {
  if (reset) {
    files.value = []
    nextCursor.value = ''
  }
  loading.value = true
  try {
    const params: any = { sort: sortBy.value, order: 'desc', limit: 100 }
    if (mediaType.value) params.media_type = mediaType.value
    if (nextCursor.value) params.cursor = nextCursor.value
    if (route.query.date_from) params.date_from = route.query.date_from
    if (route.query.date_to) params.date_to = route.query.date_to
    const res = await api.get('/files', { params })
    files.value = reset ? res.data.items : [...files.value, ...res.data.items]
    total.value = res.data.total
    nextCursor.value = res.data.nextCursor || ''
  } catch (e) {
    console.error('Failed to load files', e)
  } finally {
    loading.value = false
  }
}

function openLightbox(index: number) {
  lightboxIndex.value = index
}

watch(() => route.query, () => loadFiles(), { immediate: false })

loadFiles()
</script>
