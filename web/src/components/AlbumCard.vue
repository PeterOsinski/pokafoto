<template>
  <button @click="$emit('click')" class="relative bg-gray-800 rounded-xl border border-gray-700 hover:border-gray-500 overflow-hidden transition-colors text-left w-full">
    <div class="aspect-[3/2] bg-gray-900 flex items-center justify-center">
      <svg v-if="!hasCover" class="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <img v-else :src="coverUrl" class="w-full h-full object-cover" alt="" />
    </div>
    <div class="p-3">
      <h3 class="text-sm font-medium text-gray-200 truncate">{{ album.name }}</h3>
      <div class="flex items-center justify-between mt-1">
        <span class="text-xs text-gray-400">{{ album.item_count }} {{ album.item_count === 1 ? 'photo' : 'photos' }}</span>
        <span v-if="album.is_shared && album.owner_name" class="text-xs text-gray-500">{{ album.owner_name }}</span>
      </div>
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface AlbumProp {
  id: string
  name: string
  item_count: number
  is_shared: boolean
  owner_name?: string
  cover_url?: string
}

const props = defineProps<{
  album: AlbumProp
}>()

defineEmits<{
  click: []
}>()

const hasCover = computed(() => !!props.album.cover_url)
const coverUrl = computed(() => props.album.cover_url || '')
</script>
