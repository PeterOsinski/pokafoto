<template>
  <div class="inline-flex items-center">
    <button
      @click="triggerUpload"
      class="px-3 py-1 rounded text-sm text-white"
      style="background: var(--accent)"
    >
      {{ label }}
    </button>
      <input
        ref="fileInput"
        type="file"
        multiple
        :accept="accept"
        class="hidden"
        @change="handleChange"
      />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useUploadStore } from '../stores/upload'

const props = withDefaults(defineProps<{
  folderId?: string | null
  label?: string
  skipNameSizeDedup?: boolean
  accept?: string
}>(), {
  folderId: null,
  label: 'Upload here',
  skipNameSizeDedup: true,
  accept: '*',
})

const upload = useUploadStore()
const fileInput = ref<HTMLInputElement | null>(null)

function triggerUpload() {
  fileInput.value?.click()
}

function handleChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    upload.uploadFiles(input.files, props.folderId ?? null, props.skipNameSizeDedup)
    input.value = ''
  }
}
</script>
