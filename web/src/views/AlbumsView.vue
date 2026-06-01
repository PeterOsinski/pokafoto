<template>
  <div class="max-w-6xl mx-auto p-4">
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-semibold text-gray-200">Albums</h2>
      <button @click="showCreate = true" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors">
        + New Album
      </button>
    </div>

    <div v-if="albumStore.loading" class="text-gray-400 text-sm">Loading...</div>

    <template v-else>
      <section v-if="albumStore.myAlbums.length" class="mb-8">
        <h3 class="text-sm font-medium text-gray-400 mb-3 uppercase tracking-wider">My Albums</h3>
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
          <AlbumCard
            v-for="album in albumStore.myAlbums"
            :key="album.id"
            :album="album"
            @click="goToAlbum(album.id)"
          />
        </div>
      </section>

      <section v-if="albumStore.sharedAlbums.length">
        <h3 class="text-sm font-medium text-gray-400 mb-3 uppercase tracking-wider">Shared with me</h3>
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
          <AlbumCard
            v-for="album in albumStore.sharedAlbums"
            :key="album.id"
            :album="album"
            @click="goToAlbum(album.id)"
          />
        </div>
      </section>

      <div v-if="!albumStore.myAlbums.length && !albumStore.sharedAlbums.length" class="text-center py-12 text-gray-500">
        <p>No albums yet. Create your first album to get started.</p>
      </div>
    </template>

    <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="showCreate = false">
      <div class="bg-gray-800 rounded-xl border border-gray-700 p-6 w-full max-w-md" @click.stop>
        <h3 class="text-lg font-medium text-gray-200 mb-4">Create Album</h3>
        <input
          v-model="newName"
          type="text"
          placeholder="Album name"
          class="w-full px-3 py-2 mb-2 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500"
          @keydown.enter="doCreate"
          autofocus
        />
        <input
          v-model="newDesc"
          type="text"
          placeholder="Description (optional)"
          class="w-full px-3 py-2 mb-4 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500"
          @keydown.enter="doCreate"
        />
        <div class="flex gap-2">
          <button @click="doCreate" :disabled="!newName.trim()" class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm rounded-lg transition-colors">
            Create
          </button>
          <button @click="showCreate = false" class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 text-sm rounded-lg transition-colors">
            Cancel
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAlbumStore } from '../stores/albums'
import AlbumCard from '../components/AlbumCard.vue'

const router = useRouter()
const albumStore = useAlbumStore()
const showCreate = ref(false)
const newName = ref('')
const newDesc = ref('')

onMounted(() => {
  albumStore.fetchAlbums()
})

function goToAlbum(id: string) {
  router.push(`/albums/${id}`)
}

async function doCreate() {
  if (!newName.value.trim()) return
  await albumStore.createAlbum(newName.value.trim(), newDesc.value.trim() || undefined)
  showCreate.value = false
  newName.value = ''
  newDesc.value = ''
}
</script>
