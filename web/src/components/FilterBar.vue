<template>
  <div class="flex items-center gap-4 mb-6 flex-wrap">
    <select :modelValue="mediaType" @change="emit('update:mediaType', ($event.target as HTMLSelectElement).value)" class="px-3 py-1 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)">
      <option value="">All</option>
      <option value="photo">Photos</option>
      <option value="video">Videos</option>
      <option value="file">Files</option>
    </select>
    <select :modelValue="sortBy" @change="emit('update:sortBy', ($event.target as HTMLSelectElement).value)" class="px-3 py-1 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)">
      <option value="taken_at">Date Taken</option>
      <option value="created_at">Date Uploaded</option>
      <option value="filename">File Name</option>
    </select>
  </div>
</template>

<script setup lang="ts">
defineProps<{ mediaType: string; sortBy: string }>()
const emit = defineEmits<{
  'update:mediaType': [value: string]
  'update:sortBy': [value: string]
}>()
</script>
