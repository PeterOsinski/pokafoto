<template>
  <div>
    <h2 class="text-xl font-bold mb-6 text-[var(--text-primary)]">Admin Panel</h2>

    <div class="mb-6 p-4 rounded-md" style="background: var(--bg-surface)">
      <span class="text-sm text-[var(--text-secondary)]">Registration: {{ regEnabled ? 'Enabled' : 'Disabled' }}</span>
    </div>

    <div class="overflow-x-auto rounded-md" style="background: var(--bg-surface)">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b" style="border-color: var(--border-color)">
            <th class="text-left p-3 text-[var(--text-secondary)]">Username</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Role</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Files</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id" class="border-b" style="border-color: var(--border-color)">
            <td class="p-3 text-[var(--text-primary)]">{{ user.username }}</td>
            <td class="p-3">
              <span class="px-2 py-0.5 rounded text-xs" :class="user.role === 'admin' ? 'bg-purple-500/20 text-purple-400' : 'bg-blue-500/20 text-blue-400'">
                {{ user.role }}
              </span>
            </td>
            <td class="p-3 text-[var(--text-secondary)]">{{ user.file_count || 0 }}</td>
            <td class="p-3">
              <select
                class="px-2 py-1 rounded text-xs mr-2"
                :value="user.role"
                @change="changeRole(user.id, ($event.target as HTMLSelectElement).value)"
                style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
              >
                <option value="admin">Admin</option>
                <option value="member">Member</option>
              </select>
              <button
                @click="deleteUser(user.id)"
                class="px-2 py-1 rounded text-xs text-[var(--error)]"
                style="background: var(--bg-elevated)"
              >Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="users.length === 0 && !loading" class="p-6 text-center text-[var(--text-secondary)]">No users found.</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api/client'

interface AdminUser {
  id: string
  username: string
  role: string
  display_name?: string
  file_count?: number
}

const users = ref<AdminUser[]>([])
const regEnabled = ref(true)
const loading = ref(false)

async function loadUsers() {
  loading.value = true
  try {
    const res = await api.get('/admin/users')
    users.value = res.data.users
  } catch (e) {
    console.error('Failed to load users', e)
  } finally {
    loading.value = false
  }
}

async function changeRole(userId: string, role: string) {
  try {
    await api.put(`/admin/users/${userId}/role`, { role })
    loadUsers()
  } catch (e) {
    console.error('Failed to change role', e)
  }
}

async function deleteUser(userId: string) {
  if (!confirm('Delete this user and all their files?')) return
  try {
    await api.delete(`/admin/users/${userId}`)
    loadUsers()
  } catch (e) {
    console.error('Failed to delete user', e)
  }
}

onMounted(loadUsers)
</script>
