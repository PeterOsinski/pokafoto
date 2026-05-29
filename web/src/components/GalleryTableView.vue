<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm text-[var(--text-primary)]">
      <thead>
        <tr class="border-b border-[var(--border-color)] text-left text-xs text-[var(--text-secondary)] uppercase tracking-wide">
          <th v-if="selectionEnabled" class="py-2 px-3 w-8"></th>
          <th class="py-2 px-3">Name</th>
          <th class="py-2 px-3 hidden sm:table-cell">Type</th>
          <th class="py-2 px-3 hidden md:table-cell">Size</th>
          <th class="py-2 px-3 hidden lg:table-cell">Date</th>
          <th class="py-2 px-3">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(file, i) in files"
          :key="file.id"
          class="border-b border-[var(--border-color)] hover:bg-[var(--bg-elevated)] transition-colors"
          :class="selectedIds.has(file.id) ? 'bg-[var(--accent)]/10' : ''"
        >
          <td v-if="selectionEnabled" class="py-2 px-3">
            <button
              v-if="!selectedIds.has(file.id)"
              class="w-5 h-5 rounded border-2 border-white/30 flex items-center justify-center"
              @click.prevent.stop="$emit('select', file.id)"
            />
            <button
              v-else
              class="w-5 h-5 rounded border-2 flex items-center justify-center bg-[var(--accent)] border-[var(--accent)]"
              @click.prevent.stop="$emit('deselect', file.id)"
            >
              <span class="text-white text-xs font-bold">&#10003;</span>
            </button>
          </td>
          <td class="py-2 px-3 font-medium cursor-pointer" @click="$emit('open', i)">
            <div class="flex items-center gap-2">
              <div v-if="file.mediaType === 'photo'" class="text-lg">&#128247;</div>
              <div v-else-if="file.mediaType === 'video'" class="text-lg">&#127916;</div>
              <div v-else class="text-lg">&#128196;</div>
              <span class="truncate max-w-[200px] sm:max-w-[300px]">{{ file.originalName }}</span>
            </div>
          </td>
          <td class="py-2 px-3 hidden sm:table-cell" @click="$emit('open', i)">
            <span class="px-2 py-0.5 rounded text-xs" :class="typeBadgeClass(file.mediaType)">
              {{ file.mediaType }}
            </span>
          </td>
          <td class="py-2 px-3 text-[var(--text-secondary)] hidden md:table-cell whitespace-nowrap" @click="$emit('open', i)">
            {{ formatSize(file.sizeBytes) }}
          </td>
          <td class="py-2 px-3 text-[var(--text-secondary)] hidden lg:table-cell whitespace-nowrap" @click="$emit('open', i)">
            {{ formatDate(file.takenAt) }}
          </td>
          <td class="py-2 px-3">
            <div class="flex items-center gap-1">
              <button
                @click.prevent.stop="$emit('download', file.id)"
                class="p-1 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
                title="Download"
              >
                &#11015;
              </button>
              <button
                @click.prevent.stop="$emit('delete', file.id)"
                class="p-1 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-red-400"
                title="Delete"
              >
                &#128465;
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  durationSec?: number
  takenAt?: string
  folder_id?: string | null
  thumbnails?: any
}

defineProps<{
  files: FileItem[]
  selectedIds: Set<string>
  selectionEnabled: boolean
}>()

defineEmits<{
  select: [id: string]
  deselect: [id: string]
  open: [index: number]
  download: [id: string]
  delete: [id: string]
}>()

function formatDate(takenAt?: string): string {
  if (!takenAt) return ''
  return new Date(takenAt).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function typeBadgeClass(mediaType: string): string {
  switch (mediaType) {
    case 'video': return 'bg-purple-900/40 text-purple-300'
    case 'photo': return 'bg-blue-900/40 text-blue-300'
    default: return 'bg-gray-700/40 text-gray-300'
  }
}
</script>
