<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="$emit('close')"
    >
      <div
        class="bg-[var(--bg-surface)] rounded-lg shadow-xl p-6 w-full max-w-sm mx-4"
        style="border: 1px solid var(--border-color)"
      >
        <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-4">
          {{ mode === 'set' ? 'Set Folder Password' : mode === 'unlock' ? 'Unlock Folder' : 'Folder Password' }}
        </h3>

        <template v-if="mode === 'set'">
          <p class="text-sm text-[var(--text-secondary)] mb-3">
            Set a password to protect all contents of this folder. You will need this password to view, download, or upload files — even yourself.
          </p>
          <input
            ref="passwordInput"
            v-model="password"
            type="password"
            placeholder="Enter password..."
            class="w-full px-3 py-2 rounded text-sm mb-2"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
            @keyup.enter="submit"
          />
          <input
            v-model="hintInput"
            type="text"
            placeholder="Password hint (optional)"
            class="w-full px-3 py-2 rounded text-sm mb-2"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
          />
          <p class="text-xs text-[var(--text-secondary)] mb-4">
            Password expires after {{ expiryMinutes }} minutes of inactivity.
          </p>
          <p v-if="error" class="text-sm text-[var(--error)] mb-3">{{ error }}</p>
          <div class="flex justify-end gap-3">
            <button
              @click="$emit('close')"
              class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
            >
              Cancel
            </button>
            <button
              @click="submit"
              :disabled="!password.trim() || loading"
              class="px-4 py-2 rounded text-sm text-white disabled:opacity-50"
              style="background: var(--accent)"
            >
              {{ loading ? 'Setting...' : 'Set Password' }}
            </button>
          </div>
        </template>

        <template v-if="mode === 'unlock'">
          <p class="text-sm text-[var(--text-secondary)] mb-3">
            {{ passwordHint || 'This folder is password-protected. Enter the password to access its contents.' }}
          </p>
          <input
            ref="passwordInput"
            v-model="password"
            type="password"
            placeholder="Password..."
            class="w-full px-3 py-2 rounded text-sm mb-2"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
            @keyup.enter="submit"
          />
          <p v-if="error" class="text-sm text-[var(--error)] mb-3">{{ error }}</p>
          <div class="flex justify-end gap-3">
            <button
              @click="$emit('close')"
              class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
            >
              Cancel
            </button>
            <button
              @click="submit"
              :disabled="!password.trim() || loading"
              class="px-4 py-2 rounded text-sm text-white disabled:opacity-50"
              style="background: var(--accent)"
            >
              {{ loading ? 'Unlocking...' : 'Unlock' }}
            </button>
          </div>
        </template>

        <template v-if="mode === 'status'">
          <div class="flex items-center gap-2 mb-3">
            <span class="text-green-500 text-lg">{{ unlocked ? '\uD83D\uDD13' : '\uD83D\uDD12' }}</span>
            <span class="text-sm text-[var(--text-primary)] font-medium">
              {{ unlocked ? 'Unlocked' : 'Locked' }}
            </span>
          </div>
          <template v-if="unlocked">
            <p class="text-xs text-[var(--text-secondary)] mb-4">
              Unlock expires in {{ timeLeft }}
            </p>
          </template>
          <template v-else>
            <p class="text-xs text-[var(--text-secondary)] mb-4">
              Password expires at {{ expiresAt }}
            </p>
          </template>
          <p v-if="passwordHint" class="text-xs text-[var(--text-secondary)] mb-2 px-2 py-1 rounded" style="background: var(--bg-elevated)">Hint: {{ passwordHint }}</p>
          <div class="flex justify-end gap-3">
            <button
              @click="removePassword"
              :disabled="removing"
              class="px-4 py-2 rounded text-sm text-white disabled:opacity-50"
              style="background: #ef4444"
            >
              {{ removing ? 'Removing...' : 'Remove Password' }}
            </button>
            <button
              @click="$emit('close')"
              class="px-4 py-2 rounded text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
            >
              Close
            </button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import api from '../api/client'
import { useFolderUnlockStore } from '../stores/folderUnlock'

const props = defineProps<{
  visible: boolean
  folderId: string
  mode: 'set' | 'unlock' | 'status'
  hasPassword?: boolean
  expiresAt?: string
  passwordHint?: string
}>()

const emit = defineEmits<{
  close: []
  unlocked: [folderId: string]
  removed: []
}>()

const unlockStore = useFolderUnlockStore()
const password = ref('')
const hintInput = ref('')
const error = ref('')
const loading = ref(false)
const removing = ref(false)
const passwordInput = ref<HTMLInputElement | null>(null)
const countdown = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

const expiryMinutes = 30

const unlocked = computed(() => unlockStore.isUnlocked(props.folderId))

const timeLeft = computed(() => {
  if (!countdown.value) return ''
  const mins = Math.floor(countdown.value / 60)
  const secs = countdown.value % 60
  return `${mins}m ${secs}s`
})

watch(() => props.visible, (v) => {
  if (v) {
    error.value = ''
    password.value = ''
    hintInput.value = props.mode === 'set' ? '' : (props.passwordHint || '')
    loading.value = false
    removing.value = false
    nextTick(() => passwordInput.value?.focus())

    if (props.mode === 'status' && unlocked.value) {
      const ms = unlockStore.getTimeLeft(props.folderId)
      countdown.value = Math.floor(ms / 1000)
      timer = setInterval(() => {
        countdown.value = Math.max(0, Math.floor(unlockStore.getTimeLeft(props.folderId) / 1000))
        if (countdown.value <= 0 && timer) {
          clearInterval(timer)
          timer = null
        }
      }, 1000)
    }
  } else {
    if (timer) { clearInterval(timer); timer = null }
  }
})

async function submit() {
  if (!password.value.trim()) return
  loading.value = true
  error.value = ''

  try {
    if (props.mode === 'set') {
      await api.post(`/folders/${props.folderId}/password`, {
        password: password.value,
        password_hint: hintInput.value,
      })
      emit('close')
    } else if (props.mode === 'unlock') {
      const res = await api.post(`/folders/${props.folderId}/unlock`, { password: password.value })
      unlockStore.setToken(props.folderId, res.data.unlock_token, res.data.expires_at)
      emit('unlocked', props.folderId)
    }
  } catch (err: any) {
    const msg = err?.response?.data?.error?.message || err?.message || 'Error'
    error.value = msg
  } finally {
    loading.value = false
  }
}

async function removePassword() {
  removing.value = true
  try {
    await api.delete(`/folders/${props.folderId}/password`)
    unlockStore.removeToken(props.folderId)
    emit('removed')
  } catch (err: any) {
    error.value = err?.response?.data?.error?.message || 'Failed to remove password'
  } finally {
    removing.value = false
  }
}
</script>
