<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="$emit('close')">
    <div class="bg-gray-800 rounded-xl border border-gray-700 p-6 w-full max-w-md" @click.stop>
      <h3 class="text-lg font-medium text-gray-200 mb-4">Share Album</h3>

      <div class="flex gap-2 mb-4">
        <input v-model="username" type="text" placeholder="Username" class="flex-1 px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none focus:border-blue-500" @keydown.enter="doShare" />
        <select v-model="permission" class="px-2 py-2 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-200 outline-none">
          <option value="view">View</option>
          <option value="comment">Comment</option>
          <option value="edit">Edit</option>
        </select>
      </div>
      <button @click="doShare" :disabled="!username.trim()" class="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm rounded-lg mb-4 transition-colors">
        Share
      </button>
      <p v-if="error" class="text-sm text-red-400 mb-4">{{ error }}</p>

      <div v-if="shares.length" class="border-t border-gray-700 pt-3">
        <h4 class="text-sm text-gray-400 mb-2">Shared with</h4>
        <div v-for="s in shares" :key="s.share_id" class="flex items-center justify-between py-1">
          <div class="flex items-center gap-2">
            <span class="text-sm text-gray-200">{{ s.username }}</span>
            <span class="text-xs px-1.5 py-0.5 rounded bg-gray-700 text-gray-400">{{ s.permission }}</span>
          </div>
          <button @click="$emit('removeShare', s.share_id)" class="text-gray-500 hover:text-red-400 text-sm">Remove</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface SharedUser {
  share_id: string
  user_id: string
  username: string
  permission: string
}

defineProps<{
  visible: boolean
  shares: SharedUser[]
  error?: string
}>()

const emit = defineEmits<{
  close: []
  share: [username: string, permission: string]
  removeShare: [shareId: string]
}>()

const username = ref('')
const permission = ref('view')

function doShare() {
  if (username.value.trim()) {
    emit('share', username.value.trim(), permission.value)
    username.value = ''
  }
}
</script>
