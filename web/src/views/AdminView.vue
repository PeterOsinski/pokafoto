<template>
  <div>
    <h2 class="text-xl font-bold mb-6 text-[var(--text-primary)]">Admin Panel</h2>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <div class="p-4 rounded-md" style="background: var(--bg-surface)">
        <h3 class="text-sm font-semibold mb-3 text-[var(--text-secondary)]">Storage</h3>
        <div v-if="stats" class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Used</span>
            <span class="text-[var(--text-primary)]">{{ formatBytes(stats.disk_used_bytes) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Total</span>
            <span class="text-[var(--text-primary)]">{{ formatBytes(stats.disk_total_bytes) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Cache</span>
            <span class="text-[var(--text-primary)]">{{ formatBytes(stats.cache_size_bytes) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Files</span>
            <span class="text-[var(--text-primary)]">{{ stats.total_files }}</span>
          </div>
          <div class="mt-3">
            <div class="flex justify-between text-xs mb-1">
              <span class="text-[var(--text-secondary)]">Disk Utilization</span>
              <span class="text-[var(--text-primary)]">{{ stats ? stats.disk_utilization_pct.toFixed(1) : '0.0' }}%</span>
            </div>
            <div class="w-full h-2 rounded-full" style="background: var(--bg-elevated)">
              <div
                class="h-2 rounded-full transition-all"
                :class="utilizationClass"
                :style="{ width: Math.min(stats?.disk_utilization_pct ?? 0, 100) + '%' }"
              />
            </div>
          </div>
        </div>
        <div v-else class="text-sm text-[var(--text-secondary)]">Loading...</div>
      </div>

      <div class="p-4 rounded-md" style="background: var(--bg-surface)">
        <h3 class="text-sm font-semibold mb-3 text-[var(--text-secondary)]">Worker Pool</h3>
        <div v-if="workers" class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Workers</span>
            <span class="text-[var(--text-primary)]">{{ workers.active_workers }} / {{ workers.total_workers }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Queue</span>
            <span class="text-[var(--text-primary)]">{{ workers.queue_length }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Completed</span>
            <span class="text-[var(--accent)]">{{ workers.completed_total }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-[var(--text-secondary)]">Failed</span>
            <span class="text-[var(--error)]">{{ workers.failed_total }}</span>
          </div>
          <div v-if="workers?.processing_jobs?.length > 0" class="mt-3 border-t pt-2" style="border-color: var(--border-color)">
            <p class="text-xs text-[var(--text-secondary)] mb-1">Processing:</p>
            <div v-for="job in workers.processing_jobs" :key="job.job_id" class="text-xs space-y-0.5 mb-2">
              <div class="flex justify-between text-[var(--text-primary)]">
                <span class="truncate mr-2">{{ job.filename }}</span>
                <span>{{ (job.progress * 100).toFixed(0) }}%</span>
              </div>
              <div class="w-full h-1 rounded-full" style="background: var(--bg-elevated)">
                <div class="h-1 rounded-full bg-[var(--accent)]" :style="{ width: (job.progress * 100) + '%' }" />
              </div>
              <span class="text-[var(--text-secondary)]">{{ job.stage || job.status }}</span>
            </div>
          </div>
        </div>
        <div v-else class="text-sm text-[var(--text-secondary)]">Loading...</div>
      </div>
    </div>

    <div class="mb-6 p-4 rounded-md" style="background: var(--bg-surface)">
      <h3 class="text-sm font-semibold mb-3 text-[var(--text-secondary)]">File Breakdown</h3>
      <div v-if="breakdown" class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div>
          <h4 class="text-xs font-semibold mb-2 text-[var(--text-secondary)]">By Media Type</h4>
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b" style="border-color: var(--border-color)">
                <th class="text-left py-1 text-[var(--text-secondary)] font-normal">Type</th>
                <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Count</th>
                <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Size</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="mt in breakdown.media_types" :key="mt.media_type" class="border-b" style="border-color: var(--border-color)">
                <td class="py-1 text-[var(--text-primary)] capitalize">{{ mt.media_type }}</td>
                <td class="py-1 text-right text-[var(--text-primary)]">{{ mt.count.toLocaleString() }}</td>
                <td class="py-1 text-right text-[var(--text-primary)]">{{ formatBytes(mt.size_bytes) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div>
          <h4 class="text-xs font-semibold mb-2 text-[var(--text-secondary)]">By Extension</h4>
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b" style="border-color: var(--border-color)">
                <th class="text-left py-1 text-[var(--text-secondary)] font-normal">Extension</th>
                <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Count</th>
                <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Size</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ext in breakdown.extensions" :key="ext.extension" class="border-b" style="border-color: var(--border-color)">
                <td class="py-1 text-[var(--text-primary)]">{{ ext.extension }}</td>
                <td class="py-1 text-right text-[var(--text-primary)]">{{ ext.count.toLocaleString() }}</td>
                <td class="py-1 text-right text-[var(--text-primary)]">{{ formatBytes(ext.size_bytes) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-if="breakdown" class="mt-3 pt-3 border-t flex justify-between text-sm" style="border-color: var(--border-color)">
        <span class="text-[var(--text-secondary)]">Total Size (all files)</span>
        <span class="font-semibold text-[var(--text-primary)]">{{ formatBytes(breakdown.total_size) }}</span>
      </div>
      <div v-if="!breakdown" class="text-sm text-[var(--text-secondary)]">Loading...</div>
    </div>

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
            <th class="text-left p-3 text-[var(--text-secondary)]">Size</th>
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
            <td class="p-3 text-[var(--text-secondary)]">{{ user.total_size_bytes ? formatBytes(user.total_size_bytes) : '-' }}</td>
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
      <div v-if="users.length === 0" class="p-6 text-center text-[var(--text-secondary)]">No users found.</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import api from '../api/client'

interface AdminUser {
  id: string
  username: string
  role: string
  display_name?: string
  file_count?: number
  total_size_bytes?: number
}

interface AdminStats {
  total_files: number
  total_size_bytes: number
  cache_size_bytes: number
  disk_total_bytes: number
  disk_free_bytes: number
  disk_used_bytes: number
  disk_utilization_pct: number
  users: AdminUser[]
}

interface WorkerJob {
  job_id: string
  filename: string
  status: string
  stage?: string
  progress: number
}

interface WorkerStats {
  queue_length: number
  active_workers: number
  total_workers: number
  processing_jobs: WorkerJob[]
  completed_total: number
  failed_total: number
  skipped_total: number
}

interface MediaTypeBreakdown {
  media_type: string
  count: number
  size_bytes: number
}

interface ExtensionBreakdown {
  extension: string
  count: number
  size_bytes: number
}

interface FileBreakdown {
  media_types: MediaTypeBreakdown[]
  extensions: ExtensionBreakdown[]
  total_size: number
}

const users = ref<AdminUser[]>([])
const regEnabled = ref(true)
const stats = ref<AdminStats | null>(null)
const workers = ref<WorkerStats | null>(null)
const breakdown = ref<FileBreakdown | null>(null)
let statsTimer: ReturnType<typeof setInterval> | null = null
let workersTimer: ReturnType<typeof setInterval> | null = null

const utilizationClass = computed(() => {
  if (!stats.value) return ''
  const pct = stats.value.disk_utilization_pct
  if (pct > 80) return 'bg-[var(--error)]'
  if (pct > 60) return 'bg-[var(--warning)]'
  return 'bg-[var(--accent)]'
})

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return val.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

async function loadStats() {
  try {
    const res = await api.get('/admin/stats')
    stats.value = res.data
    users.value = res.data.users || []
  } catch (e) {
    console.error('Failed to load stats', e)
  }
}

async function loadBreakdown() {
  try {
    const res = await api.get('/admin/files/breakdown')
    breakdown.value = res.data
  } catch (e) {
    console.error('Failed to load breakdown', e)
  }
}

async function loadWorkers() {
  try {
    const res = await api.get('/admin/workers')
    workers.value = res.data
  } catch (e) {
    console.error('Failed to load workers', e)
  }
}

async function changeRole(userId: string, role: string) {
  try {
    await api.put(`/admin/users/${userId}/role`, { role })
    loadStats()
  } catch (e) {
    console.error('Failed to change role', e)
  }
}

async function deleteUser(userId: string) {
  if (!confirm('Delete this user and all their files?')) return
  try {
    await api.delete(`/admin/users/${userId}`)
    loadStats()
  } catch (e) {
    console.error('Failed to delete user', e)
  }
}

onMounted(() => {
  loadStats()
  loadBreakdown()
  loadWorkers()
  statsTimer = setInterval(() => { loadStats(); loadBreakdown() }, 10000)
  workersTimer = setInterval(loadWorkers, 5000)
})

onUnmounted(() => {
  if (statsTimer) clearInterval(statsTimer)
  if (workersTimer) clearInterval(workersTimer)
})
</script>
