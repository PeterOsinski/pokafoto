<template>
  <div class="min-h-screen flex items-center justify-center bg-[var(--bg-primary)]">
    <div class="w-full max-w-md p-8 rounded-lg" style="background: var(--bg-surface)">
      <h1 class="text-2xl font-bold text-center mb-8 text-[var(--text-primary)]">Drive</h1>
      <form @submit.prevent="handleLogin" class="space-y-4">
        <div>
          <input
            v-model="username"
            type="text"
            placeholder="Username"
            required
            class="w-full px-4 py-2 rounded-md"
            style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
          />
        </div>
        <div>
          <input
            v-model="password"
            type="password"
            placeholder="Password"
            required
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
          {{ loading ? 'Logging in...' : 'Log In' }}
        </button>
      </form>
      <p v-if="registrationAllowed" class="text-center mt-4 text-[var(--text-secondary)] text-sm">
        Don't have an account?
        <router-link to="/register" class="text-[var(--accent)] hover:underline">Register</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const registrationAllowed = ref(false)

onMounted(async () => {
  try {
    const res = await api.get('/auth/config')
    registrationAllowed.value = res.data.allow_registration
  } catch {}
})

async function handleLogin() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value, password.value)
    router.push('/')
  } catch (e: any) {
    error.value = e.response?.data?.error?.message || 'Invalid credentials'
  } finally {
    loading.value = false
  }
}
</script>
