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
        <router-link to="/folders" class="text-sm" :class="navClass('/folders')">Folders</router-link>
        <router-link to="/timeline" class="text-sm" :class="navClass('/timeline')">Timeline</router-link>
        <router-link to="/map" class="text-sm" :class="navClass('/map')">Map</router-link>
        <router-link to="/trash" class="text-sm" :class="navClass('/trash')">Trash</router-link>
        <router-link to="/albums" class="text-sm" :class="navClass('/albums')">Albums</router-link>
        <router-link to="/search" class="text-sm" :class="navClass('/search')">Search</router-link>
        <router-link v-if="auth.isAdmin" to="/admin" class="text-sm" :class="navClass('/admin')">Admin</router-link>
      </nav>
      <div class="flex items-center gap-3">
        <div v-if="quotaBytes !== null" class="hidden sm:flex items-center gap-2">
          <div class="w-32 h-2 rounded-full" style="background: var(--bg-elevated)">
            <div
              class="h-2 rounded-full transition-all"
              :class="quotaClass"
              :style="{ width: quotaPct + '%' }"
            />
          </div>
          <span class="text-xs text-[var(--text-secondary)] whitespace-nowrap">{{ quotaDisplay }}</span>
        </div>
        <div v-if="trashCount > 0" class="hidden sm:flex items-center gap-1">
          <span class="text-xs text-[var(--text-secondary)] whitespace-nowrap">
            {{ trashCount }} in trash ({{ formatBytes(trashSizeBytes) }})
          </span>
        </div>
        <span class="text-sm text-[var(--text-secondary)] hidden sm:inline">{{ auth.user?.username }}</span>
        <button @click="handleLogout" class="text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">Logout</button>
      </div>
    </header>

    <div class="flex-1 flex overflow-hidden">
      <!-- Sidebar (desktop) - hidden on Folders view which has its own tree sidebar -->
      <aside v-if="route.path !== '/folders' && route.path !== '/'" class="hidden lg:block w-56 shrink-0 border-r overflow-y-auto" style="border-color: var(--border-color); background: var(--bg-surface)">
        <DirectoryTree />
      </aside>

      <!-- Main content -->
      <main class="flex-1" :class="route.path === '/folders' ? 'overflow-hidden' : 'overflow-y-auto p-4'">
        <router-view />
      </main>
    </div>

    <GlobalUploadTracker />

    <!-- Mobile bottom nav -->
    <nav class="md:hidden h-14 flex items-center justify-around border-t shrink-0" style="border-color: var(--border-color); background: var(--bg-surface)">
      <router-link to="/" class="flex flex-col items-center text-xs" :class="navClass('/')">🏠<span>Home</span></router-link>
      <router-link to="/folders" class="flex flex-col items-center text-xs" :class="navClass('/folders')">📁<span>Folders</span></router-link>
      <router-link to="/timeline" class="flex flex-col items-center text-xs" :class="navClass('/timeline')">📅<span>Timeline</span></router-link>
      <router-link to="/map" class="flex flex-col items-center text-xs" :class="navClass('/map')">🗺️<span>Map</span></router-link>
      <router-link to="/trash" class="flex flex-col items-center text-xs" :class="navClass('/trash')">🗑<span>Trash</span></router-link>
      <router-link to="/albums" class="flex flex-col items-center text-xs" :class="navClass('/albums')">📸<span>Albums</span></router-link>
      <router-link to="/search" class="flex flex-col items-center text-xs" :class="navClass('/search')">🔍<span>Search</span></router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useChunkedUploadStore } from './stores/chunkedUpload'
import DirectoryTree from './components/DirectoryTree.vue'
import GlobalUploadTracker from './components/GlobalUploadTracker.vue'
import api from './api/client'

const auth = useAuthStore()
const upload = useChunkedUploadStore()
const router = useRouter()
const route = useRoute()
const showS3Banner = ref(false)
const usedBytes = ref(0)
const quotaBytes = ref<number | null>(null)
const trashCount = ref(0)
const trashSizeBytes = ref(0)

const quotaDisplay = computed(() => {
  if (quotaBytes.value === null) return ''
  return formatBytes(usedBytes.value) + ' / ' + formatBytes(quotaBytes.value)
})

const quotaPct = computed(() => {
  if (!quotaBytes.value || quotaBytes.value === 0) return 0
  return Math.min((usedBytes.value / quotaBytes.value) * 100, 100)
})

const quotaClass = computed(() => {
  const pct = quotaPct.value
  if (pct > 80) return 'bg-[var(--error)]'
  if (pct > 60) return 'bg-[var(--warning)]'
  return 'bg-[var(--accent)]'
})

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return val.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function navClass(path: string) {
  if (path === '/') return route.path === '/' ? 'text-[var(--accent)]' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
  return route.path.startsWith(path) ? 'text-[var(--accent)]' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(async () => {
  await auth.fetchMe()
  upload.connectWS()
  upload.checkAndResumeAll()
  try {
    const res = await api.get('/health')
    if (res.data.s3_connected === false) {
      showS3Banner.value = true
    }
  } catch {}
  if (auth.isAuthenticated) {
    try {
      const meRes = await api.get('/auth/me')
      const user = meRes.data
      if (user.space_quota) {
        quotaBytes.value = user.space_quota
      }
      const statsRes = await api.get('/stats')
      usedBytes.value = statsRes.data.total_size_bytes || 0
      try {
        const trashRes = await api.get('/trash/stats')
        trashCount.value = trashRes.data.count || 0
        trashSizeBytes.value = trashRes.data.size_bytes || 0
      } catch {}
    } catch {}
  }
})
</script>
