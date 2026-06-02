<template>
  <div class="flex items-center gap-2 mb-6 flex-wrap">
    <select
      v-if="(sortOptions || defaultSortOptions).length > 0"
      :modelValue="sortBy"
      @change="$emit('update:sortBy', ($event.target as HTMLSelectElement).value)"
      class="px-3 py-1 rounded text-sm"
      style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
    >
      <option v-for="opt in sortOptions || defaultSortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
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

    <button
      v-if="!hidePreviewToggle"
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
defineProps<{
  layout: string
  sortBy: string
  thumbLevel: number
  previewMode: string
  sortOptions?: { value: string; label: string }[]
  hidePreviewToggle?: boolean
}>()

defineEmits<{
  'update:layout': [value: string]
  'update:sortBy': [value: string]
  'update:thumbLevel': [value: number]
  'togglePreviewMode': []
}>()

const defaultSortOptions = [
  { value: 'taken_at', label: 'Date Taken' },
  { value: 'created_at', label: 'Date Uploaded' },
  { value: 'filename', label: 'File Name' },
]

const layoutOptions = [
  { value: 'tiles', label: 'Tiles', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>' },
  { value: 'list', label: 'List', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>' },
  { value: 'grouped', label: 'Calendar', icon: '<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>' },
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
