<template>
  <div class="p-4 overflow-auto h-full text-[var(--text-primary)] markdown-body" v-html="html"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'

const props = defineProps<{
  content: string
}>()

const html = computed(() => {
  try {
    return marked.parse(props.content) as string
  } catch {
    return `<pre class="text-red-400">${escapeHtml(props.content)}</pre>`
  }
})

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
</script>

<style scoped>
.markdown-body :deep(h1) {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 1rem 0 0.5rem;
}
.markdown-body :deep(h2) {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0.75rem 0 0.5rem;
}
.markdown-body :deep(h3) {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0.5rem 0 0.25rem;
}
.markdown-body :deep(p) {
  margin: 0.5rem 0;
  line-height: 1.6;
}
.markdown-body :deep(code) {
  background: rgba(255,255,255,0.1);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
}
.markdown-body :deep(pre) {
  background: rgba(0,0,0,0.4);
  padding: 0.75rem;
  border-radius: 0.375rem;
  overflow-x: auto;
  margin: 0.5rem 0;
}
.markdown-body :deep(pre code) {
  background: none;
  padding: 0;
}
.markdown-body :deep(ul), .markdown-body :deep(ol) {
  padding-left: 1.5rem;
  margin: 0.5rem 0;
}
.markdown-body :deep(li) {
  margin: 0.25rem 0;
}
.markdown-body :deep(blockquote) {
  border-left: 3px solid var(--accent);
  padding-left: 0.75rem;
  margin: 0.5rem 0;
  opacity: 0.8;
}
.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.5rem 0;
}
.markdown-body :deep(th), .markdown-body :deep(td) {
  border: 1px solid var(--border-color);
  padding: 0.375rem 0.75rem;
  text-align: left;
}
.markdown-body :deep(th) {
  background: rgba(255,255,255,0.05);
}
.markdown-body :deep(a) {
  color: var(--accent);
  text-decoration: underline;
}
.markdown-body :deep(hr) {
  border-color: var(--border-color);
  margin: 1rem 0;
}
.markdown-body :deep(img) {
  max-width: 100%;
  border-radius: 0.375rem;
}
</style>
