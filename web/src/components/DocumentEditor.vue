<template>
  <div
    v-if="fileId"
    class="fixed inset-0 z-50 flex flex-col"
    style="background: var(--bg-primary, #0d0d0d)"
    @keydown.escape="close"
    ref="wrapperEl"
    tabindex="0"
  >
    <div class="flex items-center justify-between p-3 shrink-0" style="background: var(--bg-elevated, #1a1a1a)">
      <button class="text-white text-xl hover:text-[var(--accent)]" @click="close">✕</button>
      <span class="text-white text-sm truncate max-w-[60%]">{{ documentName }}</span>
      <div class="flex items-center gap-3">
        <span v-if="saving" class="text-xs text-yellow-400">Saving...</span>
        <span v-else-if="saved" class="text-xs text-green-400">Saved</span>
        <button class="text-white text-sm hover:text-[var(--accent)]" @click="downloadDoc">⬇ Download</button>
      </div>
    </div>

    <div v-if="loading" class="flex items-center justify-center flex-1">
      <span class="text-[var(--text-secondary)] text-lg">Loading document...</span>
    </div>

    <div v-else class="flex-1 min-h-0">
      <MdEditor
        v-model="content"
        language="en-US"
        :theme="'dark'"
        preview-theme="github"
        :toolbars="toolbars as any"
        class="h-full"
        style="height: 100%"
        @on-change="onContentChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { MdEditor, config } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import api from '../api/client'

config({
  editorConfig: {
    languageUserDefined: {
      'en-US': {
        toolbarTips: {
          bold: 'Bold',
          italic: 'Italic',
          strikeThrough: 'Strikethrough',
          title: 'Title',
          quote: 'Quote',
          unorderedList: 'Unordered List',
          orderedList: 'Ordered List',
          code: 'Code',
          codeRow: 'Inline Code',
          link: 'Link',
          image: 'Image',
          table: 'Table',
          revoke: 'Undo',
          next: 'Redo',
          save: 'Save',
          preview: 'Preview',
          previewOnly: 'Preview Only',
          htmlPreview: 'HTML Preview',
          fullscreen: 'Fullscreen',
        },
      },
    },
  },
})

const props = defineProps<{
  fileId: string
  originalName: string
}>()

const emit = defineEmits<{
  close: []
}>()

const wrapperEl = ref<HTMLElement | null>(null)
const content = ref('')
const documentName = ref(props.originalName)
const loading = ref(true)
const saving = ref(false)
const saved = ref(false)
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
let destroyed = false

const toolbars = [
  'bold',
  'italic',
  'strikethrough',
  'title',
  '|',
  'quote',
  'unorderedList',
  'orderedList',
  'code',
  'codeRow',
  'link',
  'table',
  '-',
  'revoke',
  'next',
  '=',
  'preview',
]

watch(
  () => props.fileId,
  async (id) => {
    if (!id) return
    resetState()
    loading.value = true
    nextTick(() => wrapperEl.value?.focus())
    try {
      const res = await api.get(`/documents/${id}`)
      if (destroyed) return
      content.value = res.data.content || ''
      documentName.value = res.data.originalName || props.originalName
    } catch {
      if (!destroyed) content.value = ''
    } finally {
      if (!destroyed) loading.value = false
    }
  },
  { immediate: true },
)

onMounted(() => {
  nextTick(() => wrapperEl.value?.focus())
})

onUnmounted(() => {
  destroyed = true
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
})

function resetState() {
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  saving.value = false
  saved.value = false
}

function onContentChange(val: string) {
  content.value = val
  saving.value = true
  saved.value = false
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(async () => {
    if (destroyed) return
    try {
      await api.put(`/documents/${props.fileId}`, { content: content.value })
      saved.value = true
    } catch {
      saved.value = false
    } finally {
      saving.value = false
    }
  }, 2000)
}

function close() {
  emit('close')
}

function downloadDoc() {
  const blob = new Blob([content.value], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = documentName.value || 'document.md'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
</script>
