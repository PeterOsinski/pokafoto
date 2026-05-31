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
      <h3 class="text-sm font-semibold mb-3 text-[var(--text-secondary)]">Thumbnail Stats</h3>
      <div v-if="thumbnailStats" class="space-y-2">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b" style="border-color: var(--border-color)">
              <th class="text-left py-1 text-[var(--text-secondary)] font-normal">Size</th>
              <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Count</th>
              <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Total Size</th>
              <th class="text-right py-1 text-[var(--text-secondary)] font-normal">Avg Size</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in thumbnailStats.breakdown" :key="item.size" class="border-b" style="border-color: var(--border-color)">
              <td class="py-1 text-[var(--text-primary)] capitalize">{{ item.size }}</td>
              <td class="py-1 text-right text-[var(--text-primary)]">{{ item.count.toLocaleString() }}</td>
              <td class="py-1 text-right text-[var(--text-primary)]">{{ formatBytes(item.total_size) }}</td>
              <td class="py-1 text-right text-[var(--text-primary)]">{{ formatBytes(item.count > 0 ? Math.round(item.total_size / item.count) : 0) }}</td>
            </tr>
          </tbody>
        </table>
        <div class="flex justify-between text-sm border-t pt-2" style="border-color: var(--border-color)">
          <span class="text-[var(--text-secondary)]">Total Thumbnails</span>
          <span class="text-[var(--text-primary)] font-semibold">{{ thumbnailStats?.total_count?.toLocaleString() ?? '0' }}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-[var(--text-secondary)]">Total Cache Size</span>
          <span class="text-[var(--text-primary)] font-semibold">{{ thumbnailStats ? formatBytes(thumbnailStats.total_size_bytes) : '0 B' }}</span>
        </div>
      </div>
      <div v-else class="text-sm text-[var(--text-secondary)]">Loading...</div>
    </div>

    <div class="mb-6 p-4 rounded-md" style="background: var(--bg-surface)">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-sm font-semibold text-[var(--text-secondary)]">Job History</h3>
        <button
          @click="reconcileThumbnails"
          class="px-3 py-1 rounded text-xs"
          style="background: var(--bg-elevated); color: var(--accent); border: 1px solid var(--border-color)"
          :disabled="reconciling"
        >{{ reconciling ? 'Reconciling...' : 'Reconcile Thumbnails' }}</button>
      </div>
      <div class="flex gap-2 mb-3">
        <button
          v-for="tab in jobStatusTabs"
          :key="tab.value"
          @click="jobStatusFilter = tab.value; loadJobs()"
          class="px-2 py-0.5 rounded text-xs border"
          :style="{
            background: jobStatusFilter === tab.value ? 'var(--accent)' : 'var(--bg-elevated)',
            color: jobStatusFilter === tab.value ? '#fff' : 'var(--text-secondary)',
            borderColor: 'var(--border-color)',
          }"
        >{{ tab.label }} <span class="ml-1 opacity-60">{{ jobSummary[tab.value] ?? 0 }}</span></button>
      </div>
      <div v-if="jobs.length > 0" class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b" style="border-color: var(--border-color)">
              <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-normal">Filename</th>
              <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-normal">Status</th>
              <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-normal">Error / Reason</th>
              <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-normal">Created</th>
              <th class="text-right py-1 px-2 text-[var(--text-secondary)] font-normal">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="job in jobs" :key="job.id" class="border-b" style="border-color: var(--border-color)">
              <td class="py-1 px-2 text-[var(--text-primary)] max-w-[200px] truncate">{{ job.filename }}</td>
              <td class="py-1 px-2">
                <span class="px-1.5 py-0.5 rounded text-xs" :class="statusBadgeClass(job.status)">
                  {{ job.status }}
                </span>
              </td>
              <td class="py-1 px-2 text-[var(--text-secondary)] text-xs max-w-[200px] truncate">{{ job.error || job.reason || '-' }}</td>
              <td class="py-1 px-2 text-[var(--text-secondary)] text-xs">{{ formatDate(job.created_at) }}</td>
              <td class="py-1 px-2 text-right">
                <button
                  v-if="job.status === 'failed'"
                  @click="retryJob(job.id)"
                  class="px-2 py-0.5 rounded text-xs text-[var(--accent)]"
                  style="background: var(--bg-elevated)"
                >Retry</button>
              </td>
            </tr>
          </tbody>
        </table>
        <div class="flex justify-between items-center mt-2 text-xs text-[var(--text-secondary)]">
          <span>{{ jobs.length }} of {{ jobTotal }} jobs</span>
          <div class="flex gap-2">
            <button
              :disabled="jobOffset === 0"
              @click="jobOffset = Math.max(0, jobOffset - 50); loadJobs()"
              class="px-2 py-0.5 rounded disabled:opacity-30" style="background: var(--bg-elevated)"
            >Prev</button>
            <button
              :disabled="jobOffset + 50 >= jobTotal"
              @click="jobOffset += 50; loadJobs()"
              class="px-2 py-0.5 rounded disabled:opacity-30" style="background: var(--bg-elevated)"
            >Next</button>
          </div>
        </div>
      </div>
      <div v-else class="text-sm text-[var(--text-secondary)]">No jobs found.</div>
    </div>

    <div class="mb-6 p-4 rounded-md flex items-center justify-between" style="background: var(--bg-surface)">
      <span class="text-sm text-[var(--text-secondary)]">Registration: <span :class="regEnabled ? 'text-green-400' : 'text-[var(--error)]'">{{ regEnabled ? 'Enabled' : 'Disabled' }}</span></span>
      <button
        @click="toggleRegistration"
        :disabled="registrationLoading"
        class="px-3 py-1 rounded text-xs"
        :style="{ background: regEnabled ? 'var(--error)' : 'var(--accent)', color: '#fff', opacity: registrationLoading ? 0.5 : 1 }"
      >{{ registrationLoading ? '...' : (regEnabled ? 'Disable' : 'Enable') }}</button>
    </div>

    <div class="overflow-x-auto rounded-md" style="background: var(--bg-surface)">
      <div class="flex items-center justify-between p-3 border-b" style="border-color: var(--border-color)">
        <h3 class="text-sm font-semibold text-[var(--text-secondary)]">Users</h3>
        <button
          @click="showCreateUser = true"
          class="px-3 py-1 rounded text-xs"
          style="background: var(--accent); color: #fff"
        >Create User</button>
      </div>
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b" style="border-color: var(--border-color)">
            <th class="text-left p-3 text-[var(--text-secondary)]">Username</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Role</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Files</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Size</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Thumbnails</th>
            <th class="text-left p-3 text-[var(--text-secondary)]">Quota</th>
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
            <td class="p-3 text-[var(--text-secondary)]">{{ user.thumbnail_size_bytes !== undefined ? formatBytes(user.thumbnail_size_bytes) : '-' }}</td>
            <td class="p-3 text-[var(--text-primary)]">
              <template v-if="editingQuota === user.id">
                <div class="flex items-center gap-1">
                  <input
                    v-model="quotaInput"
                    type="text"
                    placeholder="bytes"
                    class="w-24 px-1 py-0.5 rounded text-xs"
                    style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)"
                    @keyup.enter="saveQuota(user.id)"
                    @keyup.escape="cancelEditQuota()"
                  />
                  <button @click="saveQuota(user.id)" class="px-1.5 py-0.5 rounded text-xs text-white" style="background: var(--accent)">Save</button>
                  <button @click="cancelEditQuota()" class="px-1.5 py-0.5 rounded text-xs" style="background: var(--bg-elevated); color: var(--text-secondary)">Cancel</button>
                </div>
                <p v-if="quotaError" class="text-[var(--error)] text-xs mt-0.5">{{ quotaError }}</p>
              </template>
              <template v-else>
                <span class="text-xs">{{ user.space_quota ? formatBytes(user.space_quota) : 'Unlimited' }}</span>
                <button @click="startEditQuota(user)" class="ml-1 px-1 py-0.5 rounded text-xs text-[var(--accent)]" style="background: var(--bg-elevated)">Edit</button>
              </template>
            </td>
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
      <div v-if="users.length === 0 && !showCreateUser" class="p-6 text-center text-[var(--text-secondary)]">No users found.</div>
    </div>

    <div v-if="showCreateUser" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="showCreateUser = false">
      <div class="w-full max-w-md p-6 rounded-lg" style="background: var(--bg-surface)">
        <h3 class="text-lg font-semibold mb-4 text-[var(--text-primary)]">Create User</h3>
        <form @submit.prevent="handleCreateUser" class="space-y-3">
          <input v-model="newUser.username" type="text" placeholder="Username (3-32 chars)" minlength="3" maxlength="32" required
            class="w-full px-3 py-2 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)" />
          <input v-model="newUser.password" type="password" placeholder="Password (8+ chars)" minlength="8" required
            class="w-full px-3 py-2 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)" />
          <select v-model="newUser.role"
            class="w-full px-3 py-2 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)">
            <option value="member">Member</option>
            <option value="admin">Admin</option>
          </select>
          <input v-model="newUser.displayName" type="text" placeholder="Display Name (optional)"
            class="w-full px-3 py-2 rounded text-sm" style="background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border-color)" />
          <p v-if="createUserError" class="text-[var(--error)] text-sm">{{ createUserError }}</p>
          <div class="flex gap-2 justify-end">
            <button type="button" @click="showCreateUser = false" class="px-4 py-2 rounded text-sm"
              style="background: var(--bg-elevated); color: var(--text-secondary)">Cancel</button>
            <button type="submit" :disabled="createUserLoading"
              class="px-4 py-2 rounded text-sm text-white" style="background: var(--accent)">
              {{ createUserLoading ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
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
  space_quota?: number | null
  thumbnail_size_bytes?: number
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

interface ThumbnailBreakdownItem {
  size: string
  count: number
  total_size: number
}

interface ThumbnailStats {
  breakdown: ThumbnailBreakdownItem[]
  total_count: number
  total_size_bytes: number
}

interface JobRecord {
  id: string
  batch_id: string
  user_id: string
  filename: string
  size_bytes: number
  status: string
  stage?: string
  progress: number
  error?: string
  reason?: string
  file_id?: string
  created_at: string
  updated_at: string
}

const users = ref<AdminUser[]>([])
const regEnabled = ref(true)
const registrationLoading = ref(false)
const showCreateUser = ref(false)
const createUserLoading = ref(false)
const createUserError = ref('')
const newUser = ref({ username: '', password: '', role: 'member', displayName: '' })
const stats = ref<AdminStats | null>(null)
const workers = ref<WorkerStats | null>(null)
const breakdown = ref<FileBreakdown | null>(null)
const jobs = ref<JobRecord[]>([])
const jobTotal = ref(0)
const jobOffset = ref(0)
const jobStatusFilter = ref('')
const jobSummary = ref<Record<string, number>>({})
const reconciling = ref(false)
const thumbnailStats = ref<ThumbnailStats | null>(null)
const editingQuota = ref<string | null>(null)
const quotaInput = ref('')
const quotaError = ref('')
let statsTimer: ReturnType<typeof setInterval> | null = null
let workersTimer: ReturnType<typeof setInterval> | null = null
let jobsTimer: ReturnType<typeof setInterval> | null = null
let thumbStatsTimer: ReturnType<typeof setInterval> | null = null

const jobStatusTabs = [
  { label: 'All', value: '' },
  { label: 'Completed', value: 'completed' },
  { label: 'Failed', value: 'failed' },
  { label: 'Skipped', value: 'skipped' },
  { label: 'Processing', value: 'processing' },
  { label: 'Queued', value: 'queued' },
]

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

async function loadRegistrationStatus() {
  try {
    const res = await api.get('/admin/registration')
    regEnabled.value = res.data.allow_registration
  } catch (e) {
    console.error('Failed to load registration status', e)
  }
}

async function toggleRegistration() {
  registrationLoading.value = true
  try {
    const res = await api.put('/admin/registration', { enabled: !regEnabled.value })
    regEnabled.value = res.data.allow_registration
  } catch (e: any) {
    console.error('Failed to toggle registration', e)
    alert('Failed to update: ' + (e.response?.data?.error?.message || 'Unknown error'))
  } finally {
    registrationLoading.value = false
  }
}

async function handleCreateUser() {
  createUserLoading.value = true
  createUserError.value = ''
  try {
    await api.post('/admin/users', {
      username: newUser.value.username,
      password: newUser.value.password,
      role: newUser.value.role,
      display_name: newUser.value.displayName || undefined,
    })
    showCreateUser.value = false
    newUser.value = { username: '', password: '', role: 'member', displayName: '' }
    loadStats()
  } catch (e: any) {
    createUserError.value = e.response?.data?.error?.message || 'Failed to create user'
  } finally {
    createUserLoading.value = false
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

async function loadJobs() {
  try {
    const params: Record<string, string> = { limit: '50', offset: String(jobOffset) }
    if (jobStatusFilter.value) params.status = jobStatusFilter.value
    const res = await api.get('/admin/jobs', { params })
    jobs.value = res.data.jobs || []
    jobTotal.value = res.data.total || 0
    jobSummary.value = res.data.summary || {}
  } catch (e) {
    console.error('Failed to load jobs', e)
  }
}

async function retryJob(jobId: string) {
  try {
    await api.post(`/admin/jobs/${jobId}/retry`)
    loadJobs()
  } catch (e) {
    console.error('Failed to retry job', e)
  }
}

async function reconcileThumbnails() {
  if (!confirm('Scan for missing thumbnails and regenerate them? This may take a while.')) return
  reconciling.value = true
  try {
    const res = await api.post('/admin/jobs/reconcile')
    alert(`Reconciliation created ${res.data.created} jobs:\n- ${res.data.details?.missing_all_thumbnails ?? 0} files with 0 thumbnails\n- ${res.data.details?.missing_preview_only ?? 0} files missing preview only`)
    loadJobs()
  } catch (e) {
    console.error('Failed to reconcile', e)
  } finally {
    reconciling.value = false
  }
}

async function loadThumbnailStats() {
  try {
    const res = await api.get('/admin/thumbnails/stats')
    thumbnailStats.value = res.data
  } catch (e) {
    console.error('Failed to load thumbnail stats', e)
  }
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'completed': return 'bg-green-500/20 text-green-400'
    case 'failed': return 'bg-red-500/20 text-red-400'
    case 'skipped': return 'bg-yellow-500/20 text-yellow-400'
    case 'processing': return 'bg-blue-500/20 text-blue-400'
    case 'queued': return 'bg-gray-500/20 text-gray-400'
    default: return ''
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString()
}

function startEditQuota(user: AdminUser) {
  editingQuota.value = user.id
  quotaError.value = ''
  if (user.space_quota) {
    quotaInput.value = String(user.space_quota)
  } else {
    quotaInput.value = ''
  }
}

function cancelEditQuota() {
  editingQuota.value = null
  quotaInput.value = ''
  quotaError.value = ''
}

async function saveQuota(userId: string) {
  const raw = quotaInput.value.trim()
  let quotaValue: number | null = null
  if (raw !== '') {
    const parsed = parseInt(raw, 10)
    if (isNaN(parsed) || parsed < 0) {
      quotaError.value = 'Invalid number'
      return
    }
    quotaValue = parsed
  }
  try {
    await api.put(`/admin/users/${userId}/quota`, { space_quota: quotaValue })
    editingQuota.value = null
    quotaInput.value = ''
    quotaError.value = ''
    loadStats()
  } catch (e: any) {
    quotaError.value = e.response?.data?.error?.message || 'Failed to update quota'
  }
}

onMounted(() => {
  loadStats()
  loadBreakdown()
  loadWorkers()
  loadJobs()
  loadThumbnailStats()
  loadRegistrationStatus()
  statsTimer = setInterval(() => { loadStats(); loadBreakdown() }, 10000)
  workersTimer = setInterval(loadWorkers, 5000)
  jobsTimer = setInterval(loadJobs, 10000)
  thumbStatsTimer = setInterval(loadThumbnailStats, 30000)
})

onUnmounted(() => {
  if (statsTimer) clearInterval(statsTimer)
  if (workersTimer) clearInterval(workersTimer)
  if (jobsTimer) clearInterval(jobsTimer)
  if (thumbStatsTimer) clearInterval(thumbStatsTimer)
})
</script>
