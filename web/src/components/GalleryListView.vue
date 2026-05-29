<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm text-[var(--text-primary)]">
      <thead>
        <tr class="border-b border-[var(--border-color)] text-left text-xs text-[var(--text-secondary)] uppercase tracking-wide">
          <th v-if="selectionEnabled" class="py-2 px-3 w-8"></th>
          <th class="py-2 px-3 w-12"></th>
          <th class="py-2 px-3">Name</th>
          <th class="py-2 px-3 hidden sm:table-cell">Date</th>
          <th class="py-2 px-3 hidden md:table-cell">Type</th>
          <th class="py-2 px-3 hidden lg:table-cell">Size</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(file, i) in files"
          :key="file.id"
          class="border-b border-[var(--border-color)] cursor-pointer hover:bg-[var(--bg-elevated)] transition-colors"
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
          <td class="py-2 px-3" @click="$emit('open', i)">
            <img
              v-if="file.thumbnails?.sm?.url"
              :src="file.thumbnails.sm.url"
              :alt="file.originalName"
              class="w-10 h-10 rounded object-cover"
              loading="lazy"
            />
            <div v-else-if="file.mediaType === 'file'" class="w-10 h-10 rounded flex flex-col items-center justify-center text-[var(--text-secondary)]" style="background: var(--bg-elevated)">
              <span class="text-sm">&#128196;</span>
              <span class="text-[10px] font-mono opacity-60 leading-none">{{ fileExtension(file) }}</span>
            </div>
            <div v-else class="w-10 h-10 rounded flex items-center justify-center text-lg" style="background: var(--bg-elevated)">
              {{ file.mediaType === 'video' ? '&#9654;' : '&#128196;' }}
            </div>
          </td>
          <td class="py-2 px-3 font-medium truncate max-w-[200px] sm:max-w-[300px]" @click="$emit('open', i)">{{ file.originalName }}</td>
          <td class="py-2 px-3 text-[var(--text-secondary)] hidden sm:table-cell whitespace-nowrap" @click="$emit('open', i)">{{ formatDate(file.takenAt) }}</td>
          <td class="py-2 px-3 hidden md:table-cell" @click="$emit('open', i)">
            <span class="px-2 py-0.5 rounded text-xs" :class="typeBadgeClass(file.mediaType)">
              {{ file.mediaType }}
            </span>
          </td>
          <td class="py-2 px-3 text-[var(--text-secondary)] hidden lg:table-cell whitespace-nowrap" @click="$emit('open', i)">{{ formatSize(file.sizeBytes) }}</td>
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
  thumbnails?: {
    sm?: { url: string; width: number; height: number }
  }
}

defineProps<{
  files: FileItem[]
  thumbSize?: string
  selectedIds: Set<string>
  selectionEnabled: boolean
}>()

defineEmits<{
  select: [id: string]
  deselect: [id: string]
  open: [index: number]
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

function fileExtension(file: FileItem): string {
  const name = file.originalName || file.filename || ''
  const ext = name.split('.').pop() || ''
  return ext ? `.${ext.toLowerCase()}` : ''
}
</script>
