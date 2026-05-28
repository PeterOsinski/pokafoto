<template>
  <div class="min-h-screen flex items-center justify-center bg-[var(--bg-primary)]">
    <div class="w-full max-w-md p-8 rounded-lg" style="background: var(--bg-surface)">
      <h1 class="text-2xl font-bold text-center mb-8 text-[var(--text-primary)]">Create Account</h1>
      <form @submit.prevent="handleRegister" class="space-y-4">
        <div>
          <input
            v-model="username"
            type="text"
            placeholder="Username (3-32 chars)"
            minlength="3"
            maxlength="32"
            required
            class="w-full px-4 py-2 rounded-md"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
          />
        </div>
        <div>
          <input
            v-model="password"
            type="password"
            placeholder="Password (8+ chars)"
            minlength="8"
            required
            class="w-full px-4 py-2 rounded-md"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
          />
        </div>
        <div>
          <input
            v-model="displayName"
            type="text"
            placeholder="Display Name (optional)"
            class="w-full px-4 py-2 rounded-md"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
          />
        </div>
        <p v-if="error" class="text-[var(--error)] text-sm">{{ error }}</p>
        <button
          type="submit"
          :disabled="loading"
          class="w-full py-2 rounded-md font-medium text-white transition-opacity"
          style="background: var(--accent)"
        >
          {{ loading ? 'Creating...' : 'Create Account' }}
        </button>
      </form>
      <p class="text-center mt-4 text-[var(--text-secondary)] text-sm">
        Already have an account?
        <router-link to="/login" class="text-[var(--accent)] hover:underline">Log In</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const displayName = ref('')
const loading = ref(false)
const error = ref('')

async function handleRegister() {
  loading.value = true
  error.value = ''
  try {
    await auth.register(username.value, password.value, displayName.value || undefined)
    await auth.login(username.value, password.value)
    router.push('/')
  } catch (e: any) {
    error.value = e.response?.data?.error?.message || 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>
