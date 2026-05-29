<template>
  <pre class="whitespace-pre-wrap font-mono text-sm p-4 overflow-auto h-full" v-html="highlighted"></pre>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  content: string
}>()

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

const highlighted = computed(() => {
  try {
    const obj = JSON.parse(props.content)
    const formatted = JSON.stringify(obj, null, 2)
    return formatSyntax(formatted)
  } catch {
    return `<span class="text-red-400">${escapeHtml(props.content)}</span>`
  }
})

function formatSyntax(json: string): string {
  return json.replace(
    /("(?:\\.|[^"\\])*")\s*:|("(?:\\.|[^"\\])*")|(-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b)|(\btrue\b|\bfalse\b|\bnull\b)/g,
    (match, key, str, num, kw) => {
      if (key) return `<span class="text-yellow-300">${escapeHtml(key)}</span>:`
      if (str) return `<span class="text-green-300">${escapeHtml(str)}</span>`
      if (num) return `<span class="text-blue-300">${escapeHtml(num)}</span>`
      if (kw) return `<span class="text-purple-300">${escapeHtml(kw)}</span>`
      return escapeHtml(match)
    },
  )
}
</script>
