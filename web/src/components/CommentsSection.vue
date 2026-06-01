<template>
  <div class="space-y-3">
    <div v-for="comment in comments" :key="comment.id" class="flex gap-3">
      <div class="flex-1">
        <div class="flex items-center gap-2 mb-0.5">
          <span class="text-sm font-medium text-gray-200">{{ comment.username }}</span>
          <span class="text-xs text-gray-500">{{ formatDate(comment.created_at) }}</span>
        </div>
        <p class="text-sm text-gray-300">{{ comment.content }}</p>
        <div class="flex items-center gap-1 mt-1">
          <button
            v-for="r in comment.reactions"
            :key="r.emoji"
            @click="emit('toggleReaction', comment.id, r.emoji)"
            class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs"
            :class="r.has_mine ? 'bg-blue-500/20 text-blue-300' : 'bg-gray-700/50 text-gray-400 hover:bg-gray-700'"
          >
            {{ r.emoji }} <span class="text-gray-500">{{ r.count }}</span>
          </button>
          <ReactionPicker @select="(e: string) => emit('toggleReaction', comment.id, e)" />
        </div>
      </div>
      <button
        v-if="comment.user_id === currentUserId"
        @click="emit('delete', comment.id)"
        class="text-gray-600 hover:text-red-400 text-xs self-start mt-0.5"
      >×</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ReactionPicker from './ReactionPicker.vue'
import { useAuthStore } from '../stores/auth'

export interface CommentItem {
  id: string
  file_id: string
  user_id: string
  username: string
  content: string
  created_at: string
  updated_at: string
  reactions: { emoji: string; count: number; has_mine: boolean }[]
}

const auth = useAuthStore()
const currentUserId = computed(() => auth.user?.id)

const props = defineProps<{
  comments: CommentItem[]
  fileId: string
}>()

const emit = defineEmits<{
  delete: [id: string]
  toggleReaction: [commentId: string, emoji: string]
}>()

function formatDate(d: string): string {
  return new Date(d).toLocaleString()
}
</script>
