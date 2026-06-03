<template>
  <button
    v-if="depth === 0"
    class="flex items-center w-full text-left text-sm py-1.5 px-2 rounded hover:bg-[var(--bg-elevated)] transition-colors"
    :class="selectedId === node.folder.id ? 'text-[var(--accent)]' : 'text-[var(--text-primary)]'"
    @click="$emit('select', node.folder.id)"
  >
    <span :class="expanded ? '' : ''" class="mr-1.5 text-xs w-4 inline-block text-center" @click.prevent.stop="expanded = !expanded">
      {{ node.children?.length ? (expanded ? '&#9660;' : '&#9654;') : '' }}
    </span>
    <span class="mr-1.5">&#128193;</span>
    <span class="flex-1 truncate">{{ node.folder.name }}</span>
    <span class="text-xs text-[var(--text-secondary)] ml-1">{{ node.fileCount }}</span>
  </button>
  <button
    v-else
    class="flex items-center w-full text-left text-sm py-1.5 px-2 rounded hover:bg-[var(--bg-elevated)] transition-colors"
    :class="selectedId === node.folder.id ? 'text-[var(--accent)]' : 'text-[var(--text-primary)]'"
    @click="$emit('select', node.folder.id)"
  >
    <span :class="expanded ? '' : ''" class="mr-1.5 text-xs w-4 inline-block text-center" @click.prevent.stop="expanded = !expanded">
      {{ node.children?.length ? (expanded ? '&#9660;' : '&#9654;') : '' }}
    </span>
    <span class="mr-1.5">&#128193;</span>
    <span class="flex-1 truncate">{{ node.folder.name }}</span>
    <span class="text-xs text-[var(--text-secondary)] ml-1">{{ node.fileCount }}</span>
  </button>
  <div v-if="expanded && node.children?.length" :style="{ paddingLeft: (depth + 1) * 16 + 'px' }">
    <FolderNode
      v-for="child in node.children"
      :key="child.folder.id"
      :node="child"
      :selectedId="selectedId"
      :depth="depth + 1"
      @select="(id: string) => $emit('select', id)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { FolderTreeNode } from '../types/gallery'

defineProps<{
  node: FolderTreeNode
  selectedId: string | null
  depth: number
}>()

defineEmits<{
  select: [id: string]
}>()

const expanded = ref(false)
</script>
