<template>
  <nav v-if="chain.length > 0" class="flex items-center gap-1 text-sm overflow-x-auto py-1" aria-label="Breadcrumb">
    <template v-for="(segment, i) in chain" :key="segment.id ?? 'root'">
      <span v-if="i > 0" class="text-[var(--text-secondary)] mx-0.5 select-none">&gt;</span>
      <button
        v-if="i < chain.length - 1"
        @click="$emit('navigate', segment.id)"
        class="text-[var(--text-secondary)] hover:text-[var(--accent)] truncate max-w-[160px] transition-colors"
      >
        {{ segment.name }}
      </button>
      <span v-else class="text-[var(--text-primary)] font-semibold truncate max-w-[200px]">
        {{ segment.name }}
      </span>
    </template>
  </nav>
</template>

<script setup lang="ts">
interface BreadcrumbSegment {
  id: string | null
  name: string
}

defineProps<{
  chain: BreadcrumbSegment[]
}>()

defineEmits<{
  navigate: [folderId: string | null]
}>()
</script>
