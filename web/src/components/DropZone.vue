<template>
  <div
    class="drop-zone-wrapper"
    @dragenter.prevent="onDragEnter"
    @dragover.prevent="onDragOver"
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
  >
    <div
      v-if="dragDepth > 0"
      class="fixed inset-0 z-50 flex items-center justify-center pointer-events-none"
    >
      <div
        class="w-full h-full flex flex-col items-center justify-center gap-3"
        style="background: var(--bg-primary); opacity: 0.92; border: 3px dashed var(--accent); border-radius: 16px; margin: 16px"
      >
        <div class="text-4xl">&#x2B07;</div>
        <div class="text-lg font-medium text-[var(--text-primary)]">Drop files here to upload</div>
        <div v-if="folderLabel" class="text-sm text-[var(--text-secondary)]">{{ folderLabel }}</div>
      </div>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useChunkedUploadStore } from '../stores/chunkedUpload'

const props = defineProps<{
  folderId?: string | null
  folderLabel?: string
}>()

const upload = useChunkedUploadStore()
const dragDepth = ref(0)

function onDragEnter(_e: DragEvent) {
  dragDepth.value++
}

function onDragOver(_e: DragEvent) {
}

function onDragLeave(_e: DragEvent) {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}

function onDrop(e: DragEvent) {
  dragDepth.value = 0
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    upload.uploadFiles(files, props.folderId ?? null, true)
  }
}
</script>
