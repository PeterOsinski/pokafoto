<template>
  <div class="max-w-6xl mx-auto p-4">
    <h2 class="text-xl font-semibold text-gray-200 mb-4">Search</h2>

    <div class="mb-4">
      <input
        v-model="query"
        type="text"
        placeholder="Search files by name, tags, comments..."
        class="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500"
        @keydown.enter="doSearch"
      />
    </div>

    <div class="mb-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
      <div>
        <label class="block text-sm text-gray-400 mb-1">Date Added After</label>
        <input v-model="createdAfter" type="date" class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500" />
      </div>
      <div>
        <label class="block text-sm text-gray-400 mb-1">Date Added Before</label>
        <input v-model="createdBefore" type="date" class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500" />
      </div>
      <div>
        <label class="block text-sm text-gray-400 mb-1">Date Created After</label>
        <input v-model="takenAfter" type="date" class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500" />
      </div>
      <div>
        <label class="block text-sm text-gray-400 mb-1">Date Created Before</label>
        <input v-model="takenBefore" type="date" class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500" />
      </div>
    </div>

    <div class="mb-4">
      <SizeRangeSlider
        :min="sizeMin"
        :max="sizeMax"
        :abs-min="0"
        :abs-max="1073741824"
        @update:min="(v: number) => sizeMin = v"
        @update:max="(v: number) => sizeMax = v"
      />
    </div>

    <div class="mb-4">
      <label class="block text-sm text-gray-400 mb-1">Tags</label>
      <TagInput v-model="tags" placeholder="Add tags..." />
    </div>

    <div class="mb-6 flex gap-2">
      <button @click="doSearch" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors">Search</button>
      <button @click="clearFilters" class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 text-sm rounded-lg transition-colors">Clear</button>
    </div>

    <div v-if="loading" class="text-gray-400 text-sm">Searching...</div>

    <template v-else>
      <p v-if="searched" class="text-sm text-gray-400 mb-3">{{ results.length }} result{{ results.length === 1 ? '' : 's' }}</p>

      <div class="space-y-2">
        <div
          v-for="item in results"
          :key="item.id"
          class="flex items-center gap-3 p-3 bg-gray-800 rounded-lg border border-gray-700 hover:border-gray-500 transition-colors cursor-pointer"
          @click="preview(item)"
        >
          <img v-if="item.thumbnails?.sm" :src="item.thumbnails.sm.url" class="w-12 h-12 object-cover rounded" alt="" loading="lazy" />
          <div v-else class="w-12 h-12 flex items-center justify-center bg-gray-900 rounded text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm text-gray-200 truncate">{{ item.originalName }}</p>
            <div class="flex flex-wrap items-center gap-1 mt-0.5">
              <span class="text-xs text-gray-500">{{ formatSize(item.sizeBytes) }}</span>
              <span class="text-xs text-gray-600">·</span>
              <span class="text-xs text-gray-500">{{ formatDate(item.createdAt) }}</span>
              <span v-if="item.folder_path" class="text-xs text-gray-600">·</span>
              <span v-if="item.folder_path" class="text-xs text-blue-400 truncate max-w-[200px]">{{ item.folder_path }}</span>
            </div>
          </div>
          <button @click.stop="downloadFile(item)" class="px-2 py-1 text-xs text-blue-400 hover:text-blue-300">Download</button>
        </div>
      </div>

      <div v-if="searched && !results.length" class="text-center py-12 text-gray-500">
        <p>No results found. Try adjusting your search filters.</p>
      </div>
    </template>

    <div v-if="previewItem" class="fixed inset-0 z-50 bg-black/80 flex flex-col" @click.self="previewItem = null">
      <div class="flex items-center justify-end p-3">
        <button @click="previewItem = null" class="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 text-white text-sm rounded-lg">Close</button>
      </div>
      <div class="flex-1 flex items-center justify-center p-4">
        <img v-if="previewItem.thumbnails?.preview" :src="previewItem.thumbnails.preview.url" class="max-w-full max-h-full object-contain rounded" alt="" />
        <div v-else class="text-gray-400">No preview available</div>
      </div>
      <div class="p-3 text-center">
        <p class="text-sm text-gray-300">{{ previewItem.originalName }} - {{ formatSize(previewItem.sizeBytes) }}</p>
        <button @click="downloadFile(previewItem)" class="mt-2 px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg">Download</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import api from '../api/client'
import TagInput from '../components/TagInput.vue'
import SizeRangeSlider from '../components/SizeRangeSlider.vue'

const query = ref('')
const sizeMin = ref(0)
const sizeMax = ref(1073741824)
const createdAfter = ref('')
const createdBefore = ref('')
const takenAfter = ref('')
const takenBefore = ref('')
const tags = ref<string[]>([])
const results = ref<any[]>([])
const loading = ref(false)
const searched = ref(false)
const previewItem = ref<any>(null)

async function doSearch() {
  loading.value = true
  searched.value = true
  try {
    const params: Record<string, string> = { limit: '100' }
    if (query.value) params.q = query.value
    if (sizeMin.value > 0) params.size_min = String(sizeMin.value)
    if (sizeMax.value < 1073741824) params.size_max = String(sizeMax.value)
    if (createdAfter.value) params.created_after = new Date(createdAfter.value).toISOString()
    if (createdBefore.value) params.created_before = new Date(createdBefore.value + 'T23:59:59').toISOString()
    if (takenAfter.value) params.taken_after = takenAfter.value
    if (takenBefore.value) params.taken_before = takenBefore.value
    if (tags.value.length) params.tags = tags.value.join(',')

    const res = await api.get('/search', { params })
    results.value = res.data.items || []
  } finally {
    loading.value = false
  }
}

function clearFilters() {
  query.value = ''
  sizeMin.value = 0
  sizeMax.value = 1073741824
  createdAfter.value = ''
  createdBefore.value = ''
  takenAfter.value = ''
  takenBefore.value = ''
  tags.value = []
  results.value = []
  searched.value = false
}

function preview(item: any) {
  previewItem.value = item
}

function downloadFile(item: any) {
  const url = `/api/v1/download/${item.id}`
  window.open(url, '_blank')
}

function formatSize(bytes: number): string {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return bytes + ' B'
}

function formatDate(d: string): string {
  return new Date(d).toLocaleDateString()
}
</script>
