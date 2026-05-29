<template>
  <div class="p-3">
    <span class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider">Folders</span>
    <div v-if="root" class="mt-2">
      <div
        v-for="child in root.children"
        :key="child.path"
      >
        <button
          @click="toggle(child.path)"
          class="flex items-center w-full text-left text-sm py-1 px-2 rounded hover:bg-[var(--bg-elevated)]"
          :class="isActive(child.path) ? 'text-[var(--accent)]' : 'text-[var(--text-secondary)]'"
        >
          <span class="mr-1 text-xs">{{ openPaths.has(child.path) ? '▼' : '▶' }}</span>
          <router-link :to="`/?path=${encodeURIComponent(child.path)}`" @click.stop class="flex-1 text-[var(--text-primary)]">
            {{ child.name }}
          </router-link>
          <span class="text-xs text-[var(--text-secondary)]">{{ child.fileCount }}</span>
        </button>
        <div v-if="openPaths.has(child.path)" class="ml-3">
          <div v-for="grandchild in child.children" :key="grandchild.path">
            <router-link
              :to="`/?path=${encodeURIComponent(grandchild.path)}`"
              class="block text-sm py-1 px-2 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-secondary)]"
              :class="isActive(grandchild.path) ? 'text-[var(--accent)]' : ''"
            >
              {{ grandchild.name }}
              <span class="text-xs opacity-60 ml-1">{{ grandchild.fileCount }}</span>
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client'

interface DirEntry {
  path: string
  name: string
  fileCount: number
  children?: DirEntry[]
}

const route = useRoute()
const root = ref<DirEntry | null>(null)
const openPaths = ref(new Set<string>())

function isActive(path: string) {
  return route.query.path === path
}

function toggle(path: string) {
  if (openPaths.value.has(path)) {
    openPaths.value.delete(path)
  } else {
    openPaths.value.add(path)
  }
  openPaths.value = new Set(openPaths.value)
}

async function loadDirs() {
  try {
    const params: any = {}
    if (route.query.all_folders === 'true') {
      params.all_folders = 'true'
    }
    const res = await api.get('/dirs', { params })
    root.value = res.data
  } catch (e) {
    console.error('Failed to load dirs', e)
  }
}

onMounted(() => {
  loadDirs()
})

watch(() => route.query.all_folders, () => {
  loadDirs()
})
</script>
