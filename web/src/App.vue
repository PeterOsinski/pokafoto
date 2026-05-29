<template>
  <div class="min-h-screen flex flex-col bg-[var(--bg-primary)]">
    <div v-if="showS3Banner" class="px-4 py-2 text-sm text-center bg-[var(--warning)]/10 text-[var(--warning)] border-b" style="border-color: var(--warning)">
      Storage backend is disconnected. Uploads will be stored locally until S3 is reconnected.
    </div>
    <header class="h-12 flex items-center justify-between px-4 border-b shrink-0" style="border-color: var(--border-color); background: var(--bg-surface)">
      <router-link to="/" class="text-[var(--text-primary)] font-bold text-lg">Drive</router-link>
      <!-- Desktop nav -->
      <nav class="hidden md:flex items-center gap-6">
        <router-link to="/" class="text-sm" :class="navClass('/')">Gallery</router-link>
        <router-link to="/timeline" class="text-sm" :class="navClass('/timeline')">Timeline</router-link>
        <router-link to="/map" class="text-sm" :class="navClass('/map')">Map</router-link>
        <router-link to="/upload" class="text-sm" :class="navClass('/upload')">Upload</router-link>
        <router-link v-if="auth.isAdmin" to="/admin" class="text-sm" :class="navClass('/admin')">Admin</router-link>
      </nav>
      <div class="flex items-center gap-3">
        <span class="text-sm text-[var(--text-secondary)] hidden sm:inline">{{ auth.user?.username }}</span>
        <button @click="handleLogout" class="text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Logout</button>
      </div>
    </header>

    <div class="flex-1 flex overflow-hidden">
      <!-- Sidebar (desktop) -->
      <aside class="hidden lg:block w-56 shrink-0 border-r overflow-y-auto" style="border-color: var(--border-color); background: var(--bg-surface)">
        <DirectoryTree />
      </aside>

      <!-- Main content -->
      <main class="flex-1 overflow-y-auto p-4">
        <router-view />
      </main>
    </div>

    <GlobalUploadTracker />

    <!-- Mobile bottom nav -->
    <nav class="md:hidden h-14 flex items-center justify-around border-t shrink-0" style="border-color: var(--border-color); background: var(--bg-surface)">
      <router-link to="/" class="flex flex-col items-center text-xs" :class="navClass('/')">🏠<span>Home</span></router-link>
      <router-link to="/timeline" class="flex flex-col items-center text-xs" :class="navClass('/timeline')">📅<span>Timeline</span></router-link>
      <router-link to="/map" class="flex flex-col items-center text-xs" :class="navClass('/map')">🗺️<span>Map</span></router-link>
      <router-link to="/upload" class="flex flex-col items-center text-xs" :class="navClass('/upload')">⬆<span>Upload</span></router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useUploadStore } from './stores/upload'
import DirectoryTree from './components/DirectoryTree.vue'
import GlobalUploadTracker from './components/GlobalUploadTracker.vue'
import api from './api/client'

const auth = useAuthStore()
const upload = useUploadStore()
const router = useRouter()
const route = useRoute()
const showS3Banner = ref(false)

function navClass(path: string) {
  return route.path === path ? 'text-[var(--accent)]' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(async () => {
  await auth.fetchMe()
  upload.connectWS()
  try {
    const res = await api.get('/health')
    if (res.data.s3_connected === false) {
      showS3Banner.value = true
    }
  } catch {}
})
</script>
