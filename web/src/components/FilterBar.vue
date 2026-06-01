<template>
  <div class="flex items-center gap-2 mb-6 flex-wrap">
    <div class="flex items-center gap-1 rounded-lg p-1" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
      <button
        v-for="opt in mediaOptions"
        :key="opt.value"
        :title="opt.label"
        class="p-1.5 rounded-md transition-colors"
        :class="opt.value === mediaType ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'"
        @click="$emit('update:mediaType', opt.value)"
        v-html="opt.icon"
      />
    </div>

    <select :modelValue="sortBy" @change="emit('update:sortBy', ($event.target as HTMLSelectElement).value)" class="px-3 py-1 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)">
      <option value="taken_at">Date Taken</option>
      <option value="created_at">Date Uploaded</option>
      <option value="filename">File Name</option>
    </select>

    <div class="flex items-center gap-1 rounded-lg p-1" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
      <button
        v-for="opt in layoutOptions"
        :key="opt.value"
        :title="opt.label"
        class="p-1.5 rounded-md transition-colors"
        :class="opt.value === layout ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'"
        @click="$emit('update:layout', opt.value)"
        v-html="opt.icon"
      />
    </div>

    <div class="flex items-center gap-2 px-3 py-1 rounded" style="background: var(--bg-elevated); border: 1px solid var(--border-color)">
      <span class="text-xs text-[var(--text-secondary)]" style="font-weight: 300">S</span>
      <input
        type="range"
        :value="thumbLevel"
        min="0"
        max="9"
        class="thumb-slider"
        @input="$emit('update:thumbLevel', Number(($event.target as HTMLInputElement).value))"
      />
      <span class="text-xs text-[var(--text-secondary)]" style="font-weight: 700">L</span>
    </div>

    <label class="flex items-center gap-2 px-3 py-1 rounded text-sm cursor-pointer select-none" style="background: var(--bg-elevated); border: 1px solid var(--border-color)" :class="includeAllFolders ? 'border-[var(--accent)]' : 'border-[var(--border-color)]'">
      <input
        type="checkbox"
        :checked="includeAllFolders"
        @change="$emit('update:includeAllFolders', ($event.target as HTMLInputElement).checked)"
        class="accent-[var(--accent)]"
      />
      <span :class="includeAllFolders ? 'text-[var(--accent)]' : 'text-[var(--text-secondary)]'">All folders</span>
    </label>

    <button
      @click="$emit('togglePreviewMode')"
      class="flex items-center gap-1 px-3 py-1 rounded text-sm"
      :style="{ background: 'var(--bg-elevated)', color: previewMode === 'sidebar' ? 'var(--accent)' : 'var(--text-secondary)', border: '1px solid ' + (previewMode === 'sidebar' ? 'var(--accent)' : 'var(--border-color)') }"
      :title="previewMode === 'sidebar' ? 'Switch to lightbox preview' : 'Switch to sidebar preview'"
    >
      <span v-if="previewMode === 'sidebar'">&#9638;</span>
      <span v-else>&#9641;</span>
      {{ previewMode === 'sidebar' ? 'Sidebar' : 'Preview' }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{ mediaType: string; sortBy: string; layout: string; thumbLevel: number; includeAllFolders: boolean; previewMode: string }>()
const emit = defineEmits<{
  'update:mediaType': [value: string]
  'update:sortBy': [value: string]
  'update:layout': [value: string]
  'update:thumbLevel': [value: number]
  'update:includeAllFolders': [value: boolean]
  togglePreviewMode: []
}>()

const mediaOptions = [
  { value: '', label: 'All', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>' },
  { value: 'photo', label: 'Photos', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/></svg>' },
  { value: 'video', label: 'Videos', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>' },
  { value: 'file', label: 'Files', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>' },
]

const layoutOptions = [
  { value: 'tiles', label: 'Tiles', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>' },
  { value: 'list', label: 'List', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>' },
  { value: 'grouped', label: 'Groups by Day', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>' },
]
</script>

<style scoped>
.thumb-slider {
  -webkit-appearance: none;
  appearance: none;
  width: 80px;
  height: 4px;
  border-radius: 2px;
  background: var(--border-color);
  outline: none;
  cursor: pointer;
}

.thumb-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--accent);
  cursor: pointer;
  border: none;
}

.thumb-slider::-moz-range-thumb {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--accent);
  cursor: pointer;
  border: none;
}
</style>
